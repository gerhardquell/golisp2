//**********************************************************************
//  lib/stdlib_cl_abweichungen_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Opus 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261123
//**********************************************************************
// CL-Angleichung 0.38.0: butlast n, assert datum, every über mehrere
// Sequenzen, macroexpand bis zum Ende, loop mit block nil und einfacher
// Form, member/remove mit :test/:test-not/:key.
//**********************************************************************

package lib

import "testing"

func TestButlastN(t *testing.T) {
  evalStdlibEq(t, `(butlast (list 1 2 3))`, "(1 2)")
  evalStdlibEq(t, `(butlast '(a b c d) 2)`, "(a b)") // CL-Beispiel
  evalStdlibEq(t, `(list (butlast (list 1 2) 2) (butlast (list 1 2) 9) (butlast ()) (butlast (list 1 2) 0))`,
    "(() () () (1 2))")
  // Original bleibt, Ergebnis ist frisch
  evalStdlibEq(t, `(let ((l (list 1 2 3))) (list (eq (butlast l 0) l) (butlast l) l))`, "(() (1 2) (1 2 3))")
  evalStdlibEq(t, `(length (butlast (iota 200000) 3))`, "199997")
  evalStdlibErr(t, `(butlast (list 1 2) -1)`)
  evalStdlibErr(t, `(butlast (list 1 2) 1.5)`)
  evalStdlibErr(t, `(butlast (list 1 2) 1 2)`)
}
