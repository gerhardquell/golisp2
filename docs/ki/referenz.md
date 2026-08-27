# GoLisp2 — KI-Kurzreferenz (tokenoptimiert)

> **Ziel:** Andere KIs brauchen diese Datei als *Initial-Context*, um GoLisp2-Code
> zu schreiben/verstehen, ohne `rg` über 50 Dateien zu werfen.
> **Format:** Tabellen, Präfixe, kein Fluff. Menschliche Ergänzung:
> `docs/golisp2-cheatsheet.md`.
> **Quelle:** `eval_core.go`, `lib/primitives.go`, `embed/stdlib.lisp` (Stand 20260730).

---

## 1. Eval-Reihenfolge (immer prüfen)

```
1. ATOM   → Env-Lookup (nil = Singleton-Nil)
2. LIST   → car prüfen:
   a. Spezialform (case in eval_core.go)  → direkt, keine Arg-Eval
   b. MACRO                              → expand, neu evaluieren
   c. FUNC/LAMBDA                        → Args eval, apply
```

Tail-Calls (`if`, `begin`, `let`, `lambda`, `case`, `cond`, `prog1/2`, `catch`,
`throw`, `tagbody/go`, ...) setzen `expr`/`env` und `continue` im Eval-Loop —
**kein neuer Stack-Frame**. Tiefe Rekursion O(1) Stack.

---

## 2. Spezialformen (55) + 2 Stdlib-Makros (`dotimes`, `dolist`)

