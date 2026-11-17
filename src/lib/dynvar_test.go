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
  "strings"
  "testing"
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
