//**********************************************************************
//  lib/cformat_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260815
//**********************************************************************

package lib

import (
  "strings"
  "testing"
)

func TestCformatToCL(t *testing.T) {
  cases := []struct{ c, cl string }{
    {"%d", "~D"},
    {"%i", "~D"},
    {"%5d", "~5D"},
    {"%05d", "~5,'0D"},
    {"%+d", "~@D"},
    {"%-5d", "~5A"},
    {"%x", "~X"},
    {"%o", "~O"},
    {"%s", "~A"},
    {"%10s", "~10:A"},
    {"%-10s", "~10A"},
    {"%f", "~,6F"},
    {"%.2f", "~,2F"},
    {"%8.2f", "~8,2F"},
    {"%08.2f", "~8,2,,,'0F"},
    {"%e", "~,6E"},
    {"%g", "~G"},
    {"%c", "~C"},
    {"%%", "%"},
    {"100%%", "100%"},
    {"~nix", "~~nix"}, // literal ~ escaped
  }
  for _, tc := range cases {
    got, err := cformatToCL(tc.c)
    if err != nil { t.Fatalf("cformatToCL(%q): %v", tc.c, err) }
    if got != tc.cl {
      t.Errorf("cformatToCL(%q) = %q, erwartet %q", tc.c, got, tc.cl)
    }
  }
}

func TestCformatToCLErrors(t *testing.T) {
  for _, bad := range []string{"%", "%q", "%.3s", "%-8.2f", "%5c"} {
    if _, err := cformatToCL(bad); err == nil {
      t.Errorf("cformatToCL(%q): Fehler erwartet, keiner gekommen", bad)
    }
  }
}

func TestSprintf(t *testing.T) {
  env := BaseEnv()
  cases := []struct{ in, want string }{
    {`(sprintf "%d" 42)`, "42"},
    {`(sprintf "%5d" 42)`, "   42"},
    {`(sprintf "%05d" 42)`, "00042"},
    {`(sprintf "%-5d|" 42)`, "42   |"},
    {`(sprintf "%x" 255)`, "ff"},
    {`(sprintf "%o" 8)`, "10"},
    {`(sprintf "%s %s" "hallo" "welt")`, "hallo welt"},
    {`(sprintf "%10s|" "hi")`, "        hi|"},
    {`(sprintf "%-10s|" "hi")`, "hi        |"},
    {`(sprintf "%.2f" 3.14159)`, "3.14"},
    {`(sprintf "%8.2f" 3.14159)`, "    3.14"},
    {`(sprintf "%f" 2.5)`, "2.500000"},
    {`(sprintf "%c" 65)`, "A"},
    {`(sprintf "%c" "ä")`, "ä"},
    {`(sprintf "100%%")`, "100%"},
    // Unicode: Padding zählt Runen, nicht Bytes
    {`(sprintf "%6s|" "hällö")`, " hällö|"},
    {`(sprintf "%d Stücke à %.2f EUR" 3 1.5)`, "3 Stücke à 1.50 EUR"},
  }
  for _, tc := range cases {
    got, err := LoadString(tc.in, env)
    if err != nil { t.Fatalf("%s: %v", tc.in, err) }
    if got.Type != STRING || got.Val != tc.want {
      t.Errorf("%s = %q, erwartet %q", tc.in, got.Val, tc.want)
    }
  }
}

func TestPrintf(t *testing.T) {
  env := BaseEnv()
  var buf strings.Builder
  SetOutputWriter(func(s string) error { buf.WriteString(s); return nil })
  defer ResetOutputWriter()

  if _, err := LoadString(`(printf "[%5d] %-6s| %.1f" 42 "ab" 2.26)`, env); err != nil {
    t.Fatalf("printf: %v", err)
  }
  want := "[   42] ab    | 2.3"
  if buf.String() != want {
    t.Errorf("printf = %q, erwartet %q", buf.String(), want)
  }
}

