//**********************************************************************
//  lib/swank/dispatch.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260618
//**********************************************************************
// Dispatch SWANK messages into GoLisp (swank-dispatch).
//**********************************************************************

package swank

import (
  "fmt"

  "golisp2/lib"
)

// HandleMessage evaluates (swank-dispatch (quote msg)) in env and returns
// the list of SWANK events that Go should send back to Emacs.
func HandleMessage(env *lib.Env, msg *lib.Cell) (*lib.Cell, error) {
  quoteExpr := lib.Cons(lib.MakeAtom("quote"), lib.Cons(msg, lib.MakeNil()))
  expr := lib.Cons(lib.MakeAtom("swank-dispatch"), lib.Cons(quoteExpr, lib.MakeNil()))
  result, err := lib.Eval(expr, env)
  if err != nil {
    return nil, fmt.Errorf("HandleMessage: %w", err)
  }
  return result, nil
}
