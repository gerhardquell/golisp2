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

func TestExecMultipleParams(t *testing.T) {
  env := BaseEnv()
  form := List(
    MakeAtom("exec"),
    MakeStr("echo"),
    MakeAtom("param:"),
    MakeStr("one"),
    MakeAtom("param:"),
    MakeStr("two"),
    MakeAtom("stdout:"),
    MakeAtom("out"),
  )
  _, err := Eval(form, env)
  if err != nil {
    t.Fatalf("exec failed: %v", err)
  }
  out, _ := env.Get("out")
  if out.Type != STRING || out.Val != "one two\n" {
    t.Fatalf("expected 'one two\\n', got %v", out)
  }
}

func TestExecStderr(t *testing.T) {
  env := BaseEnv()
  form := List(
    MakeAtom("exec"),
    MakeStr("sh"),
    MakeAtom("param:"),
    MakeStr("-c"),
    MakeAtom("param:"),
    MakeStr("echo err >&2; exit 1"),
    MakeAtom("stdout:"),
    MakeAtom("out"),
    MakeAtom("stderr:"),
    MakeAtom("err"),
    MakeAtom("exitcd:"),
    MakeAtom("cd"),
  )
  result, err := Eval(form, env)
  if err != nil {
    t.Fatalf("exec failed: %v", err)
  }
  if result == nil || result.Type != ATOM || result.Val != "t" {
    t.Fatalf("expected t (non-zero exit is not a failure), got %v", result)
  }
  errCell, _ := env.Get("err")
  if errCell.Type != STRING || errCell.Val != "err\n" {
    t.Fatalf("expected stderr 'err\\n', got %v", errCell)
  }
  cd, _ := env.Get("cd")
  if cd.Type != NUMBER || cd.Num != 1 {
    t.Fatalf("expected exit code 1, got %v", cd)
  }
}

func TestExecStdin(t *testing.T) {
  env := BaseEnv()
  form := List(
    MakeAtom("exec"),
    MakeStr("cat"),
    MakeAtom("stdin:"),
    MakeStr("hello world"),
    MakeAtom("stdout:"),
    MakeAtom("out"),
    MakeAtom("exitcd:"),
    MakeAtom("cd"),
  )
  _, err := Eval(form, env)
  if err != nil {
    t.Fatalf("exec failed: %v", err)
  }
  out, _ := env.Get("out")
  if out.Type != STRING || out.Val != "hello world" {
    t.Fatalf("expected 'hello world', got %v", out)
  }
  cd, _ := env.Get("cd")
  if cd.Type != NUMBER || cd.Num != 0 {
    t.Fatalf("expected exit code 0, got %v", cd)
  }
}

func TestExecUnknownProgram(t *testing.T) {
  env := BaseEnv()
  form := List(
    MakeAtom("exec"),
    MakeStr("/no/such/program"),
    MakeAtom("stdout:"),
    MakeAtom("out"),
    MakeAtom("exitcd:"),
    MakeAtom("cd"),
  )
  result, err := Eval(form, env)
  if err != nil {
    t.Fatalf("exec should not error on missing program: %v", err)
  }
  if result != nil && result.Type != NIL {
    t.Fatalf("expected nil on missing program, got %v", result)
  }
  cd, _ := env.Get("cd")
  if cd.Type != NUMBER || cd.Num != -1 {
    t.Fatalf("expected exit code -1, got %v", cd)
  }
}
