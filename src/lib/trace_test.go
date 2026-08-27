package lib

import (
  "strings"
  "testing"
)

func captureError(t *testing.T, fn func()) string {
  t.Helper()
  var b strings.Builder
  SetErrorWriter(func(s string) error {
    _, err := b.WriteString(s)
    return err
  })
  defer ResetErrorWriter()
  fn()
  return b.String()
}

func TestTracePrimitive(t *testing.T) {
  env := BaseEnv()
  defer evalAll("(untrace)", env)

  out := captureError(t, func() {
    _, err := evalAll("(trace '+)(+ 1 2 3)", env)
    if err != nil {
      t.Fatalf("eval failed: %v", err)
    }
  })
  if !strings.Contains(out, "(+ 1 2 3)") {
    t.Fatalf("stderr fehlt Eingabe: %q", out)
  }
  if !strings.Contains(out, "=> 6") {
    t.Fatalf("stderr fehlt Ergebnis: %q", out)
  }
}

func TestTraceLambdaNested(t *testing.T) {
  env := BaseEnv()
  defer evalAll("(untrace)", env)

  src := `(begin
    (defun f (n)
      (if (= n 0) 0 (+ 1 (f (- n 1)))))
    (trace 'f)
    (f 2))`
  out := captureError(t, func() {
    got, err := evalAll(src, env)
    if err != nil {
      t.Fatalf("eval failed: %v", err)
    }
    if got.String() != "2" {
      t.Fatalf("f(2) = %s", got.String())
    }
  })
  lines := strings.Split(strings.TrimSpace(out), "\n")
  if len(lines) < 6 {
    t.Fatalf("zu wenig Trace-Zeilen: %q", out)
  }
  // Tiefe 0: kein Indent, Tiefe 1: zwei Spaces, Tiefe 2: vier Spaces.
  if !strings.HasPrefix(lines[0], "(f 2)") {
    t.Fatalf("erste Zeile nicht Tiefe 0: %q", lines[0])
  }
  if !strings.HasPrefix(lines[1], "  (f 1)") {
    t.Fatalf("zweite Zeile nicht Tiefe 1: %q", lines[1])
  }
  if !strings.HasPrefix(lines[2], "    (f 0)") {
    t.Fatalf("dritte Zeile nicht Tiefe 2: %q", lines[2])
  }
}

func TestUntraceRestores(t *testing.T) {
  env := BaseEnv()
  defer evalAll("(untrace)", env)

  _, err := evalAll("(trace '+)(+ 1 2 3)", env)
  if err != nil {
    t.Fatalf("eval failed: %v", err)
  }

  out := captureError(t, func() {
    got, err := evalAll("(untrace '+)(+ 1 2 3)", env)
    if err != nil {
      t.Fatalf("eval failed: %v", err)
    }
    if got.String() != "6" {
      t.Fatalf("(+ 1 2 3) = %s", got.String())
    }
  })
  if strings.Contains(out, "(+ 1 2 3)") {
    t.Fatalf("nach untrace sollte keine Trace-Ausgabe kommen: %q", out)
  }
}

func TestUntraceAll(t *testing.T) {
  env := BaseEnv()
  defer evalAll("(untrace)", env)

  _, err := evalAll("(begin (trace '+) (trace 'list))", env)
  if err != nil {
    t.Fatalf("eval failed: %v", err)
  }
  got, err := evalAll("(untrace)", env)
  if err != nil {
    t.Fatalf("untrace failed: %v", err)
  }
  // Rückgabe sortiert: (+ list)
  if got.String() != "(+ list)" {
    t.Fatalf("(untrace) = %s", got.String())
  }
  p1, _ := evalAll("(trace? '+)", env)
  p2, _ := evalAll("(trace? 'list)", env)
  if p1.String() != "()" || p2.String() != "()" {
    t.Fatalf("trace? nach untrace nicht nil: %s %s", p1.String(), p2.String())
  }
}

func TestTracePredicate(t *testing.T) {
  env := BaseEnv()
  defer evalAll("(untrace)", env)

  before, _ := evalAll("(trace? '+)", env)
  if before.String() != "()" {
    t.Fatalf("trace? '+ vor trace = %s", before.String())
  }
  _, err := evalAll("(trace '+)", env)
  if err != nil {
    t.Fatalf("trace failed: %v", err)
  }
  after, _ := evalAll("(trace? '+)", env)
  if after.String() != "t" {
    t.Fatalf("trace? '+ nach trace = %s", after.String())
  }
}

func TestTraceIdempotent(t *testing.T) {
  env := BaseEnv()
  defer evalAll("(untrace)", env)

  out := captureError(t, func() {
    _, err := evalAll("(trace '+)(trace '+)(+ 1 2 3)", env)
    if err != nil {
      t.Fatalf("eval failed: %v", err)
    }
  })
  if strings.Count(out, "=> 6") != 1 {
    t.Fatalf("doppelte Trace-Ausgabe: %q", out)
  }
}

func TestTraceUnknownSymbol(t *testing.T) {
  env := BaseEnv()
  defer evalAll("(untrace)", env)

  _, err := evalAll("(trace 'unknownSymbolXYZ)", env)
  if err == nil {
    t.Fatal("Erwarteter Fehler für unbekanntes Symbol")
  }
}

func TestTraceNonFunction(t *testing.T) {
  env := BaseEnv()
  defer evalAll("(untrace)", env)

  _, err := evalAll("(begin (define x 42) (trace 'x))", env)
  if err == nil {
    t.Fatal("Erwarteter Fehler für nicht-Funktion")
  }
}
