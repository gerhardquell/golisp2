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

func evalDefine(form *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  args := form.Cdr
  if args == nil || args.Type != LIST || args.Car == nil || args.Car.Type != ATOM ||
     args.Cdr == nil || args.Cdr.Type != LIST || args.Cdr.Car == nil {
    return nil, fmt.Errorf("define: Syntax: (define name value)")
  }
  name := args.Car.Val
  val, err := evalWithCtx(args.Cdr.Car, env, ectx.child())
  if err != nil { return nil, err }
  if err := checkRootRedefine(env, name, val, form.SrcFile, form.SrcLine); err != nil { return nil, err }
  if err := env.Set(name, val); err != nil { return nil, err }
  RegisterDefinition(name, form.SrcFile, form.SrcLine)
  return MakeAtom(name), nil
}

// setq: (setq var1 val1 var2 val2 ...) → sequentielles Setzen, liefert den
// letzten Wert (CL). Bestehende Bindung wird entlang der Env-Kette
// aktualisiert; ungebundene Variable wird im aktuellen Env angelegt
// (clisp-Top-Level-Verhalten).
func evalSetq(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  result := MakeNil()
  for a := args; a != nil && a.Type == LIST; a = a.Cdr.Cdr {
    if a.Car == nil || a.Car.Type != ATOM {
      return nil, fmt.Errorf("setq: Variable muss ein Symbol sein")
    }
    if a.Cdr == nil || a.Cdr.Type != LIST {
      return nil, fmt.Errorf("setq: ungerade Argumentzahl")
    }
    val, err := evalWithCtx(a.Cdr.Car, env, ectx.child())
    if err != nil {
      return nil, err
    }
    val = Primary(val)
    name, err := symMacroTarget(env, a.Car.Val)
    if err != nil {
      return nil, err
    }
    if err := env.Update(name, val); err != nil {
      if err := env.Set(name, val); err != nil {
        return nil, err
      }
    }
    result = val
  }
  return result, nil
}

// psetq: (psetq var1 val1 var2 val2 ...) → paralleles Setzen (CL): erst
// alle Werte im alten Env auswerten, dann zuweisen.
func evalPsetq(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  var names []string
  var vals []*Cell
  for a := args; a != nil && a.Type == LIST; a = a.Cdr.Cdr {
    if a.Car == nil || a.Car.Type != ATOM {
      return nil, fmt.Errorf("psetq: Variable muss ein Symbol sein")
    }
    if a.Cdr == nil || a.Cdr.Type != LIST {
      return nil, fmt.Errorf("psetq: ungerade Argumentzahl")
    }
    val, err := evalWithCtx(a.Cdr.Car, env, ectx.child())
    if err != nil {
      return nil, err
    }
    name, err := symMacroTarget(env, a.Car.Val)
    if err != nil {
      return nil, err
    }
    names = append(names, name)
    vals = append(vals, Primary(val))
  }
  result := MakeNil()
  for i, name := range names {
    if err := env.Update(name, vals[i]); err != nil {
      if err := env.Set(name, vals[i]); err != nil {
        return nil, err
      }
    }
    result = vals[i]
  }
  return result, nil
}

// bound?: (bound? sym) → t wenn sym im aktuellen Env gebunden ist, sonst nil.
// sym wird ausgewertet, damit (bound? variable), die ein Symbol enthält,
// funktioniert (z. B. im defstruct-Makro).
func evalBound(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Car == nil {
    return nil, fmt.Errorf("bound?: Symbol erwartet")
  }
  sym, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil { return nil, err }
  if sym == nil || (sym.Type != ATOM && sym.Type != STRING) {
    return nil, fmt.Errorf("bound?: Symbol erwartet")
  }
  if _, err := env.Get(sym.Val); err == nil {
    return MakeAtom("t"), nil
  }
  return MakeNil(), nil
}

// makunbound: (makunbound 'symbol) → entfernt die globale Bindung (CL).
// Fehler, wenn das Symbol nicht gebunden ist — laut statt still.
func evalMakunbound(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil {
    return nil, fmt.Errorf("makunbound: Syntax: (makunbound 'symbol)")
  }
  sym, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil {
    return nil, err
  }
  if sym.Type != ATOM {
    return nil, fmt.Errorf("makunbound: Argument muss ein Symbol sein")
  }
  root := env.Root()
  old, err := root.Get(sym.Val)
  if err != nil {
    return nil, fmt.Errorf("makunbound: '%s' ist nicht gebunden", sym.Val)
  }
  if old.Type == FUNC || old.Type == LAMBDA || old.Type == MACRO {
    p := currentPolicy()
    if err := applyRedefPolicy(p, sym.Val, "makunbound auf "+kindOf(old)); err != nil {
      return nil, err
    }
  }
  _, _ = root.UnsetRoot(sym.Val)
  RemoveDefinition(sym.Val)
  logRedef(RedefEvent{Name: sym.Val, OldKind: kindOf(old), Action: "makunbound"})
  return sym, nil
}

