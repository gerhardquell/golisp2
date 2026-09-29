# type-of / typep — Design

**Datum:** 2026-09-29 · **Status:** genehmigt (Brainstorming mit Gerhard)
**Autor:** Gerhard Quell · **CoAutor:** claude-opus-5.5

## Ziel

`TODO.md` (28.09.2026): golisp2 soll „äquivalent zu CLisp“ die
Funktionen `type` und `typep` bekommen. CL hat keine Funktion `type` —
gemeint ist `type-of`. Ergebnis dieser Spec:

- `(type-of x)` → Symbol, das den Typ von `x` benennt.
- `(typep x typangabe)` → `t` oder `()`, mit Typnamen, Hierarchie,
  Kombinatoren und Zahlbereichen.

## Ausgangslage (live geprüft, Build vom 20260929)

- Weder `type-of` noch `typep` existieren (`unbekanntes Symbol`), auch
  kein `deftype`, `check-type`, `subtypep`.
- Vorhandene Prädikate (Go): `atom?`, `list?`, `number?`, `string?`,
  `symbol?`, `null?`, `pair?`, `hash-table-p`. Es gibt kein `integer?`,
  kein `keyword?`, kein `function?`.
- **Kern-Typen** (`LispType` in `src/lib/types.go`): `ATOM`, `NUMBER`,
  `STRING`, `LIST`, `LAMBDA`, `FUNC`, `MACRO`, `NIL`, `HASHTABLE`, dazu
  `MVALUES`/`SYMMACRO` (als Werte nicht sichtbar).
- **Zahlen sind immer `float64`.** `3.0` und `3` sind ununterscheidbar
  (beide drucken als `3`).
- **Keywords** sind ATOM-Cells, deren Name mit `:` beginnt
  (`(symbol-name :a)` → `":a"`).
- **Channels** (`chan-make`) sind FUNC-Closures (`#<func>`).
- **Keine Characters** (`#\a` → Reader-Fehler).
- **Structs sind nackte Listen:** `(make-punkt :x 1 :y 2)` →
  `(punkt 1 2)`. `punkt?` prüft nur `(car x)`. Es gibt keine
  Struct-Registry.
- **Conditions** sind `(%condition typ plist)`; Typ-Registry
  `*condition-types*` mit Elternkette und `%cond-subtype?` in
  `src/embed/condition.lisp`. Wurzel `condition` ist registriert.
- `LoadStdlib` lädt: `stdlib.lisp` → `defsystem.lisp` →
  `condition.lisp` → `loop.lisp`.
- `(funcall 'number? 3)` schlägt fehl (Symbol wird nicht aufgelöst);
  `(funcall (eval 'number?) 3)` geht.

## Entscheidungen (Gerhard)

1. **Umfang:** Typnamen + Hierarchie + Kombinatoren + Zahlbereiche.
   Kein `deftype`, `check-type`, `subtypep` in diesem Schritt.
2. **Structs:** `type-of` liefert den Struct-Namen über eine Registry,
   die `defstruct` füllt. Kein eigener Cell-Typ (wäre Breaking Change).
3. **Ort:** Lisp (`src/embed/types.lisp`) + genau ein Go-Primitiv
   `%cell-type`. Kein Go-Fastpath (zwei Typ-Tabellen = Duplikat).
4. **Alist/Plist** werden *keine* Typnamen (in CL auch nicht); Weg dafür
   ist `(satisfies …)`.

## Typtabelle

| Wert | `type-of` | Obertypen (für `typep`) |
|---|---|---|
| `3`, `3.0` | `integer` | rational, real, number, atom, t |
| `3.5` | `float` | real, number, atom, t |
| `"a"` | `string` | sequence, atom, t |
| `'foo` | `symbol` | atom, t |
| `:k` | `keyword` | symbol, atom, t |
| `t` | `boolean` | symbol, atom, t |
| `()` | `null` | boolean, symbol, list, sequence, atom, t |
| `(1 2)` | `cons` | list, sequence, t |
| Go-Funktion (`car`) | `compiled-function` | function, atom, t |
| `(lambda …)` | `function` | atom, t |
| Makro | `macro` | atom, t |
| Hash-Table | `hash-table` | atom, t |
| Struct `(punkt 1 2)` | `punkt` | structure-object, cons, list, sequence, t |
| Condition | ihr Typ (`file-error`) | Elternkette bis `condition`, cons, list, sequence, t |

