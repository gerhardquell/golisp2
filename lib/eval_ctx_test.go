//**********************************************************************
//  lib/eval_ctx_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-4.8
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260716
//**********************************************************************
// Tests für den Eval-Wrapper und den evalCtx-Kontext.
//**********************************************************************

package lib

import (
  "testing"
)

func TestEvalWrapperPassesContext(t *testing.T) {
  env := BaseEnv()
  expr, err := Read(`(+ 1 2)`)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  res, err := Eval(expr, env)
  if err != nil {
    t.Fatalf("eval: %v", err)
  }
  if res.Type != NUMBER || res.Num != 3 {
    t.Fatalf("expected 3, got %v", res)
  }
}