// macroexpand: (macroexpand form) → expandiert Makros einmal, gibt Ergebnis zurück
func evalMacroexpand(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil {
    return nil, fmt.Errorf("macroexpand: 1 Argument nötig")
  }
  form, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil { return nil, err }

  // Wenn es keine Liste ist, geben wir sie unverändert zurück
  if form == nil || form.Type != LIST || form.Car == nil {
    return form, nil
  }

  // Prüfe ob das erste Element ein Makro ist. form.Car kann eine
  // Specialform (begin, if, ...) oder ein nicht gebundenes Symbol sein —
  // dann kein Lookup-Fehler, sondern "nicht expandierbar" -> form zurück.
  fn, err := evalWithCtx(form.Car, env, ectx.child())
  if err != nil {
    return form, nil
  }

  // Wenn es ein Makro ist, expandieren wir es
  if fn.Type == MACRO {
    return applyLambda(fn, CellToSlice(form.Cdr), ectx)
  }

  // Kein Makro → Form unverändert zurückgeben
  return form, nil
}

// macroexpand-all: (macroexpand-all form) → rekursiv alle Makros expandieren.
// Expandiert Makros in allen Subformen, ohne die Form auszuwerten.
// quote / quasiquote / function werden nicht durchdrungen.
func evalMacroexpandAll(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil {
    return nil, fmt.Errorf("macroexpand-all: 1 Argument nötig")
  }
  form, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil { return nil, err }
  return macroexpandAll(form, env, ectx)
}

func macroexpandAll(form *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
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
  expanded, err := macroexpandOnce(form, env, ectx)
  if err != nil { return nil, err }
  if !cellEqual(expanded, form) {
    return macroexpandAll(expanded, env, ectx)
  }
  // Kein Makro: rekursiv in car und cdr
  newCar, err := macroexpandAll(form.Car, env, ectx)
  if err != nil { return nil, err }
  newCdr, err := macroexpandAll(form.Cdr, env, ectx)
  if err != nil { return nil, err }
  return Cons(newCar, newCdr), nil
}

