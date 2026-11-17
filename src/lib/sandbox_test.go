//**********************************************************************
//  lib/sandbox_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Opus 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261117
//**********************************************************************

package lib

import (
  "os"
  "path/filepath"
  "slices"
  "strings"
  "testing"
)

// sbEnv: globales Env mit stdlib, darin sb = (make-sandbox).
func sbEnv(t *testing.T, makeForm string) *Env {
  t.Helper()
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("LoadStdlib: %v", err)
  }
  if _, err := sbRun(env, "(define sb "+makeForm+")"); err != nil {
    t.Fatalf("%s: %v", makeForm, err)
  }
  return env
}

func sbRun(env *Env, src string) (*Cell, error) {
  forms, err := ReadAll(src)
  if err != nil {
    return nil, err
  }
  res := MakeNil()
  for f := forms; f != nil && f.Type == LIST; f = f.Cdr {
    if res, err = Eval(f.Car, env); err != nil {
      return nil, err
    }
  }
  return res, nil
}

func sbWant(t *testing.T, env *Env, src, want string) {
  t.Helper()
  got, err := sbRun(env, src)
  if err != nil {
    t.Fatalf("%s: Fehler %v", src, err)
  }
  if got.String() != want {
    t.Errorf("%s = %s, want %s", src, got, want)
  }
}

func sbWantErr(t *testing.T, env *Env, src, want string) {
  t.Helper()
  _, err := sbRun(env, src)
  if err == nil || !strings.Contains(err.Error(), want) {
    t.Errorf("%s: Fehler %v, want …%s…", src, err, want)
  }
}

func sbGesperrt(name string) string {
  return "sandbox: " + name + " nicht erlaubt (freigeben mit :erlaube)"
}

func TestSandboxFluchtwege(t *testing.T) {
  env := sbEnv(t, "(make-sandbox)")
  cases := []struct{ form, name string }{
    {`(exec "ls")`, "exec"},
    {`(load "x.lisp")`, "load"},
    {`(defmain () 1)`, "defmain"},
    {`(system "ls")`, "system"},
    {`(file-write "x.txt" "boom")`, "file-write"},
    {`(getenv "HOME")`, "getenv"},
    {`(exit 0)`, "exit"},
    {`(funcall 'system "ls")`, "system"},
    {`(apply 'system '("ls"))`, "system"},
    {`(mapcar 'system '("ls"))`, "system"},
    {`(eval '(exec "ls"))`, "exec"},
    {`(eval (list 'system "ls"))`, "system"},
    {`(begin (defmacro m () '(exec "ls")) (m))`, "exec"},
    {`(sort '("b" "a") 'system)`, "system"},
    {`(find "a" '("a") :key 'system)`, "system"},
    {`(find-if 'system '("a"))`, "system"},
    {`(map 'list 'system '("a"))`, "system"},
    {`(format nil "~/system/" "ls")`, "system"},
    {`(fprintf "x.txt" "boom")`, "fprintf"},
    {`(sigo-host "evil:9080")`, "sigo-host"},
    {`(make-sandbox)`, "make-sandbox"},
    {`(shell-output "ls")`, "exec"}, // stdlib-Funktion ruft Gesperrtes
  }
  for _, c := range cases {
    sbWantErr(t, env, "(sandbox-eval sb '"+c.form+")", sbGesperrt(c.name))
  }
}

