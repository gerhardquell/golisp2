//**********************************************************************
//  lib/reader.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260223
//**********************************************************************
// Der Reader: String → Cell-Baum
// Beispiel: "(+ 1 2)" → LIST[ATOM(+), NUMBER(1), NUMBER(2)]
//**********************************************************************

package lib

import (
  "fmt"
  "strconv"
  "strings"
  "unicode"
)

type Reader struct {
  src  []rune
  pos  int
  line int
}

func NewReader(s string) *Reader {
  return &Reader{src: []rune(s), pos: 0, line: 1}
}

// Read liest einen Ausdruck aus dem Eingabestring
func Read(s string) (*Cell, error) {
  r := NewReader(strings.TrimSpace(s))
  return r.readExpr()
}

// ReadAll liest alle Ausdrücke aus dem String und gibt sie als Liste
// zurück. Leerer/Whitespace-only-String → NIL. Für REPL-Eingaben mit
// mehreren Formen pro Block (z.B. swank:listener-eval).
func ReadAll(s string) (*Cell, error) {
  r := NewReader(s)
  var forms []*Cell
  for {
    r.skipWS()
    if r.pos >= len(r.src) { break }
    cell, err := r.readExpr()
    if err != nil { return nil, err }
    forms = append(forms, cell)
  }
  return SliceToCell(forms), nil
}

func (r *Reader) peek() (rune, bool) {
  if r.pos >= len(r.src) { return 0, false }
  return r.src[r.pos], true
}

func (r *Reader) next() (rune, bool) {
  ch, ok := r.peek()
  if ok {
    r.pos++
    if ch == '\n' {
      r.line++
    }
  }
  return ch, ok
}

func (r *Reader) skipWS() {
  for {
    ch, ok := r.peek()
    if !ok { break }
    if ch == ';' {           // Kommentar bis Zeilenende
      for {
        c, ok := r.next()
        if !ok || c == '\n' { break }
      }
      continue
    }
    // Shebang (#!/usr/local/bin/golisp2): überall Kommentar bis Zeilenende
    // (SBCL-Konvention). Muss vor readDispatch greifen, sonst Fehler
    // "unbekanntes Dispatch-Zeichen #!".
    if ch == '#' && r.pos+1 < len(r.src) && r.src[r.pos+1] == '!' {
      for {
        c, ok := r.next()
        if !ok || c == '\n' { break }
      }
      continue
    }
    if !unicode.IsSpace(ch) { break }
    r.next()
  }
}

func (r *Reader) readExpr() (*Cell, error) {
  r.skipWS()
  ch, ok := r.peek()
  if !ok { return MakeNil(), nil }

  switch {
  case ch == '(':  return r.readList()
  case ch == '\'': return r.readQuote()
  case ch == '`':  return r.readQuasiquote()
  case ch == ',':  return r.readUnquote()
  case ch == '"':  return r.readString()
  case ch == '#':  return r.readDispatch()
  default:         return r.readAtomOrNum()
  }
}

// readList liest (a b c) → verschachtelte Cons-Zellen
func (r *Reader) readList() (*Cell, error) {
  startLine := r.line
  r.next() // '(' überspringen
  r.skipWS()

  ch, ok := r.peek()
  if !ok { return nil, fmt.Errorf("reader: unerwartetes EOF in Liste") }
  if ch == ')' { r.next(); return MakeNil(), nil }  // leere Liste ()

  // erstes Element
  car, err := r.readExpr()
  if err != nil { return nil, err }

  // Rest der Liste rekursiv
  cdr, err := r.readRest()
  if err != nil { return nil, err }

  cell := Cons(car, cdr)
  cell.SrcLine = startLine
  return cell, nil
}

