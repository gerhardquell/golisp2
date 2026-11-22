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
	_ = env.Set("floor", makeFn(roundFloor.fn))
	_ = env.Set("ceiling", makeFn(roundCeiling.fn))
	_ = env.Set("truncate", makeFn(roundTruncate.fn))
	_ = env.Set("round", makeFn(roundNearest.fn))
	_ = env.Set("random", makeFn(fnRandom))
	_ = env.Set("ash", makeFn(fnAsh))
	_ = env.Set("%ganzzahl-pruefen", makeFn(fnGanzzahlPruefen))

	// Vergleiche
	_ = env.Set("=", makeFn(fnEq))
	_ = env.Set("/=", makeFn(fnNumNe))
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

// ---- Überlaufgeprüfte Ganzzahl-Arithmetik ----
// EINZIGE Stelle dafür (Chokepoint). ok = false heißt Überlauf; der
// Aufrufer meldet ihn über errOverflow mit seinem eigenen Namen.

func addInt(a, b int64) (int64, bool) {
	c := a + b
	return c, (c > a) == (b > 0)
}

func subInt(a, b int64) (int64, bool) {
	c := a - b
	return c, (c < a) == (b > 0)
}

func mulInt(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	c := a * b
	if (a == -1 && b == math.MinInt64) || (b == -1 && a == math.MinInt64) {
		return 0, false
	}
	return c, c/b == a
}

// powInt: a hoch n für n ≥ 0 (Quadrieren und Multiplizieren).
func powInt(a, n int64) (int64, bool) {
	r, ok := int64(1), true
	for n > 0 {
		if n&1 == 1 {
			if r, ok = mulInt(r, a); !ok {
				return 0, false
			}
		}
		if n >>= 1; n > 0 {
			if a, ok = mulInt(a, a); !ok {
				return 0, false
			}
		}
	}
	return r, true
}

func errOverflow(name string) error {
	return fmt.Errorf("%s: Ganzzahl-Überlauf", name)
}

// floatToInt wandelt einen ganzzahligen Float-Wert in int64; NaN, ±Inf
// und Werte außerhalb int64 sind Fehler.
func floatToInt(name string, f float64) (int64, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("%s: %v ist nicht als Ganzzahl darstellbar", name, f)
	}
	if f < -9223372036854775808.0 || f >= 9223372036854775808.0 {
		return 0, errOverflow(name)
	}
	return int64(f), nil
}

// intArg ist die EINZIGE Quelle für Index- und Anzahl-Argumente (Spec
// §3.5): nur eine Ganzzahl gilt, ein Float ist ein Fehler — auch 2.0.
func intArg(name string, c *Cell) (int, error) {
	if c == nil || c.Type != NUMBER || c.NumKind != NumInt {
		hinweis := ""
		if c != nil && c.Type == NUMBER {
			hinweis = " (floor/round/truncate liefern Ganzzahlen)"
		}
		return 0, fmt.Errorf("%s: ganze Zahl erwartet, got %s%s", name, c, hinweis)
	}
	return int(c.Int), nil
}

// %ganzzahl-pruefen: (%ganzzahl-pruefen name n) → n; intArg für Lisp-Code
// (nth, iota, dotimes, evenp …).
func fnGanzzahlPruefen(args []*Cell) (*Cell, error) {
	if len(args) != 2 || args[0].Type != STRING {
		return nil, fmt.Errorf("%%ganzzahl-pruefen: (name n) erwartet")
	}
	if _, err := intArg(args[0].Val, args[1]); err != nil {
		return nil, err
	}
	return args[1], nil
}

// ash: arithmetischer Shift, exakt auf int64 (Rechtsshift rundet wie CL
// Richtung -unendlich).
func fnAsh(args []*Cell) (*Cell, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("ash: 2 Argumente nötig")
	}
	x, err := intArg("ash", args[0])
	if err != nil {
		return nil, err
	}
	n, err := intArg("ash", args[1])
	if err != nil {
		return nil, err
	}
	i := int64(x)
	switch {
	case n <= -63:
		return MakeInt(i >> 63), nil
	case n < 0:
		return MakeInt(i >> uint(-n)), nil
	case i == 0:
		return MakeInt(0), nil
	case n >= 64 || (i<<uint(n))>>uint(n) != i:
		return nil, errOverflow("ash")
	}
	return MakeInt(i << uint(n)), nil
}

