//**********************************************************************
//  lib/env.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260223
//**********************************************************************

package lib

import (
  "fmt"
  "sync"
  "sync/atomic"
)

// envPool wiederverwendet Frame-Envs (parent != nil), die nicht mehr
// referenziert werden. Root-Envs (parent == nil) werden nie gepoolt.
var envPool = sync.Pool{
  New: func() interface{} { return &Env{} },
}

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
  // shared == true: von mindestens einer Closure referenziert – darf nicht
  // in den Pool zurueckgegeben werden.
  shared     bool
  // mu schuetzt Lese-/Schreibzugriffe fuer parfunc (mehrere Goroutinen
  // koennen dasselbe Env gleichzeitig nutzen).
  mu         sync.RWMutex
}

// redefinePolicy steuert das Verhalten beim Überschreiben von Go-Primitiven
// (FUNC-Bindungen) im Root-Env. Default ist warn.
type redefinePolicy int32

const (
  redefineAllow redefinePolicy = iota
  redefineWarn
  redefineError
)

// redefinePolicyAtomic ist thread-sicher, da parfunc mehrere Goroutinen im
// Root-Env gleichzeitig Set aufrufen kann.
var redefinePolicyAtomic atomic.Int32

func init() {
  redefinePolicyAtomic.Store(int32(redefineWarn))
}

// SetRedefinePolicy setzt die Guard-Policy anhand des Namens.
func SetRedefinePolicy(name string) error {
  var p redefinePolicy
  switch name {
  case "allow":
    p = redefineAllow
  case "warn":
    p = redefineWarn
  case "error":
    p = redefineError
  default:
    return fmt.Errorf("redefine-policy: unbekannte Policy %q", name)
  }
  redefinePolicyAtomic.Store(int32(p))
  return nil
}

// GetRedefinePolicy liefert den aktuellen Policy-Namen.
func GetRedefinePolicy() string {
  switch redefinePolicy(redefinePolicyAtomic.Load()) {
  case redefineAllow:
    return "allow"
  case redefineWarn:
    return "warn"
  case redefineError:
    return "error"
  }
  return "allow"
}

// onRootRedefine wird aus Env.Set gerufen, wenn im Root-Env ein bestehendes
// FUNC-Binding überschrieben wird. Als Variable hinterlegt, damit später das
// Define-Log am selben Hook ansetzen kann.
var onRootRedefine = defaultOnRootRedefine

func defaultOnRootRedefine(name string, old, new *Cell) error {
  loc, _ := LookupDefinition(name)
  ev := RedefEvent{
    Name:    name,
    OldKind: kindOf(old),
    NewKind: kindOf(new),
    OldFile: loc.File,
    OldLine: loc.Line,
    Action:  policyAction(),
  }
  err := applyRedefPolicy(name, "war FUNC")
  logRedef(ev)
  return err
}

// NewEnv erzeugt ein Root-Env (parent == nil) mit Map, sonst ein Frame-Env
// mit inline + Slice-Speicher (aus dem Pool wenn moeglich).
func NewEnv(parent *Env) *Env {
  if parent == nil {
    return &Env{vars: make(map[string]*Cell)}
  }
  e := envPool.Get().(*Env)
  e.parent = parent
  e.singleName = ""
  e.singleVal = nil
  e.names = e.names[:0]
  if cap(e.vals) > 0 {
    for i := range e.vals { e.vals[i] = nil }
  }
  e.vals = e.vals[:0]
  e.shared = false
  return e
}

// freeEnv gibt ein Frame-Env in den Pool zurueck, sofern es nicht shared
// ist. Root-Envs werden ignoriert.
func freeEnv(e *Env) {
  if e == nil || e.parent == nil || e.shared {
    return
  }
  e.singleName = ""
  e.singleVal = nil
  for i := range e.vals { e.vals[i] = nil }
  e.names = e.names[:0]
  e.vals = e.vals[:0]
  e.parent = nil
  e.shared = false
  envPool.Put(e)
}

// Get sucht einen Namen – erst lokal, dann im aeusseren Scope
func (e *Env) Get(name string) (*Cell, error) {
  e.mu.RLock()
  if e.parent == nil {
    if val, ok := e.vars[name]; ok {
      e.mu.RUnlock()
      return val, nil
    }
    e.mu.RUnlock()
    return nil, fmt.Errorf("env: unbekanntes Symbol '%s'", name)
  }
  if e.singleName == name {
    val := e.singleVal
    e.mu.RUnlock()
    return val, nil
  }
  for i, n := range e.names {
    if n == name {
      val := e.vals[i]
      e.mu.RUnlock()
      return val, nil
    }
  }
  parent := e.parent
  e.mu.RUnlock()
  return parent.Get(name)
}

// Set legt einen Wert im aktuellen Scope ab.
// Auf dem Root-Env (parent == nil) wird eine Redefinition existierender
// FUNC-Bindungen durch die Policy gesteuert.
func (e *Env) Set(name string, val *Cell) error {
  e.mu.Lock()
  defer e.mu.Unlock()
  if e.parent == nil {
    if old, ok := e.vars[name]; ok && old != nil && old.Type == FUNC {
      if err := onRootRedefine(name, old, val); err != nil {
        return err
      }
    }
    e.vars[name] = val
    return nil
  }
  if e.singleName == "" {
    e.singleName = name
    e.singleVal = val
    return nil
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
  e.names = append(e.names, name)
  e.vals = append(e.vals, val)
  return nil
}

// Root liefert die aeusserste Umgebung (Globalenv). Common-Lisp-Semantik
// fuer (eval form): Auswertung im globalen Environment, unabhaengig vom
// dynamischen Lambda-Scope. Ohne dies wuerde (defun ...) aus einem
// REPL-Eval heraus lokal im Child-Env definiert und ginge verloren.
func (e *Env) Root() *Env {
  cur := e
  for {
    cur.mu.RLock()
    if cur.parent == nil {
      cur.mu.RUnlock()
      return cur
    }
    next := cur.parent
    cur.mu.RUnlock()
    cur = next
  }
}

// UnsetRoot entfernt eine Bindung aus dem Root-Env.
// Liefert die entfernte Zelle; ok=false wenn nicht gebunden oder kein Root.
func (e *Env) UnsetRoot(name string) (*Cell, bool) {
  e.mu.Lock()
  defer e.mu.Unlock()
  if e.parent != nil {
    return nil, false
  }
  old, ok := e.vars[name]
  if !ok {
    return nil, false
  }
  delete(e.vars, name)
  return old, true
}

// Symbols sammelt alle bekannten Namen (inkl. aeussere Scopes, ohne Duplikate)
func (e *Env) Symbols() []string {
  seen := make(map[string]bool)
  var result []string
  cur := e
  for cur != nil {
    cur.mu.RLock()
    if cur.parent == nil {
      for name := range cur.vars {
        if !seen[name] {
          seen[name] = true
          result = append(result, name)
        }
      }
      cur.mu.RUnlock()
      break
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
    next := cur.parent
    cur.mu.RUnlock()
    cur = next
  }
  return result
}

// Update aendert einen bestehenden Wert (fuer set!)
func (e *Env) Update(name string, val *Cell) error {
  e.mu.Lock()
  defer e.mu.Unlock()
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
