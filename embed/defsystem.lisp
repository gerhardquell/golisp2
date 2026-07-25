;; ********************************************************************
;; defsystem.lisp – deklarative Systemdefinition + idempotentes Laden
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k3
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260725
;; ********************************************************************
;; defsystem-lite: ASDF-orientiert, keine ASDF-Kompatibilität.
;; Design: docs/superpowers/specs/2026-07-25-defsystem-lite-design.md
;; Fundament: DefLoc-Registry (defined-in), makunbound, Redef-Log.
;; ********************************************************************

;; *systems*: Alist (name . plist) mit plist = (:depends-on (sym…) :components ("pfad"…))
;; *loaded-files*: normalisierte Pfade (via get-file-path) bereits geladener Dateien.
;; Idempotenz auf Datei-Ebene: zwei Systeme dürfen dieselbe Datei listen.
(define *systems* '())
(define *loaded-files* '())
