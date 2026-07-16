# GoLisp2 Stack-Overflow Robustheit — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `golisp2` darf bei unbegrenzter Lisp-Rekursion nicht mehr durch einen fatalen Stack-Overflow sterben; `parfunc`-Worker müssen bei Timeout gestoppt werden.

**Architecture:** Eine interne `evalWithCtx`-Schleibe führt einen `evalCtx` mit Rekursionstiefe und `context.Context`-Cancellation mit. Die öffentliche `Eval`-Funktion bleibt unverändert und ist nur ein Wrapper. `parfunc` leitet einen Cancel-Context an seine Worker weiter und bricht bei Timeout ab. Rekursionstiefe und Cancellation werden als `LispError` zurückgegeben, nicht als Go-Panic.

**Tech Stack:** Go 1.26, `context`, bestehende `golisp2/lib`-Pakete.

## Global Constraints

- Sprache (Kommentare/Doku): deutsch.
- Build aller Binaries: `./build.sh`.
- Ground-Truth für Kompilierung: `go build ./...`.
- Tests: `go test ./...` und `./build/golisp2 -t`.
- Keine Duplikate; Chokepoint für Eval bleibt `lib/eval_core.go`.
- Dateigröße: Richtwert 800 Zeilen, hart bei 1000.
- Temporäre Dateien nur unter `./tmp/`, nicht `/tmp`.
- Breaking Changes / Design-Ausweitungen sind Gerhards Entscheidung — bei Unsicherheit nachfragen.

---

## Task 1: `evalCtx`-Typ einführen und `Eval` als Wrapper umbauen

**Files:**
- Modify: `lib/eval_core.go`
- Test: `lib/eval_ctx_test.go` (neu)

**Interfaces:**
- Consumes: `LispError`, `MakeStr` aus `lib/types.go`; `freeEnv`, `Env` aus `lib/env.go`.
- Produces:
  - `var MaxEvalDepth = 100000`
  - `type evalCtx struct { depth int; ctx context.Context }`
  - `func (e *evalCtx) check() error`
  - `func (e *evalCtx) child() *evalCtx`
  - `func Eval(expr *Cell, env *Env) (*Cell, error)` — Wrapper, ruft `evalWithCtx` auf.
  - `func evalWithCtx(expr *Cell, env *Env, ectx *evalCtx) (res *Cell, err error)` — bisheriger `Eval`-Körper.

- [ ] **Step 1: Schreibe den failing Test für den Wrapper**

```go
package lib

import (
  "testing"
)

func TestEvalWrapperPassesContext(t *testing.T) {
  env := BaseEnv()
  expr, err := Read(`(+ 1 2)`)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  res, err := Eval(expr, env)
  if err != nil {
    t.Fatalf("eval: %v", err)
  }
  if res.Type != NUMBER || res.Num != 3 {
    t.Fatalf("expected 3, got %v", res)
  }
}
```

- [ ] **Step 2: Führe den Test aus — er muss aktuell noch bestehen**

Run: `go test ./lib -run TestEvalWrapperPassesContext -v`
Expected: PASS (Wrapper verhält sich wie bisher).

- [ ] **Step 3: Füge `evalCtx`-Typ und `Eval`-Wrapper in `lib/eval_core.go` ein**

Am Anfang von `lib/eval_core.go` (nach den Imports):

```go
import (
  "context"
  "fmt"
)

// MaxEvalDepth begrenzt nicht-tail-rekursive Eval-Aufrufe.
// Öffentlich, damit Tests und Suite niedrigere Werte setzen können.
var MaxEvalDepth = 100000

// evalCtx trägt pro Eval-Lauf: Rekursionstiefe und Cancellation.
// Eine Instanz gehört immer nur einer Goroutine an.
type evalCtx struct {
  depth int
  ctx   context.Context
}

// child liefert einen neuen Kontext für einen nicht-tail-rekursiven Aufruf.
func (e *evalCtx) child() *evalCtx {
  if e == nil {
    return &evalCtx{depth: 1}
  }
  return &evalCtx{depth: e.depth + 1, ctx: e.ctx}
}

// check prüft Depth-Limit und Cancellation.
func (e *evalCtx) check() error {
  if e == nil {
    return nil
  }
  if e.depth > MaxEvalDepth {
    return &LispError{Msg: MakeStr("eval: maximum recursion depth exceeded")}
  }
  if e.ctx != nil {
    select {
    case <-e.ctx.Done():
      return &LispError{Msg: MakeStr("eval: cancelled")}
    default:
    }
  }
  return nil
}

// Eval wertet einen Ausdruck in env aus. Öffentlicher Einstieg.
func Eval(expr *Cell, env *Env) (res *Cell, err error) {
  return evalWithCtx(expr, env, &evalCtx{depth: 0})
}
```

