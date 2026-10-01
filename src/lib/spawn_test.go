//**********************************************************************
//  lib/spawn_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261001
//**********************************************************************

package lib

import (
  "testing"
  "time"
)

func spawnEnv(t *testing.T) *Env {
  t.Helper()
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("LoadStdlib: %v", err)
  }
  return env
}

func TestSpawnRunsFn(t *testing.T) {
  env := spawnEnv(t)
  res, err := LoadString(`(define ch (chan-make 1))
(spawn (lambda () (chan-send ch 42)))
(chan-recv ch)`, env)
  if err != nil {
    t.Fatalf("LoadString: %v", err)
  }
  if res.Num != 42 {
    t.Fatalf("got %s, want 42", res)
  }
}

func TestSpawnReturnsImmediately(t *testing.T) {
  env := spawnEnv(t)
  start := time.Now()
  res, err := LoadString(`(define ch (chan-make))
(spawn (lambda () (chan-recv ch)))`, env)
  if err != nil {
    t.Fatalf("LoadString: %v", err)
  }
  if res.Type != NIL {
    t.Fatalf("spawn-Rückgabe = %s, want ()", res)
  }
  if d := time.Since(start); d > time.Second {
    t.Fatalf("spawn blockierte %v", d)
  }
  // Goroutine freigeben
  if _, err := LoadString(`(chan-send ch 1)`, env); err != nil {
    t.Fatalf("chan-send: %v", err)
  }
}

func TestSpawnErrorDoesNotKill(t *testing.T) {
  env := spawnEnv(t)
  res, err := LoadString(`(define ch (chan-make 1))
(spawn (lambda () (error "kaputt")))
(spawn (lambda () (chan-send ch 7)))
(chan-recv ch)`, env)
  if err != nil {
    t.Fatalf("LoadString: %v", err)
  }
  if res.Num != 7 {
    t.Fatalf("got %s, want 7", res)
  }
}

func TestSpawnArgErrors(t *testing.T) {
  env := spawnEnv(t)
  for _, src := range []string{`(spawn)`, `(spawn 1)`, `(spawn (lambda () 1) 2)`} {
    if _, err := LoadString(src, env); err == nil {
      t.Errorf("%s: kein Fehler, want Fehler", src)
    }
  }
}
