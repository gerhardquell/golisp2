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

import "sync"

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
