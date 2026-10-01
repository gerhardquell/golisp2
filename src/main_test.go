//**********************************************************************
//  main_test.go  - GoLisp -e Expression Tests
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260711
//**********************************************************************

package main

import (
  "bytes"
  "os"
  "os/exec"
  "path/filepath"
  "runtime"
  "strings"
  "testing"
)

// TestDefmainShebangEndToEnd startet ein Skript über den echten
// Kernel-Shebang mit frisch gebautem Binary — prüft Argumente,
// Exit-Code und dass kein Rückgabewert auf stdout landet.
func TestDefmainShebangEndToEnd(t *testing.T) {
  if testing.Short() {
    t.Skip("baut Binary — nicht im -short-Modus")
  }
  if runtime.GOOS == "windows" {
    t.Skip("Shebang braucht Unix")
  }
  dir := t.TempDir()
  bin := filepath.Join(dir, "golisp2")
  if len(bin) > 200 {
    t.Skipf("Binary-Pfad zu lang für Shebang-Zeile (%d Byte)", len(bin))
  }
  build := exec.Command("go", "build", "-o", bin, ".")
  if out, err := build.CombinedOutput(); err != nil {
    t.Fatalf("go build: %v\n%s", err, out)
  }

  write := func(name, body string) string {
    p := filepath.Join(dir, name)
    if err := os.WriteFile(p, []byte("#!"+bin+"\n"+body), 0755); err != nil {
      t.Fatalf("WriteFile: %v", err)
    }
    return p
  }
  run := func(script string, args ...string) (string, string, int) {
    cmd := exec.Command(script, args...)
    var so, se bytes.Buffer
    cmd.Stdout, cmd.Stderr = &so, &se
    err := cmd.Run()
    code := 0
    if ee, ok := err.(*exec.ExitError); ok {
      code = ee.ExitCode()
    } else if err != nil {
      t.Fatalf("start %s: %v", script, err)
    }
    return so.String(), se.String(), code
  }

  // Dateiname mit Anführungszeichen: Regression für den alten String-Bau.
  greet := write(`gr"eet.lisp`, `(defun greet (name) (format t "Hallo, ~a!~%" name))
(defmain (args)
  (if (null args)
      (begin (warn "Aufruf: greet NAME") 2)
      (begin (greet (car args)) 0)))
`)
  if out, _, code := run(greet, "Anna"); out != "Hallo, Anna!\n" || code != 0 {
    t.Fatalf("mit Argument: stdout=%q code=%d, want %q/0", out, code, "Hallo, Anna!\n")
  }
  if out, errOut, code := run(greet); out != "" || code != 2 || !strings.Contains(errOut, "Aufruf") {
    t.Fatalf("ohne Argument: stdout=%q stderr=%q code=%d, want leer/Aufruf/2", out, errOut, code)
  }

  // Ohne defmain: altes Verhalten (letzter Wert auf stdout, Exit 0).
  plain := write("plain.lisp", "(+ 1 2)\n")
  if out, _, code := run(plain); out != "3\n" || code != 0 {
    t.Fatalf("ohne defmain: stdout=%q code=%d, want %q/0", out, code, "3\n")
  }

  // Fehler im Körper: ERR auf stderr, Exit 1.
  boom := write("boom.lisp", `(defmain () (error "kaputt"))`+"\n")
  if _, errOut, code := run(boom); code != 1 || !strings.Contains(errOut, "kaputt") {
    t.Fatalf("Fehler: stderr=%q code=%d, want kaputt/1", errOut, code)
  }
}
