//**********************************************************************
//  lib/eval_lambda.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260616 (aufgespalten aus eval.go)
//**********************************************************************
// Lambda/Closure-Logik: makeLambda, applyLambda, bindArgs, IsMacro.
// Lambda-Struktur: Cell{Type:LAMBDA, Car:params, Cdr:body, Env:closureEnv}
// (Makros identisch, aber Type:MACRO.)
//**********************************************************************

package lib

import "fmt"

// makeLambda baut eine Closure-Cell (Type:LAMBDA, Env=Closure).
func makeLambda(params, body *Cell, env *Env) *Cell {
  env.shared = true
  return &Cell{Type: LAMBDA, Car: params, Cdr: body, Env: env}
}

// applyLambda wendet eine Lambda/Closure auf Argumente an.
// Wird auch für Makro-Expansion genutzt (siehe Eval + evalMacroexpand).
func applyLambda(lambda *Cell, args []*Cell, ectx *evalCtx) (*Cell, error) {
  closureEnv, ok := lambda.Env.(*Env)
  if !ok {
    return nil, fmt.Errorf("applyLambda: Lambda hat keinen Closure-Env")
  }
  localEnv   := NewEnv(closureEnv)
  defer freeEnv(localEnv)
  if err := bindArgs(lambda.Car, args, closureEnv, localEnv, ectx); err != nil {
    return nil, err
  }
  return evalWithCtx(lambda.Cdr, localEnv, ectx.child())
}

// IsMacro prüft ob eine Cell ein Makro ist (exportiert für macroexpand).
func IsMacro(c *Cell) bool {
  return c != nil && c.Type == MACRO
}

// parseParamSpec zerlegt eine &optional/&key-Parameterspec (CL):
// name | (name) | (name default) | (name default supplied-p).
// supplied ist "" wenn kein supplied-p-Parameter angegeben.
// Einzige Quelle für beide Bindungswege (bindArgs/bindEvalArgs).
func parseParamSpec(param *Cell) (name string, def *Cell, supplied string) {
  if param.Type != LIST {
    return param.Val, nil, ""
  }
  name = param.Car.Val
  rest := param.Cdr
  if rest == nil || rest.Type != LIST {
    return name, nil, ""
  }
  def = rest.Car
  if rest.Cdr != nil && rest.Cdr.Type == LIST && rest.Cdr.Car != nil && rest.Cdr.Car.Type == ATOM {
    supplied = rest.Cdr.Car.Val
  }
  return name, def, supplied
}

// bindSupplied setzt das supplied-p-Flag (CL): t wenn das Argument
// geliefert wurde, sonst (). No-op ohne supplied-p-Parameter.
func bindSupplied(localEnv *Env, supplied string, delivered bool) {
  if supplied == "" {
    return
  }
  if delivered {
    _ = localEnv.Set(supplied, MakeAtom("t"))
  } else {
    _ = localEnv.Set(supplied, MakeNil())
  }
}

// bindArgs: Lambda-Parameter binden – unterstützt regulär, dotted-rest,
// &optional, &key, &rest (CL-Stil Lambda-Listen).
func bindArgs(params *Cell, args []*Cell, closureEnv *Env, localEnv *Env, ectx *evalCtx) error {
  section := 0  // 0=regulär, 1=&optional, 2=&key
  argIdx  := 0
  hasKey  := false  // &key verwendet → kein excess check

  for p := params; p != nil; {
    if p.Type == NIL { break }
    if p.Type == ATOM {
      // Dotted rest-Parameter: (lambda (a b . rest) ...)
      _ = localEnv.Set(p.Val, SliceToCell(args[argIdx:]))
      return nil
    }
    if p.Type != LIST { break }

    param := p.Car
    p = p.Cdr

    if param.Type == ATOM {
      switch param.Val {
      case "&optional": section = 1; continue
      case "&key":      section = 2; hasKey = true; continue
      case "&rest", "&body":  // &body = CL-Synonym für &rest (Makro-Lambda-Listen)
        if p == nil || p.Type != LIST || p.Car == nil {
          return fmt.Errorf("lambda: &rest braucht Parameter-Namen")
        }
        _ = localEnv.Set(p.Car.Val, SliceToCell(args[argIdx:]))
        return nil
      }
    }

    switch section {
    case 0:  // reguläre Parameter
      if param.Type != ATOM {
        return fmt.Errorf("lambda: Parameter muss Atom sein")
      }
      if argIdx >= len(args) {
        return fmt.Errorf("lambda: zu wenig Argumente (brauche '%s')", param.Val)
      }
      _ = localEnv.Set(param.Val, args[argIdx])
      argIdx++

    case 1:  // &optional
      name, def, supplied := parseParamSpec(param)
      if argIdx < len(args) {
        _ = localEnv.Set(name, args[argIdx]); argIdx++
        bindSupplied(localEnv, supplied, true)
      } else if def != nil {
        val, err := evalWithCtx(def, closureEnv, ectx.child())
        if err != nil { return err }
        _ = localEnv.Set(name, val)
        bindSupplied(localEnv, supplied, false)
      } else {
        _ = localEnv.Set(name, MakeNil())
        bindSupplied(localEnv, supplied, false)
      }

    case 2:  // &key
      name, def, supplied := parseParamSpec(param)
      keyword := ":" + name
      found := false
      for ki := argIdx; ki < len(args); ki++ {
        if args[ki].Type == ATOM && args[ki].Val == keyword && ki+1 < len(args) {
          _ = localEnv.Set(name, args[ki+1]); found = true; break
        }
      }
      if !found {
        if def != nil {
          val, err := evalWithCtx(def, closureEnv, ectx.child())
          if err != nil { return err }
          _ = localEnv.Set(name, val)
        } else {
          _ = localEnv.Set(name, MakeNil())
        }
      }
      bindSupplied(localEnv, supplied, found)
    }
  }
  if !hasKey && argIdx < len(args) {
    return fmt.Errorf("lambda: zu viele Argumente (%d überzählig)", len(args)-argIdx)
  }
  return nil
}

