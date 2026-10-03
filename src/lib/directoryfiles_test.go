//**********************************************************************
//  lib/directoryfiles_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261003
//**********************************************************************
// Tests für (directory-files dir [muster]) (TODO-20261003-luecken Punkt 4).
//**********************************************************************

package lib

import (
  "os"
  "path/filepath"
  "testing"
)

func TestDirectoryFiles(t *testing.T) {
  dir := t.TempDir()
  for _, name := range []string{"b.lisp", "a.txt", ".versteckt", "c.lisp"} {
    if err := os.WriteFile(filepath.Join(dir, name), nil, 0644); err != nil {
      t.Fatal(err)
    }
  }
  if err := os.Mkdir(filepath.Join(dir, "unter"), 0755); err != nil {
    t.Fatal(err)
  }
  env := BaseEnv()
  eq := func(src, want string) {
    t.Helper()
    got, err := LoadString(src, env)
    if err != nil {
      t.Fatalf("%s: %v", src, err)
    }
    if got.String() != want {
      t.Errorf("%s = %s, want %s", src, got, want)
    }
  }
  q := `"` + dir + `"`
  // sortiert, Dotfiles dabei, Verzeichnisse mit "/"
  eq(`(directory-files `+q+`)`, `(".versteckt" "a.txt" "b.lisp" "c.lisp" "unter/")`)
  eq(`(directory-files `+q+` "*.lisp")`, `("b.lisp" "c.lisp")`)
  eq(`(directory-files `+q+` "unt*")`, `("unter/")`)
  eq(`(directory-files `+q+` "*.go")`, `()`)
  eq(`(directory-files (string-append `+q+` "/unter"))`, `()`)

  for _, src := range []string{
    `(directory-files)`,
    `(directory-files 5)`,
    `(directory-files "/gibts/nicht")`,
    `(directory-files ` + `"` + filepath.Join(dir, "a.txt") + `")`, // Datei, kein Verzeichnis
    `(directory-files ` + q + ` "[")`,                                // kaputtes Muster
    `(directory-files ` + q + ` 1)`,
  } {
    if _, err := LoadString(src, env); err == nil {
      t.Errorf("%s: Fehler erwartet", src)
    }
  }
}

// Relativer Pfad wird wie bei file-read über working-directory aufgelöst.
func TestDirectoryFilesWorkingDirectory(t *testing.T) {
  projekt := t.TempDir()
  if err := os.Mkdir(filepath.Join(projekt, "daten"), 0755); err != nil {
    t.Fatal(err)
  }
  if err := os.WriteFile(filepath.Join(projekt, "daten", "x.csv"), nil, 0644); err != nil {
    t.Fatal(err)
  }
  workingDirectoryMu.Lock()
  oldDir := workingDirectory
  workingDirectory = projekt
  workingDirectoryMu.Unlock()
  defer func() {
    workingDirectoryMu.Lock()
    workingDirectory = oldDir
    workingDirectoryMu.Unlock()
  }()

  got, err := LoadString(`(directory-files "daten")`, BaseEnv())
  if err != nil {
    t.Fatal(err)
  }
  if got.String() != `("x.csv")` {
    t.Errorf("got %s", got)
  }
}
