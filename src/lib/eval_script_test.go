//**********************************************************************
//  lib/eval_script_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Sonnet 5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261001
//**********************************************************************
// Tests für RunScript und die Spezialform defmain.
// Spec: docs/superpowers/specs/2026-10-01-defmain-design.md
//**********************************************************************

package lib

import (
  "os"
  "path/filepath"
  "strings"
  "testing"
)

func scriptEnv(t *testing.T) *Env {
  t.Helper()
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("LoadStdlib: %v", err)
  }
  return env
}

func writeScript(t *testing.T, dir, name, src string) string {
  t.Helper()
  path := filepath.Join(dir, name)
  if err := os.WriteFile(path, []byte(src), 0644); err != nil {
    t.Fatalf("WriteFile: %v", err)
  }
  return path
}

func runScriptSrc(t *testing.T, src string, args ...string) (int, *Cell, bool, error) {
  t.Helper()
  path := writeScript(t, t.TempDir(), "main.lisp", src)
  return RunScript(path, args, scriptEnv(t))
}

func TestDefmainArgsBecomeExitCode(t *testing.T) {
  code, _, hasMain, err := runScriptSrc(t, `(defmain (args) (parse-int (car args)))`, "7")
  if err != nil { t.Fatalf("err = %v", err) }
  if !hasMain { t.Fatal("hasMain = false, want true") }
  if code != 7 { t.Fatalf("exitCode = %d, want 7", code) }
}

func TestDefmainNoParams(t *testing.T) {
  code, _, hasMain, err := runScriptSrc(t, `(defmain () 0)`, "ignoriert")
  if err != nil { t.Fatalf("err = %v", err) }
  if !hasMain || code != 0 { t.Fatalf("hasMain=%v code=%d, want true/0", hasMain, code) }
}

func TestDefmainRunsAfterWholeFileLoaded(t *testing.T) {
  src := "(defmain () (helper))\n(defun helper () 3)\n"
  code, _, _, err := runScriptSrc(t, src)
  if err != nil { t.Fatalf("err = %v", err) }
  if code != 3 { t.Fatalf("exitCode = %d, want 3", code) }
}

func TestDefmainNestedInMainFile(t *testing.T) {
  code, _, hasMain, err := runScriptSrc(t, `(when t (defmain () 4))`)
  if err != nil { t.Fatalf("err = %v", err) }
  if !hasMain || code != 4 { t.Fatalf("hasMain=%v code=%d, want true/4", hasMain, code) }
}

func TestDefmainNonNumberIsExitZero(t *testing.T) {
  for _, body := range []string{`()`, `"text"`, `(quote sym)`} {
    code, _, _, err := runScriptSrc(t, "(defmain () "+body+")")
    if err != nil { t.Fatalf("%s: err = %v", body, err) }
    if code != 0 { t.Fatalf("%s: exitCode = %d, want 0", body, code) }
  }
}

func TestDefmainBadExitCode(t *testing.T) {
  for _, body := range []string{`256`, `-1`, `2.5`} {
    code, _, _, err := runScriptSrc(t, "(defmain () "+body+")")
    if err == nil || !strings.Contains(err.Error(), "Exit-Code muss ganze Zahl 0–255 sein") {
      t.Fatalf("%s: err = %v, want Exit-Code-Fehler", body, err)
    }
    if code != 1 { t.Fatalf("%s: exitCode = %d, want 1", body, code) }
  }
}

func TestDefmainDuplicateIsError(t *testing.T) {
  src := "(define ran ())\n(defmain () (setq ran t) 0)\n(defmain () 1)\n"
  dir := t.TempDir()
  path := writeScript(t, dir, "main.lisp", src)
  env := scriptEnv(t)
  code, _, _, err := RunScript(path, nil, env)
  if err == nil || !strings.Contains(err.Error(), "defmain: bereits definiert in "+path+":2") {
    t.Fatalf("err = %v, want 'bereits definiert in %s:2'", err, path)
  }
  if code != 1 { t.Fatalf("exitCode = %d, want 1", code) }
  ran, gerr := env.Get("ran")
  if gerr != nil { t.Fatalf("Get ran: %v", gerr) }
  if ran.Type != NIL { t.Fatalf("Körper ist gelaufen (ran = %s), darf nicht", ran) }
}

