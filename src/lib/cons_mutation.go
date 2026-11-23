//**********************************************************************
//  lib/cons_mutation.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Opus 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261122
//**********************************************************************
// Echte Cons-Mutation (Spec 2026-11-22): rplaca, rplacd, nconc, nthcdr,
// %last-cons — dazu copy-list, copy-tree, %list-length und der Ringlisten-Schutz
// (ringWaechter, maxTiefe), den Drucker, equal?, Hash-Schlüssel,
// properList, CellToSlice und die Sandbox-Grenze teilen.
//
// Verändert werden ausschließlich Cells mit Type == LIST: NIL, Symbole
// und Zahlen bleiben unberührt, Singleton-NIL, Interning und der
// Small-Int-Cache sind dadurch sicher.
//**********************************************************************

package lib

import "fmt"

// maxTiefe begrenzt die Car-Rekursion (Drucker, equal?, Hash-Schlüssel,
// copy-tree): ein Ring über die Car-Seite liefe sonst in einen fatalen
// Go-Stack-Overflow.
const maxTiefe = 10000

// ringWaechter: Brent-Zykluserkennung auf der Cdr-Kette, O(1) Speicher.
// Nach jedem Cdr-Schritt zirkulaer mit der neuen LIST-Cell aufrufen.
type ringWaechter struct {
  marke            *Cell
  potenz, schritte int
}

func neuerRing(start *Cell) ringWaechter { return ringWaechter{marke: start, potenz: 1} }

func (r *ringWaechter) zirkulaer(c *Cell) bool {
  if c == r.marke {
    return true
  }
  r.schritte++
  if r.schritte == r.potenz {
    r.marke, r.potenz, r.schritte = c, r.potenz*2, 0
  }
  return false
}

func errZirkulaer(name string) error { return fmt.Errorf("%s: zirkuläre Liste", name) }

func errZuTief(name string) error {
  return fmt.Errorf("%s: zirkuläre Liste oder Car-Tiefe über %d", name, maxTiefe)
}

// RegisterConsMutation registriert die Mutations- und Ring-sicheren
// Listen-Primitiven (BaseEnv; in der Sandbox erlaubt).
func RegisterConsMutation(env *Env) {
  _ = env.Set("rplaca", makeFn(fnRplaca))
  _ = env.Set("rplacd", makeFn(fnRplacd))
  _ = env.Set("nconc", makeFn(fnNconc))
  _ = env.Set("nthcdr", makeFn(fnNthcdr))
  _ = env.Set("%last-cons", makeFn(fnLastCons))
  _ = env.Set("%list-length", makeFn(fnListLength))
  _ = env.Set("copy-list", makeFn(fnCopyList))
  _ = env.Set("copy-tree", makeFn(fnCopyTree))
}

// consArg: das erste von genau zwei Argumenten muss eine Cons-Zelle sein.
func consArg(name string, args []*Cell) (*Cell, error) {
  if len(args) != 2 {
    return nil, fmt.Errorf("%s: 2 Argumente nötig", name)
  }
  if args[0] == nil || args[0].Type != LIST {
    return nil, fmt.Errorf("%s: Cons-Zelle erwartet, got %s", name, args[0])
  }
  return args[0], nil
}

// rplaca: (rplaca c v) → c, Car von c ist jetzt v.
func fnRplaca(args []*Cell) (*Cell, error) {
  c, err := consArg("rplaca", args)
  if err != nil {
    return nil, err
  }
  c.Car = args[1]
  return c, nil
}

// rplacd: (rplacd c v) → c, Cdr von c ist jetzt v.
func fnRplacd(args []*Cell) (*Cell, error) {
  c, err := consArg("rplacd", args)
  if err != nil {
    return nil, err
  }
  c.Cdr = args[1]
  return c, nil
}

// leer: () — auch ein Go-nil-Listenende (MakeValues, Cons(x, nil)).
func leer(c *Cell) bool { return c == nil || c.Type == NIL }

// lastCons: letzte Cons-Zelle einer nicht-leeren Liste; bei Punktpaar
// oder Ring ein Fehler unter name.
func lastCons(name string, l *Cell) (*Cell, error) {
  ring := neuerRing(l)
  for {
    next := l.Cdr
    if leer(next) {
      return l, nil
    }
    if next.Type != LIST {
      return nil, fmt.Errorf("%s: keine echte Liste: %s", name, l)
    }
    if ring.zirkulaer(next) {
      return nil, errZirkulaer(name)
    }
    l = next
  }
}

