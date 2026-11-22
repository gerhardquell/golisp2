//**********************************************************************
//  lib/dynvar_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261117
//**********************************************************************
// Special-Menge und %declare-special (Spec 2026-11-17 §3.1).
// Die Menge ist prozessweit — jeder Test nimmt eigene Namen.
//**********************************************************************
package lib

import (
  "fmt"
  "net/http"
  "os"
  "path/filepath"
  "strconv"
  "strings"
  "testing"
  "time"
)

func TestSpecialNachDefvar(t *testing.T) {
  evalStdlibEq(t, `(defvar *dv-a* 1)`, "*dv-a*")
  if !isSpecial(MakeAtom("*dv-a*")) {
    t.Error("*dv-a* nach defvar nicht special")
  }
}

func TestSpecialNachDefvarAufGebundenenNamen(t *testing.T) {
  evalStdlibEq(t, `(define *dv-b* 7) (defvar *dv-b* 99)`, "*dv-b*")
  if !isSpecial(MakeAtom("*dv-b*")) {
    t.Error("*dv-b* nach defvar auf gebundenen Namen nicht special")
  }
}

func TestSpecialNachDefparameter(t *testing.T) {
  evalStdlibEq(t, `(defparameter *dv-c* 1)`, "*dv-c*")
  if !isSpecial(MakeAtom("*dv-c*")) {
    t.Error("*dv-c* nach defparameter nicht special")
  }
}

func TestNichtSpecialNachDefine(t *testing.T) {
  evalStdlibEq(t, `(define *dv-d* 1) *dv-d*`, "1")
  if isSpecial(MakeAtom("*dv-d*")) {
    t.Error("*dv-d* nach define special")
  }
}

// vergissSpecial nimmt Namen nach dem Test aus der prozessweiten Menge,
// für Tests, die den Zustand vor dem ersten defvar brauchen (-count=N).
func vergissSpecial(t *testing.T, names ...string) {
  t.Cleanup(func() {
    specialMu.Lock()
    for _, n := range names {
      delete(specialSyms, MakeAtom(n))
    }
    specialMu.Unlock()
  })
}

// Spec §3.1 (Review 17.11.): Name ohne *…* ist ein Fehler, ohne Teil-Effekt.
func TestDynDefvarOhneSternchenFehler(t *testing.T) {
  vergissSpecial(t, "dv-n", "**", "*dv-o", "dv-p*")
  for _, c := range []struct{ code, name, who string }{
    {`(defvar dv-n 1)`, "dv-n", "defvar"},
    {`(defparameter dv-n 3)`, "dv-n", "defparameter"},
    {`(defvar ** 1)`, "**", "defvar"}, // zu kurz für *…*
    {`(defvar *dv-o 2)`, "*dv-o", "defvar"},
    {`(defvar dv-p* 3)`, "dv-p*", "defvar"},
  } {
    dynFehler(t, c.code, c.who+": "+c.name+" ohne *…* — special-Namen brauchen *"+c.name+"*")
    evalStdlibEq(t, `(trap `+c.code+` (lambda (e) ())) (bound? '`+c.name+`)`, "()")
    if isSpecial(MakeAtom(c.name)) {
      t.Errorf("%s: %s nach Fehler special", c.code, c.name)
    }
  }
  out := stderrVon(t, `(defvar *dv-n* 1) (defparameter *dv-m* 2)`)
  if strings.Contains(out, "WARN") {
    t.Errorf("unerwartete Warnung: %q", out)
  }
}

// Der Fehler lässt nichts zurück: stdlib-Lambdas mit (x) laufen weiter.
func TestDynDefvarOhneSternchenStdlibHeil(t *testing.T) {
  vergissSpecial(t, "x")
  evalStdlibEq(t, `(trap (defvar x 1) (lambda (e) ()))
    (list (remove 2 (list 1 2 3)) (handler-case (error "x") (error (c) 'ok))
      (ignore-errors (error "y")) (defstruct dvf-pt a b) (typecase 1 (number 'n)) (bound? 'x))`,
    "((1 3) ok () dvf-pt? n ())")
}

