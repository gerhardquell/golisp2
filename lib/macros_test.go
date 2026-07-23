//**********************************************************************
//  lib/macros_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260616
//**********************************************************************
// Charakterisierungstests für Makro-Expansion (eval.go: defmacro,
// macroexpand, Eval-MACRO-Pfad). Makros sind Common-Lisp-Stil: NICHT
// hygienisch, gensym muss manuell für frische Symbole sorgen.
//**********************************************************************

package lib

import "testing"

// TestMacroBasic: defmacro + Aufruf expandiert und evaluiert.
func TestMacroBasic(t *testing.T) {
  src := `
    (defmacro double (x) (list '* x 2))
    (double 5)
  `
  evalEq(t, src, "10")
}

// TestMacroNoArgEval: Makro bekommt Argumente UN-EVALUIERT.
// Das ist der Kernunterschied zur Funktion: (quote-it (+ 1 2)) liefert
// die Form (+ 1 2), nicht 3.
func TestMacroNoArgEval(t *testing.T) {
  src := `
    (defmacro quote-it (x) (list 'quote x))
    (quote-it (+ 1 2))
  `
  evalEq(t, src, "(+ 1 2)")
}

// TestMacroExpand: macroexpand expandiert einmal, gibt Ergebnis zurück.
func TestMacroExpand(t *testing.T) {
  src := `
    (defmacro square (x) (list '* x x))
    (macroexpand '(square 5))
  `
  evalEq(t, src, "(* 5 5)")
}

// TestMacroExpandNotMacro: macroexpand auf Nicht-Makro → Form unverändert.
func TestMacroExpandNotMacro(t *testing.T) {
  evalEq(t, `(macroexpand '(+ 1 2))`, "(+ 1 2)")
  evalEq(t, `(macroexpand 'x)`, "x")
  evalEq(t, `(macroexpand 42)`, "42")
}

// TestMacroExpandError: macroexpand braucht 1 Argument.
func TestMacroExpandError(t *testing.T) {
  evalErr(t, `(macroexpand)`)
}

// TestMacroNested: Makro expandiert zu anderem Makro → wird weiter
// expandiert (Eval-Loop expandiert nach MACRO-Erkennung und continue't).
func TestMacroNested(t *testing.T) {
  src := `
    (defmacro inc1 (x) (list '+ x 1))
    (defmacro inc2 (x) (list 'inc1 (list 'inc1 x)))
    (inc2 10)
  `
  evalEq(t, src, "12")  // inc2 → inc1(inc1(10)) → 11+1 ... 10+1+1 = 12
}

// TestMacroSetqUpdatesInLet dokumentiert die CL-Semantik von setq (seit
// 20260723): setq sucht die Bindung entlang der Env-Kette und UPDATET sie
// (wie set!), statt im inneren Scope zu shadowen. Ein swap-Makro mit
// setq im let-Body funktioniert daher.
//
// Vorher galt: setq = define = env.Set → Shadow im inneren Env, swap
// wirkungslos. Das war die alte (Scheme-artige) Semantik und wurde mit
// dem CL-Konformitäts-Schritt bewusst geändert.
func TestMacroSetqShadowsInLet(t *testing.T) {
  src := `
    (defmacro swap-setq (a b)
      (let ((tmp (gensym)))
        (list 'let (list (list tmp a))
              (list 'setq a b)
              (list 'setq b tmp))))
    (let ((tmp 1) (other 2))
      (swap-setq tmp other)
      (list tmp other))
  `
  evalEq(t, src, "(2 1)")  // CL: setq updatet die äußere Bindung → swap wirkt
}

// TestMacroSetBangUpdatesOuter: gleiche swap-Logik, aber mit set!
// (env.Update) statt setq. set! sucht die Variable in der Env-Kette und
// updatet sie – der swap funktioniert.
func TestMacroSetBangUpdatesOuter(t *testing.T) {
  src := `
    (defmacro swap-set! (a b)
      (let ((tmp (gensym)))
        (list 'let (list (list tmp a))
              (list 'set! a b)
              (list 'set! b tmp))))
    (let ((tmp 1) (other 2))
      (swap-set! tmp other)
      (list tmp other))
  `
  evalEq(t, src, "(2 1)")
}

// TestMacroGensymUnique: gensym erzeugt frische Symbole, die nicht mit
// Nutzersymbolen kollidieren. Basis für hygienische Makros (manuell).
func TestMacroGensymUnique(t *testing.T) {
  src := `
    (defmacro capture-safe (expr)
      (let ((g (gensym)))
        (list 'let (list (list g expr)) g)))
    (let ((g 99))
      (capture-safe (+ 1 2)))
  `
  evalEq(t, src, "3")  // gensym-g ≠ Nutzer-g → expr=3, nicht Nutzer-g=99
}

// TestMacroVariadic: Makro mit &rest sammelt restliche Argumente.
func TestMacroVariadic(t *testing.T) {
  src := `
    (defmacro my-list (&rest args) (cons 'list args))
    (my-list 1 2 3)
  `
  evalEq(t, src, "(1 2 3)")
}

