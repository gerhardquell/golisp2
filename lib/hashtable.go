//**********************************************************************
//  lib/hashtable.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260723
//**********************************************************************
// CL-Hashtables: make-hash-table, gethash (mit MV: Wert + gefunden?),
// puthash (Ziel der setf-Expansion), remhash, clrhash, hash-table-count,
// hash-table-p, maphash.
//
// Repräsentation: Go-Map hinter Cell.Ht (mutable, Pointer-Identität für
// eq). Der :test-Modus ist keine Doppel-Implementierung, sondern nur die
// Wahl der Key-Funktion: eql/eq → Typ-Präfix + Wert, equal → strukturell.
// Eigener RWMutex pro Tabelle (golisp2 hat Goroutinen-Primitiven).
//**********************************************************************

package lib

import (
  "fmt"
  "strconv"
  "sync"
)

// hashEntry hält neben dem Wert auch den Original-Schlüssel als Cell,
// damit maphash die echten Schlüsselobjekte übergeben kann (CL).
type hashEntry struct {
  key *Cell
  val *Cell
}

// HashTable ist der mutable Zustand einer HASHTABLE-Cell.
type HashTable struct {
  mu   sync.RWMutex
  m    map[string]hashEntry
  test string // "eql" (default, auch "eq") oder "equal"
}

// keyOf bildet den Map-Schlüssel je nach Testmodus.
// eql/eq: Zahlen nach Wert, Symbole nach Name, Strings nach Inhalt
// (bewusste Abweichung: CL-eql vergleicht Strings nach Identität —
// golisp2-Strings haben keine vom Nutzer beobachtbare Identität),
// alles andere nach Pointer. equal: struktureller Vergleich via Print.
func (ht *HashTable) keyOf(c *Cell) string {
  if ht.test == "equal" {
    return "E:" + c.String()
  }
  switch c.Type {
  case NUMBER:
    return "n:" + strconv.FormatFloat(c.Num, 'g', -1, 64)
  case ATOM:
    return "a:" + c.Val
  case STRING:
    return "s:" + c.Val
  case NIL:
    return "a:nil"
  default:
    return fmt.Sprintf("p:%p", c)
  }
}

// RegisterHashtables registriert die Hashtabellen-Primitiven (BaseEnv).
func RegisterHashtables(env *Env) {
  _ = env.Set("make-hash-table", makeFn(fnMakeHashTable))
  _ = env.Set("gethash", makeFn(fnGethash))
  _ = env.Set("puthash", makeFn(fnPuthash))
  _ = env.Set("remhash", makeFn(fnRemhash))
  _ = env.Set("clrhash", makeFn(fnClrhash))
  _ = env.Set("hash-table-count", makeFn(fnHashTableCount))
  _ = env.Set("hash-table-p", makeFn(fnHashTableP))
  _ = env.Set("maphash", makeFn(fnMaphash))
}

// asTable prüft HASHTABLE-Typ und liefert den Zustand.
func asTable(name string, c *Cell) (*HashTable, error) {
  if c == nil || c.Type != HASHTABLE || c.Ht == nil {
    return nil, fmt.Errorf("%s: Hashtabelle erwartet, got %s", name, c)
  }
  return c.Ht, nil
}

// make-hash-table: (&key (test 'eql)) → neue leere Tabelle.
// Keyword-Paare werden positional aus dem Args-Slice gelesen
// (Primitiven bekommen keine &key-Bindung).
func fnMakeHashTable(args []*Cell) (*Cell, error) {
  test := "eql"
  for i := 0; i+1 < len(args); i += 2 {
    if args[i].Type != ATOM || args[i].Val != ":test" {
      return nil, fmt.Errorf("make-hash-table: unbekanntes Keyword %s", args[i])
    }
    if args[i+1].Type != ATOM {
      return nil, fmt.Errorf("make-hash-table: Test muss Symbol sein, got %s", args[i+1])
    }
    test = args[i+1].Val
  }
  if len(args)%2 != 0 {
    return nil, fmt.Errorf("make-hash-table: Keyword ohne Wert")
  }
  switch test {
  case "eq", "eql", "equal":
  default:
    return nil, fmt.Errorf("make-hash-table: Test '%s' nicht unterstützt (eq, eql, equal)", test)
  }
  return &Cell{Type: HASHTABLE, Ht: &HashTable{m: make(map[string]hashEntry), test: test}}, nil
}

