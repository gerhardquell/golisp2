;; ********************************************************************
;; pn-gps1/gps2.lisp – GPS Version 2 (Norvig, PAIP Kap. 4.11)
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k2.7-code
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260714
;; ********************************************************************
;; State-Passing-Variante von GPS: keine globale *state*-Mutation.
;; Jede Funktion nimmt den aktuellen Zustand entgegen und gibt einen
;; neuen Zustand zurück (oder nil, wenn das Ziel nicht erreichbar ist).
;;
;; Unterschied zum PAIP-Original: golisp2 ist lexikalisch gebunden,
;; daher wird der Operator-Satz explizit durchgereicht (Parameter ops),
;; statt über Special-Variable *ops* dynamisch gebunden zu werden.
;; Das macht die Variante zusätzlich goroutine-tauglich.
;; ********************************************************************

(defvar *ops* nil "Verfügbare Operatoren (Default für gps2).")

(defstruct op "Eine Operation"
  (action nil) (preconds nil) (add-list nil) (del-list nil))

;; === GPS-Kern =======================================================

(defun gps2 (state goals &optional (ops *ops*))
  "General Problem Solver: erreiche goals aus state mit ops.
   Rückgabe ist die Liste der ausgeführten Aktionen (inkl. (start))."
  (remove-if #'atom (achieve-all (cons '(start) state) goals nil ops)))

(defun achieve-all (state goals goal-stack ops)
  "Erreiche jedes Ziel und stelle sicher, dass alle am Ende noch gelten."
  (let ((current-state state))
    (if (and (every (lambda (g)
                      (setf current-state
                            (achieve current-state g goal-stack ops)))
                    goals)
             (subsetp goals current-state))
        current-state)))

(defun achieve (state goal goal-stack ops)
  "Ein Ziel ist erreicht, wenn es bereits im Zustand ist, oder wenn es
   eine passende, anwendbare Operation gibt. goal-stack erkennt Zyklen."
  (cond ((member goal state) state)
        ((member goal goal-stack) nil)
        (t (some (lambda (op)
                   (apply-op state goal op goal-stack ops))
                 (find-all goal ops :test #'appropriate-p)))))

(defun appropriate-p (goal op)
  "Ein Operator ist geeignet, wenn das Ziel in seiner Add-List steht."
  (member goal (op-add-list op)))

(defun apply-op (state goal op goal-stack ops)
  "Gib einen neuen, transformierten Zustand zurück, wenn op anwendbar ist."
  (let ((state2 (achieve-all state (op-preconds op)
                             (cons goal goal-stack) ops)))
    (unless (null? state2)
      (append (remove-if (lambda (x)
                           (member x (op-del-list op)))
                         state2)
              (op-add-list op)))))

;; === Hilfsfunktionen ================================================

(defun some (pred lst)
  "Wie any, gibt aber das erste nicht-nil Ergebnis zurück."
  (if (null lst)
      ()
      (let ((r (pred (car lst))))
        (if r r (some pred (cdr lst))))))

(defun remove-if (pred lst)
  "Behält alle Elemente, für die pred falsch ist."
  (filter (lambda (x) (not (pred x))) lst))

(defun subsetp (subset set)
  "T, wenn jedes Element von subset in set vorkommt (test: equal?)."
  (every (lambda (x) (member x set)) subset))

;; === Domain: Fahrt zur Kindertagesstätte ============================
;; Version-2-Operatoren tragen (executing action) in die Add-List ein.
;; Dadurch bleiben im Ergebnis die ausgeführten Aktionen erhalten,
;; während atomare Bedingungen durch remove-if herausfallen.

(define *school-ops2*
  (list
    (make-op :action 'drive-son-to-school
             :preconds '(son-at-home car-works)
             :add-list '(son-at-school (executing drive-son-to-school))
             :del-list '(son-at-home))
    (make-op :action 'shop-installs-battery
             :preconds '(car-needs-battery shop-knows-problem shop-has-money)
             :add-list '(car-works (executing shop-installs-battery)))
    (make-op :action 'tell-shop-problem
             :preconds '(in-communication-with-shop)
             :add-list '(shop-knows-problem (executing tell-shop-problem)))
    (make-op :action 'telephone-shop
             :preconds '(know-phone-number)
             :add-list '(in-communication-with-shop (executing telephone-shop)))
    (make-op :action 'look-up-number
             :preconds '(have-phone-book)
             :add-list '(know-phone-number (executing look-up-number)))
    (make-op :action 'give-shop-money
             :preconds '(have-money)
             :add-list '(shop-has-money (executing give-shop-money))
             :del-list '(have-money))))

nil
