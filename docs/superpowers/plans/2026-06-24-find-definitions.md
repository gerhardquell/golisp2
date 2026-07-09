# find-definitions-for-emacs (M-.) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `M-.` in SLIME springt zur Definition einer `defun`/`defmacro`/`define` in der echten Quelldatei (Zeile genau); REPL-definierte Funktionen erhalten einen rekonstruierten Snippet im Temp-Buffer.

**Architecture:** Source-Location auf `Cell` (`SrcFile`/`SrcLine`), vom Reader gestempelt. `defun`/`defmacro`/`define` registrieren `(symbol→file/line)` in einer Go-seitigen `sync.RWMutex`-Map. `swank:find-definitions-for-emacs` macht Map-Lookup → `:location`, mit Snippet-Fallback für REPL-Funktionen.

**Tech Stack:** Go (stdlib `sync`), GoLisp-Lisp-Handler, SLIME-Protokoll.

**Spec-Abweichung (begründet):** DefLoc-Map liegt in `lib/defloc.go` (nicht `lib/swank/defs.go`). Grund: `evalDefun` (Package `lib`) ruft `RegisterDefinition`; `swank--find-definition` (Package `swank`) ruft `LookupDefinition`. Da `swank` bereits `lib` importiert, darf `lib` nicht zurück nach `swank` importieren → Map muss in `lib` liegen. Keine Funktionalität verloren.

## Global Constraints

- Go-Konventionen: 2 Spaces Einrückung, keine Tabs, camelCase Dateinamen.
- Datei-Header: Autor `Gerhard Quell - gquell@skequell.de`, CoAutor `claude sonnet 4.6`, Copyright `2026 Gerhard Quell - SKEQuell`, Erstellt `YYYYMMDD`.
- Fehlerformat: `fmt.Errorf("funktionsname: beschreibung")`.
- Dateigröße max 1000 Zeilen.
- Build via `./build`, Temp-Dateien nach `./tmp`.
- `parfunc`-Konkurrenz: gemeinsame Map braucht `sync.RWMutex`.
- Tests via `go test ./...`, Race-Check `go test -race ./lib/`.

## File Structure

| Datei | Verantwortung |
|-------|---------------|
| `lib/types.go` | `Cell`-Struct um `SrcFile`/`SrcLine` erweitern |
| `lib/reader.go` | Zeilen-Tracking, `SrcLine` auf Listen-Cells stempeln |
| `lib/eval_load.go` | `evalLoad` stempelt `SrcFile` auf Top-Level-Formen |
| `lib/defloc.go` (neu) | `DefLoc`-Struct, Map + `sync.RWMutex`, `RegisterDefinition`/`LookupDefinition`/`ClearDefinitions` |
| `lib/eval_core.go` | 3 Dispatch-Zeilen: `evalDefun`/`evalDefmacro`/`evalDefine` erhalten `expr` statt `expr.Cdr` |
| `lib/eval_specialforms.go` | Sigs der 3 Funktionen auf `form *Cell`; `args := form.Cdr`; `RegisterDefinition` aufrufen |
| `lib/swank/env.go` | Primitive `swank--find-definition` registrieren |
| `lib/swank/swank.lisp` | `swank:find-definitions-for-emacs` Handler + Dispatch-Eintrag + `swank--reconstruct-definition` |

---

### Task 1: Cell-Felder SrcFile/SrcLine

**Files:**
- Modify: `lib/types.go:25-37` (Cell-Struct)
- Test: `lib/types_test.go` (neu oder ergänzt — prüfe ob existiert)

**Interfaces:**
- Produces: `Cell.SrcFile string`, `Cell.SrcLine int` (zero-value = unbekannt) für alle nachfolgenden Tasks.

- [ ] **Step 1: Schreibe den fehlschlagenden Test**

Lege `lib/types_test.go` an (falls nicht vorhanden, sonst ergänze):

```go
package lib

import "testing"

func TestCellSourceLocationDefaults(t *testing.T) {
  c := MakeAtom("foo")
  if c.SrcFile != "" {
    t.Fatalf("SrcFile default leer erwartet, got %q", c.SrcFile)
  }
  if c.SrcLine != 0 {
    t.Fatalf("SrcLine default 0 erwartet, got %d", c.SrcLine)
  }
}
```

- [ ] **Step 2: Testlauf — soll fehlschlagen**

Run: `go test ./lib/ -run TestCellSourceLocationDefaults`
Expected: FAIL — `c.SrcFile undefined` (Compile-Fehler).

- [ ] **Step 3: Felder zum Cell-Struct hinzufügen**

In `lib/types.go`, ersetze den Cell-Struct (Zeilen 25-37):

```go
type Cell struct {
  Type LispType
  // Atom/String/Number
  Val  string
  Num  float64
  // Liste
  Car  *Cell
  Cdr  *Cell
  // eingebaute Funktion
  Fn   func(args []*Cell) (*Cell, error)
  // Lambda-Closure: Umgebung zum Zeitpunkt der Definition
  Env  interface{} // *Env – interface{} um Zirkelimport zu vermeiden
  // Quellposition (Reader/Load gestempelt, 0 = unbekannt)
  SrcFile string
  SrcLine int
}
```

