;; ********************************************************************
;; stdlib.lisp – GoLisp Standardbibliothek
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : claude sonnet 4.6
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260224
;; ********************************************************************

;; === List-Accessoren ============================================

(defun cadr   (x) (car (cdr x)))
(defun caddr  (x) (car (cdr (cdr x))))
(defun cadddr (x) (car (cdr (cdr (cdr x)))))
(defun cddr   (x) (cdr (cdr x)))
(defun cdar   (x) (cdr (car x)))
(defun caar   (x) (car (car x)))

(defun first  (x) (car x))
(defun second (x) (cadr x))
(defun third  (x) (caddr x))
(defun fourth (x) (cadddr x))
(defun rest   (x) (cdr x))

;; === Prädikate ==================================================

(defun zero?     (n) (= n 0))
(defun positive? (n) (> n 0))
(defun negative? (n) (< n 0))
(defun pair?     (x) (not (atom x)))

;; === Kontrollfluss-Makros =======================================

(defmacro when (test . body)
  `(if ,test (begin ,@body) ()))

(defmacro unless (test . body)
  `(if ,test () (begin ,@body)))

;; let* ist eine Go-Spezialform (lib/eval_core.go) — hier absichtlich NICHT
;; als Makro definiert. Spezialformen werden vor Makros geprüft, ein Makro
;; mit diesem Namen wäre von eval aus unerreichbar und nur über
;; macroexpand sichtbar: zwei Implementierungen, die still auseinanderlaufen.
;; Bewacht von TestNoLispDefineShadowsSpecialForm.

;; === Listen =====================================================

(defun reverse (lst)
  (defun rev-acc (l acc)
    (if (null l) acc (rev-acc (cdr l) (cons (car l) acc))))
  (rev-acc lst ()))

(defun length (lst)
  (defun len-acc (l n)
    (if (null l) n (len-acc (cdr l) (+ n 1))))
  (len-acc lst 0))

(defun nth (n lst)
  (cond ((null lst) ())
        ((= n 0) (car lst))
        (t (nth (- n 1) (cdr lst)))))

(defun last (lst)
  (if (null (cdr lst)) (car lst) (last (cdr lst))))

(defun member (x lst)
  (cond
    ((null lst)            ())
    ((equal? x (car lst)) lst)
    (t                    (member x (cdr lst)))))

(defun assoc (key lst)
  (cond
    ((null lst)                      ())
    ((equal? key (caar lst))  (car lst))
    (t                        (assoc key (cdr lst)))))

(defun filter (pred lst)
  (cond
    ((null lst)          ())
    ((pred (car lst))    (cons (car lst) (filter pred (cdr lst))))
    (t                   (filter pred (cdr lst)))))

;; drop/take: Listen-Präfix-Helfer (für reduce :start/:end).
(defun drop (n lst)
  (if (or (null lst) (<= n 0))
      lst
      (drop (- n 1) (cdr lst))))

(defun take (n lst)
  (if (or (null lst) (<= n 0))
      ()
      (cons (car lst) (take (- n 1) (cdr lst)))))

;; %reduce-fold: eigentliche Faltung (tail-rekursiv, TCO). from-end:
;; f erhält (elem acc), sonst (acc elem). key wird nur auf kombinierte
;; Elemente angewandt, nicht auf den Startwert (CLHS).
(defun %reduce-fold (f s acc key from-end)
  (if (null s)
      acc
      (let ((e (if key (funcall key (car s)) (car s))))
        (%reduce-fold f (cdr s) (if from-end (f e acc) (f acc e)) key from-end))))

;; reduce: CL-Signatur — (reduce f seq &key initial-value from-end start
;; end key). Ohne initial-value ist das erste Element der Startwert;
;; leere Sequenz ohne initial-value ist ein Fehler (CL).
(defun reduce (f seq &key (initial-value nil init-p) (from-end nil) (start 0) (end nil) (key nil))
  (let* ((s (drop start seq))
         (s (if end (take (- end start) s) s))
         (s (if from-end (reverse s) s)))
    (if init-p
        (%reduce-fold f s initial-value key from-end)
        (if (null s)
            (error "reduce: leere Sequenz ohne :initial-value")
            (%reduce-fold f (cdr s) (car s) key from-end)))))

(defun for-each (f lst)
  (if (null lst)
      ()
      (begin (f (car lst)) (for-each f (cdr lst)))))

(defun any (pred lst)
  (cond
    ((null lst)          ())
    ((pred (car lst))    t)
    (t                   (any pred (cdr lst)))))

(defun every (pred lst)
  (cond
    ((null lst)               t)
    ((pred (car lst))         (every pred (cdr lst)))
    (t                        ())))

(defun flatten (lst)
  (cond
    ((null lst)       ())
    ((pair? (car lst)) (append (flatten (car lst)) (flatten (cdr lst))))
    (t                (cons (car lst) (flatten (cdr lst))))))

