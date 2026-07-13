//**********************************************************************
//  lib/genalg_prims.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6, kimi
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260626
//**********************************************************************
// Lisp-Primitive fuer den genetischen Algorithmus aus genalg.go
//**********************************************************************

package lib

import (
  "encoding/binary"
  "fmt"
  "math"
  "strings"
)

// gaHandle verpackt einen GA als Lisp-Objekt in Cell.Env.
type gaHandle struct {
  ga *GA
}

// makeGACell erzeugt einen opaken Handle fuer einen GA.
func makeGACell(ga *GA) *Cell {
  return &Cell{
    Type: FUNC,
    Val:  "ga",
    Fn: func(_ []*Cell) (*Cell, error) {
      return nil, fmt.Errorf("ga: Handle ist kein aufrufbares Objekt")
    },
    Env: &gaHandle{ga: ga},
  }
}

// getGA extrahiert den GA aus einem Handle.
func getGA(c *Cell) (*GA, error) {
  if c == nil || c.Type != FUNC || c.Val != "ga" {
    return nil, fmt.Errorf("ga-*: GA-Handle erwartet")
  }
  h, ok := c.Env.(*gaHandle)
  if !ok || h == nil || h.ga == nil {
    return nil, fmt.Errorf("ga-*: GA-Handle erwartet")
  }
  return h.ga, nil
}

// parseGenType wandelt Lisp-Werte in GenType um.
func parseGenType(c *Cell) (GenType, error) {
  if c == nil {
    return -1, fmt.Errorf("ga-create: GenType erwartet")
  }
  var name string
  switch c.Type {
  case ATOM, STRING:
    name = strings.ToLower(c.Val)
  case NUMBER:
    n := int(c.Num)
    if float64(n) == c.Num && n >= 0 && n <= 5 {
      return GenType(n), nil
    }
    return -1, fmt.Errorf("ga-create: GenType-Code muss 0-5 sein")
  default:
    return -1, fmt.Errorf("ga-create: GenType erwartet")
  }
  switch name {
  case "bit1":
    return BIT1, nil
  case "bit2":
    return BIT2, nil
  case "bit4":
    return BIT4, nil
  case "bit8":
    return BIT8, nil
  case "biti", "int":
    return BITI, nil
  case "bitf", "float":
    return BITF, nil
  }
  return -1, fmt.Errorf("ga-create: unbekannter GenType '%s'", name)
}

// isCallable prueft ob ein Wert als Funktion aufgerufen werden kann.
func isCallable(c *Cell) bool {
  return c != nil && (c.Type == FUNC || c.Type == LIST)
}

// genomeToCell wandelt ein Rohes Genom-Byte-Array in eine Lisp-Liste.
func genomeToCell(ga *GA, genome []byte) (*Cell, error) {
  if ga == nil {
    return nil, fmt.Errorf("genomeToCell: ga is nil")
  }
  values := make([]*Cell, ga.genLen)
  switch ga.genType {
  case BIT1, BIT2, BIT4, BIT8:
    for gp := 0; gp < ga.genLen; gp++ {
      v, err := getBitValueRaw(genome, ga.bytesPer, ga.genType, 0, gp)
      if err != nil {
        return nil, err
      }
      values[gp] = MakeNum(float64(v))
    }
  case BITI:
    if len(genome) < ga.genLen*8 {
      return nil, fmt.Errorf("genomeToCell: BITI Genom zu kurz")
    }
    for gp := 0; gp < ga.genLen; gp++ {
      u := binary.LittleEndian.Uint64(genome[gp*8:])
      values[gp] = MakeNum(float64(int64(u)))
    }
  case BITF:
    if len(genome) < ga.genLen*8 {
      return nil, fmt.Errorf("genomeToCell: BITF Genom zu kurz")
    }
    for gp := 0; gp < ga.genLen; gp++ {
      u := binary.LittleEndian.Uint64(genome[gp*8:])
      values[gp] = MakeNum(math.Float64frombits(u))
    }
  default:
    return nil, fmt.Errorf("genomeToCell: unbekannter GenType")
  }
  return SliceToCell(values), nil
}

// makeLispGenFunc erzeugt einen Go-Fitness-Callback, der eine Lisp-Funktion aufruft.
func makeLispGenFunc(ga *GA, fn *Cell) GenFunc {
  return func(genome []byte) (score float64) {
    defer func() {
      if r := recover(); r != nil {
        score = 0
      }
    }()
    genomeCell, err := genomeToCell(ga, genome)
    if err != nil {
      return 0
    }
    result, err := apply(fn, []*Cell{genomeCell})
    if err != nil || result == nil || result.Type != NUMBER {
      return 0
    }
    return result.Num
  }
}

// fnGaCreate: (ga-create type gen-len gen-par fitness-fn) -> GA-Handle
func fnGaCreate(args []*Cell) (*Cell, error) {
  if len(args) != 4 {
    return nil, fmt.Errorf("ga-create: 4 Argumente erwartet (type gen-len gen-par fitness-fn)")
  }
  gt, err := parseGenType(args[0])
  if err != nil {
    return nil, err
  }
  if args[1] == nil || args[1].Type != NUMBER {
    return nil, fmt.Errorf("ga-create: gen-len muss Zahl sein")
  }
  if args[2] == nil || args[2].Type != NUMBER {
    return nil, fmt.Errorf("ga-create: gen-par muss Zahl sein")
  }
  genLen := int(args[1].Num)
  genPar := int(args[2].Num)
  fn := args[3]
  if !isCallable(fn) {
    return nil, fmt.Errorf("ga-create: fitness-fn muss aufrufbar sein")
  }
  // Dummy-Fitness fuer GaCreate, wird direkt ueberschrieben.
  ga, err := GaCreate(gt, genLen, genPar, func([]byte) float64 { return 0 })
  if err != nil {
    return nil, fmt.Errorf("ga-create: %v", err)
  }
  ga.genFunc = makeLispGenFunc(ga, fn)
  return makeGACell(ga), nil
}

