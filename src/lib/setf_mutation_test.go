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
// TestHeute*: halten das HEUTIGE Verhalten fest, das Phase 2 umdreht.
//   Jeder Fall nennt im Kommentar „Phase 2: …" die CL-Erwartung.
// TestBleibt*: Verhalten, das sich durch Mutation NICHT ändern darf.
// Siehe auch TestSetfCarCdr (stdlib_reduce_setf_test.go) — dort stehen
// Alias- und Nicht-Symbol-Fälle, die Phase 2 ebenfalls umdreht.
//**********************************************************************
package lib

import "testing"

// --- Heute: setf bindet die Variable neu, Aliase sehen nichts ---

func TestHeuteSetfAliasSiehtNichts(t *testing.T) {
  // Phase 2: ((9 2 3) (9 2 3))
  evalStdlibEq(t, `(begin (define l (list 1 2 3)) (define m l)
                          (setf (car l) 9) (list l m))`, "((9 2 3) (1 2 3))")
  // Phase 2: ((1 8 9) (1 8 9))
  evalStdlibEq(t, `(begin (define l (list 1 2 3)) (define m l)
                          (setf (cdr l) (list 8 9)) (list l m))`, "((1 8 9) (1 2 3))")
  // Phase 2: ((1 99 3) (1 99 3))
  evalStdlibEq(t, `(begin (define l (list 1 2 3)) (define m l)
                          (setf (nth 1 l) 99) (list l m))`, "((1 99 3) (1 2 3))")
  // Phase 2: (0 9 3) — die Teilliste ist dieselbe Zelle
  evalStdlibEq(t, `(begin (define l (list 0 1 2 3)) (define r (cdr l))
                          (setf (car r) 9) l)`, "(0 1 2 3)")
}

func TestHeuteSetfFunktionSiehtNichts(t *testing.T) {
  // Phase 2: (9 2) — das Argument ist dieselbe Zelle wie beim Aufrufer
  evalStdlibEq(t, `(begin (defun zz-mut (x) (setf (car x) 9) x)
                          (define l (list 1 2)) (zz-mut l) l)`, "(1 2)")
}

func TestHeuteSetfVerschachteltFehler(t *testing.T) {
  // Phase 2: (1 9 3)
  evalStdlibErr(t, `(let ((l (list 1 2 3))) (setf (car (cdr l)) 9) l)`)
  // Phase 2: ((1 2 3) (4 5 99))
  evalStdlibErr(t, `(let ((m (list (list 1 2 3) (list 4 5 6))))
                      (setf (nth 2 (nth 1 m)) 99) m)`)
  // Phase 2: (1 9 3) — c[ad]+r-Places
  evalStdlibErr(t, `(let ((l (list 1 2 3))) (setf (cadr l) 9) l)`)
  evalStdlibErr(t, `(let ((l (list 1 2 3))) (setf (second l) 9) l)`)
}

func TestHeuteRplacaNconcFehlen(t *testing.T) {
  // Phase 2: (9 2), (1 . 9), (1 2 3 4); (rplaca '() 1) bleibt Fehler
  evalStdlibErr(t, `(rplaca (list 1 2) 9)`)
  evalStdlibErr(t, `(rplacd (list 1 2) 9)`)
  evalStdlibErr(t, `(nconc (list 1 2) (list 3 4))`)
  evalStdlibErr(t, `(nthcdr 1 (list 1 2))`)
}

func TestHeuteStructSetfKopiert(t *testing.T) {
  // defstruct ist eine Liste (tag v1 v2 …), setf über set-nth + Rebind.
  // Phase 2 (Entscheidung offen): 9 — CL-Structs haben Identität.
  evalStdlibEq(t, `(begin (defstruct zzpt x y) (define p (make-zzpt :x 1 :y 2))
                          (define q p) (setf (zzpt-x p) 9) (list (zzpt-x p) (zzpt-x q)))`, "(9 1)")
}

func TestHeutePushAufPlatzKopiert(t *testing.T) {
  // push auf (cdr l) geht über setf → Rebind. Phase 2: ((1 0 2) (1 0 2))
  evalStdlibEq(t, `(begin (define l (list 1 2)) (define m l)
                          (push 0 (cdr l)) (list l m))`, "((1 0 2) (1 2))")
}

// --- Heute: Code ist erreichbar, aber (noch) unveränderbar ---

func TestHeuteQuoteLiteralGeteilt(t *testing.T) {
  // quote liefert die Reader-Zelle des Funktionsrumpfs, jeder Aufruf dieselbe.
  // Phase 2: (setf (car (zz-lit)) 9) ändert den Rumpf still → (zz-lit) = (9 2 3)
  // (CL: undefiniert). Heute bleibt der Rumpf heil.
  evalStdlibEq(t, `(begin (defun zz-lit () '(1 2 3)) (eq (zz-lit) (zz-lit)))`, "t")
  evalStdlibEq(t, `(begin (defun zz-lit () '(1 2 3)) (define a (zz-lit))
                          (setf (car a) 9) (zz-lit))`, "(1 2 3)")
}

func TestHeuteCarAufLambdaLiefertCode(t *testing.T) {
  // car/cdr akzeptieren LAMBDA/MACRO: Parameterliste bzw. Rumpf.
  // Phase 2: rplaca/rplacd darauf (und auf (car f)) würden Code ändern —
  // Entscheidung offen.
  evalStdlibEq(t, `(car (lambda (a b) (+ a b)))`, "(a b)")
  evalStdlibEq(t, `(cdr (lambda (a b) (+ a b)))`, "(+ a b)")
}

func TestHeuteSandboxTeiltFormMitHost(t *testing.T) {
  // Eine Form als Cell (nicht String) geht ungeteilt in die Sandbox und
  // das Ergebnis ungeteilt zurück. Phase 2: per rplaca könnte die Sandbox
  // Host-Daten ändern — Entscheidung offen (Empfehlung: copy-tree an der Grenze).
  evalStdlibEq(t, `(begin (define sb (make-sandbox)) (define l (list 1 2))
                          (eq (sandbox-eval sb (list 'quote l)) l))`, "t")
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
