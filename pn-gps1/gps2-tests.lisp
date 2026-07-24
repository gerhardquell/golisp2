;; ********************************************************************
;; pn-gps1/gps2-tests.lisp – Tests für GPS Version 2 (state-passing)
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k2.7-code
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260714
;; ********************************************************************
;; Zeigt, dass Norvigs drei Version-1-Fehler in der state-passing-Form
;; nicht mehr auftreten, und deckt Erfolgs-/Scheitervarianten ab.
;; ********************************************************************

(load "tests/test-helpers.lisp")  ; assert= — eine Quelle für alle Testdateien

;; gps2 ersetzt bewusst die v1-Funktionen aus gps.lisp (andere Arity und
;; Semantik: state-passing statt globalem *state*). Der Override ist
;; Absicht — darum hier explizit erlaubt statt still gewarnt.
(redefine-policy 'allow)
(load "pn-gps1/gps2.lisp")
(redefine-policy 'warn)

;; === Erfolgsfälle ===================================================

(assert=
  '((start) (executing look-up-number) (executing telephone-shop)
    (executing tell-shop-problem) (executing give-shop-money)
    (executing shop-installs-battery) (executing drive-son-to-school))
  (gps2 '(son-at-home car-needs-battery have-money have-phone-book)
        '(son-at-school)
        *school-ops2*))

(assert=
  '((start) (executing drive-son-to-school))
  (gps2 '(son-at-home car-works)
        '(son-at-school)
        *school-ops2*))

;; === Scheitern ======================================================

(assert= ()
  (gps2 '(son-at-home car-needs-battery have-money)
        '(son-at-school)
        *school-ops2*))

;; === Version-1-Fehler sind behoben ==================================

;; 1. Clobbered Sibling Goal: have-money wird gelöscht, bevor die
;;    finale subsetp-Prüfung greift → gps2 gibt nil zurück.
(assert= ()
  (gps2 '(son-at-home car-needs-battery have-money have-phone-book)
        '(have-money son-at-school)
        *school-ops2*))

;; 2. Leaping before you look: Ziele in umgekehrter Reihenfolge führen
;;    zwar zu Aktionen, am Ende fehlt have-money → nil, ohne Seiteneffekt.
(assert= ()
  (gps2 '(son-at-home car-needs-battery have-money have-phone-book)
        '(son-at-school have-money)
        *school-ops2*))

;; 3. Rekursives Unterziel: goal-stack bricht die Rekursion sofort ab.
(assert= ()
  (gps2 '(have-money)
        '(know-phone-number)
        (cons (make-op :action 'ask-phone-number
                       :preconds '(know-phone-number)
                       :add-list '(know-phone-number))
              *school-ops2*)))

"gps2-tests: alle PASS"
