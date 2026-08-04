//**********************************************************************
//  lib/reader_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260616
//**********************************************************************
// Charakterisierungstests für den Reader.
// Zweck: Sicherheitsnetz vor dem eval.go-Split (Todo #1).
// Sie halten das IST-Verhalten fest, nicht ein gewünschtes SOLL –
// Abweichungen sind Bugs, die explizit markiert werden.
//**********************************************************************

package lib

import "testing"

// assertRead prüft, dass src zu dem Cell-Baum parst, dessen
// String-Repräsentation want entspricht. Cell.String() ist die
// kanonische Form: sie ist unabhängig von Pointer-Identität und
// deckt Strukturfehler (falsche Type, kaputte Cons-Kette) zuverlässig auf.
func assertRead(t *testing.T, src, want string) {
  t.Helper()
  got, err := Read(src)
  if err != nil {
    t.Fatalf("Read(%q) Fehler: %v", src, err)
  }
  if got.String() != want {
    t.Errorf("Read(%q) = %q, want %q", src, got.String(), want)
  }
}

// assertReadErr prüft, dass src einen Fehler liefert.
func assertReadErr(t *testing.T, src string) {
  t.Helper()
  _, err := Read(src)
  if err == nil {
    t.Errorf("Read(%q) sollte Fehler geben, lieferte aber nil", src)
  }
}

func TestReadAtoms(t *testing.T) {
  cases := []struct{ src, want string }{
    {"foo", "foo"},
    {"+", "+"},
    {"-", "-"},            // "-" ist Symbol, keine Zahl (ParseFloat schlägt fehl)
    {"a-b", "a-b"},
    {"car", "car"},
  }
  for _, c := range cases {
    assertRead(t, c.src, c.want)
  }
}

func TestReadNumbers(t *testing.T) {
  cases := []struct{ src, want string }{
    {"42", "42"},
    {"3.14", "3.14"},
    {"0", "0"},
    {"-5", "-5"},
    {"100", "100"},
  }
  for _, c := range cases {
    assertRead(t, c.src, c.want)
  }
}

func TestReadStrings(t *testing.T) {
  cases := []struct{ src, want string }{
    {`"hallo"`, `"hallo"`},
    {`"hallo welt"`, `"hallo welt"`},
    {`"mit\nnewline"`, `"mit\nnewline"`},   // \n → echter Zeilenumbruch
    {`"tab\there"`, `"tab\there"`},
    {`"quote\"inside"`, `"quote\"inside"`},
    {`"back\\slash"`, `"back\\slash"`},
    {`""`, `""`},                            // leerer String
  }
  for _, c := range cases {
    assertRead(t, c.src, c.want)
  }
}

func TestReadLists(t *testing.T) {
  cases := []struct{ src, want string }{
    {"(a b c)", "(a b c)"},
    {"()", "()"},                        // leere Liste → Singleton-Nil (Stringer: "()")
    {"(1 2 3)", "(1 2 3)"},
    {"(a (b c) d)", "(a (b c) d)"},      // Verschachtelung
    {"(  a   b  )", "(a b)"},            // Whitespace toleriert
    {"((x))", "((x))"},                  // tiefere Verschachtelung
  }
  for _, c := range cases {
    assertRead(t, c.src, c.want)
  }
}

func TestReadNil(t *testing.T) {
  // nil/NIL → Singleton-Nil-Cell (Type: NIL), Stringer rendert "()".
  // eq-Pointer-Gleichheit mit (list) ist gewollt (siehe CLAUDE.md).
  assertRead(t, "nil", "()")
  assertRead(t, "NIL", "()")
  // Gemischte Case wie "Nil"/"nIl" sind Symbole (IST-Verhalten).
  assertRead(t, "Nil", "Nil")
  assertRead(t, "nIl", "nIl")
}

func TestReadQuote(t *testing.T) {
  // Drucker gibt die Reader-Abkürzung aus (CL): (quote x) → 'x
  assertRead(t, "'x", "'x")
  assertRead(t, "'(a b)", "'(a b)")
  assertRead(t, "''x", "''x")  // geschachteltes Quote
}

func TestReadQuasiquote(t *testing.T) {
  assertRead(t, "`x", "(quasiquote x)")
  assertRead(t, ",x", "(unquote x)")
  assertRead(t, ",@x", "(unquote-splice x)")
  assertRead(t, "`(a ,b)", "(quasiquote (a (unquote b)))")
}

func TestReadDispatch(t *testing.T) {
  assertRead(t, "#'foo", "(function foo)")
  assertReadErr(t, "#x")   // unbekanntes Dispatch-Zeichen
  assertReadErr(t, "#")    // EOF nach #
}

func TestReadDottedPair(t *testing.T) {
  // (a . b) → Cons(a, b)
  got, err := Read("(a . b)")
  if err != nil {
    t.Fatalf("Read(\"(a . b)\") Fehler: %v", err)
  }
  if got.Type != LIST || got.Car.String() != "a" || got.Cdr.String() != "b" {
    t.Errorf("(a . b) = %q (Car=%q Cdr=%q), want Cons(a,b)",
      got.String(), got.Car.String(), got.Cdr.String())
  }
  // Trailing-Whitespace nach cdr ist ok
  assertRead(t, "(a . b)", "(a . b)")
}