// TestMacroArityError: Makro mit falscher Argumentzahl → bindArgs-Fehler.
func TestMacroArityError(t *testing.T) {
  src := `
    (defmacro needs2 (a b) (list '+ a b))
    (needs2 1)
  `
  evalErr(t, src)
}

// TestIsMacroGo prüft die exportierte IsMacro-Hilfsfunktion direkt.
// defmacro gibt den Namen als Atom zurück, nicht das Makro – das Makro
// muss via env.Get aus der Umgebung geholt werden.
func TestIsMacroGo(t *testing.T) {
  env := BaseEnv()
  defexpr, err := Read(`(defmacro m (x) x)`)
  if err != nil {
    t.Fatalf("Read defmacro: %v", err)
  }
  if _, err := Eval(defexpr, env); err != nil {
    t.Fatalf("Eval defmacro: %v", err)
  }
  mVal, err := env.Get("m")
  if err != nil {
    t.Fatalf("env.Get(m): %v", err)
  }
  if !IsMacro(mVal) {
    t.Error("IsMacro(defmacro-Ergebnis aus env) = false, will true")
  }
  // Normale Lambda ist kein Makro
  lamExpr, _ := Read(`(lambda (x) x)`)
  lam, _ := Eval(lamExpr, env)
  if IsMacro(lam) {
    t.Error("IsMacro(lambda) = true, will false")
  }
  if IsMacro(MakeNum(5)) {
    t.Error("IsMacro(5) = true, will false")
  }
}

// TestMacroReturnsAtom: defmacro gibt den Makro-Namen als Atom zurück.
func TestMacroReturnsAtom(t *testing.T) {
  evalEq(t, `(defmacro m (x) x)`, "m")
}

// TestEvalMacrolet: lokale Makros — nur im Body sichtbar, schatten globale,
// nicht-rekursiv (CL: Makro-Body sieht die macrolet-Bindungen nicht).
func TestEvalMacrolet(t *testing.T) {
  evalEq(t, `(macrolet ((doppelt (x) (list '* 2 x))) (doppelt 21))`, "42")
  evalEq(t, `(macrolet ((a () 1)) (a))`, "1")
  // mehrere Bindungen, Body mit mehreren Formen
  evalEq(t, `(macrolet ((a () 1) (b () 2)) (a) (b))`, "2")
  // schattet globale Funktion im Body, außerhalb unverändert
  evalEq(t, `
    (defun f (x) (* x 10))
    (list (macrolet ((f (x) (list '+ x 1))) (f 5)) (f 5))`, "(6 50)")
  // leere Bindungsliste verhält sich wie progn
  evalEq(t, `(macrolet () 1 2 3)`, "3")
  // defun im Body sieht das Makro (Expansion zur Def-Zeit)
  evalEq(t, `(macrolet ((m () 'global)) (defun ruft-m () (m)))`, "ruft-m")
  evalErr(t, `(macrolet (kaputt) 1)`)
}

// TestEvalSymbolMacrolet: SYMMACRO-Marker im Env. Referenz wertet die
// Expansion aus; setq wird auf Symbol-Expansionen umgeleitet; echte
// innere Bindungen (let/lambda) schatten den Marker über die Env-Kette.
func TestEvalSymbolMacrolet(t *testing.T) {
  evalEq(t, `(symbol-macrolet ((x 42)) x)`, "42")
  evalEq(t, `(symbol-macrolet ((x (+ 1 2))) (* x 10))`, "30")
  // Expansion wird bei jeder Referenz neu ausgewertet
  evalEq(t, `
    (setq *zaehler* 0)
    (symbol-macrolet ((x (setq* *zaehler* (+ *zaehler* 1))))
      x x *zaehler*)`, "2")
  // setq auf Symbol-Makro wirkt auf die Expansion (CL: wie setf)
  evalEq(t, `(let ((y 5)) (symbol-macrolet ((x y)) (setq x 9) y))`, "9")
  // Shadowing: let und lambda-Parameter gewinnen vor dem Marker
  evalEq(t, `(symbol-macrolet ((x 1)) (let ((x 2)) x))`, "2")
  evalEq(t, `(symbol-macrolet ((x 5)) ((lambda (x) (+ x 1)) 9))`, "10")
  // nach dem Body ist das Symbol wieder frei
  evalErr(t, `(begin (symbol-macrolet ((x 1)) x) x)`)
  // Form-Expansion ist lesbar, aber nicht zuweisbar
  evalErr(t, `(symbol-macrolet ((x (+ 1 2))) (setq x 9))`)
  evalErr(t, `(symbol-macrolet (kaputt) 1)`)
}

// TestEvalBodyAlias: &body ist CL-Synonym für &rest (Makro-Lambda-Listen).
func TestEvalBodyAlias(t *testing.T) {
  evalEq(t, `(defmacro mit-tmp (wert &body body) `+"`(let ((tmp ,wert)) ,@body))"+`
    (mit-tmp 10 (setq tmp (+ tmp 1)) tmp)`, "11")
  // &body ohne Rest-Argumente → leere Liste
  evalEq(t, `(defmacro nur-body (&body body) `+"`(list ,@body))"+`
    (nur-body)`, "()")
}
