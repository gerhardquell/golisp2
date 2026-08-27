//**********************************************************************
//  lib/stdlib_iter_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260813
//**********************************************************************
// dolist/dotimes mit 2-Element-Bindungsform (var lst) bzw. (var n), ohne
// Ergebnisform. Beide Makros greifen intern per caddr auf die fehlende
// dritte Position zu — das läuft über (car '()) auf der Cdr-Cdr der
// Bindungsliste. Vor dem Fix in primitives.go (TODO.md Punkt 1, 20260813)
// warf das einen Fehler ("car: Liste erwartet") statt nil zu liefern.
// evalStdlib/evalStdlibEq: siehe lib/stdlib_reduce_setf_test.go.
//**********************************************************************
package lib

import "testing"

func TestDolist2ElementBinding(t *testing.T) {
  evalStdlibEq(t,
    `(begin (define zz-sum 0) (dolist (x '(1 2 3)) (set! zz-sum (+ zz-sum x))) zz-sum)`,
    "6")
}

func TestDotimes2ElementBinding(t *testing.T) {
  evalStdlibEq(t,
    `(begin (define zz-cnt 0) (dotimes (i 5) (set! zz-cnt (+ zz-cnt 1))) zz-cnt)`,
    "5")
}
