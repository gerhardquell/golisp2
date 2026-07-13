//**********************************************************************
//  lib/eval_specialforms.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260616 (aufgespalten aus eval.go)
//**********************************************************************
// Spezialformen (nicht-tail): define/setq, defun, lambda, defmacro,
// set!/setq*, begin, mapcar, load (+ Pfad-Auflösung), LoadString,
// and/or/not, macroexpand, case (Tail-Hilfsfunktion, gibt Tripel zurück).
//**********************************************************************

package lib

import (
  "fmt"
)

func evalDefine(form *Cell, env *Env) (*Cell, error) {
  args := form.Cdr
  name := args.Car.Val
  val, err := Eval(args.Cdr.Car, env)
  if err != nil { return nil, err }
  if err := env.Set(name, val); err != nil { return nil, err }
  RegisterDefinition(name, form.SrcFile, form.SrcLine)
  return MakeAtom(name), nil
}

// macroexpand: (macroexpand form) → expandiert Makros einmal, gibt Ergebnis zurück
func evalMacroexpand(args *Cell, env *Env) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil {
    return nil, fmt.Errorf("macroexpand: 1 Argument nötig")
  }
  form, err := Eval(args.Car, env)
  if err != nil { return nil, err }

  // Wenn es keine Liste ist, geben wir sie unverändert zurück
  if form == nil || form.Type != LIST || form.Car == nil {
    return form, nil
  }

  // Prüfe ob das erste Element ein Makro ist. form.Car kann eine
  // Specialform (begin, if, ...) oder ein nicht gebundenes Symbol sein —
  // dann kein Lookup-Fehler, sondern "nicht expandierbar" -> form zurück.
  fn, err := Eval(form.Car, env)
  if err != nil {
    return form, nil
  }

  // Wenn es ein Makro ist, expandieren wir es
  if fn.Type == MACRO {
    return applyLambda(fn, CellToSlice(form.Cdr))
  }

  // Kein Makro → Form unverändert zurückgeben
  return form, nil
}

// macroexpand-all: (macroexpand-all form) → rekursiv alle Makros expandieren.
// Expandiert Makros in allen Subformen, ohne die Form auszuwerten.
// quote / quasiquote / function werden nicht durchdrungen.
func evalMacroexpandAll(args *Cell, env *Env) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil {
    return nil, fmt.Errorf("macroexpand-all: 1 Argument nötig")
  }
  form, err := Eval(args.Car, env)
  if err != nil { return nil, err }
  return macroexpandAll(form, env)
}

func macroexpandAll(form *Cell, env *Env) (*Cell, error) {
  if form == nil || form.Type != LIST {
    return form, nil
  }
  // quote / quasiquote / function nicht durchdringen
  if form.Car != nil && form.Car.Type == ATOM {
    switch form.Car.Val {
    case "quote", "quasiquote", "function":
      return form, nil
    }
  }
  // Top-Level-Makro-Expansion versuchen
  expanded, err := macroexpandOnce(form, env)
  if err != nil { return nil, err }
  if !cellEqual(expanded, form) {
    return macroexpandAll(expanded, env)
  }
  // Kein Makro: rekursiv in car und cdr
  newCar, err := macroexpandAll(form.Car, env)
  if err != nil { return nil, err }
  newCdr, err := macroexpandAll(form.Cdr, env)
  if err != nil { return nil, err }
  return Cons(newCar, newCdr), nil
}

// macroexpandOnce: expandiert form einmal, falls car ein Makro ist.
// Liefert form unverändert zurück, wenn kein Makro vorliegt.
func macroexpandOnce(form *Cell, env *Env) (*Cell, error) {
  if form == nil || form.Type != LIST || form.Car == nil {
    return form, nil
  }
  fn, err := Eval(form.Car, env)
  if err != nil {
    return form, nil // Spezialform oder ungebunden
  }
  if fn.Type == MACRO {
    return applyLambda(fn, CellToSlice(form.Cdr))
  }
  return form, nil
}

// wrapBegin: mehrere Body-Ausdrücke → (begin expr1 expr2 ...)
// Einzelner Ausdruck → direkt zurückgeben (kein unnötiger begin-Wrapper)
func wrapBegin(exprs *Cell) *Cell {
  if exprs == nil || exprs.Type != LIST {
    return MakeNil()
  }
  if exprs.Cdr == nil || exprs.Cdr.Type != LIST {
    return exprs.Car  // nur ein Ausdruck → direkt
  }
  return Cons(MakeAtom("begin"), exprs)  // mehrere → (begin ...)
}

func evalDefun(form *Cell, env *Env) (*Cell, error) {
  args := form.Cdr
  name := args.Car.Val
  lam  := makeLambda(args.Cdr.Car, wrapBegin(args.Cdr.Cdr), env)
  if err := env.Set(name, lam); err != nil { return nil, err }
  RegisterDefinition(name, form.SrcFile, form.SrcLine)
  return MakeAtom(name), nil
}

func evalLambda(args *Cell, env *Env) (*Cell, error) {
  return makeLambda(args.Car, wrapBegin(args.Cdr), env), nil
}

func evalBegin(args *Cell, env *Env) (*Cell, error) {
  var result *Cell
  var err error
  for args != nil && args.Type == LIST {
    result, err = Eval(args.Car, env)
    if err != nil { return nil, err }
    args = args.Cdr
  }
  return result, nil
}

