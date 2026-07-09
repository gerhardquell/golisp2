# Error Handling – (error) und (catch) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** `(error msg)` signalisiert einen Lisp-Fehler; `(catch body handler)` fängt ihn ab.

**Architecture:** Go's bestehende `(*Cell, error)` Rückgabe wird genutzt.
Ein neuer Typ `LispError` in `types.go` unterscheidet Lisp-Laufzeitfehler
von internen Go-Fehlern. `error` ist ein normales Primitiv (Argument wird ausgewertet).
`catch` ist eine Spezialform im Eval-Loop, die `Eval(body)` in einem eigenen
Aufruf ausführt und `*LispError` abfängt — echte Go-Fehler werden durchgereicht.

**Tech Stack:** Go, package `lib`, Dateien: `types.go`, `primitives.go`, `eval.go`, `main.go`

---

## Datenfluss

```
(error "msg")
  → fnError → return nil, &LispError{Msg: MakeStr("msg")}
  → propagiert als Go-error durch alle Eval-Ebenen
  → REPL / load druckt: ERR: msg

(catch (error "oops") (lambda (e) (string-append "gefangen: " e)))
  → evalCatch:
      Eval(body) → err = &LispError{Msg: "oops"}
      lispErr, ok := err.(*LispError)   ; ok = true
      apply(handler, [lispErr.Msg])     ; handler bekommt den Fehler-Cell
  → "gefangen: oops"

(catch (+ 1 2) (lambda (e) "fehler"))
  → evalCatch:
      Eval(body) → result = 3, err = nil
  → 3   ; kein Fehler, Handler wird nicht aufgerufen
```

---

## Task 1: LispError-Typ + `(error msg)` Primitiv

**Files:**
- Modify: `lib/types.go`
- Modify: `lib/primitives.go`
- Modify: `main.go` (Tests hinzufügen)

### Step 1: Failing tests schreiben

In `main.go`, nach `test(env, "(= (gensym) (gensym))")`, anfügen:

```go
test(env, `(catch (error "oops") (lambda (e) (string-append "caught: " e)))`)
test(env, `(catch (+ 1 2) (lambda (e) "fehler"))`)
```

### Step 2: Verify FAIL

```bash
cd /u/lisp-projekte/golisp && go run . -t
```

Erwartete Ausgabe für neue Tests:
```
ERR: ...catch undefined...
ERR: ...catch undefined...
```

Die bestehenden 8 Tests laufen weiterhin grün.

### Step 3: LispError zu types.go hinzufügen

Am Ende von `lib/types.go` (vor der letzten Leerzeile) einfügen:

```go
// LispError: Lisp-Laufzeitfehler, von (error msg) ausgelöst
// Unterscheidet sich von internen Go-Fehlern (fmt.Errorf)
type LispError struct {
  Msg *Cell
}

func (e *LispError) Error() string {
  return e.Msg.Val
}
```

### Step 4: fnError zu primitives.go hinzufügen

Am Ende von `lib/primitives.go` anfügen:

```go
// error: (error msg) → signalisiert Lisp-Laufzeitfehler
func fnError(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("error: 1 Argument nötig")
  }
  return nil, &LispError{Msg: args[0]}
}
```

In `BaseEnv()`, nach `env.Set("gensym", makeFn(fnGensym))`:

```go
env.Set("error", makeFn(fnError))
```

### Step 5: Compile-Check

```bash
cd /u/lisp-projekte/golisp && go build .
```

Erwartete Ausgabe: keine Fehler.

### Step 6: Commit

```bash
cd /u/lisp-projekte/golisp && git add lib/types.go lib/primitives.go && git commit -m "feat: LispError-Typ und (error msg) Primitiv"
```

---

## Task 2: `(catch body handler)` Spezialform

**Files:**
- Modify: `lib/eval.go`
- Modify: `main.go` (kein neuer Code – Tests aus Task 1 werden jetzt grün)

### Step 1: evalCatch zu eval.go hinzufügen

Am Ende von `lib/eval.go` (vor der letzten Leerzeile) einfügen:

