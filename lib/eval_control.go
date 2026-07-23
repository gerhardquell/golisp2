//**********************************************************************
//  lib/eval_control.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260616 (aufgespalten aus eval.go)
//**********************************************************************
// Control-Flow + Nebenläufigkeit: while, do, prog1, prog2, flet, labels,
// block, return-from, catch, eval, parfunc, lock.
//**********************************************************************

package lib

import (
  "context"
  "errors"
  "fmt"
  "time"
)

// prog1: (prog1 first rest...) → wertet first aus, dann rest (Nebenwirkung),
// liefert den Wert von first.
func evalProg1(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("prog1: Syntax: (prog1 first rest...)")
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

// prog2: (prog2 a b rest...) → wertet a und b aus, dann rest (Nebenwirkung),
// liefert den Wert von b.
func evalProg2(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Cdr == nil || args.Cdr.Type != LIST {
    return nil, fmt.Errorf("prog2: Syntax: (prog2 a b rest...)")
  }
  if _, err := evalWithCtx(args.Car, env, ectx.child()); err != nil {
    return nil, err
  }
  return evalProg1(args.Cdr, env, ectx)
}

// while: (while test body...)
// Wertet body aus solange test wahr ist, gibt nil zurück.
func evalWhile(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("while: Syntax: (while test body...)")
  }
  test := args.Car
  body := wrapBegin(args.Cdr)
  for {
    if err := ectx.check(); err != nil {
      return nil, err
    }
    cond, err := evalWithCtx(test, env, ectx.child())
    if err != nil { return nil, err }
    if !IsTruthy(cond) { return MakeNil(), nil }
    if _, err := evalWithCtx(body, env, ectx.child()); err != nil { return nil, err }
  }
}