func TestDynDefvarRueckgabe(t *testing.T) {
  evalStdlibEq(t, `(list (defvar *dvf-r1* 1) (defvar *dvf-r1* 2) (defparameter *dvf-r2* 3) (defparameter *dvf-r2* 4) *dvf-r1* *dvf-r2*)`,
    "(*dvf-r1* *dvf-r1* *dvf-r2* *dvf-r2* 1 4)")
}

// Spec §3.1: schlägt das globale Binden fehl, wird der Name nicht special.
func TestDynDefparameterGlobalFehlerNichtSpecial(t *testing.T) {
  vergissSpecial(t, "*dvf-fn*")
  withRedefinePolicy(t, GetRedefinePolicy(), func() { // stellt die Policy wieder her
    _, err := evalStdlib(t, `(define *dvf-fn* car) (redefine-policy 'error) (defparameter *dvf-fn* 1)`)
    if err == nil {
      t.Fatal("defparameter auf FUNC-Bindung unter 'error: Fehler erwartet")
    }
  })
  if isSpecial(MakeAtom("*dvf-fn*")) {
    t.Error("*dvf-fn* nach gescheitertem globalen Binden special")
  }
}

func TestDeclareSpecialRueckgabeUndFehler(t *testing.T) {
  evalStdlibEq(t, `(%declare-special '*dv-e*)`, "*dv-e*")
  _, err := evalStdlib(t, `(%declare-special 5)`)
  if err == nil || !strings.Contains(err.Error(), "%declare-special: Symbol erwartet, got 5") {
    t.Errorf("Fehler erwartet, got %v", err)
  }
}

// ── Schritt 2: dynamische Bindung (Spec §3.2, §3.4, §3.6) ──

func TestDynLetAufrufSiehtBindung(t *testing.T) {
  v := `(defvar *dv2-a* 1) (defun dv2-ga () *dv2-a*) `
  evalStdlibEq(t, v+`(let ((*dv2-a* 2)) (dv2-ga))`, "2")
  evalStdlibEq(t, v+`(let ((*dv2-a* 2)) (dv2-ga)) (dv2-ga)`, "1")
  evalStdlibEq(t, v+`(let ((*dv2-a* 2)) (let ((*dv2-a* 3)) (dv2-ga)))`, "3")
  evalStdlibEq(t, v+`(let ((*dv2-a* 2)) (let ((*dv2-a* 3)) 0) (dv2-ga))`, "2")
}

func TestDynLetSternSiehtVorigeBindung(t *testing.T) {
  evalStdlibEq(t, `(defvar *dv2-b* 1) (defun dv2-gb () *dv2-b*)
    (let* ((*dv2-b* 2) (y (dv2-gb))) y)`, "2")
}

func TestDynLetMischen(t *testing.T) {
  v := `(defvar *dv2-c* 1) (defun dv2-gc () *dv2-c*) (defun dv2-gy () (bound? 'dv2-y)) `
  evalStdlibEq(t, v+`(let ((*dv2-c* 2) (dv2-y 3)) (list (dv2-gc) dv2-y (dv2-gy)))`, "(2 3 ())")
}

func TestDynClosureNachLetSiehtAltenWert(t *testing.T) {
  v := `(defvar *dv2-d* 1) (defun dv2-mach () (let ((*dv2-d* 2)) (lambda () *dv2-d*))) `
  evalStdlibEq(t, v+`(funcall (dv2-mach))`, "1")
  evalStdlibEq(t, v+`(let ((f (dv2-mach))) (let ((*dv2-d* 5)) (funcall f)))`, "5")
}

