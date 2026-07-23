//**********************************************************************
//  lib/primitives_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260616
//**********************************************************************
// Charakterisierungstests für eingebaute Primitive (primitives.go,
// stringfuncs.go, fileio.go). Nutzt evalEq/evalErr aus eval_test.go.
// Arithmetik-Grundlagen + Booleans + eq/equal? sind in eval_test.go
// abgedeckt, hier Fokus auf: mod/abs, Typ-Prädikate, Listen-Edges,
// String-Funktionen, fileio, gensym/error.
//**********************************************************************

package lib

import (
  "os"
  "path/filepath"
  "strings"
  "testing"
)

// --- Arithmetik-Erweiterungen (mod/abs, nicht in eval_test) ---

func TestPrimitiveModAbs(t *testing.T) {
  evalEq(t, `(mod 10 3)`, "1")
  evalEq(t, `(mod 10 2)`, "0")
  evalEq(t, `(remainder 10 3)`, "1")   // Alias für mod
  evalEq(t, `(abs -5)`, "5")
  evalEq(t, `(abs 5)`, "5")
  evalEq(t, `(abs 0)`, "0")
  evalErr(t, `(mod 10 0)`)              // Division durch Null
  evalErr(t, `(mod 10)`)                // zu wenig Args
}

// --- Typ-Prädikate ---

func TestPrimitiveTypePredicates(t *testing.T) {
  cases := []struct{ src, want string }{
    {`(string? "x")`, "t"},
    {`(string? 5)`, "()"},
    {`(number? 5)`, "t"},
    {`(number? 5.5)`, "t"},
    {`(number? "x")`, "()"},
    {`(list? (list 1 2))`, "t"},
    {`(list? 5)`, "()"},
    {`(symbol? 'foo)`, "t"},
    {`(symbol? 5)`, "()"},
    {`(atom? 'foo)`, "t"},
    {`(atom? 5)`, "t"},
    {`(atom? (list 1))`, "()"},
    {`(null? '())`, "t"},
    {`(null? 5)`, "()"},
    // eq? = Pointer-Identität wie eq: zwei 'foo-Instanzen sind NICHT eq?
    {`(eq? 'foo 'foo)`, "()"},
  }
  for _, c := range cases {
    evalEq(t, c.src, c.want)
  }
}

// --- Listen-Primitive mit Edge-Cases ---

func TestPrimitiveListOps(t *testing.T) {
  evalEq(t, `(car '(1 2 3))`, "1")
  evalEq(t, `(cdr '(1 2 3))`, "(2 3)")
  evalEq(t, `(cdr '(1))`, "()")          // Cdr von Ein-Element-Liste = Nil
  evalEq(t, `(cons 1 '(2 3))`, "(1 2 3)")
  evalEq(t, `(cons 1 2)`, "(1 . 2)")     // Dotted pair
  evalEq(t, `(list 1 2 3)`, "(1 2 3)")
  evalEq(t, `(list)`, "()")
  evalEq(t, `(atom '(1 2))`, "()")
  evalEq(t, `(null '())`, "t")
  evalEq(t, `(apply + (list 1 2 3))`, "6")
  evalEq(t, `(funcall + 1 2 3)`, "6")
}

// TestPrimitiveAppendCL: append folgt Common-Lisp-Semantik (Listen-
// konkatenation, variadisch). Bisher war append single-element ("snoc"),
// was swank.lisp/flatten (CL-Stil) silent gebrochen hat.
func TestPrimitiveAppendCL(t *testing.T) {
  // 2 Listen konkatenieren (flach, nicht geschachtelt)
  evalEq(t, `(append '(1 2) '(3 4))`, "(1 2 3 4)")
  evalEq(t, `(append '(1) '(2))`, "(1 2)")
  // variadisch: >2 Listen
  evalEq(t, `(append '(1) '(2) '(3))`, "(1 2 3)")
  evalEq(t, `(append '(1 2) '(3) '(4 5 6))`, "(1 2 3 4 5 6)")
  // leere Liste als Argument
  evalEq(t, `(append '() '(1 2))`, "(1 2)")
  evalEq(t, `(append '(1 2) '())`, "(1 2)")
  // 0 / 1 Argument
  evalEq(t, `(append)`, "()")
  evalEq(t, `(append '(1 2 3))`, "(1 2 3)")
  // ursprüngliche Single-Element-Nutzung weiterhin möglich: Element
  // als letzte Liste gibt Cdr vor -> (append lst (list x)) hängt x an.
  evalEq(t, `(append '(1 2) (list 3))`, "(1 2 3)")
  // keyword-Listen wie in swank.lisp: events ist Liste-von-Events,
  // Return-Event als einzelne Liste angehängt -> flache Event-Liste.
  evalEq(t, `(append (list (list :write-string "x" :repl-result))
                     (list (list :return (list :ok (list)) 17)))`,
    `((:write-string "x" :repl-result) (:return (:ok ()) 17))`)
}

