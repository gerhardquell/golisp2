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
//   (set-working-directory "./projekt")  → Arbeitsverzeichnis setzen
//   (get-working-directory)              → aktuelles Arbeitsverzeichnis oder ()
//   (get-file-path "datei.txt")          → erster existierender Pfad
//
// Suchreihenfolge beim Lesen:
//   1. Absolute Pfade (immer Vorrang)
//   2. working-directory (falls gesetzt)
//   3. Datei wie angegeben (relativ zum Prozess-Verzeichnis)
//   4. GOLISP_PATH + Standard-Library-Verzeichnisse
//
// Beim Schreiben (file-write/file-append):
//   1. Absolute Pfade unverändert
//   2. working-directory + Dateiname (falls gesetzt)
//   3. Datei wie angegeben
//
// Pseudodateien (vor jeder Pfadauflösung erkannt):
//   "sys-stdin"  → lesen vom gemeinsamen stdinReader (wie read-line)
//   "sys-stdout" → schreiben über WriteOutput (SWANK-sichtbar)
//   "sys-stderr" → schreiben über WriteError
//
// Stream-Primitiven (C-Emulation):
//   (gets)               → eine Zeile von stdin (fgets)
//   (slurp)              → stdin bis EOF (fread bis Ende)
//   (err-write "text")   → auf stderr schreiben (fwrite)
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
  // workingDirectory: benutzerdefiniertes Arbeitsverzeichnis für relative
  // Dateinamen (leer = nicht gesetzt).
  workingDirectory   string
  workingDirectoryMu sync.RWMutex

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

  _ = env.Set("set-working-directory", makeFn(fnSetWorkingDirectory))
  _ = env.Set("get-working-directory", makeFn(fnGetWorkingDirectory))
  _ = env.Set("get-file-path",         makeFn(fnGetFilePath))

  _ = env.Set("gets",      makeFn(fnGets))
  _ = env.Set("slurp",     makeFn(fnSlurp))
  _ = env.Set("err-write", makeFn(fnErrWrite))
}

// Pseudodateinamen für die Standard-Streams.
const (
  sysStdin  = "sys-stdin"
  sysStdout = "sys-stdout"
  sysStderr = "sys-stderr"
)

// file-write: (file-write "datei.txt" "inhalt" ...)
// Mehrere Strings werden zusammengefügt
func fnFileWrite(args []*Cell) (*Cell, error) {
  if len(args) < 2 { return nil, fmt.Errorf("file-write: mindestens 2 Argumente") }
  if args[0].Val == sysStdout || args[0].Val == sysStderr {
    return writeToSysStream(args[0].Val, joinStrings(args[1:]))
  }
  filename := resolveWritePath(args[0].Val)
  content  := joinStrings(args[1:])
  if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
    return nil, fmt.Errorf("file-write '%s': %v", filename, err)
  }
  return MakeStr(filename), nil
}

