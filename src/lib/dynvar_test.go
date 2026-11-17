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
  "os"
  "path/filepath"
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

func TestDeclareSpecialWarnungOhneSternchen(t *testing.T) {
  out := stderrVon(t, `(defvar dv-n 1) (defvar dv-n 2) (defparameter dv-n 3)`)
  want := "WARN: defvar dv-n ohne *…* — jede let-Bindung von dv-n ist jetzt dynamisch"
  if n := strings.Count(out, want); n != 1 {
    t.Errorf("Warnung %d× statt 1×, stderr = %q", n, out)
  }
  out = stderrVon(t, `(defvar *dv-n* 1) (defparameter *dv-m* 2)`)
  if strings.Contains(out, "WARN") {
    t.Errorf("unerwartete Warnung: %q", out)
  }
  // ** ist zu kurz für *…*, *dv-o / dv-p* haben nur ein Sternchen
  out = stderrVon(t, `(defvar ** 1) (defvar *dv-o 2) (defvar dv-p* 3)`)
  for _, name := range []string{"**", "*dv-o", "dv-p*"} {
    if !strings.Contains(out, "WARN: defvar "+name+" ohne") {
      t.Errorf("Warnung für %s fehlt: %q", name, out)
    }
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
  evalStdlibEq(t, v+`(let ((f (dv2-mach))) (let ((*dv2-d* 5)) (f)))`, "5")
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
  if err == nil || !strings.Contains(err.Error(), "%declare-special: t ist eine Konstante") {
    t.Errorf("defvar t: Fehler erwartet, got %v", err)
  }
  _, err = evalStdlib(t, `(defvar :dv2-k 1)`)
  if err == nil || !strings.Contains(err.Error(), "%declare-special: Keyword :dv2-k ist eine Konstante") {
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
