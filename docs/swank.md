# GoLisp2 – SWANK-Server & Client

`src/lib/swank/` implementiert das **echte SWANK-Protokoll** (length-prefixed
`:emacs-rex`-RPC). `M-x slime-connect` aus Emacs funktioniert damit direkt.

`golisp2 --swank host:port` startet den Server (`src/lib/swank/server.go` →
`swank.RunServer`). Es gibt kein separates Custom-RPC. Der `golisp2-client`
spricht ebenfalls SWANK.

Status: 2026-06-21 getestet mit SLIME v2.32.

## Architektur

```
┌─────────────┐     TCP / SWANK       ┌─────────────────┐
│   Client    │ ◄──────────────────►  │ golisp2 --swank │
│ (golisp2-   │  :emacs-rex Frames    │  (SWANK-Srv)    │
│  client,    │  %06x<sexpr>          └────────┬────────┘
│  Emacs/     │                                │
│  SLIME)     │                         ┌──────┴──────┐
└─────────────┘                         │  GoLisp2    │
                                        │  Runtime    │
                                        └─────────────┘
```

| Datei | Aufgabe |
|-------|---------|
| `src/lib/swank/server.go` | TCP-Listener, per-Connection `bufio.Reader`, Env-Setup |
| `src/lib/swank/framing.go` | Framing `%06x<sexpr>` — `readFrame` nimmt einen persistenten `*bufio.Reader` (pipelining-safe) |
| `src/lib/swank/dispatch.go` | ruft `(swank-dispatch (quote msg))` auf |
| `src/lib/swank/env.go` | per-Connection Primitiven: `swank-send-event`, `swank-print`, `swank-println`, `swank--value-string` |
| `src/embed/swank.lisp` | semantische Handler auf Lisp-Seite (via `//go:embed`) |

Hybrid-Design: Transport in Go, Semantik in Lisp.

## Server starten

```bash
./build/golisp2 --swank 127.0.0.1:4242    # host:port Pflicht
./build/golisp2 --swank 5000              # kein ":" → Host defaultet auf 127.0.0.1
```

### Emacs

SLIME via quicklisp `slime-helper.el` laden, dann:

```
M-x slime-connect → Host: 127.0.0.1 → Port: 4242 → y (Versionswarnung)
```

Der REPL `*slime-repl USER*` öffnet sich. Definitionen halten, Rekursion klappt.

## Client (`golisp2-client`)

`src/cmd/golisp2-client/main.go`, nutzt `golisp2/src/lib` für robuste Cell-Verarbeitung.

| Client-Flag | SWANK-Op | Antwort |
|-------------|----------|---------|
| `--ping` | `swank:connection-info` | `:ok`-Plist |
| `--eval CODE` | `swank-repl:listener-eval` | `:write-string`-Events pro Ergebnis |
| `--complete PFX` | `swank:simple-completions` | `:ok ("m1" "m2" …)` |
| `--load FILE` | `swank:load-file` | `:ok "result"` |
| `--repl` | Schleife aus `listener-eval` | s. o. |

```bash
./build/golisp2-client --ping
# => Server ist erreichbar: pong (SWANK connection-info ok)

./build/golisp2-client --eval "(+ 1 2 3)"
# => 6

./build/golisp2-client --complete "ca"
# => car cadddr cadr caddr caar

./build/golisp2-client --load myscript.lisp

./build/golisp2-client --repl
golisp2> (defun square (x) (* x x))
golisp2> (square 5)
25
golisp2> :quit
```

**REPL-Modus:**
- Multiline: automatische Fortsetzung bei offenen Klammern
- Kommandos: `:quit`, `:complete prefix`, `:load datei`
- Error Recovery: Fehler brechen den REPL nicht ab (`:abort` → stderr, weiter)
- **Pro-Connection-Env:** jede Verbindung bekommt ein frisches Environment
  (`lib.BaseEnv()` + `LoadStdlib` in `handleConn`) — kein Zustand zwischen
  Verbindungen

## Protokoll

