//**********************************************************************
//  lib/swank/framing_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260618
//**********************************************************************
// Tests für SWANK length-prefixed framing.
//**********************************************************************

package swank

import (
  "bufio"
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
  want := "000008(foo 42)"
  if got != want {
    t.Fatalf("got %q, want %q", got, want)
  }
}

func TestReadFrame(t *testing.T) {
  input := "000008(foo 42)(foo 42)"
  r := bufio.NewReader(strings.NewReader(input))
  cell, err := readFrame(r)
  if err != nil {
    t.Fatalf("readFrame failed: %v", err)
  }
  if cell.String() != "(foo 42)" {
    t.Fatalf("got %s, want (foo 42)", cell.String())
  }
}
