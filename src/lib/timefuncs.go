//**********************************************************************
//  lib/timefuncs.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261003
//**********************************************************************
// Wanduhr-Zeit (TODO-20261003-luecken Punkt 1):
//
//   (now)                         → Unix-Sekunden als Float (Epoche 1970)
//   (format-time fmt [zeit] [:utc]) → String im Stil von strftime/date(1)
//
// zeit sind Unix-Sekunden wie von (now), Default jetzt; Ortszeit, außer
// mit :utc. get-universal-time (clcompat_prims.go) bleibt CL-Epoche 1900.
// Direktiven: %Y %y %m %d %H %M %S %j %F %T %a %A %b %B %z %Z %s %N %3N %%
// — jede andere ist ein Fehler, kein stilles Durchreichen. Tag-/Monats-
// namen sind englisch (Go kennt keine Locale).
//**********************************************************************

package lib

import (
  "fmt"
  "math"
  "strings"
  "time"
)

// RegisterTime hängt now und format-time ins Environment ein.
func RegisterTime(env *Env) {
  _ = env.Set("now",         makeFn(fnNow))
  _ = env.Set("format-time", makeFn(fnFormatTime))
}

// now: (now) → Unix-Sekunden mit Nachkommastellen
func fnNow(args []*Cell) (*Cell, error) {
  if len(args) != 0 {
    return nil, fmt.Errorf("now: keine Argumente erwartet")
  }
  return MakeNum(float64(time.Now().UnixNano()) / 1e9), nil
}

// format-time: (format-time fmt [zeit] [:utc]) → String
func fnFormatTime(args []*Cell) (*Cell, error) {
  if len(args) < 1 || len(args) > 3 {
    return nil, fmt.Errorf("format-time: 1 bis 3 Argumente erwartet (fmt [zeit] [:utc])")
  }
  if args[0].Type != STRING {
    return nil, fmt.Errorf("format-time: Format muss String sein, got %s", args[0])
  }
  t := time.Now()
  if len(args) >= 2 {
    if args[1].Type != NUMBER {
      return nil, fmt.Errorf("format-time: Zeit muss Zahl sein (Unix-Sekunden), got %s", args[1])
    }
    t = unixFloatToTime(args[1].Num)
  }
  if len(args) == 3 {
    if args[2].Type != ATOM || args[2].Val != ":utc" {
      return nil, fmt.Errorf("format-time: nur :utc erlaubt, got %s", args[2])
    }
    t = t.UTC()
  }
  s, err := strftime(args[0].Val, t)
  if err != nil {
    return nil, err
  }
  return MakeStr(s), nil
}

func unixFloatToTime(secs float64) time.Time {
  whole := math.Floor(secs)
  nanos := math.Round((secs - whole) * 1e9)
  return time.Unix(int64(whole), int64(nanos))
}

// strftimeLayouts: Direktiven, die direkt auf ein Go-Layout abbilden.
var strftimeLayouts = map[byte]string{
  'Y': "2006", 'y': "06", 'm': "01", 'd': "02",
  'H': "15", 'M': "04", 'S': "05", 'j': "002",
  'F': "2006-01-02", 'T': "15:04:05",
  'a': "Mon", 'A': "Monday", 'b': "Jan", 'B': "January",
  'z': "-0700", 'Z': "MST",
}

func strftime(format string, t time.Time) (string, error) {
  var sb strings.Builder
  for i := 0; i < len(format); i++ {
    if format[i] != '%' {
      sb.WriteByte(format[i])
      continue
    }
    i++
    if i >= len(format) {
      return "", fmt.Errorf("format-time: '%%' am Formatende")
    }
    c := format[i]
    if layout, ok := strftimeLayouts[c]; ok {
      sb.WriteString(t.Format(layout))
      continue
    }
    switch {
    case c == '%':
      sb.WriteByte('%')
    case c == 's':
      fmt.Fprintf(&sb, "%d", t.Unix())
    case c == 'N':
      fmt.Fprintf(&sb, "%09d", t.Nanosecond())
    case c == '3' && i+1 < len(format) && format[i+1] == 'N':
      fmt.Fprintf(&sb, "%03d", t.Nanosecond()/1e6)
      i++
    default:
      return "", fmt.Errorf("format-time: unbekannte Direktive '%%%c'", c)
    }
  }
  return sb.String(), nil
}
