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
