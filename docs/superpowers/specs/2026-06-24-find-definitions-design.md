# GoLisp find-definitions-for-emacs Design (M-.)

**Datum:** 2026-06-24  
**Status:** Design genehmigt, bereit für Implementierungsplan  
**Thema:** `swank:find-definitions-for-emacs` — M-. in SLIME springt zur Definition

## Ziel

`M-.` in Emacs/SLIME springt zur Definition einer `defun`/`defmacro`/`define`
in der echten Quelldatei, Zeile genau. REPL-definierte Funktionen (ohne
Quelldatei) erhalten einen rekonstruierten Snippet im Temp-Buffer.

## Ausgangslage

- SWANK-Server (`lib/swank/`) funktionsfähig, SLIME v2.32 connectet sauber.
- `swank:find-definitions-for-emacs` ist **nicht** implementiert — unbekannte
  Ops fallen in `handle-emacs-rex` auf `swank:ok-nil` → M-. liefert "no
  location".
- Kein Source-Tracking: `Cell`-Struct hat keine File/Line-Felder, `defun`
  merkt keinen Definitionsort, Reader zählt keine Zeilen, `load` kennt keinen
  Dateikontext.
- `define`/`setq` sind aliasiert auf `evalDefine`.

## Entscheidungen

| Entscheidung | Gewählt | Begründung |
|--------------|---------|------------|
| Source-Location-Speicherung | `SrcFile`+`SrcLine` auf `Cell` + Go-seitige Map | Reader stempelt Line direkt; Map für O(1)-Lookup, parfunc-safe |
| Lookup-Map in Go, nicht Lisp-Alist | `sync.RWMutex` `map[string]DefLoc` | GoLisp-Alist wäre bei `parfunc` Race-gefährdet; Go-Map mit Lock sicher |
| Dispatch-Änderung | `evalDefun(expr,env)` statt `evalDefun(expr.Cdr,env)` | Form-Cell trägt `SrcLine`/`SrcFile`; nur 3 Zeilen in `eval_core.go` |
| REPL-Funktionen | Snippet-Fallback via `swank--value-string` auf params/body | Keine Datei vorhanden; Lambda-Cell enthält genug für Rekonstruktion |
| Built-in-Primitiv | `:error` (keine Quellposition) | Built-ins sind Go-Code, keine Lisp-Source zugeordnet |
| Zeilen-Positionierung | `(:line N :align t)` | Sonst öffnet Emacs Datei ohne Cursor-Positionierung |

## Architektur & Komponenten

| Datei | Änderung |
|-------|----------|
| `lib/types.go` | `Cell`-Struct: `SrcFile string` + `SrcLine int` (zero-value = unbekannt) |
| `lib/reader.go` | Reader: `line int`, inkrement bei `\n` in `next()`; `readList` stempelt `SrcLine` auf jede Listen-Cell beim Bau |
| `lib/eval_load.go` | `evalLoad`: nach `readExpr` `expr.SrcFile = resolvedPath` stempeln (Top-Level-Formen) |
| `lib/eval_core.go` | 3 Dispatch-Zeilen: `evalDefun`/`evalDefmacro`/`evalDefine` erhalten `expr` statt `expr.Cdr` |
| `lib/eval_specialforms.go` | Sigs auf `form *Cell`; `args := form.Cdr`; Aufruf `registerDefinition(name, form.SrcFile, form.SrcLine)` |
| `lib/swank/defs.go` (neu) | Go-Map `symbol→DefLoc{File,Line}` + `sync.RWMutex`; `RegisterDefinition`/`LookupDefinition`; Primitive `swank--find-definition` |
| `lib/swank/swank.lisp` | `swank:find-definitions-for-emacs` Handler + Dispatch-Eintrag; `swank--reconstruct-definition` für REPL-Fallback |

### Definition-Registry (Go-Seite)

```go
type DefLoc struct { File string; Line int }
var defMutex sync.RWMutex
var definitions = map[string]DefLoc{}

func RegisterDefinition(name, file string, line int) { /* Lock, store */ }
func LookupDefinition(name string) (DefLoc, bool)    { /* RLock, lookup */ }
```

Registriert via Primitive `swank--find-definition` exponiert →
`("file" . line)` oder `()`.

## Datenfluss

### M-. (Cursor auf Symbol `fib`)

