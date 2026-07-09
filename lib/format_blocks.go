//**********************************************************************
//  lib/format_blocks.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260625
//**********************************************************************
// FORMAT-Block-Direktiven + Struktur-Helper (aufgeteilt aus format_dirs.go).
// ~/fun/ ~? ~[ ~{ ~( ~^  +  findBlock/splitClauses/ctrlSlice/Sub-Helper.
// Siehe format.go für Engine und Semantik.
//**********************************************************************

package lib

import (
  "fmt"
  "strings"
  "unicode"
)

// ---- ~/fun/ User-Funktion ----

// emitUserFunc: ~/name/ ruft die benannte Funktion (globalFormatEnv) mit
// dem nächsten format-arg auf, schreibt das Ergebnis aesthetic. GoLisp hat
// keine Packages → ~/<symbol>/ (kein package:prefix). CL übergibt ferner
// stream/colon/atsign/params an f; hier vereinfacht auf (f arg).
func emitUserFunc(st *fmtState, ctrl []rune, pos int, params []pval, colon, at bool) int {
  _ = params
  _ = colon
  _ = at
  pos++ // skip öffnendes /
  nameStart := pos
  for pos < len(ctrl) && ctrl[pos] != '/' {
    pos++
  }
  name := string(ctrl[nameStart:pos])
  if pos < len(ctrl) {
    pos++ // skip schließendes /
  }
  if name == "" {
    st.err = fmt.Errorf("format: ~/fun/ braucht Funktionsnamen")
    return pos
  }
  if globalFormatEnv == nil {
    st.err = fmt.Errorf("format: ~/fun/ kein Environment registriert")
    return pos
  }
  fn, err := globalFormatEnv.Get(name)
  if err != nil {
    st.err = fmt.Errorf("format: ~/%s/: %v", name, err)
    return pos
  }
  arg := st.consume()
  if st.err != nil {
    return pos
  }
  result, err := apply(fn, []*Cell{arg})
  if err != nil {
    st.err = fmt.Errorf("format: ~/%s/: %v", name, err)
    return pos
  }
  st.out.WriteString(aesthetic(result))
  return pos
}

// ---- ~? Recursive ----

func emitRecursive(st *fmtState, ctrl []rune, pos int, at bool) int {
  cstr := st.consume()
  if cstr.Type != STRING {
    st.err = fmt.Errorf("format: ~? control muss String sein")
    return pos
  }
  var subArgs []*Cell
  if at {
    subArgs = st.args[st.idx:]
    st.idx = len(st.args)
  } else {
    lst := st.consume()
    subArgs = cellToSlice(lst)
  }
  sub := &fmtState{out: &strings.Builder{}, args: subArgs}
  formatRun(sub, []rune(cstr.Val), 0)
  if sub.err != nil {
    st.err = sub.err
    return pos
  }
  st.out.WriteString(sub.out.String())
  return pos
}

// ---- ~[ ~] Conditional ----

func emitConditional(st *fmtState, ctrl []rune, pos int, colon, at bool) int {
  body, npos := findBlock(ctrl, pos, '[', ']')
  clauses, defaultIdx := splitClauses(ctrl, body)
  if at {
    // ~@[...]: wenn nächstes arg wahr (≠nil), body ausführen, arg NICHT verbrauchen
    c := st.peek()
    if c != nil && c.Type != NIL {
      sub := st.sub(clauses[0])
      formatRun(sub, ctrlSlice(ctrl, clauses[0]), 0)
      st.out.WriteString(sub.out.String())
    } else {
      st.consume()
    }
    return npos
  }
  if colon {
    // ~:[a~;b]: arg=false→clause0, arg=true→clause1
    c := st.consume()
    idx := 0
    if c.Type != NIL {
      idx = 1
    }
    if idx < len(clauses) {
      sub := st.sub(clauses[idx])
      formatRun(sub, ctrlSlice(ctrl, clauses[idx]), 0)
      st.out.WriteString(sub.out.String())
    }
    return npos
  }
  // normal: arg = Index; ~:; markiert Default-Klausel (defaultIdx)
  c := st.consume()
  idx := int(toInt(c))
  if idx >= 0 && idx < len(clauses) && idx != defaultIdx {
    sub := st.sub(clauses[idx])
    formatRun(sub, ctrlSlice(ctrl, clauses[idx]), 0)
    st.out.WriteString(sub.out.String())
  } else if defaultIdx >= 0 {
    sub := st.sub(clauses[defaultIdx])
    formatRun(sub, ctrlSlice(ctrl, clauses[defaultIdx]), 0)
    st.out.WriteString(sub.out.String())
  }
  return npos
}

