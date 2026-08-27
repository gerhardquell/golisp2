# GoLisp2 – Lisp-Semantik

Referenz für die Stellen, an denen GoLisp2 eine bewusste Entscheidung getroffen
hat. Kompass bei Unklarheiten ist die Common-Lisp-Semantik.

---

## `eq` vs. `equal?`

| Funktion | Vergleicht | Verwendung |
|----------|------------|------------|
| `eq` / `eq?` | Pointer-Identität (dasselbe Objekt im Speicher) | Symbole, Singleton-Objekte, schnelle Identitätsprüfung |
| `equal?` | Strukturelle Gleichheit (rekursiver Inhaltsvergleich) | Listen, Strings, Zahlen, verschiedene Atom-Instanzen |

```lisp
;; eq prüft, ob es DASSELBE Objekt ist
(eq 'foo 'foo)                 ; t   – Symbole sind interniert (eine Cell pro Name)
(eq (list) (list))             ; t   – Singleton-Nil, identischer Pointer
(eq 5 5)                       ; ()  – Zahlen bewusst nie identisch (s. u.)
(eq "a" "a")                   ; ()  – Strings werden nicht interniert

;; equal? prüft, ob der Inhalt gleich ist
(equal? 'foo 'foo)             ; t
(equal? (list 1 2) (list 1 2)) ; t
(equal? 5 5)                   ; t
```

**Empfehlung:** Im Zweifel `equal?`. `eq` nur, wenn Identität *gemeint* ist.

---

## `let` vs. `let*`

| Form | Bindungsmodus | Verwendung |
|------|---------------|------------|
| `let` | parallel – alle Werte werden im äußeren Env ausgewertet | unabhängige Variablen |
| `let*` | sequentiell – jede Bindung sieht die vorherigen | abhängige Variablen |

```lisp
;; let – parallele Bindungen
(let ((x 5)
      (y (+ x 1)))     ; Fehler: x ist hier noch nicht gebunden
  ...)

;; let* – sequentielle Bindungen
(let* ((x 5)
       (y (+ x 1)))    ; OK: x ist bereits 5
  y)                   ; → 6
```

---

## `setq` und `setq*`

`setq` ist ein Alias für `define` — setzt eine Variable global oder lokal:

```lisp
(setq x 10)            ; → x
x                      ; → 10
```

`setq*` setzt mehrere Variablen **sequentiell** (analog `let*`):

```lisp
(setq* a 1
       b (+ a 1)       ; b sieht a = 1
       c (+ b 1))      ; c sieht b = 2
(list a b c)           ; → (1 2 3)
```

---

## `setf` und Places

`setf` (Stdlib-Makro) schreibt an eine *Place* — generalisierte Variable.
Unterstützte Places:

```lisp
(setf x 42)                        ; Variable
(define l (list 1 2))
(setf (car l) 9)                   ; → l = (9 2)
(defstruct pt (x 0))
(define p (make-pt :x 1))
(setf (pt-x p) 9)                  ; Struct-Slot (von defstruct registriert)
(define h (make-hash-table))
(setf (gethash 'k h) 7)            ; Hash-Eintrag (Expansion: puthash)
```

`incf`/`decf` sind setf-Kurzformen (`(incf x)` = `(setf x (+ x 1))`).
Neue Places registriert `defstruct` automatisch — manuell geht es nicht.

---

## Multiple Values

`(values a b c)` produziert mehrere Werte; `floor` ist eine MV-liefernde
Primitive. Die Regel „Nicht-MV-Kontexte sehen nur den Primärwert" lebt
genau einmal in `Primary()` (`src/lib/types.go`):

```lisp
(+ 1 (values 2 3))                  ; → 3   (nur Primärwert 2)
(multiple-value-list (values 1 2 3)); → (1 2 3)
(multiple-value-list (floor 7 2))   ; → (3 1)
(nth-value 1 (values 10 20))        ; → 20  (0-basiert)
(multiple-value-bind (a b) (values 1 2)
  (+ a b))                          ; → 3
(multiple-value-setq (a b) (values 1 2))  ; a=1, b=2
```

Wie in CL sehen `apply`/`funcall` nur den Primärwert — auf alle Werte kommt
man über `multiple-value-call`:
`(multiple-value-call #'+ (values 1 2) (values 3 4))` → `10`.

---

## Hash-Tables: `gethash` liefert zwei Werte

`gethash` ist MV-liefernd: `(values wert gefunden?)`. Das unterscheidet
„Key fehlt" von „Key ist auf () gebunden":

```lisp
(define h (make-hash-table))
(puthash 'k h 5)              ; Achtung: (puthash key TABELLE wert) —
                              ; Tabelle ist das ZWEITE Argument
(gethash 'k h)                ; → 5 (nur Primärwert sichtbar)
(multiple-value-list (gethash 'k h))   ; → (5 t)
(multiple-value-list (gethash 'x h))   ; → (() ())
(remhash 'k h)
(clrhash h)
```

