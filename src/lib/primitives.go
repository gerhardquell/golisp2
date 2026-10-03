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
	"os"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

// BaseEnv erstellt die globale Umgebung mit allen eingebauten Funktionen
func BaseEnv() *Env {
	env := NewEnv(nil)

	// Arithmetik + Vergleiche (arith_prims.go)
	RegisterArith(env)
	_ = env.Set("values", makeFn(fnValues))

	// Listen-Primitiven (die klassischen 7!)
	_ = env.Set("car", makeFn(fnCar))
	_ = env.Set("cdr", makeFn(fnCdr))
	_ = env.Set("cons", makeFn(fnCons))
	_ = env.Set("atom", makeFn(fnAtom))
	_ = env.Set("null", makeFn(fnNull))
	_ = env.Set("list", makeFn(fnList))
	_ = env.Set("%make-struct", makeFn(fnMakeStruct))
	_ = env.Set("append", makeFn(fnAppend))

	// Typ-Prädikate
	_ = env.Set("string?", makeFn(fnStringP))
	_ = env.Set("number?", makeFn(fnNumberP))
	_ = env.Set("list?", makeFn(fnListP))
	_ = env.Set("symbol?", makeFn(fnSymbolP))
	// Aliase für Konsistenz
	_ = env.Set("atom?", makeFn(fnAtom))
	_ = env.Set("null?", makeFn(fnNull))
	_ = env.Set("eq?", makeFn(fnEqPtr))

	// Ausgabe
	_ = env.Set("print", makeFn(fnPrint))
	_ = env.Set("read", makeFn(fnRead))
	_ = env.Set("read-line", makeFn(fnReadLine))
	_ = env.Set("println", makeFn(fnPrintln))
	_ = env.Set("warn", makeFn(fnWarn))

	// Wahrheitswerte
	_ = env.Set("t", MakeAtom("t"))
	_ = env.Set("nil", MakeNil())

	// apply, funcall
	_ = env.Set("apply", makeFn(fnApply))
	_ = env.Set("funcall", makeFn(func(args []*Cell) (*Cell, error) {
		return fnFuncall(env, args)
	}))
	_ = env.Set("mapcar", makeFn(fnMapcar))
	_ = env.Set("exit", makeFn(fnExit))

	// gensym
	_ = env.Set("gensym", makeFn(fnGensym))

	// Symbol-Konstruktion (für Makro-Expansion, z. B. defstruct)
	_ = env.Set("intern", makeFn(fnIntern))
	_ = env.Set("symbol-name", makeFn(fnSymbolName))

	// error
	_ = env.Set("error", makeFn(fnError))

	// Memory-Profiling
	_ = env.Set("memstats", makeFn(fnMemstats))

	// Introspektion
	_ = env.Set("env-symbols", makeFn(func(args []*Cell) (*Cell, error) {
		names := env.Symbols()
		sort.Strings(names)
		cells := make([]*Cell, len(names))
		for i, n := range names {
			cells[i] = MakeStr(n)
		}
		return SliceToCell(cells), nil
	}))

	// Zeitfunktionen
	_ = env.Set("sleep", makeFn(fnSleep))
	RegisterTime(env)

	// sigoREST
	RegisterSigo(env)

	// Goroutinen
	RegisterGoroutines(env)

	// Shared Memory
	RegisterShm(env)

	// Datei-I/O
	RegisterFileIO(env)

	// Shell-Kommandos
	RegisterShellCmd(env)

	// String-Funktionen
	RegisterStringFuncs(env)

	// Hashtables (CL)
	RegisterHashtables(env)

	// Kern-Typ für type-of/typep (types.lisp)
	RegisterCellType(env)

	// FORMAT (Common-Lisp-style)
	RegisterFormat(env)

	// C-artige Formatierung (printf/sscanf, übersetzt auf FORMAT)
	RegisterCFormat(env)

	// Kommandozeile + Environment (argv/getenv/environ)
	RegisterSysinfo(env)

	// CL-Kompat: sort/sqrt (Lispbuch-Lückenanalyse, TODO 20260818 Gruppe B)
	RegisterCLCompatPrims(env)

	// PostgreSQL
	RegisterPostgres(env)

	// Maxima (CAS via externen Prozess)
	RegisterMaxima(env)

	// Web-Bridge (HTTP + WebSocket)
	RegisterHTTPFuncs(env)
	RegisterWSFuncs(env)
	RegisterWebservFuncs(env)

	// Genetischer Algorithmus
	RegisterGenAlg(env)

	// Redefine-Guard Policy
	_ = env.Set("redefine-policy", makeFn(fnRedefinePolicy))
	_ = env.Set("redef-log", makeFn(fnRedefLog))
	_ = env.Set("redef-log-clear", makeFn(fnRedefLogClear))
	_ = env.Set("defined-in", makeFn(fnDefinedIn))
	_ = env.Set("documentation", makeFn(fnDocumentation))

	// Live-Tracing einzelner Funktionen
	RegisterTrace(env)

	return env
}

