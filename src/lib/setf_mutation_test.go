//**********************************************************************
//  lib/setf_mutation_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude opus 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261122
//**********************************************************************
// Charakterisierungstests vor dem Umbau auf echte Cons-Mutation
// (setf/rplaca/rplacd/nconc, Zweig setf-mutation, Phase 1).
//
// Phase 2 (0.37.0): die früheren TestHeute*-Fälle sind auf das neue
// Verhalten umgedreht (setf ändert die Zelle, Aliase sehen es), dazu
// rplaca/rplacd/nconc/nthcdr, defstruct-Identität, Sandbox-Kopie,
// Ringlisten und Nebenläufigkeit.
// TestBleibt*: Verhalten, das sich durch Mutation NICHT ändern darf.
//**********************************************************************
package lib

import "testing"

// --- Phase 2: setf ändert die Zelle, Aliase sehen es (CL) ---

func TestSetfAliasSiehtAenderung(t *testing.T) {
  evalStdlibEq(t, `(begin (define l (list 1 2 3)) (define m l)
                          (setf (car l) 9) (list l m))`, "((9 2 3) (9 2 3))")
  evalStdlibEq(t, `(begin (define l (list 1 2 3)) (define m l)
                          (setf (cdr l) (list 8 9)) (list l m))`, "((1 8 9) (1 8 9))")
  evalStdlibEq(t, `(begin (define l (list 1 2 3)) (define m l)
                          (setf (nth 1 l) 99) (list l m))`, "((1 99 3) (1 99 3))")
  // die Teilliste ist dieselbe Zelle
  evalStdlibEq(t, `(begin (define l (list 0 1 2 3)) (define r (cdr l))
                          (setf (car r) 9) l)`, "(0 9 2 3)")
}

func TestSetfFunktionSiehtAenderung(t *testing.T) {
  // das Argument ist dieselbe Zelle wie beim Aufrufer
  evalStdlibEq(t, `(begin (defun zz-mut (x) (setf (car x) 9) x)
                          (define l (list 1 2)) (zz-mut l) l)`, "(9 2)")
}

func TestSetfVerschachtelt(t *testing.T) {
  evalStdlibEq(t, `(let ((l (list 1 2 3))) (setf (car (cdr l)) 9) l)`, "(1 9 3)")
  evalStdlibEq(t, `(let ((m (list (list 1 2 3) (list 4 5 6))))
                      (setf (nth 2 (nth 1 m)) 99) m)`, "((1 2 3) (4 5 99))")
  // c[ad]+r-Places und first … fourth, rest
  evalStdlibEq(t, `(let ((l (list 1 2 3))) (setf (cadr l) 9) l)`, "(1 9 3)")
  evalStdlibEq(t, `(let ((l (list 1 2 3))) (setf (second l) 9) l)`, "(1 9 3)")
  evalStdlibEq(t, `(let ((l (list 1 2 3 4)))
                     (setf (first l) 'a (third l) 'c (fourth l) 'd) l)`, "(a 2 c d)")
  evalStdlibEq(t, `(let ((l (list 1 2 3 4))) (setf (caddr l) 'x (cadddr l) 'y) l)`, "(1 2 x y)")
  evalStdlibEq(t, `(let ((l (list 1 2 3))) (setf (cddr l) (list 7)) l)`, "(1 2 7)")
  evalStdlibEq(t, `(let ((l (list 1 2 3))) (setf (rest l) (list 7)) l)`, "(1 7)")
  evalStdlibEq(t, `(let ((l (list (list 1 2) 3))) (setf (caar l) 'a (cdar l) 'b) l)`, "((a . b) 3)")
}

func TestSetfListenPlatzRegeln(t *testing.T) {
  // Rückgabe ist der neue Wert, Auswertung links nach rechts, je einmal
  evalStdlibEq(t, `(let ((l (list 1 2 3))) (setf (nth 1 l) 7))`, "7")
  evalStdlibEq(t, `(let ((spur ()) (l (list 1 2 3)))
                     (setf (nth (begin (push 'i spur) 1) (begin (push 'x spur) l))
                           (begin (push 'v spur) 5))
                     (list (reverse spur) l))`, "((i x v) (1 5 3))")
  // Index jenseits des Endes, () und Nicht-Listen: keine Cons-Zelle
  evalStdlibErr(t, `(let ((l (list 1 2))) (setf (nth 5 l) 9))`)
  evalStdlibErr(t, `(setf (car ()) 1)`)
  evalStdlibErr(t, `(setf (cdr 5) 1)`)
  // incf/decf/pop auf Listen-Places mutieren ebenfalls
  evalStdlibEq(t, `(begin (define l (list 1 2)) (define m l) (incf (nth 1 l) 10) (decf (car l)) m)`, "(0 12)")
  evalStdlibEq(t, `(begin (define l (list 1 (list 2 3))) (define m l) (list (pop (cadr l)) m))`, "(2 (1 (3)))")
  // set-nth bleibt die kopierende Funktion
  evalStdlibEq(t, `(let ((l (list 1 2 3))) (list (set-nth l 1 9) l))`, "((1 9 3) (1 2 3))")
}

