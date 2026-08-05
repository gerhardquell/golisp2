//**********************************************************************
//  lib/specialform_shadow_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260805
//**********************************************************************
// Kein Lisp-Define darf den Namen einer Spezialform tragen.
//
// evalList prueft Spezialformen VOR Makros. Ein (defmacro let* ...) in
// stdlib.lisp wird von eval also nie erreicht — aber es landet trotzdem im
// Root-Env und ist ueber macroexpand/macroexpand-all sichtbar. Zwei
// Implementierungen derselben Form, die sich still auseinanderentwickeln
// koennen: wer die Lisp-Haelfte reparieren will, reparierst nichts, und
// wer die Go-Haelfte aendert, macht macroexpand-all falsch.
//
// Der Redefine-Guard faengt das nicht: er warnt nur beim Ueberschreiben
// bestehender FUNC-Bindungen (env.go), und Spezialformen sind ueberhaupt
// keine Env-Bindungen, sondern case-Zweige im Trampolin.
//
// Der Test liest die Namen aus eval_core.go, statt eine Liste zu pflegen —
// eine gepflegte Liste veraltet und gibt dann falsche Sicherheit.
// White-Box im eigenen Paket ist hier bewusst (vgl. evalStr in eval_test.go).
//
// Siehe CLAUDE.md „Homoikonizitaet" und TODO.md 4.2.
//**********************************************************************

package lib

import (
  "os"
  "regexp"
  "testing"
)

// caseStringsRe findet alle Strings einer case-Zeile: case "a", "b":
var caseStringsRe = regexp.MustCompile(`(?m)^\s*case\s+("[^"]+"(?:\s*,\s*"[^"]+")*)\s*:`)
var quotedRe = regexp.MustCompile(`"([^"]+)"`)

// specialFormNames liest die Spezialform-Namen aus dem case-Switch in
// eval_core.go. Alle String-cases der Datei liegen in diesem einen Switch
// (die anderen beiden switchen auf .Type) — deshalb ist die Extraktion
// exakt und nicht nur heuristisch.
func specialFormNames(t *testing.T) []string {
  t.Helper()
  src, err := os.ReadFile("eval_core.go")
  if err != nil {
    t.Fatalf("eval_core.go lesen: %v", err)
  }
  var names []string
  for _, m := range caseStringsRe.FindAllStringSubmatch(string(src), -1) {
    for _, q := range quotedRe.FindAllStringSubmatch(m[1], -1) {
      names = append(names, q[1])
    }
  }
  if len(names) < 50 {
    t.Fatalf("nur %d Spezialformen gefunden — Extraktion kaputt?", len(names))
  }
  return names
}

// TestNoLispDefineShadowsSpecialForm: nach dem Laden der stdlib darf keine
// Spezialform eine Env-Bindung haben. Wer eine anlegt, baut ein stilles
// Duplikat.
func TestNoLispDefineShadowsSpecialForm(t *testing.T) {
  names := specialFormNames(t)

  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("stdlib: %v", err)
  }

  for _, name := range names {
    if _, err := env.Get(name); err == nil {
      t.Errorf("Spezialform %q ist zusaetzlich im Env gebunden — "+
        "stilles Duplikat, eval erreicht die Bindung nie "+
        "(sichtbar nur ueber macroexpand)", name)
    }
  }
}

// TestSpecialFormExtractionSanity: die Extraktion muss die bekannten
// Namen wirklich finden, sonst wuerde der Test oben nichts pruefen und
// trotzdem gruen sein.
func TestSpecialFormExtractionSanity(t *testing.T) {
  names := specialFormNames(t)
  set := make(map[string]bool, len(names))
  for _, n := range names {
    set[n] = true
  }
  // Einzel-case, Mehrfach-case und Tail-Formen abdecken.
  for _, want := range []string{"quote", "if", "let", "let*", "cond", "case",
    "begin", "progn", "locally", "defun", "defmacro", "lambda"} {
    if !set[want] {
      t.Errorf("Spezialform %q nicht extrahiert", want)
    }
  }
}
