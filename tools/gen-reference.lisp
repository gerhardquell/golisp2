;;**********************************************************************
;;  tools/gen-reference.lisp
;;  Autor    : Gerhard Quell - gquell@skequell.de
;;  CoAutor  : claude-sonnet-5
;;  Copyright: 2026 Gerhard Quell - SKEQuell
;;  Erstellt : 20260827
;;**********************************************************************
;; Generiert docs/referenz-generiert.md aus (env-symbols) -- Vollstaendigkeit
;; ist strukturell garantiert, da direkt aus dem Root-Env gelesen wird.
;; Aufruf: ./build/golisp2 tools/gen-reference.lisp
;;**********************************************************************

(define *ref-out* "docs/referenz-generiert.md")

(defun write-reference ()
  (let* ((names (env-symbols))
         (rows (mapcar (lambda (n) (format nil "| `~a` | |~%" n)) names))
         (header (format nil "# GoLisp2 — Generierte Funktionsreferenz~%~%> Automatisch erzeugt aus `(env-symbols)`. Die Beschreibung-Spalte ist bewusst leer (Grundgerüst) und kann von Hand ergänzt werden — ACHTUNG: dieses Skript überschreibt die Datei komplett bei jedem Lauf, von Hand ergänzte Beschreibungen gehen dabei verloren.~%> Neu generieren: `./build/golisp2 tools/gen-reference.lisp`~%~%| Symbol | Beschreibung |~%|---|---|~%"))
         (body (apply string-append rows)))
    (file-write *ref-out* (string-append header body))
    (format t "~a Symbole nach ~a geschrieben~%" (length names) *ref-out*)))

(write-reference)
