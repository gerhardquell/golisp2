# GoLisp – CLAUDE.md

## Projekt-Übersicht
GoLisp ist ein Lisp-Interpreter in Go mit nativer KI-Anbindung (sigoREST).
Ziel: Ein selbsterweiterndes System das Goroutinen, Channels und KI-Calls
als eingebaute Lisp-Primitiven beherrscht.

**Autor:** Gerhard Quell – gquell@skequell.de
**CoAutor:** claude sonnet 4.6
**Modul:** `golisp`
**Sprache:** deutsch
---

## Dateistruktur

```
golisp/
  main.go              Unix-Style CLI: stdin/-i/-e/-t/Datei + Exit-Codes
  cmd/
    golispd/           Server-Binary
      main.go          TCP-Server Entry Point
    golisp-client/     Client-Binary
      main.go          CLI-Client mit REPL
  lib/
    types.go           Cell-Datenstruktur (LispType, Cons, MakeAtom...)
    types_helpers.go   Hilfsfunktionen: SliceToCell, Append, CellToSlice
    reader.go          Parser: String → Cell-Baum (NewReader, Read)
    env.go             Umgebung: Get, Set, Update, Symbols (verkettete Scopes)
    eval.go            Herzstück: Eval, Spezialformen, defmacro, parfunc
    primitives.go      Eingebaute Funktionen + BaseEnv()
    stringfuncs.go     String-Primitiven (RegisterStringFuncs)
    format.go          FORMAT-Engine (fnFormat, formatRun, Parameter-Parser)
    format_dirs.go     FORMAT-Direktiven + Helper (~A~S~D~R~F~$~T~* — einfache Direktiven)
    format_blocks.go   FORMAT-Block-Direktiven (~?~[~{~(~^~/fun/ + findBlock/splitClauses)
    goroutine.go       parfunc, chan-make/send/recv, lock-make
    fileio.go          file-write, file-append, file-read, file-exists?, file-delete
    sigorest.go        sigo, sigo-models, sigo-host (HTTP zu sigoREST)
    readline.go        REPL: go-prompt, Syntax-Highlighting, History, Multiline
    env_test.go        Go-Tests für Env.Symbols()
    swank/             SWANK-Server für Emacs/SLIME
      server.go        TCP-Listener, per-Connection Handling
      framing.go       Length-prefixed Framing (readFrame/writeFrame)
      dispatch.go      (swank-dispatch msg) Wrapper
      env.go           per-Connection Primitives (send-event, value-string)
      lisp.go          //go:embed swank.lisp, LoadSwankLisp
      swank.lisp       Semantische Handler (connection-info, listener-eval …)
```

---

## Coding-Konventionen

- **Sprache:** Go für den Kern, Lisp für Erweiterungen
- **Einrückung:** 2 Spaces, keine Tabs
- **Dateinamen:** camelCase (außer main.go)
- **Kommentare:** sparsam, sprechende Namen bevorzugt
- **Dateigröße:** max 1000 Zeilen, ab 800 sinnvoll aufteilen
- **Datei-Header:** immer mit Autor, CoAutor, Copyright, Erstellt (YYYYMMDD)
- **Fehler:** `fmt.Errorf("funktionsname: beschreibung")`
- **Build:** verwende `./build.sh` (kompiliert golisp/golispd/golisp-client nach `./build/`)
- **tmp:** verwende ./tmp als temporäres Verzeichnis - nicht /tmp !!

### Spezialformen vs. Primitiven
- Braucht die Funktion Zugriff auf `env`? → Spezialform in `eval.go`
- Reine Berechnung ohne env? → Primitiv in `primitives.go`
- Gruppe verwandter Primitiven → eigene Datei mit `RegisterXxx(env *Env)`
- Neue Primitiven immer in `BaseEnv()` registrieren

---

## Architektur