func (r *Reader) readRest() (*Cell, error) {
  r.skipWS()
  ch, ok := r.peek()
  if !ok { return nil, fmt.Errorf("reader: fehlendes )") }

  if ch == ')' { r.next(); return MakeNil(), nil }

  // Punkt-Notation: (a . b)
  if ch == '.' {
    r.next()
    cdr, err := r.readExpr()
    if err != nil { return nil, err }
    r.skipWS()
    // Explizit ')' prüfen – früher blind r.next() (verschluckte Müll
    // wie "(a . b x)" still). Siehe Todo #7, Reader-Test-Erkenntnis.
    closer, ok := r.peek()
    if !ok || closer != ')' {
      return nil, fmt.Errorf("reader: ) nach dotted pair erwartet, got %q", closer)
    }
    r.next() // ')' konsumieren
    return cdr, nil
  }

  car, err := r.readExpr()
  if err != nil { return nil, err }
  cdr, err := r.readRest()
  if err != nil { return nil, err }
  return Cons(car, cdr), nil
}

// readQuote: 'x → (quote x)
func (r *Reader) readQuote() (*Cell, error) {
  r.next() // '\'' überspringen
  expr, err := r.readExpr()
  if err != nil { return nil, err }
  return Cons(MakeAtom("quote"), Cons(expr, MakeNil())), nil
}

// readQuasiquote: `x → (quasiquote x)
func (r *Reader) readQuasiquote() (*Cell, error) {
  r.next() // '`' überspringen
  expr, err := r.readExpr()
  if err != nil { return nil, err }
  return Cons(MakeAtom("quasiquote"), Cons(expr, MakeNil())), nil
}

// readUnquote: ,x → (unquote x)  und  ,@x → (unquote-splice x)
func (r *Reader) readUnquote() (*Cell, error) {
  r.next() // ',' überspringen
  if ch, ok := r.peek(); ok && ch == '@' {
    r.next()
    expr, err := r.readExpr()
    if err != nil { return nil, err }
    return Cons(MakeAtom("unquote-splice"), Cons(expr, MakeNil())), nil
  }
  expr, err := r.readExpr()
  if err != nil { return nil, err }
  return Cons(MakeAtom("unquote"), Cons(expr, MakeNil())), nil
}

// readString: "hallo welt"
func (r *Reader) readString() (*Cell, error) {
  r.next() // öffnendes " überspringen
  var sb strings.Builder
  for {
    ch, ok := r.next()
    if !ok { return nil, fmt.Errorf("reader: ungeschlossener String") }
    if ch == '"' { break }
    if ch == '\\' {
      esc, ok := r.next()
      if !ok { return nil, fmt.Errorf("reader: EOF nach \\") }
      switch esc {
      case 'n':  sb.WriteRune('\n')
      case 't':  sb.WriteRune('\t')
      case '"':  sb.WriteRune('"')
      case '\\': sb.WriteRune('\\')
      default:   sb.WriteRune(esc)
      }
      continue
    }
    sb.WriteRune(ch)
  }
  return MakeStr(sb.String()), nil
}

// readDispatch: #' → (function expr)
func (r *Reader) readDispatch() (*Cell, error) {
  r.next() // '#'
  ch, ok := r.peek()
  if !ok { return nil, fmt.Errorf("reader: EOF nach #") }
  if ch == '\'' {
    r.next()
    expr, err := r.readExpr()
    if err != nil { return nil, err }
    return Cons(MakeAtom("function"), Cons(expr, MakeNil())), nil
  }
  return nil, fmt.Errorf("reader: unbekanntes Dispatch-Zeichen #%c", ch)
}

// readAtomOrNum: Symbol oder Zahl
func (r *Reader) readAtomOrNum() (*Cell, error) {
  var sb strings.Builder
  for {
    ch, ok := r.peek()
    if !ok { break }
    if unicode.IsSpace(ch) || ch == '(' || ch == ')' || ch == '"' || ch == '`' || ch == ',' { break }
    sb.WriteRune(ch)
    r.next()
  }
  token := sb.String()
  if token == "" { return nil, fmt.Errorf("reader: leeres Token") }

  // Zahl?
  if n, err := strconv.ParseFloat(token, 64); err == nil {
    return MakeNum(n), nil
  }
  // Sonderfälle
  if token == "nil" || token == "NIL" { return MakeNil(), nil }

  return MakeAtom(token), nil
}
