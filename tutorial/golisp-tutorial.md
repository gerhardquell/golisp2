<!--
  golisp-tutorial.md
  Autor    : Gerhard Quell - gquell@skequell.de
  CoAutor  : claude sonnet 4.6
  Copyright: 2026 Gerhard Quell - SKEQuell
  Erstellt : 20260623
-->

# GoLisp – Tutorial

Dieses Dokument beschreibt die öffentlichen Funktionen, Spezialformen und Makros von GoLisp mit kurzen, lauffähigen Beispielen. Dateioperationen nutzen immer das Projekt-temp-Verzeichnis `./tmp`. Funktionen, die externe Dienste benötigen (sigo, PostgreSQL), enthalten beschreibende Beispiele.

Stand: 20260827 — inklusive Hash-Tables, defstruct, defgeneric/defmethod, Conditions, setf, defsystem, FORMAT, Shared Memory, Trace.

## Spezialformen

### quote

**Syntax:** `(quote ausdruck)  bzw.  'ausdruck`

Verhindert die Auswertung eines Ausdrucks und gibt ihn als Datenstruktur zurück.

```lisp
(quote (+ 1 2))
'(+ 1 2)
(car '(a b c))
```

### if

**Syntax:** `(if bedingung dann [sonst])`

Wertet dann aus, wenn die Bedingung wahr ist, sonst den optionalen else-Zweig.

```lisp
(if t 1 2)
(if nil 'ja 'nein)
(if (> 3 2) 'größer)
```

### define

**Syntax:** `(define symbol wert)  bzw.  (setq symbol wert)`

Bindet einen Wert an einen Namen im aktuellen Environment.

```lisp
(define x 10)
(setq y 20)
(+ x y)
```

### setq

**Syntax:** `(setq symbol wert)`

Alias für define – bindet einen Wert an einen Namen.

```lisp
(setq antwort 42)
```

### defun

**Syntax:** `(defun name (parameter ...) body ...)`

Definiert eine benannte Funktion. Mehrere Body-Ausdrücke werden implizit in begin gewrappt. Ein String-Literal direkt nach der Parameterliste ist ein Docstring (abfragbar via `documentation`), wenn danach noch eine Form folgt.

```lisp
(defun quadrat (x) (* x x))
(quadrat 7)
(defun add (a b) "addiert" (print a) (+ a b))
(documentation 'add 'function)
; => "addiert"
```

### lambda

**Syntax:** `(lambda (parameter ...) body ...)`

Erzeugt eine anonyme Funktion bzw. Closure.

```lisp
((lambda (x) (* x 2)) 5)
(define doppel (lambda (x) (* 2 x)))
(doppel 4)
```

### let

**Syntax:** `(let ((var wert) ...) body ...)`

Bindet Variablen parallel – alle Werte werden im äußeren Environment ausgewertet.

```lisp
(let ((a 1) (b 2)) (+ a b))
(let ((x 5) (y 10)) (list x y))
```

### let*

**Syntax:** `(let* ((var wert) ...) body ...)`

Bindet Variablen sequentiell – jede Bindung sieht die vorherigen.

```lisp
(let* ((x 1) (y (+ x 1))) y)
(let* ((a 2) (b (* a 3))) (+ a b))
```

### begin

**Syntax:** `(begin ausdruck ...)`

Wertet Ausdrücke nacheinander aus und gibt das Ergebnis des letzten zurück. `progn` und `locally` sind Aliase.

```lisp
(begin (print 1) (print 2) 3)
(begin (define x 1) (set! x 2) x)
(progn 1 2 3)
```

### set!

**Syntax:** `(set! symbol wert)`

Ändert den Wert einer bestehenden Variablen.

```lisp
(define a 1)
(set! a 99)
a
```

### setq*

**Syntax:** `(setq* var1 wert1 var2 wert2 ...)`

Setzt mehrere Variablen sequentiell; fehlende Variablen werden neu angelegt.

```lisp
(setq* a 1 b (+ a 1) c (+ b 1))
(list a b c)
```

### defmacro

**Syntax:** `(defmacro name (parameter ...) body ...)`

Definiert ein Makro, das zur Expandierungszeit transformiert wird.

