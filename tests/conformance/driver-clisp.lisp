;;;; driver-clisp.lisp — Gold-Generator für die Konformitäts-Suite
;;;; Autor: Gerhard Quell – gquell@skequell.de · CoAutor: kimi-k3
;;;; Erstellt: 20260723
;;;;
;;;; Liest eine Case-Datei (eine Form pro Zeile), evaluiert jede Form in
;;;; frischem Prozesskontext und schreibt pro Fall genau eine Gold-Zeile:
;;;;   <write-to-string des Primärwerts, downcased>   oder   ERROR
;;;; Aufruf: clisp -q -norc driver-clisp.lisp <case-datei> > <gold-datei>

(defun gold-eval (form)
  (handler-case
      (string-downcase (write-to-string (eval form) :pretty nil))
    (error () "ERROR")))

(defun gold-run (pfad out)
  (with-open-file (in pfad :direction :input)
    (loop for zeile = (read-line in nil :eof)
          until (eq zeile :eof)
          for getrimmt = (string-trim '(#\Space #\Tab #\Return) zeile)
          unless (or (zerop (length getrimmt))
                     (and (>= (length getrimmt) 2)
                          (string= ";;" (subseq getrimmt 0 2))))
          do (let ((form (handler-case
                             (read-from-string getrimmt)
                           (error () :read-error))))
               (format out "~A~%"
                       (if (eq form :read-error)
                           "ERROR"
                           (gold-eval form)))))))

(gold-run (first ext:*args*) *standard-output*)
