//**********************************************************************
//  lib/stringfuncs.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260224
//**********************************************************************
// String-Primitiven für GoLisp
// string-length, string-append, substring,
// string-upcase, string-downcase, string->number, number->string
//**********************************************************************

package lib

import (
  "fmt"
  "strconv"
  "strings"
)

func RegisterStringFuncs(env *Env) {
  _ = env.Set("string-length",   makeFn(fnStringLength))
  _ = env.Set("string-append",   makeFn(fnStringAppend))
  _ = env.Set("substring",       makeFn(fnSubstring))
  _ = env.Set("string-upcase",   makeFn(fnStringUpcase))
  _ = env.Set("string-downcase", makeFn(fnStringDowncase))
  _ = env.Set("string->number",  makeFn(fnStringToNumber))
  _ = env.Set("number->string",  makeFn(fnNumberToString))
  _ = env.Set("parse-int",       makeFn(fnParseInt))
  _ = env.Set("parse-float",     makeFn(fnParseFloat))
  _ = env.Set("string->list",    makeFn(fnStringToList))
  _ = env.Set("list->string",    makeFn(fnListToString))
  _ = env.Set("string-replace",  makeFn(fnStringReplace))
  _ = env.Set("string-trim",     makeFn(fnStringTrim))
  _ = env.Set("string-contains", makeFn(fnStringContains))
  _ = env.Set("string-split",    makeFn(fnStringSplit))
  _ = env.Set("string-join",     makeFn(fnStringJoin))
}

// string-split: (string-split s [sep]) → Liste von Strings
// Mit sep: wörtlich trennen, leere Felder bleiben. Ohne sep: an Whitespace,
// leere Felder entfallen (wie awk).
func fnStringSplit(args []*Cell) (*Cell, error) {
  if len(args) < 1 || len(args) > 2 {
    return nil, fmt.Errorf("string-split: 1 oder 2 Argumente erwartet (s [sep])")
  }
  if args[0].Type != STRING {
    return nil, fmt.Errorf("string-split: String erwartet, got %s", args[0])
  }
  var parts []string
  if len(args) == 1 {
    parts = strings.Fields(args[0].Val)
  } else {
    if args[1].Type != STRING || args[1].Val == "" {
      return nil, fmt.Errorf("string-split: Trenner muss nichtleerer String sein, got %s", args[1])
    }
    parts = strings.Split(args[0].Val, args[1].Val)
  }
  cells := make([]*Cell, len(parts))
  for i, p := range parts {
    cells[i] = MakeStr(p)
  }
  return SliceToCell(cells), nil
}

// string-join: (string-join liste sep) → String; sep Pflicht — ohne
// Trenner ist das list->string.
func fnStringJoin(args []*Cell) (*Cell, error) {
  if len(args) != 2 {
    return nil, fmt.Errorf("string-join: 2 Argumente erwartet (liste sep); ohne Trenner: list->string")
  }
  if args[1].Type != STRING {
    return nil, fmt.Errorf("string-join: Trenner muss String sein, got %s", args[1])
  }
  var parts []string
  lst := args[0]
  for lst != nil && lst.Type == LIST {
    if lst.Car.Type != STRING {
      return nil, fmt.Errorf("string-join: Alle Elemente müssen Strings sein, got %s", lst.Car)
    }
    parts = append(parts, lst.Car.Val)
    lst = lst.Cdr
  }
  if lst != nil && lst.Type != NIL {
    return nil, fmt.Errorf("string-join: echte Liste erwartet, got %s", args[0])
  }
  return MakeStr(strings.Join(parts, args[1].Val)), nil
}

// string-replace: (string-replace str old new) → str mit allen Ersetzungen
func fnStringReplace(args []*Cell) (*Cell, error) {
  if len(args) < 3 {
    return nil, fmt.Errorf("string-replace: 3 Argumente nötig")
  }
  return MakeStr(strings.ReplaceAll(args[0].Val, args[1].Val, args[2].Val)), nil
}

// string-trim: (string-trim str) → str ohne führende/abschließende Whitespace
func fnStringTrim(args []*Cell) (*Cell, error) {
  if len(args) < 1 {
    return nil, fmt.Errorf("string-trim: 1 Argument nötig")
  }
  return MakeStr(strings.TrimSpace(args[0].Val)), nil
}

// string-contains: (string-contains str sub) → t oder nil
func fnStringContains(args []*Cell) (*Cell, error) {
  if len(args) < 2 {
    return nil, fmt.Errorf("string-contains: 2 Argumente nötig")
  }
  if strings.Contains(args[0].Val, args[1].Val) {
    return MakeAtom("t"), nil
  }
  return MakeNil(), nil
}

