;; ***********************************************************************
;;  tests/golisp2web-test.lisp
;;  Autor    : Gerhard Quell - gquell@skequell.de
;;  CoAutor  : Claude Sonnet 5
;;  Copyright: 2026 Gerhard Quell - SKEQuell
;;  Erstellt : 20260821
;; ***********************************************************************
;; Integrationstest: startet golisp2web (Python-GUI-Client, separates
;; Repo golisp2web/) als externen Prozess via (system ...) und prueft die
;; Web-Bridge end-to-end -- Verbindung ueber ws-clients, dann
;; ferngesteuertes Beenden ueber die golisp2web-quit-Konvention
;; (golisp2web-Commit 985b9e8).
;;
;; parfunc faechert in drei Branches auf, die denselben Frame-Env teilen
;; (verifiziert mit -race, siehe lib/env.go) und erst gemeinsam
;; zurueckkehren, wenn ALLE fertig sind (Fork-Join, lib/eval_control.go):
;;   1. http-wait  -- haelt den Server blockierend am Leben
;;   2. system     -- startet golisp2web, blockiert bis es beendet wird,
;;                    ruft danach http-stop (loest Branch 1)
;;   3. Testlogik  -- pollt auf einen verbundenen WS-Client, sendet dann
;;                    "golisp2web-quit" (loest Branch 2)
;;
;; Braucht ein echtes X11-Display (golisp2web.py bindet QT_QPA_PLATFORM
;; hart auf xcb) -- kein Teil der normalen -t-Testsuite (main.go
;; runTests), laeuft nie headless/CI. Aufruf vom Repo-Root:
;;   ./build/golisp2 tests/golisp2web-test.lisp
;;
;; WICHTIG: vorher sicherstellen, dass keine andere golisp2web-Instanz
;; laeuft (ps aux | grep golisp2web.py) -- zwei gleichzeitige Instanzen
;; haengen reproduzierbar fest (vermutlich geteiltes QtWebEngine-
;; Profilverzeichnis), Branch 2 kehrt dann nie zurueck, parfunc wartet
;; ewig. Details: doc/emacs-golisp2web.md, Abschnitt "Grenzen".
;; ***********************************************************************

(load "tests/test-framework.lisp")

(defsuite 'golisp2web)

;; %g2w-wait-for-client: pollt ws-clients bis ein Client verbunden ist
;; oder tries aufgebraucht sind (je 500ms). t = verbunden, () = Timeout.
(defun %g2w-wait-for-client (s tries)
  (if (<= tries 0)
      ()
      (if (ws-clients s)
          t
          (begin (sleep 500) (%g2w-wait-for-client s (- tries 1))))))

(deftest golisp2web-verbindet-und-beendet-sich-fern :suite 'golisp2web
  (let ((s (webserv :port 0 :host "127.0.0.1"
                     :html "<html><head></head><body>golisp2web-integrationstest</body></html>"
                     :open nil))
        (exitcd -1)
        (verbunden ()))
    (let ((cmd (format nil "cd golisp2web && python3 golisp2web.py -t1 localhost:~a"
                        (http-port s))))
      (parfunc ergebnis
        (http-wait s)
        (begin
          (set! exitcd (system cmd))
          (http-stop s))
        (begin
          (set! verbunden (%g2w-wait-for-client s 20))
          (if verbunden (ws-emit s "golisp2web-quit" ())))))
    (is verbunden)
    (is (>= exitcd 0))))

(exit (run-tests 'golisp2web))
