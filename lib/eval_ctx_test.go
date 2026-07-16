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
  "strings"
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

func TestEvalDepthLimit(t *testing.T) {
  old := MaxEvalDepth
  MaxEvalDepth = 50
  defer func() { MaxEvalDepth = old }()

  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("stdlib: %v", err)
  }

  // Nicht-tail-rekursiv: jedes (sum (- n 1)) legt einen neuen Go-Stackframe an.
  code := `(begin
             (defun sum (n)
               (if (= n 0) 0 (+ (sum (- n 1)) 1)))
             (sum 100))`
  expr, err := Read(code)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  _, err = Eval(expr, env)
  if err == nil {
    t.Fatal("expected recursion-depth error, got nil")
  }
  if !strings.Contains(err.Error(), "maximum recursion depth exceeded") {
    t.Fatalf("unexpected error: %v", err)
  }
}
