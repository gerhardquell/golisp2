//**********************************************************************
//  lib/jsoncell_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260807
//**********************************************************************
// Round-Trip-Tests fuer JSON <-> Cell (Spec TODO.md §6 und §8.1).
// Bewusste Asymmetrien (nicht reparieren): false und null werden beide
// zu Nil; eine leere Liste ist Nil ist null, nicht {} oder [].
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
    {"false", "null"},                    // Asymmetrie: false -> Nil -> null
    {"42", "42"},                         // Ganzzahl ohne .0
    {"3.14", "3.14"},
    {"-7", "-7"},
    {`"hallo"`, `"hallo"`},
    {`"mit \"quote\""`, `"mit \"quote\""`},
    {`"Umlaut äöü"`, `"Umlaut äöü"`},
    {"[1,2,3]", "[1,2,3]"},
    {"[]", "null"},                       // leere Liste = Nil = null
    {`{"a":1}`, `{"a":1}`},
    {`{"b":2,"a":1}`, `{"a":1,"b":2}`},   // json.Marshal sortiert Keys
    {`{"a":{"b":2}}`, `{"a":{"b":2}}`},   // verschachtelte Alists
    {"[[1,2],[3,4]]", "[[1,2],[3,4]]"},   // Arrays von Arrays
    {`[["a",1],["b",2]]`, `[["a",1],["b",2]]`}, // proper lists -> Arrays!
  }
  for _, tc := range cases {
    roundTrip(t, tc.in, tc.want)
  }
}

// Die Objekt-Regel von der Lisp-Seite: dotted pairs -> Objekt,
// proper lists -> Array. Der Cdr-Typ entscheidet.
func TestCellToJSONObjektRegel(t *testing.T) {
  dotted := List(Cons(MakeStr("a"), MakeNum(1)), Cons(MakeStr("b"), MakeNum(2)))
  out, err := CellToJSON(dotted)
  if err != nil {
    t.Fatalf("CellToJSON dotted: %v", err)
  }
  if string(out) != `{"a":1,"b":2}` {
    t.Errorf("dotted alist = %s, erwartet {\"a\":1,\"b\":2}", out)
  }

  proper := List(List(MakeStr("a"), MakeNum(1)), List(MakeStr("b"), MakeNum(2)))
  out, err = CellToJSON(proper)
  if err != nil {
    t.Fatalf("CellToJSON proper: %v", err)
  }
  if string(out) != `[["a",1],["b",2]]` {
    t.Errorf("proper list = %s, erwartet [[\"a\",1],[\"b\",2]]", out)
  }

  // ATOM-Key im dotted pair -> Objekt, Symbolname als Key
  atomKey := List(Cons(MakeAtom("rot"), MakeNum(1)))
  out, err = CellToJSON(atomKey)
  if err != nil {
    t.Fatalf("CellToJSON atomKey: %v", err)
  }
  if string(out) != `{"rot":1}` {
    t.Errorf("atom-key alist = %s, erwartet {\"rot\":1}", out)
  }

  // t und sonstige Atome als Wert
  out, err = CellToJSON(List(MakeAtom("t"), MakeAtom("foo")))
  if err != nil {
    t.Fatalf("CellToJSON t: %v", err)
  }
  if string(out) != `[true,"foo"]` {
    t.Errorf("(t foo) = %s, erwartet [true,\"foo\"]", out)
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
