;; ********************************************************************
;; lib/swank/swank.lisp – SWANK protocol handlers for GoLisp.
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : claude sonnet 4.6
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260618
;; ********************************************************************

;; Redirect REPL output to Emacs.
(set! print swank-print)
(set! println swank-println)

;; === Built-in Arglist Registry =====================================
;; Autodoc / operator-arglist zeigen diese Arglisten fuer eingebaute
;; FUNC-Primitiven, da diese keine Lambda-Parameterliste besitzen.
;; Schluessel sind Strings, damit (assoc name ...) mit dem vom Protokoll
;; gelieferten String-Namen funktioniert.

(setq *swank-built-in-arglists*
  '(("+" . "(+ &rest numbers)")
    ("-" . "(- x &rest numbers)")
    ("*" . "(* &rest numbers)")
    ("/" . "(/ x y)")
    ("mod" . "(mod x y)")
    ("remainder" . "(remainder x y)")
    ("abs" . "(abs x)")
    ("random" . "(random &optional n)")
    ("=" . "(= x y)")
    ("<" . "(< x y)")
    (">" . "(> x y)")
    (">=" . "(>= x y)")
    ("<=" . "(<= x y)")
    ("equal?" . "(equal? x y)")
    ("eq" . "(eq x y)")
    ("eq?" . "(eq? x y)")
    ("string?" . "(string? x)")
    ("number?" . "(number? x)")
    ("list?" . "(list? x)")
    ("symbol?" . "(symbol? x)")
    ("atom?" . "(atom? x)")
    ("null?" . "(null? x)")
    ("car" . "(car list)")
    ("cdr" . "(cdr list)")
    ("cons" . "(cons x y)")
    ("atom" . "(atom x)")
    ("null" . "(null x)")
    ("list" . "(list &rest args)")
    ("append" . "(append list item)")
    ("apply" . "(apply fn arg1 ... list)")
    ("funcall" . "(funcall fn arg1 ...)")
    ("print" . "(print &rest args)")
    ("println" . "(println &rest args)")
    ("read" . "(read &optional string)")
    ("string-length" . "(string-length str)")
    ("string-append" . "(string-append &rest strs)")
    ("substring" . "(substring str start end)")
    ("string-upcase" . "(string-upcase str)")
    ("string-downcase" . "(string-downcase str)")
    ("string->number" . "(string->number str)")
    ("number->string" . "(number->string n)")
    ("string->list" . "(string->list str)")
    ("list->string" . "(list->string lst)")
    ("string-replace" . "(string-replace str old new)")
    ("string-trim" . "(string-trim str)")
    ("string-contains" . "(string-contains str sub)")
    ("error" . "(error msg)")
    ("catch" . "(catch body handler)")
    ("gensym" . "(gensym)")
    ("file-write" . "(file-write filename &rest contents)")
    ("file-append" . "(file-append filename &rest contents)")
    ("file-read" . "(file-read filename)")
    ("file-exists?" . "(file-exists? filename)")
    ("file-delete" . "(file-delete filename)")
    ("chan-make" . "(chan-make &optional size)")
    ("chan-send" . "(chan-send ch value)")
    ("chan-recv" . "(chan-recv ch)")
    ("lock-make" . "(lock-make)")
    ("sigo" . "(sigo prompt &optional model session-id host)")
    ("sigo-models" . "(sigo-models)")
    ("sigo-host" . "(sigo-host &optional host)")
    ("sleep" . "(sleep ms)")
    ("memstats" . "(memstats)")
    ("system" . "(system command)")
    ("file-stat" . "(file-stat path)")
    ("shell-assoc" . "(shell-assoc key alist)")
    ("symbol->string" . "(symbol->string sym)")
    ("pg-connect" . "(pg-connect conn-str)")
    ("pg-query" . "(pg-query conn query &rest params)")
    ("pg-exec" . "(pg-exec conn query &rest params)")
    ("pg-close" . "(pg-close conn)")))

