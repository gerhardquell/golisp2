//**********************************************************************
//  lib/apply_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261001
//**********************************************************************

package lib

import "testing"

func TestApplyFunc(t *testing.T) {
  env := BaseEnv()
  plus, err := env.Get("+")
  if err != nil {
    t.Fatalf("Get +: %v", err)
  }
  res, err := Apply(plus, []*Cell{MakeNum(2), MakeNum(3)})
  if err != nil {
    t.Fatalf("Apply: %v", err)
  }
  if res.Num != 5 {
    t.Fatalf("Apply(+ 2 3) = %s, want 5", res)
  }
}

func TestApplyLambda(t *testing.T) {
  env := BaseEnv()
  fn, err := LoadString("(lambda (x) (* x x))", env)
  if err != nil {
    t.Fatalf("LoadString: %v", err)
  }
  res, err := Apply(fn, []*Cell{MakeNum(7)})
  if err != nil {
    t.Fatalf("Apply: %v", err)
  }
  if res.Num != 49 {
    t.Fatalf("Apply(lambda 7) = %s, want 49", res)
  }
}

func TestApplyNoFunction(t *testing.T) {
  if _, err := Apply(MakeNum(1), nil); err == nil {
    t.Fatal("Apply(1) ohne Fehler, want Fehler")
  }
}

func TestApplyNilFn(t *testing.T) {
  if _, err := Apply(nil, nil); err == nil {
    t.Fatal("Apply(nil) ohne Fehler, want Fehler")
  }
}
