; testlib.lisp – kleine Bibliothek
(defun quadrat (n) (* n n))
(defun kubik (n) (* n n n))
(defun inc (n) (+ n 1))
(defun dec (n) (- n 1))

; TCO-Test: tiefe Rekursion
(defun countdown (n) (if (= n 0) "done" (countdown (- n 1))))
(println (countdown 1000000))

; gensym-Demo: swap!-Makro ohne Variablenkonflikt
(defmacro swap! (a b)
  (let ((tmp (gensym)))
    `(let ((,tmp ,a))
       (begin (set! ,a ,b) (set! ,b ,tmp)))))

(define x 10)
(define y 20)
(swap! x y)
(println x)   ; soll 20 ausgeben
(println y)   ; soll 10 ausgeben

; Error-Handling Demo
(defun safe-div (a b)
  (catch (if (= b 0) (error "Division durch 0") (/ a b))
         (lambda (e) (string-append "Fehler: " e))))

(println (safe-div 10 2))   ; 5  → kein Fehler
(println (safe-div 10 0))   ; "Fehler: Division durch 0"

; Multi-Body defun Demo
(defun between? (x lo hi)
  (define ok-lo (>= x lo))
  (define ok-hi (<= x hi))
  (and ok-lo ok-hi))

(println (between? 5 1 10))   ; t
(println (between? 15 1 10))  ; ()
