//**********************************************************************
//  lib/swank/lisp.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260618
//**********************************************************************
// Embedded SWANK Lisp handlers.
//**********************************************************************

package swank

import (
  _ "embed"

  "golisp2/lib"
)

//go:embed swank.lisp
var swankSrc string

// LoadSwankLisp loads the embedded SWANK handler library into env.
func LoadSwankLisp(env *lib.Env) error {
  _, err := lib.LoadString(swankSrc, env)
  return err
}
