//**********************************************************************
//  lib/evalBench_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260805
//**********************************************************************
// Zweiter Benchmark-Satz neben fibBench_test.go.
//
// fib misst genau einen Fall: einstelliger Lambda-Aufruf, Zahlen aus dem
// Small-Int-Cache, keine Cell-Allokation, keine Strings, keine Makros,
// keine Closures. Damit ist es blind fuer die Pfade, in denen die
// Korrektheitsfehler von 2026-08-05 sassen (Closure x Frame) und fuer den
// naechsten Optimierungskandidaten (Cell, 104 B).
//
// Jeder Benchmark hier misst EINEN Pfad, den fib nicht sieht. Definition
// getrennt vom Aufruf, damit nur der Aufruf im Messloop landet — dieselbe
// Konvention wie fibBench_test.go.
//
// Siehe PerfTODO §6.
//**********************************************************************

package lib

import "testing"

// benchEval baut Env + Stdlib auf, evaluiert setup und misst dann call.
func benchEval(b *testing.B, setup, call string) {
  b.Helper()
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    b.Fatalf("stdlib: %v", err)
  }
  if setup != "" {
    exprs, err := ReadAll(setup)
    if err != nil {
      b.Fatalf("read setup: %v", err)
    }
    for _, e := range CellToSlice(exprs) {
      if _, err := Eval(e, env); err != nil {
        b.Fatalf("eval setup: %v", err)
      }
    }
  }
  expr, err := Read(call)
  if err != nil {
    b.Fatalf("read call: %v", err)
  }
  // Einmal ausserhalb der Messung laufen lassen: faengt Fehler, die sonst
  // erst im Loop auffallen, und waermt die Intern-Tabelle.
  if _, err := Eval(expr, env); err != nil {
    b.Fatalf("eval call: %v", err)
  }

  b.ReportAllocs()
  b.ResetTimer()
  for i := 0; i < b.N; i++ {
    if _, err := Eval(expr, env); err != nil {
      b.Fatalf("eval: %v", err)
    }
  }
}

// BenchmarkListBuild: Cell-lastig. Baut eine 1000-elementige Liste auf und
// laeuft sie ab. Jedes Cons ist eine Cell (104 B) — der Posten, den fib
// nicht sieht, weil dort der Small-Int-Cache alles abfaengt.
func BenchmarkListBuild(b *testing.B) {
  benchEval(b, `
    (defun build (n acc) (if (= n 0) acc (build (- n 1) (cons n acc))))
    (defun total (l) (if (null l) 0 (+ (car l) (total (cdr l)))))`,
    `(total (build 1000 '()))`)
}

// BenchmarkMultiArgLambda: vierstelliges Lambda. Der erste Parameter liegt
// inline in singleSym, die drei weiteren im entries-Slice — der Pfad, den
// fib mit seinem einen Parameter niemals betritt.
func BenchmarkMultiArgLambda(b *testing.B) {
  benchEval(b, `
    (defun quad (a bb c d) (+ (+ a bb) (+ c d)))
    (defun loop4 (n acc) (if (= n 0) acc (loop4 (- n 1) (quad 1 2 3 acc))))`,
    `(loop4 2000 0)`)
}

// BenchmarkClosureCreate: Closure pro Iteration erzeugen und aufrufen.
// Genau das Muster, in dem der Env-Pool-Fehler sass (PerfTODO §4.5b) —
// und genau das, was fib nicht enthaelt.
func BenchmarkClosureCreate(b *testing.B) {
  benchEval(b, `
    (defun make-adder (n) (lambda (x) (+ x n)))
    (defun sum-adders (k acc)
      (if (= k 0) acc (sum-adders (- k 1) (funcall (make-adder k) acc))))`,
    `(sum-adders 2000 0)`)
}

// BenchmarkMacroExpand: Makro-Expansion im Eval-Pfad. Laeuft ueber
// applyLambda mit Type MACRO, also einen anderen Pfad als der
// Lambda-TCO-Zweig im Trampolin.
func BenchmarkMacroExpand(b *testing.B) {
  benchEval(b, `
    (defmacro twice (x) (list '+ x x))
    (defun loopm (n acc) (if (= n 0) acc (loopm (- n 1) (twice 1))))`,
    `(loopm 2000 0)`)
}

// BenchmarkStringOps: String-Pfad. Strings sind nicht interniert und nicht
// gecacht, jede Operation allokiert eine Cell plus den Go-String.
func BenchmarkStringOps(b *testing.B) {
  benchEval(b, `
    (defun cat (n acc)
      (if (= n 0) acc (cat (- n 1) (string-append acc "x"))))`,
    `(string-length (cat 500 ""))`)
}

// BenchmarkLetChain: verschachtelte let-Frames. Miss den Frame-Aufbau ohne
// Funktionsaufruf-Overhead — nach Schnitt 7 liegt jeder Frame auf dem Heap.
func BenchmarkLetChain(b *testing.B) {
  benchEval(b, `
    (defun nest (n acc)
      (if (= n 0) acc
        (let ((a (+ acc 1)))
          (let ((bb (+ a 1)))
            (nest (- n 1) bb)))))`,
    `(nest 2000 0)`)
}
