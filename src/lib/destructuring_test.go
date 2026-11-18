//**********************************************************************
//  lib/destructuring_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261118
//**********************************************************************
// Lambda-Listen und destructuring-bind (Spec 2026-11-18).
// Charakterisierung: was beim Umbau von bindArgs gleich bleiben muss.
//**********************************************************************

package lib

import (
  "strings"
  "testing"
)

// lambdaListFall: want ist das erwartete Ergebnis, oder bei fehler != ""
// ein Teil der Fehlermeldung.
type lambdaListFall struct{ src, want, fehler string }

func pruefeLambdaListFaelle(t *testing.T, faelle []lambdaListFall) {
  t.Helper()
  for _, f := range faelle {
    got, err := evalStdlib(t, f.src)
    if f.fehler != "" {
      if err == nil {
        t.Errorf("%s: Fehler mit %q erwartet, bekam %s", f.src, f.fehler, got)
      } else if !strings.Contains(err.Error(), f.fehler) {
        t.Errorf("%s: Fehler %q enthält nicht %q", f.src, err, f.fehler)
      }
      continue
    }
    if err != nil {
      t.Errorf("%s: Fehler %v", f.src, err)
      continue
    }
    if got.String() != f.want {
      t.Errorf("%s = %s, want %s", f.src, got, f.want)
    }
  }
}

// TestLambdaListCharakterisierung hält das IST vor dem Umbau fest.
func TestLambdaListCharakterisierung(t *testing.T) {
  pruefeLambdaListFaelle(t, []lambdaListFall{
    // destructuring-bind, flach, passende Anzahl
    {src: `(destructuring-bind (a b c) '(1 2 3) (list c b a))`, want: "(3 2 1)"},
    {src: `(destructuring-bind (a) '(1) a)`, want: "1"},
    {src: `(destructuring-bind (a b) '(1 2) (define s (+ a b)) (* s 10))`, want: "30"},
    {src: `(define n 0) (destructuring-bind (a b) (begin (setq n (+ n 1)) '(1 2)) (list a b n))`, want: "(1 2 1)"},
    {src: `(destructuring-bind (a b) (list (+ 1 1) 'x) (list a b))`, want: "(2 x)"},

    // lambda: regulär, &optional, &rest/&body, &key, Punktpaar
    {src: `((lambda (a b) (list a b)) 1 2)`, want: "(1 2)"},
    {src: `((lambda (a &optional b) (list a b)) 1)`, want: "(1 ())"},
    {src: `((lambda (a &optional (b 7)) (list a b)) 1)`, want: "(1 7)"},
    {src: `((lambda (a &optional (b 7)) (list a b)) 1 2)`, want: "(1 2)"},
    {src: `((lambda (a &optional (b (+ a 1) b-p)) (list a b b-p)) 1)`, want: "(1 2 ())"},
    {src: `((lambda (a &optional (b 0 b-p)) (list a b b-p)) 1 5)`, want: "(1 5 t)"},
    {src: `((lambda (a &rest r) (list a r)) 1 2 3)`, want: "(1 (2 3))"},
    {src: `((lambda (a &rest r) (list a r)) 1)`, want: "(1 ())"},
    {src: `((lambda (a &body r) r) 1 2 3)`, want: "(2 3)"},
    {src: `((lambda (a . r) (list a r)) 1 2 3)`, want: "(1 (2 3))"},
    {src: `((lambda (a b . r) (list a b r)) 1 2)`, want: "(1 2 ())"},
    {src: `((lambda (&key x (y 5) (z 0 z-p)) (list x y z z-p)) :x 1)`, want: "(1 5 0 ())"},
    {src: `((lambda (&key x (y 5)) (list x y)) :y 2 :x 1)`, want: "(1 2)"},
    {src: `((lambda (&key x) x) :x 1 :x 2)`, want: "1"},
    {src: `((lambda (a &rest r &key x) (list a r x)) 1 :x 3)`, want: "(1 (:x 3) 3)"},
    {src: `((lambda (&key x &allow-other-keys) x) :y 2 :x 1)`, want: "1"},
    {src: `(defun f (a &optional (b a)) (list a b)) (f 4)`, want: "(4 4)"},

    // lambda: Fehler (Präfix lambda:)
    {src: `((lambda (a b) a) 1)`, fehler: "lambda: zu wenig Argumente (brauche 'b')"},
    {src: `((lambda (a) a) 1 2)`, fehler: "lambda: zu viele Argumente"},
    {src: `(apply (lambda (a) a) '(1 2 3))`, fehler: "lambda: zu viele Argumente (2 überzählig)"},
    {src: `((lambda (&key x) x) :x)`, fehler: "lambda: Schlüsselwort-Argumente unvollständig"},
    {src: `((lambda (&key x) x) 1 2)`, fehler: "lambda: Schlüsselwort erwartet"},
    {src: `((lambda (&key x) x) :y 1)`, fehler: "lambda: unbekanntes Schlüsselwort :y"},

    // defmacro / macrolet: flache Makro-Lambda-Listen
    {src: "(defmacro mein-when (c &body body) `(if ,c (begin ,@body) ()))" + ` (mein-when t 1 2)`, want: "2"},
    {src: "(defmacro mit-opt (a &optional (b 10)) `(+ ,a ,b))" + ` (list (mit-opt 1) (mit-opt 1 2))`, want: "(11 3)"},
    {src: "(defmacro mit-key (&key (x 1)) `(* ,x 2))" + ` (mit-key :x 4)`, want: "8"},
    {src: "(macrolet ((zweimal (x) `(+ ,x ,x))) (zweimal 4))", want: "8"},
    {src: "(defmacro m2 (a b) `(list ,a ,b))" + ` (m2 1)`, fehler: "lambda: zu wenig Argumente"},
  })
}
