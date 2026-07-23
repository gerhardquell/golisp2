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
	LAMBDA                 // (lambda (...) ...)
	FUNC                   // eingebaute Funktion
	MACRO                  // defmacro
	NIL                    // ()
	MVALUES                // (values ...) – Träger mehrerer Werte (CL)
	HASHTABLE              // CL-Hashtabelle (mutable, Pointer-Identität)
	SYMMACRO               // symbol-macrolet-Marker: Car = unausgewertete Expansion
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
	// HASHTABLE: Zeiger auf die mutable Tabelle (hashtable.go)
	Ht *HashTable
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

// MakeValues baut eine MVALUES-Cell aus einer Werteliste (CL multiple
// values). Die Werte liegen als Car/Cdr-Kette: Car = Primärwert, Cdr =
// LIST der Folgewerte. Nur MV-Konsumformen (multiple-value-*) sehen die
// Cell; alle anderen Kontexte lesen über Primary() nur den ersten Wert.
func MakeValues(vals []*Cell) *Cell {
  mv := &Cell{Type: MVALUES}
  var tail *Cell
  for i := len(vals) - 1; i >= 1; i-- {
    tail = Cons(vals[i], tail)
  }
  if len(vals) > 0 {
    mv.Car = vals[0]
    mv.Cdr = tail
  }
  return mv
}

// ValuesToSlice packt eine (mögliche) MVALUES-Cell als Slice aus.
// Nicht-MVALUES liefert Ein-Element-Slice; (values) liefert leeren Slice.
func ValuesToSlice(c *Cell) []*Cell {
  if c == nil || c.Type != MVALUES {
    return []*Cell{c}
  }
  if c.Car == nil {
    return []*Cell{}
  }
  vals := []*Cell{c.Car}
  for rest := c.Cdr; rest != nil && rest.Type == LIST; rest = rest.Cdr {
    vals = append(vals, rest.Car)
  }
  return vals
}

// Primary: der Wert, den Nicht-MV-Kontexte sehen (CL: alle Kontexte außer
// den multiple-value-*-Formen nehmen den ersten Wert, bei (values) nil).
// EINZIGE Stelle dieser Regel — Chokepoint, nicht duplizieren.
func Primary(c *Cell) *Cell {
  if c != nil && c.Type == MVALUES {
    if c.Car == nil {
      return MakeNil()
    }
    return c.Car
  }
  return c
}

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
	case LAMBDA:
		return "#<lambda>"
	case MACRO:
		return "#<macro>"
	case LIST:
		// Reader-Abkürzung drucken: (quote x) → 'x (CL-Printer)
		if c.Car != nil && c.Car.Type == ATOM && c.Car.Val == "quote" &&
			c.Cdr != nil && c.Cdr.Type == LIST &&
			(c.Cdr.Cdr == nil || c.Cdr.Cdr.Type == NIL) {
			return "'" + c.Cdr.Car.String()
		}
		return listStr(c)
	case MVALUES:
		// MV ist für Nicht-MV-Kontexte unsichtbar: Primärwert drucken
		return Primary(c).String()
	case HASHTABLE:
		return "#<hash-table>"
	case SYMMACRO:
		return "#<symbol-macro>"
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