### Cell – die Grundstruktur
```go
type Cell struct {
  Type LispType        // ATOM, NUMBER, STRING, LIST, FUNC, MACRO, NIL
  Val  string          // für ATOM und STRING
  Num  float64         // für NUMBER
  Car  *Cell           // Kopf einer Liste
  Cdr  *Cell           // Rest einer Liste
  Fn   func([]*Cell) (*Cell, error)  // für FUNC
  Env  interface{}     // für Lambda-Closures (*Env) und Go-Objekte
}
```

### Eval-Reihenfolge in evalList
```
1. Spezialformen prüfen (quote, if, define, defun, lambda,
   let, begin, set!, defmacro, mapcar, load, and, or, not,
   parfunc, lock, eval, catch, while, do, quasiquote, cond)
2. Makro-Expansion (MACRO-Typ → expand → Eval des Ergebnisses)
3. Normale Anwendung: Funktion auswerten → Argumente auswerten → apply
```

### TCO – Trampolin in Eval()
Lambda-Calls und alle Tail-Spezialformen (`if`, `begin`, `let`, `cond`)
setzen `expr`/`env` und machen `continue` im `for {}`-Loop — kein neuer
Stack-Frame, O(1) Stack für beliebig tiefe Tail-Rekursion.

### `(eval form)` – globales Environment
`(eval form)` wertet im **globalen** Environment aus (`Env.Root()`),
nicht im dynamischen Lambda-Scope — Common-Lisp-Semantik. Notwendig,
damit Definitionen aus `(eval (read ...))` (REPL `swank:listener-eval`,
selbsterweiterndes Muster) global sichtbar bleiben und nicht im
Child-Env der aufrufenden Lambda-Kette verloren gehen.

### Lambda-Struktur
```go
// Lambda/Closure wird als Cell{Type:LIST} gespeichert:
Cell{Type: LIST, Car: params, Cdr: body, Env: closureEnv}
// Makro: identisch aber Type: MACRO
```

### Multi-Body: wrapBegin
`defun`, `lambda`, `defmacro` akzeptieren mehrere Body-Ausdrücke.
`wrapBegin(exprs)` wrappet sie zur Definitionszeit in `(begin ...)`.
Einzelner Ausdruck → direkt, kein Overhead.

---

## Implementierte Features

### Spezialformen
`quote` `if` `define` `setq` `defun` `lambda` `let` `let*` `begin` `set!` `setq*`
`defmacro` `mapcar` `load` `and` `or` `not` `parfunc` `lock` `eval`
`catch` `while` `do` `quasiquote` `cond` `case`

### Eingebaute Funktionen
**Arithmetik:** `+` `-` `*` `/`
**Vergleiche:** `=` `<` `>` `>=` `<=` `eq` `eq?` `equal?`
**Typ-Prädikate:** `string?` `number?` `list?` `symbol?` `atom?` `null?`
**Listen:** `car` `cdr` `cons` `atom` `null` `list` `apply`
**I/O:** `print` `println` `read`
**String:** `string-length` `string-append` `substring`
  `string-upcase` `string-downcase` `string->number` `number->string`
  `string->list` `list->string`
**Format:** `format` (Common-Lisp-HyperSpec 22.3 — `~A ~S ~D ~B ~O ~X ~R ~P ~C
  ~F ~E ~G ~$ ~% ~& ~| ~T ~* ~? ~[ ~{ ~( ~; ~^ ~/fun/ ~~ ~Newline` mit
  Parametern + Modifiern `:` `@`; dest = `t`→stdout, `nil`→String, String→anhängen;
  `~/name/` ruft benannte Funktion via globalFormatEnv, `~:;` markiert Default-Klausel)
**Fehler:** `error` `catch`
**Makro-Hilfe:** `gensym`
**Datei:** `file-write` `file-append` `file-read` `file-exists?` `file-delete`
**Nebenläufigkeit:** `chan-make` `chan-send` `chan-recv` `lock-make`
**KI:** `sigo` `sigo-models` `sigo-host`
**Zeit:** `sleep`
**Memory:** `memstats`

