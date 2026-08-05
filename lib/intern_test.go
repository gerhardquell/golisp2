//**********************************************************************
//  lib/intern_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260805
//**********************************************************************
// Symbol-Interning: eq ist Pointer-Identität, also muss dasselbe Symbol
// dieselbe Cell sein. Ohne Interning gab MakeAtom bei jedem Aufruf eine
// frische Cell zurück und (eq 'foo 'foo) war () — CL sagt T.
//
// cellT/cellNil waren ein Ad-hoc-Interning für genau zwei Werte und
// machten das Verhalten inkonsistent: member/assoc lieferten das eine
// NIL, die Vergleichsprädikate ein zweites.
//
// Sollwerte sind SBCL-Ergebnisse. Bewusste Abweichungen (Zahlen, Strings)
// sind unten als solche markiert und begründet.
//
// Siehe TODO.md Abschnitt 4.1.
//**********************************************************************

package lib

import (
  "sync"
  "testing"
)

type eqCase struct {
  name string
  src  string
  want string
}

// internedCases: alles, was durch Interning eq-korrekt werden muss.
// SBCL liefert für jeden dieser Fälle T.
var internedCases = []eqCase{
  {"symbol_gleich",        "(eq 'foo 'foo)",                  "t"},
  {"symbol_t",             "(eq 't 't)",                      "t"},
  {"keyword",              "(eq ':k ':k)",                    "t"},
  {"praedikat_liefert_t",  "(eq (= 1 1) 't)",                 "t"},
  {"praedikat_liefert_nil","(eq (= 1 2) '())",                "t"},
  {"null_liefert_t",       "(eq (null '()) 't)",              "t"},
  {"atom_liefert_t",       "(eq (atom 1) 't)",                "t"},
  {"lt_liefert_nil",       "(eq (< 2 1) '())",                "t"},
  {"gt_liefert_t",         "(eq (> 2 1) 't)",                 "t"},
  {"listenliteral",        "(let ((l '(x x))) (eq (car l) (cadr l)))", "t"},
  // Diese beiden gingen schon vorher — sie nutzten MakeNil() statt
  // cellNil. Bleiben als Konsistenzwächter drin: nach dem Fix müssen
  // ALLE nil-liefernden Primitiven dieselbe Cell liefern.
  {"member_liefert_nil",   "(eq (member 9 '(1 2)) '())",      "t"},
  {"assoc_liefert_nil",    "(eq (assoc 9 '((1 2))) '())",     "t"},
  {"nil_literal",          "(eq '() '())",                    "t"},
  // Quervergleich: nil aus einem Prädikat und nil aus einem Literal
  // müssen identisch sein. Das war der eigentliche Bruch.
  {"nil_praedikat_vs_literal", "(eq (= 1 2) (car '(())))",    "t"},
}

// negativeCases: Interning darf NICHT zu viel gleichmachen.
var negativeCases = []eqCase{
  {"verschiedene_symbole", "(eq 'foo 'bar)",   "()"},
  {"symbol_vs_keyword",    "(eq 'k ':k)",      "()"},
  {"symbol_vs_string",     "(eq 'a \"a\")",    "()"},
}

// deliberateDivergence: bewusste golisp2-Abweichungen von SBCL.
// Charakterisierungstests — sie halten eine Entscheidung fest, nicht
// einen Bug. Wer sie bricht, ändert Semantik und muss das begründen.
var deliberateDivergence = []eqCase{
  // SBCL: (eq 5 5) => T (Fixnums sind dort eq). CL lässt es für Zahlen
  // ausdrücklich UNSPEZIFIZIERT. fnEqPtr filtert NUMBER bewusst heraus,
  // damit der Small-Int-Cache nicht durch die Hintertür eq-Semantik
  // verändert. Siehe lib/primitives.go fnEqPtr.
  {"zahlen_nicht_eq", "(eq 5 5)", "()"},
  // SBCL: (eq "a" "a") => NIL. Strings werden NICHT interniert.
  {"strings_nicht_eq", "(eq \"a\" \"a\")", "()"},
  // equal? bleibt struktureller Vergleich und muss für Strings greifen.
  {"equal_auf_strings", "(equal? \"a\" \"a\")", "t"},
}

// runEqCases nutzt evalStdlib (aus stdlib_reduce_setf_test.go): member,
// assoc und cadr sind Stdlib-Lisp, nicht Go-Primitiven. Mit dem
// stdlib-freien evalStr scheiterten diese Faelle an einem unbekannten
// Symbol statt an der eq-Semantik — ein Test, der aus dem falschen Grund
// rot ist, beweist nichts.
func runEqCases(t *testing.T, cases []eqCase) {
  t.Helper()
  for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) {
      got, err := evalStdlib(t, tc.src)
      if err != nil {
        t.Fatalf("%s: eval: %v", tc.src, err)
      }
      if got.String() != tc.want {
        t.Fatalf("%s = %s, want %s", tc.src, got.String(), tc.want)
      }
    })
  }
}

// TestSymbolInterningMakesEqWork: dasselbe Symbol ist dieselbe Cell.
func TestSymbolInterningMakesEqWork(t *testing.T) {
  runEqCases(t, internedCases)
}

// TestSymbolInterningStaysDiscriminating: verschiedene Symbole bleiben
// verschieden.
func TestSymbolInterningStaysDiscriminating(t *testing.T) {
  runEqCases(t, negativeCases)
}

// TestEqDeliberateDivergenceFromSBCL: bewusste Abweichungen festhalten.
func TestEqDeliberateDivergenceFromSBCL(t *testing.T) {
  runEqCases(t, deliberateDivergence)
}

// TestMakeAtomReturnsSameCell: Interning auf Go-Ebene.
func TestMakeAtomReturnsSameCell(t *testing.T) {
  a := MakeAtom("interntest-xyz")
  b := MakeAtom("interntest-xyz")
  if a != b {
    t.Fatalf("MakeAtom liefert verschiedene Cells fuer denselben Namen: %p != %p", a, b)
  }
  c := MakeAtom("interntest-abc")
  if a == c {
    t.Fatal("MakeAtom liefert dieselbe Cell fuer verschiedene Namen")
  }
}

// TestMakeStrIsNotInterned: Strings bleiben frische Cells (CL: (eq "a" "a")
// ist NIL). Nur Symbole werden interniert.
func TestMakeStrIsNotInterned(t *testing.T) {
  if MakeStr("interntest-str") == MakeStr("interntest-str") {
    t.Fatal("MakeStr sollte NICHT internieren")
  }
}

// TestMakeAtomConcurrent: parfunc laesst mehrere Goroutinen gleichzeitig
// evaluieren, also muss die Intern-Tabelle thread-sicher sein UND unter
// Nebenlaeufigkeit denselben Pointer liefern. Mit -race laufen lassen.
func TestMakeAtomConcurrent(t *testing.T) {
  const goroutines = 32
  const names = 16

  var wg sync.WaitGroup
  results := make([][]*Cell, goroutines)

  for g := 0; g < goroutines; g++ {
    wg.Add(1)
    go func(g int) {
      defer wg.Done()
      row := make([]*Cell, names)
      for n := 0; n < names; n++ {
        row[n] = MakeAtom("concurrent-sym-" + string(rune('a'+n)))
      }
      results[g] = row
    }(g)
  }
  wg.Wait()

  for n := 0; n < names; n++ {
    want := results[0][n]
    for g := 1; g < goroutines; g++ {
      if results[g][n] != want {
        t.Fatalf("Symbol %d: Goroutine %d hat anderen Pointer (%p != %p)",
          n, g, results[g][n], want)
      }
    }
  }
}