func TestFprintf(t *testing.T) {
  env := BaseEnv()
  var outBuf, errBuf strings.Builder
  SetOutputWriter(func(s string) error { outBuf.WriteString(s); return nil })
  SetErrorWriter(func(s string) error { errBuf.WriteString(s); return nil })
  defer ResetOutputWriter()
  defer ResetErrorWriter()

  if _, err := LoadString(`(fprintf "sys-stdout" "out:%d" 1)`, env); err != nil {
    t.Fatalf("fprintf stdout: %v", err)
  }
  if outBuf.String() != "out:1" {
    t.Errorf("fprintf sys-stdout = %q, erwartet %q", outBuf.String(), "out:1")
  }
  if _, err := LoadString(`(fprintf "sys-stderr" "err:%s" "x")`, env); err != nil {
    t.Fatalf("fprintf stderr: %v", err)
  }
  if errBuf.String() != "err:x" {
    t.Errorf("fprintf sys-stderr = %q, erwartet %q", errBuf.String(), "err:x")
  }

  // Datei: append-Semantik
  dir := t.TempDir()
  if _, err := LoadString(`(set-working-directory "`+dir+`")`, env); err != nil {
    t.Fatalf("set-working-directory: %v", err)
  }
  defer LoadString(`(set-working-directory "")`, env)
  if _, err := LoadString(`(fprintf "log.txt" "zeile%d\n" 1)`, env); err != nil {
    t.Fatalf("fprintf datei 1: %v", err)
  }
  if _, err := LoadString(`(fprintf "log.txt" "zeile%d\n" 2)`, env); err != nil {
    t.Fatalf("fprintf datei 2: %v", err)
  }
  got, err := LoadString(`(file-read "log.txt")`, env)
  if err != nil { t.Fatalf("file-read: %v", err) }
  if got.Val != "zeile1\nzeile2\n" {
    t.Errorf("fprintf-Datei = %q, erwartet %q", got.Val, "zeile1\nzeile2\n")
  }
}

func TestSscanf(t *testing.T) {
  env := BaseEnv()

  got, err := LoadString(`(sscanf "42 abc 3.5" "%d %s %f")`, env)
  if err != nil { t.Fatalf("sscanf: %v", err) }
  if got.String() != `(42 "abc" 3.5)` {
    t.Errorf("sscanf = %s, erwartet (42 \"abc\" 3.5)", got)
  }

  // Literale müssen matchen
  got, err = LoadString(`(sscanf "x=7,y=8" "x=%d,y=%d")`, env)
  if err != nil { t.Fatalf("sscanf literale: %v", err) }
  if got.String() != `(7 8)` {
    t.Errorf("sscanf literale = %s, erwartet (7 8)", got)
  }

  // %c liest genau 1 Rune, kein Whitespace-Skip
  got, err = LoadString(`(sscanf "ab" "%c%c")`, env)
  if err != nil { t.Fatalf("sscanf %%c: %v", err) }
  if got.String() != `("a" "b")` {
    t.Errorf("sscanf %%c = %s, erwartet (\"a\" \"b\")", got)
  }

  // Unicode-Rune
  got, err = LoadString(`(sscanf "äx" "%c%c")`, env)
  if err != nil { t.Fatalf("sscanf unicode: %v", err) }
  if got.String() != `("ä" "x")` {
    t.Errorf("sscanf unicode = %s", got)
  }

  // Literal-Mismatch → Fehler
  if _, err = LoadString(`(sscanf "x=abc" "x=%d")`, env); err == nil {
    t.Errorf("sscanf x=abc mit %%d: Fehler erwartet")
  }
  if _, err = LoadString(`(sscanf "y=7" "x=%d")`, env); err == nil {
    t.Errorf("sscanf Literal-Mismatch: Fehler erwartet")
  }

  // negative + Float-Exponent
  got, err = LoadString(`(sscanf "-3 1.5e2" "%d %f")`, env)
  if err != nil { t.Fatalf("sscanf neg/exp: %v", err) }
  if got.String() != `(-3 150)` {
    t.Errorf("sscanf neg/exp = %s, erwartet (-3 150)", got)
  }
}
