//**********************************************************************
//  lib/genalg_prims_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6, kimi
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260626
//**********************************************************************
// Tests fuer die GA-Lisp-Primitive.
//**********************************************************************

package lib

import (
  "strings"
  "testing"
)

func TestGaCreateTypes(t *testing.T) {
  cases := []string{
    "(ga? (ga-create 'bit1 5 4 (lambda (g) 0)))",
    "(ga? (ga-create 'bit2 5 4 (lambda (g) 0)))",
    "(ga? (ga-create 'bit4 5 4 (lambda (g) 0)))",
    "(ga? (ga-create 'bit8 5 4 (lambda (g) 0)))",
    "(ga? (ga-create 'biti 5 4 (lambda (g) 0)))",
    "(ga? (ga-create 'bitf 5 4 (lambda (g) 0)))",
    "(ga? (ga-create 0 5 4 (lambda (g) 0)))",
    "(ga? (ga-create \"bit1\" 5 4 (lambda (g) 0)))",
  }
  for _, src := range cases {
    evalEq(t, src, "t")
  }
}

func TestGaCreateErrors(t *testing.T) {
  cases := []string{
    "(ga-create)",
    "(ga-create 'bit1 5 4)",
    "(ga-create 'unknown 5 4 (lambda (g) 0))",
    "(ga-create 'bit1 'five 4 (lambda (g) 0))",
    "(ga-create 'bit1 5 4 42)",
  }
  for _, src := range cases {
    evalErr(t, src)
  }
}

func TestGaLifecycle(t *testing.T) {
  src := `
    (define ga (ga-create 'bit1 5 4 (lambda (g) (apply + g))))
    (ga-init ga)
    (ga-calc ga)
    (ga-result ga)
  `
  got, err := evalStr(src)
  if err != nil {
    t.Fatalf("GA lifecycle Fehler: %v", err)
  }
  s := got.String()
  if !strings.HasPrefix(s, "(") || !strings.HasSuffix(s, ")") {
    t.Errorf("ga-result sollte Liste sein, got %s", s)
  }
}

func TestGaCrossSelectMut(t *testing.T) {
  src := `
    (defun len (lst) (if (null? lst) 0 (+ 1 (len (cdr lst)))))
    (define ga (ga-create 'bit1 4 4 (lambda (g) (apply + g))))
    (ga-init ga)
    (ga-calc ga)
    (ga-cross ga 2)
    (ga-select ga 2)
    (ga-mut ga 0.1)
    (len (ga-result ga))
  `
  evalEq(t, src, "2")
}

func TestGaRace(t *testing.T) {
  src := `
    (defun len (lst) (if (null? lst) 0 (+ 1 (len (cdr lst)))))
    (define ga (ga-create 'bit8 10 16 (lambda (g) (apply + g))))
    (ga-init ga)
    (ga-calc ga)
    (len (ga-result ga))
  `
  evalEq(t, src, "16")
}
