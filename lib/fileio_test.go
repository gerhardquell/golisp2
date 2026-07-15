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

func TestWorkingDirectories(t *testing.T) {
  wd, err := os.Getwd()
  if err != nil { t.Fatalf("getwd: %v", err) }
  defer os.Chdir(wd)

  dir := t.TempDir()
  libDir := filepath.Join(dir, "lib")
  if err := os.MkdirAll(libDir, 0755); err != nil { t.Fatalf("mkdir: %v", err) }
  if err := os.WriteFile(filepath.Join(libDir, "config.txt"), []byte("ok"), 0644); err != nil {
    t.Fatalf("write config: %v", err)
  }
  if err := os.WriteFile(filepath.Join(libDir, "module.lisp"), []byte("(define module-loaded t)\n"), 0644); err != nil {
    t.Fatalf("write module: %v", err)
  }

  if err := os.Chdir(dir); err != nil { t.Fatalf("chdir: %v", err) }

  // Globalen Zustand am Ende wiederherstellen.
  workingDirectoriesMu.Lock()
  oldDirs := append([]string(nil), workingDirectories...)
  workingDirectoriesMu.Unlock()
  defer func() {
    workingDirectoriesMu.Lock()
    workingDirectories = oldDirs
    workingDirectoriesMu.Unlock()
  }()

  env := BaseEnv()

  // String-Format mit ':'
  if _, err := LoadString(`(set-working-directories "./lib")`, env); err != nil {
    t.Fatalf("set string: %v", err)
  }
  got, err := LoadString(`(get-file-path "config.txt")`, env)
  if err != nil { t.Fatalf("get-file-path: %v", err) }
  if got.Type != STRING || got.Val != "lib/config.txt" {
    t.Fatalf("get-file-path = %q erwartet, got %q", "lib/config.txt", got.Val)
  }

  // file-read über Suchpfad
  got, err = LoadString(`(file-read "config.txt")`, env)
  if err != nil { t.Fatalf("file-read: %v", err) }
  if got.Type != STRING || got.Val != "ok" {
    t.Fatalf("file-read = %q erwartet, got %q", "ok", got.Val)
  }

  // file-exists? über Suchpfad
  got, err = LoadString(`(file-exists? "config.txt")`, env)
  if err != nil { t.Fatalf("file-exists?: %v", err) }
  if got.Type != ATOM || got.Val != "t" {
    t.Fatalf("file-exists? = t erwartet, got %s", got)
  }

  // load über Suchpfad
  if _, err := LoadString(`(load "module.lisp")`, env); err != nil {
    t.Fatalf("load: %v", err)
  }
  got, err = LoadString(`module-loaded`, env)
  if err != nil { t.Fatalf("module-loaded: %v", err) }
  if got.Type != ATOM || got.Val != "t" {
    t.Fatalf("module-loaded = t erwartet, got %s", got)
  }

  // Listen-Format
  if _, err := LoadString(`(set-working-directories '("./lib"))`, env); err != nil {
    t.Fatalf("set list: %v", err)
  }
  got, err = LoadString(`(get-working-directories)`, env)
  if err != nil { t.Fatalf("get list: %v", err) }
  if got.String() != `("./lib")` {
    t.Fatalf("get-working-directories = (\"./lib\") erwartet, got %s", got)
  }

  // cons zum Hinzufügen
  if _, err := LoadString(`(set-working-directories (cons "./lib" '()))`, env); err != nil {
    t.Fatalf("set cons: %v", err)
  }
  got, err = LoadString(`(get-working-directories)`, env)
  if err != nil { t.Fatalf("get cons: %v", err) }
  if got.String() != `("./lib")` {
    t.Fatalf("get-working-directories nach cons = (\"./lib\") erwartet, got %s", got)
  }

  // Aktuelles Verzeichnis gewinnt vor working-directories.
  if err := os.WriteFile(filepath.Join(dir, "config.txt"), []byte("cwd"), 0644); err != nil {
    t.Fatalf("write cwd config: %v", err)
  }
  got, err = LoadString(`(get-file-path "config.txt")`, env)
  if err != nil { t.Fatalf("get-file-path cwd: %v", err) }
  if got.Type != STRING || got.Val != "config.txt" {
    t.Fatalf("get-file-path = config.txt erwartet, got %q", got.Val)
  }

  // Absoluter Pfad wird direkt aufgelöst.
  abs := filepath.Join(dir, "lib", "config.txt")
  got, err = LoadString(`(get-file-path "`+abs+`")`, env)
  if err != nil { t.Fatalf("get-file-path abs: %v", err) }
  if got.Type != STRING || got.Val != abs {
    t.Fatalf("get-file-path = %q erwartet, got %q", abs, got.Val)
  }
}