// fnGaInit: (ga-init ga) -> t
func fnGaInit(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("ga-init: 1 Argument erwartet")
  }
  ga, err := getGA(args[0])
  if err != nil {
    return nil, err
  }
  if err := GaInit(ga); err != nil {
    return nil, fmt.Errorf("ga-init: %v", err)
  }
  return MakeAtom("t"), nil
}

// fnGaCross: (ga-cross ga codist) -> t
func fnGaCross(args []*Cell) (*Cell, error) {
  if len(args) != 2 {
    return nil, fmt.Errorf("ga-cross: 2 Argumente erwartet")
  }
  ga, err := getGA(args[0])
  if err != nil {
    return nil, err
  }
  if args[1] == nil || args[1].Type != NUMBER {
    return nil, fmt.Errorf("ga-cross: codist muss Zahl sein")
  }
  if err := GaCross(ga, int(args[1].Num)); err != nil {
    return nil, fmt.Errorf("ga-cross: %v", err)
  }
  return MakeAtom("t"), nil
}

// fnGaCalc: (ga-calc ga) -> t
func fnGaCalc(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("ga-calc: 1 Argument erwartet")
  }
  ga, err := getGA(args[0])
  if err != nil {
    return nil, err
  }
  if err := GaCalc(ga); err != nil {
    return nil, fmt.Errorf("ga-calc: %v", err)
  }
  return MakeAtom("t"), nil
}

// fnGaSelect: (ga-select ga keep) -> t
func fnGaSelect(args []*Cell) (*Cell, error) {
  if len(args) != 2 {
    return nil, fmt.Errorf("ga-select: 2 Argumente erwartet")
  }
  ga, err := getGA(args[0])
  if err != nil {
    return nil, err
  }
  if args[1] == nil || args[1].Type != NUMBER {
    return nil, fmt.Errorf("ga-select: keep muss Zahl sein")
  }
  if err := GaSelect(ga, int(args[1].Num)); err != nil {
    return nil, fmt.Errorf("ga-select: %v", err)
  }
  return MakeAtom("t"), nil
}

// fnGaResult: (ga-result ga) -> (score ...)
func fnGaResult(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("ga-result: 1 Argument erwartet")
  }
  ga, err := getGA(args[0])
  if err != nil {
    return nil, err
  }
  scores := GaResult(ga)
  cells := make([]*Cell, len(scores))
  for i, s := range scores {
    cells[i] = MakeNum(s)
  }
  return SliceToCell(cells), nil
}

// fnGaMut: (ga-mut ga mutf) -> t
func fnGaMut(args []*Cell) (*Cell, error) {
  if len(args) != 2 {
    return nil, fmt.Errorf("ga-mut: 2 Argumente erwartet")
  }
  ga, err := getGA(args[0])
  if err != nil {
    return nil, err
  }
  if args[1] == nil || args[1].Type != NUMBER {
    return nil, fmt.Errorf("ga-mut: mutf muss Zahl sein")
  }
  if err := GaMut(ga, args[1].Num); err != nil {
    return nil, fmt.Errorf("ga-mut: %v", err)
  }
  return MakeAtom("t"), nil
}

// fnGaPrint: (ga-print ga lines) -> t
func fnGaPrint(args []*Cell) (*Cell, error) {
  if len(args) != 2 {
    return nil, fmt.Errorf("ga-print: 2 Argumente erwartet")
  }
  ga, err := getGA(args[0])
  if err != nil {
    return nil, err
  }
  if args[1] == nil || args[1].Type != NUMBER {
    return nil, fmt.Errorf("ga-print: lines muss Zahl sein")
  }
  if err := GaPrint(ga, int(args[1].Num)); err != nil {
    return nil, fmt.Errorf("ga-print: %v", err)
  }
  return MakeAtom("t"), nil
}

// fnGaP: (ga? obj) -> t | ()
func fnGaP(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("ga?: 1 Argument erwartet")
  }
  if _, err := getGA(args[0]); err == nil {
    return MakeAtom("t"), nil
  }
  return MakeNil(), nil
}

// RegisterGenAlg registriert alle GA-Primitive im Environment.
func RegisterGenAlg(env *Env) {
  _ = env.Set("ga-create", makeFn(fnGaCreate))
  _ = env.Set("ga-init", makeFn(fnGaInit))
  _ = env.Set("ga-cross", makeFn(fnGaCross))
  _ = env.Set("ga-calc", makeFn(fnGaCalc))
  _ = env.Set("ga-select", makeFn(fnGaSelect))
  _ = env.Set("ga-result", makeFn(fnGaResult))
  _ = env.Set("ga-mut", makeFn(fnGaMut))
  _ = env.Set("ga-print", makeFn(fnGaPrint))
  _ = env.Set("ga?", makeFn(fnGaP))
}
