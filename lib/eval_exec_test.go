//**********************************************************************
//  lib/eval_exec_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260711
//**********************************************************************
// Tests fuer die exec-Spezialform
//**********************************************************************

package lib

import (
  "testing"
)

func TestExecBasicStdout(t *testing.T) {
  env := BaseEnv()
  // (exec "echo" param: "hello" stdout: out exitcd: cd)
  form := List(
    MakeAtom("exec"),
    MakeStr("echo"),
    MakeAtom("param:"),
    MakeStr("hello"),
    MakeAtom("stdout:"),
    MakeAtom("out"),
    MakeAtom("exitcd:"),
    MakeAtom("cd"),
  )
  result, err := Eval(form, env)
  if err != nil {
    t.Fatalf("exec failed: %v", err)
  }
  if result == nil || result.Type != ATOM || result.Val != "t" {
    t.Fatalf("expected t, got %v", result)
  }

  out, err := env.Get("out")
  if err != nil {
    t.Fatalf("out not set: %v", err)
  }
  if out.Type != STRING || out.Val != "hello\n" {
    t.Fatalf("expected 'hello\\n', got %v", out)
  }

  cd, _ := env.Get("cd")
  if cd == nil || cd.Type != NUMBER || cd.Num != 0 {
    t.Fatalf("expected exit code 0, got %v", cd)
  }
}