func TestDynProgv(t *testing.T) {
  evalStdlibEq(t, `(defvar *dv2-e* 1) (defun dv2-ge () *dv2-e*) (progv '(*dv2-e*) '(3) (dv2-ge))`, "3")
  evalStdlibEq(t, `(progv '(dv2-x) '(1) (let ((dv2-x 2)) dv2-x))`, "2")
  evalStdlibEq(t, `(defun dv2-gf () dv2-f) (progv '(dv2-f) '(7) (dv2-gf))`, "7")
  evalStdlibEq(t, `(defun dv2-gbq () (bound? 'dv2-bq)) (progv '(dv2-bq) '(1) (dv2-gbq))`, "t")
  if isSpecial(MakeAtom("dv2-f")) {
    t.Error("progv darf dv2-f nicht special machen")
  }
}

func TestDynWiederherstellung(t *testing.T) {
  v := `(defvar *dv2-r* 1) (defun dv2-gr () *dv2-r*) `
  evalStdlibEq(t, v+`(trap (let ((*dv2-r* 2)) (setq *dv2-r* 9) (error "x")) (lambda (e) ())) (dv2-gr)`, "1")
  evalStdlibEq(t, v+`(handler-case (let ((*dv2-r* 2)) (error "x")) (error (e) ())) (dv2-gr)`, "1")
  evalStdlibEq(t, v+`(catch 'dv2 (let ((*dv2-r* 2)) (throw 'dv2 0))) (dv2-gr)`, "1")
  evalStdlibEq(t, v+`(block dv2b (let ((*dv2-r* 2)) (return-from dv2b 0))) (dv2-gr)`, "1")
  evalStdlibEq(t, v+`(let ((*dv2-r* 5)) (catch 'dv2 (let ((*dv2-r* 6)) (throw 'dv2 0))) (dv2-gr))`, "5")
}

func TestDynSetq(t *testing.T) {
  v := `(defvar *dv2-s* 1) (defun dv2-gs () *dv2-s*) (defun dv2-set () (setq *dv2-s* 6)) `
  evalStdlibEq(t, v+`(let ((*dv2-s* 2)) (setq *dv2-s* 5) (dv2-gs))`, "5")
  evalStdlibEq(t, v+`(let ((*dv2-s* 2)) (setq *dv2-s* 5)) *dv2-s*`, "1")
  evalStdlibEq(t, v+`(let ((*dv2-s* 2)) (dv2-set) *dv2-s*)`, "6")
  evalStdlibEq(t, v+`(let ((*dv2-s* 2)) (dv2-set)) *dv2-s*`, "1")
  evalStdlibEq(t, v+`(let ((*dv2-s* 2)) (set! *dv2-s* 7) (psetq *dv2-s* (+ *dv2-s* 1)) (setf *dv2-s* (* *dv2-s* 2)) (dv2-gs))`, "16")
  evalStdlibEq(t, v+`(let ((*dv2-s* 2)) (set! *dv2-s* 7) (psetq *dv2-s* 8) (setf *dv2-s* 9)) *dv2-s*`, "1")
  evalStdlibEq(t, v+`(setq *dv2-s* 8) (dv2-gs)`, "8")
  // progv-gebundenes nicht-special Symbol: setq trifft die Bindung
  evalStdlibEq(t, `(define dv2-p 1) (defun dv2-setp () (setq dv2-p 2)) (progv '(dv2-p) '(0) (dv2-setp)) dv2-p`, "1")
}

func TestDynEvalUndLoad(t *testing.T) {
  evalStdlibEq(t, `(defvar *dv2-v* 1) (let ((*dv2-v* 2)) (eval '*dv2-v*))`, "2")
  dir, err := filepath.Abs("../../tmp")
  if err != nil {
    t.Fatal(err)
  }
  if err := os.MkdirAll(dir, 0o755); err != nil {
    t.Fatal(err)
  }
  file := filepath.Join(dir, "dv2-load.lisp")
  if err := os.WriteFile(file, []byte("(list *dv2-l* (dv2-gl))\n"), 0o644); err != nil {
    t.Fatal(err)
  }
  evalStdlibEq(t, fmt.Sprintf(`(defvar *dv2-l* 1) (defun dv2-gl () *dv2-l*) (let ((*dv2-l* 2)) (load %q))`, file), "(2 2)")
}

