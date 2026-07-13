;; ********************************************************************
;; pn-gps1/gps.lisp – General Problem Solver (Norvig, PAIP Kap. 4)
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : glm-5.2
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260713
;; ********************************************************************
;; Port von Peter Norvigs GPS (Paradigms of Artificial Intelligence
;; Programming, Kapitel 4) nach golisp2. Spec: docs/gps1.txt.
;;
;; Mittel-Ziel-Analyse (means-ends analysis): ein Ziel erreichen, indem
;; man den Unterschied zwischen Ist- und Soll-Zustand durch einen
;; geeigneten Operator beseitigt – ggf. dessen Voraussetzungen vorher als
;; Unterziele löst.
;;
;; Anmerkung zum Scope-Port: Norvigs (defun GPS (*state* goals *ops*) …)
;; nutzt CL-Special-Variablen (dynamische Bindung). golisp2 ist lexikalisch
;; – der Parameter *state* würde achieve nicht erreichen. Daher nimmt GPS
;; Plain-Namen und weist die Globals *state*/*ops* via set! zu. Semantik
;; bleibt erhalten, nur die Schreibweise weicht leicht ab.

;; === Globale Zustandsvariablen ================================

(defvar *state* nil "Aktuelle Zustand: Liste von Bedingungen.")
(defvar *ops*   nil "Verfügbare Operatoren.")

;; === Struktur: Operator =======================================

;; Ein Operator: Aktion, Voraussetzungen, Add-List, Del-List.
;; (defstruct generiert make-op, op-action, op-preconds, op-add-list,
;; op-del-list und das Prädikat op?.)
(defstruct op "Eine Operation"
  (action nil) (preconds nil) (add-list nil) (del-list nil))

;; === GPS-Kern =================================================

;; gps: löse alle Ziele aus dem Startzustand mit den Operatoren.
(defun gps (state goals ops)
  (set! *state* state)
  (set! *ops*   ops)
  (if (every achieve goals) 'solved))

;; achieve: ein Ziel ist erreicht, wenn es schon zutrifft, oder wenn ein
;; geeigneter, anwendbarer Operator existiert.
(defun achieve (goal)
  (or (member goal *state*)
      (any apply-op (find-all goal *ops* :test #'appropriate-p))))

;; appropriate-p: ein Operator ist für ein Ziel geeignet, wenn das Ziel
;; in seiner Add-List steht.
(defun appropriate-p (goal op)
  (member goal (op-add-list op)))

;; apply-op: Operator anwenden, sobald alle Voraussetzungen erfüllbar
;; sind. Gibt t zurück, wenn angewendet (Prädikat), sonst ().
(defun apply-op (op)
  (when (every achieve (op-preconds op))
    (println (list 'executing (op-action op)))
    (set! *state* (set-difference *state* (op-del-list op)))
    (set! *state* (union *state* (op-add-list op)))
    t))

;; === Domain: Fahrt zur Kindertagesstätte =====================

(define *school-ops*
  (list
    (make-op :action 'drive-son-to-school
             :preconds '(son-at-home car-works)
             :add-list '(son-at-school)
             :del-list '(son-at-home))
    (make-op :action 'shop-installs-battery
             :preconds '(car-needs-battery shop-knows-problem shop-has-money)
             :add-list '(car-works))
    (make-op :action 'tell-shop-problem
             :preconds '(in-communication-with-shop)
             :add-list '(shop-knows-problem))
    (make-op :action 'telephone-shop
             :preconds '(know-phone-number)
             :add-list '(in-communication-with-shop))
    (make-op :action 'look-up-number
             :preconds '(have-phone-book)
             :add-list '(know-phone-number))
    (make-op :action 'give-shop-money
             :preconds '(have-money)
             :add-list '(shop-has-money)
             :del-list '(have-money))))

;; === Testfälle (Norvig, Spec Zeile 213–231) ==================
;; Datei-Modus druckt nur das Ergebnis der LETZTEN Top-Level-Form.
;; Daher Fall 1 und 2 explizit in (print …); Fall 3 läuft nackt und
;; wird vom Loader-Ergebnis gedruckt.

;; Fall 1: Telefonbuch vorhanden – volle Kette rückwärts.
(println
  (gps '(son-at-home car-needs-battery have-money have-phone-book)
       '(son-at-school)
       *school-ops*))

;; Fall 2: kein Telefonbuch – look-up-number scheitert, GPS gibt () zurück.
(println
  (gps '(son-at-home car-needs-battery have-money)
       '(son-at-school)
       *school-ops*))

;; Fall 3: Auto funktioniert – direkte Fahrt.
(gps '(son-at-home car-works)
     '(son-at-school)
     *school-ops*)
