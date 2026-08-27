//**********************************************************************
//  lib/output.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6, kimi
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260626
//**********************************************************************
// Zentraler Output-Handler. Standardmäßig wird nach os.Stdout geschrieben.
// SWANK kann den Handler auf eine Funktion setzen, die :write-string-Events
// an Emacs sendet. Dadurch sehen alle Output-Primitive (print, println,
// format, ga-print) ihre Ausgaben auch in SLIME.
//**********************************************************************

package lib

import (
  "os"
  "sync"
)

// OutputWriter schreibt eine Zeichenkette. Der Rückgabewert folgt dem
// üblichen Go-error-Muster.
type OutputWriter func(string) error

var (
  outputMu     sync.Mutex
  outputWriter OutputWriter = defaultOutputWriter
  errorWriter  OutputWriter = defaultErrorWriter
)

func defaultOutputWriter(s string) error {
  _, err := os.Stdout.WriteString(s)
  return err
}

func defaultErrorWriter(s string) error {
  _, err := os.Stderr.WriteString(s)
  return err
}

// WriteOutput schreibt s über den aktuell gesetzten OutputWriter.
func WriteOutput(s string) error {
  outputMu.Lock()
  w := outputWriter
  outputMu.Unlock()
  return w(s)
}

// WriteError schreibt s über den aktuell gesetzten ErrorWriter (stderr).
func WriteError(s string) error {
  outputMu.Lock()
  w := errorWriter
  outputMu.Unlock()
  return w(s)
}

// SetOutputWriter setzt einen neuen OutputWriter. Thread-sicher.
func SetOutputWriter(w OutputWriter) {
  outputMu.Lock()
  defer outputMu.Unlock()
  outputWriter = w
}

// SetErrorWriter setzt einen neuen ErrorWriter. Thread-sicher.
func SetErrorWriter(w OutputWriter) {
  outputMu.Lock()
  defer outputMu.Unlock()
  errorWriter = w
}

// ResetOutputWriter stellt den Standard-Writer (os.Stdout) wieder her.
func ResetOutputWriter() {
  outputMu.Lock()
  defer outputMu.Unlock()
  outputWriter = defaultOutputWriter
}

// ResetErrorWriter stellt den Standard-ErrorWriter (os.Stderr) wieder her.
func ResetErrorWriter() {
  outputMu.Lock()
  defer outputMu.Unlock()
  errorWriter = defaultErrorWriter
}
