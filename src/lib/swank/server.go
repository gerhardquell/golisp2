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
  "fmt"
  "net"
  "os"

  "golisp2/src/lib"
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
  defer func() {
    if r := recover(); r != nil {
      fmt.Fprintf(os.Stderr, "swank conn panic: %v\n", r)
    }
    conn.Close()
  }()
  fmt.Fprintf(os.Stderr, "swank conn from %s\n", conn.RemoteAddr())

  env := lib.BaseEnv()
  if err := lib.LoadStdlib(env); err != nil {
    fmt.Fprintf(os.Stderr, "swank stdlib error: %v\n", err)
    return
  }
  RegisterSwankEnv(env, func(event *lib.Cell) error {
    return writeFrame(conn, event)
  })
  if err := LoadSwankLisp(env); err != nil {
    fmt.Fprintf(os.Stderr, "swank lisp error: %v\n", err)
    return
  }

  br := bufio.NewReader(conn)
  for {
    msg, err := readFrame(br)
    if err != nil {
      fmt.Fprintf(os.Stderr, "swank read error from %s: %v\n", conn.RemoteAddr(), err)
      return
    }
    events, err := HandleMessage(env, msg)
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
