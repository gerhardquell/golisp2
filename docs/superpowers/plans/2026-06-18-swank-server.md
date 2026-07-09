# GoLisp SWANK-Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Phase-1 SWANK server for GoLisp so Emacs (`M-x slime`) can connect, evaluate expressions in a REPL, and receive output/results.

**Architecture:** Go handles TCP listener and SWANK length-prefix framing, creates a fresh `*Env` per connection, and exposes per-connection primitives `swank-send-event` and `swank-print`. GoLisp implements the protocol logic (`swank-dispatch`, `connection-info`, `create-repl`, `listener-eval`) in `lib/swank/swank.lisp`.

**Tech Stack:** Go 1.22+, GoLisp runtime (`lib` package), SWANK length-prefixed UTF-8 S-expressions.

## Global Constraints

- **Language:** Go for the core, Lisp for extensions.
- **Indentation:** 2 spaces, no tabs.
- **File size:** max 1000 lines, split at 800.
- **Error format:** `fmt.Errorf("funktionsname: beschreibung")`.
- **File header:** Autor, CoAutor, Copyright, Erstellt (YYYYMMDD).
- **Primitives with env access?** Spezialform in `eval_*.go`; pure computation → primitive in `primitives.go` or registered via a `RegisterXxx(env)` helper.
- **Tests:** TDD — failing test first, then implementation, then commit.

---

## Task 1: SWANK Framing

**Files:**
- Create: `lib/swank/framing.go`
- Create: `lib/swank/framing_test.go`

**Interfaces:**
- Consumes: `lib.Read(string)`, `(*lib.Cell).String()`.
- Produces: `readFrame(r io.Reader) (*lib.Cell, error)` and `writeFrame(w io.Writer, cell *lib.Cell) error`.

SWANK frame format: `%06x\n<s-expression>` where `%06x` is the byte length of the UTF-8 encoded S-expression.

- [ ] **Step 1: Write the failing test**

```go
// lib/swank/framing_test.go
package swank

import (
  "bytes"
  "strings"
  "testing"

  "golisp2/lib"
)

func TestWriteFrame(t *testing.T) {
  var buf bytes.Buffer
  cell := lib.Cons(lib.MakeAtom("foo"), lib.Cons(lib.MakeNum(42), lib.MakeNil()))
  if err := writeFrame(&buf, cell); err != nil {
    t.Fatalf("writeFrame failed: %v", err)
  }
  got := buf.String()
  want := "00000d(foo 42)\n"
  if got != want {
    t.Fatalf("got %q, want %q", got, want)
  }
}

func TestReadFrame(t *testing.T) {
  input := "00000d(foo 42)\n(foo 42)"
  r := strings.NewReader(input)
  cell, err := readFrame(r)
  if err != nil {
    t.Fatalf("readFrame failed: %v", err)
  }
  if cell.String() != "(foo 42)" {
    t.Fatalf("got %s, want (foo 42)", cell.String())
  }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lib/swank -run TestWriteFrame -v`
Expected: FAIL — `writeFrame` not defined.

- [ ] **Step 3: Write minimal implementation**

```go
// lib/swank/framing.go
//**********************************************************************
//  lib/swank/framing.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260618
//**********************************************************************
// SWANK length-prefixed UTF-8 framing.
//**********************************************************************

package swank

import (
  "bufio"
  "fmt"
  "io"
  "strconv"

  "golisp2/lib"
)

// readFrame reads one SWANK length-prefixed S-expression.
func readFrame(r io.Reader) (*lib.Cell, error) {
  br := bufio.NewReader(r)
  line, err := br.ReadString('\n')
  if err != nil {
    return nil, fmt.Errorf("readFrame: %w", err)
  }
  line = line[:len(line)-1] // drop '\n'
  n, err := strconv.ParseInt(line, 16, 32)
  if err != nil {
    return nil, fmt.Errorf("readFrame: invalid length %q: %w", line, err)
  }
  payload := make([]byte, n)
  if _, err := io.ReadFull(br, payload); err != nil {
    return nil, fmt.Errorf("readFrame: short read: %w", err)
  }
  cell, err := lib.Read(string(payload))
  if err != nil {
    return nil, fmt.Errorf("readFrame: parse: %w", err)
  }
  return cell, nil
}

// writeFrame writes one SWANK length-prefixed S-expression.
func writeFrame(w io.Writer, cell *lib.Cell) error {
  payload := cell.String()
  frame := fmt.Sprintf("%06x\n%s", len(payload), payload)
  _, err := io.WriteString(w, frame)
  if err != nil {
    return fmt.Errorf("writeFrame: %w", err)
  }
  return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./lib/swank -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add lib/swank/framing.go lib/swank/framing_test.go
git commit -m "feat(swank): SWANK length-prefixed framing"
-m "Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 2: Per-Connection SWANK Primitives

**Files:**
- Create: `lib/swank/env.go`
- Create: `lib/swank/env_test.go`

**Interfaces:**
- Consumes: `lib.BaseEnv`, `lib.Env.Set`, `lib.makeFn` pattern.
- Produces: `RegisterSwankEnv(env *lib.Env, send func(*lib.Cell) error)` which registers `swank-send-event`, `swank-print`, `swank-println` in the supplied env.

The `send` callback is a closure over the TCP connection writer. It must be registered per connection, not globally.

- [ ] **Step 1: Write the failing test**

```go
// lib/swank/env_test.go
package swank

