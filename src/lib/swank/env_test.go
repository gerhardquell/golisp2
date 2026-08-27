//**********************************************************************
//  lib/swank/env_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260618
//**********************************************************************
// Tests für per-connection SWANK primitives.
//**********************************************************************

package swank

import (
  "strings"
  "testing"

  "golisp2/src/lib"
)

func TestRegisterSwankEnv(t *testing.T) {
  env := lib.BaseEnv()
  var sent *lib.Cell
  send := func(c *lib.Cell) error {
    sent = c
    return nil
  }
  RegisterSwankEnv(env, send)

  // (swank-send-event '(:write-string "hi" :repl-result))
  cell, err := lib.Read("(swank-send-event '(:write-string \"hi\" :repl-result))")
  if err != nil {
    t.Fatalf("read failed: %v", err)
  }
  _, err = lib.Eval(cell, env)
  if err != nil {
    t.Fatalf("eval failed: %v", err)
  }
  if sent == nil {
    t.Fatal("send callback was not invoked")
  }
  if sent.String() != "(:write-string \"hi\" :repl-result)" {
    t.Fatalf("unexpected event: %s", sent.String())
  }
}

func TestSwankPrintReturnValue(t *testing.T) {
  env := lib.BaseEnv()
  var sent *lib.Cell
  send := func(c *lib.Cell) error {
    sent = c
    return nil
  }
  RegisterSwankEnv(env, send)

  // (swank-print "hello")
  cell, err := lib.Read("(swank-print \"hello\")")
  if err != nil {
    t.Fatalf("read failed: %v", err)
  }
  result, err := lib.Eval(cell, env)
  if err != nil {
    t.Fatalf("eval failed: %v", err)
  }
  // Event darf kein :repl-result tragen
  if sent == nil {
    t.Fatal("send callback was not invoked")
  }
  if sent.String() != `(:write-string "\"hello\"")` {
    t.Fatalf("unexpected event: %s", sent.String())
  }
  // Rückgabewert muss das letzte Argument sein
  if result == nil || result.String() != "\"hello\"" {
    t.Fatalf("expected return value \"hello\", got: %v", result)
  }
}

func TestSwankOutputOnlyForm(t *testing.T) {
  env := lib.BaseEnv()
  if err := lib.LoadStdlib(env); err != nil {
    t.Fatalf("LoadStdlib failed: %v", err)
  }
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp failed: %v", err)
  }

  cases := []struct {
    expr     string
    expected bool
  }{
    {"(ga-print g 1)", true},
    {"(format t \"x\")", true},
    {"(format nil \"x\")", false},
    {"(+ 1 2)", false},
    {"(print 1)", true},
    {"(println 1)", true},
  }

  for _, tc := range cases {
    call := "(swank--output-only-form? (quote " + tc.expr + "))"
    cell, err := lib.Read(call)
    if err != nil {
      t.Fatalf("read %q failed: %v", call, err)
    }
    result, err := lib.Eval(cell, env)
    if err != nil {
      t.Fatalf("eval %q failed: %v", call, err)
    }
    got := result.Type != lib.NIL
    if got != tc.expected {
      t.Errorf("%q: expected %v, got %v", tc.expr, tc.expected, got)
    }
  }
}

func TestSwankFormatRedirect(t *testing.T) {
  env := lib.BaseEnv()
  if err := lib.LoadStdlib(env); err != nil {
    t.Fatalf("LoadStdlib failed: %v", err)
  }
  var events []string
  send := func(c *lib.Cell) error {
    events = append(events, c.String())
    return nil
  }
  RegisterSwankEnv(env, send)

  cell, err := lib.Read("(format t \"hello ~A\" 42)")
  if err != nil {
    t.Fatalf("read failed: %v", err)
  }
  result, err := lib.Eval(cell, env)
  if err != nil {
    t.Fatalf("eval failed: %v", err)
  }
  if result == nil || result.Type != lib.NIL {
    t.Fatalf("expected nil return, got %v", result)
  }
  if len(events) != 1 {
    t.Fatalf("expected 1 event, got %d: %v", len(events), events)
  }
  if events[0] != `(:write-string "hello 42")` {
    t.Fatalf("unexpected event: %s", events[0])
  }
}

func TestSwankGaPrintRedirect(t *testing.T) {
  env := lib.BaseEnv()
  if err := lib.LoadStdlib(env); err != nil {
    t.Fatalf("LoadStdlib failed: %v", err)
  }
  var events []string
  send := func(c *lib.Cell) error {
    events = append(events, c.String())
    return nil
  }
  RegisterSwankEnv(env, send)

  setup, err := lib.Read("(define g (ga-create (quote bit8) 3 2 (lambda (x) 1.0)))")
  if err != nil {
    t.Fatalf("read setup failed: %v", err)
  }
  _, err = lib.Eval(setup, env)
  if err != nil {
    t.Fatalf("eval setup failed: %v", err)
  }

  cell, err := lib.Read("(ga-print g 1)")
  if err != nil {
    t.Fatalf("read failed: %v", err)
  }
  result, err := lib.Eval(cell, env)
  if err != nil {
    t.Fatalf("eval failed: %v", err)
  }
  if result == nil || result.Type != lib.ATOM || result.Val != "t" {
    t.Fatalf("expected t return, got %v", result)
  }
  if len(events) != 1 {
    t.Fatalf("expected 1 event, got %d: %v", len(events), events)
  }
  if !strings.Contains(events[0], ":write-string") {
    t.Fatalf("expected :write-string event, got %s", events[0])
  }
  if !strings.Contains(events[0], "idx | score | values") {
    t.Fatalf("expected ga-print header in event, got %s", events[0])
  }
}

func TestSwankFindDefinition(t *testing.T) {
  env := lib.BaseEnv()
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  lib.RegisterDefinition("found", "/x.lisp", 9)

  // Teste via env.Get() + direktem Fn-Aufruf (kein callPrimitive-Helper vorhanden)
  cell, err := env.Get("swank--find-definition")
  if err != nil {
    t.Fatalf("env.Get failed: %v", err)
  }
  if cell.Type != lib.FUNC {
    t.Fatalf("expected FUNC, got %v", cell.Type)
  }

  result, err := cell.Fn([]*lib.Cell{lib.MakeStr("found")})
  if err != nil {
    t.Fatalf("call failed: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, "/x.lisp") || !strings.Contains(s, "9") {
    t.Fatalf("expected (/x.lisp . 9), got %s", s)
  }

  lib.ClearDefinitions()
  result2, err := cell.Fn([]*lib.Cell{lib.MakeStr("missing")})
  if err != nil {
    t.Fatalf("call failed: %v", err)
  }
  if result2.Type != lib.NIL {
    t.Fatalf("expected NIL für missing, got %v", result2)
  }
}