func TestRplacaRplacdNconcNthcdr(t *testing.T) {
  evalStdlibEq(t, `(rplaca (list 1 2) 9)`, "(9 2)")
  evalStdlibEq(t, `(rplacd (list 1 2) 9)`, "(1 . 9)")
  evalStdlibEq(t, `(let ((c (list 1))) (eq (rplaca c 2) c))`, "t")
  evalStdlibEq(t, `(nconc (list 1 2) (list 3 4))`, "(1 2 3 4)")
  // nur Type == LIST wird geändert — (), Symbole, Zahlen, Funktionen nie
  for _, form := range []string{
    `(rplaca () 1)`, `(rplacd () 1)`, `(rplaca 'foo 1)`, `(rplaca 5 1)`,
    `(rplaca "ab" 1)`, `(rplaca (lambda (a) a) 1)`, `(rplacd (values 1 2) 1)`, `(rplaca (list 1))`,
  } {
    evalStdlibErr(t, form)
  }
  // nconc: ()-Argumente übersprungen, letztes beliebig, Ergebnis = erste nicht-leere Liste
  evalStdlibEq(t, `(let ((a (list 1)) (b (list 2))) (list (nconc () a () b 3) a (eq (nconc a) a)))`,
    "((1 2 . 3) (1 2 . 3) t)")
  evalStdlibEq(t, `(list (nconc) (nconc () ()) (nconc () 3))`, "(() () 3)")
  evalStdlibErr(t, `(nconc 3 (list 1))`)
  evalStdlibErr(t, `(nconc (cons 1 2) (list 1))`)
  // Go-nil-Listenende (MakeValues) wie ()
  evalStdlibEq(t, `(nconc (multiple-value-list (values 1 2)) (list 3))`, "(1 2 3)")
  // nthcdr teilt, über das Ende → (), n muss Ganzzahl >= 0 sein
  evalStdlibEq(t, `(let ((l (list 1 2 3))) (list (eq (nthcdr 1 l) (cdr l)) (nthcdr 0 l) (nthcdr 9 l)))`,
    "(t (1 2 3) ())")
  evalStdlibErr(t, `(nthcdr -1 (list 1))`)
  evalStdlibErr(t, `(nthcdr 1.0 (list 1))`)
  // %last-cons
  evalStdlibEq(t, `(let ((l (list 1 2 3))) (list (%last-cons l) (eq (%last-cons l) (cddr l)) (%last-cons ())))`,
    "((3) t ())")
  evalStdlibEq(t, `(%last-cons (cdr (multiple-value-list (values 1 2))))`, "(2)")
}

func TestStructHatIdentitaet(t *testing.T) {
  evalStdlibEq(t, `(begin (defstruct zzpt x y) (define p (make-zzpt :x 1 :y 2))
                          (define q p) (setf (zzpt-x p) 9) (list (zzpt-x p) (zzpt-x q)))`, "(9 9)")
  // Setter liefern den Wert, beliebiger Ausdruck als Objekt
  evalStdlibEq(t, `(begin (defstruct zzpt2 x) (define ps (list (make-zzpt2 :x 1)))
                          (list (setf (zzpt2-x (car ps)) 5) (zzpt2-x (car ps)) (set-zzpt2-x (car ps) 6) ps))`,
    "(5 5 6 ((zzpt2 6)))")
  evalStdlibEq(t, `(begin (defstruct zzpt3 n) (define p (make-zzpt3 :n 1)) (incf (zzpt3-n p) 2) p)`, "(zzpt3 3)")
}

func TestPushAufPlatzMutiert(t *testing.T) {
  evalStdlibEq(t, `(begin (define l (list 1 2)) (define m l)
                          (push 0 (cdr l)) (list l m))`, "((1 0 2) (1 0 2))")
}