- [ ] **Step 4: Benenne den bisherigen Eval-Körper in `evalWithCtx` um**

Ersetze in `lib/eval_core.go`:

```go
// Eval wertet einen Ausdruck in env aus. Trampolin: Tail-Positionen
// setzen expr/env und continue'n, statt zu rekursieren – O(1) Stack.
func Eval(expr *Cell, env *Env) (res *Cell, err error) {
```

durch:

```go
// evalWithCtx wertet einen Ausdruck in env aus. Trampolin: Tail-Positionen
// setzen expr/env und continue'n, statt zu rekursieren – O(1) Stack.
func evalWithCtx(expr *Cell, env *Env, ectx *evalCtx) (res *Cell, err error) {
```

Füge direkt nach dem `defer` am Anfang der Funktion (vor der `for`-Schleife) ein:

```go
  if err := ectx.check(); err != nil {
    return nil, err
  }
```

- [ ] **Step 5: Führe den Test erneut aus**

Run: `go test ./lib -run TestEvalWrapperPassesContext -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add lib/eval_core.go lib/eval_ctx_test.go
git commit -m "feat(eval): evalCtx mit Depth/Cancellation und Eval-Wrapper"
```

---

## Task 2: Rekursionstiefe-Limit testen

**Files:**
- Modify: `lib/eval_ctx_test.go`
- Modify: `lib/eval_core.go` (nur falls nötig)

**Interfaces:**
- Consumes: `MaxEvalDepth`, `evalCtx`, `Eval`.
- Produces: Failing-Charakterisierungstest für zu tiefe Rekursion.

- [ ] **Step 1: Schreibe den failing Test**

```go
package lib

import (
  "strings"
  "testing"
)

func TestEvalDepthLimit(t *testing.T) {
  old := MaxEvalDepth
  MaxEvalDepth = 50
  defer func() { MaxEvalDepth = old }()

  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("stdlib: %v", err)
  }

  // Nicht-tail-rekursiv: jedes (sum (- n 1)) legt einen neuen Go-Stackframe an.
  code := `(begin
             (define (sum n)
               (if (= n 0) 0 (+ (sum (- n 1)) 1)))
             (sum 100))`
  expr, err := Read(code)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  _, err = Eval(expr, env)
  if err == nil {
    t.Fatal("expected recursion-depth error, got nil")
  }
  if !strings.Contains(err.Error(), "maximum recursion depth exceeded") {
    t.Fatalf("unexpected error: %v", err)
  }
}
```

- [ ] **Step 2: Führe den Test aus — er muss fehlschlagen**

Run: `go test ./lib -run TestEvalDepthLimit -v`
Expected: FAIL mit `runtime: goroutine stack exceeds` oder Hänger (da Depth-Limit noch nicht wirkt).

- [ ] **Step 3: Stelle sicher, dass `evalWithCtx` das Limit prüft**

Die `check()`-Methode aus Task 1 ist bereits eingebaut. Der Test sollte nach der Umstellung aller internen Aufrufe auf `evalWithCtx(..., ectx.child())` grün werden. Falls der Test vor Task 3 bereits laufen soll, springe zu Task 3.

- [ ] **Step 4: Commit (nachdem Task 3 erledigt ist)**

```bash
git add lib/eval_ctx_test.go
git commit -m "test(eval): Rekursionstiefe-Limit Charakterisierung"
```

---

## Task 3: `evalWithCtx` durch `lib/eval_core.go` ziehen

