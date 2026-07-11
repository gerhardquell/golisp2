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
