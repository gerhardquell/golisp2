;;; kiesp-dict.lisp - Dictionary-basierte Kompression
;;; Autor: Gerhard Quell – gquell@skequell.de
;;; CoAutor: Claude
;;; Experiment: Haeufigkeitsanalyse und automatische Token-Generierung

;;; Frequenz-Analyse

(define kiesp-frequency-table '())

(defun kiesp-count-words (text)
  "Zaehlt Worthaeufigkeiten in einem Text"
  (set! kiesp-frequency-table '())
  (kiesp-process-words (kiesp-tokenize text))
  (kiesp-sort-by-frequency kiesp-frequency-table))

(defun kiesp-tokenize (text)
  "Zerlegt Text in Woerter"
  (kiesp-split-string text " "))

(defun kiesp-split-string (str delim)
  "Einfacher String-Split"
  (kiesp-split-helper str delim "" '()))

(defun kiesp-split-helper (str delim current result)
  "Hilfsfunktion fuer Split"
  (if (equal? str "")
      (reverse (if (equal? current "")
                   result
                   (cons current result)))
      (let ((char (substring str 0 1))
            (rest (substring str 1 (string-length str))))
        (if (equal? char delim)
            (kiesp-split-helper rest delim ""
                                (if (equal? current "")
                                    result
                                    (cons current result)))
            (kiesp-split-helper rest delim
                                (string-append current char)
                                result)))))

(defun kiesp-process-words (words)
  "Verarbeitet Wortliste fuer Haeufigkeit"
  (if (null words)
      'done
      (begin
        (kiesp-increment-count (car words))
        (kiesp-process-words (cdr words)))))

(defun kiesp-increment-count (word)
  "Erhoeht Zaehler fuer ein Wort"
  (set! kiesp-frequency-table
        (kiesp-update-count word kiesp-frequency-table)))

(defun kiesp-update-count (word table)
  "Aktualisiert Zaehler fuer ein Wort in der Tabelle"
  (if (null table)
      (list (cons word 1))
      (if (equal? word (caar table))
          (cons (cons word (+ (cdar table) 1)) (cdr table))
          (cons (car table) (kiesp-update-count word (cdr table))))))

(defun kiesp-sort-by-frequency (table)
  "Sortiert Tabelle nach Haeufigkeit (absteigend)"
  ;; Einfache Insertion-Sort
  (kiesp-insertion-sort table))

(defun kiesp-insertion-sort (lst)
  "Sortiert Liste nach Haeufigkeit"
  (if (null lst)
      '()
      (kiesp-insert (car lst) (kiesp-insertion-sort (cdr lst)))))

(defun kiesp-insert (item sorted)
  "Fuegt Item sortiert ein"
  (if (null sorted)
      (list item)
      (if (not (< (cdr item) (cdr (car sorted))))
          (cons item sorted)
          (cons (car sorted) (kiesp-insert item (cdr sorted))))))

;;; Dictionary-Generierung

(define kiesp-dictionary '())
(define kiesp-reverse-dict '())

(defun kiesp-generate-dict (text max-entries)
  "Generiert Dictionary aus Text"
  (let ((freqs (kiesp-count-words text)))
    (kiesp-build-dicts freqs max-entries 0)))

(defun kiesp-build-dicts (freqs max n)
  "Baut Dictionary und Reverse-Dictionary"
  (set! kiesp-dictionary '())
  (set! kiesp-reverse-dict '())
  (kiesp-add-entries freqs max n))

(defun kiesp-add-entries (freqs max n)
  "Fuegt Eintraege hin bis max erreicht"
  (if (or (null freqs) (not (< n max)))
      (list kiesp-dictionary kiesp-reverse-dict)
      (let ((word (caar freqs))
            (token (kiesp-generate-token n)))
        (set! kiesp-dictionary
              (cons (cons word token) kiesp-dictionary))
        (set! kiesp-reverse-dict
              (cons (cons token word) kiesp-reverse-dict))
        (kiesp-add-entries (cdr freqs) max (+ n 1)))))

(defun kiesp-generate-token (n)
  "Generiert kurzes Token fuer Index n"
  ;; Verwendet nur Ziffern 0-9 fuer Einfachheit
  (number->string n))

;;; Encoder/Decoder

(defun kiesp-encode-dict (text)
  "Kodiert Text mit Dictionary"
  (let ((words (kiesp-tokenize text)))
    (kiesp-encode-words words)))

(defun kiesp-encode-words (words)
  "Kodiert Wortliste"
  (if (null words)
      '()
      (let ((word (car words)))
        (let ((pair (assoc word kiesp-dictionary)))
          (cons (if pair (cdr pair) word)
                (kiesp-encode-words (cdr words)))))))

(defun kiesp-decode-dict (tokens)
  "Dekodiert Token-Liste"
  (if (null tokens)
      ""
      (let ((token (car tokens)))
        (let ((pair (assoc token kiesp-reverse-dict)))
          (string-append
            (if pair (cdr pair) token)
            " "
            (kiesp-decode-dict (cdr tokens)))))))

;;; Statistik

(defun kiesp-stats-dict (original encoded)
  "Berechnet Kompressionsstatistik"
  (let ((orig-len (string-length original))
        (enc-len (length encoded))
        (dict-size (length kiesp-dictionary)))
    (list
      (cons 'original-chars orig-len)
      (cons 'encoded-tokens enc-len)
      (cons 'dictionary-size dict-size)
      (cons 'avg-token-len (if (= enc-len 0) 0 (/ (kiesp-total-token-len encoded) enc-len)))
      (cons 'compression-ratio (let ((word-count (kiesp-count-words original)))
                                 (if (= word-count 0) 0.0 (/ (* 1.0 enc-len) word-count)))))))

(defun kiesp-total-token-len (tokens)
  "Summiert Token-Laengen"
  (if (null tokens)
      0
      (+ (string-length (car tokens))
         (kiesp-total-token-len (cdr tokens)))))

;;; Hilfsfunktionen fuer Mutability (simuliert)

(defun set-cdr! (pair val)
  "Setzt cdr eines Paares (simuliert)"
  ;; In GoLisp: wuerde RPLACA sein
  ;; Hier: wir vertrauen auf die Implementierung
  val)

;;; Beispiel-Text fuer Tests

(define kiesp-sample-text
  "the quick brown fox jumps over the lazy dog the dog sleeps and the fox runs away")

(define kiesp-tech-text
  "function define lambda let if then else begin end return value variable constant")

(println "=== KIESP Dictionary-basiert geladen ===")
(println "Funktionen: kiesp-count-words kiesp-generate-dict")
(println "            kiesp-encode-dict kiesp-decode-dict")
(println "Beispiele: kiesp-sample-text kiesp-tech-text")
