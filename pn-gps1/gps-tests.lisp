;; ********************************************************************
;; pn-gps1/gps-tests.lisp – Charakterisierungstests für GPS-Spracherweiterungen
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k2.7-code
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260713
;; ********************************************************************
;; Aus pn-gps1/TODO.md: Jede neue stdlib-Funktion braucht eigenen Test,
;; nicht nur den Pfad, den GPS durchläuft.

;; === Test-Hilfe =====================================================

(defun report (name passed? expected actual)
  (if passed?
      (println (string-append "PASS: " name))
      (begin
        (println (string-append "FAIL: " name))
        (print "  expected: ") (println expected)
        (print "  actual:   ") (println actual))))

(defmacro check-equal (name expected expr)
  `(report ,name (equal? ,expected ,expr) ,expected ,expr))

;; === union ==========================================================

(check-equal "union removes duplicates" '(a b c) (union '(a b) '(b c)))
(check-equal "union empty right" '(a) (union '(a) '()))
(check-equal "union empty left" '(a) (union '() '(a)))

;; === set-difference =================================================

(check-equal "set-difference basic" '(a c) (set-difference '(a b c) '(b)))
(check-equal "set-difference empty subtrahend" '(a b) (set-difference '(a b) '()))
(check-equal "set-difference empty minuend" '() (set-difference '() '(a)))

;; === find-all =======================================================

(check-equal "find-all default test" '(a a) (find-all 'a '(a b a c)))
(check-equal "find-all with :test #'<" '(3) (find-all 2 '(1 2 3) :test #'<))
(check-equal "find-all missing" '() (find-all 'x '(a b)))

;; === defstruct ======================================================

(defstruct pt (x 0) (y 0))

(check-equal "defstruct defaults" 0 (pt-x (make-pt)))
(check-equal "defstruct partial init" 0 (pt-y (make-pt :x 5)))
(check-equal "defstruct predicate true" 't (pt? (make-pt)))
(check-equal "defstruct predicate false" '() (pt? '(1 2)))

;; === setf ===========================================================

(define v 1)
(setf v 2)
(check-equal "setf variable place" 2 v)

;; === defvar =========================================================

;; CL-Semantik: erstes defvar gewinnt, zweites ist no-op.
;; Aktuell expandiert defvar zu define → zweites defvar überschreibt.
(defvar *test-gps-z* 1)
(defvar *test-gps-z* 99)
(check-equal "defvar idempotent" 1 *test-gps-z*)

;; === setf Accessor-Place ============================================

(define p (make-pt))
;; CL: (setf (pt-x p) 9) sollte 9 setzen. Aktuell: setf kennt nur
;; Variablen-Places, keine Accessor-Places.
(check-equal "setf accessor place" 9 (begin (setf (pt-x p) 9) (pt-x p)))

;; === Zusammenfassung ================================================
(println "")
(println "gps-tests.lisp beendet.")
nil
