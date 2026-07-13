//**********************************************************************
//  lib/trace.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260713
//**********************************************************************
// Live-Tracing einzelner Funktionen: (trace 'fn), (untrace 'fn),
// (untrace), (trace? 'fn).
//**********************************************************************

package lib

import (
  "fmt"
  "sort"
  "strings"
  "sync"
  "sync/atomic"
)

var (
  traceMu     sync.RWMutex
  traceOrigs  map[string]*Cell
  traceRoot   *Env
  traceDepth  atomic.Int32
)

// RegisterTrace registriert trace/untrace/trace? im Root-Env.
func RegisterTrace(env *Env) {
  traceMu.Lock()
  traceOrigs = make(map[string]*Cell)
  traceRoot = env.Root()
  traceDepth.Store(0)
  traceMu.Unlock()

  _ = env.Set("trace",   makeFn(fnTrace))
  _ = env.Set("untrace", makeFn(fnUntrace))
  _ = env.Set("trace?",  makeFn(fnTraceP))
}

// fnTrace: (trace 'name) -> name
func fnTrace(args []*Cell) (*Cell, error) {
  if len(args) != 1 || args[0] == nil || args[0].Type != ATOM {
    return nil, fmt.Errorf("trace: 1 Atom-Argument erwartet")
  }
  name := args[0].Val

  traceMu.Lock()
  defer traceMu.Unlock()

  if traceRoot == nil {
    return nil, fmt.Errorf("trace: kein Root-Env registriert")
  }
  if _, ok := traceOrigs[name]; ok {
    return MakeAtom(name), nil
  }

  cur, err := traceRoot.Get(name)
  if err != nil {
    return nil, fmt.Errorf("trace: Symbol '%s' nicht gebunden", name)
  }
  if cur.Type != FUNC && cur.Type != LIST {
    return nil, fmt.Errorf("trace: '%s' ist keine Funktion", name)
  }

  traceOrigs[name] = cur
  wrapper := makeTracedWrapper(name, cur)
  if err := traceRoot.Update(name, wrapper); err != nil {
    delete(traceOrigs, name)
    return nil, err
  }
  return MakeAtom(name), nil
}

// fnUntrace: (untrace 'name) -> name | nil, (untrace) -> (name ...)
func fnUntrace(args []*Cell) (*Cell, error) {
  traceMu.Lock()
  defer traceMu.Unlock()

  if len(args) == 0 {
    return untraceAllLocked(), nil
  }
  if len(args) != 1 || args[0] == nil || args[0].Type != ATOM {
    return nil, fmt.Errorf("untrace: 0 oder 1 Atom-Argument erwartet")
  }
  name := args[0].Val

  orig, ok := traceOrigs[name]
  if !ok {
    return MakeNil(), nil
  }
  delete(traceOrigs, name)
  if err := traceRoot.Update(name, orig); err != nil {
    // Wiederherstellen des Registry-Eintrags, damit Zustand konsistent bleibt.
    traceOrigs[name] = orig
    return nil, err
  }
  return MakeAtom(name), nil
}

// fnTraceP: (trace? 'name) -> t | nil
func fnTraceP(args []*Cell) (*Cell, error) {
  if len(args) != 1 || args[0] == nil || args[0].Type != ATOM {
    return nil, fmt.Errorf("trace?: 1 Atom-Argument erwartet")
  }
  traceMu.RLock()
  _, ok := traceOrigs[args[0].Val]
  traceMu.RUnlock()
  if ok {
    return MakeAtom("t"), nil
  }
  return MakeNil(), nil
}

func makeTracedWrapper(name string, original *Cell) *Cell {
  return &Cell{
    Type: FUNC,
    Fn: func(args []*Cell) (*Cell, error) {
      depth := traceDepth.Add(1)
      defer traceDepth.Add(-1)

      indent := strings.Repeat("  ", int(depth-1))
      argStr := formatTraceArgs(args)
      _ = WriteError(fmt.Sprintf("%s(%s %s)\n", indent, name, argStr))

      res, err := apply(original, args)
      if err != nil {
        return nil, err
      }
      _ = WriteError(fmt.Sprintf("%s(%s %s) => %s\n", indent, name, argStr, res.String()))
      return res, nil
    },
  }
}

func formatTraceArgs(args []*Cell) string {
  if len(args) == 0 {
    return ""
  }
  var b strings.Builder
  for i, a := range args {
    if i > 0 {
      b.WriteString(" ")
    }
    b.WriteString(a.String())
  }
  return b.String()
}

func untraceAllLocked() *Cell {
  names := make([]string, 0, len(traceOrigs))
  for name := range traceOrigs {
    names = append(names, name)
  }
  sort.Strings(names)

  // Originale zurückschreiben.
  for _, name := range names {
    if orig, ok := traceOrigs[name]; ok {
      _ = traceRoot.Update(name, orig)
    }
  }
  clear(traceOrigs)

  cells := make([]*Cell, len(names))
  for i, n := range names {
    cells[i] = MakeAtom(n)
  }
  return SliceToCell(cells)
}
