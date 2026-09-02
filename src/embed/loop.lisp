;; ********************************************************************
;; embed/loop.lisp — LOOP-Makro (CL-Praxis-Kern) für GoLisp2
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : kimi-k3
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260902
;; ********************************************************************
;; Implementiert eine bewusste Untermenge des CL-LOOP als reines
;; Lisp-Makro: die Klauseln werden zur Expansionszeit geparst und in
;; let + while + set! übersetzt. Kein Go-Code nötig.
;;
;; Unterstützte Klauseln:
;;   (repeat n)
;;   (for v from a [to|below|downto b] [by s])
;;   (for v in liste)
;;   (do expr ...)  (finally expr ...)
;;   (while test)   (until test)
;;   (when test klausel)  (unless test klausel)
;;   (collect e) (append e) (sum e) (count e) (maximize e) (minimize e)
;;
;; Einschränkungen gegenüber CL (bewusst, dokumentiert):
;;   - Eine Akkumulator-Familie pro loop:
;;     collect/append  ODER  sum/count  ODER  maximize/minimize.
;;     Familienwechsel ist ein Fehler zur Expansionszeit.
;;   - when/unless gilt für genau eine Folgeklausel
;;     (do mit mehreren Formen oder eine Akkumulation), kein and/else.
;;   - Kein Destructuring in for, kein across (kein Vektor-Typ),
;;   - kein initially, kein named, kein =/then-Schritt.
;;   - Syntaxfehler werden zur Expansionszeit per error gemeldet.
;; ********************************************************************

;; --- Klausel-Erkennung ------------------------------------------------

(defun %loop-kw? (x)
  (and (symbol? x)
       (member x '(for repeat do when unless while until finally
                   collect append sum count maximize minimize))))

