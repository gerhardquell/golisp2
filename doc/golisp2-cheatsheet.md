# GoLisp2 — Cheatsheet + Semantik-Überblick (Mensch-Referenz)

> **Ziel:** Ausführliche Referenz für Menschen, die GoLisp2-Code schreiben oder
> verstehen wollen. Kompakte KI-Version: `doc/ki/referenz.md`.
> **Stand:** 20260725 · **Quelle:** `eval_core.go`, `lib/primitives.go`,
> `embed/stdlib.lisp`, `doc/lisp-semantik.md`.

---

## Inhaltsverzeichnis

1. [Eval-Reihenfolge](#1-eval-reihenfolge)
2. [Spezialformen](#2-spezialformen)
3. [Primitiven](#3-primitiven)
4. [Stdlib](#4-stdlib)
5. [Wahrheitswerte und Gleichheit](#5-wahrheitswerte-und-gleichheit)
6. [Makros und Quasiquote](#6-makros-und-quasiquote)
7. [Fehlerhandling](#7-fehlerhandling)
8. [Closures, TCO, Environment](#8-closures-tco-environment)
9. [Gotchas und CL-Abweichungen](#9-gotchas-und-cl-abweichungen)
10. [Schwächen (bewusst)](#10-schwächen-bewusst)
11. [Schnell-Lookup-Tabelle](#11-schnell-lookup-tabelle)

---

## 1. Eval-Reihenfolge

GoLisp2 wertet einen Ausdruck `expr` in Umgebung `env` aus:

1. **ATOM** → Wert aus `env` nachschlagen (unbekannt → Fehler)
2. **LIST** → `car` prüfen:
   - **Spezialform** (in `eval_core.go` per `case` dispatchen) → direkt ausführen,
     Argumente **nicht** automatisch auswerten
   - **MACRO** → expandieren, Ergebnis neu evaluieren
   - **FUNC/LAMBDA** → Argumente auswerten, `apply`

Tail-Calls (`if`, `begin`, `let`, `case`, `cond`, `prog1/2`, `catch`, `throw`,
`tagbody/go`, `block/return-from`, `do`) setzen `expr`/`env` und machen `continue`
im `for {}-Loop` — **kein neuer Stack-Frame**, O(1) Stack bei beliebig tiefer
Tail-Rekursion.

**Beispiel — TCO:**
```lisp
(defun sum-acc (n acc)
  (if (= n 0) acc (sum-acc (- n 1) (+ acc n))))

(sum-acc 1000000 0)   ; → 500000500000, kein Stack-Overflow
```

---

## 2. Spezialformen

GoLisp2 hat **55 Spezialformen** (alle in `lib/eval_core.go` dispatchend) —
plus `dotimes` und `dolist` als Stdlib-Makros.
Wichtige:

### Definitionen und Bindungen

```lisp
(define sym val)                  ; Variable definieren (global/lokal)
(set! sym val)                    ; Variable updaten (= setq)
(setq sym val)                    ; Alias für set!
(setq* s1 v1 s2 v2 ...)           ; Sequentiell setzen
(psetq s1 v1 s2 v2 ...)           ; Parallel setzen

(defun f (params) body...)        ; Funktion definieren
(defmacro m (params) body...)     ; Makro definieren
(lambda (params) body...)         ; Closure erzeugen

(let ((x 1) (y 2) ...) body...)   ; Parallel binden
(let* ((x 1) (y (+ x 1)) ...)     ; Sequentiell binden (Stdlib)

(macrolet ((m (p) body)) ...)     ; Lokale Makros (nicht-rekursiv)
(symbol-macrolet ((s exp)) ...)   ; Lokale Symbol-Makros
(flet ((f (p) body)) ...)         ; Lokale Funktionen (nicht-rekursiv)
(labels ((f (p) body)) ...)       ; Lokale Funktionen (rekursiv, gegenseitig)

(eval-when (situation) body...)   ; Situationssteuerung
(progv (syms) (vals) body...)     ; Dynamisches Binding (⚠️ siehe §9)
```

### Kontrollfluss

```lisp
(if cond then)                    ; Ein-Zweig
(if cond then else)              ; Zwei-Zweige

(cond
  (test1 result1)
  (test2 result2)
  (else default))                 ; else oder t = Default-Fall

(case key
  ((a b) 1)                       ; Mehrfach-Match
  ((c)   2)
  (else  3))                      ; → 1 für key=a oder key=b

(when test body...)               ; → nil wenn test falsy
(unless test body...)             ; → nil wenn test truthy

(and a b c)                       ; Kurzschluss, erster Falsch-Wert
(or a b c)                        ; Kurzschluss, erster Wahr-Wert
(not x)                           ; Negation

(begin expr1 expr2 expr3)         ; Sequenz, letzter Wert
(progn expr1 expr2)               ; Alias für begin
(prog1 first rest...)             ; Liefert first
(prog2 first second rest...)      ; Liefert second

(while test body...)              ; Schleife
(do ((i 0 (+ i 1)) (s 0 (+ s i))) ; Scheme-Iteration
    ((= i 5) s)                   ; → 10 (0+1+2+3+4)
  body...)

(do* ((i 0 (+ i 1))              ; Wie do, aber sequentiell (let*-Semantik)
      (j (+ i 1) (+ j 1)))
    ((= i 5) j))

(dotimes (var n) body...)         ; Zählschleife (Stdlib)
(dolist (var lst) body...)        ; Listeniteration (Stdlib)

(block name body...)              ; Named-Block (lexikalisch)
(return-from name value)          ; Nicht-lokaler Ausstieg
(return value)                    ; = (return-from nil value)

(tagbody                          ; Sprung-Marken
  start
  (set! x (+ x 1))
  (if (< x 10) (go start))       ; Sprung zu start
  x)

(catch 'tag body...)              ; Dynamic catch (tag wird EVALUIERT)
(throw 'tag value)                ; Dynamic throw
(trap expr (lambda (e) ...))      ; Einfacher catch, e = "msg" (String)
(unwind-protect                   ; Cleanup IMMER, auch bei Fehler
  protected-expr
  cleanup-expr1 cleanup-expr2)

(eval form)                       ; Globales Eval (in Env.Root())
(load "file.lisp")                ; Datei laden (⚠️ siehe §9)
```

### Multiple Values

```lisp
(values 1 2 3)                    ; Liefert nur ersten Wert weiter!
(multiple-value-list (values 1 2 3))  ; → (1 2 3)
(multiple-value-bind (a b c) (values 1 2 3)
  (+ a b c))                      ; → 6
(multiple-value-call #'+ (values 1 2)) (values 3 4)  ; → 10
(multiple-value-prog1 (values 1 2) (print "side"))     ; → 1, dann side-effect
(multiple-value-setq (a b) (values 1 2))               ; a=1, b=2
(nth-value 1 (values 10 20 30))   ; → 20 (0-basiert)
```

### Deklarationen und Introspektion

```lisp
(the type form)                   ; Typ-Deklaration (Typ wird ignoriert!)
(declare (type int x))            ; Deklaration (No-op)
(macroexpand form)                ; Makro expandieren
(macroexpand-all form)            ; Komplett expandieren
(bound? sym)                      ; → t wenn gebunden, sonst nil
(makunbound sym)                  ; Bindung entfernen
(function fn)                     ; Function-Literal (#' reader-sugar)
(exec "shell-cmd")                ; Shell-Kommando ausführen
(parfunc expr :timeout 5)         ; Parallel-Eval mit Timeout
```

### Quasiquote

```lisp
`(a b c)                          ; Reines Quote
`(a ,x c)                         ; Unquote
`(a ,@xs c)                       ; Unquote-Splice
```

---

## 3. Primitiven

GoLisp2 hat **~100 eingebauten Funktionen** (Type `FUNC`), registriert in
`BaseEnv()` (`lib/primitives.go`).

### Arithmetik
```lisp
(+ 1 2 3)                         ; → 6
(- 10 3 2)                        ; → 5
(* 2 3 4)                         ; → 24
(/ 10 3)                          ; → 3.333...
(mod 10 3)                        ; → 1
(remainder 10 3)                  ; = mod
(abs -5)                          ; → 5
(floor 3.7)                       ; → 3
(random)                          ; → Zufallszahl [0,1)
(values 1 2 3)                    ; Multi-Values
```

### Vergleiche
```lisp
(= 5 5)                           ; Zahlen-Vergleich
(< 3 5) (> 5 3) (>= 5 5) (<= 3 5)
(equal? "a" "a")                  ; Strukturell (für Zahlen, Listen, Strings)
(eq 'foo 'foo)                    ; Pointer-Identität (⚠️ für Zahlen oft nil)
(eq? (list) (list))               ; t (Singleton-Nil)
```

### Listen (klassische 7)
```lisp
(cons 'a '(b c))                  ; → (a b c)
(car '(a b c))                    ; → a
(cdr '(a b c))                    ; → (b c)
(list 1 2 3)                      ; → (1 2 3)
(append '(1 2) '(3 4))            ; → (1 2 3 4)
(atom 'foo)                       ; → t
(null '())                        ; → t
```

### Typ-Prädikate
```lisp
(string? "a") (number? 42) (list? '(1))
(symbol? 'foo) (atom? 'foo) (null? '())
```

### Symbol-Konstruktion
```lisp
(gensym)                          ; → G000001
(gensym "P")                      ; → P000002
(intern "FOO")                    ; → Symbol foo (downcased)
(symbol-name 'foo)                ; → "foo"
(symbol->string 'foo)             ; → "foo"
```

### Ausgabe
```lisp
(print "hallo")                   ; unreadable, kein newline
(println "hallo")                 ; + newline
(read)                            ; von stdin lesen
(warn "Achtung")                  ; nach stderr
```

### Fehler und Apply
```lisp
(error "Nachricht")               ; wirft Fehler (nur String)
(apply + '(1 2 3))               ; → 6
(funcall + 1 2 3)                 ; → 6
(mapcar #'car '((1 2) (3 4)))     ; → (1 3) — Primitiv, first-class:
(funcall mapcar #'car '((1 2)))   ; → (1)   ✓ funcall/apply möglich
(exit 0)                          ; Prozess sofort beenden mit Code
                                  ; (kein Cleanup — Vorsicht im SWANK-Daemon!)
```

### Zeit/Memory
```lisp
(sleep 1000)                      ; Milliseconds
(memstats)                        ; Go-Runtime-Stats
```

### sigoREST (KI-Anbindung)
```lisp
(sigo "prompt")                   ; KI-Call
(sigo-models)                     ; Verfügbare Modelle
(sigo-host)                       ; Aktueller Host
```

### Goroutinen und Concurrency
```lisp
(define c (chan-make 10))         ; Channel erstellen
(chan-send c "msg")               ; Senden
(chan-recv c)                     ; Empfangen
(define l (lock-make))            ; Mutex
```

### Shared Memory
```lisp
(shm-alloc 1024)                  ; Allokieren
(shm-write handle "data")         ; Schreiben
(shm-read handle)                 ; Lesen
(shm-status handle)               ; Status
(shm-free handle)                 ; Freigeben
(shm-cleanup)                     ; Alles aufräumen
```

### File I/O
```lisp
(file-write "f.txt" "inhalt")     ; Schreiben
(file-append "f.txt" "mehr")      ; Anhängen
(file-read "f.txt")               ; Lesen
(file-exists? "f.txt")            ; → t ()
(file-delete "f.txt")             ; Löschen
(set-working-directories "a" "b") ; Suchpfade
(get-working-directories)         ; → Liste
(get-file-path "f.txt")           ; Absoluter Pfad
```

### Shell
```lisp
(system "ls -la")                 ; Shell-Kommando
(file-stat "f.txt")               ; Datei-Info
```

### Strings
```lisp
(string-length "abc")             ; → 3
(string-append "a" "b" "c")       ; → "abc"
(substring "abc" 0 2)             ; → "ab"
(string-upcase "abc")             ; → "ABC"
(string-downcase "ABC")           ; → "abc"
(string->number "42")             ; → 42
(number->string 42)               ; → "42"
(string->list "abc")              ; → ("a" "b" "c")
(list->string '("a" "b" "c"))     ; → "abc"
(string-replace "abc" "b" "X")    ; → "aXc"
(string-trim "  abc  ")           ; → "abc"
(string-contains "abc" "b")       ; → t
```

### Hashtable (CL-kompatibel)
```lisp
(define h (make-hash-table))
(puthash 'key "val" h)
(gethash 'key h)                  ; → "val"
(remhash 'key h)
(clrhash h)
(hash-table-count h)              ; → Anzahl
(hash-table-p h)                  ; → t
(maphash (lambda (k v) (println k v)) h)
```

### FORMAT (CL-HyperSpec 22.3)
```lisp
(format nil "~A ~D" "x" 42)      ; → "x 42"
(format t "Hallo ~A" "Welt")      ; → stdout
(format nil "~{~A~^, ~}" '(1 2 3)); → "1, 2, 3"
(format nil "~,2F" 3.14159)       ; → "3.14"
```

Direktiven: `~A ~S ~D ~B ~O ~X ~R ~P ~C ~F ~E ~G ~$ ~% ~& ~| ~T ~* ~? ~[ ~{ ~( ~; ~^ ~/fun/ ~~`

### PostgreSQL
```lisp
(define conn (pg-connect "conninfo"))
(pg-query conn "SELECT 1")
(pg-exec conn "INSERT ...")
(pg-close conn)
```

### Genetischer Algorithmus
```lisp
(define ga (ga-create 'bit1 5 4 (lambda (g) (apply + g))))
(ga-init ga)
(ga-calc ga)
(ga-result ga)
(ga-cross ga)
(ga-select ga)
(ga-mut ga)
(ga-print ga)
(ga? ga)
```

### Redefine-Guard
```lisp
(redefine-policy)                 ; → aktuelle Policy (allow/warn/error)
(redefine-policy 'warn)           ; Setzen
(redef-log)                       ; Ringpuffer (256 Events)
(redef-log-clear)                 ; Leeren
(defined-in 'foo)                 ; → (file . line)
```

### Trace
```lisp
(trace '+)                        ; Aktivieren
(+ 1 2 3)                         ; stderr: (+ 1 2 3) => 6
(trace? '+)                       ; → t
(untrace '+)                      ; Deaktivieren
```

---

## 4. Stdlib (embed/stdlib.lisp)

**~50 Definitionen**, bei Start automatisch geladen.

### Accessoren
```lisp
(first '(a b c))                  ; → a  = car
(second '(a b c))                 ; → b  = cadr
(third '(a b c))                  ; → c  = caddr
(fourth '(a b c d))               ; → d  = cadddr
(rest '(a b c))                   ; → (b c) = cdr
(cadr x) (caddr x) (cadddr x) (cddr x) (cdar x) (caar x)
```

### Prädikate
```lisp
(zero? 0) (positive? 5) (negative? -3)
(pair? '(a b))                    ; → t
```

### Higher-Order / Funktional
```lisp
(reverse '(1 2 3))                ; → (3 2 1)
(length '(a b c))                 ; → 3
(nth 1 '(a b c))                  ; → b
(last '(a b c))                   ; → (c)
(member 'b '(a b c))              ; → (b c)
(assoc 'key '((a 1) (key 2)))      ; → (key 2)
(filter number? '(1 a 2 b 3))      ; → (1 2 3)
(drop 2 '(1 2 3 4))               ; → (3 4)
(take 2 '(1 2 3 4))               ; → (1 2)
(reduce + '(1 2 3 4 5))           ; → 15
(for-each println '(1 2 3))
(any number? '(a b 1))            ; → t
(every number? '(1 2 3))          ; → t
(flatten '(1 (2 (3) 4)))          ; → (1 2 3 4)
(zip '(a b) '(1 2))               ; → ((a 1) (b 2))
(list-tail '(1 2 3) 1)            ; → (2 3)
(iota 5)                          ; → (0 1 2 3 4)
(max 3 1 4 1 5)                   ; → 5
(min 3 1 4 1 5)                   ; → 1
(square 5)                        ; → 25
(expt 2 10)                       ; → 1024
(gcd 12 8)                        ; → 4
```

### Funktor-Muster
```lisp
(identity 5)                      ; → 5
(constantly 42)                   ; → (lambda (args) 42)
(complement odd?)                 ; → even?
(compose car cdr)                 ; → cadr
```

### Listen-Helfer
```lisp
(alist-set 'key 'val '())         ; → ((key val))
(alist-get 'key '((a 1) (key 2))) ; → 2
(union '(1 2 3) '(3 4 5))         ; → (1 2 3 4 5)
(set-difference '(1 2 3 4) '(2 4)) ; → (1 3)
(find-all 2 '(1 2 3 2))           ; → (2 2)
```

### Makros
```lisp
(when test body...)
(unless test body...)
(let* ((x 1) (y (+ x 1))) body...)

(dotimes (i 10)
  (println i))

(dolist (x '(1 2 3))
  (println x))

(push 1 var)                      ; var = (cons 1 var)
(pop var)                         ; car + set!

(defvar v 1)                      ; Definiert, wenn nicht gebunden
(defvar v 2)                      ; Erstzugriff gewinnt (idempotent)

(setf (car x) 42)                 ; Generisch, defstruct registriert automatisch
```

### Strukturen
```lisp
(defstruct pt (x 0) (y 0))
; Erzeugt:
; (make-pt :x 1 :y 2)            ; → (pt 1 2)
; (pt-x (make-pt :x 7))          ; → 7
; (pt-y (make-pt))               ; → 0
; (pt? (make-pt))                ; → t
; (pt? '(not-a-pt))              ; → ()

(setf (pt-x p) 9)                 ; Update via setf
```

---

## 5. Wahrheitswerte und Gleichheit

### Wahrheitswerte
```lisp
()  nil  NIL                       ; Falsch (Singleton-Nil, Pointer-Identisch)
t                                  ; Wahr
42  'foo  "string"                 ; Alles andere: wahr
```

### Gleichheit
```lisp
(eq 'foo 'foo)                     ; → ()! Zwei verschiedene Atom-Instanzen
(eq (list) (list))                 ; → t (Singleton-Nil, identischer Pointer)
(eq 5 5)                           ; → ()! eq auf Zahlen ist immer () (Design, s. 10.6)

(equal? 'foo 'foo)                 ; → t
(equal? (list 1 2) (list 1 2))     ; → t
(equal? 5 5)                       ; → t
(equal? "a" "a")                   ; → t
```

**Empfehlung:** Im Zweifel `equal?`. `eq` nur für Singleton-Objekte (nil) oder
wenn Pointer-Identität *explizit* gemeint ist.

---

## 6. Makros und Quasiquote

### Makro-Definition
```lisp
(defmacro when (test . body)
  `(if ,test (progn ,@body) nil))
```

### Macroexpand
```lisp
(macroexpand '(when t 1 2))        ; → (if t (progn 1 2) nil)
(macroexpand-all '`(a ,(+ 1 1)))   ; → (quasiquote (a 2))
```

### Quasiquote
```lisp
(setq x 42)
`(a b c)                           ; → (a b c)
`(a ,x c)                          ; → (a 42 c)
`(a ,@(list 1 2) c)               ; → (a 1 2 c)
```

### Makro-Schichten
```lisp
(macrolet ((when (tst . body) ...)) ...)   ; Lokal, nicht-rekursiv
(symbol-macrolet ((x 42)) ...)             ; Symbol-Makros
```

---

## 7. Fehlerhandling

### Fehler werfen
```lisp
(error "Datei nicht gefunden: ~a" path)
```

### Fehler fangen
```lisp
; trap — einfach, Handler bekommt die Fehlermeldung als String
(trap (error "oops") (lambda (e) (println e)))
; → "oops"

; catch/throw — dynamisch, nicht-lokal
(catch 'done
  (dolist (f files)
    (if (bad? f) (throw 'done (cons 'error f))))
  'ok)

; unwind-protect — Cleanup immer
(unwind-protect
  (with-handle h (process h))
  (close-handle h))
```

### Block/Return-From
```lisp
(block search
  (dolist (x data)
    (when (match? x) (return-from x)))
  'not-found)
```

---

## 8. Closures, TCO, Environment

### Closures
```lisp
(defun make-adder (n)
  (lambda (x) (+ x n)))

(define add5 (make-adder 5))
(add5 10)                          ; → 15
```

### TCO (Tail-Call Optimization)
```lisp
; Tail-Position: kein neuer Stack-Frame
(defun fact-trampoline (n acc)
  (if (= n 0) acc (fact-trampoline (- n 1) (* acc n))))

(fact-trampoline 100000 1)         ; O(1) Stack
```

### Environment
- Lexikalische Scopes über `Env`-Kette (parent-Verweis)
- `define` im Root → global
- `define` in `let`/`lambda` → lokal (Frame-Env)
- `(eval form)` → **immer in `Env.Root()`**, nie im Lambda-Scope

### Redefine-Policy
```lisp
(redefine-policy 'error)
(define car 42)                    ; Fehler: REDEF: car (war FUNC)

(redefine-policy 'warn)            ; Default
(defun f (x) x)
(defun f (x) (+ x 1))              ; OK: Lambda, still

(redefine 'allow)
```

---

## 9. Gotchas und CL-Abweichungen

| Fall | GoLisp2-Verhalten | CL-Verhalten |
|------|-------------------|--------------|
| `(eq 5 5)` | `()` — `eq` auf Zahlen immer `()` (Design) | Oft `t` (Small-Int-Cache) |
| `load` in `defun` | Lokal gebunden | Global |
| `progv` | Lex/dyn-Trennung fehlt | Dynamisch, lexikalische Bindungen schützen |
| `declare` | No-op | Type-Checks, Optimierungen |
| `the` | Typ ignoriert | Type-Checks zur Laufzeit |
| `(macrolet ...)` | Nicht-rekursiv | Rekursiv |
| `(eval form)` | Global (ok) | Global — identisch |
| `(eq 'foo 'foo)` | `()` (zwei Instanzen) | `t` (interned) |

### Wichtigster Gotcha: load in defun
```lisp
; FALSCH: foo wird nur lokal gebunden
(defun use-foo ()
  (load "foo.lisp")
  (foo))

; RICHTIG: eval erzwingt globales Env
(defun use-foo ()
  (eval '(load "foo.lisp"))
  (foo))
```

---

## 10. Schwächen (bewusst, eigener Abschnitt)

Diese Einschränkungen sind **Design-Entscheidungen**, keine Bugs. Sie sind
bewusst so, weil GoLisp2 einen anderen Fokus hat (Interpreter für KI-Workflows,
kein CL-Port).

### 10.1 Kein Package-System

- Alle Symbole in einem globalen Namespace.
- Kein `defpackage`, `in-package`, `export`, `import`.
- Kollisionen nur durch Namenskonvention vermeiden (Präfixe: `ga-`, `shm-`, `pg-`).

**Auswirkung:** Bei großen Projekten mit mehreren Entwicklern/Modulen
können Namenskonflikte entstehen. Workaround: Prefix-Konvention.

### 10.2 Kein CLOS (Common Lisp Object System)

- Nur `defstruct` — erzeugt Constructor (`make-pt`), Accessoren (`pt-x`), Prädikat (`pt?`).
- **Nicht vorhanden:** Klassen, Multi-Methoden, Method-Combination, `defmethod`, `defgeneric`, `defclass`, `call-method`.

**Auswirkung:** Polymorphismus nur über manuelle Dispatch-Muster (z. B. `case` auf Typ-Tag) möglich.

### 10.3 Kein Condition-System

- `error` liefert nur einen String, kein Objekt mit Slots.
- **Nicht vorhanden:** `define-condition`, `handler-case`, `handler-bind`, `restart-case`, `restart-bind`.

**Auswirkung:**
- Fehlerarten können nicht programmatisch unterschieden werden (z. B. "Datei nicht gefunden" vs "Parse-Fehler").
- Recovery-Mechanismen (Retry, Use-Value, etc.) nur manuell über `catch`/`throw` mit Strings.

**Workaround heute:**
```lisp
; Fehler-Tag als String kodieren
(error "FILE-NOT-FOUND: path=~a" path)

(trap (risky-op)
  (lambda (e)
    (if (string-contains e "FILE-NOT-FOUND")
        (use-default)
        (error e))))
```

### 10.4 Keine Lex/Dyn-Trennung bei progv

- `progv` bindet wie `let` — lexikalische Shadowings sehen den progv-Wert.
- In CL schützt eine lexikalische Bindung vor progv.

**Auswirkung:** Dynamische Variablen können unerwartet durch lexikalische
Bindungen überdeckt werden.

### 10.5 Kein Compile-File

- Reiner Interpreter. Kein `compile-file`, kein Laden von FASLs.
- `eval-when` hat keine Compiler-Situationen.

**Auswirkung:** Kein statisches Kompiliergeschwindigkeits-Boost. Alles läuft
interpretiert.

### 10.6 eq auf Zahlen liefert immer ()

- `(eq 5 5)` → `()`. `(eq 1000 1000)` → `()`. Auch bei identischem Wert.
- Intern existiert ein Small-Int-Cache (-32768..32767, `MakeNum` in
  `lib/types.go`) zur Allokations-Vermeidung — `eq` behandelt Zahlen trotzdem
  bewusst als nie identisch (`fnEqPtr` in `lib/primitives.go`).

**Auswirkung:** Immer `equal?` oder `=` für Zahlenvergleich verwenden, nie `eq`.

### 10.7 Makros nicht-rekursiv (macrolet)

- `macrolet`-Bodies sehen nicht die anderen Makros derselben Ebene.
- `labels` für Funktionen ist rekursiv — Asymmetrie zu CL.

**Auswirkung:** Verschachtelte Makro-Definitionen in `macrolet` können sich nicht
gegenseitig rekursiv aufrufen.

### 10.8 Kein Continuations, kein MOP

- Kein `call/cc`.
- `catch`/`throw` vorhanden (Abschnitt 2), aber ohne Restart-Semantik.
- Kein CLOS Meta-Object-Protocol.

**Auswirkung:** Fortgeschrittene Kontrollfluss-Muster (Backtracking, Coroutinen)
nur über Goroutinen emulierbar.

### 10.9 load in defun bindet lokal

- Workaround: `(eval '(load "file"))` für globales Laden.

**Auswirkung:** Unerwartetes Verhalten, wenn man `load` in einer Funktion nutzen
will, um globale Definitionen zu laden.

### 10.10 Kein GC-Feinsteuerung

- `memstats` liefert Go-Runtime-Stats.
- Kein `tweak`, `make-hash-table` ohne Weak-Refs.

**Auswirkung:** Keine Weak-Pointer, keine Ephemeral GC-Tuning.

### 10.11 Kein Typ-System

- `declare` und `the` sind No-ops.
- **Nicht vorhanden:** `check-type`, `typecase`, `ctypecase`, `etypecase`.

**Auswirkung:** Type-Checks nur manuell implementierbar.

### 10.12 Kein LOOP, kein Series, kein Iterate

- Iteration nur über `dolist`, `dotimes`, `do`, `do*`, `mapcar`, `reduce`.
- **Nicht vorhanden:** `loop`-Makro, `series`, `iterate`.

**Auswirkung:** Komplexe Iterationen müssen über Rekursion oder `reduce`
ausgedrückt werden.

---

## 11. Schnell-Lookup-Tabelle

| Ich will... | Nutze |
|-------------|-------|
| Zwei Zahlen addieren | `(+ 1 2)` |
| Liste durchlaufen | `(dolist (x xs) ...)`, `(mapcar f xs)` |
| Parallel auswerten | `(parfunc expr :timeout 5)` |
| Fehler werfen | `(error "msg")` |
| Fehler fangen | `(trap expr (lambda (e) ...))` |
| Dynamisch springen | `(catch 'tag ... (throw 'tag val))` |
| Struktur definieren | `(defstruct name (slot default) ...)` |
| String bauen | `(string-append "a" "b")` |
| Datei lesen/schreiben | `(file-read "f")`, `(file-write "f" "content")` |
| SQL ausführen | `(pg-query conn "SELECT ...")` |
| KI aufrufen | `(sigo "prompt")` |
| Formatierte Ausgabe | `(format nil "~A ~D" "x" 42)` |
| Makro schreiben | `(defmacro m (p) body...)` |
| Global definieren | `(define x 42)`, `(defun f () ...)` |
| Lokal binden | `(let ((x 1)) ...)` |
| Iterieren (CL-style) | `(do ((i 0 (+ i 1))) ((= i 10) result) body...)` |
| Debug-Trace | `(trace 'f)`, `(untrace 'f)` |

---

**Ende Cheatsheet.** KI-Version: `doc/ki/referenz.md`.
