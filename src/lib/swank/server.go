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
  "bufio"
  "errors"
  "fmt"
  "net"
  "os"
  "sync"

  "golisp2/src/lib"
)

// RunServer starts a SWANK server on the given address.
// Each connection gets its own environment.
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

// RunServerEnv starts a SWANK server whose connections all share env.
// exec runs each env access synchronously (gogui: on the GUI thread):
// it must execute f and return only once f has completed (happens-before),
// since callers read results written by f right after exec returns.
func RunServerEnv(addr string, env *lib.Env, exec func(func())) error {
  listener, err := net.Listen("tcp", addr)
  if err != nil {
    return fmt.Errorf("RunServerEnv: %w", err)
  }
  fmt.Fprintf(os.Stderr, "SWANK server on %s\n", listener.Addr())
  return ServeEnv(listener, env, exec)
}

// ServeEnv accepts connections on l; all share env, every env access
// goes through exec. exec must run its argument synchronously and return
// only after it has completed (happens-before) — HandleMessage's result is
// read right after exec returns. Returns when l is closed.
//
// RegisterSwankEnv/LoadSwankLisp run exactly once, here, instead of once
// per connection (unlike handleConn's per-connection env): on the shared
// env a second LoadSwankLisp would redefine every swank-* primitive and
// print a "REDEF" line per symbol. Each connection only switches where
// swank-send-event/print/println and :write-string output go, via
// swankSender.setCurrent — last-connected-wins, as before.
func ServeEnv(l net.Listener, env *lib.Env, exec func(func())) error {
  sender := &swankSender{}
  var setupErr error
  exec(func() {
    RegisterSwankEnv(env, sender.send)
    setupErr = LoadSwankLisp(env)
  })
  if setupErr != nil {
    return fmt.Errorf("ServeEnv: swank lisp error: %w", setupErr)
  }

  for {
    conn, err := l.Accept()
    if err != nil {
      if errors.Is(err, net.ErrClosed) {
        return nil
      }
      fmt.Fprintf(os.Stderr, "swank accept error: %v\n", err)
      continue
    }
    go serveSharedConn(conn, env, exec, sender)
  }
}

// swankSender forwards swank events to whichever connection connected
// last. current is guarded by mu: set by each new connection's goroutine,
// read by Lisp-side sends that may run on the exec thread (e.g. the GUI
// main thread) — both sides can touch it concurrently.
type swankSender struct {
  mu      sync.Mutex
  current func(*lib.Cell) error
}

func (s *swankSender) send(event *lib.Cell) error {
  s.mu.Lock()
  cur := s.current
  s.mu.Unlock()
  if cur == nil {
    return nil
  }
  return cur(event)
}

func (s *swankSender) setCurrent(f func(*lib.Cell) error) {
  s.mu.Lock()
  s.current = f
  s.mu.Unlock()
}

func handleConn(conn net.Conn) {
  env := lib.BaseEnv()
  if err := lib.LoadStdlib(env); err != nil {
    fmt.Fprintf(os.Stderr, "swank stdlib error: %v\n", err)
    conn.Close()
    return
  }
  serveConn(conn, env, func(f func()) { f() })
}

// serveConn registers the swank primitives and loads swank.lisp into env,
// then runs the message loop. Used by handleConn, where env is fresh per
// connection, so re-registering is correct (no prior bindings to redefine).
func serveConn(conn net.Conn, env *lib.Env, exec func(func())) {
  defer func() {
    if r := recover(); r != nil {
      fmt.Fprintf(os.Stderr, "swank conn panic: %v\n", r)
    }
    conn.Close()
  }()
  fmt.Fprintf(os.Stderr, "swank conn from %s\n", conn.RemoteAddr())

  var setupErr error
  exec(func() {
    RegisterSwankEnv(env, func(event *lib.Cell) error {
      return writeFrame(conn, event)
    })
    setupErr = LoadSwankLisp(env)
  })
  if setupErr != nil {
    fmt.Fprintf(os.Stderr, "swank lisp error: %v\n", setupErr)
    return
  }

  runMessageLoop(conn, env, exec)
}

// serveSharedConn runs one shared-env connection: no setup (ServeEnv did
// that once), just point sender at this connection and run the message
// loop. Does not clear sender.current on exit — last-connected-wins stays
// in effect until the next connection arrives, same as before this fix.
func serveSharedConn(conn net.Conn, env *lib.Env, exec func(func()), sender *swankSender) {
  defer func() {
    if r := recover(); r != nil {
      fmt.Fprintf(os.Stderr, "swank conn panic: %v\n", r)
    }
    conn.Close()
  }()
  fmt.Fprintf(os.Stderr, "swank conn from %s\n", conn.RemoteAddr())

  sender.setCurrent(func(event *lib.Cell) error {
    return writeFrame(conn, event)
  })

  runMessageLoop(conn, env, exec)
}

// runMessageLoop reads SWANK frames from conn and dispatches them through
// HandleMessage, writing back every resulting event. Shared by serveConn
// and serveSharedConn.
func runMessageLoop(conn net.Conn, env *lib.Env, exec func(func())) {
  br := bufio.NewReader(conn)
  for {
    msg, err := readFrame(br)
    if err != nil {
      fmt.Fprintf(os.Stderr, "swank read error from %s: %v\n", conn.RemoteAddr(), err)
      return
    }
    var events *lib.Cell
    exec(func() { events, err = HandleMessage(env, msg) })
    if err != nil {
      fmt.Fprintf(os.Stderr, "swank handle error: %v\n", err)
      continue
    }
    for _, event := range lib.CellToSlice(events) {
      if err := writeFrame(conn, event); err != nil {
        fmt.Fprintf(os.Stderr, "swank write error: %v\n", err)
        return
      }
    }
  }
}
