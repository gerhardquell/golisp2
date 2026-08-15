//**********************************************************************
//  lib/cformat.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260815
//**********************************************************************
// C-artige formatierte Ein-/Ausgabe (TODO 20260813 Punkt 2.3).
//
//   (printf fmt . args)         → auf stdout (WriteOutput, SWANK-sichtbar)
//   (sprintf fmt . args)        → Ergebnis-String
//   (fprintf ziel fmt . args)   → ziel: "sys-stdout"/"sys-stderr"/Datei (append)
//   (sscanf str fmt)            → Liste der gelesenen Werte
//
// Die printf-Familie übersetzt den C-Formatstring nach CL-format und ruft
// die FORMAT-Engine (lib/format.go) — eine Engine bleibt die einzige Quelle.
// Unicode: alles rune-basiert (utf8.RuneCountInString), %c = 1 Rune.
//
// Unterstütztes Kern-Set:
//   %d %i %s %f %e %g %x %X %o %c %%
//   Flags: - (linksbündig), 0 (Null-Padding), + (Vorzeichen)
//   width, .precision
//
// Bewusste Abweichungen/Lücken:
//   - %f/%e ohne precision: C-Default 6 Nachkommastellen (nicht ~F-Default)
//   - %x/%X: Ausgabe immer lowercase (Engine nutzt strconv.FormatInt)
//   - %.Ns (String-Truncation): Fehler — CL-format kann das nicht ausdrücken
//   - linksbündige Floats (%-8.2f): Fehler — ~F kennt kein linksbündig
//   - Hex/Exponential-Details folgen der Engine, nicht glibc
//**********************************************************************

package lib

import (
  "fmt"
  "os"
  "strconv"
  "strings"
  "unicode"
)

// RegisterCFormat hängt die printf/sscanf-Familie ins Environment ein.
func RegisterCFormat(env *Env) {
  _ = env.Set("printf",  makeFn(fnPrintf))
  _ = env.Set("sprintf", makeFn(fnSprintf))
  _ = env.Set("fprintf", makeFn(fnFprintf))
  _ = env.Set("sscanf",  makeFn(fnSscanf))
}

// ---- Übersetzer C-Format → CL-Format ----

// cformatToCL übersetzt einen C-printf-Formatstring in einen CL-format-
// Control-String. Rune-basiert.
func cformatToCL(cfmt string) (string, error) {
  src := []rune(cfmt)
  var out strings.Builder
  for i := 0; i < len(src); i++ {
    r := src[i]
    if r == '~' {
      out.WriteString("~~")
      continue
    }
    if r != '%' {
      out.WriteRune(r)
      continue
    }
    i++
    if i >= len(src) {
      return "", fmt.Errorf("cformat: %% am String-Ende unvollständig")
    }
    if src[i] == '%' {
      out.WriteRune('%')
      continue
    }

    // Flags
    left, zero, plus := false, false, false
    for i < len(src) {
      switch src[i] {
      case '-': left = true
      case '0': zero = true
      case '+': plus = true
      default: goto flagsDone
      }
      i++
    }
  flagsDone:

    // width
    width := 0
    for i < len(src) && src[i] >= '0' && src[i] <= '9' {
      width = width*10 + int(src[i]-'0')
      i++
    }

    // precision
    prec := -1
    if i < len(src) && src[i] == '.' {
      i++
      prec = 0
      for i < len(src) && src[i] >= '0' && src[i] <= '9' {
        prec = prec*10 + int(src[i]-'0')
        i++
      }
    }

    if i >= len(src) {
      return "", fmt.Errorf("cformat: Direktive ohne Verb am String-Ende")
    }
    verb := src[i]

    switch verb {
    case 'd', 'i':
      out.WriteString(clInt(width, zero, left, plus, 'D'))
    case 'x', 'X':
      out.WriteString(clInt(width, zero, left, plus, 'X'))
    case 'o':
      out.WriteString(clInt(width, zero, left, plus, 'O'))
    case 's':
      if prec >= 0 {
        return "", fmt.Errorf("cformat: %%.%ds (String-Truncation) nicht unterstützt", prec)
      }
      out.WriteString(clStr(width, left))
    case 'f', 'e', 'g':
      s, err := clFloat(width, prec, zero, left, plus, rune(verb-32)) // → F/E/G
      if err != nil { return "", err }
      out.WriteString(s)
    case 'c':
      if width > 0 || prec >= 0 {
        return "", fmt.Errorf("cformat: %%c mit width/precision nicht unterstützt")
      }
      out.WriteString("~C")
    default:
      return "", fmt.Errorf("cformat: unbekannte Direktive %%%c", verb)
    }
  }
  return out.String(), nil
}

