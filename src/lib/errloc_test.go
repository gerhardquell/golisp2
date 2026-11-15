//**********************************************************************
//  lib/errloc_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-fable-5-2
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261115
//**********************************************************************

package lib

import (
  "errors"
  "os"
  "path/filepath"
  "strings"
  "testing"
)

func errlocEnv(t *testing.T) *Env {
  t.Helper()
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil { t.Fatal(err) }
  return env
}

// Fehler in einer geladenen Datei: Ort der fehlerhaften Form und die
// Aufrufkette, innerste zuerst. Führende Leerzeilen zählen mit.
func TestErrLocFileAndCallChain(t *testing.T) {
  path := filepath.Join(t.TempDir(), "kette.lisp")
  src := "\n\n(defun innen (x)\n  (car x))\n(defun aussen (y)\n  (+ 1 (innen y)))\n(aussen 5)\n"
  if err := os.WriteFile(path, []byte(src), 0o600); err != nil { t.Fatal(err) }

  _, err := loadFile(path, errlocEnv(t), evalCtx{})
  if err == nil { t.Fatal("Fehler erwartet") }

  var le *locError
  if !errors.As(err, &le) { t.Fatalf("kein locError: %T %v", err, err) }
  if le.at == nil || le.at.SrcLine != 4 || le.at.SrcFile() != path {
    t.Fatalf("Ort = %v, want %s:4", le.at, path)
  }
  if le.at.String() != "(car x)" { t.Fatalf("Form = %s, want (car x)", le.at) }

  wantCalls := []struct { form string; line int }{
    {"(innen y)", 6}, {"(aussen 5)", 7},
  }
  if len(le.calls) != len(wantCalls) {
    t.Fatalf("Aufrufkette = %v, want %d Einträge", le.calls, len(wantCalls))
  }
  for i, w := range wantCalls {
    c := le.calls[i]
    if c.String() != w.form || c.SrcLine != w.line || c.SrcFile() != path {
      t.Errorf("Aufruf %d = %s Zeile %d, want %s Zeile %d", i, c, c.SrcLine, w.form, w.line)
    }
  }

  trace := ErrorTrace(err)
  if got := strings.Count(trace, "\n"); got != 3 {
    t.Errorf("ErrorTrace hat %d Zeilen, want 3:\n%s", got, trace)
  }
}

// Der Ort darf die Meldung nicht verändern: trap, handler-case und alle
// Tests vergleichen den Text.
func TestErrLocKeepsMessage(t *testing.T) {
  expr, _ := Read("(car 5)")
  _, err := Eval(expr, errlocEnv(t))
  if err == nil || err.Error() != "car: Liste erwartet" {
    t.Fatalf("Error() = %v, want unveränderte Meldung", err)
  }
}

// trap muss den Fehlerwert weiter als Cell bekommen, auch wenn der Fehler
// unterwegs einen Ort bekommen hat.
func TestErrLocTrapKeepsErrorCell(t *testing.T) {
  res, err := LoadString(`(trap (error '(a b)) (lambda (e) e))`, errlocEnv(t))
  if err != nil { t.Fatal(err) }
  if res.String() != "(a b)" { t.Fatalf("trap lieferte %s, want (a b)", res) }
}

// Kontrollfluss ist kein Fehler und bekommt keinen Ort.
func TestErrLocLeavesControlFlowAlone(t *testing.T) {
  res, err := LoadString(`(list (block b (+ 1 (return-from b 7))) (catch 'k (+ 1 (throw 'k 8))))`, errlocEnv(t))
  if err != nil { t.Fatal(err) }
  if res.String() != "(7 8)" { t.Fatalf("got %s, want (7 8)", res) }
}

// Tiefe Rekursion: die Kette bleibt begrenzt, der Rest wird gezählt.
func TestErrLocCallChainIsCapped(t *testing.T) {
  _, err := LoadString(`(defun tief (n) (if (= n 0) (car 5) (+ 1 (tief (- n 1))))) (tief 500)`, errlocEnv(t))
  var le *locError
  if !errors.As(err, &le) { t.Fatalf("kein locError: %v", err) }
  if len(le.calls) != maxLocCalls || le.more != 501-maxLocCalls {
    t.Fatalf("calls=%d more=%d, want %d und %d", len(le.calls), le.more, maxLocCalls, 501-maxLocCalls)
  }
}

// Tail-Call: der Frame merkt sich neben dem letzten Aufruf auch den
// Einstieg — sonst fehlte in der Kette die Zeile des Aufrufers.
func TestErrLocTailCallKeepsEntry(t *testing.T) {
  _, err := LoadString(`(defun ende (x) (car x)) (defun mitte (x) (ende x)) (defun start (x) (mitte x)) (start 5)`, errlocEnv(t))
  var le *locError
  if !errors.As(err, &le) { t.Fatalf("kein locError: %v", err) }
  var got []string
  for _, c := range le.calls { got = append(got, c.String()) }
  if want := "(ende x) (start 5)"; strings.Join(got, " ") != want {
    t.Fatalf("Aufrufkette = %v, want %s", got, want)
  }
}

// trap reicht den Fehlerort an einen Handler mit genau zwei Parametern
// weiter (Meldung, Ort als Text wie in der CLI). Handler mit einem
// Parameter und eingebaute Funktionen bekommen wie bisher nur die Meldung.
func TestErrLocTrapHandlerGetsTrace(t *testing.T) {
  env := errlocEnv(t)
  for _, c := range []struct{ src, want string }{
    {`(trap (car 5) (lambda (e ort) (list e ort)))`,
      `("car: Liste erwartet" "  in Zeile 1: (car 5)\n")`},
    {`(defun innen (x) (car x)) (trap (+ 1 (innen 5)) (lambda (e ort) ort))`,
      `"  in Zeile 1: (car x)\n  aufgerufen von Zeile 1: (innen 5)\n"`},
    {`(trap (car 5) (lambda (e) e))`, `"car: Liste erwartet"`},
    {`(trap (car 5) (lambda (e &optional x) (list e x)))`, `("car: Liste erwartet" ())`},
    {`(trap (car 5) (lambda args (length args)))`, `1`},
    {`(trap (car 5) list)`, `("car: Liste erwartet")`},
  } {
    res, err := LoadString(c.src, env)
    if err != nil { t.Errorf("%s: %v", c.src, err); continue }
    if res.String() != c.want { t.Errorf("%s\n   = %s\nwant %s", c.src, res, c.want) }
  }
}
