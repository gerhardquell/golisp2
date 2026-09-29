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

func TestTypepHierarchie(t *testing.T) {
  ja := []string{
    `(typep 3 'integer)`, `(typep 3 'rational)`, `(typep 3 'real)`, `(typep 3 'number)`, `(typep 3 'atom)`, `(typep 3 t)`,
    `(typep 3.5 'float)`, `(typep 3.5 'real)`,
    `(typep "a" 'string)`, `(typep "a" 'sequence)`, `(typep "a" 'atom)`,
    `(typep 'foo 'symbol)`, `(typep :k 'keyword)`, `(typep :k 'symbol)`,
    `(typep t 'boolean)`, `(typep t 'symbol)`,
    `(typep '() 'null)`, `(typep '() 'boolean)`, `(typep '() 'symbol)`, `(typep '() 'list)`, `(typep '() 'sequence)`, `(typep '() 'atom)`,
    `(typep '(1) 'cons)`, `(typep '(1) 'list)`, `(typep '(1) 'sequence)`,
    `(typep car 'compiled-function)`, `(typep car 'function)`, `(typep (lambda (x) x) 'function)`,
    `(typep (make-hash-table) 'hash-table)`,
  }
  for _, src := range ja {
    evalStdlibEq(t, src, "t")
  }
  nein := []string{
    `(typep 3 'float)`, `(typep 3.5 'integer)`, `(typep "a" 'list)`, `(typep 'foo 'keyword)`,
    `(typep t 'null)`, `(typep '() 'cons)`, `(typep '(1) 'atom)`, `(typep (lambda (x) x) 'compiled-function)`,
    `(typep (make-hash-table) 'list)`, `(typep 3 nil)`,
  }
  for _, src := range nein {
    evalStdlibEq(t, src, "()")
  }
}

func TestTypepStruct(t *testing.T) {
  evalStdlibEq(t, `(defstruct punkt x y) (typep (make-punkt :x 1 :y 2) 'punkt)`, "t")
  evalStdlibEq(t, `(defstruct punkt x y) (typep (make-punkt :x 1 :y 2) 'structure-object)`, "t")
  evalStdlibEq(t, `(defstruct punkt x y) (typep (make-punkt :x 1 :y 2) 'cons)`, "t")
  evalStdlibEq(t, `(defstruct punkt x y) (typep '(punkt 1) 'punkt)`, "()")
  evalStdlibEq(t, `(typep '(1 2) 'structure-object)`, "()")
}

func TestTypepCondition(t *testing.T) {
  pre := `(define-condition datei-fehler (error) (pfad)) `
  for _, typ := range []string{"datei-fehler", "error", "condition", "cons"} {
    evalStdlibEq(t, pre+`(handler-case (signal 'datei-fehler :pfad "x") (error (e) (typep e '`+typ+`)))`, "t")
  }
  evalStdlibEq(t, pre+`(handler-case (signal 'datei-fehler :pfad "x") (error (e) (typep e 'lisp-error)))`, "()")
  evalStdlibEq(t, `(typep '(1) 'error)`, "()")
}

func TestTypepKombinatoren(t *testing.T) {
  evalStdlibEq(t, `(typep 'a '(or string symbol))`, "t")
  evalStdlibEq(t, `(typep 3 '(or string symbol))`, "()")
  evalStdlibEq(t, `(typep 3 '(and number (not float)))`, "t")
  evalStdlibEq(t, `(typep 3 '(or))`, "()")
  evalStdlibEq(t, `(typep 3 '(and))`, "t")
  evalStdlibEq(t, `(typep 3 '(not string))`, "t")
  evalStdlibEq(t, `(typep 'b '(member a b))`, "t")
  evalStdlibEq(t, `(typep 'c '(member a b))`, "()")
  evalStdlibEq(t, `(typep 2 '(eql 2))`, "t")
  evalStdlibEq(t, `(typep 3 '(satisfies number?))`, "t")
  evalStdlibEq(t, `(typep "a" '(satisfies number?))`, "()")
  evalStdlibEq(t, `(defun gerade? (n) (= 0 (mod n 2))) (typep 4 '(and integer (satisfies gerade?)))`, "t")
  evalStdlibEq(t, `(typep '((a 1)) '(or null (and cons (satisfies list?))))`, "t")
}

func TestTypepBereiche(t *testing.T) {
  cases := []struct{ src, want string }{
    {`(typep 5 '(integer 0 10))`, "t"},
    {`(typep 11 '(integer 0 10))`, "()"},
    {`(typep 0 '(integer (0) 10))`, "()"},
    {`(typep 10 '(integer 0 (10)))`, "()"},
    {`(typep 99 '(integer 0 *))`, "t"},
    {`(typep -1 '(integer 0))`, "()"},
    {`(typep 5 '(integer))`, "t"},
    {`(typep 5.5 '(integer 0 10))`, "()"},
    {`(typep 5.5 '(real 0 10))`, "t"},
    {`(typep 5.5 '(float 5 6))`, "t"},
    {`(typep "a" '(integer 0 10))`, "()"},
  }
  for _, c := range cases {
    evalStdlibEq(t, c.src, c.want)
  }
}

func TestTypepFehler(t *testing.T) {
  for _, src := range []string{
    `(typep 1 'gibts-nicht)`,
    `(typep 1 '(not a b))`,
    `(typep 1 '(eql))`,
    `(typep "a" '(integer "x"))`,
    `(typep 1 '(integer 0 1 2))`,
    `(typep 1 '(foo 1))`,
    `(typep 1 '(satisfies gibts-nicht))`,
    `(typep 1 '(satisfies))`,
    `(typep 1 "string")`,
  } {
    evalStdlibErr(t, src)
  }
  // Fehlertexte laut Spec
  evalStdlibEq(t, `(trap (typep 1 'foo) (lambda (e) e))`, `"typep: unbekannter Typ 'foo'"`)
  evalStdlibEq(t, `(trap (typep 1 '(satisfies foo)) (lambda (e) e))`, `"typep: satisfies: 'foo' ist keine Funktion"`)
}
