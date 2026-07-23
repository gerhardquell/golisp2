; test-workaround.lisp - Testet Workaround für sigoREST Verbindungsprobleme
; Autor: Claude
; Erstellt: 2025-02-26

(load "tests/demo-utils.lisp")

(print-header "Workaround-Test für sigoREST")
(println "Testet defensive Programmierung gegenüber Verbindungsabbrüchen")
(println "")

;; ============================================================================
;; Workaround-Funktionen
;; ============================================================================

(defun safe-sigo (prompt model)
  "Sichere sigo-Aufruf mit Error-Handling"
  (trap
    (sigo prompt model)
    (lambda (e)
      (println (string-append "  [Warnung] " model " fehlgeschlagen: " e))
      nil)))

(defun check-model (model)
  "Prüft ob ein Modell verfügbar ist"
  (not (null? (safe-sigo "test" model))))

;; ============================================================================
;; Test 1: Einzelne Modell-Prüfung
;; ============================================================================

(println "Test 1: Modell-Verfügbarkeit prüfen")
(println "------------------------------------")
(println "Hinweis: catch/safe-sigo funktioniert nicht mit sigo-Fehlern")
(println "         (sigo wirft fatalen Error, nicht catchbar)")
(println "         Daher: Nur bekannte funktionierende Modelle testen")
(println "")

;; Bekannte funktionierende Modelle (aus vorherigen Tests)
(define available-models '("gemini-p" "kimi"))

(println "Bekannte funktionierende Modelle:")
(println "  ✓ gemini-p (getestet)")
(println "  ✓ kimi (getestet)")
(println "  ✗ claude-h (Circuit Breaker offen)")
(println "  ✗ gpt41 (API Fehler)")
(println "  ✗ mistral-l3 (nicht getestet)")
(println "")

;; ============================================================================
;; Test 2: Parallele Anfragen (mit bekannten funktionierenden Modellen)
;; ============================================================================

(print-header "Test 2: Parallele Anfragen")

(define prompt "Erkläre in einem Satz was Homoikonizität bedeutet.")

(println (string-append "Prompt: " prompt))
(println "")

(println "ACHTUNG: sigo-Fehler sind nicht catchbar!")
(println "         Workaround: Nur bekannte funktionierende Modelle verwenden")
(println "")

(println "Sende Anfragen an gemini-p und kimi...")
(println "")

;; Da catch nicht funktioniert, verwenden wir nur Modelle, die funktionieren
(parfunc results
  (sigo prompt "gemini-p")
  (sigo prompt "kimi"))

(println "Erhaltene Antworten:")
(println (string-append "  Anzahl: " (number->string (length results))))
(println "")

;; Zeige Ergebnisse
(if (>= (length results) 1)
    (begin
      (println "Antwort von Gemini:")
      (println "-------------------")
      (println (car results))
      (println "")))

(if (>= (length results) 2)
    (begin
      (println "Antwort von Kimi:")
      (println "-----------------")
      (println (cadr results))
      (println "")))

;; ============================================================================
;; Test 3: Sequentielle Ausführung mit Delay
;; ============================================================================

(print-header "Test 3: Sequentielle Ausführung (Fallback)")

(println "Falls parallele Ausführung fehlschlägt:")
(println "  1. Anfrage an erstes Modell")
(println "  2. 1 Sekunde warten")
(println "  3. Anfrage an zweites Modell")
(println "")

(println "Anfrage 1 an gemini-p...")
(define result1 (sigo "Was ist 2+2?" "gemini-p"))
(println "Antwort 1 erhalten:")
(println result1)
(println "")

;; Nur falls erste fehlgeschlagen (nicht erreichbar mit sigo-Fehlern)
;; Stattdessen: Sequentielle Ausführung demonstrieren
(println "Anfrage 2 an kimi...")
(define result2 (sigo "Was ist 2+2?" "kimi"))
(println "Antwort 2 erhalten:")
(println result2)
(println "")

;; ============================================================================
;; Zusammenfassung
;; ============================================================================

(print-header "Workaround-Test abgeschlossen")

(println "GETESTET:")
(println "  ✓ parfunc mit gemini-p und kimi funktioniert")
(println "  ✓ Sequentielle Ausführung möglich")
(println "")

(println "PROBLEM ENTDECKT:")
(println "  ✗ catch funktioniert NICHT mit sigo-Fehlern")
(println "    (sigo wirft fatalen Error, nicht abfangbar)")
(println "  ✗ claude-h, gpt41 haben Circuit Breaker/API-Fehler")
(println "")

(println "WORKAROUND (aktuell):")
(println "  → Nur bekannte funktionierende Modelle verwenden:")
(println "    - gemini-p (Google)")
(println "    - kimi (Moonshot)")
(println "  → Vermeide: claude-h, gpt41, deepseek-v3")
(println "")

(println "EMPFEHLUNG:")
(println "  1. sigoREST verbessern (siehe sigorest-action-items.md)")
(println "  2. ODER: Fehler in GoLisp catch-bar machen")
(println "  3. ODER: Health Check Endpoint nutzen (wenn verfügbar)")
(println "")
