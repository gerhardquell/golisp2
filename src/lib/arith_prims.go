//**********************************************************************
//  lib/arith_prims.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261001
//**********************************************************************
// Arithmetik, Rundung (CL: floor/ceiling/truncate/round) und Vergleiche
// (numerisch, equal?, eq). Aus primitives.go herausgelöst.
//**********************************************************************

package lib

import (
	"fmt"
	"math"
	"math/rand"
)

// RegisterArith registriert Arithmetik- und Vergleichs-Primitiven.
func RegisterArith(env *Env) {
	// Arithmetik
	_ = env.Set("+", makeFn(fnAdd))
	_ = env.Set("-", makeFn(fnSub))
	_ = env.Set("*", makeFn(fnMul))
	_ = env.Set("/", makeFn(fnDiv))
	_ = env.Set("mod", makeFn(fnMod))
	_ = env.Set("rem", makeFn(fnRem))
	_ = env.Set("remainder", makeFn(fnRem))
	_ = env.Set("abs", makeFn(fnAbs))
	_ = env.Set("floor", makeFn(fnFloor))
	_ = env.Set("ceiling", makeFn(fnCeiling))
	_ = env.Set("truncate", makeFn(fnTruncate))
	_ = env.Set("round", makeFn(fnRound))
	_ = env.Set("random", makeFn(fnRandom))

	// Vergleiche
	_ = env.Set("=", makeFn(fnEq))
	_ = env.Set("<", makeFn(fnLt))
	_ = env.Set(">", makeFn(fnGt))
	_ = env.Set(">=", makeFn(fnGe))
	_ = env.Set("<=", makeFn(fnLe))
	_ = env.Set("equal?", makeFn(fnEqual))
	_ = env.Set("eq", makeFn(fnEqPtr))
	_ = env.Set("eql", makeFn(fnEql))
}

// ---- Arithmetik ----

// checkNumbers stellt sicher, dass alle args NUMBER sind. Arithmetik und
// numerische Vergleiche greifen direkt auf .Num zu – ohne Check würden
// Strings (Num=0) oder Listen still als 0 einfließen (stiller Datenverlust).
func checkNumbers(name string, args []*Cell) error {
	for _, a := range args {
		if a == nil || a.Type != NUMBER {
			return fmt.Errorf("%s: Zahl erwartet, got %s", name, a)
		}
	}
	return nil
}

func fnAdd(args []*Cell) (*Cell, error) {
	if err := checkNumbers("+", args); err != nil {
		return nil, err
	}
	n := 0.0
	for _, a := range args {
		n += a.Num
	}
	return MakeNum(n), nil
}

func fnSub(args []*Cell) (*Cell, error) {
	if err := checkNumbers("-", args); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return MakeNum(0), nil
	}
	if len(args) == 1 {
		return MakeNum(-args[0].Num), nil // (- x) negiert (CL)
	}
	n := args[0].Num
	for _, a := range args[1:] {
		n -= a.Num
	}
	return MakeNum(n), nil
}

func fnMul(args []*Cell) (*Cell, error) {
	if err := checkNumbers("*", args); err != nil {
		return nil, err
	}
	n := 1.0
	for _, a := range args {
		n *= a.Num
	}
	return MakeNum(n), nil
}

func fnDiv(args []*Cell) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("/: min. 2 Argumente")
	}
	if err := checkNumbers("/", args); err != nil {
		return nil, err
	}
	if args[1].Num == 0 {
		return nil, fmt.Errorf("/: Division durch 0")
	}
	return MakeNum(args[0].Num / args[1].Num), nil
}

// ---- Vergleiche (numerisch, strict) ----

func fnEq(args []*Cell) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("=: 2 Argumente nötig")
	}
	if err := checkNumbers("=", args); err != nil {
		return nil, err
	}
	if args[0].Num == args[1].Num {
		return cellT, nil
	}
	return MakeNil(), nil
}

