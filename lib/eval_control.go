//**********************************************************************
//  lib/eval_control.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260616 (aufgespalten aus eval.go)
//**********************************************************************
// Control-Flow + Nebenläufigkeit: while, do, flet, labels, block,
// return-from, catch, eval, parfunc, lock.
//**********************************************************************

package lib

import (
  "fmt"
  "time"
)

// while: (while test body...)
// Wertet body aus solange test wahr ist, gibt nil zurück.
func evalWhile(args *Cell, env *Env) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("while: Syntax: (while test body...)")
  }
  test := args.Car
  body := wrapBegin(args.Cdr)
  for {
    cond, err := Eval(test, env)
    if err != nil { return nil, err }
    if !IsTruthy(cond) { return MakeNil(), nil }
    if _, err := Eval(body, env); err != nil { return nil, err }
  }
}

// do: (do ((var init step) ...) (test result) body...)
// Scheme-style: bindet Variablen, iteriert bis test wahr, gibt result zurück.
func evalDo(args *Cell, env *Env) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("do: Syntax: (do ((var init step) ...) (test result) body...)")
  }
  // Variablen-Bindungen initialisieren
  localEnv := NewEnv(env)
  defer freeEnv(localEnv)
  bindings := args.Car
  for b := bindings; b != nil && b.Type == LIST; b = b.Cdr {
    spec := b.Car                         // (var init step)
    name := spec.Car.Val
    init, err := Eval(spec.Cdr.Car, env)  // init im äußeren env auswerten
    if err != nil { return nil, err }
    _ = localEnv.Set(name, init)
  }
  // Abbruchbedingung: (test result...)
  termClause := args.Cdr.Car
  test   := termClause.Car
  result := wrapBegin(termClause.Cdr)
  // Optionaler Body
  body := wrapBegin(args.Cdr.Cdr)
  for {
    // Abbruchtest
    cond, err := Eval(test, localEnv)
    if err != nil { return nil, err }
    if IsTruthy(cond) { return Eval(result, localEnv) }
    // Body auswerten
    if _, err := Eval(body, localEnv); err != nil { return nil, err }
    // Alle Step-Ausdrücke gleichzeitig auswerten (im alten env!)
    var names []string
    var vals  []*Cell
    for b := bindings; b != nil && b.Type == LIST; b = b.Cdr {
      spec := b.Car
      name := spec.Car.Val
      step := spec.Cdr.Cdr  // Cdr.Cdr = step-Teil
      var newVal *Cell
      if step != nil && step.Type == LIST {
        newVal, err = Eval(step.Car, localEnv)
        if err != nil { return nil, err }
      } else {
        newVal, err = localEnv.Get(name)
        if err != nil { return nil, err }
      }
      names = append(names, name)
      vals  = append(vals, newVal)
    }
    for i, name := range names {
      _ = localEnv.Set(name, vals[i])
    }
  }
}

// eval: (eval ausdruck) → wertet einen Ausdruck nochmal aus
// Beispiel: (eval (list '+ 1 2)) → 3
//           (eval (read "(+ 1 2)")) → 3
func evalEval(args *Cell, env *Env) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("eval: 1 Argument nötig")
  }
  // Argument erst auswerten (z.B. Variable oder read-Ergebnis)
  expr, err := Eval(args.Car, env)
  if err != nil { return nil, err }
  // dann nochmal auswerten — im globalen Environment (Common-Lisp-Semantik).
  // So bleiben Definitionen aus (eval (read ...)) global sichtbar, was
  // fuer REPL (swank:listener-eval) und das selbsterweiternde Muster
  // essenziell ist.
  return Eval(expr, env.Root())
}

// blockReturn: Sentinel-Fehler für (return-from name value)
type blockReturn struct {
  name  string
  value *Cell
}

func (b *blockReturn) Error() string { return "return-from: " + b.name }

// flet: (flet ((name (params) body...) ...) body...)
// Lokale Funktionen schließen über die äußere Umgebung (keine Gegenseitigkeit).
func evalFlet(args *Cell, env *Env) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("flet: Syntax: (flet ((name params body...) ...) body...)")
  }
  localEnv := NewEnv(env)
  defer freeEnv(localEnv)
  for defs := args.Car; defs != nil && defs.Type == LIST; defs = defs.Cdr {
    def  := defs.Car
    name := def.Car.Val
    lam  := makeLambda(def.Cdr.Car, wrapBegin(def.Cdr.Cdr), env)
    _ = localEnv.Set(name, lam)
  }
  return Eval(wrapBegin(args.Cdr), localEnv)
}

// labels: wie flet, aber Funktionen sehen die gemeinsame Umgebung (Rekursion).
func evalLabels(args *Cell, env *Env) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("labels: Syntax: (labels ((name params body...) ...) body...)")
  }
  localEnv := NewEnv(env)
  defer freeEnv(localEnv)
  for defs := args.Car; defs != nil && defs.Type == LIST; defs = defs.Cdr {
    def  := defs.Car
    name := def.Car.Val
    lam  := makeLambda(def.Cdr.Car, wrapBegin(def.Cdr.Cdr), localEnv)
    _ = localEnv.Set(name, lam)
  }
  return Eval(wrapBegin(args.Cdr), localEnv)
}

