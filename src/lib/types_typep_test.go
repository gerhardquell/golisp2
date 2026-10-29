//**********************************************************************
//  lib/types_typep_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Haiku 4.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260929
//**********************************************************************
// type-of / typep (src/embed/types.lisp), Struct-Registry (stdlib.lisp)
// und Kollisionswarnungen. Spec: docs/superpowers/specs/2026-09-29-typep-design.md
//**********************************************************************

package lib

import (
  "strings"
  "testing"
)

func stderrVon(t *testing.T, src string) string {
  t.Helper()
  return captureStderr(t, func() {
    if _, err := evalStdlib(t, src); err != nil {
      t.Fatalf("eval: %v", err)
    }
  })
}

func TestKollisionStructEingebauterTyp(t *testing.T) {
  out := stderrVon(t, `(defstruct integer a)`)
  if !strings.Contains(out, "WARN: defstruct integer: Name ist ein eingebauter Typ") {
    t.Errorf("Warnung fehlt, stderr = %q", out)
  }
  // eingebauter Typ gewinnt
  evalStdlibEq(t, `(defstruct integer a) (type-of (make-integer :a 1))`, "cons")
  evalStdlibEq(t, `(defstruct integer a) (typep (make-integer :a 1) 'integer)`, "()")
}

func TestKollisionStructCondition(t *testing.T) {
  out := stderrVon(t, `(define-condition mein-fehler (error) ()) (defstruct mein-fehler a)`)
  if !strings.Contains(out, "WARN: defstruct mein-fehler: Name ist auch ein Condition-Typ") {
    t.Errorf("Warnung fehlt, stderr = %q", out)
  }
  out = stderrVon(t, `(defstruct punkt x y) (define-condition punkt (error) ())`)
  if !strings.Contains(out, "WARN: define-condition punkt: Name ist auch ein Struct") {
    t.Errorf("Warnung fehlt, stderr = %q", out)
  }
  // Struct gewinnt
  evalStdlibEq(t, `(define-condition mein-fehler (error) ()) (defstruct mein-fehler a) (typep (make-mein-fehler :a 1) 'mein-fehler)`, "t")
}

func TestKollisionConditionEingebauterTyp(t *testing.T) {
  out := stderrVon(t, `(define-condition string (error) ())`)
  if !strings.Contains(out, "WARN: define-condition string: Name ist ein eingebauter Typ") {
    t.Errorf("Warnung fehlt, stderr = %q", out)
  }
}

func TestKeineWarnungOhneKollision(t *testing.T) {
  out := stderrVon(t, `(defstruct punkt x y) (define-condition datei-fehler (error) (pfad))`)
  if strings.Contains(out, "WARN") {
    t.Errorf("unerwartete Warnung: %q", out)
  }
}

func TestStructRegistry(t *testing.T) {
  evalStdlibEq(t, `(defstruct punkt x y) *struct-types*`, "((punkt 2))")
  evalStdlibEq(t, `(defstruct leer) *struct-types*`, "((leer 0))")
  // Reload ersetzt, dupliziert nicht
  evalStdlibEq(t, `(defstruct punkt x y) (defstruct punkt x y z) *struct-types*`, "((punkt 3))")
}

func TestTypeOfKern(t *testing.T) {
  cases := []struct{ src, want string }{
    {`(type-of 3)`, "integer"},
    {`(type-of 3.0)`, "integer"}, // Abweichung: Kern kennt nur float64
    {`(type-of -7)`, "integer"},
    {`(type-of 3.5)`, "float"},
    {`(type-of "a")`, "string"},
    {`(type-of 'foo)`, "symbol"},
    {`(type-of :k)`, "keyword"},
    {`(type-of t)`, "boolean"},
    {`(type-of '())`, "null"},
    {`(type-of '(1 2))`, "cons"},
    {`(type-of car)`, "compiled-function"},
    {`(type-of (lambda (x) x))`, "function"},
    {`(type-of (make-hash-table))`, "hash-table"},
    {`(defmacro to-mm (x) x) (type-of to-mm)`, "macro"},
  }
  for _, c := range cases {
    evalStdlibEq(t, c.src, c.want)
  }
}