func TestDynDefvarKonstante(t *testing.T) {
  _, err := evalStdlib(t, `(defvar t 1)`)
  if err == nil || !strings.Contains(err.Error(), "defvar: t ist eine Konstante") {
    t.Errorf("defvar t: Fehler erwartet, got %v", err)
  }
  _, err = evalStdlib(t, `(defvar :dv2-k 1)`)
  if err == nil || !strings.Contains(err.Error(), "defvar: Keyword :dv2-k ist eine Konstante") {
    t.Errorf("defvar :k: Fehler erwartet, got %v", err)
  }
}

func TestDynTCO(t *testing.T) {
  start := time.Now()
  evalStdlibEq(t, `(defvar *dv2-z* 0) (defvar *dv2-w* 0)
    (defun dv2-loop (n) (let ((*dv2-z* n)) (if (= n 0) (list *dv2-z* *dv2-w*) (dv2-loop (- n 1)))))
    (let ((*dv2-w* 42)) (dv2-loop 1000000))`, "(0 42)")
  t.Logf("dv2-loop 1000000: %v", time.Since(start))
}

func TestDynKetteWaechstNicht(t *testing.T) {
  a, b := MakeAtom("*dv2-ka*"), MakeAtom("*dv2-kb*")
  var d *dynBinding
  d = bindDyn(d, a, MakeInt(1))
  d = bindDyn(d, b, MakeInt(2))
  alt := d
  d = bindDyn(d, a, MakeInt(3))
  n := 0
  for p := d; p != nil; p = p.next {
    n++
  }
  if n != 2 {
    t.Errorf("Kettenlänge %d, want 2", n)
  }
  if got := d.find(a).get().String(); got != "3" {
    t.Errorf("a = %s, want 3", got)
  }
  if got := alt.find(a).get().String(); got != "1" {
    t.Errorf("alte Kette verändert: a = %s, want 1", got)
  }
  if d.find(b) != alt.find(b) {
    t.Error("Box von b nicht geteilt")
  }
}

// ── Schritt 3: Binder-Fehler (Spec §3.3, §9.1, §9.2) ──

func dynFehler(t *testing.T, code, want string) {
  t.Helper()
  _, err := evalStdlib(t, code)
  if err == nil || !strings.Contains(err.Error(), want) {
    t.Errorf("%s\n  Fehler %q erwartet, got %v", code, want, err)
  }
}

func TestDynBinderFehlerLambdaListe(t *testing.T) {
  v := `(defvar *dv3-a* 1) `
  fehler := func(wer string) string { return wer + ": Parameter *dv3-a* ist special — mit let binden" }
  for _, c := range []struct{ code, wer string }{
    {`(defun dv3-f1 (*dv3-a*) 1)`, "defun"},
    {`(defun dv3-f2 (x &optional *dv3-a*) 1)`, "defun"},
    {`(lambda (&optional (*dv3-a* 2)) 1)`, "lambda"},
    {`(lambda (&optional (y 2 *dv3-a*)) 1)`, "lambda"},
    {`(lambda (&key *dv3-a*) 1)`, "lambda"},
    {`(lambda (&key (*dv3-a* 2)) 1)`, "lambda"},
    {`(lambda (&key (y 2 *dv3-a*)) 1)`, "lambda"},
    {`(lambda (&rest *dv3-a*) 1)`, "lambda"},
    {`(lambda (x . *dv3-a*) 1)`, "lambda"},
    {`(defmacro dv3-m1 (&body *dv3-a*) 1)`, "defmacro"},
    {`(defmacro dv3-m2 (x *dv3-a*) 1)`, "defmacro"},
    {`(define (dv3-f3 *dv3-a*) 1)`, "define"},
    {`(flet ((dv3-l (*dv3-a*) 1)) 0)`, "flet"},
    {`(labels ((dv3-l (*dv3-a*) 1)) 0)`, "labels"},
    {`(macrolet ((dv3-l (*dv3-a*) 1)) 0)`, "macrolet"},
    {`(defgeneric dv3-gen (x)) (defmethod dv3-gen ((*dv3-a* t)) 1)`, "lambda"},
  } {
    dynFehler(t, v+c.code, fehler(c.wer))
  }
  // Fehler bei der Definition: der Name bleibt ungebunden
  evalStdlibEq(t, v+`(trap (defun dv3-f4 (*dv3-a*) 1) (lambda (e) ())) (bound? 'dv3-f4)`, "()")
}

