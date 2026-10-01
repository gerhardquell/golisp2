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
// exec runs each env access synchronously (gogui: on the GUI thread).
func RunServerEnv(addr string, env *lib.Env, exec func(func())) error {
  listener, err := net.Listen("tcp", addr)
  if err != nil {
    return fmt.Errorf("RunServerEnv: %w", err)
  }
  fmt.Fprintf(os.Stderr, "SWANK server on %s\n", listener.Addr())
  return ServeEnv(listener, env, exec)
}

// ServeEnv accepts connections on l; all share env, every env access
// goes through exec. Returns when l is closed.
func ServeEnv(l net.Listener, env *lib.Env, exec func(func())) error {
  for {
    conn, err := l.Accept()
    if err != nil {
      if errors.Is(err, net.ErrClosed) {
        return nil
      }
      fmt.Fprintf(os.Stderr, "swank accept error: %v\n", err)
      continue
    }
    go serveConn(conn, env, exec)
  }
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

// serveConn runs the SWANK message loop for one connection.
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
