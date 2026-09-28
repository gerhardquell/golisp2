# sigo-Kontext — Design

**Datum:** 2026-09-28 · **Status:** genehmigt (Brainstorming mit Gerhard)
**Autor:** Gerhard Quell · **CoAutor:** claude-opus-5.5

## Ziel

`TODO.md` (27.09.2026), Schritt 1: golisp2 soll den sigoREST-LLMs eine
golisp2-Referenz als Vorspann (`system_prompt`) mitschicken, damit die
Modelle golisp2 nicht mehr raten und der Provider-Cache greift. Dazu
Gerhards drei Wünsche:

1. Den Prompt komplett leeren können.
2. Den Prompt aus golisp2 heraus lesen und schreiben können.
3. Kosten ermitteln können — insbesondere, ob und wie viel Cache greift.

Nicht Teil dieser Spec: TODO-Schritt 2 (Modellwahl) und Schritt 3
(`temperature`/`max_tokens`).

## Ausgangslage (Rechercheergebnis)

- **Referenz existiert bereits.** `docs/ki/referenz.md` (16 KB, grob
  4–5k Token) ist ausdrücklich als *Initial-Context für andere KIs*
  geschrieben. TODO.md plante eine neue Verdichtung — das wäre ein
  Duplikat gewesen. Der Dateikopf behauptet „generiert via
  `tools/gen-reference.lisp`“; das ist falsch, der Generator schreibt nur
  `docs/referenz-generiert.md`. `referenz.md` ist handgepflegt.
- **`go:embed` erreicht `docs/` nicht** (kein `..` aus `src/embed/`).
- **sigoREST, `/v1/chat/completions`:**
  - Ein leeres `system_prompt` im Request gilt als „nicht gesetzt“ — dann
    greift Kanal- bzw. globaler System-Prompt. Komplett leeren ist per
    Request heute nicht möglich.
  - Globales und Kanal-Memory (`memory.json`) werden **immer**
    vorangestellt, ohne Opt-out.
  - `ChatUsage` enthält nur `prompt_tokens`, `completion_tokens`,
    `total_tokens`. `cached_tokens` und `reasoning_tokens` des Providers
    werden verworfen; golisp2 kann sie nie sehen.
  - `ModelInfo` kennt `input_cost`/`output_cost` ($/1M Token), aber keinen
    Cache-Preis. `CalcCostUSD` existiert.
- **golisp2, `src/lib/sigorest.go`:** parst nur
  `choices[0].message.content`, `usage` wird weggeworfen. `sigoHost` wird
  von `(sigo-host "…")` ohne Mutex geschrieben, während
  `parfunc`-Goroutinen lesen (Data Race). Keine Go-Tests.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Quelle des Vorspanns | `docs/ki/referenz.md`, keine neue Verdichtung |
| Ablage | `git mv` nach `src/embed/ki-referenz.md`, eingebettet wie `stdlib.lisp` |
| Welcher Prompt ist „der Prompt“ | Client-Vorspann in golisp2 (prozesslokal), nicht der Server-globale |
| Bedeutung von „leeren“ | Modell sieht **nur** den User-Prompt: kein Vorspann, kein Server-Prompt, kein Memory |
| Wann Vorspann | immer, ab Start; per Funktion änderbar und leerbar |
| Kosten-API | `sigo` bleibt String; neu `sigo*` (Assoc-Liste) + Summenzähler |
| USD | Tokens durchreichen + USD ohne Cache-Rabatt (obere Schranke) |
| `sigoHost` | kommt mit unter den neuen Mutex |

## Teil 1 — sigoREST (Repo `/u/go-projekte/sigoREST`)

### 1a. Request-Feld `bare`

`ChatRequest` bekommt `Bare bool \`json:"bare,omitempty"\``.

Mit `"bare": true` legt sigoREST **nichts** vor die Messages: kein globales
Memory, kein Kanal-Memory, kein Kanal- oder globaler System-Prompt. Einzige
Ausnahme: ein nicht-leeres `system_prompt` aus dem Request wird als
System-Message gesetzt. Danach Session-Verlauf und Request-Messages wie
bisher.

Ohne `bare` (oder `false`) bleibt das Verhalten exakt wie heute.
Abwärtskompatibel.