func TestDynBinderFehlerDoUndMvb(t *testing.T) {
  v := `(defvar *dv3-b* 1) `
  dynFehler(t, v+`(do ((*dv3-b* 0 (+ *dv3-b* 1))) ((= *dv3-b* 3) 0))`, "do: Variable *dv3-b* ist special — mit let binden")
  dynFehler(t, v+`(do* ((i 0 (+ i 1)) (*dv3-b* 0)) ((= i 3) 0))`, "do*: Variable *dv3-b* ist special — mit let binden")
  dynFehler(t, v+`(multiple-value-bind (x *dv3-b*) (values 1 2) x)`, "multiple-value-bind: Variable *dv3-b* ist special — mit let binden")
}

func TestDynBinderNormalerParameter(t *testing.T) {
  evalStdlibEq(t, `(defvar *dv3-n* 10) (defun dv3-add (x) (+ x *dv3-n*))
    (list (dv3-add 1) (let ((*dv3-n* 5)) (dv3-add 1)) (funcall (lambda (y) (* y *dv3-n*)) 2))`, "(11 6 20)")
  evalStdlibEq(t, `(defvar *dv3-n2* 3) (do ((i 0 (+ i 1)) (s 0 (+ s *dv3-n2*))) ((= i 2) s))`, "6")
}

// Spec §3.3/§9.2: vor dem defvar definiert → bindet weiter lexikalisch.
func TestDynBinderGrenzeVorDefvar(t *testing.T) {
  vergissSpecial(t, "*dv3-c*")
  evalStdlibEq(t, `(defun dv3-lies-c () *dv3-c*)
    (defun dv3-alt (*dv3-c*) (list *dv3-c* (dv3-lies-c)))
    (defvar *dv3-c* 1)
    (dv3-alt 2)`, "(2 1)")
}

// Spec §9.1: Makros, die zu let expandieren, binden dynamisch.
func TestDynMakrosBindenDynamisch(t *testing.T) {
  evalStdlibEq(t, `(defvar *dv3-e* 0) (defun dv3-g () *dv3-e*)
    (let ((acc ())) (dolist (*dv3-e* '(1 2)) (setq acc (cons (dv3-g) acc))) acc)`, "(2 1)")
  evalStdlibEq(t, `(defvar *dv3-t* 99) (defun dv3-gt () *dv3-t*)
    (let ((acc ())) (dotimes (*dv3-t* 3) (setq acc (cons (dv3-gt) acc))) (cons *dv3-t* acc))`, "(99 2 1 0)")
  evalStdlibEq(t, `(defvar *dv3-l* 0) (defun dv3-gl () *dv3-l*)
    (list (loop for *dv3-l* in '(1 2) collect (dv3-gl)) (dv3-gl))`, "((1 2) 0)")
  evalStdlibEq(t, `(defvar *dv3-w* 0) (defun dv3-gw () *dv3-w*)
    (list (loop for *dv3-w* from 1 to 2 collect (dv3-gw)) (dv3-gw))`, "((1 2) 0)")
}

// ── Schritt 4: Primitiven mit Rückruf (Spec §3.2, §4) ──

const dv4 = `(defvar *dv4-x* 1) (defun dv4-g (&rest ignore) *dv4-x*) `