// clInt: Integer-Direktive. Rechtsbündig über ~D/~X/~O (pad links),
// linksbündig über ~A (pad rechts) — ~D kennt kein linksbündig.
func clInt(width int, zero, left, plus bool, verb rune) string {
  if left {
    if width > 0 { return fmt.Sprintf("~%dA", width) }
    return "~A"
  }
  var b strings.Builder
  b.WriteRune('~')
  if width > 0 { b.WriteString(strconv.Itoa(width)) }
  if zero && width > 0 { b.WriteString(",'0") }
  if plus { b.WriteRune('@') }
  b.WriteRune(verb)
  return b.String()
}

// clStr: String-Direktive. C-Default rechtsbündig (~:A), '-' linksbündig (~A).
func clStr(width int, left bool) string {
  if width == 0 { return "~A" }
  if left { return fmt.Sprintf("~%dA", width) }
  return fmt.Sprintf("~%d:A", width)
}

// clFloat: Float-Direktive ~w,d,...,padchar F/E/G.
// prec < 0 → C-Default 6 für F/E, Engine-Default für G.
func clFloat(width, prec int, zero, left, plus bool, verb rune) (string, error) {
  if left {
    // ~F/~E/~G kennen kein linksbündig — laut statt still falsch
    return "", fmt.Errorf("cformat: linksbündige Floats (%%-%c) nicht unterstützt", verb)
  }
  p := prec
  if p < 0 && verb != 'G' { p = 6 }
  var b strings.Builder
  b.WriteRune('~')
  if width > 0 { b.WriteString(strconv.Itoa(width)) }
  if width > 0 || p >= 0 {
    b.WriteRune(',')
    if p >= 0 { b.WriteString(strconv.Itoa(p)) }
  }
  if zero && width > 0 {
    // padchar steht an Parameter-Index 4 (w,d,k,overflow,padchar)
    b.WriteString(",,,'0")
  }
  if plus { b.WriteRune('@') }
  b.WriteRune(verb)
  return b.String(), nil
}

// ---- printf-Familie ----

// cformatString übersetzt fmt und rendert mit der FORMAT-Engine.
func cformatString(cfmt string, args []*Cell) (string, error) {
  cl, err := cformatToCL(cfmt)
  if err != nil {
    return "", err
  }
  res, err := fnFormat(append([]*Cell{MakeNil(), MakeStr(cl)}, args...))
  if err != nil {
    return "", err
  }
  return res.Val, nil
}

// printf: (printf fmt . args) → auf stdout
func fnPrintf(args []*Cell) (*Cell, error) {
  if len(args) < 1 { return nil, fmt.Errorf("printf: mindestens 1 Argument (fmt)") }
  if args[0].Type != STRING { return nil, fmt.Errorf("printf: fmt muss String sein") }
  s, err := cformatString(args[0].Val, args[1:])
  if err != nil { return nil, fmt.Errorf("printf: %v", err) }
  if err := WriteOutput(s); err != nil { return nil, fmt.Errorf("printf: %v", err) }
  return MakeNil(), nil
}

// sprintf: (sprintf fmt . args) → Ergebnis-String
func fnSprintf(args []*Cell) (*Cell, error) {
  if len(args) < 1 { return nil, fmt.Errorf("sprintf: mindestens 1 Argument (fmt)") }
  if args[0].Type != STRING { return nil, fmt.Errorf("sprintf: fmt muss String sein") }
  s, err := cformatString(args[0].Val, args[1:])
  if err != nil { return nil, fmt.Errorf("sprintf: %v", err) }
  return MakeStr(s), nil
}

// fprintf: (fprintf ziel fmt . args) — ziel: "sys-stdout"/"sys-stderr"
// oder Dateiname (append-Semantik, C FILE* am Dateiende).
func fnFprintf(args []*Cell) (*Cell, error) {
  if len(args) < 2 { return nil, fmt.Errorf("fprintf: mindestens 2 Argumente (ziel fmt)") }
  if args[0].Type != STRING { return nil, fmt.Errorf("fprintf: ziel muss String sein") }
  if args[1].Type != STRING { return nil, fmt.Errorf("fprintf: fmt muss String sein") }
  dest := args[0].Val
  s, err := cformatString(args[1].Val, args[2:])
  if err != nil { return nil, fmt.Errorf("fprintf: %v", err) }

  if dest == sysStdout || dest == sysStderr {
    return writeToSysStream(dest, s)
  }
  filename := resolveWritePath(dest)
  f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
  if err != nil { return nil, fmt.Errorf("fprintf '%s': %v", filename, err) }
  defer f.Close()
  if _, err := f.WriteString(s); err != nil {
    return nil, fmt.Errorf("fprintf '%s': %v", filename, err)
  }
  return MakeStr(filename), nil
}

