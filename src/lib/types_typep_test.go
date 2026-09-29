//**********************************************************************
//  lib/types_typep_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Haiku 4.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260929
//**********************************************************************
// type-of / typep (src/embed/types.lisp), Struct-Registry (stdlib.lisp)
// und Kollisionswarnungen. Spec: docs/superpowers/specs/2026-09-29-typep-design.md
//**********************************************************************

package lib

import (
  "strings"
  "testing"
)

var _ = strings.Contains // wird ab Task 5 gebraucht

func TestStructRegistry(t *testing.T) {
  evalStdlibEq(t, `(defstruct punkt x y) *struct-types*`, "((punkt 2))")
  evalStdlibEq(t, `(defstruct leer) *struct-types*`, "((leer 0))")
  // Reload ersetzt, dupliziert nicht
  evalStdlibEq(t, `(defstruct punkt x y) (defstruct punkt x y z) *struct-types*`, "((punkt 3))")
}
