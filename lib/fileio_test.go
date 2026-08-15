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
