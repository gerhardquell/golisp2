//**********************************************************************
//  lib/redefguard.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260724
//**********************************************************************
// Gemeinsame Policy-Bausteine des Redefinition-Guards.
// Eine Policy-Auswertung fuer Env.Set-Hook, Kontext-Guard und makunbound.
//**********************************************************************

package lib

import (
  "fmt"
  "os"
)

// applyRedefPolicy bildet die Policy ab: allow → nil, warn → stderr + nil,
// error → Fehler (Redefinition wird abgebrochen).
// detail beschreibt den Kontext, z. B. "war FUNC".
func applyRedefPolicy(name, detail string) error {
  switch redefinePolicy(redefinePolicyAtomic.Load()) {
  case redefineWarn:
    fmt.Fprintf(os.Stderr, "REDEF: %s (%s)\n", name, detail)
  case redefineError:
    return fmt.Errorf("REDEF: %s (%s)", name, detail)
  }
  return nil
}

// policyAction liefert den Log-Action-Namen der aktuellen Policy.
func policyAction() string {
  switch redefinePolicy(redefinePolicyAtomic.Load()) {
  case redefineWarn:
    return "warn"
  case redefineError:
    return "error"
  }
  return "redef"
}

// checkRootRedefine wird von define/defun/defmacro VOR env.Set gerufen.
// Behandelt nur LAMBDA/MACRO-Altbindungen (Lisp-Definitionen); FUNC faengt
// der Hook in Env.Set ab. Reload aus derselben Quelle (DefLoc.File) ist
// immer erlaubt und still — das ist der normale Entwicklungs-Workflow.
func checkRootRedefine(env *Env, name string, newVal *Cell, newFile string, newLine int) error {
  if env != env.Root() {
    return nil
  }
  old, err := env.Get(name)
  if err != nil {
    return nil // nicht gebunden → Definition, keine Redefinition
  }
  if old.Type != LAMBDA && old.Type != MACRO {
    return nil
  }
  loc, _ := LookupDefinition(name)
  ev := RedefEvent{
    Name:    name,
    OldKind: kindOf(old),
    NewKind: kindOf(newVal),
    OldFile: loc.File,
    OldLine: loc.Line,
    NewFile: newFile,
    NewLine: newLine,
  }
  if loc.File == newFile {
    ev.Action = "reload"
    logRedef(ev)
    return nil
  }
  ev.Action = policyAction()
  logRedef(ev)
  detail := fmt.Sprintf("%s aus %s:%d, neu aus %s:%d",
    kindOf(old), loc.File, loc.Line, newFile, newLine)
  return applyRedefPolicy(name, detail)
}
