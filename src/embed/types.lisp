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
    (foreign          (t))
    (structure-object (cons))
    (atom             (t))
    (t                ())))

;; *deftypes*: (name expander) im alist-set-Format; expander liefert die
;; Typangabe (deftype). defvar: Reload darf Definitionen nicht löschen.
(defvar *deftypes* '())

(defun %deftype-expand (name args)
  (apply (alist-get name *deftypes*) args))

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
        ((assoc name *deftypes*)        (%typep-spec x (%deftype-expand name '())))
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
                     ((assoc head *deftypes*)
                      (%typep-spec x (%deftype-expand head args)))
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
                  ())
              (if (assoc name *deftypes*)
                  (format nil "WARN: ~a ~a: Name ist auch ein deftype → typep sieht ~a" kind name
                          (if (eq kind 'defstruct) "den Struct" "die Condition"))
                  ())))))

;; === typecase / etypecase ===========================================

;; Klausel (typ form…) → cond-Klausel; t/otherwise ist der Sonst-Zweig.
;; Leerer Rumpf → () (cond gäbe sonst den Testwert zurück).
(defun %typecase-clause (g clause)
  (let ((body (if (null (cdr clause)) '(()) (cdr clause))))
    (if (member (car clause) '(t otherwise))
        `(t ,@body)
        `((typep ,g ',(car clause)) ,@body))))

(defmacro typecase (key &rest clauses)
  (let ((g (gensym)))
    `(let ((,g ,key))
       (cond ,@(mapcar (lambda (c) (%typecase-clause g c)) clauses)))))

;; etypecase: wie typecase, ohne Treffer ein Fehler mit allen Typen.
(defmacro etypecase (key &rest clauses)
  (let ((g (gensym)))
    `(let ((,g ,key))
       (cond ,@(mapcar (lambda (c) (%typecase-clause g c)) clauses)
             (t (error (format nil "etypecase: ~s passt zu keinem Typ ~s"
                               ,g ',(mapcar car clauses))))))))

;; === check-type ======================================================

;; (check-type place spec [beschreibung]) → () oder Fehler. Abweichung von
;; CL: einfacher error statt type-error, kein store-value-Restart.
(defmacro check-type (place spec &rest doc)
  (let ((g (gensym)))
    `(let ((,g ,place))
       (if (typep ,g ',spec)
           ()
           (error (format nil "check-type: ~a ist ~s, nicht ~a" ',place ,g
                          ,(if (null doc) (format nil "vom Typ ~a" spec) (car doc))))))))

;; === deftype =========================================================

;; Ein deftype auf einen eingebauten, Struct- oder Condition-Namen würde
;; von typep nie erreicht (Auflösungsreihenfolge) — laut statt tot.
(defun %deftype-check (name)
  (cond ((not (symbol? name))
         (error (format nil "deftype: Name muss Symbol sein, nicht ~s" name)))
        ((%builtin-type? name)
         (error (format nil "deftype: ~a ist ein eingebauter Typ" name)))
        ((assoc name *struct-types*)
         (error (format nil "deftype: ~a ist ein Struct" name)))
        ((assoc name *condition-types*)
         (error (format nil "deftype: ~a ist ein Condition-Typ" name)))
        (t name)))

;; (deftype name lambda-liste rumpf…) → name. Der Rumpf liefert die
;; Typangabe; (typep x '(name args…)) ruft ihn mit args. Abweichung von CL:
;; &optional ohne Default ist (), nicht * — (&optional (n '*)) schreiben.
(defmacro deftype (name lambda-list &rest body)
  `(begin
     (%deftype-check ',name)
     (set! *deftypes* (alist-set ',name (lambda ,lambda-list ,@body) *deftypes*))
     ',name))

;; === subtypep ========================================================

;; Intern liefert jede Prüfung (sub? sicher?) als Liste; subtypep gibt sie
;; als zwei Werte zurück. (() ()) = nicht entscheidbar (erlaubt laut CL).

(defun %st-ja/nein (b) (if b '(t t) '(() t)))

;; alle Teilergebnisse ja → ja; eines sicher nein → nein; sonst unklar
(defun %st-all (results)
  (cond ((every car results)                                   '(t t))
        ((any (lambda (r) (and (not (car r)) (cadr r))) results) '(() t))
        (t                                                     '(() ()))))

(defun %st-some (results)
  (if (any car results) '(t t) '(() ())))

(defun %st-deftype? (s)
  (if (pair? s)
      (and (symbol? (car s)) (assoc (car s) *deftypes*))
      (and (symbol? s) (assoc s *deftypes*))))

(defun %st-expand (s)
  (if (pair? s) (%deftype-expand (car s) (cdr s)) (%deftype-expand s '())))

(defun %st-name-kind (name)
  (cond ((%builtin-type? name)            'builtin)
        ((assoc name *struct-types*)      'struct)
        ((assoc name *condition-types*)   'condition)
        (t (error (format nil "subtypep: unbekannter Typ '~a'" name)))))

;; eingebauter Typ ohne gemeinsame Elemente mit cons → Untertyp von atom
(defun %st-cons-free? (name)
  (and (not (%builtin-subtype? 'cons name)) (not (%builtin-subtype? name 'cons))))

;; Structs sind Listen (structure-object → cons), Conditions auch (cons).
(defun %st-names (a b)
  (let ((ka (%st-name-kind a))
        (kb (%st-name-kind b)))
    (cond ((eq b 'atom)          (%st-ja/nein (and (eq ka 'builtin) (%st-cons-free? a))))
          ((eq a 'atom)          (%st-ja/nein (eq b t)))
          ((eq ka 'builtin)      (%st-ja/nein (and (eq kb 'builtin) (%builtin-subtype? a b))))
          ((eq ka 'struct)       (%st-ja/nein (and (eq kb 'builtin) (%builtin-subtype? 'structure-object b))))
          ((eq kb 'condition)    (%st-ja/nein (%cond-subtype? a b)))
          (t                     (%st-ja/nein (and (eq kb 'builtin) (%builtin-subtype? 'cons b)))))))

(defun %st-range? (s)
  (and (pair? s) (member (car s) '(integer float real rational))))

;; Grenze → * oder (wert exklusiv?). Bei Ganzzahlen wird (n) zu n±1 inklusiv.
(defun %st-bound (b lower int?)
  (cond ((eq b '*)   '*)
        ((number? b) (list b ()))
        (int?        (list (if lower (+ (car b) 1) (- (car b) 1)) ()))
        (t           (list (car b) t))))

(defun %st-range-bounds (s int?)
  (let ((args (cdr s)))
    (if (or (not (%proper-list? args)) (> (length args) 2) (not (every %valid-bound? args)))
        (%bad-spec s)
        (list (%st-bound (if (null args) '* (car args)) t int?)
              (%st-bound (if (null (cdr args)) '* (cadr args)) () int?)))))

(defun %st-bound-in? (b1 b2 lower)
  (cond ((eq b2 '*) t)
        ((eq b1 '*) ())
        (t (let ((v1 (car b1)) (v2 (car b2)))
             (or (if lower (> v1 v2) (< v1 v2))
                 (and (= v1 v2) (or (not (cadr b2)) (cadr b1))))))))

;; r1 ⊆ r2 als Zahlbereich (Typköpfe prüft der Aufrufer)
(defun %st-range-in? (r1 r2 int?)
  (let ((b1 (%st-range-bounds r1 int?))
        (b2 (%st-range-bounds r2 int?)))
    (and (%st-bound-in? (car b1) (car b2) t)
         (%st-bound-in? (cadr b1) (cadr b2) ()))))

(defun %st-unknown-compound (s)
  (if (member (car s) '(not satisfies member eql)) '(() ()) (%bad-spec s)))

;; a zusammengesetzt
(defun %st-left (a b)
  (let ((head (car a))
        (args (cdr a)))
    (cond ((eq head 'or)  (%st-all (mapcar (lambda (x) (%subtypep x b)) args)))
          ((eq head 'and) (%st-some (mapcar (lambda (x) (%subtypep x b)) args)))
          ((member head '(member eql))
           (%st-ja/nein (every (lambda (v) (typep v b)) args)))
          ((%st-range? a)
           (cond ((%st-range? b)
                  (if (and (car (%subtypep head (car b)))
                           (%st-range-in? a b (eq head 'integer)))
                      '(t t)
                      '(() ())))
                 ((pair? b) (%st-right a b))
                 ;; leerer Bereich wäre Untertyp von allem → nein nur als unklar
                 (t (let ((r (%subtypep head b))) (if (car r) r '(() ()))))))
          (t (%st-unknown-compound a)))))

;; b zusammengesetzt, a Symbol oder Zahlbereich
(defun %st-right (a b)
  (let ((head (car b))
        (args (cdr b)))
    (cond ((eq head 'or)  (%st-some (mapcar (lambda (x) (%subtypep a x)) args)))
          ((eq head 'and) (%st-all (mapcar (lambda (x) (%subtypep a x)) args)))
          ((%st-range? b)
           (let ((base (%subtypep a head)))
             (cond ((not (car base))                  base)
                   ((%st-range-in? (list a) b ())     '(t t))
                   (t                                 '(() ())))))
          (t (%st-unknown-compound b)))))

(defun %subtypep (a b)
  (cond ((equal? a b)       '(t t))
        ((eq b t)           '(t t))
        ((null a)           '(t t))
        ((%st-deftype? a)   (%subtypep (%st-expand a) b))
        ((%st-deftype? b)   (%subtypep a (%st-expand b)))
        ((null b)           (if (symbol? a) '(() t) '(() ())))
        ((pair? a)          (%st-left a b))
        ((pair? b)          (%st-right a b))
        (t                  (%st-names a b))))

;; (subtypep typ1 typ2) → zwei Werte: Untertyp? und sicher?
(defun subtypep (a b)
  (let ((r (%subtypep a b)))
    (values (car r) (cadr r))))
