//**********************************************************************
//  lib/eval_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260616
//**********************************************************************
// Eval-Grundlagen-Tests. Sicherheitsnetz für den eval.go-Split (Todo #1).
// Deckt: Primitive, Spezialformen, TCO-Trampolin, Closures, quasiquote.
// Makro-Expansion und parfunc/Channel haben eigene Testpunkte (Todo #3).
//**********************************************************************

package lib

import (
  "runtime"
  "strings"
  "testing"
)

// evalStr wertet ALLE Ausdrücke in src (eines oder mehrere) in einem
// frischen BaseEnv aus und liefert das letzte Ergebnis. Mehrere Ausdrücke
// sind nötig für defun + call. Read() allein lädt nur den ersten Ausdruck;
// deshalb Loop über den Reader (white-box: unexported Reader-Methoden OK).
func evalStr(src string) (*Cell, error) {
  env := BaseEnv()
  r := NewReader(strings.TrimSpace(src))
  result := MakeNil()
  for {
    r.skipWS()
    if _, ok := r.peek(); !ok {
      break
    }
    expr, err := r.readExpr()
    if err != nil {
      return nil, err
    }
    result, err = Eval(expr, env)
    if err != nil {
      return nil, err
    }
  }
  return result, nil
}

// evalEq wertet src aus und prüft die String-Repräsentation gegen want.
func evalEq(t *testing.T, src, want string) {
  t.Helper()
  got, err := evalStr(src)
  if err != nil {
    t.Fatalf("eval(%q) Fehler: %v", src, err)
  }
  if got.String() != want {
    t.Errorf("eval(%q) = %q, want %q", src, got.String(), want)
  }
}

// evalErr prüft, dass src einen Fehler liefert.
func evalErr(t *testing.T, src string) {
  t.Helper()
  _, err := evalStr(src)
  if err == nil {
    t.Errorf("eval(%q) sollte Fehler geben, lieferte nil", src)
  }
}

func TestEvalArith(t *testing.T) {
  cases := []struct{ src, want string }{
    {"(+ 1 2)", "3"},
    {"(- 10 3)", "7"},
    {"(* 6 7)", "42"},
    {"(/ 10 2)", "5"},
    {"(+ 1 2 3 4)", "10"},         // Variadisch
    {"(+ (* 2 3) (- 10 5))", "11"}, // Verschachtelt
    {"(- 5)", "5"},                 // IST: kein unäres Minus – fnSub(1 Arg)=args[0]
    {"(- 0 5)", "-5"},              // Negation explizit via 0
    {"(/ 1 3)", "0.3333333333333333"},
  }
  for _, c := range cases {
    evalEq(t, c.src, c.want)
  }
}

func TestEvalComparison(t *testing.T) {
  cases := []struct{ src, want string }{
    {"(= 1 1)", "t"},
    {"(= 1 2)", "()"},
    {"(< 1 2)", "t"},
    {"(> 1 2)", "()"},
    {"(>= 2 2)", "t"},
    {"(<= 1 2)", "t"},
  }
  for _, c := range cases {
    evalEq(t, c.src, c.want)
  }
}

func TestEvalBooleans(t *testing.T) {
  evalEq(t, "(and t t)", "t")
  evalEq(t, "(and t nil)", "()")
  evalEq(t, "(or nil 5)", "5")
  evalEq(t, "(or nil nil)", "()")
  evalEq(t, "(not nil)", "t")
  evalEq(t, "(not 5)", "()")
}

func TestEvalIf(t *testing.T) {
  evalEq(t, "(if t 1 2)", "1")
  evalEq(t, "(if nil 1 2)", "2")
  evalEq(t, "(if 0 1 2)", "1")   // 0 ist truthy (nur nil/() ist falsy)
  evalEq(t, "(if () 1 2)", "2")
  evalEq(t, "(if t 1)", "1")      // ohne else-Zweig
  evalEq(t, "(if nil 1)", "()")   // ohne else, falsy → Nil
  // Verschachteltes if im Tail – TCO-Pfad
  evalEq(t, "(if t (if t (if t 7)) 0)", "7")
}

