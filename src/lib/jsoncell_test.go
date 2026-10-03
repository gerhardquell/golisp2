//**********************************************************************
//  lib/jsoncell_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260807
//**********************************************************************
// Round-Trip-Tests fuer JSON <-> Cell (Abbildung seit 20261003):
// Objekt <-> Hash-Tabelle, Array <-> Liste, true <-> t, null <-> :null,
// false -> (), () -> []; :false kodiert zu false. Bewusste Asymmetrie:
// false kommt als [] zurueck. Alists sind KEINE Objekte mehr.
//**********************************************************************

package lib

import (
  "strings"
  "testing"
)

// roundTrip prueft JSON -> Cell -> JSON. expected darf sich von input
// unterscheiden (dokumentierte Asymmetrien, Map-Key-Sortierung).
func roundTrip(t *testing.T, input, expected string) {
  t.Helper()
  c, err := JSONToCell([]byte(input))
  if err != nil {
    t.Fatalf("JSONToCell(%q): %v", input, err)
  }
  out, err := CellToJSON(c)
  if err != nil {
    t.Fatalf("CellToJSON(%q): %v", input, err)
  }
  if string(out) != expected {
    t.Errorf("RoundTrip(%q) = %s, erwartet %s", input, out, expected)
  }
}

func TestJSONCellRoundTrip(t *testing.T) {
  cases := []struct{ in, want string }{
    {"null", "null"},
    {"true", "true"},
    {"false", "[]"},                      // Asymmetrie: false -> () -> []
    {"42", "42"},                         // Ganzzahl ohne .0
    {"3.14", "3.14"},
    {"-7", "-7"},
    {`"hallo"`, `"hallo"`},
    {`"mit \"quote\""`, `"mit \"quote\""`},
    {`"Umlaut äöü"`, `"Umlaut äöü"`},
    {`"<b>&</b>"`, `"<b>&</b>"`},         // kein HTML-Escaping
    {"[1,2,3]", "[1,2,3]"},
    {"[]", "[]"},
    {"{}", "{}"},
    {`{"a":1}`, `{"a":1}`},
    {`{"b":2,"a":1}`, `{"a":1,"b":2}`},   // Keys sortiert
    {`{"a":{"b":2}}`, `{"a":{"b":2}}`},
    {`{"a":["x"]}`, `{"a":["x"]}`},       // war mit Alists: [["a","x"]]
    {`{"a":"x","b":[1,2]}`, `{"a":"x","b":[1,2]}`}, // war: Fehler
    {`{"aktiv":false,"n":null}`, `{"aktiv":[],"n":null}`},
    {"[[1,2],[3,4]]", "[[1,2],[3,4]]"},
    {`[["a",1],["b",2]]`, `[["a",1],["b",2]]`},
    {`[{"k":1},{"k":2}]`, `[{"k":1},{"k":2}]`},
  }
  for _, tc := range cases {
    roundTrip(t, tc.in, tc.want)
  }
}

func TestJSONToCellAbbildung(t *testing.T) {
  cases := []struct{ in, want string }{
    {"null", ":null"},
    {"true", "t"},
    {"false", "()"},
    {"[]", "()"},
    {`["a",1]`, `("a" 1)`},
  }
  for _, tc := range cases {
    c, err := JSONToCell([]byte(tc.in))
    if err != nil {
      t.Fatalf("JSONToCell(%q): %v", tc.in, err)
    }
    if c.String() != tc.want {
      t.Errorf("JSONToCell(%q) = %s, erwartet %s", tc.in, c, tc.want)
    }
  }
  c, err := JSONToCell([]byte(`{"a":1}`))
  if err != nil || c.Type != HASHTABLE || len(c.Ht.m) != 1 {
    t.Errorf(`{"a":1} soll Hash-Tabelle mit 1 Eintrag sein, got %v %v`, c, err)
  }
  if c != nil && c.Type == HASHTABLE && c.Ht.test != "equal" {
    t.Errorf("Hash-Test = %q, erwartet equal", c.Ht.test)
  }
}

