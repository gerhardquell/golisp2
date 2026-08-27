;;; kiesp-stack.lisp - Stack-basierte fraktale Verben
;;; Autor: Gerhard Quell – gquell@skequell.de
;;; CoAutor: Claude
;;; Experiment: KIESP Stack-Fraktal mit Postfix-Notation

;;; Basis-Stack-Operationen
(define kiesp-stack '())

(defun kiesp-push (x)
  "Legt ein Element auf den Stack"
  (set! kiesp-stack (cons x kiesp-stack))
  x)

(defun kiesp-pop ()
  "Nimmt oberstes Element vom Stack"
  (if (null kiesp-stack)
      (error "KIESP: Stack underflow")
      (begin
        (define top-element (car kiesp-stack))
        (set! kiesp-stack (cdr kiesp-stack))
        top-element)))

(defun kiesp-peek ()
  "Zeigt oberstes Element ohne Entfernen"
  (if (null kiesp-stack)
      (error "KIESP: Stack empty")
      (car kiesp-stack)))

(defun kiesp-clear ()
  "Leert den Stack"
  (set! kiesp-stack '())
  'ok)

;;; Fraktale Verb-Ebenen (Postfix-Operatoren)
;;; >   : Ebene 1 - einfache Aktion
;;; >>  : Ebene 2 - iterierte Aktion
;;; >>> : Ebene 3 - rekursive Transformation
;;; >>>>: Ebene 4 - Meta-Operation

(define kiesp-verbs
  '((G . "generate")      ; Erzeugen/Generieren
    (C . "create")        ; Schaffen/Erschaffen
    (FR . "from")         ; Aus/Von
    (T . "transform")     ; Transformieren
    (A . "analyze")       ; Analysieren
    (S . "synthesize")    ; Synthetisieren
    (V . "validate")      ; Validieren
    (E . "evaluate")      ; Evaluieren
    (I . "iterate")       ; Iterieren
    (R . "reduce")        ; Reduzieren
    (M . "map")           ; Mappen
    (F . "filter")))      ; Filtern

(defun kiesp-verb (code)
  "Gibt den vollen Verb-Namen fuer einen Code zurueck"
  (let ((pair (assoc code kiesp-verbs)))
    (if pair (cdr pair) "unknown")))

;;; Fraktale Verben (Postfix-Notation)

(defun kiesp-level-1 ()
  "Ebene 1: Einfache Aktion ausfuehren"
  (let ((verb (kiesp-pop))
        (arg (kiesp-pop)))
    (kiesp-push (list 'action (kiesp-verb verb) arg))))

(defun kiesp-level-2 ()
  "Ebene 2: Iterierte Aktion"
  (let ((verb (kiesp-pop))
        (count (kiesp-pop))
        (arg (kiesp-pop)))
    (kiesp-push (list 'iterate (kiesp-verb verb) count arg))))

(defun kiesp-level-3 ()
  "Ebene 3: Rekursive Transformation"
  (let ((verb (kiesp-pop))
        (depth (kiesp-pop))
        (arg (kiesp-pop)))
    (kiesp-push (list 'recurse (kiesp-verb verb) depth arg))))

(defun kiesp-level-4 ()
  "Ebene 4: Meta-Operation"
  (let ((verb (kiesp-pop))
        (meta (kiesp-pop))
        (arg (kiesp-pop)))
    (kiesp-push (list 'meta (kiesp-verb verb) (kiesp-verb meta) arg))))

;;; Komposition-Operator

(defun kiesp-pipe ()
  "Verknuepft zwei Stack-Eintraege (Pipe)"
  (let ((right (kiesp-pop))
        (left (kiesp-pop)))
    (kiesp-push (list 'pipe left right))))

;;; Encoder/Decoder

(defun kiesp-encode-stack (tokens)
  "Kodiert eine Token-Liste in Stack-Programm"
  (kiesp-clear)
  (kiesp-encode-list tokens)
  (kiesp-peek))

(defun kiesp-encode-list (tokens)
  "Hilfsfunktion fuer Encoding"
  (if (null tokens)
      'done
      (let ((tok (car tokens)))
        (cond
          ;; Fraktal-Operatoren
          ((equal? tok '>) (kiesp-level-1))
          ((equal? tok '>>) (kiesp-level-2))
          ((equal? tok '>>>) (kiesp-level-3))
          ((equal? tok '>>>>) (kiesp-level-4))
          ((equal? tok 'P) (kiesp-pipe))
          ;; Standard: Auf Stack legen
          (else (kiesp-push tok)))
        (kiesp-encode-list (cdr tokens)))))

(defun kiesp-decode (expr)
  "Dekodiert eine KIESP-Expression zurueck in Tokens"
  (if (atom expr)
      (list expr)
      (let ((tag (car expr)))
        (cond
          ((equal? tag 'action) (list (caddr expr) (cadr expr) '>))
          ((equal? tag 'iterate) (list (cadddr expr) (caddr expr) (cadr expr) '>>))
          ((equal? tag 'recurse) (list (cadddr expr) (caddr expr) (cadr expr) '>>>))
          ((equal? tag 'meta) (list (cadddr expr) (caddr expr) (cadr expr) '>>>>))
          ((equal? tag 'pipe) (append (kiesp-decode (cadr expr)) (kiesp-decode (caddr expr)) '(P)))
          (else expr)))))

;;; Statistik

(defun kiesp-stats-stack (original kiesp-expr)
  "Berechnet Token-Effizienz"
  (let ((orig-len (length original))
        (kiesp-len (length (flatten kiesp-expr))))
    (list
      (cons 'original-tokens orig-len)
      (cons 'kiesp-tokens kiesp-len)
      (cons 'ratio (* 100.0 (/ kiesp-len orig-len)))
      (cons 'compression (* 100.0 (- 1.0 (/ kiesp-len orig-len)))))))

;;; Hilfsfunktion: Flatten

(defun flatten (x)
  "Flacht eine verschachtelte Liste"
  (if (atom x)
      (if (null x) '() (list x))
      (append (flatten (car x)) (flatten (cdr x)))))

;;; Beispiele

(define kiesp-example-1
  '(data G >))  ; (action "generate" data)

(define kiesp-example-2
  '(text 3 C >>))  ; (iterate "create" 3 text)

(define kiesp-example-3
  '(input G > output T > P))  ; Verkettung

(println "=== KIESP Stack-Fraktal geladen ===")
(println "Verfuegbare Verben: G C FR T A S V E I R M F")
(println "Fraktal-Ebenen: > >> >>> >>>>")
(println "Komposition: P")
