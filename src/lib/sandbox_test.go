//**********************************************************************
//  lib/sandbox_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Opus 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261117
//**********************************************************************

package lib

import (
  "context"
  "os"
  "path/filepath"
  "slices"
  "strings"
  "testing"
  "time"
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
  // sigo nur mit :erlaube (Gerhard 17.11.: Datei lesen + sigo wäre ein Weg nach außen)
  sbWant(t, env, `(sandbox-eval sb '(list (bound? 'sigo) (bound? 'sigo-host) (bound? 'reduce)))`, "(() () t)")
  sbWant(t, env, `(sandbox-eval (make-sandbox :erlaube '(sigo)) '(list (bound? 'sigo) (bound? 'sigo*)))`, "(t ())")
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
    "set-working-directory", "sigo", "sigo*", "sigo-budget", "sigo-costs",
    "sigo-host", "sigo-model-info", "sigo-models", "sigo-reference", "sigo-request",
    "sigo-system-prompt", "sigo-usage", "sigo-usage-reset", "slurp", "system", "trace", "trace?", "untrace",
    "webserv", "ws-call", "ws-clients", "ws-emit", "ws-emit-to", "ws-eval",
    "ws-export", "ws-unexport",
  }
  if !slices.Equal(got, want) {
    t.Errorf("Sperrliste geändert:\n got: %q\nwant: %q", got, want)
  }
}

// Hält die erlaubten BaseEnv-Namen fest (ohne stdlib-Lisp): eine neue
// Primitive in einer Grundmengen-Gruppe wäre sonst still erlaubt.
func TestSandboxErlaubtListe(t *testing.T) {
  root := BaseEnv()
  var got []string
  for _, n := range root.Symbols() {
    if sandboxErlaubt()[n] {
      got = append(got, n)
    }
  }
  slices.Sort(got)
  want := []string{
    "%cell-type", "%check-special", "%declare-special", "%ganzzahl-pruefen",
    "%last-cons", "%list-length", "%list->vector", "%make-struct", "%vector->list", "%vector-length", "*", "+", "-", "/",
    "/=", "<", "<=", "=", ">", ">=", "abs", "append", "apply", "aref", "arrayp", "aset",
    "ash", "atom", "atom?", "car", "cdr", "ceiling", "chan-close", "chan-closed?",
    "chan-make", "chan-recv", "chan-select", "chan-send", "chan-try-recv", "chan-try-send",
    "clrhash", "concatenate", "cons", "copy-list", "copy-seq", "copy-tree", "count", "count-if", "defined-in",
    "directory-files", "documentation", "drop", "elt", "env-symbols", "eq", "eq?", "eql",
    "equal?", "err-write", "error", "exp", "expt", "file-exists?", "file-read",
    "file-stat", "find", "find-if", "float", "floatp", "floor", "format", "format-time",
    "funcall", "gensym", "get-file-path", "get-universal-time", "get-working-directory",
    "gethash", "hash-table-count", "hash-table-p", "integerp", "intern", "json-encode",
    "json-parse", "list", "list->string", "list-tail", "list?", "lock-make", "log",
    "make-array", "make-hash-table", "map", "mapcar", "maphash", "memstats", "mod", "nconc", "nil",
    "now", "nth", "nthcdr", "null", "null?", "number->string", "number?", "parse-float", "parse-int",
    "position", "position-if", "print", "printf", "println", "puthash", "random", "read",
    "regex-find", "regex-find-all", "regex-match", "regex-replace", "regex-split", "rem",
    "remainder", "remhash", "round", "rplaca", "rplacd", "shell-assoc",
    "sleep", "some", "sort", "spawn", "sprintf", "sqrt", "sscanf",
    "string->list", "string->number", "string-append", "string-contains",
    "string-downcase", "string-join", "string-length", "string-replace", "string-split",
    "string-trim", "string-upcase", "string/=", "string<", "string<=", "string=",
    "string>", "string>=", "string?", "subseq", "substring", "symbol->string",
    "symbol-name", "symbol?", "t", "take", "truncate", "values", "vector", "vector-pop",
    "vector-push-extend", "vectorp", "warn",
  }
  for _, n := range got {
    if !slices.Contains(want, n) {
      t.Errorf("neue Primitive %s: in sandbox.go bewusst erlauben oder sperren", n)
    }
  }
  for _, n := range want {
    if !slices.Contains(got, n) {
      t.Errorf("%s nicht mehr erlaubt: Liste hier nachziehen", n)
    }
  }
}

// sbFrist: src muss innerhalb von 5 s mit einem Fehler enden, der want enthält.
func sbFrist(t *testing.T, env *Env, src, want string) {
  t.Helper()
  done := make(chan error, 1)
  go func() { _, err := sbRun(env, src); done <- err }()
  select {
  case err := <-done:
    if err == nil || !strings.Contains(err.Error(), want) {
      t.Errorf("%s: Fehler %v, want …%s…", src, err, want)
    }
  case <-time.After(5 * time.Second):
    t.Fatalf("%s: läuft nach 5 s noch", src)
  }
}

