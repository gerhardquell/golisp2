//**********************************************************************
//  lib/fileio_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260715
//**********************************************************************

package lib

import (
  "os"
  "path/filepath"
  "strings"
  "testing"
)

func TestWorkingDirectory(t *testing.T) {
  wd, err := os.Getwd()
  if err != nil { t.Fatalf("getwd: %v", err) }
  defer os.Chdir(wd)

  // projekt simuliert das Verzeichnis, aus dem heraus gearbeitet wird;
  // cwd ist ein anderes Verzeichnis (SWANK-Szenario).
  projekt := t.TempDir()
  cwd := t.TempDir()
  if err := os.WriteFile(filepath.Join(projekt, "config.txt"), []byte("ok"), 0644); err != nil {
    t.Fatalf("write config: %v", err)
  }
  if err := os.WriteFile(filepath.Join(projekt, "module.lisp"), []byte("(define module-loaded t)\n"), 0644); err != nil {
    t.Fatalf("write module: %v", err)
  }
  if err := os.Chdir(cwd); err != nil { t.Fatalf("chdir: %v", err) }

  // Globalen Zustand am Ende wiederherstellen.
  workingDirectoryMu.Lock()
  oldDir := workingDirectory
  workingDirectoryMu.Unlock()
  defer func() {
    workingDirectoryMu.Lock()
    workingDirectory = oldDir
    workingDirectoryMu.Unlock()
  }()

  env := BaseEnv()

  // Ohne gesetztes working-directory: get → ()
  got, err := LoadString(`(get-working-directory)`, env)
  if err != nil { t.Fatalf("get leer: %v", err) }
  if got.Type != NIL {
    t.Fatalf("get-working-directory = () erwartet, got %s", got)
  }

  // Nicht existierendes Verzeichnis → Fehler
  if _, err := LoadString(`(set-working-directory "/gibts/nicht")`, env); err == nil {
    t.Fatalf("set-working-directory mit ungültigem Pfad: Fehler erwartet")
  }

  // Setzen + Zurücklesen
  if _, err := LoadString(`(set-working-directory "`+projekt+`")`, env); err != nil {
    t.Fatalf("set: %v", err)
  }
  got, err = LoadString(`(get-working-directory)`, env)
  if err != nil { t.Fatalf("get: %v", err) }
  if got.Type != STRING || got.Val != projekt {
    t.Fatalf("get-working-directory = %q erwartet, got %q", projekt, got.Val)
  }

  // file-read über working-directory
  got, err = LoadString(`(file-read "config.txt")`, env)
  if err != nil { t.Fatalf("file-read: %v", err) }
  if got.Type != STRING || got.Val != "ok" {
    t.Fatalf("file-read = %q erwartet, got %q", "ok", got.Val)
  }

  // file-exists? über working-directory
  got, err = LoadString(`(file-exists? "config.txt")`, env)
  if err != nil { t.Fatalf("file-exists?: %v", err) }
  if got.Type != ATOM || got.Val != "t" {
    t.Fatalf("file-exists? = t erwartet, got %s", got)
  }

  // get-file-path über working-directory
  want := filepath.Join(projekt, "config.txt")
  got, err = LoadString(`(get-file-path "config.txt")`, env)
  if err != nil { t.Fatalf("get-file-path: %v", err) }
  if got.Type != STRING || got.Val != want {
    t.Fatalf("get-file-path = %q erwartet, got %q", want, got.Val)
  }

  // load über working-directory
  if _, err := LoadString(`(load "module.lisp")`, env); err != nil {
    t.Fatalf("load: %v", err)
  }
  got, err = LoadString(`module-loaded`, env)
  if err != nil { t.Fatalf("module-loaded: %v", err) }
  if got.Type != ATOM || got.Val != "t" {
    t.Fatalf("module-loaded = t erwartet, got %s", got)
  }

  // file-write schreibt ins working-directory, nicht ins Prozess-cwd
  if _, err := LoadString(`(file-write "neu.txt" "inhalt")`, env); err != nil {
    t.Fatalf("file-write: %v", err)
  }
  data, err := os.ReadFile(filepath.Join(projekt, "neu.txt"))
  if err != nil || string(data) != "inhalt" {
    t.Fatalf("file-write landete nicht im working-directory: %v", err)
  }
  if _, err := os.Stat(filepath.Join(cwd, "neu.txt")); err == nil {
    t.Fatalf("file-write schrieb ins Prozess-cwd — darf nicht sein")
  }

  // file-append hängt an (über working-directory)
  if _, err := LoadString(`(file-append "neu.txt" "-plus")`, env); err != nil {
    t.Fatalf("file-append: %v", err)
  }
  data, err = os.ReadFile(filepath.Join(projekt, "neu.txt"))
  if err != nil || string(data) != "inhalt-plus" {
    t.Fatalf("file-append = %q erwartet, got %q (%v)", "inhalt-plus", data, err)
  }

  // file-delete über working-directory
  if _, err := LoadString(`(file-delete "neu.txt")`, env); err != nil {
    t.Fatalf("file-delete: %v", err)
  }
  if _, err := os.Stat(filepath.Join(projekt, "neu.txt")); err == nil {
    t.Fatalf("file-delete: Datei existiert noch")
  }

  // working-directory gewinnt vor Prozess-cwd beim Lesen
  if err := os.WriteFile(filepath.Join(cwd, "config.txt"), []byte("cwd"), 0644); err != nil {
    t.Fatalf("write cwd config: %v", err)
  }
  got, err = LoadString(`(file-read "config.txt")`, env)
  if err != nil { t.Fatalf("file-read cwd: %v", err) }
  if got.Val != "ok" {
    t.Fatalf("working-directory sollte cwd schlagen: %q erwartet, got %q", "ok", got.Val)
  }

  // Zurücksetzen mit leerem String → cwd-Datei wird gefunden
  if _, err := LoadString(`(set-working-directory "")`, env); err != nil {
    t.Fatalf("reset: %v", err)
  }
  got, err = LoadString(`(file-read "config.txt")`, env)
  if err != nil { t.Fatalf("file-read nach reset: %v", err) }
  if got.Val != "cwd" {
    t.Fatalf("nach reset cwd-Datei erwartet: %q, got %q", "cwd", got.Val)
  }

  // Absoluter Pfad geht immer direkt
  abs := filepath.Join(projekt, "config.txt")
  got, err = LoadString(`(get-file-path "`+abs+`")`, env)
  if err != nil { t.Fatalf("get-file-path abs: %v", err) }
  if got.Type != STRING || got.Val != abs {
    t.Fatalf("get-file-path = %q erwartet, got %q", abs, got.Val)
  }
}

