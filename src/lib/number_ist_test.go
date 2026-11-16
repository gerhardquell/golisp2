//**********************************************************************
//  lib/number_ist_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-fable-5-2
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261116
//**********************************************************************
// Charakterisierungstest zum Ganzzahl-Umbau (Spec 2026-11-16, §1 und §3):
// hält das Zahlenverhalten fest. Jede Änderung an einem Erwartungswert
// hier muss in Spec §5 („Was bricht“) stehen — die Zeile nennt dann den
// Stand vor dem Umbau im Kommentar. Wird in Schritt 6 aufgelöst.
//**********************************************************************

package lib

import "testing"

func TestNumberIst(t *testing.T) {
  cases := []struct{ src, want string }{
    // Spec §1
    {`(+ 9007199254740992 1)`, "9007199254740993"}, // §1: war 9007199254740992 (gerundet)
    // §5 Überlauf ist Fehler: (* 99999999999 99999999999) war 9.9999999998e+21,
    // (random 1.0) war 0 — beide jetzt in number_int_test.go.
    // §5 Float als Index ist Fehler: (nth 1.7 (list 1 2 3)) war () — jetzt
    // in number_index_test.go.
    {`(type-of 3)`, "integer"},
    // Druckform
    {`3`, "3"},
    {`-7`, "-7"},
    {`+3`, "3"},
    {`3.0`, "3.0"},               // §5: war 3
    {`1e3`, "1000.0"},            // §5: war 1000
    {`3.14`, "3.14"},
    {`(sqrt 16)`, "4.0"},         // §5: war 4
    {`(/ 7.0 7)`, "1.0"},         // §5: war 1
    {`(* 1.5 2)`, "3.0"},         // §5: war 3
    {`(number->string 3.0)`, `"3.0"`},  // §5: war "3"
    {`(format nil "~a" 3.0)`, `"3.0"`}, // §5: war "3"
    {`(format nil "~d" 3.0)`, `"3.0"`}, // §3.6: war "3"
    // Gleichheit
    {`(eql 1 1.0)`, "()"},                        // §5: war t
    {`(equal? 1 1.0)`, "()"},                     // §5: war t
    {`(equal? (list 1 2) (list 1.0 2.0))`, "()"}, // §5: war t
    {`(= 1 1.0)`, "t"},
    {`(eq 5 5)`, "()"},
    {`(member 1.0 (list 1 2))`, "()"},            // §5 (folgt eql): war (1 2)
    // Typen
    {`(type-of 3.0)`, "float"},      // §5: war integer
    {`(type-of 3.5)`, "float"},
    {`(typep 3.0 'integer)`, "()"}, // §5: war t
    {`(typep 3 'integer)`, "t"},
    // Rechnen
    {`(+ 1 2)`, "3"},
    {`(+ 1 2.5)`, "3.5"},
    {`(/ 7 2)`, "3.5"},
    {`(/ 6 2)`, "3"},
    {`(mod 7 2)`, "1"},
    {`(mod -3 2)`, "1"},
    {`(rem -3 2)`, "-1"},
    {`(abs -3)`, "3"},
    {`(max 1 2.5)`, "2.5"},
    {`(min 1 2.5)`, "1"},
    {`(floor 7 2)`, "3"},
    {`(floor 3.7)`, "3"},
    {`(ceiling 3.2)`, "4"},
    {`(truncate -3.7)`, "-3"},
    {`(round 2.5)`, "2"},
    {`(expt 2 10)`, "1024"},
    {`(expt 2 0.5)`, "1.4142135623730951"},
    {`(1+ 5)`, "6"},
    {`(1- 5)`, "4"},
    {`(ash 1 10)`, "1024"},
    {`(ash 1024 -3)`, "128"},
    {`(< (random 10) 10)`, "t"},
    {`(length (list 1 2 3))`, "3"},
    // Hash-Schlüssel
    {`(let ((h (make-hash-table))) (setf (gethash 1 h) 'a) (gethash 1.0 h))`, "()"}, // §5: war a
    // JSON-Rundlauf
    {`(json-encode (json-parse "[1, 2.5, 3.0]"))`, `"[1,2.5,3]"`},
    {`(json-parse "9007199254740993")`, "9007199254740993"}, // §3.6: war 9007199254740992 (gerundet)
  }
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("stdlib: %v", err)
  }
  for _, c := range cases {
    got, err := evalAll(c.src, env)
    if err != nil {
      t.Errorf("eval(%q) Fehler: %v", c.src, err)
      continue
    }
    if got.String() != c.want {
      t.Errorf("eval(%q) = %q, want %q", c.src, got.String(), c.want)
    }
  }
}
