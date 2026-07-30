;; ********************************************************************
;; tests/defsystem-tests.lisp — Tests für defsystem-lite
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k3
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260725
;; ********************************************************************
;; Fixtures unter tests/fixtures/. Läuft über tests/test-framework.lisp.
;; Aufräumen (unload + Registry zurücksetzen) ist der letzte Test der
;; Suite — läuft auch nach Fehlschlägen, da run-tests nicht abbricht.
;; ********************************************************************

(load "tests/test-framework.lisp")

;; --- defsystem: Registrierung ---------------------------------------
(defsuite 'defsystem-registry)

(deftest registrierung :suite 'defsystem-registry
  (defsystem fx-sys-c :components ("tests/fixtures/fx-c.lisp"))
  (defsystem fx-sys-b :depends-on (fx-sys-c) :components ("tests/fixtures/fx-b.lisp"))
  (defsystem fx-sys-a :depends-on (fx-sys-b) :components ("tests/fixtures/fx-a.lisp"))
  (is (equal? t (if (assoc 'fx-sys-a *systems*) t ())))
  (is (equal? '(fx-sys-c) (%sys-get (%sys-entry 'fx-sys-b) :depends-on)))
  (is (equal? '("tests/fixtures/fx-a.lisp") (%sys-get (%sys-entry 'fx-sys-a) :components)))
  ;; :depends-on optional, Default ()
  (is (equal? '() (%sys-get (%sys-entry 'fx-sys-c) :depends-on))))

