;; ********************************************************************
;; tests/loop-tests.lisp — Tests für das LOOP-Makro (embed/loop.lisp)
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k3
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260902
;; ********************************************************************
;; Läuft über tests/test-framework.lisp: Tests registrieren sich,
;; Ausführung via (run-tests) — Fehler brechen den Load nicht ab.
;; ********************************************************************

(load "tests/test-framework.lisp")

;; === Iteration: for from / to / below / by / downto ====================

(defsuite 'loop-iteration)

(deftest loop-from-to :suite 'loop-iteration
  (is (equal? '(1 2 3 4 5) (loop for i from 1 to 5 collect i)))
  (is (equal? '(1)         (loop for i from 1 to 1 collect i)))
  (is (equal? '()          (loop for i from 5 to 1 collect i))))

(deftest loop-from-below :suite 'loop-iteration
  (is (equal? '(0 1 2 3)   (loop for i from 0 below 4 collect i)))
  (is (equal? '()          (loop for i from 4 below 4 collect i))))

(deftest loop-by :suite 'loop-iteration
  (is (equal? '(0 2 4 6 8) (loop for i from 0 to 8 by 2 collect i)))
  (is (equal? '(1 3 5)     (loop for i from 1 below 6 by 2 collect i))))

(deftest loop-downto :suite 'loop-iteration
  (is (equal? '(5 4 3 2 1) (loop for i from 5 downto 1 collect i)))
  (is (equal? '(10 7 4 1) (loop for i from 10 downto 1 by 3 collect i))))

(deftest loop-for-in :suite 'loop-iteration
  (is (equal? '(a b c)     (loop for x in '(a b c) collect x)))
  (is (equal? '(2 4 6)     (loop for x in '(1 2 3) collect (* 2 x))))
  (is (equal? '()          (loop for x in '() collect x))))

(deftest loop-repeat :suite 'loop-iteration
  (is (equal? '(x x x)     (loop repeat 3 collect 'x)))
  (is (equal? '()          (loop repeat 0 collect 'x))))

;; === Akkumulation ======================================================

(defsuite 'loop-akkumulation)

(deftest loop-collect-append :suite 'loop-akkumulation
  (is (equal? '(1 2 3)     (loop for i from 1 to 3 collect i)))
  (is (equal? '(1 1 2 4)   (loop for i from 1 to 2 append (list i (* i i))))))

(deftest loop-sum-count :suite 'loop-akkumulation
  (is (equal? 15           (loop for i from 1 to 5 sum i)))
  (is (equal? 5            (loop for i from 1 to 5 count i)))
  (is (equal? 3            (loop for x in '(1 () 2 () 3) count x))))

(deftest loop-maximize-minimize :suite 'loop-akkumulation
  (is (equal? 9            (loop for x in '(3 9 1 7) maximize x)))
  (is (equal? 1            (loop for x in '(3 9 1 7) minimize x)))
  ;; negative Werte: Init darf nicht 0 sein
  (is (equal? -1           (loop for x in '(-5 -1 -9) maximize x))))

;; === Bedingte Klauseln =================================================

(defsuite 'loop-bedingt)

(deftest loop-when :suite 'loop-bedingt
  (is (equal? '(2 4 6)     (loop for i from 1 to 6 when (= 0 (mod i 2)) collect i)))
  (is (equal? 12           (loop for i from 1 to 6 when (= 0 (mod i 2)) sum i))))

(deftest loop-unless :suite 'loop-bedingt
  (is (equal? '(1 3 5)     (loop for i from 1 to 6 unless (= 0 (mod i 2)) collect i))))

(deftest loop-while-until :suite 'loop-bedingt
  (is (equal? '(1 2 3)     (loop for i from 1 to 10 while (<= i 3) collect i)))
  (is (equal? '(1 2 3)     (loop for i from 1 to 10 until (> i 3) collect i))))

;; === do / finally ======================================================

(defsuite 'loop-body)

(deftest loop-do-finally :suite 'loop-body
  ;; eval: define im deftest-Rumpf wäre lokal gebunden (Lambda-Scope)
  (eval '(define %loop-log '()))
  (is (equal? '()
              (loop for i from 1 to 3
                    do (set! %loop-log (cons i %loop-log))
                    finally (set! %loop-log (cons 'fertig %loop-log)))))
  (is (equal? '(fertig 3 2 1) %loop-log)))

(deftest loop-kein-akkumulator :suite 'loop-body
  ;; ohne Akkumulationsklausel: Ergebnis ()
  (is (equal? '()          (loop repeat 2 do 42))))

;; === Kombinationen =====================================================

(defsuite 'loop-kombi)

(deftest loop-kombi-for-in-when :suite 'loop-kombi
  (is (equal? '(b d)       (loop for x in '(a b c d)
                                 for i from 1
                                 when (= 0 (mod i 2)) collect x))))

(deftest loop-kombi-until-collect :suite 'loop-kombi
  (is (equal? '(10 9 8)    (loop for i from 10 downto 1
                                 until (< i 8) collect i))))

;; === Fehlerfälle =======================================================

(defsuite 'loop-fehler)

(deftest loop-fehler-familienmix :suite 'loop-fehler
  ;; collect (list) + sum (num0): Expansionsfehler
  (is (equal? 'fehler
              (handler-case (begin (macroexpand '(loop for i from 1 to 3 collect i sum i))
                                   'kein-fehler)
                (lisp-error (e) 'fehler)))))

(deftest loop-fehler-unbekannte-klausel :suite 'loop-fehler
  (is (equal? 'fehler
              (handler-case (begin (macroexpand '(loop for i from 1 to 3 sammle i))
                                   'kein-fehler)
                (lisp-error (e) 'fehler)))))
