# Redefinition-Guard mit Kontext, Redef-Log, makunbound — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Root-Redefinitionen von Lisp-Funktionen (LAMBDA/MACRO) werden mit Quell-Kontext bewacht (Reload derselben Datei bleibt still), alle Redefinitionen landen in einem abfragbaren Log, und `makunbound` macht Fehldefinitionen reparierbar.

**Architecture:**
Bestehender Guard in `Env.Set` (nur FUNC, Policy allow/warn/error) bleibt als Backstop unverändert bestehen und wird nur ums Logging erweitert. Neuer kontextbewusster Guard `checkRootRedefine` wird in den drei definierenden Spezialformen (`define`/`defun`/`defmacro`) VOR `env.Set` gerufen und behandelt ausschließlich LAMBDA/MACRO-Altzbindungen: gleiche Quelle (DefLoc.File) → stiller Reload, fremde Quelle → Policy. `makunbound` wird Spezialform (Primitiven haben kein `env`), nutzt denselben Policy-Helfer. Alle Events landen in einem Ringpuffer (256), abfragbar via `(redef-log)`.

**Tech Stack:** Go (kein neues Dependency), GoLisp2-eigene Test-Helfer (`evalStr`, `withRedefinePolicy`).

## Global Constraints

- Einrückung: **2 Spaces** in `env.go`, `defloc.go`, `eval_specialforms.go`, `eval_core.go` und allen neuen Dateien. **`primitives.go` verwendet Tabs** — dort Tabs.
- Fehlerformat: `fmt.Errorf("funktionsname: beschreibung")`.
- Neue Dateien mit Datei-Header (Autor/CoAutor/Copyright/Erstellt, Stil wie `lib/defloc.go`).
- Kommentare/Doku deutsch.
- Build: `./build.sh`; Tests: `go test ./... -count=1` (Cache!).
- Commit-Trailer: `Co-Authored-By: kimi-k3 <noreply@moonshot.cn>`
- Commit-Style Historie: `feat(redef): ...` / `doc(redef): ...`, deutsch.

### Bekannte Ist-Zustände (verifiziert, Plan baut darauf)

- `Env.Set` (`lib/env.go:178`) ruft am Root `onRootRedefine` nur bei `old.Type == FUNC`.
- `defaultOnRootRedefine` (`lib/env.go:99`), Policy via `redefinePolicyAtomic`, Default `warn`.
- `RegisterDefinition`/`LookupDefinition`/`ClearDefinitions` in `lib/defloc.go`; `SrcFile` wird **nur** von `load` gesetzt (`lib/eval_load.go:54`) — REPL/stdin/swank → `""`. `SrcLine` setzt der Reader (`lib/reader.go:120`).
- `evalDefine` (`lib/eval_specialforms.go:19`), `evalDefun` (:213), `evalDefmacro` (:338): alle `env.Set(name, …)` dann `RegisterDefinition(name, form.SrcFile, form.SrcLine)` — Lookup VOR Set sieht also die ALTE Quelle. Genau das nutzt Task 3.
- Primitiven-Signatur `func([]*Cell) (*Cell, error)` — kein `env`. Deshalb `makunbound` als Spezialform (wie `bound?`, `lib/eval_specialforms.go:103`).
- Test-Helfer: `evalStr` (`lib/eval_test.go:25`, frisches BaseEnv, KEINE stdlib), `withRedefinePolicy` (`lib/env_test.go:33`), stderr-Capture-Muster (`lib/env_test.go:94`).
- `List(...*Cell)` (`lib/types_helpers.go:14`), leer → `MakeNil()`. `MakeNumber(float64)`, `MakeString`, `MakeAtom`, `Cell.Num`.
- `length` ist stdlib-LISP-Funktion, in `evalStr`-Tests NICHT verfügbar.

### Bewusste Grenzen (nicht Teil dieses Plans)

- `setq`/`progv` am Root über LAMBDA: weiter still (kein Quell-Kontext verfügbar). FUNC bleibt dort durch den Set-Hook bewacht.
- FUNC-Log-Events haben `NewFile == ""` (Set-Hook kennt Quelle nicht; goroutine-sichere Übergabe bewusst vermieden).
- `swank/env.go` hat eigenes Env-Handling — unverändert.

---

### Task 1: Redef-Log Kern (Ringpuffer + Go-API)

**Files:**
- Create: `lib/redeflog.go`
- Test: `lib/redeflog_test.go`

**Interfaces:**
- Consumes: `Cell`-Typen aus `lib/types.go` (`FUNC`, `LAMBDA`, `MACRO`).
- Produces: `RedefEvent` (Struct), `logRedef(RedefEvent)`, `RedefLog() []RedefEvent`, `ClearRedefLog()`, `kindOf(*Cell) string` — konsumiert von Tasks 2–5.

