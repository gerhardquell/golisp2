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

func TestAssertDatum(t *testing.T) {
  evalStdlibEq(t, `(assert (= 1 1))`, "()")
  evalStdlibEq(t, `(trap (assert (= 1 2)) (lambda (e) e))`, `"assert failed: (= 1 2)"`)
  // CL-Beispiel: places ignoriert, datum + args als Format
  evalStdlibEq(t, `(let ((x 5)) (trap (assert (< x 3) (x) "x zu groß: ~a" x) (lambda (e) e)))`, `"x zu groß: 5"`)
  evalStdlibEq(t, `(trap (assert nil () "fest") (lambda (e) e))`, `"fest"`)
  evalStdlibEq(t, `(assert t (a b) "nie ~a" (error "args nur im Fehlerfall"))`, "()")
}

func TestEveryMehrereSequenzen(t *testing.T) {
  evalStdlibEq(t, `(list (every #'evenp (list 2 4)) (every #'evenp (list 2 3)) (every #'evenp ()))`, "(t () t)")
  evalStdlibEq(t, `(every #'evenp #(2 4))`, "t") // Vektoren über some
  evalStdlibEq(t, `(list (every #'< (list 1 2) (list 2 3)) (every #'< (list 1 5) (list 2 3)))`, "(t ())")
  // Ende bei der kürzesten Sequenz
  evalStdlibEq(t, `(every #'< (list 1 2 99) (list 2 3))`, "t")
  evalStdlibEq(t, `(every 'evenp (list 2 4))`, "t")
  evalStdlibEq(t, `(every #'number? (iota 200000))`, "t")
  evalStdlibErr(t, `(every #'evenp 5)`)
  evalStdlibErr(t, `(every #'evenp)`)
}

func TestMacroexpandVoll(t *testing.T) {
  kette := `(defmacro mx-m2 (x) (list '+ x 1)) (defmacro mx-m1 (x) (list 'mx-m2 x)) `
  evalStdlibEq(t, `(begin `+kette+`(macroexpand '(mx-m1 5)))`, "(+ 5 1)")
  evalStdlibEq(t, `(begin `+kette+`(macroexpand-1 '(mx-m1 5)))`, "(mx-m2 5)")
  evalStdlibEq(t, `(begin `+kette+`(multiple-value-list (macroexpand '(mx-m1 5))))`, "((+ 5 1) t)")
  evalStdlibEq(t, `(begin `+kette+`(multiple-value-list (macroexpand-1 '(mx-m1 5))))`, "((mx-m2 5) t)")
  evalStdlibEq(t, `(list (multiple-value-list (macroexpand '(+ 1 2))) (multiple-value-list (macroexpand-1 'x)))`,
    "(((+ 1 2) ()) (x ()))")
  evalStdlibErr(t, `(macroexpand-1)`)
  // endlose Expansion ist kein Fehler, aber abbrechbar — hier nur Fehler im Makro
  evalStdlibErr(t, `(begin (defmacro mx-kaputt () (error "nein")) (macroexpand '(mx-kaputt)))`)
}

func TestLoopBlockNil(t *testing.T) {
  evalStdlibEq(t, `(loop (return 5))`, "5")
  evalStdlibEq(t, `(let ((i 0)) (loop (set! i (+ i 1)) (when (> i 3) (return i))))`, "4")
  evalStdlibEq(t, `(loop for x in (list 1 2 3 4) do (when (= x 3) (return (* x 10))))`, "30")
  evalStdlibEq(t, `(loop for i from 1 to 3 collect i)`, "(1 2 3)")
  evalStdlibEq(t, `(block aussen (loop (return-from aussen 'raus)))`, "raus")
  evalStdlibEq(t, `(loop for i from 1 to 200000 do (when (= i 200000) (return i)))`, "200000")
  evalStdlibErr(t, `(loop for i from 1 to 3 sammle i)`)
  evalStdlibErr(t, `(loop (error "innen"))`)
}

func TestMemberKeyTest(t *testing.T) {
  // Default-Vergleich bleibt equal? (CL wäre eql)
  evalStdlibEq(t, `(list (member 2 (list 1 2 3)) (member "b" (list "a" "b")) (member (list 1) (list (list 1) 2)) (member 9 (list 1)))`,
    `((2 3) ("b") ((1) 2) ())`)
  evalStdlibEq(t, `(let ((l (list 1 2 3))) (eq (member 2 l) (cdr l)))`, "t")
  // CL-Beispiele
  evalStdlibEq(t, `(member 2 '((1 . 2) (3 . 4)) :test-not #'= :key #'cdr)`, "((3 . 4))")
  evalStdlibEq(t, `(member 'e '(a b c d))`, "()")
  evalStdlibEq(t, `(list (member 'b '((a 1) (b 2)) :key #'car) (member 'b '((a 1) (b 2)) :key 'car) (member 1 (list 1 2) :key nil))`,
    "(((b 2)) ((b 2)) (1 2))")
  evalStdlibEq(t, `(member 1 (list 1 2 3) :test #'<)`, "(2 3)")
  evalStdlibEq(t, `(length (member 199999 (iota 200000)))`, "1")
  evalStdlibErr(t, `(member 1 #(1 2))`)
  evalStdlibErr(t, `(member 1 5)`)
  evalStdlibErr(t, `(member 1 (list 1) :test #'= :test-not #'=)`)
  evalStdlibErr(t, `(member 1 (list 1) :foo 1)`)
  evalStdlibErr(t, `(member 1 (list 1) :key)`)
}

func TestRemoveKeyTest(t *testing.T) {
  evalStdlibEq(t, `(list (remove 2 (list 1 2 3 2)) (remove "a" (list "a" "b")) (remove 9 ()))`, `((1 3) ("b") ())`)
  // CL-Beispiele
  evalStdlibEq(t, `(remove 4 '(1 3 4 5 9))`, "(1 3 5 9)")
  evalStdlibEq(t, `(remove 3 '(1 2 4 1 3 4 5) :test #'>)`, "(4 3 4 5)")
  evalStdlibEq(t, `(list (remove 'a '((a 1) (b 2)) :key #'car) (remove 1 (list 1 2 1) :test-not #'=))`, "(((b 2)) (1 1))")
  evalStdlibEq(t, `(remove 1 #(1 2 1 3))`, "#(2 3)")
  evalStdlibEq(t, `(let ((l (list 1 2))) (list (eq (remove 9 l) l) l))`, "(() (1 2))")
  evalStdlibEq(t, `(length (remove 5 (iota 200000)))`, "199999")
  evalStdlibErr(t, `(remove 1 5)`)
  evalStdlibErr(t, `(remove 1 (list 1) :test #'= :test-not #'=)`)
}