(defun swank--built-in-arglist (name)
  (let ((key (if (symbol? name) (symbol->string name) name)))
    (let ((entry (assoc key *swank-built-in-arglists*)))
      (if (null entry) () (cdr entry)))))

;; === Symbol Description Registry ===================================
;; Statische Kurzbeschreibungen fuer describe-symbol. Erweiterbar zur
;; Laufzeit via (setq *swank-symbol-descriptions* ...).

(setq *swank-symbol-descriptions*
  '(("car" . "Gibt das erste Element einer Liste zurück.")
    ("cdr" . "Gibt den Rest einer Liste zurück.")
    ("cons" . "Konstruiert ein neues Paar (car . cdr).")
    ("list" . "Erzeugt eine Liste aus den Argumenten.")
    ("append" . "Fügt ein Element ans Ende einer Liste an.")
    ("apply" . "Wendet eine Funktion auf Argumente an.")
    ("eval" . "Wertet einen Ausdruck im globalen Environment aus.")
    ("print" . "Gibt Werte auf die Standardausgabe aus.")
    ("println" . "Gibt Werte mit Zeilenumbruch aus.")
    ("read" . "Liest einen Lisp-Ausdruck aus einem String.")
    ("sigo" . "Sendet einen Prompt an den sigoREST-Server.")
    ("parfunc" . "Wertet Ausdrücke parallel aus und speichert die Ergebnisse.")
    ("catch" . "Fängt Lisp-Laufzeitfehler ab.")
    ("error" . "Löst einen Lisp-Laufzeitfehler aus.")
    ("gensym" . "Erzeugt ein eindeutiges Symbol.")))

(defun swank--static-description (name)
  (let ((entry (assoc name *swank-symbol-descriptions*)))
    (if (null entry) () (cdr entry))))

(defun swank--describe-symbol (name)
  (let ((cell (catch (eval (read name)) (lambda (err) ()))))
    (let ((type (if (null? cell) "unbound" (swank--cell-type cell)))
          (arglist (catch (swank--arglist name) (lambda (err) ())))
          (static (swank--static-description name)))
      (string-append
        "Symbol: " name "\n"
        "Typ: " type "\n"
        (if (null? arglist) "" (string-append "Arglist: " arglist "\n"))
        (if (null? static) "" (string-append "\n" static))))))

(defun swank:describe-symbol (name id)
  (catch
    (let ((content (swank--describe-symbol name)))
      (list (list :return (list :ok (list :title name :content content)) id)))
    (lambda (err)
      (list (list :return (list :abort (swank--value-string err)) id)))))

(defun swank-dispatch (msg)
  (case (car msg)
    ((:emacs-rex)
     (let ((form (cadr msg))
           (pkg (caddr msg))
           (thread (cadddr msg))
           (id (car (cdr (cdr (cdr (cdr msg)))))))
       (handle-emacs-rex form pkg thread id)))
    (else (list (list :return (list :abort "unhandled message") 0)))))

