;; ********************************************************************
;; tests/defsystem-tests.lisp — Tests für defsystem-lite
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k3
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260725
;; ********************************************************************
;; Fixtures unter tests/fixtures/. Aufräumen am Ende: unload + Registry
;; zurücksetzen, damit nachfolgende Suites unberührt bleiben.
;; ********************************************************************

(load "tests/test-helpers.lisp")  ; assert=

;; --- defsystem: Registrierung ---------------------------------------
(defsystem fx-sys-c :components ("tests/fixtures/fx-c.lisp"))
(defsystem fx-sys-b :depends-on (fx-sys-c) :components ("tests/fixtures/fx-b.lisp"))
(defsystem fx-sys-a :depends-on (fx-sys-b) :components ("tests/fixtures/fx-a.lisp"))

(assert= t (if (assoc 'fx-sys-a *systems*) t ()))
(assert= '(fx-sys-c) (%sys-get (%sys-entry 'fx-sys-b) :depends-on))
(assert= '("tests/fixtures/fx-a.lisp") (%sys-get (%sys-entry 'fx-sys-a) :components))
;; :depends-on optional, Default ()
(assert= '() (%sys-get (%sys-entry 'fx-sys-c) :depends-on))

;; --- defsystem: Validierung zur Expansionszeit ----------------------
(assert= 'err (trap (eval '(defsystem bad1 :falsch ())) (lambda (e) 'err)))
(assert= 'err (trap (eval '(defsystem bad2 :depends-on ("string-statt-symbol"))) (lambda (e) 'err)))
(assert= 'err (trap (eval '(defsystem bad3 :components (symbol-statt-string))) (lambda (e) 'err)))
