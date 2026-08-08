# webserv — Design

**Datum:** 2026-08-08 · **Status:** genehmigt (Brainstorming mit Gerhard)
**Autor:** Gerhard Quell · **CoAutor:** Claude Sonnet 5

## Ziel

Ein-Aufruf-Bootstrap für die Web-Bridge. Heute braucht ein lauffähiges
GUI-Fenster 4-5 Schritte von Hand: `http-serve` + `http-static`/eigene
HTML-Datei mit manuellem `<script src="/_golisp/boot.js">` + `browser-open`
+ `http-wait`. Zu umständlich für den eigentlichen Zweck — interaktive GUIs
(XHTML/CSS, epub3-artig: eine Datei) direkt aus golisp2 heraus zeigen.

`webserv` fasst das zu einem Aufruf zusammen. Bestehende Primitiven
(`http-serve`, `ws-export`, `ws-emit`, `ws-eval`, `ws-call`, `ws-clients`)
bleiben unverändert — `webserv` ist reiner Komfort-Wrapper obendrauf, kein
zweiter Server-Aufbau-Pfad. Baut intern auf `fnHTTPServe` auf.

## Entscheidungen (aus Brainstorming)

1. **Scope-Split:** Ein JSF/PrimeFaces-artiges Komponentensystem mit
   `#{...}`-Data-Binding wäre der eigentliche Wunsch hinter der Anfrage,
   ist aber ein eigenes, großes Vorhaben (Renderer, Converter-Registry,
   Component-Lifecycle) — explizit Stufe-2+-Gebiet, kein Teil dieser Spec.
   `webserv` löst den akuten Schmerz (Boilerplate) jetzt; das
   Komponentensystem bleibt eigenes Folge-Kapitel ("Winterprojekt"),
   voraussichtlich realisierbar als reine Lisp-Makro-DSL auf `webserv`/
   `ws-*`/`jsoncell.go`, ohne neuen Go-Kern.
2. **RPC-API unangetastet:** `ws-export`/`ws-emit`/`ws-call`/`ws-clients`
   bleiben exakt wie heute — Live-Redefinition von Handlern zur Laufzeit
   (Kernfeature der Web-Bridge) bleibt erhalten.
3. **`:htmlpath`-Reload:** Datei wird bei **jedem** Request frisch von
   Platte gelesen, kein Caching. Editieren + Browser-Reload zeigt sofort
   den neuen Stand, kein Server-Restart nötig — passt zum Live-Charakter
   der Bridge.
4. **Signatur einheitlich Keyword-basiert:** `:html` und `:htmlpath` als
   gleichwertige, sich ausschließende Keywords — kein Mix aus
   positional/keyword-Argumenten.

## API

```lisp
(webserv &key port html htmlpath open)
```

- `:port N` — optional, Default `0` (freier Port vom OS, wie `http-serve`)
- `:html "<html>...</html>"` — Inline-Content (eine Datei, epub3-artig)
- `:htmlpath "/pfad/seite.html"` — Datei-Content, frisch gelesen pro Request
- genau eines von `:html` / `:htmlpath` nötig
- `:open nil` — optional, Default `t` (Browser automatisch öffnen via
  `browser-open`); `:open nil` für Tests/Skripte ohne Browser
- Rückgabe: Server-Objekt, identisch zu `http-serve` — `ws-export` etc.
  funktionieren direkt darauf weiter

**Beispiel:**

```lisp
(define s (webserv :port 8083 :htmlpath "/u/websites/godo.html"))
(ws-export s "ask" (lambda (c frage) (string-append "Echo: " frage)))
(http-wait s)
```

## boot.js-Injektion

`/_golisp/boot.js` ist bereits immer erreichbar (bestehende
`registerWSRoute`, unabhängig von `webserv`). `webserv` prüft den
HTML-Content auf `/_golisp/boot.js`; fehlt der Script-Tag, wird
`<script src="/_golisp/boot.js"></script>` automatisch vor `</head>`
eingefügt (kein `<head>` gefunden → vor Content-Anfang). Autor muss nie
mehr manuell dran denken — `golisp.call`/`golisp.on` funktionieren ohne
Zutun.

## Fehlerfälle

| Situation | Verhalten |
|---|---|
| weder `:html` noch `:htmlpath` gesetzt | `error` beim `webserv`-Aufruf |
| beide gesetzt | `error` beim `webserv`-Aufruf (Eindeutigkeit erzwingen) |
| `:htmlpath` zeigt ins Leere (Request-Zeit) | 404, kein Server-Crash — Datei kann noch nicht existieren, wenn Server vorab gestartet wird |
| unbekanntes Keyword | `error` |

## Go-Teil

Neu: `lib/webserv.go`, ~100-150 Zeilen.

- `RegisterWebservFuncs(env *Env)`, aus `BaseEnv()` aufgerufen
  (`lib/primitives.go`).
- `fnWebServ(env, args)`: Keyword-Parsing (Muster wie `fnHTTPWait`), ruft
  intern `fnHTTPServe` für Listener+Mux, registriert `"/"`-Handler
  (Inline-String oder Datei-frisch-lesen je nach Modus, boot.js-Injektion),
  ruft optional `fnBrowserOpen` mit der fertigen URL.
- Kein neuer HTTP-Client, kein neuer Parser, keine neue Truthiness-Logik —
  Chokepoints unberührt.

## Dateien

| Datei | Änderung |
|---|---|
| `lib/webserv.go` | neu, `webserv`-Primitiv |
| `lib/webserv_test.go` | neu, Go-Test: `:html`/`:htmlpath`-Modus, boot.js-Injektion, `:htmlpath`-Reload zwischen zwei Requests, Fehlerfälle |
| `lib/primitives.go` | `RegisterWebservFuncs`-Aufruf in `BaseEnv()` |
| `tests/webbridge.lisp` | Ergänzung: `webserv`-Smoke-Test (`:open nil`) |
| `doc/struktur.md` | Eintrag für `lib/webserv.go` nachtragen |

## Verifikation

`./build.sh` · `go test ./...` · `./build/golisp2 -t` · manueller
Smoke-Test: `(webserv :htmlpath "...")` im REPL, Browser öffnet sich,
`ws-export`-Callback funktioniert ohne eigenes Script-Tag.

## Bewusst nicht (YAGNI)

- HTTPS/TLS — weiterhin nur `127.0.0.1`, wie ganze Web-Bridge heute
- Komponentensystem (JSF/PrimeFaces-Stil, `#{...}`-Binding, Converter,
  Renderer) — eigenes Folge-Spec
- Mehrteilige Sites (eigene `.css`/`.js`/Bilder-Dateien) — `http-static`
  bleibt dafür die bestehende Lösung, `webserv` deckt nur Ein-Datei-Fall ab
- Hot-Reload für `:html`-Inline-Modus (Content kommt aus laufendem
  Lisp-Prozess, ist per Definition schon "frisch")
