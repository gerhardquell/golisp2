# Final-Review Fix-Wave — Report

Alle Findings (Important 1–2, Minor 3–8) aus final-findings.md behoben.
TDD: pro Code-Finding (1–6) zuerst ein RED-Test in
`src/lib/types_typep_test.go`, dann minimaler Fix in `src/embed/types.lisp`.
Finding 8 (fehlende Supertyp-Zeilen) war bereits grün — als Regression
festgehalten. Finding 7 (ki-referenz.md) live gegen den gebauten Binary
geprüft.

## Finding 1 — `%struct-instance?`: `length` crasht auf improper list

**Ursache:** `(length x)` (stdlib.lisp, unveränderbar) läuft rekursiv über
`car`/`cdr` und bricht mit `cdr: Liste erwartet` auf improper lists ab, z. B.
`(cons 'punkt 2)`.

**Fix:** neuer Helper `%proper-length` in `types.lisp` (rekursiv, liefert
`-1` für improper lists statt zu crashen); `%struct-instance?` benutzt ihn
statt `length`.

RED:
```
$ go test ./src/lib/ -run TestStructInstanceImproperList -count=1 -v
eval("(defstruct punkt x y) (type-of (cons 'punkt 2))") Fehler: cdr: Liste erwartet
```
GREEN:
```
$ go test ./src/lib/ -run TestStructInstanceImproperList -count=1 -v
--- PASS: TestStructInstanceImproperList
```
Live (`./build.sh` danach):
```
$ ./build/golisp2 -e "(begin (defstruct punkt x y) (println (type-of (cons 'punkt 2))))"
cons
$ ./build/golisp2 -e "(begin (defstruct punkt x y) (println (typep (cons 'punkt 2) 'cons)))"
t
```

## Finding 2 — `%typep-satisfies`: `(bound? f)` sieht eigene Locals

**Ursache:** `bound?` prüft lexikalisch im aktuellen Env. Innerhalb von
`%typep-satisfies` sind `x`, `spec`, `args`, `f` selbst lokale Bindungen —
`(typep 1 '(satisfies f))` findet also die lokale Variable `f` und hält sie
für "gebunden", `(eval f)` löst dann aber im Root-Env auf (Projekt-Invariante:
`eval` läuft immer in `Env.Root()`) und findet dort kein `f` →
`env: unbekanntes Symbol 'f'` statt des Spec-Texts.

**Fix:** `bound?`-Vorprüfung entfernt, stattdessen `(trap (eval f) (lambda (e) ()))`
— löst direkt im Root-Env auf und fängt "unbekanntes Symbol" ab, statt ihn
lexikalisch vorzufiltern.

RED:
```
$ go test ./src/lib/ -run TestTypepSatisfiesRootEnv -count=1 -v
eval(...) = "\"env: unbekanntes Symbol 'f'\"", want "\"typep: satisfies: 'f' ist keine Funktion\""
(ebenso für x, spec, args)
```
GREEN:
```
$ go test ./src/lib/ -run TestTypepSatisfiesRootEnv -count=1 -v
--- PASS: TestTypepSatisfiesRootEnv
```
Live:
```
$ ./build/golisp2 -e "(typep 1 '(satisfies f))"
ERR: "typep: satisfies: 'f' ist keine Funktion"
$ ./build/golisp2 -e "(typep 1 '(satisfies x))"
ERR: "typep: satisfies: 'x' ist keine Funktion"
$ ./build/golisp2 -e "(typep 1 '(satisfies spec))"
ERR: "typep: satisfies: 'spec' ist keine Funktion"
$ ./build/golisp2 -e "(typep 1 '(satisfies args))"
ERR: "typep: satisfies: 'args' ist keine Funktion"
```

## Finding 3 — `%number-type`: +Inf/-Inf gelten als `integer`

**Ursache:** `(= x (floor x))` ist für Inf/-Inf wahr (Floor von Inf ist Inf),
NaN war schon vorher korrekt (NaN ≠ NaN).

**Fix:** neuer Helper `%infinite?` — `x` ist unendlich, wenn `x ≠ 0` und
`x = 2x` (Verdopplung ändert einen endlichen Wert immer, außer bei 0;
Inf verdoppelt zu sich selbst). `%number-type` verlangt zusätzlich
`(not (%infinite? x))`.

RED:
```
$ go test ./src/lib/ -run TestNumberTypeInfinite -count=1 -v
eval("(type-of (* 1e308 10))") = "integer", want "float"
```
GREEN:
```
$ go test ./src/lib/ -run TestNumberTypeInfinite -count=1 -v
--- PASS: TestNumberTypeInfinite
```
Live:
```
$ ./build/golisp2 -e "(type-of (* 1e308 10))"
float
$ ./build/golisp2 -e "(type-of (- (* 1e308 10) (* 1e308 10)))"   ; NaN
float
```

