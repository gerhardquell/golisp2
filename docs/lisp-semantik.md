# GoLisp2 – Lisp-Semantik

Referenz für die Stellen, an denen GoLisp2 eine bewusste Entscheidung getroffen
hat. Kompass bei Unklarheiten ist die Common-Lisp-Semantik.

---

## `eq` vs. `equal?`

| Funktion | Vergleicht | Verwendung |
|----------|------------|------------|
| `eq` / `eq?` | Pointer-Identität (dasselbe Objekt im Speicher) | Symbole, Singleton-Objekte, schnelle Identitätsprüfung |
| `equal?` | Strukturelle Gleichheit (rekursiver Inhaltsvergleich) | Listen, Strings, Zahlen gleicher Zahlart (`(equal? 1 1.0)` → `()`) |

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
  wird `handler` mit der Fehlermeldung (String-Cell) aufgerufen. Hat der Handler
  genau zwei Parameter — `(lambda (e ort) ...)` —, bekommt er als zweiten den Fehlerort
  samt Aufrufkette als Text, so wie die CLI ihn unter `ERR:` druckt (`docs/cli.md`).
  `handler-case` gibt den Ort nicht weiter. Kontrollfluss-
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
im Child-Env der aufrufenden Lambda-Kette verschwinden. Dynamische Bindungen
des Aufrufers sieht `eval` trotzdem (CL).

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

## Zahlen

Es gibt zwei Zahlarten: **Ganzzahl** (`int64`) und **Float** (`float64`).
`1` und `1.0` sind verschiedene Werte. Keine Bignums, keine Brüche.

**Lesen:** Nur Vorzeichen und Ziffern ergeben eine Ganzzahl (`42`, `-7`, `+3`).
Alles andere ist ein Float (`3.14`, `1e3`, `1.`). Eine Ganzzahl außerhalb von
`int64` ist ein Lesefehler.

**Drucken:** Ganzzahl ohne Punkt (`3`). Ein Float ist immer als Float
erkennbar: `3.0`, `1000.0`. Sonst die kürzesten Ziffern — Festkomma für
1e-4 ≤ |x| < 1e16, darüber und darunter Exponent (`1e+16`, `1e-05`).
Das gilt für Printer, `~a` und `number->string` (`numberStr`/`FormatNumber`,
`src/lib/types.go`). Eigene Darstellung: `(format nil "~,3f" x)`.

**Rechnen:**

| Operation | Regel | Beispiel |
|---|---|---|
| `+ - *`, `1+ 1-` | alle Ganzzahl → Ganzzahl; ein Float dabei → Float | `(+ 1 2)` → `3`, `(* 1.5 2)` → `3.0` |
| `/` | alle Ganzzahl und teilbar → Ganzzahl, sonst Float | `(/ 6 2)` → `3`, `(/ 7 2)` → `3.5` |
| `mod rem remainder` | beide Ganzzahl → Ganzzahl, sonst Float | `(mod 7.5 2)` → `1.5` |
| `abs max min` | die Zahlart bleibt erhalten | `(max 1 2.5)` → `2.5`, `(min 1 2.5)` → `1` |
| `floor ceiling truncate round` | Ergebnis immer Ganzzahl | `(floor 3.7)` → `3` |
| `sqrt exp log` | immer Float | `(sqrt 16)` → `4.0` |
| `expt` | Ganzzahl hoch Ganzzahl ≥ 0 → Ganzzahl, sonst Float | `(expt 2 10)` → `1024`, `(expt 2 -1)` → `0.5` |
| `random` | `(random 10)` → Ganzzahl, `(random 1.0)` und `(random)` → Float in [0, 1) | |
| `ash` | nur Ganzzahlen | `(ash 1 10)` → `1024` |

**Überlauf ist ein Fehler**, kein stiller Wechsel zu Float:
`(* 99999999999 99999999999)` → `*: Ganzzahl-Überlauf`. Ebenso
`(+ 9223372036854775807 1)`, `(expt 2 64)`, `(floor 1e30)`.

**Vergleichen:**

