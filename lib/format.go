//**********************************************************************
//  lib/format.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260625
//**********************************************************************
// FORMAT-Primitive (Common-Lisp-HyperSpec 22.3) für GoLisp.
//
// (format dest control . args)
//   dest = t      → Ausgabe auf stdout, Rückgabe nil
//   dest = nil    → Rückgabe als String
//   dest = string → String wird an dest angehängt, Rückgabe neuer String
//
// Unterstützte Direktiven (mit Parametern + Modifiern `:` `@`):
//   ~A ~S ~D ~B ~O ~X ~R ~P ~C ~F ~E ~G ~$ ~% ~& ~| ~T ~* ~? ~[ ~{ ~( ~; ~^
//   ~~  ~Newline
//
// Known-Limitations:
//   - ~/fun/ nicht implementiert (braucht env-Zugriff, Primitive hat keins).
//   - ~F/~E/~G: Mainstream-Parameter abgedeckt; exotische Edge-Cases
//     (overflowchar-Skaling, k≠0 Skaling) sind vereinfacht.
//   - ~R: Cardinal/Ordinal nur Englisch, 0..999999. Römisch 1..3999.
//**********************************************************************

package lib

import (
  "fmt"
  "strconv"
  "strings"
  "unicode/utf8"
)

// RegisterFormat hängt (format ...) ins Environment ein.
func RegisterFormat(env *Env) {
  _ = env.Set("format", makeFn(fnFormat))
  globalFormatEnv = env // für ~/fun/-Direktive (Lookup der User-Funktion)
}

// globalFormatEnv: das BaseEnv, von RegisterFormat gestzt. ~/fun/ braucht
// env-Zugriff zum Look-up der benannten Funktion — Primitive hat kein env.
// BaseEnv wird mehrfach aufgerufen (Tests); letzter Aufruf gewinnt.
var globalFormatEnv *Env

// fnFormat: (format dest control . args)
func fnFormat(args []*Cell) (*Cell, error) {
  if len(args) < 2 {
    return nil, fmt.Errorf("format: min. 2 Argumente (dest control . args)")
  }
  dest := args[0]
  if dest.Type != STRING && dest.Type != NIL &&
     !(dest.Type == ATOM && dest.Val == "t") {
    return nil, fmt.Errorf("format: dest muss t, nil oder String sein")
  }
  if args[1].Type != STRING {
    return nil, fmt.Errorf("format: control muss String sein")
  }
  ctrl := []rune(args[1].Val)
  st := &fmtState{out: &strings.Builder{}, args: args[2:]}
  formatRun(st, ctrl, 0)
  if st.err != nil {
    return nil, st.err
  }
  out := st.out.String()
  switch {
  case dest.Type == ATOM && dest.Val == "t":
    if err := WriteOutput(out); err != nil {
      return nil, fmt.Errorf("format: %w", err)
    }
    return MakeNil(), nil
  case dest.Type == NIL:
    return MakeStr(out), nil
  default: // STRING: an bestehenden String anhängen
    return MakeStr(dest.Val + out), nil
  }
}

// ---- Engine-State ----

type fmtState struct {
  out  *strings.Builder
  args []*Cell
  idx  int   // nächstes zu verbrauchendes Argument
  err  error
}

// pkind: Art eines Direktiven-Parameters.
type pkind int

const (
  pMissing pkind = iota
  pInt
  pChar   // 'x' Literal
  pV      // aus args verbrauchen
  pHash   // = restliche arg-Anzahl
)

type pval struct {
  kind pkind
  ival int
  rval rune
}

