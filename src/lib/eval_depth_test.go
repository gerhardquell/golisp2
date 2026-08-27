//**********************************************************************
//  lib/eval_depth_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260805
//**********************************************************************
// Charakterisierungsnetz fuer evalCtx: Rekursionstiefe und Cancellation.
// Haelt das IST fest, BEVOR evalCtx umgebaut wird (PerfTODO §6) —
// evalCtx.child() ist mit 84,9 % der dominante Allokationsposten.
//
// Bewusst nur ueber die oeffentliche API (Eval + Lisp-Code), damit die
// Tests eine Signaturaenderung an evalWithCtx/child unveraendert
// ueberstehen. Ein Netz, das beim Refactoring selbst umgeschrieben werden
// muss, sichert das Refactoring nicht ab.
//
// Die Grenzwerte sind bei maxEvalDepth=50 kalibriert: (sum n) verbraucht
// rund 2 Tiefeneinheiten pro Ebene, laeuft bis n=20 und scheitert ab n=24.
//**********************************************************************

package lib

import (
  "strings"
  "testing"
  "time"
)

// sumDef: nicht-tail-rekursiv. Jede Ebene legt einen Go-Frame an, weil das
// Ergebnis noch in (+ ... 1) verrechnet wird.
const sumDef = `(defun sum (n) (if (= n 0) 0 (+ (sum (- n 1)) 1)))`

// tailDef: tail-rekursiv. Darf die Tiefe NICHT verbrauchen (Trampolin).
const tailDef = `(defun tl (n) (if (= n 0) 'ok (tl (- n 1))))`

// withDepth setzt das Limit fuer die Dauer von fn und stellt es zurueck.
func withDepth(t *testing.T, limit int, fn func()) {
  t.Helper()
  old := GetMaxEvalDepth()
  SetMaxEvalDepth(limit)
  defer SetMaxEvalDepth(old)
  fn()
}

// TestDepthTailCallsAreFree: 10 000 Tail-Calls bei einem Limit von 50.
// Geht nur, wenn Tail-Positionen die Tiefe nicht erhoehen — das ist die
// Kopplung zwischen TCO-Trampolin und evalCtx.
func TestDepthTailCallsAreFree(t *testing.T) {
  withDepth(t, 50, func() {
    got, err := evalStdlib(t, tailDef+" (tl 10000)")
    if err != nil {
      t.Fatalf("Tail-Rekursion verbraucht Tiefe: %v", err)
    }
    if got.String() != "ok" {
      t.Fatalf("got %s, want ok", got.String())
    }
  })
}

// TestDepthLimitFiresOnNonTail: nicht-tail-rekursiv muss anschlagen.
func TestDepthLimitFiresOnNonTail(t *testing.T) {
  withDepth(t, 50, func() {
    _, err := evalStdlib(t, sumDef+" (sum 100)")
    if err == nil {
      t.Fatal("erwartete Tiefenbegrenzung, bekam nil")
    }
    if !strings.Contains(err.Error(), "maximum recursion depth exceeded") {
      t.Fatalf("unerwarteter Fehler: %v", err)
    }
  })
}

// TestDepthResetsBetweenTopLevelForms: das Budget ist pro Eval-Aufruf, nicht
// global. Zwei aufeinanderfolgende Formen duerfen sich nicht addieren, und
// ein Fehler darf das Budget nicht dauerhaft verbrauchen.
func TestDepthResetsBetweenTopLevelForms(t *testing.T) {
  withDepth(t, 50, func() {
    // dreimal knapp unter dem Limit, hintereinander im selben Env
    got, err := evalStdlib(t, sumDef+" (sum 20) (sum 20) (sum 20)")
    if err != nil {
      t.Fatalf("Tiefe addiert sich ueber Top-Level-Formen: %v", err)
    }
    if got.String() != "20" {
      t.Fatalf("got %s, want 20", got.String())
    }
  })
}