### REPL (readline.go) – `golisp -i`
- **Start:** `./golisp -i` (benötigt TTY – im Script/CI kommt Fehlermeldung)
- **Syntax-Highlighting:** Klammern nach Tiefe eingefärbt (6 Farben, fett)
  Strings grün · Kommentare grau · Quote-Zeichen gelb
- **Multi-line:** Enter bei offenem Ausdruck → automatische Einrückung
- **History:** persistent `~/.golisp_history` (500 Einträge)
- **Library:** `github.com/elk-language/go-prompt`

---

## Unix-Style CLI

GoLisp verhält sich wie ein typisches Unix-Tool:

| Flag | Beschreibung | Beispiel |
|------|--------------|----------|
| *(default)* | Liest von stdin, gibt nur Ergebnis aus | `echo "(+ 1 2)" \| ./golisp` |
| `-i` | Interaktiver REPL mit go-prompt | `./golisp -i` |
| `-e EXPR` | Expression direkt ausführen | `./golisp -e "(* 6 7)"` |
| `-t` | Tests ausführen | `./golisp -t` |
| `--swank HOST:PORT` | SWANK-Server starten (für Emacs/SLIME) | `./golisp --swank 127.0.0.1:4242` |
| `DATEI` | Lisp-Datei laden | `./golisp script.lisp` |

### Exit-Codes
- **0** – Erfolg
- **1** – Fehler (Parser, Eval, unbekanntes Symbol, etc.)

### Multiline-Support (stdin)
Expression wird erst ausgewertet wenn Klammern ausgeglichen sind:
```bash
cat <<'EOF' | ./golisp
(defun square (x)
  (* x x))
(square 5)
EOF
# → 25
```

### Fehlerbehandlung
- Alle Fehler gehen zu `stderr`
- Ergebnisse gehen zu `stdout`
- Bei Fehler in Pipe/Datei: weitere Expressions werden verarbeitet, Exit-Code 1

---

## GoLisp Server (golispd) – SWANK-ähnlicher TCP-Server

GoLisp kann als Server laufen, der via TCP S-Expression-RPC mit Clients kommuniziert – ähnlich wie SWANK für Common Lisp.

### Architektur
```
┌─────────────┐     TCP Socket      ┌─────────────┐
│   Client    │ ◄─────────────────► │  golispd    │
│  (Emacs,    │   S-Expression RPC  │   (Server)  │
│   VS Code,  │                     └──────┬──────┘
│   CLI)      │                            │
└─────────────┘                     ┌──────┴──────┐
                                    │  GoLisp     │
                                    │  Runtime    │
                                    └─────────────┘
```

### Server starten

```bash
# Default (localhost:4321)
golispd

# Custom port
golispd --port 5000

# Umgebungsvariablen
export GOLISP_HOST=0.0.0.0
export GOLISP_PORT=5000
golispd
```

### Client-Befehle

```bash
# Ping (Health-Check)
golisp-client --ping

# Expression auswerten
golisp-client --eval "(+ 1 2 3)"
# => 6

# Autocomplete
golisp-client --complete "ca"
# => ((car . "Eingebaute Funktion") (cadr . "Lambda/Closure") ...)

# Datei laden
golisp-client --load myscript.lisp

# Interaktiver REPL
golisp-client --repl
golisp> (defun square (x) (* x x))
golisp> (square 5)
25
golisp> :quit
```

### Protokoll (S-Expression-RPC)

**Request:**
```lisp
(:id 1 :method "eval" :params ("(+ 1 2)"))
(:id 2 :method "complete" :params ("def"))
(:id 3 :method "symbols" :params ())
```

**Response:**
```lisp
(:id 1 :status "ok" :result "3")
(:id 2 :status "ok" :result (("defun" . "Lambda/Closure")))
(:id 3 :status "error" :error "unbekanntes Symbol 'x'")
```

### Unterstützte Methoden