- `= /= < > <= >=` vergleichen nach Wert: `(= 1 1.0)` → `t`.
- `eql` und `equal?` verlangen gleiche Zahlart **und** gleichen Wert:
  `(eql 1 1.0)` → `()`. Dem folgen `member`, `position`, `assoc`, `case`
  und die Tags von `catch`/`throw`.
- Hash-Tabellen: `1` und `1.0` sind zwei Schlüssel.
- Ganzzahl gegen Ganzzahl ist exakt. Bei Ganzzahl gegen Float wird die
  Ganzzahl nach Float gewandelt. Über 2^53 ist das ungenau:
  `(= 9007199254740993 9007199254740992.0)` → `t`.

**Umwandeln und prüfen:** `(float 3)` → `3.0`. `floor`/`round`/`truncate`/
`ceiling` machen aus einem Float eine Ganzzahl. `integerp` und `floatp`
prüfen die Zahlart. `parse-int` liefert eine Ganzzahl, `parse-float` einen
Float, `string->number` folgt der Leseregel (`"42"` → `42`, `"42.0"` → `42.0`).

**Indizes und Anzahlen** müssen Ganzzahlen sein (`nth`, `aref`, `substring`,
`iota`, `dotimes`, `loop repeat` …). Ein Float ist ein Fehler, auch `2.0`:
`(nth 2.0 lst)` → `nth: ganze Zahl erwartet, got 2.0`. Abhilfe:
`(nth (floor x) lst)`. Zählende Funktionen (`length`, `position` …) liefern
Ganzzahlen.

**JSON:** Eine Zahl ohne Punkt und Exponent wird eine Ganzzahl, sonst ein
Float; liegt sie außerhalb von `int64`, wird sie ein Float. Ganzzahlen werden
exakt geschrieben. Ein Float mit ganzzahligem Wert wird als `3.0` geschrieben
und kommt als Float zurück.

**Sonst:** `(now)` ist ein Float. Postgres liefert `int2/int4/int8` als
Ganzzahl und nimmt Ganzzahl-Parameter als `int64`.

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
- `~D ~B ~O ~X` drucken Ganzzahlen exakt. Ein Float geht den `~A`-Weg, auch
  `3.0`: `(format nil "~d" 3.0)` → `"3.0"`. `printf` bleibt C-treu: `%d`, `%x`
  und `%o` drucken einen Float mit ganzzahligem Wert als Ganzzahl
  (`(sprintf "%d" 3.0)` → `"3"`), einen echten Bruch wie `~A` (`3.7`).
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

## Lambda-Listen

Ein Bindungsweg für alle Aufrufe (`bindArgs` in `src/lib/eval_lambda.go`; der direkte Aufruf
wertet nur die führenden regulären Parameter ohne Zwischen-Slice aus und übergibt den Rest):

- Reihenfolge wie CL: regulär, `&optional`, `&rest`/`&body`, `&key`, `&allow-other-keys`;
  Dotted-Rest `(a b . rest)` geht ebenfalls.
- Default-Ausdrücke sehen die früheren Parameter: `(a &optional (b a))`.
- `&key` liest die Argumente paarweise aus den **Werten** (`(f k 1)` mit `k` = `:a` bindet `a`),
  das erste Vorkommen gewinnt. Fehler (seit 0.29.0, vorher still ignoriert): unbekanntes
  Schlüsselwort ohne `&allow-other-keys`, Schlüssel ohne Wert, Nicht-Schlüsselwort an
  Schlüsselposition. `&rest` vor `&key` sieht dieselben Argumente.
- `:allow-other-keys t` im Aufruf wird nicht unterstützt — dafür `&allow-other-keys` in der
  Definition.

## Rekursionstiefe und `parfunc`

- `eval` bricht ab, wenn die nicht-tail-rekursive Tiefe `MaxEvalDepth` (Default 100000) überschreitet. Ergebnis ist ein `LispError`, kein Prozessabbruch.
- `(parfunc erg [:timeout N] e1 e2 ...)` wertet die Formen als parallele Zweige aus.
  `:timeout N` (Sekunden, Zahl-Literal direkt nach dem Namen) bricht wartende Zweige ab,
  an ihrer Stelle steht `()`. Scheitert ein Zweig, steht dort `(:error "text")` (seit 0.25.1,
  vorher still `()`); die anderen laufen durch.

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

