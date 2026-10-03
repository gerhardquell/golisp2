# sigo-request, sigo*/sigo-usage als Hash-Tabelle, Kostenabfragen — Design

**Datum:** 2026-10-03
**Status:** von Gerhard abschnittsweise freigegeben (Brainstorming 20261003)
**Betrifft:** `src/lib/sigorest.go` (einziger HTTP-Weg zu sigoREST),
`src/lib/sigorest_test.go`, Doku (`src/embed/ki-referenz.md`,
`tools/gen-reference.lisp`, `src/embed/swank.lisp`, `docs/sigo.md`)

## Ziel

golisp2 soll sigoREST für **strukturierte Antworten** und **Messungen**
nutzen können und dabei **Kosten messen**, um kostengünstig zu arbeiten.
Heute ist der Request starr (eine User-Nachricht, Modell, Session, Host),
`sigo*` liefert eine Alist (seit 20261003 nicht mehr als JSON kodierbar),
und Kosten/Budget/Preise sind aus Lisp nicht abfragbar.

## Leitlinien

- **Ein HTTP-Weg:** alles läuft durch `src/lib/sigorest.go` (Chokepoint
  laut CLAUDE.md).
- **Eine JSON-Abbildung:** Request und Antwort laufen über `jsoncell.go`
  (Objekt ↔ Hash-Tabelle, `:test 'equal`, String-Keys).
- **Laut statt still:** Felder, die sigoREST nicht kennt, sind in golisp2
  ein Fehler — sigoREST würde sie beim Dekodieren still verwerfen.

## A. `(sigo-request h [host])`

**Eingabe:** Hash-Tabelle mit String-Keys. Erlaubt sind genau die Felder
von sigoREST `ChatRequest` (`sigoREST/main.go:513`):

`model` `messages` `temperature` `max_tokens` `session_id` `timeout`
`retries` `system_prompt` `bare` `channel`

- Pflicht: `model` (String), `messages` (Liste). Die Typen der übrigen
  Felder prüft sigoREST (HTTP 400 → Fehler in golisp2).
- Jeder andere Key ist ein Fehler:
  `sigo-request: Feld 'response_format' kennt sigoREST nicht`.
- `stream` ist ausdrücklich gesperrt (eigene Meldung), solange sigoREST
  bei Streaming falsch bucht (`sigoREST/TODO-20261003-kosten.md`).
- Nicht-String-Key → Fehler.
- `host` (optional, String): wie das 4. Argument von `sigo`; Default
  `(sigo-host)`.

**Defaults** (nur wenn der Key fehlt; die Eingabetabelle wird nicht
verändert, golisp2 arbeitet auf einer Kopie):

| Key | Default |
|---|---|
| `bare` | `t` |
| `system_prompt` | aktueller `(sigo-system-prompt)`, nur wenn nicht leer |
| `timeout` | `GOLISP_SIGO_TIMEOUT` (Standard 120 s) |

`timeout` (Zahl > 0, Sekunden) gilt exakt für golisp2s HTTP-Client und
geht aufgerundet auf ganze Sekunden an sigoREST (dort `int`). Nicht-Zahl
oder ≤ 0 ist ein Fehler.

**Ablauf:** Rate-Limiter und 500-ms-Mindestabstand wie bisher → Request per
`CellToJSON` → POST `/v1/chat/completions` → Antwort per `JSONToCell`.

**Ausgabe:** die vollständige sigoREST-Antwort als Hash-Tabelle, ergänzt um
`"elapsed"` (Float-Sekunden). `elapsed` misst **nur den HTTP-Aufruf**
(Senden bis Antwort gelesen), nicht golisp2s eigene Drossel.

**Usage-Summe:** `usage` der Antwort (`prompt_tokens`, `completion_tokens`,
`prompt_tokens_details.cached_tokens`,
`completion_tokens_details.reasoning_tokens`, `cost_usd`) plus `elapsed`
fließen in die Summe für `(sigo-usage)`. Fehlende Felder zählen als 0.

**Fehler:**
- HTTP ≠ 200 → `sigo-request: HTTP <code>: <body>` (402 = Budget-Stopp).
- Timeout → `sigo-request: Timeout nach N s`.
- Verbindungsfehler → `sigo-request: <err>`.
- Antwort kein Objekt oder ohne nichtleeres `choices` → Fehler.
- Fehlgeschlagene Calls zählen nicht in `(sigo-usage)`.

## B. `sigo`, `sigo*`, `sigo-usage` auf `sigo-request`

**`(sigo "prompt" [model] [session-id] [host])`** — Signatur und Ergebnis
(Antworttext) bleiben. Baut `model`, `messages` (eine User-Nachricht),
optional `session_id` und ruft `sigo-request`. **Neu:** Prompt, Modell,
Session und Host müssen Strings sein, sonst Fehler (heute schickt
`(sigo 5)` still einen leeren Prompt).

**`(sigo* …)`** — Signatur wie `sigo`, Ergebnis **Hash-Tabelle** (Breaking;
keine bekannten Lisp-Nutzer) mit Keys:

