//**********************************************************************
//  lib/stringcompare_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261029
//**********************************************************************
// string=, string/=, string<, string>, string<=, string>= (CL):
// string= liefert t/(), die anderen den Index der ersten Abweichung
// (in Zeichen, nicht Bytes) oder ().
//**********************************************************************

package lib

import "testing"

func TestStringVergleich(t *testing.T) {
  evalEq(t, `(list (string= "abc" "abc") (string= "abc" "abd"))`, "(t ())")
  evalEq(t, `(list (string< "abc" "abd") (string< "abd" "abc") (string< "abc" "abc"))`, "(2 () ())")
  // Präfix ist kleiner, Index = Länge des kürzeren
  evalEq(t, `(list (string< "ab" "abc") (string> "abc" "ab"))`, "(2 2)")
  // gleich: <=/>= liefern die Länge, /= liefert ()
  evalEq(t, `(list (string<= "abc" "abc") (string>= "abc" "abc") (string/= "abc" "abc"))`, "(3 3 ())")
  evalEq(t, `(string/= "abc" "aXc")`, "1")
  // Index in Zeichen: Umlaut zählt als eins
  evalEq(t, `(string< "äa" "äb")`, "1")
  // Symbole sind String-Bezeichner (CL)
  evalEq(t, `(string= 'abc "abc")`, "t")
  evalEq(t, `(sort (list "b" "c" "a") string<)`, `("a" "b" "c")`)
}

func TestStringVergleichFehler(t *testing.T) {
  evalErr(t, `(string< "a")`)
  evalErr(t, `(string< "a" 1)`)
  evalErr(t, `(string= 1 "a")`)
}
