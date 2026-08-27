//**********************************************************************
//  lib/stdlib_reduce_setf_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260723
//**********************************************************************
// Tests für CL-konformes reduce (&key initial-value/from-end/start/end/key)
// und setf auf ungebundenen Symbolen (CL: globale Definition).
// Entscheidungen: Gerhard, 20260723 (TODO.md Aufgabe 20270723).
package lib

import (
  "strings"
  "testing"
)

// evalStdlib wertet src in einem frischen Env mit geladener stdlib aus.
func evalStdlib(t *testing.T, src string) (*Cell, error) {
  t.Helper()
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("stdlib: %v", err)
  }
  exprs, err := ReadAll(strings.TrimSpace(src))
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result := MakeNil()
  for _, e := range CellToSlice(exprs) {
    result, err = Eval(e, env)
    if err != nil {
      return nil, err
    }
  }
  return result, nil
}

func evalStdlibEq(t *testing.T, src, want string) {
  t.Helper()
  got, err := evalStdlib(t, src)
  if err != nil {
    t.Fatalf("eval(%q) Fehler: %v", src, err)
  }
  if got.String() != want {
    t.Errorf("eval(%q) = %q, want %q", src, got.String(), want)
  }
}

func evalStdlibErr(t *testing.T, src string) {
  t.Helper()
  if _, err := evalStdlib(t, src); err == nil {
    t.Errorf("eval(%q) sollte Fehler geben", src)
  }
}

// --- reduce (CL-Signatur) ---

func TestReduceCL(t *testing.T) {
  // ohne initial-value: erstes Element ist Startwert
  evalStdlibEq(t, `(reduce + '(1 2 4 7 55))`, "69")
  evalStdlibEq(t, `(reduce - '(1 2 3))`, "-4")           // (- (- 1 2) 3)
  evalStdlibEq(t, `(reduce + '(5))`, "5")                 // einelementig, kein f-Aufruf
  // mit initial-value
  evalStdlibEq(t, `(reduce + '(1 2 3) :initial-value 100)`, "106")
  evalStdlibEq(t, `(reduce + '() :initial-value 10)`, "10")
  // leere Sequenz ohne initial-value → Fehler (CL)
  evalStdlibErr(t, `(reduce + '())`)
  // from-end: f erhält (elem acc) statt (acc elem)
  evalStdlibEq(t, `(reduce - '(1 2 3) :from-end t)`, "2") // (- 1 (- 2 3))
  // start/end
  evalStdlibEq(t, `(reduce + '(1 2 3 4 5) :start 1 :end 4)`, "9")
  // key: ohne init bleibt erstes Element roh, Rest durch key
  evalStdlibEq(t, `(reduce + '(2 3 4) :key (lambda (x) (* x x)))`, "27") // 2 + 9 + 16
  evalStdlibEq(t, `(reduce + '(2 3 4) :initial-value 0 :key (lambda (x) (* x x)))`, "29")
  // max/min (interne reduce-Nutzer) müssen mitwandern
  evalStdlibEq(t, `(max 3 1 4 1 5)`, "5")
  evalStdlibEq(t, `(min 3 1 4)`, "1")
  evalStdlibEq(t, `(max 7)`, "7")
}

// --- setf auf ungebundenem Symbol (CL: globale Definition) ---

func TestSetfUnbound(t *testing.T) {
  // frisches Symbol → global definiert
  evalStdlibEq(t, `(begin (setf zz-fresh 42) zz-fresh)`, "42")
  // Rückgabewert ist der zugewiesene Wert (CL)
  evalStdlibEq(t, `(setf zz-ret 99)`, "99")
  // aus Lambda heraus landet die Definition global (eval läuft im Root-Env)
  evalStdlibEq(t, `(begin (defun zz-f () (setf zz-q 7)) (zz-f) zz-q)`, "7")
  // gebundenes Symbol weiterhin via set!
  evalStdlibEq(t, `(begin (define zz-b 1) (setf zz-b 2) zz-b)`, "2")
  // Listen-Wert (quote schützt vor Re-Evaluierung)
  evalStdlibEq(t, `(begin (setf zz-l '(1 2 3)) zz-l)`, "(1 2 3)")
  // Places bleiben unverändert funktionsfähig
  evalStdlibEq(t, `(begin (define h (make-hash-table)) (setf (gethash 'a h) 42) (gethash 'a h))`, "42")
  evalStdlibEq(t, `(begin (define xs '(1 2 3)) (setf (nth 1 xs) 9) xs)`, "(1 9 3)")
}

// --- setf auf car/cdr-Place (Symbol-Rebind, keine CL-Aliasing-Semantik) ---

func TestSetfCarCdr(t *testing.T) {
  // Basis-Fall
  evalStdlibEq(t, `(begin (define xs '(1 2 3)) (setf (car xs) 9) xs)`, "(9 2 3)")
  evalStdlibEq(t, `(begin (define xs '(1 2 3)) (setf (cdr xs) '(8 9)) xs)`, "(1 8 9)")
  // Rückgabewert ist der zugewiesene Wert (CL)
  evalStdlibEq(t, `(begin (define xs '(1 2 3)) (setf (car xs) 9))`, "9")
  // Wert wird genau einmal ausgewertet
  evalStdlibEq(t, `(begin (define n 0) (define xs '(1 2))
                          (setf (car xs) (begin (setf n (+ n 1)) n)) n)`, "1")
  // Rebind-statt-Mutation: ein Alias (zweites Symbol auf dieselbe Liste)
  // sieht die Änderung NICHT — abweichend von CL-rplaca, wie dokumentiert.
  evalStdlibEq(t, `(begin (define xs '(1 2 3)) (define ys xs) (setf (car xs) 9) ys)`, "(1 2 3)")
  // Nicht-Symbol-Argument → Fehler (wie bei nth)
  evalStdlibErr(t, `(setf (car (list 1 2)) 9)`)
  evalStdlibErr(t, `(setf (cdr (list 1 2)) 9)`)
  // nil ist keine Cons-Zelle → Fehler (CL: rplaca/rplacd auf nil)
  evalStdlibErr(t, `(let ((x nil)) (setf (car x) 1))`)
  evalStdlibErr(t, `(let ((x nil)) (setf (cdr x) 1))`)
}
