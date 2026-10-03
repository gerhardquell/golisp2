//**********************************************************************
//  lib/stringsplit_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261003
//**********************************************************************
// Tests für string-split / string-join (TODO-20261003-luecken Punkt 4).
//**********************************************************************

package lib

import "testing"

func TestStringSplit(t *testing.T) {
  cases := []struct{ src, want string }{
    // mit sep: wörtlich, leere Felder bleiben
    {`(string-split "a,,b" ",")`, `("a" "" "b")`},
    {`(string-split "x\ny\n" "\n")`, `("x" "y" "")`},
    {`(string-split "a::b::c" "::")`, `("a" "b" "c")`},
    {`(string-split "abc" ",")`, `("abc")`},
    {`(string-split "" ",")`, `("")`},
    // ohne sep: Whitespace, leere Felder weg
    {`(string-split "  zwei  Wörter ")`, `("zwei" "Wörter")`},
    {`(string-split "a\tb\nc")`, `("a" "b" "c")`},
    {`(string-split "   ")`, `()`},
  }
  for _, c := range cases {
    evalEq(t, c.src, c.want)
  }
  for _, src := range []string{
    `(string-split)`,
    `(string-split 5)`,
    `(string-split "abc" "")`,  // leerer Trenner
    `(string-split "abc" 1)`,
    `(string-split "a" "," ",")`,
  } {
    evalErr(t, src)
  }
}

func TestStringJoin(t *testing.T) {
  evalEq(t, `(string-join '("a" "b") ", ")`, `"a, b"`)
  evalEq(t, `(string-join '("a") ",")`, `"a"`)
  evalEq(t, `(string-join () ",")`, `""`)
  evalEq(t, `(string-join '("a" "b") "")`, `"ab"`)
  // Rundreise
  evalEq(t, `(string-join (string-split "a,,b" ",") ",")`, `"a,,b"`)
  for _, src := range []string{
    `(string-join '("a" "b"))`,     // sep Pflicht (sonst list->string)
    `(string-join '("a" 1) ",")`,   // nur Strings
    `(string-join "ab" ",")`,       // keine Liste
    `(string-join (cons "a" "b") ",")`,
    `(string-join '("a") 1)`,
  } {
    evalErr(t, src)
  }
}
