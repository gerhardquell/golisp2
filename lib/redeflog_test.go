//**********************************************************************
//  lib/redeflog_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260724
//**********************************************************************

package lib

import (
  "fmt"
  "io"
  "os"
  "strings"
  "testing"
)

func TestRedefLogAppendAndOrder(t *testing.T) {
  ClearRedefLog()
  logRedef(RedefEvent{Name: "a", OldKind: "lambda", Action: "reload"})
  logRedef(RedefEvent{Name: "b", OldKind: "func", Action: "warn"})
  events := RedefLog()
  if len(events) != 2 {
    t.Fatalf("2 Events erwartet, got %d", len(events))
  }
  if events[0].Name != "a" || events[1].Name != "b" {
    t.Fatalf("Reihenfolge älteste→neueste verletzt: %+v", events)
  }
}

func TestRedefLogRingOverflow(t *testing.T) {
  ClearRedefLog()
  for i := 0; i < redefLogSize+10; i++ {
    logRedef(RedefEvent{Name: fmt.Sprintf("n%d", i)})
  }
  events := RedefLog()
  if len(events) != redefLogSize {
    t.Fatalf("Ring muss bei %d kappen, got %d", redefLogSize, len(events))
  }
  if events[0].Name != "n10" {
    t.Fatalf("ältestes Event muss n10 sein, got %q", events[0].Name)
  }
  want := fmt.Sprintf("n%d", redefLogSize+9)
  if events[len(events)-1].Name != want {
    t.Fatalf("neuestes Event muss %q sein, got %q", want, events[len(events)-1].Name)
  }
}

func TestRedefLogReturnsCopy(t *testing.T) {
  ClearRedefLog()
  logRedef(RedefEvent{Name: "x"})
  events := RedefLog()
  events[0].Name = "mutiert"
  if RedefLog()[0].Name != "x" {
    t.Fatal("RedefLog muss eine Kopie liefern")
  }
}

func TestKindOf(t *testing.T) {
  cases := []struct {
    cell *Cell
    want string
  }{
    {cell: &Cell{Type: FUNC}, want: "func"},
    {cell: &Cell{Type: LAMBDA}, want: "lambda"},
    {cell: &Cell{Type: MACRO}, want: "macro"},
    {cell: &Cell{Type: NUMBER}, want: "value"},
    {cell: &Cell{Type: ATOM}, want: "value"},
  }
  for _, c := range cases {
    if got := kindOf(c.cell); got != c.want {
      t.Errorf("kindOf(%v) = %q, want %q", c.cell.Type, got, c.want)
    }
  }
}

func TestFuncRedefLogged(t *testing.T) {
  ClearRedefLog()
  withRedefinePolicy(t, "allow", func() {
    if _, err := evalStr("(define car 42)"); err != nil {
      t.Fatalf("allow-Policy muss still durchlassen: %v", err)
    }
  })
  events := RedefLog()
  if len(events) != 1 {
    t.Fatalf("1 Event erwartet, got %d (%+v)", len(events), events)
  }
  e := events[0]
  if e.Name != "car" || e.OldKind != "func" || e.NewKind != "value" || e.Action != "redef" {
    t.Fatalf("Event unerwartet: %+v", e)
  }
}

func TestFuncRedefErrorLogged(t *testing.T) {
  ClearRedefLog()
  withRedefinePolicy(t, "error", func() {
    if _, err := evalStr("(define car 42)"); err == nil {
      t.Fatal("error-Policy muss blockieren")
    }
  })
  events := RedefLog()
  if len(events) != 1 || events[0].Action != "error" {
    t.Fatalf("1 error-Event erwartet, got %+v", events)
  }
}

// captureStderr faengt os.Stderr fuer die Dauer von fn ab.
func captureStderr(t *testing.T, fn func()) string {
  t.Helper()
  r, w, err := os.Pipe()
  if err != nil {
    t.Fatalf("os.Pipe: %v", err)
  }
  old := os.Stderr
  os.Stderr = w
  defer func() { os.Stderr = old }()
  fn()
  _ = w.Close()
  out, _ := io.ReadAll(r)
  return string(out)
}

