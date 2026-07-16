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