func TestDynRueckrufFuncallApply(t *testing.T) {
  evalStdlibEq(t, dv4+`(let ((*dv4-x* 2)) (list (funcall #'dv4-g) (funcall 'dv4-g) (apply #'dv4-g '(1 2)) (apply 'dv4-g 1 '(2))))`, "(2 2 2 2)")
  // applyWithCtx-Pfad: FUNC-Cell (funcall) als Ziel eines Go-Aufrufs
  evalStdlibEq(t, dv4+`(let ((*dv4-x* 2)) (multiple-value-call #'funcall (values 'dv4-g)))`, "2")
}

func TestDynRueckrufMapcar(t *testing.T) {
  evalStdlibEq(t, dv4+`(let ((*dv4-x* 2)) (list (mapcar (lambda (e) (dv4-g)) '(1)) (mapcar 'dv4-g '(1))))`, "((2) (2))")
  evalStdlibEq(t, dv4+`(let ((*dv4-x* 2)) (mapcar #'funcall (list #'dv4-g)))`, "(2)")
}

func TestDynRueckrufSort(t *testing.T) {
  evalStdlibEq(t, dv4+`(let ((*dv4-x* 2)) (sort '(1 2 3) (lambda (a b) (if (= *dv4-x* 2) (> a b) (< a b)))))`, "(3 2 1)")
  evalStdlibEq(t, dv4+`(let ((*dv4-x* 2)) (sort '(1 2 3) #'< :key (lambda (e) (if (= *dv4-x* 2) (- e) e))))`, "(3 2 1)")
}

func TestDynRueckrufReduce(t *testing.T) {
  evalStdlibEq(t, dv4+`(let ((*dv4-x* 2)) (reduce (lambda (a b) (+ a b *dv4-x*)) '(1 2 3)))`, "10")
  evalStdlibEq(t, dv4+`(let ((*dv4-x* 2)) (reduce #'+ '(1 2 3) :initial-value 0 :key (lambda (e) (* e *dv4-x*))))`, "12")
}

func TestDynRueckrufMaphash(t *testing.T) {
  evalStdlibEq(t, dv4+`(let ((h (make-hash-table)) (r nil)) (setf (gethash 'a h) 1)
    (let ((*dv4-x* 2)) (maphash (lambda (k v) (setq r (dv4-g))) h)) r)`, "2")
}

func TestDynRueckrufFormat(t *testing.T) {
  evalStdlibEq(t, dv4+`(let ((*dv4-x* 2)) (format nil "~/dv4-g/ ~{~/dv4-g/~}" 0 '(0 0)))`, `"2 22"`)
}

func TestDynRueckrufSequenz(t *testing.T) {
  evalStdlibEq(t, dv4+`(let ((*dv4-x* 2)) (list (find-if (lambda (e) (= e *dv4-x*)) '(1 2 3))
    (find 2 '(1 2 3) :key (lambda (e) (* e *dv4-x*)))
    (position-if (lambda (e) (= e *dv4-x*)) '(1 2 3))
    (map 'list #'dv4-g '(0)) (some (lambda (e) (= e *dv4-x*)) '(2))
    (remove-if (lambda (e) (= e *dv4-x*)) '(1 2 3))))`, "(2 1 1 (2) t (1 3))")
}

func TestDynRueckrufTrace(t *testing.T) {
  evalStdlibEq(t, dv4+`(defun dv4-t () *dv4-x*) (trace 'dv4-t)
    (let ((r (let ((*dv4-x* 2)) (list (dv4-t) (funcall 'dv4-t))))) (untrace 'dv4-t) r)`, "(2 2)")
}

func TestDynRueckrufOhneBindungUnveraendert(t *testing.T) {
  evalStdlibEq(t, dv4+`(list (mapcar 'dv4-g '(1)) (funcall #'dv4-g) (sort '(2 1) #'<) (format nil "~/dv4-g/" 0))`, `((1) 1 (1 2) "1")`)
}

// ── Schritt 5: Goroutinen erben die Bindungen (Spec §3.5) ──