- [ ] **Step 1: Failing Test schreiben** — `lib/redeflog_test.go`:

```go
//**********************************************************************
//  lib/redeflog_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260724
//**********************************************************************

package lib

import (
  "fmt"
  "testing"
)

func TestRedefLogAppendAndOrder(t *testing.T) {
  ClearRedefLog()
  logRedef(RedefEvent{Name: "a", OldKind: "lambda", Action: "reload"})
  logRedef(RedefEvent{Name: "b", OldKind: "func", Action: "warn"})
  events := RedefLog()
  if len(events) != 2 {
    t.Fatalf("2 Events erwartet, got %d", len(events))
  }
  if events[0].Name != "a" || events[1].Name != "b" {
    t.Fatalf("Reihenfolge älteste→neueste verletzt: %+v", events)
  }
}

func TestRedefLogRingOverflow(t *testing.T) {
  ClearRedefLog()
  for i := 0; i < redefLogSize+10; i++ {
    logRedef(RedefEvent{Name: fmt.Sprintf("n%d", i)})
  }
  events := RedefLog()
  if len(events) != redefLogSize {
    t.Fatalf("Ring muss bei %d kappen, got %d", redefLogSize, len(events))
  }
  if events[0].Name != "n10" {
    t.Fatalf("ältestes Event muss n10 sein, got %q", events[0].Name)
  }
  want := fmt.Sprintf("n%d", redefLogSize+9)
  if events[len(events)-1].Name != want {
    t.Fatalf("neuestes Event muss %q sein, got %q", want, events[len(events)-1].Name)
  }
}

func TestRedefLogReturnsCopy(t *testing.T) {
  ClearRedefLog()
  logRedef(RedefEvent{Name: "x"})
  events := RedefLog()
  events[0].Name = "mutiert"
  if RedefLog()[0].Name != "x" {
    t.Fatal("RedefLog muss eine Kopie liefern")
  }
}

func TestKindOf(t *testing.T) {
  cases := map[*Cell]string{
    {Type: FUNC}:   "func",
    {Type: LAMBDA}: "lambda",
    {Type: MACRO}:  "macro",
    {Type: NUMBER}: "value",
    {Type: ATOM}:   "value",
  }
  for c, want := range cases {
    if got := kindOf(c); got != want {
      t.Errorf("kindOf(%v) = %q, want %q", c.Type, got, want)
    }
  }
}
```

- [ ] **Step 2: Test laufen lassen, muss fehlschlagen**

Run: `go test ./lib/ -run 'TestRedefLog|TestKindOf' -count=1`
Expected: FAIL — `undefined: RedefEvent` (bzw. Kompilierfehler).

- [ ] **Step 3: Implementierung** — `lib/redeflog.go`:

```go
//**********************************************************************
//  lib/redeflog.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260724
//**********************************************************************
// Redef-Log: Ringpuffer aller Root-Redefinitionen und makunbound-Events.
// Beobachtbarkeit statt Verbot: der selbsterweiternde Pfad
// (eval (read (sigo ...))) bleibt erlaubt, aber nachvollziehbar.
//**********************************************************************

package lib

import "sync"

// RedefEvent beschreibt eine Root-Redefinition oder ein makunbound.
type RedefEvent struct {
  Name    string
  OldKind string // "func", "lambda", "macro", "value"
  NewKind string // "" bei makunbound
  OldFile string // "" = interaktiv (REPL/stdin/swank)
  OldLine int
  NewFile string // "" = Quelle unbekannt (Set-Hook) oder interaktiv
  NewLine int
  Action  string // "reload", "redef", "warn", "error", "makunbound"
}

const redefLogSize = 256

var (
  redefMu   sync.Mutex
  redefRing = make([]RedefEvent, 0, redefLogSize)
)

// logRedef haengt ein Event an; bei vollem Ring faellt das aelteste raus.
func logRedef(e RedefEvent) {
  redefMu.Lock()
  defer redefMu.Unlock()
  if len(redefRing) < redefLogSize {
    redefRing = append(redefRing, e)
    return
  }
  copy(redefRing, redefRing[1:])
  redefRing[redefLogSize-1] = e
}

// RedefLog liefert eine Kopie aller Events, aelteste zuerst.
func RedefLog() []RedefEvent {
  redefMu.Lock()
  defer redefMu.Unlock()
  out := make([]RedefEvent, len(redefRing))
  copy(out, redefRing)
  return out
}

// ClearRedefLog leert das Log (Tests).
func ClearRedefLog() {
  redefMu.Lock()
  defer redefMu.Unlock()
  redefRing = redefRing[:0]
}

// kindOf klassifiziert eine Bindung fuers Log.
func kindOf(c *Cell) string {
  switch c.Type {
  case FUNC:
    return "func"
  case LAMBDA:
    return "lambda"
  case MACRO:
    return "macro"
  }
  return "value"
}
```