Bewusste Grenzen: `setq` am Root über LAMBDA bleibt still (kein
Quell-Kontext); `progv` schreibt nicht ins Root-Env. FUNC-Log-Events kennen die neue Quelle nicht (`NewFile ""`).

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
  (`*generic-registry*`); `defmethod` desselben Tags überschreibt still
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

Beide binden eine globale Variable (CL-Compat, Makros in stdlib.lisp) und
machen den Namen special (siehe „Dynamische Variablen“).
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

## Dynamische Variablen (seit 0.33.0)

`defvar` und `defparameter` erklären den Namen für **special** (CL), auch
wenn er schon gebunden ist und `defvar` den Wert stehen lässt. Das gilt
global und für immer; `makunbound` entfernt den Wert, nicht die Eigenschaft.
`define` und `setq` auf Ungebundenes machen nichts special. Ein Name ohne
`*…*` ist ein **Fehler** — sonst würde jedes `let` und jedes stdlib-Lambda
`(lambda (n) …)` auf ihn dynamisch bzw. scheitern:

```
ERR: defvar: n ohne *…* — special-Namen brauchen *n*
```

Kein Teil-Effekt: Nach dem Fehler ist `n` weder gebunden noch special.
Ebenso wird der Name nicht special, wenn das globale Binden scheitert
(z. B. `(redefine-policy 'error)` auf eine FUNC-Bindung).

`defparameter` innerhalb einer dynamischen Bindung desselben Namens setzt
die innerste Bindung (CL `symbol-value`), nicht den globalen Wert:
`(let ((*x* 2)) (defparameter *x* 9) (g))` → `9`, danach `*x*` → `1`.
`makunbound` auf einen dynamisch gebundenen Namen ist ein Fehler
(`makunbound: *x* ist dynamisch gebunden`).

**Binden:** `let`, `let*` und `progv` binden special-Namen dynamisch, für
die Dauer des Rumpfs. Alles, was in dieser Zeit läuft, sieht die Bindung —
aufgerufene Funktionen, Callbacks aus Primitiven (`mapcar`, `sort`,
`reduce`, `maphash`, …), `eval` und `load`. Danach gilt wieder der vorige
Wert, auch nach `error`, `throw` oder `return-from`. Ein `let` darf
special- und normale Namen mischen. Makros, die zu `let` expandieren
(`dolist`, `dotimes`, `destructuring-bind`, `loop … for`), binden dadurch
ebenfalls dynamisch.

```lisp
(defvar *x* 1)
(defun g () *x*)
(let ((*x* 2)) (g))              ; → 2
(progv '(*x*) '(3) (g))          ; → 3
(g)                              ; → 1
```

**Lesen:** lexikalische Frames, dann die innerste dynamische Bindung, dann
der globale Wert. Eine lexikalische Bindung schützt also vor `progv`:
`(let ((x 2)) (progv '(x) '(1) x))` → `2`.

**Zuweisen:** `setq`, `set!`, `psetq` und `setf` auf einen special-Namen
treffen die innerste dynamische Bindung, ohne Bindung den globalen Wert.
Nach dem `let` ist die Zuweisung weg:
`(let ((*x* 2)) (setq *x* 5)) *x*` → `1`.

**Closures merken sich die Bindung nicht.** Nach dem `let` sehen sie den
dann gültigen Wert:
`(defun mach () (let ((*x* 2)) (lambda () *x*)))  (funcall (mach))` → `1`.

**Binder-Fehler:** Lambda-Parameter (`defun`, `lambda`, `defmacro`,
`flet`/`labels`, `&optional`/`&key`/`&rest`), `do`/`do*` und
`multiple-value-bind` binden lexikalisch und weisen special-Namen ab:
`defun: Parameter *x* ist special — mit let binden`. Lambda-Listen werden
**bei der Definition** geprüft, nicht beim Aufruf; `do`/`do*` und
`multiple-value-bind` prüfen alle Variablen, bevor etwas ausgewertet wird.
Grenze: Wird das Lambda vor dem `defvar` definiert, bindet sein Parameter
weiter lexikalisch — und ein `let` auf diesen Namen im Rumpf ebenfalls, so
verhält sich der alte Code durchgehend wie vorher (`progv` bindet immer
dynamisch):