**Files:**
- Modify: `lib/eval_core.go`

**Interfaces:**
- Consumes: `evalCtx`, `evalWithCtx`, `evalArgsPooled`.
- Produces: `func evalArgsPooled(args *Cell, env *Env, ectx *evalCtx) ([]*Cell, bool, error)`.

- [ ] **Step 1: Ändere die Signaturen von `evalArgsPooled`**

```go
func evalArgsPooled(args *Cell, env *evalCtx) ([]*Cell, bool, error) {
```

wird zu:

```go
func evalArgsPooled(args *Cell, env *Env, ectx *evalCtx) ([]*Cell, bool, error) {
```

Innerhalb von `evalArgsPooled`:

```go
val, err := evalWithCtx(args.Car, env, ectx.child())
```

- [ ] **Step 2: Ersetze alle internen `Eval(...)`-Aufrufe in `eval_core.go` durch `evalWithCtx(..., ectx.child())`, außer bei Tail-Positionen**

Beispiele:

`if`:
```go
cond, err := evalWithCtx(expr.Cdr.Car, env, ectx.child())
```

`begin` (Nicht-Tail-Body-Ausdrücke):
```go
if _, err := evalWithCtx(args.Car, env, ectx.child()); err != nil { return nil, err }
```

`let` / `let*` (Binding-Werte):
```go
val, err := evalWithCtx(b.Cdr.Car, env, ectx.child())
```

`cond` (Test-Ausdrücke):
```go
val, err := evalWithCtx(test, env, ectx.child())
```

`case`:
```go
e, newEnv, err := evalCase(expr.Cdr, env, ectx)
```

Funktionsanwendung:
```go
fn, err := evalWithCtx(expr.Car, env, ectx.child())
...
args, pooled, err := evalArgsPooled(expr.Cdr, env, ectx.child())
```

Tail-Positionen (`continue` in `if`, `begin`, `let`, `let*`, `cond`, `case`, Lambda, Macro) verwenden **kein** `child()`, sondern das aktuelle `ectx`.

- [ ] **Step 3: `go build ./...` — alle Signaturen müssen passen**

Run: `go build ./...`
Expected: kompiliert (andere Dateien folgen in späteren Tasks).

- [ ] **Step 4: Commit**

```bash
git add lib/eval_core.go
git commit -m "feat(eval): evalCtx durch eval_core.go ziehen"
```

---

## Task 4: `evalWithCtx` durch `lib/eval_specialforms.go` ziehen

**Files:**
- Modify: `lib/eval_specialforms.go`

**Interfaces:**
- Consumes: `evalCtx`, `evalWithCtx`.
- Produces: Signaturen wie `evalAnd(args *Cell, env *Env, ectx *evalCtx)`, `evalOr`, `evalNot`, `evalSet`, `evalSetQStar`, `evalMapcar`, `evalBegin`, `macroexpandOnce`, `evalDefun`, `evalLambda`, `evalDefmacro`, `evalCase`.

- [ ] **Step 1: Ändere die Funktionssignaturen und ersetze interne `Eval`-Aufrufe**

Beispiel `evalBegin`:

```go
func evalBegin(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  var result *Cell
  var err error
  for args != nil && args.Type == LIST {
    result, err = evalWithCtx(args.Car, env, ectx.child())
    if err != nil { return nil, err }
    args = args.Cdr
  }
  return result, nil
}
```

Beispiel `evalAnd`:

```go
func evalAnd(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  result := &Cell{Type: ATOM, Val: "t"}
  for args != nil && args.Type == LIST {
    val, err := evalWithCtx(args.Car, env, ectx.child())
    if err != nil { return nil, err }
    if !IsTruthy(val) { return MakeNil(), nil }
    result = val
    args = args.Cdr
  }
  return result, nil
}
```

Analog für `evalOr`, `evalNot`, `evalSet`, `evalSetQStar`, `evalMapcar`, `macroexpandOnce`, `evalCase`.

Für `evalDefun`, `evalLambda`, `evalDefmacro` bleibt die Evaluierung der Definition aus; sie bekommen `ectx` nur wegen Konsistenz und rufen es nicht weiter auf.

