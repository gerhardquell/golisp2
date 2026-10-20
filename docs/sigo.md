# GoLisp2 – sigoREST-Anbindung

GoLisp2 spricht mit dem sigoREST-Server:

```
Host:     http://127.0.0.1:9080 (Default)
Endpoint: POST /v1/chat/completions
```

Implementierung: `src/lib/sigorest.go` — Primitiven `sigo`, `sigo*`, `sigo-request`, `sigo-models`, `sigo-host`, `sigo-costs`, `sigo-budget`, `sigo-model-info`, `sigo-usage`, `sigo-system-prompt`.

## Konfiguration (Umgebungsvariablen)

| Env-Var | Default | Bedeutung |
|---------|---------|-----------|
| `GOLISP_SIGO_HOST` | `http://127.0.0.1:9080` | sigoREST-Host für `(sigo …)` |
| `GOLISP_SIGO_MODEL` | `zai-glm53-f` | Default-Modell, wenn `(sigo "prompt")` ohne Modell |
| `GOLISP_SIGO_TIMEOUT` | `120s` | Request-Timeout; z. B. `30s`, `5m`, `2m30s` |
| `GOLISP_SIGO_TEMPERATURE` | `0.2` | Default-`temperature` (0 < t ≤ 2); ungültig → Warnung, Default bleibt |

```bash
GOLISP_SIGO_HOST="http://mammouth:9080" ./build/golisp2 -i
GOLISP_SIGO_MODEL="cl48-o" ./build/golisp2 -e '(sigo "Erkläre TCO")'

# Lokales LLM braucht mehr Zeit
GOLISP_SIGO_TIMEOUT=300s ./build/golisp2 -e '(sigo "schreib fib in lisp" "ollama-qwen3-coder-30b")'
```

Zur Laufzeit änderbar: `(sigo-host "http://…")` oder als 4. Parameter pro Call.
Das Timeout ist für `sigo`/`sigo*` nur per Env-Var konfigurierbar;
`sigo-request` nimmt `timeout` (Sekunden) pro Call.

## Modelle

**Es gibt hier bewusst keine Modell-Tabelle.** Provider deployen laufend neue
Versionen, alte fallen weg — jede statische Liste ist innerhalb von Wochen
falsch. Die einzige Wahrheit:

```lisp
(sigo-models)   ; ~80 Modelle, ID + Shortcode-Paare, live vom Server
```

**Achtung — Name+Alias-Paarung (TODO.md, 20260813):** `(sigo-models)` kann
für dasselbe Modell zwei aufeinanderfolgende Einträge liefern (voller Name +
Kurz-Alias), z. B. `"gemini-3.1-flash-lite-image" "gem31-fltimg" ...`.
`fnSigoModels` (`src/lib/sigorest.go`) gibt 1:1 weiter, was der externe
sigoREST-Dienst unter `/v1/models` liefert — die Paarung entsteht dort, nicht
in golisp2, ein Fix gehört ins sigoREST-Projekt. Wer z. B. "die ersten N
Modelle" nehmen will (`(take 3 (sigo-models))`), bekommt sonst dasselbe
Modell doppelt statt N verschiedene — muss selbst dedupen.

**Default und Thinking-Modelle (gemessen 20261020):** Default ist
`zai-glm53-f` (glm-5.3-flash) — rund 14× billiger als `zai-glm53` und bei
golisp2-Aufgaben praktisch gleich gut (Messung unten). **Beide denken**: bei
Code-Aufgaben sind ~90 % der Completion-Tokens Reasoning. Schwere Aufgaben
gezielt an ein starkes Thinking-Modell geben, dann aber `max_tokens`
großzügig setzen: die Denkphase zählt mit, bei zu knappem Budget kommt ein
**leerer** Text mit `finish-reason` `length` zurück (beobachtet mit gemma4).
`reasoning-tokens` aus `sigo*` zeigt, wie viel gedacht wurde.

