//**********************************************************************
//  lib/docstring_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260820
//**********************************************************************
// Tests für Docstring-Registry (symbol -> Docstring) und die
// (documentation ...)-Primitive.
//**********************************************************************

package lib

import (
  "strings"
  "sync"
  "testing"
)

func TestRegisterAndLookupDocstring(t *testing.T) {
  ClearDocstrings()
  RegisterDocstring("foo", "tut etwas")
  doc, ok := LookupDocstring("foo")
  if !ok || doc != "tut etwas" {
    t.Fatalf("got %q, %v", doc, ok)
  }
}

func TestLookupUnknownDocstring(t *testing.T) {
  ClearDocstrings()
  _, ok := LookupDocstring("nope")
  if ok {
    t.Fatalf("nope sollte nicht gefunden werden")
  }
}

func TestConcurrentRegisterDocstring(t *testing.T) {
  ClearDocstrings()
  var wg sync.WaitGroup
  for i := 0; i < 50; i++ {
    wg.Add(1)
    go func() {
      defer wg.Done()
      RegisterDocstring("c", "doc")
    }()
  }
  wg.Wait()
  _, ok := LookupDocstring("c")
  if !ok {
    t.Fatalf("c nach concurrent writes nicht gefunden")
  }
}

func TestDefunWithDocstringRegistersIt(t *testing.T) {
  ClearDocstrings()
  res, err := evalStr(`
    (defun quadrat (x) "Quadriert x." (* x x))
    (documentation 'quadrat 'function)
  `)
  if err != nil {
    t.Fatalf("evalStr: %v", err)
  }
  if res.Type != STRING || res.Val != "Quadriert x." {
    t.Fatalf("got %s", res.String())
  }
}

func TestDefunWithDocstringStillCallable(t *testing.T) {
  res, err := evalStr(`
    (defun quadrat (x) "Quadriert x." (* x x))
    (quadrat 5)
  `)
  if err != nil {
    t.Fatalf("evalStr: %v", err)
  }
  if res.Num != 25 {
    t.Fatalf("got %s, want 25", res.String())
  }
}

func TestDefunWithoutDocstringHasNoDoc(t *testing.T) {
  res, err := evalStr(`
    (defun quadrat2 (x) (* x x))
    (documentation 'quadrat2 'function)
  `)
  if err != nil {
    t.Fatalf("evalStr: %v", err)
  }
  if res.Type != NIL {
    t.Fatalf("got %s, want nil", res.String())
  }
}

// Ein einzelner String-Body bleibt CL-konform der Rückgabewert, kein
// Docstring — sonst könnte man keine Ein-Zeilen-Funktion schreiben, die
// schlicht einen String liefert.
func TestDefunSingleStringBodyIsReturnValueNotDoc(t *testing.T) {
  res, err := evalStr(`
    (defun gruss () "hallo")
    (gruss)
  `)
  if err != nil {
    t.Fatalf("evalStr: %v", err)
  }
  if res.Type != STRING || res.Val != "hallo" {
    t.Fatalf("got %s, want \"hallo\"", res.String())
  }
  ClearDocstrings() // Setup für nächsten Check
  if _, err := evalStr(`(defun gruss () "hallo")`); err != nil {
    t.Fatalf("evalStr: %v", err)
  }
  if _, ok := LookupDocstring("gruss"); ok {
    t.Fatalf("Ein-String-Body wurde faelschlich als Docstring registriert")
  }
}

func TestDefmacroWithDocstringRegistersIt(t *testing.T) {
  ClearDocstrings()
  res, err := evalStr(`
    (defmacro when2 (test body) "Wie when." (list 'if test body))
    (documentation 'when2 'function)
  `)
  if err != nil {
    t.Fatalf("evalStr: %v", err)
  }
  if res.Type != STRING || res.Val != "Wie when." {
    t.Fatalf("got %s", res.String())
  }
}

func TestRedefinitionWithoutDocClearsOldDoc(t *testing.T) {
  ClearDocstrings()
  res, err := evalStr(`
    (defun f (x) "alte Doku" (+ x 1))
    (defun f (x) (+ x 2))
    (documentation 'f 'function)
  `)
  if err != nil {
    t.Fatalf("evalStr: %v", err)
  }
  if res.Type != NIL {
    t.Fatalf("got %s, alte Doku haette geloescht werden muessen", res.String())
  }
}

func TestMakunboundRemovesDocstring(t *testing.T) {
  ClearDocstrings()
  ClearDefinitions()
  if _, err := evalStr(`(defun f2 (x) "doc" (+ x 1))`); err != nil {
    t.Fatalf("evalStr: %v", err)
  }
  if _, ok := LookupDocstring("f2"); !ok {
    t.Fatalf("f2 sollte vor makunbound registriert sein")
  }
  if _, err := evalStr(`(defun f2 (x) "doc" (+ x 1)) (makunbound 'f2)`); err != nil {
    t.Fatalf("evalStr: %v", err)
  }
  if _, ok := LookupDocstring("f2"); ok {
    t.Fatalf("f2-Docstring haette per makunbound entfernt werden sollen")
  }
}

func TestDocumentationUnknownDocTypeReturnsNil(t *testing.T) {
  ClearDocstrings()
  res, err := evalStr(`
    (defun f3 (x) "doc" (+ x 1))
    (documentation 'f3 'variable)
  `)
  if err != nil {
    t.Fatalf("evalStr: %v", err)
  }
  if res.Type != NIL {
    t.Fatalf("got %s, want nil fuer unbekannten doc-type", res.String())
  }
}

func TestDocumentationRegistered(t *testing.T) {
  // Registrierung in BaseEnv: darf nicht "unbekanntes Symbol" melden.
  if _, err := evalStr(`(documentation 'foo 'function)`); err != nil &&
    strings.Contains(err.Error(), "unbekanntes Symbol") {
    t.Fatalf("documentation nicht in BaseEnv registriert: %v", err)
  }
}
