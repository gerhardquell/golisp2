//**********************************************************************
//  lib/stdlib.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260616
//**********************************************************************
// Zentrale Einbettung und Ladung der Standardbibliothek.
// Sowohl die CLI (main.go) als auch der Server (cmd/golisp2d) nutzen
// LoadStdlib – eine Quelle, keine Drift zwischen inline- und embed-Varianten.
//**********************************************************************

package lib

import (
  _ "embed"

  "golisp2/embed"
)

// LoadStdlib lädt die eingebettete Standardbibliothek (stdlib + defsystem) in env.
// Einmal pro Env aufrufen (nach BaseEnv). Fehler = Syntaxfehler in
// stdlib.lisp/defsystem.lisp – sollte zur Compile-Zeit nie passieren.
func LoadStdlib(env *Env) error {
  if _, err := LoadString(assets.Stdlib, env); err != nil {
    return err
  }
  _, err := LoadString(assets.Defsystem, env)
  return err
}