func TestQuoteLiteralWirdGeaendert(t *testing.T) {
  // quote liefert die Reader-Zelle des Funktionsrumpfs, jeder Aufruf dieselbe.
  // Ändern ist undefiniert (CL) und ändert still den Rumpf — nur dokumentiert.
  evalStdlibEq(t, `(begin (defun zz-lit () '(1 2 3)) (eq (zz-lit) (zz-lit)))`, "t")
  evalStdlibEq(t, `(begin (defun zz-lit () '(1 2 3)) (define a (zz-lit))
                          (setf (car a) 9) (zz-lit))`, "(9 2 3)")
}

func TestCarAufLambdaLiefertCode(t *testing.T) {
  // car/cdr akzeptieren LAMBDA/MACRO: Parameterliste bzw. Rumpf (Spec
  // Entscheidung 5, bleibt). rplaca auf die Lambda selbst ist ein Fehler.
  evalStdlibEq(t, `(car (lambda (a b) (+ a b)))`, "(a b)")
  evalStdlibEq(t, `(cdr (lambda (a b) (+ a b)))`, "(+ a b)")
  evalStdlibErr(t, `(rplaca (lambda (a b) (+ a b)) 1)`)
}

func TestSandboxKopiertAnDerGrenze(t *testing.T) {
  // Form hinein und Ergebnis heraus gehen per Baumkopie
  evalStdlibEq(t, `(begin (define sb (make-sandbox)) (define l (list 1 2))
                          (eq (sandbox-eval sb (list 'quote l)) l))`, "()")
  evalStdlibEq(t, `(begin (define sb (make-sandbox)) (define l (list 1 (list 2)))
                          (list (equal? (sandbox-eval sb (list 'quote l)) l)
                                (sandbox-eval sb (list 'rplaca (list 'quote l) 9)) l))`,
    "(t (9 (2)) (1 (2)))")
  // Ergebnis: die Sandbox behält ihre Liste, der Host bekommt eine Kopie
  evalStdlibEq(t, `(begin (define sb (make-sandbox))
                          (sandbox-eval sb "(define *zz-sb* (list 1 (list 2)))")
                          (define r (sandbox-eval sb '*zz-sb*))
                          (setf (car (cadr r)) 9)
                          (list r (sandbox-eval sb '*zz-sb*)))`, "((1 (9)) (1 (2)))")
  // Mehrfachwerte wertweise kopiert
  evalStdlibEq(t, `(begin (define sb (make-sandbox))
                          (multiple-value-list (sandbox-eval sb "(values (list 1) 2)")))`, "((1) 2)")
  // Vektoren und Hash-Tabellen werden mitkopiert, in beide Richtungen
  evalStdlibEq(t, `(begin (define sb (make-sandbox)) (define hl (list 1)) (define h (make-hash-table))
                          (puthash 'k h hl) (define v (vector hl))
                          (sandbox-eval sb (list 'rplaca (list 'gethash ''k (list 'quote h)) 9))
                          (sandbox-eval sb (list 'rplaca (list 'aref (list 'quote v) 0) 8))
                          (sandbox-eval sb "(define *zz-v* (vector (list 1))) (define *zz-h* (make-hash-table))
                                            (puthash (list 1) *zz-h* 'eql-liste)")
                          (rplaca (aref (sandbox-eval sb '*zz-v*) 0) 7)
                          (list hl (sandbox-eval sb '*zz-v*)
                                (eq (sandbox-eval sb '*zz-h*) (sandbox-eval sb '*zz-h*))
                                (hash-table-count (sandbox-eval sb '*zz-h*))))`,
    "((1) #((1)) () 1)")
  // geteilte Struktur bleibt geteilt, O(n): 2^60 logische Zellen
  evalStdlibEq(t, `(begin (define sb (make-sandbox))
                          (define d (let ((x (list 1))) (dotimes (i 60) (set! x (list x x))) x))
                          (let ((k (sandbox-eval sb (list 'quote d))))
                            (list (eq (car k) (cadr k)) (eq k d))))`, "(t ())")
  evalStdlibEq(t, `(begin (define sb (make-sandbox))
                          (let ((r (sandbox-eval sb "(let ((x (list 1))) (dotimes (i 60) (setq x (list x x))) x)")))
                            (eq (car r) (cadr r))))`, "t")
  // Zyklus über einen Vektor: laut
  evalStdlibErr(t, `(begin (define sb (make-sandbox))
                           (sandbox-eval sb "(let ((v (vector 1))) (aset v 0 (list v)) v)"))`)
  // Ringliste an der Grenze: laut
  evalStdlibErr(t, `(begin (define sb (make-sandbox)) (define l (list 1 2)) (rplacd (cdr l) l)
                           (sandbox-eval sb (list 'quote l)))`)
  evalStdlibErr(t, `(begin (define sb (make-sandbox))
                           (sandbox-eval sb "(let ((l (list 1 2))) (rplacd (cdr l) l) l)"))`)
}

