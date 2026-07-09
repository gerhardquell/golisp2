//**********************************************************************
//  lib/env.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260223
//**********************************************************************

package lib

import "fmt"

// Env ist eine verkettete Umgebung: lokaler Scope -> aeusserer Scope.
// Root-Env (parent == nil) nutzt eine Hash-Map fuer ~80+ eingebaute Symbole.
// Frame-Envs (parent != nil) nutzen inline singleName/singleVal fuer den
// ersten Eintrag und parallele Slices fuer weitere Eintraege. Damit fallen
// die map-Allokationen pro Lambda-/Let-Frame weg.
type Env struct {
  parent     *Env
  // Root-Modus: parent == nil
  vars       map[string]*Cell
  // Frame-Modus: parent != nil
  singleName string
  singleVal  *Cell
  names      []string
  vals       []*Cell
}

// NewEnv erzeugt ein Root-Env (parent == nil) mit Map, sonst ein Frame-Env
// mit inline + Slice-Speicher.
func NewEnv(parent *Env) *Env {
  if parent == nil {
    return &Env{vars: make(map[string]*Cell)}
  }
  return &Env{parent: parent}
}

// Get sucht einen Namen – erst lokal, dann im aeusseren Scope
func (e *Env) Get(name string) (*Cell, error) {
  if e.parent == nil {
    if val, ok := e.vars[name]; ok {
      return val, nil
    }
    return nil, fmt.Errorf("env: unbekanntes Symbol '%s'", name)
  }
  if e.singleName == name {
    return e.singleVal, nil
  }
  for i, n := range e.names {
    if n == name {
      return e.vals[i], nil
    }
  }
  if e.parent != nil {
    return e.parent.Get(name)
  }
  return nil, fmt.Errorf("env: unbekanntes Symbol '%s'", name)
}

// Set legt einen Wert im aktuellen Scope ab
func (e *Env) Set(name string, val *Cell) {
  if e.parent == nil {
    e.vars[name] = val
    return
  }
  if e.singleName == "" {
    e.singleName = name
    e.singleVal = val
    return
  }
  if e.singleName == name {
    e.singleVal = val
    return
  }
  for i, n := range e.names {
    if n == name {
      e.vals[i] = val
      return
    }
  }
  e.names = append(e.names, name)
  e.vals = append(e.vals, val)
}

// Root liefert die aeusserste Umgebung (Globalenv). Common-Lisp-Semantik
// fuer (eval form): Auswertung im globalen Environment, unabhaengig vom
// dynamischen Lambda-Scope. Ohne dies wuerde (defun ...) aus einem
// REPL-Eval heraus lokal im Child-Env definiert und ginge verloren.
func (e *Env) Root() *Env {
  cur := e
  for cur.parent != nil {
    cur = cur.parent
  }
  return cur
}

// Symbols sammelt alle bekannten Namen (inkl. aeussere Scopes, ohne Duplikate)
func (e *Env) Symbols() []string {
  seen := make(map[string]bool)
  var result []string
  for cur := e; cur != nil; cur = cur.parent {
    if cur.parent == nil {
      for name := range cur.vars {
        if !seen[name] {
          seen[name] = true
          result = append(result, name)
        }
      }
      continue
    }
    if cur.singleName != "" && !seen[cur.singleName] {
      seen[cur.singleName] = true
      result = append(result, cur.singleName)
    }
    for _, name := range cur.names {
      if !seen[name] {
        seen[name] = true
        result = append(result, name)
      }
    }
  }
  return result
}

// Update aendert einen bestehenden Wert (fuer set!)
func (e *Env) Update(name string, val *Cell) error {
  if e.parent == nil {
    if _, ok := e.vars[name]; ok {
      e.vars[name] = val
      return nil
    }
    return fmt.Errorf("env: set! – Symbol '%s' nicht gefunden", name)
  }
  if e.singleName == name {
    e.singleVal = val
    return nil
  }
  for i, n := range e.names {
    if n == name {
      e.vals[i] = val
      return nil
    }
  }
  if e.parent != nil {
    return e.parent.Update(name, val)
  }
  return fmt.Errorf("env: set! – Symbol '%s' nicht gefunden", name)
}
