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

`setq` setzt eine bestehende Variable entlang der Env-Kette und liefert den
Wert (CL):

```lisp
(define x 1)
(setq x 10)            ; → 10
```

Ist die Variable ungebunden, wird sie wie in SBCL **global** angelegt, mit
Warnung auf stderr — auch aus einer Funktion heraus. Gilt gleich für
`setq*`, `setf` (Symbol-Place) und `multiple-value-setq` (`assignVar` in
`eval_specialforms.go`). `set!` bleibt strikt: Fehler bei ungebunden.

```lisp
(defun f () (setq neu 5))
(f)                    ; stderr: WARN: undefinierte Variable neu – global angelegt
neu                    ; → 5
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

## Funktionsbezeichner: `'f` wie `#'f`

Wo eine Funktion erwartet wird (`funcall`, `apply`, `mapcar`, `sort`,
`reduce`, `filter` …), darf wie in CL auch ein gequotetes Symbol stehen.
Es wird **global** aufgelöst (Root-Env), nicht lexikalisch:
`(let ((car cdr)) (mapcar 'car '((1 2))))` → `(1)`. Dasselbe gilt für den
Aufruf einer Variablen, deren Wert ein Symbol ist: `(let ((f '+)) (f 1 2))`
→ `3` — so rufen in Lisp geschriebene Funktionen höherer Ordnung ihr
Argument auf. Unbekannte Symbole und Makros bleiben ein Fehler.
Eine Quelle: `resolveFn` (`src/lib/eval_core.go`).

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

## Zahlen drucken

Eine Druckform für Printer, `~a` und `number->string` (`FormatNumber`,
`src/lib/types.go`): ganzzahlige Werte ohne Punkt (`3.0` → `3`), sonst die
kürzesten Ziffern — Festkomma für 1e-4 ≤ |x| < 1e16, darüber und darunter
Exponent (Regel wie Python-`repr`). `(now)` → `1792441863.9679012`,
`0.00001` → `1e-05`. Eigene Darstellung: `(format nil "~,3f" x)`.

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
- `~,0:F` (golisp-Erweiterung): wie `~,0F`, aber ohne Punkt am Ende —
  `(format nil "~,0:f" 1.5)` → `"2"` statt `"2."`. Dafür übersetzt
  `printf "%.0f"`, damit es C-treu bleibt.

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

### defclass — Objekte mit Identität

`defstruct` hat Wertsemantik: `(setf (punkt-x p) 9)` bindet `p` an eine
neue Liste, andere Referenzen sehen den alten Wert. Für geteilten,
veränderlichen Zustand: `(defclass schiff (name (energie 100)))`.

- Objekt = eql-Hashtabelle mit `:class` → `schiff`; `eq` ist Identität.
- `make-schiff` (&key), `schiff?`, `schiff-SLOT`, `set-schiff-SLOT`.
  Accessoren auf fremden Objekten → Fehler (nicht still `nil`).
- `setf`/`incf`/`decf` ändern in place über `*setf-mutators*`
  (`register-setf-mutator`) — der Place darf ein Ausdruck sein:
  `(setf (schiff-energie (car flotte)) 5)`.
- `defmethod` dispatcht über `(gethash :class obj)` wie über den Struct-Tag.
- `type-of`/`typep`/`typecase` kennen den Klassennamen (`*class-types*`),
  Druckform `#<schiff>`. Siehe „Typen“.
- Jede Tabelle hat ein eigenes Lock — Einzelzugriffe sind sicher,
  Lesen-Ändern-Schreiben aus mehreren Goroutinen nicht: `(lock m ...)`.
- Keine Vererbung. Syntax bewusst nicht CL (`(defclass n (supers) (slots))`).

---

## `defvar` vs. `defparameter`

Beide binden eine globale Variable (CL-Compat, Makros in stdlib.lisp).
Der Unterschied liegt im Rebind-Verhalten:

| Form | bereits gebunden? | Verhalten |
|------|-------------------|-----------|
| `defvar` | ja | **no-op** — alter Wert bleibt |
| `defparameter` | ja | **überschreibt immer** |

```lisp
(defvar *x* 1)
(defvar *x* 2)         ; no-op, *x* bleibt 1

(defparameter *y* 1)
(defparameter *y* 2)   ; *y* ist jetzt 2
```

Optionaler Docstring (3. Argument) wird akzeptiert und ignoriert.
Wert-Rebinds lösen keinen Redefine-Warn aus (der Guard greift nur bei
FUNC/LAMBDA/MACRO), landen aber im `redef-log`.

---

## Typen: `type-of` und `typep`

```lisp
(type-of 3)                        ; → integer
(type-of 3.5)                      ; → float
(type-of :k)                       ; → keyword
(typep 3 'number)                  ; → t   (integer → rational → real → number)
(typep x '(or string symbol))
(typep n '(integer 0 (10)))        ; 0 ≤ n < 10
(typep x '(satisfies gerade?))     ; eigenes Prädikat
(typecase x                        ; Verzweigung nach Typ, key einmal ausgewertet
  (integer 'ganz)
  ((or string symbol) 'text)
  (otherwise 'sonst))              ; t geht auch; ohne Treffer → ()
(etypecase x (integer 'ganz))      ; ohne Treffer → Fehler
(check-type x (integer 0 *))       ; → () oder Fehler "check-type: x ist -1, nicht vom Typ …"
(subtypep 'integer 'number)        ; → t, t   (zwei Werte: Untertyp? sicher?)
(subtypep 'integer '(integer 0 9)) ; → (), () (nicht entscheidbar)
(deftype ziffer () '(integer 0 9)) ; eigener Typname
(deftype bis (&optional (n '*))    ; mit Parametern: (typep x '(bis 5))
  (list 'integer 0 n))
```

Hierarchie und Typnamen stehen an genau einer Stelle: `*type-parents*`
in `src/embed/types.lisp`. Structs (`defstruct`) und Conditions
(`define-condition`) sind eigene Typen; ihre Namen kollidieren laut
(`WARN:`), nie still — der eingebaute Typ gewinnt vor Struct, Struct vor
Condition.

**Bewusste Abweichungen von CL:**
- `3.0` ist `integer` — der Kern kennt nur `float64`.
- `type-of` liefert `integer`/`float`, nicht `fixnum`/`double-float`.
- Structs sind Listen: `(typep p 'cons)` → `t`. Ein Literal `'(punkt 1 2)`
  gilt als `punkt`, wenn `punkt` registriert ist und die Länge passt.
- defclass-Objekte sind Hashtabellen: `(type-of s)` → `schiff`,
  `(typep s 'hash-table)` und `(typep s 'atom)` → `t`, `(subtypep 'schiff
  'hash-table)` → `t t`. Druckform `#<schiff>` (nur der Name — Objekte
  dürfen zyklisch sein). Eine Tabelle mit `:class` → unregistriertes Symbol
  bleibt `hash-table` für `type-of`, druckt sich aber als `#<symbol>`.
- Channels sind `compiled-function`; `macro` ist ein golisp2-eigener Typ.
- Keine Characters. Vektoren nur eindimensional, ohne Fill-Pointer. Das
  Literal `#(…)` wertet seine Elemente nicht aus und liefert bei jedem
  Lesen einen neuen Vektor; Vektoren sind selbstauswertend. `equal?`
  vergleicht Vektoren nach Identität (CL `equal`). `sort` nimmt Vektoren
  (neuer Vektor, nicht-destruktiv); `mapcar`/`apply`/`append` sind wie in
  CL nur für Listen und werfen bei Vektoren oder Punktpaaren einen Fehler.
- Sequenzfunktionen (`elt subseq copy-seq concatenate map some find
  find-if position position-if count count-if`) nehmen Listen und Vektoren,
  **keine Strings** (keine Characters). Nur `:test`/`:key`, kein
  `:start`/`:end`/`:from-end`. Default-Test `eql` (Go-Primitive; Zahlen nach
  Wert, sonst Identität) — Strings mit `:test 'equal?` suchen.
- `deftype`: `&optional` ohne Default ist `()`, nicht `*` — `(&optional (n '*))`
  schreiben. Namen eingebauter Typen, Structs und Conditions sind ein Fehler
  (der deftype wäre nie erreichbar); `defstruct`/`define-condition` über einen
  deftype-Namen warnt, der Struct bzw. die Condition gewinnt. Ein deftype,
  der sich selbst enthält, läuft endlos (wie in CL).
- `etypecase`/`check-type` werfen einfachen `error` (fangbar als `error`/
  `lisp-error`), kein `type-error`; `check-type` ohne `store-value`-Restart.

**Nicht vorhanden / keine Typen:**
- Alist/Plist sind keine Typen (auch in CL nicht) → `(satisfies …)`.
- `subtypep` entscheidet eingebaute Typen, `atom`, Structs, Conditions,
  `deftype`, `or`/`and`/`member`/`eql` und Zahlbereiche; `not`, `satisfies`
  und Bereichsfälle, die nicht sicher passen, liefern `() ()`.
- Noch nicht vorhanden: `ctypecase`.

---

## `loop` — Iteration (CL-Praxis-Kern)

`loop` (embed/loop.lisp) ist ein reines Lisp-Makro: die Klauseln werden
zur Expansionszeit geparst und in `let` + `while` übersetzt — keine
Laufzeitkosten gegenüber handgeschriebener Schleife.

**Klauseln:**

```lisp
(loop for i from 1 to 5 collect i)            ; → (1 2 3 4 5)
(loop for i from 0 below 4 by 2 collect i)    ; → (0 2)   below = exklusiv
(loop for i from 5 downto 1 collect i)        ; → (5 4 3 2 1)
(loop for x in '(a b c) collect x)            ; → (a b c)
(loop repeat 3 do (tick))
(loop for i from 1 to 6 when (= 0 (mod i 2)) sum i)   ; → 12
(loop for x in '(3 9 1) maximize x)           ; → 9
(loop for i from 1 to 10 until (> i 3) collect i)     ; → (1 2 3)
(loop for i from 1 to 3 do (log i) finally (fertig))
```

- Iteration: `repeat`, `for … from [to|below|downto] [by]`, `for … in`
- Bedingungen: `when`/`unless` (je eine Folgeklausel), `while`/`until`
- Rumpf: `do`, `finally`
- Akkumulation: `collect`, `append`, `sum`, `count`, `maximize`, `minimize`

**Einschränkungen gegenüber CL (bewusst):**

- Eine Akkumulator-Familie pro loop: `collect`/`append` **oder**
  `sum`/`count` **oder** `maximize`/`minimize`. Familienwechsel ist ein
  Fehler zur Expansionszeit (in CL ebenfalls ein Fehler).
- `when`/`unless` ohne `and`-Kettung und ohne `else`.
- Kein Destructuring in `for`, kein `across`,
  kein `initially`, kein `= / then`-Schritt, kein `named`.
- Syntaxfehler melden sich zur Expansionszeit mit Klausel-Angabe.
