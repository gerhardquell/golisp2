//**********************************************************************
//  lib/types.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260223
//**********************************************************************

package lib

import "fmt"

type LispType int

const (
  ATOM    LispType = iota // Symbol: foo, bar, +
  NUMBER                  // 42, 3.14
  STRING                  // "hallo"
  LIST                    // (a b c)
  FUNC                    // eingebaute Funktion
  MACRO                   // defmacro
  NIL                     // ()
)

type Cell struct {
  Type LispType
  // Atom/String/Number
  Val  string
  Num  float64
  // Liste
  Car  *Cell
  Cdr  *Cell
  // eingebaute Funktion
  Fn   func(args []*Cell) (*Cell, error)
  // Lambda-Closure: Umgebung zum Zeitpunkt der Definition
  Env  interface{} // *Env – interface{} um Zirkelimport zu vermeiden
  // Quellposition (Reader/Load gestempelt, 0 = unbekannt)
  SrcFile string
  SrcLine int
}

// Singleton nil cell - vermeidet Allokationen für jedes ()
var nilCell = &Cell{Type: NIL}

// Hilfskonstruktoren
func MakeAtom(name string) *Cell   { return &Cell{Type: ATOM, Val: name} }
func MakeNum(n float64) *Cell      { return &Cell{Type: NUMBER, Num: n} }
func MakeStr(s string) *Cell       { return &Cell{Type: STRING, Val: s} }
func MakeNil() *Cell               { return nilCell }
func Cons(car, cdr *Cell) *Cell    { return &Cell{Type: LIST, Car: car, Cdr: cdr} }

// String-Darstellung für Print
func (c *Cell) String() string {
  if c == nil { return "NIL" }
  switch c.Type {
  case NIL:    return "()"
  case ATOM:   return c.Val
  case NUMBER:
    if c.Num == float64(int(c.Num)) {
      return fmt.Sprintf("%d", int(c.Num))
    }
    return fmt.Sprintf("%g", c.Num)
  case STRING: return fmt.Sprintf("%q", c.Val)
  case FUNC:   return "#<func>"
  case MACRO:  return "#<macro>"
  case LIST:   return listStr(c)
  }
  return "?"
}

func listStr(c *Cell) string {
  s := "("
  for c != nil && c.Type == LIST {
    s += c.Car.String()
    if c.Cdr != nil && c.Cdr.Type != NIL {
      if c.Cdr.Type != LIST {
        // Dotted pair: (a . b)
        s += " . " + c.Cdr.String()
        break
      }
      s += " "
    }
    c = c.Cdr
  }
  return s + ")"
}

// LispError: Lisp-Laufzeitfehler, von (error msg) ausgelöst
// Unterscheidet sich von internen Go-Fehlern (fmt.Errorf)
type LispError struct {
  Msg *Cell
}

func (e *LispError) Error() string {
  if e.Msg.Type == STRING || e.Msg.Type == ATOM {
    return e.Msg.Val
  }
  return e.Msg.String()
}
