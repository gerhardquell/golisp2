//**********************************************************************
//  lib/jsoncell.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260807
//**********************************************************************
// JSON <-> Cell — einzige Abbildung, genutzt von json-parse/json-encode
// und der Web-Bridge (wsbridge.go). Seit 20261003:
//
//   JSON      → Cell                 Cell                → JSON
//   Objekt    → Hash-Tabelle (equal) Hash-Tabelle        → Objekt
//   Array     → Liste                Liste (proper)      → Array
//   true      → t                    t                   → true
//   false     → ()                   () / NIL            → []
//   null      → :null                :null / :false      → null / false
//   Zahl/Text → NUMBER/STRING        anderes Symbol      → Name als String
//
// Bewusste Asymmetrie: false kommt als [] zurück. Alists sind keine
// Objekte (mehrdeutig: (("a" "x")) wäre {"a":["x"]} und [["a","x"]]).
// Zyklen fängt die Tiefenbegrenzung ab.
//**********************************************************************

package lib

import (
  "bytes"
  "encoding/json"
  "fmt"
)

const jsonMaxDepth = 64

// RegisterJSON hängt json-parse und json-encode ins Environment ein.
func RegisterJSON(env *Env) {
  _ = env.Set("json-parse",  makeFn(fnJSONParse))
  _ = env.Set("json-encode", makeFn(fnJSONEncode))
}

// json-parse: (json-parse "text") → Cell
func fnJSONParse(args []*Cell) (*Cell, error) {
  if len(args) != 1 || args[0].Type != STRING {
    return nil, fmt.Errorf("json-parse: 1 String erwartet")
  }
  c, err := JSONToCell([]byte(args[0].Val))
  if err != nil {
    return nil, fmt.Errorf("json-parse: %v", err)
  }
  return c, nil
}

// json-encode: (json-encode wert) → kompakter JSON-String
func fnJSONEncode(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("json-encode: 1 Argument erwartet")
  }
  js, err := CellToJSON(Primary(args[0]))
  if err != nil {
    return nil, fmt.Errorf("json-encode: %v", err)
  }
  return MakeStr(string(js)), nil
}

// CellToJSON kodiert eine Cell als kompaktes JSON (Keys sortiert, kein
// HTML-Escaping).
func CellToJSON(c *Cell) ([]byte, error) {
  v, err := cellToJSONValue(c, 0)
  if err != nil {
    return nil, err
  }
  var buf bytes.Buffer
  enc := json.NewEncoder(&buf)
  enc.SetEscapeHTML(false)
  if err := enc.Encode(v); err != nil {
    return nil, fmt.Errorf("CellToJSON: %v", err)
  }
  return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

func cellToJSONValue(c *Cell, depth int) (interface{}, error) {
  if depth > jsonMaxDepth {
    return nil, fmt.Errorf("CellToJSON: Tiefe %d überschritten", jsonMaxDepth)
  }
  if c == nil {
    return []interface{}{}, nil
  }
  switch c.Type {
  case NIL:
    return []interface{}{}, nil
  case NUMBER:
    return c.Num, nil
  case STRING:
    return c.Val, nil
  case ATOM:
    switch {
    case c == cellT:
      return true, nil
    case c.Val == ":false":
      return false, nil
    case c.Val == ":null":
      return nil, nil
    }
    return c.Val, nil
  case HASHTABLE:
    return hashToJSONObject(c.Ht, depth)
  case LIST:
    arr := []interface{}{}
    p := c
    for ; p != nil && p.Type == LIST; p = p.Cdr {
      v, err := cellToJSONValue(p.Car, depth+1)
      if err != nil {
        return nil, err
      }
      arr = append(arr, v)
    }
    if p != nil && p.Type != NIL {
      return nil, fmt.Errorf("CellToJSON: improper Liste nicht darstellbar (Objekte als Hash-Tabelle)")
    }
    return arr, nil
  default:
    return nil, fmt.Errorf("CellToJSON: Typ %v nicht darstellbar", c.Type)
  }
}

// hashToJSONObject: Keys müssen Strings oder Symbole (Name) sein; zwei
// Keys mit gleichem Namen ("a" und 'a) sind ein Fehler, kein stilles
// Überschreiben.
func hashToJSONObject(ht *HashTable, depth int) (interface{}, error) {
  entries := ht.snapshot()
  m := make(map[string]interface{}, len(entries))
  for _, e := range entries {
    if e.key.Type != STRING && e.key.Type != ATOM {
      return nil, fmt.Errorf("CellToJSON: Objekt-Key muss String oder Symbol sein, got %s", e.key)
    }
    if _, dup := m[e.key.Val]; dup {
      return nil, fmt.Errorf("CellToJSON: Objekt-Key '%s' doppelt", e.key.Val)
    }
    v, err := cellToJSONValue(e.val, depth+1)
    if err != nil {
      return nil, err
    }
    m[e.key.Val] = v
  }
  return m, nil
}

// JSONToCell parst genau einen JSON-Wert (Abbildung siehe Dateikopf).
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
    return MakeAtom(":null"), nil
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
    obj := newStringHash()
    for k, raw := range x {
      val, err := jsonValueToCell(raw, depth+1)
      if err != nil {
        return nil, err
      }
      obj.Ht.putStr(k, val)
    }
    return obj, nil
  default:
    return nil, fmt.Errorf("JSONToCell: Typ %T nicht darstellbar", v)
  }
}
