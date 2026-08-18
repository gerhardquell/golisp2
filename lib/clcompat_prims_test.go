//**********************************************************************
//  lib/clcompat_prims_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-sonnet-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260818
//**********************************************************************
// Tests zu TODO 20260818 Gruppe B: sort, sqrt, get-universal-time.
package lib

import "testing"

func TestSort(t *testing.T) {
  evalStdlibEq(t, `(sort '(3 1 4 1 5 9 2 6) <)`, "(1 1 2 3 4 5 6 9)")
  evalStdlibEq(t, `(sort '(3 1 4 1 5) >)`, "(5 4 3 1 1)")
  evalStdlibEq(t, `(sort '() <)`, "()")
  evalStdlibEq(t, `(sort '(5) <)`, "(5)")
  // sort ist nicht-destruktiv (immutable cells) — Original bleibt unverändert
  evalStdlibEq(t, `(begin (define zz-xs '(3 1 2)) (sort zz-xs <) zz-xs)`, "(3 1 2)")
  // :key
  evalStdlibEq(t, `(sort '("bb" "a" "ccc") < :key string-length)`, `("a" "bb" "ccc")`)
  // Fehler im Prädikat wird durchgereicht
  evalStdlibErr(t, `(sort '(1 2) (lambda (a b) (error "boom")))`)
}

func TestSqrt(t *testing.T) {
  evalStdlibEq(t, `(sqrt 4)`, "2")
  evalStdlibEq(t, `(sqrt 2)`, "1.4142135623730951")
  evalStdlibEq(t, `(sqrt 0)`, "0")
  evalStdlibErr(t, `(sqrt -1)`)
}

func TestGetUniversalTime(t *testing.T) {
  got, err := evalStdlib(t, `(get-universal-time)`)
  if err != nil {
    t.Fatalf("get-universal-time: %v", err)
  }
  if got.Type != NUMBER {
    t.Fatalf("get-universal-time: Zahl erwartet, got %v", got)
  }
  // CL-Epoche 1900-01-01; Sekunden für 2020-01-01 liegen bei ~3.786e9 —
  // grober Plausibilitätscheck, kein exakter Zeitvergleich.
  if got.Num < 3.7e9 {
    t.Errorf("get-universal-time = %v, erwarte Wert nach 2020 (CL-Epoche)", got.Num)
  }
}
