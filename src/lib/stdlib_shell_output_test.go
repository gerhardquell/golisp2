//**********************************************************************
//  lib/stdlib_shell_output_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261003
//**********************************************************************
// Tests für (shell-output "cmd") — Fassade über exec in stdlib.lisp
// (TODO-20261003-luecken Punkt 2).
//**********************************************************************

package lib

import (
  "strings"
  "testing"
)

func TestShellOutput(t *testing.T) {
  evalStdlibEq(t, `(shell-output "echo hallo")`, `"hallo"`)
  // wie $(…): alle abschließenden Newlines weg, führende Leerzeichen bleiben
  evalStdlibEq(t, `(shell-output "printf '  a\n\n'")`, `"  a"`)
  evalStdlibEq(t, `(shell-output "printf 'a\nb\n'")`, `"a\nb"`)
  evalStdlibEq(t, `(shell-output "true")`, `""`)
  // Shell-Syntax (Pipe) funktioniert
  evalStdlibEq(t, `(shell-output "echo abc | tr a-z A-Z")`, `"ABC"`)
  // keine Bindung leckt ins globale Env
  evalStdlibEq(t, `(begin (shell-output "true") (bound? 'out))`, "()")
}

func TestShellOutputErrors(t *testing.T) {
  evalStdlibErr(t, `(shell-output 5)`)
  cases := []struct{ src, want string }{
    {`(shell-output "exit 3")`, "Exit 3"},
    {`(shell-output "echo kaputt >&2; exit 1")`, "kaputt"},
  }
  for _, c := range cases {
    _, err := evalStdlib(t, c.src)
    if err == nil || !strings.Contains(err.Error(), "shell-output:") || !strings.Contains(err.Error(), c.want) {
      t.Errorf("%s: Fehler soll shell-output und %q nennen, got %v", c.src, c.want, err)
    }
  }
}
