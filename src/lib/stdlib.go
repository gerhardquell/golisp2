//**********************************************************************
//  lib/stdlib.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260616
//**********************************************************************
// Zentrale Einbettung und Ladung der Standardbibliothek.
// Sowohl CLI- als auch SWANK-Server-Modus (main.go, beide über --swank
// erreichbar) nutzen LoadStdlib – eine Quelle, keine Drift zwischen
// inline- und embed-Varianten.
//**********************************************************************

package lib

import (
  _ "embed"

  "golisp2/src/embed"
)

// LoadStdlib lädt die eingebettete Standardbibliothek (stdlib + defsystem +
// condition) in env. Einmal pro Env aufrufen (nach BaseEnv). Fehler =
// Syntaxfehler in den .lisp-Dateien – sollte zur Compile-Zeit nie passieren.
func LoadStdlib(env *Env) error {
  if _, err := LoadString(assets.Stdlib, env); err != nil {
    return err
  }
  if _, err := LoadString(assets.Defsystem, env); err != nil {
    return err
  }
  _, err := LoadString(assets.Condition, env)
  return err
}