func fnLt(args []*Cell) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("<: 2 Argumente nötig")
	}
	if err := checkNumbers("<", args); err != nil {
		return nil, err
	}
	if args[0].Num < args[1].Num {
		return cellT, nil
	}
	return MakeNil(), nil
}

func fnGt(args []*Cell) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf(">: 2 Argumente nötig")
	}
	if err := checkNumbers(">", args); err != nil {
		return nil, err
	}
	if args[0].Num > args[1].Num {
		return cellT, nil
	}
	return MakeNil(), nil
}

func fnGe(args []*Cell) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf(">=: 2 Argumente nötig")
	}
	if err := checkNumbers(">=", args); err != nil {
		return nil, err
	}
	if args[0].Num >= args[1].Num {
		return cellT, nil
	}
	return MakeNil(), nil
}

func fnLe(args []*Cell) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("<=: 2 Argumente nötig")
	}
	if err := checkNumbers("<=", args); err != nil {
		return nil, err
	}
	if args[0].Num <= args[1].Num {
		return cellT, nil
	}
	return MakeNil(), nil
}

func fnEqual(args []*Cell) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("equal?: 2 Argumente nötig")
	}
	if cellEqual(args[0], args[1]) {
		return cellT, nil
	}
	return MakeNil(), nil
}

// eq: Pointer-Gleichheit (identisches Objekt im Speicher).
// Zahlen werden trotz Small-Integer-Cache als nicht-identisch betrachtet,
// damit (eq 5 5) weiterhin () liefert.
func fnEqPtr(args []*Cell) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("eq: 2 Argumente nötig")
	}
	if args[0] == args[1] {
		if args[0] != nil && args[0].Type == NUMBER {
			return MakeNil(), nil
		}
		return cellT, nil
	}
	return MakeNil(), nil
}

// eql: Zahlen nach Wert (golisp2 kennt nur float64, also (eql 1.0 1) → t),
// sonst Pointer-Identität wie eq. Default-Test der Sequenzfunktionen.
func eql(a, b *Cell) bool {
	if a != nil && b != nil && a.Type == NUMBER && b.Type == NUMBER {
		return a.Num == b.Num
	}
	return a == b
}

func fnEql(args []*Cell) (*Cell, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("eql: 2 Argumente nötig")
	}
	if eql(args[0], args[1]) {
		return cellT, nil
	}
	return MakeNil(), nil
}

func fnRandom(args []*Cell) (*Cell, error) {
	if len(args) == 0 {
		return MakeNum(float64(rand.Int())), nil
	}
	n := int(args[0].Num)
	if n <= 0 {
		return nil, fmt.Errorf("random: positive Zahl erwartet")
	}
	return MakeNum(float64(rand.Intn(n))), nil
}

// divArgs prüft (name a b) und liefert a, b; b = 0 ist ein Fehler.
func divArgs(name string, args []*Cell) (float64, float64, error) {
	if len(args) < 2 {
		return 0, 0, fmt.Errorf("%s: 2 Argumente nötig", name)
	}
	if err := checkNumbers(name, args); err != nil {
		return 0, 0, err
	}
	if args[1].Num == 0 {
		return 0, 0, fmt.Errorf("%s: Division durch Null", name)
	}
	return args[0].Num, args[1].Num, nil
}

// mod: CL — rundet ab, Ergebnis hat das Vorzeichen des Divisors:
// (mod -3 2) → 1.
func fnMod(args []*Cell) (*Cell, error) {
	a, b, err := divArgs("mod", args)
	if err != nil {
		return nil, err
	}
	r := math.Mod(a, b)
	if r != 0 && (r < 0) != (b < 0) {
		r += b
	}
	return MakeNum(r), nil
}