func TestDefmainBadSyntax(t *testing.T) {
  for _, src := range []string{
    `(defmain (a b) 0)`,
    `(defmain args 0)`,
    `(defmain)`,
    `(defmain (&rest a) 0)`,
  } {
    _, _, _, err := runScriptSrc(t, src)
    if err == nil || !strings.Contains(err.Error(), "defmain: Syntax") {
      t.Fatalf("%s: err = %v, want Syntaxfehler", src, err)
    }
  }
}

func TestDefmainInLoadedLibraryIsIgnored(t *testing.T) {
  dir := t.TempDir()
  // Bibliothek mit eigenem (sogar syntaktisch kaputtem) defmain:
  // unter load wird nichts geprüft und nichts registriert.
  lib := writeScript(t, dir, "lib.lisp", "(defmain (a b) 99)\n(define lib-loaded t)\n")
  main := writeScript(t, dir, "main.lisp",
    "(load \""+lib+"\")\n(defmain () (if lib-loaded 5 6))\n")
  code, _, hasMain, err := RunScript(main, nil, scriptEnv(t))
  if err != nil { t.Fatalf("err = %v", err) }
  if !hasMain || code != 5 { t.Fatalf("hasMain=%v code=%d, want true/5", hasMain, code) }
}

func TestDefmainOutsideRunScriptIsNil(t *testing.T) {
  env := scriptEnv(t)
  res, err := LoadString(`(defmain () 1)`, env)
  if err != nil { t.Fatalf("LoadString: %v", err) }
  if res.Type != NIL { t.Fatalf("LoadString: defmain = %s, want nil", res) }

  form, err := Read(`(defmain (a b c) 1)`) // ungültig, aber außerhalb ungeprüft
  if err != nil { t.Fatalf("Read: %v", err) }
  res, err = Eval(form, env)
  if err != nil { t.Fatalf("Eval: %v", err) }
  if res.Type != NIL { t.Fatalf("Eval: defmain = %s, want nil", res) }
}

func TestDefmainViaEvalInMainFileIsIgnored(t *testing.T) {
  _, _, hasMain, err := runScriptSrc(t, `(eval (quote (defmain () 9)))`)
  if err != nil { t.Fatalf("err = %v", err) }
  if hasMain { t.Fatal("hasMain = true, want false ((eval …) läuft ohne Script-Kontext)") }
}

func TestDefmainInsideBodyIsIgnored(t *testing.T) {
  code, _, _, err := runScriptSrc(t, `(defmain () (defmain () 1) 0)`)
  if err != nil { t.Fatalf("err = %v", err) }
  if code != 0 { t.Fatalf("exitCode = %d, want 0", code) }
}

func TestDefmainBodyError(t *testing.T) {
  code, _, hasMain, err := runScriptSrc(t, `(defmain () (error "kaputt"))`)
  if err == nil || !strings.Contains(err.Error(), "kaputt") {
    t.Fatalf("err = %v, want 'kaputt'", err)
  }
  if !hasMain || code != 1 { t.Fatalf("hasMain=%v code=%d, want true/1", hasMain, code) }
}

func TestRunScriptWithoutDefmain(t *testing.T) {
  code, res, hasMain, err := runScriptSrc(t, "(define x 1)\n(+ x 2)\n")
  if err != nil { t.Fatalf("err = %v", err) }
  if hasMain { t.Fatal("hasMain = true, want false") }
  if code != 0 { t.Fatalf("exitCode = %d, want 0", code) }
  if res.Type != NUMBER || res.Num != 3 { t.Fatalf("result = %s, want 3", res) }
}

func TestRunScriptQuoteInFilename(t *testing.T) {
  path := writeScript(t, t.TempDir(), `a"b.lisp`, `(defmain () 0)`)
  code, _, hasMain, err := RunScript(path, nil, scriptEnv(t))
  if err != nil { t.Fatalf("err = %v", err) }
  if !hasMain || code != 0 { t.Fatalf("hasMain=%v code=%d", hasMain, code) }
}

func TestRunScriptMissingFile(t *testing.T) {
  code, _, _, err := RunScript(filepath.Join(t.TempDir(), "gibtsnicht.lisp"), nil, scriptEnv(t))
  if err == nil { t.Fatal("err = nil, want Ladefehler") }
  if code != 1 { t.Fatalf("exitCode = %d, want 1", code) }
}
