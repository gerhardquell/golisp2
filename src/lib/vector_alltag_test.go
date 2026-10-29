//**********************************************************************
//  lib/vector_alltag_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261029
//**********************************************************************

package lib

import "testing"

func TestVectorPredicates(t *testing.T) {
  evalEq(t, "(vectorp (vector 1))", "t")
  evalEq(t, "(vectorp '(1))", "()")
  evalEq(t, "(vectorp ())", "()")
  evalEq(t, "(arrayp (vector))", "t")
  evalEq(t, "(arrayp \"s\")", "()")
}

func TestVectorPop(t *testing.T) {
  evalEq(t, "(let ((v (vector 1 2))) (list (vector-pop v) v))", "(2 #(1))")
  evalErr(t, "(vector-pop (vector))")
  evalErr(t, "(vector-pop '(1))")
}

func TestJSONEncodeVector(t *testing.T) {
  want, err := evalStr(`(json-encode (list 1 "a" (list 2)))`)
  if err != nil {
    t.Fatal(err)
  }
  evalEq(t, `(json-encode (vector 1 "a" (vector 2)))`, want.String())
  evalEq(t, `(json-encode (vector))`, `"[]"`)
}

func TestVectorLiteral(t *testing.T) {
  evalEq(t, "#(1 2 3)", "#(1 2 3)")
  evalEq(t, "#()", "#()")
  evalEq(t, `#(a (b c) "s")`, `#(a (b c) "s")`)
  evalEq(t, "(vectorp '#(1))", "t")
  evalEq(t, "(vectorp #(1))", "t")
  evalEq(t, "(aref #(1 2) 1)", "2")
  evalEq(t, "(eval (vector 1))", "#(1)")
  evalEq(t, "#(#(1))", "#(#(1))")
  evalEq(t, "(vectorp (aref #(#(1)) 0))", "t")
  evalErr(t, "#(1 2")
  evalErr(t, "#(1 . 2)")
}