// TestPrimitiveListEdges dokumentiert IST-Verhalten an Listengrenzen.
func TestPrimitiveListEdges(t *testing.T) {
  // car/cdr auf leerer Liste → Fehler (NIL ist nicht LIST-Typ)
  evalErr(t, `(car '())`)
  evalErr(t, `(cdr '())`)
  // atom? auf leerer Liste → t! Leere Liste (NIL-Typ) gilt als Atom.
  // Überraschend, aber IST – because NIL ≠ LIST in der Cell-Typisierung.
  evalEq(t, `(atom? '())`, "t")
  // car auf Nicht-Liste → Fehler
  evalErr(t, `(car 5)`)
  evalErr(t, `(cdr 5)`)
}

// --- String-Funktionen ---

func TestPrimitiveStrings(t *testing.T) {
  evalEq(t, `(string-length "hello")`, "5")
  evalEq(t, `(string-length "")`, "0")
  evalEq(t, `(string-append "foo" "bar" "baz")`, `"foobarbaz"`)
  evalEq(t, `(substring "hello" 1 4)`, `"ell"`)
  evalEq(t, `(substring "hello" 0 5)`, `"hello"`)
  evalEq(t, `(string-upcase "abc")`, `"ABC"`)
  evalEq(t, `(string-downcase "ABC")`, `"abc"`)
  evalEq(t, `(number->string 42)`, `"42"`)
  evalEq(t, `(number->string 3.14)`, `"3.14"`)
  evalErr(t, `(string-length 5)`)        // Typfehler: Zahl statt String
  evalErr(t, `(substring "hi" 0 10)`)    // Index außerhalb
  evalErr(t, `(substring "hi" 0)`)       // zu wenig Args
}

func TestPrimitiveStringConvert(t *testing.T) {
  evalEq(t, `(string->number "42")`, "42")
  evalEq(t, `(string->number "3.14")`, "3.14")
  // string->list: String → Liste von Einzelzeichen-Strings
  evalEq(t, `(string->list "ab")`, `("a" "b")`)
  // list->string: umgekehrt
  evalEq(t, `(list->string (list "a" "b"))`, `"ab"`)
}

func TestPrimitiveStringSearch(t *testing.T) {
  evalEq(t, `(string-contains "hello" "ell")`, "t")   // Atom t
  evalEq(t, `(string-contains "hello" "xyz")`, "()")
  evalEq(t, `(string-replace "foo bar foo" "foo" "X")`, `"X bar X"`)
  evalEq(t, `(string-trim "  hi  ")`, `"hi"`)
}

// --- gensym / error / memstats ---

func TestPrimitiveGensym(t *testing.T) {
  // gensym erzeugt jedes Mal ein neues, unvergleichliches Symbol
  g1, _ := evalStr(`(gensym)`)
  g2, _ := evalStr(`(gensym)`)
  if g1 == nil || g2 == nil {
    t.Fatal("gensym: nil zurück")
  }
  if g1.String() == g2.String() {
    t.Errorf("gensym nicht eindeutig: %s == %s", g1, g2)
  }
  // gensym-Symbole sind ungleich
  evalEq(t, `(eq (gensym) (gensym))`, "()")
}

