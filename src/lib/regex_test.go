//**********************************************************************
//  lib/regex_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Opus 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261020
//**********************************************************************
// Tests für die Regex-Primitiven (regex.go).
//**********************************************************************

package lib

import (
  "strings"
  "testing"
)

func TestRegexMatch(t *testing.T) {
  evalEq(t, `(regex-match "^[0-9]+$" "123")`, "t")
  evalEq(t, `(regex-match "^[0-9]+$" "12a")`, "()")
  evalEq(t, `(regex-match "b" "abc")`, "t") // Teiltreffer reicht
}

func TestRegexFind(t *testing.T) {
  evalEq(t, `(regex-find "[a-z]+" "12abc34")`, `"abc"`)
  evalEq(t, `(regex-find "[a-z]+" "1234")`, "()")
  evalEq(t, `(multiple-value-list (regex-find "(\\w+)@(\\w+)" "an: gq@skequell"))`, `("gq@skequell" ("gq" "skequell"))`)
  evalEq(t, `(multiple-value-list (regex-find "a(x)?b" "ab"))`, `("ab" (()))`) // Gruppe ohne Treffer → ()
  evalEq(t, `(multiple-value-list (regex-find "ab" "ab"))`, `("ab" ())`)
  evalEq(t, `(multiple-value-list (regex-find "x" "ab"))`, `(() ())`)
}

func TestRegexFindAll(t *testing.T) {
  evalEq(t, `(regex-find-all "[0-9]+" "a1b22c333")`, `("1" "22" "333")`)
  evalEq(t, `(regex-find-all "[0-9]+" "abc")`, "()")
}

func TestRegexReplace(t *testing.T) {
  evalEq(t, `(regex-replace "[0-9]+" "a1b22" "#")`, `"a#b#"`)
  evalEq(t, `(regex-replace "(\\w+)@(\\w+)" "gq@sk" "$2 bei $1")`, `"sk bei gq"`)
  evalEq(t, `(regex-replace "x" "abc" "y")`, `"abc"`)
}

func TestRegexSplit(t *testing.T) {
  evalEq(t, `(regex-split ",\\s*" "a, b,c")`, `("a" "b" "c")`)
  evalEq(t, `(regex-split "," "a,,b")`, `("a" "" "b")`) // leere Felder bleiben, wie string-split
}

func TestRegexFehler(t *testing.T) {
  for _, src := range []string{
    `(regex-match "(" "x")`,
    `(regex-match 1 "x")`,
    `(regex-match "x")`,
    `(regex-find "x" 1)`,
    `(regex-replace "x" "y")`,
    `(regex-replace "x" "y" 1)`,
    `(regex-split "x" nil)`,
  } {
    evalErr(t, src)
  }
  _, err := evalStr(`(regex-match "(" "x")`)
  if err == nil || !strings.Contains(err.Error(), "regex-match: ungültiges Muster") {
    t.Errorf("Fehlertext: %v", err)
  }
}
