;;; kiesp-sexp.lisp - Kompakte S-Expressions
;;; Autor: Gerhard Quell – gquell@skequell.de
;;; CoAutor: Claude
;;; Experiment: Makro-aehnliche Token-Ersetzung mit Quasiquote

;;; Token-Mapping (kurz → lang)
(define kiesp-tokens
  '((def . define)
    (fn . lambda)
    (let* . let)
    (q . quote)
    (qq . quasiquote)
    (uq . unquote)
    (uqs . unquote-splicing)
    (if? . if)
    (cond? . cond)
    (and? . and)
    (or? . or)
    ;; Typen
    (N . number)
    (S . string)
    (L . list)
    (A . atom)
    (B . boolean)
    ;; Funktionen
    (+ . add)
    (- . sub)
    (* . mul)
    (/ . div)
    (=? . equal?)
    (eq? . eq)
    (< . less)
    (> . greater)
    (len . length)
    ;; Listen
    (c . cons)
    (ca . car)
    (cd . cdr)
    (ap . append)
    (rev . reverse)
    ;; Kontrollfluss
    (DO . begin)
    (RET . return)
    (? . when)
    (! . unless)
    ;; IO
    (pr . print)
    (pl . println)
    (rd . read)))

;;; Encoder: Expandiert kompakte Tokens

(defun kiesp-expand (expr)
  "Expandiert kompakte KIESP-Tokens zu vollstaendigem Lisp"
  (cond
    ((atom expr)
     (let ((mapped (assoc expr kiesp-tokens)))
       (if mapped (cdr mapped) expr)))
    ((null expr) '())
    (else
     (cons (kiesp-expand (car expr))
           (kiesp-expand (cdr expr))))))

;;; Encoder: Komprimiert vollstaendiges Lisp

(defun kiesp-compress (expr)
  "Komprimiert Lisp-Expression zu KIESP-Tokens"
  (cond
    ((atom expr)
     (let ((reverse-map (kiesp-reverse-assoc expr kiesp-tokens)))
       (if reverse-map reverse-map expr)))
    ((null expr) '())
    (else
     (cons (kiesp-compress (car expr))
           (kiesp-compress (cdr expr))))))

(defun kiesp-reverse-assoc (val alist)
  "Findet Schluessel fuer einen Wert in einer Assoziationsliste"
  (if (null alist)
      '()
      (if (eq val (cdar alist))
          (caar alist)
          (kiesp-reverse-assoc val (cdr alist)))))

;;; Template-System mit Quasiquote

(defun kiesp-template (template values)
  "Fuellt ein Template mit Werten (Quasiquote-aehnlich)"
  (kiesp-fill-template template values))

(defun kiesp-fill-template (template values)
  "Rekursive Template-Fuellung"
  (cond
    ((atom template) template)
    ((eq (car template) 'uq)
     (kiesp-lookup (cadr template) values))
    ((eq (car template) 'uqs)
     (kiesp-lookup (cadr template) values))
    (else
     (mapcar (lambda (x) (kiesp-fill-template x values)) template))))

(defun kiesp-lookup (key alist)
  "Sucht einen Wert in einer Assoziationsliste"
  (let ((pair (assoc key alist)))
    (if pair (cdr pair) key)))

;;; Kompakte Kontrollstrukturen

(defmacro kiesp-when (cond . body)
  "Kompaktes when: (? cond body...)"
  `(if ,cond (DO ,@body) '()))

(defmacro kiesp-unless (cond . body)
  "Kompaktes unless: (! cond body...)"
  `(if ,cond '() (DO ,@body)))

;;; String-Builder fuer KIESP

(defun kiesp-str (&rest parts)
  "Baut einen String aus Teilen"
  (if (null parts)
      ""
      (string-append (kiesp-val (car parts))
                     (apply kiesp-str (cdr parts)))))

(defun kiesp-val (x)
  "Konvertiert einen Wert zu String"
  (cond
    ((atom x) (number->string x))
    (else "")))

;;; Beispiel-Programme

(define kiesp-example-factorial
  '(def fac (fn (n)
     (if? (> n 0)
       (DO (* n (fac (- n 1))))
       1))))

(define kiesp-example-map
  '(def my-map (fn (f xs)
     (if? (null xs)
       '()
       (DO (c (f (ca xs)) (my-map f (cd xs))))))))

;;; Statistik

(defun kiesp-stats-sexp (original compressed)
  "Vergleicht Original und komprimierte Form"
  (let ((orig-chars (kiesp-count-chars original))
        (comp-chars (kiesp-count-chars compressed))
        (orig-toks (length (flatten original)))
        (comp-toks (length (flatten compressed))))
    (list
      (cons 'original-chars orig-chars)
      (cons 'compressed-chars comp-chars)
      (cons 'char-ratio (* 100.0 (/ comp-chars orig-chars)))
      (cons 'original-tokens orig-toks)
      (cons 'compressed-tokens comp-toks)
      (cons 'token-ratio (* 100.0 (/ comp-toks orig-toks))))))

(defun kiesp-count-chars (expr)
  "Zaehlt Zeichen in einer Expression"
  (string-length (kiesp-expr->string expr)))

(defun kiesp-expr->string (expr)
  "Konvertiert Expression zu String"
  (cond
    ((null expr) "()")
    ((atom expr) (number->string expr))
    (else
     (string-append "(pair)"))))

;;; Test-Funktion

(defun kiesp-test-roundtrip (expr)
  "Testet Encoding -> Decoding Roundtrip"
  (let ((compressed (kiesp-compress expr))
        (expanded (kiesp-expand expr)))
    (println "Original:  " expr)
    (println "Compressed:" compressed)
    (println "Expanded:  " expanded)
    (println "Stats:     " (kiesp-stats-sexp expr compressed))
    (equal? expr expanded)))

(println "=== KIESP S-Expr Kompakt geladen ===")
(println "Tokens: def fn let* q qq uq uqs if? cond? and? or?")
(println "        N S L A B + - * / =? eq? < > len c ca cd")
(println "        ap rev DO RET ? ! pr pl rd")