| Methode | Beschreibung | Beispiel |
|---------|--------------|----------|
| `ping` | Health-Check | `()` → `"pong"` |
| `eval` | Code auswerten (String) | `("(+ 1 2)")` → `"3"` |
| `eval-return` | Code auswerten (Cell) | `("(list 1 2)")` → `(1 2)` |
| `complete` | Autocomplete | `("ca")` → Symbole mit Prefix |
| `symbols` | Alle Symbole listen | `()` → Liste aller Namen |
| `describe` | Dokumentation | `("car")` → Doc-String |
| `load-file` | Datei laden | `("script.lisp")` → Status |
| `disconnect` | Verbindung schließen | `()` → `()` |

### REPL-Modus Features

- **Multiline:** Automatische Fortsetzung bei offenen Klammern
- **Kommandos:** `:quit`, `:complete prefix`, `:load datei`
- **Error Recovery:** Fehler brechen REPL nicht ab
- **Shared Environment:** Alle Clients teilen denselben Zustand

### Installation

```bash
go build -o /usr/local/bin/golispd ./cmd/golispd/
go build -o /usr/local/bin/golisp-client ./cmd/golisp-client/
```

---

## SWANK-Server für Emacs/SLIME (`golisp --swank`)

Echte SWANK-Protokoll-Implementierung in `lib/swank/` — spricht das
SLIME-Protokoll direkt, so dass `M-x slime-connect` aus Emacs klappt.
(Unabhängig vom oben beschriebenen `golispd` Custom-RPC.)

### Start

```bash
./golisp --swank 127.0.0.1:4242
```

Dann in Emacs (SLIME geladen via quicklisp `slime-helper.el`):

```
M-x slime-connect  →  Host: 127.0.0.1  →  Port: 4242  →  y (Versionswarnung)
```

REPL `*slime-repl USER*` öffnet sich, Definitionen halten, Rekursion klappt.
Status: 2026-06-21 getestet mit SLIME v2.32.

### Architektur (hybrid: Go + GoLisp)

| Datei | Aufgabe |
|-------|---------|
| `lib/swank/server.go` | TCP-Listener, per-Connection `bufio.Reader`, Env-Setup |
| `lib/swank/framing.go` | Length-prefixed Framing `%06x<sexpr>` (readFrame nimmt persistenten `*bufio.Reader` — pipelining-safe) |
| `lib/swank/dispatch.go` | `(swank-dispatch (quote msg))` aufrufen |
| `lib/swank/env.go` | per-Connection Primitives: `swank-send-event`, `swank-print`, `swank-println`, `swank--value-string` |
| `lib/swank/swank.lisp` | Semantische Handler (Lisp-Seite, via `//go:embed`) |

### Implementierte SLIME-Methoden

| Op | Verhalten |
|----|-----------|
| `swank:connection-info` | Implementation-Info `(:type "GoLisp" :version "0.2" :style :spawn)` |
| `swank:swank-require` | Stub `(:ok ())` — keine Contribs geladen |
| `swank:init-presentations` | Stub `(:ok ())` |
| `swank-repl:create-repl` | `(:ok ("USER" "USER"))` + `:new-package` |
| `swank-repl:listener-eval` | Eval-Code (alle Formen via `ReadAll`) → pro Ergebnis `:write-string "<w>\n" :repl-result` + `(:ok nil)` |
| `swank:simple-completions` | `(:ok (strings...))` — Basis-Completion |
| `swank:completions` | `(:ok ((name) (name) ...))` — c-p-c Contrib (das Op das SLIME für TAB sendet) |
| `swank:load-file` | `(load filename)` → `(:ok "<result>")` — C-c C-l |
| `swank:operator-arglist` | `(:ok "(name args)")` für Lambda/Macro/Built-in, `(:ok ())` sonst — C-c C-d C-a |
| `swank:autodoc` | `(:ok (arglist-string cache-p))` für Lambda/Macro/Built-in, `(:not-available nil)` sonst |
| `swank:swank-macroexpand-1` / `swank-expand-1` | Eine Top-Level-Expansion → `(:ok "<exp>")` — C-c C-m |
| `swank:swank-macroexpand` / `swank-expand` | Wiederholt bis stabil (Top-Level) |
| `swank:swank-macroexpand-all` / `swank-expand-all` | Rekursive Expansion in alle Subformen |
| `swank:describe-symbol` | `(:ok (:title name :content text))` — statische Registry |
| `swank:compile-string-for-emacs` | `(:ok t)` nach stillem Evaluieren des Strings |
| `swank:compile-file-for-emacs` | `(:ok (:filename file :result ...))` via `(load file)` — C-c C-k |
| `swank:find-definitions-for-emacs` | Map-Lookup → `:location (:file ... :line N)`, REPL-Snippet-Fallback, Built-in `:error` — M-. |
| sonstige | graceful `(:ok ())` statt `:abort` (SLIME-Contribs degradieren sauber) |