// file-append: (file-append "datei.txt" "inhalt")
func fnFileAppend(args []*Cell) (*Cell, error) {
  if len(args) < 2 { return nil, fmt.Errorf("file-append: mindestens 2 Argumente") }
  if args[0].Val == sysStdout || args[0].Val == sysStderr {
    return writeToSysStream(args[0].Val, joinStrings(args[1:]))
  }
  filename := resolveWritePath(args[0].Val)
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
// Pseudodatei "sys-stdin": liest stdin bis EOF.
func fnFileRead(args []*Cell) (*Cell, error) {
  if len(args) < 1 { return nil, fmt.Errorf("file-read: 1 Argument nötig") }
  if args[0].Type != STRING { return nil, fmt.Errorf("file-read: String erwartet") }
  if args[0].Val == sysStdin {
    s, err := slurpStdin()
    if err != nil { return nil, fmt.Errorf("file-read '%s': %v", sysStdin, err) }
    return MakeStr(s), nil
  }
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
  resolved, err := resolvePath(args[0].Val)
  if err != nil { return nil, fmt.Errorf("file-delete '%s': %v", args[0].Val, err) }
  if err := os.Remove(resolved); err != nil {
    return nil, fmt.Errorf("file-delete '%s': %v", resolved, err)
  }
  return MakeAtom("t"), nil
}

// set-working-directory:
//   (set-working-directory "./projekt")  → Arbeitsverzeichnis setzen
//   (set-working-directory "")           → zurücksetzen
// Fehler, wenn der Pfad kein existierendes Verzeichnis ist.
func fnSetWorkingDirectory(args []*Cell) (*Cell, error) {
  if len(args) != 1 { return nil, fmt.Errorf("set-working-directory: 1 Argument nötig") }
  if args[0].Type != STRING { return nil, fmt.Errorf("set-working-directory: String erwartet") }
  dir := args[0].Val
  if dir != "" {
    info, err := os.Stat(dir)
    if err != nil || !info.IsDir() {
      return nil, fmt.Errorf("set-working-directory '%s': kein existierendes Verzeichnis", dir)
    }
  }
  workingDirectoryMu.Lock()
  workingDirectory = dir
  workingDirectoryMu.Unlock()
  return MakeAtom("t"), nil
}

// get-working-directory → aktuelles Arbeitsverzeichnis oder ()
func fnGetWorkingDirectory(args []*Cell) (*Cell, error) {
  workingDirectoryMu.RLock()
  dir := workingDirectory
  workingDirectoryMu.RUnlock()
  if dir == "" { return MakeNil(), nil }
  return MakeStr(dir), nil
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
// Nur für Lese-Zugriffe: die Datei muss existieren.
func resolvePath(filename string) (string, error) {
  // 1. Absolute Pfade haben immer Vorrang.
  if filepath.IsAbs(filename) {
    if _, err := os.Stat(filename); err == nil {
      return filename, nil
    }
    return "", fmt.Errorf("'%s' nicht gefunden", filename)
  }

  // 2. working-directory (falls gesetzt)
  workingDirectoryMu.RLock()
  wd := workingDirectory
  workingDirectoryMu.RUnlock()
  if wd != "" {
    full := filepath.Join(wd, filename)
    if _, err := os.Stat(full); err == nil {
      return full, nil
    }
  }

  // 3. Relativer Pfad wie angegeben (Prozess-Verzeichnis).
  if _, err := os.Stat(filename); err == nil {
    return filename, nil
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

// resolveWritePath löst einen Dateinamen für Schreib-Zugriffe auf.
// Keine Existenzprüfung: die Datei darf neu sein.
func resolveWritePath(filename string) string {
  if filepath.IsAbs(filename) { return filename }
  workingDirectoryMu.RLock()
  wd := workingDirectory
  workingDirectoryMu.RUnlock()
  if wd != "" { return filepath.Join(wd, filename) }
  return filename
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

// writeToSysStream schreibt content auf sys-stdout/sys-stderr über die
// Output-Chokepoints (SWANK-sichtbar).
func writeToSysStream(name, content string) (*Cell, error) {
  var err error
  if name == sysStderr {
    err = WriteError(content)
  } else {
    err = WriteOutput(content)
  }
  if err != nil { return nil, fmt.Errorf("file-write '%s': %v", name, err) }
  return MakeStr(name), nil
}

// gets: (gets) → eine Zeile von stdin ohne Newline (C fgets).
// Teilt sich den stdinReader mit read-line.
func fnGets(args []*Cell) (*Cell, error) {
  if len(args) != 0 { return nil, fmt.Errorf("gets: keine Argumente erwartet") }
  line, err := readLineFromStdin()
  if err != nil { return nil, fmt.Errorf("gets: %w", err) }
  return MakeStr(line), nil
}

// slurp: (slurp) → stdin bis EOF als String (C fread bis Ende).
func fnSlurp(args []*Cell) (*Cell, error) {
  if len(args) != 0 { return nil, fmt.Errorf("slurp: keine Argumente erwartet") }
  s, err := slurpStdin()
  if err != nil { return nil, fmt.Errorf("slurp: %w", err) }
  return MakeStr(s), nil
}

// err-write: (err-write "text" ...) → Strings auf stderr (C fwrite(stderr)).
func fnErrWrite(args []*Cell) (*Cell, error) {
  if len(args) < 1 { return nil, fmt.Errorf("err-write: mindestens 1 Argument") }
  if err := WriteError(joinStrings(args)); err != nil {
    return nil, fmt.Errorf("err-write: %v", err)
  }
  return MakeAtom("t"), nil
}