// do: (do ((var init step) ...) (test result) body...)
// Scheme-style: bindet Variablen, iteriert bis test wahr, gibt result zurück.
func evalDo(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST ||
     args.Cdr == nil || args.Cdr.Type != LIST || args.Cdr.Car == nil {
    return nil, fmt.Errorf("do: Syntax: (do ((var init step) ...) (test result) body...)")
  }
  // Variablen-Bindungen initialisieren
  localEnv := NewEnv(env)
  defer freeEnv(localEnv)
  bindings := args.Car
  for b := bindings; b != nil && b.Type == LIST; b = b.Cdr {
    spec := b.Car                         // (var init step)
    if spec == nil || spec.Type != LIST || spec.Car == nil || spec.Car.Type != ATOM {
      return nil, fmt.Errorf("do: Bindung muss (var init step) sein")
    }
    name := spec.Car.Val
    init, err := evalWithCtx(spec.Cdr.Car, env, ectx.child())  // init im äußeren env auswerten
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
    if err := ectx.check(); err != nil {
      return nil, err
    }
    // Abbruchtest
    cond, err := evalWithCtx(test, localEnv, ectx.child())
    if err != nil { return nil, err }
    if IsTruthy(cond) { return evalWithCtx(result, localEnv, ectx.child()) }
    // Body auswerten
    if _, err := evalWithCtx(body, localEnv, ectx.child()); err != nil { return nil, err }
    // Alle Step-Ausdrücke gleichzeitig auswerten (im alten env!)
    var names []string
    var vals  []*Cell
    for b := bindings; b != nil && b.Type == LIST; b = b.Cdr {
      spec := b.Car
      if spec == nil || spec.Type != LIST || spec.Car == nil || spec.Car.Type != ATOM {
        return nil, fmt.Errorf("do: Bindung muss (var init step) sein")
      }
      name := spec.Car.Val
      step := spec.Cdr.Cdr  // Cdr.Cdr = step-Teil
      var newVal *Cell
      if step != nil && step.Type == LIST {
        newVal, err = evalWithCtx(step.Car, localEnv, ectx.child())
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

// do*: (do* ((var init step) ...) (test result...) body...)
// Wie do, aber SEQUENTIELL (CL): Init- und Step-Formen sehen die neuen
// Bindungen der vorherigen Variablen derselben Runde (let*-Semantik),
// während do sie parallel bindet (let-Semantik).
func evalDoStar(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST ||
     args.Cdr == nil || args.Cdr.Type != LIST || args.Cdr.Car == nil {
    return nil, fmt.Errorf("do*: Syntax: (do* ((var init step) ...) (test result) body...)")
  }
  localEnv := NewEnv(env)
  defer freeEnv(localEnv)
  bindings := args.Car
  // Init sequentiell: jede Bindung sieht die vorherigen (im localEnv!)
  for b := bindings; b != nil && b.Type == LIST; b = b.Cdr {
    spec := b.Car
    if spec == nil || spec.Type != LIST || spec.Car == nil || spec.Car.Type != ATOM {
      return nil, fmt.Errorf("do*: Bindung muss (var init step) sein")
    }
    var init *Cell
    var err error
    if spec.Cdr != nil && spec.Cdr.Type == LIST {
      init, err = evalWithCtx(spec.Cdr.Car, localEnv, ectx.child())
      if err != nil { return nil, err }
    } else {
      init = MakeNil() // (var) ohne Init → nil (CL)
    }
    _ = localEnv.Set(spec.Car.Val, Primary(init))
  }
  termClause := args.Cdr.Car
  test   := termClause.Car
  result := wrapBegin(termClause.Cdr)
  body := wrapBegin(args.Cdr.Cdr)
  for {
    if err := ectx.check(); err != nil {
      return nil, err
    }
    cond, err := evalWithCtx(test, localEnv, ectx.child())
    if err != nil { return nil, err }
    if IsTruthy(cond) { return evalWithCtx(result, localEnv, ectx.child()) }
    if _, err := evalWithCtx(body, localEnv, ectx.child()); err != nil { return nil, err }
    // Steps sequentiell: auswerten UND sofort binden — der nächste
    // Step sieht den neuen Wert (Unterschied zu do)
    for b := bindings; b != nil && b.Type == LIST; b = b.Cdr {
      spec := b.Car
      if spec == nil || spec.Type != LIST || spec.Car == nil || spec.Car.Type != ATOM {
        return nil, fmt.Errorf("do*: Bindung muss (var init step) sein")
      }
      step := spec.Cdr.Cdr
      if step != nil && step.Type == LIST {
        newVal, err := evalWithCtx(step.Car, localEnv, ectx.child())
        if err != nil { return nil, err }
        _ = localEnv.Set(spec.Car.Val, Primary(newVal))
      }
      // kein Step-Teil: Variable bleibt unverändert (CL)
    }
  }
}

// eval: (eval ausdruck) → wertet einen Ausdruck nochmal aus
// Beispiel: (eval (list '+ 1 2)) → 3
//           (eval (read "(+ 1 2)")) → 3
func evalEval(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("eval: 1 Argument nötig")
  }
  // Argument erst auswerten (z.B. Variable oder read-Ergebnis)
  expr, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil { return nil, err }
  // dann nochmal auswerten — im globalen Environment (Common-Lisp-Semantik).
  // So bleiben Definitionen aus (eval (read ...)) global sichtbar, was
  // fuer REPL (swank:listener-eval) und das selbsterweiternde Muster
  // essenziell ist.
  return evalWithCtx(expr, env.Root(), &evalCtx{depth: 0, ctx: ectx.ctx})
}

// blockReturn: Sentinel-Fehler für (return-from name value)
type blockReturn struct {
  name  string
  value *Cell
}

func (b *blockReturn) Error() string { return "return-from: " + b.name }

// flet: (flet ((name (params) body...) ...) body...)
// Lokale Funktionen schließen über die äußere Umgebung (keine Gegenseitigkeit).
func evalFlet(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("flet: Syntax: (flet ((name params body...) ...) body...)")
  }
  localEnv := NewEnv(env)
  defer freeEnv(localEnv)
  for defs := args.Car; defs != nil && defs.Type == LIST; defs = defs.Cdr {
    def  := defs.Car
    if def == nil || def.Type != LIST || def.Car == nil || def.Car.Type != ATOM {
      return nil, fmt.Errorf("flet: Definition muss (name (params...) body...) sein")
    }
    name := def.Car.Val
    lam  := makeLambda(def.Cdr.Car, wrapBegin(def.Cdr.Cdr), env)
    _ = localEnv.Set(name, lam)
  }
  return evalWithCtx(wrapBegin(args.Cdr), localEnv, ectx.child())
}

// labels: wie flet, aber Funktionen sehen die gemeinsame Umgebung (Rekursion).
func evalLabels(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("labels: Syntax: (labels ((name params body...) ...) body...)")
  }
  localEnv := NewEnv(env)
  defer freeEnv(localEnv)
  for defs := args.Car; defs != nil && defs.Type == LIST; defs = defs.Cdr {
    def  := defs.Car
    if def == nil || def.Type != LIST || def.Car == nil || def.Car.Type != ATOM {
      return nil, fmt.Errorf("labels: Definition muss (name (params...) body...) sein")
    }
    name := def.Car.Val
    lam  := makeLambda(def.Cdr.Car, wrapBegin(def.Cdr.Cdr), localEnv)
    _ = localEnv.Set(name, lam)
  }
  return evalWithCtx(wrapBegin(args.Cdr), localEnv, ectx.child())
}

// block: (block name body...) → benannter Block; return-from verlässt ihn.
// blockName liest einen Block-Namen: nil ist ein gültiger Symbolname
// (CL: der implizite Block) und wird zum String "nil" normalisiert.
func blockName(c *Cell) string {
  if c.Type == NIL {
    return "nil"
  }
  return c.Val
}

func evalBlock(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil ||
     (args.Car.Type != ATOM && args.Car.Type != NIL) {
    return nil, fmt.Errorf("block: Syntax: (block name body...)")
  }
  name := blockName(args.Car)
  result, err := evalWithCtx(wrapBegin(args.Cdr), env, ectx.child())
  if err != nil {
    if br, ok := err.(*blockReturn); ok && br.name == name {
      return br.value, nil
    }
    return nil, err
  }
  return result, nil
}

// tagbody: (tagbody stmt...) → CL: atomare Statements sind Tags
// (Sprungziele, werden nicht evaluiert), Listen-Statements werden der
// Reihe nach evaluiert. (go tag) setzt den PC auf das dem Tag folgende
// Statement. Liefert nil.
func evalTagbody(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  var stmts []*Cell
  tags := map[string]int{}
  for a := args; a != nil && a.Type == LIST; a = a.Cdr {
    if a.Car != nil && a.Car.Type == ATOM {
      tags[a.Car.Val] = len(stmts) // Tag zeigt auf das NÄCHSTE Statement
    } else {
      stmts = append(stmts, a.Car)
    }
  }
  pc := 0
  for pc < len(stmts) {
    if err := ectx.check(); err != nil {
      return nil, err
    }
    _, err := evalWithCtx(stmts[pc], env, ectx.child())
    if err != nil {
      if gs, ok := err.(*goSignal); ok {
        if idx, found := tags[gs.tag]; found {
          pc = idx
          continue
        }
      }
      return nil, err
    }
    pc++
  }
  return MakeNil(), nil
}

// go: (go tag) → Sprung zum Tag im lexikalisch umschließenden tagbody.
// Das Tag wird NICHT evaluiert (lexikalisch, wie return-from).
func evalGo(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil || args.Car.Type != ATOM {
    return nil, fmt.Errorf("go: Syntax: (go tag)")
  }
  return nil, &goSignal{tag: args.Car.Val}
}

// goSignal: Sentinel-Fehler für (go tag)
type goSignal struct {
  tag string
}

func (g *goSignal) Error() string { return "go: kein passendes tagbody für Tag " + g.tag }

// return-from: (return-from name [value]) → nicht-lokaler Ausstieg aus block.
func evalReturnFrom(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil ||
     (args.Car.Type != ATOM && args.Car.Type != NIL) {
    return nil, fmt.Errorf("return-from: Syntax: (return-from name [value])")
  }
  name := blockName(args.Car)
  val  := MakeNil()
  if args.Cdr != nil && args.Cdr.Type == LIST {
    var err error
    val, err = evalWithCtx(args.Cdr.Car, env, ectx.child())
    if err != nil { return nil, err }
  }
  return nil, &blockReturn{name: name, value: val}
}

// return: (return [value]) — CL-Abkürzung für (return-from nil [value]).
func evalReturn(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  val := MakeNil()
  if args != nil && args.Type == LIST {
    var err error
    val, err = evalWithCtx(args.Car, env, ectx.child())
    if err != nil { return nil, err }
  }
  return nil, &blockReturn{name: "nil", value: val}
}

// catch: (catch tag body...) → CL-Semantik: dynamischer nicht-lokaler
// Ausstieg. tag wird EVALUIERT (Unterschied zu block/return-from, die
// lexikalisch sind). Ein passendes (throw tag wert) im Body liefert wert.
func evalCatch(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil {
    return nil, fmt.Errorf("catch: Syntax: (catch tag body...)")
  }
  tag, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil {
    return nil, err
  }
  result, err := evalWithCtx(wrapBegin(args.Cdr), env, ectx.child())
  if err != nil {
    if tv, ok := err.(*throwValue); ok && sameTag(tv.tag, tag) {
      return tv.value, nil
    }
    return nil, err
  }
  return result, nil
}

// throw: (throw tag wert) → löst den nicht-lokalen Ausstieg zum
// dynamisch nächsten catch mit gleichem Tag aus. Fehlt der: Laufzeitfehler.
func evalThrow(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Cdr == nil || args.Cdr.Type != LIST {
    return nil, fmt.Errorf("throw: Syntax: (throw tag wert)")
  }
  tag, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil {
    return nil, err
  }
  val, err := evalWithCtx(args.Cdr.Car, env, ectx.child())
  if err != nil {
    return nil, err
  }
  return nil, &throwValue{tag: tag, value: val}
}

// throwValue: Sentinel-Fehler für (throw tag wert)
type throwValue struct {
  tag   *Cell
  value *Cell
}

func (t *throwValue) Error() string {
  return "throw: kein passendes catch für Tag " + t.tag.String()
}

// sameTag: Tag-Vergleich für catch/throw. CL verlangt eq; da Symbole bei
// uns nicht interniert sind, zählt Namensgleichheit (ATOM) bzw. Wert-
// gleichheit (NUMBER, NIL) — das praktische eq-Äquivalent.
func sameTag(a, b *Cell) bool {
  if a == nil || b == nil {
    return a == b
  }
  if a.Type != b.Type {
    return false
  }
  switch a.Type {
  case ATOM, STRING:
    return a.Val == b.Val
  case NUMBER:
    return a.Num == b.Num
  case NIL:
    return true
  }
  return a == b
}

// unwind-protect: (unwind-protect geschützt cleanup...) → CL: cleanup
// läuft IMMER nach geschützt — bei normalem Wert, Fehler, throw, go,
// return-from. Wert/Fehler von geschützt wird danach weitergereicht;
// ein Cleanup-Fehler tritt nur dann an dessen Stelle, wenn geschützt
// selbst fehlerfrei war.
func evalUnwindProtect(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("unwind-protect: Syntax: (unwind-protect form cleanup...)")
  }
  result, err := evalWithCtx(args.Car, env, ectx.child())
  for c := args.Cdr; c != nil && c.Type == LIST; c = c.Cdr {
    if _, cerr := evalWithCtx(c.Car, env, ectx.child()); cerr != nil && err == nil {
      err = cerr
      result = nil
    }
  }
  return result, err
}