- [ ] **Step 4: Testlauf — soll durchgehen**

Run: `go test ./lib/ -run TestCellSourceLocationDefaults`
Expected: PASS.

- [ ] **Step 5: Build prüfen + Commit**

```bash
go build ./...
git add lib/types.go lib/types_test.go
git commit -m "feat(types): Cell-Felder SrcFile/SrcLine fuer Quellposition

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 2: Reader-Zeilen-Tracking + SrcLine-Stempel

**Files:**
- Modify: `lib/reader.go:21-28` (Reader-Struct, NewReader), `:57-61` (next), `:96-113` (readList)
- Test: `lib/reader_test.go` (ergänze)

**Interfaces:**
- Consumes: `Cell.SrcLine` aus Task 1.
- Produces: Jede nicht-leere Listen-Cell (`Cons`) trägt `SrcLine` = Zeile der öffnenden Klammer.

- [ ] **Step 1: Schreibe den fehlschlagenden Test**

Ergänze in `lib/reader_test.go`:

```go
func TestReaderStampsSrcLine(t *testing.T) {
  src := "(defun f (x)\n  (* x x))\n(defun g () 1)"
  forms, err := ReadAll(src)
  if err != nil {
    t.Fatalf("ReadAll: %v", err)
  }
  // forms = (form1 form2), beide Listen-Cells
  f1 := forms.Car
  f2 := forms.Cdr.Car
  if f1.SrcLine != 1 {
    t.Fatalf("form1 SrcLine = 1 erwartet, got %d", f1.SrcLine)
  }
  if f2.SrcLine != 2 {
    t.Fatalf("form2 SrcLine = 2 erwartet, got %d", f2.SrcLine)
  }
}
```

- [ ] **Step 2: Testlauf — soll fehlschlagen**

Run: `go test ./lib/ -run TestReaderStampsSrcLine`
Expected: FAIL — `f1.SrcLine` ist 0.

- [ ] **Step 3: Reader-Struct + NewReader erweitern**

In `lib/reader.go`, ersetze Zeilen 21-28:

```go
type Reader struct {
  src  []rune
  pos  int
  line int
}

func NewReader(s string) *Reader {
  return &Reader{src: []rune(s), pos: 0, line: 1}
}
```

- [ ] **Step 4: next() zählt Zeilen**

In `lib/reader.go`, ersetze die `next()`-Methode (Zeilen 57-61):

```go
func (r *Reader) next() (rune, bool) {
  ch, ok := r.peek()
  if ok {
    r.pos++
    if ch == '\n' {
      r.line++
    }
  }
  return ch, ok
}
```

- [ ] **Step 5: readList stempelt SrcLine auf Cons-Cell**

In `lib/reader.go`, ersetze die `readList()`-Methode (Zeilen 96-113):

```go
func (r *Reader) readList() (*Cell, error) {
  startLine := r.line
  r.next() // '(' überspringen
  r.skipWS()

  ch, ok := r.peek()
  if !ok { return nil, fmt.Errorf("reader: unerwartetes EOF in Liste") }
  if ch == ')' { r.next(); return MakeNil(), nil }  // leere Liste ()

  // erstes Element
  car, err := r.readExpr()
  if err != nil { return nil, err }

  // Rest der Liste rekursiv
  cdr, err := r.readRest()
  if err != nil { return nil, err }

  cell := Cons(car, cdr)
  cell.SrcLine = startLine
  return cell, nil
}
```

- [ ] **Step 6: Testlauf — soll durchgehen**

Run: `go test ./lib/ -run TestReaderStampsSrcLine`
Expected: PASS.

- [ ] **Step 7: Volle Reader-Tests + Race + Commit**

```bash
go test ./lib/ -run TestReader -v
git add lib/reader.go lib/reader_test.go
git commit -m "feat(reader): Zeilen-Tracking, stempelt SrcLine auf Listen-Cells

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 3: load stempelt SrcFile

**Files:**
- Modify: `lib/eval_load.go:55-85` (evalLoad)
- Test: `lib/eval_load_test.go` (neu oder ergänzt — prüfe ob existiert)

**Interfaces:**
- Consumes: `Cell.SrcFile` aus Task 1, Reader aus Task 2.
- Produces: Top-Level-Formen aus `(load "file")` tragen `SrcFile` = absoluter Pfad.

- [ ] **Step 1: Schreibe den fehlschlagenden Test**

Lege `lib/eval_load_test.go` an (falls nicht vorhanden) mit Header (Autor/CoAutor/Copyright/Erstellt 20260624) und:

