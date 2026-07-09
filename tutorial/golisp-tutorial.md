<!--
  golisp2-tutorial.md
  Autor    : Gerhard Quell - gquell@skequell.de
  CoAutor  : claude sonnet 4.6
  Copyright: 2026 Gerhard Quell - SKEQuell
  Erstellt : 20260623
-->

# GoLisp – Tutorial

Dieses Dokument beschreibt die öffentlichen Funktionen, Spezialformen und Makros von GoLisp mit kurzen, lauffähigen Beispielen. Dateioperationen nutzen immer das Projekt-temp-Verzeichnis `./tmp`. Funktionen, die externe Dienste benötigen (sigo, PostgreSQL), enthalten beschreibende Beispiele.

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

Definiert eine benannte Funktion. Mehrere Body-Ausdrücke werden implizit in begin gewrappt.

```lisp
(defun quadrat (x) (* x x))
(quadrat 7)
(defun add (a b) (print a) (+ a b))
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

Wertet Ausdrücke nacheinander aus und gibt das Ergebnis des letzten zurück.

```lisp
(begin (print 1) (print 2) 3)
(begin (define x 1) (set! x 2) x)
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

### macroexpand-all

**Syntax:** `(macroexpand-all form)`

Expandiert Makros rekursiv in allen Subformen.

```lisp
(macroexpand-all (doppelt (+ 1 2)))
```

### function

**Syntax:** `(function ausdruck)`

Wertet den übergebenen Ausdruck aus und gibt seinen Wert zurück.

```lisp
(function +)
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

### mapcar

**Syntax:** `(mapcar funktion liste)`

Wendet eine Funktion auf jedes Element einer Liste an und liefert die Ergebnisliste.

```lisp
(mapcar (lambda (x) (* x x)) '(1 2 3))
(mapcar atom? '(1 (2 3) "a"))
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

Wertet einen bereits ausgewerteten Ausdruck nochmals im globalen Environment aus.

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

Pointer-Identität – prüft, ob beide Argumente dasselbe Objekt im Speicher sind.

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

**Syntax:** `(append liste element)`

Hängt ein einzelnes Element an das Ende einer Liste an.

```lisp
(append '(1 2) 3)
(append '(a) 'b)
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

## Fehler

### error

**Syntax:** `(error wert)`

Signalisiert einen Lisp-Laufzeitfehler, der von catch aufgefangen werden kann.

```lisp
(error "etwas ist schief")
(catch (error 'x) (lambda (e) (println e)))
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

### file-stat

**Syntax:** `(file-stat "pfad")`

Liefert eine Assoziationsliste mit size und mtime oder nil, wenn die Datei nicht existiert.

```lisp
(system "mkdir -p ./tmp")
(file-write "./tmp/demo.txt" "Hallo")
(file-stat "./tmp/demo.txt")
(assoc 'size (file-stat "./tmp/demo.txt"))
```

### assoc

**Syntax:** `(assoc key alist)`

Sucht das erste Paar (key . val) in einer Assoziationsliste mittels equal?.

```lisp
(assoc 'b '((a . 1) (b . 2) (c . 3)))
(assoc 'x '((a . 1)))
```

### symbol->string

**Syntax:** `(symbol->string 'symbol)`

Wandelt ein Symbol in seinen String-Namen um.

```lisp
(symbol->string 'foo)
(symbol->string 'my-var)
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

### append

**Syntax:** `(append lst1 lst2)`

Hängt zwei Listen aneinander.

```lisp
(append '(1 2) '(3 4))
(append '(a) '(b c d))
```

### reverse

**Syntax:** `(reverse liste)`

Dreht eine Liste um.

```lisp
(reverse '(1 2 3))
(reverse '(a b c d))
```

### length

**Syntax:** `(length liste)`

Liefert die Anzahl der Elemente einer Liste.

```lisp
(length '(1 2 3))
(length '())
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

**Syntax:** `(assoc key liste)`

Sucht key in einer Assoziationsliste und gibt das erste passende (key val)-Paar zurück.

```lisp
(assoc 'b '((a 1) (b 2) (c 3)))
(assoc 'x '((a 1)))
```

### filter

**Syntax:** `(filter pred liste)`

Liefert alle Elemente, für die die Prädikatfunktion wahr ist.

```lisp
(filter (lambda (x) (> x 2)) '(1 2 3 4))
(filter atom? '((1 2) a (3 4) b))
```

### reduce

**Syntax:** `(reduce f init liste)`

Faltet eine Liste mit einer Funktion und einem Startwert zusammen.

```lisp
(reduce + 0 '(1 2 3 4))
(reduce * 1 '(1 2 3 4))
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

Makro: iteriert über eine Liste.

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

