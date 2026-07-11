//**********************************************************************
//  lib/types.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260223
//**********************************************************************

package lib

import (
  "fmt"
)

type LispType int

const (
	ATOM   LispType = iota // Symbol: foo, bar, +
	NUMBER                 // 42, 3.14
	STRING                 // "hallo"
	LIST                   // (a b c)
	FUNC                   // eingebaute Funktion
	MACRO                  // defmacro
	NIL                    // ()
)

type Cell struct {
	Type LispType
	// Atom/String/Number
	Val string
	Num float64
	// Liste
	Car *Cell
	Cdr *Cell
	// eingebaute Funktion
	Fn func(args []*Cell) (*Cell, error)
	// Lambda-Closure: Umgebung zum Zeitpunkt der Definition
	Env interface{} // *Env – interface{} um Zirkelimport zu vermeiden
	// Quellposition (Reader/Load gestempelt, 0 = unbekannt)
	SrcFile string
	SrcLine int
}

// Singleton nil cell - vermeidet Allokationen fuer jedes ()
var nilCell = &Cell{Type: NIL}
var cellT = &Cell{Type: ATOM, Val: "t"}
var cellNil = &Cell{Type: NIL}

// Cache fuer kleine Ganzzahlen: die meisten Zahlen in Lisp-Programmen
// (Zaehler, Indizes, arithmetische Zwischenergebnisse) liegen in diesem
// Bereich. NUMBER-Cells sind unveraenderlich, daher thread-sicher.
const smallIntMin = -32768
const smallIntMax = 32767
const smallIntCount = smallIntMax - smallIntMin + 1

var smallIntCache [smallIntCount]*Cell

func init() {
  for i := 0; i < smallIntCount; i++ {
    n := float64(i + smallIntMin)
    smallIntCache[i] = &Cell{Type: NUMBER, Num: n}
  }
}

// Hilfskonstruktoren
func MakeAtom(name string) *Cell { return &Cell{Type: ATOM, Val: name} }
func MakeNum(n float64) *Cell {
  if n == float64(int64(n)) {
    i := int(n)
    if i >= smallIntMin && i <= smallIntMax {
      return smallIntCache[i-smallIntMin]
    }
  }
  return &Cell{Type: NUMBER, Num: n}
}
func MakeStr(s string) *Cell     { return &Cell{Type: STRING, Val: s} }
func MakeNil() *Cell             { return nilCell }
func Cons(car, cdr *Cell) *Cell  { return &Cell{Type: LIST, Car: car, Cdr: cdr} }

// String-Darstellung für Print
func (c *Cell) String() string {
	if c == nil {
		return "NIL"
	}
	switch c.Type {
	case NIL:
		return "()"
	case ATOM:
		return c.Val
	case NUMBER:
		if c.Num == float64(int(c.Num)) {
			return fmt.Sprintf("%d", int(c.Num))
		}
		return fmt.Sprintf("%g", c.Num)
	case STRING:
		return fmt.Sprintf("%q", c.Val)
	case FUNC:
		return "#<func>"
	case MACRO:
		return "#<macro>"
	case LIST:
		return listStr(c)
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