`puthash` ist die setf-Expansion von `(gethash key h)`.

---

## `case` – syntaktischer Zucker für `cond`

```lisp
(case 'b
  ((a)   1)            ; einzelner Wert
  ((b c) 2)            ; Liste von Werten
  (else  3))           ; → 2

(case 5
  ((1 2 3) "klein")
  ((4 5 6) "mittel")
  (else    "groß"))    ; → "mittel"
```

Der Vergleich erfolgt mit `equal?` (strukturelle Gleichheit).
`else` oder `t` als Test ist der Default-Fall.

---

## Fehlermodell: `catch`/`throw` vs. `trap`

Zwei getrennte Mechanismen, leicht zu verwechseln:

- **`catch`/`throw`** (`(catch tag body…)` / `(throw tag wert)`) —
  CL-Semantik: dynamischer, nicht-lokaler Sprung zum nächsten `catch` mit
  gleichem Tag. Fängt **ausschließlich** `throw`-Sentinels. Ein Fehler
  (`(error …)`, Go-Primitive-Fehler) läuft **durch** ein `catch` hindurch —
  es ist kein Error-Handler. Fehlt ein passendes `catch`, ist `throw` ein
  Laufzeitfehler.
- **`trap`** (`(trap body handler)`) — projekteigene Fehlerbehandlung, kein
  CL-Konstrukt. Wertet `body` aus; bei `LispError` oder Go-Primitive-Fehler
  wird `handler` mit der Fehlermeldung (String-Cell) aufgerufen. Kontrollfluss-
  Sentinels (`throw`, `return-from`, `parfunc`-Signale) reicht `trap`
  unverändert durch — es fängt nur echte Fehler.

Kurz: **Fehler → `trap`. Nicht-lokaler Sprung ohne Fehler → `catch`/`throw`.**
Quelle: `src/lib/eval_control.go` (`evalCatch`, `evalThrow`, `evalTrap`).

---

## `(eval form)` – globales Environment

`(eval form)` wertet im **globalen** Environment aus (`Env.Root()`), nicht im
dynamischen Lambda-Scope. Das ist Common-Lisp-Semantik und für GoLisp2
essenziell: Definitionen aus `(eval (read …))` — REPL, `swank-repl:listener-eval`,
das selbsterweiternde Muster — müssen global sichtbar bleiben und dürfen nicht
im Child-Env der aufrufenden Lambda-Kette verschwinden.

---

## `defun` / `lambda` / `defmacro` mit mehreren Body-Ausdrücken

Erlaubt. `wrapBegin(exprs)` wrappt sie zur **Definitionszeit** in `(begin …)`.
Ein einzelner Ausdruck bleibt unverpackt — kein Overhead.

---

## `redefine-policy`

Steuert das Überschreiben von Go-Primitiven (`Type == FUNC`) im Root-Env.

```lisp
(redefine-policy)           ; → aktuelle Policy als Atom (allow / warn / error)
(redefine-policy 'allow)    ; still durchlassen
(redefine-policy 'warn)     ; Default: Meldung nach stderr, Überschreiben erlaubt
(redefine-policy 'error)    ; Fehler zurückgeben, altes Binding bleibt
```

Nur **existierende Bindungen im Root-Env**, deren alter Wert `FUNC` ist, werden
geschützt. Eigene `defun`s/Lambdas und lokales Shadowing (z. B. Lambda-
Parameter) werden nicht geschützt.

```lisp
(redefine-policy 'error)
(define car 42)             ; Fehler: REDEF: car (war FUNC)

(defun f (x) x)
(defun f (x) (+ x 1))       ; OK: Lambda, kein FUNC

((lambda (car) car) 7)      ; OK: Frame-Env, lokales Shadowing
```

Die Warnung `REDEF: <name> (war FUNC)` geht immer nach **stderr**, nie nach
stdout.

---

## FORMAT

`format` folgt Common-Lisp-HyperSpec 22.3.
Implementierung: `src/lib/format.go`, `format_dirs.go`, `format_blocks.go`.

**Direktiven:**
`~A ~S ~D ~B ~O ~X ~R ~P ~C ~F ~E ~G ~$ ~% ~& ~| ~T ~* ~? ~[ ~{ ~( ~; ~^ ~/fun/ ~~ ~Newline`

Mit Parametern und den Modifiern `:` und `@`.

**Destination:**
- `t` → stdout
- `nil` → Rückgabe als String
- String → anhängen

**Besonderheiten:**
- `~/name/` ruft eine benannte Funktion via `globalFormatEnv`
- `~:;` markiert die Default-Klausel