```lisp
(defun lies () *z*)
(defun alt (*z*) (let ((*z* 3)) (list *z* (lies))))
(defvar *z* 1)
(alt 2)                          ; → (3 1)
```

**Goroutinen:** `parfunc`, `spawn`, `pool-map` und die Fitness-Worker von
`ga-calc` erben die Bindungen, die beim Start galten — dieselbe Bindung,
kein Schnappschuss: ein `setq` im Worker sieht auch der Starter. Ein `let`
im Worker gilt nur dort. HTTP-Handler (`http-serve`, `webserv`), WS-Bridge,
GUI-Callbacks und SWANK-Verbindungen beginnen ohne Bindungen und sehen den
globalen Wert.

Nicht vorhanden: `(declare (special x))`, `defconstant`, `symbol-value`/`set`.

---

## Sandbox (seit 0.34.0)

Eigene Root-Umgebung für KI-Code — zusätzlich zu `(eval (read (sigo …)))`,
das unverändert volle Rechte hat (`docs/sigo.md`).

```lisp
(make-sandbox &key zeitlimit erlaube)   ; → #<foreign sandbox>
(sandbox-eval sb form-oder-string)      ; → Wert der (letzten) Form

(define sb (make-sandbox))
(sandbox-eval sb "(defun a () 1) (defun b () 2) (+ (a) (b))")   ; → 3
(sandbox-eval sb '(system "ls"))
; ERR: sandbox: system nicht erlaubt (freigeben mit :erlaube)
```

- Eine **Form** wird in der Sandbox ausgewertet, ein **String** ganz
  gelesen (alle Formen der Reihe nach, Wert der letzten) — `read` nähme
  still nur die erste.
- Definitionen (`defun`, `define`, `defmacro` …) bleiben in der Sandbox und
  gelten für spätere Aufrufe derselben Sandbox. Zwei Sandboxen sehen
  einander nicht, global ist nichts davon sichtbar. `eval` in der Sandbox
  läuft in der Sandbox-Root.
- Fehler kommen als normaler Lisp-Fehler beim Aufrufer an (`trap`,
  `handler-case`, `ignore-errors`), Text unverändert.

**Whitelist.** Erlaubt ist nur, was `src/lib/sandbox.go` ausdrücklich
freigibt (`sandboxGruppen`, `sandboxEinzeln`); jede andere Primitive aus
`BaseEnv()` — auch jede neue — ist gesperrt. Zwei Tests halten beide Listen
fest (`TestSandboxSperrliste`, `TestSandboxErlaubtListe`): Eine neue
Primitive fällt dort auf und muss bewusst eingeordnet werden. Die stdlib
(Lisp-Code) wird ganz geladen; ruft eine stdlib-Funktion etwas Gesperrtes,
scheitert sie beim Aufruf: `(shell-output "ls")` →
`sandbox: exec nicht erlaubt …`. Die Sperre greift auch über Umwege:
`(funcall 'system …)`, `apply`, `mapcar`, `(eval (list 'system …))`, ein
Makro, das zu `exec` expandiert.

**Erlaubt:** Grundmenge — Rechnen, Listen- und Kern-Primitiven, Strings,
Regex, Vektoren, Sequenzen, Hash-Tabellen, JSON, `type-of`/`typep`,
CL-Kompat (`sort` …), `format`, `sprintf`, Zeit (`now`, `format-time`,
`sleep`), `read`, `documentation`, `memstats`, `env-symbols` — und die
Standard-Freigaben:

| Gruppe | Namen |
|---|---|
| Ausgabe | `print`, `println`, `printf`, `format t`, `warn`, `err-write` |
| Goroutinen | `parfunc`, `spawn`, `chan-*`, `lock-make`, `lock` |
| sigo | `sigo`, `sigo*`, `sigo-request`, `sigo-models`, `sigo-model-info`, `sigo-reference`, `sigo-usage`, `sigo-costs`, `sigo-budget` |
| Datei lesen | `file-read`, `file-exists?`, `file-stat`, `directory-files`, `get-working-directory`, `get-file-path` |

