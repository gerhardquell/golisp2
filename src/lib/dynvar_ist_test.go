//**********************************************************************
//  lib/dynvar_ist_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261117
//**********************************************************************
// Charakterisierung vor dynamischen Variablen (Spec 2026-11-17):
// hält das Verhalten von 0.32.0 fest, in dem let/progv nur lexikalisch
// binden. Folgeschritte ändern Erwartungen nur gemäß Spec §5.
//**********************************************************************
package lib

import "testing"

const dynIstVorspann = `(defvar *x* 1) (defun g () *x*) `

func TestDynIstLetUndProgv(t *testing.T) {
  // Spec §3.2
  evalStdlibEq(t, dynIstVorspann+`(let ((*x* 2)) (g))`, "2")
  // Spec §3.2
  evalStdlibEq(t, dynIstVorspann+`(progv '(*x*) '(3) (g))`, "3")
  evalStdlibEq(t, dynIstVorspann+`(progv '(*x*) '(3) *x*)`, "3")
}

func TestDynIstSetqInLet(t *testing.T) {
  evalStdlibEq(t, dynIstVorspann+`(let ((*x* 2)) (setq *x* 5))`, "5")
  evalStdlibEq(t, dynIstVorspann+`(let ((*x* 2)) (setq *x* 5)) *x*`, "1")
}

func TestDynIstParfuncInLet(t *testing.T) {
  // ändert sich: Spec §3.5 (Worker erben die Bindung → (5 5))
  evalStdlibEq(t, dynIstVorspann+`(let ((*x* 5)) (parfunc r (g) (g)) r)`, "(1 1)")
}

func TestDynIstClosureAusLet(t *testing.T) {
  // Spec §3.2 (Closure merkt sich die Bindung nicht)
  evalStdlibEq(t, dynIstVorspann+`(defun mach () (let ((*x* 2)) (lambda () *x*))) (funcall (mach))`, "1")
  evalStdlibEq(t, dynIstVorspann+`(defun mach () (let ((*x* 2)) (lambda () *x*))) ((mach))`, "1")
}

func TestDynIstDefvarAufGebundenenNamen(t *testing.T) {
  evalStdlibEq(t, `(define *y* 7) (defvar *y* 99) *y*`, "7")
}