Die Hierarchie steht in **einer** Tabelle `*type-parents*` in
`types.lisp` (Typ → direkte Obertypen). `typep` mit einem Namen läuft
die Kette von `(type-of x)` nach oben. Ausnahme: `atom` ist als
`(not cons)` definiert — so ist es auch für Structs/Conditions korrekt.

### Struct-Erkennung

Ein Wert ist genau dann Struct `name`, wenn:
- er eine Liste ist,
- `(car x)` in `*struct-types*` registriert ist, und
- die Länge `1 + slot-anzahl` ist.

Das ist strenger als das heutige `punkt?` (nur `car`). `punkt?` bleibt
unverändert — Verhaltensänderung an bestehenden Prädikaten ist nicht Teil
dieser Spec. Restrisiko (akzeptiert): ein Literal `'(punkt 1 2)` gilt
ebenfalls als `punkt`.

### Reihenfolge in `type-of`

1. `%cell-type` → nicht `cons`: fertig (Zahl → integer/float nach
   Nachkommaanteil, Symbol → keyword/boolean/symbol, FUNC →
   compiled-function, LAMBDA → function, …).
2. `cons`: Condition (`%cond?`) → ihr Typ.
3. Registrierter Struct (Regel oben) → Struct-Name.
4. Sonst `cons`.

## Typangaben in `typep`

| Angabe | Bedeutung |
|---|---|
| Symbol aus der Typtabelle | Name + Hierarchie |
| `t` / `nil` | immer wahr / immer falsch |
| registrierter Struct-Name | Struct-Erkennung oben |
| registrierter Condition-Typ | `%cond-type?` (inkl. Eltern) |
| `(or a b …)` | mindestens einer |
| `(and a b …)` | alle |
| `(not a)` | Negation |
| `(member v …)` | `eql` mit einem der Werte |
| `(eql v)` | `eql` mit `v` |
| `(satisfies f)` | `f` ist Symbol einer 1-stelligen Funktion; Wert über `(eval f)` aufgelöst |
| `(integer lo hi)`, `(float lo hi)`, `(real lo hi)`, `(rational lo hi)` | Typ + Bereich |

Bereichsgrenzen: Zahl = inklusiv, `(zahl)` = exklusiv, `*` oder
weggelassen = unbegrenzt. `(integer 0)` ≡ `(integer 0 *)`.

### Namensauflösung und Kollisionen

Reihenfolge für ein Symbol: eingebaute Typnamen → Struct-Registry →
Condition-Registry → Fehler.

Kollisionen werden **laut** gemeldet, nicht still aufgelöst:
- `defstruct` mit einem eingebauten Typnamen (`(defstruct integer …)`)
  → `WARN:` beim Definieren; `typep` sieht weiterhin den eingebauten Typ.
- `define-condition` mit eingebautem Typnamen → ebenso `WARN:`.
- Struct- und Condition-Name gleich → `WARN:` bei der zweiten
  Definition; Struct gewinnt in `typep`.

Die Prüfung nutzt `%builtin-type?` aus `types.lisp`. Da `defstruct` in
`stdlib.lisp` vor `types.lisp` geladen wird, aber erst zur
Benutzungszeit expandiert, ist die Funktion dann gebunden; zur
Sicherheit mit `(bound? '%builtin-type?)` geschützt.

## Fehler

- Unbekannter Typname: `typep: unbekannter Typ 'foo'` (CL: auch Fehler).
- Ungültige Angabe (`(or)` ist gültig = nil; `(not a b)`, `(integer "x")`,
  unbekannter Kopf `(foo 1)`): `typep: ungültige Typangabe …`.
