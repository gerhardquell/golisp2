//**********************************************************************
//  lib/goroutine.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260224
//**********************************************************************
// Go-Nebenläufigkeit als Lisp-Primitiven:
//   (parfunc ergebnis expr1 expr2 ...)  → parallel auswerten
//   (spawn fn)                           → fn in neuer Goroutine, sofort Ergebnis-Channel
//                                          mit genau einer Nachricht (:ok wert) / (:error text)
//   (chan-make)                          → neuen Channel erstellen
//   (chan-send ch wert)                  → Wert senden (blockiert)
//   (chan-try-send ch wert)              → t gesendet / nil (voll, kein Empfänger)
//   (chan-recv ch [:timeout sek])        → (values x t) / (values nil nil)
//   (chan-try-recv ch)                   → (values x t) / sofort (values nil nil)
//   (chan-select chans [:timeout sek])   → (values x t i) vom ersten liefernden Channel,
//                                          sonst (values nil nil nil)
//   (chan-close ch)                      → Channel schließen (t)
//   (chan-closed? ch)                    → t wenn geschlossen und leer
//   (lock-make)                          → neuen Mutex erstellen
//   (lock mu expr)                       → atomar ausführen
//**********************************************************************

package lib

import (
  "fmt"
  "math"
  "os"
  "reflect"
  "sync"
  "sync/atomic"
  "time"
)

// goChannel: Go-Channel verpackt in Cell. Sender halten mu.RLock, close(ch)
// läuft unter mu.Lock – so laufen Senden und Schließen nie gleichzeitig;
// done weckt blockierte Sender beim Schließen.
type goChannel struct {
  ch     chan *Cell
  done   chan struct{}
  mu     sync.RWMutex
  closed atomic.Bool
}

// goMutex: Go-Mutex verpackt in Cell
type goMutex struct {
  mu sync.Mutex
}

// makeChannelCell erstellt eine Cell die einen Channel hält
func makeChannelCell(size int) *Cell {
  gc := &goChannel{ch: make(chan *Cell, size), done: make(chan struct{})}
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
  _ = env.Set("chan-select", makeFn(fnChanSelect))
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

// timeoutArg liest die Zahl hinter :timeout (Sekunden, >= 0, endlich);
// sehr große Werte werden auf die längste Duration gedeckelt
func timeoutArg(c *Cell) (time.Duration, bool) {
  if c == nil || c.Type != NUMBER || c.Num < 0 || math.IsNaN(c.Num) || math.IsInf(c.Num, 0) {
    return 0, false
  }
  ns := c.Num * float64(time.Second)
  if ns >= math.MaxInt64 { return math.MaxInt64, true }
  return time.Duration(ns), true
}

// recvValues: (values x t) bei Erfolg, (values nil nil) sonst
func recvValues(v *Cell, ok bool) *Cell {
  if !ok { return MakeValues([]*Cell{MakeNil(), MakeNil()}) }
  return MakeValues([]*Cell{v, MakeAtom("t")})
}

// chanPut: Senden unter mu.RLock; geschlossener Channel → Fehler statt panic
func chanPut(gc *goChannel, v *Cell, block bool, name string) (bool, error) {
  gc.mu.RLock()
  defer gc.mu.RUnlock()
  closedErr := fmt.Errorf("%s: Channel geschlossen", name)
  if gc.closed.Load() { return false, closedErr }
  if block {
    select {
    case gc.ch <- v: return true, nil
    case <-gc.done:  return false, closedErr
    }
  }
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

// chan-select: (chan-select chans [:timeout sek]) → (values x t i) / (values nil nil nil)
// Geschlossene und leere Channels scheiden aus; sind alle weg, endet das Warten.
func fnChanSelect(args []*Cell) (*Cell, error) {
  syntax := fmt.Errorf("chan-select: Syntax: (chan-select chans [:timeout sek])")
  if len(args) != 1 && len(args) != 3 { return nil, syntax }
  var d time.Duration
  hasTimeout := len(args) == 3
  if hasTimeout {
    if args[1] == nil || args[1].Val != ":timeout" { return nil, syntax }
    var ok bool
    if d, ok = timeoutArg(args[2]); !ok { return nil, syntax }
  }
  if args[0] != nil && args[0].Type != LIST && args[0].Type != NIL { return nil, syntax }
  chans := CellToSlice(args[0])
  if len(chans) == 0 { return nil, fmt.Errorf("chan-select: leere Channel-Liste") }
  cases := make([]reflect.SelectCase, len(chans), len(chans)+1)
  index := make([]int, len(chans))
  for i, c := range chans {
    gc, err := getChannel(c)
    if err != nil { return nil, fmt.Errorf("chan-select: kein Channel an Stelle %d", i) }
    cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(gc.ch)}
    index[i] = i
  }
  timeoutCase := -1
  if hasTimeout {
    timeoutCase = len(cases)
    if d == 0 {
      cases = append(cases, reflect.SelectCase{Dir: reflect.SelectDefault})
    } else {
      timer := time.NewTimer(d)
      defer timer.Stop()
      cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(timer.C)})
    }
  }
  none := MakeValues([]*Cell{MakeNil(), MakeNil(), MakeNil()})
  for len(index) > 0 {
    chosen, v, ok := reflect.Select(cases)
    if chosen == timeoutCase { return none, nil }
    if !ok {
      cases = append(cases[:chosen], cases[chosen+1:]...)
      index = append(index[:chosen], index[chosen+1:]...)
      if hasTimeout { timeoutCase-- }
      continue
    }
    x, _ := v.Interface().(*Cell)
    return MakeValues([]*Cell{x, MakeAtom("t"), MakeNum(float64(index[chosen]))}), nil
  }
  return none, nil
}