Verworfene Alternative: `SystemPrompt` auf `*string` umstellen, sodass
`""` „leer“ heißt. Bricht jeden Client, der heute `""` schickt, und löst
das Memory-Problem nicht.

Bewusste Konsequenz: Für golisp2-Calls entfallen „Antworte kurz und auf
Deutsch“ und das Nutzer-Memory. Wer das will, schreibt es in den Vorspann.

### 1b. Usage vollständig durchreichen

`ChatUsage` wird OpenAI-kompatibel erweitert (alle neuen Felder
`omitempty`):

```json
"usage": {
  "prompt_tokens": 58, "completion_tokens": 115, "total_tokens": 173,
  "prompt_tokens_details":     {"cached_tokens": 0},
  "completion_tokens_details": {"reasoning_tokens": 98},
  "cost_usd": 0.00041
}
```

- Beim Parsen der Provider-Antwort werden die Detailfelder mitgelesen.
  Liefert ein Provider sie nicht, bleiben sie 0.
- `cost_usd` ist eine sigoREST-eigene Erweiterung, berechnet mit dem
  vorhandenen `CalcCostUSD` aus `input_cost`/`output_cost` — **ohne
  Cache-Rabatt**, also eine obere Schranke.

### 1c. Tests (in `main_test.go`, vorhandener Stil)

Fake-Upstream via `httptest.Server`. Geprüft wird die **Struktur** der beim
Upstream ankommenden `messages`:

- `bare` + nicht-leeres `system_prompt` → genau eine System-Message (der
  Prompt), kein Memory.
- `bare` + leeres `system_prompt` → keine System-Message.
- ohne `bare` → Verhalten wie bisher (Memory + Default-Prompt).

Sowie: `cached_tokens`, `reasoning_tokens` und `cost_usd` kommen in der
Antwort an den Client an.

## Teil 2 — golisp2

### 2a. Referenz einbetten

- `git mv docs/ki/referenz.md src/embed/ki-referenz.md`
- `src/embed/assets.go`: `//go:embed ki-referenz.md` → `var KiReferenz string`
- Dateikopf korrigieren: falschen Generator-Hinweis entfernen, Hinweis
  ergänzen, dass Anfang der Datei stabil bleiben muss (kein Datum, keine
  Version in den ersten Zeilen — sonst wertlos für den Provider-Cache).
- Verweise nachziehen: `CLAUDE.md`, `docs/golisp2-cheatsheet.md` (2×).
- `docs/ki/referenz_en.md` und `referenz_cn.md` bleiben, wo sie sind.

### 2b. Zustand in `sigorest.go`

`sigorest.go` bleibt der einzige Ort für HTTP gegen sigoREST.

```go
sigoSystemPrompt = assets.KiReferenz // Start-Default
sigoUsageSum     sigoUsage           // Summe seit Start
sigoStateMu      sync.Mutex          // schützt Vorspann, Summe, sigoHost
```

- `sigoCallToHost` schickt immer `"bare": true` und
  `"system_prompt": <Vorspann>`. Ist der Vorspann leer, fehlt das Feld;
  mit `bare` sieht das Modell dann nur den Prompt.
- Rückgabe wird `sigoResult{Text, Model, FinishReason, Usage}` statt
  `string`.
- Jeder erfolgreiche Call addiert seine Usage unter Mutex auf
  `sigoUsageSum`.
- Alle Lese-/Schreibzugriffe auf `sigoHost` laufen über `sigoStateMu`.

### 2c. Primitiven (in `RegisterSigo`, damit über `BaseEnv()`)

| Aufruf | Ergebnis |
|---|---|
| `(sigo-system-prompt)` | aktueller Vorspann (String) |
| `(sigo-system-prompt "text")` | setzen, neuer Wert zurück |
| `(sigo-system-prompt "")` | leeren |
| `(sigo-reference)` | eingebettete Referenz, z. B. `(sigo-system-prompt (sigo-reference))` zum Zurücksetzen |
| `(sigo* prompt [model] [session-id] [host])` | Assoc-Liste (s. u.) |
| `(sigo-usage)` | Summen seit Start + `(calls . n)` |
| `(sigo-usage-reset)` | Summen auf 0 |

`sigo*` liefert, im Stil von `(memstats)` mit einfachen Symbolen:

```lisp
((text . "…") (model . "zai-glm53") (finish-reason . "stop")
 (prompt-tokens . 5210) (completion-tokens . 115)
 (cached-tokens . 5120) (reasoning-tokens . 98) (cost-usd . 0.0031))
```

`(sigo …)` bleibt unverändert String-liefernd und ist intern `sigo*` ohne
Hülle — ein Request-Pfad, zwei Sichten. Bestehende Aufrufer
(`tests/parallel-mind-*`, `(eval (read (sigo …)))`) merken nur, dass der
Vorspann mitgeht.

`finish-reason` ist absichtlich dabei: so wird der Fall „leerer Text bei
`length`“ (gemma4, 27.09.) sichtbar statt rätselhaft.

`(sigo-usage)` liefert dieselben Token-Schlüssel wie `sigo*` plus
`(calls . n)`, ohne `text`/`model`/`finish-reason`.

### 2d. SWANK

`src/embed/swank.lisp`: Arglisten und Doku-Strings für die neuen
Funktionen (neben den vorhandenen `sigo`-Einträgen).

### 2e. Tests (neu: `src/lib/sigorest_test.go`)

Fake-sigoREST via `httptest.Server`, angesteuert über die vorhandene
Host-Umschaltung. Assertions auf JSON-Struktur, nicht auf Substrings:

- Request enthält `bare: true` und exakt den gesetzten Vorspann.
- Nach `(sigo-system-prompt "")` fehlt das Feld `system_prompt` im
  Request (golisp2 lässt es bei leerem Vorspann weg; `bare` bleibt gesetzt).
- `sigo*` liefert die Assoc-Liste mit korrekten Werten; `sigo` den reinen
  String.
- `sigo-usage` summiert über mehrere Calls, `sigo-usage-reset` nullt.
- Parallele Calls + `sigo-host`-Umschaltung sind mit `go test -race`
  sauber.

## Reihenfolge

1. sigoREST: Tests → Implementierung → Commit im sigoREST-Repo.
   Deployment des Live-Dienstes macht Gerhard (root-Service).
2. golisp2: Referenz verschieben → Tests → Implementierung → Doku → Commit.

Die Reihenfolge ist robust: Ein altes sigoREST ignoriert das unbekannte
`bare`. Dann greifen Memory und Server-Prompt wieder, der Vorspann kommt
trotzdem an, Usage-Details sind 0. golisp2 bricht nicht, wenn es vor dem
Deployment läuft.

## Messung (TODO-Schritt 4, verkleinert)

Vorgehen aus TODO.md §4: Kopie von `/var/sigoREST`, Test-Server auf Port
19080 mit `-comm-log`, `GOLISP_SIGO_HOST` darauf.

- Zwei gleiche Code-Prompts direkt nacheinander. Erwartung: beim zweiten
  Call `cached-tokens > 0` — direkt in `(sigo* …)` sichtbar.
- Stichprobe: drei golisp2-spezifische Aufgaben (z. B. `setq*`,
  `parfunc`, `defstruct`), je mit und ohne Vorspann. Gerhard urteilt, ob
  die Antworten golisp2-korrekter werden.
- Danach Kopie und Log **löschen** (Prompts im Klartext).

## Doku

- `docs/sigo.md`: neue Funktionen, `bare`-Semantik, Hinweis „`cost-usd`
  ist obere Schranke ohne Cache-Rabatt“.
- `tools/gen-reference.lisp`: `*ref-docs*`-Einträge für die neuen Symbole,
  dann `docs/referenz-generiert.md` neu generieren.
- `src/embed/ki-referenz.md`: `sigo`-Abschnitt um die neuen Funktionen
  ergänzen.
- `TODO.md`: Schritt 1 abhaken; Schritt 2 und 3 bleiben offen.

## Bewusst ausgeschlossen (YAGNI)

- Cache-Preise in der Modell-CSV / exakter Rabatt
- `temperature`/`max_tokens` (TODO-Schritt 3)
- Modellwahl (TODO-Schritt 2)
- Retrieval über `/v1/embeddings`
- clientseitige Session-Zusammenfassung
- Wrapper für die Server-globalen Endpunkte `/api/system-prompt` und
  `/api/memory`