- `(satisfies f)` mit ungebundenem `f`: `typep: satisfies: 'f' ist keine Funktion`.
- `type-of` wirft nie.

## Komponenten

| Datei | Änderung |
|---|---|
| `src/lib/primitives.go` | neu `%cell-type` (1 Arg → Symbol: `number`, `string`, `symbol`, `cons`, `null`, `lambda`, `func`, `macro`, `hash-table`), registriert in `BaseEnv()` |
| `src/embed/stdlib.lisp` | `(defvar *struct-types* '())`; `defstruct` registriert `(name slot-anzahl)` und warnt bei Kollision |
| `src/embed/condition.lisp` | `define-condition` warnt bei Kollision mit eingebautem Typ oder Struct |
| `src/embed/types.lisp` | **neu**: `*type-parents*`, `%builtin-type?`, `%struct-instance?`, `type-of`, `typep`, `%typep-spec`, Bereichsprüfung |
| `src/embed/assets.go` | `//go:embed types.lisp` |
| `src/lib/stdlib.go` | `LoadStdlib` lädt `types.lisp` nach `condition.lisp` |

Bestehende Prädikate (`number?` usw.) bleiben unverändert; `types.lisp`
baut auf ihnen und `%cell-type` auf.

## Bewusste Abweichungen von CL

- `3.0` ist `integer` (Kern kennt nur `float64`); `(typep 3.0 'float)` → `()`.
- `type-of` liefert `integer`/`float`, nicht `fixnum`/`double-float`
  bzw. `(integer 0 4611686018427387903)` wie SBCL.
- Channels sind `compiled-function`.
- `macro` ist ein golisp2-eigener Typ (in CL sind Makros keine Werte).
- Keine Characters, keine Vektoren/Arrays.
- Structs sind Listen: `(typep p 'cons)` → `t` (in CL `()`).

## Nicht-Ziele

`deftype`, `check-type`, `subtypep`, `typecase`/`etypecase`,
`coerce`, Alist/Plist-Typen, eigener STRUCT-Cell-Typ, Änderung an
`punkt?`-Prädikaten.

## Tests

`src/lib/types_typep_test.go` (Helfer `evalStdlibEq`/`evalStdlibErr`
aus `stdlib_reduce_setf_test.go`), jeweils zuerst rot:
- je Zeile der Typtabelle: `type-of` und `typep` gegen jeden Obertyp,
  plus mindestens ein negativer Fall;
- Struct: registriert + richtige Länge → Name; falsche Länge → `cons`;
  Literal ohne `defstruct` → `cons`;
- Condition: `type-of` → Typ, `typep` gegen Eltern und `condition`;
- jeder Kombinator, inkl. `(or)`, `(and)`, verschachtelt;
- Bereiche: inklusiv, exklusiv, `*`, weggelassen, Float gegen `integer`;
- Fehlerfälle aus „Fehler“;
- Kollisions-Warnungen (`defstruct integer`, gleichnamige Condition).

Danach `go test ./... -count=1` und `./build/golisp2 -t` grün.

## Doku

- `src/embed/ki-referenz.md`: `type-of`/`typep` + Abweichungen (Vorspann
  für sigo — muss stimmen, siehe Retrospektive 20260928).
- `docs/lisp-semantik.md`: Abschnitt Typen inkl. Abweichungen.
- `docs/golisp2-cheatsheet.md`: Kurzeintrag.
- `docs/referenz-generiert.md` via `tools/gen-reference.lisp` neu erzeugen.

## Nebenfunde (nicht Teil dieser Spec)

- `(- 5)` → `5` (CL: `-5`) — einstelliges Minus negiert nicht.
- `gcd` in `stdlib.lisp` nutzt `/` (Float-Division) → `(gcd 12 18)` → `18`.
- `(funcall 'f …)` löst Symbole nicht auf (CL: tut es).