const dv5 = `(defvar *dv5-x* 1) (defun dv5-g () *dv5-x*) `

func TestDynGoroutineParfunc(t *testing.T) {
  evalStdlibEq(t, dv5+`(let ((*dv5-x* 5)) (parfunc r (dv5-g) (dv5-g)) r)`, "(5 5)")
}

func TestDynGoroutinePoolMap(t *testing.T) {
  evalStdlibEq(t, dv5+`(let ((*dv5-x* 5)) (pool-map 2 (lambda (e) (+ e (dv5-g))) '(1 2 3)))`, "(6 7 8)")
}

func TestDynGoroutineSpawn(t *testing.T) {
  evalStdlibEq(t, dv5+`(let ((*dv5-x* 5)) (chan-recv (spawn #'dv5-g)))`, "(:ok 5)")
  evalStdlibEq(t, dv5+`(chan-recv (spawn (lambda () (dv5-g))))`, "(:ok 1)")
}

func TestDynGoroutineGaCalc(t *testing.T) {
  evalStdlibEq(t, dv5+`(define ga (ga-create 'bit1 5 4 (lambda (g) (dv5-g)))) (ga-init ga)
    (let ((*dv5-x* 7)) (ga-calc ga)) (ga-result ga)`, "(7.0 7.0 7.0 7.0)")
  // nach dem let: wieder der globale Wert
  evalStdlibEq(t, dv5+`(define ga (ga-create 'bit1 5 4 (lambda (g) (dv5-g)))) (ga-init ga)
    (let ((*dv5-x* 7)) (ga-calc ga)) (ga-calc ga) (ga-result ga)`, "(1.0 1.0 1.0 1.0)")
}

func TestDynGoroutineSetqTeiltBindung(t *testing.T) {
  evalStdlibEq(t, dv5+`(list (let ((*dv5-x* 5)) (parfunc r (setq *dv5-x* 9)) *dv5-x*) *dv5-x*)`, "(9 1)")
  evalStdlibEq(t, dv5+`(list (let ((*dv5-x* 5)) (chan-recv (spawn (lambda () (setq *dv5-x* 9)))) *dv5-x*) *dv5-x*)`, "(9 1)")
}

func TestDynGoroutineLetBleibtImWorker(t *testing.T) {
  evalStdlibEq(t, dv5+`(let ((*dv5-x* 5)) (parfunc r (let ((*dv5-x* 8)) (dv5-g))) (list r *dv5-x*))`, "((8) 5)")
}

// Für -race: viele Worker schreiben und lesen dieselbe geerbte Box.
func TestDynGoroutineRaceSetq(t *testing.T) {
  var b strings.Builder
  for i := 1; i <= 32; i++ {
    fmt.Fprintf(&b, " (begin (setq *dv5-x* %d) (dv5-g))", i)
  }
  evalStdlibEq(t, dv5+`(let ((*dv5-x* 0)) (parfunc r`+b.String()+`) (list (length r) (> *dv5-x* 0))) *dv5-x*`, "1")
  evalStdlibEq(t, dv5+`(let ((*dv5-x* 0)) (pool-map 8 (lambda (e) (setq *dv5-x* e) (dv5-g)) '(1 2 3 4 5 6 7 8 9 10 11 12)) (> *dv5-x* 0))`, "t")
}

