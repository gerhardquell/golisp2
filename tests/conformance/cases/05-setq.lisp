;; 05-setq.lisp — setq, psetq, setf
(let ((a 0)) (setq a 5) a)
(let ((a 0) (b 0)) (setq a 1 b 2) (list a b))
(let ((a 1) (b 2)) (psetq a b b a) (list a b))
(let ((a 1) (b 2)) (setq a b b a) (list a b))
(let ((x nil)) (setf (car x) 1))
(let ((xs (list 1 2 3))) (setf (nth 1 xs) 99) xs)
(setq *glob-aus-test* 7)
*glob-aus-test*
(let ((a 0)) (setq a (+ a 1) a (+ a 1)) a)
(let ((a 1) (b 2) (c 3)) (psetq a b b c c a) (list a b c))
