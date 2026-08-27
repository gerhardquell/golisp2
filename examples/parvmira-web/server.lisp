#!/bin/env golisp2
;; server.lisp
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : Claude Sonnet 5
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260826
;;;; PARVMIRA-Entscheidungsmodell (Projekt NEXORA) - Web-Bridge-Server.
;;;; V1: nur Schritte 1-3 (Vorgabe, Plus/Minus/Interessant). Anders als
;;;; beim ABC-Generator lebt der State HIER im Server (aktuelle-liste),
;;;; nicht im Browser - der Browser ist nur Anzeige/Eingabe-Oberflaeche.

(load "/u/golisp2-projekte/nexora/src/parvmira/pmi.lisp")
(load "/u/golisp2-projekte/nexora/src/parvmira/ki-ensemble.lisp")

(system "mkdir -p /u/golisp2-projekte/nexora/tmp")

;; Pfad-Selbstlokalisierung ueber (argv): http-static loest den public-Pfad
;; via Gos rohem os.Stat auf (nicht ueber golisp2s set-working-directory),
;; darum braucht der Server den absoluten Pfad zu seinem eigenen Skript.
;; Funktioniert nur bei direktem Shebang-Start mit "/" im Aufrufpfad
;; (./server.lisp, src/parvmira-web/server.lisp, absoluter Pfad) - NICHT bei
;; interaktivem (load "server.lisp") in SWANK/REPL und NICHT bei
;; "golisp2 server.lisp" ohne "/" im Pfad (dort liefert (cadr (argv))
;; keinen Slash, Ergebnis "." - nur korrekt, wenn cwd zufaellig passt).
(defun letzter-index (s zeichen)
  (let ((n (string-length s)) (i -1) (found -1))
    (while (< (+ i 1) n)
      (set! i (+ i 1))
      (if (equal? (substring s i (+ i 1)) zeichen)
          (set! found i)))
    found))

(defun skript-verzeichnis ()
  (let* ((pfad (cadr (argv)))
         (idx (letzter-index pfad "/")))
    (if (< idx 0) "." (substring pfad 0 idx))))

(define aktuelle-liste (make-pmi-liste :vorgabe ""))

