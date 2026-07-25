# defsystem-lite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deklarative Systemdefinition (`defsystem`) + dependency-geordnetes, idempotentes Laden (`load-system`) für GoLisp2, gemäß Spec `docs/superpowers/specs/2026-07-25-defsystem-lite-design.md`.

**Architecture:** Reines Lisp (`embed/defsystem.lisp`, ~150–200 Zeilen) auf vorhandener Infrastruktur (DefLoc-Registry, `makunbound`, Redef-Log, `load`, `get-file-path`). Einziger Go-Zuwachs: Primitiv `(defined-in "pfad")` in `lib/defloc.go`. Geladen via `LoadStdlib` (einzige Quelle).

**Tech Stack:** Go (Interpreter-Kern), GoLisp2-Lisp (Stdlib-Erweiterung), `assert=` aus `tests/test-helpers.lisp` für Lisp-Tests.

## Global Constraints

- Build immer `./build.sh` (baut nach `./build/`); Ground Truth: `go build ./...` + `go test ./...`
- Einrückung 2 Spaces, keine Tabs; Kommentare deutsch, sparsam
- Datei-Header: Autor Gerhard Quell, CoAutor kimi-k3, Copyright 2026, Erstellt 20260725
- Fehlerformat Go: `fmt.Errorf("funktionsname: beschreibung")`
- Kein zweites Registry neben DefLoc; `LoadStdlib` bleibt einzige Stdlib-Quelle
- Fixtures definieren nur Symbole mit Präfix `fx-` / Systeme `fx-sys-*` (keine Kollision mit Bestand)
- Commit pro Task; Message mit `Co-Authored-By: kimi-k3 <noreply@anthropic.com>`
- `tmp/` statt `/tmp`, falls temporäre Dateien nötig

## Verifizierte Fakten (Implementierung darauf aufbauen)

- `load` speichert `SrcFile` als **absoluten** Pfad (`filepath.Abs` nach `resolvePath`, `lib/eval_load.go:33`)
- `makunbound` und `bound?` **evaluieren** ihr Argument — Variable direkt nutzbar: `(makunbound s)`
- `makunbound` auf LAMBDA unter Default-Policy `warn` **gibt Warnung aus** (`REDEF: sym (makunbound auf lambda)`) → unload braucht Policy-Hülle
- `(trap form (lambda (e) ...))` fängt `error`; `catch` fängt nur `throw`
- Stdlib-Helfer vorhanden: `member assoc alist-set alist-get filter mapcar every any when unless dolist push cddr reverse append`
- `get-file-path` (lib/fileio.go:58) = `resolvePath` von Lisp aus; **ohne** `filepath.Abs`
- `golisp2 -t`: hardcodierte `test(env, ...)`-Liste in `main.go` (~Zeile 95–106); neue Testdatei dort eintragen
- `evalStr` Helfer für Go-Tests: `lib/eval_test.go:25`
- `MakeAtom(name string) *Cell` erzeugt Symbol; `List(cells ...*Cell) *Cell` baut Liste (`lib/types.go:72`, `lib/types_helpers.go:14`)

---

### Task 1: Go-Primitiv `defined-in`

**Files:**
- Modify: `lib/defloc.go` (Primitiv am Ende anfügen)
- Modify: `lib/primitives.go` (Registrierung nach Zeile 129, bei `redef-log-clear`)
- Test: `lib/defloc_test.go`

**Interfaces:**
- Produces: Lisp-Primitiv `(defined-in "pfad")` → sortierte Symbolliste (ATOM-Zellen). Normalisiert Argument wie `load` (`resolvePath` + `filepath.Abs`). Fehler bei nicht-existierendem Pfad. Konsumiert von Task 5+6 (`system-symbols`, `unload-system`).

- [ ] **Step 1: Failing Tests schreiben** — an `lib/defloc_test.go` anhängen:

