# SWANK-Inspector — Design

**Datum:** 2026-10-29 · **Status:** genehmigt (Diskussion mit Gerhard), drei
Detailentscheidungen von Claude markiert (⚑) — bitte bestätigen
**Autor:** Gerhard Quell · **CoAutor:** claude-opus-5.5

## Ziel

`C-c I` (`slime-inspect`) in Emacs/SLIME soll golisp2-Objekte aufklappen:
Slots eines `defclass`-Objekts, Einträge einer Hashtabelle, Felder eines
Structs, Elemente von Listen und Vektoren — jeder Wert klickbar, mit
Rück-/Vorwärts-History. Hauptnutzen: **geteilten, veränderlichen Zustand**
(seit 0.20.0 `defclass`) live beobachten und über `inspector-eval` ändern.

## Ausgangslage (live geprüft, Stand 5479958)

- Alle SWANK-Ops liegen in `src/embed/swank.lisp` (493 Zeilen), Dispatch in
  `handle-emacs-rex` (`swank.lisp:200`). Unbekannte Ops → `(:ok ())`
  (`swank:ok-nil`). Heute landen alle `swank:inspect*`-Ops dort.
- `docs/swank.md:147` führt den Inspector unter „Offen“.
- Fehlerform wie `listener-eval`: `(:return (:abort "<text>") id)`.
- `--swank` (`RunServer`) gibt jeder Verbindung ein eigenes Env;
  `ServeEnv` (golisp2web) teilt ein Env über alle Verbindungen.
- `*struct-types*` (`stdlib.lisp:385`) speichert `(name slot-anzahl)` —
  keine Slot-Namen. `*class-types*` speichert `(name (slot-namen))`.
  Leser von `*struct-types*`: nur `src/embed/types.lisp` (`%struct-instance?`
  nutzt die Anzahl, sonst nur `assoc`), Tests `types_typep_test.go:66-69`,
  Doku-String `tools/gen-reference.lisp:96`.
- Funktionen haben keine Introspektions-Primitiven (keine Lambda-Liste/
  Body aus Lisp abfragbar). Für **benannte** Funktionen gibt es
  `swank--arglist` und `swank--find-definition` (Go, `swank/env.go`).
- **`*` ist in golisp2 die Multiplikation** (Lisp-1). SLIME dokumentiert für
  `inspector-eval` „`*` ist an das inspizierte Objekt gebunden“.
- **Es gibt keinen String-Vergleich** (`string<` u. ä. fehlen); `maphash`
  iteriert in Go-Map-Reihenfolge (zufällig pro Aufruf).

### Protokoll (gelesen in `slime.el` 2.27, `/usr/share/emacs/site-lisp/elpa-src/slime-2.27/`)

| Op (Client-Zeile) | Antwort |
|---|---|
| `(swank:init-inspector "ausdruck")` (6386) | `(:title s :id n :content chunk)` |
| `(swank:inspect-nth-part n)` (6519) | wie init, Teil n |
| `(swank:inspector-pop)` (6546) | wie init oder `nil` → „No previous object“ |
| `(swank:inspector-next)` (6557) | wie init oder `nil` |
| `(swank:inspector-reinspect)` (6657) | wie init, aktuelles Objekt neu |
| `(swank:quit-inspector)` (6567) | egal |
| `(swank:inspector-range from to)` (6704) | chunk |
| `(swank:inspector-eval "form")` (6641) | String (Minibuffer) |

`chunk` = `(ispecs len start end)`; der Client holt per `inspector-range`
nach, solange `end < len` (Limit 500 ispecs pro Abruf). `ispec` ist ein
String, `(:value "text" n)` (klickbar → `inspect-nth-part n`),
`(:label "text")` oder `(:action "text" n)`. `:id` ist die Teilnummer des
Titels — klickbar, muss auf das Objekt selbst zeigen.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Lisp oder Go? | **Lisp** in `swank.lisp` — wie alle anderen Ops; keine neue Go-Primitive |
| Umfang | **Navigation + `inspector-eval`** (Gerhard: Option 2b) |
| Struct-Slot-Namen | **Ja:** `*struct-types*` speichert `(name (slot-namen))`, Anzahl = Länge |
| Name des Objekts in `inspector-eval` ⚑ | **`*inspectee*`** (globale Variable), nicht `*` — `*` umbinden bräche jede Multiplikation im Ausdruck |
| `inspector-range` ⚑ | **Ja** (≈10 Zeilen) — ohne Range würde eine Liste mit 10 000 Elementen entweder abgeschnitten oder als Riesen-Frame geschickt |
| Reihenfolge Hashtabellen-Einträge ⚑ | **Unsortiert** (Go-Map-Reihenfolge, kann bei `g` springen). Sortieren braucht `string<` — eigene Lücke, separat |

