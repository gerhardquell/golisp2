;; ********************************************************************
;; tests/endlosrekursion.lisp – Endlose Rekursion bricht sauber ab
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : claude-opus-5.5
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20261113
;; ********************************************************************
;; Ersatz für pn-gps1/gps-norvig-bugs.lisp (Norvigs Bug 3, rekursives
;; Unterziel): eine nicht-tail-rekursive Endlosschleife in parfunc darf
;; den Prozess nicht reißen. Geladen von -t und TestSwankSurvivesEndlosrekursion.
;; ********************************************************************

(load "tests/test-helpers.lisp")

(defun endlos-tief (n) (+ 1 (endlos-tief n)))

(parfunc endlos-ergebnis :timeout 1 (endlos-tief 0))

;; Der Zweig scheitert am Tiefenlimit; parfunc meldet das an seiner Stelle
;; als (:error text) statt still ().
(assert= '(:error "eval: maximum recursion depth exceeded") (car endlos-ergebnis))

(println "endlosrekursion: alle PASS")
nil