```lisp
(sigo* "Beweise …" "zai-glm53")                        ; gezielt stark
(sigo-request h)   ; mit "max_tokens" 16000 für Thinking-Modelle
```

## Aufruf

```lisp
(sigo "prompt")                                  ; Default-Modell, Default-Host
(sigo "prompt" "modell")
(sigo "prompt" "modell" "session-id")
(sigo "prompt" "modell" "session-id" "http://host:9080")
```

## Vorspann, Kontext und Kosten

Jeder `sigo`-Call schickt `"bare": true` und einen **Vorspann** als
`system_prompt`. Mit `bare` legt sigoREST **nichts** davor — kein
`memory.json`, kein Server-System-Prompt. Was das Modell sieht, bestimmt
allein golisp2.

Default-Vorspann ist die eingebettete KI-Kurzreferenz
(`src/embed/ki-referenz.md`, ~4–5k Token). Sie liegt über der
Cache-Schwelle der Provider; ab dem zweiten Call wird sie typischerweise
aus dem Cache gelesen (`cached-tokens`).

| Aufruf | Wirkung |
|---|---|
| `(sigo-system-prompt)` | aktuellen Vorspann lesen |
| `(sigo-system-prompt "text")` | Vorspann setzen |
| `(sigo-system-prompt "")` | leeren — Modell sieht nur den Prompt (plus Session-Verlauf, falls `session-id`) |
| `(sigo-reference)` | eingebettete Referenz; `(sigo-system-prompt (sigo-reference))` setzt zurück |
| `(sigo* prompt …)` | wie `sigo`, aber Hash-Tabelle: `text model finish-reason prompt-tokens completion-tokens cached-tokens reasoning-tokens cost-usd elapsed` |
| `(sigo-usage)` | Summen seit Start/Reset (Tokens, `cost-usd`, `elapsed`) plus `calls` |
| `(sigo-usage-reset)` | Summen auf 0 |

```lisp
(let ((r (sigo* "Schreibe eine Funktion quadrat.")))
  (println (gethash "text" r))
  (println "cached: " (gethash "cached-tokens" r) "  Sekunden: " (gethash "elapsed" r)))
```

**`cost-usd` ist eine obere Schranke** (bei OpenAI-kompatiblen Providern —
Anthropic-Kanäle sind derzeit nicht konfiguriert): sigoREST rechnet mit
`input_cost`/`output_cost` ohne Cache-Rabatt. Ob der Cache greift, zeigt
`cached-tokens > 0`.

Der Vorspann ist prozessweit (alle Goroutinen, auch `parfunc`) und
mutex-geschützt. Ein älteres sigoREST ohne `bare`-Unterstützung ignoriert
das Feld: dann kommt das Memory (global/Kanal) wieder dazu, und die
Detailfelder bleiben 0. Ein nicht-leerer Vorspann überschreibt den
Server-System-Prompt weiterhin — der kommt nur zurück, wenn der Vorspann
leer ist.

`finish-reason` `"length"` mit leerem `text` heißt: Das Token-Budget wurde
vom Thinking aufgebraucht.

## Rate-Limiting

`sigo` bringt automatisches Rate-Limiting mit — Schutz vor dem Circuit-Breaker
des Servers:

- **Mindestabstand:** 500 ms zwischen Calls
- **Globaler Ticker:** max. 1 Request pro 2 Sekunden

Bei sequenziellen Calls ggf. selbst pausieren:

```lisp
(sigo "Erste Frage" "cl46-s")
(sleep 2000)
(sigo "Zweite Frage" "gem25-f")
```

## Multi-Server-Verteilung

Der 4. Parameter erlaubt Lastverteilung über mehrere sigoREST-Instanzen:

