//**********************************************************************
//  lib/swank/lisp_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260618
//**********************************************************************
// Smoke test für eingebettete SWANK Lisp-Handler.
//**********************************************************************

package swank

import (
  "fmt"
  "os"
  "strings"
  "testing"

  "golisp2/src/lib"
)

func TestSwankLisp(t *testing.T) {
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  cell, err := lib.Read("(:emacs-rex (swank:connection-info) nil t 1)")
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  if result == nil || !strings.Contains(result.String(), ":return") {
    t.Fatalf("unexpected result: %v", result)
  }
}

func TestSwankOperatorArglistBuiltIn(t *testing.T) {
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  cell, err := lib.Read("(:emacs-rex (swank:operator-arglist \"car\" \"USER\") nil t 1)")
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, "(car list)") {
    t.Fatalf("expected built-in arglist (car list), got: %s", s)
  }
}

func TestSwankAutodocBuiltIn(t *testing.T) {
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  cell, err := lib.Read("(:emacs-rex (swank:autodoc (quote (car x)) \"USER\") nil t 1)")
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, "(car list)") {
    t.Fatalf("expected autodoc (car list), got: %s", s)
  }
  if strings.Contains(s, ":not-available") {
    t.Fatalf("autodoc should be available for built-in car, got: %s", s)
  }
}

func TestSwankOperatorArglistLambda(t *testing.T) {
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  defCell, err := lib.Read("(defun mytestfn (a b) (+ a b))")
  if err != nil {
    t.Fatalf("read defun: %v", err)
  }
  if _, err := lib.Eval(defCell, env); err != nil {
    t.Fatalf("eval defun: %v", err)
  }
  cell, err := lib.Read("(:emacs-rex (swank:operator-arglist \"mytestfn\" \"USER\") nil t 1)")
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, "(mytestfn a b)") {
    t.Fatalf("expected lambda arglist (mytestfn a b), got: %s", s)
  }
  if strings.Contains(s, "(mytestfn &rest") {
    t.Fatalf("lambda should not fall back to built-in registry, got: %s", s)
  }
}

func TestSwankDescribeSymbolBuiltIn(t *testing.T) {
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  cell, err := lib.Read("(:emacs-rex (swank:describe-symbol \"car\" \"USER\") nil t 1)")
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, ":title \"car\"") {
    t.Fatalf("expected title car, got: %s", s)
  }
  if !strings.Contains(s, "function") {
    t.Fatalf("expected type function, got: %s", s)
  }
  if !strings.Contains(s, "erste Element") {
    t.Fatalf("expected static description, got: %s", s)
  }
}

func TestSwankDescribeSymbolLambda(t *testing.T) {
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  defCell, err := lib.Read("(defun mydescfn (x) (* x x))")
  if err != nil {
    t.Fatalf("read defun: %v", err)
  }
  if _, err := lib.Eval(defCell, env); err != nil {
    t.Fatalf("eval defun: %v", err)
  }
  cell, err := lib.Read("(:emacs-rex (swank:describe-symbol \"mydescfn\" \"USER\") nil t 1)")
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, ":title \"mydescfn\"") {
    t.Fatalf("expected title mydescfn, got: %s", s)
  }
  if !strings.Contains(s, "lambda") {
    t.Fatalf("expected type lambda, got: %s", s)
  }
  if !strings.Contains(s, "(mydescfn x)") {
    t.Fatalf("expected arglist (mydescfn x), got: %s", s)
  }
}

func TestSwankDescribeSymbolUnbound(t *testing.T) {
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  cell, err := lib.Read("(:emacs-rex (swank:describe-symbol \"definitely-not-bound-symbol\" \"USER\") nil t 1)")
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, ":title \"definitely-not-bound-symbol\"") {
    t.Fatalf("expected title, got: %s", s)
  }
  if !strings.Contains(s, "unbound") {
    t.Fatalf("expected unbound type, got: %s", s)
  }
}

func TestSwankCompileString(t *testing.T) {
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  cell, err := lib.Read("(:emacs-rex (swank:compile-string-for-emacs \"(defun compile-string-test () 123)\") nil t 1)")
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, ":ok t") {
    t.Fatalf("expected :ok t, got: %s", s)
  }
  checkCell, err := lib.Read("(compile-string-test)")
  if err != nil {
    t.Fatalf("read check: %v", err)
  }
  val, err := lib.Eval(checkCell, env)
  if err != nil {
    t.Fatalf("eval check: %v", err)
  }
  if val.String() != "123" {
    t.Fatalf("expected 123, got: %s", val.String())
  }
}