import (
  "testing"

  "golisp2/lib"
)

func TestRegisterSwankEnv(t *testing.T) {
  env := lib.BaseEnv()
  var sent *lib.Cell
  send := func(c *lib.Cell) error {
    sent = c
    return nil
  }
  RegisterSwankEnv(env, send)

  // (swank-send-event '(:write-string "hi" :repl-result))
  cell, err := lib.Read("(swank-send-event '(:write-string \"hi\" :repl-result))")
  if err != nil {
    t.Fatalf("read failed: %v", err)
  }
  _, err = lib.Eval(cell, env)
  if err != nil {
    t.Fatalf("eval failed: %v", err)
  }
  if sent == nil {
    t.Fatal("send callback was not invoked")
  }
  if sent.String() != "(:write-string \"hi\" :repl-result)" {
    t.Fatalf("unexpected event: %s", sent.String())
  }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lib/swank -run TestRegisterSwankEnv -v`
Expected: FAIL — `RegisterSwankEnv` and `swank-send-event` not defined.

- [ ] **Step 3: Write minimal implementation**

```go
// lib/swank/env.go
//**********************************************************************
//  lib/swank/env.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260618
//**********************************************************************
// Per-connection SWANK primitives: send-event, print, println.
//**********************************************************************

package swank

import (
  "fmt"
  "strings"

  "golisp2/lib"
)

// RegisterSwankEnv registers connection-bound SWANK primitives.
// send writes an event Cell to Emacs.
func RegisterSwankEnv(env *lib.Env, send func(*lib.Cell) error) {
  env.Set("swank-send-event", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    if len(args) < 1 {
      return nil, fmt.Errorf("swank-send-event: 1 Argument nötig")
    }
    if err := send(args[0]); err != nil {
      return nil, fmt.Errorf("swank-send-event: %w", err)
    }
    return lib.MakeNil(), nil
  }))

  env.Set("swank-print", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    return swankPrint(args, send, false)
  }))

  env.Set("swank-println", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    return swankPrint(args, send, true)
  }))

  env.Set("swank--value-string", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    if len(args) < 1 {
      return nil, fmt.Errorf("swank--value-string: 1 Argument nötig")
    }
    return lib.MakeStr(args[0].String()), nil
  }))
}

func makeFn(f func([]*lib.Cell) (*lib.Cell, error)) *lib.Cell {
  return &lib.Cell{Type: lib.FUNC, Fn: f}
}

