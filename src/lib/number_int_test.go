//**********************************************************************
//  lib/number_int_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-fable-5-2
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261116
//**********************************************************************
// Ganzzahl-Typ neben float64 (Spec 2026-11-16): Lesen/Drucken (§3.1),
// Rechnen (§3.2), Vergleichen (§3.3). want mit Präfix "ERR:" verlangt
// einen Fehler, dessen Text den Rest enthält.
//**********************************************************************

package lib

import (
  "strings"
  "testing"
)

const (
  maxI = "9223372036854775807"
  minI = "-9223372036854775808"
)

func runNumCases(t *testing.T, cases []struct{ src, want string }) {
  t.Helper()
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("stdlib: %v", err)
  }
  for _, c := range cases {
    got, err := evalAll(c.src, env)
    if wantErr, isErr := strings.CutPrefix(c.want, "ERR:"); isErr {
      if err == nil {
        t.Errorf("eval(%q) = %s, want Fehler mit %q", c.src, got, wantErr)
      } else if !strings.Contains(err.Error(), wantErr) {
        t.Errorf("eval(%q) Fehler %q, want Fehler mit %q", c.src, err, wantErr)
      }
      continue
    }
    if err != nil {
      t.Errorf("eval(%q) Fehler: %v", c.src, err)
      continue
    }
    if got.String() != c.want {
      t.Errorf("eval(%q) = %q, want %q", c.src, got.String(), c.want)
    }
  }
}

func TestIntLesenDrucken(t *testing.T) {
  runNumCases(t, []struct{ src, want string }{
    {`42`, "42"},
    {`-7`, "-7"},
    {`+3`, "3"},
    {maxI, maxI},
    {minI, minI},
    {`9007199254740993`, "9007199254740993"},
    {`9223372036854775808`, "ERR:außerhalb des int64-Bereichs"},
    {`-9223372036854775809`, "ERR:außerhalb des int64-Bereichs"},
    {`3.0`, "3.0"},
    {`1.`, "1.0"},
    {`1e3`, "1000.0"},
    {`3.14`, "3.14"},
    {`-0.5`, "-0.5"},
    {`1e17`, "1e+17"},
    {`1e19`, "1e+19"},
    {`1e-5`, "1e-05"},
    {`'(1 2.0 -3)`, "(1 2.0 -3)"},
    {`'1+`, "1+"},
    {`'-`, "-"},
  })
}

func TestIntZellen(t *testing.T) {
  if c := MakeInt(5); c.NumKind != NumInt || c.Int != 5 || c.Num != 5 {
    t.Errorf("MakeInt(5) = %+v", c)
  }
  if MakeInt(5) != MakeInt(5) {
    t.Error("MakeInt nutzt den Small-Int-Cache nicht")
  }
  if c := MakeNum(5); c.NumKind != NumFloat || c.Num != 5 {
    t.Errorf("MakeNum(5) muss Float bleiben, ist %+v", c)
  }
  if c := MakeInt(1 << 60); c.Int != 1<<60 || c.Num != float64(int64(1)<<60) {
    t.Errorf("MakeInt(1<<60) = %+v", c)
  }
}

