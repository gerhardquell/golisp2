//**********************************************************************
//  lib/ring_container_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Opus 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261123
//**********************************************************************
// Zyklen über Vektoren und Hash-Tabellen (0.38.2): Drucker markiert statt
// fatalem Stack-Overflow, equal?/Hash-Schlüssel laufen durch (Identität),
// tiefe Verschachtelung über Vektoren endet im Tiefenlimit.
//**********************************************************************

package lib

import (
  "strings"
  "testing"
)

// zyklusFaelle: jeder Ausdruck baut eine zirkuläre Struktur und liefert sie.
var zyklusFaelle = map[string]string{
  "vektor-in-sich": `(define v (make-array 2 :initial-element 0)) (aset v 0 v) v`,
  "hash-in-sich":   `(define h (make-hash-table)) (puthash :a h h) h`,
  "liste-vektor": `(define v (make-array 1 :initial-element 0))
                   (define l (list 1 v)) (aset v 0 l) l`,
  "hash-liste": `(define h (make-hash-table))
                 (define l (list 1 h)) (puthash :a h l) l`,
}

func TestZyklusContainerDrucken(t *testing.T) {
  erwartet := map[string]string{
    "vektor-in-sich": "#(#<…zirkulär> 0)",
    "hash-in-sich":   "#<hash-table>",
    "liste-vektor":   "(1 #((1 #<…zirkulär>)))",
    "hash-liste":     "(1 #<hash-table>)",
  }
  for name, src := range zyklusFaelle {
    c, err := evalStr(src)
    if err != nil {
      t.Fatalf("%s: %v", name, err)
    }
    if got := c.String(); got != erwartet[name] {
      t.Errorf("%s: got %q, want %q", name, got, erwartet[name])
    }
  }
}

// Vektor zweimal in einer Liste, aber nicht zirkulär: normal drucken.
func TestGeteilterVektorNormal(t *testing.T) {
  c, err := evalStr(`(define v (vector 1 2)) (list v v (vector v v))`)
  if err != nil {
    t.Fatal(err)
  }
  if got, want := c.String(), "(#(1 2) #(1 2) #(#(1 2) #(1 2)))"; got != want {
    t.Errorf("got %q, want %q", got, want)
  }
}

// equal? und equal-Hash-Schlüssel vergleichen Vektoren/Hash-Tabellen nach
// Identität (CL equal) — sie betreten Container nicht, also kein Zyklus.
func TestZyklusContainerEqualUndSchluessel(t *testing.T) {
  for name, src := range zyklusFaelle {
    for _, prüf := range []string{
      `(equal? x x)`,
      `(equal? (list x) (list x))`,
      `(let ((h (make-hash-table :test 'equal))) (puthash (list x) h 1) (gethash (list x) h))`,
    } {
      // letzter Ausdruck jedes Falls ist der Variablenname
      var_ := strings.Fields(src)[len(strings.Fields(src))-1]
      c, err := evalStr(src + " (define x " + var_ + ") " + prüf)
      if err != nil {
        t.Fatalf("%s %s: %v", name, prüf, err)
      }
      if c.String() != "t" && c.String() != "1" {
        t.Errorf("%s %s: got %s", name, prüf, c)
      }
    }
  }
}

// tiefeVektoren: 100000 ineinander geschachtelte Vektoren, abwechselnd
// mit Listen, iterativ gebaut.
func tiefeVektoren(n int, mitListen bool) *Cell {
  c := MakeInt(0)
  for i := 0; i < n; i++ {
    c = makeVector([]*Cell{c})
    if mitListen {
      c = Cons(c, MakeNil())
    }
  }
  return c
}

func TestTiefeVektorVerschachtelung(t *testing.T) {
  for _, mitListen := range []bool{false, true} {
    s := tiefeVektoren(100000, mitListen).String()
    if !strings.Contains(s, "…") {
      t.Errorf("mitListen=%v: Tiefenlimit-Markierung fehlt", mitListen)
    }
  }
}

func grosseListe(n int) *Cell {
  items := make([]*Cell, n)
  for i := range items {
    items[i] = MakeInt(int64(i))
  }
  return SliceToCell(items)
}

func BenchmarkDruckListe1e5(b *testing.B) {
  l := grosseListe(100000)
  for i := 0; i < b.N; i++ {
    _ = l.String()
  }
}

func BenchmarkDruckVektor1e5(b *testing.B) {
  items := make([]*Cell, 100000)
  for i := range items {
    items[i] = MakeInt(int64(i))
  }
  v := makeVector(items)
  for i := 0; i < b.N; i++ {
    _ = v.String()
  }
}

func BenchmarkEqualListe1e5(b *testing.B) {
  x, y := grosseListe(100000), grosseListe(100000)
  for i := 0; i < b.N; i++ {
    if ok, err := cellEqual(x, y); !ok || err != nil {
      b.Fatal(ok, err)
    }
  }
}
