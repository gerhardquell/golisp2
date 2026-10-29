//**********************************************************************
//  lib/sequence_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Opus 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261029
//**********************************************************************
// Tests für den Sequenz-Kern (sequence.go) und eql als Go-Primitive.
//**********************************************************************

package lib

import "testing"

func TestSeqElt(t *testing.T) {
  evalStdlibEq(t, `(elt '(a b c) 1)`, "b")
  evalStdlibEq(t, `(elt (vector 1 2) 1)`, "2")
  evalStdlibErr(t, `(elt '(1) 5)`)
  evalStdlibErr(t, `(elt () 0)`)
  evalStdlibErr(t, `(elt "abc" 0)`)
  evalStdlibErr(t, `(elt '(1 2) -1)`)
}

func TestSeqSubseq(t *testing.T) {
  evalStdlibEq(t, `(subseq '(1 2 3 4) 1 3)`, "(2 3)")
  evalStdlibEq(t, `(subseq (vector 1 2 3) 1)`, "#(2 3)")
  evalStdlibEq(t, `(subseq '(1 2) 2)`, "()")
  evalStdlibEq(t, `(subseq (vector) 0)`, "#()")
  evalStdlibErr(t, `(subseq '(1 2) 1 5)`)
  evalStdlibErr(t, `(subseq '(1 2) 2 1)`)
  evalStdlibErr(t, `(subseq "abc" 1)`)
}

func TestSeqCopySeq(t *testing.T) {
  evalStdlibEq(t, `(copy-seq (vector 1 2))`, "#(1 2)")
  evalStdlibEq(t, `(let ((v (vector 1 2))) (eq v (copy-seq v)))`, "()")
  evalStdlibEq(t, `(let ((v (vector 1 2)) (w ())) (setq w (copy-seq v)) (aset w 0 9) v)`, "#(1 2)")
  evalStdlibEq(t, `(copy-seq '(1 2))`, "(1 2)")
  evalStdlibEq(t, `(copy-seq ())`, "()")
  evalStdlibErr(t, `(copy-seq 5)`)
}

func TestSeqConcatenate(t *testing.T) {
  evalStdlibEq(t, `(concatenate 'list '(1) (vector 2) ())`, "(1 2)")
  evalStdlibEq(t, `(concatenate 'vector '(1) (vector 2))`, "#(1 2)")
  evalStdlibEq(t, `(concatenate 'vector)`, "#()")
  evalStdlibErr(t, `(concatenate 'string "a")`)
  evalStdlibErr(t, `(concatenate 'list "a")`)
}

func TestSeqMap(t *testing.T) {
  evalStdlibEq(t, `(map 'list + '(1 2 3) (vector 10 20))`, "(11 22)")
  evalStdlibEq(t, `(map 'vector (lambda (x) (* x 2)) '(1 2))`, "#(2 4)")
  evalStdlibEq(t, `(map nil print '(1))`, "()")
  evalStdlibEq(t, `(map 'list '1+ '(1))`, "(2)")
  evalStdlibEq(t, `(map 'list + ())`, "()")
  evalStdlibErr(t, `(map 'string + '(1))`)
  evalStdlibErr(t, `(map 'list + "ab")`)
  evalStdlibErr(t, `(map 'list +)`)
}

func TestSeqSome(t *testing.T) {
  evalStdlibEq(t, `(some (lambda (x) (and (> x 1) (* x 10))) '(1 2 3))`, "20")
  evalStdlibEq(t, `(some evenp (vector 1 3))`, "()")
  evalStdlibEq(t, `(some 'evenp '(1 2))`, "t")
  evalStdlibEq(t, `(some evenp ())`, "()")
  evalStdlibEq(t, `(some (lambda (a b) (and (= a b) a)) '(1 2 3) (vector 0 2))`, "2")
  evalStdlibErr(t, `(some evenp "ab")`)
}

func TestSeqFind(t *testing.T) {
  evalStdlibEq(t, `(find 2 '(1 2 3))`, "2")
  evalStdlibEq(t, `(find "a" '("a"))`, "()")
  evalStdlibEq(t, `(find "a" '("a") :test 'equal?)`, `"a"`)
  evalStdlibEq(t, `(find "a" (vector "a") :test equal?)`, `"a"`)
  evalStdlibEq(t, `(find 3 (vector '(1 . a) '(3 . b)) :key 'car)`, "(3 . b)")
  evalStdlibEq(t, `(find 1 ())`, "()")
  evalStdlibEq(t, `(find-if evenp (vector 1 4 6))`, "4")
  evalStdlibEq(t, `(find-if 'evenp '((1) (2)) :key car)`, "(2)")
  evalStdlibEq(t, `(find-if evenp ())`, "()")
  evalStdlibErr(t, `(find 1 "abc")`)
  evalStdlibErr(t, `(find-if evenp '(1) :test equal?)`)
}

func TestSeqPosition(t *testing.T) {
  evalStdlibEq(t, `(position 3 '(1 2 3))`, "2")
  evalStdlibEq(t, `(position 9 '(1))`, "()")
  evalStdlibEq(t, `(position 'b (vector 'a 'b))`, "1")
  evalStdlibEq(t, `(position 2 '((1) (2)) :key 'car)`, "1")
  evalStdlibEq(t, `(position-if evenp (vector 1 2))`, "1")
  evalStdlibEq(t, `(position-if evenp ())`, "()")
  evalStdlibErr(t, `(position 1 5)`)
}

func TestSeqCount(t *testing.T) {
  evalStdlibEq(t, `(count 1 '(1 2 1))`, "2")
  evalStdlibEq(t, `(count "a" (vector "a" "a") :test 'equal?)`, "2")
  evalStdlibEq(t, `(count 1 ())`, "0")
  evalStdlibEq(t, `(count-if evenp (vector 2 4 5))`, "2")
  evalStdlibEq(t, `(count-if 'evenp '((2) (3)) :key 'car)`, "1")
  evalStdlibErr(t, `(count 1 '(1) :foo 2)`)
  evalStdlibErr(t, `(count 1 '(1) :test)`)
  evalStdlibErr(t, `(count-if evenp "ab")`)
}

// Charakterisierung: Verhalten des bisherigen Lisp-eql (stdlib.lisp) bleibt.
func TestEqlPrimitive(t *testing.T) {
  evalStdlibEq(t, `(eql 1 1)`, "t")
  evalStdlibEq(t, `(eql 1.0 1)`, "t") // golisp2 kennt nur float64
  evalStdlibEq(t, `(eql "a" "a")`, "()")
  evalStdlibEq(t, `(eql 'a 'a)`, "t")
  evalStdlibEq(t, `(eql (list 1) (list 1))`, "()")
  evalStdlibEq(t, `(let ((x (list 1))) (eql x x))`, "t")
  evalStdlibEq(t, `(eql () ())`, "t")
  evalStdlibEq(t, `(eql 1 'a)`, "()")
  evalStdlibErr(t, `(eql 1)`)
  evalStdlibErr(t, `(eql 1 2 3)`)
  // Go-Primitive: auch ohne Stdlib vorhanden
  evalEq(t, `(eql 2 2)`, "t")
}
