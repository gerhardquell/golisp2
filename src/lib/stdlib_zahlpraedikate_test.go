//**********************************************************************
//  lib/stdlib_zahlpraedikate_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Opus 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261020
//**********************************************************************
// CL-Zahlenprädikate (stdlib.lisp): evenp, oddp, plusp, minusp, 1+, 1-.
//**********************************************************************

package lib

import "testing"

func TestEvenpOddp(t *testing.T) {
  evalStdlibEq(t, `(list (evenp 4) (evenp 3) (evenp 0) (evenp -2) (evenp -3))`, "(t () t t ())")
  evalStdlibEq(t, `(list (oddp 3) (oddp 4) (oddp -3) (oddp -2))`, "(t () t ())")
  // CL: nur für ganze Zahlen definiert
  evalStdlibErr(t, `(evenp 1.5)`)
  evalStdlibErr(t, `(oddp "3")`)
  evalStdlibEq(t, `(trap (evenp 1.5) (lambda (e) e))`, `"evenp: ganze Zahl erwartet, got 1.5"`)
}

func TestPluspMinusp(t *testing.T) {
  evalStdlibEq(t, `(list (plusp 1) (plusp 0) (plusp -1) (plusp 0.5))`, "(t () () t)")
  evalStdlibEq(t, `(list (minusp -1) (minusp 0) (minusp 1) (minusp -0.5))`, "(t () () t)")
  evalStdlibErr(t, `(plusp "1")`)
}

func TestEinsPlusMinus(t *testing.T) {
  evalStdlibEq(t, `(list (1+ 1) (1- 1) (1+ 1.5) (1- 0))`, "(2 0 2.5 -1)")
  evalStdlibEq(t, `(mapcar '1+ '(1 2 3))`, "(2 3 4)")
  evalStdlibErr(t, `(1+ "a")`)
}