## Finding 4 — `%symbol-type`: `(intern "")` crasht in `substring`

**Ursache:** `(substring (symbol-name x) 0 1)` verlangt Index `1` in einem
leeren String → `substring: Index außerhalb des Bereichs`.

**Fix:** Längenprüfung `(> (string-length (symbol-name x)) 0)` vor dem
`substring`-Aufruf.

RED:
```
$ go test ./src/lib/ -run TestSymbolTypeEmptyName -count=1 -v
eval("(type-of (intern \"\"))") Fehler: substring: Index außerhalb des Bereichs
```
GREEN:
```
$ go test ./src/lib/ -run TestSymbolTypeEmptyName -count=1 -v
--- PASS: TestSymbolTypeEmptyName
```
Live:
```
$ ./build/golisp2 -e '(type-of (intern ""))'
symbol
```

## Finding 5 — `%cons-type`: malformte `%condition`-Marker

**Ursache:** `(%cond? x)` prüft nur `(car x)`. Bei `'(%condition)` ist
`(cadr x)` dann `()` (aus dem echten `car` von `()`) statt eines
Typnamens → `%builtin-type?` verneint, `(cadr x)` = `()` wird fälschlich
als Typname zurückgegeben. Bei `(cons '%condition 2)` ist `(cdr x)` = `2`
(kein Paar), `(cadr x)` versucht `(car 2)` → `car: Liste erwartet`.

**Fix:** Guard `(pair? (cdr x)) (symbol? (cadr x))` vor dem
`%builtin-type?`-Check — bei Nichterfüllung fällt der Cons durch die
nächste Klausel (Struct-Check, ebenfalls sicher) bis `'cons`. `%cond?`
selbst (in `condition.lisp`) unverändert.

RED:
```
$ go test ./src/lib/ -run TestConsTypeMalformedCondition -count=1 -v
eval("(type-of '(%condition))") = "()", want "cons"
eval("(type-of (cons '%condition 2))") Fehler: car: Liste erwartet
```
GREEN:
```
$ go test ./src/lib/ -run TestConsTypeMalformedCondition -count=1 -v
--- PASS: TestConsTypeMalformedCondition
```
Live:
```
$ ./build/golisp2 -e "(type-of '(%condition))"
cons
$ ./build/golisp2 -e "(type-of (cons '%condition 2))"
cons
```

## Finding 6 — Dotted Typangaben crashen mit rohem car/cdr-Fehler

**Ursache:** `(or . x)`, `(member . 1)`, `(integer 0 . 5)` liefern für
`args` (`(cdr spec)`) ein Atom bzw. eine improper list; `any`/`every`/
`length` laufen darüber und crashen roh statt `typep: ungültige Typangabe …`
zu melden.

**Fix:** neuer Helper `%proper-list?`; `%typep-spec` prüft `args` darauf,
*bevor* dispatcht wird — bei improper list sofort `%bad-spec`.

RED:
```
$ go test ./src/lib/ -run TestTypepSpecDottedArgs -count=1 -v
eval("(trap (typep 1 '(or . x)) (lambda (e) e))") = "\"car: Liste erwartet\"", want "\"typep: ungültige Typangabe (or . x)\""
eval("(trap (typep 1 '(integer 0 . 5)) (lambda (e) e))") = "\"cdr: Liste erwartet\"", ...
```
GREEN:
```
$ go test ./src/lib/ -run TestTypepSpecDottedArgs -count=1 -v
--- PASS: TestTypepSpecDottedArgs
```
Live:
```
$ ./build/golisp2 -e "(typep 1 '(or . x))"
ERR: "typep: ungültige Typangabe (or . x)"
$ ./build/golisp2 -e "(typep 1 '(member . 1))"
ERR: "typep: ungültige Typangabe (member . 1)"
$ ./build/golisp2 -e "(typep 1 '(integer 0 . 5))"
ERR: "typep: ungültige Typangabe (integer 0 . 5)"
```

## Finding 7 — `ki-referenz.md`

- Zeile "Kein Float-Typ" (Gotchas-Tabelle, Abschnitt 9) war falsch —
  `(type-of 3.5)` liefert längst `float`. Umbenannt zu
  "Ganzzahlige Floats sind integer" (Zeile behält den korrekten
  `(type-of 3.0)` → `integer`-Beleg).