// gethash: (key table &optional default) → zwei Werte (CL): Wert (oder
// default) und gefunden?-Flag. MV-fähige Aufrufer sehen beides,
// Nicht-MV-Kontexte nur den Wert (Primary-Regel).
func fnGethash(args []*Cell) (*Cell, error) {
  if len(args) < 2 || len(args) > 3 {
    return nil, fmt.Errorf("gethash: 2-3 Argumente nötig")
  }
  ht, err := asTable("gethash", args[1])
  if err != nil {
    return nil, err
  }
  ht.mu.RLock()
  e, found := ht.m[ht.keyOf(args[0])]
  ht.mu.RUnlock()
  if found {
    return MakeValues([]*Cell{e.val, MakeAtom("t")}), nil
  }
  def := MakeNil()
  if len(args) == 3 {
    def = args[2]
  }
  return MakeValues([]*Cell{def, MakeNil()}), nil
}

// puthash: (key table value) → value. Internes Primitiv, Ziel der
// setf-Expansion für (setf (gethash k t) v) — siehe stdlib.lisp.
func fnPuthash(args []*Cell) (*Cell, error) {
  if len(args) != 3 {
    return nil, fmt.Errorf("puthash: 3 Argumente nötig")
  }
  ht, err := asTable("puthash", args[1])
  if err != nil {
    return nil, err
  }
  ht.mu.Lock()
  ht.m[ht.keyOf(args[0])] = hashEntry{key: args[0], val: args[2]}
  ht.mu.Unlock()
  return args[2], nil
}

// remhash: (key table) → t wenn Eintrag entfernt, nil wenn nicht vorhanden.
func fnRemhash(args []*Cell) (*Cell, error) {
  if len(args) != 2 {
    return nil, fmt.Errorf("remhash: 2 Argumente nötig")
  }
  ht, err := asTable("remhash", args[1])
  if err != nil {
    return nil, err
  }
  k := ht.keyOf(args[0])
  ht.mu.Lock()
  _, found := ht.m[k]
  if found {
    delete(ht.m, k)
  }
  ht.mu.Unlock()
  if found {
    return MakeAtom("t"), nil
  }
  return MakeNil(), nil
}

// clrhash: (table) → leert die Tabelle, liefert sie zurück (CL).
func fnClrhash(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("clrhash: 1 Argument nötig")
  }
  ht, err := asTable("clrhash", args[0])
  if err != nil {
    return nil, err
  }
  ht.mu.Lock()
  ht.m = make(map[string]hashEntry)
  ht.mu.Unlock()
  return args[0], nil
}

// hash-table-count: (table) → Anzahl der Einträge.
func fnHashTableCount(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("hash-table-count: 1 Argument nötig")
  }
  ht, err := asTable("hash-table-count", args[0])
  if err != nil {
    return nil, err
  }
  ht.mu.RLock()
  n := len(ht.m)
  ht.mu.RUnlock()
  return MakeNum(float64(n)), nil
}

// hash-table-p: (x) → t wenn x eine Hashtabelle ist.
func fnHashTableP(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("hash-table-p: 1 Argument nötig")
  }
  if args[0].Type == HASHTABLE {
    return MakeAtom("t"), nil
  }
  return MakeNil(), nil
}

// maphash: (fn table) → ruft (funcall fn key value) für jeden Eintrag,
// liefert nil (CL). Iterationsreihenfolge ist bewusst unspezifiziert
// (Go-Map) — wie in CL, wo sie implementierungsabhängig ist.
func fnMaphash(args []*Cell) (*Cell, error) {
  if len(args) != 2 {
    return nil, fmt.Errorf("maphash: 2 Argumente nötig")
  }
  ht, err := asTable("maphash", args[1])
  if err != nil {
    return nil, err
  }
  // Snapshot unter Lock, Callbacks außerhalb — fn darf die Tabelle
  // selbst modifizieren, ohne den Lock zu verklemmen (CL erlaubt
  // add/remove des aktuellen Keys).
  ht.mu.RLock()
  entries := make([]hashEntry, 0, len(ht.m))
  for _, e := range ht.m {
    entries = append(entries, e)
  }
  ht.mu.RUnlock()
  for _, e := range entries {
    if _, err := apply(args[0], []*Cell{e.key, e.val}); err != nil {
      return nil, err
    }
  }
  return MakeNil(), nil
}