// allInts: sind alle (bereits als NUMBER geprüften) Argumente Ganzzahlen?
func allInts(args []*Cell) bool {
	for _, a := range args {
		if a.NumKind != NumInt {
			return false
		}
	}
	return true
}

func fnAdd(args []*Cell) (*Cell, error) {
	if err := checkNumbers("+", args); err != nil {
		return nil, err
	}
	if allInts(args) {
		var i int64
		ok := true
		for _, a := range args {
			if i, ok = addInt(i, a.Int); !ok {
				return nil, errOverflow("+")
			}
		}
		return MakeInt(i), nil
	}
	n := 0.0
	for _, a := range args {
		n += a.Num
	}
	return MakeNum(n), nil
}

func fnSub(args []*Cell) (*Cell, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("-: min. 1 Argument")
	}
	if err := checkNumbers("-", args); err != nil {
		return nil, err
	}
	if allInts(args) {
		i, ok := args[0].Int, true
		if len(args) == 1 { // (- x) negiert (CL)
			if i, ok = subInt(0, i); !ok {
				return nil, errOverflow("-")
			}
			return MakeInt(i), nil
		}
		for _, a := range args[1:] {
			if i, ok = subInt(i, a.Int); !ok {
				return nil, errOverflow("-")
			}
		}
		return MakeInt(i), nil
	}
	if len(args) == 1 {
		return MakeNum(-args[0].Num), nil
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
	if allInts(args) {
		i, ok := int64(1), true
		for _, a := range args {
			if i, ok = mulInt(i, a.Int); !ok {
				return nil, errOverflow("*")
			}
		}
		return MakeInt(i), nil
	}
	n := 1.0
	for _, a := range args {
		n *= a.Num
	}
	return MakeNum(n), nil
}

// /: n-är, linksassoziativ. Solange Ganzzahlen aufgehen, bleibt es eine
// Ganzzahl; ab dem ersten Float oder Rest ist es ein Float.
func fnDiv(args []*Cell) (*Cell, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("/: min. 1 Argument")
	}
	if len(args) == 1 { // (/ x) ≡ (/ 1 x), CL
		args = []*Cell{MakeInt(1), args[0]}
	}
	if err := checkNumbers("/", args); err != nil {
		return nil, err
	}
	isInt, i, f := args[0].NumKind == NumInt, args[0].Int, args[0].Num
	for _, b := range args[1:] {
		if b.Num == 0 {
			return nil, fmt.Errorf("/: Division durch 0")
		}
		if isInt && b.NumKind == NumInt && i%b.Int == 0 {
			if i == math.MinInt64 && b.Int == -1 {
				return nil, errOverflow("/")
			}
			i /= b.Int
			f = float64(i)
			continue
		}
		isInt, f = false, f/b.Num
	}
	if isInt {
		return MakeInt(i), nil
	}
	return MakeNum(f), nil
}

// ---- Vergleiche (numerisch, strict) ----

// numEq/numLt: Ganzzahl gegen Ganzzahl exakt, sonst über Float (eine
// Ganzzahl über 2^53 ist dabei ungenau — bewusste Vereinfachung).
func numEq(a, b *Cell) bool {
	if a.NumKind == NumInt && b.NumKind == NumInt {
		return a.Int == b.Int
	}
	return a.Num == b.Num
}

func numLt(a, b *Cell) bool {
	if a.NumKind == NumInt && b.NumKind == NumInt {
		return a.Int < b.Int
	}
	return a.Num < b.Num
}

