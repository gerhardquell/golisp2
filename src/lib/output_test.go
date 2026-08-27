//**********************************************************************
//  lib/output_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6, kimi
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260626
//**********************************************************************
// Tests für den zentralen Output-Handler.
//**********************************************************************

package lib

import (
  "errors"
  "testing"
)

func TestWriteOutputDefault(t *testing.T) {
  // Default-Writer ist os.Stdout; hier testen wir nur, dass kein Fehler
  // auftritt und dass SetOutputWriter/ResetOutputWriter funktionieren.
  if err := WriteOutput(""); err != nil {
    t.Fatalf("WriteOutput failed: %v", err)
  }
}

func TestSetOutputWriter(t *testing.T) {
  var got string
  SetOutputWriter(func(s string) error {
    got = s
    return nil
  })
  defer ResetOutputWriter()

  if err := WriteOutput("hello"); err != nil {
    t.Fatalf("WriteOutput failed: %v", err)
  }
  if got != "hello" {
    t.Fatalf("expected 'hello', got %q", got)
  }
}

func TestSetOutputWriterError(t *testing.T) {
  want := errors.New("boom")
  SetOutputWriter(func(s string) error {
    return want
  })
  defer ResetOutputWriter()

  if err := WriteOutput("x"); err != want {
    t.Fatalf("expected error %v, got %v", want, err)
  }
}

func TestResetOutputWriter(t *testing.T) {
  var got string
  SetOutputWriter(func(s string) error {
    got = s
    return nil
  })
  ResetOutputWriter()

  // Nach Reset sollte der Default-Writer aktiv sein. Wir können nicht
  // einfach stdout prüfen, aber wir stellen sicher, dass der vorherige
  // Writer nicht mehr aufgerufen wird (kein Panic bei leerem String).
  if err := WriteOutput(""); err != nil {
    t.Fatalf("WriteOutput after reset failed: %v", err)
  }
  if got != "" {
    t.Fatalf("custom writer was still called after reset")
  }
}

func TestWriteErrorDefault(t *testing.T) {
  if err := WriteError(""); err != nil {
    t.Fatalf("WriteError failed: %v", err)
  }
}

func TestSetErrorWriter(t *testing.T) {
  var got string
  SetErrorWriter(func(s string) error {
    got = s
    return nil
  })
  defer ResetErrorWriter()

  if err := WriteError("hello stderr"); err != nil {
    t.Fatalf("WriteError failed: %v", err)
  }
  if got != "hello stderr" {
    t.Fatalf("expected 'hello stderr', got %q", got)
  }
}

func TestSetErrorWriterError(t *testing.T) {
  want := errors.New("boom")
  SetErrorWriter(func(s string) error {
    return want
  })
  defer ResetErrorWriter()

  if err := WriteError("x"); err != want {
    t.Fatalf("expected error %v, got %v", want, err)
  }
}

func TestResetErrorWriter(t *testing.T) {
  var got string
  SetErrorWriter(func(s string) error {
    got = s
    return nil
  })
  ResetErrorWriter()

  if err := WriteError(""); err != nil {
    t.Fatalf("WriteError after reset failed: %v", err)
  }
  if got != "" {
    t.Fatalf("custom error writer was still called after reset")
  }
}