---

## trace / untrace

Live-Tracing einzelner Root-Env-Funktionen.

```lisp
(trace '+)
(+ 1 2 3)
; stderr: (+ 1 2 3)
; stderr: (+ 1 2 3) => 6

(defun f (n)
  (if (= n 0) 0 (+ 1 (f (- n 1)))))
(trace 'f)
(f 2)
; stderr: (f 2)
; stderr:   (f 1)
; stderr:     (f 0)
; stderr:     (f 0) => 0
; stderr:   (f 1) => 1
; stderr: (f 2) => 2

(untrace 'f)              ; f wiederherstellen
(untrace)                 ; alle getraceden Funktionen wiederherstellen
(trace? 'f)               ; t | ()
```

API:

- `(trace 'name)` — wrappt die aktuelle Root-Env-Bindung. Erlaubt für
  eingebaute Primitiven (`FUNC`) und Lambdas (`LAMBDA`). `name` zurückgeben.
- `(untrace 'name)` — stellt das Original wieder her; `nil` wenn `name` nicht
  getraced war.
- `(untrace)` — stellt alle getraceden Funktionen wieder her und gibt eine
  sortierte Liste ihrer Namen zurück.
- `(trace? 'name)` — `t` wenn getraced, sonst `()`.

Einschränkungen:

- Tracing wirkt nur auf **Root-Env-Bindungen**. Lokale `let`/`lambda`-Shadowing
  oder Frame-Envs werden nicht verfolgt.
- **Makros** (`MACRO`) können nicht getraced werden, weil ein `FUNC`-Wrapper
  die Makro-Expansion nicht korrekt re-evaluiert.
- Getracede Tail-Calls sind nicht mehr TCO-optimiert: der Wrapper fügt einen
  zusätzlichen Go-Stackframe hinzu.
- Trace-Ausgaben gehen immer nach **stderr**.

---

## Rekursionstiefe und `parfunc`

- `eval` bricht ab, wenn die nicht-tail-rekursive Tiefe `MaxEvalDepth` (Default 100000) überschreitet. Ergebnis ist ein `LispError`, kein Prozessabbruch.
- `parfunc` kennt **keine** Keywords: `(parfunc erg e1 e2 ...)` wertet *alle*
  weiteren Formen als parallele Zweige aus — ein `:timeout` wird als Zweig
  evaluiert, nicht als Option. Absicherung gegen Endlos-Zweige: `MaxEvalDepth`.

---

## Redefinition, Redef-Log, `makunbound`

Das Root-Env bewacht Redefinitionen über `(redefine-policy 'allow|'warn|'error)`
(Default: `warn`):

- **FUNC** (Go-Primitiv) überschreiben → immer Policy, egal aus welcher Quelle.
- **LAMBDA/MACRO** (Lisp-Definition) überschreiben → Policy nur bei *fremder*
  Quelle. Reload derselben Datei (gleiches `SrcFile`, interaktiv = `""`) ist
  still — das ist der normale Entwicklungs-Workflow.
- Alle Redefinitionen landen im Ringpuffer (256 Events), abfragbar via
  `(redef-log)` — Event-Format:
  `(name old-kind new-kind old-file old-line new-file new-line action)`.
  `(redef-log-clear)` leert.
- `(makunbound 'sym)` entfernt eine Root-Bindung samt DefLoc-Eintrag.
  Fehler bei ungebundenem Symbol; auf FUNC/LAMBDA/MACRO greift die Policy.

Bewusste Grenzen: `setq`/`progv` am Root über LAMBDA bleiben still (kein
Quell-Kontext). FUNC-Log-Events kennen die neue Quelle nicht (`NewFile ""`).

## Generische Funktionen (CLOS-light)

`defgeneric`/`defmethod` (stdlib.lisp, TODO 20260813 Punkt 2.5):
Single-Dispatch auf den Struct-Tag (`car`) des ersten Arguments.

- Erster Eintrag der Methoden-Lambda-Liste ist `(var tag)` — `var` wird
  im Body gebunden, `tag` ist der Struct-Tag oder `t` (Default/Fallback).
- Dispatch ruft die Methode mit **allen** Originalargumenten auf —
  Extra-Parameter hinter dem Spec-Paar möglich:
  `(defmethod skaliere ((x kreis) faktor) ...)`.
- Ohne passenden Tag und ohne `t`-Methode: Fehler `keine Methode für …`.
- Methoden sind Lambdas in einer Registry-Hashtabelle
  (`%generic-registry`); `defmethod` desselben Tags überschreibt still
  (Hot-Patching per SWANK).
- Explizit nicht dabei: Vererbung, `call-next-method`, `:before`/`:after`,
  Multi-Dispatch. Implementierung rein Lisp, kein Kernel-Eingriff.