// TestPrimitiveGensymArgs: CLHS — (gensym &optional x), x = String (Prefix)
// oder Integer (Suffix). Bug 20260723: Arg wurde mit "keine Argumente
// erwartet" abgelehnt, my-reduce-Makro brach.
func TestPrimitiveGensymArgs(t *testing.T) {
  // String-Prefix
  g, err := evalStr(`(gensym "ACC")`)
  if err != nil {
    t.Fatalf(`(gensym "ACC") Fehler: %v`, err)
  }
  if !strings.HasPrefix(g.String(), "ACC") {
    t.Errorf("Prefix ACC erwartet, got %s", g)
  }
  // Int-Suffix
  g2, err := evalStr(`(gensym 42)`)
  if err != nil {
    t.Fatalf(`(gensym 42) Fehler: %v`, err)
  }
  if !strings.HasSuffix(g2.String(), "42") {
    t.Errorf("Suffix 42 erwartet, got %s", g2)
  }
  // weiterhin eindeutig trotz gleichem Prefix
  evalEq(t, `(eq (gensym "A") (gensym "A"))`, "()")
  // Fehler bei falschem Typ
  if _, err := evalStr(`(gensym '(1 2))`); err == nil {
    t.Error("(gensym '(1 2)) sollte Fehler geben")
  }
}

func TestPrimitiveError(t *testing.T) {
  _, err := evalStr(`(error "boom")`)
  if err == nil {
    t.Fatal(`(error "boom") sollte Fehler geben`)
  }
  // error mit Symbol (nicht nur String)
  _, err2 := evalStr(`(error 'my-err)`)
  if err2 == nil {
    t.Error(`(error 'my-err) sollte Fehler geben`)
  }
}

func TestPrimitiveMemstats(t *testing.T) {
  got, err := evalStr(`(memstats)`)
  if err != nil {
    t.Fatalf("memstats Fehler: %v", err)
  }
  if got == nil || got.Type != LIST {
    t.Fatalf("memstats = %q, will Liste", got)
  }
  // Assoziationsliste: ((heapalloc . N) ...)
  if got.String() == "()" {
    t.Error("memstats liefert leere Liste")
  }
}

// --- fileio mit TempDir ---

func TestPrimitiveFileIO(t *testing.T) {
  dir := t.TempDir()
  path := filepath.Join(dir, "test.txt")
  // IST: file-write/file-append liefern den Pfad als String zurück (nicht t).
  writtenPath := `"` + path + `"`

  // schreiben
  evalEq(t, `(file-write "`+path+`" "hallo welt")`, writtenPath)
  // existiert?
  evalEq(t, `(file-exists? "`+path+`")`, "t")
  evalEq(t, `(file-exists? "`+path+`/nofile")`, "()")
  // lesen
  evalEq(t, `(file-read "`+path+`")`, `"hallo welt"`)
  // append (liefert ebenfalls Pfad)
  evalEq(t, `(file-append "`+path+`" "!")`, writtenPath)
  evalEq(t, `(file-read "`+path+`")`, `"hallo welt!"`)
  // delete (liefert t)
  evalEq(t, `(file-delete "`+path+`")`, "t")
  evalEq(t, `(file-exists? "`+path+`")`, "()")
  // Lesen nicht existierender Datei → Fehler
  evalErr(t, `(file-read "`+path+`")`)
}

// TestPrimitiveFileIOWritePerm sichert, dass file-write das Verzeichnis
// anlegt/handhabt wie vom Code vorgesehen (IST-Dokumentation).
func TestPrimitiveFileIOWriteFailure(t *testing.T) {
  // Schreiben in nicht existierendes Verzeichnis → Fehler
  evalErr(t, `(file-write "/nonexistent_dir_xyz/file.txt" "x")`)
  // Vermeide Datei-Leck: nothing to clean (write failed)
  _ = os.Remove("/nonexistent_dir_xyz/file.txt")
}

// TestPrimitivePrintReturnValue sichert, dass print/println das letzte
// Argument zurückgeben, damit der REPL nicht `()` hinter die Ausgabe druckt.
func TestPrimitivePrintReturnValue(t *testing.T) {
  evalEq(t, `(print "hello")`, `"hello"`)
  evalEq(t, `(println 1 2 3)`, "3")
  evalEq(t, `(print)`, "()")
}
