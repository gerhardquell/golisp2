//**********************************************************************
//  lib/number_raender_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-fable-5-2
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261116
//**********************************************************************
// Ganzzahl-Typ, Spec 2026-11-16 §3.6: Ränder — FORMAT, printf,
// catch/throw-Tags, loop repeat, Postgres-Parameter, sigo-Usage.
//**********************************************************************

package lib

import "testing"

func TestIntFormatExakt(t *testing.T) {
  runNumCases(t, []struct{ src, want string }{
    {`(format nil "~d" 9007199254740993)`, `"9007199254740993"`},
    {`(format nil "~:d" 9007199254740993)`, `"9,007,199,254,740,993"`},
    {`(format nil "~x" 9007199254740993)`, `"20000000000001"`},
    {`(format nil "~o" 9007199254740993)`, `"400000000000000001"`},
    {`(format nil "~b" 9007199254740993)`, `"100000000000000000000000000000000000000000000000000001"`},
    {`(format nil "~d" -9223372036854775808)`, `"-9223372036854775808"`},
    {`(format nil "~36r" 9007199254740993)`, `"2gosa7pa2gx"`},
    {`(format nil "~2r" -9223372036854775808)`, `"-1000000000000000000000000000000000000000000000000000000000000000"`},
    // Float geht den ~a-Weg (§3.6)
    {`(format nil "~d" 3.0)`, `"3.0"`},
    {`(format nil "~x" 255.0)`, `"255.0"`},
    {`(format nil "~d" 3.5)`, `"3.5"`},
    {`(format nil "~5d|" 3.0)`, `"  3.0|"`},
    // ~p, ~c und v-Parameter
    {`(format nil "~d cat~:p" 1)`, `"1 cat"`},
    {`(format nil "~d cat~:p" 9007199254740993)`, `"9007199254740993 cats"`},
    {`(format nil "~c" 65)`, `"A"`},
    {`(format nil "~vd|" 5 3)`, `"    3|"`},
    {`(format nil "~v,vd|" 5 48 3)`, `"00003|"`},
    // printf/sprintf laufen über FORMAT
    {`(sprintf "%d %x" 9007199254740993 9007199254740993)`, `"9007199254740993 20000000000001"`},
    {`(sprintf "%d" 3.0)`, `"3.0"`},
  })
}

func TestIntCatchTagEql(t *testing.T) {
  runNumCases(t, []struct{ src, want string }{
    {`(catch 1 (throw 1 'ok))`, "ok"},
    {`(catch 1.5 (throw 1.5 'ok))`, "ok"},
    {`(catch 'a (throw 'a 'ok))`, "ok"},
    {`(catch 9007199254740993 (throw 9007199254740992 'falsch))`, "ERR:throw: kein passendes catch"},
    {`(catch 1 (throw 1.0 'falsch))`, "ERR:throw: kein passendes catch"},
  })
}

func TestIntLoopRepeat(t *testing.T) {
  runNumCases(t, []struct{ src, want string }{
    {`(loop repeat 3 collect 1)`, "(1 1 1)"},
    {`(loop repeat 0 collect 1)`, "()"},
    {`(loop repeat 2.5 collect 1)`, "ERR:loop: ganze Zahl erwartet"},
    {`(loop repeat 2.0 collect 1)`, "ERR:loop: ganze Zahl erwartet"},
  })
}

// Ohne Datenbank: nur die Umwandlungsfunktion.
func TestIntPostgresParameter(t *testing.T) {
  if v, ok := cellToInterface(MakeInt(9007199254740993)).(int64); !ok || v != 9007199254740993 {
    t.Errorf("Ganzzahl-Parameter = %#v, want int64(9007199254740993)", cellToInterface(MakeInt(9007199254740993)))
  }
  if v, ok := cellToInterface(MakeNum(3)).(float64); !ok || v != 3 {
    t.Errorf("Float-Parameter = %#v, want float64(3)", cellToInterface(MakeNum(3)))
  }
  if c := interfaceToCell(int64(9007199254740993)); c.NumKind != NumInt || c.Int != 9007199254740993 {
    t.Errorf("int8-Spalte = %s", c)
  }
}

// Token-Zahlen sind Ganzzahlen, Kosten Floats — auch wenn JSON "0" liefert.
func TestIntSigoUsage(t *testing.T) {
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("stdlib: %v", err)
  }
  resp, err := evalAll(`(json-parse "{\"usage\":{\"prompt_tokens\":9007199254740993,\"completion_tokens\":7.0,\"cost_usd\":0,\"prompt_tokens_details\":{\"cached_tokens\":5}}}")`, env)
  if err != nil {
    t.Fatal(err)
  }
  u := sigoUsageFromResponse(resp)
  if u.PromptTokens != 9007199254740993 || u.CompletionTokens != 7 || u.CachedTokens != 5 {
    t.Errorf("usage = %+v", u)
  }
  h := sigoUsageHash(u)
  for key, kind := range map[string]NumKind{"prompt-tokens": NumInt, "completion-tokens": NumInt,
    "cached-tokens": NumInt, "reasoning-tokens": NumInt, "cost-usd": NumFloat, "elapsed": NumFloat} {
    if v := hashAt(h, key); v == nil || v.Type != NUMBER || v.NumKind != kind {
      t.Errorf("%s = %v, falsche Zahlart", key, v)
    }
  }
}