func swankPrint(args []*lib.Cell, send func(*lib.Cell) error, newline bool) (*lib.Cell, error) {
  var b strings.Builder
  for i, a := range args {
    if i > 0 {
      b.WriteString(" ")
    }
    b.WriteString(a.String())
  }
  if newline {
    b.WriteString("\n")
  }
  event := lib.Cons(
    lib.MakeAtom(":write-string"),
    lib.Cons(
      lib.MakeStr(b.String()),
      lib.Cons(lib.MakeAtom(":repl-result"), lib.MakeNil()),
    ),
  )
  if err := send(event); err != nil {
    return nil, fmt.Errorf("swank-print: %w", err)
  }
  return lib.MakeNil(), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./lib/swank -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add lib/swank/env.go lib/swank/env_test.go
git commit -m "feat(swank): per-connection primitives send-event, print, println"
-m "Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 3: Go Dispatch Wrapper

**Files:**
- Create: `lib/swank/dispatch.go`
- Create: `lib/swank/dispatch_test.go`

**Interfaces:**
- Consumes: `lib.Eval`, `lib.Cons`, `lib.MakeAtom`, `lib.MakeNil`.
- Produces: `HandleMessage(env *lib.Env, msg *lib.Cell) (*lib.Cell, error)` returns the list of SWANK events produced by `swank-dispatch`.

- [ ] **Step 1: Write the failing test**

```go
// lib/swank/dispatch_test.go
package swank

import (
  "testing"

  "golisp2/lib"
)

func TestHandleMessage(t *testing.T) {
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  // Define a mock swank-dispatch in env
  _, err := lib.LoadString(`
    (defun swank-dispatch (msg)
      (list (list :return (list :ok :pong) 1)))
  `, env)
  if err != nil {
    t.Fatalf("load mock: %v", err)
  }

  msg := lib.Cons(lib.MakeAtom(":ping"), lib.MakeNil())
  result, err := HandleMessage(env, msg)
  if err != nil {
    t.Fatalf("HandleMessage failed: %v", err)
  }
  if result == nil || result.String() != "((:return (:ok :pong) 1))" {
    t.Fatalf("unexpected result: %v", result)
  }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./lib/swank -run TestHandleMessage -v`
Expected: FAIL — `HandleMessage` not defined.

- [ ] **Step 3: Write minimal implementation**

```go
// lib/swank/dispatch.go
//**********************************************************************
//  lib/swank/dispatch.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260618
//**********************************************************************
// Dispatch SWANK messages into GoLisp (swank-dispatch).
//**********************************************************************

package swank

import (
  "fmt"

  "golisp2/lib"
)

// HandleMessage evaluates (swank-dispatch msg) in env and returns the
// list of SWANK events that Go should send back to Emacs.
func HandleMessage(env *lib.Env, msg *lib.Cell) (*lib.Cell, error) {
  expr := lib.Cons(lib.MakeAtom("swank-dispatch"), lib.Cons(msg, lib.MakeNil()))
  result, err := lib.Eval(expr, env)
  if err != nil {
    return nil, fmt.Errorf("HandleMessage: %w", err)
  }
  return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./lib/swank -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add lib/swank/dispatch.go lib/swank/dispatch_test.go
git commit -m "feat(swank): dispatch wrapper (swank-dispatch)"
-m "Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 4: Lisp-Side Protocol Handlers

**Files:**
- Create: `lib/swank/swank.lisp`
- Create: `lib/swank/lisp.go` (embed loader)
- Create: `lib/swank/lisp_test.go` (smoke test)

**Interfaces:**
- Consumes: `swank-send-event`, `swank-print`, standard library helpers (`cadr`, `caddr`, etc.).
- Produces: `swank-dispatch`, `swank:connection-info`, `swank:create-repl`, `swank:listener-eval` handlers.

- [ ] **Step 1: Write the handler file**

```lisp
;; lib/swank/swank.lisp
;; ********************************************************************
;; SWANK protocol handlers for GoLisp.
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : claude sonnet 4.6
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260618
;; ********************************************************************

;; Redirect REPL output to Emacs.
(set! print swank-print)
(set! println swank-println)

(defun swank-dispatch (msg)
  (case (car msg)
    ((:emacs-rex)
     (let ((form (cadr msg))
           (pkg (caddr msg))
           (thread (cadddr msg))
           (id (car (cdr (cdr (cdr (cdr msg)))))))
       (handle-emacs-rex form pkg thread id)))
    (else (list (list :return (list :abort "unhandled message") 0)))))

(defun handle-emacs-rex (form pkg thread id)
  (let ((op (car form)))
    (cond
      ((equal? op 'swank:connection-info)
       (swank:connection-info id))
      ((equal? op 'swank:create-repl)
       (swank:create-repl id))
      ((equal? op 'swank:listener-eval)
       (swank:listener-eval (cadr form) id))
      (else
       (list (list :return (list :abort (string-append "unknown op: " (swank--value-string op))) id))))))

(defun swank:connection-info (id)
  (list (list :return
              (list :ok
                    (list :pid 0
                          :style :spawn
                          :encoding (list :coding-systems (list "utf-8-unix"))
                          :implementation (list :type "GoLisp"
                                                :version "0.2"
                                                :program "golisp2")
                          :machine (list :instance "unknown")
                          :package (list :name "USER")
                          :features (list)))
              id)))

(defun swank:create-repl (id)
  (list (list :return (list :ok (list "USER" "USER")) id)
        (list :new-package "USER" "USER")))

(defun swank:listener-eval (string id)
  (catch
    (let ((expr (read string)))
      (let ((result (eval expr)))
        (list (list :return (list :ok (swank--value-string result)) id))))
    (lambda (err)
      (list (list :return (list :abort err) id)))))
```

Note: `swank.lisp` lives inside `lib/swank/` so `//go:embed swank.lisp` works.

- [ ] **Step 2: Add embed loader**

```go
// lib/swank/lisp.go
//**********************************************************************
//  lib/swank/lisp.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260618
//**********************************************************************
// Embedded SWANK Lisp handlers.
//**********************************************************************

package swank

import (
  _ "embed"

  "golisp2/lib"
)

//go:embed swank.lisp
var swankSrc string

// LoadSwankLisp loads the embedded SWANK handler library into env.
func LoadSwankLisp(env *lib.Env) error {
  _, err := lib.LoadString(swankSrc, env)
  return err
}
```

Wait — `swank.lisp` is in the parent directory (`lib/`), not in `lib/swank/`. `//go:embed` cannot reference parent directories. Move `lib/swank.lisp` into `lib/swank/swank.lisp` or keep it in `lib/` and embed from `lib/swank/lisp.go` using `../swank.lisp`. Go `embed` supports `..`? I think yes, as of Go 1.16+? Actually `//go:embed` patterns cannot contain `..`. So we must place `swank.lisp` inside `lib/swank/`.

Decision: move handlers to `lib/swank/swank.lisp`.

- [ ] **Step 3: Adjust paths**

Create `lib/swank/swank.lisp` (content above) and update `lib/swank/lisp.go`:

```go
//go:embed swank.lisp
var swankSrc string
```

This works because `swank.lisp` is in the same package directory.

- [ ] **Step 4: Run a smoke test**

Run:
```bash
go test ./lib/swank -v
```
Add `lib/swank/lisp_test.go` with a smoke test:

```go
// lib/swank/lisp_test.go
package swank

import (
  "strings"
  "testing"

  "golisp2/lib"
)

func TestSwankLisp(t *testing.T) {
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  cell, err := lib.Read("(:emacs-rex (swank:connection-info) nil t 1)")
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  if result == nil || !strings.Contains(result.String(), ":return") {
    t.Fatalf("unexpected result: %v", result)
  }
}
```

Expected: PASS (after fixing any missing primitives).

- [ ] **Step 5: Commit**

```bash
git add lib/swank/swank.lisp lib/swank/lisp.go lib/swank/lisp_test.go
git commit -m "feat(swank): Lisp-side handlers (connection-info, create-repl, listener-eval)"
-m "Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 5: Refactor `lib/swank/server.go` for SWANK Mode

**Files:**
- Modify: `lib/swank/server.go`
- Delete or deprecate: `lib/swank/protocol.go` (old custom RPC)

**Interfaces:**
- Consumes: `readFrame`, `writeFrame`, `HandleMessage`, `RegisterSwankEnv`, `LoadSwankLisp`, `lib.BaseEnv`, `lib.LoadStdlib`.
- Produces: `RunServer(addr string) error`.

- [ ] **Step 1: Replace `server.go` contents**

```go
//**********************************************************************
//  lib/swank/server.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260301 (refactored 20260618)
//**********************************************************************
// SWANK server entry point for `golisp2 --swank`.
//**********************************************************************

package swank

import (
  "fmt"
  "net"
  "os"

  "golisp2/lib"
)

// RunServer starts a SWANK server on the given address.
func RunServer(addr string) error {
  listener, err := net.Listen("tcp", addr)
  if err != nil {
    return fmt.Errorf("RunServer: %w", err)
  }
  fmt.Fprintf(os.Stderr, "SWANK server on %s\n", listener.Addr())

  for {
    conn, err := listener.Accept()
    if err != nil {
      fmt.Fprintf(os.Stderr, "swank accept error: %v\n", err)
      continue
    }
    go handleConn(conn)
  }
}

func handleConn(conn net.Conn) {
  defer conn.Close()

  env := lib.BaseEnv()
  if err := lib.LoadStdlib(env); err != nil {
    fmt.Fprintf(os.Stderr, "swank stdlib error: %v\n", err)
    return
  }
  if err := LoadSwankLisp(env); err != nil {
    fmt.Fprintf(os.Stderr, "swank lisp error: %v\n", err)
    return
  }
  RegisterSwankEnv(env, func(event *lib.Cell) error {
    return writeFrame(conn, event)
  })

  for {
    msg, err := readFrame(conn)
    if err != nil {
      return
    }
    events, err := HandleMessage(env, msg)
    if err != nil {
      fmt.Fprintf(os.Stderr, "swank handle error: %v\n", err)
      continue
    }
    for _, event := range lib.CellToSlice(events) {
      if err := writeFrame(conn, event); err != nil {
        return
      }
    }
  }
}
```

- [ ] **Step 2: Remove old protocol.go**

```bash
rm lib/swank/protocol.go
```

- [ ] **Step 3: Build and run unit tests**

Run: `go test ./lib/swank -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add lib/swank/server.go lib/swank/protocol.go
git commit -m "feat(swank): wire framing + dispatch into server.go"
-m "Removes old custom RPC protocol, replaces with SWANK framing."
-m "Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 6: `--swank` CLI Flag

**Files:**
- Modify: `main.go`

**Interfaces:**
- Consumes: `flag.String` for `--swank`, `swank.RunServer(addr)`.
- Produces: When `--swank` is provided, main starts SWANK server instead of REPL/stdin.

- [ ] **Step 1: Write the failing integration expectation**

No new Go test file here; behavior is covered by Task 7 integration test. Build must still pass.

- [ ] **Step 2: Modify `main.go`**

Add near the other flags:

```go
swankFlag := flag.String("swank", "", "SWANK-Server starten (Format: host:port, z.B. 127.0.0.1:4005)")
```

After stdlib loading and before test-mode handling:

```go
// SWANK-Server Modus: golisp2 --swank [host:port]
if *swankFlag != "" {
  addr := *swankFlag
  if !strings.Contains(addr, ":") {
    addr = "127.0.0.1:" + addr
  }
  if err := swank.RunServer(addr); err != nil {
    fmt.Fprintln(os.Stderr, "swank server error:", err)
    os.Exit(1)
  }
  os.Exit(0)
}
```

Add import `"golisp2/lib/swank"`.

- [ ] **Step 3: Build to verify**

Run: `go build .`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "feat(cli): add --swank flag to start SWANK server"
-m "Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 7: Integration Test

**Files:**
- Create: `lib/swank/integration_test.go`

**Interfaces:**
- Consumes: `RunServer`, `readFrame`, `writeFrame`.
- Produces: A test that starts the server, sends `connection-info`, and validates the response.

- [ ] **Step 1: Write the test**

```go
// lib/swank/integration_test.go
package swank

import (
  "net"
  "strings"
  "testing"
  "time"

  "golisp2/lib"
)

func TestSwankServerConnectionInfo(t *testing.T) {
  listener, err := net.Listen("tcp", "127.0.0.1:0")
  if err != nil {
    t.Fatalf("listen: %v", err)
  }
  defer listener.Close()

  go func() {
    for {
      conn, err := listener.Accept()
      if err != nil {
        return
      }
      go handleConn(conn)
    }
  }()

  conn, err := net.Dial("tcp", listener.Addr().String())
  if err != nil {
    t.Fatalf("dial: %v", err)
  }
  defer conn.Close()

  // Send connection-info request
  msg := lib.Cons(lib.MakeAtom(":emacs-rex"),
    lib.Cons(lib.Cons(lib.MakeAtom("swank:connection-info"), lib.MakeNil()),
      lib.Cons(lib.MakeNil(),
        lib.Cons(lib.MakeAtom("t"),
          lib.Cons(lib.MakeNum(1), lib.MakeNil())))))
  if err := writeFrame(conn, msg); err != nil {
    t.Fatalf("writeFrame: %v", err)
  }

  // Set read deadline to avoid hanging
  conn.SetReadDeadline(time.Now().Add(2 * time.Second))

  resp, err := readFrame(conn)
  if err != nil {
    t.Fatalf("readFrame: %v", err)
  }
  s := resp.String()
  if !strings.Contains(s, ":return") || !strings.Contains(s, "GoLisp") {
    t.Fatalf("unexpected response: %s", s)
  }
}
```

- [ ] **Step 2: Run test**

Run: `go test ./lib/swank -run TestSwankServerConnectionInfo -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add lib/swank/integration_test.go
git commit -m "test(swank): integration test for connection-info"
-m "Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Self-Review Checklist

- **Spec coverage:**
  - SWANK framing ✅ Task 1
  - `--swank` flag ✅ Task 6
  - `swank-dispatch` in Lisp ✅ Task 3
  - `connection-info`/`create-repl`/`listener-eval` ✅ Task 4
  - `swank-send-event` primitive ✅ Task 2
  - Output redirect ✅ Task 4 (`set! print swank-print`)
  - Per-connection Env ✅ Task 5
  - Error handling (abort) ✅ Task 4
  - Tests ✅ Tasks 1, 2, 3, 4, 7
- **Placeholder scan:** No TBD/TODO/"implement later".
- **Type consistency:** `readFrame`/`writeFrame` use `*lib.Cell` everywhere; `HandleMessage` returns `*lib.Cell` (event list); `RegisterSwankEnv` uses `func(*lib.Cell) error`.

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-18-swank-server.md`.

Two execution options:

1. **Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.
2. **Inline Execution** — Execute tasks in this session using `superpowers:executing-plans`, batch execution with checkpoints.

Which approach?
