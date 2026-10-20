//**********************************************************************
//  lib/vector_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Opus 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261020
//**********************************************************************
// Tests für Vektoren (vector.go) und ihre Anbindung an length, coerce,
// setf, type-of/typep/subtypep (stdlib.lisp, types.lisp).
//**********************************************************************

package lib

import "testing"

func TestVectorGrundlagen(t *testing.T) {
  evalStdlibEq(t, `(make-array 3)`, "#(() () ())")
  evalStdlibEq(t, `(make-array 3 :initial-element 0)`, "#(0 0 0)")
  evalStdlibEq(t, `(make-array 0)`, "#()")
  evalStdlibEq(t, `(vector 1 "a" 'b)`, `#(1 "a" b)`)
  evalStdlibEq(t, `(aref (vector 1 2 3) 2)`, "3")
  evalStdlibEq(t, `(let ((v (make-array 3 :initial-element 0))) (list (aset v 1 42) v))`, "(42 #(0 42 0))")
  evalStdlibEq(t, `(let ((v (vector 1 2))) (list (vector-push-extend 7 v) v))`, "(2 #(1 2 7))")
  evalStdlibEq(t, `(vector (vector 1) (list 2))`, "#(#(1) (2))")
}

// Vektoren sind mutabel und werden geteilt, nicht kopiert.
func TestVectorGeteilt(t *testing.T) {
  evalStdlibEq(t, `(let* ((v (vector 1 2)) (w v)) (aset w 0 9) (aref v 0))`, "9")
  evalStdlibEq(t, `(let ((v (vector 1))) (defun f (x) (vector-push-extend 5 x)) (f v) v)`, "#(1 5)")
}

func TestVectorSetfLengthCoerce(t *testing.T) {
  evalStdlibEq(t, `(let ((v (vector 1 2 3))) (list (setf (aref v 0) 'x) v))`, "(x #(x 2 3))")
  evalStdlibEq(t, `(length (vector 1 2 3))`, "3")
  evalStdlibEq(t, `(length (make-array 0))`, "0")
  evalStdlibEq(t, `(length '(1 2))`, "2")
  evalStdlibEq(t, `(coerce (vector 1 2) 'list)`, "(1 2)")
  evalStdlibEq(t, `(coerce '(1 2) 'vector)`, "#(1 2)")
  evalStdlibEq(t, `(let ((v (vector 1))) (eq v (coerce v 'vector)))`, "t")
}

func TestVectorTypen(t *testing.T) {
  evalStdlibEq(t, `(type-of (vector 1))`, "vector")
  evalStdlibEq(t, `(list (typep (vector) 'vector) (typep (vector) 'sequence) (typep (vector) 'atom) (typep (vector) 'list))`, "(t t t ())")
  evalStdlibEq(t, `(multiple-value-list (subtypep 'vector 'sequence))`, "(t t)")
  evalStdlibEq(t, `(typecase (vector) (list 'l) (vector 'v))`, "v")
}

// CL equal: Vektoren nur bei Identität gleich.
func TestVectorGleichheit(t *testing.T) {
  evalStdlibEq(t, `(let ((v (vector 1))) (list (eq v v) (equal? v v)))`, "(t t)")
  evalStdlibEq(t, `(list (eq (vector 1) (vector 1)) (equal? (vector 1) (vector 1)))`, "(() ())")
}

func TestVectorFehler(t *testing.T) {
  for _, src := range []string{
    `(make-array -1)`,
    `(make-array 1.5)`,
    `(make-array '(2 3))`,
    `(make-array 2 :fill-pointer 0)`,
    `(make-array 2 :initial-element)`,
    `(aref (vector 1 2) 2)`,
    `(aref (vector 1 2) -1)`,
    `(aref (vector 1 2) 0.5)`,
    `(aref '(1 2) 0)`,
    `(aset (vector 1) 1 0)`,
    `(aset (vector 1) 0)`,
    `(vector-push-extend 1 '(1))`,
    `(coerce 1 'vector)`,
  } {
    evalStdlibErr(t, src)
  }
  evalStdlibEq(t, `(trap (aref (vector 1 2) 5) (lambda (e) e))`, `"aref: Index 5 außerhalb 0..1"`)
}
