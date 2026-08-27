//**********************************************************************
//  lib/stdlib_todo20260818_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-sonnet-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260818
//**********************************************************************
// Tests zu TODO 20260818 (Lispbuch-Fehlerarchiv): defstruct -p-Alias,
// setf Multi-Place, ignore-errors.
package lib

import "testing"

// --- defstruct: <name>-p als Alias zu <name>? ---

func TestDefstructPredicateAlias(t *testing.T) {
  evalStdlibEq(t, `(begin
                     (defstruct zz-punkt x y)
                     (define p (make-zz-punkt :x 1 :y 2))
                     (zz-punkt-p p))`, "t")
  evalStdlibEq(t, `(begin
                     (defstruct zz-punkt2 x y)
                     (zz-punkt2-p 5))`, "()")
  // beide Prädikate bleiben nutzbar
  evalStdlibEq(t, `(begin
                     (defstruct zz-punkt3 x y)
                     (define p (make-zz-punkt3 :x 1 :y 2))
                     (and (zz-punkt3? p) (zz-punkt3-p p)))`, "t")
}

// --- setf: Multi-Place ---

func TestSetfMultiPlace(t *testing.T) {
  evalStdlibEq(t, `(begin (define zz-a 1) (define zz-b 2)
                          (setf zz-a 10 zz-b 20)
                          (list zz-a zz-b))`, "(10 20)")
  // Rückgabewert ist der letzte zugewiesene Wert (CL)
  evalStdlibEq(t, `(begin (define zz-c 0) (define zz-d 0)
                          (setf zz-c 1 zz-d 2))`, "2")
  // mischt Places und Symbole
  evalStdlibEq(t, `(begin (define h (make-hash-table)) (define zz-e 0)
                          (setf (gethash 'k h) 9 zz-e 3)
                          (list (gethash 'k h) zz-e))`, "(9 3)")
  // einzelnes Paar bleibt wie bisher gültig
  evalStdlibEq(t, `(begin (define zz-f 0) (setf zz-f 7) zz-f)`, "7")
}

// --- ignore-errors ---

func TestIgnoreErrors(t *testing.T) {
  evalStdlibEq(t, `(ignore-errors (+ 1 2))`, "3")
  evalStdlibEq(t, `(ignore-errors (error "boom"))`, "()")
  evalStdlibEq(t, `(ignore-errors (/ 1 0))`, "()")
  // throw läuft weiterhin durch (kein Fehler-Sentinel)
  evalStdlibErr(t, `(catch 'nope (ignore-errors (throw 'zztag 1)))`)
  evalStdlibEq(t, `(catch 'zztag (ignore-errors (throw 'zztag 42)))`, "42")
}