```go
package lib

import (
  "os"
  "path/filepath"
  "testing"
)

func TestLoadStampsSrcFile(t *testing.T) {
  dir := t.TempDir()
  path := filepath.Join(dir, "src.lisp")
  if err := os.WriteFile(path, []byte("(defun h () 1)\n"), 0644); err != nil {
    t.Fatalf("write: %v", err)
  }
  env := BaseEnv()
  if _, err := LoadString(`(load "`+path+`")`, env); err != nil {
    t.Fatalf("load: %v", err)
  }
  loc, ok := LookupDefinition("h")
  if !ok {
    t.Fatalf("Definition h nicht registriert")
  }
  if loc.File != path {
    t.Fatalf("SrcFile = %q erwartet, got %q", path, loc.File)
  }
  if loc.Line != 1 {
    t.Fatalf("SrcLine = 1 erwartet, got %d", loc.Line)
  }
}
```

Hinweis: Dieser Test greift auf `LookupDefinition` (Task 4) und `RegisterDefinition`-Aufruf in `defun` (Task 5) vor. Damit er schon in Task 3 kompiliert, legst du in Task 4 die Map an — **führe Task 4 vor Task 3 Step 2 aus**, oder führe Task 3+4 zusammen. Sauberste Reihenfolge: Task 4 zuerst (Map existiert, `LookupDefinition` aufrufbar), dann Task 3 (Test kompiliert), dann Task 5 (defun registriert → Test grün). Siehe Task-Neuordnung unten.

- [ ] **Step 2: Testlauf — vorerst überspringen (Task 4 vorab nötig)**

Dieser Test wird erst nach Task 4 + Task 5 grün. Führe ihn nach Task 5 aus.

- [ ] **Step 3: evalLoad stempelt SrcFile**

In `lib/eval_load.go`, ersetze die `evalLoad()`-Funktion (Zeilen 55-85). Füge das `SrcFile`-Stempeln nach `readExpr` ein:

```go
func evalLoad(args *Cell, env *Env) (*Cell, error) {
  filenameCell, err := Eval(args.Car, env)
  if err != nil { return nil, err }

  resolvedPath, err := resolveLibraryPath(filenameCell.Val)
  if err != nil {
    return nil, fmt.Errorf("load: %v", err)
  }

  data, err := os.ReadFile(resolvedPath)
  if err != nil {
    return nil, fmt.Errorf("load: '%s' nicht lesbar: %w", resolvedPath, err)
  }

  src := strings.TrimSpace(string(data))
  var result *Cell

  r := NewReader(src)
  for {
    r.skipWS()
    if r.pos >= len(r.src) { break }

    expr, err := r.readExpr()
    if err != nil { return nil, fmt.Errorf("load %s: %w", resolvedPath, err) }

    if expr.Type == LIST {
      expr.SrcFile = resolvedPath
    }

    result, err = Eval(expr, env)
    if err != nil { return nil, fmt.Errorf("load %s: %w", resolvedPath, err) }
  }
  return result, nil
}
```

- [ ] **Step 4: Commit (Test folgt in Task 5)**

```bash
go build ./...
git add lib/eval_load.go
git commit -m "feat(load): stempelt SrcFile auf Top-Level-Formen

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 4: DefLoc-Map (lib/defloc.go)

**Files:**
- Create: `lib/defloc.go`
- Test: `lib/defloc_test.go` (neu)

**Interfaces:**
- Produces (für Task 5, 6, 3):
  - `type DefLoc struct { File string; Line int }`
  - `func RegisterDefinition(name, file string, line int)` — thread-safe, last-write-wins
  - `func LookupDefinition(name string) (DefLoc, bool)` — thread-safe Read
  - `func ClearDefinitions()` — für Tests, leert die Map

- [ ] **Step 1: Schreibe den fehlschlagenden Test**

Lege `lib/defloc_test.go` an mit Header (Erstellt 20260624):

```go
package lib

import (
  "sync"
  "testing"
)

func TestRegisterAndLookupDefinition(t *testing.T) {
  ClearDefinitions()
  RegisterDefinition("foo", "/a/b.lisp", 7)
  loc, ok := LookupDefinition("foo")
  if !ok {
    t.Fatalf("foo nicht gefunden")
  }
  if loc.File != "/a/b.lisp" || loc.Line != 7 {
    t.Fatalf("got %+v", loc)
  }
}

func TestLookupUnknownDefinition(t *testing.T) {
  ClearDefinitions()
  _, ok := LookupDefinition("nope")
  if ok {
    t.Fatalf("nope sollte nicht gefunden werden")
  }
}

