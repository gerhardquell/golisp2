# GoLisp – CLAUDE.md

## Projekt-Übersicht
GoLisp ist ein Lisp-Interpreter in Go mit nativer KI-Anbindung (sigoREST).
Ziel: Ein selbsterweiterndes System das Goroutinen, Channels und KI-Calls
als eingebaute Lisp-Primitiven beherrscht.

**Autor:** Gerhard Quell – gquell@skequell.de
**CoAutor:** claude sonnet 4.6
**Modul:** `golisp`

---

## Dateistruktur

```
golisp/
  main.go              REPL + Testmodus (-t) + Datei-Loader
  lib/
    types.go           Cell-Datenstruktur (LispType, Cons, MakeAtom...)
    reader.go          Parser: String → Cell-Baum (NewReader, Read)
    env.go             Umgebung: Get, Set, Update (verkettete Scopes)
    eval.go            Herzstück: Eval, Spezialformen, defmacro, parfunc
    primitives.go      Eingebaute Funktionen + BaseEnv()
    goroutine.go       parfunc, chan-make/send/recv, lock-make
    fileio.go          file-write, file-append, file-read, file-exists?, file-delete
    sigorest.go        sigo, sigo-models, sigo-host (HTTP zu sigoREST)
    readline.go        REPL-Input mit History + Multiline
```

---

## Coding-Konventionen

- **Sprache:** Go für den Kern, Lisp für Erweiterungen
- **Einrückung:** 2 Spaces, keine Tabs
- **Dateinamen:** camelCase (außer main.go)
- **Kommentare:** sparsam, sprechende Namen bevorzugt
- **Dateigröße:** max 300 Zeilen, ab 500 aufteilen
- **Datei-Header:** immer mit Autor, CoAutor, Copyright, Erstellt (YYYYMMDD)
- **Fehler:** `fmt.Errorf("funktionsname: beschreibung")` 

### Spezialformen vs. Primitiven
- Braucht die Funktion Zugriff auf `env`? → Spezialform in `eval.go`
- Reine Berechnung ohne env? → Primitiv in `primitives.go`
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
   parfunc, lock, eval)
2. Makro-Expansion (MACRO-Typ → expand → Eval des Ergebnisses)
3. Normale Anwendung: Funktion auswerten → Argumente auswerten → apply
```

### Lambda-Struktur
```go
// Lambda/Closure wird als Cell{Type:LIST} gespeichert:
Cell{Type: LIST, Car: params, Cdr: body, Env: closureEnv}
// Makro: identisch aber Type: MACRO
```

---

## Implementierte Features

### Spezialformen
`quote` `if` `define` `defun` `lambda` `let` `begin` `set!`
`defmacro` `mapcar` `load` `and` `or` `not` `parfunc` `lock` `eval`

### Eingebaute Funktionen
**Arithmetik:** `+` `-` `*` `/`
**Vergleiche:** `=` `<` `>`
**Listen:** `car` `cdr` `cons` `atom` `null` `list`
**I/O:** `print` `println` `read`
**Datei:** `file-write` `file-append` `file-read` `file-exists?` `file-delete`
**Nebenläufigkeit:** `chan-make` `chan-send` `chan-recv` `lock-make`
**KI:** `sigo` `sigo-models` `sigo-host`

---

## Fehlende Features (Priorität für ClaudeCode)

### Kritisch
1. **Quasiquote** – `` ` `` `,` `,@` im Reader + Eval
   → Makros werden lesbar statt `(list (quote ...) ...)`
2. **Tail-Call-Optimierung (TCO)** – Trampolin-Muster in eval.go
   → Tiefe Rekursion ohne Stack-Overflow
3. **`apply`** – `(apply + (list 1 2 3))`
4. **`cond`** – `(cond ((= x 1) ...) ((= x 2) ...) (t ...))`

### Wichtig
5. **String-Funktionen** – `string-length` `string-append` `substring`
   `string->number` `number->string` `string-upcase` `string-downcase`
6. **Error-Handling** – `(error "msg")` `catch`/`throw`
7. **`gensym`** – eindeutige Symbole für Makros

### Nice-to-have
8. **`do`/`while`** – Schleifen
9. **`equal?`** – struktureller Listen-Vergleich
10. **History-Persistenz** in readline (`.golisp_history`)
11. **Tab-Completion** in readline

---

## sigoREST Anbindung

GoLisp spricht mit dem sigoREST-Server:
```
Host: http://127.0.0.1:9080 (Default)
Endpoint: POST /v1/chat/completions
```

Verfügbare Modell-Shortcodes: `claude-h` `gemini-p` `gpt41`
und alle lokalen Ollama-Modelle (z.B. `ollama-gemma3-4b`)

### Das selbsterweiternde Muster
```lisp
; KI schreibt Code → GoLisp führt ihn aus
(eval (read (sigo "schreibe (defun fib (n) ...)" "claude-h")))
(fib 10)

; Ensemble: 3 KIs parallel
(parfunc antworten
  (sigo "problem" "claude-h")
  (sigo "problem" "gemini-p")
  (sigo "problem" "gpt41"))
```

**Wichtig für sigo-Prompts:** Den Prompt so formulieren dass die KI
*nur* den Lisp-Code zurückgibt ohne Erklärungen – z.B.:
`"Schreibe nur den Lisp-Code, keine Erklärungen: defun fib ..."`

---

## Build & Test

```bash
go build .          # kompilieren
go run . -t         # Testmodus
go run .            # REPL starten
go run . skript.lisp # Datei ausführen
```

### readline (auf Gerhard's Debian-Rechner)
```bash
go get github.com/chzyer/readline
go mod tidy
```
Dann `lib/readline.go` durch chzyer-Version ersetzen für
vollständige Multiline-REPL mit Pfeiltasten-History.

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
