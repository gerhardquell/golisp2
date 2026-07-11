//**********************************************************************
//  lib/shm_lisp_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260711
//**********************************************************************
// Tests fuer Shared-Memory Primitiven (Task 1: shm-alloc / shm-free)
//**********************************************************************

package lib

import (
  "testing"
)

func TestShmAllocFree(t *testing.T) {
  env := BaseEnv()
  allocFn, err := env.Get("shm-alloc")
  if err != nil {
    t.Fatal("shm-alloc nicht registriert")
  }
  handle, err := allocFn.Fn(nil)
  if err != nil {
    t.Fatalf("shm-alloc fehlgeschlagen: %v", err)
  }
  if handle == nil || handle.Val != "shm-block" {
    t.Fatalf("erwartet shm-block handle, got %v", handle)
  }

  freeFn, _ := env.Get("shm-free")
  _, err = freeFn.Fn([]*Cell{handle})
  if err != nil {
    t.Fatalf("shm-free fehlgeschlagen: %v", err)
  }
}

func TestShmWriteRead(t *testing.T) {
  env := BaseEnv()
  allocFn, _ := env.Get("shm-alloc")
  handle, _ := allocFn.Fn(nil)
  defer func() {
    freeFn, _ := env.Get("shm-free")
    freeFn.Fn([]*Cell{handle})
  }()

  writeFn, _ := env.Get("shm-write")
  readFn, _ := env.Get("shm-read")

  input := MakeStr("Hallo SHM")
  _, err := writeFn.Fn([]*Cell{handle, input})
  if err != nil {
    t.Fatalf("shm-write fehlgeschlagen: %v", err)
  }

  out, err := readFn.Fn([]*Cell{handle})
  if err != nil {
    t.Fatalf("shm-read fehlgeschlagen: %v", err)
  }
  if out.Type != STRING || out.Val != "Hallo SHM" {
    t.Fatalf("erwartet 'Hallo SHM', got %v", out)
  }
}

func TestShmReadExplicitLength(t *testing.T) {
  env := BaseEnv()
  allocFn, _ := env.Get("shm-alloc")
  handle, _ := allocFn.Fn(nil)
  defer func() {
    freeFn, _ := env.Get("shm-free")
    freeFn.Fn([]*Cell{handle})
  }()

  writeFn, _ := env.Get("shm-write")
  readFn, _ := env.Get("shm-read")

  writeFn.Fn([]*Cell{handle, MakeStr("ABCDEFGH")})
  out, err := readFn.Fn([]*Cell{handle, MakeNum(4)})
  if err != nil {
    t.Fatalf("shm-read mit Länge fehlgeschlagen: %v", err)
  }
  if out.Val != "ABCD" {
    t.Fatalf("erwartet 'ABCD', got %v", out)
  }
}

func TestShmStatusCleanup(t *testing.T) {
  env := BaseEnv()
  statusFn, _ := env.Get("shm-status")
  cleanupFn, _ := env.Get("shm-cleanup")
  allocFn, _ := env.Get("shm-alloc")

  _, err := statusFn.Fn(nil)
  if err != nil {
    t.Fatalf("shm-status fehlgeschlagen: %v", err)
  }
  _, err = cleanupFn.Fn(nil)
  if err != nil {
    t.Fatalf("shm-cleanup fehlgeschlagen: %v", err)
  }

  // Nach Cleanup müssen weitere SHM-Operationen sauber fehlschlagen.
  _, err = allocFn.Fn(nil)
  if err == nil {
    t.Fatal("erwartet Fehler für shm-alloc nach cleanup")
  }
}

func TestShmInvalidHandle(t *testing.T) {
  env := BaseEnv()
  writeFn, _ := env.Get("shm-write")
  _, err := writeFn.Fn([]*Cell{MakeNum(42), MakeStr("x")})
  if err == nil {
    t.Fatal("erwartet Fehler für ungültigen Handle")
  }
}
