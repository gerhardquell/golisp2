package lib

import (
  "io"
  "os"
  "slices"
  "strings"
  "testing"
)

func TestSymbolsContainsBuiltins(t *testing.T) {
  env := BaseEnv()
  syms := env.Symbols()

  // nur Primitiven testen (Spezialformen wie defun/lambda leben im evalList-Switch, nicht in BaseEnv)
  for _, want := range []string{"+", "-", "*", "/", "=", "<", ">", ">=", "<=", "car", "cdr", "cons", "println", "gensym"} {
    if !slices.Contains(syms, want) {
      t.Errorf("Symbols() fehlt: %q", want)
    }
  }
}

func TestSymbolsIncludesUserDefined(t *testing.T) {
  env := BaseEnv()
  if err := env.Set("meine-fn", MakeAtom("t")); err != nil {
    t.Fatalf("Set failed: %v", err)
  }
  syms := env.Symbols()
  if !slices.Contains(syms, "meine-fn") {
    t.Error("Symbols() enthält keine user-definierten Symbole")
  }
}

func withRedefinePolicy(t *testing.T, name string, fn func()) {
  t.Helper()
  old := GetRedefinePolicy()
  if err := SetRedefinePolicy(name); err != nil {
    t.Fatalf("SetRedefinePolicy %q: %v", name, err)
  }
  defer func() {
    if err := SetRedefinePolicy(old); err != nil {
      t.Fatalf("restore policy %q: %v", old, err)
    }
  }()
  fn()
}

func TestRedefineGuardError(t *testing.T) {
  withRedefinePolicy(t, "error", func() {
    _, err := evalStr("(define car 42)")
    if err == nil {
      t.Fatal("Erwarteter Fehler für Redefinition eines Primitivs")
    }
    if !strings.Contains(err.Error(), "REDEF: car") {
      t.Fatalf("Fehler unerwartet: %v", err)
    }
    // Binding muss unverändert sein
    got, err := evalStr("(car '(1 2))")
    if err != nil {
      t.Fatalf("car funktioniert nicht: %v", err)
    }
    if got.String() != "1" {
      t.Fatalf("car wurde geändert: %s", got.String())
    }
  })
}

// evalAll wertet alle Ausdrücke in src in einem bestehenden env aus und
// liefert das letzte Ergebnis. Wie evalStr, aber mit wiederverwendetem Env.
func evalAll(src string, env *Env) (*Cell, error) {
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

func TestRedefineGuardWarn(t *testing.T) {
  withRedefinePolicy(t, "warn", func() {
    env := BaseEnv()
    oldStderr := os.Stderr
    r, w, err := os.Pipe()
    if err != nil { t.Fatal(err) }
    os.Stderr = w

    _, err = evalAll("(define car 42)", env)
    if err != nil { t.Fatalf("Unerwarteter Fehler: %v", err) }
    w.Close()
    os.Stderr = oldStderr
    out, _ := io.ReadAll(r)
    if !strings.Contains(string(out), "REDEF: car") {
      t.Fatalf("stderr = %q", out)
    }
    // Überschreiben ist erlaubt
    got, err := evalAll("car", env)
    if err != nil { t.Fatal(err) }
    if got.String() != "42" {
      t.Fatalf("car = %s", got.String())
    }
  })
}

func TestRedefineGuardAllow(t *testing.T) {
  withRedefinePolicy(t, "allow", func() {
    env := BaseEnv()
    _, err := evalAll("(define car 42)", env)
    if err != nil { t.Fatalf("Unerwarteter Fehler: %v", err) }
    got, err := evalAll("car", env)
    if err != nil { t.Fatal(err) }
    if got.String() != "42" {
      t.Fatalf("car = %s", got.String())
    }
  })
}

func TestRedefineGuardNewSymbol(t *testing.T) {
  withRedefinePolicy(t, "error", func() {
    env := BaseEnv()
    got, err := evalAll("(define neuesSymbol 99)", env)
    if err != nil { t.Fatal(err) }
    if got.String() != "neuesSymbol" {
      t.Fatalf("Rückgabe = %s", got.String())
    }
    got, err = evalAll("neuesSymbol", env)
    if err != nil { t.Fatal(err) }
    if got.String() != "99" {
      t.Fatalf("neuesSymbol = %s", got.String())
    }
  })
}

func TestRedefineGuardLambdaTwice(t *testing.T) {
  withRedefinePolicy(t, "error", func() {
    got, err := evalStr("(begin (defun f (x) x) (defun f (x) (+ x 1)) (f 5))")
    if err != nil { t.Fatal(err) }
    if got.String() != "6" {
      t.Fatalf("f(5) = %s", got.String())
    }
  })
}

func TestRedefineGuardLambdaParamShadowing(t *testing.T) {
  withRedefinePolicy(t, "error", func() {
    got, err := evalStr("((lambda (car) car) 7)")
    if err != nil { t.Fatal(err) }
    if got.String() != "7" {
      t.Fatalf("Lambda-Parameter car = %s", got.String())
    }
  })
}

func TestRedefinePolicyPrimitive(t *testing.T) {
  withRedefinePolicy(t, "warn", func() {
    got, err := evalStr("(redefine-policy)")
    if err != nil { t.Fatal(err) }
    if got.String() != "warn" {
      t.Fatalf("Policy = %s", got.String())
    }
  })
  withRedefinePolicy(t, "allow", func() {
    got, err := evalStr("(redefine-policy 'error)")
    if err != nil { t.Fatal(err) }
    if got.String() != "error" {
      t.Fatalf("Rückgabe = %s", got.String())
    }
    if GetRedefinePolicy() != "error" {
      t.Fatalf("Go-Policy nicht gesetzt")
    }
  })
}

func TestRedefinePolicyInvalid(t *testing.T) {
  _, err := evalStr("(redefine-policy 'unknown)")
  if err == nil {
    t.Fatal("Erwarteter Fehler für unbekannte Policy")
  }
}

