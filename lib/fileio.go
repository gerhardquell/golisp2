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
//
// Pfad-Auflösung:
//   (set-working-directories "./lib:./test")  → Suchpfad setzen
//   (set-working-directories '("./lib" "./test"))
//   (get-working-directories)                 → Liste der Suchpfade
//   (get-file-path "datei.txt")               → erstes existierendes Verzeichnis
//
// Suchreihenfolge:
//   1. Absolute Pfade (immer Vorrang)
//   2. Datei wie angegeben (relativ zum aktuellen Verzeichnis)
//   3. working-directories
//   4. GOLISP_PATH + Standard-Library-Verzeichnisse
//**********************************************************************

package lib

import (
  "fmt"
  "os"
  "path/filepath"
  "strings"
  "sync"
)

var (
  // workingDirectories: benutzerdefinierte Suchpfade für relative Dateinamen.
  workingDirectories   []string
  workingDirectoriesMu sync.RWMutex

  // librarySearchPaths: systemweite Library-Pfade (GOLISP_PATH, /lib/golib, ...).
  librarySearchPaths     []string
  librarySearchPathsOnce sync.Once
)

// RegisterFileIO fügt alle Datei-Funktionen in env ein
func RegisterFileIO(env *Env) {
  _ = env.Set("file-write",   makeFn(fnFileWrite))
  _ = env.Set("file-append",  makeFn(fnFileAppend))
  _ = env.Set("file-read",    makeFn(fnFileRead))
  _ = env.Set("file-exists?", makeFn(fnFileExists))
  _ = env.Set("file-delete",  makeFn(fnFileDelete))

  _ = env.Set("set-working-directories", makeFn(fnSetWorkingDirectories))
  _ = env.Set("get-working-directories", makeFn(fnGetWorkingDirectories))
  _ = env.Set("get-file-path",           makeFn(fnGetFilePath))
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
  if args[0].Type != STRING { return nil, fmt.Errorf("file-read: String erwartet") }
  resolved, err := resolvePath(args[0].Val)
  if err != nil { return nil, fmt.Errorf("file-read '%s': %v", args[0].Val, err) }
  data, err := os.ReadFile(resolved)
  if err != nil { return nil, fmt.Errorf("file-read '%s': %v", resolved, err) }
  return MakeStr(string(data)), nil
}

// file-exists?: (file-exists? "datei.txt") → t oder nil
func fnFileExists(args []*Cell) (*Cell, error) {
  if len(args) < 1 { return nil, fmt.Errorf("file-exists?: 1 Argument nötig") }
  if args[0].Type != STRING { return nil, fmt.Errorf("file-exists?: String erwartet") }
  if _, err := resolvePath(args[0].Val); err == nil {
    return MakeAtom("t"), nil
  }
  return MakeNil(), nil
}

// file-delete: (file-delete "datei.txt")
func fnFileDelete(args []*Cell) (*Cell, error) {
  if len(args) < 1 { return nil, fmt.Errorf("file-delete: 1 Argument nötig") }
  if args[0].Type != STRING { return nil, fmt.Errorf("file-delete: String erwartet") }
  if err := os.Remove(args[0].Val); err != nil {
    return nil, fmt.Errorf("file-delete '%s': %v", args[0].Val, err)
  }
  return MakeAtom("t"), nil
}

// set-working-directories:
//   (set-working-directories "./lib:./test")      → String mit ':'
//   (set-working-directories '("./lib" "./test")) → Liste von Strings
func fnSetWorkingDirectories(args []*Cell) (*Cell, error) {
  if len(args) != 1 { return nil, fmt.Errorf("set-working-directories: 1 Argument nötig") }
  dirs, err := cellToPathList(args[0])
  if err != nil { return nil, fmt.Errorf("set-working-directories: %v", err) }
  workingDirectoriesMu.Lock()
  workingDirectories = dirs
  workingDirectoriesMu.Unlock()
  return MakeAtom("t"), nil
}

