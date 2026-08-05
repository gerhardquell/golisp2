//**********************************************************************
//  lib/eval_quasiquote.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260616 (aufgespalten aus eval.go)
//**********************************************************************
// quasiquote / unquote / unquote-splice mit Verschachtelungs-Tiefenzählung.
// `,x` wertet x aus; `,@x` splietet eine Liste in die umgebende Liste.
// `depth` trackt geschachtelte Quasiquote-Ebenen (`` ` `,x ` ``).
//**********************************************************************

package lib

import "fmt"

// quasiquote: `expr → wertet unquote/unquote-splice innerhalb aus
func evalQuasiquote(args *Cell, env *Env, ectx evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("quasiquote: 1 Argument nötig")
  }
  return evalQQ(args.Car, env, 1, ectx)
}

func evalQQ(expr *Cell, env *Env, depth int, ectx evalCtx) (*Cell, error) {
  if expr == nil { return MakeNil(), nil }
  if expr.Type != LIST { return expr, nil }

  if expr.Car != nil && expr.Car.Type == ATOM {
    switch expr.Car.Val {
    case "quasiquote":
      inner, err := evalQQ(expr.Cdr.Car, env, depth+1, ectx)
      if err != nil { return nil, err }
      return Cons(MakeAtom("quasiquote"), Cons(inner, MakeNil())), nil
    case "unquote":
      if depth == 1 { return evalWithCtx(expr.Cdr.Car, env, ectx.child()) }
      inner, err := evalQQ(expr.Cdr.Car, env, depth-1, ectx)
      if err != nil { return nil, err }
      return Cons(MakeAtom("unquote"), Cons(inner, MakeNil())), nil
    case "unquote-splice":
      if depth == 1 {
        return nil, fmt.Errorf("unquote-splice: nur in Liste erlaubt")
      }
      inner, err := evalQQ(expr.Cdr.Car, env, depth-1, ectx)
      if err != nil { return nil, err }
      return Cons(MakeAtom("unquote-splice"), Cons(inner, MakeNil())), nil
    }
  }
  return evalQQList(expr, env, depth, ectx)
}

func evalQQList(lst *Cell, env *Env, depth int, ectx evalCtx) (*Cell, error) {
  if lst == nil || lst.Type != LIST { return lst, nil }
  car := lst.Car

  // unquote-splice: ,@expr → gesplicete Liste einfügen
  if car != nil && car.Type == LIST &&
    car.Car != nil && car.Car.Type == ATOM && car.Car.Val == "unquote-splice" {
    if depth == 1 {
      spliced, err := evalWithCtx(car.Cdr.Car, env, ectx.child())
      if err != nil { return nil, err }
      rest, err := evalQQList(lst.Cdr, env, depth, ectx)
      if err != nil { return nil, err }
      return appendList(spliced, rest), nil
    }
  }

  processedCar, err := evalQQ(car, env, depth, ectx)
  if err != nil { return nil, err }
  processedCdr, err := evalQQList(lst.Cdr, env, depth, ectx)
  if err != nil { return nil, err }
  return Cons(processedCar, processedCdr), nil
}

func appendList(lst, tail *Cell) *Cell {
  if lst == nil || lst.Type == NIL { return tail }
  if lst.Type != LIST { return tail }
  return Cons(lst.Car, appendList(lst.Cdr, tail))
}
