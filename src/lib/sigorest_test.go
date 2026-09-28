//**********************************************************************
//  lib/sigorest_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Haiku 4.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260928
//**********************************************************************
// Tests für die sigoREST-Anbindung gegen einen Fake-sigoREST
// (httptest). Kein Live-Dienst nötig.
//**********************************************************************

package lib

import (
  "strings"
  "testing"

  "golisp2/src/embed"
)

func TestKiReferenzEmbedded(t *testing.T) {
  if !strings.HasPrefix(assets.KiReferenz, "# GoLisp2 — KI-Kurzreferenz") {
    t.Fatalf("KiReferenz beginnt falsch: %.60q", assets.KiReferenz)
  }
  if !strings.Contains(assets.KiReferenz, "sigo") {
    t.Fatal("KiReferenz erwähnt sigo nicht")
  }
}
