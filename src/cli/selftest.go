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

  "golisp2/src/lib"
)

func test(env *lib.Env, s string) {
  cell, _ := lib.Read(s)
  result, err := lib.Eval(cell, env)
  if err != nil {
    fmt.Printf("ERR: %v\n", err)
    return
  }
  fmt.Printf("%-45s => %s\n", s, result)
}

func runTests(env *lib.Env) {
  fmt.Println("=== GoLisp Test ===")
  fmt.Println()
  test(env, "(+ 1 2)")
  test(env, "(defun fak (n) (if (= n 0) 1 (* n (fak (- n 1)))))")
  test(env, "(fak 6)")
  test(env, "(mapcar (lambda (x) (* x x)) (list 1 2 3 4 5))")
  test(env, "(parfunc r (* 6 7) (+ 100 23))")
  test(env, "r")
  test(env, "(atom (gensym))")
  test(env, "(= (gensym) (gensym))")
  test(env, `(catch (error "oops") (lambda (e) (string-append "caught: " e)))`)
  test(env, `(catch (+ 1 2) (lambda (e) "fehler"))`)
  test(env, `(defun add-and-show (x y) (define s (+ x y)) s)`)
  test(env, `(add-and-show 3 4)`)
  test(env, `((lambda (x) (define d (* x 2)) d) 5)`)
  test(env, `(>= 5 3)`)
  test(env, `(>= 3 3)`)
  test(env, `(>= 2 3)`)
  test(env, `(<= 2 3)`)
  test(env, `(<= 3 3)`)
  test(env, `(<= 5 3)`)
  // while
  test(env, `(define w 0)`)
  test(env, `(while (< w 3) (set! w (+ w 1)))`)
  test(env, `w`)
  // do: (do ((i 0 (+ i 1)) (s 0 (+ s i))) ((= i 5) s))  → 0+1+2+3+4 = 10
  test(env, `(do ((i 0 (+ i 1)) (s 0 (+ s i))) ((= i 5) s))`)
  // TCO: tiefe Rekursion ohne Stack-Overflow
  test(env, `(defun sum-acc (n acc) (if (= n 0) acc (sum-acc (- n 1) (+ acc n))))`)
  test(env, `(sum-acc 1000000 0)`)
  test(env, `(defun even? (n) (if (= n 0) t (odd?  (- n 1))))`)
  test(env, `(defun odd?  (n) (if (= n 0) () (even? (- n 1))))`)
  test(env, `(even? 1000000)`)
  // equal?
  test(env, `(equal? 1 1)`)
  test(env, `(equal? 1 2)`)
  test(env, `(equal? "a" "a")`)
  test(env, `(equal? (list 1 2) (list 1 2))`)
  test(env, `(equal? (list 1 2) (list 1 3))`)
  test(env, `(equal? (list 1 (list 2 3)) (list 1 (list 2 3)))`)
  test(env, `(equal? () ())`)
  // &optional / &key Parameter
  test(env, `(defun greet (name &optional (greeting "Hallo")) (string-append greeting " " name))`)
  test(env, `(greet "Gerhard")`)
  test(env, `(greet "Gerhard" "Hi")`)
  test(env, `(defun f-key (x &key (y 0)) (+ x y))`)
  test(env, `(f-key 10)`)
  test(env, `(f-key 10 :y 5)`)
  // #' Reader-Makro und funcall
  test(env, `(mapcar #'car '((1 2) (3 4)))`)
  test(env, `(funcall #'+ 3 4)`)
  // flet
  test(env, `(flet ((sq (x) (* x x))) (sq 7))`)
  // labels: gegenseitig rekursive Funktionen
  test(env, `(labels ((even? (n) (if (= n 0) t (odd?  (- n 1)))) (odd? (n) (if (= n 0) () (even? (- n 1))))) (even? 10))`)
  // block / return-from
  test(env, `(block outer (return-from outer 42) 0)`)
  test(env, `(block b (+ 1 (return-from b 99)) 0)`)
  // Gemeinsame Test-Helfer (assert=) — eine Quelle für alle Testdateien
  test(env, `(load "tests/test-helpers.lisp")`)
  // Isolierte stdlib-Tests (defstruct, setf, defvar, union, set-difference, find-all)
  test(env, `(load "tests/stdlib-test.lisp")`)
  // defsystem-lite: Systemdefinition, topo-Laden, unload
  test(env, `(load "tests/defsystem-tests.lisp")`)
  // Condition-lite: Typ-Registry, Vererbung, handler-case, lisp-error-Fallback
  test(env, `(load "tests/condition-tests.lisp")`)
  // LOOP-Makro: Iteration, Akkumulation, Bedingungen, Fehlerfälle
  test(env, `(load "tests/loop-tests.lisp")`)
  // Norvigs drei bekannte GPS-Fehler (Timeout für rekursiven Fall)
  test(env, `(load "pn-gps1/gps-norvig-bugs.lisp")`)
  // GPS Version 2: state-passing, goroutine-tauglich
  test(env, `(load "pn-gps1/gps2-tests.lisp")`)
  // Genetischer Algorithmus
  test(env, `(define ga (ga-create 'bit1 5 4 (lambda (g) (apply + g))))`)
  test(env, `(ga-init ga)`)
  test(env, `(ga-calc ga)`)
  test(env, `(ga-result ga)`)
  // Test-Framework: alle registrierten Suiten laufen lassen.
  // Exit-Code des Prozesses = Anzahl FAILs (0 = grün), CI-tauglich.
  test(env, `(exit (run-tests))`)
}