- [ ] **Step 4: Test laufen lassen, muss grün sein**

Run: `go test ./lib/ -run 'TestRedefLog|TestKindOf' -count=1`
Expected: PASS (4 Tests).

- [ ] **Step 5: Commit**

```bash
git add lib/redeflog.go lib/redeflog_test.go
git commit -m "feat(redef): Redef-Log Ringpuffer (Go-API)

Co-Authored-By: kimi-k3 <noreply@moonshot.cn>"
```

---

### Task 2: Policy-Helfer + FUNC-Hook loggt

**Files:**
- Create: `lib/redefguard.go`
- Modify: `lib/env.go:99-110` (`defaultOnRootRedefine`)
- Test: `lib/redeflog_test.go` (erweitern)

**Interfaces:**
- Consumes: `logRedef`, `kindOf`, `RedefEvent` (Task 1); `LookupDefinition` (`lib/defloc.go:37`); `redefinePolicyAtomic`, `redefineWarn`, `redefineError` (`lib/env.go:48-58`).
- Produces: `applyRedefPolicy(name, detail string) error`, `policyAction() string` — konsumiert von Tasks 3+4. Fehlertext `REDEF: <name> (<detail>)` bleibt kompatibel zu bestehenden Tests (`lib/env_test.go:54` erwartet `REDEF: car`).

- [ ] **Step 1: Failing Test** — an `lib/redeflog_test.go` anhängen:

```go
func TestFuncRedefLogged(t *testing.T) {
  ClearRedefLog()
  withRedefinePolicy(t, "allow", func() {
    if _, err := evalStr("(define car 42)"); err != nil {
      t.Fatalf("allow-Policy muss still durchlassen: %v", err)
    }
  })
  events := RedefLog()
  if len(events) != 1 {
    t.Fatalf("1 Event erwartet, got %d (%+v)", len(events), events)
  }
  e := events[0]
  if e.Name != "car" || e.OldKind != "func" || e.NewKind != "value" || e.Action != "redef" {
    t.Fatalf("Event unerwartet: %+v", e)
  }
}

func TestFuncRedefErrorLogged(t *testing.T) {
  ClearRedefLog()
  withRedefinePolicy(t, "error", func() {
    if _, err := evalStr("(define car 42)"); err == nil {
      t.Fatal("error-Policy muss blockieren")
    }
  })
  events := RedefLog()
  if len(events) != 1 || events[0].Action != "error" {
    t.Fatalf("1 error-Event erwartet, got %+v", events)
  }
}
```

- [ ] **Step 2: Tests laufen lassen, müssen fehlschlagen**

Run: `go test ./lib/ -run 'TestFuncRedef' -count=1`
Expected: FAIL — Log ist leer (`1 Event erwartet, got 0`).

- [ ] **Step 3: Policy-Helfer** — `lib/redefguard.go`:

```go
//**********************************************************************
//  lib/redefguard.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260724
//**********************************************************************
// Gemeinsame Policy-Bausteine des Redefinition-Guards.
// Eine Policy-Auswertung fuer Env.Set-Hook, Kontext-Guard und makunbound.
//**********************************************************************

package lib

import (
  "fmt"
  "os"
)

// applyRedefPolicy bildet die Policy ab: allow → nil, warn → stderr + nil,
// error → Fehler (Redefinition wird abgebrochen).
// detail beschreibt den Kontext, z. B. "war FUNC".
func applyRedefPolicy(name, detail string) error {
  switch redefinePolicy(redefinePolicyAtomic.Load()) {
  case redefineWarn:
    fmt.Fprintf(os.Stderr, "REDEF: %s (%s)\n", name, detail)
  case redefineError:
    return fmt.Errorf("REDEF: %s (%s)", name, detail)
  }
  return nil
}

// policyAction liefert den Log-Action-Namen der aktuellen Policy.
func policyAction() string {
  switch redefinePolicy(redefinePolicyAtomic.Load()) {
  case redefineWarn:
    return "warn"
  case redefineError:
    return "error"
  }
  return "redef"
}
```

- [ ] **Step 4: Hook umbauen** — in `lib/env.go` `defaultOnRootRedefine` (Zeilen 99-110) ersetzen durch:

```go
func defaultOnRootRedefine(name string, old, new *Cell) error {
  loc, _ := LookupDefinition(name)
  ev := RedefEvent{
    Name:    name,
    OldKind: kindOf(old),
    NewKind: kindOf(new),
    OldFile: loc.File,
    OldLine: loc.Line,
    Action:  policyAction(),
  }
  err := applyRedefPolicy(name, "war FUNC")
  logRedef(ev)
  return err
}
```

