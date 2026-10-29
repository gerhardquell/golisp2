//**********************************************************************
//  lib/sequence.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Opus 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261029
//**********************************************************************
// CL-Sequenzfunktionen für Listen und Vektoren (keine Strings — golisp2
// hat keine Characters):
//   (elt s i)  (subseq s start [end])  (copy-seq s)
//   (concatenate 'list|'vector s…)  (map 'list|'vector|nil f s…)
//   (some p s…)  find/position/count x s &key test key
//   find-if/position-if/count-if p s &key key
// Alles nicht-destruktiv. Default-Test ist eql (arith_prims.go).
//**********************************************************************

package lib

import "fmt"

func RegisterSequence(env *Env) {
  _ = env.Set("elt",         makeFn(fnElt))
  _ = env.Set("subseq",      makeFn(fnSubseq))
  _ = env.Set("copy-seq",    makeFn(fnCopySeq))
  _ = env.Set("concatenate", makeFn(fnConcatenate))
  _ = env.Set("map",         makeFn(withFnDesignator(env, 1, fnMap)))
  _ = env.Set("some",        makeFn(withFnDesignator(env, 0, fnSome)))
  for _, name := range []string{"find", "position", "count"} {
    _ = env.Set(name,         makeFn(seqSearchFn(env, name, false)))
    _ = env.Set(name + "-if", makeFn(seqSearchFn(env, name + "-if", true)))
  }
}

// seqItems: Elemente einer Liste (properList) oder eine Kopie der
// Vektor-Elemente; isVec meldet den Typ für gleichartige Ergebnisse.
func seqItems(name string, c *Cell) ([]*Cell, bool, error) {
  if c != nil && c.Type == VECTOR {
    v, err := asVector(name, c)
    if err != nil {
      return nil, false, err
    }
    v.mu.RLock()
    defer v.mu.RUnlock()
    return append([]*Cell(nil), v.items...), true, nil
  }
  if c == nil || (c.Type != LIST && c.Type != NIL) {
    return nil, false, fmt.Errorf("%s: Sequenz (Liste oder Vektor) erwartet, got %s", name, c)
  }
  items, err := properList(name, c)
  return items, false, err
}

func makeSeq(items []*Cell, isVec bool) *Cell {
  if isVec {
    return makeVector(items)
  }
  return SliceToCell(items)
}

// seqResultType: Typ-Argument von map/concatenate → isVec; nil nur bei map.
func seqResultType(name string, c *Cell, allowNil bool) (isVec, isNil bool, err error) {
  switch {
  case c != nil && c.Type == ATOM && c.Val == "list":
    return false, false, nil
  case c != nil && c.Type == ATOM && c.Val == "vector":
    return true, false, nil
  case allowNil && c != nil && c.Type == NIL:
    return false, true, nil
  }
  return false, false, fmt.Errorf("%s: Ergebnistyp list oder vector erwartet, got %s", name, c)
}

func seqInt(name string, c *Cell) (int, error) {
  if c == nil || c.Type != NUMBER || c.Num != float64(int(c.Num)) {
    return 0, fmt.Errorf("%s: ganze Zahl erwartet, got %s", name, c)
  }
  return int(c.Num), nil
}

func fnElt(args []*Cell) (*Cell, error) {
  if len(args) != 2 {
    return nil, fmt.Errorf("elt: 2 Argumente erwartet (sequenz index)")
  }
  items, _, err := seqItems("elt", args[0])
  if err != nil {
    return nil, err
  }
  i, err := vectorIndex("elt", args[1], len(items))
  if err != nil {
    return nil, err
  }
  return items[i], nil
}

func fnSubseq(args []*Cell) (*Cell, error) {
  if len(args) < 2 || len(args) > 3 {
    return nil, fmt.Errorf("subseq: 2 oder 3 Argumente erwartet (sequenz start [end])")
  }
  items, isVec, err := seqItems("subseq", args[0])
  if err != nil {
    return nil, err
  }
  start, err := seqInt("subseq", args[1])
  if err != nil {
    return nil, err
  }
  end := len(items)
  if len(args) == 3 {
    if end, err = seqInt("subseq", args[2]); err != nil {
      return nil, err
    }
  }
  if start < 0 || start > end || end > len(items) {
    return nil, fmt.Errorf("subseq: Bereich %d..%d außerhalb 0..%d", start, end, len(items))
  }
  return makeSeq(items[start:end], isVec), nil
}

func fnCopySeq(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("copy-seq: 1 Argument erwartet")
  }
  items, isVec, err := seqItems("copy-seq", args[0])
  if err != nil {
    return nil, err
  }
  return makeSeq(items, isVec), nil
}

