# GoLisp SWANK-Server Design

**Datum:** 2026-06-18  
**Status:** Design genehmigt, bereit für Implementierungsplan  
**Thema:** SWANK-Server für Emacs/SLIME-Integration

## Ziel

GoLisp bekommt einen echten SWANK-Server, mit dem sich Emacs über `M-x slime` verbinden kann. Der Server wird als Hybrid aus Go (TCP, Framing, Dispatch-Primitive) und GoLisp (Protokoll-Handler) realisiert.

## Ausgangslage

- `lib/swank/server.go` und `lib/swank/protocol.go` existieren bereits, implementieren aber ein eigenes S-Expression-RPC-Protokoll, nicht das echte SWANK-Protokoll.
- SWANK verwendet length-prefixed UTF-8 S-Expressions (`Hex-Length:S-Expr`) und Nachrichten wie `connection-info`, `create-repl`, `listener-eval`.
- Der neue Server soll über `golisp2 --swank [host:port]` starten.

## Entscheidungen

| Entscheidung | Gewählt | Begründung |
|--------------|---------|------------|
| Server-Standort | `golisp2 --swank` Flag im Hauptbinary | Einfacher Start, kein separates Binary nötig |
| Architektur | Hybrid: Go TCP/Framing + GoLisp Handler | Balance aus Stabilität und „nativ in GoLisp“ |
| Env pro Verbindung | Frisches `*Env` pro TCP-Verbindung | Isolation zwischen Emacs-Sitzungen |
| Async Output | Primitive `swank-send-event` in Go | Erlaubt `write-string` während `listener-eval` |
| Fehler MVP | `(:return (:abort "msg") id)` | Debugger kommt in Phase 3 |

## Architektur

```
┌─────────┐     SWANK-Frames          ┌──────────────┐
│  Emacs  │  ───────────────────────► │ golisp2 --swank│
│ (SLIME) │  ◄─────────────────────── │              │
└─────────┘                           │  ┌──────────┐ │
                                      │  │ TCP/     │ │
                                      │  │ Framing  │ │  (Go)
                                      │  └────┬─────┘ │
                                      │       │       │
                                      │  ┌────┴─────┐ │
                                      │  │ (swank-  │ │
                                      │  │ dispatch │ │  (GoLisp)
                                      │  │ msg)     │ │
                                      │  └────┬─────┘ │
                                      │       │       │
                                      │  ┌────┴─────┐ │
                                      │  │ Handler  │ │
                                      │  │ lib/     │ │  (GoLisp)
                                      │  │ swank.lisp│ │
                                      │  └──────────┘ │
                                      └───────────────┘
```

## Komponenten

### Go-Seite (`lib/swank/`)

- `server.go`
  - CLI-Flag `--swank [host:port]` (Default: `127.0.0.1:4005`).
  - TCP-Listener, Accept-Loop.
  - Pro Verbindung: frisches `*lib.Env`, Laden der Standardbibliothek + `lib/swank.lisp`.
  - Verbindungs-Lebenszyklus: Cleanup bei Disconnect.

- `framing.go`
  - `readFrame(r io.Reader) (*Cell, error)` — liest Hex-Länge + S-Expr.
  - `writeFrame(w io.Writer, cell *Cell) error` — schreibt Hex-Länge + S-Expr.

- `dispatch.go`
  - `handleMessage(conn, env, cell)` — ruft `(swank-dispatch <cell>)` auf.
  - Sendet jedes zurückgegebene Event als Frame.
  - Registriert Primitive `swank-send-event` in `BaseEnv()` oder nur im SWANK-Env.

### GoLisp-Seite (`lib/swank.lisp`)

- `(swank-dispatch msg)`
  - Top-Level-Dispatch auf `:method` bzw. `:emacs-rex`.