// ---- sscanf ----

// sscanf: (sscanf str fmt) → Liste der gelesenen Werte.
// Direktiven: %d (Zahl), %f (Zahl), %s (Token bis Whitespace), %c (1 Rune),
// %% (literal %). Whitespace im Format matcht beliebig viel Whitespace
// (auch keinen). Literale müssen exakt matchen.
func fnSscanf(args []*Cell) (*Cell, error) {
  if len(args) != 2 { return nil, fmt.Errorf("sscanf: 2 Argumente (str fmt)") }
  if args[0].Type != STRING || args[1].Type != STRING {
    return nil, fmt.Errorf("sscanf: str und fmt müssen Strings sein")
  }
  in := []rune(args[0].Val)
  f := []rune(args[1].Val)
  var vals []*Cell
  ip := 0 // Eingabe-Position

  skipWS := func() {
    for ip < len(in) && unicode.IsSpace(in[ip]) { ip++ }
  }

  for fp := 0; fp < len(f); fp++ {
    r := f[fp]
    if unicode.IsSpace(r) {
      skipWS()
      continue
    }
    if r != '%' {
      if ip >= len(in) || in[ip] != r {
        return nil, fmt.Errorf("sscanf: Literal '%c' erwartet an Position %d", r, ip)
      }
      ip++
      continue
    }
    fp++
    if fp >= len(f) {
      return nil, fmt.Errorf("sscanf: %% am Format-Ende unvollständig")
    }
    verb := f[fp]
    switch verb {
    case '%':
      if ip >= len(in) || in[ip] != '%' {
        return nil, fmt.Errorf("sscanf: Literal '%%' erwartet an Position %d", ip)
      }
      ip++
    case 'c':
      if ip >= len(in) {
        return nil, fmt.Errorf("sscanf: %%c — Eingabe zu Ende")
      }
      vals = append(vals, MakeStr(string(in[ip])))
      ip++
    case 's':
      skipWS()
      start := ip
      for ip < len(in) && !unicode.IsSpace(in[ip]) { ip++ }
      if ip == start {
        return nil, fmt.Errorf("sscanf: %%s — kein Token an Position %d", start)
      }
      vals = append(vals, MakeStr(string(in[start:ip])))
    case 'd', 'i':
      skipWS()
      tok, n := scanNumber(in[ip:], false)
      if n == 0 {
        return nil, fmt.Errorf("sscanf: %%d — keine Zahl an Position %d", ip)
      }
      ip += n
      v, _ := strconv.ParseFloat(tok, 64)
      vals = append(vals, MakeNum(v))
    case 'f':
      skipWS()
      tok, n := scanNumber(in[ip:], true)
      if n == 0 {
        return nil, fmt.Errorf("sscanf: %%f — keine Zahl an Position %d", ip)
      }
      ip += n
      v, _ := strconv.ParseFloat(tok, 64)
      vals = append(vals, MakeNum(v))
    default:
      return nil, fmt.Errorf("sscanf: unbekannte Direktive %%%c", verb)
    }
  }
  return SliceToCell(vals), nil
}

// scanNumber liest ein numerisches Token ([+-]digits[.digits][eE[+-]digits]).
// allowFloat steuert, ob '.'/Exponent erlaubt sind. Liefert Token + Länge
// in Runen (0 = keine Zahl).
func scanNumber(in []rune, allowFloat bool) (string, int) {
  n := 0
  if n < len(in) && (in[n] == '+' || in[n] == '-') { n++ }
  digits := 0
  for n < len(in) && in[n] >= '0' && in[n] <= '9' { n++; digits++ }
  if allowFloat && n < len(in) && in[n] == '.' {
    n++
    for n < len(in) && in[n] >= '0' && in[n] <= '9' { n++; digits++ }
  }
  if digits == 0 { return "", 0 }
  if allowFloat && n < len(in) && (in[n] == 'e' || in[n] == 'E') {
    m := n + 1
    if m < len(in) && (in[m] == '+' || in[m] == '-') { m++ }
    expDigits := 0
    for m < len(in) && in[m] >= '0' && in[m] <= '9' { m++; expDigits++ }
    if expDigits > 0 { n = m }
  }
  return string(in[:n]), n
}
