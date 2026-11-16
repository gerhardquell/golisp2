//**********************************************************************
//  lib/env_frame_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260805
//**********************************************************************
// Charakterisierungsnetz fuer die Frame-Env-Repraesentation.
//
// Ein Frame-Env (parent != nil) legt den ERSTEN Eintrag inline ab
// (singleName/singleVal) und alle weiteren in Slices. env_test.go deckte
// nur Root-Env und die Redefine-Policy ab — der Slice-Pfad und der
// Uebergang inline -> Slice waren untestet, obwohl jede Lambda mit mehr
// als einem Parameter dort landet.
//
// Netz vor dem Umbau der Feldstruktur (PerfTODO §6, Richtung 1).
// White-Box im eigenen Paket, weil die Repraesentation genau der
// Gegenstand ist (vgl. evalStr in eval_test.go).
//**********************************************************************

package lib

import (
  "sort"
  "testing"
)

// mustGet liest name und schlaegt fehl, wenn ungebunden.
func mustGet(t *testing.T, e *Env, name string) string {
  t.Helper()
  v, err := e.Get(name)
  if err != nil {
    t.Fatalf("Get(%q): %v", name, err)
  }
  return v.String()
}

// TestFrameInlineToSliceTransition: der erste Eintrag geht inline, ab dem
// zweiten in den Slice-Speicher. Beide muessen ueber Get erreichbar sein,
// und die Reihenfolge der Eintragung darf keine Rolle spielen.
func TestFrameInlineToSliceTransition(t *testing.T) {
  root := NewEnv(nil)
  f := NewEnv(root)

  // erster Eintrag: inline
  if err := f.Set("a", MakeInt(1)); err != nil {
    t.Fatal(err)
  }
  if got := mustGet(t, f, "a"); got != "1" {
    t.Fatalf("a = %s, want 1", got)
  }

  // weitere Eintraege: Slice-Pfad
  for i, name := range []string{"b", "c", "d", "e"} {
    if err := f.Set(name, MakeInt(int64(i+2))); err != nil {
      t.Fatal(err)
    }
  }
  for name, want := range map[string]string{"a": "1", "b": "2", "c": "3", "d": "4", "e": "5"} {
    if got := mustGet(t, f, name); got != want {
      t.Fatalf("%s = %s, want %s", name, got, want)
    }
  }
}

// TestFrameSetOverwritesInPlace: ein zweites Set auf denselben Namen
// ueberschreibt, es entsteht kein Duplikat. Getrennt fuer inline und Slice,
// weil das zwei verschiedene Codepfade sind.
func TestFrameSetOverwritesInPlace(t *testing.T) {
  f := NewEnv(NewEnv(nil))
  _ = f.Set("inline", MakeInt(1))
  _ = f.Set("slot", MakeInt(2))

  _ = f.Set("inline", MakeInt(10)) // inline-Pfad
  _ = f.Set("slot", MakeInt(20))   // Slice-Pfad

  if got := mustGet(t, f, "inline"); got != "10" {
    t.Fatalf("inline = %s, want 10", got)
  }
  if got := mustGet(t, f, "slot"); got != "20" {
    t.Fatalf("slot = %s, want 20", got)
  }
  // kein Duplikat: Symbols nennt jeden Namen genau einmal
  n := 0
  for _, s := range f.Symbols() {
    if s == "slot" {
      n++
    }
  }
  if n != 1 {
    t.Fatalf("slot erscheint %dx in Symbols, want 1", n)
  }
}

// TestFrameUpdateHitsBothPaths: Update (set!) muss inline UND Slice
// treffen, und bei Nichtfinden zum Parent aufsteigen.
func TestFrameUpdateHitsBothPaths(t *testing.T) {
  root := NewEnv(nil)
  _ = root.Set("global", MakeInt(100))
  f := NewEnv(root)
  _ = f.Set("inline", MakeInt(1))
  _ = f.Set("slot", MakeInt(2))

  if err := f.Update("inline", MakeInt(11)); err != nil {
    t.Fatal(err)
  }
  if err := f.Update("slot", MakeInt(22)); err != nil {
    t.Fatal(err)
  }
  if err := f.Update("global", MakeInt(200)); err != nil {
    t.Fatal(err)
  }
  if got := mustGet(t, f, "inline"); got != "11" {
    t.Fatalf("inline = %s, want 11", got)
  }
  if got := mustGet(t, f, "slot"); got != "22" {
    t.Fatalf("slot = %s, want 22", got)
  }
  if got := mustGet(t, root, "global"); got != "200" {
    t.Fatalf("global = %s, want 200", got)
  }
  if err := f.Update("gibtsnicht", MakeInt(1)); err == nil {
    t.Fatal("Update auf ungebundenen Namen sollte fehlschlagen")
  }
}

