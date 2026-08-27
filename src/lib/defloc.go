//**********************************************************************
//  lib/defloc.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260624
//**********************************************************************
// Definition-Registry: symbol -> (file, line). Thread-safe via
// sync.RWMutex (parfunc-safe). Genutzt von defun/defmacro/define und
// swank:find-definitions-for-emacs (M-.).
//**********************************************************************

package lib

import (
  "fmt"
  "path/filepath"
  "sort"
  "sync"
)

// DefLoc speichert Quellposition einer Definition.
type DefLoc struct {
  File string
  Line int
}

var (
  defMu       sync.RWMutex
  definitions = map[string]DefLoc{}
)

// RegisterDefinition merkt sich die Quellposition eines Symbols.
// Last-write-wins: Neu-Definition überschreibt alten Eintrag.
func RegisterDefinition(name, file string, line int) {
  defMu.Lock()
  defer defMu.Unlock()
  definitions[name] = DefLoc{File: file, Line: line}
}

// LookupDefinition liefert die gespeicherte Quellposition oder ok=false.
func LookupDefinition(name string) (DefLoc, bool) {
  defMu.RLock()
  defer defMu.RUnlock()
  loc, ok := definitions[name]
  return loc, ok
}

// ClearDefinitions leert die Registry (nur für Tests).
func ClearDefinitions() {
  defMu.Lock()
  defer defMu.Unlock()
  definitions = map[string]DefLoc{}
}

// RemoveDefinition entfernt den Registry-Eintrag (makunbound).
func RemoveDefinition(name string) {
  defMu.Lock()
  defer defMu.Unlock()
  delete(definitions, name)
}

// defined-in: (defined-in "pfad") → sortierte Liste der Symbole, deren
// DefLoc.File dem normalisierten Pfad entspricht. Normalisierung identisch
// zu load (resolvePath + filepath.Abs), damit relative Angaben matchen.
// Leere Liste, wenn die Datei nichts definiert hat.
func fnDefinedIn(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("defined-in: 1 Argument nötig")
  }
  if args[0].Type != STRING {
    return nil, fmt.Errorf("defined-in: String erwartet")
  }
  resolved, err := resolvePath(args[0].Val)
  if err != nil {
    return nil, fmt.Errorf("defined-in: %v", err)
  }
  if abs, aerr := filepath.Abs(resolved); aerr == nil {
    resolved = abs
  }
  defMu.RLock()
  names := []string{}
  for name, loc := range definitions {
    if loc.File == resolved {
      names = append(names, name)
    }
  }
  defMu.RUnlock()
  sort.Strings(names)
  cells := make([]*Cell, len(names))
  for i, n := range names {
    cells[i] = MakeAtom(n)
  }
  return List(cells...), nil
}
