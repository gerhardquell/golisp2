//**********************************************************************
//  lib/tutorial_coverage_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Opus 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261020
//**********************************************************************
// Jedes öffentliche Symbol (BaseEnv + Stdlib) braucht einen Eintrag im
// Tutorial ("### name", mehrere als "### a / b"). Sonst veraltet es still —
// am 20261020 fehlten 35 Einträge, ohne dass es jemand merkte.
//**********************************************************************

package lib

import (
  "os"
  "sort"
  "strings"
  "testing"
)

// Bewusst ohne eigenen Tutorial-Eintrag.
var tutorialAusnahmen = map[string]bool{
  "nil": true, "t": true, // Konstanten
  "ash-left": true, "ash-right": true, // Hilfsfunktionen von ash
}

func TestTutorialDecktAlleSymbole(t *testing.T) {
  data, err := os.ReadFile("../../tutorial/golisp-tutorial.md")
  if err != nil {
    t.Fatalf("Tutorial lesen: %v", err)
  }
  eintraege := map[string]bool{}
  for _, zeile := range strings.Split(string(data), "\n") {
    if name, ok := strings.CutPrefix(zeile, "### "); ok {
      for _, n := range strings.Split(name, " / ") {
        eintraege[strings.TrimSpace(n)] = true
      }
    }
  }

  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("stdlib: %v", err)
  }
  var fehlend []string
  for _, s := range env.Symbols() {
    intern := strings.HasPrefix(s, "%") || strings.HasPrefix(s, "*") || strings.HasPrefix(s, "swank")
    if !intern && !tutorialAusnahmen[s] && !eintraege[s] {
      fehlend = append(fehlend, s)
    }
  }
  sort.Strings(fehlend)
  if len(fehlend) > 0 {
    t.Errorf("%d Symbole ohne Tutorial-Eintrag (tutorial/golisp-tutorial.md, \"### name\"): %s",
      len(fehlend), strings.Join(fehlend, " "))
  }
}
