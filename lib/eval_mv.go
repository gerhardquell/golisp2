//**********************************************************************
//  lib/eval_mv.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260723
//**********************************************************************
// Multiple Values (CL): die Konsumformen multiple-value-list/-bind/-call/
// -prog1/-setq. Produziert wird via (values ...) bzw. MV-liefernden
// Primitiven (floor). Die Regel "Nicht-MV-Kontexte sehen nur den
// Primärwert" lebt EINMAL in Primary() (types.go) — nicht hier.
//**********************************************************************

package lib

import (
  "fmt"
)

// multiple-value-list: (multiple-value-list form) → Liste aller Werte.
func evalMultipleValueList(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("multiple-value-list: Syntax: (multiple-value-list form)")
  }
  v, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil {
    return nil, err
  }
  return SliceToCell(ValuesToSlice(v)), nil
}

// multiple-value-bind: (multiple-value-bind (var...) form body...) →
// bindet die Werte von form an var (fehlende → nil, überschüssige
// verworfen) und wertet body aus.
func evalMultipleValueBind(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil ||
    (args.Car.Type != LIST && args.Car.Type != NIL) ||
    args.Cdr == nil || args.Cdr.Type != LIST {
    return nil, fmt.Errorf("multiple-value-bind: Syntax: (multiple-value-bind (var...) form body...)")
  }
  v, err := evalWithCtx(args.Cdr.Car, env, ectx.child())
  if err != nil {
    return nil, err
  }
  vals := ValuesToSlice(v)
  localEnv := NewEnv(env)
  defer freeEnv(localEnv)
  i := 0
  for vars := args.Car; vars != nil && vars.Type == LIST; vars = vars.Cdr {
    if vars.Car == nil || vars.Car.Type != ATOM {
      return nil, fmt.Errorf("multiple-value-bind: Variable muss Symbol sein")
    }
    val := MakeNil()
    if i < len(vals) {
      val = vals[i]
    }
    _ = localEnv.Set(vars.Car.Val, val)
    i++
  }
  return evalWithCtx(wrapBegin(args.Cdr.Cdr), localEnv, ectx.child())
}

// multiple-value-call: (multiple-value-call fn form...) → ruft fn mit
// ALLEN Werten aller formen auf (nicht nur Primärwerte).
func evalMultipleValueCall(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("multiple-value-call: Syntax: (multiple-value-call fn form...)")
  }
  fn, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil {
    return nil, err
  }
  var vals []*Cell
  for forms := args.Cdr; forms != nil && forms.Type == LIST; forms = forms.Cdr {
    v, err := evalWithCtx(forms.Car, env, ectx.child())
    if err != nil {
      return nil, err
    }
    vals = append(vals, ValuesToSlice(v)...)
  }
  return applyWithCtx(fn, vals, ectx)
}

// multiple-value-prog1: (multiple-value-prog1 form rest...) → liefert
// ALLE Werte von form (anders als prog1, das nur den Primärwert reicht).
func evalMultipleValueProg1(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("multiple-value-prog1: Syntax: (multiple-value-prog1 form rest...)")
  }
  result, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil {
    return nil, err
  }
  for rest := args.Cdr; rest != nil && rest.Type == LIST; rest = rest.Cdr {
    if _, err := evalWithCtx(rest.Car, env, ectx.child()); err != nil {
      return nil, err
    }
  }
  return result, nil
}

// nth-value: (nth-value n form) → n-ter Wert von form (0-basiert),
// nil wenn form weniger Werte liefert. CL definiert es als Makro auf
// nth + multiple-value-list; hier als Spezialform, damit es ohne
// Stdlib verfügbar ist (Go-Tests nutzen nur BaseEnv) und alle
// MV-Formen an einem Ort leben.
func evalNthValue(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Cdr == nil || args.Cdr.Type != LIST {
    return nil, fmt.Errorf("nth-value: Syntax: (nth-value n form)")
  }
  n, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil {
    return nil, err
  }
  if n.Type != NUMBER || n.Num < 0 {
    return nil, fmt.Errorf("nth-value: Index muss nicht-negative Zahl sein, got %s", n)
  }
  v, err := evalWithCtx(args.Cdr.Car, env, ectx.child())
  if err != nil {
    return nil, err
  }
  vals := ValuesToSlice(v)
  i := int(n.Num)
  if i >= len(vals) {
    return MakeNil(), nil
  }
  return vals[i], nil
}

// multiple-value-setq: (multiple-value-setq (var...) form) → weist die
// Werte von form den Variablen zu (fehlende → nil), liefert Primärwert.
func evalMultipleValueSetq(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil ||
    (args.Car.Type != LIST && args.Car.Type != NIL) ||
    args.Cdr == nil || args.Cdr.Type != LIST {
    return nil, fmt.Errorf("multiple-value-setq: Syntax: (multiple-value-setq (var...) form)")
  }
  v, err := evalWithCtx(args.Cdr.Car, env, ectx.child())
  if err != nil {
    return nil, err
  }
  vals := ValuesToSlice(v)
  i := 0
  for vars := args.Car; vars != nil && vars.Type == LIST; vars = vars.Cdr {
    if vars.Car == nil || vars.Car.Type != ATOM {
      return nil, fmt.Errorf("multiple-value-setq: Variable muss Symbol sein")
    }
    val := MakeNil()
    if i < len(vals) {
      val = vals[i]
    }
    if err := env.Update(vars.Car.Val, val); err != nil {
      if err := env.Set(vars.Car.Val, val); err != nil {
        return nil, err
      }
    }
    i++
  }
  return Primary(v), nil
}