// TestEvalTCO ist der wichtigste Test für den eval.go-Split:
// Tail-rekursive Funktion mit Akkumulator MUSS ohne Stack-Overflow
// durchlaufen. Das Trampolin im for-Loop ersetzt den Stack-Frame
// (expr/env setzen + continue). Ein falsches `return` statt `continue`
// beim Split macht diesen Test zum Segfault.
func TestEvalTCO(t *testing.T) {
  // Tiefe Tail-Rekursion, die ohne TCO den Go-Stack sprengt.
  // 200.000 ist deutlich über dem Default-Goroutine-Stack (8 KB → growbar,
  // aber klassische Rekursion würde bei ~50k-100k Frames crashen).
  src := `
    (defun count-down (n acc)
      (if (= n 0) acc (count-down (- n 1) (+ acc 1))))
    (count-down 200000 0)
  `
  got, err := evalStr(src)
  if err != nil {
    t.Fatalf("TCO count-down Fehler: %v", err)
  }
  if got.String() != "200000" {
    t.Errorf("TCO count-down = %q, want 200000", got.String())
  }
}

// TestEvalTCOBegin sichert, dass begin den letzten Ausdruck in Tail setzt.
func TestEvalTCOBegin(t *testing.T) {
  src := `
    (defun loop (n)
      (if (= n 0) 'done (begin (loop (- n 1)))))
    (loop 100000)
  `
  got, err := evalStr(src)
  if err != nil {
    t.Fatalf("TCO begin Fehler: %v", err)
  }
  if got.String() != "done" {
    t.Errorf("TCO begin = %q, want done", got.String())
  }
}

func TestEvalBegin(t *testing.T) {
  evalEq(t, "(begin 1 2 3)", "3")        // letzter Wert
  evalEq(t, "(begin)", "()")              // leer → Nil
  evalEq(t, "(begin 1 (* 2 3))", "6")
}

func TestEvalLetLetStar(t *testing.T) {
  // let: parallele Bindung – y sieht x NICHT
  evalErr(t, `(let ((x 5) (y (+ x 1))) y)`)
  // let: unabhängige Bindung funktioniert
  evalEq(t, `(let ((x 5) (y 10)) (+ x y))`, "15")
  // let*: sequentiell – y sieht x
  evalEq(t, `(let* ((x 5) (y (+ x 1))) y)`, "6")
}

func TestEvalCond(t *testing.T) {
  evalEq(t, `(cond (t 1))`, "1")
  evalEq(t, `(cond (nil 1) (t 2))`, "2")
  evalEq(t, `(cond ((= 1 2) 'a) ((= 1 1) 'b))`, "b")
  evalEq(t, `(cond (nil 1) (else 99))`, "99")
  evalEq(t, `(cond (nil 1))`, "()")   // kein Treffer → Nil
}

func TestEvalCase(t *testing.T) {
  evalEq(t, `(case 'b ((a) 1) ((b c) 2) (else 3))`, "2")
  evalEq(t, `(case 'x ((a) 1) ((b c) 2) (else 3))`, "3")
  evalEq(t, `(case 5 ((1 2 3) "klein") ((4 5 6) "mittel"))`, `"mittel"`)
}

func TestEvalLambda(t *testing.T) {
  evalEq(t, `((lambda (x) (* x x)) 5)`, "25")
  evalEq(t, `((lambda (x y) (+ x y)) 3 4)`, "7")
  evalEq(t, `((lambda () 42))`, "42")   // keine Parameter
  // apply/funcall mit Lambda
  evalEq(t, `(apply (lambda (x) (+ x 1)) '(2))`, "3")
  evalEq(t, `(funcall (lambda (x y) (+ x y)) 3 4)`, "7")
  // Datenliste als Funktion -> Fehler, kein Panic
  evalErr(t, `((list 'a 'b))`)
}

func TestEvalLambdaType(t *testing.T) {
  // Lambda ist ein Atom, aber keine Liste
  evalEq(t, `(atom? (lambda (x) x))`, "t")
  evalEq(t, `(list? (lambda (x) x))`, "()")
  // car/cdr auf Lambda liefern Parameter/Body (für SWANK-Introspektion)
  evalEq(t, `(car (lambda (x) x))`, "(x)")
  evalEq(t, `(car (lambda (x y) (+ x y)))`, "(x y)")
}