// evalInEnv wertet src im uebergebenen Env aus (im Gegensatz zu evalStr,
// das jedes Mal ein frisches BaseEnv erzeugt). Noetig fuer Cross-File-
// Redefinitionstests, bei denen Definition und simulierte Herkunft in
// derselben Env-Lebenszeit passieren muessen.
func evalInEnv(env *Env, src string) (*Cell, error) {
  r := NewReader(strings.TrimSpace(src))
  result := MakeNil()
  for {
    r.skipWS()
    if _, ok := r.peek(); !ok {
      break
    }
    expr, err := r.readExpr()
    if err != nil {
      return nil, err
    }
    result, err = Eval(expr, env)
    if err != nil {
      return nil, err
    }
  }
  return result, nil
}

func TestRedefSameSourceReloadSilent(t *testing.T) {
  ClearRedefLog()
  ClearDefinitions()
  out := captureStderr(t, func() {
    // Zwei defuns aus derselben Quelle (SrcFile "" bei evalStr):
    if _, err := evalStr("(defun foo (x) x) (defun foo (x) (cons x nil))"); err != nil {
      t.Fatalf("Reload derselben Quelle muss erlaubt sein: %v", err)
    }
  })
  if out != "" {
    t.Fatalf("Reload muss still bleiben, stderr: %q", out)
  }
  events := RedefLog()
  if len(events) != 1 || events[0].Action != "reload" || events[0].OldKind != "lambda" {
    t.Fatalf("1 reload-Event erwartet, got %+v", events)
  }
}

func TestRedefCrossFileWarns(t *testing.T) {
  ClearRedefLog()
  ClearDefinitions()
  withRedefinePolicy(t, "warn", func() {
    env := BaseEnv()
    if _, err := evalInEnv(env, "(defun bar (x) x)"); err != nil {
      t.Fatal(err)
    }
    RegisterDefinition("bar", "a.lisp", 12) // simuliert Herkunft aus Datei
    out := captureStderr(t, func() {
      // SrcFile "" ≠ "a.lisp" → fremde Quelle
      if _, err := evalInEnv(env, "(defun bar (x) 42)"); err != nil {
        t.Fatalf("warn-Policy muss durchlassen: %v", err)
      }
    })
    if !strings.Contains(out, "REDEF: bar") {
      t.Fatalf("REDEF-Warnung erwartet, stderr: %q", out)
    }
    if !strings.Contains(out, "a.lisp") {
      t.Fatalf("Warnung muss alte Quelle nennen, stderr: %q", out)
    }
  })
  events := RedefLog()
  if len(events) != 1 || events[0].Action != "warn" || events[0].OldFile != "a.lisp" {
    t.Fatalf("warn-Event mit Quelle erwartet, got %+v", events)
  }
}

func TestRedefCrossFileErrorBlocks(t *testing.T) {
  ClearRedefLog()
  ClearDefinitions()
  withRedefinePolicy(t, "error", func() {
    env := BaseEnv()
    if _, err := evalInEnv(env, "(defun baz (x) x)"); err != nil {
      t.Fatal(err)
    }
    RegisterDefinition("baz", "a.lisp", 1)
    if _, err := evalInEnv(env, "(defun baz (x) 99)"); err == nil ||
      !strings.Contains(err.Error(), "REDEF: baz") {
      t.Fatalf("REDEF-Fehler erwartet, got %v", err)
    }
    got, err := evalInEnv(env, "(baz 5)")
    if err != nil || got.Num != 5 {
      t.Fatalf("alte Bindung muss erhalten bleiben: %v, %v", got, err)
    }
  })
}

func TestRedefValueOverLambdaWarns(t *testing.T) {
  ClearRedefLog()
  ClearDefinitions()
  withRedefinePolicy(t, "warn", func() {
    env := BaseEnv()
    if _, err := evalInEnv(env, "(defun qux (x) x)"); err != nil {
      t.Fatal(err)
    }
    RegisterDefinition("qux", "b.lisp", 3)
    out := captureStderr(t, func() {
      if _, err := evalInEnv(env, "(define qux 5)"); err != nil {
        t.Fatal(err)
      }
    })
    if !strings.Contains(out, "REDEF: qux") {
      t.Fatalf("Wert-über-Funktion muss warnen, stderr: %q", out)
    }
  })
  events := RedefLog()
  if len(events) != 1 || events[0].OldKind != "lambda" || events[0].NewKind != "value" {
    t.Fatalf("Event lambda→value erwartet, got %+v", events)
  }
}

