//**********************************************************************
//  lib/eval_core.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260616 (aufgespalten aus eval.go)
//**********************************************************************
// Herzstück: der Eval-Trampolin-Loop mit TCO.
// Die Tail-Spezialformen (if, begin, let, let*, cond, case) bleiben hier
// INLINE – sie setzen expr/env und machen continue im for-Loop. Auslagern
// würde das Trampolin zerstören (echter Funktionsaufruf statt O(1)-Loop).
// Nur case delegiert an evalCase (Rückgabe-Tripel) in eval_specialforms.go.
//**********************************************************************

package lib

import (
  "fmt"
  "sync"
)

// Eval wertet einen Ausdruck in env aus. Trampolin: Tail-Positionen
// setzen expr/env und continue'n, statt zu rekursieren – O(1) Stack.
func Eval(expr *Cell, env *Env) (*Cell, error) {
  // ownEnv trackt den letzten Frame, den dieser Eval-Aufruf im Tail-Call
  // angelegt hat. Er wird am Ende freigegeben; bei Tail-Calls wird der
  // Vorgaenger vor dem Uebergang freigegeben, damit Rekursion O(1) allokiert.
  var ownEnv *Env
  takeEnv := func(newEnv *Env) *Env {
    if ownEnv != nil && ownEnv != newEnv.parent && ownEnv.parent != nil && !ownEnv.shared {
      freeEnv(ownEnv)
    }
    ownEnv = newEnv
    return newEnv
  }
  defer func() { freeEnv(ownEnv) }()

  for {
    if expr == nil { return MakeNil(), nil }

    switch expr.Type {
    case NIL, NUMBER, STRING, FUNC: return expr, nil
    case ATOM:
      if len(expr.Val) > 0 && expr.Val[0] == ':' { return expr, nil } // Keywords selbst-auswertend
      return env.Get(expr.Val)
    case LIST:  // handled below
    default:    return nil, fmt.Errorf("eval: unbekannter Typ")
    }

    // ── LIST: Spezialformen und Funktionsanwendung ──
    if expr.Car == nil { return MakeNil(), nil }

    if expr.Car.Type == ATOM {
      switch expr.Car.Val {

      // ── Nicht-Tail: Ergebnis sofort zurückgeben ──
      case "quote":        return expr.Cdr.Car, nil
      case "macroexpand":  return evalMacroexpand(expr.Cdr, env)
      case "bound?":       return evalBound(expr.Cdr, env)
      case "macroexpand-all": return evalMacroexpandAll(expr.Cdr, env)
      case "exec":         return evalExec(expr.Cdr, env)
      case "define", "setq":  return evalDefine(expr, env)
      case "defun":        return evalDefun(expr, env)
      case "defmacro":     return evalDefmacro(expr, env)
      case "lambda":       return evalLambda(expr.Cdr, env)
      case "set!":         return evalSet(expr.Cdr, env)
      case "setq*":        return evalSetQStar(expr.Cdr, env)
      case "mapcar":       return evalMapcar(expr.Cdr, env)
      case "load":         return evalLoad(expr.Cdr, env)
      case "and":          return evalAnd(expr.Cdr, env)
      case "or":           return evalOr(expr.Cdr, env)
      case "not":          return evalNot(expr.Cdr, env)
      case "parfunc":      return evalParfunc(expr.Cdr, env)
      case "lock":         return evalLock(expr.Cdr, env)
      case "eval":         return evalEval(expr.Cdr, env)
      case "catch":        return evalCatch(expr.Cdr, env)
      case "while":        return evalWhile(expr.Cdr, env)
      case "do":           return evalDo(expr.Cdr, env)
      case "quasiquote":   return evalQuasiquote(expr.Cdr, env)
      case "function":     return Eval(expr.Cdr.Car, env)
      case "flet":         return evalFlet(expr.Cdr, env)
      case "labels":       return evalLabels(expr.Cdr, env)
      case "block":        return evalBlock(expr.Cdr, env)
      case "return-from":  return evalReturnFrom(expr.Cdr, env)
      case "unquote":      return nil, fmt.Errorf("unquote: außerhalb von quasiquote")
      case "unquote-splice": return nil, fmt.Errorf("unquote-splice: außerhalb von quasiquote")

      // ── Tail: expr/env setzen, Loop weiter ──
      case "if":
        cond, err := Eval(expr.Cdr.Car, env)
        if err != nil { return nil, err }
        if IsTruthy(cond) {
          expr = expr.Cdr.Cdr.Car
        } else if expr.Cdr.Cdr != nil && expr.Cdr.Cdr.Cdr != nil && expr.Cdr.Cdr.Cdr.Type == LIST {
          expr = expr.Cdr.Cdr.Cdr.Car
        } else {
          return MakeNil(), nil
        }
        continue

      case "begin":
        args := expr.Cdr
        for args != nil && args.Cdr != nil && args.Cdr.Type == LIST {
          if _, err := Eval(args.Car, env); err != nil { return nil, err }
          args = args.Cdr
        }
        if args == nil || args.Type != LIST { return MakeNil(), nil }
        expr = args.Car
        continue

      case "let":
        localEnv := NewEnv(env)
        bindings := expr.Cdr.Car
        for bindings != nil && bindings.Type == LIST {
          b := bindings.Car
          val, err := Eval(b.Cdr.Car, env)
          if err != nil { freeEnv(localEnv); return nil, err }
          _ = localEnv.Set(b.Car.Val, val)
          bindings = bindings.Cdr
        }
        // Handle multiple body expressions in let
        body := expr.Cdr.Cdr
        if body == nil {
          freeEnv(localEnv)
          return MakeNil(), nil
        }
        // Evaluate all but the last expression
        for body.Cdr != nil && body.Cdr.Type == LIST {
          _, err := Eval(body.Car, localEnv)
          if err != nil { freeEnv(localEnv); return nil, err }
          body = body.Cdr
        }
        // Tail call optimization for the last expression
        expr = body.Car
        env = takeEnv(localEnv)
        continue

      case "let*":
        localEnv := NewEnv(env)
        bindings := expr.Cdr.Car
        // Sequentielle Bindungen: jede sieht die vorherigen
        for bindings != nil && bindings.Type == LIST {
          b := bindings.Car
          val, err := Eval(b.Cdr.Car, localEnv)  // Im lokalen env auswerten!
          if err != nil { freeEnv(localEnv); return nil, err }
          _ = localEnv.Set(b.Car.Val, val)
          bindings = bindings.Cdr
        }
        // Body ausführen
        body := expr.Cdr.Cdr
        if body == nil {
          freeEnv(localEnv)
          return MakeNil(), nil
        }
        for body.Cdr != nil && body.Cdr.Type == LIST {
          _, err := Eval(body.Car, localEnv)
          if err != nil { freeEnv(localEnv); return nil, err }
          body = body.Cdr
        }
        expr = body.Car
        env = takeEnv(localEnv)
        continue

      case "cond":
        matched := false
        for c := expr.Cdr; c != nil && c.Type == LIST; c = c.Cdr {
          clause := c.Car
          if clause == nil || clause.Type != LIST {
            return nil, fmt.Errorf("cond: Klausel muss Liste sein")
          }
          test := clause.Car
          hit := test.Type == ATOM && (test.Val == "t" || test.Val == "else")
          if !hit {
            val, err := Eval(test, env)
            if err != nil { return nil, err }
            hit = IsTruthy(val)
          }
          if hit {
            body := clause.Cdr
            for body != nil && body.Cdr != nil && body.Cdr.Type == LIST {
              if _, err := Eval(body.Car, env); err != nil { return nil, err }
              body = body.Cdr
            }
            if body == nil || body.Type != LIST { return MakeNil(), nil }
            expr = body.Car
            matched = true
            break
          }
        }
        if !matched { return MakeNil(), nil }
        continue

      case "case":
        e, newEnv, err := evalCase(expr.Cdr, env)
        if err != nil { return nil, err }
        expr, env = e, newEnv
        continue
      }
    }

    // ── Funktionsanwendung ──
    fn, err := Eval(expr.Car, env)
    if err != nil { return nil, err }

    // Makro → expandieren, Loop weiter (TCO)
    if fn.Type == MACRO {
      expanded, err := applyLambda(fn, CellToSlice(expr.Cdr))
      if err != nil { return nil, err }
      expr = expanded
      continue
    }

    // Lambda → Argumente direkt binden, Loop weiter (TCO)
    if fn.Type == LIST {
      closureEnv := fn.Env.(*Env)
      localEnv := NewEnv(closureEnv)
      if err := bindEvalArgs(fn.Car, expr.Cdr, env, closureEnv, localEnv); err != nil {
        freeEnv(localEnv)
        return nil, err
      }
      expr = fn.Cdr   // body
      env = takeEnv(localEnv)
      continue
    }

    // Eingebaute Funktion: Argumente in Slice auswerten
    if fn.Type != FUNC {
      return nil, fmt.Errorf("eval: '%s' ist keine Funktion", fn)
    }
    args, pooled, err := evalArgsPooled(expr.Cdr, env)
    if err != nil { return nil, err }
    res, err := fn.Fn(args)
    if pooled { putArgSlice(args) }
    return res, err
  }
}

