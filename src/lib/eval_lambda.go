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

import (
  "fmt"
  "slices"
  "strings"
)

// makeLambda baut eine Closure-Cell (Type:LAMBDA, Env=Closure).
// Der gefangene Env braucht keine Markierung: die Frame-Lebensdauer
// gehoert dem Go-GC, und der haelt die parent-Kette transitiv am Leben.
// Mit makeMacro einziger Erzeuger von LAMBDA/MACRO-Cells, darum hier die special-Prüfung
// der Lambda-Liste (who = definierende Form für die Fehlermeldung).
func makeLambda(who string, params, body *Cell, env *Env) (*Cell, error) {
  return makeClosure(LAMBDA, who, params, body, env)
}

// makeMacro: wie makeLambda, aber Makro-Lambda-Listen zerlegen
// verschachtelt (CL), lambda/defun nicht.
func makeMacro(who string, params, body *Cell, env *Env) (*Cell, error) {
  return makeClosure(MACRO, who, params, body, env)
}

func makeClosure(typ LispType, who string, params, body *Cell, env *Env) (*Cell, error) {
  if err := checkLambdaList(who, params, typ == MACRO); err != nil {
    return nil, err
  }
  return &Cell{Type: typ, Car: params, Cdr: body, Env: env}, nil
}