```go
func TestDefinedInPrimitive(t *testing.T) {
  ClearDefinitions()
  dir := t.TempDir()
  f := dir + "/mod.lisp"
  if err := os.WriteFile(f, []byte("(defun dummy () 1)"), 0644); err != nil {
    t.Fatal(err)
  }
  RegisterDefinition("alpha", f, 3)
  RegisterDefinition("beta", f, 9)
  RegisterDefinition("gamma", "/irgendwo/anders.lisp", 1)

  got, err := fnDefinedIn([]*Cell{MakeStr(f)})
  if err != nil {
    t.Fatalf("defined-in: %v", err)
  }
  // sortiert: alpha vor beta
  want := List(MakeAtom("alpha"), MakeAtom("beta"))
  if got.String() != want.String() {
    t.Fatalf("got %s, want %s", got.String(), want.String())
  }
}

func TestDefinedInNoMatch(t *testing.T) {
  ClearDefinitions()
  dir := t.TempDir()
  f := dir + "/leer.lisp"
  if err := os.WriteFile(f, []byte(""), 0644); err != nil {
    t.Fatal(err)
  }
  got, err := fnDefinedIn([]*Cell{MakeStr(f)})
  if err != nil {
    t.Fatalf("defined-in: %v", err)
  }
  if got.String() != MakeNil().String() {
    t.Fatalf("leere Liste erwartet, got %s", got.String())
  }
}

func TestDefinedInMissingFile(t *testing.T) {
  if _, err := fnDefinedIn([]*Cell{MakeStr("/gibts/garantiert/nicht.lisp")}); err == nil {
    t.Fatal("Fehler bei nicht-existierendem Pfad erwartet")
  }
}

func TestDefinedInRegistered(t *testing.T) {
  // Registrierung in BaseEnv: darf nicht "unbekanntes Symbol" melden.
  // (CWD von go test ist lib/ — "defloc.go" existiert dort.)
  if _, err := evalStr(`(defined-in "defloc.go")`); err != nil &&
    strings.Contains(err.Error(), "unbekanntes Symbol") {
    t.Fatalf("defined-in nicht in BaseEnv registriert: %v", err)
  }
}
```

Import-Block von `lib/defloc_test.go` um `"os"` und `"strings"` erweitern.

- [ ] **Step 2: Tests laufen lassen, Fehlschlag verifizieren**

Run: `go test ./lib/ -run TestDefinedIn -count=1`
Expected: FAIL — `undefined: fnDefinedIn`

- [ ] **Step 3: Implementierung** — an `lib/defloc.go` anhängen:

```go
// defined-in: (defined-in "pfad") → sortierte Liste der Symbole, deren
// DefLoc.File dem normalisierten Pfad entspricht. Normalisierung identisch
// zu load (resolvePath + filepath.Abs), damit relative Angaben matchen.
// Leere Liste, wenn die Datei nichts definiert hat.
func fnDefinedIn(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("defined-in: 1 Argument nötig")
  }
  if args[0].Type != STRING {
    return nil, fmt.Errorf("defined-in: String erwartet")
  }
  resolved, err := resolvePath(args[0].Val)
  if err != nil {
    return nil, fmt.Errorf("defined-in: %v", err)
  }
  if abs, aerr := filepath.Abs(resolved); aerr == nil {
    resolved = abs
  }
  defMu.RLock()
  names := []string{}
  for name, loc := range definitions {
    if loc.File == resolved {
      names = append(names, name)
    }
  }
  defMu.RUnlock()
  sort.Strings(names)
  cells := make([]*Cell, len(names))
  for i, n := range names {
    cells[i] = MakeAtom(n)
  }
  return List(cells...), nil
}
```

Import-Block von `lib/defloc.go`: `"fmt"`, `"path/filepath"`, `"sort"`, `"sync"` (sync schon da).

Registrierung in `lib/primitives.go` nach der `redef-log-clear`-Zeile (~129):

```go
	_ = env.Set("defined-in", makeFn(fnDefinedIn))
```

