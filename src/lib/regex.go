//**********************************************************************
//  lib/regex.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Opus 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261020
//**********************************************************************
// Reguläre Ausdrücke auf Go regexp (RE2-Syntax, keine Rückverweise).
// Muster immer zuerst, wie cl-ppcre:
//   (regex-match re s)        → t / ()
//   (regex-find re s)         → erster Treffer oder (), 2. Wert: Gruppen
//   (regex-find-all re s)     → Liste aller Treffer
//   (regex-replace re s neu)  → alle Treffer ersetzt, $1/${name} für Gruppen
//   (regex-split re s)        → Liste der Teile
//**********************************************************************

package lib

import (
  "fmt"
  "regexp"
)

func RegisterRegex(env *Env) {
  _ = env.Set("regex-match",    makeFn(fnRegexMatch))
  _ = env.Set("regex-find",     makeFn(fnRegexFind))
  _ = env.Set("regex-find-all", makeFn(fnRegexFindAll))
  _ = env.Set("regex-replace",  makeFn(fnRegexReplace))
  _ = env.Set("regex-split",    makeFn(fnRegexSplit))
}

// regexArgs prüft (re s …) mit n String-Argumenten und kompiliert re.
func regexArgs(name string, args []*Cell, n int) (*regexp.Regexp, error) {
  if len(args) != n {
    return nil, fmt.Errorf("%s: %d Argumente erwartet", name, n)
  }
  for _, a := range args {
    if a.Type != STRING {
      return nil, fmt.Errorf("%s: String erwartet, got %s", name, a)
    }
  }
  re, err := regexp.Compile(args[0].Val)
  if err != nil {
    return nil, fmt.Errorf("%s: ungültiges Muster: %v", name, err)
  }
  return re, nil
}

func stringsToCell(parts []string) *Cell {
  cells := make([]*Cell, len(parts))
  for i, p := range parts {
    cells[i] = MakeStr(p)
  }
  return SliceToCell(cells)
}

func fnRegexMatch(args []*Cell) (*Cell, error) {
  re, err := regexArgs("regex-match", args, 2)
  if err != nil { return nil, err }
  if re.MatchString(args[1].Val) { return cellT, nil }
  return MakeNil(), nil
}

// regex-find: Gruppen ohne Treffer sind () — nicht "", sonst wäre
// "Gruppe leer" von "Gruppe nicht beteiligt" nicht unterscheidbar.
func fnRegexFind(args []*Cell) (*Cell, error) {
  re, err := regexArgs("regex-find", args, 2)
  if err != nil { return nil, err }
  s := args[1].Val
  idx := re.FindStringSubmatchIndex(s)
  if idx == nil {
    return MakeValues([]*Cell{MakeNil(), MakeNil()}), nil
  }
  groups := make([]*Cell, 0, len(idx)/2-1)
  for i := 2; i < len(idx); i += 2 {
    if idx[i] < 0 {
      groups = append(groups, MakeNil())
    } else {
      groups = append(groups, MakeStr(s[idx[i]:idx[i+1]]))
    }
  }
  return MakeValues([]*Cell{MakeStr(s[idx[0]:idx[1]]), SliceToCell(groups)}), nil
}

func fnRegexFindAll(args []*Cell) (*Cell, error) {
  re, err := regexArgs("regex-find-all", args, 2)
  if err != nil { return nil, err }
  return stringsToCell(re.FindAllString(args[1].Val, -1)), nil
}

func fnRegexReplace(args []*Cell) (*Cell, error) {
  re, err := regexArgs("regex-replace", args, 3)
  if err != nil { return nil, err }
  return MakeStr(re.ReplaceAllString(args[1].Val, args[2].Val)), nil
}

func fnRegexSplit(args []*Cell) (*Cell, error) {
  re, err := regexArgs("regex-split", args, 2)
  if err != nil { return nil, err }
  return stringsToCell(re.Split(args[1].Val, -1)), nil
}