func TestSwankCompileFile(t *testing.T) {
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  if err := os.MkdirAll("./tmp", 0755); err != nil {
    t.Fatalf("mkdir tmp: %v", err)
  }
  tmpFile := "./tmp/compile-file-test.lisp"
  if err := os.WriteFile(tmpFile, []byte("(defun compile-file-test () 456)"), 0644); err != nil {
    t.Fatalf("write tmp file: %v", err)
  }
  defer os.Remove(tmpFile)

  req := fmt.Sprintf("(:emacs-rex (swank:compile-file-for-emacs %q) nil t 1)", tmpFile)
  cell, err := lib.Read(req)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, ":ok") || strings.Contains(s, ":abort") {
    t.Fatalf("expected ok result, got: %s", s)
  }
  checkCell, err := lib.Read("(compile-file-test)")
  if err != nil {
    t.Fatalf("read check: %v", err)
  }
  val, err := lib.Eval(checkCell, env)
  if err != nil {
    t.Fatalf("eval check: %v", err)
  }
  if val.String() != "456" {
    t.Fatalf("expected 456, got: %s", val.String())
  }
}

func TestSwankMacroexpandAll(t *testing.T) {
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  cell, err := lib.Read("(:emacs-rex (swank:swank-macroexpand-all \"(list (when t 1))\") nil t 1)")
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, "(list (if t (begin 1) ()))") {
    t.Fatalf("expected recursive expansion, got: %s", s)
  }
  if strings.Contains(s, "when") {
    t.Fatalf("macroexpand-all should not leave when unexpanded, got: %s", s)
  }
}

func TestSwankListenerEvalPrintSuppressesResult(t *testing.T) {
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  var writes []string
  RegisterSwankEnv(env, func(c *lib.Cell) error {
    writes = append(writes, c.String())
    return nil
  })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  cell, err := lib.Read(`(:emacs-rex (swank-repl:listener-eval "(print \"test\")") "USER" t 1)`)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if strings.Contains(s, ":repl-result") {
    t.Fatalf("print result should not be emitted as repl-result, got: %s", s)
  }
  found := false
  for _, w := range writes {
    if strings.Contains(w, "test") {
      found = true
    }
  }
  if !found {
    t.Fatalf("expected print output event, got writes: %v", writes)
  }
}

func TestSwankListenerEvalNormalResult(t *testing.T) {
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  cell, err := lib.Read(`(:emacs-rex (swank-repl:listener-eval "(+ 1 2)") "USER" t 1)`)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, ":repl-result") {
    t.Fatalf("normal result should have :repl-result, got: %s", s)
  }
  if !strings.Contains(s, "3") {
    t.Fatalf("expected result 3, got: %s", s)
  }
}

func TestSwankFindDefinitionsFileLocation(t *testing.T) {
  lib.ClearDefinitions()
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  lib.RegisterDefinition("myfn", "/abs/src.lisp", 12)
  cell, err := lib.Read(`(:emacs-rex (swank:find-definitions-for-emacs "myfn") nil t 1)`)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, ":location") || !strings.Contains(s, "/abs/src.lisp") || !strings.Contains(s, ":line") {
    t.Fatalf("expected file location, got: %s", s)
  }
  // SLIME erwartet (name location)-Paare, keine nackte Location.
  if !strings.Contains(s, `("myfn" (:location`) {
    t.Fatalf("expected (name location) dspec-wrapper, got: %s", s)
  }
  // :line N ohne :align (SLIME-kompatibel).
  if !strings.Contains(s, "(:line 12)") || strings.Contains(s, ":align") {
    t.Fatalf("expected (:line 12) without :align, got: %s", s)
  }
}

func TestSwankFindDefinitionsBuiltInError(t *testing.T) {
  lib.ClearDefinitions()
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  cell, err := lib.Read(`(:emacs-rex (swank:find-definitions-for-emacs "car") nil t 1)`)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, ":error") {
    t.Fatalf("expected :error for built-in, got: %s", s)
  }
}

func TestSwankFindDefinitionsReplSnippet(t *testing.T) {
  lib.ClearDefinitions()
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  // REPL-definiert (kein SrcFile): via listener-eval definieren
  leval, err := lib.Read(`(:emacs-rex (swank-repl:listener-eval "(defun rfn (n) (* n 2))") nil t 1)`)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  if _, err := HandleMessage(env, leval); err != nil {
    t.Fatalf("listener-eval: %v", err)
  }
  cell, err := lib.Read(`(:emacs-rex (swank:find-definitions-for-emacs "rfn") nil t 1)`)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  result, err := HandleMessage(env, cell)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, ":buffer") || !strings.Contains(s, "rfn") {
    t.Fatalf("expected buffer snippet for REPL fn, got: %s", s)
  }
}