(Einrückung dort mit Tabs — vorhandenen Stil der Umgebungszeilen kopieren.)

- [ ] **Step 4: Tests bestehen**

Run: `go test ./lib/ -run TestDefinedIn -count=1 && go test ./... -count=1`
Expected: PASS (gesamte Suite grün)

- [ ] **Step 5: Commit**

```bash
git add lib/defloc.go lib/defloc_test.go lib/primitives.go
git commit -m "feat(defloc): Primitiv defined-in — Lisp-Zugriff auf DefLoc-Registry

Co-Authored-By: kimi-k3 <noreply@anthropic.com>"
```

---

### Task 2: Embed-Wiring + defsystem.lisp Skeleton

**Files:**
- Create: `embed/defsystem.lisp` (Skeleton: Header + Root-Variablen)
- Modify: `embed/assets.go`
- Modify: `lib/stdlib.go`

**Interfaces:**
- Produces: `*systems*` (Alist), `*loaded-files*` (Liste) als Root-Bindungen, in jeder Env nach `LoadStdlib` verfügbar. Konsumiert von Tasks 3–6.

- [ ] **Step 1: Skeleton-Datei**

`embed/defsystem.lisp`:

```lisp
;; ********************************************************************
;; defsystem.lisp – deklarative Systemdefinition + idempotentes Laden
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k3
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260725
;; ********************************************************************
;; defsystem-lite: ASDF-orientiert, keine ASDF-Kompatibilität.
;; Design: docs/superpowers/specs/2026-07-25-defsystem-lite-design.md
;; Fundament: DefLoc-Registry (defined-in), makunbound, Redef-Log.
;; ********************************************************************

;; *systems*: Alist (name . plist) mit plist = (:depends-on (sym…) :components ("pfad"…))
;; *loaded-files*: normalisierte Pfade (via get-file-path) bereits geladener Dateien.
;; Idempotenz auf Datei-Ebene: zwei Systeme dürfen dieselbe Datei listen.
(define *systems* '())
(define *loaded-files* '())
```

- [ ] **Step 2: Embed-Deklaration** — `embed/assets.go` an die `Swank`-Deklaration anhängen:

```go
//go:embed defsystem.lisp
var Defsystem string
```

- [ ] **Step 3: LoadStdlib erweitern** — `lib/stdlib.go`, Body von `LoadStdlib`:

```go
func LoadStdlib(env *Env) error {
  if _, err := LoadString(assets.Stdlib, env); err != nil {
    return err
  }
  _, err := LoadString(assets.Defsystem, env)
  return err
}
```

Kommentar über `LoadStdlib` anpassen: „lädt die eingebettete Standardbibliothek (stdlib + defsystem)".

- [ ] **Step 4: Build + Smoke-Test**

Run: `./build.sh && ./build/golisp2 -e "(println (list (bound? '*systems*) (bound? '*loaded-files*)))"`
Expected: `(t t)`

- [ ] **Step 5: Gesamte Testsuite bleibt grün**

Run: `go test ./... -count=1 && ./build/golisp2 -t`
Expected: PASS / Exit 0

- [ ] **Step 6: Commit**

```bash
git add embed/defsystem.lisp embed/assets.go lib/stdlib.go
git commit -m "feat(defsystem): Embed-Wiring — defsystem.lisp via LoadStdlib

Co-Authored-By: kimi-k3 <noreply@anthropic.com>"
```

---

### Task 3: `defsystem`-Makro + Validierung + Test-Grundgerüst

**Files:**
- Modify: `embed/defsystem.lisp`
- Create: `tests/defsystem-tests.lisp`
- Create: `tests/fixtures/fx-a.lisp`, `fx-b.lisp`, `fx-c.lisp`, `fx-shared.lisp`, `fx-u.lisp`
- Modify: `main.go` (Testdatei in `-t`-Suite eintragen, nach der `stdlib-test.lisp`-Zeile ~97)