Hinweis: `NewFile` bleibt hier `""` — der Set-Hook kennt die Quelle nicht
(bewusste Grenze, siehe Header). Die Policy-Meldung `REDEF: car (war FUNC)`
bleibt textidentisch, bestehende Tests (`lib/env_test.go:54,103`) laufen unverändert.

- [ ] **Step 5: Neue + alte Tests grün**

Run: `go test ./lib/ -run 'TestFuncRedef|TestRedefine|TestRedefLog' -count=1`
Expected: PASS. Dann volle Suite: `go test ./... -count=1` — PASS.

- [ ] **Step 6: Commit**

```bash
git add lib/redefguard.go lib/env.go lib/redeflog_test.go
git commit -m "refactor(redef): Policy-Helfer + FUNC-Events ins Log

Co-Authored-By: kimi-k3 <noreply@moonshot.cn>"
```

---

### Task 3: Kontext-Guard für LAMBDA/MACRO (define/defun/defmacro)

**Files:**
- Modify: `lib/redefguard.go` (`checkRootRedefine` dazu)
- Modify: `lib/eval_specialforms.go:19-31` (evalDefine), `:213-223` (evalDefun), `:338-349` (evalDefmacro)
- Test: `lib/redeflog_test.go` (erweitern)

**Interfaces:**
- Consumes: `applyRedefPolicy`, `policyAction` (Task 2); `LookupDefinition`; `Env.Root()` (`lib/env.go:214`).
- Produces: `checkRootRedefine(env *Env, name string, newVal *Cell, newFile string, newLine int) error`.

Regel: Alt-Bindung LAMBDA/MACRO und `DefLoc.File == newFile` → `reload` (still, immer erlaubt — Entwicklungs-Workflow). Quelle verschieden → Policy. FUNC wird hier bewusst NICHT behandelt (Set-Hook, Task 2). Nicht-Root-Env → sofort `nil`.

- [ ] **Step 1: Failing Tests** — an `lib/redeflog_test.go` anhängen:

```go
// captureStderr faengt os.Stderr fuer die Dauer von fn ab.
func captureStderr(t *testing.T, fn func()) string {
  t.Helper()
  r, w, err := os.Pipe()
  if err != nil {
    t.Fatalf("os.Pipe: %v", err)
  }
  old := os.Stderr
  os.Stderr = w
  fn()
  _ = w.Close()
  os.Stderr = old
  out, _ := io.ReadAll(r)
  return string(out)
}

func TestRedefSameSourceReloadSilent(t *testing.T) {
  ClearRedefLog()
  ClearDefinitions()
  out := captureStderr(t, func() {
    // Zwei defuns aus derselben Quelle (SrcFile "" bei evalStr):
    if _, err := evalStr("(defun foo (x) x) (defun foo (x) (cons x nil))"); err != nil {
      t.Fatalf("Reload derselben Quelle muss erlaubt sein: %v", err)
    }
  })
  if out != "" {
    t.Fatalf("Reload muss still bleiben, stderr: %q", out)
  }
  events := RedefLog()
  if len(events) != 1 || events[0].Action != "reload" || events[0].OldKind != "lambda" {
    t.Fatalf("1 reload-Event erwartet, got %+v", events)
  }
}

func TestRedefCrossFileWarns(t *testing.T) {
  ClearRedefLog()
  ClearDefinitions()
  withRedefinePolicy(t, "warn", func() {
    if _, err := evalStr("(defun bar (x) x)"); err != nil {
      t.Fatal(err)
    }
    RegisterDefinition("bar", "a.lisp", 12) // simuliert Herkunft aus Datei
    out := captureStderr(t, func() {
      // SrcFile "" ≠ "a.lisp" → fremde Quelle
      if _, err := evalStr("(defun bar (x) 42)"); err != nil {
        t.Fatalf("warn-Policy muss durchlassen: %v", err)
      }
    })
    if !strings.Contains(out, "REDEF: bar") {
      t.Fatalf("REDEF-Warnung erwartet, stderr: %q", out)
    }
    if !strings.Contains(out, "a.lisp") {
      t.Fatalf("Warnung muss alte Quelle nennen, stderr: %q", out)
    }
  })
  events := RedefLog()
  if len(events) != 1 || events[0].Action != "warn" || events[0].OldFile != "a.lisp" {
    t.Fatalf("warn-Event mit Quelle erwartet, got %+v", events)
  }
}

func TestRedefCrossFileErrorBlocks(t *testing.T) {
  ClearRedefLog()
  ClearDefinitions()
  withRedefinePolicy(t, "error", func() {
    if _, err := evalStr("(defun baz (x) x)"); err != nil {
      t.Fatal(err)
    }
    RegisterDefinition("baz", "a.lisp", 1)
    if _, err := evalStr("(defun baz (x) 99)"); err == nil ||
      !strings.Contains(err.Error(), "REDEF: baz") {
      t.Fatalf("REDEF-Fehler erwartet, got %v", err)
    }
    got, err := evalStr("(baz 5)")
    if err != nil || got.Num != 5 {
      t.Fatalf("alte Bindung muss erhalten bleiben: %v, %v", got, err)
    }
  })
}

func TestRedefValueOverLambdaWarns(t *testing.T) {
  ClearRedefLog()
  ClearDefinitions()
  withRedefinePolicy(t, "warn", func() {
    if _, err := evalStr("(defun qux (x) x)"); err != nil {
      t.Fatal(err)
    }
    RegisterDefinition("qux", "b.lisp", 3)
    out := captureStderr(t, func() {
      if _, err := evalStr("(define qux 5)"); err != nil {
        t.Fatal(err)
      }
    })
    if !strings.Contains(out, "REDEF: qux") {
      t.Fatalf("Wert-über-Funktion muss warnen, stderr: %q", out)
    }
  })
  events := RedefLog()
  if len(events) != 1 || events[0].OldKind != "lambda" || events[0].NewKind != "value" {
    t.Fatalf("Event lambda→value erwartet, got %+v", events)
  }
}

func TestRedefNonRootUntouched(t *testing.T) {
  ClearRedefLog()
  ClearDefinitions()
  // define im Lambda-Body schreibt in den lokalen Frame, nicht an Root:
  if _, err := evalStr("(defun mk () (define lokales-symbol 1)) (mk)"); err != nil {
    t.Fatal(err)
  }
  if events := RedefLog(); len(events) != 0 {
    t.Fatalf("lokale Defines duerfen nicht geloggt werden, got %+v", events)
  }
}
```

