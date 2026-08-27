//**********************************************************************
//  lib/types_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260624
//**********************************************************************

package lib

import "testing"

func TestCellSourceLocationDefaults(t *testing.T) {
  c := MakeAtom("foo")
  if c.SrcFile() != "" {
    t.Fatalf("SrcFile default leer erwartet, got %q", c.SrcFile())
  }
  if c.SrcLine != 0 {
    t.Fatalf("SrcLine default 0 erwartet, got %d", c.SrcLine)
  }
}