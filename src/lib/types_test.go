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
// Zahlen-Druckform (Python-repr-Regel): Festkomma für 1e-4 ≤ |x| < 1e16,
// sonst Exponent; Ganzzahlen ohne Punkt.
func TestFormatNumber(t *testing.T) {
  for _, tc := range []struct {
    n    float64
    want string
  }{
    {3, "3"}, {-42, "-42"}, {0, "0"},
    {1792441863.9679012, "1792441863.9679012"},
    {1234567.5, "1234567.5"}, {-0.5, "-0.5"},
    {0.0001, "0.0001"}, {0.00001, "1e-05"},
    {1.5e17, "150000000000000000"}, // ganzzahlig → ohne Punkt
    {1.5e19, "1.5e+19"},            // jenseits int64
    {1234567890123456.5, "1234567890123456.5"},
    {12345678901234567.5, "12345678901234568"},
  } {
    if got := FormatNumber(tc.n); got != tc.want {
      t.Errorf("FormatNumber(%v) = %q, want %q", tc.n, got, tc.want)
    }
  }
}

func TestNumberPrintWege(t *testing.T) {
  evalEq(t, `1792441863.5`, "1792441863.5")
  evalEq(t, `(format nil "~a" 1792441863.5)`, `"1792441863.5"`)
  evalEq(t, `(number->string 1792441863.5)`, `"1792441863.5"`)
  evalEq(t, `(number->string 0.00001)`, `"1e-05"`)
}
