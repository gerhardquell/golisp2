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

;; %infinite?: +Inf/-Inf verdoppeln sich zu sich selbst (x≠0 ausgenommen,
;; sonst verdoppelt sich auch 0 zu sich selbst). NaN fällt schon vorher aus
;; (= x (floor x)) raus, weil NaN nie sich selbst gleich ist.
(defun %infinite? (x)
  (and (not (= x 0)) (= x (* 2 x))))

(defun %number-type (x)
  (if (and (= x (floor x)) (not (%infinite? x))) 'integer 'float))

(defun %symbol-type (x)
  (cond ((eq x t)                                                     'boolean)
        ((and (> (string-length (symbol-name x)) 0)
              (equal? (substring (symbol-name x) 0 1) ":"))          'keyword)
        (t                                                            'symbol)))

;; %proper-length: Länge einer echten Liste, -1 bei improper list — schützt
;; vor dem length-Crash (cdr: Liste erwartet) auf z. B. (cons 'punkt 2).
(defun %proper-length (x)
  (cond ((null x)        0)
        ((not (pair? x)) -1)
        (t (let ((n (%proper-length (cdr x))))
             (if (= n -1) -1 (+ n 1))))))

;; %struct-instance?: x ist Struct name — registriert UND Länge passt.
(defun %struct-instance? (x name)
  (let ((entry (assoc name *struct-types*)))
    (if (and entry (pair? x) (eq (car x) name)
             (= (%proper-length x) (+ (cadr entry) 1)))
        t
        ())))

;; Struct-/Condition-Namen, die eingebaute Typen verdecken würden, zählen
;; nicht (Kollision wird bei der Definition gewarnt).
(defun %cons-type (x)
  (cond ((and (%cond? x) (pair? (cdr x)) (symbol? (cadr x))
              (not (%builtin-type? (cadr x))))
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

;; === typep ===========================================================

(defun %bad-spec (spec)
  (error (format nil "typep: ungültige Typangabe ~a" spec)))

;; %type-start: Einstieg in *type-parents* für x. Structs starten bei
;; structure-object, Conditions bei cons.
(defun %type-start (x)
  (let ((k (type-of x)))
    (cond ((%builtin-type? k) k)
          ((%cond? x)         'cons)
          (t                  'structure-object))))

;; Auflösung: eingebaut → Struct → Condition → Fehler (Spec).
(defun %typep-name (x name)
  (cond ((eq name t)                    t)
        ((eq name 'atom)                (not (pair? x)))
        ((%builtin-type? name)          (%builtin-subtype? (%type-start x) name))
        ((assoc name *struct-types*)    (%struct-instance? x name))
        ((assoc name *condition-types*) (if (%cond-type? x name) t ()))
        (t (error (format nil "typep: unbekannter Typ '~a'" name)))))

(defun %valid-bound? (b)
  (or (eq b '*)
      (number? b)
      (and (pair? b) (number? (car b)) (null (cdr b)))))

(defun %lower-ok? (x b)
  (cond ((eq b '*)   t)
        ((number? b) (>= x b))
        (t           (> x (car b)))))

(defun %upper-ok? (x b)
  (cond ((eq b '*)   t)
        ((number? b) (<= x b))
        (t           (< x (car b)))))

;; (integer lo hi) u. ä.: Grenze = Zahl (inklusiv), (zahl) (exklusiv),
;; * oder weggelassen (unbegrenzt). Grenzen werden vor dem Typ geprüft,
;; damit eine kaputte Angabe immer auffällt.
(defun %typep-range (x head args spec)
  (if (or (> (length args) 2) (not (every %valid-bound? args)))
      (%bad-spec spec)
      (if (and (%typep-name x head)
               (%lower-ok? x (if (null args) '* (car args)))
               (%upper-ok? x (if (null (cdr args)) '* (cadr args))))
          t
          ())))

;; (satisfies f): f ist Symbol; (eval f) löst im Root-Env auf und erlaubt
;; per trap den Spec-Fehlertext (funcall 'f hätte eigenen Text). (bound? f) wäre falsch: bound? prüft env lexikalisch
;; und sähe hier die eigenen let*-Locals (x, spec, args, f) statt global
;; aufzulösen — deshalb direkt (eval f) mit trap statt bound?-Vorprüfung.
(defun %typep-satisfies (x spec)
  (let ((args (cdr spec)))
    (if (not (and (pair? args) (symbol? (car args)) (null (cdr args))))
        (%bad-spec spec)
        (let* ((f  (car args))
               (fn (trap (eval f) (lambda (e) ()))))
          (if (not (member (%cell-type fn) '(lambda func)))
              (error (format nil "typep: satisfies: '~a' ist keine Funktion" f))
              (if (funcall fn x) t ()))))))

(defun %one-arg? (args)
  (and (pair? args) (null (cdr args))))

;; %proper-list?: keine dotted list — schützt or/and/member/integer & co.
;; vor rohem car/cdr-Fehler bei z. B. (or . x), (integer 0 . 5).
(defun %proper-list? (x)
  (or (null x) (and (pair? x) (%proper-list? (cdr x)))))

(defun %typep-spec (x spec)
  (cond ((null spec)        ())
        ((symbol? spec)     (%typep-name x spec))
        ((not (pair? spec)) (%bad-spec spec))
        (t
         (let ((head (car spec))
               (args (cdr spec)))
           (if (not (%proper-list? args))
               (%bad-spec spec)
               (cond ((eq head 'or)
                      (any (lambda (s) (%typep-spec x s)) args))
                     ((eq head 'and)
                      (every (lambda (s) (%typep-spec x s)) args))
                     ((eq head 'not)
                      (if (%one-arg? args) (not (%typep-spec x (car args))) (%bad-spec spec)))
                     ((eq head 'member)
                      (any (lambda (v) (eql x v)) args))
                     ((eq head 'eql)
                      (if (%one-arg? args) (eql x (car args)) (%bad-spec spec)))
                     ((eq head 'satisfies)
                      (%typep-satisfies x spec))
                     ((member head '(integer float real rational))
                      (%typep-range x head args spec))
                     (t (%bad-spec spec))))))))

(defun typep (x spec)
  (%typep-spec x spec))

;; === Kollisionen =====================================================

;; %type-name-warnings: (warn …)-Formen für defstruct/define-condition,
;; wenn name einen eingebauten Typ oder die jeweils andere Registry trifft.
;; Wird zur Expansionszeit aufgerufen; kind = 'defstruct | 'define-condition.
(defun %type-name-warnings (kind name)
  (mapcar (lambda (m) (list 'warn m))
          (filter identity
            (list
              (if (%builtin-type? name)
                  (format nil "WARN: ~a ~a: Name ist ein eingebauter Typ → typep/type-of sehen den eingebauten Typ" kind name)
                  ())
              (if (and (eq kind 'defstruct) (assoc name *condition-types*))
                  (format nil "WARN: defstruct ~a: Name ist auch ein Condition-Typ → typep sieht den Struct" name)
                  ())
              (if (and (eq kind 'define-condition) (assoc name *struct-types*))
                  (format nil "WARN: define-condition ~a: Name ist auch ein Struct → typep sieht den Struct" name)
                  ())))))
