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
  default:
    return "t"
  }
}