| Form | Semantik | Anmerkung |
|------|----------|-----------|
| `(quote x)` | `x` unausgewertet | `'x` reader-sugar |
| `(if c t [e])` | Conditional | Tail-fähig |
| `(begin . body)` | Sequenz | Tail; Multi-Body via `wrapBegin` |
| `(let (bind) . body)` | Parallel-Bindung | `let*` sequentiell (Stdlib) |
| `(lambda (p) . body)` | Closure | `&optional`, `&key`, `&rest` |
| `(defun f (p) . body)` | Globale Funktion | Multi-Body via `wrapBegin` |
| `(defmacro m (p) . body)` | Globale Makro | |
| `(define sym val)` | Var-Def | Global oder lokal |
| `(set! sym val)` | Update | `setq` Alias |
| `(setq* s1 v1 s2 v2 ...)` | Sequentiell setzen | |
| `(psetq s1 v1 s2 v2 ...)` | Parallel setzen | |
| `(macrolet ((m . spec)) . body)` | Lokale Makro | Nicht-rekursiv |
| `(symbol-macrolet ((s expansion) . body)` | Symbol-Macro | |
| `(flet ((f . spec)) . body)` | Lokale Funktion | Nicht-rekursiv |
| `(labels ((f . spec)) . body)` | Lokale Funktion | Rekursiv (gegenseitig) |
| `(block name . body)` | Named-Block | Lexikalisch |
| `(return-from name [val])` | Nicht-lokaler Ausstieg | |
| `(return [val])` | `(return-from nil [val])` | |
| `(tagbody stmt ...)` | Sprung-Tags | Atome = Tags |
| `(go tag)` | Sprung | Lexikalisch, nicht evaluiert |
| `(catch tag . body)` | Dynamic-Catch | Tag wird EVALUIERT |
| `(throw tag val)` | Dynamic-Throw | |
| `(trap expr handler)` | Einfacher Catch | `(trap expr (lambda (e) ...))`, e = Msg-String |
| `(unwind-protect protected cleanup)` | Cleanup immer | |
| `(eval form)` | Globales Eval | Immer `Env.Root()` |
| `(load "file")` | Datei laden | **Achtung:** in `defun` → lokal gebunden! |
| `(progn . body)` | Sequenz | Tail |
| `(prog1 first . rest)` | Ersten Wert returnen | |
| `(prog2 a b . rest)` | b returnen | |
| `(cond (test result) ...)` | Conditional | `else`/`t` = Default |
| `(case key (vals result) ...)` | Structural dispatch | `equal?`-Vergleich |
| `(the type form)` | Deklaration | Typ **ignoriert** (kein Typsystem) |
| `(declare . decls)` | Deklaration | No-op |
| `(eval-when (situation) . body)` | Situationssteuerung | |
| `(progv syms vals . body)` | Dynamic binding | **Achtung:** lex/dyn-Trennung fehlt |
| `(and . exprs)` | Kurzschluss | |
| `(or . exprs)` | Kurzschluss | |
| `(not x)` | Negation | |
| `(parfunc expr . opts)` | Parallel-Eval | `:timeout N`, `:workers N` |
| `(while test . body)` | Schleife | |
| `(do ((var step) ...) (test result) . body)` | Scheme-Iteration | Parallel step |
| `(do* ((var step) ...) (test result) . body)` | Scheme-Iteration | Sequentiell step |
| `(dotimes (var n) . body)` | Zählschleife | Stdlib-Makro |
| `(dolist (var lst) . body)` | Listeniteration | Stdlib-Makro |
| `(multiple-value-list form)` | Werte→Liste | |
| `(multiple-value-bind (vars) form . body)` | Werte binden | |
| `(multiple-value-call fn . forms)` | Werte übergeben | |
| `(multiple-value-prog1 form . rest)` | Ersten Wert behalten | |
| `(multiple-value-setq vars form)` | Werte setzen | |
| `(nth-value n form)` | n-ter Wert | |
| `(function fn)` | Function-Literal | `#'` reader-sugar |
| `(macroexpand form)` | Makro expandieren | |
| `(macroexpand-all form)` | Komplett expandieren | |
| `(bound? sym)` | Gebunden? | |
| `(makunbound sym)` | Bindung entfernen | |
| `(exec shell-cmd)` | Shell-Kommando | |
| `(quasiquote x)` | Quasi-Quote | `` `x ``, `,x`=unquote,`,@x`=splice |

**Eval-Schlüsselwörter (Tail-Position):** `if`, `begin`, `let`, `cond`, `case`,
`prog1`, `prog2`, `catch`, `throw`, `tagbody/go`, `block/return-from`, `do`.

---

## 3. Wichtige Primitiven (FUNC, Go-seitig)

### Arithmetik/Vergleich
`+ - * / mod remainder abs floor random values`
`= < > >= <= equal? eq eq?`  — `eq`=Pointer, `equal?=strukturell`

### Listen (klassische 7)
`car cdr cons atom null list append`
`atom? null? string? number? list? symbol?` — Typ-Prädikate
`mapcar` — Primitiv (first-class: `funcall`/`apply` ok)

### Symbol/Atom
`gensym intern symbol-name symbol->string`

### Ausgabe
`print println read warn`

### Control/Errors
`error apply funcall exit`  — `error` liefert **nur String**, kein Condition-Objekt
`exit` — Prozess sofort beenden, Code als Zahl (kein Cleanup!)

### Environment/Introspection
`memstats sleep`

### Domänen (eigene Register-Xxx)
- **sigoREST:** `sigo sigo-models sigo-host`
- **Goroutinen:** `chan-make chan-send chan-recv lock-make`
- **Shared Memory:** `shm-alloc shm-free shm-write shm-read shm-status shm-cleanup`
- **File I/O:** `file-write file-append file-read file-exists? file-delete set-working-directory get-working-directory get-file-path gets slurp err-write printf sprintf fprintf sscanf argv getenv environ`
- **Shell:** `system file-stat shell-assoc`
- **Strings:** `string-length string-append substring string-upcase string-downcase string->number number->string string->list list->string string-replace string-trim string-contains`
- **Hashtable:** `make-hash-table gethash puthash remhash clrhash hash-table-count hash-table-p maphash`
- **FORMAT:** `format` — CL-HyperSpec 22.3, `~A ~S ~D ~B ~O ~X ~R ~P ~C ~F ~E ~G ~$ ~% ~& ~| ~T ~* ~? ~[ ~{ ~( ~; ~^ ~/fun/ ~~`
  - Rundung: half-to-even (Go-`strconv`), nicht half-up wie C — `%.2f` von `2.25` → `"2.2"`
- **PostgreSQL:** `pg-connect pg-query pg-exec pg-close`
- **GenAlg:** `ga-create ga-init ga-cross ga-calc ga-select ga-result ga-mut ga-print ga?`
- **Redefine:** `redefine-policy redef-log redef-log-clear defined-in`
- **Trace:** `trace untrace trace?`

---

## 4. Stdlib (embed/stdlib.lisp, ~50 Definitionen)

### Accessoren/Shortcuts
`cadr caddr cadddr cddr cdar caar first second third fourth rest`
`zero? positive? negative? pair?`

### Higher-Order/Functional
`identity constantly complement compose reverse length nth last member assoc filter drop take reduce for-each any every flatten zip list-tail iota max min square expt gcd`

### Listen-Helfer
`alist-set alist-get union set-difference find-all set-nth`

### Makros
`when unless let* dotimes dolist push pop defvar setf defstruct defgeneric defmethod`

### Iteratoren
`dotimes (var n) body` — `(dotimes (i 10) ...)`
`dolist (var lst) body` — `(dolist (x xs) ...)`

### Strukturen
`(defstruct name (slot default) ...)` — erzeugt: `make-name`, `name-slot`, `name?`
`(setf place val)` — generisch, `(defstruct ...)` registriert Accessoren automatisch

---

## 5. Wahrheitswerte / Nil

- `()` / `nil` / `NIL` → Singleton-Nil (Pointer-Identität!)
- `t` → Wahr, aber *nicht* der einzige wahre Wert — alles außer Nil ist wahr
- `(eq '() '())` → `t` (Singleton)
- `(eq 5 5)` → `()` — **Design:** `eq` auf Zahlen liefert immer `()` (siehe 10.6). Im Zweifel `equal?`

---

## 6. Quasiquote-Muster

```lisp
`(a b c)           ; reines quote
`(a ,x c)          ; unquote
`(a ,@xs c)        ; unquote-splice
```

---

## 7. Fehlerhandling

```lisp
; Fehler werfen
(error "Nachricht")           ; bricht, liefert nur String

; Fangen
(trap expr (lambda (e) ...))  ; e = "msg" (Message-String)

; Dynamisch
(catch 'tag body ...)
(throw 'tag value)
```

**Condition-lite** (`embed/condition.lisp`, automatisch geladen) —
strukturierte Fehler mit Typ-Hierarchie und Slots:

```lisp
(define-condition file-error (io-error) (path))  ; Typ + Eltern + Slots
(signal 'file-error :path "x.lisp")              ; wirft, unwindet immer!
(handler-case (load "x.lisp")
  (file-error (e) (file-error-path e))  ; Reader automatisch: typ-slot
  (io-error  (e) "irgendein io-fehler") ; matcht auch Subtypen
  (error     ()  "ohne Var-Bindung"))   ; Var darf () sein
```

- Basis-Hierarchie: `condition` → `error` → `lisp-error`
- **Go-Fehler** (file-read o. ä.) werden in `handler-case` zu `lisp-error`,
  Message via `(lisp-error-msg e)`
- Kein Match → **Re-Signal** an äußeren Handler
- Neudefinition eines Typs ersetzt still (Reload-Semantik)
- **CL-Abweichung:** `signal` unwindet immer (verhält sich wie CLs `error`,
  nicht wie CLs `signal`). Keine Restarts, kein `handler-bind`.
- Slot-Namen müssen über die Vererbungshierarchie hinweg eindeutig sein
  (flache plist, keine Verdeckung).

**Test-Framework:** `tests/test-framework.lisp` — `defsuite`, `deftest`
(`:suite`, `:expected-failure`), `is`, `run-tests` → FAIL-Anzahl.
Typisch: `(exit (run-tests))` → Exit-Code = FAILs.

---

## 8. Eval-Umgebung

- `(eval form)` → **immer `Env.Root()`**, nie Lambda-Scope
- `load` → **AUSNAHME:** in `defun`-Body → Bindung lokal, nicht global
  - Workaround: `(eval '(load "file"))` für globales Laden
- `redefine-policy` → `'allow` / `'warn` / `'error` (Default `warn`)

---

## 9. Gotchas / Abweichungen CL

| Fall | GoLisp2 | CL |
|------|---------|-----|
| `(eq 5 5)` | `()` — `eq` auf Zahlen immer `()` (Design) | oft `t` (Small-Int) |
| `load` in `defun` | Lokal gebunden | Global |
| `progv` | Lex/dyn-Trennung fehlt | Dynamisch |
| `declare` | No-op | Type-Checks |
| `the` | Typ ignoriert | Type-Checks |
| `(eval form)` | Global | Global (ok) |
| `macrolet` | Nicht-rekursiv | Rekursiv |

---

## 10. Schwächen (bewusst, eigener Abschnitt)

Diese Einschränkungen sind **Design-Entscheidungen**, keine Bugs. KI muss sie
kennen, um nicht zu raten:

### 10.1 Kein Package-System
- Alle Symbole in einem globalen Namespace.
- Kein `defpackage`, `in-package`, `export`, `import`.
- Kollisionen nur durch Namenskonvention vermeiden (Präfixe wie `ga-`, `shm-`).

### 10.2 Kein CLOS
- `defstruct` (Constructor, Accessoren, Prädikat).
- CLOS-light: `defgeneric`/`defmethod` — Single-Dispatch auf Struct-Tag, `t` = Default-Methode.
- Keine Klassen, keine Vererbung, kein `call-next-method`, keine Method-Combination.
- `defclass`, `call-method` — **nicht vorhanden**.

### 10.3 Nur Condition-lite, kein volles CL-Condition-System
- `define-condition`/`signal`/`handler-case` vorhanden (Hierarchie, Slots,
  Vererbungs-Dispatch, `lisp-error`-Fallback für Go-Fehler).
- **Aber:** `signal` unwindet immer (wie CLs `error`), keine Restarts,
  kein `handler-bind`, kein `restart-case`, kein MOP.
- Slot-Namen flach — keine Verdeckung über Vererbung hinweg.

### 10.4 Keine Lex/Dyn-Trennung bei progv
- `progv` bindet wie `let` — lexikalische Shadowings sehen den progv-Wert.
- CL: lexikalische Bindung schützt vor progv.

### 10.5 Kein Compile-File
- Reiner Interpreter. Kein `compile-file`, `load` von FASLs.
- Kein `eval-when`-Setup für Compiler/Kombilierzeit.

### 10.6 `eq` auf Zahlen liefert immer `()`
- `(eq 5 5)` → `()`. `(eq 1000 1000)` → `()`. Auch bei identischem Wert.
- Intern existiert ein Small-Int-Cache (-32768..32767, `MakeNum` in `lib/types.go`)
  zur Allokations-Vermeidung — `eq` behandelt Zahlen trotzdem bewusst als nie
  identisch (`fnEqPtr` in `lib/primitives.go`).
- Immer `equal?` oder `=` für Zahlenvergleich, nie `eq`.

### 10.7 Makros nicht-rekursiv (macrolet)
- `macrolet`-Bodies sehen nicht die anderen Makros derselben Ebene.
- `labels` für Funktionen ist rekursiv — Asymmetrie zu CL.

### 10.8 Keine Continuations, kein MOP
- Kein `call/cc`.
- `catch`/`throw` vorhanden (Abschnitt 2), aber ohne Restart-Semantik.
- Kein CLOS Meta-Object-Protocol.

### 10.9 `load` in `defun` bindet lokal
- Workaround: `(eval '(load "file"))` für globales Laden.
- Begründung: `eval` läuft in `Env.Root()`.

### 10.10 Kein GC-Feinsteuerung
- `memstats` liefert Go-Runtime-Stats.
- Kein `tweak`, `make-hash-table` ohne Weak-Refs.

### 10.11 Kein Typ-System
- `declare` und `the` sind No-ops.
- Kein `check-type`, `typecase`, `ctypecase`, `etypecase`.

### 10.12 Kein LOOP, kein Series, kein Iterate
- Iteration nur über `dolist`, `dotimes`, `do`, `do*`, `mapcar`, `reduce`.
- Kein `loop`-Makro, kein `series`, kein `iterate`.

---

## 11. Dateistruktur (KI-Scan)

```
lib/
  types*.go          Cell-Datenstruktur
  reader.go          Read / ReadAll
  env.go             Env (RWMutex, Pool)
  eval_core.go       Eval-Trampolin + Spezialformen-Dispatch
  eval_*.go          Spezialformen, Lambda, Control, Quasiquote
  primitives.go      BaseEnv() — alle Go-Primitiven
  *.go               goroutine, fileio, shellcmd, postgres, genalg, shm, sigorest
  stdlib.go          //go:embed stdlib.lisp
  format*.go         CL-HyperSpec 22.3
  trace.go           trace/untrace
main.go              CLI
```

---

## 12. Schnell-Lookup (KI-Cheatsheet)

| Brauche ich... | Nutze |
|----------------|-------|
| Arithmetik | `+ - * / mod` |
| Vergleich | `equal?` (Zahlen!), `eq` (Pointer) |
| Liste bauen | `cons list append` |
| Iteration | `dolist dotimes mapcar reduce` |
| Parallel | `parfunc` |
| Fehler | `error` + `trap` |
| Dynamisch | `catch/throw` |
| Struktur | `defstruct` |
| String | `string-append substring string-replace` |
| Datei | `file-read file-write` |
| DB | `pg-connect pg-query` |
| KI | `sigo` |
| Debug | `trace` |

---

**Ende KI-Referenz.** Menschliche Version: `docs/golisp2-cheatsheet.md`.
English: `docs/ki/referenz_en.md` · 中文: `docs/ki/referenz_cn.md`.