// macroexpandOnce: expandiert form einmal, falls car ein Makro ist.
// Liefert form unverändert zurück, wenn kein Makro vorliegt.
func macroexpandOnce(form *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if form == nil || form.Type != LIST || form.Car == nil {
    return form, nil
  }
  fn, err := evalWithCtx(form.Car, env, ectx.child())
  if err != nil {
    return form, nil // Spezialform oder ungebunden
  }
  if fn.Type == MACRO {
    return applyLambda(fn, CellToSlice(form.Cdr), ectx)
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

func evalDefun(form *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  args := form.Cdr
  if args == nil || args.Type != LIST || args.Car == nil || args.Car.Type != ATOM ||
     args.Cdr == nil || args.Cdr.Type != LIST {
    return nil, fmt.Errorf("defun: Syntax: (defun name (params...) body...)")
  }
  name := args.Car.Val
  lam  := makeLambda(args.Cdr.Car, wrapBegin(args.Cdr.Cdr), env)
  if err := checkRootRedefine(env, name, lam, form.SrcFile, form.SrcLine); err != nil { return nil, err }
  if err := env.Set(name, lam); err != nil { return nil, err }
  RegisterDefinition(name, form.SrcFile, form.SrcLine)
  return MakeAtom(name), nil
}

func evalLambda(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  return makeLambda(args.Car, wrapBegin(args.Cdr), env), nil
}

func evalBegin(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  var result *Cell
  var err error
  for args != nil && args.Type == LIST {
    result, err = evalWithCtx(args.Car, env, ectx.child())
    if err != nil { return nil, err }
    args = args.Cdr
  }
  return result, nil
}

func evalSet(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil || args.Car.Type != ATOM ||
     args.Cdr == nil || args.Cdr.Type != LIST || args.Cdr.Car == nil {
    return nil, fmt.Errorf("set!: Syntax: (set! name value)")
  }
  val, err := evalWithCtx(args.Cdr.Car, env, ectx.child())
  if err != nil { return nil, err }
  name, err := symMacroTarget(env, args.Car.Val)
  if err != nil { return nil, err }
  return MakeAtom(args.Car.Val), env.Update(name, Primary(val))
}

// setq*: (setq* var1 val1 var2 val2 ...) → sequentielles Setzen
func evalSetQStar(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
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
    val, err := evalWithCtx(a.Cdr.Car, env, ectx.child())
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
func evalMapcar(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  fn, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil { return nil, err }

  lst, err := evalWithCtx(args.Cdr.Car, env, ectx.child())
  if err != nil { return nil, err }

  var results []*Cell
  for lst != nil && lst.Type == LIST {
    res, err := applyWithCtx(fn, []*Cell{lst.Car}, ectx)
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
func evalAnd(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  result := &Cell{Type: ATOM, Val: "t"}
  for args != nil && args.Type == LIST {
    val, err := evalWithCtx(args.Car, env, ectx.child())
    if err != nil { return nil, err }
    if !IsTruthy(val) { return MakeNil(), nil }  // Kurzschluss!
    result = val
    args = args.Cdr
  }
  return result, nil
}

// or: (or a b c ...) → gibt ersten wahren Wert zurück, sonst nil
func evalOr(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  for args != nil && args.Type == LIST {
    val, err := evalWithCtx(args.Car, env, ectx.child())
    if err != nil { return nil, err }
    if IsTruthy(val) { return val, nil }  // Kurzschluss!
    args = args.Cdr
  }
  return MakeNil(), nil
}

// not: (not x) → t wenn x falsch, sonst nil
func evalNot(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  val, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil { return nil, err }
  if IsTruthy(val) { return MakeNil(), nil }
  return MakeAtom("t"), nil
}

// defmacro: (defmacro name (params) body)
// Wie defun, aber speichert MACRO statt LIST
func evalDefmacro(form *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  args := form.Cdr
  if args == nil || args.Type != LIST || args.Car == nil || args.Car.Type != ATOM ||
     args.Cdr == nil || args.Cdr.Type != LIST {
    return nil, fmt.Errorf("defmacro: Syntax: (defmacro name (params...) body...)")
  }
  name := args.Car.Val
  lam  := makeLambda(args.Cdr.Car, wrapBegin(args.Cdr.Cdr), env)
  lam.Type = MACRO   // ← einziger Unterschied zu defun!
  if err := checkRootRedefine(env, name, lam, form.SrcFile, form.SrcLine); err != nil { return nil, err }
  if err := env.Set(name, lam); err != nil { return nil, err }
  RegisterDefinition(name, form.SrcFile, form.SrcLine)
  return MakeAtom(name), nil
}

// macrolet: (macrolet ((name (params...) body...) ...) body...)
// Lokale Makros: nur im body sichtbar, Schatten globale Definitionen.
// Wie in CL nicht-rekursiv: die Makro-Bodies laufen im ÄUSSEREN env
// (sehen die anderen macrolet-Makros nicht — Gegensatz zu labels).
func evalMacrolet(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil ||
     (args.Car.Type != LIST && args.Car.Type != NIL) {
    return nil, fmt.Errorf("macrolet: Syntax: (macrolet ((name (params...) body...) ...) body...)")
  }
  localEnv := NewEnv(env)
  defer freeEnv(localEnv)
  for b := args.Car; b != nil && b.Type == LIST; b = b.Cdr {
    spec := b.Car
    if spec == nil || spec.Type != LIST || spec.Car == nil || spec.Car.Type != ATOM ||
       spec.Cdr == nil || spec.Cdr.Type != LIST {
      return nil, fmt.Errorf("macrolet: Bindung muss (name (params...) body...) sein")
    }
    lam := makeLambda(spec.Cdr.Car, wrapBegin(spec.Cdr.Cdr), env)
    lam.Type = MACRO
    _ = localEnv.Set(spec.Car.Val, lam)
  }
  return evalWithCtx(wrapBegin(args.Cdr), localEnv, ectx.child())
}

// symbol-macrolet: (symbol-macrolet ((sym expansion) ...) body...)
// Bindet sym an eine SYMMACRO-Marker-Cell (Car = unausgewertete
// Expansion). Die eigentliche Arbeit leisten zwei Haken: der
// ATOM-Fall in evalWithCtx wertet bei Marker die Expansion aus,
// symMacroTarget leitet Zuweisungen um. Shadowing durch echte innere
// Bindungen (let, lambda, dolist) entsteht gratis aus der Env-Kette.
func evalSymbolMacrolet(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil ||
     (args.Car.Type != LIST && args.Car.Type != NIL) {
    return nil, fmt.Errorf("symbol-macrolet: Syntax: (symbol-macrolet ((sym expansion) ...) body...)")
  }
  localEnv := NewEnv(env)
  defer freeEnv(localEnv)
  for b := args.Car; b != nil && b.Type == LIST; b = b.Cdr {
    spec := b.Car
    if spec == nil || spec.Type != LIST || spec.Car == nil || spec.Car.Type != ATOM ||
       spec.Cdr == nil || spec.Cdr.Type != LIST || spec.Cdr.Car == nil {
      return nil, fmt.Errorf("symbol-macrolet: Bindung muss (symbol expansion) sein")
    }
    _ = localEnv.Set(spec.Car.Val, &Cell{Type: SYMMACRO, Car: spec.Cdr.Car})
  }
  return evalWithCtx(wrapBegin(args.Cdr), localEnv, ectx.child())
}

// symMacroTarget löst das Zuweisungsziel auf: Ist name aktuell an einen
// symbol-macrolet-Marker gebunden, gilt die Zuweisung der Expansion
// (CL: setq auf ein Symbol-Makro wirkt wie setf der Expansion — hier
// auf den Fall beschränkt, dass die Expansion selbst ein Symbol ist).
// Eine echte (shadownde) Bindung gewinnt, weil env.Get die Kette in
// Reihenfolge durchläuft und den Marker dann gar nicht sieht.
func symMacroTarget(env *Env, name string) (string, error) {
  cur, err := env.Get(name)
  if err != nil || cur.Type != SYMMACRO {
    return name, nil
  }
  if cur.Car == nil || cur.Car.Type != ATOM {
    return "", fmt.Errorf("setq: Symbol-Makro '%s' hat Form-Expansion — nur Symbol-Expansionen sind zuweisbar", name)
  }
  return cur.Car.Val, nil
}

// eval-when: (eval-when (situation...) body...) — golisp2 ist reiner
// Eval-Kontext (kein Compiler, kein separater Load-Modus), darum feuert
// der Body genau dann, wenn :execute (oder der Altname eval) in den
// Situationen steht. :compile-toplevel/:load-toplevel allein → nil.
// Entspricht exakt clisps Verhalten unter (eval ...).
func evalEvalWhen(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil ||
     (args.Car.Type != LIST && args.Car.Type != NIL) {
    return nil, fmt.Errorf("eval-when: Syntax: (eval-when (situation...) body...)")
  }
  run := false
  for s := args.Car; s != nil && s.Type == LIST; s = s.Cdr {
    if s.Car != nil && s.Car.Type == ATOM && (s.Car.Val == ":execute" || s.Car.Val == "eval") {
      run = true
      break
    }
  }
  if !run {
    return MakeNil(), nil
  }
  return evalWithCtx(wrapBegin(args.Cdr), env, ectx.child())
}

// progv: (progv symbole werte body...) → bindet die Symbole dynamisch
// an die Werte und wertet body aus (CL). Symbole ohne korrespondierenden
// Wert werden an nil gebunden (CL: "unbound" — golisp2-Pragmatik).
// Abweichung: golisp2 kennt keine lexikalisch/dynamisch-Trennung, darum
// sieht eine lexikalische let-Bindung desselben Namens den progv-Wert
// (in CL bliebe sie davon unberührt).
func evalProgv(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Cdr == nil || args.Cdr.Type != LIST {
    return nil, fmt.Errorf("progv: Syntax: (progv symbole werte body...)")
  }
  syms, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil { return nil, err }
  syms = Primary(syms)
  vals, err := evalWithCtx(args.Cdr.Car, env, ectx.child())
  if err != nil { return nil, err }
  vals = Primary(vals)
  localEnv := NewEnv(env)
  defer freeEnv(localEnv)
  for s, v := syms, vals; s != nil && s.Type == LIST; s = s.Cdr {
    if s.Car == nil || s.Car.Type != ATOM {
      return nil, fmt.Errorf("progv: Symbolliste darf nur Symbole enthalten, got %s", s.Car)
    }
    val := MakeNil()
    if v != nil && v.Type == LIST {
      val = Primary(v.Car)
      v = v.Cdr
    }
    _ = localEnv.Set(s.Car.Val, val)
  }
  return evalWithCtx(wrapBegin(args.Cdr.Cdr), localEnv, ectx.child())
}

// case: (case key-expr ((val1 val2) result1) (else result3) ...)
// Syntaktischer Zucker fuer cond mit strukturellem Vergleich.
// Gibt Tripel zurück, damit der Eval-Loop TCO-fähig bleibt (case ist Tail).
func evalCase(args *Cell, env *Env, ectx *evalCtx) (*Cell, *Env, error) {
  if args == nil || args.Type != LIST {
    return nil, nil, fmt.Errorf("case: Syntax: (case key-expr clause...)")
  }
  key, err := evalWithCtx(args.Car, env, ectx.child())
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
        _, err := evalWithCtx(body.Car, env, ectx.child())
        if err != nil { return nil, nil, err }
        body = body.Cdr
      }
      return body.Car, env, nil
    }
  }
  return MakeNil(), env, nil
}

// the: (the typ form) → Wert von form. golisp2 hat kein Typsystem —
// der Typ wird ignoriert (CL würde ihn prüfen). Transparent für
// Multiple Values: die Werte von form gehen unverändert durch.
func evalThe(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Cdr == nil || args.Cdr.Type != LIST {
    return nil, fmt.Errorf("the: Syntax: (the typ form)")
  }
  return evalWithCtx(args.Cdr.Car, env, ectx.child())
}