- **Framing:** 6-stellige Hex-Länge + S-Expression, kein Newline (`framing.go`)
- **Request:** `(:emacs-rex (op args…) "USER" t ID)`
- **Response:** Liste von Frames — pro Zwischenergebnis ein
  `(:write-string "text" :repl-result)`, abschließend `(:return (:ok val) ID)`
  bzw. `(:return (:abort "msg") ID)` bei Fehler.
  Der Client liest Frames, bis das `:return` mit passender ID eintrifft.

### Implementierte SLIME-Ops

| Op | Verhalten |
|----|-----------|
| `swank:connection-info` | `(:type "GoLisp" :version "0.2" :style :spawn)` |
| `swank:swank-require` | Stub `(:ok ())` — keine Contribs |
| `swank:init-presentations` | Stub `(:ok ())` |
| `swank-repl:create-repl` | `(:ok ("USER" "USER"))` + `:new-package` |
| `swank-repl:listener-eval` | alle Formen via `ReadAll` → pro Ergebnis `:write-string "<w>\n" :repl-result` + `(:ok nil)` |
| `swank:simple-completions` | `(:ok (strings…))` |
| `swank:completions` | `(:ok ((name) (name) …))` — c-p-c Contrib; das Op, das SLIME für TAB sendet |
| `swank:load-file` | `(load filename)` → `(:ok "<r>")` — C-c C-l |
| `swank:operator-arglist` | `(:ok "(name args)")` für Lambda/Macro/Built-in, sonst `(:ok ())` — C-c C-d C-a |
| `swank:autodoc` | `(:ok (arglist-string cache-p))`, sonst `(:not-available nil)` |
| `swank:swank-macroexpand-1` / `swank-expand-1` | eine Top-Level-Expansion → `(:ok "<exp>")` — C-c C-m |
| `swank:swank-macroexpand` / `swank-expand` | wiederholt bis stabil (Top-Level) |
| `swank:swank-macroexpand-all` / `swank-expand-all` | rekursiv in alle Subformen |
| `swank:describe-symbol` | `(:ok (:title name :content text))` — statische Registry |
| `swank:compile-string-for-emacs` | `(:ok t)` nach stillem Evaluieren |
| `swank:compile-file-for-emacs` | `(:ok (:filename file :result …))` via `(load file)` — C-c C-k |
| `swank:find-definitions-for-emacs` | Map-Lookup → `:location (:file … :line N)`; REPL-Snippet-Fallback; Built-in → `:error` — M-. |
| *sonstige* | graceful `(:ok ())` statt `:abort` — SLIME-Contribs degradieren sauber |

### Fallstricke (teuer gelernt)

- **`listener-eval`-Return:** `:write-string` mit `:repl-result`-Tag senden,
  dann `(:ok nil)`. **Nicht** `(:ok "<string>")` — SLIME destrukturiert `:ok`
  als Liste.
- **Mehrere Formen:** `listener-eval` liest alle Formen via `swank--read-all`
  (`lib.ReadAll`), evaluiert jede, sendet pro Ergebnis ein `:write-string`.
- **`completions`-Format:** Liste von 1-Element-Listen `((name) …)` — der Client
  destrukturiert `(symbol-name classification symbol)`, fehlende Felder = nil.
- **`autodoc`:** für Built-ins `(:not-available nil)` liefern — sonst
  `insert nil`-Error in `slime-autodoc--format`. Arglist via `swank--arglist`
  (Lambda: `Type:LIST` + `Env != nil`, `Car` = Parameter).
- **eval global:** `listener-eval` nutzt `(eval (read string))`. Damit `defun`
  global persistiert, wertet `eval` im `Env.Root()` aus — siehe Invarianten in
  CLAUDE.md.
- **C-c C-m Cursor:** `slime-bounds-of-sexp-at-point` greift bei Cursor auf einem
  Symbol nur das Symbol. Für die ganze Form den Cursor auf `(` setzen.
- **M-.:** Lookup in der `lib.defloc`-Map (`defun`/`defmacro`/`define`
  registrieren `SrcFile`/`SrcLine`). REPL-definierte Funktionen (kein SrcFile)
  → Snippet-Buffer. Built-ins → `:error`.

### Offen (nicht im aktuellen Scope)

Inspector, Debugger/Restarts, `disassemble-symbol`.