func TestTypeOfStruct(t *testing.T) {
  evalStdlibEq(t, `(defstruct punkt x y) (type-of (make-punkt :x 1 :y 2))`, "punkt")
  evalStdlibEq(t, `(defstruct leer) (type-of (make-leer))`, "leer")
  // falsche Länge → keine Instanz
  evalStdlibEq(t, `(defstruct punkt x y) (type-of '(punkt 1))`, "cons")
  // ohne defstruct → nur eine Liste
  evalStdlibEq(t, `(type-of '(punkt 1 2))`, "cons")
}

func TestTypeOfCondition(t *testing.T) {
  evalStdlibEq(t, `
    (define-condition datei-fehler (error) (pfad))
    (handler-case (signal 'datei-fehler :pfad "x") (error (e) (type-of e)))`, "datei-fehler")
  evalStdlibEq(t, `(handler-case (car 5) (error (e) (type-of e)))`, "lisp-error")
}

func TestTypepHierarchie(t *testing.T) {
  ja := []string{
    `(typep 3 'integer)`, `(typep 3 'rational)`, `(typep 3 'real)`, `(typep 3 'number)`, `(typep 3 'atom)`, `(typep 3 t)`,
    `(typep 3.5 'float)`, `(typep 3.5 'real)`,
    `(typep "a" 'string)`, `(typep "a" 'sequence)`, `(typep "a" 'atom)`,
    `(typep 'foo 'symbol)`, `(typep :k 'keyword)`, `(typep :k 'symbol)`,
    `(typep t 'boolean)`, `(typep t 'symbol)`,
    `(typep '() 'null)`, `(typep '() 'boolean)`, `(typep '() 'symbol)`, `(typep '() 'list)`, `(typep '() 'sequence)`, `(typep '() 'atom)`,
    `(typep '(1) 'cons)`, `(typep '(1) 'list)`, `(typep '(1) 'sequence)`,
    `(typep car 'compiled-function)`, `(typep car 'function)`, `(typep (lambda (x) x) 'function)`,
    `(typep (make-hash-table) 'hash-table)`,
  }
  for _, src := range ja {
    evalStdlibEq(t, src, "t")
  }
  nein := []string{
    `(typep 3 'float)`, `(typep 3.5 'integer)`, `(typep "a" 'list)`, `(typep 'foo 'keyword)`,
    `(typep t 'null)`, `(typep '() 'cons)`, `(typep '(1) 'atom)`, `(typep (lambda (x) x) 'compiled-function)`,
    `(typep (make-hash-table) 'list)`, `(typep 3 nil)`,
  }
  for _, src := range nein {
    evalStdlibEq(t, src, "()")
  }
}

func TestTypepStruct(t *testing.T) {
  evalStdlibEq(t, `(defstruct punkt x y) (typep (make-punkt :x 1 :y 2) 'punkt)`, "t")
  evalStdlibEq(t, `(defstruct punkt x y) (typep (make-punkt :x 1 :y 2) 'structure-object)`, "t")
  evalStdlibEq(t, `(defstruct punkt x y) (typep (make-punkt :x 1 :y 2) 'cons)`, "t")
  evalStdlibEq(t, `(defstruct punkt x y) (typep '(punkt 1) 'punkt)`, "()")
  evalStdlibEq(t, `(typep '(1 2) 'structure-object)`, "()")
}

func TestTypepCondition(t *testing.T) {
  pre := `(define-condition datei-fehler (error) (pfad)) `
  for _, typ := range []string{"datei-fehler", "error", "condition", "cons"} {
    evalStdlibEq(t, pre+`(handler-case (signal 'datei-fehler :pfad "x") (error (e) (typep e '`+typ+`)))`, "t")
  }
  evalStdlibEq(t, pre+`(handler-case (signal 'datei-fehler :pfad "x") (error (e) (typep e 'lisp-error)))`, "()")
  evalStdlibEq(t, `(typep '(1) 'error)`, "()")
}