// get-working-directories → Liste der aktuellen Suchpfade
func fnGetWorkingDirectories(args []*Cell) (*Cell, error) {
  workingDirectoriesMu.RLock()
  dirs := append([]string(nil), workingDirectories...)
  workingDirectoriesMu.RUnlock()
  cells := make([]*Cell, len(dirs))
  for i, d := range dirs { cells[i] = MakeStr(d) }
  return SliceToCell(cells), nil
}

// get-file-path: (get-file-path "datei.txt") → erster existierender Pfad
func fnGetFilePath(args []*Cell) (*Cell, error) {
  if len(args) != 1 { return nil, fmt.Errorf("get-file-path: 1 Argument nötig") }
  if args[0].Type != STRING { return nil, fmt.Errorf("get-file-path: String erwartet") }
  resolved, err := resolvePath(args[0].Val)
  if err != nil { return nil, fmt.Errorf("get-file-path '%s': %v", args[0].Val, err) }
  return MakeStr(resolved), nil
}

// resolvePath löst einen Dateinamen nach der dokumentierten Reihenfolge auf.
func resolvePath(filename string) (string, error) {
  // 1. Absolute Pfade haben immer Vorrang.
  if filepath.IsAbs(filename) {
    if _, err := os.Stat(filename); err == nil {
      return filename, nil
    }
    return "", fmt.Errorf("'%s' nicht gefunden", filename)
  }

  // 2. Relativer Pfad wie angegeben (aktuelles Verzeichnis).
  if _, err := os.Stat(filename); err == nil {
    return filename, nil
  }

  // 3. working-directories
  workingDirectoriesMu.RLock()
  dirs := append([]string(nil), workingDirectories...)
  workingDirectoriesMu.RUnlock()
  for _, dir := range dirs {
    full := filepath.Join(dir, filename)
    if _, err := os.Stat(full); err == nil {
      return full, nil
    }
  }

  // 4. Systemweite Library-Suchpfade
  for _, dir := range getLibrarySearchPaths() {
    full := filepath.Join(dir, filename)
    if _, err := os.Stat(full); err == nil {
      return full, nil
    }
  }

  return "", fmt.Errorf("'%s' nicht gefunden in Suchpfaden", filename)
}

// getLibrarySearchPaths liefert die einmalig initialisierten Library-Pfade.
func getLibrarySearchPaths() []string {
  librarySearchPathsOnce.Do(func() {
    librarySearchPaths = initLibrarySearchPaths()
  })
  return librarySearchPaths
}

// initLibrarySearchPaths baut die systemweiten Suchpfade auf.
func initLibrarySearchPaths() []string {
  var paths []string
  paths = append(paths, "/lib/golib")
  paths = append(paths, "/usr/local/lib/golib")
  paths = append(paths, "./golib")
  if golispPath := os.Getenv("GOLISP_PATH"); golispPath != "" {
    for _, p := range strings.Split(golispPath, ":") {
      if p != "" {
        paths = append(paths, p)
      }
    }
  }
  return paths
}

// cellToPathList wandelt String oder Liste von Strings in []string um.
func cellToPathList(c *Cell) ([]string, error) {
  if c == nil { return nil, fmt.Errorf("nil-Wert ungültig") }
  switch c.Type {
  case STRING:
    return splitPathString(c.Val), nil
  case LIST:
    var dirs []string
    for lst := c; lst != nil && lst.Type == LIST; lst = lst.Cdr {
      if lst.Car == nil || lst.Car.Type != STRING {
        return nil, fmt.Errorf("Liste muss Strings enthalten")
      }
      dirs = append(dirs, lst.Car.Val)
    }
    return dirs, nil
  case NIL:
    return nil, nil
  default:
    return nil, fmt.Errorf("String oder Liste erwartet")
  }
}

// splitPathString teilt einen ':'-separierten Pfad-String.
func splitPathString(s string) []string {
  parts := strings.Split(s, ":")
  var dirs []string
  for _, p := range parts {
    if p != "" { dirs = append(dirs, p) }
  }
  return dirs
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
