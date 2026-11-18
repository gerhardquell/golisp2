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

// TestMakroLambdaListeVerschachtelt: Makro-Lambda-Listen zerlegen
// verschachtelt (Spec §3.2), lambda/defun bleiben flach (Spec §5).
func TestMakroLambdaListeVerschachtelt(t *testing.T) {
  vergissSpecial(t, "*dbs-x*")
  mitPaar := "(defmacro mit-paar ((x y) &body body) `(let ((,x 1) (,y 2)) ,@body)) "
  mp := "(defmacro mp ((a b)) `(list ,a ,b)) "
  pruefeLambdaListFaelle(t, []lambdaListFall{
    {src: mitPaar + `(mit-paar (p q) (+ p q))`, want: "3"},
    {src: mitPaar + `(macroexpand '(mit-paar (p q) (+ p q)))`, want: "(let ((p 1) (q 2)) (+ p q))"},
    {src: "(defmacro mdp ((a . b) c) `(list ',a ',b ,c)) (mdp (1 2 3) 4)", want: "(1 (2 3) 4)"},
    {src: "(defmacro mo ((a &optional (b 9)) c) `(list ,a ,b ,c)) (list (mo (1) 3) (mo (1 2) 3))", want: "((1 9 3) (1 2 3))"},
    {src: "(defmacro tief (((a) b) . r) `(list ,a ,b ',r)) (tief ((1) 2) 3)", want: "(1 2 (3))"},
    {src: "(defmacro mk ((a &key (b 2)) c) `(list ,a ,b ,c)) (mk (1 :b 5) 3)", want: "(1 5 3)"},
    {src: "(macrolet ((mp ((a b)) `(+ ,a ,b))) (mp (3 4)))", want: "7"},
    {src: mp + `(mp ())`, fehler: "lambda: zu wenig Argumente (brauche 'a')"},

    // Fehler beim Aufruf: Präfix lambda: (Spec §3.3)
    {src: mp + `(mp 5)`, fehler: "lambda: Liste erwartet für (a b), bekam 5"},
    {src: mp + `(mp (1 . 2))`, fehler: "lambda: Liste erwartet für (a b), bekam (1 . 2)"},
    {src: mp + `(mp (1))`, fehler: "lambda: zu wenig Argumente (brauche 'b')"},
    {src: mp + `(mp (1 2 3))`, fehler: "lambda: zu viele Argumente (1 überzählig)"},
    {src: "(defmacro mp2 (x (a b)) x) (mp2 1)", fehler: "lambda: zu wenig Argumente (brauche '(a b)')"},

    // Fehler bei der Definition
    {src: `(defmacro mx (&optional ((a b))) a)`, fehler: "defmacro: verschachteltes Muster nur an Pflichtstellen"},
    {src: `(defmacro mx (&key ((a b) 1)) a)`, fehler: "defmacro: verschachteltes Muster nur an Pflichtstellen"},
    {src: `(defmacro mx (&rest (a b)) a)`, fehler: "defmacro: verschachteltes Muster nur an Pflichtstellen"},
    {src: `(macrolet ((mx (&body (a)) a)) 1)`, fehler: "macrolet: verschachteltes Muster nur an Pflichtstellen"},
    {src: `(defvar *dbs-x* 1) (defmacro ms ((a *dbs-x*)) a)`, fehler: "defmacro: Parameter *dbs-x* ist special"},
    {src: `(lambda ((a b)) a)`, fehler: "lambda: Parameter muss Atom sein"},
    {src: `(defun fx (x (a b)) a)`, fehler: "defun: Parameter muss Atom sein"},
    {src: `(lambda (&rest (a b)) a)`, fehler: "lambda: Parameter muss Atom sein"},
    {src: `(lambda (&optional ((a b))) a)`, fehler: "lambda: Parameter muss Atom sein"},
  })
}

// TestBindArgsPraefix: bindArgs mit Schalter und Präfix, wie
// destructuring-bind es braucht (Spec §3.3, Schritt 2 des Plans).
func TestBindArgsPraefix(t *testing.T) {
  faelle := []struct{ muster, werte, fehler string }{
    {"(a b)", "(1)", "destructuring-bind: zu wenig Werte (brauche 'b')"},
    {"(a)", "(1 2 3)", "destructuring-bind: zu viele Werte (2 überzählig)"},
    {"(x (a b))", "(1 5)", "destructuring-bind: Liste erwartet für (a b), bekam 5"},
    {"(x (a b))", "(1 (2))", "destructuring-bind: zu wenig Werte (brauche 'b')"},
  }
  for _, f := range faelle {
    muster, _ := Read(f.muster)
    werte, _ := Read(f.werte)
    err := bindArgs(muster, CellToSlice(werte), NewEnv(BaseEnv()), evalCtx{}, true, "destructuring-bind")
    if err == nil || err.Error() != f.fehler {
      t.Errorf("%s gegen %s: Fehler %v, want %q", f.muster, f.werte, err, f.fehler)
    }
  }
  muster, _ := Read("((a b) c &key (d 4))")
  werte, _ := Read("((1 2) 3 :d 5)")
  env := NewEnv(BaseEnv())
  if err := bindArgs(muster, CellToSlice(werte), env, evalCtx{}, true, "destructuring-bind"); err != nil {
    t.Fatalf("Bindung: %v", err)
  }
  var got []string
  for _, n := range []string{"a", "b", "c", "d"} {
    v, _ := env.Get(n)
    got = append(got, v.String())
  }
  if s := strings.Join(got, " "); s != "1 2 3 5" {
    t.Errorf("Bindung a b c d = %s, want 1 2 3 5", s)
  }
}