func TestTypepKombinatoren(t *testing.T) {
  evalStdlibEq(t, `(typep 'a '(or string symbol))`, "t")
  evalStdlibEq(t, `(typep 3 '(or string symbol))`, "()")
  evalStdlibEq(t, `(typep 3 '(and number (not float)))`, "t")
  evalStdlibEq(t, `(typep 3 '(or))`, "()")
  evalStdlibEq(t, `(typep 3 '(and))`, "t")
  evalStdlibEq(t, `(typep 3 '(not string))`, "t")
  evalStdlibEq(t, `(typep 'b '(member a b))`, "t")
  evalStdlibEq(t, `(typep 'c '(member a b))`, "()")
  evalStdlibEq(t, `(typep 2 '(eql 2))`, "t")
  evalStdlibEq(t, `(typep 3 '(satisfies number?))`, "t")
  evalStdlibEq(t, `(typep "a" '(satisfies number?))`, "()")
  evalStdlibEq(t, `(defun gerade? (n) (= 0 (mod n 2))) (typep 4 '(and integer (satisfies gerade?)))`, "t")
  evalStdlibEq(t, `(typep '((a 1)) '(or null (and cons (satisfies list?))))`, "t")
}

func TestTypepBereiche(t *testing.T) {
  cases := []struct{ src, want string }{
    {`(typep 5 '(integer 0 10))`, "t"},
    {`(typep 11 '(integer 0 10))`, "()"},
    {`(typep 0 '(integer (0) 10))`, "()"},
    {`(typep 10 '(integer 0 (10)))`, "()"},
    {`(typep 99 '(integer 0 *))`, "t"},
    {`(typep -1 '(integer 0))`, "()"},
    {`(typep 5 '(integer))`, "t"},
    {`(typep 5.5 '(integer 0 10))`, "()"},
    {`(typep 5.5 '(real 0 10))`, "t"},
    {`(typep 5.5 '(float 5 6))`, "t"},
    {`(typep "a" '(integer 0 10))`, "()"},
  }
  for _, c := range cases {
    evalStdlibEq(t, c.src, c.want)
  }
}

// --- Fixe aus dem Final-Review (Finding 1-6, 8) ---

func TestStructInstanceImproperList(t *testing.T) {
  // Finding 1: %struct-instance? crashte via length auf improper list.
  evalStdlibEq(t, `(defstruct punkt x y) (type-of (cons 'punkt 2))`, "cons")
  evalStdlibEq(t, `(defstruct punkt x y) (typep (cons 'punkt 2) 'cons)`, "t")
}

func TestTypepSatisfiesRootEnv(t *testing.T) {
  // Finding 2: (bound? f) sah lokale Variablen der Helper-Funktion statt
  // global aufzulösen — Namenskollision mit x, spec, args, f.
  for _, name := range []string{"f", "x", "spec", "args"} {
    evalStdlibEq(t,
      `(trap (typep 1 '(satisfies `+name+`)) (lambda (e) e))`,
      `"typep: satisfies: '`+name+`' ist keine Funktion"`)
  }
}

func TestNumberTypeInfinite(t *testing.T) {
  // Finding 3: +Inf/-Inf erfüllten (= x (floor x)) und galten als integer.
  evalStdlibEq(t, `(type-of (* 1e308 10))`, "float")
  evalStdlibEq(t, `(type-of (- (* 1e308 10) (* 1e308 10) 0))`, "float") // -Inf
  evalStdlibEq(t, `(type-of (- (* 1e308 10) (* 1e308 10)))`, "float")   // NaN, blieb schon float
}

func TestSymbolTypeEmptyName(t *testing.T) {
  // Finding 4: (intern "") crashte in substring statt symbol zu liefern.
  evalStdlibEq(t, `(type-of (intern ""))`, "symbol")
}

func TestConsTypeMalformedCondition(t *testing.T) {
  // Finding 5: malformte %condition-Marker crashten oder lieferten ().
  evalStdlibEq(t, `(type-of '(%condition))`, "cons")
  evalStdlibEq(t, `(type-of (cons '%condition 2))`, "cons")
}

func TestTypepSpecDottedArgs(t *testing.T) {
  // Finding 6: dotted Typangaben crashten mit rohem car/cdr-Fehler statt
  // "typep: ungültige Typangabe …".
  for _, src := range []string{
    `(typep 1 '(or . x))`,
    `(typep 1 '(member . 1))`,
    `(typep 1 '(integer 0 . 5))`,
  } {
    evalStdlibErr(t, src)
  }
  evalStdlibEq(t, `(trap (typep 1 '(or . x)) (lambda (e) e))`, `"typep: ungültige Typangabe (or . x)"`)
  evalStdlibEq(t, `(trap (typep 1 '(integer 0 . 5)) (lambda (e) e))`, `"typep: ungültige Typangabe (integer 0 . 5)"`)
}

