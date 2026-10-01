//**********************************************************************
//  lib/eval_script.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Sonnet 5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261001
//**********************************************************************
// Skript-Ausführung: RunScript (Datei als Hauptprogramm) und Spezialform
// defmain. Spec: docs/superpowers/specs/2026-10-01-defmain-design.md
//**********************************************************************

package lib

import (
  "fmt"
  "math"
  "path/filepath"
  "strings"
)

// mainScriptState existiert nur, während RunScript die Hauptdatei lädt.
// Er reist als Zeiger in evalCtx.script mit — kein globaler Zustand,
// unsichtbar für Goroutinen, (eval …) und nachgeladene Dateien.
type mainScriptState struct {
  path      string // Hauptdatei, absolut aufgelöst (für Fehlermeldungen, wie loadFile)
  body      *Cell  // Closure aus defmain; nil = kein defmain gesehen
  takesArgs bool   // (defmain (args) …) statt (defmain () …)
  srcLine   int    // Zeile des ersten defmain
}

// RunScript lädt path als Hauptprogramm, ruft danach ggf. den
// defmain-Körper mit args auf und liefert den Exit-Code.
// Ohne defmain: result = Wert der letzten Form, hasMain = false.
// Jeder Fehler (Laden, Körper, ungültiger Exit-Code) → exitCode 1.
func RunScript(path string, args []string, env *Env) (exitCode int, result *Cell, hasMain bool, err error) {
  // Pfad VOR loadFile auflösen (derselbe Chokepoint wie loadFile:
  // resolvePath + filepath.Abs) — sonst steht st.path in einer anderen
  // Schreibweise als der Pfad, den loadFile für Fehlermeldungen nutzt.
  // Scheitert die Auflösung: Pfad unverändert lassen, loadFile erzeugt
  // dann seinen gewohnten `load: …`-Fehler.
  resolvedPath := path
  if rp, rerr := resolvePath(path); rerr == nil {
    if abs, aerr := filepath.Abs(rp); aerr == nil {
      rp = abs
    }
    resolvedPath = rp
  }
  st := &mainScriptState{path: resolvedPath}
  result, err = loadFile(resolvedPath, env, evalCtx{script: st})
  if err != nil {
    return 1, nil, st.body != nil, err
  }
  if st.body == nil {
    return 0, result, false, nil
  }

  var callArgs []*Cell
  if st.takesArgs {
    cells := make([]*Cell, len(args))
    for i, a := range args {
      cells[i] = MakeStr(a)
    }
    callArgs = []*Cell{SliceToCell(cells)}
  }
  // apply startet mit frischem evalCtx (script == nil): ein defmain im
  // Körper ist wirkungslos.
  var ret *Cell
  ret, err = apply(st.body, callArgs)
  if err != nil {
    return 1, nil, true, err
  }
  exitCode, err = mainExitCode(Primary(ret))
  return exitCode, ret, true, err
}

// defmain: (defmain () body...) / (defmain (args) body...)
// Nur in der Hauptdatei wirksam (ectx.script != nil): merkt den Körper,
// RunScript ruft ihn nach dem Laden auf. Überall sonst: nil, ungeprüft.
func evalDefmain(form *Cell, env *Env, ectx evalCtx) (*Cell, error) {
  st := ectx.script
  if st == nil {
    return MakeNil(), nil
  }
  args := form.Cdr
  if args == nil || args.Type != LIST {
    return nil, fmt.Errorf("defmain: Syntax: (defmain () body...) oder (defmain (args) body...)")
  }
  n, ok := mainParamCount(args.Car)
  if !ok {
    return nil, fmt.Errorf("defmain: Syntax: (defmain () body...) oder (defmain (args) body...)")
  }
  if st.body != nil {
    return nil, fmt.Errorf("defmain: bereits definiert in %s:%d", st.path, st.srcLine)
  }
  st.body = makeLambda(args.Car, wrapBegin(args.Cdr), env)
  st.takesArgs = n == 1
  st.srcLine = form.SrcLine
  return MakeNil(), nil
}

// mainParamCount prüft die defmain-Parameterliste: () → 0, (name) → 1.
// Lambda-Listen-Schlüsselwörter (&rest, &optional, …) sind nicht erlaubt.
func mainParamCount(params *Cell) (int, bool) {
  if params == nil || params.Type == NIL {
    return 0, true
  }
  if params.Type != LIST || params.Car == nil || params.Car.Type != ATOM {
    return 0, false
  }
  if strings.HasPrefix(params.Car.Val, "&") {
    return 0, false
  }
  if params.Cdr != nil && params.Cdr.Type != NIL {
    return 0, false
  }
  return 1, true
}

// mainExitCode: ganze Zahl 0–255 → Exit-Code; Nicht-Zahl/nil → 0;
// sonst laut scheitern (der Kernel schnitte 256 still auf 0 ab).
func mainExitCode(v *Cell) (int, error) {
  if v == nil || v.Type != NUMBER {
    return 0, nil
  }
  if v.Num != math.Trunc(v.Num) || v.Num < 0 || v.Num > 255 {
    return 1, fmt.Errorf("defmain: Exit-Code muss ganze Zahl 0–255 sein, got %s", v)
  }
  return int(v.Num), nil
}