func TestCellToJSONLispSeite(t *testing.T) {
  ht := func(pairs ...*Cell) *Cell {
    c, _ := fnMakeHashTable([]*Cell{MakeAtom(":test"), MakeAtom("equal")})
    for i := 0; i+1 < len(pairs); i += 2 {
      if _, err := fnPuthash([]*Cell{pairs[i], c, pairs[i+1]}); err != nil {
        t.Fatal(err)
      }
    }
    return c
  }
  ok := []struct {
    name string
    c    *Cell
    want string
  }{
    {"t und Symbol", List(MakeAtom("t"), MakeAtom("foo")), `[true,"foo"]`},
    {":false/:null", List(MakeAtom(":false"), MakeAtom(":null")), `[false,null]`},
    {"nil", MakeNil(), `[]`},
    {"Symbol-Key", ht(MakeAtom("rot"), MakeNum(1)), `{"rot":1}`},
    {"Hash mit Array", ht(MakeStr("m"), List(MakeNum(1))), `{"m":[1]}`},
  }
  for _, tc := range ok {
    out, err := CellToJSON(tc.c)
    if err != nil {
      t.Errorf("%s: %v", tc.name, err)
      continue
    }
    if string(out) != tc.want {
      t.Errorf("%s = %s, erwartet %s", tc.name, out, tc.want)
    }
  }
  bad := []struct {
    name string
    c    *Cell
  }{
    // Alist ist kein Objekt mehr: dotted pair = improper Liste
    {"dotted Alist", List(Cons(MakeStr("a"), MakeNum(1)))},
    {"Zahl-Key", ht(MakeNum(1), MakeNum(2))},
    {"Key-Kollision", ht(MakeStr("a"), MakeNum(1), MakeAtom("a"), MakeNum(2))},
    {"Funktion", makeFn(fnNow)},
  }
  for _, tc := range bad {
    if out, err := CellToJSON(tc.c); err == nil {
      t.Errorf("%s: Fehler erwartet, got %s", tc.name, out)
    }
  }
}

func TestJSONCellTiefenbegrenzung(t *testing.T) {
  // 64 Ebenen sind erlaubt, 65 muessen fehlschlagen.
  tiefe := func(n int) string { return strings.Repeat("[", n) + "1" + strings.Repeat("]", n) }

  if _, err := JSONToCell([]byte(tiefe(64))); err != nil {
    t.Errorf("Tiefe 64 muss gehen: %v", err)
  }
  if _, err := JSONToCell([]byte(tiefe(65))); err == nil {
    t.Error("Tiefe 65 muss Fehler liefern")
  }

  deepCell, err := JSONToCell([]byte(tiefe(64)))
  if err != nil {
    t.Fatalf("Aufbau deepCell: %v", err)
  }
  if _, err := CellToJSON(deepCell); err != nil {
    t.Errorf("CellToJSON Tiefe 64 muss gehen: %v", err)
  }
  wrapped := List(deepCell) // 65 Ebenen
  if _, err := CellToJSON(wrapped); err == nil {
    t.Error("CellToJSON Tiefe 65 muss Fehler liefern")
  } else if !strings.Contains(err.Error(), "Tiefe 64 überschritten") {
    t.Errorf("Fehlermeldung = %q, erwartet 'Tiefe 64 überschritten'", err)
  }
}

func TestJSONToCellKaputt(t *testing.T) {
  if _, err := JSONToCell([]byte("{kaputt")); err == nil {
    t.Error("kaputtes JSON muss Fehler liefern")
  }
}

// Lisp-Seite: json-parse / json-encode.
func TestJSONPrimitiven(t *testing.T) {
  evalEq(t, `(gethash "model" (json-parse "{\"model\":\"glm\",\"n\":2}"))`, `"glm"`)
  evalEq(t, `(json-encode (json-parse "{\"a\":[1,2],\"b\":null}"))`, `"{\"a\":[1,2],\"b\":null}"`)
  evalEq(t, `(json-parse "[1,\"x\",true,false,null]")`, `(1 "x" t () :null)`)
  evalEq(t, `(json-encode (list 1 "x" t () :false :null))`, `"[1,\"x\",true,[],false,null]"`)
  evalEq(t, `(let ((h (make-hash-table :test 'equal)))
                (puthash "bare" h :false)
                (puthash "messages" h ())
                (json-encode h))`, `"{\"bare\":false,\"messages\":[]}"`)
  evalEq(t, `(hash-table-count (json-parse "{}"))`, "0")
  for _, src := range []string{
    `(json-parse)`,
    `(json-parse 5)`,
    `(json-parse "{kaputt")`,
    `(json-parse "1 2")`,             // Müll nach dem Wert
    `(json-encode)`,
    `(json-encode 1 2)`,
    `(json-encode (list (cons "a" 1)))`, // Alist ist kein Objekt
  } {
    evalErr(t, src)
  }
}