(defun handle-emacs-rex (form pkg thread id)
  (let ((op (car form)))
    (cond
      ((equal? op 'swank:connection-info)
       (swank:connection-info id))
      ((equal? op 'swank:swank-require)
       (swank:swank-require id))
      ((equal? op 'swank:init-presentations)
       (swank:ok-nil id))
      ((equal? op 'swank:autodoc)
       (swank:autodoc form id))
      ((equal? op 'swank:operator-arglist)
       (swank:operator-arglist (cadr form) id))
      ((equal? op 'swank:describe-symbol)
       (swank:describe-symbol (cadr form) id))
      ((equal? op 'swank:swank-macroexpand-1)
       (swank:macroexpand-1 (cadr form) id))
      ((equal? op 'swank:swank-macroexpand)
       (swank:macroexpand-full (cadr form) id))
      ((equal? op 'swank:swank-macroexpand-all)
       (swank:macroexpand-all-handler (cadr form) id))
      ;; SLIMEs eigene expand-Familie (C-c C-m default). Wie macroexpand,
      ;; aber immer String-Return (sonst char-or-string-p nil in Emacs).
      ((equal? op 'swank:swank-expand-1)
       (swank:macroexpand-1 (cadr form) id))
      ((equal? op 'swank:swank-expand)
       (swank:macroexpand-full (cadr form) id))
      ;; swank-repl contrib nutzt eigenes Package-Prefix
      ((equal? op 'swank-repl:create-repl)
       (swank:create-repl id))
      ((equal? op 'swank-repl:listener-eval)
       (swank:listener-eval (cadr form) id))
      ((equal? op 'swank:simple-completions)
       (swank:simple-completions (cadr form) id))
      ((equal? op 'swank:completions)
       (swank:completions (cadr form) id))
      ((equal? op 'swank:load-file)
       (swank:load-file (cadr form) id))
      ((equal? op 'swank:compile-file-for-emacs)
       (swank:compile-file-for-emacs (cadr form) id))
      ((equal? op 'swank:compile-string-for-emacs)
       (swank:compile-string-for-emacs (cadr form) id))
      ((equal? op 'swank:find-definitions-for-emacs)
       (swank:find-definitions-for-emacs (cadr form) id))
      ;; Legacy-Prefix (Manuelle Tests)
      ((equal? op 'swank:create-repl)
       (swank:create-repl id))
      ((equal? op 'swank:listener-eval)
       (swank:listener-eval (cadr form) id))
      ;; Unbekannte Ops: graceful leere Liste statt :abort. SLIME-Contribs
      ;; degradieren sauber; :abort wuerfe Sync-Eval-Fehler in Emacs.
      (else
       (swank:ok-nil id)))))

;; Generic OK-Stub: liefert echte leere Liste () als :ok-Wert.
;; Viele SLIME-Ops (autodoc, init-*) erwarten Liste, kein String.
(defun swank:ok-nil (id)
  (list (list :return (list :ok (list)) id)))

(defun swank:connection-info (id)
  (list (list :return
              (list :ok
                    (list :pid 0
                          :style :spawn
                          :encoding (list :coding-systems (list "utf-8-unix"))
                          :implementation (list :type "GoLisp"
                                                :version "0.2"
                                                :program "golisp")
                          :machine (list :instance "unknown")
                          :package (list :name "USER" :prompt "USER")
                          :features (list)
                          :version "0.2"))
              id)))

(defun swank:create-repl (id)
  (list (list :return (list :ok (list "USER" "USER")) id)
        (list :new-package "USER" "USER")))

;; Stub: keine Contribs implementiert. SLIME akzeptiert leere Liste
;; (geladene Module), Connect laeuft durch.
(defun swank:swank-require (id)
  (list (list :return (list :ok (list)) id)))

;; swank:simple-completions (prefix pkg) -> (:ok (matching-strings...)).
;; SLIME nutzt completion-table-dynamic, erwartet Liste von Strings.
(defun swank:simple-completions (prefix id)
  (let ((matches (swank--filter-prefix prefix (swank--symbols) (list))))
    (list (list :return (list :ok matches) id))))

(defun swank--filter-prefix (prefix syms acc)
  (if (null? syms)
      acc
      (let ((s (car syms)))
        (swank--filter-prefix
          prefix
          (cdr syms)
          (if (swank--prefix? prefix s) (append acc (list s)) acc)))))

(defun swank--prefix? (prefix s)
  (if (> (string-length prefix) (string-length s))
      ()
      (equal? prefix (substring s 0 (string-length prefix)))))

;; swank:completions (prefix pkg) -> (:ok ((name) (name)...)).
;; swank-c-p-c Contrib: Client destrukturiert (symbol-name classification
;; symbol) pro Element; fehlende = nil. Also 1-Element-Liste pro Match.
(defun swank:completions (prefix id)
  (let ((matches (swank--filter-prefix prefix (swank--symbols) (list))))
    (list (list :return (list :ok (swank--wrap-each matches (list))) id))))

(defun swank--wrap-each (lst acc)
  (if (null? lst)
      acc
      (swank--wrap-each (cdr lst) (append acc (list (list (car lst)))))))

;; swank:operator-arglist (name pkg) -> (:ok "(name args)") | (:ok ()).
;; C-c C-d C-a / company-docsig. Lambda/Macro zuerst, dann Built-in.
(defun swank:operator-arglist (name id)
  (let ((al (or (swank--arglist name) (swank--built-in-arglist name))))
    (list (list :return (list :ok al) id))))

;; swank:autodoc (raw-form :print-right-margin N) -> (:ok (string cache-p)).
;; Vereinfacht: Operator aus raw-form, Arglist zeigen (ohne Highlighting
;; des aktuellen Args). Lambda/Macro zuerst, dann Built-in.
(defun swank:autodoc (form id)
  (let* ((quoted (cadr form))
         (rawform (cadr quoted))
         (op (car rawform)))
    (let ((al (or (swank--arglist op) (swank--built-in-arglist op))))
      (if (null? al)
          (list (list :return (list :ok (list :not-available nil)) id))
          (list (list :return (list :ok (list al nil)) id))))))

;; swank:swank-macroexpand-1 (string) -> (:ok "<expanded>").
;; C-c C-m. Eine Expansion via GoLisp macroexpand-Spezialform.
(defun swank:macroexpand-1 (string id)
  (catch
    (let ((form (read string)))
      (let ((expanded (macroexpand form)))
        (list (list :return (list :ok (swank--value-string expanded)) id))))
    (lambda (err)
      (list (list :return (list :abort (swank--value-string err)) id)))))

;; swank:swank-macroexpand / swank-expand (string) -> (:ok "<expanded>").
;; Wiederhole macroexpand bis stabil auf Top-Level.
(defun swank:macroexpand-full (string id)
  (catch
    (let ((form (read string)))
      (let ((expanded (swank--expand-top form)))
        (list (list :return (list :ok (swank--value-string expanded)) id))))
    (lambda (err)
      (list (list :return (list :abort (swank--value-string err)) id)))))

;; swank:swank-macroexpand-all / swank-expand-all (string) -> (:ok "<expanded>").
;; Echte rekursive Expansion in alle Subformen via GoLisp macroexpand-all.
(defun swank:macroexpand-all-handler (string id)
  (catch
    (let ((form (read string)))
      (let ((expanded (macroexpand-all form)))
        (list (list :return (list :ok (swank--value-string expanded)) id))))
    (lambda (err)
      (list (list :return (list :abort (swank--value-string err)) id)))))

(defun swank--expand-top (form)
  (let ((expanded (macroexpand form)))
    (if (equal? expanded form)
        form
        (swank--expand-top expanded))))

;; swank:load-file (filename) -> (:ok "<result>"). C-c C-l in Emacs.
;; Nutzt GoLisp load-Spezialform.
(defun swank:load-file (filename id)
  (catch
    (let ((result (eval (list (quote load) filename))))
      (list (list :return (list :ok (swank--value-string result)) id)))
    (lambda (err)
      (list (list :return (list :abort (swank--value-string err)) id)))))

(defun swank:listener-eval (string id)
  (catch
    (let ((forms (swank--read-all string)))
      (let ((events (swank--eval-forms forms (list))))
        (append events (list (list :return (list :ok (list)) id)))))
    (lambda (err)
      (list (list :return (list :abort (swank--value-string err)) id)))))

;; Wertet alle Formen, sammelt (:write-string "<wert>\n" :repl-result)
;; Events. eval ist Spezialform, daher Wrapper als echte FUNC.
(defun swank--eval1 (form) (eval form))

(defun swank--output-only-form? (form)
  (and (list? form)
       (not (null? form))
       (let ((op (car form)))
         (or (equal? op 'print)
             (equal? op 'println)
             (equal? op 'swank-print)
             (equal? op 'swank-println)
             (equal? op 'ga-print)
             (and (equal? op 'format)
                  (not (null? (cdr form)))
                  (equal? (cadr form) t))))))

(defun swank--eval-forms (forms acc)
  (if (null? forms)
      acc
      (let ((result (swank--eval1 (car forms))))
        (if (swank--output-only-form? (car forms))
            (swank--eval-forms (cdr forms) acc)
            (swank--eval-forms
              (cdr forms)
              (append acc (list (list :write-string
                                      (string-append (swank--value-string result) "\n")
                                      :repl-result))))))))

;; swank--eval-forms-silently: wie swank--eval-forms, aber ohne
;; :write-string Events. Für compile-string-for-emacs.
(defun swank--eval-forms-silently (forms)
  (if (null? forms)
      ()
      (begin
        (swank--eval1 (car forms))
        (swank--eval-forms-silently (cdr forms)))))

;; swank:compile-file-for-emacs (filename) -> (:ok (:filename ...)).
;; C-c C-k in Emacs. GoLisp hat keinen Compiler, daher ist "kompilieren"
;; synonym zu "laden".
(defun swank:compile-file-for-emacs (filename id)
  (catch
    (let ((result (eval (list (quote load) filename))))
      (list (list :return (list :ok (list :filename filename
                                          :result (swank--value-string result))) id)))
    (lambda (err)
      (list (list :return (list :abort (swank--value-string err)) id)))))

;; swank:compile-string-for-emacs (string) -> (:ok t).
;; Wertet alle Formen im String still aus.
(defun swank:compile-string-for-emacs (string id)
  (catch
    (let ((forms (swank--read-all string)))
      (swank--eval-forms-silently forms)
      (list (list :return (list :ok t) id)))
    (lambda (err)
      (list (list :return (list :abort (swank--value-string err)) id)))))

;; swank:find-definitions-for-emacs (name) -> (:ok ((name location))) | (:ok (:error "...")).
;; M-. in SLIME. Map-Lookup zuerst; sonst REPL-Snippet-Fallback oder :error.
;; SLIME erwartet Liste von (dspec location)-Paaren, keine nackten Locations.
(defun swank:find-definitions-for-emacs (name id)
  (catch
    (let ((loc (swank--find-definition name)))
      (let ((location
              (if (null? loc)
                  (swank--location-or-error name)
                  (list :location
                        (list :file (car loc))
                        (list :line (cdr loc))
                        (list)))))
        (let ((result
                (if (swank--error-location? location)
                    location
                    (list (list name location)))))
          (list (list :return (list :ok result) id)))))
    (lambda (err)
      (list (list :return (list :abort (swank--value-string err)) id)))))

;; Test ob eine Location ein :error-Ergebnis ist (kein Sprung-Ziel).
(defun swank--error-location? (loc)
  (and (list? loc) (not (null? loc)) (equal? (car loc) :error)))

;; Kein Map-Treffer: REPL-definiert (Lambda/Macro) -> Snippet-Buffer;
;; Built-in (FUNC ohne Env) oder unbound -> :error.
(defun swank--location-or-error (name)
  (let ((kind (swank--definition-kind name)))
    (cond
      ((or (equal? kind "lambda") (equal? kind "macro"))
       (swank--snippet-location name kind (swank--definition-cell name)))
      ((equal? kind "builtin")
       (list :error
         (string-append "eingebaute Funktion '" name "' hat keine Quellposition")))
      (else
       (list :error (string-append "Symbol '" name "' nicht definiert"))))))

(defun swank--snippet-location (name kind cell)
  (let ((header (if (equal? kind "macro")
                    "(defmacro " "(defun ")))
    (let ((snippet (string-append header name " "
                    (swank--value-string (car cell)) " "
                    (swank--value-string (cdr cell)) ")")))
      (list :location
            (list :buffer (string-append "*slime-source " name "*")
                  (list :source snippet))
            (list :position 1)
            (list)))))
