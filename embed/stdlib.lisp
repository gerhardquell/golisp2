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
(defun null?     (x) (null x))
(defun list?     (x) (or (null x) (pair? x)))

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

(defun append (lst1 lst2)
  (if (null lst1)
      lst2
      (cons (car lst1) (append (cdr lst1) lst2))))

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

(defun abs (n)
  (if (negative? n) (- 0 n) n))

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
