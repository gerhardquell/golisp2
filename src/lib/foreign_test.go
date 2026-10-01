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

// TestCellSize belegt die Size-Class-Optimierung aus types.go:49-52: Cell
// darf nicht über 96 Byte wachsen (sonst vergibt der Allocator 112 Byte
// pro Cell). FOREIGN darf dafür kein eigenes Feld bekommen (siehe
// MakeForeign/ForeignObj, die das bestehende Env-Feld mitbenutzen).
func TestCellSize(t *testing.T) {
  if got := unsafe.Sizeof(Cell{}); got != 96 {
    t.Fatalf("unsafe.Sizeof(Cell{}) = %d, want 96", got)
  }
}