```go
// catch: (catch body-expr handler-expr)
// Wertet body-expr aus. Bei LispError → handler mit Fehler-Cell aufrufen.
// Echte Go-Fehler (interne Fehler) werden durchgereicht.
func evalCatch(args *Cell, env *Env) (*Cell, error) {
  if args == nil || args.Type != LIST ||
    args.Cdr == nil || args.Cdr.Type != LIST {
    return nil, fmt.Errorf("catch: Syntax: (catch body handler)")
  }

  // Body auswerten – eigener Eval-Aufruf damit Fehler abgefangen werden kann
  result, err := Eval(args.Car, env)
  if err == nil {
    return result, nil  // kein Fehler → normal zurückgeben
  }

  // Nur LispError abfangen; Go-interne Fehler durchreichen
  lispErr, ok := err.(*LispError)
  if !ok {
    return nil, err
  }

  // Handler auswerten und mit Fehler-Cell aufrufen
  handler, err := Eval(args.Cdr.Car, env)
  if err != nil { return nil, err }
  return apply(handler, []*Cell{lispErr.Msg})
}
```

### Step 2: Case in Eval-Loop eintragen

In `lib/eval.go`, im nicht-Tail-Abschnitt des switch (nach `case "eval": ...`):

```go
case "catch": return evalCatch(expr.Cdr, env)
```

### Step 3: Verify PASS

```bash
cd /u/lisp-projekte/golisp && go build . && go run . -t
```

Erwartete Ausgabe für neue Tests:
```
(catch (error "oops") ...)                    => "caught: oops"
(catch (+ 1 2) ...)                           => 3
```

### Step 4: Commit

```bash
cd /u/lisp-projekte/golisp && git add lib/eval.go && git commit -m "feat: (catch body handler) Spezialform für Fehlerbehandlung"
```

---

## Task 3: REPL-Verbesserung + testlib.lisp Demo

**Files:**
- Modify: `main.go` (load-Fehler unterscheiden)
- Modify: `testlib.lisp`

### Step 1: Fehlerausgabe in load verbessern

In `main.go` wird bei Datei-Laden ein Fehler so ausgegeben:
```go
if _, err := lib.Eval(cell, env); err != nil {
  fmt.Println("ERR:", err); os.Exit(1)
}
```

Ersetze durch eine Version die bei `*lib.LispError` den Fehler ohne Präfix ausgibt:

```go
if _, err := lib.Eval(cell, env); err != nil {
  var le *lib.LispError
  if errors.As(err, &le) {
    fmt.Println("ERR:", le.Msg)
  } else {
    fmt.Println("ERR:", err)
  }
  os.Exit(1)
}
```

Füge `"errors"` zu den Imports in `main.go` hinzu:
```go
import (
  "errors"
  "fmt"
  "os"
  "golisp/lib"
)
```

### Step 2: Demo in testlib.lisp

Am Ende von `testlib.lisp` anfügen:

```lisp
; Error-Handling Demo
(defun safe-div (a b)
  (catch (if (= b 0) (error "Division durch 0") (/ a b))
         (lambda (e) (string-append "Fehler: " e))))

(println (safe-div 10 2))   ; "5"  → kein Fehler
(println (safe-div 10 0))   ; "Fehler: Division durch 0"
```

### Step 3: Verify

```bash
cd /u/lisp-projekte/golisp && go build . && go run . testlib.lisp
```

Letzte zwei Ausgabe-Zeilen:
```
5
"Fehler: Division durch 0"
```

### Step 4: Commit

```bash
cd /u/lisp-projekte/golisp && git add main.go testlib.lisp && git commit -m "feat: REPL LispError-Ausgabe + safe-div Demo"
```

---

## Verifikation REPL

```lisp
(error "test")                                ; → ERR: test
(catch (error "oops") (lambda (e) e))         ; → "oops"
(catch (+ 1 2) (lambda (e) "nein"))           ; → 3
(catch (/ 1 0) (lambda (e) "go-fehler"))      ; → ERR: /: Division durch 0
                                              ;    (Go-Fehler, kein LispError)
```