`os` und `io` zu den Imports von `lib/redeflog_test.go` dazu.

- [ ] **Step 2: Tests laufen lassen, müssen fehlschlagen**

Run: `go test ./lib/ -run 'TestRedef' -count=1`
Expected: FAIL — `TestRedefSameSourceReloadSilent` sieht 0 Events, `TestRedefCrossFileWarns` sieht leeres stderr.

- [ ] **Step 3: Guard implementieren** — an `lib/redefguard.go` anhängen:

```go
// checkRootRedefine wird von define/defun/defmacro VOR env.Set gerufen.
// Behandelt nur LAMBDA/MACRO-Altbindungen (Lisp-Definitionen); FUNC faengt
// der Hook in Env.Set ab. Reload aus derselben Quelle (DefLoc.File) ist
// immer erlaubt und still — das ist der normale Entwicklungs-Workflow.
func checkRootRedefine(env *Env, name string, newVal *Cell, newFile string, newLine int) error {
  if env != env.Root() {
    return nil
  }
  old, err := env.Get(name)
  if err != nil {
    return nil // nicht gebunden → Definition, keine Redefinition
  }
  if old.Type != LAMBDA && old.Type != MACRO {
    return nil
  }
  loc, _ := LookupDefinition(name)
  ev := RedefEvent{
    Name:    name,
    OldKind: kindOf(old),
    NewKind: kindOf(newVal),
    OldFile: loc.File,
    OldLine: loc.Line,
    NewFile: newFile,
    NewLine: newLine,
  }
  if loc.File == newFile {
    ev.Action = "reload"
    logRedef(ev)
    return nil
  }
  ev.Action = policyAction()
  logRedef(ev)
  detail := fmt.Sprintf("%s aus %s:%d, neu aus %s:%d",
    kindOf(old), loc.File, loc.Line, newFile, newLine)
  return applyRedefPolicy(name, detail)
}
```

- [ ] **Step 4: Verdrahten** — in `lib/eval_specialforms.go` je eine Zeile VOR `env.Set`:

In `evalDefine` (vor Zeile 28 `if err := env.Set(name, val); …`):
```go
  if err := checkRootRedefine(env, name, val, form.SrcFile, form.SrcLine); err != nil { return nil, err }
```
In `evalDefun` (vor `if err := env.Set(name, lam); …`):
```go
  if err := checkRootRedefine(env, name, lam, form.SrcFile, form.SrcLine); err != nil { return nil, err }
```
In `evalDefmacro` (vor `if err := env.Set(name, lam); …`):
```go
  if err := checkRootRedefine(env, name, lam, form.SrcFile, form.SrcLine); err != nil { return nil, err }
```

- [ ] **Step 5: Tests grün + Vollsuite**

Run: `go test ./lib/ -run 'TestRedef' -count=1` — PASS.
Dann: `go test ./... -count=1` — PASS.
Plus Stdlib-Rauchtest (darf keine REDEF-Zeile auf stderr produzieren):
Run: `./build/golisp2 -e "(+ 1 1)" 2>&1 >/dev/null` — Expected: leere Ausgabe.
(Falls doch: stdlib/swank definieren ein Symbol doppelt aus verschiedenen Quellen — Fund prüfen, nicht Policy aufweichen.)