func TestSandboxFreigabe(t *testing.T) {
  pfad, err := filepath.Abs("../../tmp/sandbox-freigabe.txt")
  if err != nil {
    t.Fatal(err)
  }
  defer os.Remove(pfad)
  env := sbEnv(t, "(make-sandbox :erlaube '(file-write) :zeitlimit 2)")
  sbRun(env, `(sandbox-eval sb '(file-write "`+pfad+`" "ok"))`)
  if b, err := os.ReadFile(pfad); err != nil || string(b) != "ok" {
    t.Fatalf("file-write freigegeben: %q, %v", b, err)
  }
  sbWantErr(t, env, `(sandbox-eval sb '(file-delete "`+pfad+`"))`, sbGesperrt("file-delete"))
  if st := sandboxen[mustSandboxRoot(t, env)]; st.zeitlimit != 2 {
    t.Errorf("zeitlimit = %v, want 2", st.zeitlimit)
  }
  sbWant(t, sbEnv(t, "(make-sandbox :erlaube '(exec))"), `(sandbox-eval sb '(exec "true"))`, "t")
}

func mustSandboxRoot(t *testing.T, env *Env) *Env {
  t.Helper()
  sb, _ := env.Get("sb")
  obj, ok := ForeignObj(sb, "sandbox")
  if !ok {
    t.Fatalf("sb ist keine Sandbox: %s", sb)
  }
  return obj.(*Env)
}

func TestSandboxMakeFehler(t *testing.T) {
  env := sbEnv(t, "(make-sandbox)")
  sbWantErr(t, env, "(make-sandbox :erlaube '(gibtsnicht))", "make-sandbox: gibtsnicht ist keine sperrbare Funktion")
  sbWantErr(t, env, "(make-sandbox :erlaube '(car))", "make-sandbox: car ist keine sperrbare Funktion")
  sbWantErr(t, env, "(make-sandbox :zeitlimit 0)", ":zeitlimit muss Zahl > 0 sein")
  sbWantErr(t, env, `(make-sandbox :zeitlimit "x")`, ":zeitlimit muss Zahl > 0 sein")
  sbWantErr(t, env, "(make-sandbox :foo 1)", "unbekanntes Schlüsselwort")
  sbWantErr(t, env, "(make-sandbox :zeitlimit)", "Schlüsselwort ohne Wert")
  sbWantErr(t, env, "(sandbox-eval 5 1)", "Sandbox erwartet")
  if st := sandboxen[mustSandboxRoot(t, env)]; st.zeitlimit != 10 {
    t.Errorf("zeitlimit = %v, want 10", st.zeitlimit)
  }
  sbWant(t, env, "sb", "#<foreign sandbox>")
}

func TestSandboxIsolation(t *testing.T) {
  env := sbEnv(t, "(make-sandbox)")
  sbRun(env, "(define sb2 (make-sandbox))")
  sbWant(t, env, `(sandbox-eval sb "(defun a () 1) (defun b () 2) (+ (a) (b))")`, "3")
  sbWant(t, env, `(sandbox-eval sb "(list (a) (b))")`, "(1 2)")
  sbWantErr(t, env, `(sandbox-eval sb2 '(a))`, "unbekanntes Symbol 'a'")
  sbWant(t, env, "(bound? 'a)", "()")
  // Fehler kommen unverändert und fangbar an
  sbWant(t, env, `(handler-case (sandbox-eval sb '(error "kaputt")) (error (e) "gefangen"))`, `"gefangen"`)
}

func TestSandboxFunktionAussenBleibtEingeschraenkt(t *testing.T) {
  env := sbEnv(t, "(make-sandbox)")
  sbRun(env, `(sandbox-eval sb "(defun f (x) (system x)) (defun g (x) (* x 2))")`)
  sbWant(t, env, "(funcall (sandbox-eval sb 'g) 21)", "42")
  sbWantErr(t, env, `(funcall (sandbox-eval sb 'f) "ls")`, sbGesperrt("system"))
  sbWantErr(t, env, `((sandbox-eval sb 'f) "ls")`, sbGesperrt("system"))
}