func TestRedefNonRootUntouched(t *testing.T) {
  ClearRedefLog()
  ClearDefinitions()
  // define im Lambda-Body schreibt in den lokalen Frame, nicht an Root:
  if _, err := evalStr("(defun mk () (define lokales-symbol 1)) (mk)"); err != nil {
    t.Fatal(err)
  }
  if events := RedefLog(); len(events) != 0 {
    t.Fatalf("lokale Defines duerfen nicht geloggt werden, got %+v", events)
  }
}

func TestMakunbound(t *testing.T) {
  ClearRedefLog()
  ClearDefinitions()
  // defun + makunbound + bound? in EINEM evalStr: evalStr baut pro Aufruf
  // ein frisches Env, Bindungen ueberleben keinen zweiten Aufruf.
  got, err := evalStr("(defun mkb (x) x) (makunbound 'mkb) (bound? 'mkb)")
  if err != nil {
    t.Fatal(err)
  }
  if IsTruthy(got) {
    t.Fatal("mkb muss nach makunbound ungebunden sein")
  }
  if _, ok := LookupDefinition("mkb"); ok {
    t.Fatal("DefLoc-Eintrag muss entfernt sein")
  }
  events := RedefLog()
  if len(events) != 1 || events[0].Action != "makunbound" || events[0].OldKind != "lambda" {
    t.Fatalf("makunbound-Event erwartet, got %+v", events)
  }
}

func TestMakunboundUnboundError(t *testing.T) {
  _, err := evalStr("(makunbound 'gibts-nicht)")
  if err == nil || !strings.Contains(err.Error(), "nicht gebunden") {
    t.Fatalf("Fehler fuer ungebundenes Symbol erwartet, got %v", err)
  }
}

func TestMakunboundFuncErrorPolicy(t *testing.T) {
  withRedefinePolicy(t, "error", func() {
    if _, err := evalStr("(makunbound 'car)"); err == nil ||
      !strings.Contains(err.Error(), "REDEF: car") {
      t.Fatalf("error-Policy muss makunbound auf FUNC blockieren, got %v", err)
    }
    got, err := evalStr("(car '(1 2))")
    if err != nil || got.Num != 1 {
      t.Fatalf("car muss erhalten bleiben: %v, %v", got, err)
    }
  })
}

func TestMakunboundFuncAllow(t *testing.T) {
  ClearRedefLog()
  withRedefinePolicy(t, "allow", func() {
    // ebenfalls in einem evalStr (frisches Env pro Aufruf, siehe oben)
    got, err := evalStr("(makunbound 'cdr) (bound? 'cdr)")
    if err != nil {
      t.Fatal(err)
    }
    if IsTruthy(got) {
      t.Fatal("cdr muss entfernt sein")
    }
  })
  events := RedefLog()
  if len(events) != 1 || events[0].OldKind != "func" || events[0].Action != "makunbound" {
    t.Fatalf("func-makunbound-Event erwartet, got %+v", events)
  }
}

func TestRedefLogPrimitive(t *testing.T) {
  ClearRedefLog()
  ClearDefinitions()
  withRedefinePolicy(t, "allow", func() {
    if _, err := evalStr("(define car 42)"); err != nil {
      t.Fatal(err)
    }
    // Name des ersten (einzigen) Events:
    got, err := evalStr("(car (car (redef-log)))")
    if err != nil {
      t.Fatal(err)
    }
    if got.Type != ATOM || got.Val != "car" {
      t.Fatalf("Event-Name car erwartet, got %v", got)
    }
    if _, err := evalStr("(redef-log-clear)"); err != nil {
      t.Fatal(err)
    }
    empty, err := evalStr("(redef-log)")
    if err != nil {
      t.Fatal(err)
    }
    if empty.Type != NIL {
      t.Fatalf("leeres Log muss nil sein, got %v", empty)
    }
  })
}
