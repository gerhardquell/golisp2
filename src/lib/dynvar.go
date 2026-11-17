//**********************************************************************
//  lib/dynvar.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261117
//**********************************************************************
// Dynamische Variablen (Spec 2026-11-17): Menge der special-Namen.
// Schlüssel ist die internierte Symbol-Cell; die Cell selbst bleibt
// unverändert (Interning-Invariante). Special gilt global und für immer.
//**********************************************************************
package lib

import (
  "fmt"
  "strings"
  "sync"
)

var (
  specialMu   sync.RWMutex
  specialSyms = map[*Cell]struct{}{}
)

func isSpecial(sym *Cell) bool {
  specialMu.RLock()
  _, ok := specialSyms[sym]
  specialMu.RUnlock()
  return ok
}

// declareSpecial markiert sym und meldet, ob der Name neu war.
func declareSpecial(sym *Cell) bool {
  specialMu.Lock()
  defer specialMu.Unlock()
  if _, ok := specialSyms[sym]; ok {
    return false
  }
  specialSyms[sym] = struct{}{}
  return true
}

func RegisterDynvar(env *Env) {
  _ = env.Set("%declare-special", makeFn(fnDeclareSpecial))
}

// (%declare-special 'name) — von defvar/defparameter (stdlib.lisp).
func fnDeclareSpecial(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("%%declare-special: 1 Argument nötig, got %d", len(args))
  }
  if args[0].Type != ATOM {
    return nil, fmt.Errorf("%%declare-special: Symbol erwartet, got %s", args[0])
  }
  sym := args[0]
  name := sym.Val
  if declareSpecial(sym) && !(len(name) >= 3 && strings.HasPrefix(name, "*") && strings.HasSuffix(name, "*")) {
    _ = WriteError(fmt.Sprintf("WARN: defvar %s ohne *…* — jede let-Bindung von %s ist jetzt dynamisch\n", name, name))
  }
  return sym, nil
}