## Verhalten

### Zustand

Global im Env (pro `--swank`-Verbindung also eigener Zustand; bei
`ServeEnv` teilen sich Verbindungen einen Inspector — dokumentiert,
akzeptiert):

- `*inspectee*` — aktuelles Objekt (für `inspector-eval`)
- Teil-Tabelle Nummer → Objekt, bei jedem Öffnen neu aufgebaut; Teil 0 ist
  das Objekt selbst (= `:id`)
- volle ispec-Liste des aktuellen Objekts (für `inspector-range`)
- History: Rück-Stack und Vorwärts-Stack von Objekten

### Darstellung

Jeder Inhalt beginnt mit `"Typ: " (:value "<type-of>" n) "\n"`. Danach:

| Objekt (Prüfreihenfolge) | Inhalt |
|---|---|
| defclass-Objekt (`type-of` ∈ `*class-types*`) | `Slots:`-Label, je Slot in Definitionsreihenfolge `"  slot = " (:value …)` |
| andere Hashtabelle | `Einträge: n`, je Eintrag `"  " (:value key) " = " (:value wert)` |
| Struct (`type-of` ∈ `*struct-types*`) | je Slot `"  slot = " (:value …)` |
| Vektor | `"  i: " (:value …)` |
| echte Liste | `"  i: " (:value …)` |
| Punktpaar/improper | `"  car: " …`, `"  cdr: " …` |
| Symbol | `Wert:` (falls gebunden), `Arglist:` (falls Funktion/Makro), `Quelle: datei:zeile` (falls bekannt) |
| sonst (Zahl, String, Funktion, Channel …) | `"Wert: " <Druckform>` |

Werte werden über die vorhandene `swank--value-string` gedruckt —
defclass-Objekte also als `#<schiff>`, zyklenfest.

### Ops

- `init-inspector s`: liest und evaluiert `s` (erste Form), leert die
  History, öffnet das Ergebnis. Fehler → `:abort "<text>"`.
- `inspect-nth-part n`: aktuelles Objekt auf den Rück-Stack, Vorwärts-Stack
  leeren, Teil n öffnen. Unbekanntes n → `:abort "inspect-nth-part: kein Teil n"`.
- `inspector-pop` / `inspector-next`: zwischen den Stacks wechseln; leerer
  Stack → `(:ok ())`.
- `inspector-reinspect`: aktuelles Objekt neu rendern — **zeigt Mutationen**.
- `quit-inspector`: Zustand zurücksetzen, `(:ok ())`.
- `inspector-range from to`: Ausschnitt der ispec-Liste.
- `inspector-eval s`: alle Formen von `s` evaluieren (Root-Env,
  `*inspectee*` gebunden), Druckform des letzten Werts als String.
  Fehler → `:abort`. Der Buffer aktualisiert sich nicht von selbst —
  `g` drücken (wie in SBCL/SLIME).

Kein Treffer in der Teil-Tabelle für `:id`-Klick ist unmöglich (Teil 0
existiert immer).

## Nicht dabei (bleiben bei `(:ok ())`)

`describe-inspectee`, `pprint-inspector-part`, `inspector-history`,
`inspector-toggle-verbose`, `inspector-call-nth-action` (keine Actions),
`inspect-current-condition`/`inspect-frame-var` (kein Debugger),
`find-source-location-for-emacs '(:inspector n)`, fancy-inspector-Contrib.
Keine Anzeige von Lambda-Listen/Bodies anonymer Funktionen (keine
Introspektion; würde Go-Primitive brauchen).

## Tests

Go, `src/lib/swank/inspector_test.go`, Struktur statt Substrings (CLAUDE.md
Protokoll-Regel): Antwort parsen, `:title`/`:content` aus der Plist holen,
`(:value "text" n)`-Einträge suchen. Fälle: Klasse, Struct mit Slot-Namen,
Hashtabelle, Liste/Punktpaar, Navigation nth-part/pop/next, leerer Pop,
unbekannter Teil, Reinspect nach Mutation, `inspector-eval` mit
`*inspectee*`, Range bei großer Liste, Fehler in `init-inspector`.
Abschluss: manueller Test in Emacs (`C-c I`, `RET`, `l`, `n`, `g`, `e`, `q`).

## Doku

`docs/swank.md` (Op-Tabelle, Abschnitt Inspector, „Offen“ kürzen),
`tools/gen-reference.lisp` (`*struct-types*`-Beschreibung) + Regenerierung,
Version 0.21.0.

## Folgearbeit (nicht Teil dieser Spec)

- `string<`/`string=`-Familie → dann Hashtabellen-Einträge sortiert anzeigen.
- Veraltete Aussagen „kein defclass“: Lispbuch Kap. 17/25,
  `cltl2/CLOS.md`, Zutat `golisp2.md` §10.2.