**Interfaces:**
- Produces: `(defsystem name :depends-on (sym…) :components ("pfad"…))` — Makro, validiert zur Expansionszeit, aktualisiert `*systems*`. `%sys-entry`, `%sys-get` — interne Zugriffshelfer für Tasks 4–6.
- Consumes: `*systems*` aus Task 2.

- [ ] **Step 1: Fixtures anlegen**

`tests/fixtures/fx-a.lisp`:

```lisp
;; Fixture für defsystem-Tests — nur Symbole mit fx-Präfix!
(defun fx-a () 'a)
```

`tests/fixtures/fx-b.lisp`:

```lisp
;; Fixture für defsystem-Tests — nur Symbole mit fx-Präfix!
(defun fx-b () 'b)
```

`tests/fixtures/fx-c.lisp`:

```lisp
;; Fixture für defsystem-Tests — nur Symbole mit fx-Präfix!
(defun fx-c () 'c)
```

`tests/fixtures/fx-shared.lisp`:

```lisp
;; Fixture für defsystem-Tests — nur Symbole mit fx-Präfix!
;; Wird von zwei Systemen gemeinsam gelistet (Shared-File-Fall).
(defun fx-shared () 'shared)
```

`tests/fixtures/fx-u.lisp` (wird **nie** geladen — Referenz für „nicht geladen"):

```lisp
;; Fixture für defsystem-Tests — nur Symbole mit fx-Präfix!
;; Wird absichtlich nie geladen: Referenz für loaded-systems/unload-no-op.
(defun fx-u () 'u)
```

- [ ] **Step 2: Failing Tests schreiben** — `tests/defsystem-tests.lisp`:

```lisp
;; ********************************************************************
;; tests/defsystem-tests.lisp — Tests für defsystem-lite
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k3
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260725
;; ********************************************************************
;; Fixtures unter tests/fixtures/. Aufräumen am Ende: unload + Registry
;; zurücksetzen, damit nachfolgende Suites unberührt bleiben.
;; ********************************************************************

(load "tests/test-helpers.lisp")  ; assert=

;; --- defsystem: Registrierung ---------------------------------------
(defsystem fx-sys-c :components ("tests/fixtures/fx-c.lisp"))
(defsystem fx-sys-b :depends-on (fx-sys-c) :components ("tests/fixtures/fx-b.lisp"))
(defsystem fx-sys-a :depends-on (fx-sys-b) :components ("tests/fixtures/fx-a.lisp"))

(assert= t (if (assoc 'fx-sys-a *systems*) t ()))
(assert= '(fx-sys-c) (%sys-get (%sys-entry 'fx-sys-b) :depends-on))
(assert= '("tests/fixtures/fx-a.lisp") (%sys-get (%sys-entry 'fx-sys-a) :components))
;; :depends-on optional, Default ()
(assert= '() (%sys-get (%sys-entry 'fx-sys-c) :depends-on))

;; --- defsystem: Validierung zur Expansionszeit ----------------------
(assert= 'err (trap (eval '(defsystem bad1 :falsch ())) (lambda (e) 'err)))
(assert= 'err (trap (eval '(defsystem bad2 :depends-on ("string-statt-symbol"))) (lambda (e) 'err)))
(assert= 'err (trap (eval '(defsystem bad3 :components (symbol-statt-string))) (lambda (e) 'err)))
```

- [ ] **Step 3: Test in `-t`-Suite eintragen** — `main.go`, nach der `stdlib-test.lisp`-Zeile:

```go
  // defsystem-lite: Systemdefinition, topo-Laden, unload
  test(env, `(load "tests/defsystem-tests.lisp")`)
```

- [ ] **Step 4: Fehlschlag verifizieren**

Run: `./build.sh && ./build/golisp2 tests/defsystem-tests.lisp`
Expected: ERR — `unbekanntes Symbol 'defsystem'` (bzw. `%sys-entry`)

- [ ] **Step 5: Implementierung** — an `embed/defsystem.lisp` anhängen:

```lisp
;; === Interne Zugriffshelfer =========================================

;; %sys-get: Wert aus Property-Liste ((:k1 v1 :k2 v2) :k1) → v1, sonst ()
(defun %sys-get (plist key)
  (cond ((null plist)                '())
        ((equal? (car plist) key)    (cadr plist))
        (t                           (%sys-get (cddr plist) key))))

;; %sys-entry: System-Plist aus *systems*, Fehler wenn unbekannt
(defun %sys-entry (name)
  (let ((e (assoc name *systems*)))
    (if (null e)
        (error (format nil "defsystem: unbekanntes System '~a'" name))
        (cadr e))))

;; === defsystem =======================================================

;; (defsystem name :depends-on (sym…) :components ("pfad"…))
;; Validiert zur Expansionszeit; Neudefinition ersetzt still (Reload-Semantik).
(defmacro defsystem (name &rest kvs)
  (let ((deps '())
        (comps '())
        (rest-kv kvs))
    (while (not (null rest-kv))
      (let ((k (car rest-kv))
            (v (cadr rest-kv)))
        (cond ((equal? k :depends-on) (set! deps v))
              ((equal? k :components) (set! comps v))
              (t (error (format nil "defsystem: unbekanntes Keyword ~a" k))))
        (set! rest-kv (cddr rest-kv))))
    (if (not (every (lambda (s) (symbol? s)) deps))
        (error "defsystem: :depends-on muss Symbolliste sein"))
    (if (not (every (lambda (s) (string? s)) comps))
        (error "defsystem: :components muss Stringliste sein"))
    `(set! *systems*
           (alist-set ',name (list :depends-on ',deps :components ',comps) *systems*))))
```

- [ ] **Step 6: Tests bestehen**

Run: `./build.sh && ./build/golisp2 tests/defsystem-tests.lisp && ./build/golisp2 -t`
Expected: PASS-Zeilen, Exit 0

- [ ] **Step 7: Commit**

```bash
git add embed/defsystem.lisp tests/defsystem-tests.lisp tests/fixtures/ main.go
git commit -m "feat(defsystem): defsystem-Makro mit Expansionszeit-Validierung

Co-Authored-By: kimi-k3 <noreply@anthropic.com>"
```

---

### Task 4: `load-system` — Topo-Sort, Zyklus, Idempotenz

**Files:**
- Modify: `embed/defsystem.lisp`
- Test: `tests/defsystem-tests.lisp`

**Interfaces:**
- Produces: `(load-system name)` → Topo-Liste der geladenen Systeme (Deps zuerst). `%topo` intern. Konsumiert `%sys-entry`/`%sys-get` aus Task 3.

- [ ] **Step 1: Failing Tests anhängen** — an `tests/defsystem-tests.lisp`:

```lisp
;; --- load-system: Topo-Reihenfolge + Idempotenz ----------------------
(load-system 'fx-sys-a)
(assert= t (bound? 'fx-a))
(assert= t (bound? 'fx-b))
(assert= t (bound? 'fx-c))
;; Deps zuerst geladen; push präpendiert → zuletzt geladenes oben
(assert= (list (get-file-path "tests/fixtures/fx-a.lisp")
               (get-file-path "tests/fixtures/fx-b.lisp")
               (get-file-path "tests/fixtures/fx-c.lisp"))
         *loaded-files*)
;; Idempotenz: zweiter Aufruf lädt nichts nach
(let ((n (length *loaded-files*)))
  (load-system 'fx-sys-a)
  (assert= n (length *loaded-files*)))

;; --- load-system: Fehlerfälle ----------------------------------------
(assert= 'err (trap (load-system 'gibts-nicht) (lambda (e) 'err)))
(defsystem fx-cy-a :depends-on (fx-cy-b) :components ())
(defsystem fx-cy-b :depends-on (fx-cy-a) :components ())
(assert= 'err (trap (load-system 'fx-cy-a) (lambda (e) 'err)))
```

- [ ] **Step 2: Fehlschlag verifizieren**

Run: `./build/golisp2 tests/defsystem-tests.lisp`
Expected: ERR — `unbekanntes Symbol 'load-system'`

- [ ] **Step 3: Implementierung** — an `embed/defsystem.lisp` anhängen:

```lisp
;; === load-system =====================================================

;; %topo: DFS mit Zykluserkennung. visiting = aktueller DFS-Stack,
;; done = fertige Systeme. Rückgabe: done mit name oben (Deps darunter).
(defun %topo (name visiting done)
  (cond
    ((member name visiting)
     (error (format nil "defsystem: Abhängigkeitszyklus: ~a"
                    (reverse (cons name visiting)))))
    ((member name done) done)
    (t
     (let ((d done)
           (vis2 (cons name visiting)))
       (dolist (dep (%sys-get (%sys-entry name) :depends-on))
         (set! d (%topo dep vis2 d)))
       (cons name d)))))

;; (load-system 'name) → Topo-Liste der beteiligten Systeme (Deps zuerst).
;; Idempotent auf Datei-Ebene: bereits geladene Komponenten werden
;; übersprungen. Fehler in einer Komponente bricht ab; der Teilzustand
;; bleibt stehen und ein erneuter Aufruf setzt an der Fehlerstelle fort.
(defun load-system (name)
  (let ((order (reverse (%topo name '() '()))))
    (dolist (sys order)
      (dolist (comp (%sys-get (%sys-entry sys) :components))
        (let ((norm (get-file-path comp)))
          (unless (member norm *loaded-files*)
            (load comp)
            (push norm *loaded-files*)))))
    order))
```

- [ ] **Step 4: Tests bestehen**

Run: `./build/golisp2 tests/defsystem-tests.lisp && ./build/golisp2 -t && go test ./... -count=1`
Expected: PASS / Exit 0

- [ ] **Step 5: Commit**

```bash
git add embed/defsystem.lisp tests/defsystem-tests.lisp
git commit -m "feat(defsystem): load-system — Topo-Sort, Zykluserkennung, idempotent

Co-Authored-By: kimi-k3 <noreply@anthropic.com>"
```

---

### Task 5: `loaded-systems` + `system-symbols`

**Files:**
- Modify: `embed/defsystem.lisp`
- Test: `tests/defsystem-tests.lisp`

**Interfaces:**
- Produces: `(loaded-systems)` → Liste der Systemnamen, deren sämtliche Komponenten geladen sind (berechnet aus `*loaded-files*`). `(system-symbols name)` → Symbolliste via `defined-in` (Task 1). `%sys-loaded?` intern, auch von Task 6 genutzt.

- [ ] **Step 1: Failing Tests anhängen**

```lisp
;; --- loaded-systems + system-symbols ---------------------------------
(assert= t (if (member 'fx-sys-a (loaded-systems)) t ()))
(assert= t (if (member 'fx-sys-b (loaded-systems)) t ()))
;; fx-sys-u listet eine Datei, die nie geladen wird → nicht geladen
(defsystem fx-sys-u :components ("tests/fixtures/fx-u.lisp"))
(assert= '() (if (member 'fx-sys-u (loaded-systems)) '(drin) '()))
(assert= '(fx-a) (system-symbols 'fx-sys-a))
(assert= 'err (trap (system-symbols 'gibts-nicht) (lambda (e) 'err)))
```

- [ ] **Step 2: Fehlschlag verifizieren**

Run: `./build/golisp2 tests/defsystem-tests.lisp`
Expected: ERR — `unbekanntes Symbol 'loaded-systems'`

- [ ] **Step 3: Implementierung** — an `embed/defsystem.lisp` anhängen:

```lisp
;; === Introspection ===================================================

;; %sys-loaded?: t, wenn alle Komponenten des Systems in *loaded-files*
(defun %sys-loaded? (entry)
  (every (lambda (c) (member (get-file-path c) *loaded-files*))
         (%sys-get entry :components)))

;; (loaded-systems) → Namen aller vollständig geladenen Systeme.
;; Berechnet aus *loaded-files* — einzige Wahrheit, kein Extra-Flag.
(defun loaded-systems ()
  (mapcar #'car (filter (lambda (e) (%sys-loaded? (cadr e))) *systems*)))

;; (system-symbols 'name) → alle Symbole, die die Komponenten des
;; Systems definiert haben (via DefLoc-Registry, je Datei sortiert).
(defun system-symbols (name)
  (let ((acc '()))
    (dolist (c (%sys-get (%sys-entry name) :components) acc)
      (set! acc (append acc (defined-in c))))))
```

- [ ] **Step 4: Tests bestehen**

Run: `./build/golisp2 tests/defsystem-tests.lisp && ./build/golisp2 -t`
Expected: PASS / Exit 0

- [ ] **Step 5: Commit**

```bash
git add embed/defsystem.lisp tests/defsystem-tests.lisp
git commit -m "feat(defsystem): loaded-systems + system-symbols (Introspection)

Co-Authored-By: kimi-k3 <noreply@anthropic.com>"
```

---

### Task 6: `unload-system` — Shared-File-Logik + Policy-Hülle

**Files:**
- Modify: `embed/defsystem.lisp`
- Test: `tests/defsystem-tests.lisp`

**Interfaces:**
- Produces: `(unload-system name)` → Liste entfernter Symbole. Shared Files (von anderen geladenen Systemen mitgelistet) bleiben unangetastet. Deps werden nicht mit-entladen. Nicht-geladenes System → `()`.
- Consumes: `%sys-entry`, `%sys-get`, `%sys-loaded?` (Task 3/5), `defined-in` (Task 1).

**Design-Notiz (Abweichung von Spec-YAGNI-Liste, vom Plan-Review zu bestätigen):**
`makunbound` auf LAMBDA warnt unter Default-Policy `warn` pro Symbol (`REDEF: …`). unload-system würde pro Symbol eine Zeile Rauschen erzeugen — konkreter Bedarf für die im Spec zurückgestellte Policy-Hülle: unload sichert die Policy, setzt `allow`, stellt am Ende die alte wieder her. Das Redef-Log protokolliert die Events trotzdem.

- [ ] **Step 1: Failing Tests anhängen**

```lisp
;; --- unload-system: einfacher Fall ------------------------------------
(assert= '(fx-a) (unload-system 'fx-sys-a))
(assert= '() (bound? 'fx-a))
;; b und c sind Deps von a — werden NICHT mit-entladen
(assert= t (bound? 'fx-b))
(assert= t (bound? 'fx-c))
(assert= '() (if (member 'fx-sys-a (loaded-systems)) '(drin) '()))

;; --- unload-system: Shared File bleibt beim ersten unload stehen ------
(defsystem fx-sys-s1 :components ("tests/fixtures/fx-shared.lisp"))
(defsystem fx-sys-s2 :components ("tests/fixtures/fx-shared.lisp"))
(load-system 'fx-sys-s1)
(load-system 'fx-sys-s2)
(assert= '() (unload-system 'fx-sys-s1))     ; shared → nichts entfernt
(assert= t (bound? 'fx-shared))
(assert= '(fx-shared) (unload-system 'fx-sys-s2))
(assert= '() (bound? 'fx-shared))

;; --- unload-system: nicht-geladenes System = no-op --------------------
(assert= '() (unload-system 'fx-sys-s1))
(assert= '() (unload-system 'fx-sys-u))
(assert= 'err (trap (unload-system 'gibts-nicht) (lambda (e) 'err)))

;; --- Aufräumen: Registry + Symbole zurücksetzen -----------------------
(unload-system 'fx-sys-b)
(unload-system 'fx-sys-c)
(set! *systems*
      (filter (lambda (e) (not (member (car e) '(fx-sys-a fx-sys-b fx-sys-c
                                                 fx-cy-a fx-cy-b fx-sys-u
                                                 fx-sys-s1 fx-sys-s2))))
              *systems*))
```

- [ ] **Step 2: Fehlschlag verifizieren**

Run: `./build/golisp2 tests/defsystem-tests.lisp`
Expected: ERR — `unbekanntes Symbol 'unload-system'`

- [ ] **Step 3: Implementierung** — an `embed/defsystem.lisp` anhängen:

```lisp
;; === unload-system ===================================================

;; %remove-first: erstes Vorkommen von x aus Liste entfernen
(defun %remove-first (x lst)
  (cond ((null lst)             '())
        ((equal? x (car lst))   (cdr lst))
        (t (cons (car lst) (%remove-first x (cdr lst))))))

;; %file-shared?: t, wenn ein ANDERES geladenes System die Datei mitlistet
(defun %file-shared? (norm self)
  (any (lambda (e)
         (and (not (equal? (car e) self))
              (%sys-loaded? (cadr e))
              (any (lambda (c) (equal? (get-file-path c) norm))
                   (%sys-get (cadr e) :components))))
       *systems*))

;; (unload-system 'name) → Liste der entfernten Symbole.
;; Shared Files (anderes geladenes System listet sie) bleiben unangetastet.
;; Deps werden nicht mit-entladen. Policy-Hülle: makunbound warnt sonst
;; pro Symbol — Events landen trotzdem im Redef-Log. trap stellt die
;; Policy auch im Fehlerfall wieder her (prozess-globaler Zustand!).
(defun unload-system (name)
  (let ((entry (%sys-entry name))
        (removed '())
        (old-policy (redefine-policy)))
    (redefine-policy 'allow)
    (trap
      (dolist (c (%sys-get entry :components))
        (let ((norm (get-file-path c)))
          (when (and (member norm *loaded-files*)
                     (not (%file-shared? norm name)))
            (dolist (s (defined-in c))
              (when (bound? s)
                (makunbound s)
                (push s removed)))
            (set! *loaded-files* (%remove-first norm *loaded-files*)))))
      (lambda (e)
        (begin (redefine-policy old-policy) (error e))))
    (redefine-policy old-policy)
    removed))
```

- [ ] **Step 4: Tests bestehen**

Run: `./build/golisp2 tests/defsystem-tests.lisp && ./build/golisp2 -t && go test ./... -count=1`
Expected: PASS / Exit 0; **keine** `REDEF:`-Warnungen im Output der unload-Tests

- [ ] **Step 5: Commit**

```bash
git add embed/defsystem.lisp tests/defsystem-tests.lisp
git commit -m "feat(defsystem): unload-system mit Shared-File-Schutz + Policy-Hülle

Co-Authored-By: kimi-k3 <noreply@anthropic.com>"
```

---

### Task 7: TODO.md abhaken + Endverifikation

**Files:**
- Modify: `TODO.md`

- [ ] **Step 1: TODO.md aktualisieren** — Abschnitt „Aufgabe — `defsystem`-lite" als erledigt markieren: Überschrift zu `## Erledigt 2026-07-25 — defsystem-lite (siehe docs/superpowers/specs/2026-07-25-defsystem-lite-design.md)` ändern, Rest des Abschnitts darunter unverändert lassen.

- [ ] **Step 2: Endverifikation komplett**

Run: `./build.sh && go test ./... -count=1 && ./build/golisp2 -t`
Expected: alles grün, Exit 0

Zusätzlich REPL-Smoke-Test:

Run: `./build/golisp2 -e "(defsystem demo :components (\"tests/fixtures/fx-a.lisp\")) (load-system 'demo) (println (loaded-systems)) (println (system-symbols 'demo)) (println (unload-system 'demo))"`
Expected: `(demo)` / `(fx-a)` / `(fx-a)`

- [ ] **Step 3: Commit**

```bash
git add TODO.md
git commit -m "doc(todo): defsystem-lite erledigt

Co-Authored-By: kimi-k3 <noreply@anthropic.com>"
```