// ---- ~{ ~} Iteration ----

func emitIteration(st *fmtState, ctrl []rune, pos int, colon, at bool) (int, bool) {
  body, npos := findBlock(ctrl, pos, '{', '}')
  bodyRunes := ctrlSlice(ctrl, body)

  var items []*Cell
  if at {
    items = st.args[st.idx:]
    st.idx = len(st.args)
  } else {
    lst := st.consume()
    items = cellToSlice(lst)
  }

  // step führt Body einmal auf args aus; liefert verbleibende args + esc.
  step := func(args []*Cell) ([]*Cell, bool) {
    sub := &fmtState{out: st.out, args: args, idx: 0}
    _, esc := formatRun(sub, bodyRunes, 0)
    if sub.err != nil {
      st.err = sub.err
      return nil, true
    }
    if sub.idx == 0 {
      // Body verbraucht nichts → Endlosschleifen-Schutz
      return nil, esc
    }
    return args[sub.idx:], esc
  }

  if colon {
    // ~:{ — Liste von Listen: jedes Element ist eigene Subliste
    for _, outer := range items {
      subArgs := cellToSlice(outer)
      for len(subArgs) > 0 {
        rest, esc := step(subArgs)
        if st.err != nil {
          return npos, false
        }
        if rest == nil && len(subArgs) > 0 {
          break // Body verbrauchte nichts
        }
        subArgs = rest
        if esc {
          return npos, true
        }
      }
    }
    return npos, false
  }

  // normal / @: iteriere über items
  for len(items) > 0 {
    rest, esc := step(items)
    if st.err != nil {
      return npos, false
    }
    if rest == nil && len(items) > 0 {
      break
    }
    items = rest
    if esc {
      break
    }
  }
  return npos, false
}

// ---- ~( ~) Case Conversion ----

func emitCase(st *fmtState, ctrl []rune, pos int, colon, at bool) int {
  body, npos := findBlock(ctrl, pos, '(', ')')
  sub := &fmtState{out: &strings.Builder{}, args: st.args, idx: st.idx}
  formatRun(sub, ctrlSlice(ctrl, body), 0)
  if sub.err != nil {
    st.err = sub.err
    return npos
  }
  st.idx = sub.idx
  st.out.WriteString(convertCase(sub.out.String(), colon, at))
  return npos
}

func convertCase(s string, colon, at bool) string {
  rs := []rune(s)
  switch {
  case colon && at: // ~:@(...~) → upcase all
    return strings.ToUpper(s)
  case colon: // ~:(...~) → capitalize each word
    var b strings.Builder
    cap := true
    for _, r := range rs {
      if unicode.IsSpace(r) || unicode.IsPunct(r) {
        b.WriteRune(r)
        cap = true
      } else if cap {
        b.WriteRune(unicode.ToUpper(r))
        cap = false
      } else {
        b.WriteRune(unicode.ToLower(r))
      }
    }
    return b.String()
  case at: // ~@(...~) → capitalize first word only, rest downcased
    var b strings.Builder
    cap := true
    done := false
    for _, r := range rs {
      if unicode.IsSpace(r) || unicode.IsPunct(r) {
        b.WriteRune(r)
      } else if cap && !done {
        b.WriteRune(unicode.ToUpper(r))
        cap = false
        done = true
      } else {
        b.WriteRune(unicode.ToLower(r))
      }
    }
    return b.String()
  default: // ~(...~) → downcase all
    return strings.ToLower(s)
  }
}

// ---- ~^ Escape ----

// emitEscape: true = Block abbrechen.
func emitEscape(st *fmtState, params []pval, colon, at bool) bool {
  a := paramGet(params, 0)
  b := paramGet(params, 1)
  c := paramGet(params, 2)
  // Keine Parameter: abbrechen wenn keine args mehr
  if a.kind == pMissing {
    return st.idx >= len(st.args)
  }
  av := resolveInt(st, a, 0)
  if b.kind == pMissing {
    return av == 0
  }
  bv := resolveInt(st, b, 0)
  if c.kind == pMissing {
    return av <= bv
  }
  cv := resolveInt(st, c, 0)
  return av <= bv && bv <= cv
}

// ---- Block-Helper ----