func TestSysStreams(t *testing.T) {
  env := BaseEnv()

  // stdin simulieren
  SetStdinReader(strings.NewReader("zeile1\nzeile2\nrest ohne newline"))
  defer ResetStdinReader()

  // gets liest zeilenweise, teilt den Reader mit slurp/file-read
  got, err := LoadString(`(gets)`, env)
  if err != nil { t.Fatalf("gets 1: %v", err) }
  if got.Type != STRING || got.Val != "zeile1" {
    t.Fatalf("gets = %q erwartet, got %q", "zeile1", got.Val)
  }
  got, err = LoadString(`(gets)`, env)
  if err != nil { t.Fatalf("gets 2: %v", err) }
  if got.Val != "zeile2" {
    t.Fatalf("gets = %q erwartet, got %q", "zeile2", got.Val)
  }

  // slurp liest den Rest bis EOF (Puffer geteilt — nichts verloren)
  got, err = LoadString(`(slurp)`, env)
  if err != nil { t.Fatalf("slurp: %v", err) }
  if got.Val != "rest ohne newline" {
    t.Fatalf("slurp = %q erwartet, got %q", "rest ohne newline", got.Val)
  }

  // file-read "sys-stdin" nach EOF der simulierten Quelle: neu befüllen
  SetStdinReader(strings.NewReader("alles\n"))
  got, err = LoadString(`(file-read "sys-stdin")`, env)
  if err != nil { t.Fatalf("file-read sys-stdin: %v", err) }
  if got.Val != "alles\n" {
    t.Fatalf("file-read sys-stdin = %q erwartet, got %q", "alles\n", got.Val)
  }

  // stdout/stderr abfangen
  var outBuf, errBuf strings.Builder
  SetOutputWriter(func(s string) error { outBuf.WriteString(s); return nil })
  SetErrorWriter(func(s string) error { errBuf.WriteString(s); return nil })
  defer ResetOutputWriter()
  defer ResetErrorWriter()

  if _, err := LoadString(`(file-write "sys-stdout" "hallo" " " "welt")`, env); err != nil {
    t.Fatalf("file-write sys-stdout: %v", err)
  }
  if outBuf.String() != "hallo welt" {
    t.Fatalf("sys-stdout = %q erwartet, got %q", "hallo welt", outBuf.String())
  }

  if _, err := LoadString(`(file-append "sys-stderr" "fehler!")`, env); err != nil {
    t.Fatalf("file-append sys-stderr: %v", err)
  }
  if errBuf.String() != "fehler!" {
    t.Fatalf("sys-stderr = %q erwartet, got %q", "fehler!", errBuf.String())
  }

  if _, err := LoadString(`(err-write "oops" 42)`, env); err != nil {
    t.Fatalf("err-write: %v", err)
  }
  if errBuf.String() != "fehler!oops42" {
    t.Fatalf("err-write akkumuliert = %q erwartet, got %q", "fehler!oops42", errBuf.String())
  }
}