func TestSandboxStandardFreigaben(t *testing.T) {
  pfad, err := filepath.Abs("../../tmp/sandbox-lesen.txt")
  if err != nil {
    t.Fatal(err)
  }
  if err := os.WriteFile(pfad, []byte("inhalt"), 0644); err != nil {
    t.Fatal(err)
  }
  defer os.Remove(pfad)
  env := sbEnv(t, "(make-sandbox)")
  sbWant(t, env, `(sandbox-eval sb '(println "hallo"))`, `"hallo"`)
  sbWant(t, env, `(sandbox-eval sb '(format nil "~a-~a" 1 2))`, `"1-2"`)
  sbWant(t, env, `(sandbox-eval sb '(parfunc r (* 1 10) (* 2 10)))`, "(10 20)")
  sbWant(t, env, `(sandbox-eval sb '(chan-recv (spawn (lambda () 7))))`, "(:ok 7)")
  sbWant(t, env, `(sandbox-eval sb '(file-read "`+pfad+`"))`, `"inhalt"`)
  sbWant(t, env, `(sandbox-eval sb '(list (bound? 'sigo) (bound? 'sigo-host) (bound? 'reduce)))`, "(t () t)")
  sbWant(t, env, `(sandbox-eval sb '(sort (list 3 1 2) '<))`, "(1 2 3)")
}

// Ein zweites BaseEnv darf die globale Umgebung nicht umbiegen.
func TestSandboxStoertGlobalNicht(t *testing.T) {
  env := sbEnv(t, "(make-sandbox)")
  sbWant(t, env, `(begin (defun up (s) (string-upcase s)) (make-sandbox) (format nil "~/up/" "hi"))`, `"HI"`)
  sbWant(t, env, `(begin (defun tr (x) x) (trace 'tr) (make-sandbox) (untrace 'tr))`, "tr")
}

// Kein Name in der Sandbox-Root darf auf eine gesperrte Primitive zeigen
// (Alias in stdlib wäre ein Fluchtweg).
func TestSandboxKeinAlias(t *testing.T) {
  root := BaseEnv()
  gesperrt := map[*Cell]string{}
  for _, n := range root.Symbols() {
    if !sandboxErlaubt()[n] {
      c, _ := root.Get(n)
      gesperrt[c] = n
    }
  }
  if err := LoadStdlib(root); err != nil {
    t.Fatal(err)
  }
  for _, n := range root.Symbols() {
    c, _ := root.Get(n)
    if orig, ok := gesperrt[c]; ok && orig != n {
      t.Errorf("%s ist Alias der gesperrten Primitive %s", n, orig)
    }
  }
}

// Hält die Sperrliste fest: eine neue Primitive in BaseEnv fällt hier auf.
func TestSandboxSperrliste(t *testing.T) {
  env := sbEnv(t, "(make-sandbox)")
  st := sandboxen[mustSandboxRoot(t, env)]
  var got []string
  for n := range st.entfernt {
    got = append(got, n)
  }
  slices.Sort(got)
  want := []string{
    "argv", "browser-open", "defmain", "environ", "exec", "exit",
    "file-append", "file-delete", "file-write", "fprintf",
    "ga-calc", "ga-create", "ga-cross", "ga-init", "ga-mut", "ga-print",
    "ga-result", "ga-select", "ga?", "getenv", "gets",
    "http-port", "http-serve", "http-static", "http-stop", "http-upload",
    "http-wait", "lisp-implementation-version", "load", "make-sandbox",
    "maxima-close", "maxima-eval", "maxima-open",
    "pg-close", "pg-connect", "pg-exec", "pg-query", "read-line",
    "redef-log", "redef-log-clear", "redefine-policy", "sandbox-eval",
    "set-working-directory", "sigo-host", "sigo-system-prompt",
    "sigo-usage-reset", "slurp", "system", "trace", "trace?", "untrace",
    "webserv", "ws-call", "ws-clients", "ws-emit", "ws-emit-to", "ws-eval",
    "ws-export", "ws-unexport",
  }
  if !slices.Equal(got, want) {
    t.Errorf("Sperrliste geändert:\n got: %q\nwant: %q", got, want)
  }
}