func makeFn(f func([]*Cell) (*Cell, error)) *Cell {
	return &Cell{Type: FUNC, Fn: f}
}

// values: (values ...) → liefert alle Argumente als multiple values (CL).
// Ein Argument bleibt schlichter Wert (kein Wrapper) — wie in CL.
func fnValues(args []*Cell) (*Cell, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	return MakeValues(args), nil
}

// ---- Listen ----

// fnCar: CL-Semantik — (car '()) → nil statt Fehler (TODO.md Punkt 1,
// 20260813). NIL zusätzlich zu LIST/LAMBDA/MACRO akzeptiert.
func fnCar(args []*Cell) (*Cell, error) {
	if len(args) < 1 || (args[0].Type != LIST && args[0].Type != LAMBDA && args[0].Type != MACRO && args[0].Type != NIL) {
		return nil, fmt.Errorf("car: Liste erwartet")
	}
	if args[0].Type == NIL {
		return MakeNil(), nil
	}
	return args[0].Car, nil
}

// fnCdr: CL-Semantik — (cdr '()) → nil statt Fehler (TODO.md Punkt 1,
// 20260813). NIL zusätzlich zu LIST/LAMBDA/MACRO akzeptiert.
func fnCdr(args []*Cell) (*Cell, error) {
	if len(args) < 1 || (args[0].Type != LIST && args[0].Type != LAMBDA && args[0].Type != MACRO && args[0].Type != NIL) {
		return nil, fmt.Errorf("cdr: Liste erwartet")
	}
	if args[0].Type == NIL || args[0].Cdr == nil {
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
		return MakeNil(), nil
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
	return MakeNil(), nil
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

// %make-struct: (make-struct tag slot1 slot2 ...) → (tag slot1 slot2 ...)
// Internes Primitiv für defstruct. Verwendet statt list, damit ein Slot
// namens "list" das Primitiv nicht shadowed.
func fnMakeStruct(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("%%make-struct: min. 1 Argument (tag)")
	}
	tag := args[0]
	if tag == nil || tag.Type != ATOM {
		return nil, fmt.Errorf("%%make-struct: tag muss Symbol sein")
	}
	result := MakeNil()
	for i := len(args) - 1; i >= 1; i-- {
		if args[i] == nil {
			return nil, fmt.Errorf("%%make-struct: Argument %d ist nil", i+1)
		}
		result = Cons(args[i], result)
	}
	return Cons(tag, result), nil
}

// append: (append list item) → Liste mit item am Ende (single-item, wie helper).
func fnAppend(args []*Cell) (*Cell, error) {
	return Append(args...), nil
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

// warn: (warn arg1 arg2 ...) → schreibt Argumente nach stderr.
// Rückgabewert ist das letzte Argument (wie print/println).
func fnWarn(args []*Cell) (*Cell, error) {
	var b strings.Builder
	for _, a := range args {
		b.WriteString(a.String())
	}
	b.WriteString("\n")
	if err := WriteError(b.String()); err != nil {
		return nil, fmt.Errorf("warn: %w", err)
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

// funcall: (funcall fn arg1 arg2 ...) → fn auf Argumente anwenden.
// Ein Symbol wird wie in CL global aufgelöst (Root-Env), nicht lexikalisch.
func fnFuncall(env *Env, args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("funcall: mindestens 1 Argument nötig")
	}
	fn := args[0]
	if fn.Type == ATOM {
		resolved, err := env.Root().GetSym(fn)
		if err != nil {
			return nil, fmt.Errorf("funcall: '%s' ist keine Funktion", fn)
		}
		fn = resolved
	}
	return apply(fn, args[1:])
}

// exit: (exit [code]) → beendet den Prozess sofort mit Exit-Code (Default 0).
// Kein unwind-protect, kein Cleanup — hartes os.Exit. Vorsicht im
// SWANK-Daemon: killt den Server. Hauptnutzen: (exit (run-tests)) in
// Testdateien → CI-taugliche Exit-Codes.
func fnExit(args []*Cell) (*Cell, error) {
	code := 0
	if len(args) > 0 {
		if args[0] == nil || args[0].Type != NUMBER {
			return nil, fmt.Errorf("exit: Code muss Zahl sein")
		}
		code = int(args[0].Num)
	}
	os.Exit(code)
	return nil, nil // unreachable
}

// mapcar: (mapcar fn liste) → wendet fn auf jedes Element an.
// Primitiv (keine Spezialform): beide Args normal evaluiert, damit mapcar
// first-class bleibt — (funcall mapcar ...), (apply mapcar ...) etc.
func fnMapcar(args []*Cell) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("mapcar: 2 Argumente nötig")
	}
	fn := args[0]
	var results []*Cell
	for lst := args[1]; lst != nil && lst.Type == LIST; lst = lst.Cdr {
		res, err := apply(fn, []*Cell{lst.Car})
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return SliceToCell(results), nil
}

// read: (read "string") → parst String zu Cell
func fnRead(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("read: 1 Argument nötig")
	}
	return Read(args[0].Val)
}

// gensym: global-atomarer Zähler für eindeutige Symbole.
// CLHS: (gensym &optional x) — x String = Prefix, x Int = Suffix.
var gensymCounter int64

func fnGensym(args []*Cell) (*Cell, error) {
	if len(args) > 1 {
		return nil, fmt.Errorf("gensym: höchstens 1 Argument erwartet")
	}
	n := atomic.AddInt64(&gensymCounter, 1)
	prefix := "G"
	if len(args) == 1 {
		switch args[0].Type {
		case STRING:
			prefix = args[0].Val
		case NUMBER:
			if args[0].Num != float64(int64(args[0].Num)) {
				return nil, fmt.Errorf("gensym: Zahl muss ganzzahlig sein, nicht %v", args[0].Num)
			}
			return MakeAtom(fmt.Sprintf("G%d", int64(args[0].Num))), nil
		default:
			return nil, fmt.Errorf("gensym: Argument muss String oder Int sein, nicht %v", args[0].Type)
		}
	}
	return MakeAtom(fmt.Sprintf("%s__%d", prefix, n)), nil
}

// intern: (intern string) → Symbol mit diesem Namen.
// ATOM idempotent, STRING → Atom. Benötigt für Makro-Expansion (defstruct),
// die Symbolnamen wie make-op / op-action zur Expansionszeit konstruiert.
func fnIntern(args []*Cell) (*Cell, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("intern: 1 Argument nötig")
	}
	switch args[0].Type {
	case ATOM:
		return args[0], nil
	case STRING:
		return MakeAtom(args[0].Val), nil
	default:
		return nil, fmt.Errorf("intern: String oder Symbol erwartet, got %s", args[0])
	}
}

