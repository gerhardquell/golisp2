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

var traceDepth atomic.Int32

// traceState gehört einem Root-Env. Pro RegisterTrace ein eigener statt
// globaler Zustand: ein weiteres BaseEnv (Sandbox, sandbox.go) überschriebe
// sonst Root und Originale des ersten.
type traceState struct {
  mu    sync.RWMutex
  origs map[string]*Cell
  root  *Env
}

// RegisterTrace registriert trace/untrace/trace? im Root-Env.
func RegisterTrace(env *Env) {
  ts := &traceState{origs: make(map[string]*Cell), root: env.Root()}
  _ = env.Set("trace",   makeFn(ts.fnTrace))
  _ = env.Set("untrace", makeFn(ts.fnUntrace))
  _ = env.Set("trace?",  makeFn(ts.fnTraceP))
}

// fnTrace: (trace 'name) -> name
func (ts *traceState) fnTrace(args []*Cell) (*Cell, error) {
  if len(args) != 1 || args[0] == nil || args[0].Type != ATOM {
    return nil, fmt.Errorf("trace: 1 Atom-Argument erwartet")
  }
  name := args[0].Val

  ts.mu.Lock()
  defer ts.mu.Unlock()

  if _, ok := ts.origs[name]; ok {
    return MakeAtom(name), nil
  }

  cur, err := ts.root.Get(name)
  if err != nil {
    return nil, fmt.Errorf("trace: Symbol '%s' nicht gebunden", name)
  }
  if cur.Type != FUNC && cur.Type != LAMBDA {
    return nil, fmt.Errorf("trace: '%s' ist keine Funktion", name)
  }

  ts.origs[name] = cur
  wrapper := makeTracedWrapper(name, cur)
  if err := ts.root.Update(name, wrapper); err != nil {
    delete(ts.origs, name)
    return nil, err
  }
  return MakeAtom(name), nil
}

// fnUntrace: (untrace 'name) -> name | nil, (untrace) -> (name ...)
func (ts *traceState) fnUntrace(args []*Cell) (*Cell, error) {
  ts.mu.Lock()
  defer ts.mu.Unlock()

  if len(args) == 0 {
    return ts.untraceAllLocked(), nil
  }
  if len(args) != 1 || args[0] == nil || args[0].Type != ATOM {
    return nil, fmt.Errorf("untrace: 0 oder 1 Atom-Argument erwartet")
  }
  name := args[0].Val

  orig, ok := ts.origs[name]
  if !ok {
    return MakeNil(), nil
  }
  delete(ts.origs, name)
  if err := ts.root.Update(name, orig); err != nil {
    // Wiederherstellen des Registry-Eintrags, damit Zustand konsistent bleibt.
    ts.origs[name] = orig
    return nil, err
  }
  return MakeAtom(name), nil
}

// fnTraceP: (trace? 'name) -> t | nil
func (ts *traceState) fnTraceP(args []*Cell) (*Cell, error) {
  if len(args) != 1 || args[0] == nil || args[0].Type != ATOM {
    return nil, fmt.Errorf("trace?: 1 Atom-Argument erwartet")
  }
  ts.mu.RLock()
  _, ok := ts.origs[args[0].Val]
  ts.mu.RUnlock()
  if ok {
    return MakeAtom("t"), nil
  }
  return MakeNil(), nil
}

func makeTracedWrapper(name string, original *Cell) *Cell {
  return makeCtxFn(func(args []*Cell, ectx evalCtx) (*Cell, error) {
    depth := traceDepth.Add(1)
    defer traceDepth.Add(-1)

    indent := strings.Repeat("  ", int(depth-1))
    argStr := formatTraceArgs(args)
    _ = WriteError(fmt.Sprintf("%s(%s %s)\n", indent, name, argStr))

    res, err := applyWithCtx(original, args, ectx)
    if err != nil {
      return nil, err
    }
    _ = WriteError(fmt.Sprintf("%s(%s %s) => %s\n", indent, name, argStr, res.String()))
    return res, nil
  })
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

func (ts *traceState) untraceAllLocked() *Cell {
  names := make([]string, 0, len(ts.origs))
  for name := range ts.origs {
    names = append(names, name)
  }
  sort.Strings(names)

  // Originale zurückschreiben.
  for _, name := range names {
    if orig, ok := ts.origs[name]; ok {
      _ = ts.root.Update(name, orig)
    }
  }
  clear(ts.origs)

  cells := make([]*Cell, len(names))
  for i, n := range names {
    cells[i] = MakeAtom(n)
  }
  return SliceToCell(cells)
}