func TestEvalDefunRecursion(t *testing.T) {
  // Nicht-tail-rekursive Funktion (wächst auf dem Stack, aber klein)
  src := `
    (defun fact (n) (if (= n 0) 1 (* n (fact (- n 1)))))
    (fact 10)
  `
  evalEq(t, src, "3628800")
}

func TestEvalClosure(t *testing.T) {
  // Closure fängt y aus umgebendem let
  src := `
    (defun make-adder (n)
      (lambda (x) (+ x n)))
    (define add5 (make-adder 5))
    (add5 10)
  `
  evalEq(t, src, "15")
}

func TestDefineFunctionSyntax(t *testing.T) {
  // (define (name args...) body...) → Zucker für
  // (define name (lambda (args...) body...)), Semantik wie defun.
  evalEq(t, `(define (sq x) (* x x)) (sq 7)`, "49")
  evalEq(t, `(define (f) 42) (f)`, "42")
  evalEq(t, `(define (g x y) (+ x y)) (g 3 4)`, "7")
  // &rest wie bei defun
  evalEq(t, `(define (h x &rest xs) (cons x xs)) (h 1 2 3)`, "(1 2 3)")
  // mehrere Body-Formen: letzter Wert
  evalEq(t, `(define (m x) (define y (* x 2)) (+ x y)) (m 5)`, "15")
  // Rekursion
  evalEq(t, `(define (fact n) (if (= n 0) 1 (* n (fact (- n 1))))) (fact 5)`, "120")
  // Closure erzeugen
  evalEq(t, `(define (make-adder n) (lambda (x) (+ x n)))
             (define add2 (make-adder 2))
             (add2 40)`, "42")
  // Fehlerfälle: Name muss Symbol sein, Signatur nicht leer
  evalErr(t, `(define (1 x) x)`)
  evalErr(t, `(define () 42)`)
  // Wert-Syntax bleibt unverändert
  evalEq(t, `(define z 9) z`, "9")
}

func TestEvalQuote(t *testing.T) {
  evalEq(t, `(quote (a b c))`, "(a b c)")
  evalEq(t, `'x`, "x")
  evalEq(t, `'(1 2 3)`, "(1 2 3)")
  evalEq(t, `'(a . b)`, "(a . b)")
}

func TestEvalSetq(t *testing.T) {
  evalEq(t, `(begin (setq x 10) x)`, "10")
  evalEq(t, `(begin (setq* a 1 b (+ a 1) c (+ b 1)) (list a b c))`, "(1 2 3)")
}

func TestEvalQuasiquote(t *testing.T) {
  evalEq(t, "`(a b c)", "(a b c)")              // ohne unquote = reines quote
  evalEq(t, "`(a ,(+ 1 1) c)", "(a 2 c)")       // unquote
  evalEq(t, "`(a ,@(list 1 2) c)", "(a 1 2 c)") // unquote-splice
  evalEq(t, "`,(+ 1 2)", "3")                   // einzelnes unquote
}

func TestEvalEqEqual(t *testing.T) {
  // eq = Pointer-Identität; equal? = strukturell
  evalEq(t, `(eq (list) (list))`, "t")          // Singleton-Nil: identisch
  evalEq(t, `(eq 'foo 'foo)`, "t")              // interniert: eine Cell pro Symbol (CL)
  evalEq(t, `(equal? 'foo 'foo)`, "t")
  evalEq(t, `(equal? (list 1 2) (list 1 2))`, "t")
  evalEq(t, `(equal? 5 5)`, "t")
  // Zahlen: bewusste Abweichung. Der Small-Int-Cache liefert fuer 5 zwar
  // dieselbe Cell, aber fnEqPtr filtert NUMBER heraus — eine Optimierung
  // soll eq-Semantik nicht durch die Hintertuer aendern. CL laesst eq auf
  // Zahlen ausdruecklich unspezifiziert. Details: intern_test.go.
  evalEq(t, `(eq 5 5)`, "()")
}

func TestEvalErrors(t *testing.T) {
  evalErr(t, `undefined-symbol`)          // unbekanntes Symbol
  evalErr(t, `(/ 1 0)`)                   // Division durch Null
  evalErr(t, `(car 5)`)                    // car auf Nicht-Liste
  // (error ...) liefert LispError – als Fehler sichtbar
  _, err := evalStr(`(error "boom")`)
  if err == nil || !strings.Contains(err.Error(), "boom") {
    t.Errorf(`(error "boom") err=%v, will "boom" enthalten`, err)
  }
}

