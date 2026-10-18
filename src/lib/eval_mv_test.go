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

import (
  "os"
  "path/filepath"
  "testing"
)

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

// Spezialformen, die das Ergebnis eines Teilausdrucks inspizieren,
// packen MV über Primary aus (Task 1, 0.2.1).
func TestEvalMVLock(t *testing.T) {
  evalEq(t, `(let ((h (make-hash-table)))
               (puthash "mu" h (lock-make))
               (lock (gethash "mu" h) 7))`, "7")
}

func TestEvalMVCase(t *testing.T) {
  evalEq(t, `(case (values 'a 2) ((a) "ok") (t "nein"))`, `"ok"`)
}

func TestEvalMVCatchThrowTag(t *testing.T) {
  evalEq(t, `(catch (values 'a 1) (throw 'a 5))`, "5")
  evalEq(t, `(catch 'a (throw (values 'a 1) 5))`, "5")
}

func TestEvalMVTrapHandler(t *testing.T) {
  evalEq(t, `(trap (error "x") (values (lambda (e) 42) 2))`, "42")
}

func TestEvalMVEval(t *testing.T) {
  evalEq(t, `(eval (values '(+ 1 2) 3))`, "3")
}

func TestEvalMVBoundMakunbound(t *testing.T) {
  evalEq(t, `(bound? (values 'car 1))`, "t")
  evalEq(t, `(define mv-unb 1) (makunbound (values 'mv-unb 1))`, "mv-unb")
}

func TestEvalMVMacroexpand(t *testing.T) {
  evalEq(t, `(defmacro mv-mm () '(+ 1 2)) (macroexpand (values '(mv-mm) 1))`, "(+ 1 2)")
  evalEq(t, `(defmacro mv-mm () '(+ 1 2)) (macroexpand-all (values '(list (mv-mm)) 1))`, "(list (+ 1 2))")
}

func TestEvalMVMakroExpansion(t *testing.T) {
  // Makro, dessen Body MV liefert: Expansion = Primärwert (CL)
  evalEq(t, `(defmacro mv-m2 () (values '(+ 1 2) 9)) (mv-m2)`, "3")
  evalEq(t, `(defmacro mv-m3 () (values '(mv-m2) 9)) (defmacro mv-m2 () '(+ 1 2))
             (macroexpand-all '(list (mv-m3)))`, "(list (+ 1 2))")
}

func TestEvalMVFunktionsposition(t *testing.T) {
  evalEq(t, `((values (lambda (x) (* x 2)) 1) 5)`, "10")
}

func TestEvalMVMultipleValueCallFn(t *testing.T) {
  evalEq(t, `(multiple-value-call (values (function list) 1) 1 2)`, "(1 2)")
}

func TestEvalMVNthValueIndex(t *testing.T) {
  evalEq(t, `(nth-value (values 1 0) (values 'a 'b))`, "b")
}

func TestEvalMVCondOhneBody(t *testing.T) {
  evalEq(t, `(cond ((values 1 2)))`, "1")
}

func TestEvalMVQuasiquote(t *testing.T) {
  evalEq(t, "(equal? `(,(values 1 2)) '(1))", "t")
  evalEq(t, "`(a ,@(values '(1 2) 3))", "(a 1 2)")
}

func TestEvalMVExec(t *testing.T) {
  evalEq(t, `(exec (values "echo" 1) param: (values "hi" 1) stdout: out) out`, `"hi\n"`)
  evalEq(t, `(exec "cat" stdin: (values "x" 1) env: (values "A=1" 1) timeout: (values 5 1) stdout: out) out`, `"x"`)
}

func TestEvalMVLoad(t *testing.T) {
  f := filepath.Join(t.TempDir(), "mv.lisp")
  if err := os.WriteFile(f, []byte("(+ 40 2)"), 0o644); err != nil {
    t.Fatal(err)
  }
  evalEq(t, `(load (values "`+f+`" 1))`, "42")
}

// Bindungen und Datenstrukturen speichern nur den Primärwert (wie let,
// setq, do*): sonst sieht multiple-value-list später alle Werte.
func TestEvalMVBindungen(t *testing.T) {
  evalEq(t, `(do ((x (values 1 2))) (t (multiple-value-list x)))`, "(1)")
  evalEq(t, `(do ((x 0 (values 1 2)) (i 0 (+ i 1))) ((= i 1) (multiple-value-list x)))`, "(1)")
  evalEq(t, `(define mv-dd (values 1 2)) (multiple-value-list mv-dd)`, "(1)")
  evalEq(t, `(setq* mv-ss (values 1 2)) (multiple-value-list mv-ss)`, "(1)")
  evalEq(t, `((lambda (&optional (o (values 1 2))) (multiple-value-list o)))`, "(1)")
  evalEq(t, `((lambda (&key (k (values 1 2))) (multiple-value-list k)))`, "(1)")
  evalEq(t, `(apply (lambda (&optional (o (values 1 2))) (multiple-value-list o)) '())`, "(1)")
  evalEq(t, `(apply (lambda (&key (k (values 1 2))) (multiple-value-list k)) '())`, "(1)")
  evalEq(t, `(parfunc mv-r (values 1 2)) (multiple-value-list (car mv-r))`, "(1)")
}

// Task 4b (0.5.1): Klausel ohne Körper, prog1/prog2 und vordere or-Formen
// liefern nur den Primärwert; die letzte or-Form und and reichen MV durch.
func TestEvalMVCondOhneKoerperPrimaer(t *testing.T) {
  evalEq(t, `(multiple-value-list (cond ((values 1 2))))`, "(1)")
}

func TestEvalMVProg1(t *testing.T) {
  evalEq(t, `(multiple-value-list (prog1 (values 1 2) 3))`, "(1)")
}

func TestEvalMVProg2(t *testing.T) {
  evalEq(t, `(multiple-value-list (prog2 0 (values 1 2) 3))`, "(1)")
}

func TestEvalMVOrVordereForm(t *testing.T) {
  evalEq(t, `(multiple-value-list (or (values 1 2) 3))`, "(1)")
}

func TestEvalMVOrLetzteForm(t *testing.T) {
  evalEq(t, `(multiple-value-list (or nil (values 1 2)))`, "(1 2)")
}

func TestEvalMVAndCharakterisierung(t *testing.T) {
  evalEq(t, `(multiple-value-list (and 1 (values 1 2)))`, "(1 2)")
}
