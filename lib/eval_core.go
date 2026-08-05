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
  "context"
  "fmt"
  "sync"
  "sync/atomic"
)

// maxEvalDepth begrenzt nicht-tail-rekursive Eval-Aufrufe.
// Wird atomar gelesen/geschrieben, damit parallele Eval-Aufrufe keine
// Data-Race produzieren, während Tests den Wert temporär absenken.
var maxEvalDepth int32 = 100000

// GetMaxEvalDepth liefert das aktuelle Rekursionslimit.
func GetMaxEvalDepth() int { return int(atomic.LoadInt32(&maxEvalDepth)) }

// SetMaxEvalDepth setzt das Rekursionslimit (Thread-sicher).
func SetMaxEvalDepth(v int) {
  if v < 0 { v = 0 }
  atomic.StoreInt32(&maxEvalDepth, int32(v))
}

// evalCtx trägt pro Eval-Lauf: Rekursionstiefe und Cancellation.
// Eine Instanz gehört immer nur einer Goroutine an.
//
// Wird per VALUE durchgereicht, nicht per Pointer. Der Wert ist 24 Byte
// (int + Interface), passt damit in Register und wird nie geschrieben —
// child() erzeugt immer eine neue Instanz, niemand mutiert eine
// bestehende. Als Pointer kostete jedes child() eine Heap-Allokation und
// machte 84,9 % aller Allokationen des Interpreters aus (PerfTODO §4.5d).
type evalCtx struct {
  depth int
  ctx   context.Context
}

// child liefert einen neuen Kontext für einen nicht-tail-rekursiven Aufruf.
func (e evalCtx) child() evalCtx {
  return evalCtx{depth: e.depth + 1, ctx: e.ctx}
}

// check prüft Depth-Limit und Cancellation.
func (e evalCtx) check() error {
  if e.depth > int(atomic.LoadInt32(&maxEvalDepth)) {
    return &LispError{Msg: MakeStr("eval: maximum recursion depth exceeded")}
  }
  if e.ctx != nil {
    select {
    case <-e.ctx.Done():
      return &LispError{Msg: MakeStr("eval: cancelled")}
    default:
    }
  }
  return nil
}

// Eval wertet einen Ausdruck in env aus. Öffentlicher Einstieg.
func Eval(expr *Cell, env *Env) (res *Cell, err error) {
  return evalWithCtx(expr, env, evalCtx{depth: 0})
}