func TestConcurrentRegisterDefinition(t *testing.T) {
  ClearDefinitions()
  var wg sync.WaitGroup
  for i := 0; i < 50; i++ {
    wg.Add(1)
    go func(n int) {
      defer wg.Done()
      RegisterDefinition("c", "/c.lisp", n)
    }(i)
  }
  wg.Wait()
  _, ok := LookupDefinition("c")
  if !ok {
    t.Fatalf("c nach concurrent writes nicht gefunden")
  }
}
```

- [ ] **Step 2: Testlauf — soll fehlschlagen**

Run: `go test ./lib/ -run TestRegisterAndLookupDefinition`
Expected: FAIL — `undefined: RegisterDefinition` (Compile-Fehler).

- [ ] **Step 3: defloc.go implementieren**

Lege `lib/defloc.go` an mit Header (Erstellt 20260624):

```go
//**********************************************************************
//  lib/defloc.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260624
//**********************************************************************
// Definition-Registry: symbol -> (file, line). Thread-safe via
// sync.RWMutex (parfunc-safe). Genutzt von defun/defmacro/define und
// swank:find-definitions-for-emacs (M-.).
//**********************************************************************

package lib

import "sync"

// DefLoc speichert Quellposition einer Definition.
type DefLoc struct {
  File string
  Line int
}

var (
  defMu       sync.RWMutex
  definitions = map[string]DefLoc{}
)

// RegisterDefinition merkt sich die Quellposition eines Symbols.
// Last-write-wins: Neu-Definition überschreibt alten Eintrag.
func RegisterDefinition(name, file string, line int) {
  defMu.Lock()
  defer defMu.Unlock()
  definitions[name] = DefLoc{File: file, Line: line}
}

// LookupDefinition liefert die gespeicherte Quellposition oder ok=false.
func LookupDefinition(name string) (DefLoc, bool) {
  defMu.RLock()
  defer defMu.RUnlock()
  loc, ok := definitions[name]
  return loc, ok
}

// ClearDefinitions leert die Registry (nur für Tests).
func ClearDefinitions() {
  defMu.Lock()
  defer defMu.Unlock()
  definitions = map[string]DefLoc{}
}
```

- [ ] **Step 4: Testlauf — soll durchgehen (inkl. Race)**

Run: `go test -race ./lib/ -run TestRegisterAndLookupDefinition -v`
Run: `go test -race ./lib/ -run TestLookupUnknownDefinition -v`
Run: `go test -race ./lib/ -run TestConcurrentRegisterDefinition -v`
Expected: alle PASS, kein Race-Detector-Alarm.

- [ ] **Step 5: Commit**

```bash
git add lib/defloc.go lib/defloc_test.go
git commit -m "feat(defloc): thread-safe Definition-Registry (symbol->file/line)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 5: defun/defmacro/define registrieren Location

**Files:**
- Modify: `lib/eval_core.go:44-46` (Dispatch), `lib/eval_specialforms.go:19-25` (evalDefine), `:122-127` (evalDefun), `:235-241` (evalDefmacro)
- Test: `lib/eval_load_test.go::TestLoadStampsSrcFile` (aus Task 3, jetzt grün), plus neuer Test.

**Interfaces:**
- Consumes: `RegisterDefinition` (Task 4), `Cell.SrcFile`/`SrcLine` (Task 1, 2, 3).
- Produces: Side-Effect — Definitionen in der Registry; Task 6/7 lesen via `LookupDefinition`.

- [ ] **Step 1: Schreibe den fehlschlagenden Test**

Ergänze in `lib/eval_load_test.go`:

```go
func TestDefunRegistersLocation(t *testing.T) {
  ClearDefinitions()
  env := BaseEnv()
  src := "(defun sq (x) (* x x))"
  form, err := Read(src)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  form.SrcFile = "/test.lisp"
  form.SrcLine = 3
  if _, err := evalForm(form, env); err != nil {
    t.Fatalf("eval: %v", err)
  }
  loc, ok := LookupDefinition("sq")
  if !ok {
    t.Fatalf("sq nicht registriert")
  }
  if loc.File != "/test.lisp" || loc.Line != 3 {
    t.Fatalf("got %+v", loc)
  }
}
```

Hinweis: `evalForm` ist der Test-Zugang zu `Eval`. Falls `Eval` direkt exported ist, nutze `Eval(form, env)`. Prüfe Signatur in `lib/eval_core.go` — falls `Eval` exportiert ist, ersetze `evalForm` durch `Eval`.

- [ ] **Step 2: Testlauf — soll fehlschlagen**

Run: `go test ./lib/ -run TestDefunRegistersLocation`
Expected: FAIL — `sq` nicht registriert (`ok == false`).

- [ ] **Step 3: Dispatch gibt form statt args durch**

In `lib/eval_core.go`, ersetze die drei Dispatch-Zeilen (44-46):

```go
      case "define", "setq":  return evalDefine(expr, env)
      case "defun":           return evalDefun(expr, env)
      case "defmacro":        return evalDefmacro(expr, env)
```

- [ ] **Step 4: evalDefine auf form + Register**

In `lib/eval_specialforms.go`, ersetze `evalDefine` (Zeilen 19-25):

```go
func evalDefine(form *Cell, env *Env) (*Cell, error) {
  args := form.Cdr
  name := args.Car.Val
  val, err := Eval(args.Cdr.Car, env)
  if err != nil { return nil, err }
  env.Set(name, val)
  RegisterDefinition(name, form.SrcFile, form.SrcLine)
  return MakeAtom(name), nil
}
```

