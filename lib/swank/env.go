//**********************************************************************
//  lib/swank/env.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260618
//**********************************************************************
// Per-connection SWANK primitives: send-event, print, println,
// value-string.
//**********************************************************************

package swank

import (
  "fmt"
  "strings"

  "golisp2/lib"
)

// RegisterSwankEnv registers connection-bound SWANK primitives.
// send writes an event Cell to Emacs.
func RegisterSwankEnv(env *lib.Env, send func(*lib.Cell) error) {
  // Redirect all lib output (format t, ga-print, print, println) to Emacs.
  lib.SetOutputWriter(func(s string) error {
    event := lib.Cons(
      lib.MakeAtom(":write-string"),
      lib.Cons(lib.MakeStr(s), lib.MakeNil()),
    )
    return send(event)
  })

  _ = env.Set("swank-send-event", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    if len(args) < 1 {
      return nil, fmt.Errorf("swank-send-event: 1 Argument nötig")
    }
    if err := send(args[0]); err != nil {
      return nil, fmt.Errorf("swank-send-event: %w", err)
    }
    return lib.MakeNil(), nil
  }))

  _ = env.Set("swank-print", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    return swankPrint(args, send, false)
  }))

  _ = env.Set("swank-println", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    return swankPrint(args, send, true)
  }))

  _ = env.Set("swank--value-string", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    if len(args) < 1 {
      return nil, fmt.Errorf("swank--value-string: 1 Argument nötig")
    }
    return lib.MakeStr(args[0].String()), nil
  }))

  // swank--read-all: String -> Liste aller gelesenen Formen. Für
  // listener-eval mit mehreren Formen pro Eingabeblock.
  _ = env.Set("swank--read-all", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    if len(args) < 1 {
      return nil, fmt.Errorf("swank--read-all: 1 Argument nötig")
    }
    if args[0].Type != lib.STRING {
      return nil, fmt.Errorf("swank--read-all: String erwartet")
    }
    return lib.ReadAll(args[0].Val)
  }))

  // swank--symbols: Liste aller Symbolnamen im Env (inkl. äußere Scopes).
  // Für swank:simple-completions (Tab-Completion).
  _ = env.Set("swank--symbols", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    names := env.Symbols()
    cells := make([]*lib.Cell, len(names))
    for i, n := range names {
      cells[i] = lib.MakeStr(n)
    }
    return lib.SliceToCell(cells), nil
  }))

  // swank--arglist: (name) -> "(name p1 p2 ...)" oder NIL.
  // Nutzt Lambda-Struktur (Type:LIST mit Env) bzw. Macro (Type:MACRO),
  // deren Car die Parameterliste. Built-in FUNC hat keine Arglist -> NIL.
  _ = env.Set("swank--arglist", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    if len(args) < 1 {
      return lib.MakeNil(), nil
    }
    name := args[0].Val
    cell, err := env.Get(name)
    if err != nil {
      return lib.MakeNil(), nil
    }
    var params *lib.Cell
    switch {
    case cell.Type == lib.LIST && cell.Env != nil: // Lambda/Closure
      params = cell.Car
    case cell.Type == lib.MACRO:
      params = cell.Car
    default: // FUNC, ATOM, NUMBER, STRING, NIL
      return lib.MakeNil(), nil
    }
    var b strings.Builder
    b.WriteString("(")
    b.WriteString(name)
    for p := params; p != nil && p.Type == lib.LIST; p = p.Cdr {
      if p.Car != nil {
        b.WriteString(" ")
        b.WriteString(p.Car.String())
      }
    }
    b.WriteString(")")
    return lib.MakeStr(b.String()), nil
  }))
  // swank--cell-type: Cell -> Typ-Name als String. Für describe-symbol,
  // damit FUNC/MACRO/LIST/... unterschieden werden können.
  _ = env.Set("swank--cell-type", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    if len(args) < 1 || args[0] == nil {
      return lib.MakeStr("unknown"), nil
    }
    switch args[0].Type {
    case lib.ATOM:
      return lib.MakeStr("atom"), nil
    case lib.NUMBER:
      return lib.MakeStr("number"), nil
    case lib.STRING:
      return lib.MakeStr("string"), nil
    case lib.LIST:
      return lib.MakeStr("lambda"), nil
    case lib.FUNC:
      return lib.MakeStr("function"), nil
    case lib.MACRO:
      return lib.MakeStr("macro"), nil
    case lib.NIL:
      return lib.MakeStr("nil"), nil
    default:
      return lib.MakeStr("unknown"), nil
    }
  }))

  // swank--find-definition: (name) -> ("file" . line) | NIL.
  // Map-Lookup in lib.LookupDefinition (defun/defmacro/define registriert).
  // NIL wenn Datei leer (REPL-defined) oder nicht registriert.
  _ = env.Set("swank--find-definition", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    if len(args) < 1 {
      return lib.MakeNil(), nil
    }
    loc, ok := lib.LookupDefinition(args[0].Val)
    if !ok || loc.File == "" {
      return lib.MakeNil(), nil
    }
    return lib.Cons(lib.MakeStr(loc.File), lib.MakeNum(float64(loc.Line))), nil
  }))

  // swank--definition-kind: (name) -> "lambda" | "macro" | "builtin" | "unbound".
  // Lambda = Type:LIST mit Env!=nil; Macro = Type:MACRO; sonst builtin/unbound.
  _ = env.Set("swank--definition-kind", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    if len(args) < 1 {
      return lib.MakeStr("unbound"), nil
    }
    cell, err := env.Get(args[0].Val)
    if err != nil || cell == nil {
      return lib.MakeStr("unbound"), nil
    }
    switch {
    case cell.Type == lib.LIST && cell.Env != nil:
      return lib.MakeStr("lambda"), nil
    case cell.Type == lib.MACRO:
      return lib.MakeStr("macro"), nil
    case cell.Type == lib.FUNC:
      return lib.MakeStr("builtin"), nil
    default:
      return lib.MakeStr("unbound"), nil
    }
  }))

  // swank--definition-cell: (name) -> Lambda/Macro-Cell | NIL.
  // Für swank--reconstruct-definition (REPL-Snippet).
  _ = env.Set("swank--definition-cell", makeFn(func(args []*lib.Cell) (*lib.Cell, error) {
    if len(args) < 1 {
      return lib.MakeNil(), nil
    }
    cell, err := env.Get(args[0].Val)
    if err != nil {
      return lib.MakeNil(), nil
    }
    if (cell.Type == lib.LIST && cell.Env != nil) || cell.Type == lib.MACRO {
      return cell, nil
    }
    return lib.MakeNil(), nil
  }))
}

func makeFn(f func([]*lib.Cell) (*lib.Cell, error)) *lib.Cell {
  return &lib.Cell{Type: lib.FUNC, Fn: f}
}

func swankPrint(args []*lib.Cell, send func(*lib.Cell) error, newline bool) (*lib.Cell, error) {
  var b strings.Builder
  for i, a := range args {
    if i > 0 {
      b.WriteString(" ")
    }
    b.WriteString(a.String())
  }
  if newline {
    b.WriteString("\n")
  }
  event := lib.Cons(
    lib.MakeAtom(":write-string"),
    lib.Cons(
      lib.MakeStr(b.String()),
      lib.MakeNil(),
    ),
  )
  if err := send(event); err != nil {
    return nil, fmt.Errorf("swank-print: %w", err)
  }
  if len(args) == 0 {
    return lib.MakeNil(), nil
  }
  return args[len(args)-1], nil
}