// findBlock sucht das zur öffnenden Direktiv-Char gehörende schließende ~X.
// Liefert [start, end) Indizes des Inhalts (zwischen ~{ ... ~}), sowie
// npos = Position NACH dem schließenden ~}.
func findBlock(ctrl []rune, pos int, open, close rune) ([2]int, int) {
  depth := 1
  start := pos
  for pos < len(ctrl) {
    r := ctrl[pos]
    if r == '~' {
      tildeStart := pos
      pos++
      // Parameter/Modifier überspringen
      for pos < len(ctrl) && (ctrl[pos] == ',' || ctrl[pos] == ':' || ctrl[pos] == '@' ||
        ctrl[pos] == '\'' || ctrl[pos] == 'v' || ctrl[pos] == 'V' || ctrl[pos] == '#' ||
        (ctrl[pos] >= '0' && ctrl[pos] <= '9') || ctrl[pos] == '-' || ctrl[pos] == '+') {
        if ctrl[pos] == '\'' {
          pos++
        }
        pos++
      }
      if pos >= len(ctrl) {
        break
      }
      d := ctrl[pos]
      if d == open || unicode.ToUpper(d) == unicode.ToUpper(open) {
        depth++
        pos++
      } else if d == close || unicode.ToUpper(d) == unicode.ToUpper(close) {
        depth--
        pos++
        if depth == 0 {
          // Body endet VOR der schließenden Tilde
          return [2]int{start, tildeStart}, pos
        }
      } else {
        pos++
      }
    } else {
      pos++
    }
  }
  return [2]int{start, pos}, pos
}

// splitClauses teilt Block-Inhalt bei top-level ~; . Liefert Klauseln und
// defaultIdx = Index der Klausel nach einem ~:; (-1 falls kein Default).
// ~:; markiert die *folgende* Klausel als Default (CLHS 22.3.8.3).
func splitClauses(ctrl []rune, body [2]int) ([][2]int, int) {
  var clauses [][2]int
  defaultIdx := -1
  start := body[0]
  pos := body[0]
  depth := 0
  for pos < body[1] {
    r := ctrl[pos]
    if r == '~' {
      save := pos
      pos++
      // Parameter überspringen (keine Modifier hier — die tracken wir extra)
      for pos < body[1] && (ctrl[pos] == ',' || ctrl[pos] == '\'' ||
        ctrl[pos] == 'v' || ctrl[pos] == 'V' || ctrl[pos] == '#' ||
        (ctrl[pos] >= '0' && ctrl[pos] <= '9') || ctrl[pos] == '-' || ctrl[pos] == '+') {
        if ctrl[pos] == '\'' {
          pos++
        }
        pos++
      }
      // Modifier tracken (: markiert ~:; als Default-Separator)
      colon2 := false
      for pos < body[1] && (ctrl[pos] == ':' || ctrl[pos] == '@') {
        if ctrl[pos] == ':' {
          colon2 = true
        }
        pos++
      }
      if pos < body[1] {
        d := ctrl[pos]
        if d == '{' || d == '[' || d == '(' {
          depth++
        } else if d == '}' || d == ']' || d == ')' {
          depth--
        } else if d == ';' && depth == 0 {
          if colon2 {
            defaultIdx = len(clauses) + 1
          }
          clauses = append(clauses, [2]int{start, save})
          start = pos + 1
        }
        pos++
      }
    } else {
      pos++
    }
  }
  clauses = append(clauses, [2]int{start, body[1]})
  return clauses, defaultIdx
}

// ctrlSlice extrahiert ctrl[start:end] als separaten Slice.
func ctrlSlice(ctrl []rune, rng [2]int) []rune {
  if rng[1] > len(ctrl) {
    rng[1] = len(ctrl)
  }
  if rng[0] >= rng[1] {
    return nil
  }
  return ctrl[rng[0]:rng[1]]
}

// ---- fmtState-Helper für Sub-Kontexte ----

func (st *fmtState) peek() *Cell {
  if st.idx >= len(st.args) {
    return nil
  }
  return st.args[st.idx]
}

// sub erzeugt einen Sub-State mit getrenntem out (für clauses).
func (st *fmtState) sub(rng [2]int) *fmtState {
  return &fmtState{out: &strings.Builder{}, args: st.args, idx: st.idx}
}

// cellToSlice: Liste → []*Cell. Nutzt types_helpers.
func cellToSlice(c *Cell) []*Cell {
  var out []*Cell
  for c != nil && c.Type == LIST {
    out = append(out, c.Car)
    c = c.Cdr
  }
  return out
}
