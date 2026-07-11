//**********************************************************************
//  lib/primitives.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260223
//**********************************************************************

package lib

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

// BaseEnv erstellt die globale Umgebung mit allen eingebauten Funktionen
func BaseEnv() *Env {
	env := NewEnv(nil)

	// Arithmetik
	env.Set("+", makeFn(fnAdd))
	env.Set("-", makeFn(fnSub))
	env.Set("*", makeFn(fnMul))
	env.Set("/", makeFn(fnDiv))
	env.Set("mod", makeFn(fnMod))
	env.Set("remainder", makeFn(fnMod))
	env.Set("abs", makeFn(fnAbs))
	env.Set("random", makeFn(fnRandom))

	// Vergleiche
	env.Set("=", makeFn(fnEq))
	env.Set("<", makeFn(fnLt))
	env.Set(">", makeFn(fnGt))
	env.Set(">=", makeFn(fnGe))
	env.Set("<=", makeFn(fnLe))
	env.Set("equal?", makeFn(fnEqual))
	env.Set("eq", makeFn(fnEqPtr))

	// Listen-Primitiven (die klassischen 7!)
	env.Set("car", makeFn(fnCar))
	env.Set("cdr", makeFn(fnCdr))
	env.Set("cons", makeFn(fnCons))
	env.Set("atom", makeFn(fnAtom))
	env.Set("null", makeFn(fnNull))
	env.Set("list", makeFn(fnList))
	env.Set("append", makeFn(fnAppend))

	// Typ-Prädikate
	env.Set("string?", makeFn(fnStringP))
	env.Set("number?", makeFn(fnNumberP))
	env.Set("list?", makeFn(fnListP))
	env.Set("symbol?", makeFn(fnSymbolP))
	// Aliase für Konsistenz
	env.Set("atom?", makeFn(fnAtom))
	env.Set("null?", makeFn(fnNull))
	env.Set("eq?", makeFn(fnEqPtr))

	// Ausgabe
	env.Set("print", makeFn(fnPrint))
	env.Set("read", makeFn(fnRead))
	env.Set("println", makeFn(fnPrintln))

	// Wahrheitswerte
	env.Set("t", MakeAtom("t"))
	env.Set("nil", MakeNil())

	// apply, funcall
	env.Set("apply", makeFn(fnApply))
	env.Set("funcall", makeFn(fnFuncall))

	// gensym
	env.Set("gensym", makeFn(fnGensym))

	// error
	env.Set("error", makeFn(fnError))

	// Memory-Profiling
	env.Set("memstats", makeFn(fnMemstats))

	// Zeitfunktionen
	env.Set("sleep", makeFn(fnSleep))

	// sigoREST
	RegisterSigo(env)

	// Goroutinen
	RegisterGoroutines(env)

	// Datei-I/O
	RegisterFileIO(env)

	// Shell-Kommandos
	RegisterShellCmd(env)

	// String-Funktionen
	RegisterStringFuncs(env)

	// FORMAT (Common-Lisp-style)
	RegisterFormat(env)

	// PostgreSQL
	RegisterPostgres(env)

	// Genetischer Algorithmus
	RegisterGenAlg(env)

	return env
}

func makeFn(f func([]*Cell) (*Cell, error)) *Cell {
	return &Cell{Type: FUNC, Fn: f}
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
	return cellNil, nil
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
	return cellNil, nil
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
	return cellNil, nil
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
	return cellNil, nil
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
	return cellNil, nil
}

func fnEqual(args []*Cell) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("equal?: 2 Argumente nötig")
	}
	if cellEqual(args[0], args[1]) {
		return cellT, nil
	}
	return cellNil, nil
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
			return cellNil, nil
		}
		return cellT, nil
	}
	return cellNil, nil
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

func fnMod(args []*Cell) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("mod: 2 Argumente nötig")
	}
	if err := checkNumbers("mod", args); err != nil {
		return nil, err
	}
	b := args[1].Num
	if b == 0 {
		return nil, fmt.Errorf("mod: Division durch Null")
	}
	return MakeNum(math.Mod(args[0].Num, b)), nil
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
	case LIST:
		return cellEqual(a.Car, b.Car) && cellEqual(a.Cdr, b.Cdr)
	}
	return false
}

// ---- Listen ----

func fnCar(args []*Cell) (*Cell, error) {
	if len(args) < 1 || args[0].Type != LIST {
		return nil, fmt.Errorf("car: Liste erwartet")
	}
	return args[0].Car, nil
}

func fnCdr(args []*Cell) (*Cell, error) {
	if len(args) < 1 || args[0].Type != LIST {
		return nil, fmt.Errorf("cdr: Liste erwartet")
	}
	if args[0].Cdr == nil {
		return MakeNil(), nil
	}
	return args[0].Cdr, nil
}

func fnCons(args []*Cell) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("cons: 2 Argumente nötig")
	}
	return Cons(args[0], args[1]), nil
}

func fnAtom(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("atom: 1 Argument nötig")
	}
	if args[0].Type == LIST {
		return cellNil, nil
	}
	return cellT, nil
}

func fnNull(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("null: 1 Argument nötig")
	}
	if args[0].Type == NIL {
		return cellT, nil
	}
	return cellNil, nil
}

