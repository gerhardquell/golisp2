; test-workaround.lisp - Testet Workaround für sigoREST Verbindungsprobleme
; Autor: Claude
; Erstellt: 2025-02-26
;
; Seit 20261117 umgedreht: sigo-Fehler sind abfangbar (trap), der frühere
; „Workaround" (nur bekannte Modelle benutzen) ist nicht mehr nötig.
; Macht 2 echte sigo-Aufrufe (ein paar Cent); bricht mit Fehler ab, wenn
; eine Erwartung nicht stimmt.

(load "tests/demo-utils.lisp")

(define live-model "che-gem3-f")
(define dead-model "gibt-es-nicht")

(defun safe-sigo (prompt model &optional (host ""))
  "sigo-Aufruf, der bei Fehler warnt und () liefert"
  (trap
    (if (equal? host "") (sigo prompt model) (sigo prompt model "" host))
    (lambda (e)
      (println (string-append "  [Warnung] " model " fehlgeschlagen: " e))
      nil)))

(defun check-model (model)
  "t, wenn das Modell antwortet"
  (not (null? (safe-sigo "Antworte nur mit: ok" model))))

(defun expect (ok text)
  (if ok
      (println (string-append "  ✓ " text))
      (error (string-append "test-workaround: " text " — stimmt nicht"))))

(print-header "Test 1: Modell-Verfügbarkeit mit trap")
(expect (check-model live-model) (string-append live-model " antwortet"))
(expect (not (check-model dead-model)) (string-append dead-model " wird abgefangen"))

(print-header "Test 2: sigoREST nicht erreichbar")
(expect (null? (safe-sigo "test" live-model "http://127.0.0.1:1"))
        "Verbindungsfehler wird abgefangen")

(print-header "Test 3: parfunc mit einem scheiternden Zweig")
(parfunc results
  (sigo "Antworte nur mit: ok" live-model)
  (sigo "test" dead-model))
(expect (string? (car results)) "lebender Zweig liefert seine Antwort")
(expect (eq (car (cadr results)) :error) "toter Zweig liefert (:error \"…\")")

(print-header "Workaround-Test abgeschlossen")