### Wichtigste Protokoll-Details

- **`listener-eval`-Return:** `:write-string` mit `:repl-result`-Tag senden,
  dann `(:ok nil)`. Nicht `(:ok "<string>")` — SLIME destrukturiert `:ok` als
  Liste.
- **Mehrere Formen:** `listener-eval` liest alle Formen via `swank--read-all`
  (`lib.ReadAll`), evaluiert jede, sendet pro Ergebnis ein `:write-string`.
- **`completions` Format:** Liste von 1-Element-Listen `((name) ...)` — Client
  destrukturiert `(symbol-name classification symbol)`, fehlende = nil.
- **`autodoc`:** `(:not-available nil)` für Built-ins — sonst `insert nil`-
  Error in `slime-autodoc--format`. Arglist via `swank--arglist` (Lambda:
  `Type:LIST`+`Env!=nil`, Car = Parameter).
- **eval global:** `listener-eval` nutzt `(eval (read string))`; damit
  `defun` global persistiert, wertet `eval` im `Env.Root()` (siehe oben).
- **C-c C-m Cursor:** `slime-bounds-of-sexp-at-point` greift bei Cursor auf
  Symbol das Symbol — Cursor auf `(` setzen für ganze Form.
- **`find-definitions-for-emacs` (M-.):** Lookup in lib.defloc-Map (defun/defmacro/define registrieren `SrcFile`/`SrcLine`). REPL-definierte Funktionen (kein SrcFile) → Snippet-Buffer; Built-ins → `:error`.

### Offen für volle SLIME-Integration

Optional ausbaufähig (nicht im aktuellen Scope): Inspector, Debugger/Restarts,
`disassemble-symbol`. Die als offen markierten Punkte `describe-symbol`,
`macroexpand-all`, `compile-string`/`compile-file-for-emacs` und
`find-definitions-for-emacs` sind implementiert.

---

## sigoREST Anbindung

GoLisp spricht mit dem sigoREST-Server:
```
Host: http://127.0.0.1:9080 (Default)
Endpoint: POST /v1/chat/completions
```

### Konfiguration via Umgebungsvariablen

Beim Start liest GoLisp (analog `GOLISP_HOST`/`GOLISP_PORT` für `golispd`):

| Env-Var | Default | Bedeutung |
|---------|---------|-----------|
| `GOLISP_SIGO_HOST` | `http://127.0.0.1:9080` | sigoREST-Host für `(sigo ...)` |
| `GOLISP_SIGO_MODEL` | `gem25-flt` | Default-Modell wenn `(sigo "prompt")` ohne Modell |

```bash
# Multi-Server-Setup
GOLISP_SIGO_HOST="http://mammouth:9080" ./golisp -i
GOLISP_SIGO_MODEL="cl48-o" ./golisp -e '(sigo "Erkläre TCO")'
```

Zur Laufzeit zusätzlich änderbar: `(sigo-host "http://...")` oder als
4. Parameter pro Call: `(sigo "prompt" "model" "" "http://host:9080")`.