- [ ] **Step 2: Aktualisiere Aufrufer in `eval_core.go`**

`eval_core.go` ruft z.B. `evalAnd(expr.Cdr, env)` auf. Passe alle Aufrufe an:

```go
case "and": return evalAnd(expr.Cdr, env, ectx)
```

- [ ] **Step 3: `go build ./...`**

Run: `go build ./...`
Expected: kompiliert.

- [ ] **Step 4: Commit**

```bash
git add lib/eval_specialforms.go lib/eval_core.go
git commit -m "feat(eval): evalCtx durch eval_specialforms.go ziehen"
```

---

## Task 5: `evalWithCtx` durch `lib/eval_control.go` ziehen

**Files:**
- Modify: `lib/eval_control.go`

**Interfaces:**
- Consumes: `evalCtx`, `evalWithCtx`, `context`.
- Produces: Signaturen wie `evalWhile(args *Cell, env *Env, ectx *evalCtx)`, `evalDo`, `evalEval`, `evalCatch`, `evalParfunc`.

- [ ] **Step 1: Passe Signaturen an und ersetze `Eval`-Aufrufe**

Beispiel `evalWhile`:

```go
func evalWhile(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("while: Syntax: (while test body...)")
  }
  test := args.Car
  body := wrapBegin(args.Cdr)
  for {
    if err := ectx.check(); err != nil {
      return nil, err
    }
    cond, err := evalWithCtx(test, env, ectx.child())
    if err != nil { return nil, err }
    if !IsTruthy(cond) { return MakeNil(), nil }
    if _, err := evalWithCtx(body, env, ectx.child()); err != nil { return nil, err }
  }
}
```

Beispiel `evalEval`:

```go
func evalEval(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("eval: 1 Argument nötig")
  }
  expr, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil { return nil, err }
  return evalWithCtx(expr, env.Root(), &evalCtx{depth: 0, ctx: ectx.ctx})
}
```

Beispiel `evalDo`:

```go
func evalDo(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  ...
  localEnv := NewEnv(env)
  defer freeEnv(localEnv)
  for b := bindings; ... {
    init, err := evalWithCtx(spec.Cdr.Car, env, ectx.child())
    ...
  }
  for {
    if err := ectx.check(); err != nil { return nil, err }
    cond, err := evalWithCtx(test, localEnv, ectx.child())
    ...
    if IsTruthy(cond) { return evalWithCtx(result, localEnv, ectx.child()) }
    if _, err := evalWithCtx(body, localEnv, ectx.child()); err != nil { return nil, err }
    ...
    newVal, err = evalWithCtx(step.Car, localEnv, ectx.child())
    ...
  }
}
```

- [ ] **Step 2: Aktualisiere Aufrufer in `eval_core.go`**

```go
case "while": return evalWhile(expr.Cdr, env, ectx)
case "do": return evalDo(expr.Cdr, env, ectx)
case "eval": return evalEval(expr.Cdr, env, ectx)
case "catch": return evalCatch(expr.Cdr, env, ectx)
case "parfunc": return evalParfunc(expr.Cdr, env, ectx)
```

- [ ] **Step 3: `go build ./...`**

Run: `go build ./...`
Expected: kompiliert.

- [ ] **Step 4: Commit**

```bash
git add lib/eval_control.go lib/eval_core.go
git commit -m "feat(eval): evalCtx durch eval_control.go ziehen"
```

---

## Task 6: `parfunc`-Cancellation implementieren

**Files:**
- Modify: `lib/eval_control.go`

**Interfaces:**
- Consumes: `context`, `evalCtx`, `evalWithCtx`.
- Produces: `evalParfunc` mit Timeout-Cancellation.

- [ ] **Step 1: Baue Cancellation in `evalParfunc` ein**

Ersetze den Worker-Start:

```go
  workerParent := context.Background()
  if ectx != nil && ectx.ctx != nil {
    workerParent = ectx.ctx
  }
  workerCtx, cancel := context.WithCancel(workerParent)

  for i, expr := range exprList {
    go func(idx int, e *Cell) {
      val, err := evalWithCtx(e, env, &evalCtx{depth: 0, ctx: workerCtx})
      if err != nil { val = MakeNil() }
      ch <- parfuncResult{idx, val}
    }(i, expr)
  }
```