// --- Ringlisten: kein Hänger, kein Absturz ---

func TestRinglisten(t *testing.T) {
  ring := `(define r (list 1 2 3)) (rplacd (cddr r) r)`
  // Drucker bricht mit Markierung ab — auch über format und Car-Ring
  evalStdlibEq(t, `(begin `+ring+` (format nil "~a" r))`, `"(1 2 3 1 2 3 …zirkulär)"`)
  evalStdlibEq(t, `(begin (define c (list 1)) (rplacd c c) (format nil "~s" c))`, `"(1 …zirkulär)"`)
  evalStdlibEq(t, `(begin (define a (list 1 2)) (rplaca a a)
                          (> (length (format nil "~a" a)) 10000))`, "t")
  // laute Fehler statt Endlosschleife
  for _, form := range []string{
    `(length r)`, `(copy-tree r)`,
    `(let ((h (make-hash-table :test 'equal))) (puthash r h 1))`,
    `(let ((h (make-hash-table :test 'equal))) (gethash r h))`,
    `(append r (list 4))`, `(apply list r)`, `(%last-cons r)`, `(nconc r (list 4))`,
    `(list->vector r)`,
  } {
    evalStdlibErr(t, `(begin `+ring+` `+form+`)`)
  }
  // zwei verschiedene Ringe gleichen Inhalts; derselbe Ring ist equal
  evalStdlibErr(t, `(begin `+ring+` (define s (list 1 2 3)) (rplacd (cddr s) s) (equal? r s))`)
  evalStdlibEq(t, `(begin `+ring+` (equal? r r))`, "t")
  // Ring gegen endliche Liste: einfach ungleich
  evalStdlibEq(t, `(begin `+ring+` (equal? r (list 1 2 3)))`, "()")
  // Car-Ringe: equal? meldet statt Go-Stack-Overflow
  evalStdlibErr(t, `(begin (define a (list 1)) (rplaca a a) (define b (list 1)) (rplaca b b) (equal? a b))`)
  // eql-Hashtabelle nimmt den Ring nach Identität
  evalStdlibEq(t, `(begin `+ring+` (let ((h (make-hash-table))) (puthash r h 1) (gethash r h)))`, "1")
}

func TestRingCellToSlicePanikWirdFehler(t *testing.T) {
  r := Cons(MakeInt(1), MakeNil())
  r.Cdr = r
  defer func() {
    if recover() == nil {
      t.Error("CellToSlice auf Ringliste: panic erwartet")
    }
  }()
  CellToSlice(r)
}

// --- Nebenläufigkeit: Mutation unter lock, -race sauber ---

func TestRplacaUnterLockAusSpawns(t *testing.T) {
  evalStdlibEq(t, `(begin
    (define zz-l (list 0))
    (define zz-m (lock-make))
    (define zz-ch (chan-make 8))
    (dotimes (i 8)
      (spawn (lambda ()
               (dotimes (j 50) (lock zz-m (rplaca zz-l (+ (car zz-l) 1))))
               (chan-send zz-ch t))))
    (dotimes (i 8) (chan-recv zz-ch))
    (car zz-l))`, "400")
}

// --- Bleibt: darf sich durch Mutation nicht ändern ---

func TestBleibtEqUndNil(t *testing.T) {
  evalStdlibEq(t, `(eq 'foo 'foo)`, "t")
  evalStdlibEq(t, `(eq '() nil)`, "t")
  evalStdlibEq(t, `(eq (cdr (list 1)) nil)`, "t")
  evalStdlibEq(t, `(eq (= 1 2) '())`, "t")
  evalStdlibEq(t, `(eq 5 5)`, "()")
  evalStdlibEq(t, `(eq (list 1) (list 1))`, "()")
  evalStdlibEq(t, `(let ((l (list 1))) (eq l l))`, "t")
}

