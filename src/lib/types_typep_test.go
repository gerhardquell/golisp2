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

func TestTypeOfKern(t *testing.T) {
  cases := []struct{ src, want string }{
    {`(type-of 3)`, "integer"},
    {`(type-of 3.0)`, "integer"}, // Abweichung: Kern kennt nur float64
    {`(type-of -7)`, "integer"},
    {`(type-of 3.5)`, "float"},
    {`(type-of "a")`, "string"},
    {`(type-of 'foo)`, "symbol"},
    {`(type-of :k)`, "keyword"},
    {`(type-of t)`, "boolean"},
    {`(type-of '())`, "null"},
    {`(type-of '(1 2))`, "cons"},
    {`(type-of car)`, "compiled-function"},
    {`(type-of (lambda (x) x))`, "function"},
    {`(type-of (make-hash-table))`, "hash-table"},
    {`(defmacro to-mm (x) x) (type-of to-mm)`, "macro"},
  }
  for _, c := range cases {
    evalStdlibEq(t, c.src, c.want)
  }
}

func TestTypeOfStruct(t *testing.T) {
  evalStdlibEq(t, `(defstruct punkt x y) (type-of (make-punkt :x 1 :y 2))`, "punkt")
  evalStdlibEq(t, `(defstruct leer) (type-of (make-leer))`, "leer")
  // falsche Länge → keine Instanz
  evalStdlibEq(t, `(defstruct punkt x y) (type-of '(punkt 1))`, "cons")
  // ohne defstruct → nur eine Liste
  evalStdlibEq(t, `(type-of '(punkt 1 2))`, "cons")
}

func TestTypeOfCondition(t *testing.T) {
  evalStdlibEq(t, `
    (define-condition datei-fehler (error) (pfad))
    (handler-case (signal 'datei-fehler :pfad "x") (error (e) (type-of e)))`, "datei-fehler")
  evalStdlibEq(t, `(handler-case (car 5) (error (e) (type-of e)))`, "lisp-error")
}
