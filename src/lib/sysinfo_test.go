//**********************************************************************
//  lib/sysinfo_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260815
//**********************************************************************

package lib

import (
  "os"
  "testing"
)

func TestArgv(t *testing.T) {
  old := os.Args
  os.Args = []string{"/usr/bin/golisp2", "skript.lisp", "a", "b"}
  defer func() { os.Args = old }()

  env := BaseEnv()
  got, err := LoadString(`(argv)`, env)
  if err != nil { t.Fatalf("argv: %v", err) }
  want := `("/usr/bin/golisp2" "skript.lisp" "a" "b")`
  if got.String() != want {
    t.Fatalf("argv = %s, erwartet %s", got, want)
  }

  // Zugriff per car/cdr aus Lisp heraus (kein cadr in GoLisp)
  got, err = LoadString(`(car (cdr (argv)))`, env)
  if err != nil { t.Fatalf("cadr argv: %v", err) }
  if got.Type != STRING || got.Val != "skript.lisp" {
    t.Fatalf("car (cdr (argv)) = %q, erwartet %q", got.Val, "skript.lisp")
  }
}

func TestGetenv(t *testing.T) {
  t.Setenv("GOLISP_TEST_VAR", "wert123")
  t.Setenv("GOLISP_TEST_LEER", "")

  env := BaseEnv()

  got, err := LoadString(`(getenv "GOLISP_TEST_VAR")`, env)
  if err != nil { t.Fatalf("getenv: %v", err) }
  if got.Type != STRING || got.Val != "wert123" {
    t.Fatalf("getenv = %q, erwartet %q", got.Val, "wert123")
  }

  // gesetzt, aber leer → leerer String (nicht NIL)
  got, err = LoadString(`(getenv "GOLISP_TEST_LEER")`, env)
  if err != nil { t.Fatalf("getenv leer: %v", err) }
  if got.Type != STRING || got.Val != "" {
    t.Fatalf("getenv leer = %v, erwartet leerer String", got)
  }

  // nicht gesetzt → ()
  got, err = LoadString(`(getenv "GOLISP_TEST_GIBTS_NICHT")`, env)
  if err != nil { t.Fatalf("getenv unset: %v", err) }
  if got.Type != NIL {
    t.Fatalf("getenv unset = %s, erwartet ()", got)
  }
}

func TestEnviron(t *testing.T) {
  t.Setenv("GOLISP_TEST_ENVIRON", "sichtbar")

  env := BaseEnv()
  got, err := LoadString(`(assoc "GOLISP_TEST_ENVIRON" (environ))`, env)
  if err != nil {
    // assoc evtl. nicht vorhanden → Fallback: cdr-Findung per Lisp
    got, err = LoadString(`(environ)`, env)
    if err != nil { t.Fatalf("environ: %v", err) }
  }
  if got.Type == NIL {
    t.Fatalf("environ: GOLISP_TEST_ENVIRON nicht gefunden")
  }
}
