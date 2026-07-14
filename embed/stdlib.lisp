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

;; let* – sequentielle Bindungen (jede sieht die vorherigen)
(defmacro let* (bindings . body)
  (if (null bindings)
      `(begin ,@body)
      `(let (,(car bindings))
         (let* ,(cdr bindings) ,@body))))

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
  (if (= n 0) (car lst) (nth (- n 1) (cdr lst))))

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

(defun reduce (f init lst)
  (if (null lst)
      init
      (reduce f (f init (car lst)) (cdr lst))))

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

(defun max (a &rest rest) (reduce (lambda (x y) (if (>= x y) x y)) a rest))
(defun min (a &rest rest) (reduce (lambda (x y) (if (<= x y) x y)) a rest))

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
        (n   (cadr var-n)))
    `(let ((,var 0))
       (while (< ,var ,n)
         ,@body
         (set! ,var (+ ,var 1))))))

(defmacro dolist (var-lst . body)
  (let ((var (car  var-lst))
        (lst (cadr var-lst))
        (tmp (gensym)))
    `(let ((,tmp ,lst))
       (while (not (null ,tmp))
         (let ((,var (car ,tmp)))
           ,@body)
         (set! ,tmp (cdr ,tmp))))))

;; === Datenstrukturen ============================================

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
;; - Accessor-Place mit Symbol-Argument: (setf (pt-x p) 9)
;;   → (begin (set! p (set-pt-x p 9)) 9)
;;   (erfordert, dass der Accessor via register-setf-expander registriert ist).
;; Rückgabewert ist der zugewiesene Wert (wie Common-Lisp setf).
(defmacro setf (place val)
  (if (atom place)
      `(begin (set! ,place ,val) ,val)
      (let ((accessor (car place))
            (arg (cadr place)))
        (if (not (atom arg))
            (error "setf: Place-Argument muss ein Symbol sein")
            (let ((entry (assoc accessor *setf-expanders*)))
              (if (null? entry)
                  (error "setf: unbekannter Place")
                  `(begin (set! ,arg (,(cdr entry) ,arg ,val)) ,val)))))))

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
;; Generiert: make-<name> (&key slot…), <name>-<slot> je Slot, <name>? Prädikat,
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
         ;; Reload? make-name oder name? existiert bereits → keine Ausweichung.
         (reload?    (or (bound? mk) (bound? pred))))
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
       (defun ,pred (x) (and (list? x) (equal? (car x) ',name))))))