;; --- defsystem: Validierung zur Expansionszeit ----------------------
(defsuite 'defsystem-validierung)

(deftest keyword-validierung :suite 'defsystem-validierung
  (is (equal? 'err (trap (eval '(defsystem bad1 :falsch ())) (lambda (e) 'err))))
  (is (equal? 'err (trap (eval '(defsystem bad2 :depends-on ("string-statt-symbol"))) (lambda (e) 'err))))
  (is (equal? 'err (trap (eval '(defsystem bad3 :components (symbol-statt-string))) (lambda (e) 'err))))
  (is (equal? 'err (trap (eval '(defsystem bad4 :depends-on)) (lambda (e) 'err)))))

;; --- load-system: Topo-Reihenfolge + Idempotenz ----------------------
(defsuite 'defsystem-load)

(deftest topo-und-idempotenz :suite 'defsystem-load
  (load-system 'fx-sys-a)
  (is (equal? t (bound? 'fx-a)))
  (is (equal? t (bound? 'fx-b)))
  (is (equal? t (bound? 'fx-c)))
  ;; Deps zuerst geladen; push präpendiert → zuletzt geladenes oben
  (is (equal? (list (get-file-path "tests/fixtures/fx-a.lisp")
                    (get-file-path "tests/fixtures/fx-b.lisp")
                    (get-file-path "tests/fixtures/fx-c.lisp"))
              *loaded-files*))
  ;; Idempotenz: zweiter Aufruf lädt nichts nach
  (let ((n (length *loaded-files*)))
    (load-system 'fx-sys-a)
    (is (equal? n (length *loaded-files*)))))

;; --- load-system: Diamond-Dependency ---------------------------------

(deftest diamond-dependency :suite 'defsystem-load
  (defsystem fx-sys-dm-d :components ("tests/fixtures/fx-d.lisp"))
  (defsystem fx-sys-dm-b :depends-on (fx-sys-dm-d) :components ())
  (defsystem fx-sys-dm-c :depends-on (fx-sys-dm-d) :components ())
  (defsystem fx-sys-dm-a :depends-on (fx-sys-dm-b fx-sys-dm-c) :components ())
  (load-system 'fx-sys-dm-a)
  (is (equal? t (bound? 'fx-d)))
  ;; fx-d.lisp wird über zwei Pfade referenziert, aber nur einmal geladen
  (is (equal? 1 (length (find-all (get-file-path "tests/fixtures/fx-d.lisp") *loaded-files*)))))

;; --- load-system: Fehlerfälle ----------------------------------------

(deftest load-fehlerfaelle :suite 'defsystem-load
  (is (equal? 'err (trap (load-system 'gibts-nicht) (lambda (e) 'err))))
  (defsystem fx-cy-a :depends-on (fx-cy-b) :components ())
  (defsystem fx-cy-b :depends-on (fx-cy-a) :components ())
  (is (equal? 'err (trap (load-system 'fx-cy-a) (lambda (e) 'err)))))

;; --- loaded-systems + system-symbols ---------------------------------
(defsuite 'defsystem-introspektion)

(deftest loaded-und-symbols :suite 'defsystem-introspektion
  (is (equal? t (if (member 'fx-sys-a (loaded-systems)) t ())))
  (is (equal? t (if (member 'fx-sys-b (loaded-systems)) t ())))
  ;; fx-sys-u listet eine Datei, die nie geladen wird -> nicht geladen
  (defsystem fx-sys-u :components ("tests/fixtures/fx-u.lisp"))
  (is (equal? '() (if (member 'fx-sys-u (loaded-systems)) '(drin) '())))
  (is (equal? '(fx-a) (system-symbols 'fx-sys-a)))
  (is (equal? 'err (trap (system-symbols 'gibts-nicht) (lambda (e) 'err)))))

;; --- unload-system ----------------------------------------------------
(defsuite 'defsystem-unload)

(deftest unload-einfach :suite 'defsystem-unload
  (is (equal? '(fx-a) (unload-system 'fx-sys-a)))
  (is (equal? '() (bound? 'fx-a)))
  ;; b und c sind Deps von a — werden NICHT mit-entladen
  (is (equal? t (bound? 'fx-b)))
  (is (equal? t (bound? 'fx-c)))
  (is (equal? '() (if (member 'fx-sys-a (loaded-systems)) '(drin) '()))))

(deftest unload-shared-file :suite 'defsystem-unload
  (defsystem fx-sys-s1 :components ("tests/fixtures/fx-shared.lisp"))
  (defsystem fx-sys-s2 :components ("tests/fixtures/fx-shared.lisp"))
  (load-system 'fx-sys-s1)
  (load-system 'fx-sys-s2)
  (is (equal? '() (unload-system 'fx-sys-s1)))     ; shared → nichts entfernt
  (is (equal? t (bound? 'fx-shared)))
  (is (equal? '(fx-shared) (unload-system 'fx-sys-s2)))
  (is (equal? '() (bound? 'fx-shared))))

(deftest unload-noop-und-unbekannt :suite 'defsystem-unload
  (is (equal? '() (unload-system 'fx-sys-s1)))
  (is (equal? '() (unload-system 'fx-sys-u)))
  (is (equal? 'err (trap (unload-system 'gibts-nicht) (lambda (e) 'err)))))

(deftest load-teilzustand-bei-fehler :suite 'defsystem-unload
  (defsystem fx-sys-kaputt :components ("tests/fixtures/fx-kaputt.lisp"))
  (is (equal? 'err (trap (load-system 'fx-sys-kaputt) (lambda (e) 'err))))
  (is (equal? t (bound? 'fx-kaputt-ok)))
  (is (equal? '() (if (member 'fx-sys-kaputt (loaded-systems)) '(drin) '()))))

;; --- Aufräumen: Registry + Symbole zurücksetzen -----------------------
;; Letzter Test der Suite: läuft auch nach Fehlschlägen (run-tests
;; bricht nicht ab), solange die Suite selbst läuft.

(deftest cleanup :suite 'defsystem-unload
  (unload-system 'fx-sys-b)
  (unload-system 'fx-sys-c)
  (set! *systems*
        (filter (lambda (e) (not (member (car e) '(fx-sys-a fx-sys-b fx-sys-c
                                                   fx-cy-a fx-cy-b fx-sys-u
                                                   fx-sys-s1 fx-sys-s2
                                                   fx-sys-dm-a fx-sys-dm-b
                                                   fx-sys-dm-c fx-sys-dm-d
                                                   fx-sys-kaputt))))
                *systems*))
  ;; fx-kaputt-ok wurde vor dem Fehler definiert; unterdrücke makunbound-Warnung
  (let ((p (redefine-policy)))
    (redefine-policy 'allow)
    (when (bound? 'fx-kaputt-ok) (makunbound 'fx-kaputt-ok))
    (redefine-policy p))
  ;; *loaded-files*: load-system pusht erst nach erfolgreichem load, daher
  ;; bei Fehler kein Eintrag für fx-kaputt.lisp (nichts zu entfernen).
  (is (equal? '() (if (assoc 'fx-sys-a *systems*) '(drin) '()))))