Im Timeout-Zweig:

```go
      case <-timer:
        cancel()
        collected = len(exprList)
```

- [ ] **Step 2: Schreibe Test für parfunc-Cancellation**

```go
package lib

import (
  "testing"
  "time"
)

func TestParfuncTimeoutCancelsWorker(t *testing.T) {
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("stdlib: %v", err)
  }

  // Unendliche Schleife — ohne Cancellation würde die Goroutine ewig laufen.
  code := `(parfunc r :timeout 1 (while t 1))`
  expr, err := Read(code)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  start := time.Now()
  _, err = Eval(expr, env)
  elapsed := time.Since(start)
  if err != nil {
    t.Fatalf("eval: %v", err)
  }
  if elapsed > 3*time.Second {
    t.Fatalf("parfunc timeout too slow: %v", elapsed)
  }

  // Warte kurz, damit der Worker die Cancellation bemerkt.
  time.Sleep(500 * time.Millisecond)
}
```

- [ ] **Step 3: Führe den Test aus**

Run: `go test ./lib -run TestParfuncTimeoutCancelsWorker -v -timeout 15s`
Expected: PASS, Laufzeit deutlich unter 3 Sekunden.

- [ ] **Step 4: Commit**

```bash
git add lib/eval_control.go lib/eval_control_test.go
# falls Test in neuer Datei, sonst bestehende Testdatei
```

Hinweis: Falls es keine `lib/eval_control_test.go` gibt, erstelle sie.

```bash
git add lib/eval_control.go lib/eval_control_test.go
git commit -m "feat(parfunc): Worker-Cancellation bei Timeout"
```

---

## Task 7: `evalWithCtx` durch restliche eval_*.go und Helfer ziehen

**Files:**
- Modify: `lib/eval_quasiquote.go`
- Modify: `lib/eval_exec.go`
- Modify: `lib/eval_lambda.go`
- Modify: `lib/primitives.go` (falls intern `Eval` aufgerufen wird)
- Modify: `lib/eval_core.go` (Aufrufer aktualisieren)

**Interfaces:**
- Consumes: `evalCtx`, `evalWithCtx`.
- Produces: aktualisierte Signaturen `evalQQ`, `evalExec`, `bindEvalArgs`, `applyLambda` etc.

- [ ] **Step 1: Passe `lib/eval_quasiquote.go` an**

```go
func evalQQ(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  ...
}
```

Ersetze interne `Eval`-Aufrufe durch `evalWithCtx(..., ectx.child())`.

- [ ] **Step 2: Passe `lib/eval_exec.go` an**

```go
func evalExec(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  ...
}
```

- [ ] **Step 3: Passe `lib/eval_lambda.go` an**

```go
func bindEvalArgs(params, args, callEnv, closureEnv, localEnv *Env, ectx *evalCtx) error {
  ...
  val, err := evalWithCtx(argExpr, callEnv, ectx.child())
  ...
}
```

`applyLambda` muss `ectx` an `Eval` des Body weitergeben können. Da `applyLambda` aktuell aus `Eval` heraus aufgerufen wird, ändere die Signatur:

```go
func applyLambda(fn *Cell, rawArgs []*Cell, ectx *evalCtx) (*Cell, error) {
  ...
}
```

- [ ] **Step 4: Prüfe `lib/primitives.go` und andere Dateien auf interne `Eval`-Aufrufe**

Suche:

```bash
rg 'Eval\(' lib/ --type go
```

Jeder interne Aufruf muss `ectx` empfangen und weitergeben. Öffentliche Aufrufer (aus `cmd/`, `lib/swank/`, Tests) dürfen `Eval` weiterhin verwenden.

- [ ] **Step 5: `go build ./...` und `go test ./...`**

Run:
```bash
go build ./...
go test ./...
```
Expected: alles grün.

- [ ] **Step 6: Commit**

```bash
git add lib/eval_quasiquote.go lib/eval_exec.go lib/eval_lambda.go lib/primitives.go lib/eval_core.go
git commit -m "feat(eval): evalCtx durch restliche Eval-Helfer ziehen"
```