// TestEvalStrictArith sichert das strict-Typing in Arithmetik und
// numerischen Vergleichen: Nicht-Zahlen (Strings, Listen) werfen Fehler
// statt still als 0 einzufließen. (Früher latenter Bug: (+ 1 "x")=1;
// gefixt 2026-06-16 über checkNumbers in primitives.go.)
func TestEvalStrictArith(t *testing.T) {
  evalErr(t, `(+ 1 "x")`)          // String in Arithmetik → Fehler
  evalErr(t, `(- 5 "y")`)
  evalErr(t, `(* 2 (list 1))`)     // Liste in Arithmetik → Fehler
  evalErr(t, `(/ 1 "a")`)
  evalErr(t, `(mod 10 "b")`)
  evalErr(t, `(abs "c")`)
  evalErr(t, `(= 1 "x")`)          // numerischer Vergleich mit String → Fehler
  evalErr(t, `(< "a" 5)`)
  // Korrekte Nutzung bleibt funktionieren
  evalEq(t, `(+ 1 2)`, "3")
  evalEq(t, `(= 1 1)`, "t")
}

// TestEvalIfPermissive dokumentiert IST: degenerierte if-Formen werfen
// keinen Fehler, sondern liefern Nil.
func TestEvalIfPermissive(t *testing.T) {
  evalEq(t, `(if)`, "()")          // kein cond → Nil
  evalEq(t, `(if nil)`, "()")      // kein else → Nil
}

// TestEvalTrap sichert Fehlerbehandlung (projekteigen, kein CL).
// Syntax: (trap body handler). handler wird ausgewertet (→ Lambda) und
// mit der Fehler-Cell als Argument aufgerufen.
func TestEvalTrap(t *testing.T) {
  evalEq(t, `(trap (error "x") (lambda (e) 'fallback))`, "fallback")
  evalEq(t, `(trap 42 (lambda (e) 0))`, "42")   // kein Fehler → Wert durch
  // Kontrollfluss-Sentinels sind keine Fehler: throw geht durch trap durch
  evalEq(t, `(catch 'f (trap (throw 'f 7) (lambda (e) 'geschluckt)))`, "7")
}

// TestEvalCatchThrow sichert CL-Semantik: (catch tag body...) fängt
// (throw tag wert) dynamisch; Tag wird evaluiert.
func TestEvalCatchThrow(t *testing.T) {
  evalEq(t, `(catch 'f (throw 'f 42) 1)`, "42")
  evalEq(t, `(catch 'f 1 2 3)`, "3")
  evalEq(t, `(catch 'a (catch 'b (throw 'a 1)) 2)`, "1")
  evalEq(t, `(+ 1 (catch 'f (throw 'f 5)))`, "6")
  // kein passender Catch → Laufzeitfehler
  evalErr(t, `(throw 'nix 1)`)
  // Fehler sind kein throw: catch lässt LispError durch
  evalErr(t, `(catch 'f (error "boom"))`)
}

// TestEvalTagbody sichert CL-Semantik: Tags sind Sprungziele (werden
// nicht evaluiert), go setzt den PC, tagbody liefert nil. go springt
// auch aus tief geschachtelten Formen heraus.
func TestEvalTagbody(t *testing.T) {
  evalEq(t, `(let ((x 0)) (tagbody (setq x 1) (go ende) (setq x 99) ende) x)`, "1")
  evalEq(t, `(let ((x 0) (i 0)) (tagbody loop (setq x (+ x 1)) (setq i (+ i 1)) (if (< i 3) (go loop))) x)`, "3")
  evalEq(t, `(tagbody (go mitte) mitte)`, "()")
  // go aus verschachteltem block/catch heraus
  evalEq(t, `(let ((x 0)) (tagbody (block b (catch 'f (go raus))) raus (setq x 1)) x)`, "1")
  // tagbody gibt nil zurück
  evalEq(t, `(tagbody (setq y 1))`, "()")
  // go ohne passendes tagbody → Laufzeitfehler
  evalErr(t, `(go nirgendwo)`)
}