(defun %loop-acc-kw? (x)
  (member x '(collect append sum count maximize minimize)))

;; Formen bis zum nächsten Klausel-Schlüsselwort aufsammeln/überspringen
;; (für do und finally, die mehrere Formen umfassen).
(defun %loop-take-nonkw (cs)
  (if (or (null cs) (%loop-kw? (car cs)))
      '()
      (cons (car cs) (%loop-take-nonkw (cdr cs)))))

(defun %loop-drop-nonkw (cs)
  (if (or (null cs) (%loop-kw? (car cs)))
      cs
      (%loop-drop-nonkw (cdr cs))))

;; --- Akkumulator ------------------------------------------------------
;; acc: () | (familie var)
;; familie: 'list (collect/append) | 'num0 (sum/count, Init 0)
;;          | 'numn (maximize/minimize, Init ())
;; Liefert (aktion neuer-acc); Fehler bei Familienwechsel.

(defun %loop-acc (kw expr acc)
  (let ((familie (cond ((member kw '(collect append))     'list)
                       ((member kw '(sum count))          'num0)
                       (t                                 'numn))))
    (if (and (not (null acc)) (not (eq (car acc) familie)))
        (error (format nil "loop: ~a passt nicht zur Akkumulator-Familie ~a — eine Familie pro loop"
                       kw (car acc))))
    (let ((var (if (null acc) (gensym) (cadr acc))))
      (list
        (cond
          ((eq kw 'collect)  `(set! ,var (cons ,expr ,var)))
          ((eq kw 'append)   `(set! ,var (append (reverse ,expr) ,var)))
          ((eq kw 'sum)      `(set! ,var (+ ,var ,expr)))
          ((eq kw 'count)    `(if ,expr (set! ,var (+ ,var 1))))
          ((eq kw 'maximize) `(set! ,var (if (null ,var) ,expr (max ,var ,expr))))
          ((eq kw 'minimize) `(set! ,var (if (null ,var) ,expr (min ,var ,expr))))
          (t (error (format nil "loop: unbekannte Akkumulation ~a" kw))))
        (list familie var)))))

;; --- for-Varianten ----------------------------------------------------

;; (for v from a [to|below|downto b] [by s])
;; to = inklusiv, below = exklusiv, downto = abwärts inklusiv.
(defun %loop-from (v cs inits guards pres acts posts fins acc)
  (let ((start (car cs))
        (rest  (cdr cs))
        (end-kw '())
        (end-expr '())
        (step '()))
    (if (and (not (null rest)) (member (car rest) '(to below downto)))
        (begin (set! end-kw   (car rest))
               (set! end-expr (cadr rest))
               (set! rest     (cddr rest))))
    (if (and (not (null rest)) (eq (car rest) 'by))
        (begin (set! step (cadr rest))
               (set! rest (cddr rest))))
    (let ((down (eq end-kw 'downto))
          (step-expr (if (null step) 1 step))
          (lim (gensym)))
      (%loop-parse
        rest
        (if (null end-kw)
            (cons (list v start) inits)
            (cons (list lim end-expr) (cons (list v start) inits)))
        (if (null end-kw)
            guards
            (cons (cond ((eq end-kw 'to)    `(<= ,v ,lim))
                        ((eq end-kw 'below) `(< ,v ,lim))
                        (t                  `(>= ,v ,lim)))
                  guards))
        pres acts
        (cons (if down
                  `(set! ,v (- ,v ,step-expr))
                  `(set! ,v (+ ,v ,step-expr)))
              posts)
        fins acc))))

(defun %loop-for (cs inits guards pres acts posts fins acc)
  (let ((v (car cs)) (dir (cadr cs)))
    (cond
      ((eq dir 'in)
       (let ((xs (gensym)))
         (%loop-parse (cddr (cdr cs))
                      (cons (list xs (caddr cs)) (cons (list v '()) inits))
                      (cons `(not (null ,xs)) guards)
                      (cons `(set! ,v (car ,xs)) pres)
                      acts
                      (cons `(set! ,xs (cdr ,xs)) posts)
                      fins acc)))
      ((eq dir 'from)
       (%loop-from v (cddr cs) inits guards pres acts posts fins acc))
      (t (error (format nil "loop: for: erwartet in|from, got ~a" dir))))))

;; --- when / unless ----------------------------------------------------
;; Genau eine Folgeklausel: do (mehrere Formen) oder eine Akkumulation.

(defun %loop-when (test neg cs inits guards pres acts posts fins acc)
  (if (null cs)
      (error "loop: when/unless ohne Folgeklausel"))
  (let ((kw (car cs))
        (wrap (lambda (f) (if neg `(if (not ,test) ,f) `(if ,test ,f)))))
    (cond
      ((eq kw 'do)
       (let ((forms (%loop-take-nonkw (cdr cs))))
         (%loop-parse (%loop-drop-nonkw (cdr cs))
                      inits guards pres
                      (cons (wrap (cons 'begin forms)) acts)
                      posts fins acc)))
      ((%loop-acc-kw? kw)
       (let ((r (%loop-acc kw (cadr cs) acc)))
         (%loop-parse (cddr cs)
                      inits guards pres
                      (cons (wrap (car r)) acts)
                      posts fins (cadr r))))
      (t (error (format nil "loop: when/unless: erwartet do|collect|append|sum|count|maximize|minimize, got ~a"
                        kw))))))

;; --- Haupt-Parser -----------------------------------------------------
;; Zustand: (inits guards pres acts posts fins acc)
;; inits: let-Bindungen, guards: Abbruch-/Laufbedingungen (and),
;; pres: for-in Zuweisungen, acts: Rumpf-Aktionen in Quellreihenfolge,
;; posts: Schritte, fins: finally-Formen, acc: Akkumulator.

(defun %loop-parse (cs inits guards pres acts posts fins acc)
  (if (null cs)
      (list inits guards pres acts posts fins acc)
      (let ((kw (car cs)))
        (cond
          ((eq kw 'repeat)
           (let ((n (gensym)))
             (%loop-parse (cddr cs)
                          (cons (list n (cadr cs)) inits)
                          (cons `(> ,n 0) guards)
                          pres acts
                          (cons `(set! ,n (- ,n 1)) posts)
                          fins acc)))
          ((eq kw 'for)
           (%loop-for (cdr cs) inits guards pres acts posts fins acc))
          ((eq kw 'do)
           (%loop-parse (%loop-drop-nonkw (cdr cs))
                        inits guards pres
                        (append acts (%loop-take-nonkw (cdr cs)))
                        posts fins acc))
          ((or (eq kw 'when) (eq kw 'unless))
           (%loop-when (cadr cs) (eq kw 'unless) (cddr cs)
                       inits guards pres acts posts fins acc))
          ((eq kw 'while)
           (%loop-parse (cddr cs) inits
                        (cons (cadr cs) guards)
                        pres acts posts fins acc))
          ((eq kw 'until)
           (%loop-parse (cddr cs) inits
                        (cons `(not ,(cadr cs)) guards)
                        pres acts posts fins acc))
          ((eq kw 'finally)
           (%loop-parse (%loop-drop-nonkw (cdr cs))
                        inits guards pres acts posts
                        (append fins (%loop-take-nonkw (cdr cs)))
                        acc))
          ((%loop-acc-kw? kw)
           (let ((r (%loop-acc kw (cadr cs) acc)))
             (%loop-parse (cddr cs) inits guards pres
                          (append acts (list (car r)))
                          posts fins (cadr r))))
          (t (error (format nil "loop: unbekannte Klausel ~a" kw)))))))

;; --- Das Makro --------------------------------------------------------

(defmacro loop (&rest clauses)
  (let ((st (%loop-parse clauses '() '() '() '() '() '() '())))
    (let ((inits  (nth 0 st)) (guards (nth 1 st))
          (pres   (nth 2 st)) (acts   (nth 3 st))
          (posts  (nth 4 st)) (fins   (nth 5 st))
          (acc    (nth 6 st)))
      (let ((familie (if (null acc) '() (car acc)))
            (acc-var (if (null acc) '() (cadr acc))))
        `(let (,@(cond ((null acc) '())
                       ((eq familie 'num0) `((,acc-var 0)))
                       (t `((,acc-var '()))))
               ,@inits)
           (while ,(if (null guards) t (cons 'and (reverse guards)))
             ,@(reverse pres)
             ,@acts
             ,@(reverse posts))
           ,@fins
           ,(cond ((null acc) '())
                  ((eq familie 'list) `(reverse ,acc-var))
                  (t acc-var)))))))
