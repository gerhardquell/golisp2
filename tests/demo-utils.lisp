; demo-utils.lisp – Gemeinsame Hilfsfunktionen für GoLisp Demos
; Autor: Claude
; CoAutor: Gerhard Quell
; Erstellt: 20260226

;; Zufallszahlen-Setup (einfacher linearer Kongruenzgenerator)
;; Verwendet kleine Zahlen für effiziente mod-Berechnung
(define random-seed 123)
(define random-modulus 997)  ; Kleine Primzahl
(define random-multiplier 31)
(define random-increment 17)

(defun random-next ()
  "Aktualisiert random-seed und gibt neuen Wert zurück"
  ;; Berechne (a * seed + c) mod m durch wiederholte Addition
  (let ((product (* random-multiplier random-seed)))
    (set! random-seed (fast-mod (+ product random-increment) random-modulus))
    random-seed))

(defun fast-mod (a b)
  "Effizientes mod für Zahlen bis ca. 100000"
  (if (< a b)
      a
      (if (> a (* b 10))
          (fast-mod (- a (* b 10)) b)
          (fast-mod (- a b) b))))

(defun random-int (min max)
  "Gibt eine zufällige Ganzzahl zwischen min (inklusiv) und max (exklusiv)"
  (random-next)
  (+ min (fast-mod random-seed (- max min))))

(defun random-choice (lst)
  "Wählt ein zufälliges Element aus der Liste"
  (nth (random-int 0 (length lst)) lst))

;; Zufällige Ausdrücke generieren
(define *default-terminals* '(x y z 1 2 3 4 5 0 -1 -2))
(define *default-functions* '(+ - * / abs min max))

(defun random-expr (depth)
  "Generiert einen zufälligen Lisp-Ausdruck mit maximaler Tiefe"
  (if (or (<= depth 0) (< (random) 0.3))
      (random-choice *default-terminals*)
      (let ((op (random-choice *default-functions*)))
        (list op
              (random-expr (- depth 1))
              (random-expr (- depth 1))))))

(defun random-expr-with-vars (depth vars constants)
  "Generiert zufälligen Ausdruck mit gegebenen Variablen und Konstanten"
  (if (or (<= depth 0) (< (random) 0.3))
      (if (< (random) 0.5)
          (random-choice vars)
          (random-choice constants))
      (let ((op (random-choice '(+ - *))))
        (if (eq op '-)
            (list op
                  (random-expr-with-vars (- depth 1) vars constants)
                  (random-expr-with-vars (- depth 1) vars constants))
            (list op
                  (random-expr-with-vars (- depth 1) vars constants)
                  (random-expr-with-vars (- depth 1) vars constants))))))

;; Ausdrucks-Analyse
(defun expr-size (expr)
  "Zählt die Knoten im AST"
  (if (atom expr)
      1
      (+ 1 (sum (mapcar expr-size expr)))))

(defun expr-depth (expr)
  "Maximale Verschachtelungstiefe"
  (if (atom expr)
      0
      (+ 1 (max-list (mapcar expr-depth expr)))))

(defun max-list (lst)
  "Findet das Maximum in einer Liste"
  (if (null? (cdr lst))
      (car lst)
      (let ((rest-max (max-list (cdr lst))))
        (if (> (car lst) rest-max)
            (car lst)
            rest-max))))

(defun sum (lst)
  "Summiert alle Elemente der Liste"
  (if (null? lst)
      0
      (+ (car lst) (sum (cdr lst)))))

;; Zeitmessung
(defun timed-eval (expr timeout-ms)
  "Wertet expr aus mit Zeitlimit (vereinfacht – kein echtes Timeout in GoLisp)"
  (cons 'ok (eval expr)))

;; Liste manipulieren
(defun range (start end)
  "Erzeugt Liste [start, end)"
  (if (>= start end)
      '()
      (cons start (range (+ start 1) end))))

;; Ausgabe-Hilfen
(defun println-list (lst)
  "Gibt eine Liste von Werten aus, jeweils in einer Zeile"
  (if (null? lst)
      nil
      (begin
        (println (car lst))
        (println-list (cdr lst)))))

(defun print-header (title)
  "Gibt einen formatierten Header aus"
  (println "")
  (println (string-append "=== " title " ==="))
  (println ""))

;; Test-Hilfsfunktionen
(defun assert-equal (expected actual msg)
  "Prüft Gleichheit und gibt Fehler aus wenn ungleich"
  (if (equal? expected actual)
      (print ".")
      (begin
        (println "")
        (println (string-append "FAIL: " msg))
        (println (string-append "  Expected: " (number->string expected)))
        (println (string-append "  Actual: " (number->string actual))))))

;; Sortieren (einfacher Quicksort)
(defun qsort (lst key-fn)
  "Sortiert Liste nach Schlüsselfunktion"
  (if (null? lst)
      '()
      (let ((pivot (car lst))
            (rest (cdr lst)))
        (append
         (qsort (filter (lambda (x) (< (key-fn x) (key-fn pivot))) rest) key-fn)
         (list pivot)
         (qsort (filter (lambda (x) (>= (key-fn x) (key-fn pivot))) rest) key-fn)))))

(println "demo-utils.lisp geladen – Hilfsfunktionen verfügbar")