// TestEvalUnwindProtect sichert CL-Semantik: cleanup läuft bei jedem
// Ausstieg — Wert, Fehler, throw, go, return-from.
func TestEvalUnwindProtect(t *testing.T) {
  // normaler Wert: cleanup läuft, Wert bleibt
  evalEq(t, `(let ((x 0)) (list (unwind-protect 42 (setq x 1)) x))`, "(42 1)")
  // throw: cleanup läuft, throw geht weiter
  evalEq(t, `(let ((x 0)) (list (catch 'f (unwind-protect (throw 'f 'raus) (setq x 99))) x))`, "(raus 99)")
  // Fehler: cleanup läuft, Fehler geht weiter
  evalEq(t, `(let ((x 0)) (trap (unwind-protect (error "boom") (setq x 7)) (lambda (e) x)))`, "7")
  // go: cleanup läuft, Sprung geht weiter
  evalEq(t, `(let ((x 0)) (tagbody (unwind-protect (go raus) (setq x 5)) raus) x)`, "5")
  // return-from: cleanup läuft, Ausstieg geht weiter
  evalEq(t, `(let ((x 0)) (list (block b (unwind-protect (return-from b 1) (setq x 3))) x))`, "(1 3)")
}

// TestEvalNoLeakGoroutines ist eine Sanity-Prüfung, dass reine Eval-Pfade
// keine Goroutinen offen lassen (parfunc ist hier nicht beteiligt).
func TestEvalNoLeakGoroutines(t *testing.T) {
  before := runtime.NumGoroutine()
  for i := 0; i < 1000; i++ {
    _, _ = evalStr(`(+ 1 2 3)`)
  }
  after := runtime.NumGoroutine()
  if after > before+2 { // Toleranz für GC/Runtime-Goroutines
    t.Errorf("Goroutine-Leak: vor=%d nach=%d", before, after)
  }
}

// TestMacroexpandAll sichert rekursive Makro-Expansion ohne Evaluierung.
func TestMacroexpandAll(t *testing.T) {
  // Hilfe: definiert when-Makro, da evalStr nur BaseEnv (kein stdlib) lädt.
  run := func(body, want string) {
    t.Helper()
    src := "(begin (defmacro when (test . body) `(if ,test (begin ,@body) ())) " + body + ")"
    evalEq(t, src, want)
  }
  // Top-Level-Makro wird expandiert
  run("(macroexpand-all '(when t 1))", "(if t (begin 1) ())")
  // Rekursive Expansion in Subformen
  run("(macroexpand-all '(list (when t 1)))", "(list (if t (begin 1) ()))")
  // quote wird nicht durchdrungen (Drucker: Reader-Abkürzung)
  run("(macroexpand-all '(quote (when t 1)))", "'(when t 1)")
  // Atom bleibt unverändert
  run("(macroexpand-all 42)", "42")
  // Lambda-Body wird expandiert
  run("(macroexpand-all '(lambda (x) (when t x)))", "(lambda (x) (if t (begin x) ()))")
}

// TestEvalDoStar: do* bindet Init und Steps sequentiell (let*-Semantik),
// im Gegensatz zum parallelen do (let-Semantik).
func TestEvalDoStar(t *testing.T) {
  // Init sequentiell: y sieht das frisch gebundene x
  evalEq(t, `(do* ((x 1) (y (* x 2))) (t (list x y)))`, "(1 2)")
  // Step sequentiell: s bekommt das NEUE i und akkumuliert 1+2+3+4
  // (parallel-do würde s nur das alte i sehen → anderes Ergebnis)
  evalEq(t, `(do* ((i 0 (+ i 1)) (s i (+ s i))) ((= i 4) s))`, "10")
  // leere Bindungsliste
  evalEq(t, `(do* () (t 'sofort))`, "sofort")
  // kein Result-Form → nil
  evalEq(t, `(do* ((i 0 (+ i 1))) ((= i 2)))`, "()")
  // Bindung ohne Init → nil; ohne Step unverändert
  evalEq(t, `(do* ((a) (b 5)) (t (list a b)))`, "(() 5)")
  // Fehler: kaputte Bindung
  evalErr(t, `(do* (1) (t))`)
}

