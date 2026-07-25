;; Fixture für defsystem-Tests — Teilzustand bei Fehler
(defun fx-kaputt-ok () 'ok)
(error "fx-kaputt: absichtlicher Fehler")