- [ ] **Step 6: Commit**

```bash
git add lib/redefguard.go lib/eval_specialforms.go lib/redeflog_test.go
git commit -m "feat(redef): Kontext-Guard fuer LAMBDA/MACRO mit DefLoc-Quelle

Reload derselben Datei bleibt still; fremde Quelle folgt der Policy.

Co-Authored-By: kimi-k3 <noreply@moonshot.cn>"
```

---

### Task 4: makunbound (Spezialform)

**Files:**
- Modify: `lib/defloc.go` (`RemoveDefinition` dazu)
- Modify: `lib/env.go` (`UnsetRoot` dazu, hinter `Root()` ~Zeile 227)
- Modify: `lib/eval_specialforms.go` (`evalMakunbound` dazu, hinter `evalBound` ~Zeile 112)
- Modify: `lib/eval_core.go:145` (case hinter `"bound?"`)
- Test: `lib/redeflog_test.go` (erweitern)

**Interfaces:**
- Consumes: `applyRedefPolicy` (Task 2), `logRedef`, `kindOf` (Task 1), `evalWithCtx` (`lib/eval_core.go:78`).
- Produces: `RemoveDefinition(name string)`, `(*Env).UnsetRoot(name string) (*Cell, bool)`, Spezialform `(makunbound 'symbol)`.

Semantik: Argument wird ausgewertet (wie `bound?`), muss ATOM sein. Nur Root-Bindung. Ungebunden → Fehler (laut statt still). FUNC/LAMBDA/MACRO → Policy (error blockiert das Entfernen). Entfernt auch den DefLoc-Eintrag und loggt `makunbound`.

- [ ] **Step 1: Failing Tests** — an `lib/redeflog_test.go` anhängen:

```go
func TestMakunbound(t *testing.T) {
  ClearRedefLog()
  ClearDefinitions()
  // defun + makunbound + bound? in EINEM evalStr: evalStr baut pro Aufruf
  // ein frisches Env, Bindungen ueberleben keinen zweiten Aufruf.
  got, err := evalStr("(defun mkb (x) x) (makunbound 'mkb) (bound? 'mkb)")
  if err != nil {
    t.Fatal(err)
  }
  if IsTruthy(got) {
    t.Fatal("mkb muss nach makunbound ungebunden sein")
  }
  if _, ok := LookupDefinition("mkb"); ok {
    t.Fatal("DefLoc-Eintrag muss entfernt sein")
  }
  events := RedefLog()
  if len(events) != 1 || events[0].Action != "makunbound" || events[0].OldKind != "lambda" {
    t.Fatalf("makunbound-Event erwartet, got %+v", events)
  }
}

func TestMakunboundUnboundError(t *testing.T) {
  _, err := evalStr("(makunbound 'gibts-nicht)")
  if err == nil || !strings.Contains(err.Error(), "nicht gebunden") {
    t.Fatalf("Fehler fuer ungebundenes Symbol erwartet, got %v", err)
  }
}

func TestMakunboundFuncErrorPolicy(t *testing.T) {
  withRedefinePolicy(t, "error", func() {
    if _, err := evalStr("(makunbound 'car)"); err == nil ||
      !strings.Contains(err.Error(), "REDEF: car") {
      t.Fatalf("error-Policy muss makunbound auf FUNC blockieren, got %v", err)
    }
    got, err := evalStr("(car '(1 2))")
    if err != nil || got.Num != 1 {
      t.Fatalf("car muss erhalten bleiben: %v, %v", got, err)
    }
  })
}

func TestMakunboundFuncAllow(t *testing.T) {
  ClearRedefLog()
  withRedefinePolicy(t, "allow", func() {
    // ebenfalls in einem evalStr (frisches Env pro Aufruf, siehe oben)
    got, err := evalStr("(makunbound 'cdr) (bound? 'cdr)")
    if err != nil {
      t.Fatal(err)
    }
    if IsTruthy(got) {
      t.Fatal("cdr muss entfernt sein")
    }
  })
  events := RedefLog()
  if len(events) != 1 || events[0].OldKind != "func" || events[0].Action != "makunbound" {
    t.Fatalf("func-makunbound-Event erwartet, got %+v", events)
  }
}
```

- [ ] **Step 2: Tests laufen lassen, müssen fehlschlagen**

Run: `go test ./lib/ -run 'TestMakunbound' -count=1`
Expected: FAIL — `env: unbekanntes Symbol 'makunbound'`.

- [ ] **Step 3: Implementierung**

`lib/defloc.go` anhängen:
```go
// RemoveDefinition entfernt den Registry-Eintrag (makunbound).
func RemoveDefinition(name string) {
  defMu.Lock()
  defer defMu.Unlock()
  delete(definitions, name)
}
```