Aktuelle Modell-Shortcodes (sigoREST dynamisch via Mammouth/Moonshot/ZAI,
Stand 2026-06-16). **Modelle sind runtime-dynamisch** – Provider deployen
neue Versionen, alte fallen weg. Diese Tabelle ist eine Momentaufnahme;
die wahrheitsgemäße Quelle ist immer `(sigo-models)`.

| Shortcode | Modell | Shortcode | Modell |
|-----------|--------|-----------|--------|
| `cl4-s`   | claude-sonnet-4 | `cl45-h`  | claude-haiku-4-5 |
| `cl45-s`  | claude-sonnet-4-5 | `cl46-s`  | claude-sonnet-4-6 |
| `cl45-o`  | claude-opus-4-5 | `cl46-o`  | claude-opus-4-6 |
| `cl47-o`  | claude-opus-4-7 | `cl48-o`  | claude-opus-4-8 |
| `gem25-f` | gemini-2.5-flash | `gem25-p` | gemini-2.5-pro |
| `gem35-f` | gemini-3.5-flash | `gem3-fpv`| gemini-3-flash-preview |
| `gpt41`   | gpt-4.1 | `gpt5`    | gpt-5 |
| `gpt52`   | gpt-5.2 | `gpt52-ch`| gpt-5.2-chat |
| `gpt54-n` | gpt-5.4-nano | `gpt5-m` | gpt-5-mini |
| `kimi`    | kimi-k2.5 | `kimik26` | kimi-k2.6 |
| `kimik27-cod` | kimi-k2.7-code | `kimik27-codhs` | kimi-k2.7-code-highspeed |
| `ds32`    | deepseek-v3.2 | `ds4-p`   | deepseek-v4-pro |
| `ds4-f`   | deepseek-v4-flash | `glm5` | glm-5 |
| `glm47`   | glm-4.7 | `grok43`  | grok-4.3 |
| `llama4-mv` | llama-4-maverick | `qwen3-c52pl` | qwen3-coder-plus |

Vollständige Live-Liste (~80 Modelle, ID+Shortcode-Paare): `(sigo-models)`

### Rate-Limiting & Best Practices

`sigo` hat automatisches Rate-Limiting eingebaut:
- **Mindestabstand:** 500ms zwischen Calls
- **Globaler Ticker:** max 1 Request pro 2 Sekunden
- **Schutz vor Circuit-Breaker:** Verhindert Server-Überlastung

Für sequenzielle Calls mit Pausen:
```lisp
(sigo "Erste Frage" "cl46-s")
(sleep 2000)  ; 2 Sekunden Pause
(sigo "Zweite Frage" "gem25-f")
```

### Multi-Server Verteilung (mammouth/moonshot/zai)

`sigo` unterstützt einen optionalen 4. Parameter für den Host:
```lisp
; Syntax: (sigo "prompt" "model" "session-id" "host")
(sigo "Hallo" "cl46-s"  "" "http://mammouth:9080")
(sigo "Hallo" "gem25-f" "" "http://moonshot:9080")
(sigo "Hallo" "gpt41"   "" "http://zai:9080")
```

**Anwendung: 6-Hüte-Modell mit Lastverteilung**
```lisp
(define mammouth "http://mammouth:9080")
(define moonshot "http://moonshot:9080")
(define zai      "http://zai:9080")

(parfunc sechs-huete
  (sigo "Fakten..." "cl46-s"  "" mammouth)  ; ⚪ Weiß
  (sigo "Gefühl..." "gem25-f" "" moonshot)  ; 🔴 Rot
  (sigo "Risiken..." "gpt41"  "" zai)       ; ⚫ Schwarz
  (sigo "Chancen..." "cl46-s" "" mammouth)  ; 🟡 Gelb
  (sigo "Ideen..."  "gem25-f" "" moonshot)  ; 🟢 Grün
  (sigo "Meta..."   "gpt41"   "" zai))      ; 🔵 Blau
```

