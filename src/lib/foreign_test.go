//**********************************************************************
//  lib/foreign_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261001
//**********************************************************************

package lib

import (
  "testing"
  "unsafe"
)

type fooObj struct{ n int }

func TestMakeForeign(t *testing.T) {
  c := MakeForeign("view", &fooObj{n: 3})
  if c.Type != FOREIGN {
    t.Fatalf("Type = %v, want FOREIGN", c.Type)
  }
  if got := c.String(); got != "#<foreign view>" {
    t.Fatalf("String = %q, want %q", got, "#<foreign view>")
  }
  obj, ok := ForeignObj(c, "view")
  if !ok || obj.(*fooObj).n != 3 {
    t.Fatalf("ForeignObj = %v,%v", obj, ok)
  }
  if _, ok := ForeignObj(c, "window"); ok {
    t.Fatal("ForeignObj mit falschem Tag: ok=true, want false")
  }
  if _, ok := ForeignObj(MakeNum(1), "view"); ok {
    t.Fatal("ForeignObj auf Zahl: ok=true, want false")
  }
}

func TestForeignTypeOf(t *testing.T) {
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("LoadStdlib: %v", err)
  }
  if err := env.Set("fx", MakeForeign("view", nil)); err != nil {
    t.Fatalf("Set: %v", err)
  }
  res, err := LoadString("(list (type-of fx) (typep fx 'foreign) (typep fx t))", env)
  if err != nil {
    t.Fatalf("LoadString: %v", err)
  }
  if got := res.String(); got != "(foreign t t)" {
    t.Fatalf("got %s, want (foreign t t)", got)
  }
}

// TestCellSize hält die Size-Class fest: seit dem Ganzzahl-Typ (NumKind +
// Int, Spec 2026-11-16 §7) 112 Byte statt 96. Darüber liegt die nächste
// Klasse (128) — FOREIGN bekommt deshalb weiter kein eigenes Feld (siehe
// MakeForeign/ForeignObj, die das bestehende Env-Feld mitbenutzen).
func TestCellSize(t *testing.T) {
  if got := unsafe.Sizeof(Cell{}); got != 112 {
    t.Fatalf("unsafe.Sizeof(Cell{}) = %d, want 112", got)
  }
}

// pruefeHandle: Go-Handles (pg-connect, maxima-open, http-serve) sind
// FOREIGN — type-of ohne Panic, Druckform #<foreign tag>, length lehnt ab.
func pruefeHandle(t *testing.T, h *Cell, tag string) {
  t.Helper()
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("LoadStdlib: %v", err)
  }
  if err := env.Set("h", h); err != nil {
    t.Fatalf("Set: %v", err)
  }
  if got, want := h.String(), "#<foreign "+tag+">"; got != want {
    t.Errorf("Druckform = %q, want %q", got, want)
  }
  res, err := LoadString("(type-of h)", env)
  if err != nil || res.String() != "foreign" {
    t.Errorf("(type-of h) = %v, %v; want foreign", res, err)
  }
  if res, err := LoadString("(length h)", env); err == nil {
    t.Errorf("(length h) = %v, want Fehler", res)
  }
  if res, err := LoadString("(funcall h)", env); err == nil {
    t.Errorf("(funcall h) = %v, want Fehler", res)
  }
  if res, err := LoadString("(typep h 'function)", env); err != nil || res.Type != NIL {
    t.Errorf("(typep h 'function) = %v, %v; want ()", res, err)
  }
}

// keinHandle ist eine normale Liste, die die Handle-Funktionen ablehnen.
var keinHandle = Cons(MakeInt(1), MakeNil())