- Handler-Funktionen
  - `connection-info`
  - `create-repl`
  - `listener-eval`
  - `simple-completions` (Phase 2)
  - `operator-arglist` (Phase 2)
  - `describe-symbol` (Phase 2)
  - Debugger-Handler (Phase 3)

- Hilfsfunktionen
  - Formatierung von `(:return ...)` Events.
  - `*swank-output-stream*`-Hook: `print`/`println` im SWANK-Env auf `(swank-send-event '(:write-string ...))` umleiten.

## Datenfluss

1. Emacs sendet Frame: `000027(:emacs-rex (swank:connection-info) nil t)`.
2. Go liest Länge `27`, liest S-Expr, parst zu GoLisp-Cell.
3. Go evaluiert `(swank-dispatch <cell>)` im Verbindungs-Env.
4. Handler gibt Liste von Events zurück, z.B.:
   ```lisp
   ((:return (:ok (:pid 123 :implementation (:type "GoLisp") :version "0.1")) 1))
   ```
5. Go schickt jedes Event als eigenen Frame zurück.

### Async Output

Während `listener-eval` läuft, kann `print` Output erzeugen. Der Output-Hook ruft `(swank-send-event '(:write-string "text" :repl-result))` auf. Das Primitive sendet den Event sofort an Emacs, ohne auf das Ende von `swank-dispatch` zu warten.

## Fehlerbehandlung

- **Parser-Fehler in Go:** Verbindung trennen oder `(:reader-error ...)` senden.
- **Fehler in `swank-dispatch`:** Handler fängt Fehler mit `catch` ab und gibt `(:return (:abort "Fehlertext") id)` zurück.
- **Debugger/Restarts:** In Phase 3 wird ein `(:debug ...)` Event mit Condition, Restarts und Backtrace gesendet.

## Testing

- Go-Unit-Tests für `framing.go` mit bekannten SWANK-Frames.
- Go-Test für `dispatch.go` Wrapper (Mock-Env, Mock-Message).
- GoLisp-Tests für `lib/swank.lisp` Handler mit statischen Nachrichten.
- Integrationstest: `golisp2 --swank` starten, per TCP `connection-info` senden, Antwort auf Struktur prüfen.

## Phasen

### Phase 1 — MVP (Emacs REPL verbindet)

- SWANK-Framing in Go.
- `--swank` Flag.
- `swank-dispatch` in Lisp.
- Handler: `connection-info`, `create-repl`, `listener-eval`.
- Output-Redirect über `swank-send-event`.
- Erfolgskriterium: `M-x slime` verbindet sich und `(+ 1 2)` liefert `3`.

### Phase 2 — Komfort-IDE

- `simple-completions`
- `operator-arglist`
- `describe-symbol`
- `load-file`
- `interactive-eval`, `pprint-eval`
- Erfolgskriterium: Autocomplete, Dokumentation und Datei-Laden funktionieren in Emacs.

### Phase 3 — Voll-SLIME

- Debugger (`:debug`, `:invoke-restart`, Backtrace)
- Inspector (`inspect-*`)
- Compiler-Notes
- Presentations
- Erfolgskriterium: Fehler öffnen Debugger, Daten sind inspizierbar.

## Offene Punkte

- Soll `swank-send-event` global in `BaseEnv()` registriert werden oder nur lokal im SWANK-Env?
- Soll `lib/swank.lisp` in die Standardbibliothek eingebunden oder nur beim `--swank`-Start geladen werden?
- Sollen mehrere gleichzeitige Emacs-Verbindungen ein geteiltes Env haben? (Aktuelles Design: nein, pro Conn isoliert.)

## Abgrenzung

- `slime-tramp` für Emacs (TODO.md Punkt 2) ist ein separater Design-Schritt nach diesem Server.
- Das alte `golisp2d`-Binary und das eigene RPC-Protokoll in `lib/swank/` werden durch diesen Design-Schritt nicht entfernt, sondern `lib/swank/` wird auf echtes SWANK umgestellt.
