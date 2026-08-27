;;; kiesp-hybrid.lisp - Hybrid-Ansatz

(load "experiment/kiesp-stack.lisp")
(load "experiment/kiesp-sexp.lisp")
(load "experiment/kiesp-dict.lisp")

(define kiesp-context 'unknown)

(defun kiesp-detect-context (text)
  "Erkennt Kontexttyp"
  (set! kiesp-context
    (cond
      ((kiesp-codep text) 'code)
      ((kiesp-narrativep text) 'narrative)
      ((kiesp-datap text) 'data)
      (else 'general)))
  kiesp-context)

(defun kiesp-codep (text)
  "Prueft auf Code-Muster"
  (or (kiesp-has text "(")
      (kiesp-has text "defun")
      (kiesp-has text "lambda")
      (kiesp-has text "define")))

(defun kiesp-narrativep (text)
  "Prueft auf narrative Muster"
  (or (kiesp-has text "the ")
      (kiesp-has text " and ")
      (kiesp-has text ". ")))

(defun kiesp-datap (text)
  "Prueft auf Daten-Muster"
  (or (kiesp-has text "{")
      (kiesp-has text "[")
      (kiesp-has text ": ")))

(defun kiesp-has-help (text substr start)
  "Hilfsfunktion"
  (if (> (+ start (string-length substr)) (string-length text))
      '()
      (if (equal? (substring text start (+ start (string-length substr))) substr)
          't
          (kiesp-has-help text substr (+ start 1)))))

(defun kiesp-has (text substr)
  "Prueft ob String enthalten"
  (kiesp-has-help text substr 0))

(defun kiesp-encode (text)
  "Waehlt Encoder"
  (kiesp-detect-context text)
  (cond
    ((eq kiesp-context 'code) (kiesp-encode-code text))
    ((eq kiesp-context 'narrative) (kiesp-encode-narrative text))
    ((eq kiesp-context 'data) (kiesp-encode-data text))
    (else (kiesp-encode-narrative text))))

(defun kiesp-encode-code (text)
  "Kodiert Code"
  (list
    (cons 'context 'code)
    (cons 'method 'sexp)
    (cons 'compressed 'code-markers)
    (cons 'original-chars (string-length text))))

(defun kiesp-encode-narrative (text)
  "Kodiert Narrativ"
  (let ((dict-result (kiesp-generate-dict text 20)))
    (list
      (cons 'context 'narrative)
      (cons 'method 'dictionary)
      (cons 'compressed (kiesp-encode-dict text))
      (cons 'dictionary (car dict-result))
      (cons 'stats (kiesp-stats-dict text (kiesp-encode-dict text))))))

(defun kiesp-encode-data (text)
  "Kodiert Daten"
  (let ((tokens (kiesp-tokenize text)))
    (list
      (cons 'context 'data)
      (cons 'method 'stack)
      (cons 'compressed 'stack-mode)
      (cons 'original-tokens (length tokens)))))

(defun kiesp-measure (original encoded)
  "Misst Effizienz"
  (let ((orig-size (max 1 (kiesp-size original)))
        (enc-size (kiesp-size encoded)))
    (list
      (cons 'original-bytes orig-size)
      (cons 'encoded-bytes enc-size)
      (cons 'ratio (/ (* 100.0 enc-size) orig-size))
      (cons 'savings (* 100.0 (- 1.0 (/ enc-size orig-size))))
      (cons 'efficiency (kiesp-eff-score orig-size enc-size)))))

(defun kiesp-size (x)
  "Schaetzt Groesse"
  (cond
    ((null x) 2)
    ((atom x) 8)
    (else (+ 2 (kiesp-size (car x)) (kiesp-size (cdr x))))))

(defun kiesp-eff-score (orig enc)
  "Effizienz-Score"
  (if (= orig 0)
      0.0
      (let ((compression (- 1.0 (/ enc orig)))
            (overhead (if (< enc orig) 0.0 (- 1.0 (/ orig (max 1 enc))))))
        (* 50.0 (+ compression (- 1.0 overhead))))))

(define kiesp-benchmark-results '())

(defun kiesp-benchmark (name text)
  "Fuehrt Benchmark durch"
  (let ((start (memstats))
        (encoded (kiesp-encode text))
        (end (memstats)))
    (let ((result (list
                    (cons 'name name)
                    (cons 'context kiesp-context)
                    (cons 'encoded encoded)
                    (cons 'measurement (kiesp-measure text encoded))
                    (cons 'memory-used (- (cdr (assoc 'heapalloc end))
                                          (cdr (assoc 'heapalloc start)))))))
      (set! kiesp-benchmark-results (cons result kiesp-benchmark-results))
      result)))

(defun kiesp-report ()
  "Benchmark-Report"
  (println "=== KIESP Hybrid Benchmark Report ===")
  (kiesp-print-results kiesp-benchmark-results))

(defun kiesp-print-results (results)
  "Druckt Ergebnisse"
  (if (null results)
      (println "=== Ende Report ===")
      (begin
        (println "Test: " (cdr (assoc 'name (car results))))
        (println "  Context: " (cdr (assoc 'context (car results))))
        (println "  Stats:   " (cdr (assoc 'measurement (car results))))
        (kiesp-print-results (cdr results)))))

(define kiesp-test-code
  "(define factorial (lambda (n) (if (= n 0) 1 (* n (factorial (- n 1))))))")

(define kiesp-test-narrative
  "The quick brown fox jumps over the lazy dog and runs away into the forest")

(define kiesp-test-data
  "name: John age: 30 city: New York active: true")

(println "=== KIESP Hybrid geladen ===")
(println "Funktionen: kiesp-detect-context kiesp-encode kiesp-measure kiesp-benchmark")
(println "Test-Texte: kiesp-test-code kiesp-test-narrative kiesp-test-data")