(defun zip (lst1 lst2)
  (if (or (null lst1) (null lst2))
      ()
      (cons (list (car lst1) (car lst2))
            (zip (cdr lst1) (cdr lst2)))))

(defun list-tail (lst n)
  (if (= n 0) lst (list-tail (cdr lst) (- n 1))))

(defun iota (n)
  (defun iota-acc (i acc)
    (if (< i 0) acc (iota-acc (- i 1) (cons i acc))))
  (iota-acc (- n 1) ()))

;; === Zahlen =====================================================

(defun max (a &rest rest) (reduce (lambda (x y) (if (>= x y) x y)) rest :initial-value a))
(defun min (a &rest rest) (reduce (lambda (x y) (if (<= x y) x y)) rest :initial-value a))

(defun square (x) (* x x))

(defun expt (base exp)
  (cond
    ((= exp 0) 1)
    ((= exp 1) base)
    (t         (* base (expt base (- exp 1))))))

(defun gcd (a b)
  (if (= b 0) a (gcd b (- a (* (/ a b) b)))))

;; === Iteratoren =================================================

(defmacro dotimes (var-n . body)
  (let ((var (car  var-n))
        (n   (cadr var-n))
        (res (caddr var-n))
        (tmp (gensym)))
    `(let ((,tmp ,n)
           (,var 0))
       (while (< ,var ,tmp)
         ,@body
         (set! ,var (+ ,var 1)))
       ,res)))

(defmacro dolist (var-lst . body)
  (let ((var (car  var-lst))
        (lst (cadr var-lst))
        (res (caddr var-lst))
        (tmp (gensym)))
    `(let ((,tmp ,lst))
       (while (not (null ,tmp))
         (let ((,var (car ,tmp)))
           ,@body)
         (set! ,tmp (cdr ,tmp)))
       (let ((,var ()))
         ,res))))

;; === Datenstrukturen ============================================

;; nth-value: seit 20260723 Go-Spezialform (eval_mv.go) – früher Makro
;; auf nth + multiple-value-list; als Makro hier wäre es toter Code.

;; push/pop – verändern eine Variable
(defmacro push (val var)
  `(set! ,var (cons ,val ,var)))

(defmacro pop (var)
  (let ((tmp (gensym)))
    `(let ((,tmp (car ,var)))
       (set! ,var (cdr ,var))
       ,tmp)))

;; Assoziationsliste: Wert setzen oder hinzufügen
(defun alist-set (key val lst)
  (cond
    ((null lst)               (list (list key val)))
    ((equal? key (caar lst))  (cons (list key val) (cdr lst)))
    (t                        (cons (car lst) (alist-set key val (cdr lst))))))

(defun alist-get (key lst)
  (let ((entry (assoc key lst)))
    (if (null entry) () (cadr entry))))

;; === Höhere Ordnung =============================================

(defun identity    (x)   x)
(defun constantly  (x)   (lambda args x))
(defun complement  (f)   (lambda (x) (not (f x))))
(defun compose     (f g) (lambda (x) (f (g x))))

;; === Mengenoperationen ==========================================

;; union: Vereinigung zweier Mengen (als Listen, ohne Duplikate).
(defun union (a b)
  (append a (filter (lambda (x) (not (member x a))) b)))

;; set-difference: Elemente in a, aber nicht in b.
(defun set-difference (a b)
  (filter (lambda (x) (not (member x b))) a))

;; find-all: alle Elemente aus seq, für die (test item element) wahr ist.
;; CL-Compat: (find-all item seq &key (test equal?)).
(defun find-all (item seq &key (test equal?))
  (filter (lambda (x) (test item x)) seq))

;; === CL-Compat: Variablen / Zuweisung ===========================

;; defvar: deklariert eine globale Variable (CL-Compat). Docstring
;; (optionales 3. Argument) wird ignoriert. Zweites defvar für dasselbe
;; Symbol ist no-op, wenn es bereits gebunden ist.
(defmacro defvar (name &rest rest)
  (let ((val (if (null rest) () (car rest))))
    `(if (bound? ',name)
         ',name
         (define ,name ,val))))

;; Registry für setf-Places: Assoc-Liste (accessor . setter).
(define *setf-expanders* '())

(defun register-setf-expander (accessor setter)
  (set! *setf-expanders* (cons (cons accessor setter) *setf-expanders*)))

;; setf: generalisierte Zuweisung (CL-Compat).
;; - Variable: (setf x 1) → (begin (set! x 1) 1)
;; - gethash-Place: (setf (gethash k t) v) → (puthash k t v)
;;   (mutiert die Tabelle direkt, kein register-setf-expander nötig)
;; - nth-Place: (setf (nth i xs) v) → (set! xs (set-nth xs i v))
;;   (immutable Cells: neue Liste, Symbol-Rebind statt In-Place-Mutation;
;;   Index und Wert werden je genau einmal ausgewertet)
;; - Accessor-Place mit Symbol-Argument: (setf (pt-x p) 9)
;;   → (begin (set! p (set-pt-x p 9)) 9)
;;   (erfordert, dass der Accessor via register-setf-expander registriert ist).
;; Rückgabewert ist der zugewiesene Wert (wie Common-Lisp setf).
;; Ungebundenes Symbol wird CL-artig global definiert: eval läuft im
;; Root-Env, das quote um den Wert verhindert Re-Evaluierung.
;; Multi-Place: (setf a 1 b 2 …) — je Paar eine setf-Form, sequenziell in
;; einem (begin …), Rückgabe ist der letzte zugewiesene Wert (CL-Semantik).
(defmacro setf (place val &rest more)
  (if more
      `(begin (setf ,place ,val) (setf ,@more))
      (%setf-one place val)))