`lib/env.go` hinter `Root()` anhängen:
```go
// UnsetRoot entfernt eine Bindung aus dem Root-Env.
// Liefert die entfernte Zelle; ok=false wenn nicht gebunden oder kein Root.
func (e *Env) UnsetRoot(name string) (*Cell, bool) {
  e.mu.Lock()
  defer e.mu.Unlock()
  if e.parent != nil {
    return nil, false
  }
  old, ok := e.vars[name]
  if !ok {
    return nil, false
  }
  delete(e.vars, name)
  return old, true
}
```

`lib/eval_specialforms.go` hinter `evalBound` einfügen:
```go
// makunbound: (makunbound 'symbol) → entfernt die globale Bindung (CL).
// Fehler, wenn das Symbol nicht gebunden ist — laut statt still.
func evalMakunbound(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil {
    return nil, fmt.Errorf("makunbound: Syntax: (makunbound 'symbol)")
  }
  sym, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil {
    return nil, err
  }
  if sym.Type != ATOM {
    return nil, fmt.Errorf("makunbound: Argument muss ein Symbol sein")
  }
  root := env.Root()
  old, err := root.Get(sym.Val)
  if err != nil {
    return nil, fmt.Errorf("makunbound: '%s' ist nicht gebunden", sym.Val)
  }
  if old.Type == FUNC || old.Type == LAMBDA || old.Type == MACRO {
    if err := applyRedefPolicy(sym.Val, "makunbound auf "+kindOf(old)); err != nil {
      return nil, err
    }
  }
  _, _ = root.UnsetRoot(sym.Val)
  RemoveDefinition(sym.Val)
  logRedef(RedefEvent{Name: sym.Val, OldKind: kindOf(old), Action: "makunbound"})
  return sym, nil
}
```

`lib/eval_core.go` Zeile 145, hinter `case "bound?":` einfügen:
```go
      case "makunbound":  return evalMakunbound(expr.Cdr, env, ectx)
```

- [ ] **Step 4: Tests grün + Vollsuite**

Run: `go test ./lib/ -run 'TestMakunbound' -count=1` — PASS.
Dann: `go test ./... -count=1` — PASS.

- [ ] **Step 5: Commit**

```bash
git add lib/defloc.go lib/env.go lib/eval_specialforms.go lib/eval_core.go lib/redeflog_test.go
git commit -m "feat(cl-compat): makunbound-Spezialform mit Policy und Log

Co-Authored-By: kimi-k3 <noreply@moonshot.cn>"
```

---

### Task 5: Lisp-Zugang `(redef-log)` / `(redef-log-clear)`

**Files:**
- Modify: `lib/redeflog.go` (`fnRedefLog`, `fnRedefLogClear` dazu)
- Modify: `lib/primitives.go:127` (Registrierung hinter `redefine-policy`; Tabs!)
- Test: `lib/redeflog_test.go` (erweitern)

**Interfaces:**
- Consumes: `RedefLog`, `ClearRedefLog` (Task 1); `List`, `MakeAtom`, `MakeString`, `MakeNumber` (`lib/types_helpers.go`); `makeFn` (`lib/primitives.go:135`).
- Produces: Primitiven `redef-log`, `redef-log-clear`. Event-Format: `(name old-kind new-kind old-file old-line new-file new-line action)`.

- [ ] **Step 1: Failing Test** — an `lib/redeflog_test.go` anhängen:

```go
func TestRedefLogPrimitive(t *testing.T) {
  ClearRedefLog()
  ClearDefinitions()
  withRedefinePolicy(t, "allow", func() {
    if _, err := evalStr("(define car 42)"); err != nil {
      t.Fatal(err)
    }
    // Name des ersten (einzigen) Events:
    got, err := evalStr("(car (car (redef-log)))")
    if err != nil {
      t.Fatal(err)
    }
    if got.Type != ATOM || got.Val != "car" {
      t.Fatalf("Event-Name car erwartet, got %v", got)
    }
    if _, err := evalStr("(redef-log-clear)"); err != nil {
      t.Fatal(err)
    }
    empty, err := evalStr("(redef-log)")
    if err != nil {
      t.Fatal(err)
    }
    if empty.Type != NIL {
      t.Fatalf("leeres Log muss nil sein, got %v", empty)
    }
  })
}
```

- [ ] **Step 2: Test laufen lassen, muss fehlschlagen**

Run: `go test ./lib/ -run 'TestRedefLogPrimitive' -count=1`
Expected: FAIL — `env: unbekanntes Symbol 'redef-log'`.

- [ ] **Step 3: Implementierung** — an `lib/redeflog.go` anhängen (`fmt` zum Import dazu):