func TestSandboxZeitlimit(t *testing.T) {
  env := sbEnv(t, "(make-sandbox :zeitlimit 1)")
  meldung := "sandbox: Zeitlimit 1 s überschritten"
  sbFrist(t, env, "(sandbox-eval sb '(while t ()))", meldung)
  sbFrist(t, env, "(sandbox-eval sb '(mapcar (lambda (x) (while t ())) '(1)))", meldung)
  sbFrist(t, env, `(sandbox-eval sb '(sort (list 2 1) (lambda (a b) (while t ()))))`, meldung)
  // parfunc-Worker: sandbox-eval kehrt zur Frist zurück, nicht erst mit parfunc
  start := time.Now()
  sbFrist(t, env, "(sandbox-eval sb '(parfunc r (while t ())))", meldung)
  if d := time.Since(start); d > 3*time.Second {
    t.Errorf("parfunc-Worker: %v bis zum Abbruch", d)
  }
  sbFrist(t, sbEnv(t, "(make-sandbox :zeitlimit 0.5)"), "(sandbox-eval sb '(while t ()))",
    "sandbox: Zeitlimit 0.5 s überschritten")
  // Standard 10 s: ein kurzer Aufruf läuft durch
  sbWant(t, sbEnv(t, "(make-sandbox)"), "(sandbox-eval sb '(+ 1 2))", "3")
}

// Eine spawn-Goroutine, die sandbox-eval überlebt, endet mit der Frist.
func TestSandboxZeitlimitSpawn(t *testing.T) {
  env := sbEnv(t, "(make-sandbox :zeitlimit 1)")
  start := time.Now()
  sbRun(env, "(define r (sandbox-eval sb '(spawn (lambda () (while t ())))))")
  // überlebt das sandbox-eval, bis zur Frist
  sbWant(t, env, "(chan-recv (sandbox-eval sb '(spawn (lambda () (sleep 0.2) 42))))", "(:ok 42)")
  sbFrist(t, env, `(let ((m (chan-recv r))) (error (format nil "~a" m)))`, "cancelled")
  if d := time.Since(start); d < 900*time.Millisecond {
    t.Errorf("spawn-Goroutine nach %v beendet, vor der Frist", d)
  }
}

// Spec §5: ein Callback im abgebrochenen parfunc-Worker bricht jetzt ab.
// Vorher lief die Schleife im mapcar-Callback nach dem :timeout weiter.
func TestDynCallbackSiehtAbbruch(t *testing.T) {
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatal(err)
  }
  sbWant(t, env, `(begin
    (define c (chan-make 100000))
    (parfunc r :timeout 0.2
      (mapcar (lambda (x) (while t (chan-try-send c 1) (sleep 0.001))) '(1)))
    (sleep 0.1)
    (while (chan-try-recv c) ())
    (sleep 0.2)
    (chan-try-recv c))`, "()")
}

// Blockierendes Warten (sleep, chan-*, lock-Deadlock) endet mit der Frist.
func TestSandboxZeitlimitBlockiert(t *testing.T) {
  env := sbEnv(t, "(make-sandbox :zeitlimit 0.5)")
  meldung := "sandbox: Zeitlimit 0.5 s überschritten"
  for _, form := range []string{
    "(sleep 3000)",
    "(chan-recv (chan-make))",
    "(chan-send (chan-make) 1)",
    "(chan-select (list (chan-make)))",
    "(let ((l (lock-make))) (lock l (lock l 1)))",
  } {
    start := time.Now()
    sbFrist(t, env, "(sandbox-eval sb '"+form+")", meldung)
    if d := time.Since(start); d > 2500*time.Millisecond {
      t.Errorf("%s: %v bis zum Abbruch", form, d)
    }
  }
}

func TestSandboxNichtFreigebbar(t *testing.T) {
  env := sbEnv(t, "(make-sandbox)")
  sbWantErr(t, env, "(make-sandbox :erlaube '(make-sandbox))", "make-sandbox: make-sandbox ist nicht freigebbar")
  sbWantErr(t, env, "(make-sandbox :erlaube '(sandbox-eval))", "make-sandbox: sandbox-eval ist nicht freigebbar")
}

func TestSandboxFileReadNurRegulaer(t *testing.T) {
  pfad := filepath.Join(t.TempDir(), "regulaer.txt")
  if err := os.WriteFile(pfad, []byte("inhalt"), 0644); err != nil {
    t.Fatal(err)
  }
  env := sbEnv(t, "(make-sandbox :zeitlimit 2)")
  sbWantErr(t, env, `(sandbox-eval sb '(file-read "/dev/zero"))`, "file-read: in der Sandbox nur reguläre Dateien: /dev/zero")
  sbWantErr(t, env, `(sandbox-eval sb '(file-read "sys-stdin"))`, "file-read: in der Sandbox nur reguläre Dateien: sys-stdin")
  sbWant(t, env, `(sandbox-eval sb '(file-read "`+pfad+`"))`, `"inhalt"`)
}

// Außerhalb der Sandbox: sleep hört auf den Abbruch-Kontext.
func TestSleepAbbruch(t *testing.T) {
  ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
  defer cancel()
  start := time.Now()
  _, err := fnSleep([]*Cell{MakeInt(3000)}, evalCtx{run: &runState{ctx: ctx}})
  if err == nil || err.Error() != "eval: cancelled" {
    t.Errorf("sleep: Fehler %v, want eval: cancelled", err)
  }
  if d := time.Since(start); d > time.Second {
    t.Errorf("sleep: %v bis zum Abbruch", d)
  }
}