// HTTP-Handler beginnen ohne Bindungen (Spec §3.5): der Handler läuft in
// einer Server-Goroutine, während der Starter noch im let steckt.
func TestDynGoroutineHTTPHandlerOhneBindung(t *testing.T) {
  ws, env := wsTestServer(t)
  srv := MakeForeign("webserver", ws)
  evalCode(t, env, `(%declare-special '*dv5-h*) (define *dv5-h* 1) (define dv5-got nil)`)
  handler := evalCode(t, env, `(lambda (f c) (setq dv5-got *dv5-h*))`)
  if _, err := fnHTTPUpload([]*Cell{srv, MakeStr("/up"), handler}); err != nil {
    t.Fatalf("http-upload: %v", err)
  }
  url := "http://127.0.0.1:" + strconv.Itoa(ws.port) + "/up"
  _ = env.Set("dv5-post", makeFn(func([]*Cell) (*Cell, error) {
    body, ct := multipartBody(t, "file", "a.txt", "x")
    resp, err := http.Post(url, ct, body)
    if err != nil {
      return nil, err
    }
    resp.Body.Close()
    return MakeInt(int64(resp.StatusCode)), nil
  }))
  got := evalCode(t, env, `(let ((*dv5-h* 7)) (list (dv5-post) *dv5-h* dv5-got))`)
  if got.String() != "(200 7 1)" {
    t.Fatalf("got %s, want (200 7 1)", got)
  }
}

// ── Review 17.11. ──

// Spec §3.2: special-Name, der an der Stelle schon lexikalisch gebunden
// ist (Parameter von vor dem defvar), bindet let/let* lexikalisch weiter.
func TestDynLetAufLexikalischGebundenenNamen(t *testing.T) {
  vergissSpecial(t, "*dvf-p*", "*dvf-z*", "*dvf-q*")
  evalStdlibEq(t, `(defun dvf-dbl (*dvf-p*) (let ((*dvf-p* (* *dvf-p* 2))) *dvf-p*))
    (defvar *dvf-p* 0) (dvf-dbl 5)`, "10")
  evalStdlibEq(t, `(defun dvf-lies () *dvf-z*)
    (defun dvf-alt (*dvf-z*) (let ((*dvf-z* 3)) (list *dvf-z* (dvf-lies))))
    (defvar *dvf-z* 1) (dvf-alt 2)`, "(3 1)")
  evalStdlibEq(t, `(defun dvf-lies-q () *dvf-q*)
    (defun dvf-alt-q (*dvf-q*) (let* ((*dvf-q* (+ *dvf-q* 1)) (y *dvf-q*)) (list *dvf-q* y (dvf-lies-q))))
    (defvar *dvf-q* 1) (dvf-alt-q 2)`, "(3 3 1)")
}

// Spec §3.1: defparameter in dynamischer Bindung setzt die innerste Bindung.
func TestDynDefparameterInBindung(t *testing.T) {
  evalStdlibEq(t, `(defvar *dvf-x* 1) (defun dvf-g () *dvf-x*)
    (list (let ((*dvf-x* 2)) (defparameter *dvf-x* 9) (dvf-g)) *dvf-x*)`, "(9 1)")
}

// Spec §3.1: makunbound auf dynamisch gebundenen Namen ist ein Fehler.
func TestDynMakunboundInBindung(t *testing.T) {
  dynFehler(t, `(defvar *dvf-m* 1) (let ((*dvf-m* 2)) (makunbound '*dvf-m*))`,
    "makunbound: *dvf-m* ist dynamisch gebunden")
  evalStdlibEq(t, `(defvar *dvf-m* 1) (trap (let ((*dvf-m* 2)) (makunbound '*dvf-m*)) (lambda (e) ())) *dvf-m*`, "1")
  evalStdlibEq(t, `(defvar *dvf-m2* 1) (makunbound '*dvf-m2*) (bound? '*dvf-m2*)`, "()")
}

// Review-Befund 6: Binder prüfen alle Variablen vor jeder Auswertung.
func TestDynBinderPruefungVorAuswertung(t *testing.T) {
  v := `(defvar *dvf-b* 1) (define dvf-se 0) `
  for _, c := range []string{
    `(multiple-value-bind (x *dvf-b*) (setq dvf-se 1) x)`,
    `(do ((i (setq dvf-se 1) (+ i 1)) (*dvf-b* 0)) ((= i 3) 0))`,
    `(do* ((i (setq dvf-se 1) (+ i 1)) (*dvf-b* 0)) ((= i 3) 0))`,
  } {
    evalStdlibEq(t, v+`(trap `+c+` (lambda (e) ())) dvf-se`, "0")
  }
}
