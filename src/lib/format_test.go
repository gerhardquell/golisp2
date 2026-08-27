//**********************************************************************
//  lib/format_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260625
//**********************************************************************
// Tests für die FORMAT-Primitive. Nutzt evalEq/evalErr aus eval_test.go.
//**********************************************************************

package lib

import (
  "io"
  "os"
  "testing"
)

func TestFormatBasic(t *testing.T) {
  cases := []struct{ src, want string }{
    {`(format nil "hallo")`, `"hallo"`},
    {`(format nil "~a ~a" 42 "x")`, `"42 x"`},
    {`(format nil "~s" "x")`, `"\"x\""`},       // readable = quoted
    {`(format nil "~a" "x")`, `"x"`},           // aesthetic = unquoted
    {`(format nil "~a" (list 1 2))`, `"(1 2)"`},
    {`(format nil "~a" nil)`, `"()"`},
    {`(format nil "~d" 255)`, `"255"`},
    {`(format nil "~b" 10)`, `"1010"`},
    {`(format nil "~x" 255)`, `"ff"`},
    {`(format nil "~o" 8)`, `"10"`},
    {`(format nil "~r" 4)`, `"four"`},
    {`(format nil "~:r" 3)`, `"third"`},        // ordinal
    {`(format nil "~@r" 14)`, `"XIV"`},          // römisch
    {`(format nil "~@r" 1999)`, `"MCMXCIX"`},
    {`(format nil "~p" 1)`, `""`},
    {`(format nil "~p" 2)`, `"s"`},
    {`(format nil "~:p" 1)`, `"y"`},
    {`(format nil "~:p" 2)`, `"ies"`},
    {`(format nil "~$" 3.14159)`, `"3.14"`},
    {`(format nil "~f" 3.14)`, `"3.14"`},
    {`(format nil "~,2f" 3.14159)`, `"3.14"`},
    {`(format nil "~5d" 42)`, `"   42"`},        // padding int
    {`(format nil "~5a" 42)`, `"42   "`},        // ~A = left-justified (pad rechts)
    {`(format nil "~5:a" 42)`, `"   42"`},       // ~:A = right-justified
    {`(format nil "~,,2,'.a" 5)`, `"5.."`},      // padchar + minpad
    {`(format nil "~[zero~;one~;two~]" 1)`, `"one"`},
    {`(format nil "~[zero~;one~;two~]" 0)`, `"zero"`},
    {`(format nil "~[zero~;one~;two~]" 5)`, `""`},
    {`(format nil "~:{~a~}" (list (list 1 2)))`, `"12"`},
    {`(format nil "~{~a~^,~}" (list 1 2 3))`, `"1,2,3"`},
    {`(format nil "~{~a~}" (list 1 2 3))`, `"123"`},
    {`(format nil "~(~a~)" "ABC")`, `"abc"`},
    {`(format nil "~@(~a~)" "abc")`, `"Abc"`},
    {`(format nil "~:(~a~)" "abc def")`, `"Abc Def"`},
    {`(format nil "~:@(~a~)" "abc")`, `"ABC"`},
    {`(format nil "~a~%" "x")`, `"x\n"`},
    {`(format nil "~a~&~a" 1 2)`, `"1\n2"`},
    {`(format nil "~~")`, `"~"`},
    {`(format nil "~3%")`, `"\n\n\n"`},
    {`(format nil "~?" "~a-~a" (list 7 8))`, `"7-8"`},
    {`(format nil "~a-~a" 1 2)`, `"1-2"`},
    {`(format nil "Wert: ~3,'0d" 5)`, `"Wert: 005"`},
    {`(format nil "~@d" 5)`, `"+5"`},
    {`(format nil "~:d" 1234567)`, `"1,234,567"`},
    {`(format nil "~c" 65)`, `"A"`},
    // ~:[ Default-Klausel via ~:;
    {`(format nil "~[zero~;one~:;many~]" 0)`, `"zero"`},
    {`(format nil "~[zero~;one~:;many~]" 1)`, `"one"`},
    {`(format nil "~[zero~;one~:;many~]" 5)`, `"many"`},
    {`(format nil "~[zero~;one~:;many~]" 99)`, `"many"`},
    // ~F k-Skaling (×10^k)
    {`(format nil "~,2,1f" 3.14)`, `"31.40"`},
    {`(format nil "~,2,-1f" 3.14)`, `"0.31"`},
    {`(format nil "~,2f" 3.14)`, `"3.14"`},
  }
  for _, c := range cases {
    evalEq(t, c.src, c.want)
  }
}

// TestFormatUserFunc: ~/fun/ ruft benannte Funktion (globalFormatEnv) auf.
func TestFormatUserFunc(t *testing.T) {
  evalEq(t, `(begin
              (defun up (s) (string-upcase s))
              (format nil "~/up/" "hi"))`, `"HI"`)
  evalEq(t, `(begin
              (defun double (n) (* n 2))
              (format nil "~/double/" 21))`, `"42"`)
}

func TestFormatErrors(t *testing.T) {
  evalErr(t, `(format nil "~a")`)     // zu wenige args
  evalErr(t, `(format 5 "x")`)        // bad dest
  evalErr(t, `(format nil 5)`)        // control kein string
  evalErr(t, `(format nil "~z" 1)`)   // unbekannte Direktive
  evalErr(t, `(format)`)              // zu wenige argumente
}

// TestFormatT prüft dest=t: schreibt auf stdout, Rückgabe nil.
func TestFormatT(t *testing.T) {
  // stdout abfangen
  old := os.Stdout
  r, w, _ := os.Pipe()
  os.Stdout = w
  defer func() { os.Stdout = old }()

  got, err := evalStr(`(format t "~a: ~d~%" "Antwort" 42)`)
  w.Close()
  out, _ := io.ReadAll(r)
  os.Stdout = old

  if err != nil {
    t.Fatalf("eval Fehler: %v", err)
  }
  if got.Type != NIL {
    t.Errorf("dest=t soll nil zurückgeben, got %s", got)
  }
  if string(out) != "Antwort: 42\n" {
    t.Errorf("stdout = %q, will %q", string(out), "Antwort: 42\n")
  }
}

// TestFormatAppendString: dest=String → angehängt.
func TestFormatAppendString(t *testing.T) {
  evalEq(t, `(format "x=" "~a" 5)`, `"x=5"`)
}
