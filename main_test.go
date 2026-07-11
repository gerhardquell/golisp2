//**********************************************************************
//  main_test.go  - GoLisp -e Expression Tests
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260711
//**********************************************************************

package main

import (
  "bytes"
  "strings"
  "testing"

  "golisp2/lib"
)

func TestRunExpressionSingleFormPrintsResult(t *testing.T) {
  var out bytes.Buffer
  var errOut bytes.Buffer
  env := lib.BaseEnv()
  lib.SetOutputWriter(func(s string) error { _, err := out.WriteString(s); return err })
  defer lib.ResetOutputWriter()

  code := runExpression("(+ 1 2)", env, &out, &errOut)

  if code != 0 {
    t.Fatalf("runExpression exit code = %d, want 0", code)
  }
  if errOut.Len() != 0 {
    t.Fatalf("unexpected stderr: %q", errOut.String())
  }
  if got := out.String(); got != "3\n" {
    t.Fatalf("stdout = %q, want %q", got, "3\n")
  }
}

func TestRunExpressionSingleSideEffectFormPrintsReturnValue(t *testing.T) {
  var out bytes.Buffer
  var errOut bytes.Buffer
  env := lib.BaseEnv()
  lib.SetOutputWriter(func(s string) error { _, err := out.WriteString(s); return err })
  defer lib.ResetOutputWriter()

  code := runExpression(`(println "a")`, env, &out, &errOut)

  if code != 0 {
    t.Fatalf("runExpression exit code = %d, want 0", code)
  }
  if errOut.Len() != 0 {
    t.Fatalf("unexpected stderr: %q", errOut.String())
  }
  if got := out.String(); got != "\"a\"\n\"a\"\n" {
    t.Fatalf("stdout = %q, want %q", got, "\"a\"\n\"a\"\n")
  }
}

func TestRunExpressionMultiFormSuppressesFinalResult(t *testing.T) {
  var out bytes.Buffer
  var errOut bytes.Buffer
  env := lib.BaseEnv()
  lib.SetOutputWriter(func(s string) error { _, err := out.WriteString(s); return err })
  defer lib.ResetOutputWriter()

  code := runExpression("(println \"a\") (println \"b\")", env, &out, &errOut)

  if code != 0 {
    t.Fatalf("runExpression exit code = %d, want 0", code)
  }
  if errOut.Len() != 0 {
    t.Fatalf("unexpected stderr: %q", errOut.String())
  }

  lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
  if len(lines) != 2 || lines[0] != `"a"` || lines[1] != `"b"` {
    t.Fatalf("stdout = %q, want \"\\\"a\\\"\\n\\\"b\\\"\\n\"", out.String())
  }
}

func TestRunExpressionParseError(t *testing.T) {
  var out bytes.Buffer
  var errOut bytes.Buffer
  env := lib.BaseEnv()

  code := runExpression("(+ 1 2", env, &out, &errOut)

  if code != 1 {
    t.Fatalf("runExpression exit code = %d, want 1", code)
  }
  if errOut.Len() == 0 {
    t.Fatalf("expected stderr output, got none")
  }
  if !strings.Contains(errOut.String(), "ERR read:") {
    t.Fatalf("stderr = %q, want it to contain 'ERR read:'", errOut.String())
  }
}

func TestRunExpressionEvalError(t *testing.T) {
  var out bytes.Buffer
  var errOut bytes.Buffer
  env := lib.BaseEnv()

  code := runExpression("(error 'x)", env, &out, &errOut)

  if code != 1 {
    t.Fatalf("runExpression exit code = %d, want 1", code)
  }
  if errOut.Len() == 0 {
    t.Fatalf("expected stderr output, got none")
  }
  if !strings.Contains(errOut.String(), "ERR:") {
    t.Fatalf("stderr = %q, want it to contain 'ERR:'", errOut.String())
  }
}