// TestReadDottedPairStrict sichert den Fix (Todo #7): früher verschluckte
// readRest nach dotted pair blind das nächste Zeichen via r.next() – Müll
// wie "(a . b x)" wurde still akzeptiert. Jetzt explizite )-Prüfung.
func TestReadDottedPairStrict(t *testing.T) {
  assertReadErr(t, "(a . b x)")   // Element nach dotted pair → Fehler
  assertReadErr(t, "(a . b . c)") // doppelte dotted pair → Fehler
  assertReadErr(t, "(a . b")      // fehlendes ) nach dotted pair
}

func TestReadComments(t *testing.T) {
  assertRead(t, "; nur Kommentar", "()")           // nur Kommentar → Nil → "()"
  assertRead(t, "a ; trailing", "a")
  assertRead(t, "(a ; Kommentar in Liste\n b)", "(a b)")
}

func TestReadWhitespaceTrim(t *testing.T) {
  assertRead(t, "  (a b)  ", "(a b)")
  assertRead(t, "\n\n42\n", "42")
}

func TestReadErrors(t *testing.T) {
  assertReadErr(t, "(a b")           // fehlendes ) / EOF in Liste
  assertReadErr(t, `"ungeschlossen`) // ungeschlossener String
  assertReadErr(t, "(")              // nacktes ( ohne Inhalt
  // Hinweis: "\" (Backslash außerhalb eines Strings) ist KEIN Fehler –
  // readAtomOrNum liest es als Symbol. Backslash ist nur innerhalb von
  // Strings als Escape special. IST-Verhalten, dokumentiert in TestReadAtoms.
  assertRead(t, `\`, `\`)
}

// TestReadNestedDeep sichert die TCO-unabhängige Reader-Rekursion:
// tiefe Listen dürfen den Stack nicht sprengen (Reader ist nicht
// trampoliniert, geht aber praktisch nie tief genug). Marker-Test.
func TestReadNestedDeep(t *testing.T) {
  // 50-fach verschachtelt: ((((( ... )))))
  src := ""
  for i := 0; i < 50; i++ {
    src += "("
  }
  src += "x"
  for i := 0; i < 50; i++ {
    src += ")"
  }
  got, err := Read(src)
  if err != nil {
    t.Fatalf("tief verschachtelt: %v", err)
  }
  // Kern muss "x" sein; 50 Hüllen drumherum.
  cur := got
  for i := 0; i < 50; i++ {
    if cur.Type != LIST || cur.Cdr.String() != "()" {
      t.Fatalf("Hülle %d kaputt: %q", i, cur.String())
    }
    cur = cur.Car
  }
  if cur.String() != "x" {
    t.Errorf("Kern = %q, want x", cur.String())
  }
}

func TestReaderStampsSrcLine(t *testing.T) {
  src := "(defun f (x)\n  (* x x))\n(defun g () 1)"
  forms, err := ReadAll(src)
  if err != nil {
    t.Fatalf("ReadAll: %v", err)
  }
  // forms = (form1 form2), beide Listen-Cells
  f1 := forms.Car
  f2 := forms.Cdr.Car
  if f1.SrcLine != 1 {
    t.Fatalf("form1 SrcLine = 1 erwartet, got %d", f1.SrcLine)
  }
  // form2 beginnt auf Zeile 3 (nach \n nach ))
  if f2.SrcLine != 3 {
    t.Fatalf("form2 SrcLine = 3 erwartet, got %d", f2.SrcLine)
  }
}

func TestReaderShebang(t *testing.T) {
  // Shebang-Zeile (#!/usr/local/bin/golisp2) wird wie ein Kommentar
  // übersprungen — Skripte sind direkt ausführbar.
  src := "#!/usr/local/bin/golisp2\n(+ 1 2)\n(* 3 4)"
  forms, err := ReadAll(src)
  if err != nil {
    t.Fatalf("ReadAll mit Shebang: %v", err)
  }
  if forms.Car.String() != "(+ 1 2)" {
    t.Errorf("form1 = %q, want (+ 1 2)", forms.Car.String())
  }
  if forms.Cdr.Car.String() != "(* 3 4)" {
    t.Errorf("form2 = %q, want (* 3 4)", forms.Cdr.Car.String())
  }
  // Zeilenzählung darf nicht verrutschen: form2 beginnt auf Zeile 3
  if forms.Cdr.Car.SrcLine != 3 {
    t.Errorf("form2 SrcLine = %d, want 3", forms.Cdr.Car.SrcLine)
  }

  // Shebang ohne nachfolgende Form → nur NIL (kein Fehler)
  only, err := ReadAll("#!/usr/bin/env golisp2\n")
  if err != nil {
    t.Fatalf("ReadAll nur Shebang: %v", err)
  }
  if only.String() != "()" {
    t.Errorf("nur Shebang = %q, want ()", only.String())
  }

  // #' darf nicht brechen — kein Rückfall in Kommentar-Behandlung
  fn, err := Read("#'car")
  if err != nil {
    t.Fatalf("#'car nach Shebang-Änderung: %v", err)
  }
  if fn.Car.String() != "function" {
    t.Errorf("#'car = %q, want (function car)", fn.String())
  }
}
