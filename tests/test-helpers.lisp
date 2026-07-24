;; ********************************************************************
;; tests/test-helpers.lisp — gemeinsame Test-Helfer (eine Quelle)
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k3
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260724
;; ********************************************************************
;; assert= lebte vorher als identische Kopie in stdlib-test.lisp und
;; gps2-tests.lisp (REDEF-Warnung der Testsuite). Jetzt: eine Quelle.
;; Mehrfach-Load ist harmlos (gleiche Quelle → stiller Reload).
;; ********************************************************************

(defmacro assert= (expected expr)
  `(let ((actual ,expr))
     (if (equal? actual ,expected)
         (println (format nil "PASS: ~a => ~a" ',expr actual))
         (error (format nil "FAIL: ~a erwartet ~a, got ~a" ',expr ,expected actual)))))