- [ ] **Step 5: evalDefun auf form + Register**

In `lib/eval_specialforms.go`, ersetze `evalDefun` (Zeilen 122-127):

```go
func evalDefun(form *Cell, env *Env) (*Cell, error) {
  args := form.Cdr
  name := args.Car.Val
  lam  := makeLambda(args.Cdr.Car, wrapBegin(args.Cdr.Cdr), env)
  env.Set(name, lam)
  RegisterDefinition(name, form.SrcFile, form.SrcLine)
  return MakeAtom(name), nil
}
```

- [ ] **Step 6: evalDefmacro auf form + Register**

In `lib/eval_specialforms.go`, ersetze `evalDefmacro` (Zeilen 235-241):

```go
func evalDefmacro(form *Cell, env *Env) (*Cell, error) {
  args := form.Cdr
  name := args.Car.Val
  lam  := makeLambda(args.Cdr.Car, wrapBegin(args.Cdr.Cdr), env)
  lam.Type = MACRO   // ← einziger Unterschied zu defun!
  env.Set(name, lam)
  RegisterDefinition(name, form.SrcFile, form.SrcLine)
  return MakeAtom(name), nil
}
```

- [ ] **Step 7: Alle Tests durch + Race + Commit**

```bash
go test -race ./lib/ -run TestDefunRegistersLocation -v
go test -race ./lib/ -run TestLoadStampsSrcFile -v   # Task 3 Test jetzt grün
go test ./...
git add lib/eval_core.go lib/eval_specialforms.go lib/eval_load_test.go
git commit -m "feat(eval): defun/defmacro/define registrieren Quellposition

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 6: Primitive swank--find-definition

**Files:**
- Modify: `lib/swank/env.go` (RegisterSwankEnv, neuen Primitive ergänzen)
- Test: `lib/swank/env_test.go` (ergänze)

**Interfaces:**
- Consumes: `lib.LookupDefinition` (Task 4), `env.Get` (für Built-in/Lambda-Erkennung).
- Produces: Lisp-Primitive `swank--find-definition` → liefert `("file" . line)`-Cons oder `()`.

- [ ] **Step 1: Schreibe den fehlschlagenden Test**

Ergänze in `lib/swank/env_test.go`:

```go
func TestSwankFindDefinition(t *testing.T) {
  env := lib.BaseEnv()
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  lib.RegisterDefinition("found", "/x.lisp", 9)
  cell, err := callPrimitive(env, "swank--find-definition", lib.MakeStr("found"))
  if err != nil {
    t.Fatalf("call: %v", err)
  }
  s := cell.String()
  if !strings.Contains(s, "/x.lisp") || !strings.Contains(s, "9") {
    t.Fatalf("expected (/x.lisp . 9), got %s", s)
  }

  lib.ClearDefinitions()
  cell2, err := callPrimitive(env, "swank--find-definition", lib.MakeStr("missing"))
  if err != nil {
    t.Fatalf("call: %v", err)
  }
  if cell2.Type != lib.NIL {
    t.Fatalf("expected NIL für missing, got %v", cell2)
  }
}
```

Falls `callPrimitive`-Helper nicht existiert, evaluiere stattdessen via `lib.LoadString` + `lib.Read`/`Eval`. Prüfe `env_test.go` auf vorhandene Helper-Namenskonvention und passe an.

- [ ] **Step 2: Testlauf — soll fehlschlagen**

Run: `go test ./lib/swank/ -run TestSwankFindDefinition`
Expected: FAIL — `swank--find-definition` nicht registriert (`Get`-Fehler) oder `undefined: callPrimitive`.

- [ ] **Step 3: Primitive registrieren**

In `lib/swank/env.go`, füge vor der schließenden Klammer von `RegisterSwankEnv` (vor Zeile 130) ein:

```go
  // swank--find-definition: (name) -> ("file" . line) | NIL.
  // Map-Lookup in lib.LookupDefinition (defun/defmacro/define registriert).
  env.Set("swank--find-definition", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    if len(args) < 1 {
      return lib.MakeNil(), nil
    }
    loc, ok := lib.LookupDefinition(args[0].Val)
    if !ok {
      return lib.MakeNil(), nil
    }
    return lib.Cons(lib.MakeStr(loc.File), lib.MakeNum(float64(loc.Line))), nil
  }))
```

- [ ] **Step 4: Testlauf — soll durchgehen**

Run: `go test ./lib/swank/ -run TestSwankFindDefinition -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add lib/swank/env.go lib/swank/env_test.go
git commit -m "feat(swank): Primitive swank--find-definition (Map-Lookup)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 7: find-definitions-for-emacs Handler (Lisp)

**Files:**
- Modify: `lib/swank/swank.lisp` (Dispatch-Eintrag + Handler + Rekonstruktion)
- Test: `lib/swank/lisp_test.go` (ergänze 3 Tests)

