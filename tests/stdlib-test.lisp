;; ********************************************************************
;; tests/stdlib-test.lisp — isolierte Tests für stdlib-Funktionen
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k2.7-code
;; ********************************************************************
;; Diese Funktionen hatten bisher nur gps.lisp als Zeugen.
;; Ein Fehler hier bricht den Datei-Load mit einem FAIL ab.
;; ********************************************************************

(defmacro assert= (expected expr)
  `(let ((actual ,expr))
     (if (equal? actual ,expected)
         (println (format nil "PASS: ~a => ~a" ',expr actual))
         (error (format nil "FAIL: ~a erwartet ~a, got ~a" ',expr ,expected actual)))))

;; === Mengenoperationen =================================================

(assert= '(1 2 3 4 5) (union '(1 2 3) '(3 4 5)))
(assert= '(a b c)     (union '(a b) '(b c)))
(assert= '()          (union '() '()))

(assert= '(1 3)       (set-difference '(1 2 3 4) '(2 4)))
(assert= '(1 2 3)     (set-difference '(1 2 3) '()))
(assert= '()          (set-difference '(1 2 3) '(1 2 3)))

(assert= '(2 2)       (find-all 2 '(1 2 3 2)))
(assert= '(a a)       (find-all 'a '(a b a c) :test equal?))
(assert= '()          (find-all 'x '(a b c)))

;; === Variablen / Zuweisung =============================================

(assert= 'v           (defvar v 1))
(assert= 'v           (defvar v 2))
(assert= 1            v)

;; === Strukturen ========================================================

(assert= 'pt?         (defstruct pt (x 0) (y 0)))
(assert= '(pt 7 0)    (make-pt :x 7))
(assert= 0             (pt-x (make-pt)))
(assert= 2             (pt-y (make-pt :y 2)))

(define p (make-pt :x 1 :y 2))
(assert= 9             (setf (pt-x p) 9))
(assert= 9             (pt-x p))
(assert= 2             (pt-y p))

(assert= 'box?         (defstruct box (list nil)))
(assert= '(box (1 2 3)) (make-box :list '(1 2 3)))
(assert= '(1 2 3)      (box-list (make-box :list '(1 2 3))))
(assert= t              (pt? (make-pt)))
(assert= ()             (pt? '(not-a-pt)))

;; === Kollision: defstruct set mit Slot difference ======================
;; 'set-difference' existiert bereits in der stdlib. Der Accessor muss
;; ausweichen und auf stderr warnen.

(assert= 'set?         (defstruct set difference))
(assert= '(1)          (set-difference '(1 2) '(2 3)))
(assert= '(a b)        (set--difference (make-set :difference '(a b))))

"stdlib-test: alle PASS"