// TestDepthBudgetIsPerNesting: der Trennfall. Bei Limit 50 laeuft (sum 12)
// allein, aber nicht mehr, wenn 12 Ebenen Verschachtelung davor liegen.
// Ohne diesen Test koennte ein Refactoring die Tiefe versehentlich pro
// Aufruf zuruecksetzen, ohne dass etwas auffaellt.
func TestDepthBudgetIsPerNesting(t *testing.T) {
  withDepth(t, 50, func() {
    if _, err := evalStdlib(t, sumDef+" (sum 12)"); err != nil {
      t.Fatalf("(sum 12) sollte allein laufen: %v", err)
    }
    nested := sumDef + `
      (defun dd (n) (if (= n 0) (sum 12) (+ (dd (- n 1)) 0)))
      (dd 12)`
    if _, err := evalStdlib(t, nested); err == nil {
      t.Fatal("12 Ebenen + (sum 12) sollten das Limit reissen")
    }
  })
}

// TestEvalFormResetsDepthBudget: (eval form) startet bei Tiefe 0 —
// evalEval erzeugt bewusst einen frischen Kontext (eval_control.go).
// Gegenprobe zu TestDepthBudgetIsPerNesting mit identischer Verschachtelung.
func TestEvalFormResetsDepthBudget(t *testing.T) {
  withDepth(t, 50, func() {
    viaEval := sumDef + `
      (defun de (n) (if (= n 0) (eval (quote (sum 12))) (+ (de (- n 1)) 0)))
      (de 12)`
    got, err := evalStdlib(t, viaEval)
    if err != nil {
      t.Fatalf("(eval form) setzt das Tiefenbudget nicht zurueck: %v", err)
    }
    if got.String() != "12" {
      t.Fatalf("got %s, want 12", got.String())
    }
  })
}

// TestFuncallResetsDepthBudget: apply/funcall erzeugen ebenfalls einen
// frischen Kontext (apply in eval_core.go). Dokumentiert das IST.
//
// Nebenbefund, NICHT von diesem Refactoring verursacht: damit ist die
// Tiefenbegrenzung kein hartes Limit — Lisp-Code kann sie durch Bouncen
// ueber funcall/eval beliebig weit umgehen, obwohl beide echten Go-Stack
// verbrauchen. Siehe PerfTODO §6.
func TestFuncallResetsDepthBudget(t *testing.T) {
  withDepth(t, 50, func() {
    viaFuncall := sumDef + `
      (defun df (n) (if (= n 0) (funcall (lambda () (sum 12))) (+ (df (- n 1)) 0)))
      (df 12)`
    got, err := evalStdlib(t, viaFuncall)
    if err != nil {
      t.Fatalf("funcall setzt das Tiefenbudget nicht zurueck: %v", err)
    }
    if got.String() != "12" {
      t.Fatalf("got %s, want 12", got.String())
    }
  })
}

// TestCancellationReachesTailLoop: parfunc ist der einzige Pfad, der einen
// context.Context in evalCtx setzt. Eine endlose TAIL-Schleife hat keine
// steigende Tiefe, muss also ueber den periodischen check() im Trampolin
// abbrechen. Bricht die ctx-Weitergabe an child(), haengt das hier ewig.
func TestCancellationReachesTailLoop(t *testing.T) {
  // Env und Form in der Test-Goroutine aufbauen: evalStdlib ruft intern
  // t.Fatalf, und das ist aus einer Nicht-Test-Goroutine unzulaessig
  // (runtime.Goexit im falschen Stack).
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("stdlib: %v", err)
  }
  expr, err := Read(`(parfunc r :timeout 1 (while t 1))`)
  if err != nil {
    t.Fatalf("read: %v", err)
  }

  errc := make(chan error, 1)
  go func() { _, e := Eval(expr, env); errc <- e }()

  select {
  case e := <-errc:
    if e != nil {
      t.Fatalf("parfunc: %v", e)
    }
  case <-time.After(10 * time.Second):
    t.Fatal("Cancellation erreicht die Tail-Schleife nicht — haengt")
  }
}