// numChain: CL-Vergleichskette – t, wenn cmp für jedes benachbarte Paar gilt.
func numChain(name string, args []*Cell, cmp func(a, b *Cell) bool) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("%s: 2 Argumente nötig", name)
	}
	if err := checkNumbers(name, args); err != nil {
		return nil, err
	}
	for i := 1; i < len(args); i++ {
		if !cmp(args[i-1], args[i]) {
			return MakeNil(), nil
		}
	}
	return cellT, nil
}

func fnEq(args []*Cell) (*Cell, error) {
	return numChain("=", args, numEq)
}

// fnNumNe: CL-/= – t, wenn alle Argumente paarweise verschieden sind.
func fnNumNe(args []*Cell) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("/=: 2 Argumente nötig")
	}
	if err := checkNumbers("/=", args); err != nil {
		return nil, err
	}
	for i := range args {
		for j := i + 1; j < len(args); j++ {
			if numEq(args[i], args[j]) {
				return MakeNil(), nil
			}
		}
	}
	return cellT, nil
}

func fnLt(args []*Cell) (*Cell, error) {
	return numChain("<", args, numLt)
}

func fnGt(args []*Cell) (*Cell, error) {
	return numChain(">", args, func(a, b *Cell) bool { return numLt(b, a) })
}

func fnGe(args []*Cell) (*Cell, error) {
	return numChain(">=", args, func(a, b *Cell) bool { return numLt(b, a) || numEq(a, b) })
}