// evalWithCtx wertet einen Ausdruck in env aus. Trampolin: Tail-Positionen
// setzen expr/env und continue'n, statt zu rekursieren – O(1) Stack.
func evalWithCtx(expr *Cell, env *Env, ectx evalCtx) (res *Cell, err error) {
  defer func() {
    if r := recover(); r != nil {
      res = nil
      err = fmt.Errorf("eval: panic recovered: %v", r)
    }
  }()

  if err := ectx.check(); err != nil {
    return nil, err
  }

  // Iterationszaehler fuer periodische Depth-/Cancellation-Checks im
  // Trampolin-Loop. Reine Tail-Calls inkrementieren die Tiefe nicht, daher
  // wuerde eine endlose Tail-Rekursion sonst nie check() erreichen.
  const checkInterval = 1024
  var iter int

  for {
    if iter%checkInterval == 0 {
      if err := ectx.check(); err != nil {
        return nil, err
      }
    }
    iter++

    if expr == nil { return MakeNil(), nil }

    switch expr.Type {
    case NIL, NUMBER, STRING, FUNC, LAMBDA, MACRO: return expr, nil
    case ATOM:
      if len(expr.Val) > 0 && expr.Val[0] == ':' { return expr, nil } // Keywords selbst-auswertend
      v, err := env.Get(expr.Val)
      if err != nil { return nil, err }
      if v.Type == SYMMACRO {
        // symbol-macrolet: Referenz wertet die Expansion im aktuellen
        // env aus (CL: Expansion wird bei jeder Referenz neu evaluiert).
        // Shadowing passiert über die Env-Kette: eine echte innere
        // Bindung wird von env.Get vor dem Marker gefunden.
        return evalWithCtx(v.Car, env, ectx.child())
      }
      return v, nil
    case LIST:  // handled below
    default:    return nil, fmt.Errorf("eval: unbekannter Typ")
    }

    // ── LIST: Spezialformen und Funktionsanwendung ──
    if expr.Car == nil { return MakeNil(), nil }

    if expr.Car.Type == ATOM {
      switch expr.Car.Val {

      // ── Nicht-Tail: Ergebnis sofort zurückgeben ──
      case "quote":        return expr.Cdr.Car, nil
      case "macroexpand":  return evalMacroexpand(expr.Cdr, env, ectx)
      case "bound?":       return evalBound(expr.Cdr, env, ectx)
      case "makunbound":   return evalMakunbound(expr.Cdr, env, ectx)
      case "macroexpand-all": return evalMacroexpandAll(expr.Cdr, env, ectx)
      case "exec":         return evalExec(expr.Cdr, env, ectx)
      case "define":       return evalDefine(expr, env, ectx)
      case "setq":         return evalSetq(expr.Cdr, env, ectx)
      case "psetq":        return evalPsetq(expr.Cdr, env, ectx)
      case "defun":        return evalDefun(expr, env, ectx)
      case "defmacro":     return evalDefmacro(expr, env, ectx)
      case "macrolet":     return evalMacrolet(expr.Cdr, env, ectx)
      case "symbol-macrolet": return evalSymbolMacrolet(expr.Cdr, env, ectx)
      case "progv":        return evalProgv(expr.Cdr, env, ectx)
      case "eval-when":    return evalEvalWhen(expr.Cdr, env, ectx)
      case "lambda":       return evalLambda(expr.Cdr, env, ectx)
      case "set!":         return evalSet(expr.Cdr, env, ectx)
      case "setq*":        return evalSetQStar(expr.Cdr, env, ectx)
      case "load":         return evalLoad(expr.Cdr, env, ectx)
      case "and":          return evalAnd(expr.Cdr, env, ectx)
      case "or":           return evalOr(expr.Cdr, env, ectx)
      case "not":          return evalNot(expr.Cdr, env, ectx)
      case "parfunc":      return evalParfunc(expr.Cdr, env, ectx)
      case "lock":         return evalLock(expr.Cdr, env, ectx)
      case "eval":         return evalEval(expr.Cdr, env, ectx)
      case "catch":        return evalCatch(expr.Cdr, env, ectx)
      case "throw":        return evalThrow(expr.Cdr, env, ectx)
      case "trap":         return evalTrap(expr.Cdr, env, ectx)
      case "unwind-protect": return evalUnwindProtect(expr.Cdr, env, ectx)
      case "multiple-value-list":  return evalMultipleValueList(expr.Cdr, env, ectx)
      case "multiple-value-bind":  return evalMultipleValueBind(expr.Cdr, env, ectx)
      case "multiple-value-call":  return evalMultipleValueCall(expr.Cdr, env, ectx)
      case "multiple-value-prog1": return evalMultipleValueProg1(expr.Cdr, env, ectx)
      case "multiple-value-setq":  return evalMultipleValueSetq(expr.Cdr, env, ectx)
      case "nth-value":    return evalNthValue(expr.Cdr, env, ectx)
      case "while":        return evalWhile(expr.Cdr, env, ectx)
      case "do":           return evalDo(expr.Cdr, env, ectx)
      case "do*":          return evalDoStar(expr.Cdr, env, ectx)
      case "prog1":        return evalProg1(expr.Cdr, env, ectx)
      case "prog2":        return evalProg2(expr.Cdr, env, ectx)
      case "quasiquote":   return evalQuasiquote(expr.Cdr, env, ectx)
      case "function":     return evalWithCtx(expr.Cdr.Car, env, ectx.child())
      case "flet":         return evalFlet(expr.Cdr, env, ectx)
      case "labels":       return evalLabels(expr.Cdr, env, ectx)
      case "block":        return evalBlock(expr.Cdr, env, ectx)
      case "return-from":  return evalReturnFrom(expr.Cdr, env, ectx)
      case "return":       return evalReturn(expr.Cdr, env, ectx)
      case "tagbody":      return evalTagbody(expr.Cdr, env, ectx)
      case "go":           return evalGo(expr.Cdr, env, ectx)
      case "declare":      return MakeNil(), nil // Deklarationen: ignoriert (kein Typsystem)
      case "the":          return evalThe(expr.Cdr, env, ectx)
      case "unquote":      return nil, fmt.Errorf("unquote: außerhalb von quasiquote")
      case "unquote-splice": return nil, fmt.Errorf("unquote-splice: außerhalb von quasiquote")

      // ── Tail: expr/env setzen, Loop weiter ──
      case "if":
        cond, err := evalWithCtx(expr.Cdr.Car, env, ectx.child())
        if err != nil { return nil, err }
        if IsTruthy(cond) {
          expr = expr.Cdr.Cdr.Car
        } else if expr.Cdr.Cdr != nil && expr.Cdr.Cdr.Cdr != nil && expr.Cdr.Cdr.Cdr.Type == LIST {
          expr = expr.Cdr.Cdr.Cdr.Car
        } else {
          return MakeNil(), nil
        }
        continue

      case "begin", "progn", "locally":
        args := expr.Cdr
        for args != nil && args.Cdr != nil && args.Cdr.Type == LIST {
          if _, err := evalWithCtx(args.Car, env, ectx.child()); err != nil { return nil, err }
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
          val, err := evalWithCtx(b.Cdr.Car, env, ectx.child())
          if err != nil { return nil, err }
          _ = localEnv.Set(b.Car.Val, Primary(val))
          bindings = bindings.Cdr
        }
        // Handle multiple body expressions in let
        body := expr.Cdr.Cdr
        if body == nil {
          return MakeNil(), nil
        }
        // Evaluate all but the last expression
        for body.Cdr != nil && body.Cdr.Type == LIST {
          _, err := evalWithCtx(body.Car, localEnv, ectx.child())
          if err != nil { return nil, err }
          body = body.Cdr
        }
        // Tail call optimization for the last expression
        expr = body.Car
        env = localEnv
        continue

      case "let*":
        localEnv := NewEnv(env)
        bindings := expr.Cdr.Car
        // Sequentielle Bindungen: jede sieht die vorherigen
        for bindings != nil && bindings.Type == LIST {
          b := bindings.Car
          val, err := evalWithCtx(b.Cdr.Car, localEnv, ectx.child())  // Im lokalen env auswerten!
          if err != nil { return nil, err }
          _ = localEnv.Set(b.Car.Val, Primary(val))
          bindings = bindings.Cdr
        }
        // Body ausführen
        body := expr.Cdr.Cdr
        if body == nil {
          return MakeNil(), nil
        }
        for body.Cdr != nil && body.Cdr.Type == LIST {
          _, err := evalWithCtx(body.Car, localEnv, ectx.child())
          if err != nil { return nil, err }
          body = body.Cdr
        }
        expr = body.Car
        env = localEnv
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
          var testVal *Cell
          if hit {
            testVal = MakeAtom("t")
          } else {
            val, err := evalWithCtx(test, env, ectx.child())
            if err != nil { return nil, err }
            hit = IsTruthy(val)
            testVal = val
          }
          if hit {
            body := clause.Cdr
            for body != nil && body.Cdr != nil && body.Cdr.Type == LIST {
              if _, err := evalWithCtx(body.Car, env, ectx.child()); err != nil { return nil, err }
              body = body.Cdr
            }
            // Klausel ohne Body (z. B. (cond (t))) → Wert des Tests (CL)
            if body == nil || body.Type != LIST {
              expr = testVal
              matched = true
              break
            }
            expr = body.Car
            matched = true
            break
          }
        }
        if !matched { return MakeNil(), nil }
        continue

      case "case":
        e, newEnv, err := evalCase(expr.Cdr, env, ectx)
        if err != nil { return nil, err }
        expr, env = e, newEnv
        continue
      }
    }

    // ── Funktionsanwendung ──
    fn, err := evalWithCtx(expr.Car, env, ectx.child())
    if err != nil { return nil, err }

    // Makro → expandieren, Loop weiter (TCO)
    if fn.Type == MACRO {
      expanded, err := applyLambda(fn, CellToSlice(expr.Cdr), ectx)
      if err != nil { return nil, err }
      expr = expanded
      continue
    }

    // Lambda → Argumente direkt binden, Loop weiter (TCO)
    if fn.Type == LAMBDA {
      closureEnv, ok := fn.Env.(*Env)
      if !ok {
        return nil, fmt.Errorf("eval: Lambda hat keinen Closure-Env")
      }
      localEnv := NewEnv(closureEnv)
      if err := bindEvalArgs(fn.Car, expr.Cdr, env, closureEnv, localEnv, ectx); err != nil {
        return nil, err
      }
      expr = fn.Cdr   // body
      env = localEnv
      continue
    }

    // Eingebaute Funktion: Argumente in Slice auswerten
    if fn.Type != FUNC {
      return nil, fmt.Errorf("eval: '%s' ist keine Funktion", fn)
    }
    args, pooled, err := evalArgsPooled(expr.Cdr, env, ectx.child())
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
func evalArgsPooled(args *Cell, env *Env, ectx evalCtx) ([]*Cell, bool, error) {
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
    val, err := evalWithCtx(args.Car, env, ectx.child())
    if err != nil {
      if pooled { argSlicePool.Put((*[8]*Cell)(buf[:8])) }
      return nil, false, err
    }
    result = append(result, Primary(val))
    args = args.Cdr
  }
  return result, pooled, nil
}

