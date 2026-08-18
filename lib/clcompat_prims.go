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

// RegisterCLCompatPrims hängt sort/sqrt/get-universal-time ins Environment ein.
func RegisterCLCompatPrims(env *Env) {
  _ = env.Set("sort", makeFn(fnSort))
  _ = env.Set("sqrt", makeFn(fnSqrt))
  _ = env.Set("get-universal-time", makeFn(fnGetUniversalTime))
}

// sort: (sort liste pred &key key) → neue, sortierte Liste (nicht-
// destruktiv, immutable cells). pred(a b) wahr → a kommt vor b. key
// (optional) wird vor dem Vergleich auf jedes Element angewandt.
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

  items := CellToSlice(args[0])
  sorted := make([]*Cell, len(items))
  copy(sorted, items)

  keyOf := func(c *Cell) (*Cell, error) {
    if key == nil {
      return c, nil
    }
    return apply(key, []*Cell{c})
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

// get-universal-time: (get-universal-time) → Sekunden seit CL-Epoche
// (1900-01-01 UTC), ganzzahlig — CLHS 25.1.4.
func fnGetUniversalTime(args []*Cell) (*Cell, error) {
  if len(args) != 0 {
    return nil, fmt.Errorf("get-universal-time: keine Argumente erwartet")
  }
  secs := time.Now().UTC().Sub(clEpoch).Seconds()
  return MakeNum(math.Floor(secs)), nil
}