func TestTypepFehlendeSupertypZeilen(t *testing.T) {
  // Finding 8: fehlende Supertyp-Zeilen aus dem Review — können schon
  // grün sein, dann sind sie hier nur als Regression festgehalten.
  evalStdlibEq(t, `(defmacro to-mm (x) x) (typep to-mm 'macro)`, "t")
  evalStdlibEq(t, `(defstruct punkt x y) (typep (make-punkt :x 1 :y 2) 'list)`, "t")
  evalStdlibEq(t, `(defstruct punkt x y) (typep (make-punkt :x 1 :y 2) 'sequence)`, "t")
  evalStdlibEq(t, `
    (define-condition datei-fehler (error) (pfad))
    (handler-case (signal 'datei-fehler :pfad "x") (error (e) (typep e 'list)))`, "t")
}

func TestTypepFehler(t *testing.T) {
  for _, src := range []string{
    `(typep 1 'gibts-nicht)`,
    `(typep 1 '(not a b))`,
    `(typep 1 '(eql))`,
    `(typep "a" '(integer "x"))`,
    `(typep 1 '(integer 0 1 2))`,
    `(typep 1 '(foo 1))`,
    `(typep 1 '(satisfies gibts-nicht))`,
    `(typep 1 '(satisfies))`,
    `(typep 1 "string")`,
  } {
    evalStdlibErr(t, src)
  }
  // Fehlertexte laut Spec
  evalStdlibEq(t, `(trap (typep 1 'foo) (lambda (e) e))`, `"typep: unbekannter Typ 'foo'"`)
  evalStdlibEq(t, `(trap (typep 1 '(satisfies foo)) (lambda (e) e))`, `"typep: satisfies: 'foo' ist keine Funktion"`)
}

func TestTypecase(t *testing.T) {
  klass := `(defun klass (x) (typecase x (integer 'ganz) ((or string symbol) 'text) (t 'sonst)))`
  evalStdlibEq(t, klass+` (klass 3)`, "ganz")
  evalStdlibEq(t, klass+` (klass "a")`, "text")
  evalStdlibEq(t, klass+` (klass 'a)`, "text")
  evalStdlibEq(t, klass+` (klass 1.5)`, "sonst")
  evalStdlibEq(t, `(typecase 1.5 (integer 'ganz))`, "()")              // kein Treffer → nil
  evalStdlibEq(t, `(typecase 1 (integer))`, "()")                       // leerer Rumpf → nil
  evalStdlibEq(t, `(typecase 1 (string 'a) (otherwise 'b))`, "b")
  evalStdlibEq(t, `(typecase 1 (number 'a 'b))`, "b")                   // mehrere Formen
  evalStdlibEq(t, `(let ((n 0)) (typecase (setq n (+ n 1)) (string 'a) (integer n)))`, "1") // key einmal
}

func TestEtypecase(t *testing.T) {
  evalStdlibEq(t, `(etypecase 1 (string 'a) (integer 'b))`, "b")
  evalStdlibErr(t, `(etypecase 1.5 (string 'a) (integer 'b))`)
  evalStdlibEq(t, `(trap (etypecase 1.5 (string 'a) (integer 'b)) (lambda (e) e))`,
    `"etypecase: 1.5 passt zu keinem Typ (string integer)"`)
}

func TestCheckType(t *testing.T) {
  evalStdlibEq(t, `(let ((x 3)) (check-type x integer))`, "()")
  evalStdlibEq(t, `(let ((x 3)) (check-type x (integer 0 10)) x)`, "3")
  evalStdlibErr(t, `(let ((x 1.5)) (check-type x integer))`)
  evalStdlibEq(t, `(let ((x 1.5)) (trap (check-type x integer) (lambda (e) e)))`,
    `"check-type: x ist 1.5, nicht vom Typ integer"`)
  evalStdlibEq(t, `(let ((x -1)) (trap (check-type x (integer 0 *) "eine natürliche Zahl") (lambda (e) e)))`,
    `"check-type: x ist -1, nicht eine natürliche Zahl"`)
  evalStdlibEq(t, `(let ((x "a")) (trap (check-type x integer) (lambda (e) e)))`,
    `"check-type: x ist \"a\", nicht vom Typ integer"`)
  evalStdlibEq(t, `(let ((n 0)) (check-type (setq n (+ n 1)) integer) n)`, "1") // place einmal
  evalStdlibEq(t, `(handler-case (let ((x 1.5)) (check-type x integer)) (error (e) :gefangen))`, ":gefangen")
}