**Interfaces:**
- Consumes: `swank--find-definition` (Task 6), `swank--value-string` (existiert), `env.Get`/Lambda-Erkennung (für REPL-Fallback + Built-in-Erkennung via `swank--cell-type`).
- Produces: SWANK-Op `swank:find-definitions-for-emacs` → `(:ok ((:location ...)))` | `(:ok (:error "..."))`.

- [ ] **Step 1: Schreibe die fehlschlagenden Tests**

Ergänze in `lib/swank/lisp_test.go`:

```go
func TestSwankFindDefinitionsFileLocation(t *testing.T) {
  lib.ClearDefinitions()
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  lib.RegisterDefinition("myfn", "/abs/src.lisp", 12)
  cell, err := lib.Read(`(:emacs-rex (swank:find-definitions-for-emacs "myfn") nil t 1)`)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, ":location") || !strings.Contains(s, "/abs/src.lisp") || !strings.Contains(s, ":line") {
    t.Fatalf("expected file location, got: %s", s)
  }
}

func TestSwankFindDefinitionsBuiltInError(t *testing.T) {
  lib.ClearDefinitions()
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  cell, err := lib.Read(`(:emacs-rex (swank:find-definitions-for-emacs "car") nil t 1)`)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, ":error") {
    t.Fatalf("expected :error for built-in, got: %s", s)
  }
}

func TestSwankFindDefinitionsReplSnippet(t *testing.T) {
  lib.ClearDefinitions()
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  // REPL-definiert (kein SrcFile): via listener-eval definieren
  leval, err := lib.Read(`(:emacs-rex (swank-repl:listener-eval "(defun rfn (n) (* n 2))") nil t 1)`)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  if _, err := HandleMessage(env, leval); err != nil {
    t.Fatalf("listener-eval: %v", err)
  }
  cell, err := lib.Read(`(:emacs-rex (swank:find-definitions-for-emacs "rfn") nil t 1)`)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, ":buffer") || !strings.Contains(s, "rfn") {
    t.Fatalf("expected buffer snippet for REPL fn, got: %s", s)
  }
}
```

- [ ] **Step 2: Testlauf — soll fehlschlagen**

Run: `go test ./lib/swank/ -run TestSwankFindDefinitions -v`
Expected: FAIL — Op fällt auf `ok-nil`, kein `:location`/`:error`/`:buffer`.

- [ ] **Step 3: Dispatch-Eintrag in handle-emacs-rex**

In `lib/swank/swank.lisp`, in der `cond` von `handle-emacs-rex`, füge vor der `else`-Klausel (vor Zeile 199) ein:

```lisp
      ((equal? op 'swank:find-definitions-for-emacs)
       (swank:find-definitions-for-emacs (cadr form) id))
```

- [ ] **Step 4: Handler + Rekonstruktion implementieren**

In `lib/swank/swank.lisp`, füge am Ende der Datei an:

```lisp
;; swank:find-definitions-for-emacs (name) -> (:ok ((:location ...))) | (:ok (:error "...")).
;; M-. in SLIME. Map-Lookup zuerst; sonst REPL-Snippet-Fallback oder :error.
(defun swank:find-definitions-for-emacs (name id)
  (catch
    (let ((loc (swank--find-definition name)))
      (let ((location
              (if (null? loc)
                  (swank--location-or-error name)
                  (list :location
                        (list :file (car loc))
                        (list :line (cdr loc) :align t)
                        (list)))))
        (list (list :return (list :ok (list location)) id))))
    (lambda (err)
      (list (list :return (list :abort (swank--value-string err)) id)))))

;; Kein Map-Treffer: REPL-definiert (Lambda/Macro) -> Snippet-Buffer;
;; Built-in (FUNC ohne Env) oder unbound -> :error.
(defun swank--location-or-error (name)
  (let ((cell (catch (eval (read name)) (lambda (err) ()))))
    (if (null? cell)
        (list :error (string-append "Symbol '" name "' nicht definiert"))
        (if (swank--lambda-or-macro? cell)
            (swank--snippet-location name cell)
            (list :error
              (string-append "keine Quellposition für '" name "'"))))))

(defun swank--lambda-or-macro? (cell)
  (or (and (list? cell) (not (null? (swank--closure-env cell))))
      (equal? (swank--cell-type cell) "macro")))

;; Lambda-Cell Env-Feld pruefen (Closure hat Env!=nil, Built-in FUNC nicht).
(defun swank--closure-env (cell)
  (catch (eval (read "(lambda (x) x)")) (lambda (e) ()))) ; Platzhalter – siehe Hinweis
```

Wichtiger Hinweis zur `swank--closure-env`: Lambda-Erkennung über das interne `Env`-Feld ist aus Lisp nicht direkt zugänglich. Nutze stattdessen die bestehende Heuristik aus `swank--arglist` (in `env.go`): eine Cell mit `Type:LIST` und `Env!=nil` ist ein Lambda; `Type:MACRO` ist ein Macro. Da Lisp das `Env`-Feld nicht sieht, **ersetze** `swank--lambda-or-macro?` durch einen neuen Go-Primitive `swank--definition-source` (siehe Task 7b unten), der die ganze Logik serverseitig kapselt. Führe Task 7b aus, anstatt `swank--closure-env` zu nutzen.