```lisp
(define mammouth "http://mammouth:9080")
(define moonshot "http://moonshot:9080")
(define zai      "http://zai:9080")

;; 6-Hüte-Modell, parallel und verteilt
(parfunc sechs-huete
  (sigo "Fakten..."  "cl46-s"  "" mammouth)   ; Weiß
  (sigo "Gefühl..."  "gem25-f" "" moonshot)   ; Rot
  (sigo "Risiken..." "gpt41"   "" zai)        ; Schwarz
  (sigo "Chancen..." "cl46-s"  "" mammouth)   ; Gelb
  (sigo "Ideen..."   "gem25-f" "" moonshot)   ; Grün
  (sigo "Meta..."    "gpt41"   "" zai))       ; Blau
```

Ohne Host-Parameter gilt der Default-Host (`sigo-host`).

## Das selbsterweiternde Muster

```lisp
;; KI schreibt Code → GoLisp2 führt ihn aus
(eval (read (sigo "schreibe (defun fib (n) ...)" "cl46-s")))
(fib 10)

;; Ensemble: mehrere KIs parallel
(parfunc antworten
  (sigo "problem" "cl46-s")
  (sigo "problem" "gem25-f")
  (sigo "problem" "gpt41"))
```

Damit das trägt, muss `(eval …)` im globalen Environment auswerten
(`Env.Root()`) — sonst verschwinden die Definitionen im Child-Env.
Siehe Invarianten in `CLAUDE.md`.

**Prompt-Hygiene:** Den Prompt so formulieren, dass die KI *nur* Lisp-Code
zurückgibt, ohne Erklärungen und ohne Markdown-Fences:

```lisp
(sigo "Schreibe nur den Lisp-Code, keine Erklärungen, kein Markdown: defun fib …")
```

## Messen und Kosten

`(sigo-request h [host])` schickt einen frei gebauten Chat-Request. `h` ist
eine Hash-Tabelle mit **genau** den Feldern, die sigoREST kennt:
`model messages temperature max_tokens session_id timeout retries
system_prompt bare channel`. Jedes andere Feld ist ein Fehler — sigoREST
würde es still verwerfen (z. B. `response_format`). `stream` ist gesperrt,
solange sigoREST Streaming-Kosten falsch bucht. Fehlen `bare`,
`system_prompt`, `timeout` oder `temperature`, setzt golisp2 die Defaults
(`t`, aktueller Vorspann, `GOLISP_SIGO_TIMEOUT`, `GOLISP_SIGO_TEMPERATURE`).
Ohne golisp2-Default nähme sigoREST die Mitte des Modellbereichs (meist 1.0 —
zu hoch für Code); es begrenzt die temperature auf den Modellbereich,
Modelle mit fester temperature bekommen immer ihren Wert. `temperature` 0 ist
ein Fehler, weil sigoREST 0 als „nicht gesetzt" liest (→ Modellmitte); für
„so deterministisch wie möglich" `0.01` nehmen. `max_tokens` setzt golisp2
**nicht**: sigoREST nimmt dann das Modell-Maximum aus seiner Liste — das
großzügigste sichere Budget. Ein größerer Wert geht ungeprüft an den Provider
und kann dort mit 400 scheitern. Abschalten geht nur mit `:false`
(`(puthash "bare" h :false)`); `()` würde als `[]` gesendet und von sigoREST
abgewiesen. Ergebnis: die volle Antwort als
Hash-Tabelle plus `"elapsed"` — Sekunden nur für den HTTP-Aufruf, ohne
golisp2s eigene Drossel.

```lisp
(let ((h (make-hash-table :test 'equal)))
  (puthash "model" h "ci-gpt-6-luna")
  (puthash "messages" h (list (json-parse "{\"role\":\"user\",\"content\":\"2+2?\"}")))
  (puthash "max_tokens" h 50)
  (puthash "temperature" h 0)
  (puthash "timeout" h 300)
  (let ((r (sigo-request h)))
    (list (gethash "elapsed" r) (gethash "cost_usd" (gethash "usage" r)))))
```

Kosten und Budget:

