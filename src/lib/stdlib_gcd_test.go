//**********************************************************************
//  lib/stdlib_gcd_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260929
//**********************************************************************
// gcd (stdlib.lisp) nach CL: groesster gemeinsamer Teiler, nie negativ.
//**********************************************************************

package lib

import "testing"

func TestGcd(t *testing.T) {
  evalStdlibEq(t, `(gcd 12 18)`, "6")
  evalStdlibEq(t, `(gcd 18 12)`, "6")
  evalStdlibEq(t, `(gcd 7 2)`, "1")
  evalStdlibEq(t, `(gcd 5 0)`, "5")
  evalStdlibEq(t, `(gcd 0 5)`, "5")
  evalStdlibEq(t, `(gcd 0 0)`, "0")
}

func TestGcdNegativNieNegativ(t *testing.T) {
  evalStdlibEq(t, `(gcd -12 18)`, "6")
  evalStdlibEq(t, `(gcd 12 -18)`, "6")
  evalStdlibEq(t, `(gcd -4 0)`, "4")
}