// block: (block name body...) → benannter Block; return-from verlässt ihn.
func evalBlock(args *Cell, env *Env) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("block: Syntax: (block name body...)")
  }
  name := args.Car.Val
  result, err := Eval(wrapBegin(args.Cdr), env)
  if err != nil {
    if br, ok := err.(*blockReturn); ok && br.name == name {
      return br.value, nil
    }
    return nil, err
  }
  return result, nil
}

// return-from: (return-from name [value]) → nicht-lokaler Ausstieg aus block.
func evalReturnFrom(args *Cell, env *Env) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("return-from: Syntax: (return-from name [value])")
  }
  name := args.Car.Val
  val  := MakeNil()
  if args.Cdr != nil && args.Cdr.Type == LIST {
    var err error
    val, err = Eval(args.Cdr.Car, env)
    if err != nil { return nil, err }
  }
  return nil, &blockReturn{name: name, value: val}
}

// catch: (catch body-expr handler-expr)
// Wertet body-expr aus. Bei LispError → handler mit Fehler-Cell aufrufen.
// Echte Go-Fehler (interne Fehler) werden durchgereicht.
func evalCatch(args *Cell, env *Env) (*Cell, error) {
  if args == nil || args.Type != LIST ||
    args.Cdr == nil || args.Cdr.Type != LIST ||
    args.Cdr.Car == nil {
    return nil, fmt.Errorf("catch: Syntax: (catch body handler)")
  }

  // Body auswerten – eigener Eval-Aufruf damit Fehler abgefangen werden kann
  result, err := Eval(args.Car, env)
  if err == nil {
    return result, nil  // kein Fehler → normal zurückgeben
  }

  // Alle Fehler abfangen (LispError + Go-Primitive-Fehler)
  lispErr, ok := err.(*LispError)
  if !ok {
    lispErr = &LispError{Msg: MakeStr(err.Error())}
  }

  // Handler auswerten und mit Fehler-Cell aufrufen
  handler, herr := Eval(args.Cdr.Car, env)
  if herr != nil { return nil, herr }
  return apply(handler, []*Cell{lispErr.Msg})
}

// parfunc: (parfunc ergebnis [:timeout N] expr1 expr2 ...)
// Wertet alle Ausdrücke parallel aus, sammelt Ergebnisse als Liste.
// Optionaler :timeout N (Sekunden): bei Ablauf liefert die Goroutine nil.
func evalParfunc(args *Cell, env *Env) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("parfunc: Syntax: (parfunc name [:timeout N] expr...)")
  }

  // erstes Argument: Name für die Ergebnisliste
  resultName := args.Car.Val
  rest       := args.Cdr

  // optionalen :timeout N Parameter lesen
  timeout := time.Duration(0)
  if rest != nil && rest.Type == LIST &&
     rest.Car != nil && rest.Car.Val == ":timeout" &&
     rest.Cdr != nil && rest.Cdr.Type == LIST {
    timeout = time.Duration(rest.Cdr.Car.Num) * time.Second
    rest = rest.Cdr.Cdr
  }

  // Ausdrücke sammeln
  var exprList []*Cell
  for e := rest; e != nil && e.Type == LIST; e = e.Cdr {
    exprList = append(exprList, e.Car)
  }
  if len(exprList) == 0 {
    if err := env.Set(resultName, MakeNil()); err != nil { return nil, err }
    return MakeNil(), nil
  }

  // Parallel auswerten — jede Goroutine sendet ihr Ergebnis in einen Channel
  type parfuncResult struct {
    idx int
    val *Cell
  }
  ch := make(chan parfuncResult, len(exprList))

  for i, expr := range exprList {
    go func(idx int, e *Cell) {
      val, err := Eval(e, env)
      if err != nil { val = MakeNil() }
      ch <- parfuncResult{idx, val}
    }(i, expr)
  }

  // Ergebnisse einsammeln — mit oder ohne Timeout
  gathered := make([]*Cell, len(exprList))
  for i := range gathered { gathered[i] = MakeNil() } // Default: nil

  var timer <-chan time.Time
  if timeout > 0 {
    timer = time.After(timeout)
  }

  collected := 0
  for collected < len(exprList) {
    if timer != nil {
      select {
      case r := <-ch:
        gathered[r.idx] = r.val
        collected++
      case <-timer:
        // Timeout: restliche Goroutinen laufen weiter, Ergebnisse werden nil
        collected = len(exprList)
      }
    } else {
      r := <-ch
      gathered[r.idx] = r.val
      collected++
    }
  }

  // Ergebnisse als Lisp-Liste aufbauen
  listResult := MakeNil()
  for i := len(gathered) - 1; i >= 0; i-- {
    listResult = Cons(gathered[i], listResult)
  }

  // In env speichern
  if err := env.Set(resultName, listResult); err != nil { return nil, err }
  return listResult, nil
}

// lock: (lock mu expr1 expr2 ...) → atomar ausführen
func evalLock(args *Cell, env *Env) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("lock: Syntax: (lock mutex expr...)")
  }

  muCell, err := Eval(args.Car, env)
  if err != nil { return nil, err }

  gm, err := getMutex(muCell)
  if err != nil { return nil, err }

  gm.mu.Lock()
  defer gm.mu.Unlock()

  return evalBegin(args.Cdr, env)
}