func TestIntRechnen(t *testing.T) {
  runNumCases(t, []struct{ src, want string }{
    // + - *
    {`(+ 9007199254740992 1)`, "9007199254740993"},
    {`(+)`, "0"},
    {`(*)`, "1"},
    {`(+ 1 2 3)`, "6"},
    {`(+ 1 2.0)`, "3.0"},
    {`(+ 1.5 1.5)`, "3.0"},
    {`(- 5)`, "-5"},
    {`(- 5.0)`, "-5.0"},
    {`(- 10 3 2)`, "5"},
    {`(- 10 0.5)`, "9.5"},
    {`(* 6 7)`, "42"},
    {`(* 1.5 2)`, "3.0"},
    {`(+ ` + maxI + ` 1)`, "ERR:+: Ganzzahl-Überlauf"},
    {`(+ ` + minI + ` -1)`, "ERR:+: Ganzzahl-Überlauf"},
    {`(- ` + minI + `)`, "ERR:-: Ganzzahl-Überlauf"},
    {`(- ` + minI + ` 1)`, "ERR:-: Ganzzahl-Überlauf"},
    {`(- 0 ` + minI + `)`, "ERR:-: Ganzzahl-Überlauf"},
    {`(* ` + maxI + ` 2)`, "ERR:*: Ganzzahl-Überlauf"},
    {`(* ` + minI + ` -1)`, "ERR:*: Ganzzahl-Überlauf"},
    {`(* 99999999999 99999999999)`, "ERR:*: Ganzzahl-Überlauf"},
    {`(+ ` + maxI + ` 0)`, maxI},
    {`(- ` + maxI + `)`, "-9223372036854775807"},
    {`(* ` + minI + ` 1)`, minI},
    {`(+ ` + maxI + ` ` + minI + `)`, "-1"},
    {`(+ ` + maxI + ` 1.0)`, "9.223372036854776e+18"},
    // /
    {`(/ 6 2)`, "3"},
    {`(/ 7 2)`, "3.5"},
    {`(/ -6 3)`, "-2"},
    {`(/ 6.0 2)`, "3.0"},
    {`(/ 7.0 7)`, "1.0"},
    {`(/ 1 0)`, "ERR:/: Division durch 0"},
    {`(/ 1 0.0)`, "ERR:/: Division durch 0"},
    {`(/ ` + minI + ` -1)`, "ERR:/: Ganzzahl-Überlauf"},
    // mod rem remainder
    {`(mod 7 2)`, "1"},
    {`(mod -3 2)`, "1"},
    {`(mod 3 -2)`, "-1"},
    {`(mod 7.5 2)`, "1.5"},
    {`(mod 7.0 2)`, "1.0"},
    {`(rem -3 2)`, "-1"},
    {`(rem 7.5 2)`, "1.5"},
    {`(remainder -7 2)`, "-1"},
    {`(mod ` + minI + ` -1)`, "0"},
    {`(mod 1 0)`, "ERR:mod: Division durch Null"},
    // abs max min
    {`(abs -3)`, "3"},
    {`(abs -3.5)`, "3.5"},
    {`(abs -3.0)`, "3.0"},
    {`(abs ` + minI + `)`, "ERR:abs: Ganzzahl-Überlauf"},
    {`(max 1 2.5)`, "2.5"},
    {`(max 3 2.5)`, "3"},
    {`(min 1 2.5)`, "1"},
    {`(min 1.0 2)`, "1.0"},
    {`(max 1 2 3)`, "3"},
    // floor ceiling truncate round: Quotient immer Ganzzahl
    {`(floor 3.7)`, "3"},
    {`(floor -3.7)`, "-4"},
    {`(floor 7 2)`, "3"},
    {`(floor -7 2)`, "-4"},
    {`(floor 7 -2)`, "-4"},
    {`(ceiling 3.2)`, "4"},
    {`(ceiling 7 2)`, "4"},
    {`(ceiling -7 2)`, "-3"},
    {`(truncate -3.7)`, "-3"},
    {`(truncate -7 2)`, "-3"},
    {`(round 2.5)`, "2"},
    {`(round 3.5)`, "4"},
    {`(round 5 2)`, "2"},
    {`(round 7 2)`, "4"},
    {`(round -5 2)`, "-2"},
    {`(round -7 2)`, "-4"},
    {`(round 7 3)`, "2"},
    {`(round 8 3)`, "3"},
    {`(floor 3.0)`, "3"},
    {`(multiple-value-list (floor 7 2))`, "(3 1)"},
    {`(multiple-value-list (floor -7 2))`, "(-4 1)"},
    {`(multiple-value-list (ceiling 7 2))`, "(4 -1)"},
    {`(multiple-value-list (truncate -7 2))`, "(-3 -1)"},
    {`(multiple-value-list (round 7 2))`, "(4 -1)"},
    {`(multiple-value-list (round -7 2))`, "(-4 1)"},
    {`(multiple-value-list (floor 7.5 2))`, "(3 1.5)"},
    {`(multiple-value-list (floor 7 2.0))`, "(3 1.0)"},
    {`(floor ` + maxI + `)`, maxI},
    {`(floor 9007199254740993 2)`, "4503599627370496"},
    {`(floor 1e19)`, "ERR:floor: Ganzzahl-Überlauf"},
    {`(round -1e19)`, "ERR:round: Ganzzahl-Überlauf"},
    {`(truncate (expt 10.0 400))`, "ERR:"},
    {`(floor ` + minI + ` -1)`, "ERR:floor: Ganzzahl-Überlauf"},
    {`(floor 1 0)`, "ERR:floor: Division durch Null"},
    {`(ceiling 1 0.0)`, "ERR:ceiling: Division durch Null"},
    // sqrt exp log: immer Float
    {`(sqrt 16)`, "4.0"},
    {`(exp 0)`, "1.0"},
    {`(log 1)`, "0.0"},
    // expt
    {`(expt 2 10)`, "1024"},
    {`(expt 2 62)`, "4611686018427387904"},
    {`(expt -2 63)`, minI},
    {`(expt 2 63)`, "ERR:expt: Ganzzahl-Überlauf"},
    {`(expt 10 19)`, "ERR:expt: Ganzzahl-Überlauf"},
    {`(expt 7 0)`, "1"},
    {`(expt 0 0)`, "1"},
    {`(expt 1 1000000)`, "1"},
    {`(expt -1 1000001)`, "-1"},
    {`(expt 2 -1)`, "0.5"},
    {`(expt 2.0 3)`, "8.0"},
    {`(expt 4 0.5)`, "2.0"},
    // random
    {`(let ((r (random 1.0))) (and (>= r 0) (< r 1)))`, "t"},
    {`(let ((r (random 3))) (and (>= r 0) (< r 3)))`, "t"},
    {`(random 0)`, "ERR:random: positive Zahl erwartet"},
    {`(random "x")`, "ERR:random: Zahl erwartet"},
    // 1+ 1-
    {`(1+ 5)`, "6"},
    {`(1- 5)`, "4"},
    {`(1+ 1.5)`, "2.5"},
    {`(1+ ` + maxI + `)`, "ERR:+: Ganzzahl-Überlauf"},
    {`(1- ` + minI + `)`, "ERR:-: Ganzzahl-Überlauf"},
    // ash
    {`(ash 1 10)`, "1024"},
    {`(ash 1 62)`, "4611686018427387904"},
    {`(ash 1024 -3)`, "128"},
    {`(ash -5 -1)`, "-3"},
    {`(ash 4611686018427387905 -1)`, "2305843009213693952"},
    {`(ash 1 63)`, "ERR:Ganzzahl-Überlauf"},
    // evenp/oddp jenseits 2^53
    {`(oddp 9007199254740993)`, "t"},
    {`(evenp 9007199254740993)`, "()"},
  })
}