// Pool fuer kurze Argument-Slices eingebauter Funktionen. Die meisten
// FUNC-Aufrufe haben <= 8 Argumente; der Pool vermeidet eine Heap-Allokation
// pro Aufruf. Groessere Argumentlisten fallen auf make([]*Cell, ...) zurueck.
var argSlicePool = sync.Pool{
  New: func() interface{} { return new([8]*Cell) },
}

// evalArgsPooled wertet Argumente aus und liefert einen Slice. pooled==true
// bedeutet, der Slice stammt aus argSlicePool und muss mit putArgSlice
// zurueckgegeben werden.
func evalArgsPooled(args *Cell, env *Env) ([]*Cell, bool, error) {
  buf := argSlicePool.Get().(*[8]*Cell)
  result := buf[:0]
  pooled := true

  for args != nil && args.Type == LIST {
    if pooled && len(result) >= cap(result) {
      // Mehr als 8 Argumente: auf Heap-Buffer umschalten, Pool-Buffer freigeben.
      heap := make([]*Cell, len(result), len(result)+4)
      copy(heap, result)
      argSlicePool.Put((*[8]*Cell)(buf[:8]))
      result = heap
      pooled = false
    }
    val, err := Eval(args.Car, env)
    if err != nil {
      if pooled { argSlicePool.Put((*[8]*Cell)(buf[:8])) }
      return nil, false, err
    }
    result = append(result, val)
    args = args.Cdr
  }
  return result, pooled, nil
}

// putArgSlice gibt einen aus evalArgsPooled stammenden Slice zurueck.
func putArgSlice(s []*Cell) {
  if cap(s) == 8 {
    argSlicePool.Put((*[8]*Cell)(s[:8]))
  }
}

func evalArgs(args *Cell, env *Env) ([]*Cell, error) {
  var result []*Cell
  for args != nil && args.Type == LIST {
    val, err := Eval(args.Car, env)
    if err != nil { return nil, err }
    result = append(result, val)
    args = args.Cdr
  }
  return result, nil
}

func apply(fn *Cell, args []*Cell) (*Cell, error) {
  switch fn.Type {
  case FUNC: return fn.Fn(args)
  case LIST: return applyLambda(fn, args)
  default:   return nil, fmt.Errorf("apply: '%s' ist keine Funktion", fn)
  }
}

// IsTruthy, sliceToCell, CellToSlice sind vereinheitlicht in types_helpers.go
// als exportierte IsTruthy, SliceToCell, CellToSlice – eine Quelle, keine
// Duplikate (Todo #5, 2026-06-16).