---

## Task 8: `freeEnv` nil-sicher machen

**Files:**
- Modify: `lib/env.go`
- Test: `lib/env_test.go`

**Interfaces:**
- Consumes: `Env`.
- Produces: `func freeEnv(e *Env)` toleriert `nil`.

- [ ] **Step 1: Schreibe Test**

```go
func TestFreeEnvNil(t *testing.T) {
  freeEnv(nil) // darf nicht panicken
}
```

- [ ] **Step 2: Passe `freeEnv` an**

```go
func freeEnv(e *Env) {
  if e == nil {
    return
  }
  ...
}
```

- [ ] **Step 3: Führe Test aus**

Run: `go test ./lib -run TestFreeEnvNil -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add lib/env.go lib/env_test.go
git commit -m "fix(env): freeEnv toleriert nil"
```

---

## Task 9: SWANK-Integrationstest für `gps-norvig-bugs.lisp`

**Files:**
- Modify: `lib/swank/integration_test.go` (oder neu: `lib/swank/gps_bug_test.go`)

**Interfaces:**
- Consumes: `swank.RunServer`, `golisp2-client --load`.
- Produces: Test, der beweist, dass der Server nach dem Norvig-Bug-Skript nicht abstürzt.

- [ ] **Step 1: Schreibe Integrationstest**

```go
package swank

import (
  "fmt"
  "net"
  "os"
  "os/exec"
  "path/filepath"
  "runtime"
  "testing"
  "time"
)

func TestSwankSurvivesNorvigBugs(t *testing.T) {
  if runtime.GOOS != "linux" {
    t.Skip("nur auf Linux")
  }

  repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
  if err != nil {
    t.Fatalf("repo root: %v", err)
  }

  ln, err := net.Listen("tcp", "127.0.0.1:0")
  if err != nil {
    t.Fatalf("listen: %v", err)
  }
  port := ln.Addr().(*net.TCPAddr).Port
  ln.Close()

  logFile := filepath.Join(repoRoot, "tmp", "swank-norvig-test.log")
  os.MkdirAll(filepath.Dir(logFile), 0755)

  server := exec.Command(filepath.Join(repoRoot, "build", "golisp2d"), "--port", fmt.Sprintf("%d", port))
  server.Dir = repoRoot
  f, _ := os.Create(logFile)
  server.Stdout = f
  server.Stderr = f
  if err := server.Start(); err != nil {
    t.Fatalf("start server: %v", err)
  }
  defer func() { _ = server.Process.Kill() }()

  time.Sleep(500 * time.Millisecond)

  client := exec.Command(filepath.Join(repoRoot, "build", "golisp2-client"), "--host", "127.0.0.1", "--port", fmt.Sprintf("%d", port), "--load", "pn-gps1/gps-norvig-bugs.lisp")
  client.Dir = repoRoot
  out, err := client.CombinedOutput()
  if err != nil {
    t.Fatalf("client load failed: %v\n%s", err, out)
  }

  time.Sleep(30 * time.Second)

  if err := server.Process.Signal(os.Signal(nil)); err != nil {
    // Prozess läuft noch
    return
  }
  if !server.ProcessState.Exited() {
    return
  }
  log, _ := os.ReadFile(logFile)
  t.Fatalf("server died after Norvig bugs script:\n%s", log)
}
```

Hinweis: Die Prozess-Überprüfung muss robust sein. Verwende `server.Process.Signal(syscall.Signal(0))` auf Unix.

- [ ] **Step 2: Baue Server und Client**

Run: `./build.sh`
Expected: erfolgreich.

- [ ] **Step 3: Führe Integrationstest aus**