// formatRun parst control ab pos, schreibt in st.out.
// Return: (neue Position, escape-Flag für ~^).
func formatRun(st *fmtState, ctrl []rune, pos int) (int, bool) {
  for pos < len(ctrl) {
    r := ctrl[pos]
    if r != '~' {
      st.out.WriteRune(r)
      pos++
      continue
    }
    pos++ // skip ~
    params, npos := parseParams(st, ctrl, pos)
    pos = npos
    colon, at, mpos := parseModifiers(ctrl, pos)
    pos = mpos
    if pos >= len(ctrl) {
      st.err = fmt.Errorf("format: Direktive am String-Ende unvollständig")
      return pos, false
    }
    // ~/fun/ — benannte Funktion aufrufen (GoLisp hat keine Packages)
    if ctrl[pos] == '/' {
      pos = emitUserFunc(st, ctrl, pos, params, colon, at)
      if st.err != nil {
        return pos, false
      }
      continue
    }
    d := ctrl[pos]
    pos++

    // Newline-Sonderbehandlung (Tilde gefolgt von Newline)
    if d == '\n' {
      n := resolveInt(st, paramGet(params, 0), 1)
      if at {
        writeN(st.out, '\n', n)
      } else if colon {
        writeN(st.out, ' ', n)
      } else {
        // ignoriere Newline + folgende Whitespaces
        for pos < len(ctrl) && (ctrl[pos] == ' ' || ctrl[pos] == '\t') {
          pos++
        }
      }
      continue
    }

    switch d {
    case 'A', 'a':
      s := aesthetic(st.consume())
      emitPadded(st, s, params, colon)
    case 'S', 's':
      s := readable(st.consume())
      emitPadded(st, s, params, colon)
    case 'D', 'd':
      emitRadix(st, params, colon, at, 10)
    case 'B', 'b':
      emitRadix(st, params, colon, at, 2)
    case 'O', 'o':
      emitRadix(st, params, colon, at, 8)
    case 'X', 'x':
      emitRadix(st, params, colon, at, 16)
    case 'R', 'r':
      emitR(st, params, colon, at)
    case 'P', 'p':
      emitP(st, colon, at)
    case 'C', 'c':
      emitC(st, params, colon, at)
    case 'F', 'f':
      emitF(st, params, at)
    case 'E', 'e':
      emitE(st, params, at)
    case 'G', 'g':
      emitG(st, params, at)
    case '$':
      emitDollar(st, params, at)
    case '%':
      writeN(st.out, '\n', resolveInt(st, paramGet(params, 0), 1))
    case '&':
      freshLine(st.out, resolveInt(st, paramGet(params, 0), 1))
    case '|':
      writeN(st.out, '\f', resolveInt(st, paramGet(params, 0), 1))
    case 'T', 't':
      emitT(st, params, colon, at)
    case '*':
      emitGoto(st, params, colon, at)
    case '?', '!':
      pos = emitRecursive(st, ctrl, pos, at)
    case '[':
      pos = emitConditional(st, ctrl, pos, colon, at)
    case '{':
      ipos, esc := emitIteration(st, ctrl, pos, colon, at)
      pos = ipos
      if esc {
        return pos, true
      }
    case '(':
      pos = emitCase(st, ctrl, pos, colon, at)
    case '^':
      if emitEscape(st, params, colon, at) {
        return pos, true
      }
    case ';':
      // Separator außerhalb eines Blocks: ignorieren (sicher)
    case '~':
      writeN(st.out, '~', resolveInt(st, paramGet(params, 0), 1))
    default:
      st.err = fmt.Errorf("format: unbekannte Direktive ~%c", d)
      return pos, false
    }
    if st.err != nil {
      return pos, false
    }
  }
  return pos, false
}

// consume holt das nächste Argument.
func (st *fmtState) consume() *Cell {
  if st.idx >= len(st.args) {
    st.err = fmt.Errorf("format: keine Argumente mehr")
    return MakeNil()
  }
  c := st.args[st.idx]
  st.idx++
  return c
}

// ---- Parameter- & Modifier-Parsing ----

func parseParams(st *fmtState, ctrl []rune, pos int) ([]pval, int) {
  var params []pval
  // Nur parsen wenn ein Parameter folgt: Zahl, ', v, #, oder - (nur bei ',')
  for {
    if pos >= len(ctrl) {
      break
    }
    r := ctrl[pos]
    if r == ',' {
      // leeres Feld → missing
      params = append(params, pval{kind: pMissing})
      pos++
      continue
    }
    if r == 'v' || r == 'V' {
      params = append(params, pval{kind: pV})
      pos++
    } else if r == '#' {
      params = append(params, pval{kind: pHash})
      pos++
    } else if r == '\'' {
      // 'x' char-Literal
      pos++
      if pos < len(ctrl) {
        params = append(params, pval{kind: pChar, rval: ctrl[pos]})
        pos++
      }
    } else if r >= '0' && r <= '9' || r == '-' || r == '+' {
      num, npos := parseIntRunes(ctrl, pos)
      params = append(params, pval{kind: pInt, ival: num})
      pos = npos
    } else {
      break
    }
    // Nach Parameter: Komma erwartet (oder Ende der Parameter)
    if pos < len(ctrl) && ctrl[pos] == ',' {
      pos++
      continue
    }
    break
  }
  return params, pos
}

func parseIntRunes(ctrl []rune, pos int) (int, int) {
  start := pos
  if pos < len(ctrl) && (ctrl[pos] == '-' || ctrl[pos] == '+') {
    pos++
  }
  for pos < len(ctrl) && ctrl[pos] >= '0' && ctrl[pos] <= '9' {
    pos++
  }
  n, _ := strconv.Atoi(string(ctrl[start:pos]))
  return n, pos
}

func parseModifiers(ctrl []rune, pos int) (colon, at bool, npos int) {
  for pos < len(ctrl) && (ctrl[pos] == ':' || ctrl[pos] == '@') {
    if ctrl[pos] == ':' {
      colon = true
    } else {
      at = true
    }
    pos++
  }
  return colon, at, pos
}

func paramGet(params []pval, i int) pval {
  if i < len(params) {
    return params[i]
  }
  return pval{kind: pMissing}
}

// resolveInt löst einen int-Parameter auf (mit Default bei missing/v/#).
func resolveInt(st *fmtState, p pval, def int) int {
  switch p.kind {
  case pMissing:
    return def
  case pInt:
    return p.ival
  case pChar:
    return int(p.rval)
  case pV:
    c := st.consume()
    if c.Type == NUMBER {
      return int(c.Num)
    }
    return def
  case pHash:
    return len(st.args) - st.idx
  }
  return def
}

func resolveRune(st *fmtState, p pval, def rune) rune {
  switch p.kind {
  case pMissing:
    return def
  case pChar:
    return p.rval
  case pInt:
    return rune(p.ival)
  case pV:
    c := st.consume()
    if c.Type == STRING && utf8.RuneCountInString(c.Val) == 1 {
      r, _ := utf8.DecodeRuneInString(c.Val)
      return r
    }
    if c.Type == NUMBER {
      return rune(int(c.Num))
    }
    return def
  case pHash:
    return def
  }
  return def
}
