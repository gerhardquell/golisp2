;; ********************************************************************
;; tests/condition-tests.lisp — Tests für Condition-lite
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k3
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260730
;; ********************************************************************
;; Deckt: Typ-Registry, Vererbungs-Dispatch, Reader, Re-Signal,
;; Go-Fehler-Fallback (lisp-error), Validierung, Reload-Semantik.
;; Läuft über tests/test-framework.lisp (run-tests).
;; ********************************************************************

(load "tests/test-framework.lisp")

(defsuite 'condition)

;; --- Typ-Registry + Vererbung ---------------------------------------

(deftest typ-registry :suite 'condition
  (eval '(define-condition io-error (error) ()))
  (eval '(define-condition file-error (io-error) (path)))
  (eval '(define-condition parse-error (error) (line)))
  (is (equal? '(io-error) (alist-get 'file-error *condition-types*)))
  (is (equal? '(error)    (alist-get 'io-error *condition-types*)))
  ;; Vererbungs-Transitivität: file-error → io-error → error → condition
  (is (equal? t (%cond-subtype? 'file-error 'io-error)))
  (is (equal? t (%cond-subtype? 'file-error 'error)))
  (is (equal? t (%cond-subtype? 'file-error 'condition)))
  (is (equal? '() (%cond-subtype? 'io-error 'file-error)))
  (is (equal? '() (%cond-subtype? 'file-error 'parse-error))))

;; --- signal + handler-case: Direkt-Treffer und Reader ----------------

(deftest direkt-treffer-und-reader :suite 'condition
  (is (equal? "fehlt: x.lisp"
        (handler-case (signal 'file-error :path "x.lisp")
          (file-error (e) (string-append "fehlt: " (file-error-path e))))))
  ;; Reader fehlender Slot → ()
  (is (equal? '()
        (handler-case (signal 'file-error)
          (file-error (e) (file-error-path e))))))

;; --- Eltern-Match und Klausel-Reihenfolge ----------------------------

(deftest eltern-match :suite 'condition
  (is (equal? "io"
        (handler-case (signal 'file-error :path "y.lisp")
          (io-error (e) "io"))))
  ;; erste passende Klausel gewinnt
  (is (equal? "speziell"
        (handler-case (signal 'file-error :path "y.lisp")
          (file-error (e) "speziell")
          (io-error  (e) "allgemein"))))
  (is (equal? "allgemein"
        (handler-case (signal 'file-error :path "y.lisp")
          (parse-error (e) "nie")
          (io-error  (e) "allgemein")))))

;; --- Re-Signal bei Nicht-Match ---------------------------------------

(deftest re-signal :suite 'condition
  (is (equal? "aussen"
        (handler-case
          (handler-case (signal 'file-error :path "z.lisp")
            (parse-error (e) "innen"))
          (file-error (e) "aussen"))))
  ;; unkonditioniertes Re-Signal läuft bis trap
  (is (equal? "gefangen"
        (trap (handler-case (signal 'file-error :path "z.lisp")
                (parse-error (e) "innen"))
              (lambda (e) "gefangen")))))

;; --- Go-Fehler → lisp-error-Fallback ---------------------------------

(deftest go-fehler-fallback :suite 'condition
  (is (equal? t
        (handler-case (file-read "/gibts/es/nicht.txt")
          (lisp-error (e) (if (string-contains (lisp-error-msg e) "nicht gefunden") t ())))))
  ;; Fallback matcht auch über Eltern (error, condition)
  (is (equal? "via-error"
        (handler-case (file-read "/gibts/es/nicht.txt")
          (error (e) "via-error"))))
  ;; klassischer (error "...")-Wurf landet ebenfalls als lisp-error
  (is (equal? "klassisch"
        (handler-case (error "boom")
          (lisp-error (e) "klassisch")))))

;; --- Klausel ohne Var-Bindung, normaler Wert -------------------------

(deftest klausel-varianten :suite 'condition
  (is (equal? "ohne-var"
        (handler-case (signal 'lisp-error :msg "x")
          (lisp-error () "ohne-var"))))
  (is (equal? 3
        (handler-case (+ 1 2)
          (error (e) "nie")))))

;; --- Validierung ------------------------------------------------------

(deftest validierung :suite 'condition
  (is (equal? 'err (trap (signal 'gibts-nicht-als-typ) (lambda (e) 'err))))
  (is (equal? 'err (trap (eval '(define-condition 42 (error) ())) (lambda (e) 'err))))
  (is (equal? 'err (trap (eval '(define-condition bad-cond (error) ("string-slot"))) (lambda (e) 'err)))))

;; --- Reload-Semantik: Neudefinition ersetzt still ---------------------

(deftest reload-ersetzt :suite 'condition
  (eval '(define-condition reload-me (error) (a)))
  (eval '(define-condition reload-me (error) (a b)))
  (is (equal? "b-wert"
        (handler-case (signal 'reload-me :a 1 :b "b-wert")
          (reload-me (e) (reload-me-b e))))))