func fnConcatenate(args []*Cell) (*Cell, error) {
  if len(args) < 1 {
    return nil, fmt.Errorf("concatenate: Ergebnistyp fehlt")
  }
  isVec, _, err := seqResultType("concatenate", args[0], false)
  if err != nil {
    return nil, err
  }
  var all []*Cell
  for _, s := range args[1:] {
    items, _, err := seqItems("concatenate", s)
    if err != nil {
      return nil, err
    }
    all = append(all, items...)
  }
  return makeSeq(all, isVec), nil
}

// seqColumns: Elemente mehrerer Sequenzen, gekürzt auf die kürzeste.
func seqColumns(name string, seqs []*Cell) ([][]*Cell, int, error) {
  if len(seqs) == 0 {
    return nil, 0, fmt.Errorf("%s: mindestens eine Sequenz nötig", name)
  }
  cols := make([][]*Cell, len(seqs))
  n := -1
  for j, s := range seqs {
    items, _, err := seqItems(name, s)
    if err != nil {
      return nil, 0, err
    }
    cols[j] = items
    if n < 0 || len(items) < n {
      n = len(items)
    }
  }
  return cols, n, nil
}

func seqRow(cols [][]*Cell, i int) []*Cell {
  row := make([]*Cell, len(cols))
  for j := range cols {
    row[j] = cols[j][i]
  }
  return row
}

func fnMap(args []*Cell) (*Cell, error) {
  if len(args) < 2 {
    return nil, fmt.Errorf("map: mindestens 3 Argumente nötig (typ f sequenz…)")
  }
  isVec, isNil, err := seqResultType("map", args[0], true)
  if err != nil {
    return nil, err
  }
  cols, n, err := seqColumns("map", args[2:])
  if err != nil {
    return nil, err
  }
  results := make([]*Cell, 0, n)
  for i := 0; i < n; i++ {
    r, err := apply(args[1], seqRow(cols, i))
    if err != nil {
      return nil, err
    }
    results = append(results, Primary(r))
  }
  if isNil {
    return MakeNil(), nil
  }
  return makeSeq(results, isVec), nil
}

func fnSome(args []*Cell) (*Cell, error) {
  if len(args) < 2 {
    return nil, fmt.Errorf("some: mindestens 2 Argumente nötig (p sequenz…)")
  }
  cols, n, err := seqColumns("some", args[1:])
  if err != nil {
    return nil, err
  }
  for i := 0; i < n; i++ {
    r, err := apply(args[0], seqRow(cols, i))
    if err != nil {
      return nil, err
    }
    if r = Primary(r); IsTruthy(r) {
      return r, nil
    }
  }
  return MakeNil(), nil
}

// seqSearchFn baut find/position/count und ihre -if-Varianten. Das erste
// Argument ist bei -if ein Prädikat, sonst das gesuchte Objekt (kann selbst
// ein Symbol sein — daher kein withFnDesignator).
func seqSearchFn(env *Env, name string, isIf bool) func([]*Cell) (*Cell, error) {
  return func(args []*Cell) (*Cell, error) {
    if len(args) < 2 {
      return nil, fmt.Errorf("%s: mindestens 2 Argumente nötig", name)
    }
    items, _, err := seqItems(name, args[1])
    if err != nil {
      return nil, err
    }
    var test, key *Cell
    rest := args[2:]
    if len(rest)%2 != 0 {
      return nil, fmt.Errorf("%s: Keyword ohne Wert", name)
    }
    for i := 0; i < len(rest); i += 2 {
      kw := rest[i]
      switch {
      case kw.Type == ATOM && kw.Val == ":key":
        key = resolveFn(env, rest[i+1])
      case kw.Type == ATOM && kw.Val == ":test" && !isIf:
        test = resolveFn(env, rest[i+1])
      default:
        return nil, fmt.Errorf("%s: unbekanntes Keyword %s", name, kw)
      }
    }
    item := args[0]
    if isIf {
      test = resolveFn(env, item)
    }
    count := 0
    for i, x := range items {
      k := x
      if key != nil {
        r, err := apply(key, []*Cell{x})
        if err != nil {
          return nil, err
        }
        k = Primary(r)
      }
      var hit bool
      switch {
      case isIf:
        r, err := apply(test, []*Cell{k})
        if err != nil {
          return nil, err
        }
        hit = IsTruthy(Primary(r))
      case test != nil:
        r, err := apply(test, []*Cell{item, k})
        if err != nil {
          return nil, err
        }
        hit = IsTruthy(Primary(r))
      default:
        hit = eql(item, k)
      }
      if !hit {
        continue
      }
      switch name {
      case "find", "find-if":
        return x, nil
      case "position", "position-if":
        return MakeNum(float64(i)), nil
      }
      count++
    }
    if name == "count" || name == "count-if" {
      return MakeNum(float64(count)), nil
    }
    return MakeNil(), nil
  }
}
