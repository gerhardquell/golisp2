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

;; --- load-system: Topo-Reihenfolge + Idempotenz ----------------------
(load-system 'fx-sys-a)
(assert= t (bound? 'fx-a))
(assert= t (bound? 'fx-b))
(assert= t (bound? 'fx-c))
;; Deps zuerst geladen; push präpendiert → zuletzt geladenes oben
(assert= (list (get-file-path "tests/fixtures/fx-a.lisp")
               (get-file-path "tests/fixtures/fx-b.lisp")
               (get-file-path "tests/fixtures/fx-c.lisp"))
         *loaded-files*)
;; Idempotenz: zweiter Aufruf lädt nichts nach
(let ((n (length *loaded-files*)))
  (load-system 'fx-sys-a)
  (assert= n (length *loaded-files*)))

;; --- load-system: Diamond-Dependency ---------------------------------
(defsystem fx-sys-dm-d :components ("tests/fixtures/fx-d.lisp"))
(defsystem fx-sys-dm-b :depends-on (fx-sys-dm-d) :components ())
(defsystem fx-sys-dm-c :depends-on (fx-sys-dm-d) :components ())
(defsystem fx-sys-dm-a :depends-on (fx-sys-dm-b fx-sys-dm-c) :components ())
(load-system 'fx-sys-dm-a)
(assert= t (bound? 'fx-d))
;; fx-d.lisp wird über zwei Pfade referenziert, aber nur einmal geladen
(assert= 1 (length (find-all (get-file-path "tests/fixtures/fx-d.lisp") *loaded-files*)))

;; --- load-system: Fehlerfälle ----------------------------------------
(assert= 'err (trap (load-system 'gibts-nicht) (lambda (e) 'err)))
(defsystem fx-cy-a :depends-on (fx-cy-b) :components ())
(defsystem fx-cy-b :depends-on (fx-cy-a) :components ())
(assert= 'err (trap (load-system 'fx-cy-a) (lambda (e) 'err)))

;; --- loaded-systems + system-symbols ---------------------------------
(assert= t (if (member 'fx-sys-a (loaded-systems)) t ()))
(assert= t (if (member 'fx-sys-b (loaded-systems)) t ()))
;; fx-sys-u listet eine Datei, die nie geladen wird -> nicht geladen
(defsystem fx-sys-u :components ("tests/fixtures/fx-u.lisp"))
(assert= '() (if (member 'fx-sys-u (loaded-systems)) '(drin) '()))
(assert= '(fx-a) (system-symbols 'fx-sys-a))
(assert= 'err (trap (system-symbols 'gibts-nicht) (lambda (e) 'err)))

;; --- unload-system: einfacher Fall ------------------------------------
(assert= '(fx-a) (unload-system 'fx-sys-a))
(assert= '() (bound? 'fx-a))
;; b und c sind Deps von a — werden NICHT mit-entladen
(assert= t (bound? 'fx-b))
(assert= t (bound? 'fx-c))
(assert= '() (if (member 'fx-sys-a (loaded-systems)) '(drin) '()))

;; --- unload-system: Shared File bleibt beim ersten unload stehen ------
(defsystem fx-sys-s1 :components ("tests/fixtures/fx-shared.lisp"))
(defsystem fx-sys-s2 :components ("tests/fixtures/fx-shared.lisp"))
(load-system 'fx-sys-s1)
(load-system 'fx-sys-s2)
(assert= '() (unload-system 'fx-sys-s1))     ; shared → nichts entfernt
(assert= t (bound? 'fx-shared))
(assert= '(fx-shared) (unload-system 'fx-sys-s2))
(assert= '() (bound? 'fx-shared))

;; --- unload-system: nicht-geladenes System = no-op --------------------
(assert= '() (unload-system 'fx-sys-s1))
(assert= '() (unload-system 'fx-sys-u))
(assert= 'err (trap (unload-system 'gibts-nicht) (lambda (e) 'err)))

;; --- Aufräumen: Registry + Symbole zurücksetzen -----------------------
(unload-system 'fx-sys-b)
(unload-system 'fx-sys-c)
(set! *systems*
      (filter (lambda (e) (not (member (car e) '(fx-sys-a fx-sys-b fx-sys-c
                                                 fx-cy-a fx-cy-b fx-sys-u
                                                 fx-sys-s1 fx-sys-s2
                                                 fx-sys-dm-a fx-sys-dm-b
                                                 fx-sys-dm-c fx-sys-dm-d))))
              *systems*))