- Stdlib-Eintrag `type-of`/`typep` (Abschnitt "Typen") ergänzt: zusätzlich
  gültige Typnamen `rational real number atom list sequence
  structure-object t`; `fixnum character vector double-float
  single-float` → `typep: unbekannter Typ '…'`; `check-type`, `subtypep`,
  `typecase`, `etypecase`, `deftype` fehlen; Typnamen case-sensitiv
  (`'INTEGER` unbekannt, nur `'integer`).

Jede Behauptung live gegen `./build/golisp2` (nach `./build.sh`) geprüft:
```
$ ./build/golisp2 -e "(typep 3 'rational)"          → t
$ ./build/golisp2 -e "(typep 3 'real)"              → t
$ ./build/golisp2 -e "(typep 3 'number)"            → t
$ ./build/golisp2 -e "(typep 3 'atom)"              → t
$ ./build/golisp2 -e "(typep 3 'list)"              → ()   (kein Fehler, bekannter Typname)
$ ./build/golisp2 -e "(typep 3 'sequence)"          → ()   (kein Fehler, bekannter Typname)
$ ./build/golisp2 -e "(typep 3 'structure-object)"  → ()   (kein Fehler, bekannter Typname)
$ ./build/golisp2 -e "(typep 3 't)"                 → t
$ ./build/golisp2 -e "(typep 3 'fixnum)"            → ERR: "typep: unbekannter Typ 'fixnum'"
$ ./build/golisp2 -e "(typep 3 'character)"         → ERR: "typep: unbekannter Typ 'character'"
$ ./build/golisp2 -e "(typep 3 'vector)"            → ERR: "typep: unbekannter Typ 'vector'"
$ ./build/golisp2 -e "(typep 3 'double-float)"      → ERR: "typep: unbekannter Typ 'double-float'"
$ ./build/golisp2 -e "(typep 3 'single-float)"      → ERR: "typep: unbekannter Typ 'single-float'"
$ ./build/golisp2 -e "(typep 3 'INTEGER)"           → ERR: "typep: unbekannter Typ 'INTEGER'"
```
(`subtypep`/`deftype` per `rg` bestätigt: existieren nirgends in `src/lib/*.go`
oder `src/embed/*.lisp`.)

## Finding 8 — Tests

Regressionstests für alle 6 Code-Findings hinzugefügt (siehe oben, je
eigene `TestXxx`-Funktion in `src/lib/types_typep_test.go`). Zusätzlich
`TestTypepFehlendeSupertypZeilen` für die im Review vermissten
Supertyp-Zeilen:
- `(typep to-mm 'macro)` — Makro vs. `'macro`
- `(typep (make-punkt …) 'list)` / `'sequence` — Struct vs. `'list`/`'sequence`
- Condition vs. `'list`

Diese vier waren bereits vor dem Fix grün (Hierarchie über `structure-object
→ cons → list → sequence` bzw. `%type-start` → `cons` für Conditions trägt
das schon) — als Regression jetzt fest im Test-Netz, nicht als neues
Verhalten.

```
$ go test ./src/lib/ -run TestTypepFehlendeSupertypZeilen -count=1 -v
--- PASS: TestTypepFehlendeSupertypZeilen
```

## Full-Suite-Ergebnis

```
$ ./build.sh
✓ Binaries in ./build/: golisp2 golisp2-client

$ go test ./... -count=1
Go test: 417 passed in 6 packages

$ ./build/golisp2 -t
=== Report: 141 PASS, 0 FAIL, 0 XFAIL, 0 XPASS ===
```

## Geänderte Dateien

- `src/embed/types.lisp` — 6 minimale Fixes (Findings 1–6), keine
  Verhaltensänderung außerhalb der gemeldeten Fälle.
- `src/lib/types_typep_test.go` — 7 neue Testfunktionen (Findings 1–6, 8).
- `src/embed/ki-referenz.md` — Finding 7 (2 Stellen).
- `condition.lisp`, `stdlib.lisp` — unverändert, wie vorgeschrieben.

## Self-Review

- Alle Fixes sind additive Guards/Helper, keine bestehende Logik entfernt
  oder umgebaut. `%proper-length`/`%proper-list?` sind neue, klar benannte
  Helfer ohne Kollision mit vorhandenen Symbolen (`rg` geprüft).
- Keine zweite Typ-Tabelle angelegt — `*type-parents*` unverändert.
- Bestehende Prädikate (`number?`, `punkt?` usw.) nicht angefasst.
- `condition.lisp`/`stdlib.lisp` nicht verändert (Vorgabe eingehalten).
- Keine Datei über 1000 Zeilen (`types.lisp` 217, Testdatei 265).
- Keine offenen Concerns.
