//**********************************************************************
//  lib/shellcmd.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260306
//**********************************************************************
// Shell & Dateisystem-Primitiven:
//   (system "cmd")         → Exit-Code als Zahl (0 = OK)
//   (file-stat "path")     → ((size . N) (mtime . N)) oder nil
//**********************************************************************

package lib

import (
  "fmt"
  "os"
  "os/exec"
)

// RegisterShellCmd fügt system, file-stat, shell-assoc und symbol->string in env ein
func RegisterShellCmd(env *Env) {
  _ = env.Set("system",        makeFn(fnSystem))
  _ = env.Set("file-stat",     makeFn(fnFileStat))
  _ = env.Set("shell-assoc",   makeFn(fnAssoc))
  _ = env.Set("symbol->string", makeFn(fnSymbolToString))
}

// symbol->string: (symbol->string 'foo) → "foo"
func fnSymbolToString(args []*Cell) (*Cell, error) {
  if len(args) < 1 { return nil, fmt.Errorf("symbol->string: 1 Argument nötig") }
  return MakeStr(args[0].String()), nil
}

// shell-assoc: (shell-assoc key alist) → erstes Paar (key . val) oder nil
// Vergleicht mit equal? (strukturell)
func fnAssoc(args []*Cell) (*Cell, error) {
  if len(args) < 2 {
    return nil, fmt.Errorf("shell-assoc: 2 Argumente nötig")
  }
  key  := args[0]
  list := args[1]
  for list != nil && list.Type == LIST {
    pair := list.Car
    if pair != nil && (pair.Type == LIST) {
      if cellEqual(pair.Car, key) {
        return pair, nil
      }
    }
    list = list.Cdr
  }
  return MakeNil(), nil
}


// system: (system "shell-kommando") → Exit-Code (Zahl)
// Führt Kommando via /bin/sh -c aus, gibt Exit-Code zurück.
// Kein Fehler bei non-zero Exit — Aufrufer prüft selbst.
func fnSystem(args []*Cell) (*Cell, error) {
  if len(args) < 1 {
    return nil, fmt.Errorf("system: 1 Argument nötig")
  }
  cmd := exec.Command("/bin/sh", "-c", args[0].Val)
  err := cmd.Run()
  if err != nil {
    if exitErr, ok := err.(*exec.ExitError); ok {
      return MakeNum(float64(exitErr.ExitCode())), nil
    }
    return MakeNum(1), nil
  }
  return MakeNum(0), nil
}

// file-stat: (file-stat "path") → ((size . N) (mtime . N)) oder nil
func fnFileStat(args []*Cell) (*Cell, error) {
  if len(args) < 1 {
    return nil, fmt.Errorf("file-stat: 1 Argument nötig")
  }
  info, err := os.Stat(args[0].Val)
  if err != nil {
    return MakeNil(), nil
  }
  // Baue Assoziationsliste: ((size . N) (mtime . N))
  sizePair  := Cons(MakeAtom("size"),  MakeNum(float64(info.Size())))
  mtimePair := Cons(MakeAtom("mtime"), MakeNum(float64(info.ModTime().Unix())))
  return SliceToCell([]*Cell{sizePair, mtimePair}), nil
}