(defun kategorie-symbol (s)
  (cond ((equal? s "plus") 'plus)
        ((equal? s "minus") 'minus)
        ((equal? s "interessant") 'interessant)
        (t (error "kategorie-symbol: unbekannt"))))

(defun kategorie-liste (l kat)
  (cond ((eq kat 'plus) (pmi-liste-plus l))
        ((eq kat 'minus) (pmi-liste-minus l))
        ((eq kat 'interessant) (pmi-liste-interessant l))
        (t (error "kategorie-liste: unbekannt"))))

(defun kategorie-liste-setzen! (kat neue)
  (cond ((eq kat 'plus) (set! aktuelle-liste (set-pmi-liste-plus aktuelle-liste neue)))
        ((eq kat 'minus) (set! aktuelle-liste (set-pmi-liste-minus aktuelle-liste neue)))
        ((eq kat 'interessant) (set! aktuelle-liste (set-pmi-liste-interessant aktuelle-liste neue)))
        (t (error "kategorie-liste-setzen!: unbekannt"))))

(defun punkt->alist (p)
  (list (cons "kurzbezeichnung" (punkt-kurzbezeichnung p))
        (cons "beschreibung" (punkt-beschreibung p))
        (cons "kategorie" (symbol->string (punkt-kategorie p)))
        (cons "erstellerTyp" (symbol->string (punkt-ersteller-typ p)))
        (cons "erstellerName" (punkt-ersteller-name p))
        (cons "persoenlichkeit" (punkt-persoenlichkeit p))))

(defun vorgabe-setzen (client-id text)
  (set! aktuelle-liste (make-pmi-liste :vorgabe text))
  text)

(defun vorgabe-abrufen (client-id) (pmi-liste-vorgabe aktuelle-liste))

(defun punkte-abrufen (client-id kategorie)
  ;; golisp2 ist ein Lisp-1: Funktionsreferenzen als Wert MUESSEN unquotiert
  ;; sein (der Symbolwert IST die Funktion) - 'punkt->alist waere nur das
  ;; Symbol-Atom, kein Funktions-Cell, und apply schlaegt fehl. Empirisch
  ;; verifiziert 20260826 (Testlauf brach mit "apply: 'X' ist keine
  ;; Funktion" ab, obwohl (bound? 'X) t liefert) - neuer Fallstrick,
  ;; noch nicht in CLAUDE.md dokumentiert.
  (mapcar punkt->alist (kategorie-liste aktuelle-liste (kategorie-symbol kategorie))))

(defun punkt-hinzufuegen (client-id kategorie kurzbezeichnung beschreibung name)
  (let* ((kat (kategorie-symbol kategorie))
         (p (make-punkt :kurzbezeichnung kurzbezeichnung :beschreibung beschreibung
                         :kategorie kat :ersteller-typ 'mensch :ersteller-name name)))
    (kategorie-liste-setzen! kat (append (kategorie-liste aktuelle-liste kat) (list p)))
    (punkt->alist p)))

;; sigo-models kann Alias+Kanonischen-Namen doppelt liefern (Fallstrick 8,
;; CLAUDE.md) - remove-duplicates (stdlib, equal?-basiert) genuegt hier.
(defun modelle-abrufen (client-id)
  (remove-duplicates (sigo-models)))

;; ensemble-roh kommt vom Client als JSON-Array von 2er-Arrays [modell
;; persoenlichkeit] und laeuft deshalb als Liste 2-elementiger Listen ein,
;; nicht als dotted pair (JSON kennt keine dotted pairs) - erst hier zu
;; den von ki-punkte erwarteten (modell . persoenlichkeit)-Paaren gebaut.
(defun ki-abrufen (client-id kategorie ensemble-roh)
  (let* ((kat (kategorie-symbol kategorie))
         (ensemble (mapcar (lambda (paar) (cons (car paar) (cadr paar))) ensemble-roh))
         (neue (ki-punkte (pmi-liste-vorgabe aktuelle-liste) kat ensemble)))
    (kategorie-liste-setzen! kat (append (kategorie-liste aktuelle-liste kat) neue))
    (mapcar punkt->alist neue)))

(defun vorgabeSlug (vorgabe)
  (let ((kurz (if (> (string-length vorgabe) 40) (substring vorgabe 0 40) vorgabe)))
    (string-replace (string-replace (string-trim kurz) " " "_") "/" "_")))

(defun punktZeile (p)
  (string-append
    (punkt-kurzbezeichnung p) ": " (punkt-beschreibung p)
    " [Ersteller: " (symbol->string (punkt-ersteller-typ p)) "/"
    (punkt-ersteller-name p) "/" (punkt-persoenlichkeit p) "]\n"))

(defun kategorieText (ueberschrift punkte)
  (let ((out (string-append ueberschrift ":\n")))
    (dolist (p punkte)
      (set! out (string-append out (punktZeile p))))
    (string-append out "\n")))

;; zeitstempel/datum-text kommen vom Client (analog abcliste/server.lisp) -
;; golisp2 hat kein Primitiv, das Wanduhrzeit als lesbaren String liefert
;; (nur get-universal-time als CL-Epochensekunden), ein Datumsformatierer
;; in reinem Lisp waere Spekulation ueber zukuenftigen Bedarf.
(defun speichern (client-id zeitstempel datum-text)
  (when (< (string-length (string-trim (pmi-liste-vorgabe aktuelle-liste))) 1)
    (error "speichern: Vorgabe fehlt"))
  (let* ((slug (vorgabeSlug (pmi-liste-vorgabe aktuelle-liste)))
         (dateiname (string-append "/u/golisp2-projekte/nexora/tmp/parvmira_" slug "_" zeitstempel ".txt"))
         (inhalt (string-append
                   " PARVMIRA-Entscheidungsmodell - PMI-Analyse\n"
                   "====================================\n\n"
                   "Vorgabe: " (pmi-liste-vorgabe aktuelle-liste) "\n"
                   "Datum: " datum-text "\n"
                   "====================================\n\n"
                   (kategorieText "PLUS" (pmi-liste-plus aktuelle-liste))
                   (kategorieText "MINUS" (pmi-liste-minus aktuelle-liste))
                   (kategorieText "INTERESSANT" (pmi-liste-interessant aktuelle-liste)))))
    (file-write dateiname inhalt)
    dateiname))

(define srv (http-serve 8091 :host "127.0.0.1"))

(http-static srv "/" (string-append (skript-verzeichnis) "/public"))

;; exec braucht timeout: -1 -- ohne das killt der Default (60s,
;; lib/eval_exec.go) golisp2web mitten in der Sitzung per SIGKILL, egal
;; ob User aktiv ist (siehe Commit c18e600, "feat(exec): timeout:-Keyword").
;; Branch 2 (ws-export-Registrierung + http-wait) braucht begin -- sonst
;; ist es syntaktisch EINE Anwendungsform statt einer Sequenz: Operator
;; waere die erste ws-export-Form, der Rest ihre "Argumente". Faellt heute
;; nicht auf, weil evalParfunc Branch-Fehler lautlos zu nil schluckt und
;; niemand ret auswertet -- Landmine fuer spaeter.
(parfunc ret (exec "golisp2web" param: "-t1" param: "localhost:8091"
                    param: "-t2" param: "localhost:8090"
                    timeout: -1)
 (begin
  (ws-export srv "vorgabe-setzen" vorgabe-setzen)
  (ws-export srv "vorgabe-abrufen" vorgabe-abrufen)
  (ws-export srv "punkte-abrufen" punkte-abrufen)
  (ws-export srv "punkt-hinzufuegen" punkt-hinzufuegen)
  (ws-export srv "modelle-abrufen" modelle-abrufen)
  (ws-export srv "ki-abrufen" ki-abrufen)
  (ws-export srv "speichern" speichern)
  (http-wait srv)))


