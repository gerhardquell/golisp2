//**********************************************************************
//  lib/celltype.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Haiku 4.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260929
//**********************************************************************
// %cell-type: Kern-Typ einer Cell als Symbol. Einzige Go-Stelle, an der
// types.lisp (type-of/typep) Kernfakten abfragt — die Typhierarchie
// selbst lebt ausschliesslich in types.lisp.
//**********************************************************************

package lib

import "fmt"

func RegisterCellType(env *Env) {
  _ = env.Set("%cell-type", makeFn(fnCellType))
  _ = env.Set("integerp", makeFn(numKindP("integerp", NumInt)))
  _ = env.Set("floatp", makeFn(numKindP("floatp", NumFloat)))
  _ = env.Set("float", makeFn(fnFloat))
}

// integerp/floatp: t nur für eine Zahl dieser Zahlart (auch für
// %number-type in types.lisp).
func numKindP(name string, kind NumKind) func([]*Cell) (*Cell, error) {
  return func(args []*Cell) (*Cell, error) {
    if len(args) != 1 {
      return nil, fmt.Errorf("%s: 1 Argument nötig", name)
    }
    if args[0].Type == NUMBER && args[0].NumKind == kind {
      return cellT, nil
    }
    return MakeNil(), nil
  }
}

// float: (float x) → x als Float (Ganzzahl über 2^53 wird gerundet).
func fnFloat(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("float: 1 Argument nötig")
  }
  if err := checkNumbers("float", args); err != nil {
    return nil, err
  }
  if args[0].NumKind == NumFloat {
    return args[0], nil
  }
  return MakeNum(args[0].Num), nil
}

func fnCellType(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("%%cell-type: 1 Argument nötig")
  }
  return MakeAtom(cellTypeName(args[0])), nil
}

func cellTypeName(c *Cell) string {
  switch c.Type {
  case NUMBER:
    return "number"
  case STRING:
    return "string"
  case ATOM:
    return "symbol"
  case LIST:
    return "cons"
  case NIL:
    return "null"
  case LAMBDA:
    return "lambda"
  case FUNC:
    return "func"
  case MACRO:
    return "macro"
  case HASHTABLE:
    return "hash-table"
  case FOREIGN:
    return "foreign"
  case VECTOR:
    return "vector"
  default:
    return "t"
  }
}
