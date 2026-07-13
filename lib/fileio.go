//**********************************************************************
//  lib/fileio.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260224
//**********************************************************************
// Datei-I/O Primitiven:
//   (file-write "datei.txt" "inhalt")   → schreibt/überschreibt
//   (file-append "datei.txt" "inhalt")  → hängt an
//   (file-read "datei.txt")             → liest als String
//   (file-exists? "datei.txt")          → t oder nil
//   (file-delete "datei.txt")           → löscht
//**********************************************************************

package lib

import (
  "fmt"
  "os"
  "strings"
)

// RegisterFileIO fügt alle Datei-Funktionen in env ein
func RegisterFileIO(env *Env) {
  _ = env.Set("file-write",   makeFn(fnFileWrite))
  _ = env.Set("file-append",  makeFn(fnFileAppend))
  _ = env.Set("file-read",    makeFn(fnFileRead))
  _ = env.Set("file-exists?", makeFn(fnFileExists))
  _ = env.Set("file-delete",  makeFn(fnFileDelete))
}

// file-write: (file-write "datei.txt" "inhalt" ...)
// Mehrere Strings werden zusammengefügt
func fnFileWrite(args []*Cell) (*Cell, error) {
  if len(args) < 2 { return nil, fmt.Errorf("file-write: mindestens 2 Argumente") }
  filename := args[0].Val
  content  := joinStrings(args[1:])
  if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
    return nil, fmt.Errorf("file-write '%s': %v", filename, err)
  }
  return MakeStr(filename), nil
}

// file-append: (file-append "datei.txt" "inhalt")
func fnFileAppend(args []*Cell) (*Cell, error) {
  if len(args) < 2 { return nil, fmt.Errorf("file-append: mindestens 2 Argumente") }
  filename := args[0].Val
  content  := joinStrings(args[1:])

  f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
  if err != nil { return nil, fmt.Errorf("file-append '%s': %v", filename, err) }
  defer f.Close()

  if _, err := f.WriteString(content); err != nil {
    return nil, fmt.Errorf("file-append write: %v", err)
  }
  return MakeStr(filename), nil
}

// file-read: (file-read "datei.txt") → String mit Inhalt
func fnFileRead(args []*Cell) (*Cell, error) {
  if len(args) < 1 { return nil, fmt.Errorf("file-read: 1 Argument nötig") }
  data, err := os.ReadFile(args[0].Val)
  if err != nil { return nil, fmt.Errorf("file-read '%s': %v", args[0].Val, err) }
  return MakeStr(string(data)), nil
}

// file-exists?: (file-exists? "datei.txt") → t oder nil
func fnFileExists(args []*Cell) (*Cell, error) {
  if len(args) < 1 { return nil, fmt.Errorf("file-exists?: 1 Argument nötig") }
  if _, err := os.Stat(args[0].Val); err == nil {
    return MakeAtom("t"), nil
  }
  return MakeNil(), nil
}

// file-delete: (file-delete "datei.txt")
func fnFileDelete(args []*Cell) (*Cell, error) {
  if len(args) < 1 { return nil, fmt.Errorf("file-delete: 1 Argument nötig") }
  if err := os.Remove(args[0].Val); err != nil {
    return nil, fmt.Errorf("file-delete '%s': %v", args[0].Val, err)
  }
  return MakeAtom("t"), nil
}

// joinStrings verbindet mehrere Cell-Werte zu einem String
func joinStrings(args []*Cell) string {
  var sb strings.Builder
  for _, a := range args {
    switch a.Type {
    case STRING: sb.WriteString(a.Val)
    case NUMBER: sb.WriteString(a.String())
    default:     sb.WriteString(a.String())
    }
  }
  return sb.String()
}
