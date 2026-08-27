//**********************************************************************
//  lib/defloc_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260624
//**********************************************************************
// Tests für Definition-Registry (symbol -> file/line)
//**********************************************************************

package lib

import (
  "os"
  "strings"
  "sync"
  "testing"
)

func TestRegisterAndLookupDefinition(t *testing.T) {
  ClearDefinitions()
  RegisterDefinition("foo", "/a/b.lisp", 7)
  loc, ok := LookupDefinition("foo")
  if !ok {
    t.Fatalf("foo nicht gefunden")
  }
  if loc.File != "/a/b.lisp" || loc.Line != 7 {
    t.Fatalf("got %+v", loc)
  }
}

func TestLookupUnknownDefinition(t *testing.T) {
  ClearDefinitions()
  _, ok := LookupDefinition("nope")
  if ok {
    t.Fatalf("nope sollte nicht gefunden werden")
  }
}

func TestConcurrentRegisterDefinition(t *testing.T) {
  ClearDefinitions()
  var wg sync.WaitGroup
  for i := 0; i < 50; i++ {
    wg.Add(1)
    go func(n int) {
      defer wg.Done()
      RegisterDefinition("c", "/c.lisp", n)
    }(i)
  }
  wg.Wait()
  _, ok := LookupDefinition("c")
  if !ok {
    t.Fatalf("c nach concurrent writes nicht gefunden")
  }
}

func TestDefinedInPrimitive(t *testing.T) {
  ClearDefinitions()
  dir := t.TempDir()
  f := dir + "/mod.lisp"
  if err := os.WriteFile(f, []byte("(defun dummy () 1)"), 0644); err != nil {
    t.Fatal(err)
  }
  RegisterDefinition("alpha", f, 3)
  RegisterDefinition("beta", f, 9)
  RegisterDefinition("gamma", "/irgendwo/anders.lisp", 1)

  got, err := fnDefinedIn([]*Cell{MakeStr(f)})
  if err != nil {
    t.Fatalf("defined-in: %v", err)
  }
  // sortiert: alpha vor beta
  want := List(MakeAtom("alpha"), MakeAtom("beta"))
  if got.String() != want.String() {
    t.Fatalf("got %s, want %s", got.String(), want.String())
  }
}

func TestDefinedInNoMatch(t *testing.T) {
  ClearDefinitions()
  dir := t.TempDir()
  f := dir + "/leer.lisp"
  if err := os.WriteFile(f, []byte(""), 0644); err != nil {
    t.Fatal(err)
  }
  got, err := fnDefinedIn([]*Cell{MakeStr(f)})
  if err != nil {
    t.Fatalf("defined-in: %v", err)
  }
  if got.String() != MakeNil().String() {
    t.Fatalf("leere Liste erwartet, got %s", got.String())
  }
}

func TestDefinedInMissingFile(t *testing.T) {
  if _, err := fnDefinedIn([]*Cell{MakeStr("/gibts/garantiert/nicht.lisp")}); err == nil {
    t.Fatal("Fehler bei nicht-existierendem Pfad erwartet")
  }
}

func TestDefinedInRegistered(t *testing.T) {
  // Registrierung in BaseEnv: darf nicht "unbekanntes Symbol" melden.
  // (CWD von go test ist lib/ — "defloc.go" existiert dort.)
  if _, err := evalStr(`(defined-in "defloc.go")`); err != nil &&
    strings.Contains(err.Error(), "unbekanntes Symbol") {
    t.Fatalf("defined-in nicht in BaseEnv registriert: %v", err)
  }
}