- [ ] **Step 5: Task 7b — Go-Primitive swank--definition-source (statt closure-env-Heuristik)**

Die saubere Lösung: ein Go-Primitive, das Symbol → `(kind . cell)` liefert, wobei `kind` ∈ `lambda|macro|builtin|unbound` ist (nutzt `Env!=nil`-Check, den nur Go kann).

In `lib/swank/env.go`, füge in `RegisterSwankEnv` ein:

```go
  // swank--definition-kind: (name) -> "lambda" | "macro" | "builtin" | "unbound".
  // Lambda = Type:LIST mit Env!=nil; Macro = Type:MACRO; sonst builtin/unbound.
  env.Set("swank--definition-kind", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    if len(args) < 1 {
      return lib.MakeStr("unbound"), nil
    }
    cell, err := env.Get(args[0].Val)
    if err != nil || cell == nil {
      return lib.MakeStr("unbound"), nil
    }
    switch {
    case cell.Type == lib.LIST && cell.Env != nil:
      return lib.MakeStr("lambda"), nil
    case cell.Type == lib.MACRO:
      return lib.MakeStr("macro"), nil
    case cell.Type == lib.FUNC:
      return lib.MakeStr("builtin"), nil
    default:
      return lib.MakeStr("unbound"), nil
    }
  }))
```

Und der Primitive, der die Cell für die Snippet-Rekonstruktion liefert (Lambda/Macro als Cell zurück, sonst `()`):

```go
  // swank--definition-cell: (name) -> Lambda/Macro-Cell | NIL.
  // Für swank--reconstruct-definition (REPL-Snippet).
  env.Set("swank--definition-cell", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    if len(args) < 1 {
      return lib.MakeNil(), nil
    }
    cell, err := env.Get(args[0].Val)
    if err != nil {
      return lib.MakeNil(), nil
    }
    if (cell.Type == lib.LIST && cell.Env != nil) || cell.Type == lib.MACRO {
      return cell, nil
    }
    return lib.MakeNil(), nil
  }))
```

Ersetze dann in `swank.lisp` die Platzhalter-Funktionen durch:

```lisp
(defun swank--location-or-error (name)
  (let ((kind (swank--definition-kind name)))
    (cond
      ((or (equal? kind "lambda") (equal? kind "macro"))
       (swank--snippet-location name (swank--definition-cell name)))
      ((equal? kind "builtin")
       (list :error
         (string-append "eingebaute Funktion '" name "' hat keine Quellposition")))
      (else
       (list :error (string-append "Symbol '" name "' nicht definiert"))))))

(defun swank--snippet-location (name cell)
  (let ((header (if (equal? (swank--definition-kind name) "macro")
                    "(defmacro " "(defun ")))
    (let ((snippet (string-append header name " "
                    (swank--value-string (car cell)) " "
                    (swank--value-string (cdr cell)) ")")))
      (list :location
            (list :buffer (string-append "*slime-source " name "*")
                  (list :source snippet))
            (list :position 1)
            (list)))))

;; Kompatibilität: nicht mehr benötigte Hilfsfunktionen entfernen.
```

Entferne die Platzhalter `swank--lambda-or-macro?`, `swank--closure-env` aus Step 4 wieder.

- [ ] **Step 6: Testlauf — soll durchgehen**

Run: `go test ./lib/swank/ -run TestSwankFindDefinitions -v`
Expected: alle 3 PASS (file-location, builtin-error, repl-snippet).

- [ ] **Step 7: Volle Suite + Commit**

