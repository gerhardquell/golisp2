//**********************************************************************
//  lib/stdlib_todo20260818b_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-sonnet-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260818
//**********************************************************************
// Tests zu TODO 20260818 Gruppe A (Lispbuch-Lückenanalyse, /u/golisp2-
// projekte/lispbuch): CL-Kompat-Ergänzungen in embed/stdlib.lisp.
package lib

import "testing"

func TestBuchluecken20260818Sequenzen(t *testing.T) {
  evalStdlibEq(t, `(remove-if-not (lambda (x) (> x 2)) '(1 2 3 4))`, "(3 4)")
  evalStdlibEq(t, `(remove-if (lambda (x) (> x 2)) '(1 2 3 4))`, "(1 2)")
  evalStdlibEq(t, `(remove 2 '(1 2 3 2 4))`, "(1 3 4)")
  evalStdlibEq(t, `(remove-duplicates '(1 2 1 3 2 4))`, "(1 2 3 4)")
  evalStdlibEq(t, `(butlast '(1 2 3 4))`, "(1 2 3)")
  evalStdlibEq(t, `(butlast '(1))`, "()")
  evalStdlibEq(t, `(copy-list '(1 2 3))`, "(1 2 3)")
  evalStdlibEq(t, `(equal? (copy-list '(1 2 3)) '(1 2 3))`, "t")
  evalStdlibEq(t, `(copy-tree '(1 (2 3) (4 (5 6))))`, "(1 (2 3) (4 (5 6)))")
  evalStdlibEq(t, `(make-list 3 :initial-element 'x)`, "(x x x)")
  evalStdlibEq(t, `(make-list 0)`, "()")
}

func TestBuchluecken20260818Plist(t *testing.T) {
  evalStdlibEq(t, `(getf '(a 1 b 2) 'b)`, "2")
  evalStdlibEq(t, `(getf '(a 1 b 2) 'c)`, "()")
  evalStdlibEq(t, `(getf '(a 1 b 2) 'c 'default)`, "default")
}

func TestBuchluecken20260818Zahlen(t *testing.T) {
  evalStdlibEq(t, `(zerop 0)`, "t")
  evalStdlibEq(t, `(zerop 1)`, "()")
  evalStdlibEq(t, `(begin (define zz-n 5) (incf zz-n) zz-n)`, "6")
  evalStdlibEq(t, `(begin (define zz-n 5) (incf zz-n 3) zz-n)`, "8")
  evalStdlibEq(t, `(begin (define zz-n 5) (decf zz-n) zz-n)`, "4")
  evalStdlibEq(t, `(begin (define zz-n 5) (decf zz-n 2) zz-n)`, "3")
}

func TestBuchluecken20260818Eql(t *testing.T) {
  evalStdlibEq(t, `(eql 5 5)`, "t")
  evalStdlibEq(t, `(eql 5 5.0)`, "t")
  evalStdlibEq(t, `(eql 'a 'a)`, "t")
  // eql ist bei Conses Pointer-Identität (wie eq), nicht strukturell —
  // zwei gleich aussehende, aber verschiedene Listen sind NICHT eql.
  evalStdlibEq(t, `(eql '(1 2) (list 1 2))`, "()")
  evalStdlibEq(t, `(begin (define zz-l '(1 2)) (eql zz-l zz-l))`, "t")
}

func TestBuchluecken20260818Makros(t *testing.T) {
  evalStdlibEq(t, `(assert (= 1 1))`, "t")
  evalStdlibErr(t, `(assert (= 1 2))`)
  evalStdlibEq(t, `(macroexpand-1 '(when t 1))`, "(if t (begin 1) ())")
}

func TestBuchluecken20260818Strings(t *testing.T) {
  evalStdlibEq(t, `(coerce "abc" 'list)`, `("a" "b" "c")`)
  evalStdlibEq(t, `(coerce (list "a" "b" "c") 'string)`, `"abc"`)
  evalStdlibEq(t, `(string-find "cd" "abcdef")`, "2")
  evalStdlibEq(t, `(string-find "xy" "abcdef")`, "()")
}

func TestBuchluecken20260818DestructuringBind(t *testing.T) {
  evalStdlibEq(t, `(destructuring-bind (a b c) '(1 2 3) (list c b a))`, "(3 2 1)")
  // expr wird genau einmal ausgewertet
  evalStdlibEq(t, `(begin
                     (define zz-calls 0)
                     (defun zz-src () (begin (setf zz-calls (+ zz-calls 1)) '(1 2)))
                     (destructuring-bind (a b) (zz-src) (list a b zz-calls)))`, "(1 2 1)")
}
