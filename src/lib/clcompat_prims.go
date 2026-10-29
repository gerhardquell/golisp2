//**********************************************************************
//  lib/clcompat_prims.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-sonnet-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260818
//**********************************************************************
// Go-Primitiven aus der Lispbuch-Lückenanalyse (TODO 20260818 Gruppe B),
// die sich nicht als reine stdlib.lisp-Ergänzung bauen lassen: sort
// braucht einen Lisp-Callback als Go-Sortierprädikat (analog ga-create-
// Fitness-Fn), sqrt braucht math.Sqrt (eine Lisp-Näherung wäre ungenau),
// get-universal-time braucht die Systemuhr (kein Zeit-Primitiv vorhanden).
//**********************************************************************

package lib

import (
  "fmt"
  "math"
  "sort"
  "time"
)

// clEpoch: Common-Lisp-Universalzeit zählt Sekunden seit 1900-01-01 UTC
// (CLHS 25.1.4), 70 Jahre vor der Unix-Epoche.
var clEpoch = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)

// RegisterCLCompatPrims hängt sort/sqrt/exp/log/expt/get-universal-time ins Environment ein.
func RegisterCLCompatPrims(env *Env) {
  _ = env.Set("sort", makeFn(withFnDesignator(env, 1, fnSort)))
  _ = env.Set("sqrt", makeFn(fnSqrt))
  _ = env.Set("exp", makeFn(fnExp))
  _ = env.Set("log", makeFn(fnLog))
  _ = env.Set("expt", makeFn(fnExpt))
  _ = env.Set("get-universal-time", makeFn(fnGetUniversalTime))
}

// sort: (sort seq pred &key key) → neue, sortierte Liste bzw. neuer Vektor
// (nicht-destruktiv, auch bei Vektoren). pred(a b) wahr → a kommt vor b.
// key (optional) wird vor dem Vergleich auf jedes Element angewandt.
func fnSort(args []*Cell) (*Cell, error) {
  if len(args) < 2 {
    return nil, fmt.Errorf("sort: mindestens 2 Argumente nötig (liste pred)")
  }
  pred := args[1]
  if !isCallable(pred) {
    return nil, fmt.Errorf("sort: pred muss aufrufbar sein")
  }
  var key *Cell
  for i := 2; i+1 < len(args); i += 2 {
    if args[i].Type == ATOM && args[i].Val == ":key" {
      key = args[i+1]
    }
  }
  if key != nil && !isCallable(key) {
    return nil, fmt.Errorf("sort: :key muss aufrufbar sein")
  }

  var sorted []*Cell
  isVec := args[0] != nil && args[0].Type == VECTOR
  if isVec {
    v, err := asVector("sort", args[0])
    if err != nil {
      return nil, err
    }
    v.mu.RLock()
    sorted = append([]*Cell{}, v.items...)
    v.mu.RUnlock()
  } else {
    items, err := properList("sort", args[0])
    if err != nil {
      return nil, err
    }
    sorted = items
  }

  keyOf := func(c *Cell) (*Cell, error) {
    if key == nil {
      return c, nil
    }
    k, err := apply(key, []*Cell{c})
    return Primary(k), err
  }

  var callErr error
  sort.SliceStable(sorted, func(i, j int) bool {
    if callErr != nil {
      return false
    }
    a, err := keyOf(sorted[i])
    if err != nil {
      callErr = err
      return false
    }
    b, err := keyOf(sorted[j])
    if err != nil {
      callErr = err
      return false
    }
    res, err := apply(pred, []*Cell{a, b})
    if err != nil {
      callErr = err
      return false
    }
    return IsTruthy(res)
  })
  if callErr != nil {
    return nil, callErr
  }
  if isVec {
    return makeVector(sorted), nil
  }
  return SliceToCell(sorted), nil
}

// sqrt: (sqrt n) → Quadratwurzel. Negative Zahlen sind ein Fehler —
// golisp2 hat keine komplexen Zahlen (ehrliche Grenze, keine Pseudo-Portierung).
func fnSqrt(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("sqrt: 1 Argument nötig")
  }
  if args[0].Type != NUMBER {
    return nil, fmt.Errorf("sqrt: Zahl erwartet")
  }
  if args[0].Num < 0 {
    return nil, fmt.Errorf("sqrt: negative Zahl nicht unterstützt (keine komplexen Zahlen)")
  }
  return MakeNum(math.Sqrt(args[0].Num)), nil
}

// exp: (exp x) → e hoch x.
func fnExp(args []*Cell) (*Cell, error) {
  if len(args) != 1 || args[0].Type != NUMBER {
    return nil, fmt.Errorf("exp: 1 Zahl erwartet")
  }
  return MakeNum(math.Exp(args[0].Num)), nil
}

// log: (log x) → natürlicher Logarithmus, nur x > 0 (keine komplexen Zahlen).
func fnLog(args []*Cell) (*Cell, error) {
  if len(args) != 1 || args[0].Type != NUMBER {
    return nil, fmt.Errorf("log: 1 Zahl erwartet")
  }
  if args[0].Num <= 0 {
    return nil, fmt.Errorf("log: nur Zahlen > 0 (keine komplexen Zahlen)")
  }
  return MakeNum(math.Log(args[0].Num)), nil
}

// expt: (expt basis exp) → basis hoch exp, auch negativ und gebrochen.
// NaN (negative Basis, gebrochener Exponent) und ±Inf (0 hoch negativ) sind Fehler.
func fnExpt(args []*Cell) (*Cell, error) {
  if len(args) != 2 || args[0].Type != NUMBER || args[1].Type != NUMBER {
    return nil, fmt.Errorf("expt: 2 Zahlen erwartet (basis exp)")
  }
  r := math.Pow(args[0].Num, args[1].Num)
  if math.IsNaN(r) || math.IsInf(r, 0) {
    return nil, fmt.Errorf("expt: %v hoch %v nicht darstellbar", args[0].Num, args[1].Num)
  }
  return MakeNum(r), nil
}

// get-universal-time: (get-universal-time) → Sekunden seit CL-Epoche
// (1900-01-01 UTC), ganzzahlig — CLHS 25.1.4.
func fnGetUniversalTime(args []*Cell) (*Cell, error) {
  if len(args) != 0 {
    return nil, fmt.Errorf("get-universal-time: keine Argumente erwartet")
  }
  secs := time.Now().UTC().Sub(clEpoch).Seconds()
  return MakeNum(math.Floor(secs)), nil
}
