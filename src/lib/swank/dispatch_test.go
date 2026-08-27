//**********************************************************************
//  lib/swank/dispatch_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260618
//**********************************************************************
// Tests für SWANK dispatch wrapper.
//**********************************************************************

package swank

import (
  "testing"

  "golisp2/src/lib"
)

func TestHandleMessage(t *testing.T) {
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  // Define a mock swank-dispatch in env
  _, err := lib.LoadString(`
    (defun swank-dispatch (msg)
      (list (list :return (list :ok :pong) 1)))
  `, env)
  if err != nil {
    t.Fatalf("load mock: %v", err)
  }

  msg := lib.Cons(lib.MakeAtom(":ping"), lib.MakeNil())
  result, err := HandleMessage(env, msg)
  if err != nil {
    t.Fatalf("HandleMessage failed: %v", err)
  }
  if result == nil || result.String() != "((:return (:ok :pong) 1))" {
    t.Fatalf("unexpected result: %v", result)
  }
}
