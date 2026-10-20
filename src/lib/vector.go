//**********************************************************************
//  lib/vector.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Opus 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261020
//**********************************************************************
// Eindimensionale CL-Vektoren mit O(1)-Zugriff. Mutabel und geteilt wie
// Hash-Tabellen; equal? vergleicht nur Identität (CL equal).
//   (make-array n &key initial-element)  (vector x …)
//   (aref v i)  (aset v i x)  (vector-push-extend x v) → neuer Index
// length/coerce/setf (stdlib.lisp) nutzen %vector-length/%vector->list/
// %list->vector. Kein Reader-Literal #(…) — Druckform ist nur Ausgabe.
//**********************************************************************

package lib

import (
  "fmt"
  "strings"
  "sync"
)

// Vector ist der mutable Zustand einer VECTOR-Cell; der Zustand liegt im
// Env-Feld, damit Cell nicht wächst (Size-Class, siehe types.go).
type Vector struct {
  mu    sync.RWMutex
  items []*Cell
}

func RegisterVector(env *Env) {
  _ = env.Set("make-array",         makeFn(fnMakeArray))
  _ = env.Set("vector",             makeFn(fnVector))
  _ = env.Set("aref",               makeFn(fnAref))
  _ = env.Set("aset",               makeFn(fnAset))
  _ = env.Set("vector-push-extend", makeFn(fnVectorPushExtend))
  _ = env.Set("%vector-length",     makeFn(fnVectorLength))
  _ = env.Set("%vector->list",      makeFn(fnVectorToList))
  _ = env.Set("%list->vector",      makeFn(fnListToVector))
}

func makeVector(items []*Cell) *Cell {
  return &Cell{Type: VECTOR, Env: &Vector{items: items}}
}

func asVector(name string, c *Cell) (*Vector, error) {
  if c != nil && c.Type == VECTOR {
    if v, ok := c.Env.(*Vector); ok {
      return v, nil
    }
  }
  return nil, fmt.Errorf("%s: Vektor erwartet, got %s", name, c)
}

// vectorIndex prüft i als ganze Zahl in 0..n-1.
func vectorIndex(name string, c *Cell, n int) (int, error) {
  if c.Type != NUMBER || c.Num != float64(int(c.Num)) {
    return 0, fmt.Errorf("%s: Index muss ganze Zahl sein, got %s", name, c)
  }
  i := int(c.Num)
  if i < 0 || i >= n {
    return 0, fmt.Errorf("%s: Index %d außerhalb 0..%d", name, i, n-1)
  }
  return i, nil
}

func vectorStr(c *Cell) string {
  v := c.Env.(*Vector)
  v.mu.RLock()
  defer v.mu.RUnlock()
  parts := make([]string, len(v.items))
  for i, x := range v.items {
    parts[i] = x.String()
  }
  return "#(" + strings.Join(parts, " ") + ")"
}

// make-array: nur eindimensional; einziges Keyword :initial-element.
func fnMakeArray(args []*Cell) (*Cell, error) {
  if len(args) < 1 {
    return nil, fmt.Errorf("make-array: Größe fehlt")
  }
  n := args[0]
  if n.Type != NUMBER || n.Num != float64(int(n.Num)) || n.Num < 0 {
    return nil, fmt.Errorf("make-array: Größe muss ganze Zahl >= 0 sein (nur eindimensional), got %s", n)
  }
  init := MakeNil()
  rest := args[1:]
  if len(rest)%2 != 0 {
    return nil, fmt.Errorf("make-array: Keyword ohne Wert")
  }
  for i := 0; i < len(rest); i += 2 {
    if rest[i].Type != ATOM || rest[i].Val != ":initial-element" {
      return nil, fmt.Errorf("make-array: unbekanntes Keyword %s (nur :initial-element)", rest[i])
    }
    init = rest[i+1]
  }
  items := make([]*Cell, int(n.Num))
  for i := range items {
    items[i] = init
  }
  return makeVector(items), nil
}

func fnVector(args []*Cell) (*Cell, error) {
  return makeVector(append([]*Cell(nil), args...)), nil
}

func fnAref(args []*Cell) (*Cell, error) {
  if len(args) != 2 {
    return nil, fmt.Errorf("aref: 2 Argumente erwartet (vektor index)")
  }
  v, err := asVector("aref", args[0])
  if err != nil { return nil, err }
  v.mu.RLock()
  defer v.mu.RUnlock()
  i, err := vectorIndex("aref", args[1], len(v.items))
  if err != nil { return nil, err }
  return v.items[i], nil
}

// aset: (aset v i x) → x, wie Emacs Lisp; Ziel von (setf (aref v i) x).
func fnAset(args []*Cell) (*Cell, error) {
  if len(args) != 3 {
    return nil, fmt.Errorf("aset: 3 Argumente erwartet (vektor index wert)")
  }
  v, err := asVector("aset", args[0])
  if err != nil { return nil, err }
  v.mu.Lock()
  defer v.mu.Unlock()
  i, err := vectorIndex("aset", args[1], len(v.items))
  if err != nil { return nil, err }
  v.items[i] = args[2]
  return args[2], nil
}

// vector-push-extend: (vector-push-extend x v) → Index des neuen Elements.
func fnVectorPushExtend(args []*Cell) (*Cell, error) {
  if len(args) != 2 {
    return nil, fmt.Errorf("vector-push-extend: 2 Argumente erwartet (wert vektor)")
  }
  v, err := asVector("vector-push-extend", args[1])
  if err != nil { return nil, err }
  v.mu.Lock()
  defer v.mu.Unlock()
  v.items = append(v.items, args[0])
  return MakeNum(float64(len(v.items) - 1)), nil
}

func fnVectorLength(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("%%vector-length: 1 Argument erwartet")
  }
  v, err := asVector("length", args[0])
  if err != nil { return nil, err }
  v.mu.RLock()
  defer v.mu.RUnlock()
  return MakeNum(float64(len(v.items))), nil
}

func fnVectorToList(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("%%vector->list: 1 Argument erwartet")
  }
  v, err := asVector("coerce", args[0])
  if err != nil { return nil, err }
  v.mu.RLock()
  defer v.mu.RUnlock()
  return SliceToCell(append([]*Cell(nil), v.items...)), nil
}

func fnListToVector(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("%%list->vector: 1 Argument erwartet")
  }
  return makeVector(CellToSlice(args[0])), nil
}