// chan-close: (chan-close ch) → t; zweites Mal Fehler
func fnChanClose(args []*Cell) (*Cell, error) {
  if len(args) != 1 { return nil, fmt.Errorf("chan-close: 1 Argument nötig") }
  gc, err := getChannel(args[0])
  if err != nil { return nil, err }
  if !gc.closed.CompareAndSwap(false, true) {
    return nil, fmt.Errorf("chan-close: Channel bereits geschlossen")
  }
  close(gc.done)
  gc.mu.Lock()
  close(gc.ch)
  gc.mu.Unlock()
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
// und kehrt sofort mit einem Ergebnis-Channel (Puffer 1) zurück (Gegenstück
// zu parfunc, das wartet). Der Channel erhält genau eine Nachricht:
// (:ok wert) mit dem Primärwert, (:error text) bei Fehler oder panic.
// Fehler und panics gehen zusätzlich auf stderr. Ist der Channel beim
// Senden geschlossen oder vom Benutzer gefüllt, wird das Ergebnis verworfen
// (Meldung auf stderr) – die Goroutine endet trotzdem.
func fnSpawn(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("spawn: genau 1 Argument nötig (Funktion)")
  }
  fn := args[0]
  if fn == nil || (fn.Type != FUNC && fn.Type != LAMBDA) {
    return nil, fmt.Errorf("spawn: '%s' ist keine Funktion", fn)
  }
  result := makeChannelCell(1)
  gc, _ := getChannel(result)
  go func() {
    msg := List(MakeAtom(":error"), MakeStr("spawn: keine Nachricht"))
    defer func() {
      if r := recover(); r != nil {
        fmt.Fprintf(os.Stderr, "spawn: panic: %v\n", r)
        msg = List(MakeAtom(":error"), MakeStr(fmt.Sprintf("panic: %v", r)))
      }
      if sent, _ := chanPut(gc, msg, false, "spawn"); !sent {
        fmt.Fprintln(os.Stderr, "spawn: Ergebnis verworfen (Channel geschlossen oder voll)")
      }
    }()
    v, err := apply(fn, nil)
    if err != nil {
      fmt.Fprintf(os.Stderr, "spawn: %v\n", err)
      msg = List(MakeAtom(":error"), MakeStr(err.Error()))
      return
    }
    msg = List(MakeAtom(":ok"), Primary(v))
  }()
  return result, nil
}