func evalSet(args *Cell, env *Env) (*Cell, error) {
  val, err := Eval(args.Cdr.Car, env)
  if err != nil { return nil, err }
  return MakeAtom(args.Car.Val), env.Update(args.Car.Val, val)
}

// setq*: (setq* var1 val1 var2 val2 ...) → sequentielles Setzen
func evalSetQStar(args *Cell, env *Env) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("setq*: Syntax: (setq* var1 val1 var2 val2 ...)")
  }
  var lastName string
  for a := args; a != nil && a.Type == LIST; a = a.Cdr.Cdr {
    if a.Car == nil || a.Car.Type != ATOM {
      return nil, fmt.Errorf("setq*: Variable muss ein Symbol sein")
    }
    name := a.Car.Val
    lastName = name
    if a.Cdr == nil || a.Cdr.Type != LIST {
      return nil, fmt.Errorf("setq*: Wert für '%s' fehlt", name)
    }
    val, err := Eval(a.Cdr.Car, env)
    if err != nil { return nil, err }
    // Update existierende Variable oder neu definieren
    if _, getErr := env.Get(name); getErr == nil {
      if err := env.Update(name, val); err != nil { return nil, err }  // Existiert → updaten
    } else {
      if err := env.Set(name, val); err != nil { return nil, err }     // Neu → definieren
    }
  }
  return MakeAtom(lastName), nil
}

// mapcar: (mapcar fn liste) → wendet fn auf jedes Element an
func evalMapcar(args *Cell, env *Env) (*Cell, error) {
  fn, err := Eval(args.Car, env)
  if err != nil { return nil, err }

  lst, err := Eval(args.Cdr.Car, env)
  if err != nil { return nil, err }

  var results []*Cell
  for lst != nil && lst.Type == LIST {
    res, err := apply(fn, []*Cell{lst.Car})
    if err != nil { return nil, err }
    results = append(results, res)
    lst = lst.Cdr
  }

  // Ergebnisliste aufbauen
  result := MakeNil()
  for i := len(results) - 1; i >= 0; i-- {
    result = Cons(results[i], result)
  }
  return result, nil
}

// and: (and a b c ...) → gibt ersten falschen Wert zurück, sonst letzten
func evalAnd(args *Cell, env *Env) (*Cell, error) {
  result := &Cell{Type: ATOM, Val: "t"}
  for args != nil && args.Type == LIST {
    val, err := Eval(args.Car, env)
    if err != nil { return nil, err }
    if !IsTruthy(val) { return MakeNil(), nil }  // Kurzschluss!
    result = val
    args = args.Cdr
  }
  return result, nil
}

// or: (or a b c ...) → gibt ersten wahren Wert zurück, sonst nil
func evalOr(args *Cell, env *Env) (*Cell, error) {
  for args != nil && args.Type == LIST {
    val, err := Eval(args.Car, env)
    if err != nil { return nil, err }
    if IsTruthy(val) { return val, nil }  // Kurzschluss!
    args = args.Cdr
  }
  return MakeNil(), nil
}

// not: (not x) → t wenn x falsch, sonst nil
func evalNot(args *Cell, env *Env) (*Cell, error) {
  val, err := Eval(args.Car, env)
  if err != nil { return nil, err }
  if IsTruthy(val) { return MakeNil(), nil }
  return MakeAtom("t"), nil
}

// defmacro: (defmacro name (params) body)
// Wie defun, aber speichert MACRO statt LIST
func evalDefmacro(form *Cell, env *Env) (*Cell, error) {
  args := form.Cdr
  name := args.Car.Val
  lam  := makeLambda(args.Cdr.Car, wrapBegin(args.Cdr.Cdr), env)
  lam.Type = MACRO   // ← einziger Unterschied zu defun!
  if err := env.Set(name, lam); err != nil { return nil, err }
  RegisterDefinition(name, form.SrcFile, form.SrcLine)
  return MakeAtom(name), nil
}

// case: (case key-expr ((val1 val2) result1) (else result3) ...)
// Syntaktischer Zucker fuer cond mit strukturellem Vergleich.
// Gibt Tripel zurück, damit der Eval-Loop TCO-fähig bleibt (case ist Tail).
func evalCase(args *Cell, env *Env) (*Cell, *Env, error) {
  if args == nil || args.Type != LIST {
    return nil, nil, fmt.Errorf("case: Syntax: (case key-expr clause...)")
  }
  key, err := Eval(args.Car, env)
  if err != nil { return nil, nil, err }

  for clauses := args.Cdr; clauses != nil && clauses.Type == LIST; clauses = clauses.Cdr {
    clause := clauses.Car
    if clause == nil || clause.Type != LIST { continue }

    test := clause.Car
    isElse := test.Type == ATOM && (test.Val == "else" || test.Val == "t")

    match := false
    if !isElse && test.Type == LIST {
      // Liste von Werten: ((a b c) result)
      for vals := test; vals != nil && vals.Type == LIST; vals = vals.Cdr {
        if cellEqual(key, vals.Car) { match = true; break }
      }
    } else if !isElse {
      // Einzelner Wert: (a result)
      if cellEqual(key, test) { match = true }
    }

    if isElse || match {
      body := clause.Cdr
      if body == nil || body.Type != LIST { return MakeNil(), env, nil }
      // Evaluiere alle Ausdruecke ausser dem letzten
      for body.Cdr != nil && body.Cdr.Type == LIST {
        _, err := Eval(body.Car, env)
        if err != nil { return nil, nil, err }
        body = body.Cdr
      }
      return body.Car, env, nil
    }
  }
  return MakeNil(), env, nil
}