Ohne Host-Parameter wird der Default-Host (`sigo-host`) verwendet.

### Das selbsterweiternde Muster
```lisp
; KI schreibt Code → GoLisp führt ihn aus
(eval (read (sigo "schreibe (defun fib (n) ...)" "cl46-s")))
(fib 10)

; Ensemble: 3 KIs parallel
(parfunc antworten
  (sigo "problem" "cl46-s")
  (sigo "problem" "gem25-f")
  (sigo "problem" "gpt41"))
```

**Wichtig für sigo-Prompts:** Den Prompt so formulieren dass die KI
*nur* den Lisp-Code zurückgibt ohne Erklärungen – z.B.:
`"Schreibe nur den Lisp-Code, keine Erklärungen: defun fib ..."`

---

## Memory Management

GoLisp vertraut vollständig auf Go's Garbage Collector – es gibt kein
manuelles Memory-Management. Das bedeutet:

### Wie es funktioniert

- **Cell-Allokation:** Jedes `&Cell{}` landet auf Go's Heap
- **Kein Object-Pooling:** Keine `sync.Pool` oder ähnliche Optimierungen
- **Zirkuläre Referenzen:** Go's GC erkennt Zyklen (Lambdas, `labels`)
- **Singleton Nil:** `MakeNil()` gibt immer dieselbe Instanz zurück

### Memory-Statistiken

Die Funktion `(memstats)` gibt aktuelle Go-Runtime-Stats zurück:

```lisp
(memstats)
;; => ((heapalloc . 421376)       ; aktueller Heap in Bytes
;;     (heapsys . 7864320)        ; vom OS reservierter Heap
;;     (heapobjects . 1247)       ; Anzahl allozierter Objekte
;;     (numgc . 5)                ; Anzahl GC-Zyklen
;;     (pausetotalns . 234567)    ; totale GC-Pause in Nanosekunden
;;     (totalalloc . 1234567))    ; kumulative Allokation
```

### Best Practices

1. **Keine Angst vor Allokationen:** Go's GC ist für kurzlebige Objekte optimiert
2. **Externe Ressourcen schließen:** PostgreSQL-Verbindungen mit `pg-close` freigeben
3. **Globales Environment:** Wächst permanent – keine `undefine` Funktion
4. **Monitoring:** Bei Langzeit-Prozessen `(memstats)` regelmäßig loggen

### Singleton-Nil Optimierung

Vor der Optimierung: Jedes `()`, `nil`, leere Liste erzeugte eine neue Cell.
Nach der Optimierung: Alle verwenden dieselbe `nilCell` Instanz.

```lisp
(eq (list) (list))  ; => t (identische Pointer)
(eq nil nil)        ; => t (immer dieselbe Instanz)
(eq '() '())        ; => t (auch quote-nil ist identisch)
```

**Hinweis:** `eq` prüft Pointer-Gleichheit (identisches Objekt im Speicher),
während `equal?` strukturelle Gleichheit prüft (gleicher Inhalt).

**Thread-Sicherheit:** Die Singleton-Nil ist sicher für `parfunc` –
sie wird nur gelesen, nie modifiziert.

---

## eq vs equal? – Wann welchen Vergleich verwenden

| Funktion | Vergleicht | Verwendung |
|----------|------------|------------|
| `eq` / `eq?` | Pointer-Identität (identisches Objekt im Speicher) | Schneller Identitätsvergleich für Symbole, Singleton-Objekte |
| `equal?` | Strukturelle Gleichheit (rekursiver Inhaltsvergleich) | Listen, Strings, Zahlen, verschiedene Atom-Instanzen |

```lisp
;; eq prüft, ob es DASSELBE Objekt ist
(eq 'foo 'foo)           ; ()  - zwei verschiedene Atom-Instanzen
(eq (list) (list))       ; t   - Singleton-Nil, identischer Pointer
(eq 5 5)                 ; ()  - jede Zahl ist neue Cell

;; equal? prüft, ob der Inhalt gleich ist
(equal? 'foo 'foo)       ; t   - gleicher Inhalt
equal? (list 1 2) (list 1 2))  ; t   - gleiche Struktur
(equal? 5 5)             ; t   - gleicher Wert
```

