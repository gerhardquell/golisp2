//**********************************************************************
//  lib/jsoncell.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260807
//**********************************************************************
// JSON <-> Cell-Konvertierung fuer die Web-Bridge (Spec TODO.md §6).
// Bewusste Asymmetrien: false/null -> Nil; leere Liste -> null (nicht {}
// oder []). Zyklische Strukturen werden ueber eine Tiefenbegrenzung
// abgefangen.
//**********************************************************************

package lib

import (
  "encoding/json"
  "fmt"
  "sort"
)

const jsonMaxDepth = 64

// CellToJSON kodiert eine Cell als JSON. NIL -> null, t -> true, NUMBER ->
// Zahl (Ganzzahlen ohne .0), STRING -> String, sonstige ATOMs -> Symbolname
// als String. LIST wird Objekt, wenn jedes Element ein dotted pair mit
// STRING/ATOM-Car ist (Cdr kein LIST) — sonst Array.
func CellToJSON(c *Cell) ([]byte, error) {
  v, err := cellToJSONValue(c, 0)
  if err != nil {
    return nil, err
  }
  return json.Marshal(v)
}

func cellToJSONValue(c *Cell, depth int) (interface{}, error) {
  if depth > jsonMaxDepth {
    return nil, fmt.Errorf("CellToJSON: Tiefe %d überschritten", jsonMaxDepth)
  }
  if c == nil {
    return nil, nil
  }
  switch c.Type {
  case NIL:
    return nil, nil
  case NUMBER:
    return c.Num, nil
  case STRING:
    return c.Val, nil
  case ATOM:
    if c == cellT {
      return true, nil
    }
    return c.Val, nil
  case LIST:
    if isAlistObject(c) {
      m := make(map[string]interface{})
      for p := c; p != nil && p.Type == LIST; p = p.Cdr {
        pair := p.Car
        val, err := cellToJSONValue(pair.Cdr, depth+1)
        if err != nil {
          return nil, err
        }
        m[pair.Car.Val] = val
      }
      return m, nil
    }
    var arr []interface{}
    p := c
    for ; p != nil && p.Type == LIST; p = p.Cdr {
      v, err := cellToJSONValue(p.Car, depth+1)
      if err != nil {
        return nil, err
      }
      arr = append(arr, v)
    }
    if p != nil && p.Type != NIL {
      return nil, fmt.Errorf("CellToJSON: improper Liste nicht darstellbar")
    }
    return arr, nil
  default:
    return nil, fmt.Errorf("CellToJSON: Typ %v nicht darstellbar", c.Type)
  }
}

// isAlistObject: LIST wird genau dann JSON-Objekt, wenn sie nicht leer ist
// und jedes Element ein Cons mit STRING/ATOM-Car ist, dessen Cdr kein LIST
// oder selbst ein Alist-Objekt ist (Rekursion fuer verschachtelte Alists).
// (("a" . 1)) -> Objekt, (("a" 1)) -> Array. Improper Listen -> false.
func isAlistObject(c *Cell) bool {
  if c == nil || c.Type != LIST {
    return false
  }
  for p := c; p != nil && p.Type == LIST; p = p.Cdr {
    elem := p.Car
    if elem == nil || elem.Type != LIST {
      return false
    }
    if elem.Car == nil || (elem.Car.Type != STRING && elem.Car.Type != ATOM) {
      return false
    }
    // Cdr darf LIST sein, wenn es selbst ein Alist-Objekt ist —
    // sonst waeren verschachtelte Alists ({"a":{"b":2}}) unmoeglich.
    if elem.Cdr != nil && elem.Cdr.Type == LIST && !isAlistObject(elem.Cdr) {
      return false
    }
  }
  // kommen wir hier an, war der Rest NIL (proper) oder die Schleife endete
  // an einem Nicht-LIST-Cdr — letzteres ist improper, kein Objekt.
  return alistProperTail(c)
}

func alistProperTail(c *Cell) bool {
  p := c
  for p != nil && p.Type == LIST {
    p = p.Cdr
  }
  return p == nil || p.Type == NIL
}

// JSONToCell parst JSON in eine Cell. null/false -> Nil, true -> t, Objekt
// -> Alist mit STRING-Keys als dotted pairs, Array -> LIST.
func JSONToCell(data []byte) (*Cell, error) {
  var v interface{}
  if err := json.Unmarshal(data, &v); err != nil {
    return nil, fmt.Errorf("JSONToCell: %v", err)
  }
  return jsonValueToCell(v, 0)
}

func jsonValueToCell(v interface{}, depth int) (*Cell, error) {
  if depth > jsonMaxDepth {
    return nil, fmt.Errorf("JSONToCell: Tiefe %d überschritten", jsonMaxDepth)
  }
  switch x := v.(type) {
  case nil:
    return MakeNil(), nil
  case bool:
    if x {
      return cellT, nil
    }
    return MakeNil(), nil
  case float64:
    return MakeNum(x), nil
  case string:
    return MakeStr(x), nil
  case []interface{}:
    items := make([]*Cell, 0, len(x))
    for _, e := range x {
      c, err := jsonValueToCell(e, depth+1)
      if err != nil {
        return nil, err
      }
      items = append(items, c)
    }
    return SliceToCell(items), nil
  case map[string]interface{}:
    keys := make([]string, 0, len(x))
    for k := range x {
      keys = append(keys, k)
    }
    sort.Strings(keys) // deterministisch: Go-Maps iterieren zufaellig
    result := MakeNil()
    for i := len(keys) - 1; i >= 0; i-- {
      val, err := jsonValueToCell(x[keys[i]], depth+1)
      if err != nil {
        return nil, err
      }
      result = Cons(Cons(MakeStr(keys[i]), val), result)
    }
    return result, nil
  default:
    return nil, fmt.Errorf("JSONToCell: Typ %T nicht darstellbar", v)
  }
}