| Aufruf | Wirkung |
|---|---|
| `(sigo-usage)` / `(sigo-usage-reset)` | Summen dieses Prozesses (Tokens, `cost-usd`, `elapsed`, `calls`) |
| `(sigo-costs [seit [bis]])` | `/api/costs` — serverweit, `seit`/`bis` in Unix-Sekunden wie `(now)`; ohne Argument: heute |
| `(sigo-budget)` | `/api/budget` — Hash mit `config` (Tageslimit) und `status` (Verbrauch, `blocked`), z. B. `(gethash "blocked" (gethash "status" (sigo-budget)))` |
| `(sigo-model-info [modell])` | `/api/models` — alle Modelle oder eins per `id`/`shortcode`; `input_cost`/`output_cost` in USD pro 1 Mio. Tokens |

**Preis 0 heißt „unbekannt“, nicht „gratis“.** Rund die Hälfte der Modelle
hat in sigoREST keinen Preis; für sie ist `cost-usd` immer 0. Wer sparen
will, prüft vorher `(sigo-model-info m)`. Messfenster:

```lisp
(let ((t0 (now)))
  (sigo* "…" "ci-gpt-6-luna")
  (gethash "total_cost_usd" (sigo-costs t0)))
```

HTTP 402 von sigoREST (Budget-Stopp) kommt als Fehler
`sigo-request: HTTP 402: …` an.

### Messung 20261020 (TODO Schritt 4)

Je 6 golisp2-Aufgaben × 2 Durchläufe, `temperature` 0.2, per `pool-map 4`
parallel; jede Antwort wurde geladen und gegen einen Testfall ausgeführt
(Regex, Vektoren, `typecase`, Hash-Tabelle, `format`, Listen).

| | `zai-glm53-f` | `zai-glm53` |
|---|---|---|
| korrekt | 11/12 | 12/12 |
| Markdown-Fences trotz Verbot | 4/12 | 0/12 |
| Reasoning-Tokens gesamt | 9 137 | 12 546 |
| Kosten gesamt | 0,010 USD | 0,143 USD |
| Zeit je Aufruf | 8–24 s | 8–48 s |

- **Cache greift:** Ab dem zweiten Aufruf je Modell sind 7 488 von ~7 520
  Prompt-Tokens gecacht (99,5 %). Die ersten parallelen Aufrufe treffen den
  Cache noch nicht (66 % über alles) — vor einem großen `pool-map` einen
  einzelnen Aufruf vorausschicken, dann ist der Vorspann warm.
- **Thinking dominiert** Laufzeit und Completion-Kosten, auch beim „flash"-Modell.
- **Der eine Fehlschlag** war korrektes CL: `(mapcar 'string->number …)` —
  golisp2 akzeptiert gequotete Symbole als Funktion nur in `funcall`, nicht in
  `mapcar`/`apply`/`reduce` (TODO.md).
- **Fences:** `zai-glm53-f` hält sich schlechter an „kein Markdown" — vor
  `(eval (read …))` Fences entfernen, z. B. `(regex-replace "(?m)^```.*$" text "")`.

### Kostenabgleich `cost-usd` ↔ `/api/costs` (20261020)

Kontrolliert gemessen: Zeitpunkt merken, 4 Calls (3× `zai-glm53-f`,
1× `zai-glm53`), dann `(sigo-costs t0)`. Ergebnis: **exakt gleich** —
4 Requests, 0,009657 USD, auch je Modell. golisp2 reicht `usage.cost_usd`
unverändert durch; sigoREST verbucht denselben Wert.

**Beides ist eine obere Schranke:** sigoREST hat für die `zai`-Modelle keinen
Cache-Preis (`cached_input_cost` = 0 heißt „kein bekannter Rabatt") und
rechnet gecachte Prompt-Tokens zum vollen Eingabepreis. Ein Call mit 7 488 von
7 529 Tokens im Cache kostet laut Abrechnung so viel wie einer ohne Cache.
Die echte Provider-Rechnung liegt bei ~99 % Cache-Treffern vermutlich deutlich
darunter. Korrigieren lässt sich das nur in sigoREST (Spalte
`cached_input_cost` der Modell-CSV) mit dem echten Cache-Preis des Providers.