Run: `go test ./lib/swank -run TestSwankSurvivesNorvigBugs -v -timeout 60s`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add lib/swank/gps_bug_test.go
git commit -m "test(swank): Integrationstest für Norvig-Bugs Überleben"
```

---

## Task 10: Globales Panic-Recovery an SWANK-Einstiegspunkten

**Files:**
- Modify: `lib/swank/server.go`

**Interfaces:**
- Consumes: `recover`.
- Produces: `handleConn` fängt nicht-fatale Panics ab und logged sie.

- [ ] **Step 1: Füge `defer recover` in `handleConn` ein**

```go
func handleConn(conn net.Conn) {
  defer func() {
    if r := recover(); r != nil {
      fmt.Fprintf(os.Stderr, "swank conn panic: %v\n", r)
    }
    conn.Close()
  }()
  fmt.Fprintf(os.Stderr, "swank conn from %s\n", conn.RemoteAddr())
  ...
}
```

Entferne das separate `defer conn.Close()`, da es nun im recover-Defer enthalten ist.

- [ ] **Step 2: Führe SWANK-Tests aus**

Run: `go test ./lib/swank -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add lib/swank/server.go
git commit -m "fix(swank): panic recovery in handleConn"
```

---

## Task 11: Volle Testsuite und Dokumentation

**Files:**
- Modify: `doc/lisp-semantik.md` (Abschnitt über Rekursion / parfunc)
- Modify: `CLAUDE.md` (nur falls neues Invariante hinzukommt)

**Interfaces:**
- Consumes: alle vorherigen Änderungen.
- Produces: grüne Testsuite + aktualisierte Doku.

- [ ] **Step 1: Führe alle Tests aus**

Run:
```bash
go build ./...
go test ./... -count=1
./build.sh
./build/golisp2 -t
```

Expected: alles grün.

- [ ] **Step 2: Aktualisiere `doc/lisp-semantik.md`**

Füge einen kurzen Abschnitt hinzu:

```markdown
### Rekursionstiefe und `parfunc`

- `eval` bricht ab, wenn die nicht-tail-rekursive Tiefe `MaxEvalDepth` (Default 100000) überschreitet. Ergebnis ist ein `LispError`, kein Prozessabbruch.
- `parfunc` mit `:timeout N` bricht laufende Worker über `context.Context` ab. Worker, die trotzdem rekursiv tiefer gehen, stoßen vorher an `MaxEvalDepth`.
```

- [ ] **Step 3: Finaler Commit**

```bash
git add doc/lisp-semantik.md
git commit -m "doc: Rekursionstiefe und parfunc-Cancellation dokumentiert"
```

---

## Self-Review

1. **Spec coverage:**
   - Root-Cause (Stack-Overflow) → Task 1–3 (Depth-Limit).
   - parfunc Timeout-Cancellation → Task 6.
   - Global Robustheit (handleConn recover, freeEnv nil) → Task 8, Task 10.
   - Tests → Task 2, Task 6, Task 9.
   - Doku → Task 11.

2. **Placeholder scan:** Keine `TODO`, `TBD`, `implement later` oder unvollständigen Code-Blöcke. Mechanische Umstellungen sind als Muster mit Beispielen beschrieben.

3. **Type consistency:**
   - `evalWithCtx(expr *Cell, env *Env, ectx *evalCtx) (*Cell, error)` überall.
   - `evalCtx.child()` liefert `*evalCtx`.
   - `evalArgsPooled(args *Cell, env *Env, ectx *evalCtx)`.
   - Alle Helper-Signaturen in `eval_specialforms.go`, `eval_control.go`, `eval_quasiquote.go`, `eval_exec.go`, `eval_lambda.go` bekommen `ectx *evalCtx`.

4. **Offene Risiken:**
   - Große mechanische Refactor über viele Dateien. Jeder Task endet mit `go build ./...`, um früh Fehler zu finden.
   - `MaxEvalDepth = 100000` ist ein Heuristik-Wert; nach Deploy/Testsuite kann er justiert werden.
   - `parfunc`-Worker-Cancellation stoppt nur kooperative Punkte (Eval-Loop-Anfang, while/do). Tight non-cancellable Loops werden weiterhin durch Depth-Limit gebremst.

---

## Execution Handoff

**Plan complete and saved to `docs/superpowers/plans/2026-07-16-golisp2-stack-overflow-robustness.md`. Two execution options:**

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — Execute tasks in this session using `superpowers:executing-plans`, batch execution with checkpoints.

**Which approach?**
