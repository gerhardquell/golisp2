//**********************************************************************
//  lib/celltype_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Haiku 4.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260929
//**********************************************************************

package lib

import "testing"

func TestCellTypeKernTypen(t *testing.T) {
  evalEq(t, `(%cell-type 3.5)`, "number")
  evalEq(t, `(%cell-type "a")`, "string")
  evalEq(t, `(%cell-type 'foo)`, "symbol")
  evalEq(t, `(%cell-type :k)`, "symbol")
  evalEq(t, `(%cell-type '(1 2))`, "cons")
  evalEq(t, `(%cell-type '())`, "null")
  evalEq(t, `(%cell-type (lambda (x) x))`, "lambda")
  evalEq(t, `(%cell-type car)`, "func")
  evalEq(t, `(%cell-type (make-hash-table))`, "hash-table")
}

func TestCellTypeMakro(t *testing.T) {
  evalEq(t, `(progn (defmacro ct-mm (x) x) (%cell-type ct-mm))`, "macro")
}

func TestCellTypeArity(t *testing.T) {
  if _, err := evalStr(`(%cell-type)`); err == nil {
    t.Errorf("(%%cell-type) ohne Argument sollte Fehler geben")
  }
}
