//**********************************************************************
//  lib/eval_load.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260616 (aufgespalten aus eval.go)
//**********************************************************************
// Laden von Lisp-Dateien via zentraler Pfadauflösung in fileio.go.
//**********************************************************************

package lib

import (
  "fmt"
  "os"
  "path/filepath"
  "strings"
)

// load: (load "datei.lisp") → liest und wertet alle Ausdrücke aus
func evalLoad(args *Cell, env *Env, ectx evalCtx) (*Cell, error) {
  filenameCell, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil { return nil, err }
  if filenameCell == nil || filenameCell.Type != STRING {
    return nil, fmt.Errorf("load: Dateiname muss String sein")
  }

  resolvedPath, err := resolvePath(filenameCell.Val)
  if err != nil {
    return nil, fmt.Errorf("load: %v", err)
  }
  if abs, aerr := filepath.Abs(resolvedPath); aerr == nil {
    resolvedPath = abs
  }

  data, err := os.ReadFile(resolvedPath)
  if err != nil {
    return nil, fmt.Errorf("load: '%s' nicht lesbar: %w", resolvedPath, err)
  }

  src := strings.TrimSpace(string(data))
  var result *Cell

  // Mehrere Ausdrücke in der Datei nacheinander auswerten
  r := NewReader(src)
  for {
    r.skipWS()
    if r.pos >= len(r.src) { break }

    expr, err := r.readExpr()
    if err != nil { return nil, fmt.Errorf("load %s: %w", resolvedPath, err) }

    if expr.Type == LIST {
      expr.SrcFile = resolvedPath
    }

    result, err = evalWithCtx(expr, env, ectx.child())
    if err != nil { return nil, fmt.Errorf("load %s: %w", resolvedPath, err) }
  }
  return result, nil
}

// LoadString: Mehrere Ausdrücke aus einem String auswerten
func LoadString(src string, env *Env) (*Cell, error) {
  return loadStringWithCtx(src, env, evalCtx{depth: 0})
}

func loadStringWithCtx(src string, env *Env, ectx evalCtx) (*Cell, error) {
  src = strings.TrimSpace(src)
  var result *Cell
  r := NewReader(src)
  for {
    r.skipWS()
    if r.pos >= len(r.src) { break }
    expr, err := r.readExpr()
    if err != nil { return nil, fmt.Errorf("stdlib: %w", err) }
    result, err = evalWithCtx(expr, env, ectx.child())
    if err != nil { return nil, fmt.Errorf("stdlib: %w", err) }
  }
  return result, nil
}
