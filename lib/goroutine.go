//**********************************************************************
//  lib/goroutine.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260224
//**********************************************************************
// Go-Nebenläufigkeit als Lisp-Primitiven:
//   (parfunc ergebnis expr1 expr2 ...)  → parallel auswerten
//   (chan-make)                          → neuen Channel erstellen
//   (chan-send ch wert)                  → Wert senden
//   (chan-recv ch)                       → Wert empfangen
//   (lock-make)                          → neuen Mutex erstellen
//   (lock mu expr)                       → atomar ausführen
//**********************************************************************

package lib

import (
  "fmt"
  "sync"
)

// goChannel: Go-Channel verpackt in Cell
type goChannel struct {
  ch chan *Cell
}

// goMutex: Go-Mutex verpackt in Cell
type goMutex struct {
  mu sync.Mutex
}

// makeChannelCell erstellt eine Cell die einen Channel hält
func makeChannelCell(size int) *Cell {
  gc := &goChannel{ch: make(chan *Cell, size)}
  return &Cell{
    Type: FUNC,
    Val:  "channel",
    Fn: func(_ []*Cell) (*Cell, error) {
      return (*Cell)(nil), nil  // Platzhalter
    },
    Env: gc,
  }
}

// makeLockelCell erstellt eine Cell die einen Mutex hält
func makeMutexCell() *Cell {
  gm := &goMutex{}
  return &Cell{
    Type: FUNC,
    Val:  "mutex",
    Fn: func(_ []*Cell) (*Cell, error) {
      return (*Cell)(nil), nil
    },
    Env: gm,
  }
}

// getChannel extrahiert goChannel aus einer Cell
func getChannel(c *Cell) (*goChannel, error) {
  if c == nil || c.Val != "channel" {
    return nil, fmt.Errorf("kein Channel")
  }
  gc, ok := c.Env.(*goChannel)
  if !ok { return nil, fmt.Errorf("kein Channel") }
  return gc, nil
}

// getMutex extrahiert goMutex aus einer Cell
func getMutex(c *Cell) (*goMutex, error) {
  if c == nil || c.Val != "mutex" {
    return nil, fmt.Errorf("kein Mutex")
  }
  gm, ok := c.Env.(*goMutex)
  if !ok { return nil, fmt.Errorf("kein Mutex") }
  return gm, nil
}

// RegisterGoroutines fügt alle Nebenläufigkeits-Primitiven in env ein
func RegisterGoroutines(env *Env) {
  _ = env.Set("chan-make", makeFn(fnChanMake))
  _ = env.Set("chan-send", makeFn(fnChanSend))
  _ = env.Set("chan-recv", makeFn(fnChanRecv))
  _ = env.Set("lock-make", makeFn(fnLockMake))
}

// ---- Channel-Funktionen ----

// chan-make: (chan-make) oder (chan-make 10) für gebufferten Channel
func fnChanMake(args []*Cell) (*Cell, error) {
  size := 0
  if len(args) > 0 { size = int(args[0].Num) }
  return makeChannelCell(size), nil
}

// chan-send: (chan-send ch wert)
func fnChanSend(args []*Cell) (*Cell, error) {
  if len(args) < 2 { return nil, fmt.Errorf("chan-send: 2 Argumente nötig") }
  gc, err := getChannel(args[0])
  if err != nil { return nil, err }
  gc.ch <- args[1]
  return args[1], nil
}

// chan-recv: (chan-recv ch)
func fnChanRecv(args []*Cell) (*Cell, error) {
  if len(args) < 1 { return nil, fmt.Errorf("chan-recv: 1 Argument nötig") }
  gc, err := getChannel(args[0])
  if err != nil { return nil, err }
  val := <-gc.ch
  return val, nil
}

// ---- Mutex ----

// lock-make: (lock-make)
func fnLockMake(args []*Cell) (*Cell, error) {
  return makeMutexCell(), nil
}