**Gesperrt** (Stand 0.34.0, Wahrheit ist `TestSandboxSperrliste`):

| Gruppe | Namen |
|---|---|
| Spezialformen | `exec`, `load`, `defmain` |
| Shell | `system` |
| Datei schreiben | `file-write`, `file-append`, `file-delete`, `fprintf`, `set-working-directory` |
| Prozess | `exit`, `argv`, `getenv`, `environ`, `lisp-implementation-version` |
| stdin | `read-line`, `gets`, `slurp` |
| Redefinition, Trace | `redefine-policy`, `redef-log`, `redef-log-clear`, `trace`, `trace?`, `untrace` |
| sigo global | `sigo-host`, `sigo-system-prompt`, `sigo-usage-reset` |
| Sandbox | `make-sandbox`, `sandbox-eval` |
| Dienste | `pg-*`, `maxima-*`, `http-*`, `ws-*`, `webserv`, `ga-*`, `browser-open` |

**`:erlaube`** schaltet einzelne gesperrte Namen frei:
`(make-sandbox :erlaube '(file-write getenv))`. Ein unbekannter oder ohnehin
erlaubter Name ist ein Fehler — laut statt still:
`make-sandbox: car ist keine sperrbare Funktion`.

**Zeitlimit.** `:zeitlimit` in Sekunden, Zahl > 0, Standard 10. Es gilt pro
`sandbox-eval`-Aufruf, auch in Callbacks (`mapcar`, `sort`, `maphash`,
`format ~/fun/` …), in `parfunc`-Workern und in `spawn`-Goroutinen. Eine
`spawn`-Goroutine, die das `sandbox-eval` überlebt, endet mit der Frist.
Meldungen:

- direkt und in Callbacks: `sandbox: Zeitlimit 1 s überschritten`;
- `parfunc`-Worker: `(:error "eval: cancelled")` an seiner Stelle im Ergebnis;
- `spawn`: `spawn: eval: cancelled` auf stderr, im Channel
  `(:error "eval: cancelled")`.

Grenzen: Eine einzelne lange Go-Operation (`(sleep 3000)`, ein großes
`sort`, ein Regex, ein `sigo`-HTTP-Aufruf) wird nicht mittendrin
unterbrochen; die Frist greift beim nächsten Funktionsaufruf oder
Schleifendurchlauf. Kommt danach keiner mehr, liefert `sandbox-eval` den
Wert trotz Fristablauf. `sigo` hat seinen eigenen Timeout
(`GOLISP_SIGO_TIMEOUT`) und dauert oft länger als 10 s — KI-Code, der selbst
`sigo` ruft, braucht ein höheres `:zeitlimit`.

**Funktion aus der Sandbox.** `(sandbox-eval sb 'fib)` liefert die Funktion.
Außerhalb aufgerufen bleibt sie eingeschränkt (ihre Umgebung ist die
Sandbox), läuft aber **ohne Zeitlimit** — das gilt nur innerhalb von
`sandbox-eval`. Umgekehrt läuft eine Closure, die der Aufrufer selbst in die
Sandbox reicht, mit ihren vollen Rechten.

**Prozessweit bleibt:** Ein `defvar` in der Sandbox macht den Namen auch
außerhalb special. Docstrings (`documentation`) und Redefinitionen
(`(redef-log)`, `REDEF:`-Warnungen) der Sandbox landen in den globalen
Registern.

**Speicher:** Jede Sandbox kostet rund 1 MB (eigenes `BaseEnv` + stdlib) und
wird bis Prozessende nicht freigegeben. Eine Sandbox anlegen und
wiederverwenden, nicht eine pro Aufruf. Speicher- und CPU-Quoten gibt es nicht.

**Datei lesen ist erlaubt:** KI-Code kann damit Geheimnisse lesen
(`~/.netrc`, Schlüsseldateien). `getenv` bleibt gesperrt.

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
- `(typep 3.0 'integer)` → `()`: `integer` und `float` folgen der Zahlart,
  nicht dem Wert (siehe „Zahlen“).
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
  `:start`/`:end`/`:from-end`. Default-Test `eql` (Go-Primitive; Zahlen nach Zahlart und
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
