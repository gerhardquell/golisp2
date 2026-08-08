;; ********************************************************************
;; tests/webbridge.lisp — Integrationstest Web-Bridge (Spec TODO.md §8.3)
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k3
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260807
;; ********************************************************************
;; Prueft die Lisp-Sicht ohne Browser: Server-Objekt, Export/Unexport,
;; Emit ohne Clients, idempotentes Stop, ws-call auf Phantom-Client.
;; Die Szenarien §8.4 A/B sind durch lib/wsbridge_test.go automatisiert.

(load "tests/test-framework.lisp")

(defsuite 'webbridge)

(deftest serve-port :suite 'webbridge
  (let ((srv (http-serve 0)))
    ;; Port 0 → OS vergibt freien Port > 0
    (is (> (http-port srv) 0))
    ;; http-port konsistent
    (is (= (http-port srv) (http-port srv)))
    ;; ws-export/-unexport-Rueckgaben
    (is (eq t (ws-export srv "ping" (lambda (c) "pong"))))
    (is (eq t (ws-export srv 'ping2 (lambda (c) "pong"))))  ; ATOM-Name
    (is (eq t (ws-unexport srv "ping")))
    (is (eq t (ws-unexport srv 'ping2)))
    (is (null (ws-unexport srv "ping")))                    ; schon weg → nil
    ;; ws-emit ohne Clients → 0 Empfaenger
    (is (= 0 (ws-emit srv 'tick 42)))
    ;; ws-clients ohne Clients → leere Liste
    (is (null (ws-clients srv)))
    ;; ws-call auf unbekannten Client → Fehler, kein Haenger
    (is (eq :gefangen
            (handler-case (ws-call srv 999 "1")
              (error (e) :gefangen))))
    ;; http-stop idempotent
    (is (eq t (http-stop srv)))
    (is (eq t (http-stop srv)))))

(deftest serve-fehlerfaelle :suite 'webbridge
  ;; kein Server-Objekt → Lisp-Error
  (is (eq :gefangen
          (handler-case (http-port 42)
            (error (e) :gefangen))))
  (is (eq :gefangen
          (handler-case (ws-emit 42 'tick 1)
            (error (e) :gefangen)))))

(deftest webserv-smoke :suite 'webbridge
  ;; Inline-HTML-Modus, kein Browser (Test-Umgebung)
  (let ((srv (webserv :port 0 :html "<html><head></head><body>hi</body></html>" :open ())))
    (is (> (http-port srv) 0))
    ;; ws-export funktioniert unveraendert auf dem webserv-Server-Objekt
    (is (eq t (ws-export srv "ping" (lambda (c) "pong"))))
    (is (eq t (http-stop srv))))
  ;; Fehlerfall: weder :html noch :htmlpath
  (is (eq :gefangen
          (handler-case (webserv :port 0 :open ())
            (error (e) :gefangen)))))

(let ((fails (run-tests 'webbridge)))
  (exit (if (> fails 0) 1 0)))