```bash
go test ./...
git add lib/swank/env.go lib/swank/swank.lisp
git commit -m "feat(swank): find-definitions-for-emacs (M-.) mit Snippet-Fallback

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 8: Integrationstest + manuelle Verifikation

**Files:**
- Test: `lib/swank/integration_test.go` (ergänze)

**Interfaces:**
- Consumes: alle vorherigen Tasks.
- Produces: End-to-End-Verifikation dass M-. über das SWANK-Protokoll funktioniert.

- [ ] **Step 1: Schreibe den Integrationstest**

Ergänze in `lib/swank/integration_test.go`:

```go
func TestIntegrationFindDefinitionsLoadFile(t *testing.T) {
  lib.ClearDefinitions()
  // Testdatei mit defun an bekannter Zeile
  dir := t.TempDir()
  path := filepath.Join(dir, "mod.lisp")
  content := ";; comment\n(defun loaded-fn (x) (* x x))\n"
  if err := os.WriteFile(path, []byte(content), 0644); err != nil {
    t.Fatalf("write: %v", err)
  }
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  // Datei via swank:load-file laden (stempelt SrcFile)
  loadMsg, err := lib.Read(`(:emacs-rex (swank:load-file "` + path + `") nil t 1)`)
  if err != nil {
    t.Fatalf("read load: %v", err)
  }
  if _, err := HandleMessage(env, loadMsg); err != nil {
    t.Fatalf("load-file: %v", err)
  }
  // M-. auf loaded-fn
  findMsg, err := lib.Read(`(:emacs-rex (swank:find-definitions-for-emacs "loaded-fn") nil t 1)`)
  if err != nil {
    t.Fatalf("read find: %v", err)
  }
  result, err := HandleMessage(env, findMsg)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, path) {
    t.Fatalf("expected path %s in result, got: %s", path, s)
  }
  if !strings.Contains(s, ":line") {
    t.Fatalf("expected :line in result, got: %s", s)
  }
}
```

Stelle sicher, dass `filepath`, `os`, `strings` importiert sind (ggf. ergänzen).

- [ ] **Step 2: Testlauf — soll durchgehen**

Run: `go test ./lib/swank/ -run TestIntegrationFindDefinitions -v`
Expected: PASS.

- [ ] **Step 3: Volle Suite + Race**

```bash
go test -race ./...
```
Expected: alle PASS, kein Race.

- [ ] **Step 4: Build**

```bash
./build
```
Expected: `golisp2`-Binary gebaut ohne Fehler.

- [ ] **Step 5: Manuelle Emacs-Verifikation**

```bash
./golisp2 --swank 127.0.0.1:4242 &
```

In Emacs (SLIME geladen):
1. `M-x slime-connect` → Host `127.0.0.1`, Port `4242`, `y` bei Versionswarnung.
2. Datei `tmp/test.lisp` anlegen mit Inhalt:
   ```lisp
   (defun test-fn (n)
     (* n n))
   ```
3. In REPL: `(load "tmp/test.lisp")` ausführen.
4. Cursor auf `test-fn` irgendwo → `M-.`.
5. Erwartet: Emacs springt zu `tmp/test.lisp` Zeile 1 (die `defun`-Zeile).
6. `M-,` kehrt zurück.

- [ ] **Step 6: CLAUDE.md aktualisieren + Commit**

In `CLAUDE.md`, ergänze in der SWANK-Methoden-Tabelle eine Zeile:

```
| `swank:find-definitions-for-emacs` | Map-Lookup → `:location (:file ... :line N)`, REPL-Snippet-Fallback, Built-in `:error` — M-. |
```

Und im Abschnitt "Wichtigste Protokoll-Details" einen Punkt ergänzen:

```
- **`find-definitions-for-emacs` (M-.):** Lookup in lib.defloc-Map (defun/defmacro/define registrieren `SrcFile`/`SrcLine`). REPL-definierte Funktionen (kein SrcFile) → Snippet-Buffer; Built-ins → `:error`.
```

```bash
git add lib/swank/integration_test.go CLAUDE.md
git commit -m "test(swank): Integration M-. über load-file; CLAUDE.md ergänzt

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Self-Review

**1. Spec-Abdeckung:**
- Cell SrcFile/SrcLine → Task 1 ✓
- Reader Zeilen-Tracking + SrcLine-Stempel → Task 2 ✓
- load stempelt SrcFile → Task 3 ✓
- Go-Map + RWMutex + Register/Lookup → Task 4 ✓ (in `lib/defloc.go`, Spec-Abweichung begründet)
- Dispatch + evalDefun/Defmacro/Define registrieren → Task 5 ✓
- swank--find-definition Primitive → Task 6 ✓
- find-definitions-for-emacs Handler + Snippet-Fallback + Built-in-Error → Task 7 ✓
- Tests (Go + Lisp + Integration) → Task 4, 5, 6, 7, 8 ✓
- Race-Check → Task 4, 5, 8 ✓
- Manuelle Verifikation → Task 8 ✓

**2. Placeholder-Scan:** Task 7 Step 4 enthält bewusst eine Platzhalter-Funktion (`swank--closure-env`), die in Step 5 (Task 7b) durch saubere Go-Primitive ersetzt wird. Das ist im Plan textlich markiert und aufgelöst — kein offener Platzhalter am Ende. Alle Code-Blöcke vollständig.

**3. Typ-Konsistenz:**
- `DefLoc{File, Line}` konsistent in Task 4, 5, 6 ✓
- `RegisterDefinition(name, file string, line int)` / `LookupDefinition(name) (DefLoc, bool)` / `ClearDefinitions()` konsistent ✓
- `swank--find-definition`, `swank--definition-kind`, `swank--definition-cell` konsistent benannt ✓
- `evalDefun(form *Cell, env *Env)` Signatur in Task 5 Dispatch + Body konsistent ✓
- `Cell.SrcFile`/`SrcLine` konsistent ✓

**Hinweis zur Task-Reihenfolge:** Task 3 (load-Test) kompiliert erst nach Task 4 (Map). Führe Task 4 vor Task 3 Step 2 aus. Empfohlene Reihenfolge: 1 → 2 → 4 → 3 → 5 → 6 → 7 → 8.
