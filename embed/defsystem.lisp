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

;; === Interne Zugriffshelfer =========================================

;; %sys-get: Wert aus Property-Liste ((:k1 v1 :k2 v2) :k1) → v1, sonst ()
(defun %sys-get (plist key)
  (cond ((null plist)                '())
        ((equal? (car plist) key)    (cadr plist))
        (t                           (%sys-get (cddr plist) key))))

;; %sys-entry: System-Plist aus *systems*, Fehler wenn unbekannt
(defun %sys-entry (name)
  (let ((e (assoc name *systems*)))
    (if (null e)
        (error (format nil "defsystem: unbekanntes System '~a'" name))
        (cadr e))))

;; === defsystem =======================================================

;; (defsystem name :depends-on (sym…) :components ("pfad"…))
;; Validiert zur Expansionszeit; Neudefinition ersetzt still (Reload-Semantik).
(defmacro defsystem (name &rest kvs)
  (let ((deps '())
        (comps '())
        (rest-kv kvs))
    (while (not (null rest-kv))
      (let ((k (car rest-kv))
            (v (cadr rest-kv)))
        (cond ((equal? k :depends-on) (set! deps v))
              ((equal? k :components) (set! comps v))
              (t (error (format nil "defsystem: unbekanntes Keyword ~a" k))))
        (set! rest-kv (cddr rest-kv))))
    (if (not (every (lambda (s) (symbol? s)) deps))
        (error "defsystem: :depends-on muss Symbolliste sein"))
    (if (not (every (lambda (s) (string? s)) comps))
        (error "defsystem: :components muss Stringliste sein"))
    `(set! *systems*
           (alist-set ',name (list :depends-on ',deps :components ',comps) *systems*))))
