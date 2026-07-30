;; ********************************************************************
;; tests/test-framework.lisp — Mini-Test-Framework (FiveAM-orientiert)
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k3
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260730
;; ********************************************************************
;; Nachfolger von assert= (tests/test-helpers.lisp): Fehler werden
;; gesammelt statt den Load abzubrechen, Abschluss-Report, Exit-relevant.
;;
;;   (defsuite 'mengen)
;;   (deftest union-basis :suite 'mengen
;;     (is (equal? '(1 2 3) (union '(1 2) '(2 3)))))
;;   (deftest bekannter-bug :suite 'mengen :expected-failure
;;     (is (equal? 'x (buggy-fn))))
;;
;;   (run-tests)           ; alle Suiten → Anzahl FAILs (0 = alles grün)
;;   (run-tests 'mengen)   ; nur eine Suite
;;
;; Kategorien: PASS / FAIL / XFAIL (erwarteter Fehlschlag) /
;;             XPASS (unerwartet bestanden — Hinweis, zählt nicht als FAIL)
;; Typisch: (exit (run-tests)) → Prozess-Exit-Code = FAIL-Anzahl.
;; Mehrfach-Load harmlos (gleiche Quelle → stiller Reload).
;; ********************************************************************

;; === Zustand =========================================================
;; defvar statt define: Mehrfach-Load (jede Testdatei lädt das Framework)
;; darf Registry und Zähler NICHT zurücksetzen.

(defvar *tf-suites* '())    ; Suite-Namen in Definitionsreihenfolge
(defvar *tf-registry* '())  ; (name suite thunk xfail?) in Def-Reihenfolge

(defvar *tf-pass* 0)
(defvar *tf-fail* 0)
(defvar *tf-xfail* 0)
(defvar *tf-xpass* 0)
(defvar *tf-failures* '())  ; (test expr detail) je FAIL

(defvar *tf-current* '())        ; Name des laufenden Tests
(defvar *tf-xfail-mode* '())     ; t während xfail-Tests
(defvar *tf-test-had-fail* '())  ; t wenn aktueller Test einen Fehlschlag hatte

;; Eindeutiges Fehler-Tag (gensym → keine Kollision mit Testwerten möglich)
(defvar %tf-err-tag (gensym "TF-ERR"))

;; === Interne Helfer ==================================================

;; %tf-unquote: 'sym → sym, sonst unverändert (Makro-Zeit-Helfer)
(defun %tf-unquote (form)
  (if (and (pair? form) (equal? (car form) 'quote))
      (cadr form)
      form))

;; %tf-error-result?: ist r ein getaggtes Fehler-Ergebnis?
(defun %tf-error-result? (r)
  (and (pair? r) (equal? (car r) %tf-err-tag)))

;; %tf-register: Test eintragen; gleicher Name ersetzt still
;; (Reload-Semantik, wie defsystem)
(defun %tf-register (name suite thunk xfail)
  (set! *tf-registry*
        (append (filter (lambda (e) (not (equal? (car e) name))) *tf-registry*)
                (list (list name suite thunk xfail))))
  name)

;; %tf-check-fail: ein Check ist fehlgeschlagen
(defun %tf-check-fail (expr detail)
  (set! *tf-test-had-fail* t)
  (if *tf-xfail-mode*
      (println (format nil "  xfail: ~a — ~a" expr detail))
      (begin
        (set! *tf-fail* (+ *tf-fail* 1))
        (set! *tf-failures*
              (append *tf-failures* (list (list *tf-current* expr detail))))
        (println (format nil "  FAIL: ~a — ~a" expr detail)))))

;; %tf-record: Ergebnis eines is-Checks verbuchen
(defun %tf-record (expr r)
  (cond ((%tf-error-result? r)
         (%tf-check-fail expr (format nil "Fehler: ~a" (cadr r))))
        (r
         (set! *tf-pass* (+ *tf-pass* 1))
         (println (format nil "  PASS: ~a" expr)))
        (t
         (%tf-check-fail expr "ergab ()"))))

;; === API =============================================================

;; defsuite: (defsuite 'name) — registriert eine Suite (idempotent)
(defun defsuite (name)
  (if (member name *tf-suites*)
      name
      (begin
        (set! *tf-suites* (append *tf-suites* (list name)))
        name)))

;; deftest: (deftest name [:suite 's] [:expected-failure] body...)
;; Registriert den Test, läuft NICHT sofort. Reihenfolge = Def-Reihenfolge.
(defmacro deftest (name &rest args)
  (let ((suite '())
        (xfail '())
        (rest-args args))
    ;; Keywords am Kopf abziehen, Rest ist Body
    (while (and (not (null rest-args))
                (or (equal? (car rest-args) :suite)
                    (equal? (car rest-args) :expected-failure)))
      (if (equal? (car rest-args) :suite)
          (begin
            (set! suite (%tf-unquote (cadr rest-args)))
            (set! rest-args (cddr rest-args)))
          (begin
            (set! xfail t)
            (set! rest-args (cdr rest-args)))))
    (if (and suite (not (member suite *tf-suites*)))
        (error (format nil "deftest: Suite '~a' nicht definiert — defsuite zuerst" suite)))
    `(%tf-register ',name ',suite (lambda () ,@rest-args) ',xfail)))

;; is: (is expr) — ein Check. Wahr = PASS, () oder Fehler = FAIL
;; (bzw. xfail im :expected-failure-Kontext). Bricht niemals ab.
(defmacro is (expr)
  `(let ((r (trap ,expr (lambda (e) (list %tf-err-tag (format nil "~a" e))))))
     (%tf-record ',expr r)))

;; run-tests: (run-tests ['suite]) → läuft Tests (gefiltert), druckt
;; Report, liefert Anzahl FAILs (0 = alles grün).
(defun run-tests (&optional (suite '()))
  (set! *tf-pass* 0)
  (set! *tf-fail* 0)
  (set! *tf-xfail* 0)
  (set! *tf-xpass* 0)
  (set! *tf-failures* '())
  (println (if suite
               (format nil "=== Tests (Suite: ~a) ===" suite)
               "=== Tests (alle Suiten) ==="))
  (dolist (entry *tf-registry* ())
    (let ((name        (car entry))
          (entry-suite (cadr entry))
          (thunk       (caddr entry))
          (xfail       (cadddr entry)))
      (when (or (null suite) (equal? entry-suite suite))
        (set! *tf-current* name)
        (set! *tf-xfail-mode* xfail)
        (set! *tf-test-had-fail* '())
        (println (format nil "~a:" name))
        (let ((r (trap (funcall thunk)
                       (lambda (e) (list %tf-err-tag (format nil "~a" e))))))
          ;; Fehler im Rumpf außerhalb eines is (Setup o. ä.)
          (when (%tf-error-result? r)
            (set! *tf-test-had-fail* t)
            (when (not xfail)
              (set! *tf-fail* (+ *tf-fail* 1))
              (set! *tf-failures*
                    (append *tf-failures* (list (list name '(test-rumpf) (cadr r))))))
            (println (format nil "  FEHLER im Test-Rumpf: ~a" (cadr r))))
          (when xfail
            (if *tf-test-had-fail*
                (begin
                  (set! *tf-xfail* (+ *tf-xfail* 1))
                  (println "  → XFAIL (erwartet)"))
                (begin
                  (set! *tf-xpass* (+ *tf-xpass* 1))
                  (println "  → XPASS (unerwartet bestanden!)"))))))))
  (println "")
  (println (format nil "=== Report: ~a PASS, ~a FAIL, ~a XFAIL, ~a XPASS ==="
                   *tf-pass* *tf-fail* *tf-xfail* *tf-xpass*))
  (when (not (null *tf-failures*))
    (println "Fehlschläge:")
    (dolist (f *tf-failures* ())
      (println (format nil "  ~a: ~a — ~a" (car f) (cadr f) (caddr f)))))
  *tf-fail*)
