//**********************************************************************
//  lib/eval_control_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k2.7-code
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260716
//**********************************************************************
// Tests für Control-Flow- und Nebenläufigkeits-Spezialformen.
//**********************************************************************

package lib

import (
  "testing"
  "time"
)

func TestParfuncTimeoutCancelsWorker(t *testing.T) {
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("stdlib: %v", err)
  }

  // Unendliche Schleife — ohne Cancellation würde die Goroutine ewig laufen.
  code := `(parfunc r :timeout 1 (while t 1))`
  expr, err := Read(code)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  start := time.Now()
  _, err = Eval(expr, env)
  elapsed := time.Since(start)
  if err != nil {
    t.Fatalf("eval: %v", err)
  }
  if elapsed > 3*time.Second {
    t.Fatalf("parfunc timeout too slow: %v", elapsed)
  }

  // Warte kurz, damit der Worker die Cancellation bemerkt.
  time.Sleep(500 * time.Millisecond)
}

// cond-Klausel ohne Körper liefert den Testwert, ohne ihn erneut
// auszuwerten (Task 4b, 0.5.1).
func TestCondOhneKoerperSymbol(t *testing.T) {
  evalEq(t, `(cond ('foo))`, "foo")
}

func TestCondOhneKoerperListe(t *testing.T) {
  evalEq(t, `(cond ((list 1 2)))`, "(1 2)")
}

// :timeout mit Bruchteilen von Sekunden (Task 4b, 0.5.1)
func TestParfuncTimeoutBruchteil(t *testing.T) {
  start := time.Now()
  evalEq(t, `(parfunc r :timeout 0.2 (progn (sleep 1000) 1)) r`, "(())")
  if d := time.Since(start); d > 700*time.Millisecond {
    t.Fatalf("parfunc :timeout 0.2 dauerte %v", d)
  }
}

// :timeout 0 bedeutet weiterhin: kein Timeout
func TestParfuncTimeoutNull(t *testing.T) {
  evalEq(t, `(parfunc r :timeout 0 (progn (sleep 50) 1)) r`, "(1)")
}

func TestParfuncTimeoutKeineZahl(t *testing.T) {
  evalErr(t, `(parfunc r :timeout "x" 1)`)
}