func TestBleibtStrukturteilung(t *testing.T) {
  // append teilt das letzte Argument, kopiert die übrigen (CL)
  evalStdlibEq(t, `(let ((a (list 1)) (b (list 2 3))) (list (eq (cdr (append a b)) b) (eq (append a b) a)))`, "(t ())")
  evalStdlibEq(t, `(let ((a (list 1))) (eq (append a) a))`, "t")
  // Tails/Einträge sind geteilt (CL)
  evalStdlibEq(t, `(let ((l (list 1 2 3))) (eq (member 2 l) (cdr l)))`, "t")
  evalStdlibEq(t, `(let ((al (list (list 'a 1)))) (eq (assoc 'a al) (car al)))`, "t")
  evalStdlibEq(t, `(let ((l (list 1 2 3))) (list (eq (drop 1 l) (cdr l)) (eq (list-tail l 1) (cdr l)) (eq (cdr l) (cdr l))))`, "(t t t)")
  // frisch: copy-list, copy-tree, reverse, sort, filter/remove, mapcar, take, subseq, copy-seq
  evalStdlibEq(t, `(let ((l (list 1 (list 2) 3)))
                     (list (eq (copy-list l) l) (eq (cadr (copy-list l)) (cadr l))
                           (eq (cadr (copy-tree l)) (cadr l)) (equal? (copy-tree l) l)))`, "(() t () t)")
  evalStdlibEq(t, `(let ((l (list 1 2 3)))
                     (list (eq (reverse (reverse l)) l) (eq (sort l <) l) (eq (remove 9 l) l)
                           (eq (filter (lambda (x) t) l) l) (eq (mapcar identity l) l)
                           (eq (take 3 l) l) (eq (subseq l 0) l) (eq (copy-seq l) l)))`,
    "(() () () () () () () ())")
  // last liefert heute das Element, nicht die letzte Cons-Zelle (CL: (3))
  evalStdlibEq(t, `(last (list 1 2 3))`, "3")
}

func TestBleibtRestListen(t *testing.T) {
  // lambda-&rest ist frisch (CL erlaubt Teilen, golisp2 kopiert)
  evalStdlibEq(t, `(begin (defun zz-r (&rest r) r) (let ((l (list 1 2))) (eq (apply zz-r l) l)))`, "()")
  // destructuring-bind-&rest teilt das Listenende (docs/lisp-semantik.md)
  evalStdlibEq(t, `(let ((l (list 1 2 3))) (destructuring-bind (a &rest r) l (eq r (cdr l))))`, "t")
  // Makro-&rest teilt die Aufrufform — Mutation dort änderte Quellcode
  evalStdlibEq(t, `(begin (defmacro zz-m (&rest r) (list 'quote r)) (zz-m 1 2))`, "(1 2)")
}

func TestBleibtQuasiquoteFrisch(t *testing.T) {
  // jede Auswertung baut neue Zellen, auch ein ,@ am Ende wird kopiert
  evalStdlibEq(t, "(begin (defun zz-qq () `(1 2 3)) (eq (zz-qq) (zz-qq)))", "()")
  evalStdlibEq(t, "(let ((x (list 2 3))) (eq (cdr `(1 ,@x)) x))", "()")
  // ,x setzt den Wert selbst ein (geteilt, CL)
  evalStdlibEq(t, "(let ((x (list 2 3))) (eq (cadr `(1 ,x)) x))", "t")
}

func TestBleibtMultipleValuesFrisch(t *testing.T) {
  evalStdlibEq(t, `(begin (defun zz-mv () (values 1 2))
                          (eq (multiple-value-list (zz-mv)) (multiple-value-list (zz-mv))))`, "()")
}

func TestBleibtHashtabelle(t *testing.T) {
  // equal-Schlüssel: strukturell, Schlüssel-String bei puthash berechnet
  evalStdlibEq(t, `(let ((h (make-hash-table :test 'equal)))
                     (puthash (list 1 2) h 'v) (gethash (list 1 2) h))`, "v")
  // eql-Schlüssel: Listen nach Identität
  evalStdlibEq(t, `(let ((h (make-hash-table)) (k (list 1 2)))
                     (puthash k h 'v) (list (gethash k h) (gethash (list 1 2) h)))`, "(v ())")
}

func TestBleibtSandboxIsoliert(t *testing.T) {
  // Definitionen der Sandbox erreichen den Host nicht
  evalStdlibEq(t, `(begin (define sb (make-sandbox))
                          (sandbox-eval sb "(defun zz-sbf () 1) (define *zz-sbv* (list 1))")
                          (list (bound? 'zz-sbf) (bound? '*zz-sbv*)))`, "(() ())")
  // stdlib wird je Env frisch gelesen: Sandbox-reverse ist nicht Host-reverse
  evalStdlibEq(t, `(begin (define sb (make-sandbox))
                          (eq (sandbox-eval sb 'reverse) reverse))`, "()")
}

func TestBleibtTCOTief(t *testing.T) {
  evalStdlibEq(t, `(begin (defun zz-tco (n acc) (if (= n 0) acc (zz-tco (- n 1) (cons n acc))))
                          (length (zz-tco 200000 ())))`, "200000")
}