// TestDestructuringBindSpezialform: Spec §3.1–§3.3, §5 (Schritt 2 des Plans).
func TestDestructuringBindSpezialform(t *testing.T) {
  vergissSpecial(t, "*dbs-y*")
  syntax := "destructuring-bind: Syntax: (destructuring-bind lambda-liste ausdruck rumpf…)"
  pruefeLambdaListFaelle(t, []lambdaListFall{
    // Spec §3.2
    {src: `(destructuring-bind ((a b) c &key (d 4)) '((1 2) 3 :d 5) (list a b c d))`, want: "(1 2 3 5)"},
    {src: `(destructuring-bind (a (b &optional (c 9)) . r) '(1 (2) 3 4) (list a b c r))`, want: "(1 2 9 (3 4))"},
    // Punktpaar auf jeder Ebene
    {src: `(destructuring-bind (a . b) '(1 2) b)`, want: "(2)"},
    {src: `(destructuring-bind (a (b . c) . d) '(1 (2 3 4) 5 6) (list a b c d))`, want: "(1 2 (3 4) (5 6))"},
    {src: `(destructuring-bind ((a . b) . c) '((1) 2) (list a b c))`, want: "(1 () (2))"},
    // Lambda-Keywords
    {src: `(destructuring-bind (a &key (k 7)) '(1) (list a k))`, want: "(1 7)"},
    {src: `(destructuring-bind (a &key (k 7 k-p)) '(1 :k 2) (list a k k-p))`, want: "(1 2 t)"},
    {src: `(destructuring-bind (a b &rest c) '(1 2 3 4) c)`, want: "(3 4)"},
    {src: `(destructuring-bind (a &optional b) '(1) b)`, want: "()"},
    {src: `(destructuring-bind (a &optional (b (+ a 1))) '(1) b)`, want: "2"},
    // Form: ohne Rumpf (), leeres Muster, lexikalisch und nicht nach außen
    {src: `(destructuring-bind (a) '(1))`, want: "()"},
    {src: `(destructuring-bind () '() 5)`, want: "5"},
    {src: `(define a 10) (list (destructuring-bind (a) '(1) a) a)`, want: "(1 10)"},
    {src: `(define z 3) (destructuring-bind (a) (list z) (+ a z))`, want: "6"},
    // Ausdruck genau einmal ausgewertet
    {src: `(define n 0) (destructuring-bind ((a) b) (begin (setq n (+ n 1)) '((1) 2)) (list a b n))`, want: "(1 2 1)"},
    // Spec §5: keine let-Kette mehr
    {src: `(define f '(destructuring-bind (a b) '(1 2) (+ a b))) (equal? (macroexpand f) f)`, want: "t"},

    // Spec §3.3
    {src: `(destructuring-bind (a b) '(1) a)`, fehler: "destructuring-bind: zu wenig Werte (brauche 'b')"},
    {src: `(destructuring-bind (a) '(1 2 3) a)`, fehler: "destructuring-bind: zu viele Werte (2 überzählig)"},
    {src: `(destructuring-bind (x (a b)) '(1 5) a)`, fehler: "destructuring-bind: Liste erwartet für (a b), bekam 5"},
    {src: `(destructuring-bind (x (a b)) '(1 (2)) a)`, fehler: "destructuring-bind: zu wenig Werte (brauche 'b')"},
    {src: `(destructuring-bind (a b) 5 a)`, fehler: "destructuring-bind: Liste erwartet für (a b), bekam 5"},
    {src: `(destructuring-bind (a b) '(1 . 2) a)`, fehler: "destructuring-bind: Liste erwartet für (a b), bekam (1 . 2)"},
    {src: `(destructuring-bind (a &optional ((b c))) '(1) a)`, fehler: "destructuring-bind: verschachteltes Muster nur an Pflichtstellen"},
    {src: `(destructuring-bind (a &key ((b c) 1)) '(1) a)`, fehler: "destructuring-bind: verschachteltes Muster nur an Pflichtstellen"},
    {src: `(destructuring-bind (a &rest (b c)) '(1) a)`, fehler: "destructuring-bind: verschachteltes Muster nur an Pflichtstellen"},
    {src: `(defvar *dbs-y* 1) (destructuring-bind (a *dbs-y*) '(1 2) a)`, fehler: "destructuring-bind: Parameter *dbs-y* ist special — mit let binden"},
    {src: `(defvar *dbs-y* 1) (destructuring-bind (a (b . *dbs-y*)) '(1 (2)) a)`, fehler: "destructuring-bind: Parameter *dbs-y* ist special"},
    {src: `(destructuring-bind)`, fehler: syntax},
    {src: `(destructuring-bind (a))`, fehler: syntax},
  })
}

// TestDestructuringBindTCO: Spec §6 — Endrekursion durch den Rumpf ist O(1)
// Stack: 1 000 000 Tail-Aufrufe. Bewusst mit kurzen Listen statt (iota 1000000):
// bindArgs kopiert Werte und Rest je Schritt (properList, SliceToCell), das
// Spec-Beispiel wäre O(n²) — 10 000 Elemente ≈ 3 s, 1 000 000 endet nicht.
func TestDestructuringBindTCO(t *testing.T) {
  evalStdlibEq(t, `(defun lauf (x n) (destructuring-bind (a . r) x (if (> a 0) (lauf (list (- a 1)) (+ n 1)) n)))
    (lauf (list 1000000) 0)`, "1000000")
  evalStdlibEq(t, `(defun lauf2 (x n) (destructuring-bind (a . r) x (if r (lauf2 r (+ n 1)) n)))
    (lauf2 (iota 1000) 0)`, "999")
}
