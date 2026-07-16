# GoLisp2 – sigoREST-Anbindung

GoLisp2 spricht mit dem sigoREST-Server:

```
Host:     http://127.0.0.1:9080 (Default)
Endpoint: POST /v1/chat/completions
```

Implementierung: `lib/sigorest.go` — Primitiven `sigo`, `sigo-models`, `sigo-host`.

## Konfiguration (Umgebungsvariablen)

Analog zu `GOLISP_HOST`/`GOLISP_PORT` für `golisp2d`:

| Env-Var | Default | Bedeutung |
|---------|---------|-----------|
| `GOLISP_SIGO_HOST` | `http://127.0.0.1:9080` | sigoREST-Host für `(sigo …)` |
| `GOLISP_SIGO_MODEL` | `gem25-flt` | Default-Modell, wenn `(sigo "prompt")` ohne Modell |
| `GOLISP_SIGO_TIMEOUT` | `120s` | Request-Timeout; z. B. `30s`, `5m`, `2m30s` |

```bash
GOLISP_SIGO_HOST="http://mammouth:9080" ./build/golisp2 -i
GOLISP_SIGO_MODEL="cl48-o" ./build/golisp2 -e '(sigo "Erkläre TCO")'

# Lokales LLM braucht mehr Zeit
GOLISP_SIGO_TIMEOUT=300s ./build/golisp2 -e '(sigo "schreib fib in lisp" "ollama-qwen3-coder-30b")'
```

Zur Laufzeit änderbar: `(sigo-host "http://…")` oder als 4. Parameter pro Call.
**Das Timeout ist nur per Env-Var konfigurierbar.**

## Modelle

**Es gibt hier bewusst keine Modell-Tabelle.** Provider deployen laufend neue
Versionen, alte fallen weg — jede statische Liste ist innerhalb von Wochen
falsch. Die einzige Wahrheit:

```lisp
(sigo-models)   ; ~80 Modelle, ID + Shortcode-Paare, live vom Server
```

## Aufruf

```lisp
(sigo "prompt")                                  ; Default-Modell, Default-Host
(sigo "prompt" "modell")
(sigo "prompt" "modell" "session-id")
(sigo "prompt" "modell" "session-id" "http://host:9080")
```

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