func TestDeftype(t *testing.T) {
  evalStdlibEq(t, `(deftype ziffer () '(integer 0 9))`, "ziffer")
  evalStdlibEq(t, `(deftype ziffer () '(integer 0 9)) (list (typep 5 'ziffer) (typep 10 'ziffer))`, "(t ())")
  // Parameter, &optional mit CL-Default *
  bis := `(deftype bis (&optional (n '*)) (list 'integer 0 n)) `
  evalStdlibEq(t, bis+`(list (typep 5 '(bis 5)) (typep 6 '(bis 5)) (typep 1000 'bis))`, "(t () t)")
  // geschachtelt in Kombinatoren, typecase, check-type
  evalStdlibEq(t, `(deftype ziffer () '(integer 0 9)) (typep 3 '(or string ziffer))`, "t")
  evalStdlibEq(t, `(deftype ziffer () '(integer 0 9)) (typecase 3 (ziffer 'z) (t 'n))`, "z")
  evalStdlibEq(t, `(deftype ziffer () '(integer 0 9)) (trap (let ((x 12)) (check-type x ziffer)) (lambda (e) e))`,
    `"check-type: x ist 12, nicht vom Typ ziffer"`)
  // deftype auf deftype, Neudefinition gewinnt
  evalStdlibEq(t, `(deftype ziffer () '(integer 0 9)) (deftype kleine-ziffer () '(and ziffer (integer * 4))) (typep 7 'kleine-ziffer)`, "()")
  evalStdlibEq(t, `(deftype ziffer () '(integer 0 9)) (deftype ziffer () 'string) (typep "a" 'ziffer)`, "t")
}

func TestDeftypeKollision(t *testing.T) {
  // eingebauter Typ, Struct, Condition: deftype wäre tot → laut
  evalStdlibEq(t, `(trap (deftype integer () 'string) (lambda (e) e))`,
    `"deftype: integer ist ein eingebauter Typ"`)
  evalStdlibEq(t, `(defstruct punkt x) (trap (deftype punkt () 'string) (lambda (e) e))`,
    `"deftype: punkt ist ein Struct"`)
  evalStdlibEq(t, `(define-condition mein-f (error) ()) (trap (deftype mein-f () 'string) (lambda (e) e))`,
    `"deftype: mein-f ist ein Condition-Typ"`)
  evalStdlibErr(t, `(deftype "x" () 'string)`)
  // umgekehrt: defstruct über deftype-Namen → Warnung, Struct gewinnt
  out := stderrVon(t, `(deftype ding () 'string) (defstruct ding a)`)
  if !strings.Contains(out, "WARN: defstruct ding: Name ist auch ein deftype") {
    t.Errorf("Warnung fehlt, stderr = %q", out)
  }
}

func TestSubtypep(t *testing.T) {
  st := func(a, b, want string) {
    t.Helper()
    evalStdlibEq(t, `(multiple-value-list (subtypep '`+a+` '`+b+`))`, want)
  }
  // eingebaute Hierarchie
  st("integer", "number", "(t t)")
  st("number", "integer", "(() t)")
  st("null", "list", "(t t)")
  st("string", "sequence", "(t t)")
  st("string", "cons", "(() t)")
  st("integer", "t", "(t t)")
  st("t", "integer", "(() t)")
  st("integer", "integer", "(t t)")
  st("()", "string", "(t t)") // nil ist der leere Typ
  // atom = (not cons)
  st("integer", "atom", "(t t)")
  st("null", "atom", "(t t)")
  st("list", "atom", "(() t)")
  st("atom", "number", "(() t)")
  // Kombinatoren
  st("(or integer float)", "number", "(t t)")
  st("(or integer string)", "number", "(() t)")
  st("integer", "(or string number)", "(t t)")
  st("integer", "(and number atom)", "(t t)")
  st("(and integer string)", "integer", "(t t)")
  st("(member 1 2 3)", "integer", "(t t)")
  st("(member 1 \"a\")", "integer", "(() t)")
  st("(eql 5)", "(integer 0 9)", "(t t)")
  // Bereiche
  st("(integer 0 9)", "integer", "(t t)")
  st("(integer 0 9)", "number", "(t t)")
  st("(integer 0 9)", "(integer 0 *)", "(t t)")
  st("(integer 2 5)", "(integer (1) (6))", "(t t)")
  st("integer", "(integer * *)", "(t t)")
  st("string", "(integer 0 9)", "(() t)")
  // unentscheidbar → (() ())
  st("integer", "(integer 0 9)", "(() ())")
  st("(satisfies evenp)", "integer", "(() ())")
  st("integer", "(not string)", "(() ())")
}