**Empfehlung:** Für sichere Vergleiche immer `equal?` verwenden.
`eq` nur wenn explizite Identitätsprüfung benötigt wird.

---

## let vs let* – Parallele vs sequentielle Bindungen

| Form | Bindungsmodus | Verwendung |
|------|---------------|------------|
| `let` | Parallel – alle Werte werden im äußeren env ausgewertet | Unabhängige Variablen |
| `let*` | Sequentiell – jede Bindung sieht die vorherigen | Abhängige Variablen (z.B. `(y (+ x 1))`) |

```lisp
;; let – parallele Bindungen
(let ((x 5)
      (y (+ x 1)))      ; Fehler: x ist noch nicht gebunden!
  ...)

;; let* – sequentielle Bindungen
(let* ((x 5)
       (y (+ x 1)))     ; OK: x ist bereits 5
  y)                    ; → 6
```

---

## setq und setq* – Common Lisp Kompatibilität

`setq` ist ein Alias für `define` – setzt eine Variable global oder lokal:
```lisp
(setq x 10)              ; → x (Variable wird gesetzt)
x                        ; → 10
```

`setq*` setzt mehrere Variablen sequentiell:
```lisp
(setq* a 1
       b (+ a 1)         ; b sieht a = 1
       c (+ b 1))        ; c sieht b = 2
(list a b c)             ; → (1 2 3)
```

---

## case – Syntaktischer Zucker für cond

`case` vergleicht einen Schlüsselwert mit mehreren Alternativen:

```lisp
(case 'b
  ((a) 1)                ; einzelner Wert
  ((b c) 2)              ; Liste von Werten
  (else 3))              ; → 2

(case 5
  ((1 2 3) "klein")
  ((4 5 6) "mittel")
  (else "groß"))         ; → "mittel"
```

Der Vergleich erfolgt mit `equal?` (strukturelle Gleichheit).
`else` oder `t` als Test fungiert als Default-Fall.

---

## Build & Test

```bash
go build .                              # kompilieren (golisp)
go build ./cmd/golispd/                 # Server kompilieren
go build ./cmd/golisp-client/           # Client kompilieren
go test ./...                           # Go-Unit-Tests

# Installation
sudo cp golisp golispd golisp-client /usr/local/bin/

# CLI-Modi (golisp Hauptbinary)
go run . -t                             # Testmodus (26 Tests)
go run . -i                             # Interaktiver REPL (benötigt TTY)
go run . -e "(+ 1 2)"                   # Expression direkt ausführen
go run . skript.lisp                    # Datei ausführen
echo "(+ 1 2)" | go run .              # Stdin-Modus (Default)

# Server/Client-Modus
golispd --port 4321                     # Server starten
golisp-client --eval "(+ 1 2)"          # Client: Expression
golisp-client --repl                    # Client: Interaktiver REPL

# Exit-Codes: 0 = Erfolg, 1 = Fehler
echo "(+ 1 2)" | ./golisp; echo $?      # → 0
./golisp -e "(error 'x')"; echo $?      # → 1
```

---

## Philosophie

GoLisp ist ein **U-Boot-Projekt** – es reift in Ruhe bevor es
der Welt gezeigt wird. Ziele:

- **Nexialistisch:** verbindet Go-Effizienz + Lisp-Eleganz + KI-Power
- **Selbsterweiternd:** GoLisp kann sich durch KI-Calls selbst vervollständigen
- **Ensemble-fähig:** mehrere KIs parallel → Synthese durch Claude
- **Centaur-Ansatz:** Mensch als Meta-Entscheider, KIs als Spezialisten

> "Code = Daten + KI = sich selbst erweiterndes System"
> – Gerhard & Claude, Februar 2026