// symbol-name: (symbol-name sym) → Name als String.
func fnSymbolName(args []*Cell) (*Cell, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("symbol-name: 1 Argument nötig")
	}
	switch args[0].Type {
	case ATOM, STRING:
		return MakeStr(args[0].Val), nil
	default:
		return nil, fmt.Errorf("symbol-name: Symbol erwartet, got %s", args[0])
	}
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
	return MakeNil(), nil
}

func fnNumberP(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("number?: 1 Argument nötig")
	}
	if args[0].Type == NUMBER {
		return cellT, nil
	}
	return MakeNil(), nil
}

func fnListP(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("list?: 1 Argument nötig") 
	}

	if args[0].Type == NIL || args[0].Type == LIST {
		return cellT, nil
	}
	return MakeNil(), nil
}

func fnSymbolP(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("symbol?: 1 Argument nötig")
	}
	if args[0].Type == ATOM {
		return cellT, nil
	}

	return MakeNil(), nil
}

// redefine-policy: (redefine-policy) -> aktuelle Policy als Atom
//                  (redefine-policy 'allow|'warn|'error) -> setzt Policy
func fnRedefinePolicy(args []*Cell) (*Cell, error) {
	if len(args) == 0 {
		return MakeAtom(GetRedefinePolicy()), nil
	}
	if len(args) != 1 || args[0] == nil || args[0].Type != ATOM {
		return nil, fmt.Errorf("redefine-policy: Syntax: (redefine-policy ['allow|'warn|'error])")
	}
	if err := SetRedefinePolicy(args[0].Val); err != nil {
		return nil, err
	}
	return args[0], nil
}