// TestFrameShadowsParent: eine Bindung im Frame verdeckt den Parent,
// sowohl inline als auch im Slice-Pfad. Update im Frame darf den Parent
// NICHT anfassen.
func TestFrameShadowsParent(t *testing.T) {
  root := NewEnv(nil)
  _ = root.Set("x", MakeInt(1))
  _ = root.Set("y", MakeInt(2))

  f := NewEnv(root)
  _ = f.Set("x", MakeInt(10)) // inline
  _ = f.Set("z", MakeInt(30)) // Slice
  _ = f.Set("y", MakeInt(20)) // Slice, verdeckt root.y

  if got := mustGet(t, f, "x"); got != "10" {
    t.Fatalf("f.x = %s, want 10", got)
  }
  if got := mustGet(t, f, "y"); got != "20" {
    t.Fatalf("f.y = %s, want 20", got)
  }
  _ = f.Update("y", MakeInt(99))
  if got := mustGet(t, root, "y"); got != "2" {
    t.Fatalf("root.y wurde durchgeschrieben: %s, want 2", got)
  }
}

// TestFrameSymbolsAcrossChain: Symbols sammelt inline, Slice und Parent
// ohne Duplikate. Verdeckte Parent-Namen erscheinen genau einmal.
func TestFrameSymbolsAcrossChain(t *testing.T) {
  root := NewEnv(nil)
  _ = root.Set("shared", MakeInt(1))
  _ = root.Set("onlyroot", MakeInt(2))

  f := NewEnv(root)
  _ = f.Set("inline", MakeInt(3))
  _ = f.Set("slot1", MakeInt(4))
  _ = f.Set("slot2", MakeInt(5))
  _ = f.Set("shared", MakeInt(6)) // verdeckt root

  got := f.Symbols()
  sort.Strings(got)
  want := []string{"inline", "onlyroot", "shared", "slot1", "slot2"}
  if len(got) != len(want) {
    t.Fatalf("Symbols = %v, want %v", got, want)
  }
  for i := range want {
    if got[i] != want[i] {
      t.Fatalf("Symbols = %v, want %v", got, want)
    }
  }
}

// TestFrameSymAPIInteropWithStringAPI: SetSym/GetSym und Set/Get schreiben
// und lesen dasselbe Feld. Weil Symbole interniert sind (internTable in
// types.go), ist Pointer-Gleichheit aequivalent zu Namensgleichheit — beide
// Wege muessen daher beliebig gemischt funktionieren. Ohne diese Aequivalenz
// waere der Pointer-Vergleich im heissen Pfad still falsch.
func TestFrameSymAPIInteropWithStringAPI(t *testing.T) {
  f := NewEnv(NewEnv(nil))
  sym := MakeAtom("mixed")

  // per String schreiben, per Symbol lesen
  if err := f.Set("mixed", MakeInt(1)); err != nil {
    t.Fatal(err)
  }
  v, err := f.GetSym(sym)
  if err != nil {
    t.Fatalf("GetSym nach Set(string): %v", err)
  }
  if v.String() != "1" {
    t.Fatalf("GetSym = %s, want 1", v.String())
  }

  // per Symbol schreiben, per String lesen
  if err := f.SetSym(sym, MakeInt(2)); err != nil {
    t.Fatal(err)
  }
  if got := mustGet(t, f, "mixed"); got != "2" {
    t.Fatalf("Get(string) nach SetSym = %s, want 2", got)
  }

  // dasselbe fuer den Slice-Pfad (zweiter Eintrag)
  sym2 := MakeAtom("second")
  if err := f.SetSym(sym2, MakeInt(3)); err != nil {
    t.Fatal(err)
  }
  if got := mustGet(t, f, "second"); got != "3" {
    t.Fatalf("Slice-Pfad Get(string) = %s, want 3", got)
  }
  v2, err := f.GetSym(sym2)
  if err != nil || v2.String() != "3" {
    t.Fatalf("Slice-Pfad GetSym = %v, %v", v2, err)
  }
}

// TestFrameGetSymFallsThroughToRoot: GetSym muss die Kette hochlaufen und im
// Root-Env (Map, per String indiziert) landen.
func TestFrameGetSymFallsThroughToRoot(t *testing.T) {
  root := NewEnv(nil)
  _ = root.Set("global", MakeInt(7))
  f := NewEnv(root)
  _ = f.SetSym(MakeAtom("local"), MakeInt(8))

  v, err := f.GetSym(MakeAtom("global"))
  if err != nil {
    t.Fatalf("GetSym auf Root-Bindung: %v", err)
  }
  if v.String() != "7" {
    t.Fatalf("GetSym(global) = %s, want 7", v.String())
  }
  if _, err := f.GetSym(MakeAtom("gibtsnicht")); err == nil {
    t.Fatal("GetSym auf ungebundenes Symbol sollte fehlschlagen")
  }
}

// TestFrameManyBindings: ueber die vermutliche Slice-Wachstumsgrenze
// hinaus, damit ein append-Reallocation-Bug auffaellt.
func TestFrameManyBindings(t *testing.T) {
  f := NewEnv(NewEnv(nil))
  const n = 64
  for i := 0; i < n; i++ {
    if err := f.Set(string(rune('A'+i%26))+string(rune('a'+i/26)), MakeNum(float64(i))); err != nil {
      t.Fatal(err)
    }
  }
  for i := 0; i < n; i++ {
    name := string(rune('A'+i%26)) + string(rune('a'+i/26))
    if got := mustGet(t, f, name); got != MakeNum(float64(i)).String() {
      t.Fatalf("%s = %s, want %d", name, got, i)
    }
  }
}