func fnStringLength(args []*Cell) (*Cell, error) {
  if len(args) < 1 || args[0].Type != STRING {
    return nil, fmt.Errorf("string-length: String erwartet")
  }
  return MakeNum(float64(len([]rune(args[0].Val)))), nil
}

func fnStringAppend(args []*Cell) (*Cell, error) {
  var sb strings.Builder
  for i, a := range args {
    if a == nil {
      return nil, fmt.Errorf("string-append: Argument %d ist nil", i+1)
    }
    if a.Type != STRING {
      return nil, fmt.Errorf("string-append: Argument %d ist kein String", i+1)
    }
    sb.WriteString(a.Val)
  }
  return MakeStr(sb.String()), nil
}

func fnSubstring(args []*Cell) (*Cell, error) {
  if len(args) < 3 {
    return nil, fmt.Errorf("substring: 3 Argumente nötig (string start end)")
  }
  if args[0].Type != STRING {
    return nil, fmt.Errorf("substring: erstes Argument muss String sein")
  }
  runes := []rune(args[0].Val)
  start := int(args[1].Num)
  end   := int(args[2].Num)
  if start < 0 || end > len(runes) || start > end {
    return nil, fmt.Errorf("substring: Index außerhalb des Bereichs")
  }
  return MakeStr(string(runes[start:end])), nil
}

func fnStringUpcase(args []*Cell) (*Cell, error) {
  if len(args) < 1 || args[0].Type != STRING {
    return nil, fmt.Errorf("string-upcase: String erwartet")
  }
  return MakeStr(strings.ToUpper(args[0].Val)), nil
}

func fnStringDowncase(args []*Cell) (*Cell, error) {
  if len(args) < 1 || args[0].Type != STRING {
    return nil, fmt.Errorf("string-downcase: String erwartet")
  }
  return MakeStr(strings.ToLower(args[0].Val)), nil
}

func fnStringToNumber(args []*Cell) (*Cell, error) {
  if len(args) < 1 || args[0].Type != STRING {
    return nil, fmt.Errorf("string->number: String erwartet")
  }
  n, err := strconv.ParseFloat(args[0].Val, 64)
  if err != nil {
    return nil, fmt.Errorf("string->number: '%s' ist keine Zahl", args[0].Val)
  }
  return MakeNum(n), nil
}

func fnParseInt(args []*Cell) (*Cell, error) {
  if len(args) < 1 || args[0].Type != STRING {
    return nil, fmt.Errorf("parse-int: String erwartet")
  }
  n, err := strconv.ParseInt(strings.TrimSpace(args[0].Val), 10, 64)
  if err != nil {
    return nil, fmt.Errorf("parse-int: '%s' ist keine Ganzzahl", args[0].Val)
  }
  return MakeNum(float64(n)), nil
}

func fnParseFloat(args []*Cell) (*Cell, error) {
  if len(args) < 1 || args[0].Type != STRING {
    return nil, fmt.Errorf("parse-float: String erwartet")
  }
  n, err := strconv.ParseFloat(strings.TrimSpace(args[0].Val), 64)
  if err != nil {
    return nil, fmt.Errorf("parse-float: '%s' ist keine Fließkommazahl", args[0].Val)
  }
  return MakeNum(n), nil
}

func fnNumberToString(args []*Cell) (*Cell, error) {
  if len(args) < 1 || args[0].Type != NUMBER {
    return nil, fmt.Errorf("number->string: Zahl erwartet")
  }
  if args[0].Num == float64(int(args[0].Num)) {
    return MakeStr(fmt.Sprintf("%d", int(args[0].Num))), nil
  }
  return MakeStr(fmt.Sprintf("%g", args[0].Num)), nil
}

// string->list: Wandelt String in Liste von Single-Character-Strings um
func fnStringToList(args []*Cell) (*Cell, error) {
  if len(args) < 1 || args[0].Type != STRING {
    return nil, fmt.Errorf("string->list: String erwartet")
  }
  runes := []rune(args[0].Val)
  result := MakeNil()
  for i := len(runes) - 1; i >= 0; i-- {
    result = Cons(MakeStr(string(runes[i])), result)
  }
  return result, nil
}

// list->string: Verkettet Liste von Strings zu einem String
func fnListToString(args []*Cell) (*Cell, error) {
  if len(args) < 1 {
    return nil, fmt.Errorf("list->string: 1 Argument nötig")
  }
  lst := args[0]
  var sb strings.Builder
  for lst != nil && lst.Type == LIST {
    elem := lst.Car
    if elem.Type != STRING {
      return nil, fmt.Errorf("list->string: Alle Elemente müssen Strings sein")
    }
    sb.WriteString(elem.Val)
    lst = lst.Cdr
  }
  if lst != nil && lst.Type != NIL {
    return nil, fmt.Errorf("list->string: echte Liste erwartet, got %s", lst.String())
  }
  return MakeStr(sb.String()), nil
}
