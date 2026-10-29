//**********************************************************************
//  lib/listargs_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261029
//**********************************************************************
// Listen-Primitiven mit Vektoren/Punktpaaren: früher still () bzw. 0
// (die Schleife brach beim ersten Nicht-LIST ab), jetzt laut — außer
// sort, das wie CL auch Vektoren sortiert.
//**********************************************************************

package lib

import "testing"

func TestSortVektor(t *testing.T) {
  evalEq(t, `(sort (vector 3 1 2) <)`, "#(1 2 3)")
  // nicht-destruktiv wie bei Listen
  evalEq(t, `(let ((v (vector 3 1 2))) (sort v <) v)`, "#(3 1 2)")
  evalEq(t, `(sort (vector) <)`, "#()")
  evalEq(t, `(sort () <)`, "()")
  evalErr(t, `(sort 5 <)`)
  evalErr(t, `(sort (cons 2 1) <)`)
}

func TestListenArgumentLaut(t *testing.T) {
  evalErr(t, `(mapcar (lambda (x) x) (vector 1 2))`)
  evalErr(t, `(mapcar (lambda (x) x) (cons 1 2))`)
  evalErr(t, `(apply + (vector 1 2))`)
  evalErr(t, `(apply + 1 (cons 2 3))`)
  evalErr(t, `(append (vector 1) (list 2))`)
  evalErr(t, `(append (cons 1 2) (list 3))`)
  evalErr(t, `(shell-assoc 'a (vector 1))`)
  // Unverändert erlaubt
  evalEq(t, `(mapcar (lambda (x) x) ())`, "()")
  evalEq(t, `(apply + ())`, "0")
  evalEq(t, `(apply + 1 2 (list 3))`, "6")
  evalEq(t, `(append (list 1) (list 2) 3)`, "(1 2 . 3)")
  evalEq(t, `(append () (vector 1))`, "#(1)")
  evalEq(t, `(shell-assoc 'a '((a . 1)))`, "(a . 1)")
}
