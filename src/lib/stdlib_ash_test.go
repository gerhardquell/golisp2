//**********************************************************************
//  lib/stdlib_ash_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260929
//**********************************************************************
// ash (stdlib.lisp) nach CL: Rechtsshift rundet Richtung -∞ (floor),
// nicht Richtung Null — relevant nur für negative Zahlen.
//**********************************************************************

package lib

import "testing"

func TestAshLinks(t *testing.T) {
  evalStdlibEq(t, `(ash 1 10)`, "1024")
  evalStdlibEq(t, `(ash -3 2)`, "-12")
  evalStdlibEq(t, `(ash 5 0)`, "5")
}

func TestAshRechtsPositiv(t *testing.T) {
  evalStdlibEq(t, `(ash 5 -1)`, "2")
  evalStdlibEq(t, `(ash 1024 -10)`, "1")
  evalStdlibEq(t, `(ash 0 -3)`, "0")
}

func TestAshRechtsNegativRundetGegenMinusUnendlich(t *testing.T) {
  evalStdlibEq(t, `(ash -5 -1)`, "-3")
  evalStdlibEq(t, `(ash -1 -1)`, "-1")
  evalStdlibEq(t, `(ash -1 -10)`, "-1")
  evalStdlibEq(t, `(ash -8 -2)`, "-2")
  evalStdlibEq(t, `(ash -9 -2)`, "-3")
}
