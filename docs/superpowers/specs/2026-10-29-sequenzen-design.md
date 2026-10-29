# Vektor-Alltag und Sequenz-Kern — Design

**Datum:** 2026-10-29 · **Status:** genehmigt (Gerhard: „Vektor-Alltag + Sequenz-Kern“)
**Autor:** Gerhard Quell · **CoAutor:** claude-opus-5.5

## Ziel

Vektoren (seit 0.17/`vector.go`) sollen im Alltag benutzbar sein, und die
wichtigsten CL-Sequenzfunktionen sollen für Listen **und** Vektoren
existieren. Stufe 1 (0.23.0) hat die stillen Fehler beseitigt; hier geht es
um fehlende Funktionen.

## Ausgangslage (live geprüft, Stand 0cc7e67)

- Vorhanden für Listen: `nth take drop reverse remove remove-if remove-if-not
  filter find-all every any reduce member last butlast remove-duplicates
  copy-list sort` (sort auch Vektor).
- Fehlen ganz: `elt subseq find find-if position position-if count count-if
  map concatenate copy-seq some vectorp arrayp vector-pop`.
- `#(1 2 3)` → `reader: unbekanntes Dispatch-Zeichen #(` (`reader.go:230`).
- `(eval v)` mit Vektor → `eval: unbekannter Typ` (`eval_core.go:108-122`):
  Vektoren sind nicht selbstauswertend.
- `(json-encode (vector 1 2))` → `CellToJSON: Typ 12 nicht darstellbar`.
- `(loop for x across v …)` → `loop: for: erwartet in|from`.
- `eql` ist Lisp (`stdlib.lisp:616`): Zahlen nach Wert, sonst `eq`.
- Funktionsbezeichner: `resolveFn(env, atom)` / `withFnDesignator` (`eval_core.go:415`).
- `pn-gps1/gps2.lisp:65` definiert ein eigenes `some` (PAIP-Demo).

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Umfang | Vektor-Alltag + Sequenz-Kern; bestehende Listenfunktionen bleiben unverändert (laute Fehler bei Vektoren) |
| Strings als Sequenz? | **Nein** — golisp2 hat keine Characters. Strings → Fehler „Sequenz (Liste oder Vektor) erwartet“; dafür gibt es `substring`/`string->list` |
| Keywords | nur `:test` und `:key`, wo CL sie hat; kein `:start`/`:end`/`:from-end`/`:count` |
| Funktionsbezeichner | positional und in `:test`/`:key`: Lambda, Funktion oder Symbol (über `resolveFn`, wie `sort`) |
| Default-Test | `eql` — wird dafür **Go-Primitive**; die Lisp-Definition in `stdlib.lisp` entfällt (eine Quelle) |
| Ergebnistyp | `subseq`/`copy-seq`: Typ des Arguments. `map`/`concatenate`: per Typ-Argument `'list`/`'vector` (`map` zusätzlich `nil` → nur Nebenwirkung, Ergebnis `()`) |
| Destruktiv? | Nein, alles liefert neue Sequenzen (wie `sort`) |
| `#(…)` | Reader-Literal; Elemente werden **nicht** ausgewertet (CL); jedes Lesen erzeugt einen neuen Vektor. Vektoren werden selbstauswertend |

## Verhalten

Gemeinsamer Helfer `seqItems(name, c) ([]*Cell, isVec bool, err)`:
`()`/Liste → `properList` (Stufe 1); Vektor → Kopie unter `RLock`; sonst
`<name>: Sequenz (Liste oder Vektor) erwartet, got <x>`.

| Funktion | Signatur | Ergebnis |
|---|---|---|
| `elt` | `(elt s i)` | Element; Index außerhalb → Fehler (auch bei Listen, CL) |
| `subseq` | `(subseq s start [end])` | gleicher Typ; `0 ≤ start ≤ end ≤ len`, sonst Fehler |
| `copy-seq` | `(copy-seq s)` | flache Kopie, gleicher Typ |
| `concatenate` | `(concatenate typ s…)` | `typ` ∈ `list`/`vector` |
| `map` | `(map typ f s1 s2…)` | bis zur kürzesten Sequenz; `typ` ∈ `list`/`vector`/`nil` |
| `some` | `(some p s1 s2…)` | erster wahre Wert von `p`, sonst `()` |
| `find` | `(find x s &key test key)` | Element oder `()` |
| `find-if` | `(find-if p s &key key)` | Element oder `()` |
| `position` | `(position x s &key test key)` | Index oder `()` |
| `position-if` | `(position-if p s &key key)` | Index oder `()` |
| `count` | `(count x s &key test key)` | Anzahl |
| `count-if` | `(count-if p s &key key)` | Anzahl |
| `vectorp`/`arrayp` | `(vectorp x)` | `t`/`()` (`arrayp` = `vectorp`, es gibt nur eindimensionale) |
| `vector-pop` | `(vector-pop v)` | letztes Element, entfernt; leer → Fehler |

`:key f` wird vor dem Test auf jedes Element angewandt (nicht auf `x`).
Unbekanntes Keyword oder Keyword ohne Wert → Fehler. Falscher Typ-Name bei
`map`/`concatenate` → Fehler.

Weitere Punkte:
- `json-encode`: Vektor → JSON-Array (wie Liste). `json-parse` bleibt bei Listen.
- `loop`: `for x across v` (Vektor; Liste → Fehler).
- `gps2.lisp`: eigenes `some` entfernen (identische Semantik, sonst
  Redefine-Warnung gegen die Go-Primitive).

## Tests

Go-Tests pro Funktion (Liste, Vektor, leere Sequenz, Fehlerfälle,
`:key`/`:test`), Reader-/Eval-Test für `#(…)`, JSON-Test, Lisp-Test für
`loop across`. Tutorial-Abdeckungstest erzwingt Einträge für alle neuen Symbole.

## Nicht dabei

Strings als Sequenz, `:start/:end/:from-end/:count`, `remove`/`reverse`/
`every`/`reduce` usw. für Vektoren (Option „Alles“ abgelehnt),
mehrdimensionale Arrays, Fill-Pointer, `map-into`, `search`, `mismatch`,
`substitute`, `merge`, `fill`, `replace`.