```go
// redef-log: (redef-log) → Liste der Events, aelteste zuerst.
// Jedes Event: (name old-kind new-kind old-file old-line new-file new-line action)
func fnRedefLog(args []*Cell) (*Cell, error) {
  if len(args) != 0 {
    return nil, fmt.Errorf("redef-log: Syntax: (redef-log)")
  }
  events := RedefLog()
  cells := make([]*Cell, 0, len(events))
  for _, e := range events {
    cells = append(cells, List(
      MakeAtom(e.Name),
      MakeAtom(e.OldKind),
      MakeAtom(e.NewKind),
      MakeString(e.OldFile),
      MakeNumber(float64(e.OldLine)),
      MakeString(e.NewFile),
      MakeNumber(float64(e.NewLine)),
      MakeAtom(e.Action),
    ))
  }
  return List(cells...), nil
}

// redef-log-clear: (redef-log-clear) → leert das Log, liefert nil.
func fnRedefLogClear(args []*Cell) (*Cell, error) {
  if len(args) != 0 {
    return nil, fmt.Errorf("redef-log-clear: Syntax: (redef-log-clear)")
  }
  ClearRedefLog()
  return MakeNil(), nil
}
```

`lib/primitives.go` hinter Zeile 127 (`redefine-policy`-Registrierung), **mit Tabs**:
```go
	_ = env.Set("redef-log", makeFn(fnRedefLog))
	_ = env.Set("redef-log-clear", makeFn(fnRedefLogClear))
```

- [ ] **Step 4: Test grün + Vollsuite**

Run: `go test ./lib/ -run 'TestRedefLogPrimitive' -count=1` — PASS.
Dann: `go test ./... -count=1` — PASS.
Manueller Smoke: `./build.sh && echo '(define car 5)(println (redef-log))' | ./build/golisp2` — Expected: Liste mit einem car-Event auf stdout.

- [ ] **Step 5: Commit**

```bash
git add lib/redeflog.go lib/primitives.go lib/redeflog_test.go
git commit -m "feat(redef): redef-log und redef-log-clear Primitiven

Co-Authored-By: kimi-k3 <noreply@moonshot.cn>"
```

---

### Task 6: Integration, Doku, TODO abhaken

**Files:**
- Modify: `doc/lisp-semantik.md` (neuer Abschnitt)
- Modify: `TODO.md` (abhaken; Archivierung nach Projekt-Muster)

- [ ] **Step 1: Volle Verifikation**

```bash
./build.sh
go test ./... -count=1
./build/golisp2 -t
./build/golisp2 -e "(+ 1 1)" 2>&1 >/dev/null        # kein REDEF beim Start
echo '(defun f (x) x)(defun f (x) 2)(redef-log)' | ./build/golisp2   # reload-Event
```

Expected: Build ok, alle Tests grün, Testsuite grün, stderr beim Start leer,
letztes Kommando zeigt `(((f lambda lambda "" 1 "" 2 reload)))`-ähnliche Ausgabe.

- [ ] **Step 2: Doku** — in `doc/lisp-semantik.md` neuen Abschnitt anfügen:

```markdown
## Redefinition, Redef-Log, makunbound

Das Root-Env bewacht Redefinitionen über `(redefine-policy 'allow|'warn|'error)`
(Default: `warn`):

- **FUNC** (Go-Primitiv) überschreiben → immer Policy, egal aus welcher Quelle.
- **LAMBDA/MACRO** (Lisp-Definition) überschreiben → Policy nur bei *fremder*
  Quelle. Reload derselben Datei (gleiches `SrcFile`, interaktiv = `""`) ist
  still — das ist der normale Entwicklungs-Workflow.
- Alle Redefinitionen landen im Ringpuffer (256 Events), abfragbar via
  `(redef-log)` — Event-Format:
  `(name old-kind new-kind old-file old-line new-file new-line action)`.
  `(redef-log-clear)` leert.
- `(makunbound 'sym)` entfernt eine Root-Bindung samt DefLoc-Eintrag.
  Fehler bei ungebundenem Symbol; auf FUNC/LAMBDA/MACRO greift die Policy.

Bewusste Grenzen: `setq`/`progv` am Root über LAMBDA bleiben still (kein
Quell-Kontext). FUNC-Log-Events kennen die neue Quelle nicht (`NewFile ""`).
```

- [ ] **Step 3: TODO abhaken** — `TODO.md` Aufgabe als erledigt markieren und nach Projekt-Muster archivieren (vgl. Commit `d14d18b`: `todos/TODO.md-<datum>-done`).

- [ ] **Step 4: Commit**

```bash
git add doc/lisp-semantik.md TODO.md todos/
git commit -m "doc(redef): Semantik-Doku + TODO 20260724 abgehakt

Co-Authored-By: kimi-k3 <noreply@moonshot.cn>"
```
