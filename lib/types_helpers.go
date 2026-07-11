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

// Append fügt ein Element am Ende einer Liste an
// Gibt eine neue Liste zurück (funktionaler Stil)
func Append(list, item *Cell) *Cell {
	if list == nil || list.Type == NIL {
		return Cons(item, MakeNil())
	}
	if list.Type != LIST {
		return Cons(item, MakeNil())
	}
	// Rekursiv: Kopiere die Liste und hänge an
	return Cons(list.Car, Append(list.Cdr, item))
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
	return c != nil && c.Type != NIL
}
