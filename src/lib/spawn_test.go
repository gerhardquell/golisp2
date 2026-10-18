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
  if Primary(res).Num != 42 {
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
  if _, err := getChannel(Primary(res)); err != nil {
    t.Fatalf("spawn-Rückgabe = %s, want Channel", res)
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
  if Primary(res).Num != 7 {
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

func TestSpawnResultOk(t *testing.T) {
  evalEq(t, `(chan-recv (spawn (lambda () 42)))`, "(:ok 42)")
}

func TestSpawnResultOkIsPrimary(t *testing.T) {
  evalEq(t, `(chan-recv (spawn (lambda () (values 1 2))))`, "(:ok 1)")
}

func TestSpawnResultError(t *testing.T) {
  evalEq(t, `(chan-recv (spawn (lambda () (error "kaputt"))))`, `(:error "kaputt")`)
}

func TestSpawnResultGoError(t *testing.T) {
  evalEq(t, `(chan-recv (spawn (lambda () (car 1))))`, `(:error "car: Liste erwartet")`)
}

func TestSpawnResultPanic(t *testing.T) {
  env := spawnEnv(t)
  _ = env.Set("boom", makeFn(func(_ []*Cell) (*Cell, error) { panic("bumm") }))
  res, err := LoadString(`(chan-recv (spawn boom))`, env)
  if err != nil {
    t.Fatalf("LoadString: %v", err)
  }
  if got, want := Primary(res).String(), `(:error "panic: bumm")`; got != want {
    t.Fatalf("got %s, want %s", got, want)
  }
}

func TestSpawnResultOnlyOnce(t *testing.T) {
  evalEq(t, `(define ch (spawn (lambda () 1)))
    (chan-recv ch)
    (multiple-value-list (chan-try-recv ch))`, "(() ())")
}