// nconc: (nconc l1 … ln) hängt destruktiv an, ()-Argumente werden
// übersprungen, das letzte Argument darf alles sein (CL).
func fnNconc(args []*Cell) (*Cell, error) {
  var kopf, ende *Cell
  for i, a := range args {
    if leer(a) {
      continue
    }
    if kopf == nil {
      kopf = a
    } else {
      ende.Cdr = a
    }
    if i == len(args)-1 {
      break
    }
    if a.Type != LIST {
      return nil, fmt.Errorf("nconc: Liste erwartet, got %s", a)
    }
    var err error
    if ende, err = lastCons("nconc", a); err != nil {
      return nil, err
    }
  }
  if kopf == nil {
    return MakeNil(), nil
  }
  return kopf, nil
}

// nthcdr: (nthcdr n l) → n-ter Rest ohne Kopie; über das Ende → ().
func fnNthcdr(args []*Cell) (*Cell, error) {
  n, lst, err := listIndexArgs("nthcdr", args, 0)
  if err != nil {
    return nil, err
  }
  if n < 0 {
    return nil, fmt.Errorf("nthcdr: nicht-negative Ganzzahl erwartet, got %d", n)
  }
  return dropCells(lst, n)
}

// %last-cons: letzte Cons-Zelle (intern; last liefert das Element).
func fnLastCons(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("%%last-cons: 1 Argument nötig")
  }
  if leer(args[0]) {
    return MakeNil(), nil
  }
  if args[0].Type != LIST {
    return nil, fmt.Errorf("%%last-cons: Liste erwartet, got %s", args[0])
  }
  return lastCons("%last-cons", args[0])
}

// %list-length: Länge einer echten Liste für length (stdlib.lisp).
func fnListLength(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("length: 1 Argument nötig")
  }
  n := 0
  ring := neuerRing(args[0])
  p := args[0]
  for ; p != nil && p.Type == LIST; p = p.Cdr {
    if n > 0 && ring.zirkulaer(p) {
      return nil, errZirkulaer("length")
    }
    n++
  }
  if p != nil && p.Type != NIL {
    return nil, fmt.Errorf("length: keine echte Liste: %s", args[0])
  }
  return MakeInt(int64(n)), nil
}

// copy-list: (copy-list l) — neue Cons-Zellen über die Cdr-Kette, Cars
// geteilt, gepunktetes Ende bleibt; iterativ und Ring-sicher.
func fnCopyList(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("copy-list: 1 Argument nötig")
  }
  if leer(args[0]) {
    return MakeNil(), nil
  }
  if args[0].Type != LIST {
    return nil, fmt.Errorf("copy-list: Liste erwartet, got %s", args[0])
  }
  var kopf, ende *Cell
  ring := neuerRing(args[0])
  for p := args[0]; ; {
    z := Cons(p.Car, MakeNil())
    if kopf == nil {
      kopf = z
    } else {
      ende.Cdr = z
    }
    ende = z
    p = p.Cdr
    if leer(p) {
      return kopf, nil
    }
    if p.Type != LIST {
      ende.Cdr = p
      return kopf, nil
    }
    if ring.zirkulaer(p) {
      return nil, errZirkulaer("copy-list")
    }
  }
}

// copy-tree: (copy-tree x) — neue Cons-Zellen bis in jede Tiefe; Atome,
// Zahlen, Strings und alles Nicht-LIST bleiben geteilt.
func fnCopyTree(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("copy-tree: 1 Argument nötig")
  }
  return copyTree("copy-tree", args[0], 0)
}

// copyTree: Baumkopie, Cdr iterativ (lange Listen), Car rekursiv.
// Gemeinsame Quelle von copy-tree und der Sandbox-Grenze.
func copyTree(name string, c *Cell, tiefe int) (*Cell, error) {
  if c == nil || c.Type != LIST {
    return c, nil
  }
  if tiefe > maxTiefe {
    return nil, errZuTief(name)
  }
  var kopf, ende *Cell
  ring := neuerRing(c)
  for p := c; ; {
    car, err := copyTree(name, p.Car, tiefe+1)
    if err != nil {
      return nil, err
    }
    z := Cons(car, MakeNil())
    if kopf == nil {
      kopf = z
    } else {
      ende.Cdr = z
    }
    ende = z
    p = p.Cdr
    if leer(p) {
      return kopf, nil
    }
    if p.Type != LIST {
      ende.Cdr = p
      return kopf, nil
    }
    if ring.zirkulaer(p) {
      return nil, errZirkulaer(name)
    }
  }
}