// TestEvalDeclarationen: declare ist No-Op (kein Typsystem), locally ist
// progn mit Deklarationen, the ignoriert den Typ und ist MV-transparent.
func TestEvalDeclarationen(t *testing.T) {
  evalEq(t, `(declare (ignore x))`, "()")
  evalEq(t, `(let ((x 1)) (declare (ignore x)) 2)`, "2")
  evalEq(t, `(locally 1 2 3)`, "3")
  evalEq(t, `(locally (declare) 42)`, "42")
  evalEq(t, `(the fixnum (+ 1 2))`, "3")
  // the reicht Multiple Values unverändert durch (CL)
  evalEq(t, `(multiple-value-list (the t (values 1 2)))`, "(1 2)")
  evalErr(t, `(the)`)
}

// TestEvalProgv: dynamische Bindung zur Laufzeit. Symbole/Werte werden
// ausgewertet; mehr Symbole als Werte → nil; MV des Body geht durch.
// Abweichung zu CL: keine lexikalisch/dynamisch-Trennung.
func TestEvalProgv(t *testing.T) {
  evalEq(t, `(progv '(dyn) '(99) dyn)`, "99")
  evalEq(t, `(progv '() '() 7)`, "7")
  // mehrere Bindungen, Rest-Symbole → nil
  evalEq(t, `(progv '(a b c) '(1 2) (list a b c))`, "(1 2 ())")
  // Symbolliste wird ausgewertet, nicht gequotet verarbeitet
  evalEq(t, `(let ((s 'x)) (progv (list s) '(5) x))`, "5")
  // nach progv ist die Bindung weg
  evalErr(t, `(begin (progv '(fluechtig) '(1) fluechtig) fluechtig)`)
  // Nicht-Symbol in der Liste → Fehler
  evalErr(t, `(progv '(a 5) '(1 2) a)`)
  evalErr(t, `(progv)`)
}

// TestEvalEvalWhen: golisp2 ist Eval-Kontext — Body feuert nur bei
// :execute (oder Altname eval), sonst nil. Wie clisp unter (eval ...).
func TestEvalEvalWhen(t *testing.T) {
  evalEq(t, `(eval-when (:execute) 5)`, "5")
  evalEq(t, `(eval-when (:execute :load-toplevel) 1 2 3)`, "3")
  evalEq(t, `(eval-when (eval) 5)`, "5")
  evalEq(t, `(eval-when (:compile-toplevel) 5)`, "()")
  evalEq(t, `(eval-when (:load-toplevel) 5)`, "()")
  evalEq(t, `(eval-when () 5)`, "()")
  evalErr(t, `(eval-when)`)
}

// TestLambdaSuppliedP: CL-Supplied-p-Parameter (name default supplied-p)
// für &optional und &key — supplied-p ist t, wenn das Argument geliefert
// wurde. Bug 20260723 (TODO.md my-reduce): init-p blieb ungebunden.
func TestLambdaSuppliedP(t *testing.T) {
  // &optional: nicht geliefert → supplied-p nil, Default greift
  evalEq(t, `((lambda (a &optional (b 9 b-p)) (if b-p b -1)) 5)`, "-1")
  // &optional: geliefert → supplied-p t, Wert greift
  evalEq(t, `((lambda (a &optional (b 9 b-p)) (if b-p b -1)) 5 3)`, "3")
  // explizit nil geliefert zählt als "geliefert"
  evalEq(t, `((lambda (&optional (x 9 x-p)) (if x-p 1 0)) ())`, "1")
  // &key: nicht geliefert
  evalEq(t, `((lambda (&key (k 1 k-p)) (if k-p k -1)))`, "-1")
  // &key: geliefert
  evalEq(t, `((lambda (&key (k 1 k-p)) (if k-p k -1)) :k 7)`, "7")
  // defmacro-Lambda-Liste (der my-reduce-Fall aus TODO.md)
  evalEq(t, "(begin (defmacro m (x &optional (y nil y-p)) `(if ,y-p 'geliefert 'default)) (m 1))", "default")
  evalEq(t, "(begin (defmacro m (x &optional (y nil y-p)) `(if ,y-p 'geliefert 'default)) (m 1 2))", "geliefert")
}