// rem/remainder: Richtung Null, Vorzeichen des Dividenden: (rem -3 2) → -1.
func fnRem(args []*Cell) (*Cell, error) {
	a, b, err := divArgs("rem", args)
	if err != nil {
		return nil, err
	}
	return MakeNum(math.Mod(a, b)), nil
}

func fnAbs(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("abs: 1 Argument nötig")
	}
	if err := checkNumbers("abs", args); err != nil {
		return nil, err
	}
	return MakeNum(math.Abs(args[0].Num)), nil
}

// floor: (floor x [y=1]) → zwei Werte (CL): ganzzahliger Quotient
// (Richtung -inf gerundet) und Rest mit Vorzeichen des Divisors.
func fnFloor(args []*Cell) (*Cell, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("floor: 1-2 Argumente nötig")
	}
	if err := checkNumbers("floor", args); err != nil {
		return nil, err
	}
	y := 1.0
	if len(args) == 2 {
		y = args[1].Num
	}
	if y == 0 {
		return nil, fmt.Errorf("floor: Division durch Null")
	}
	q := math.Floor(args[0].Num / y)
	r := args[0].Num - q*y
	return MakeValues([]*Cell{MakeNum(q), MakeNum(r)}), nil
}

// ceiling: (ceiling x [y=1]) → zwei Werte (CL): ganzzahliger Quotient
// (Richtung +inf gerundet) und Rest mit Vorzeichen des Divisors.
func fnCeiling(args []*Cell) (*Cell, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("ceiling: 1-2 Argumente nötig")
	}
	if err := checkNumbers("ceiling", args); err != nil {
		return nil, err
	}
	y := 1.0
	if len(args) == 2 {
		y = args[1].Num
	}
	if y == 0 {
		return nil, fmt.Errorf("ceiling: Division durch Null")
	}
	q := math.Ceil(args[0].Num / y)
	r := args[0].Num - q*y
	return MakeValues([]*Cell{MakeNum(q), MakeNum(r)}), nil
}

// truncate: (truncate x [y=1]) → zwei Werte (CL): ganzzahliger Quotient
// (Richtung Null gerundet) und Rest mit Vorzeichen des Dividenden.
func fnTruncate(args []*Cell) (*Cell, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("truncate: 1-2 Argumente nötig")
	}
	if err := checkNumbers("truncate", args); err != nil {
		return nil, err
	}
	y := 1.0
	if len(args) == 2 {
		y = args[1].Num
	}
	if y == 0 {
		return nil, fmt.Errorf("truncate: Division durch Null")
	}
	q := math.Trunc(args[0].Num / y)
	r := args[0].Num - q*y
	return MakeValues([]*Cell{MakeNum(q), MakeNum(r)}), nil
}

// round: (round x [y=1]) → zwei Werte (CL): auf-Ganzzahl-gerundeter
// Quotient (Rundung zum geraden Nachbarn bei .5, CL-Regel) und Rest.
func fnRound(args []*Cell) (*Cell, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("round: 1-2 Argumente nötig")
	}
	if err := checkNumbers("round", args); err != nil {
		return nil, err
	}
	y := 1.0
	if len(args) == 2 {
		y = args[1].Num
	}
	if y == 0 {
		return nil, fmt.Errorf("round: Division durch Null")
	}
	q := math.RoundToEven(args[0].Num / y)
	r := args[0].Num - q*y
	return MakeValues([]*Cell{MakeNum(q), MakeNum(r)}), nil
}

// cellEqual: struktureller Vergleich zweier Cells (rekursiv)
func cellEqual(a, b *Cell) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case NIL:
		return true
	case NUMBER:
		return a.Num == b.Num
	case ATOM:
		return a.Val == b.Val
	case STRING:
		return a.Val == b.Val
	case LIST, LAMBDA, MACRO:
		return cellEqual(a.Car, b.Car) && cellEqual(a.Cdr, b.Cdr)
	case VECTOR:
		return a == b // CL equal: Vektoren nur bei Identität
	}
	return false
}