// bindEvalArgs bindet Lambda-Parameter direkt aus den unevaluierten
// Argument-Ausdrücken. Jeder Argument-Wert wird in callerEnv ausgewertet,
// Default-Ausdrücke für &optional/&key in closureEnv. Damit entfällt der
// Zwischen-Slice []\*Cell, den evalArgs/bindArgs sonst benötigen.
func bindEvalArgs(params *Cell, argExprs *Cell, callerEnv, closureEnv, localEnv *Env, ectx *evalCtx) error {
  section := 0  // 0=regulär, 1=&optional, 2=&key
  hasKey := false

  for p := params; p != nil; {
    if p.Type == NIL { break }
    if p.Type == ATOM {
      // Dotted rest-Parameter: (lambda (a b . rest) ...)
      rest, err := evalExprList(argExprs, callerEnv, ectx)
      if err != nil { return err }
      _ = localEnv.Set(p.Val, SliceToCell(rest))
      return nil
    }
    if p.Type != LIST { break }

    param := p.Car
    p = p.Cdr

    if param.Type == ATOM {
      switch param.Val {
      case "&optional": section = 1; continue
      case "&key":      section = 2; hasKey = true; continue
      case "&rest", "&body":  // &body = CL-Synonym für &rest (Makro-Lambda-Listen)
        if p == nil || p.Type != LIST || p.Car == nil {
          return fmt.Errorf("lambda: &rest braucht Parameter-Namen")
        }
        rest, err := evalExprList(argExprs, callerEnv, ectx)
        if err != nil { return err }
        _ = localEnv.Set(p.Car.Val, SliceToCell(rest))
        return nil
      }
    }

    switch section {
    case 0:  // reguläre Parameter
      if param.Type != ATOM {
        return fmt.Errorf("lambda: Parameter muss Atom sein")
      }
      if argExprs == nil || argExprs.Type != LIST {
        return fmt.Errorf("lambda: zu wenig Argumente (brauche '%s')", param.Val)
      }
      val, err := evalWithCtx(argExprs.Car, callerEnv, ectx.child())
      if err != nil { return err }
      _ = localEnv.Set(param.Val, Primary(val))
      argExprs = argExprs.Cdr

    case 1:  // &optional
      name, def, supplied := parseParamSpec(param)
      if argExprs != nil && argExprs.Type == LIST {
        val, err := evalWithCtx(argExprs.Car, callerEnv, ectx.child())
        if err != nil { return err }
        _ = localEnv.Set(name, Primary(val))
        argExprs = argExprs.Cdr
        bindSupplied(localEnv, supplied, true)
      } else if def != nil {
        val, err := evalWithCtx(def, closureEnv, ectx.child())
        if err != nil { return err }
        _ = localEnv.Set(name, val)
        bindSupplied(localEnv, supplied, false)
      } else {
        _ = localEnv.Set(name, MakeNil())
        bindSupplied(localEnv, supplied, false)
      }

    case 2:  // &key
      name, def, supplied := parseParamSpec(param)
      keyword := ":" + name
      found := false
      for a := argExprs; a != nil && a.Type == LIST; a = a.Cdr {
        if a.Car != nil && a.Car.Type == ATOM && a.Car.Val == keyword {
          if a.Cdr == nil || a.Cdr.Type != LIST {
            return fmt.Errorf("lambda: Keyword %s ohne Wert", keyword)
          }
          val, err := evalWithCtx(a.Cdr.Car, callerEnv, ectx.child())
          if err != nil { return err }
          _ = localEnv.Set(name, Primary(val))
          found = true
          break
        }
      }
      if !found {
        if def != nil {
          val, err := evalWithCtx(def, closureEnv, ectx.child())
          if err != nil { return err }
          _ = localEnv.Set(name, val)
        } else {
          _ = localEnv.Set(name, MakeNil())
        }
      }
      bindSupplied(localEnv, supplied, found)
    }
  }

  if !hasKey && argExprs != nil && argExprs.Type == LIST {
    return fmt.Errorf("lambda: zu viele Argumente")
  }
  return nil
}

// evalExprList wertet eine Liste von Ausdrücken aus und liefert einen Slice.
func evalExprList(exprs *Cell, env *Env, ectx *evalCtx) ([]*Cell, error) {
  var result []*Cell
  for exprs != nil && exprs.Type == LIST {
    val, err := evalWithCtx(exprs.Car, env, ectx.child())
    if err != nil { return nil, err }
    result = append(result, Primary(val))
    exprs = exprs.Cdr
  }
  return result, nil
}