func TestIntVergleichen(t *testing.T) {
  runNumCases(t, []struct{ src, want string }{
    {`(= 1 1.0)`, "t"},
    {`(= 1 1)`, "t"},
    {`(= 1 2)`, "()"},
    {`(= 9007199254740993 9007199254740992)`, "()"},
    {`(/= 9007199254740993 9007199254740992)`, "t"},
    {`(/= 1 1.0)`, "()"},
    {`(< 9007199254740992 9007199254740993)`, "t"},
    {`(> 9007199254740993 9007199254740992)`, "t"},
    {`(<= 9007199254740993 9007199254740992)`, "()"},
    {`(>= 9007199254740992 9007199254740993)`, "()"},
    {`(< 1 1.5 2)`, "t"},
    {`(> 2 1.5 1)`, "t"},
    {`(<= 1 1.0 2)`, "t"},
    {`(>= 2 2.0 1)`, "t"},
    {`(< ` + minI + ` ` + maxI + `)`, "t"},
    {`(eql 1 1)`, "t"},
    {`(eql 1.0 1.0)`, "t"},
    {`(eql 1 1.0)`, "()"},
    {`(eql 9007199254740993 9007199254740992)`, "()"},
    {`(equal? 1 1.0)`, "()"},
    {`(equal? 1 1)`, "t"},
    {`(equal? (list 1 2) (list 1 2))`, "t"},
    {`(equal? (list 1 2) (list 1.0 2.0))`, "()"},
    {`(eq 5 5)`, "()"},
    {`(eq 5 5.0)`, "()"},
    // Hash-Schlüssel: 1 und 1.0 sind verschieden
    {`(let ((h (make-hash-table))) (setf (gethash 1 h) 'a) (gethash 1.0 h))`, "()"},
    {`(let ((h (make-hash-table))) (setf (gethash 1 h) 'a) (setf (gethash 1.0 h) 'b) (list (gethash 1 h) (gethash 1.0 h)))`, "(a b)"},
    {`(let ((h (make-hash-table :test 'equal))) (setf (gethash (list 1 2) h) 'a) (list (gethash (list 1 2) h) (gethash (list 1.0 2) h)))`, "(a ())"},
    // case, member, position folgen eql
    {`(member 1.0 (list 1 2))`, "()"},
    {`(member 1 (list 1 2))`, "(1 2)"},
    {`(position 2.0 (list 1 2))`, "()"},
    {`(position 2 (list 1 2))`, "1"},
    {`(case 1 (1.0 'float) (1 'int))`, "int"},
    {`(case 1.0 (1 'int) (1.0 'float))`, "float"},
  })
}

// Zahlart ohne Umweg über den Printer.
func TestIntZahlart(t *testing.T) {
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("stdlib: %v", err)
  }
  cases := []struct {
    src  string
    want NumKind
  }{
    {`(random 10)`, NumInt},
    {`(random)`, NumInt},
    {`(random 1.0)`, NumFloat},
    {`(random 10.0)`, NumFloat},
    {`(+ 1 2)`, NumInt},
    {`(+ 1 2.0)`, NumFloat},
    {`(/ 6 2)`, NumInt},
    {`(/ 7 2)`, NumFloat},
    {`(floor 3.0)`, NumInt},
    {`(sqrt 16)`, NumFloat},
    {`(max 1 2.0)`, NumFloat},
    {`(min 1 2.0)`, NumInt},
  }
  for _, c := range cases {
    got, err := evalAll(c.src, env)
    if err != nil {
      t.Errorf("eval(%q) Fehler: %v", c.src, err)
      continue
    }
    if got = Primary(got); got.Type != NUMBER || got.NumKind != c.want {
      t.Errorf("eval(%q) = %s mit Zahlart %d, want %d", c.src, got, got.NumKind, c.want)
    }
  }
}
