//**********************************************************************
//  cli/selftest.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261001
//**********************************************************************
// Eingebauter Selbsttest für golisp2 -t (verschoben aus main.go).
//**********************************************************************

package cli

import (
  "fmt"
  "os"

  "golisp2/src/lib"
)

// test wertet s aus und druckt das Ergebnis. Liefert 1 bei einem Fehler,
// sonst 0 — runTests zählt das mit, damit ein scheiterndes assert= in einer
// geladenen Testdatei den Lauf nicht mehr grün lässt.
func test(env *lib.Env, s string) int {
  cell, _ := lib.Read(s)
  result, err := lib.Eval(cell, env)
  if err != nil {
    printErr(os.Stdout, err)
    return 1
  }
  fmt.Printf("%-45s => %s\n", s, result)
  return 0
}

func runTests(env *lib.Env) int {
  errs := 0
  fmt.Println("=== GoLisp Test ===")
  fmt.Println()
  errs += test(env, "(+ 1 2)")
  errs += test(env, "(defun fak (n) (if (= n 0) 1 (* n (fak (- n 1)))))")
  errs += test(env, "(fak 6)")
  errs += test(env, "(mapcar (lambda (x) (* x x)) (list 1 2 3 4 5))")
  errs += test(env, "(parfunc r (* 6 7) (+ 100 23))")
  errs += test(env, "r")
  errs += test(env, "(atom (gensym))")
  errs += test(env, "(eq (gensym) (gensym))")
  errs += test(env, `(trap (error "oops") (lambda (e) (string-append "caught: " e)))`)
  errs += test(env, `(trap (+ 1 2) (lambda (e) "fehler"))`)
  errs += test(env, `(defun add-and-show (x y) (define s (+ x y)) s)`)
  errs += test(env, `(add-and-show 3 4)`)
  errs += test(env, `((lambda (x) (define d (* x 2)) d) 5)`)
  errs += test(env, `(>= 5 3)`)
  errs += test(env, `(>= 3 3)`)
  errs += test(env, `(>= 2 3)`)
  errs += test(env, `(<= 2 3)`)
  errs += test(env, `(<= 3 3)`)
  errs += test(env, `(<= 5 3)`)
  // while
  errs += test(env, `(define w 0)`)
  errs += test(env, `(while (< w 3) (set! w (+ w 1)))`)
  errs += test(env, `w`)
  // do: (do ((i 0 (+ i 1)) (s 0 (+ s i))) ((= i 5) s))  → 0+1+2+3+4 = 10
  errs += test(env, `(do ((i 0 (+ i 1)) (s 0 (+ s i))) ((= i 5) s))`)
  // TCO: tiefe Rekursion ohne Stack-Overflow
  errs += test(env, `(defun sum-acc (n acc) (if (= n 0) acc (sum-acc (- n 1) (+ acc n))))`)
  errs += test(env, `(sum-acc 1000000 0)`)
  errs += test(env, `(defun even? (n) (if (= n 0) t (odd?  (- n 1))))`)
  errs += test(env, `(defun odd?  (n) (if (= n 0) () (even? (- n 1))))`)
  errs += test(env, `(even? 1000000)`)
  // equal?
  errs += test(env, `(equal? 1 1)`)
  errs += test(env, `(equal? 1 2)`)
  errs += test(env, `(equal? "a" "a")`)
  errs += test(env, `(equal? (list 1 2) (list 1 2))`)
  errs += test(env, `(equal? (list 1 2) (list 1 3))`)
  errs += test(env, `(equal? (list 1 (list 2 3)) (list 1 (list 2 3)))`)
  errs += test(env, `(equal? () ())`)
  // &optional / &key Parameter
  errs += test(env, `(defun greet (name &optional (greeting "Hallo")) (string-append greeting " " name))`)
  errs += test(env, `(greet "Gerhard")`)
  errs += test(env, `(greet "Gerhard" "Hi")`)
  errs += test(env, `(defun f-key (x &key (y 0)) (+ x y))`)
  errs += test(env, `(f-key 10)`)
  errs += test(env, `(f-key 10 :y 5)`)
  // #' Reader-Makro und funcall
  errs += test(env, `(mapcar #'car '((1 2) (3 4)))`)
  errs += test(env, `(funcall #'+ 3 4)`)
  // flet
  errs += test(env, `(flet ((sq (x) (* x x))) (sq 7))`)
  // labels: gegenseitig rekursive Funktionen
  errs += test(env, `(labels ((even? (n) (if (= n 0) t (odd?  (- n 1)))) (odd? (n) (if (= n 0) () (even? (- n 1))))) (even? 10))`)
  // block / return-from
  errs += test(env, `(block outer (return-from outer 42) 0)`)
  errs += test(env, `(block b (+ 1 (return-from b 99)) 0)`)
  // Gemeinsame Test-Helfer (assert=) — eine Quelle für alle Testdateien
  errs += test(env, `(load "tests/test-helpers.lisp")`)
  // Isolierte stdlib-Tests (defstruct, setf, defvar, union, set-difference, find-all)
  errs += test(env, `(load "tests/stdlib-test.lisp")`)
  // defsystem-lite: Systemdefinition, topo-Laden, unload
  errs += test(env, `(load "tests/defsystem-tests.lisp")`)
  // Condition-lite: Typ-Registry, Vererbung, handler-case, lisp-error-Fallback
  errs += test(env, `(load "tests/condition-tests.lisp")`)
  // LOOP-Makro: Iteration, Akkumulation, Bedingungen, Fehlerfälle
  errs += test(env, `(load "tests/loop-tests.lisp")`)
  // Web-Bridge + webserv-Smoke: Server auf Port 0, kein Browser
  errs += test(env, `(load "tests/webbridge.lisp")`)
  // Endlose Rekursion in parfunc bricht sauber ab (Ersatz für Norvig-GPS-Bug 3)
  errs += test(env, `(load "tests/endlosrekursion.lisp")`)
  // Genetischer Algorithmus
  errs += test(env, `(define ga (ga-create 'bit1 5 4 (lambda (g) (apply + g))))`)
  errs += test(env, `(ga-init ga)`)
  errs += test(env, `(ga-calc ga)`)
  errs += test(env, `(ga-result ga)`)
  // Test-Framework: alle registrierten Suiten laufen lassen.
  // Exit-Code des Prozesses = FAILs der Suiten + Fehler der Zeilen oben
  // (0 = grün), CI-tauglich. Bei 255 gedeckelt: Exit-Codes laufen sonst um.
  fails, err := lib.LoadString(`(run-tests)`, env)
  if err != nil {
    printErr(os.Stdout, err)
    errs++
  } else {
    errs += int(fails.Num)
  }
  if errs > 0 {
    fmt.Printf("=== -t: %d Fehler ===\n", errs)
  }
  return min(errs, 255)
}