`"text"` `"model"` `"finish-reason"` `"prompt-tokens"` `"completion-tokens"`
`"cached-tokens"` `"reasoning-tokens"` `"cost-usd"` `"elapsed"`

Bindestrich-Keys, weil `sigo*` eine aufbereitete Lisp-Zusammenfassung ist;
die Rohform liefert `sigo-request`. `model` fällt auf das angefragte Modell
zurück, wenn die Antwort keins nennt (wie bisher).

**`(sigo-usage)`** — Hash-Tabelle mit denselben Token-/Kosten-Keys plus
`"calls"` und `"elapsed"` (Summe der Antwortzeiten). `(sigo-usage-reset)`
unverändert, setzt auch `elapsed` zurück.

## C. Kostenabfragen

Alle drei liefern die sigoREST-JSON-Antwort roh (ohne Umbenennung) als
Hash-Tabelle bzw. Liste; HTTP ≠ 200 → Fehler.

- **`(sigo-budget)`** → GET `/api/budget`.
- **`(sigo-costs [seit [bis]])`** → GET `/api/costs?since=…&until=…`.
  `seit`/`bis` sind Unix-Sekunden (wie `(now)`), golisp2 wandelt in RFC3339
  (Ortszeit). Ohne Argument: sigoREST-Default (heute ab 0 Uhr).
  Nicht-Zahl → Fehler.
- **`(sigo-model-info [modell])`** → GET `/api/models`. Ohne Argument:
  Liste aller Modelle als Hash-Tabellen. Mit Argument: das Modell, dessen
  `id` **oder** `shortcode` gleich `modell` ist; kein Treffer → Fehler.
  Preise `input_cost`/`output_cost` sind USD pro 1 Mio. Tokens.

**`(sigo-models)`** bleibt (Liste der IDs aus `/v1/models`), prüft aber
künftig den HTTP-Status und meldet Lesefehler.

**Bekannte Grenze (nur dokumentiert):** 89 von 192 Modellen haben in
sigoREST keinen Preis; für sie ist `cost_usd` bzw. `"cost-usd"` = 0 und
heißt „unbekannt“, nicht „gratis“. Wer sparen will, prüft vorher
`(sigo-model-info m)`. Behebung in sigoREST (`cost_usd: null`) steht im
sigoREST-`TODO.md`. `cost_usd` ist zudem eine Obergrenze ohne Cache-Rabatt.

## Nicht-Ziele

- `response_format`/JSON-Modus — sigoREST kennt das Feld nicht
  (sigoREST-`TODO.md`). Bis dahin per Prompt.
- Streaming (`stream`) — gesperrt, siehe oben.
- Getrennte Ausweisung der Wartezeit durch golisp2s Drossel.
- Ein Preis-Cache in golisp2 oder automatische Kostenschätzung bei fehlendem
  Preis.
- `/v1/embeddings`, `/v1/messages`, weitere sigoREST-Endpunkte.

## Tests

Go-Tests in `src/lib/sigorest_test.go` gegen einen `httptest`-Fake-Server
(Muster `fakeSigo`), kein Live-Dienst:

- `sigo-request`: Whitelist (unbekannter Key, `stream`, Nicht-String-Key,
  fehlendes `model`/`messages` → Fehler); Defaults (`bare`,
  `system_prompt`, `timeout`) landen im gesendeten Body, explizite Werte
  überschreiben sie; Eingabetabelle bleibt unverändert; `elapsed` > 0;
  Host-Argument; HTTP 402 → Fehler mit Body; Timeout greift (Fake-Server
  schläft länger als `timeout`).
- `sigo`: Typfehler bei Nicht-String; Body enthält eine User-Nachricht.
- `sigo*`: Hash mit allen neun Keys und Werten aus der Fake-Usage.
- `sigo-usage`: Summe inkl. `calls`/`elapsed`, Reset; fehlgeschlagene
  Calls zählen nicht.
- `sigo-budget`/`sigo-costs`/`sigo-model-info`: Pfad und Query (`since`/
  `until` als RFC3339) beim Fake-Server; Treffer per id und shortcode;
  kein Treffer → Fehler; HTTP ≠ 200 → Fehler.
- `sigo-models`: HTTP ≠ 200 → Fehler.
- Vorhandene Tests (`TestSigoStar_ReturnsAlist` u. a.) werden auf die
  Hash-Tabelle umgestellt; Race-Test bleibt.

Danach: `go test ./... -count=1`, `./build/golisp2 -t`, ein Live-Lauf gegen
sigoREST (`sigo*`, `sigo-costs`, `sigo-model-info`) als Abnahme.

## Doku

`src/embed/ki-referenz.md` (sigoREST-Zeile: `sigo-request`, Hash-Ergebnisse,
Kostenfunktionen, Preis-0-Grenze), `tools/gen-reference.lisp` +
Neugenerierung, Arglisten in `src/embed/swank.lisp`, `docs/sigo.md`
(Abschnitt Messen und Kosten).
