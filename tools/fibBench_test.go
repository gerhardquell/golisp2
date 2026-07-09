//**********************************************************************
//  lib/fibBench_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude opus 4.8
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260626
//**********************************************************************
// Mikrobenchmark: misst Eval-Overhead pro Funktionsaufruf via fib.
// fib ist baumrekursiv (kein TCO) -> jeder Call laeuft den vollen
// Funktionsanwendungs-Pfad ganz unten in Eval. allocs/op ist die
// Schluesselzahl: bestaetigt o. widerlegt den Allokations-Verdacht.
//**********************************************************************

package lib

import "testing"

// Definition getrennt vom Aufruf, damit nur der Aufruf im Messloop
// landet (defun-Overhead raus). fib 25 ~ 242k Calls/Iteration.
const fibDefSrc  = `(defun fib (n) (if (< n 2) n (+ (fib (- n 1)) (fib (- n 2)))))`
const fibCallSrc = `(fib 25)`

// BenchmarkFib misst einen kompletten fib(25)-Lauf pro Iteration.
func BenchmarkFib(b *testing.B) {
  env := BaseEnv()
  def, err := Read(fibDefSrc)
  if err != nil { b.Fatal(err) }
  if _, err := Eval(def, env); err != nil { b.Fatal(err) }

  call, err := Read(fibCallSrc)
  if err != nil { b.Fatal(err) }

  b.ReportAllocs()
  b.ResetTimer()
  for i := 0; i < b.N; i++ {
    if _, err := Eval(call, env); err != nil { b.Fatal(err) }
  }
}
