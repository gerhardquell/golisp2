;; ********************************************************************
;; tests/stdlib-test.lisp — isolierte Tests für stdlib-Funktionen
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k2.7-code
;; ********************************************************************
;; Diese Funktionen hatten bisher nur gps.lisp als Zeugen.
;; Läuft über tests/test-framework.lisp: Tests registrieren sich,
;; Ausführung via (run-tests) — Fehler brechen den Load nicht ab.
;; ********************************************************************

(load "tests/test-framework.lisp")

;; === Mengenoperationen =================================================

(defsuite 'stdlib-mengen)

(deftest union-ops :suite 'stdlib-mengen
  (is (equal? '(1 2 3 4 5) (union '(1 2 3) '(3 4 5))))
  (is (equal? '(a b c)     (union '(a b) '(b c))))
  (is (equal? '()          (union '() '()))))

(deftest set-difference-ops :suite 'stdlib-mengen
  (is (equal? '(1 3)       (set-difference '(1 2 3 4) '(2 4))))
  (is (equal? '(1 2 3)     (set-difference '(1 2 3) '())))
  (is (equal? '()          (set-difference '(1 2 3) '(1 2 3)))))

(deftest find-all-ops :suite 'stdlib-mengen
  (is (equal? '(2 2)       (find-all 2 '(1 2 3 2))))
  (is (equal? '(a a)       (find-all 'a '(a b a c) :test equal?)))
  (is (equal? '()          (find-all 'x '(a b c)))))

;; === Variablen / Zuweisung =============================================

(defsuite 'stdlib-variablen)

(deftest defvar-ops :suite 'stdlib-variablen
  ;; eval: defvar im Test-Rumpf wäre lokal gebunden (Lambda-Scope)
  (is (equal? 'v (eval '(defvar v 1))))
  (is (equal? 'v (eval '(defvar v 2))))
  (is (equal? 1  v)))

;; === Strukturen ========================================================

(defsuite 'stdlib-strukturen)

(deftest defstruct-pt :suite 'stdlib-strukturen
  ;; eval: defstruct im Test-Rumpf wäre lokal — pt?/make-pt müssen
  ;; global sichtbar bleiben (defstruct-box nutzt pt? ebenfalls)
  (is (equal? 'pt?      (eval '(defstruct pt (x 0) (y 0)))))
  (is (equal? '(pt 7 0) (make-pt :x 7)))
  (is (equal? 0         (pt-x (make-pt))))
  (is (equal? 2         (pt-y (make-pt :y 2))))
  (define p (make-pt :x 1 :y 2))
  (is (equal? 9         (setf (pt-x p) 9)))
  (is (equal? 9         (pt-x p)))
  (is (equal? 2         (pt-y p))))

(deftest defstruct-box :suite 'stdlib-strukturen
  (is (equal? 'box?         (eval '(defstruct box (list nil)))))
  (is (equal? '(box (1 2 3)) (make-box :list '(1 2 3))))
  (is (equal? '(1 2 3)      (box-list (make-box :list '(1 2 3)))))
  (is (equal? t             (pt? (make-pt))))
  (is (equal? ()            (pt? '(not-a-pt)))))

;; === Kollision: defstruct set mit Slot difference ======================
;; 'set-difference' existiert bereits in der stdlib. Der Accessor muss
;; ausweichen und auf stderr warnen.

(deftest defstruct-set-kollision :suite 'stdlib-strukturen
  (is (equal? 'set?         (eval '(defstruct set difference))))
  (is (equal? '(1)          (set-difference '(1 2) '(2 3))))
  (is (equal? '(a b)        (set--difference (make-set :difference '(a b))))))

"stdlib-test: registriert"