func TestSubtypepNutzertypen(t *testing.T) {
  st := func(pre, a, b, want string) {
    t.Helper()
    evalStdlibEq(t, pre+` (multiple-value-list (subtypep '`+a+` '`+b+`))`, want)
  }
  st(`(defstruct punkt x)`, "punkt", "structure-object", "(t t)")
  st(`(defstruct punkt x)`, "punkt", "cons", "(t t)")
  st(`(defstruct punkt x)`, "punkt", "number", "(() t)")
  st(`(defstruct punkt x)`, "punkt", "atom", "(() t)")
  st(`(define-condition mein-f (error) ())`, "mein-f", "error", "(t t)")
  st(`(define-condition mein-f (error) ())`, "error", "mein-f", "(() t)")
  st(`(define-condition mein-f (error) ())`, "mein-f", "string", "(() t)")
  st(`(deftype ziffer () '(integer 0 9))`, "ziffer", "integer", "(t t)")
  st(`(deftype ziffer () '(integer 0 9))`, "(eql 3)", "ziffer", "(t t)")
  evalStdlibErr(t, `(subtypep 'gibts-nicht 'integer)`)
}

// defclass-Objekte: type-of/typep/subtypep kennen den Klassennamen,
// eine Klasse ist Untertyp von hash-table (Repräsentation) und atom.
func TestTypenDefclass(t *testing.T) {
  def := `(defclass schiff (name)) (define s (make-schiff :name "B")) `
  evalStdlibEq(t, def+`(type-of s)`, "schiff")
  evalStdlibEq(t, def+`(list (typep s 'schiff) (typep s 'hash-table) (typep s 'atom) (typep s 'cons))`, "(t t t ())")
  evalStdlibEq(t, def+`(typep (make-hash-table) 'schiff)`, "()")
  evalStdlibEq(t, `(type-of (make-hash-table))`, "hash-table")
  // :class ohne registrierte Klasse bleibt hash-table
  evalStdlibEq(t, `(let ((h (make-hash-table))) (puthash :class h 'nix) (type-of h))`, "hash-table")
  evalStdlibEq(t, def+`(multiple-value-list (subtypep 'schiff 'hash-table))`, "(t t)")
  evalStdlibEq(t, def+`(multiple-value-list (subtypep 'schiff 'atom))`, "(t t)")
  evalStdlibEq(t, def+`(multiple-value-list (subtypep 'schiff 'cons))`, "(() t)")
  evalStdlibEq(t, def+`(typecase s (schiff 'ja) (t 'nein))`, "ja")
}

func TestKollisionDefclass(t *testing.T) {
  out := stderrVon(t, `(defclass vector (a))`)
  if !strings.Contains(out, "WARN: defclass vector: Name ist ein eingebauter Typ") {
    t.Errorf("Warnung fehlt, stderr = %q", out)
  }
  out = stderrVon(t, `(defstruct boot a) (defclass boot (a))`)
  if !strings.Contains(out, "WARN: defclass boot: Name ist auch ein Struct") {
    t.Errorf("Warnung fehlt, stderr = %q", out)
  }
  out = stderrVon(t, `(defclass kahn (a)) (defstruct kahn a)`)
  if !strings.Contains(out, "WARN: defstruct kahn: Name ist auch eine Klasse") {
    t.Errorf("Warnung fehlt, stderr = %q", out)
  }
  evalStdlibEq(t, `(defclass floss (a)) (trap (deftype floss () 'integer) (lambda (e) 'fehler))`, "fehler")
}
