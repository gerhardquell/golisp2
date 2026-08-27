//**********************************************************************
//  lib/sysinfo.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260815
//**********************************************************************
// Kommandozeile und Environment lesen (TODO 20260813 Punkt 2.4).
//
//   (argv)          → Liste der Strings aus os.Args (roh, ungekürzt)
//   (getenv "NAME") → String oder () wenn nicht gesetzt
//   (environ)       → Alist (("NAME" . "wert") ...) aller Env-Vars
//
// Bei Shebang-Skripten ("./s.lisp a b") ist argv die volle Zeile:
// ("…/golisp2" "s.lisp" "a" "b") — das Skript entscheidet selbst,
// welche Elemente es braucht. Eine Quelle der Wahrheit, kein Filtern.
//**********************************************************************

package lib

import (
  "fmt"
  "os"
  "strings"
)

// RegisterSysinfo hängt argv/getenv/environ ins Environment ein.
func RegisterSysinfo(env *Env) {
  _ = env.Set("argv",    makeFn(fnArgv))
  _ = env.Set("getenv",  makeFn(fnGetenv))
  _ = env.Set("environ", makeFn(fnEnviron))
}

// argv: (argv) → Liste der Kommandozeilen-Strings
func fnArgv(args []*Cell) (*Cell, error) {
  if len(args) != 0 { return nil, fmt.Errorf("argv: keine Argumente erwartet") }
  cells := make([]*Cell, len(os.Args))
  for i, a := range os.Args { cells[i] = MakeStr(a) }
  return SliceToCell(cells), nil
}

// getenv: (getenv "NAME") → Wert oder () wenn nicht gesetzt
func fnGetenv(args []*Cell) (*Cell, error) {
  if len(args) != 1 { return nil, fmt.Errorf("getenv: 1 Argument nötig") }
  if args[0].Type != STRING { return nil, fmt.Errorf("getenv: String erwartet") }
  v, ok := os.LookupEnv(args[0].Val)
  if !ok { return MakeNil(), nil }
  return MakeStr(v), nil
}

// environ: (environ) → Alist aller Environment-Variablen
func fnEnviron(args []*Cell) (*Cell, error) {
  if len(args) != 0 { return nil, fmt.Errorf("environ: keine Argumente erwartet") }
  var pairs []*Cell
  for _, kv := range os.Environ() {
    name, val, _ := strings.Cut(kv, "=")
    pairs = append(pairs, Cons(MakeStr(name), MakeStr(val)))
  }
  return SliceToCell(pairs), nil
}
