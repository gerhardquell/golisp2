//**********************************************************************
//  lib/eval_mv_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260723
//**********************************************************************
// Multiple Values: Produktion (values, floor), Konsumformen und die
// Primärwert-Regel in Nicht-MV-Kontexten.
//**********************************************************************

package lib

import "testing"

func TestEvalValues(t *testing.T) {
  evalEq(t, `(values)`, "()")
  evalEq(t, `(values 1)`, "1")
  evalEq(t, `(values 1 2 3)`, "1") // Top-Level sieht Primärwert
}

func TestEvalPrimaryRegel(t *testing.T) {
  // Nicht-MV-Kontexte nehmen den Primärwert, Rest wird verworfen (CL)
  evalEq(t, `(+ (values 1 2) 10)`, "11")
  evalEq(t, `(list (values 1 2 3))`, "(1)")
  evalEq(t, `(if (values nil 99) 'ja 'nein)`, "nein")
  evalEq(t, `(let ((x (values 1 2))) x)`, "1")
  evalEq(t, `(setq mv-test-q (values 5 6))`, "5")
  evalEq(t, `((lambda (x) x) (values 7 8))`, "7")
}

func TestEvalMultipleValueFormen(t *testing.T) {
  evalEq(t, `(multiple-value-list (values 1 2 3))`, "(1 2 3)")
  evalEq(t, `(multiple-value-list (values))`, "()")
  evalEq(t, `(multiple-value-bind (a b) (floor 7 2) (list a b))`, "(3 1)")
  evalEq(t, `(multiple-value-bind (a b c) (values 1 2) (list a b c))`, "(1 2 ())")
  evalEq(t, `(multiple-value-bind () (values 1 2) 'keine)`, "keine")
  evalEq(t, `(multiple-value-call (function +) (values 1 2) 3 (values 4 5))`, "15")
  evalEq(t, `(multiple-value-prog1 (values 1 2) 'nw)`, "1")
  evalEq(t, `(multiple-value-list (multiple-value-prog1 (values 1 2) 99))`, "(1 2)")
  evalEq(t, `(let ((a 0) (b 0)) (multiple-value-setq (a b) (floor 17 5)) (list a b))`, "(3 2)")
  evalEq(t, `(nth-value 1 (floor 7 2))`, "1")
  evalEq(t, `(nth-value 5 (values 1 2))`, "()")
}

func TestEvalFloor(t *testing.T) {
  evalEq(t, `(floor 7 2)`, "3")
  evalEq(t, `(multiple-value-list (floor 7 2))`, "(3 1)")
  evalEq(t, `(multiple-value-list (floor -7 2))`, "(-4 1)")
  evalEq(t, `(multiple-value-list (floor 5))`, "(5 0)")
}

func TestEvalMVPropagierung(t *testing.T) {
  // MV durchquert Funktionsaufrufe unbeschadet (CL)
  evalEq(t, `
    (defun zwei-werte () (values 'erster 'zweiter))
    (list (zwei-werte) (multiple-value-list (zwei-werte)))`, "(erster (erster zweiter))")
  // unwind-protect bewahrt MV (Sonderfall: Cleanup darf Werte nicht stören)
  evalEq(t, `(multiple-value-list (unwind-protect (values 1 2) 'aufraeumen))`, "(1 2)")
}
