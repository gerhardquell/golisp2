//**********************************************************************
//  lib/types_helpers.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260301
//**********************************************************************
// Zusätzliche Hilfsfunktionen für Cell-Manipulation
//**********************************************************************

package lib

// List baut eine Lisp-Liste aus beliebig vielen Cells
func List(cells ...*Cell) *Cell {
  return SliceToCell(cells)
}

// SliceToCell konvertiert einen Go-Slice von Cells in eine Lisp-Liste
func SliceToCell(slice []*Cell) *Cell {
  result := MakeNil()
  for i := len(slice) - 1; i >= 0; i-- {
    result = Cons(slice[i], result)
  }
  return result
}

// Append konkateniert Listen nach Common-Lisp-Semantik (variadisch):
// alle Argumente außer dem letzten werden als Listen kopiert, das letzte
// Argument wird unverändert als Cdr gesetzt. (append) → nil, (append x) → x.
// Früher war Append single-element ("snoc"); swank.lisp/flatten nutzen aber
// CL-Stil, daher diese Semantik. appendCopy kopiert list mit tail als Cdr.
func Append(lists ...*Cell) *Cell {
  if len(lists) == 0 {
    return MakeNil()
  }
  if len(lists) == 1 {
    return lists[0]
  }
  return appendCopy(lists[0], Append(lists[1:]...))
}

// appendCopy kopiert list elementweise und setzt tail als Cdr der Kopie.
// NIL/nicht-Liste als list: tail direkt bzw. dotted pair.
func appendCopy(list, tail *Cell) *Cell {
  if list == nil || list.Type == NIL {
    return tail
  }
  if list.Type != LIST {
    return Cons(list, tail)
  }
  return Cons(list.Car, appendCopy(list.Cdr, tail))
}

// MakeNumber erstellt eine NUMBER-Cell (Alias für MakeNum für Konsistenz)
func MakeNumber(n float64) *Cell {
  return MakeNum(n)
}

// MakeString erstellt eine STRING-Cell (Alias für MakeStr für Konsistenz)
func MakeString(s string) *Cell {
  return MakeStr(s)
}

// CellToSlice konvertiert eine Lisp-Liste in einen Go-Slice
// Exportierte Version der internen cellToSlice Funktion
func CellToSlice(list *Cell) []*Cell {
  var result []*Cell
  for list != nil && list.Type == LIST {
    result = append(result, list.Car)
    list = list.Cdr
  }
  return result
}

// IsTruthy prüft ob ein Wert "wahr" ist (nicht nil)
func IsTruthy(c *Cell) bool {
  c = Primary(c) // MVALUES zählt nach Primärwert (CL)
  return c != nil && c.Type != NIL
}