// putArgSlice gibt einen aus evalArgsPooled stammenden Slice zurueck.
//
// Nullt die Eintraege absichtlich NICHT. sync.Pool wirft seinen Inhalt bei
// jedem GC-Lauf weg, die Retention der alten *Cell-Zeiger reicht also
// hoechstens bis zum naechsten GC — genau bis zu dem Zyklus, in dem sie
// ohnehin einsammelt wuerden. Gemessen (fib 25, A/B in einer Session):
// Nullen kostet +1,1 % ns/op auf dem heissesten Pfad, die Ranges
// ueberlappen nicht. Schlechter Handel, bewusst nicht gemacht.
func putArgSlice(s []*Cell) {
  if cap(s) == 8 {
    argSlicePool.Put((*[8]*Cell)(s[:8]))
  }
}

func apply(fn *Cell, args []*Cell) (*Cell, error) {
  return applyWithCtx(fn, args, evalCtx{depth: 0})
}

func applyWithCtx(fn *Cell, args []*Cell, ectx evalCtx) (*Cell, error) {
  switch fn.Type {
  case FUNC: return fn.Fn(args)
  case LAMBDA: return applyLambda(fn, args, ectx)
  default:   return nil, fmt.Errorf("apply: '%s' ist keine Funktion", fn)
  }
}

// IsTruthy, sliceToCell, CellToSlice sind vereinheitlicht in types_helpers.go
// als exportierte IsTruthy, SliceToCell, CellToSlice – eine Quelle, keine
// Duplikate (Todo #5, 2026-06-16).