func fnLe(args []*Cell) (*Cell, error) {
	return numChain("<=", args, func(a, b *Cell) bool { return numLt(a, b) || numEq(a, b) })
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

// numEql: gleiche Zahlart UND gleicher Wert — (eql 1 1.0) → ().
func numEql(a, b *Cell) bool {
	return a.NumKind == b.NumKind && numEq(a, b)
}

// eql: Zahlen nach Zahlart und Wert, sonst Pointer-Identität wie eq.
// Default-Test der Sequenzfunktionen.
func eql(a, b *Cell) bool {
	if a != nil && b != nil && a.Type == NUMBER && b.Type == NUMBER {
		return numEql(a, b)
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

// random: (random) → Float in [0, 1); (random n) → Ganzzahl in
// [0, n) für Ganzzahl n, Float in [0, n) für Float n.
func fnRandom(args []*Cell) (*Cell, error) {
	if len(args) == 0 {
		return MakeNum(rand.Float64()), nil
	}
	if err := checkNumbers("random", args[:1]); err != nil {
		return nil, err
	}
	n := args[0]
	if !(n.Num > 0) {
		return nil, fmt.Errorf("random: positive Zahl erwartet")
	}
	if n.NumKind == NumInt {
		return MakeInt(rand.Int63n(n.Int)), nil
	}
	return MakeNum(rand.Float64() * n.Num), nil
}

// divArgs prüft (name a b) und liefert a, b; b = 0 ist ein Fehler.
func divArgs(name string, args []*Cell) (*Cell, *Cell, error) {
	if len(args) < 2 {
		return nil, nil, fmt.Errorf("%s: 2 Argumente nötig", name)
	}
	if err := checkNumbers(name, args); err != nil {
		return nil, nil, err
	}
	if args[1].Num == 0 {
		return nil, nil, fmt.Errorf("%s: Division durch Null", name)
	}
	return args[0], args[1], nil
}

// mod: CL — rundet ab, Ergebnis hat das Vorzeichen des Divisors:
// (mod -3 2) → 1.
func fnMod(args []*Cell) (*Cell, error) {
	a, b, err := divArgs("mod", args)
	if err != nil {
		return nil, err
	}
	if a.NumKind == NumInt && b.NumKind == NumInt {
		r := a.Int % b.Int
		if r != 0 && (r < 0) != (b.Int < 0) {
			r += b.Int
		}
		return MakeInt(r), nil
	}
	r := math.Mod(a.Num, b.Num)
	if r != 0 && (r < 0) != (b.Num < 0) {
		r += b.Num
	}
	return MakeNum(r), nil
}

// rem/remainder: Richtung Null, Vorzeichen des Dividenden: (rem -3 2) → -1.
func fnRem(args []*Cell) (*Cell, error) {
	a, b, err := divArgs("rem", args)
	if err != nil {
		return nil, err
	}
	if a.NumKind == NumInt && b.NumKind == NumInt {
		return MakeInt(a.Int % b.Int), nil
	}
	return MakeNum(math.Mod(a.Num, b.Num)), nil
}

func fnAbs(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("abs: 1 Argument nötig")
	}
	if err := checkNumbers("abs", args); err != nil {
		return nil, err
	}
	a := args[0]
	if a.NumKind != NumInt {
		return MakeNum(math.Abs(a.Num)), nil
	}
	if a.Int >= 0 {
		return a, nil
	}
	i, ok := subInt(0, a.Int)
	if !ok {
		return nil, errOverflow("abs")
	}
	return MakeInt(i), nil
}

// ---- Rundung: floor / ceiling / truncate / round ----
// (name x [y=1]) → zwei Werte (CL): Quotient, immer Ganzzahl, und Rest
// (Ganzzahl, wenn x und y Ganzzahlen sind, sonst Float).
//   floor    Richtung -inf, Rest hat das Vorzeichen des Divisors
//   ceiling  Richtung +inf
//   truncate Richtung Null, Rest hat das Vorzeichen des Dividenden
//   round    zum nächsten, bei .5 zum geraden Nachbarn (CL-Regel)

type roundMode struct {
	name  string
	float func(float64) float64
	// adjust: Korrektur (-1, 0, +1) des Richtung Null gekürzten
	// Ganzzahl-Quotienten q; r = Rest ≠ 0, b = Divisor.
	adjust func(q, r, b int64) int64
}

func uabs(i int64) uint64 {
	if i < 0 {
		return -uint64(i)
	}
	return uint64(i)
}

var (
	roundFloor = roundMode{"floor", math.Floor, func(q, r, b int64) int64 {
		if (r < 0) != (b < 0) {
			return -1
		}
		return 0
	}}
	roundCeiling = roundMode{"ceiling", math.Ceil, func(q, r, b int64) int64 {
		if (r < 0) == (b < 0) {
			return 1
		}
		return 0
	}}
	roundTruncate = roundMode{"truncate", math.Trunc, func(q, r, b int64) int64 { return 0 }}
	roundNearest  = roundMode{"round", math.RoundToEven, func(q, r, b int64) int64 {
		ur, rest := uabs(r), uabs(b)-uabs(r)
		if ur < rest || (ur == rest && q%2 == 0) {
			return 0
		}
		if (r < 0) == (b < 0) {
			return 1
		}
		return -1
	}}
)

func (m roundMode) fn(args []*Cell) (*Cell, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("%s: 1-2 Argumente nötig", m.name)
	}
	if err := checkNumbers(m.name, args); err != nil {
		return nil, err
	}
	x, y := args[0], smallIntCache[1-smallIntMin]
	if len(args) == 2 {
		y = args[1]
	}
	if y.Num == 0 {
		return nil, fmt.Errorf("%s: Division durch Null", m.name)
	}
	if x.NumKind == NumInt && y.NumKind == NumInt {
		a, b := x.Int, y.Int
		if a == math.MinInt64 && b == -1 {
			return nil, errOverflow(m.name)
		}
		q, r := a/b, a%b
		if r != 0 {
			d := m.adjust(q, r, b)
			q, r = q+d, r-d*b
		}
		return MakeValues([]*Cell{MakeInt(q), MakeInt(r)}), nil
	}
	qf := m.float(x.Num / y.Num)
	q, err := floatToInt(m.name, qf)
	if err != nil {
		return nil, err
	}
	return MakeValues([]*Cell{MakeInt(q), MakeNum(x.Num - qf*y.Num)}), nil
}

// cellEqual: struktureller Vergleich zweier Cells (rekursiv)
func cellEqual(a, b *Cell) bool {
	if a == b { // CL: (equal x x) ist immer t — auch für Typen ohne eigenen case
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
		return numEql(a, b)
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
