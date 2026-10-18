//**********************************************************************
//  lib/goroutine.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260224
//**********************************************************************
// Go-Nebenläufigkeit als Lisp-Primitiven:
//   (parfunc ergebnis expr1 expr2 ...)  → parallel auswerten
//   (spawn fn)                           → fn in neuer Goroutine, sofort zurück
//   (chan-make)                          → neuen Channel erstellen
//   (chan-send ch wert)                  → Wert senden (blockiert)
//   (chan-try-send ch wert)              → t gesendet / nil (voll, kein Empfänger)
//   (chan-recv ch [:timeout sek])        → (values x t) / (values nil nil)
//   (chan-try-recv ch)                   → (values x t) / sofort (values nil nil)
//   (chan-close ch)                      → Channel schließen (t)
//   (chan-closed? ch)                    → t wenn geschlossen und leer
//   (lock-make)                          → neuen Mutex erstellen
//   (lock mu expr)                       → atomar ausführen
//**********************************************************************

package lib

import (
  "fmt"
  "os"
  "sync"
  "sync/atomic"
  "time"
)

// goChannel: Go-Channel verpackt in Cell
type goChannel struct {
  ch     chan *Cell
  closed atomic.Bool
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
  _ = env.Set("chan-try-recv", makeFn(fnChanTryRecv))
  _ = env.Set("chan-try-send", makeFn(fnChanTrySend))
  _ = env.Set("chan-close", makeFn(fnChanClose))
  _ = env.Set("chan-closed?", makeFn(fnChanClosedP))
  _ = env.Set("lock-make", makeFn(fnLockMake))
  _ = env.Set("spawn", makeFn(fnSpawn))
}

// ---- Channel-Funktionen ----

// chan-make: (chan-make) oder (chan-make 10) für gebufferten Channel
func fnChanMake(args []*Cell) (*Cell, error) {
  size := 0
  if len(args) > 0 { size = int(args[0].Num) }
  return makeChannelCell(size), nil
}

// timeoutArg liest die Zahl hinter :timeout (Sekunden, >= 0)
func timeoutArg(c *Cell) (time.Duration, bool) {
  if c == nil || c.Type != NUMBER || c.Num < 0 { return 0, false }
  return time.Duration(c.Num * float64(time.Second)), true
}

// recvValues: (values x t) bei Erfolg, (values nil nil) sonst
func recvValues(v *Cell, ok bool) *Cell {
  if !ok { return MakeValues([]*Cell{MakeNil(), MakeNil()}) }
  return MakeValues([]*Cell{v, MakeAtom("t")})
}

// chanPut: Senden mit Schutz gegen close-Wettlauf (Go panict bei send auf closed)
func chanPut(gc *goChannel, v *Cell, block bool, name string) (sent bool, err error) {
  if gc.closed.Load() { return false, fmt.Errorf("%s: Channel geschlossen", name) }
  defer func() {
    if r := recover(); r != nil { sent, err = false, fmt.Errorf("%s: Channel geschlossen", name) }
  }()
  if block { gc.ch <- v; return true, nil }
  select {
  case gc.ch <- v: return true, nil
  default:         return false, nil
  }
}

// chan-send: (chan-send ch wert)
func fnChanSend(args []*Cell) (*Cell, error) {
  if len(args) < 2 { return nil, fmt.Errorf("chan-send: 2 Argumente nötig") }
  gc, err := getChannel(args[0])
  if err != nil { return nil, err }
  if _, err := chanPut(gc, args[1], true, "chan-send"); err != nil { return nil, err }
  return args[1], nil
}

// chan-try-send: (chan-try-send ch wert) → t oder nil
func fnChanTrySend(args []*Cell) (*Cell, error) {
  if len(args) != 2 { return nil, fmt.Errorf("chan-try-send: 2 Argumente nötig") }
  gc, err := getChannel(args[0])
  if err != nil { return nil, err }
  sent, err := chanPut(gc, args[1], false, "chan-try-send")
  if err != nil { return nil, err }
  if sent { return MakeAtom("t"), nil }
  return MakeNil(), nil
}

// chan-recv: (chan-recv ch [:timeout sek])
func fnChanRecv(args []*Cell) (*Cell, error) {
  syntax := fmt.Errorf("chan-recv: Syntax: (chan-recv ch [:timeout sek])")
  if len(args) != 1 && len(args) != 3 { return nil, syntax }
  gc, err := getChannel(args[0])
  if err != nil { return nil, err }
  if len(args) == 1 {
    v, ok := <-gc.ch
    return recvValues(v, ok), nil
  }
  if args[1] == nil || args[1].Val != ":timeout" { return nil, syntax }
  d, ok := timeoutArg(args[2])
  if !ok { return nil, syntax }
  timer := time.NewTimer(d)
  defer timer.Stop()
  select {
  case v, ok := <-gc.ch:
    return recvValues(v, ok), nil
  case <-timer.C:
    return recvValues(nil, false), nil
  }
}

// chan-try-recv: (chan-try-recv ch)
func fnChanTryRecv(args []*Cell) (*Cell, error) {
  if len(args) != 1 { return nil, fmt.Errorf("chan-try-recv: 1 Argument nötig") }
  gc, err := getChannel(args[0])
  if err != nil { return nil, err }
  select {
  case v, ok := <-gc.ch:
    return recvValues(v, ok), nil
  default:
    return recvValues(nil, false), nil
  }
}

// chan-close: (chan-close ch) → t; zweites Mal Fehler
func fnChanClose(args []*Cell) (*Cell, error) {
  if len(args) != 1 { return nil, fmt.Errorf("chan-close: 1 Argument nötig") }
  gc, err := getChannel(args[0])
  if err != nil { return nil, err }
  if !gc.closed.CompareAndSwap(false, true) {
    return nil, fmt.Errorf("chan-close: Channel bereits geschlossen")
  }
  close(gc.ch)
  return MakeAtom("t"), nil
}

// chan-closed?: t nur wenn geschlossen und leer
func fnChanClosedP(args []*Cell) (*Cell, error) {
  if len(args) != 1 { return nil, fmt.Errorf("chan-closed?: 1 Argument nötig") }
  gc, err := getChannel(args[0])
  if err != nil { return nil, err }
  if gc.closed.Load() && len(gc.ch) == 0 { return MakeAtom("t"), nil }
  return MakeNil(), nil
}

// ---- Mutex ----

// lock-make: (lock-make)
func fnLockMake(args []*Cell) (*Cell, error) {
  return makeMutexCell(), nil
}

// spawn: (spawn fn) → startet fn ohne Argumente in einer neuen Goroutine
// und kehrt sofort mit nil zurück (Fire-and-forget, Gegenstück zu
// parfunc, das wartet). Fehler und panics von fn landen auf stderr.
func fnSpawn(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("spawn: genau 1 Argument nötig (Funktion)")
  }
  fn := args[0]
  if fn == nil || (fn.Type != FUNC && fn.Type != LAMBDA) {
    return nil, fmt.Errorf("spawn: '%s' ist keine Funktion", fn)
  }
  go func() {
    defer func() {
      if r := recover(); r != nil {
        fmt.Fprintf(os.Stderr, "spawn: panic: %v\n", r)
      }
    }()
    if _, err := apply(fn, nil); err != nil {
      fmt.Fprintf(os.Stderr, "spawn: %v\n", err)
    }
  }()
  return MakeNil(), nil
}
