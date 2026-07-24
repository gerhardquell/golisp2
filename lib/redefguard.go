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
