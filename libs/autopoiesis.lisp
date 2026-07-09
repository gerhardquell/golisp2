;**********************************************************************
;  autopoiesis.lisp
;  Autor    : Gerhard Quell - gquell@skequell.de
;  CoAutor  : claude sonnet 4.6
;  Copyright: 2026 Gerhard Quell - SKEQuell
;  Erstellt : 20260225
;**********************************************************************
;
; Autopoietische Schleife - Selbst-modifizierender Prompt
; Der Prompt analysiert sich selbst und wird iterativ besser
;
; Usage: (autopoiesis "initial-prompt" iterations)
;**********************************************************************

; Iterationszaehler
(define *autopoiesis-iteration* 0)

; Die autopoietische Funktion (Lambda-Version fuer GoLisp)
(define autopoiesis
  (lambda (initial-prompt max-iterations)
    "Fuehrt einen selbst-verbessernden Prompt durch n Iterationen"
    (if (= max-iterations 0)
        initial-prompt
        (begin
          (set! *autopoiesis-iteration* (+ *autopoiesis-iteration* 1))
          (println "")
          (println (string-append "=== ITERATION "
                                  (number->string *autopoiesis-iteration*)
                                  " ==="))

          ; KI analysiert und verbessert den Prompt
          (define improved-prompt
            (sigo (string-append
                   "Als Prompt-Engineer: Analysiere und verbessere DIESEN PROMPT. "
                   "Der neue Prompt soll spezifischer, praeziser und effektiver sein. "
                   "Gib NUR den verbesserten Prompt zurueck (1-2 Saetze), keine Erklaerungen: "
                   initial-prompt)
                  "claude-h"))

          (println "Verbesserter Prompt:")
          (println improved-prompt)

          ; Rekursiv mit verringerter Iteration
          (autopoiesis improved-prompt (- max-iterations 1))))))

; Hilfsfunktion: Generiert Code mit finalem Prompt
(define autopoiesis-code
  (lambda (final-prompt)
    "Generiert Lisp-Code mit dem finalen autopoietischen Prompt"
    (println "")
    (println "=== CODE-GENERIERUNG ===")
    (define code
      (sigo (string-append
             "Schreibe NUR Lisp-Code, keine Erklaerungen, kein Markdown. "
             "Nur reiner Code der direkt mit (eval (read ...)) ausfuehrbar ist: "
             final-prompt)
            "claude-h"))
    (println "Generierter Code:")
    (println code)
    code))

; Beispiel-Usage:
;(define start "Schreibe eine Funktion die Fibonacci berechnet.")
;(define final-prompt (autopoiesis start 3))
;(define code (autopoiesis-code final-prompt))
;(eval (read code))
;(println (fibonacci 10))