// trap: (trap body handler) → projekteigene Fehlerbehandlung (kein CL).
// Wertet body aus. Bei LispError → handler mit Fehler-Cell aufrufen.
// Echte Go-Fehler (interne Fehler) werden durchgereicht.
func evalTrap(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST ||
    args.Cdr == nil || args.Cdr.Type != LIST ||
    args.Cdr.Car == nil {
    return nil, fmt.Errorf("trap: Syntax: (trap body handler)")
  }

  // Body auswerten – eigener Eval-Aufruf damit Fehler abgefangen werden kann
  result, err := evalWithCtx(args.Car, env, ectx.child())
  if err == nil {
    return result, nil // kein Fehler → normal zurückgeben
  }

  // Kontrollfluss-Sentinels sind keine Fehler: durchreichen!
  var br *blockReturn
  var tv *throwValue
  var gs *goSignal
  if errors.As(err, &br) || errors.As(err, &tv) || errors.As(err, &gs) {
    return nil, err
  }

  // Alle Fehler abfangen (LispError + Go-Primitive-Fehler)
  lispErr, ok := err.(*LispError)
  if !ok {
    lispErr = &LispError{Msg: MakeStr(err.Error())}
  }

  // Handler auswerten und mit Fehler-Cell aufrufen
  handler, herr := evalWithCtx(args.Cdr.Car, env, ectx.child())
  if herr != nil {
    return nil, herr
  }
  return applyWithCtx(handler, []*Cell{lispErr.Msg}, ectx)
}

// parfunc: (parfunc ergebnis [:timeout N] expr1 expr2 ...)
// Wertet alle Ausdrücke parallel aus, sammelt Ergebnisse als Liste.
// Optionaler :timeout N (Sekunden): bei Ablauf liefert die Goroutine nil.
func evalParfunc(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST || args.Car == nil || args.Car.Type != ATOM {
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

  workerParent := context.Background()
  if ectx != nil && ectx.ctx != nil {
    workerParent = ectx.ctx
  }
  workerCtx, cancel := context.WithCancel(workerParent)
  defer cancel()

  for i, expr := range exprList {
    go func(idx int, e *Cell) {
      val, err := evalWithCtx(e, env, &evalCtx{depth: 0, ctx: workerCtx})
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
        // Timeout: laufende Worker canceln, restliche Ergebnisse bleiben nil
        cancel()
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
func evalLock(args *Cell, env *Env, ectx *evalCtx) (*Cell, error) {
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("lock: Syntax: (lock mutex expr...)")
  }

  muCell, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil { return nil, err }

  gm, err := getMutex(muCell)
  if err != nil { return nil, err }

  gm.mu.Lock()
  defer gm.mu.Unlock()

  return evalBegin(args.Cdr, env, ectx)
}