(defun %setf-one (place val)
  (let ((v (gensym)))
    (if (atom place)
        `(let ((,v ,val))
           (if (bound? ',place)
               (set! ,place ,v)
               (eval (list 'define ',place (list 'quote ,v))))
           ,v)
        (let ((accessor (car place))
              (arg (cadr place)))
          (if (equal? accessor 'gethash)
              `(puthash ,(cadr place) ,(caddr place) ,val)
              (if (equal? accessor 'nth)
                  (let ((i (gensym)))
                    (if (not (atom (caddr place)))
                        (error "setf: nth-Place-Argument muss ein Symbol sein")
                        `(let ((,i ,(cadr place)) (,v ,val))
                           (set! ,(caddr place) (set-nth ,(caddr place) ,i ,v))
                           ,v)))
                  (if (not (atom arg))
                      (error "setf: Place-Argument muss ein Symbol sein")
                      (let ((entry (assoc accessor *setf-expanders*)))
                        (if (null? entry)
                            (error "setf: unbekannter Place")
                            `(let ((,v ,val))
                               (set! ,arg (,(cdr entry) ,arg ,v))
                               ,v))))))))))

;; === Fehlerbehandlung ============================================

;; ignore-errors: (ignore-errors body…) → Wert von body, oder () bei Fehler.
;; Dünner Wrapper über trap (siehe doc/lisp-semantik.md „Fehlermodell").
;; throw/return-from/parfunc-Signale laufen unverändert durch (trap fängt
;; nur echte Fehler) — ignore-errors ist kein catch-Ersatz.
(defmacro ignore-errors (&rest body)
  `(trap (begin ,@body) (lambda (e) ())))

;; === Strukturen =================================================

;; set-nth: Liste an Position n (0-basiert) durch val ersetzen.
(defun set-nth (lst n val)
  (if (= n 0)
      (cons val (cdr lst))
      (cons (car lst) (set-nth (cdr lst) (- n 1) val))))

;; Hilfsfunktion für defstruct: finde einen freien Namen, falls der Primärname
;; bereits gebunden ist. Bei Reload (reload? = t) wird der Primärname beibehalten.
;; Kollisionsvermeidung fügt Bindestriche zwischen name und slot ein:
;;   prefix=""  → <name>-<slot>, <name>--<slot>, …
;;   prefix="set-" → set-<name>-<slot>, set-<name>--<slot>, …
(defun defstruct-resolve-name (prefix name slot sep reload?)
  (let ((candidate (intern (format nil "~a~a~a~a" prefix name sep slot))))
    (if (or reload? (not (bound? candidate)))
        candidate
        (defstruct-resolve-name prefix name slot (string-append sep "-") reload?))))

;; defstruct: (defstruct name [docstring] slot…) mit slot = sym | (sym [default]).
;; Repräsentation als Liste (tag val1 val2 …) – golisp2 hat keine Vektoren.
;; Generiert: make-<name> (&key slot…), <name>-<slot> je Slot, <name>? Prädikat
;; (plus <name>-p als CL-Alias, gleiche Kollisionsvermeidung wie Accessoren),
;; sowie Setter-Funktionen set-<name>-<slot> und registriert sie für setf.
;; Kollisionsvermeidung: Ist der Primärname bereits gebunden, wird eine
;; freie Alternative mit zusätzlichem Bindestrich verwendet (z. B. set--difference).
;; Beim erneuten Laden desselben defstruct bleiben die Primärnamen erhalten,
;; damit alte Aufrufer nicht auf ausgewichene Namen verweisen.
(defmacro defstruct (name &rest body)
  (let* ((slots      (filter (lambda (x) (not (string? x))) body))
         (slot-names (mapcar (lambda (s) (if (list? s) (car s) s)) slots))
         (n          (length slots))
         (mk         (intern (format nil "make-~a" name)))
         (pred       (intern (format nil "~a?" name)))
         (pred-p     (intern (format nil "~a-p" name)))
         ;; Reload? make-name oder name? existiert bereits → keine Ausweichung.
         (reload?    (or (bound? mk) (bound? pred)))
         (pred-p-final (defstruct-resolve-name "" name "p" "-" reload?))
         (warn-pred-p  (if (equal? pred-p pred-p-final)
                            '()
                            `((warn ,(format nil "WARN: defstruct ~a: '~a' existiert → Prädikat heißt '~a'" name pred-p pred-p-final))))))
    `(begin
       (defun ,mk (&key ,@slots) (%make-struct ',name ,@slot-names))
       ,@(mapcar (lambda (p)
                   (let* ((s (car p))
                          (i (cadr p))
                          (acc (intern (format nil "~a-~a" name s)))
                          (setter (intern (format nil "set-~a-~a" name s)))
                          (idx (+ i 1))
                          (acc-final (defstruct-resolve-name "" name s "-" reload?))
                          (setter-final (defstruct-resolve-name "set-" name s "-" reload?))
                          (warn-acc (if (equal? acc acc-final)
                                        '()
                                        `((warn ,(format nil "WARN: defstruct ~a: '~a' existiert → Accessor heißt '~a'" name acc acc-final)))))
                          (warn-setter (if (equal? setter setter-final)
                                           '()
                                           `((warn ,(format nil "WARN: defstruct ~a: '~a' existiert → Setter heißt '~a'" name setter setter-final))))))
                     `(begin
                        ,@warn-acc
                        (defun ,acc-final (x) (nth ,idx x))
                        ,@warn-setter
                        (defun ,setter-final (obj val) (set-nth obj ,idx val))
                        (register-setf-expander ',acc-final ',setter-final))))
                 (zip slot-names (iota n)))
       ,@warn-pred-p
       (defun ,pred-p-final (x) (and (list? x) (equal? (car x) ',name)))
       (defun ,pred (x) (and (list? x) (equal? (car x) ',name))))))
;; === Generische Funktionen (CLOS-light, TODO 20260813 Punkt 2.5) =====
;; defgeneric/defmethod: Single-Dispatch auf den Struct-Tag (car) des
;; ersten Arguments. Kein Kernel-Eingriff — Methoden sind Lambdas in
;; einer Registry-Hashtabelle. Hot-Patching: defmethod desselben Tags
;; überschreibt still (bewusst: SWANK-Workflow).
;; Explizit NICHT dabei: Vererbung, call-next-method, :before/:after,
;; Multi-Dispatch. Registry-Format lässt Raum für spätere Erweiterung.

;; %generic-registry: generic-name → Hashtabelle(tag → lambda)
(defvar %generic-registry (make-hash-table))

;; %generic-methods: liefert (legt ggf. an) die Methoden-Tabelle für name.
(defun %generic-methods (name)
  (let ((tbl (gethash name %generic-registry)))
    (if (hash-table-p tbl)
        tbl
        (let ((neu (make-hash-table)))
          (puthash name %generic-registry neu)
          neu))))

;; %generic-dispatch: Tag aus (car obj), Methode suchen, mit allen
;; Originalargumenten aufrufen. 't-Tag = Default/Fallback.
(defun %generic-dispatch (name args)
  (if (null args)
      (error (format nil "~a: generische Funktion braucht mindestens 1 Argument" name)))
  (let* ((obj (car args))
         (tag (if (and (list? obj) (symbol? (car obj)))
                  (car obj)
                  '()))
         (tbl (%generic-methods name))
         (fn  (gethash tag tbl)))
    (if (null fn)
        (let ((default (gethash 't tbl)))
          (if (null default)
              (error (format nil "~a: keine Methode für ~a" name
                             (if (null tag) obj (format nil "Tag '~a'" tag))))
              (apply default args)))
        (apply fn args))))

;; defgeneric: (defgeneric fläche (x)) — deklariert den Dispatcher.
;; params ist aktuell nur Doku (Dispatch läuft über &rest).
(defmacro defgeneric (name params)
  `(defun ,name (&rest args) (%generic-dispatch ',name args)))

;; defmethod: (defmethod fläche ((x kreis) . rest-params) body ...)
;; Erster Eintrag der Methoden-Lambda-Liste ist (var tag): var wird im
;; Body gebunden, tag ist der Struct-Tag (oder 't für die Default-Methode).
;; Signatur flach — GoLisp-Makros destrukturieren nicht geschachtelt.
(defmacro defmethod (name lambda-list &rest body)
  (let ((spec   (car lambda-list))
        (params (cdr lambda-list)))
    (let ((var (car spec))
          (tag (car (cdr spec))))
      `(puthash ',tag (%generic-methods ',name)
                (lambda (,var ,@params) ,@body)))))