// applyLambda wendet eine Lambda/Closure auf Argumente an.
// Wird auch für Makro-Expansion genutzt (siehe Eval + evalMacroexpand).
func applyLambda(lambda *Cell, args []*Cell, ectx evalCtx) (*Cell, error) {
  closureEnv, ok := lambda.Env.(*Env)
  if !ok {
    return nil, fmt.Errorf("applyLambda: Lambda hat keinen Closure-Env")
  }
  localEnv   := NewEnv(closureEnv)
  if err := bindArgs(lambda.Car, args, nil, localEnv, ectx, lambda.Type == MACRO, "lambda"); err != nil {
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

// bindArgs ist der EINE Bindungsweg für Lambda-Listen (CL-Stil): regulär,
// &optional, &rest/&body, &key, &allow-other-keys, dotted-rest. args sind
// bereits ausgewertet. Default-Ausdrücke laufen in localEnv: sie sehen die
// schon gebundenen früheren Parameter (CL) und über die Env-Kette die
// Closure. Schlüsselwort-Argumente werden paarweise aus den Werten gelesen;
// unbekannte Schlüssel, ein Schlüssel ohne Wert und ein Nicht-Schlüsselwort
// an Schlüsselposition sind Fehler.
// nested: eine Liste an Pflichtstelle ist selbst eine Lambda-Liste
// (Makros, destructuring-bind); who ist das Präfix der Meldungen.
// tail: Listenende nach args (musterWerte) — &rest/Punktpaar binden es
// geteilt; nil bei lambda-Aufrufen (args sind alle Werte).
// Die Struktur der Liste hat checkLambdaList schon geprüft.
func bindArgs(params *Cell, args []*Cell, tail *Cell, localEnv *Env, ectx evalCtx, nested bool, who string) error {
  section := 0  // 0=regulär, 1=&optional, 2=&key
  argIdx  := 0
  hasRest := false
  allowOther := false
  var keyArgs []*Cell  // Paare ab dem ersten Schlüsselwort-Argument
  var keyNames []string

  for p := params; p != nil; {
    if p.Type == NIL { break }
    if p.Type == ATOM {
      // Dotted rest-Parameter: (lambda (a b . rest) ...)
      _ = localEnv.Set(p.Val, restListe(args[argIdx:], tail))
      hasRest = true
      break
    }
    if p.Type != LIST { break }

    param := p.Car
    p = p.Cdr

    if param.Type == ATOM {
      switch param.Val {
      case "&optional": section = 1; continue
      case "&allow-other-keys": allowOther = true; continue
      case "&key":
        section = 2
        keyArgs = args[argIdx:]
        if len(keyArgs)%2 != 0 {
          return fmt.Errorf("%s: Schlüsselwort-Argumente unvollständig (%s ohne Wert)", who, keyArgs[len(keyArgs)-1])
        }
        for i := 0; i < len(keyArgs); i += 2 {
          if k := keyArgs[i]; k.Type != ATOM || !strings.HasPrefix(k.Val, ":") {
            return fmt.Errorf("%s: Schlüsselwort erwartet, bekam %s", who, k)
          }
        }
        continue
      case "&rest", "&body":  // &body = CL-Synonym für &rest (Makro-Lambda-Listen)
        if p == nil || p.Type != LIST || p.Car == nil {
          return fmt.Errorf("%s: &rest braucht Parameter-Namen", who)
        }
        _ = localEnv.Set(p.Car.Val, restListe(args[argIdx:], tail))
        hasRest = true
        p = p.Cdr  // &key darf folgen und liest dieselben Argumente
        continue
      }
    }

    switch section {
    case 0:  // reguläre Parameter
      if argIdx >= len(args) {
        return fmt.Errorf("%s: zu wenig %s (brauche '%s')", who, anzahlWort(who), param)
      }
      if param.Type != ATOM {
        // () als Untermuster passt nur auf () (CL)
        if !nested || (param.Type != LIST && param.Type != NIL) {
          return fmt.Errorf("%s: Parameter muss Atom sein", who)
        }
        vals, ende, err := musterWerte(who, param, args[argIdx])
        if err != nil { return err }
        if err := bindArgs(param, vals, ende, localEnv, ectx, true, who); err != nil { return err }
        argIdx++
        continue
      }
      _ = localEnv.SetSym(param, args[argIdx])
      argIdx++

    case 1:  // &optional
      name, def, supplied := parseParamSpec(param)
      if argIdx < len(args) {
        _ = localEnv.Set(name, args[argIdx]); argIdx++
        bindSupplied(localEnv, supplied, true)
      } else if def != nil {
        val, err := evalWithCtx(def, localEnv, ectx.child())
        if err != nil { return err }
        _ = localEnv.Set(name, Primary(val))
        bindSupplied(localEnv, supplied, false)
      } else {
        _ = localEnv.Set(name, MakeNil())
        bindSupplied(localEnv, supplied, false)
      }

    case 2:  // &key
      name, def, supplied := parseParamSpec(param)
      keyNames = append(keyNames, name)
      found := false
      for i := 0; i < len(keyArgs); i += 2 {  // erstes Vorkommen gewinnt (CL)
        if keyArgs[i].Val[1:] == name {
          _ = localEnv.Set(name, keyArgs[i+1]); found = true; break
        }
      }
      if !found {
        if def != nil {
          val, err := evalWithCtx(def, localEnv, ectx.child())
          if err != nil { return err }
          _ = localEnv.Set(name, Primary(val))
        } else {
          _ = localEnv.Set(name, MakeNil())
        }
      }
      bindSupplied(localEnv, supplied, found)
    }
  }

  if section == 2 {
    if !allowOther {
      for i := 0; i < len(keyArgs); i += 2 {
        if !slices.Contains(keyNames, keyArgs[i].Val[1:]) {
          return fmt.Errorf("%s: unbekanntes Schlüsselwort %s", who, keyArgs[i].Val)
        }
      }
    }
    return nil
  }
  if !hasRest && (argIdx < len(args) || tail != nil && tail.Type != NIL) {
    return fmt.Errorf("%s: zu viele %s (%d überzählig)", who, anzahlWort(who), len(args)-argIdx+zelleZahl(tail))
  }
  return nil
}

// musterWerte teilt den Wert für ein Muster (destructuring-bind, Makro-
// Muster): nur so viele Elemente wie Pflicht- und &optional-Stellen in den
// Slice, das Listenende geht geteilt an bindArgs (Spec §4 „Rest teilen“).
// Mit &key wird der ganze Wert gewandelt (Schlüssel stehen überall), ende
// ist dann nil.
func musterWerte(who string, muster, wert *Cell) (vals []*Cell, ende *Cell, err error) {
  stellen, mitKey, offen := 0, false, true
  for p := muster; p != nil && p.Type == LIST; p = p.Cdr {
    switch p.Car.Val {
    case "&key": mitKey, offen = true, false
    case "&rest", "&body": offen = false
    case "&optional", "&allow-other-keys":
    default:
      if offen { stellen++ }
    }
  }
  if mitKey {
    if vals, err = properList(who, wert); err != nil { return nil, nil, listeErwartet(who, muster, wert) }
    return vals, nil, nil
  }
  ende = wert
  for ; stellen > 0 && ende != nil && ende.Type == LIST; stellen-- {
    vals = append(vals, ende.Car)
    ende = ende.Cdr
  }
  if ende != nil && ende.Type != LIST && ende.Type != NIL && (stellen > 0 || len(vals) == 0) {
    return nil, nil, listeErwartet(who, muster, wert)
  }
  return vals, ende, nil
}

func listeErwartet(who string, muster, wert *Cell) error {
  return fmt.Errorf("%s: Liste erwartet für %s, bekam %s", who, muster, wert)
}

// restListe: args vor das geteilte Listenende tail; ohne tail (lambda) eine
// frische Liste.
func restListe(args []*Cell, tail *Cell) *Cell {
  if tail == nil {
    return SliceToCell(args)
  }
  for i := len(args) - 1; i >= 0; i-- {
    tail = Cons(args[i], tail)
  }
  return tail
}

// zelleZahl: überzählige Werte im Listenende — Cons-Zellen, ein unechtes
// Ende zählt eins. Nur im Fehlerpfad.
func zelleZahl(c *Cell) (n int) {
  for ; c != nil && c.Type == LIST; c = c.Cdr {
    n++
  }
  if c != nil && c.Type != NIL {
    n++
  }
  return n
}

// anzahlWort: lambda zählt Argumente, destructuring-bind Werte (Spec §3.3).
func anzahlWort(who string) string {
  if who == "lambda" {
    return "Argumente"
  }
  return "Werte"
}

// bindEvalArgs bindet Lambda-Parameter direkt aus den unevaluierten
// Argument-Ausdrücken (ausgewertet in callerEnv). Schneller Weg nur für
// die führenden regulären Parameter — ohne Zwischen-Slice. Ab dem ersten
// &-Marker, Muster oder Dotted-Rest werden die restlichen Argumente ausgewertet
// und an bindArgs übergeben: die Lambda-Listen-Logik steht nur dort.
func bindEvalArgs(params *Cell, argExprs *Cell, callerEnv, localEnv *Env, ectx evalCtx) error {
  p := params
  for p != nil && p.Type == LIST {
    param := p.Car
    if param.Type != ATOM || (len(param.Val) > 0 && param.Val[0] == '&') {
      break
    }
    if argExprs == nil || argExprs.Type != LIST {
      return fmt.Errorf("lambda: zu wenig Argumente (brauche '%s')", param.Val)
    }
    val, err := evalWithCtx(argExprs.Car, callerEnv, ectx.child())
    if err != nil { return err }
    _ = localEnv.SetSym(param, Primary(val))
    argExprs = argExprs.Cdr
    p = p.Cdr
  }
  if p == nil || p.Type == NIL {
    if argExprs != nil && argExprs.Type == LIST {
      return fmt.Errorf("lambda: zu viele Argumente")
    }
    return nil
  }
  rest, err := evalExprList(argExprs, callerEnv, ectx)
  if err != nil { return err }
  return bindArgs(p, rest, nil, localEnv, ectx, false, "lambda")
}

// evalExprList wertet eine Liste von Ausdrücken aus und liefert einen Slice.
func evalExprList(exprs *Cell, env *Env, ectx evalCtx) ([]*Cell, error) {
  var result []*Cell
  for exprs != nil && exprs.Type == LIST {
    val, err := evalWithCtx(exprs.Car, env, ectx.child())
    if err != nil { return nil, err }
    result = append(result, Primary(val))
    exprs = exprs.Cdr
  }
  return result, nil
}