func fnList(args []*Cell) (*Cell, error) {
	result := MakeNil()
	for i := len(args) - 1; i >= 0; i-- {
		if args[i] == nil {
			return nil, fmt.Errorf("list: Argument %d ist nil", i+1)
		}
		result = Cons(args[i], result)
	}
	return result, nil
}

// append: (append list item) → Liste mit item am Ende (single-item, wie helper).
func fnAppend(args []*Cell) (*Cell, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("append: 2 Argumente nötig (list item)")
	}
	return Append(args[0], args[1]), nil
}

// ---- Ausgabe ----

// fnPrint: (print arg1 arg2 ...) → gibt alle Argumente aus,
// ohne Zeilenumbruch. Rückgabewert ist das letzte Argument (Common-Lisp-
// Semantik), damit der REPL nicht `()` hinter die Ausgabe druckt.
func fnPrint(args []*Cell) (*Cell, error) {
	var b strings.Builder
	for _, a := range args {
		b.WriteString(a.String())
	}
	if err := WriteOutput(b.String()); err != nil {
		return nil, fmt.Errorf("print: %w", err)
	}
	if len(args) == 0 {
		return MakeNil(), nil
	}
	return args[len(args)-1], nil
}

// fnPrintln: wie print, aber mit abschließendem Zeilenumbruch pro Argument.
func fnPrintln(args []*Cell) (*Cell, error) {
	var b strings.Builder
	for _, a := range args {
		b.WriteString(a.String())
		b.WriteString("\n")
	}
	if err := WriteOutput(b.String()); err != nil {
		return nil, fmt.Errorf("println: %w", err)
	}
	if len(args) == 0 {
		return MakeNil(), nil
	}
	return args[len(args)-1], nil
}

// apply: (apply fn arg1 ... liste) → fn auf alle Argumente anwenden
// Letztes Argument muss LIST sein, wird gespliced
func fnApply(args []*Cell) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("apply: 2+ Argumente nötig")
	}
	fn := args[0]
	combined := make([]*Cell, 0, len(args))
	// feste Argumente zwischen fn und der letzten Liste
	for i := 1; i < len(args)-1; i++ {
		combined = append(combined, args[i])
	}
	// letzte Liste entfalten
	last := args[len(args)-1]
	for lst := last; lst != nil && lst.Type == LIST; lst = lst.Cdr {
		combined = append(combined, lst.Car)
	}
	return apply(fn, combined)
}

// funcall: (funcall fn arg1 arg2 ...) → fn auf Argumente anwenden
func fnFuncall(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("funcall: mindestens 1 Argument nötig")
	}
	return apply(args[0], args[1:])
}

// read: (read "string") → parst String zu Cell
func fnRead(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("read: 1 Argument nötig")
	}
	return Read(args[0].Val)
}

// gensym: global-atomarer Zähler für eindeutige Symbole
var gensymCounter int64

func fnGensym(args []*Cell) (*Cell, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("gensym: keine Argumente erwartet")
	}
	n := atomic.AddInt64(&gensymCounter, 1)
	return MakeAtom(fmt.Sprintf("G__%d", n)), nil
}

// error: (error msg) → signalisiert Lisp-Laufzeitfehler
func fnError(args []*Cell) (*Cell, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("error: 1 Argument nötig")
	}
	return nil, &LispError{Msg: args[0]}
}

// memstats: (memstats) → Go Runtime Memory-Statistiken
func fnMemstats(args []*Cell) (*Cell, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("memstats: keine Argumente erwartet")
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Erstelle Property-Liste: ((heapalloc . 123) (heapsys . 456) ...)
	stats := []struct {
		name string
		val  float64
	}{
		{"heapalloc", float64(m.HeapAlloc)},
		{"heapsys", float64(m.HeapSys)},
		{"heapobjects", float64(m.HeapObjects)},
		{"numgc", float64(m.NumGC)},
		{"pausetotalns", float64(m.PauseTotalNs)},
		{"totalalloc", float64(m.TotalAlloc)},
	}

	result := MakeNil()
	for i := len(stats) - 1; i >= 0; i-- {
		pair := Cons(MakeAtom(stats[i].name), MakeNum(stats[i].val))
		result = Cons(pair, result)
	}
	return result, nil
}

// sleep: (sleep ms) → pausiert für n Millisekunden
func fnSleep(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("sleep: 1 Argument nötig (Millisekunden)")
	}
	ms := int64(args[0].Num)
	if ms < 0 {
		return nil, fmt.Errorf("sleep: negative Zeit nicht erlaubt")
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
	return MakeNil(), nil
}

// ---- Typ-Prädikate ----

func fnStringP(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("string?: 1 Argument nötig")
	}
	if args[0].Type == STRING {
		return cellT, nil
	}
	return cellNil, nil
}

func fnNumberP(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("number?: 1 Argument nötig")
	}
	if args[0].Type == NUMBER {
		return cellT, nil
	}
	return cellNil, nil
}

func fnListP(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("list?: 1 Argument nötig") 
	}

	if args[0].Type == NIL || args[0].Type == LIST {
		return cellT, nil
	}
	return cellNil, nil
}

func fnSymbolP(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("symbol?: 1 Argument nötig") 
	}
	if args[0].Type == ATOM {
		return cellT, nil
	}

	return cellNil, nil
}
