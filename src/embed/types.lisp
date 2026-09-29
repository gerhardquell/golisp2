;; ********************************************************************
;; types.lisp – type-of / typep nach CL
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : Claude Haiku 4.5
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260929
;; ********************************************************************
;; Einzige Quelle der Typhierarchie (Spec:
;; docs/superpowers/specs/2026-09-29-typep-design.md).
;; Kernfakten liefert %cell-type (Go, celltype.go), Structs kommen aus
;; *struct-types* (stdlib.lisp), Conditions aus *condition-types*
;; (condition.lisp). Abweichungen von CL: docs/lisp-semantik.md, "Typen".
;; ********************************************************************

;; *type-parents*: (typ (direkte-obertypen …)) im alist-set-Format.
;; atom steht nur zur Namenserkennung drin — typep prüft es als (not cons).
(define *type-parents*
  '((integer          (rational))
    (rational         (real))
    (float            (real))
    (real             (number))
    (number           (t))
    (string           (sequence))
    (keyword          (symbol))
    (boolean          (symbol))
    (null             (boolean list))
    (symbol           (t))
    (cons             (list))
    (list             (sequence))
    (sequence         (t))
    (compiled-function (function))
    (function         (t))
    (macro            (t))
    (hash-table       (t))
    (structure-object (cons))
    (atom             (t))
    (t                ())))

(defun %builtin-type? (name)
  (if (assoc name *type-parents*) t ()))

(defun %builtin-subtype? (sub super)
  (cond ((eq sub super) t)
        ((eq sub t)     ())
        (t (any (lambda (p) (%builtin-subtype? p super))
                (alist-get sub *type-parents*)))))

;; === type-of =========================================================

(defun %number-type (x)
  (if (= x (floor x)) 'integer 'float))

(defun %symbol-type (x)
  (cond ((eq x t)                                        'boolean)
        ((equal? (substring (symbol-name x) 0 1) ":")    'keyword)
        (t                                               'symbol)))

;; %struct-instance?: x ist Struct name — registriert UND Länge passt.
(defun %struct-instance? (x name)
  (let ((entry (assoc name *struct-types*)))
    (if (and entry (pair? x) (eq (car x) name)
             (= (length x) (+ (cadr entry) 1)))
        t
        ())))

;; Struct-/Condition-Namen, die eingebaute Typen verdecken würden, zählen
;; nicht (Kollision wird bei der Definition gewarnt).
(defun %cons-type (x)
  (cond ((and (%cond? x) (not (%builtin-type? (cadr x))))
         (cadr x))
        ((and (symbol? (car x)) (not (%builtin-type? (car x)))
              (%struct-instance? x (car x)))
         (car x))
        (t 'cons)))

(defun type-of (x)
  (let ((k (%cell-type x)))
    (cond ((eq k 'number) (%number-type x))
          ((eq k 'symbol) (%symbol-type x))
          ((eq k 'lambda) 'function)
          ((eq k 'func)   'compiled-function)
          ((eq k 'cons)   (%cons-type x))
          (t              k))))
