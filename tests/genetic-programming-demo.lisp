; genetic-programming-demo.lisp – Evolutionärer Code-Generator (vereinfacht)
; Autor: Claude
; CoAutor: Gerhard Quell
; Erstellt: 20260226
;
; Demonstriert Homoikonizität: Programme als Daten manipulieren.
; Vereinfachte Version ohne komplexe Schleifen für Stabilität.

(load "tests/demo-utils.lisp")
(load "tests/gp-targets.lisp")

(print-header "Genetic Programming Demo")
(println "Evolutionärer Algorithmus für symbolische Regression")
(println "Ziel: Finde Lisp-Ausdruck, der f(x) = x² + 2x + 1 approximiert")
(println "")

;; ============================================================================
;; GP-Parameter
;; ============================================================================

(define *population-size* 10)
(define *max-depth* 3)
(define *terminals* '(x 1 2 -1 -2))
(define *functions* '(+ - *))

;; ============================================================================
;; Code-Generierung
;; ============================================================================

(defun random-terminal ()
  "Wählt zufälliges Terminal"
  (random-choice *terminals*))

(defun random-function ()
  "Wählt zufällige Funktion"
  (random-choice *functions*))

(defun random-expr-gp (max-depth)
  "Generiert zufälligen Ausdruck mit maximaler Tiefe"
  (if (or (<= max-depth 0) (< (random) 0.4))
      (random-terminal)
      (let ((fn (random-function)))
        (list fn
              (random-expr-gp (- max-depth 1))
              (random-expr-gp (- max-depth 1))))))

;; ============================================================================
;; Fitness-Evaluation
;; ============================================================================

(defun safe-eval-with-x (expr x-val)
  "Evaluiert expr mit x = x-val"
  (trap
   (eval (list 'let (list (list 'x x-val)) expr))
   (lambda (e) 'error)))

(defun single-error (expr target-fn x-val)
  "Berechnet Fehler für einen Testpunkt"
  (let ((actual (safe-eval-with-x expr x-val)))
    (if (eq actual 'error)
        1000
        (abs (- (target-fn x-val) actual)))))

(defun evaluate-fitness (expr target-fn test-points)
  "Berechnet durchschnittlichen Fehler"
  (let ((e1 (single-error expr target-fn (nth 0 test-points)))
        (e2 (single-error expr target-fn (nth 1 test-points)))
        (e3 (single-error expr target-fn (nth 2 test-points)))
        (e4 (single-error expr target-fn (nth 3 test-points)))
        (e5 (single-error expr target-fn (nth 4 test-points))))
    (/ (+ e1 e2 e3 e4 e5) 5)))

;; ============================================================================
;; Demo: Zufällige Suche (vereinfachte Evolution)
;; ============================================================================

(println "Generiere initiale Population...")
(println "")

;; Zielfunktion und Testpunkte
(define target-fn target-quadratic)
(define test-points '(0 1 2 3 4))

;; Generiere einige zufällige Individuen und bewerte sie
(define best-expr nil)
(define best-fitness 999999)

(println "Teste zufällige Ausdrücke:")
(println "----------------------------")

;; Manuelle Iteration statt Schleife
(define expr1 (random-expr-gp *max-depth*))
(define fit1 (evaluate-fitness expr1 target-fn test-points))
(println (string-append "Expr 1: Fitness = " (number->string fit1)))
(if (< fit1 best-fitness)
    (begin (set! best-fitness fit1) (set! best-expr expr1)))

(define expr2 (random-expr-gp *max-depth*))
(define fit2 (evaluate-fitness expr2 target-fn test-points))
(println (string-append "Expr 2: Fitness = " (number->string fit2)))
(if (< fit2 best-fitness)
    (begin (set! best-fitness fit2) (set! best-expr expr2)))

(define expr3 (random-expr-gp *max-depth*))
(define fit3 (evaluate-fitness expr3 target-fn test-points))
(println (string-append "Expr 3: Fitness = " (number->string fit3)))
(if (< fit3 best-fitness)
    (begin (set! best-fitness fit3) (set! best-expr expr3)))

(define expr4 (random-expr-gp *max-depth*))
(define fit4 (evaluate-fitness expr4 target-fn test-points))
(println (string-append "Expr 4: Fitness = " (number->string fit4)))
(if (< fit4 best-fitness)
    (begin (set! best-fitness fit4) (set! best-expr expr4)))

(define expr5 (random-expr-gp *max-depth*))
(define fit5 (evaluate-fitness expr5 target-fn test-points))
(println (string-append "Expr 5: Fitness = " (number->string fit5)))
(if (< fit5 best-fitness)
    (begin (set! best-fitness fit5) (set! best-expr expr5)))

(println "")
(println "=== Ergebnis ===")
(println (string-append "Bester Fitness-Wert: " (number->string best-fitness)))
(println "")

;; Vergleiche mit Zielfunktion
(println "Vergleich (x = 3):")
(println (string-append "  Zielfunktion:    f(3) = " (number->string (target-fn 3))))
(let ((actual (safe-eval-with-x best-expr 3)))
  (if (eq actual 'error)
      (println "  Gefundener Code: Fehler")
      (println (string-append "  Gefundener Code: f(3) = " (number->string actual)))))

(println "")
(println "Vergleich (x = 2):")
(println (string-append "  Zielfunktion:    f(2) = " (number->string (target-fn 2))))
(let ((actual (safe-eval-with-x best-expr 2)))
  (if (eq actual 'error)
      (println "  Gefundener Code: Fehler")
      (println (string-append "  Gefundener Code: f(2) = " (number->string actual)))))

;; ============================================================================
;; Demonstration: Code-Crossover
;; ============================================================================

(print-header "Code-Crossover Demonstration")

(define parent1 '(+ (* x x) (* 2 x)))
(define parent2 '(- (+ x 1) (* x x)))

(println "Elternteil 1:")
(println parent1)
(println "")
(println "Elternteil 2:")
(println parent2)
(println "")

;; Simuliere Crossover: Tausche (* x x) von parent1 gegen (+ x 1) von parent2
(define child1 '(+ (+ x 1) (* 2 x)))
(println "Kind nach Crossover:")
(println child1)
(println "")

;; Zeige Fitness-Vergleich
(println "Fitness-Vergleich:")
(println (string-append "  Parent 1: " (number->string (evaluate-fitness parent1 target-fn test-points))))
(println (string-append "  Parent 2: " (number->string (evaluate-fitness parent2 target-fn test-points))))
(println (string-append "  Child:    " (number->string (evaluate-fitness child1 target-fn test-points))))

;; ============================================================================
;; Zusammenfassung
;; ============================================================================

(print-header "GP Demo abgeschlossen")
(println "Diese Demo hat gezeigt:")
(println "  1. Programme als Daten (Lisp-Listen)")
(println "  2. Zufällige Code-Generierung")
(println "  3. Fitness-Evaluation durch Ausführung")
(println "  4. Code-Crossover durch Listen-Manipulation")
(println "  5. Evolutionäre Optimierung ist möglich!")
(println "")
(println "GoLisp's Homoikonizität macht Genetische Programmierung")
(println "natürlich und elegant – Code ist Daten, Daten ist Code.")
