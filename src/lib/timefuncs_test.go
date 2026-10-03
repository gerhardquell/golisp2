//**********************************************************************
//  lib/timefuncs_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261003
//**********************************************************************
// Tests für (now) und (format-time …) — TODO-20261003-luecken Punkt 1.
//**********************************************************************

package lib

import (
  "strconv"
  "testing"
  "time"
)

func TestNowUnixSeconds(t *testing.T) {
  got, err := evalStr(`(now)`)
  if err != nil {
    t.Fatalf("(now): %v", err)
  }
  secs, err := strconv.ParseFloat(got.String(), 64)
  if err != nil {
    t.Fatalf("(now) keine Zahl: %q", got.String())
  }
  if diff := secs - float64(time.Now().UnixNano())/1e9; diff > 1 || diff < -1 {
    t.Errorf("(now) = %v, weicht %v s von der Uhr ab", secs, diff)
  }
  evalErr(t, `(now 1)`)
}

// get-universal-time zählt ab 1900, now ab 1970: Abstand fest 2208988800 s.
func TestNowVsUniversalTime(t *testing.T) {
  evalEq(t, `(let ((d (- (get-universal-time) (floor (now))))) (or (= d 2208988800) (= d 2208988801)))`, "t")
}

func TestFormatTimeUTC(t *testing.T) {
  cases := []struct{ src, want string }{
    {`(format-time "%F %T" 0 :utc)`, `"1970-01-01 00:00:00"`},
    {`(format-time "%Y-%m-%dT%H:%M:%S%z" 1791046147 :utc)`, `"2026-10-03T16:49:07+0000"`},
    {`(format-time "%y %j %Z" 1791046147 :utc)`, `"26 276 UTC"`},
    {`(format-time "%a %A %b %B" 1791046147 :utc)`, `"Sat Saturday Oct October"`},
    {`(format-time "%s" 1791046147.25 :utc)`, `"1791046147"`},
    {`(format-time "%3N|%N" 1791046147.25 :utc)`, `"250|250000000"`},
    {`(format-time "100%% %d" 0 :utc)`, `"100% 01"`},
    {`(format-time "" 0 :utc)`, `""`},
  }
  for _, c := range cases {
    evalEq(t, c.src, c.want)
  }
}

func TestFormatTimeLocalDefault(t *testing.T) {
  want := strconv.Quote(time.Unix(1791046147, 0).Format("2006-01-02 15:04:05"))
  evalEq(t, `(format-time "%F %T" 1791046147)`, want)
  // ohne Zeit-Argument: jetzt (Datum reicht, Mitternachts-Wechsel ignoriert)
  evalEq(t, `(format-time "%F")`, strconv.Quote(time.Now().Format("2006-01-02")))
}

func TestFormatTimeErrors(t *testing.T) {
  for _, src := range []string{
    `(format-time)`,
    `(format-time 5)`,
    `(format-time "%Q" 0)`,      // unbekannte Direktive
    `(format-time "abc%" 0)`,    // % am Ende
    `(format-time "%2N" 0)`,     // nur %3N/%N
    `(format-time "%F" "x")`,    // Zeit keine Zahl
    `(format-time "%F" 0 :gmt)`, // unbekanntes Keyword
    `(format-time "%F" 0 :utc 1)`,
  } {
    evalErr(t, src)
  }
}