```
Emacs SLIME ──(:emacs-rex (swank:find-definitions-for-emacs "fib") ...)──► GoLisp
  handle-emacs-rex → swank:find-definitions-for-emacs "fib" id
    1. swank--find-definition "fib"   ; Go-Primitive, Map-Lookup
    2a. Treffer (file . line):
        → (:location (:file "/abs/file.lisp") (:line N :align t) nil)
    2b. Kein Treffer, Symbol = Lambda/Macro (REPL-definiert, SrcFile leer):
        → swank--reconstruct-definition "fib"
        → (:location (:buffer "*slime-source*" (:source "<snippet>")) (:position 1) nil)
    2c. Built-in-Primitiv (FUNC, Env==nil) oder unbound:
        → (:error "keine Quellposition für 'fib'")
    3. Return: (list (list :return (list :ok (<location>)) id))
```

### Registrierung (beim Laden/Definieren)

```
load file.lisp → readExpr (Reader stempelt SrcLine auf Listen-Cell)
  → expr.SrcFile = "/abs/file.lisp"          ; evalLoad stempelt Top-Level
  → Eval (defun fib (n) ...)
    → evalDefun(form, env)
       → name = "fib"
       → registerDefinition("fib", form.SrcFile, form.SrcLine)
       → rest wie bisher (Lambda bauen, in env setzen)
```

### Snippet-Rekonstruktion (Fallback C)

Lambda-Cell: `Cell{Type:LIST, Car:params, Cdr:body, Env:closureEnv}`.

```lisp
(defun swank--reconstruct-definition (name cell)
  (string-append "(defun " name " "
    (swank--value-string (car cell)) " "   ; params
    (swank--value-string (cdr cell)) ")"))  ; body (begin-wrapped)
```

Für Macro entsprechend `(defmacro ...)`. `swank--value-string` existiert bereits.

## Fehlerbehandlung & Randfälle

| Fall | Verhalten |
|------|-----------|
| Symbol unbound | `(:error "Symbol 'x' nicht definiert")` |
| Built-in-Primitiv (`car`, `+`, …) | `(:error "eingebaute Funktion 'car' hat keine Quellposition")` — FUNC-Cell, `Env==nil`, nicht in Map |
| REPL-definiert (Lambda, SrcFile leer) | Fallback C → Snippet-Buffer |
| `define`/`setq` Variable | Map-Eintrag, M-. springt zur Definitionszeile |
| Datei gelöscht nach Laden | Location unverändert; SLIME zeigt eigenen Öffnen-Fehler |
| `parfunc` mit internen `defun`s | Map `sync.RWMutex` → sicher; last-write-wins |
| Multiline-Form | `SrcLine` = Zeile der öffnenden Klammer |
| `let`/`flet`/`labels`-lokale Funktionen | Nicht registriert (gehen nicht durch Spezialform-Top-Level-Dispatch) |

**Concurrency:** `RegisterDefinition` → `Lock`/`Unlock`. `LookupDefinition` →
`RLock`/`RUnlock`. `parfunc`-safe.

## Testing

**Go-Tests:**

- `lib/swank/defs_test.go` (neu):
  - `TestRegisterAndLookupDefinition` — registrieren, lookup, Treffer.
  - `TestLookupUnknown` — nicht-registriert → leer.
  - `TestConcurrentRegister` — parallele `RegisterDefinition` (goroutines), `go test -race`.
- `lib/reader_test.go` (Ergänzung):
  - `TestReaderLineTracking` — `SrcLine` korrekt auf Listen-Cell (3-Zeilen-Form → Zeile 1).

**Lisp-Tests (`lib/swank/lisp_test.go` Ergänzung):**

- `find-definitions-for-emacs` mit geladener Testdatei → File-Location.
- REPL-definierte Funktion → Buffer-Location mit Snippet.
- Built-in `car` → `:error`.

**Integration (`lib/swank/integration_test.go`):**

- Volle Runde: Testdatei mit `(defun x ...)` laden, `find-definitions-for-emacs "x"` dispatchen, Location verifizieren.

**Manuelle Verifikation:**

```bash
./build && ./golisp2 --swank 127.0.0.1:4242
# Emacs: slime-connect, Datei mit defun laden, M-. auf Funktionsnamen → springt zur Zeile
```

## Nicht im Scope

- `find-definitions` für Built-in-Primitive (Go-Source zugeordnet) — `:error`.
- Mehrfach-Definitionen pro Symbol (CLOS-Methoden etc.) — eine Location pro Symbol.
- Filename-Translation / slime-tramp (remote golisp2d) — lokaler Use-Case, kein TRAMP.
- Inspector, Debugger/Restarts — separate Specs.

## Abhängigkeiten / Voraussetzungen

- `Cell`-Struct-Änderung berührt alle Cell-Erzeugungen — zero-value-Felder
  sicher (bestehender Code unbeeinflusst, Felder optional).
- `evalDefine`-Sig-Änderung betrifft `define`+`setq`-Alias — beide registrieren
  Location (für Variablen-M-.).
