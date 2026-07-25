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
;; *loaded-systems*: Namen explizit geladener Systeme (für Shared-File-Entscheidungen).
;; Idempotenz auf Datei-Ebene: zwei Systeme dürfen dieselbe Datei listen.
(define *systems* '())
(define *loaded-files* '())
(define *loaded-systems* '())

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

;; === load-system =====================================================

;; %topo: DFS mit Zykluserkennung. visiting = aktueller DFS-Stack,
;; done = fertige Systeme. Rückgabe: done mit name oben (Deps darunter).
(defun %topo (name visiting done)
  (cond
    ((member name visiting)
     (error (format nil "defsystem: Abhängigkeitszyklus: ~a"
                    (reverse (cons name visiting)))))
    ((member name done) done)
    (t
     (let ((d done)
           (vis2 (cons name visiting)))
       (dolist (dep (%sys-get (%sys-entry name) :depends-on) ())
         (set! d (%topo dep vis2 d)))
       (cons name d)))))

;; (load-system 'name) → Topo-Liste der beteiligten Systeme (Deps zuerst).
;; Idempotent auf Datei-Ebene: bereits geladene Komponenten werden
;; übersprungen. Fehler in einer Komponente bricht ab; der Teilzustand
;; bleibt stehen und ein erneuter Aufruf setzt an der Fehlerstelle fort.
(defun load-system (name)
  (let ((order (reverse (%topo name '() '()))))
    (dolist (sys order ())
      (dolist (comp (%sys-get (%sys-entry sys) :components) ())
        (let ((norm (get-file-path comp)))
          (unless (member norm *loaded-files*)
            (eval (list 'load comp))
            (push norm *loaded-files*))))
      (set! *loaded-systems*
            (cons sys (filter (lambda (n) (not (equal? n sys))) *loaded-systems*))))
    order))

;; === Introspection ===================================================

;; %sys-loaded?: t, wenn das System explizit geladen wurde und alle
;; Komponenten in *loaded-files* stehen.
(defun %sys-loaded? (entry)
  (and (member (car entry) *loaded-systems*)
       (every (lambda (c) (member (get-file-path c) *loaded-files*))
              (%sys-get (cadr entry) :components))))

;; (loaded-systems) -> Namen aller vollständig geladenen Systeme.
;; Berechnet aus *loaded-systems* und *loaded-files*.
(defun loaded-systems ()
  (mapcar #'car (filter (lambda (e) (%sys-loaded? e)) *systems*)))

;; (system-symbols 'name) -> alle Symbole, die die Komponenten des
;; Systems definiert haben (via DefLoc-Registry, je Datei sortiert).
(defun system-symbols (name)
  (let ((acc '()))
    (dolist (c (%sys-get (%sys-entry name) :components) acc)
      (set! acc (append acc (defined-in c))))))

;; === unload-system ===================================================

;; %remove-first: erstes Vorkommen von x aus Liste entfernen
(defun %remove-first (x lst)
  (cond ((null lst)             ())
        ((equal? x (car lst))   (cdr lst))
        (t (cons (car lst) (%remove-first x (cdr lst))))))

;; %file-shared?: t, wenn ein ANDERES geladenes System die Datei mitlistet
(defun %file-shared? (norm self)
  (any (lambda (e)
         (and (not (equal? (car e) self))
              (%sys-loaded? e)
              (any (lambda (c) (equal? (get-file-path c) norm))
                   (%sys-get (cadr e) :components))))
       *systems*))

;; (unload-system 'name) → Liste der entfernten Symbole.
;; Shared Files (anderes geladenes System listet sie) bleiben unangetastet.
;; Deps werden nicht mit-entladen. Policy-Hülle: makunbound warnt sonst
;; pro Symbol — Events landen trotzdem im Redef-Log. trap stellt die
;; Policy auch im Fehlerfall wieder her (prozess-globaler Zustand!).
(defun unload-system (name)
  (let ((entry (%sys-entry name))
        (removed ())
        (old-policy (redefine-policy)))
    (set! *loaded-systems*
          (filter (lambda (n) (not (equal? n name))) *loaded-systems*))
    (redefine-policy 'allow)
    (trap
      (dolist (c (%sys-get entry :components) ())
        (let ((norm (get-file-path c)))
          (when (and (member norm *loaded-files*)
                     (not (%file-shared? norm name)))
            (dolist (s (defined-in c) ())
              (when (bound? s)
                (makunbound s)
                (push s removed)))
            (set! *loaded-files* (%remove-first norm *loaded-files*)))))
      (lambda (e)
        (begin (redefine-policy old-policy) (error e))))
    (redefine-policy old-policy)
    removed))
