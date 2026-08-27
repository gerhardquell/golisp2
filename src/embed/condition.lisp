;; ********************************************************************
;; condition.lisp – Condition-lite: Fehler mit strukturiertem Kontext
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k3
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260730
;; ********************************************************************
;; Orientierung am CL-Condition-System, stark reduziert ("lite"):
;;
;;   (define-condition file-error (io-error) (path))
;;   (signal 'file-error :path "x.lisp")
;;   (handler-case (load "x.lisp")
;;     (file-error (e) (format nil "fehlt: ~a" (file-error-path e)))
;;     (io-error  (e) "irgendein io-fehler"))
;;
;; - Condition = (%condition typ plist); Typ-Registry mit Elternkette.
;; - Reader automatisch: Slot path in file-error → file-error-path.
;; - handler-case dispatcht inkl. Eltern (file-error matcht io-error),
;;   erste passende Klausel gewinnt, kein Match → Re-Signal.
;; - Go-Fehler (file-read o. ä.) werden in handler-case zur generischen
;;   Condition lisp-error mit Slot msg.
;;
;; BEWUSSTE ABWEICHUNGEN von CL (lite!):
;; - signal unwindet immer (verhält sich wie CLs error, nicht wie CLs
;;   nicht-unwindendes signal).
;; - Keine Restarts, kein handler-bind, kein MOP.
;; - Slot-Namen über eine Vererbungshierarchie hinweg müssen eindeutig
;;   sein (flache plist, keine Verdeckung).
;; ********************************************************************

;; === Zustand =========================================================

;; *condition-types*: Alist (typ . elternliste) — defvar: Mehrfach-Load
;; (z. B. über LoadString in Tests) darf registrierte Typen nicht killen.
(defvar *condition-types* '())

;; === Interne Helfer ==================================================

;; %cond-get: Wert aus plist ((:k1 v1 :k2 v2) :k1) → v1, sonst ()
(defun %cond-get (plist key)
  (cond ((null plist)             '())
        ((equal? (car plist) key) (cadr plist))
        (t                        (%cond-get (cddr plist) key))))

;; %cond?: ist c eine Condition-Struktur?
(defun %cond? (c)
  (and (pair? c) (equal? (car c) '%condition)))

;; %cond-subtype?: ist sub gleich oder unterhalb von super?
(defun %cond-subtype? (sub super)
  (cond ((equal? sub super) t)
        ((null sub)         '())
        (t (let ((parents (alist-get sub *condition-types*)))
             (if (null parents)
                 '()
                 (any (lambda (p) (%cond-subtype? p super)) parents))))))

;; %cond-type?: matcht Condition c auf Typ type (inkl. Vererbung)?
(defun %cond-type? (c type)
  (and (%cond? c) (%cond-subtype? (cadr c) type)))

;; %cond-slot: Slot-Wert einer Condition
(defun %cond-slot (c key)
  (%cond-get (caddr c) key))

;; %cond-coerce: beliebigen trap-Fang in eine Condition wandeln.
;; Condition → unverändert; String (Go-Fehler) → lisp-error mit :msg.
(defun %cond-coerce (e)
  (if (%cond? e)
      e
      (list '%condition 'lisp-error (list :msg (format nil "~a" e)))))

;; === Vordefinierte Basis-Hierarchie ==================================
;; condition (Wurzel) → error → lisp-error (Go-/String-Fehler-Fallback)

(set! *condition-types*
      (alist-set 'condition '() *condition-types*))
(set! *condition-types*
      (alist-set 'error '(condition) *condition-types*))
(set! *condition-types*
      (alist-set 'lisp-error '(error) *condition-types*))

;; lisp-error-msg: Slot-Reader für die Fallback-Condition
(defun lisp-error-msg (c) (%cond-slot c :msg))

;; === API =============================================================

;; define-condition: (define-condition name (eltern...) (slots...))
;; Registriert Typ, erzeugt Reader name-slot je Slot.
;; Neudefinition ersetzt still (Reload-Semantik, wie defsystem).
(defmacro define-condition (name parents slots)
  (if (not (symbol? name))
      (error "define-condition: Name muss Symbol sein"))
  (if (not (every (lambda (p) (symbol? p)) parents))
      (error "define-condition: Eltern müssen Symbole sein"))
  (if (not (every (lambda (s) (symbol? s)) slots))
      (error "define-condition: Slots müssen Symbole sein"))
  `(progn
     (set! *condition-types*
           (alist-set ',name ',parents *condition-types*))
     ,@(mapcar (lambda (s)
                 (let ((reader (intern (string-append (symbol->string name)
                                                      "-"
                                                      (symbol->string s))))
                       ;; plist-Keys sind Keywords (:path), Slot-Name ist
                       ;; Symbol (path) → Reader sucht nach Keyword
                       (key (intern (string-append ":" (symbol->string s)))))
                   `(defun ,reader (c) (%cond-slot c ',key))))
               slots)
     ',name))

;; signal: (signal 'typ :slot1 v1 :slot2 v2 ...) → wirft Condition.
;; ABWEICHUNG CL: unwindet immer (wie CLs error).
(defun signal (type &rest kvs)
  (if (not (assoc type *condition-types*))
      (error (format nil "signal: unbekannter Condition-Typ '~a'" type))
      (error (list '%condition type kvs))))

;; handler-case: (handler-case body (typ (e) ...) ...)
;; Erste passende Klausel (inkl. Elternkette) gewinnt.
;; Klausel-Var kann () sein: (typ () ...) → ohne Bindung.
;; Kein Match → Re-Signal der Condition.
(defmacro handler-case (body &rest clauses)
  (let ((raw (gensym "RAW"))
        (cnd (gensym "CND")))
    `(let ((,raw (trap ,body (lambda (e) (list '%cond-caught e)))))
       (if (and (pair? ,raw) (equal? (car ,raw) '%cond-caught))
           (let ((,cnd (%cond-coerce (cadr ,raw))))
             (cond
               ,@(mapcar (lambda (cl)
                           (let ((v (if (null (cadr cl)) '() (car (cadr cl)))))
                             (if (null v)
                                 `((%cond-type? ,cnd ',(car cl)) ,@(cddr cl))
                                 `((%cond-type? ,cnd ',(car cl))
                                   (let ((,v ,cnd)) ,@(cddr cl))))))
                         clauses)
               (t (error (cadr ,raw)))))
           ,raw))))
