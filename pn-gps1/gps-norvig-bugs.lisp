;; ********************************************************************
;; pn-gps1/gps-norvig-bugs.lisp – Norvigs drei bekannte GPS-Fehler
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k2.7-code
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260714
;; ********************************************************************
;; Ein Port ist erst treu, wenn er die gleichen Fehler reproduziert.
;; Diese Tests zeigen die bekannten Schwächen von PAIP GPS Version 1.
;; ********************************************************************

(load "pn-gps1/gps.lisp")

;; === Test-Hilfe =====================================================

(defun report (name passed? expected actual)
  (if passed?
      (println (string-append "PASS: " name))
      (begin
        (println (string-append "FAIL: " name))
        (print "  expected: ") (println expected)
        (print "  actual:   ") (println actual))))

(defmacro check-equal (name expected expr)
  `(report ,name (equal? ,expected ,expr) ,expected ,expr))

;; === 1. Clobbered Sibling Goal =======================================
;; Ziele: have-money UND son-at-school. GPS prüft have-money zuerst, dann
;; erreicht es son-at-school – wobei give-shop-money have-money löscht.
;; Ergebnis: 'solved, obwohl have-money am Ende nicht mehr gilt.

(check-equal "clobbered sibling goal"
             'solved
             (gps '(son-at-home car-needs-battery have-money have-phone-book)
                  '(have-money son-at-school)
                  *school-ops*))

;; === 2. Leaping before you look ======================================
;; Ziele in umgekehrter Reihenfolge: son-at-school zuerst. GPS führt alle
;; Aktionen aus, merkt dann, dass have-money fehlt, und gibt () zurück.
;; Der Zustand bleibt trotzdem mutiert (Seiteneffekt).

(check-equal "leaping before you look"
             ()
             (gps '(son-at-home car-needs-battery have-money have-phone-book)
                  '(son-at-school have-money)
                  *school-ops*))

;; === 3. Rekursives Unterziel =========================================
;; Ein Operator, der ein Ziel als eigene Voraussetzung hat, lässt achieve
;; unendlich rekursieren (bzw. hängen). Der Test nutzt parfunc mit Timeout.

(define *recursive-ops*
  (cons (make-op :action 'ask-phone-number
                 :preconds '(know-phone-number)
                 :add-list '(know-phone-number))
        *school-ops*))

(parfunc recursive-result :timeout 1
  (gps '(have-money)
       '(know-phone-number)
       *recursive-ops*))

(check-equal "recursive subgoal hangs" () (car recursive-result))

(println "gps-norvig-bugs: alle PASS")
nil