```lisp
(defmacro unless (test . body) `(if ,test () (begin ,@body)))
(unless nil 'ok)
```

### macroexpand

**Syntax:** `(macroexpand form)`

Expandiert Makros einmal auf Top-Level und gibt das Ergebnis zurück.

```lisp
(defmacro doppelt (x) `(+ ,x ,x))
(macroexpand (doppelt 5))
```

### macroexpand-1

**Syntax:** `(macroexpand-1 form)`

Wie macroexpand – expandiert einen Expansionsschritt.

```lisp
(macroexpand-1 '(when t 1))
; => (if t (begin 1) ())
```

### macroexpand-all

**Syntax:** `(macroexpand-all form)`

Expandiert Makros rekursiv in allen Subformen.

```lisp
(macroexpand-all (doppelt (+ 1 2)))
```

### function

**Syntax:** `(function f)  bzw.  #'f`

Liefert die Funktion hinter dem Symbol als Wert — nötig, um Funktionen als Argumente zu übergeben. `#'` ist die Reader-Abkürzung.

```lisp
(funcall (function +) 1 2)
; => 3
(mapcar #'car '((1 2) (3 4)))
; => (1 3)
```

### flet

**Syntax:** `(flet ((name (parameter ...) body ...) ...) body ...)`

Definiert lokale Funktionen, die über das äußere Environment schließen.

```lisp
(flet ((inc (x) (+ x 1))) (inc 5))
```

### labels

**Syntax:** `(labels ((name (parameter ...) body ...) ...) body ...)`

Wie flet, aber die lokalen Funktionen können sich gegenseitig rekursiv aufrufen.

```lisp
(labels ((fac (n) (if (= n 0) 1 (* n (fac (- n 1)))))) (fac 5))
```

### block

**Syntax:** `(block name body ...)`

Benannter Block; return-from kann ihn vorzeitig verlassen.

```lisp
(block aus (return-from aus 42) (print 'nie))
```

### return-from

**Syntax:** `(return-from name [wert])`

Verlässt den benannten Block nicht-lokal und liefert den optionalen Wert.

```lisp
(block outer (return-from outer 99) 'unten)
```

### load

**Syntax:** `(load "datei.lisp")`

Lädt eine Lisp-Datei, wertet alle enthaltenen Ausdrücke aus und gibt das letzte Ergebnis zurück.

```lisp
(load "stdlib.lisp")
```

### and

**Syntax:** `(and ausdruck ...)`

Wertet Ausdrücke von links nach rechts aus und bricht bei nil ab. Liefert den letzten wahren Wert.

```lisp
(and 1 2 3)
(and t nil 'x)
(and)
```

### or

**Syntax:** `(or ausdruck ...)`

Wertet Ausdrücke von links nach rechts aus und liefert den ersten wahren Wert, sonst nil.

```lisp
(or nil 'a 'b)
(or nil nil)
(or 0 'x)
```

### not

**Syntax:** `(not x)`

Gibt t zurück, wenn x falsch (nil) ist, sonst nil.

```lisp
(not nil)
(not t)
(not 0)
```

### cond

**Syntax:** `(cond (test body ...) ... [(else body ...)])`

Verzweigungskette; der erste wahre Test bestimmt den ausgeführten Body.

```lisp
(cond ((= 1 2) 'nein) (t 'ja))
(cond ((> 5 3) 'größer) (else 'kleiner))
```

### case

**Syntax:** `(case schlüssel ((wert ...) body ...) ... [(else body ...)])`

Wählt einen Zweig durch strukturelle Gleichheit (equal?) des Schlüssels aus.

```lisp
(case 'b ((a) 1) ((b c) 2) (else 3))
(case 5 ((1 2 3) "klein") ((4 5 6) "mittel"))
```

### while

**Syntax:** `(while test body ...)`

Wertet den Body so lange aus, wie der Test wahr ist. Gibt nil zurück.

```lisp
(define i 0)
(while (< i 3) (print i) (set! i (+ i 1)))
```

### do

**Syntax:** `(do ((var init schritt) ...) (test ergebnis) body ...)`

Scheme-artige Schleife mit parallelen Bindungen, Abbruchtest und optionalem Body.

```lisp
(do ((i 0 (+ i 1))) ((= i 3) i) (print i))
(do ((x 1 (* x 2)) (n 0 (+ n 1))) ((> x 10) n))
```

### quasiquote

**Syntax:** `(quasiquote ausdruck)  bzw.  `ausdruck`

Wie quote, erlaubt aber unquote (,x) und unquote-splice (,@x) innerhalb.

```lisp
(define x 5)
`(liste ,x)
`(a ,@'(b c))
```

### eval

**Syntax:** `(eval ausdruck)`

Wertet einen bereits ausgewerteten Ausdruck nochmals im globalen Environment aus (Common-Lisp-Semantik — Definitionen aus `(eval (read ...))` bleiben global sichtbar).

```lisp
(eval '(+ 1 2))
(define code '(+ 3 4))
(eval code)
```

### catch

**Syntax:** `(catch body handler)`

Fängt Fehler im Body ab und ruft handler mit der Fehler-Cell auf.

```lisp
(catch (error "ups") (lambda (e) (list 'fehler e)))
(catch (/ 1 0) (lambda (e) 'recovered))
```

### parfunc

**Syntax:** `(parfunc ergebnis [:timeout n] ausdruck ...)`

Wertet alle Ausdrücke parallel aus, speichert die Ergebnisliste in ergebnis und gibt sie zurück.

```lisp
(parfunc ergebnis (+ 1 2) (* 3 4) (- 10 5))
(car ergebnis)
```

### lock

**Syntax:** `(lock mutex ausdruck ...)`

Führt die Ausdrücke atomar unter dem angegebenen Mutex aus.

```lisp
(define m (lock-make))
(define x 0)
(lock m (set! x (+ x 1)) (set! x (+ x 1)))
x
```

### exec

**Syntax:**

```lisp
(exec "programm"
      param: "arg1"
      param: "arg2"
      stdin:  eingabe
      stdout: ausgabe-var
      stderr: fehler-var
      exitcd: code-var)
```

Führt ein externes Programm direkt aus (ohne Shell). `param:` und `stdin:` werden ausgewertet, `stdout:`, `stderr:` und `exitcd:` sind Namen von Variablen, die im aktuellen Environment gesetzt werden.

- Rückgabe `t`, wenn das Programm gestartet und beendet wurde.
- Exit-Code ≠ 0 ist kein Fehler; er landet in `exitcd:`.
- Technischer Fehler (Programm nicht gefunden, Timeout nach 60 s) → Rückgabe `nil`, `exitcd:` wird `-1` (falls angegeben).
- Nicht angeforderte Streams werden verworfen.

```lisp
(exec "echo" param: "hallo" stdout: out exitcd: cd)
out
; => "hallo\n"
cd
; => 0

(exec "sh" param: "-c" param: "echo fehler >&2; exit 1"
      stdout: out stderr: err exitcd: cd)
err
; => "fehler\n"
cd
; => 1

(exec "cat" stdin: "Eingabe" stdout: out exitcd: cd)
out
; => "Eingabe"

(exec "/kein/solches/programm" stdout: out exitcd: cd)
; => nil
cd
; => -1
```

## Arithmetik

### +

**Syntax:** `(+ zahl ...)`

Addiert alle übergebenen Zahlen.

```lisp
(+ 1 2 3)
(+ 5.5 4.5)
```

### -

**Syntax:** `(- zahl ...)  bzw.  (- x y)`

Subtrahiert die folgenden Zahlen von der ersten. Ohne Argumente liefert es 0.

```lisp
(- 10 3)
(- 5)
```

### *

**Syntax:** `(* zahl ...)`

Multipliziert alle übergebenen Zahlen.

```lisp
(* 2 3 4)
(* 7)
```

### /

**Syntax:** `(/ x y)`

Dividiert x durch y. Fehler bei Division durch 0.

```lisp
(/ 10 2)
(/ 1 4)
```

### mod

**Syntax:** `(mod x y)`

Liefert den Rest der Division x durch y (Gleitkomma).

```lisp
(mod 10 3)
(mod 7.5 2.5)
```

### remainder

**Syntax:** `(remainder x y)`

Alias für mod.

```lisp
(remainder 10 3)
```

### abs

**Syntax:** `(abs x)`

Liefert den absoluten Betrag einer Zahl.

```lisp
(abs -5)
(abs -3.14)
```

### max

**Syntax:** `(max zahl ...)`

Liefert die größte der übergebenen Zahlen.

```lisp
(max 1 5 3)
; => 5
```

### min

**Syntax:** `(min zahl ...)`

Liefert die kleinste der übergebenen Zahlen.

```lisp
(min 4 2 9)
; => 2
```

### sqrt

**Syntax:** `(sqrt x)`

Quadratwurzel.

```lisp
(sqrt 16)
; => 4
```

### floor

**Syntax:** `(floor x)`

Rundet gegen die nächste ganze Zahl ab.

```lisp
(floor 3.7)
; => 3
```

### random

**Syntax:** `(random)  bzw.  (random n)`

Liefert eine Zufallszahl. Ohne Argument eine große ganze Zahl, mit n einen Wert von 0 bis n-1.

```lisp
(random 6)
(random 100)
```

## Vergleiche

### =

**Syntax:** `(= x y)`

Numerische Gleichheit.

```lisp
(= 5 5)
(= 1 2)
```

### <

**Syntax:** `(< x y)`

Numerisch kleiner.

```lisp
(< 2 5)
(< 5 2)
```

### >

**Syntax:** `(> x y)`

Numerisch größer.

```lisp
(> 5 2)
(> 2 5)
```

### >=

**Syntax:** `(>= x y)`

Numerisch größer oder gleich.

```lisp
(>= 5 5)
```

### <=

**Syntax:** `(<= x y)`

Numerisch kleiner oder gleich.

```lisp
(<= 3 7)
```

### equal?

**Syntax:** `(equal? x y)`

Strukturelle Gleichheit (rekursiv für Listen).

```lisp
(equal? '(1 2) '(1 2))
(equal? 'a 'a)
(equal? 5 5)
```

### eq

**Syntax:** `(eq x y)`

Pointer-Identität – prüft, ob beide Argumente dasselbe Objekt im Speicher sind. Für Symbole CL-korrekt: `(eq 'foo 'foo)` → `t` (Symbol-Interning). Zahlen und Strings sind nicht interniert: `(eq 5 5)` → `()`.

```lisp
(eq nil nil)
(eq (list) (list))
(eq 'foo 'foo)
```

### eq?

**Syntax:** `(eq? x y)`

Alias für eq.

```lisp
(eq? nil nil)
```

### eql

**Syntax:** `(eql a b)`

Wie eq, aber Zahlen mit gleichem Wert gelten als eql (CL-Semantik).

```lisp
(eql 1 1)
; => t
(eql 'a 'a)
; => t
```

## Typ-Prädikate

### string?

**Syntax:** `(string? x)`

Gibt t zurück, wenn x ein String ist.

```lisp
(string? "hallo")
(string? 42)
```

### number?

**Syntax:** `(number? x)`

Gibt t zurück, wenn x eine Zahl ist.

```lisp
(number? 42)
(number? "42")
```

### list?

**Syntax:** `(list? x)`

Gibt t zurück, wenn x eine Liste oder nil ist.

```lisp
(list? '(1 2))
(list? nil)
(list? 5)
```

### symbol?

**Syntax:** `(symbol? x)`

Gibt t zurück, wenn x ein Symbol (Atom) ist.

```lisp
(symbol? 'foo)
(symbol? "foo")
```

### atom?

**Syntax:** `(atom? x)`

Alias für atom – gibt t zurück, wenn x kein Cons/Liste ist.

```lisp
(atom? 5)
(atom? '(1 2))
```

### null?

**Syntax:** `(null? x)`

Alias für null – gibt t zurück, wenn x nil ist.

```lisp
(null? nil)
(null? '(1))
```

### atom

**Syntax:** `(atom x)`

Gibt t zurück, wenn x kein Cons/Liste ist.

```lisp
(atom 5)
(atom 'x)
(atom '(1 2))
```

### null

**Syntax:** `(null x)`

Gibt t zurück, wenn x nil ist.

```lisp
(null nil)
(null '())
(null '(1))
```

## Listen

### car

**Syntax:** `(car liste)`

Liefert das erste Element einer Liste.

```lisp
(car '(a b c))
(car '(1 2))
```

### cdr

**Syntax:** `(cdr liste)`

Liefert die Restliste ohne das erste Element.

```lisp
(cdr '(a b c))
(cdr '(1))
```

### cons

**Syntax:** `(cons x liste)`

Fügt x als neues Kopfelement vor eine Liste.

```lisp
(cons 1 '(2 3))
(cons 'a nil)
```

### list

**Syntax:** `(list element ...)`

Erzeugt eine neue Liste aus den Argumenten.

```lisp
(list 1 2 3)
(list 'a 'b)
```

### append

**Syntax:** `(append liste ...)`

Verkettet beliebig viele Listen zu einer neuen Liste. Ein Atom als letztes Argument erzeugt eine improper List (dotted pair).

```lisp
(append '(1 2) '(3 4))
; => (1 2 3 4)
(append '(1 2) 3)
; => (1 2 . 3)
```

### mapcar

**Syntax:** `(mapcar funktion liste ...)`

Wendet eine Funktion auf jedes Element an und liefert die Ergebnisliste. Primitiv (keine Spezialform) — Funktionen sind first-class, `#'`-Syntax funktioniert.

```lisp
(mapcar (lambda (x) (* x x)) '(1 2 3))
; => (1 4 9)
(mapcar #'car '((1 2) (3 4)))
; => (1 3)
```

### apply

**Syntax:** `(apply funktion arg1 ... liste)`

Wendet eine Funktion an; das letzte Argument wird als Liste gesplict.

```lisp
(apply + '(1 2 3))
(apply + 1 2 '(3 4))
```

### funcall

**Syntax:** `(funcall funktion arg ...)`

Ruft eine Funktion mit den angegebenen Argumenten auf.

```lisp
(funcall + 1 2 3)
(funcall (lambda (x) (* x x)) 6)
```

### sort

**Syntax:** `(sort liste vergleich)`

Liefert eine neue, nach vergleich sortierte Liste. Die Original-Liste bleibt unverändert.

```lisp
(sort '(3 1 2) <)
; => (1 2 3)
```

### remove

**Syntax:** `(remove element liste)`

Entfernt alle Vorkommen von element (equal?-Vergleich).

```lisp
(remove 2 '(1 2 3 2))
; => (1 3)
```

### remove-if

**Syntax:** `(remove-if prädikat liste)`

Entfernt alle Elemente, für die prädikat wahr ist.

```lisp
(remove-if (lambda (x) (= 1 (mod x 2))) '(1 2 3 4))
; => (2 4)
```

### remove-if-not

**Syntax:** `(remove-if-not prädikat liste)`

Wie filter — behält alle Elemente, für die prädikat wahr ist.

```lisp
(remove-if-not (lambda (x) (= 1 (mod x 2))) '(1 2 3 4))
; => (1 3)
```

### remove-duplicates

**Syntax:** `(remove-duplicates liste)`

Entfernt doppelte Elemente (behält erstes Vorkommen).

```lisp
(remove-duplicates '(1 2 2 3))
; => (1 2 3)
```

### union

**Syntax:** `(union liste-a liste-b)`

Vereinigungsmenge zweier Listen.

```lisp
(union '(1 2) '(2 3))
; => (1 2 3)
```

### set-difference

**Syntax:** `(set-difference liste-a liste-b)`

Elemente von liste-a ohne die aus liste-b.

```lisp
(set-difference '(1 2 3) '(2))
; => (1 3)
```

### find-all

**Syntax:** `(find-all element seq &key test)`

Liefert alle Vorkommen von element in seq (Default-Test equal?).

```lisp
(find-all 2 '(1 2 2 3))
; => (2 2)
```

### butlast

**Syntax:** `(butlast liste)`

Liefert die Liste ohne das letzte Element.

```lisp
(butlast '(1 2 3))
; => (1 2)
```

### take

**Syntax:** `(take n liste)`

Liefert die ersten n Elemente.

```lisp
(take 2 '(1 2 3))
; => (1 2)
```

### drop

**Syntax:** `(drop n liste)`

Liefert die Liste ohne die ersten n Elemente.

```lisp
(drop 2 '(1 2 3))
; => (3)
```

### copy-list

**Syntax:** `(copy-list liste)`

Flache Kopie.

```lisp
(copy-list '(1 2))
```

### copy-tree

**Syntax:** `(copy-tree x)`

Tiefe Kopie (rekursiv auch in Sublisten).

```lisp
(copy-tree '(1 (2 3)))
```

### make-list

**Syntax:** `(make-list n &key initial-element)`

Erzeugt eine Liste mit n Elementen (Default nil).

```lisp
(make-list 3 :initial-element 'x)
; => (x x x)
```

### set-nth

**Syntax:** `(set-nth liste n wert)`

Liefert Kopie mit n-tem Element (0-basiert) ersetzt.

```lisp
(set-nth '(1 2 3) 1 99)
; => (1 99 3)
```

### length

**Syntax:** `(length seq)`

Anzahl Elemente einer Liste bzw. Zeichen (Runes) eines Strings.

```lisp
(length '(1 2 3))
(length "abc")
; => 3
```

## Strings

### string-length

**Syntax:** `(string-length str)`

Liefert die Anzahl der Zeichen (Runes) eines Strings.

```lisp
(string-length "Hallo")
(string-length "αβγ")
```

### string-append

**Syntax:** `(string-append str ...)`

Verkettet mehrere Strings zu einem neuen String.

```lisp
(string-append "Hallo" " " "Welt")
(string-append "a" "b" "c")
```

### substring

**Syntax:** `(substring str start end)`

Liefert den Teilstring von start (inklusiv) bis end (exklusiv).

```lisp
(substring "Hallo" 0 2)
(substring "Welt" 1 3)
```

### string-upcase

**Syntax:** `(string-upcase str)`

Wandelt einen String in Großbuchstaben um.

```lisp
(string-upcase "hallo")
```

### string-downcase

**Syntax:** `(string-downcase str)`

Wandelt einen String in Kleinbuchstaben um.

```lisp
(string-downcase "HALLO")
```

### string->number

**Syntax:** `(string->number str)`

Parst einen String als Zahl.

```lisp
(string->number "42")
(string->number "3.14")
```

### number->string

**Syntax:** `(number->string zahl)`

Wandelt eine Zahl in einen String um.

```lisp
(number->string 42)
(number->string 3.14)
```

### string->list

**Syntax:** `(string->list str)`

Wandelt einen String in eine Liste von Einzelzeichen-Strings um.

```lisp
(string->list "abc")
```

### list->string

**Syntax:** `(list->string liste)`

Verkettet eine Liste von Strings zu einem String.

```lisp
(list->string '("a" "b" "c"))
```

### string-replace

**Syntax:** `(string-replace str alt neu)`

Ersetzt alle Vorkommen von alt durch neu.

```lisp
(string-replace "Hallo Welt" "Welt" "GoLisp")
```

### string-trim

**Syntax:** `(string-trim str)`

Entfernt führende und abschließende Leerzeichen.

```lisp
(string-trim "  hallo  ")
```

### string-contains

**Syntax:** `(string-contains str sub)`

Gibt t zurück, wenn sub in str enthalten ist.

```lisp
(string-contains "Hallo" "all")
(string-contains "Hallo" "xyz")
```

### string-find

**Syntax:** `(string-find nadel heuhaufen)`

Liefert die Position (0-basiert) des ersten Vorkommens oder nil.

```lisp
(string-find "ll" "hello")
; => 2
```

### coerce

**Syntax:** `(coerce x typ)`

Typumwandlung. Unterstützte Typen: `'list` (String → Zeichenliste) und `'string` (Stringliste → String).

```lisp
(coerce "ab" 'list)
; => ("a" "b")
(coerce '("a" "b") 'string)
; => "ab"
```

### symbol->string

**Syntax:** `(symbol->string 'symbol)`

Wandelt ein Symbol in seinen String-Namen um.

```lisp
(symbol->string 'foo)
(symbol->string 'my-var)
```

### symbol-name

**Syntax:** `(symbol-name symbol)`

Wie symbol->string — Name des Symbols als String.

```lisp
(symbol-name 'abc)
; => "abc"
```

### intern

**Syntax:** `(intern "name")`

Liefert das internierte Symbol zum String (die eine Symbol-Cell aus der Interning-Tabelle).

```lisp
(intern "foo")
; => foo
(eq (intern "foo") 'foo)
; => t
```

## Formatieren

### format

**Syntax:** `(format ziel format-string arg ...)`

CL-FORMAT-Engine (HyperSpec 22.3). Ziel `nil` → formatierter String als Rückgabe; Ziel `t` → Ausgabe auf stdout. Direktiven wie `~a` (aesthetic), `~s` (readable), `~d`, `~f`, `~%` (newline).

```lisp
(format nil "~a + ~a = ~s" 1 2 3)
; => "1 + 2 = 3"
(format t "Wert: ~a~%" 42)
```

### sprintf

**Syntax:** `(sprintf format-string arg ...)`

C-Stil (printf-Direktiven wie `%d`, `%s`, `%f`) → liefert String.

```lisp
(sprintf "%05.2f" 3.14159)
; => "03.14"
```

### printf

**Syntax:** `(printf format-string arg ...)`

C-Stil, gibt direkt auf stdout aus.

```lisp
(printf "%d %s\n" 42 "text")
```

### fprintf

**Syntax:** `(fprintf ziel format-string arg ...)`

C-Stil, schreibt in eine Datei (Dateiname als String — appended an bestehende Datei) oder auf den Systemstream `"stdout"`/`"stderr"`.

```lisp
(system "mkdir -p ./tmp")
(fprintf "./tmp/log.txt" "a=%d b=%s\n" 1 "x")
(fprintf "stderr" "fehler %d\n" 7)
(file-read "./tmp/log.txt")
```

### sscanf

**Syntax:** `(sscanf string format-string)`

Parst einen String nach C-scanf-Muster; liefert Liste der gelesenen Werte.

```lisp
(sscanf "42 hallo" "%d %s")
; => (42 "hallo")
```

## Ein-/Ausgabe

### print

**Syntax:** `(print wert ...)`

Gibt Werte ohne abschließenden Zeilenumbruch auf stdout aus. Rückgabewert ist das letzte Argument.

```lisp
(print "Hallo")
(print 1 " " 2)
```

### println

**Syntax:** `(println wert ...)`

Gibt Werte mit abschließendem Zeilenumbruch auf stdout aus. Rückgabewert ist das letzte Argument.

```lisp
(println "Zeile")
(println 1 2 3)
```

### read

**Syntax:** `(read str)`

Parst einen String und gibt die entsprechende Lisp-Datenstruktur zurück.

```lisp
(read "(+ 1 2)")
(car (read "(a b c)"))
```

### read-line

**Syntax:** `(read-line)`

Liest eine Zeile von stdin.

```lisp
;; echo "zeile1" | ./build/golisp2 -e "(read-line)"
; => "zeile1"
```

### gets

**Syntax:** `(gets)`

Wie read-line — liest eine Zeile von stdin.

```lisp
(gets)
```

### slurp

**Syntax:** `(slurp)`

Liest stdin komplett als String.

```lisp
;; echo "hallo stdin" | ./build/golisp2 -e "(slurp)"
; => "hallo stdin\n"
```

### err-write

**Syntax:** `(err-write string)`

Schreibt einen String auf stderr.

```lisp
(err-write "auf stderr\n")
```

## Fehler & Conditions

### error

**Syntax:** `(error wert)`

Signalisiert einen Lisp-Laufzeitfehler, der von catch aufgefangen werden kann.

```lisp
(error "etwas ist schief")
(catch (error 'x) (lambda (e) (println e)))
```

### assert

**Syntax:** `(assert form)`

Makro: signalisiert einen Fehler, wenn form nil ergibt.

```lisp
(assert (= 1 1))
```

### warn

**Syntax:** `(warn string)`

Gibt eine Warnung aus (kein Fehler, Auswertung läuft weiter).

```lisp
(warn "nur eine warnung")
```

### ignore-errors

**Syntax:** `(ignore-errors body ...)`

Makro: wertet body aus; bei Fehler liefert es nil statt des Fehlers.

```lisp
(ignore-errors (/ 1 0))
; => ()
```

### define-condition

**Syntax:** `(define-condition name (eltern ...) (slot ...) ...)`

Definiert einen Condition-Typ mit Slots (Schlüsselwort-Slots wie `:msg`).

```lisp
(define-condition meine-fehler () ())
(define-condition netz-fehler (meine-fehler) ())
```

### signal

**Syntax:** `(signal typ &key slot-wert ...)`

Signalisiert eine Condition des definierten Typs.

```lisp
(signal 'meine-fehler :msg "hoppla")
```

### handler-case

**Syntax:** `(handler-case body (typ (var) handler-body ...) ...)`

Fängt signalierte Conditions (und Laufzeitfehler) typbasiert ab.

```lisp
(define-condition meine-fehler () ())
(handler-case (signal 'meine-fehler :msg "hoppla")
  (meine-fehler (c) (list 'gefangen (lisp-error-msg c))))
; => (gefangen "hoppla")
```

### lisp-error-msg

**Syntax:** `(lisp-error-msg condition)`

Liefert den `:msg`-Slot einer Condition.

```lisp
(lisp-error-msg c)
```

## setf & Places

### setf

**Syntax:** `(setf place wert)`

Generalisierte Zuweisung. Places: Variablen, `(car lst)`, `(gethash key tbl)`, Struktur-Slots wie `(punkt-x p)`.

```lisp
(define x 1)
(setf x 9)
; => 9

(define lst '(1 2 3))
(setf (car lst) 99)
lst
; => (99 2 3)

(define h (make-hash-table))
(setf (gethash "k" h) 7)
(gethash "k" h)
; => 7
```

### incf

**Syntax:** `(incf place [d])`

Erhöht place um d (Default 1).

```lisp
(define z 5)
(incf z)
; => 6
(incf z 10)
; => 16
```

### decf

**Syntax:** `(decf place [d])`

Vermindert place um d (Default 1).

```lisp
(define z 16)
(decf z 2)
; => 14
```

### register-setf-expander

**Syntax:** `(register-setf-expander accessor setter)`

Registriert einen setf-Expander für eigene Accessor-Funktionen.

```lisp
;; Details: siehe src/embed/stdlib.lisp (*setf-expanders*)
```

## Werte & Bindung

### values

**Syntax:** `(values wert ...)`

Liefert mehrere Werte; verwendende Stellen sehen den Hauptwert (CL-multiple-values-lite).

```lisp
(values 1 2 3)
; => 1
```

### destructuring-bind

**Syntax:** `(destructuring-bind (var ...) expr body ...)`

Bindet Listenelemente an Variablen. Flache Muster — keine dotted pairs, keine Verschachtelung.

```lisp
(destructuring-bind (a b c) '(1 2 3) (list a b c))
; => (1 2 3)
```

### defvar

**Syntax:** `(defvar name [wert])`

Definiert eine Variable mit Dokumentationskonvention; Neuauswertung definiert nicht neu (CL-defvar-Semantik in defsystem-Kontext).

```lisp
(defvar *zaehler* 0)
```

### documentation

**Syntax:** `(documentation symbol 'function)`

Liefert den Docstring einer defun/defmacro-Definition (String-Literal nach der Parameterliste) oder nil. Andere doc-types liefern nil (CL-konform).

```lisp
(defun add (a b) "addiert zwei Zahlen" (+ a b))
(documentation 'add 'function)
; => "addiert zwei Zahlen"
```

## Hash-Tables

### make-hash-table

**Syntax:** `(make-hash-table)`

Erzeugt eine Hash-Table (thread-sicher).

```lisp
(define h (make-hash-table))
```

### gethash

**Syntax:** `(gethash key tabelle [default])`

Liefert zwei Werte: den Eintrag (oder default) und t/nil als Gefunden-Indikator.

```lisp
(puthash "a" h 1)
(gethash "a" h)
; => 1
(gethash "nix" h 'default)
; => default
```

### puthash

**Syntax:** `(puthash key tabelle wert)`

Setzt einen Eintrag.

```lisp
(puthash "b" h 2)
```

### remhash

**Syntax:** `(remhash key tabelle)`

Entfernt einen Eintrag.

```lisp
(remhash "a" h)
```

### clrhash

**Syntax:** `(clrhash tabelle)`

Leert die Hash-Table.

```lisp
(clrhash h)
```

### hash-table-count

**Syntax:** `(hash-table-count tabelle)`

Anzahl der Einträge.

```lisp
(hash-table-count h)
```

### hash-table-p

**Syntax:** `(hash-table-p obj)`

Prädikat: t, wenn obj eine Hash-Table ist.

```lisp
(hash-table-p h)
; => t
```

### maphash

**Syntax:** `(maphash funktion tabelle)`

Wendet funktion (zwei Argumente: key, wert) auf jeden Eintrag an.

```lisp
(maphash (lambda (k v) (println k "=" v)) h)
```

## Strukturen (defstruct)

### defstruct

**Syntax:** `(defstruct name slot ...)`

Definiert eine Struktur mit Slots. Erzeugt automatisch: `make-name` (Konstruktor mit `:slot wert`-Keywords), `name-slot`-Accessoren, `name-p`-Prädikat. Slots sind setf-Places.

```lisp
(defstruct punkt x y)
(define p (make-punkt :x 1 :y 2))
(punkt-x p)
; => 1
(setf (punkt-x p) 10)
(punkt-x p)
; => 10
(punkt-p p)
; => t
```

### defstruct-resolve-name

**Syntax:** `(defstruct-resolve-name praefix name slot trennzeichen reload?)`

Interner Helfer des defstruct-Makros (Namensauflösung für Accessoren).

```lisp
;; Intern — siehe src/embed/stdlib.lisp
```

## Generische Funktionen

### defgeneric

**Syntax:** `(defgeneric name (parameter ...))`

Definiert eine generische Funktion; die Methodenauswahl erfolgt per Dispatch auf dem ersten Argument.

```lisp
(defgeneric flaeche (obj))
```

### defmethod

**Syntax:** `(defmethod name (parameter) body ...)`

Definiert eine Methode für eine generische Funktion. Das Dispatch-Argument wird als `(var typ)` geschrieben — typ ist ein defstruct-Name oder `(eql wert)`.

```lisp
(defstruct punkt x y)
(defgeneric flaeche (obj))
(defmethod flaeche ((obj punkt)) (* (punkt-x obj) (punkt-y obj)))
(define p (make-punkt :x 10 :y 2))
(flaeche p)
; => 20

(defmethod flaeche ((obj (eql :einheit-quadrat))) 1)
(flaeche :einheit-quadrat)
; => 1
```

## Systeme (defsystem)

### defsystem

**Syntax:** `(defsystem name &key depends-on components)`

Definiert ein System aus Dateien (`:components`, Stringliste) mit Abhängigkeiten zu anderen Systemen (`:depends-on`, Symbolliste).

```lisp
;; tmp/sys/a.lisp: (defun sys-a () 1)
;; tmp/sys/b.lisp: (defun sys-b () (+ 1 (sys-a)))
(defsystem mein-sys :components ("tmp/sys/a.lisp" "tmp/sys/b.lisp"))
```

### load-system

**Syntax:** `(load-system name)`

Lädt ein System topologisch sortiert (Abhängigkeiten zuerst, mit Zyklenerkennung). Bereits geladene Dateien werden übersprungen.

```lisp
(load-system 'mein-sys)
(sys-b)
; => 2
```

### unload-system

**Syntax:** `(unload-system name)`

Entfernt ein System aus der Ladestatistik (Definitionen bleiben im Env).

```lisp
(unload-system 'mein-sys)
```

### loaded-systems

**Syntax:** `(loaded-systems)`

Liste der geladenen Systeme.

```lisp
(loaded-systems)
; => (mein-sys)
```

### system-symbols

**Syntax:** `(system-symbols name)`

Liste der Symbole, die ein System definiert hat.

```lisp
(system-symbols 'mein-sys)
; => (sys-b sys-a)
```

## Makro-Hilfe

### gensym

**Syntax:** `(gensym)`

Erzeugt ein eindeutiges Symbol für Makros.

```lisp
(gensym)
(gensym)
```

## Datei-I/O

### file-write

**Syntax:** `(file-write "pfad" inhalt ...)`

Schreibt Inhalte in eine Datei (überschreibt).

```lisp
(system "mkdir -p ./tmp")
(file-write "./tmp/demo.txt" "Hallo")
(file-write "./tmp/zahlen.txt" 1 " " 2)
```

### file-append

**Syntax:** `(file-append "pfad" inhalt ...)`

Hängt Inhalte an eine Datei an.

```lisp
(system "mkdir -p ./tmp")
(file-write "./tmp/demo.txt" "Hallo")
(file-append "./tmp/demo.txt" " Welt")
```

### file-read

**Syntax:** `(file-read "pfad")`

Liest den Inhalt einer Datei als String.

```lisp
(system "mkdir -p ./tmp")
(file-write "./tmp/demo.txt" "Hallo")
(file-read "./tmp/demo.txt")
```

### file-exists?

**Syntax:** `(file-exists? "pfad")`

Gibt t zurück, wenn die Datei existiert.

```lisp
(system "mkdir -p ./tmp")
(file-exists? "./tmp/demo.txt")
(file-exists? "./tmp/nichtda.txt")
```

### file-delete

**Syntax:** `(file-delete "pfad")`

Löscht eine Datei. Gibt t zurück.

```lisp
(system "mkdir -p ./tmp")
(file-write "./tmp/demo.txt" "x")
(file-delete "./tmp/demo.txt")
```

## Nebenläufigkeit

### chan-make

**Syntax:** `(chan-make)  bzw.  (chan-make n)`

Erzeugt einen Channel (optional mit Puffergröße n).

```lisp
(define ch (chan-make))
(define ch2 (chan-make 10))
```

### chan-send

**Syntax:** `(chan-send ch wert)`

Sendet einen Wert in einen Channel.

```lisp
(chan-send ch 42)
```

### chan-recv

**Syntax:** `(chan-recv ch)`

Empfängt einen Wert aus einem Channel (blockierend, wenn leer).

```lisp
(chan-recv ch)
```

### lock-make

**Syntax:** `(lock-make)`

Erzeugt einen Mutex für lock.

```lisp
(define mu (lock-make))
```

## KI

### sigo

**Syntax:** `(sigo prompt [modell] [session-id] [host])`

Sendet einen Prompt an sigoREST und gibt die Antwort als String zurück. Default-Modell und Host können via Env-Variablen gesetzt werden.

```lisp
(sigo "Erkläre TCO")
(sigo "Hallo" "cl46-s")
```

### sigo-models

**Syntax:** `(sigo-models)`

Liefert eine Liste der verfügbaren sigoREST-Modelle.

```lisp
(sigo-models)
```

### sigo-host

**Syntax:** `(sigo-host "http://...")`

Setzt oder liest den sigoREST-Host.

```lisp
(sigo-host "http://mammouth:9080")
```

## Genetische Algorithmen

### ga-create

**Syntax:** `(ga-create typ gen-len gen-par fitness-fn)`

Erzeugt einen genetischen Algorithmus. `typ` ist ein Atom (`bit1`, `bit2`,
`bit4`, `bit8`, `biti`, `bitf`), `gen-len` die Länge eines Genoms,
`gen-par` die Populationsgröße und `fitness-fn` eine Lisp-Funktion, die ein
Genom als Liste entgegennimmt und eine Zahl zurückgibt.

```lisp
(define ga (ga-create 'bit1 10 20 (lambda (g) (apply + g))))
(ga? ga)
```

### ga-init

**Syntax:** `(ga-init ga)`

Initialisiert die Population mit Zufallswerten.

```lisp
(ga-init ga)
```

### ga-calc

**Syntax:** `(ga-calc ga)`

Berechnet die Fitness jedes Genoms, sortiert die Population absteigend.
Die Fitness-Funktion wird parallel aufgerufen und muss rein sein.

```lisp
(ga-calc ga)
```

### ga-cross

**Syntax:** `(ga-cross ga codist)`

Führt Crossover durch. `codist` bestimmt die Blockgröße, in der zwischen
Eltern gewechselt wird.

```lisp
(ga-cross ga 2)
```

### ga-select

**Syntax:** `(ga-select ga keep)`

Behält die besten `keep` Genome.

```lisp
(ga-select ga 4)
```

### ga-mut

**Syntax:** `(ga-mut ga mutf)`

Mutiert jedes Gen mit Wahrscheinlichkeit `mutf` (0.0 – 1.0).

```lisp
(ga-mut ga 0.1)
```

### ga-result

**Syntax:** `(ga-result ga)`

Liefert die aktuellen Fitness-Scores als sortierte Liste.

```lisp
(ga-result ga)
```

### ga-print

**Syntax:** `(ga-print ga lines)`

Gibt die Population formatiert auf stdout aus. `lines` > 0 begrenzt die
Ausgabe, `lines` = 0 unterdrückt sie, `lines` < 0 gibt alle aus.

```lisp
(ga-print ga 3)
```

### ga?

**Syntax:** `(ga? obj)`

Prädikat: liefert `t`, wenn `obj` ein GA-Handle ist.

```lisp
(ga? ga)
(ga? 42)
```

## Zeit

### sleep

**Syntax:** `(sleep ms)`

Pausiert für die angegebene Anzahl Millisekunden.

```lisp
(sleep 100)
(sleep 2000)
```

### get-universal-time

**Syntax:** `(get-universal-time)`

Sekunden seit 1.1.1900 00:00 UTC (CL-universal-time).

```lisp
(get-universal-time)
; => z.B. 3996832127
```

## Memory

### memstats

**Syntax:** `(memstats)`

Liefert Go-Runtime-Memory-Statistiken als Assoziationsliste.

```lisp
(memstats)
(assoc 'heapalloc (memstats))
```

## Shell & System

### system

**Syntax:** `(system "shell-kommando")`

Führt ein Shell-Kommando aus und gibt den Exit-Code zurück (0 = OK).

```lisp
(system "echo hallo")
(system "test -f ./tmp/demo.txt")
```

### shell-assoc

**Syntax:** `(shell-assoc key assoc-liste)`

Sucht key in einer Assoziationsliste von Paaren (für Shell-Output-Parsing); liefert das passende Paar oder nil.

```lisp
(shell-assoc "sh" '(("bash" . "/usr/bin/bash") ("sh" . "/bin/sh")))
; => ("sh" . "/bin/sh")
```

### exec

**Syntax:**

```lisp
(exec "programm"
      param: "arg1"
      param: "arg2"
      stdin:  eingabe
      stdout: ausgabe-var
      stderr: fehler-var
      exitcd: code-var)
```

Führt ein externes Programm direkt aus (ohne Shell). Siehe ausführliche Beschreibung unter Spezialformen.

```lisp
(exec "echo" param: "hallo" stdout: out exitcd: cd)
(println out)
(println cd)
```

### file-stat

**Syntax:** `(file-stat "pfad")`

Liefert eine Assoziationsliste mit size und mtime oder nil, wenn die Datei nicht existiert.

```lisp
(system "mkdir -p ./tmp")
(file-write "./tmp/demo.txt" "Hallo")
(file-stat "./tmp/demo.txt")
(assoc 'size (file-stat "./tmp/demo.txt"))
```

### getenv

**Syntax:** `(getenv "name")`

Wert einer Umgebungsvariablen (String) oder nil.

```lisp
(getenv "HOME")
```

### environ

**Syntax:** `(environ)`

Alle Umgebungsvariablen als Assoziationsliste `(name . wert)`.

```lisp
(car (environ))
; => ("SHELL" . "/bin/bash")
```

### argv

**Syntax:** `(argv)`

Kommandozeilenargumente des golisp2-Prozesses als Liste.

```lisp
(length (argv))
```

### exit

**Syntax:** `(exit [code])`

Beendet den Lisp-Prozess mit Exit-Code (Default 0).

```lisp
(exit 1)
```

### get-working-directory

**Syntax:** `(get-working-directory)`

Aktuelles Arbeitsverzeichnis.

```lisp
(get-working-directory)
```

### set-working-directory

**Syntax:** `(set-working-directory "pfad")`

Wechselt das Arbeitsverzeichnis.

```lisp
(set-working-directory "/tmp")
```

### get-file-path

**Syntax:** `(get-file-path "pfad")`

Löst einen Pfad auf (inklusive Suchpfad-Logik von load).

```lisp
(get-file-path "stdlib.lisp")
```

## Shared Memory

### shm-alloc

**Syntax:** `(shm-alloc [worker-id])`

Belegt einen Block im Shared-Memory-Pool; liefert ein Handle.

```lisp
(define z (shm-alloc))
```

### shm-write

**Syntax:** `(shm-write handle string)`

Schreibt einen String in den Block; liefert den String zurück.

```lisp
(shm-write z "daten")
```

### shm-read

**Syntax:** `(shm-read handle [n])`

Liest den Block (maximal n Zeichen) als String.

```lisp
(shm-read z 5)
; => "daten"
```

### shm-free

**Syntax:** `(shm-free handle)`

Gibt den Block frei.

```lisp
(shm-free z)
```

### shm-status

**Syntax:** `(shm-status)`

Pool-Statistik als Aliste (`total`, `used`, `free`).

```lisp
(shm-status)
; => ((total . 150) (used . 1) (free . 149))
```

### shm-cleanup

**Syntax:** `(shm-cleanup)`

Räumt den Shared-Memory-Pool auf.

```lisp
(shm-cleanup)
```

## Trace & Debug

### trace

**Syntax:** `(trace 'funktion)`

Aktiviert Live-Tracing einer Funktion: jeder Aufruf und Rückgabewert wird auf stderr protokolliert.

```lisp
(defun tf (x) (* x 2))
(trace 'tf)
(tf 21)
;; Ausgabe: (tf 21)
;;          (tf 21) => 42
```

### untrace

**Syntax:** `(untrace 'funktion)`

Deaktiviert das Tracing.

```lisp
(untrace 'tf)
```

### trace?

**Syntax:** `(trace? 'funktion)`

Prädikat: t, wenn die Funktion gerade getraced wird.

```lisp
(trace? 'tf)
```

## Introspektion

### env-symbols

**Syntax:** `(env-symbols)`

Liste aller im Root-Environment gebundenen Symbole.

```lisp
(length (env-symbols))
; => z.B. 258
```

### defined-in

**Syntax:** `(defined-in "datei.lisp")`

Liste der Symbole, die in der angegebenen (geladenen) Datei definiert wurden.

```lisp
(load-system 'mein-sys)
(defined-in "tmp/sys/a.lisp")
; => (sys-a)
```

### redefine-policy

**Syntax:** `(redefine-policy 'warn)` — lesen: `(redefine-policy)`

Redifinitions-Politik des Root-Env: `'allow`, `'warn` (Default) oder `'error`. Bewacht, dass der letzte Schreiber nicht mehr lautlos gewinnt.

```lisp
(redefine-policy)
; => warn
```

### redef-log

**Syntax:** `(redef-log)`

Ringpuffer der Redefinitionen.

```lisp
(redef-log)
```

### redef-log-clear

**Syntax:** `(redef-log-clear)`

Leert den Redefinitions-Log.

```lisp
(redef-log-clear)
```

## PostgreSQL

### pg-connect

**Syntax:** `(pg-connect "connection-string")`

Öffnet eine PostgreSQL-Verbindung. Connection-String z.B. host=... port=... user=... password=... dbname=...

```lisp
(define conn (pg-connect "host=localhost user=golisp2 dbname=test sslmode=disable"))
```

### pg-query

**Syntax:** `(pg-query conn "SELECT ..." param ...)`

Führt eine SELECT-Abfrage aus und gibt eine Liste von Zeilen (Assoziationslisten) zurück.

```lisp
(pg-query conn "SELECT * FROM users WHERE id = $1" 42)
```

### pg-exec

**Syntax:** `(pg-exec conn "INSERT/UPDATE ..." param ...)`

Führt eine schreibende Abfrage aus und gibt die Anzahl betroffener Zeilen zurück.

```lisp
(pg-exec conn "INSERT INTO users (name) VALUES ($1)" "Alice")
```

### pg-close

**Syntax:** `(pg-close conn)`

Schließt eine PostgreSQL-Verbindung.

```lisp
(pg-close conn)
```

## Web-Bridge

### webserv

**Syntax:** `(webserv &key port host html htmlpath open)`

Ein-Aufruf-Bootstrap für eine Browser-Anbindung: startet einen HTTP-Server,
liefert `:html` (Inline-String) oder `:htmlpath` (Datei, bei jedem Request
neu gelesen) aus, injiziert automatisch `boot.js` und öffnet den Browser
(`:open` Default `t`). Genau eines von `:html`/`:htmlpath` ist Pflicht.
`:host` bindet das Interface (Default `127.0.0.1`). Gibt das Server-Objekt
zurück, auf dem `ws-export`/`ws-emit`/... normal weiterlaufen.

```lisp
(define s (webserv :htmlpath "./public/index.html"))
(ws-export s "ask" (lambda (client frage) (string-append "Echo: " frage)))
(http-wait s)
```

### http-serve

**Syntax:** `(http-serve port)` bzw. `(http-serve port :host "0.0.0.0")`

Startet einen HTTP-Server auf `host:port` (Default-Host `127.0.0.1`,
`port` `0` → freier Port vom OS), kehrt sofort zurück. Tiefere
Alternative zu `webserv` für Multi-File-Sites oder eigenes Routing.

```lisp
(define s (http-serve 0))
(http-static s "/" "./public")
(http-port s)
;; => z.B. 41213
```

### http-static

**Syntax:** `(http-static srv urlpath dir)`

Mountet ein Verzeichnis unter `urlpath` (muss mit `/` beginnen). Kein
Directory-Listing, mehrfach aufrufbar für weitere Pfade.

```lisp
(http-static s "/assets" "./public/assets")
```

### http-port

**Syntax:** `(http-port srv)`

Liefert den tatsächlich gebundenen Port — nützlich nach `(http-serve 0)`.

```lisp
(http-port s)
;; => 41213
```

### http-upload

**Syntax:** `(http-upload srv urlpath handler)`

Registriert einen POST-Endpoint für Multipart-File-Uploads. handler wird pro hochgeladener Datei mit zwei Argumenten aufgerufen: Dateiname (String) und Dateiinhalt (String).

```lisp
(http-upload s "/upload"
             (lambda (name inhalt)
               (file-write (string-append "./tmp/" name) inhalt)))
```

### http-wait

**Syntax:** `(http-wait srv &key idle-exit)`

Blockiert bis `http-stop`, SIGINT/SIGTERM (beendet dann den Prozess) oder —
mit `:idle-exit` (ms) — bis so lange kein Client mehr verbunden war.
Typischer Abschluss eines Skripts nach `webserv`/`http-serve`.

```lisp
(http-wait s :idle-exit 60000)   ; beendet sich nach 60s ohne Client
```

### http-stop

**Syntax:** `(http-stop srv)`

Graceful Shutdown (2s), idempotent — mehrfacher Aufruf ist unproblematisch.

```lisp
(http-stop s)
```

### browser-open

**Syntax:** `(browser-open url)`

Öffnet `url` im Browser (chromium/chrome/xdg-open, in dieser Reihenfolge),
Prozess detached, kein Warten. `webserv` ruft das intern selbst auf, außer
mit `:open ()`.

```lisp
(browser-open (string-append "http://127.0.0.1:" (number->string (http-port s))))
```

### ws-export

**Syntax:** `(ws-export srv name fn)`

Macht `fn` unter `name` als vom Browser aufrufbare Operation verfügbar
(`golisp.call('name', ...)`). `fn` bekommt die Client-ID als erstes
Argument. Erneutes `ws-export` überschreibt still, während der Client
verbunden bleibt — der Kern der Live-Image-Idee.

```lisp
(ws-export s "ask" (lambda (client frage) (string-append "Echo: " frage)))
;; spaeter, waehrend der Browser verbunden bleibt — Verhalten aendert sich live:
(ws-export s "ask" (lambda (client frage) (string-append "Neu: " frage)))
```

### ws-unexport

**Syntax:** `(ws-unexport srv name)`

Entfernt eine exportierte Operation. `t`, wenn sie registriert war, sonst
`nil`.

```lisp
(ws-unexport s "ask")
```

### ws-emit

**Syntax:** `(ws-emit srv event data)`

Server-Push an alle verbundenen Clients (`golisp.on('event', ...)` im
Browser). Liefert die Anzahl der Empfänger.

```lisp
(ws-emit s 'tick 42)
```

### ws-emit-to

**Syntax:** `(ws-emit-to srv client event data)`

Wie `ws-emit`, aber gezielt an eine Client-ID (aus `ws-clients` oder dem
ersten Argument eines `ws-export`-Handlers). `nil` bei unbekanntem Client.

```lisp
(ws-emit-to s 1 'private-msg "nur für dich")
```

### ws-eval

**Syntax:** `(ws-eval srv js)`

Feuert JavaScript-Code an alle Clients, ohne auf ein Ergebnis zu warten.

```lisp
(ws-eval s "console.log('hallo aus golisp2')")
```

### ws-call

**Syntax:** `(ws-call srv client js &key timeout)`

Ruft JS im Browser des angegebenen Clients auf und **blockiert** auf das
Ergebnis (Default-Timeout 5000 ms). Reentrant-sicher, auch aus dem eigenen
`ws-export`-Handler heraus aufrufbar.

```lisp
(ws-export s "breite" (lambda (c) (ws-call s c "return window.innerWidth")))
```

### ws-clients

**Syntax:** `(ws-clients srv)`

Liste der aktuell verbundenen Client-IDs.

```lisp
(ws-clients s)
;; => (1 2)
```

## Standardbibliothek

### cadr

**Syntax:** `(cadr liste)`

Kurzform für (car (cdr liste)).

```lisp
(cadr '(1 2 3))
(cadr '(a b c))
```

### caddr

**Syntax:** `(caddr liste)`

Kurzform für (car (cdr (cdr liste))).

```lisp
(caddr '(1 2 3))
(caddr '(a b c d))
```

### cadddr

**Syntax:** `(cadddr liste)`

Kurzform für (car (cdr (cdr (cdr liste)))).

```lisp
(cadddr '(1 2 3 4))
```

### cddr

**Syntax:** `(cddr liste)`

Kurzform für (cdr (cdr liste)).

```lisp
(cddr '(1 2 3))
(cddr '(a b c d))
```

### cdar

**Syntax:** `(cdar liste)`

Kurzform für (cdr (car liste)).

```lisp
(cdar '((a b c)))
```

### caar

**Syntax:** `(caar liste)`

Kurzform für (car (car liste)).

```lisp
(caar '((a b)))
```

### first

**Syntax:** `(first liste)`

Alias für car.

```lisp
(first '(a b c))
```

### second

**Syntax:** `(second liste)`

Alias für cadr.

```lisp
(second '(a b c))
```

### third

**Syntax:** `(third liste)`

Alias für caddr.

```lisp
(third '(a b c))
```

### fourth

**Syntax:** `(fourth liste)`

Alias für cadddr.

```lisp
(fourth '(a b c d))
```

### rest

**Syntax:** `(rest liste)`

Alias für cdr.

```lisp
(rest '(a b c))
```

### zero?

**Syntax:** `(zero? n)`

Gibt t zurück, wenn n gleich 0 ist.

```lisp
(zero? 0)
(zero? 5)
```

### zerop

**Syntax:** `(zerop n)`

CL-Name für zero?.

```lisp
(zerop 0)
; => t
```

### positive?

**Syntax:** `(positive? n)`

Gibt t zurück, wenn n größer als 0 ist.

```lisp
(positive? 5)
(positive? -1)
```

### negative?

**Syntax:** `(negative? n)`

Gibt t zurück, wenn n kleiner als 0 ist.

```lisp
(negative? -3)
(negative? 1)
```

### pair?

**Syntax:** `(pair? x)`

Gibt t zurück, wenn x ein Cons / keine Atom ist.

```lisp
(pair? '(1 2))
(pair? 5)
```

### when

**Syntax:** `(when test body ...)`

Makro: wertet body nur aus, wenn test wahr ist.

```lisp
(when t 'ja)
(when nil 'nein)
```

### unless

**Syntax:** `(unless test body ...)`

Makro: wertet body nur aus, wenn test falsch ist.

```lisp
(unless nil 'ja)
(unless t 'nein)
```

### reverse

**Syntax:** `(reverse liste)`

Dreht eine Liste um.

```lisp
(reverse '(1 2 3))
(reverse '(a b c d))
```

### nth

**Syntax:** `(nth n liste)`

Liefert das n-te Element einer Liste (0-basiert).

```lisp
(nth 0 '(a b c))
(nth 2 '(a b c))
```

### last

**Syntax:** `(last liste)`

Liefert das letzte Element einer Liste.

```lisp
(last '(1 2 3))
(last '(a b))
```

### member

**Syntax:** `(member x liste)`

Sucht x in einer Liste und gibt die Restliste ab dem Treffer zurück.

```lisp
(member 'b '(a b c))
(member 'x '(a b c))
```

### assoc

**Syntax:** `(assoc key alist)`

Sucht key in einer Assoziationsliste und gibt das erste passende (key . val)-Paar zurück. Vergleicht mit `equal?`.

```lisp
(assoc 'b '((a . 1) (b . 2) (c . 3)))
(assoc 'x '((a . 1)))
```

### getf

**Syntax:** `(getf plist key [default])`

Sucht key in einer Property-Liste (`(key wert key wert ...)`); liefert den Wert oder default.

```lisp
(getf '(:a 1 :b 2) :b)
; => 2
```

### filter

**Syntax:** `(filter pred liste)`

Liefert alle Elemente, für die die Prädikatfunktion wahr ist.

```lisp
(filter (lambda (x) (> x 2)) '(1 2 3 4))
(filter atom? '((1 2) a (3 4) b))
```

### reduce

**Syntax:** `(reduce f seq &key initial-value from-end ...)`

Faltet eine Liste mit einer Funktion. Startwert über `:initial-value`.

```lisp
(reduce + '(1 2 3 4) :initial-value 0)
; => 10
(reduce * '(1 2 3 4) :initial-value 1)
; => 24
```

### for-each

**Syntax:** `(for-each f liste)`

Wendet f auf jedes Element an, gibt nil zurück.

```lisp
(for-each println '(1 2 3))
(for-each (lambda (x) (print (* x x))) '(2 3))
```

### any

**Syntax:** `(any pred liste)`

Gibt t zurück, wenn mindestens ein Element die Prädikatfunktion erfüllt.

```lisp
(any (lambda (x) (> x 5)) '(1 2 6))
(any (lambda (x) (> x 5)) '(1 2 3))
```

### every

**Syntax:** `(every pred liste)`

Gibt t zurück, wenn alle Elemente die Prädikatfunktion erfüllen.

```lisp
(every number? '(1 2 3))
(every number? '(1 a 3))
```

### flatten

**Syntax:** `(flatten liste)`

Entfernt eine Ebene von Verschachtelungen; rekursiv flach.

```lisp
(flatten '((1 2) (3 (4))))
(flatten '(a (b c) d))
```

### zip

**Syntax:** `(zip lst1 lst2)`

Verbindet zwei Listen elementweise zu Paaren.

```lisp
(zip '(1 2) '(a b))
(zip '(1 2 3) '(a b))
```

### list-tail

**Syntax:** `(list-tail liste n)`

Liefert den n-ten Rest der Liste.

```lisp
(list-tail '(1 2 3) 0)
(list-tail '(1 2 3) 2)
```

### iota

**Syntax:** `(iota n)`

Erzeugt die Liste (0 1 ... n-1).

```lisp
(iota 5)
(iota 0)
```

### square

**Syntax:** `(square x)`

Quadriert x.

```lisp
(square 5)
(square 3)
```

### expt

**Syntax:** `(expt base exp)`

Berechnet base hoch exp (exp ganze Zahl, rekursiv).

```lisp
(expt 2 10)
(expt 3 3)
```

### gcd

**Syntax:** `(gcd a b)`

Berechnet den größten gemeinsamen Teiler.

```lisp
(gcd 48 18)
(gcd 100 35)
```

### dotimes

**Syntax:** `(dotimes (var n) body ...)`

Makro: wiederholt body für var von 0 bis n-1.

```lisp
(dotimes (i 3) (print i))
(dotimes (i 3) (println (* i i)))
```

### dolist

**Syntax:** `(dolist (var liste) body ...)`

Makro: iteriert über eine Liste. Ergebnisform ist ().

```lisp
(dolist (x '(1 2 3)) (print x))
(dolist (c '("a" "b")) (println c))
```

### push

**Syntax:** `(push wert var)`

Makro: fügt wert am Anfang der Liste in var ein.

```lisp
(define s '())
(push 1 s)
(push 2 s)
s
```

### pop

**Syntax:** `(pop var)`

Makro: entfernt und liefert das erste Element der Liste in var.

```lisp
(define s '(1 2 3))
(pop s)
s
```

### alist-set

**Syntax:** `(alist-set key val liste)`

Setzt oder ersetzt einen Eintrag in einer Assoziationsliste.

```lisp
(alist-set 'b 99 '((a 1) (b 2)))
(alist-set 'd 4 '((a 1)))
```

### alist-get

**Syntax:** `(alist-get key liste)`

Liefert den Wert zu key aus einer Assoziationsliste oder nil.

```lisp
(alist-get 'b '((a 1) (b 2)))
(alist-get 'x '((a 1)))
```

### identity

**Syntax:** `(identity x)`

Gibt x unverändert zurück.

```lisp
(identity 42)
(identity '(1 2 3))
```

### constantly

**Syntax:** `(constantly x)`

Liefert eine Funktion, die bei jedem Aufruf x zurückgibt.

```lisp
((constantly 5) 'a 'b)
((constantly 'ok) 1 2 3)
```

### complement

**Syntax:** `(complement f)`

Liefert das logische Komplement einer Funktion.

```lisp
((complement null?) '(1 2))
((complement zero?) 5)
```

### compose

**Syntax:** `(compose f g)`

Liefert eine Funktion, die f(g(x)) auf ein Argument anwendet.

```lisp
((compose number->string (lambda (x) (* x 2))) 5)
((compose string-upcase symbol->string) 'foo)
```
