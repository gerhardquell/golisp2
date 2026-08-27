//**********************************************************************
//  lib/stdin_read_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260813
//**********************************************************************
// (read-line)-Primitiv (TODO.md Punkt 3, 20260813). os.Stdin wird per
// os.Pipe() + SetStdinReader/ResetStdinReader umgeleitet, damit der Test
// nicht vom echten Terminal-stdin abhängt.
//**********************************************************************
package lib

import (
  "os"
  "testing"
)

func TestReadLinePrimitive(t *testing.T) {
  r, w, err := os.Pipe()
  if err != nil {
    t.Fatalf("os.Pipe: %v", err)
  }
  SetStdinReader(r)
  defer ResetStdinReader()

  go func() {
    w.WriteString("hallo welt\n")
    w.WriteString("zweite zeile\n")
    w.Close()
  }()

  // gepufferter Reader muss über zwei Aufrufe hinweg erhalten bleiben
  evalEq(t, `(read-line)`, `"hallo welt"`)
  evalEq(t, `(read-line)`, `"zweite zeile"`)
}

func TestReadLinePrimitiveEOF(t *testing.T) {
  r, w, err := os.Pipe()
  if err != nil {
    t.Fatalf("os.Pipe: %v", err)
  }
  w.Close()
  SetStdinReader(r)
  defer ResetStdinReader()

  evalErr(t, `(read-line)`)
}

func TestReadLinePrimitiveArgs(t *testing.T) {
  evalErr(t, `(read-line 1)`)
}
