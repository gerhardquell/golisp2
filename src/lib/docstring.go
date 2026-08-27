//**********************************************************************
//  lib/docstring.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260820
//**********************************************************************
// Docstring-Registry: symbol -> Dokumentationsstring. Thread-safe via
// sync.RWMutex (parfunc-safe). Analog zu defloc.go (Quellposition), nur
// für den optionalen Docstring aus defun/defmacro. Genutzt von
// (documentation 'name 'function).
//**********************************************************************

package lib

import (
  "fmt"
  "sync"
)

var (
  docMu   sync.RWMutex
  docs    = map[string]string{}
)

// RegisterDocstring merkt sich den Docstring eines Symbols.
// Last-write-wins: Neu-Definition überschreibt alten Eintrag.
func RegisterDocstring(name, doc string) {
  docMu.Lock()
  defer docMu.Unlock()
  docs[name] = doc
}

// LookupDocstring liefert den gespeicherten Docstring oder ok=false.
func LookupDocstring(name string) (string, bool) {
  docMu.RLock()
  defer docMu.RUnlock()
  doc, ok := docs[name]
  return doc, ok
}

// RemoveDocstring entfernt den Registry-Eintrag (Redefinition ohne
// Docstring, makunbound).
func RemoveDocstring(name string) {
  docMu.Lock()
  defer docMu.Unlock()
  delete(docs, name)
}

// ClearDocstrings leert die Registry (nur für Tests).
func ClearDocstrings() {
  docMu.Lock()
  defer docMu.Unlock()
  docs = map[string]string{}
}

// extractDocstring trennt einen optionalen Docstring vom Function-Body.
// CL-Regel: ein String-Literal direkt nach der Parameterliste ist nur
// dann Docstring, wenn danach noch mindestens eine weitere Form folgt —
// sonst ist er der Rückgabewert (Ein-Zeilen-Funktion, die einen String
// liefert).
func extractDocstring(body *Cell) (doc string, found bool, rest *Cell) {
  if body != nil && body.Type == LIST && body.Car != nil && body.Car.Type == STRING &&
     body.Cdr != nil && body.Cdr.Type == LIST {
    return body.Car.Val, true, body.Cdr
  }
  return "", false, body
}

// documentation: (documentation 'name 'function) → Docstring oder nil.
// Nur der doc-type 'function liefert etwas (deckt defun UND defmacro ab,
// wie in CL) — jeder andere doc-type liefert nil, kein Error (CL-konform:
// unbekannte/ungenutzte doc-types sind kein Fehlerfall).
func fnDocumentation(args []*Cell) (*Cell, error) {
  if len(args) != 2 {
    return nil, fmt.Errorf("documentation: 2 Argumente nötig (symbol doc-type)")
  }
  if args[0].Type != ATOM {
    return nil, fmt.Errorf("documentation: Symbol erwartet, got %s", args[0])
  }
  if args[1].Type != ATOM || args[1].Val != "function" {
    return MakeNil(), nil
  }
  doc, ok := LookupDocstring(args[0].Val)
  if !ok {
    return MakeNil(), nil
  }
  return MakeStr(doc), nil
}
