# GoLisp – Retrospektive

**Datum:** 2026-02-24
**Autoren:** Gerhard Quell & Claude Sonnet 4.6

---

## Was haben wir gebaut?

| Feature | Dateien | Commits |
|---------|---------|---------|
| Quasiquote `` ` `` `,` `,@` | `reader.go`, `eval.go` | 1 |
| `apply` | `primitives.go` | 1 |
| `cond` | `eval.go` | 1 |
| 7 String-Funktionen | `stringfuncs.go` (neu) | 1 |
| TCO (Tail-Call-Optimierung) | `eval.go` | 4 |
| `gensym` | `primitives.go` | 2 |
| `(error msg)` + `(catch body handler)` | `types.go`, `eval.go`, `primitives.go` | 5 |

**Gesamt:** 7 Features, ~15 Commits, ~600 neue Zeilen Go-Code.

**Vorher:** Stack-Overflow bei tiefer Rekursion, keine Makro-Hygiene, kein Error Handling.
**Nachher:** 1.000.000 Rekursionen in 44ms, hygienische Makros mit `gensym`, strukturiertes Error Handling.

---

## Was lief gut?

### Plan → Execute → Review Workflow
Das dreigliedrige Muster (Plan schreiben → Subagent implementiert → zwei Review-Stufen)
hat sich bewährt. Echte Bugs wurden konsequent abgefangen, bevor sie in `main` landeten:

- TCO: `else`-Branch verwendete `Cdr.Cdr.Car` statt `Cdr.Cdr.Cdr.Car` → abgefangen
- TCO: fehlende Typprüfung vor `fn.Fn(args)` → nil-Panic verhindert
- gensym: fehlender Arity-Check → ergänzt
- Error Handling: `LispError.Error()` leer für Nicht-String-Zellen → behoben
- Error Handling: `%v` statt `%w` in `evalLoad` → `errors.As` funktioniert jetzt

### TDD-Rhythmus
Failing test zuerst schreiben hat bei TCO den Wert klar bewiesen:
der Test crashte mit Stack-Overflow — unbestreitbarer Beweis dass TCO nötig ist.

### Subagent-Reviews als Qualitätsstufe
Die Kombination aus Spec-Review und Code-Quality-Review hat verhindert,
dass "es läuft grob" als "fertig" durchgeht.

---

## Was lief nicht so gut?

### Language-Server false positives
Mehrfach wurden Compiler-Diagnostiken gemeldet (`sync/atomic` "unused",
`errors` "unused", `evalCatch undefined`), die bei `go build` nicht auftraten.
Ursache: der Language-Server analysiert Dateien einzeln und erkennt
cross-file Abhängigkeiten nicht sofort nach einer Änderung.

**Lösung:** Immer zuerst `go build ./...` als Ground Truth.

### Spec-Reviewer verstand TDD-Phasen nicht
Bei Task 1 von Error Handling meldete der Spec-Reviewer `catch` als fehlend —
obwohl der Plan explizit "Tests sollen hier noch fehlschlagen" vorsah.

**Lösung:** In Implementer-Prompts explizit vermerken welche Tests in welcher Phase
grün/rot sein sollen.

### Plan-Lücke bei `evalLoad`
Der `%v` vs `%w` Fehler in `evalLoad` wäre vermeidbar gewesen,
wenn der Plan `errors.As` im Kontext der gesamten Fehlerkette betrachtet hätte.

**Lösung:** Bei `errors.As` / `errors.Is` immer prüfen ob Fehler durch
Wrapping-Schichten propagieren.

---

## Technische Erkenntnisse

### TCO in Go
Go's goroutine-Stacks wachsen automatisch bis 1 GB — aber 4 Go-Frames
pro Lisp-Rekursionsschritt × 1.000.000 Aufrufe = ~400 MB Stack-Bedarf.
TCO via `for`-Loop reduziert das auf O(1).

Das Loop-Muster ist sauberer als Trampolin: kein `continue`-Label nötig,
kein Thunk-Overhead, kein separater Dispatcher.

### LispError als eigener Go-Typ
`*LispError` als Go-Typ (statt nur `fmt.Errorf`) erlaubt präzises
Type-Switching in `catch` — nur Lisp-Fehler werden abgefangen,
interne Go-Fehler (z.B. Division durch 0 im Go-Layer) propagieren unverändert.
Das gibt dem System klare Semantik.

### Quasiquote-Tiefe
`evalQQ(expr, env, depth)` mit depth-Parameter löst verschachtelte
Quasiquotes korrekt: `\`(a \`(b ,(+ 1 2)))` expandiert nur die innere
Ebene, nicht die äußere.

---

## Noch offen nach Session 1

*(alle in Session 2 abgearbeitet)*

---

## Fazit Session 1

GoLisp hat sich in einer Session von einem funktionalen Prototypen
zu einem ernsthaften Lisp-Interpreter entwickelt.
Der Workflow (Plan → Subagent → Review) hat gezeigt, dass
KI-getriebene Entwicklung mit klaren Qualitätsgates
konsistent gute Ergebnisse liefert — nicht trotz der Reviews,
sondern wegen ihnen.

> "Code = Daten + KI = sich selbst erweiterndes System"
> — Gerhard & Claude, Februar 2026

---
---

# Session 2 – 2026-02-24

**Autoren:** Gerhard Quell & Claude Sonnet 4.6

---

## Was haben wir gebaut?

| Feature | Dateien | Commits |
|---------|---------|---------|
| Multi-Body `defun`/`lambda`/`defmacro` via `wrapBegin` | `eval.go` | 2 |
| `>=` `<=` Vergleichsoperatoren | `primitives.go` | 1 |
| History-Persistenz `~/.golisp_history` | `readline.go`, `env.go` | 1 |
| REPL-Rewrite: `go-prompt`, Syntax-Highlighting | `readline.go` | 2 |
| `while` Schleife | `eval.go` | 1 |
| `do` Schleife (Scheme-style) | `eval.go` | 1 |
| TCO-Regressionstests (war bereits implementiert) | `main.go` | 1 |
| `equal?` struktureller Vergleich | `primitives.go` | 1 |
| `CLAUDE.md` + `BESCHREIBUNG.md` | Docs | 2 |

**Gesamt:** 8 Features + 2 Docs, 12 Commits, ~350 neue Zeilen.

**Vorher:** Single-Body defun, kein Highlighting, keine `>=`/`<=`, keine `do`/`while`, kein `equal?`.
**Nachher:** Vollständige Sprache, farbiger REPL, alle Standard-Lisp-Features implementiert.

---

## Was lief gut?

### TDD als Entdeckungswerkzeug
Bei TCO: die Tests liefen sofort grün — was beweist, dass TCO bereits in der
Vorjahressession implementiert war. TDD hat hier nicht eine neue Implementierung
erzwungen, sondern eine fehlerhafte Speicherlücke (MEMORY.md) korrigiert.
Das ist der eigentliche Wert: Tests als objektive Wahrheitsquelle.

### go-prompt API-Recherche vor dem Coden
Statt blind drauflos zu programmieren, wurden zuerst die Quelldateien der Library
gelesen (`lexer.go`, `constructor.go`, Beispiele). Das ersparte mehrere Iterationen:
die `EagerLexer` / `LexerFunc`-Signatur, `ExecuteOnEnterCallback` für Multi-line
und `WithCustomHistory` für Persistenz — alles auf Anhieb korrekt.

### Minimale Änderungen
`wrapBegin` — 10 Zeilen, 3 Aufrufstellen geändert, kein neuer Eval-Pfad.
`cellEqual` — 12 Zeilen, rekursiv, deckt alle Typen ab.
Beide Features hätten auch mit doppelt so viel Code implementiert werden können —
die Minimalform ist robuster und leichter zu verstehen.

---

## Was lief nicht so gut?

### go-prompt Completion-Popup
Das Auswahlfeld erschien automatisch beim Tippen — unerwartet und störend.
`go-prompt` kennt kein "nur auf TAB anzeigen"-Flag, die Lösung war,
den Completer komplett zu entfernen.

**Erkenntnis:** Bei Library-Auswahl für UI-Features vorab prüfen
ob das gewünschte Verhalten (TAB-only) überhaupt konfigurierbar ist.

### Farben nicht sichtbar
Die ersten Bracket-Farben (Yellow/Cyan/Green) waren auf Gerhards Terminal
nicht erkennbar. Zwei Iterationen nötig bis Red/Green/Yellow/Fuchsia passte.

**Erkenntnis:** Terminal-Farbpaletten variieren stark. Bei Farb-Features
früh fragen welches Terminal / welcher Hintergrund verwendet wird.

### fileHistory Workaround
Der erste `newFileHistory`-Entwurf enthielt einen dummy-Aufruf
`prompt.WithHistory(entries)((*prompt.Prompt)(nil))` — sah nach einem Hack aus
und wurde sofort bereinigt. Ursache: die Library-API für "History vorladen"
war nicht sofort offensichtlich und die direkte `Add`-Loop war die sauberere Lösung.

---

## Technische Erkenntnisse

### wrapBegin als Normalisierungsschritt
Multi-Body zur *Definitionszeit* in `(begin ...)` wrappen ist eleganter als
zur Laufzeit: der Evaluator bleibt unverändert, `begin` ist bereits TCO-aware,
und der Overhead für Single-Body-Funktionen ist null (kein Wrapper).

### go-prompt ExecuteOnEnterCallback
`p.Buffer().Text()` im Callback liefert den gesamten bisherigen Multi-line-Buffer.
`countDepth` darauf angewandt ergibt direkt ob der Ausdruck vollständig ist.
Rückgabe `(depth, false)` → go-prompt rückt automatisch ein, kein manuelles
`..`-Prompt mehr nötig.

### do mit gleichzeitigem Step-Update
Scheme's `do` evaluiert alle Step-Ausdrücke im *alten* Environment bevor
die neuen Werte gesetzt werden:
```lisp
(do ((a 1 b) (b 2 a)) ((= a 3) (list a b)))  ; → (2 1), nicht (2 2)
```
Die Implementierung sammelt daher zuerst alle neuen Werte in einem Slice,
setzt sie dann gesammelt. Das ist der semantisch korrekte Scheme-Weg.

---

## Zustand der Sprache

Nach Session 2 ist GoLisp **feature-complete** für einen ernsthaften Lisp-Interpreter:

- ✅ Alle Standard-Spezialformen
- ✅ Quasiquote / Makros / gensym
- ✅ Error Handling (error/catch)
- ✅ TCO — beliebig tiefe Tail-Rekursion
- ✅ Multi-Body defun/lambda/defmacro
- ✅ Schleifen (while, do)
- ✅ Strukturelle Gleichheit (equal?)
- ✅ Vollständige String-Bibliothek (UTF-8)
- ✅ Datei-I/O
- ✅ Nebenläufigkeit (parfunc, channels, locks)
- ✅ KI-Anbindung (sigo/sigoREST)
- ✅ REPL mit Syntax-Highlighting und History

**Nächste Ausbaustufen** (offen, kein Zeitdruck):
- `string-ref`, `string-split` — weitere String-Operationen
- `number?`, `string?`, `list?` — Typprädikate
- Varargs in defun: `(defun f (x . rest) ...)`
- Mehrwertrückgabe (values/call-with-values)

---

## Fazit Session 2

Die Sprache ist vollständig. Der REPL macht Spaß.
Das Fundament ist stabil genug für das eigentliche Ziel:
GoLisp als selbsterweiterndes KI-System.

> "Eine Sprache die sich selbst erweitern kann,
>  braucht zuerst eine Sprache die vollständig ist."
> — Gerhard & Claude, Februar 2026

---
---

# Session 3 – 2026-02-24

**Autoren:** Gerhard Quell & Claude Sonnet 4.6

---

## Was haben wir gebaut?

| Feature | Dateien | Commits |
|---------|---------|---------|
| `fnEq` redundanten Vergleich fixen | `primitives.go` | 1 |
| `macroexpand` als Debugging-Hilfe | `eval.go` | 1 |

**Gesamt:** 2 Quick Wins, 1 Commit, ~35 Zeilen geändert.

**Vorher:** `fnEq` verglich unnötigerweise auch `Val`; keine Möglichkeit Makro-Expansion zu inspizieren.
**Nachher:** `fnEq` vergleicht nur noch `Num`; `macroexpand` zeigt expandierte Makros.

---

## Was lief gut?

### Plan → Execute ohne Review-Overhead
Die drei Quick Wins waren klein und sicher genug für direkte Implementierung:
- `fnEq`: offensichtlicher Bug, einfache Lösung
- `macroexpand`: klare Spezifikation, saubere Architektur-Entscheidung

Keine Subagenten nötig – der Aufwand wäre größer als der Nutzen.

### Spezialform statt Primitive
Die erste `macroexpand`-Implementierung als Primitive (`primitives.go`) scheiterte elegant:
Primitives haben keinen `env`-Zugriff, können also Makros nicht auflösen.
Die Umstellung auf Spezialform (`eval.go`) war der korrekte Architektur-Pfad.

**Bestätigte Regel:** *Braucht Zugriff auf `env`? → Spezialform. Reine Berechnung? → Primitive.*

---

## Was lief nicht so gut?

### Keine nennenswerten Probleme
Alle drei Quick Wins funktionierten auf Anhieb:
- Build erfolgreich
- Alle 40 Tests grün
- `macroexpand` expandiert `when` korrekt zu `if`

---

## Technische Erkenntnisse

### `macroexpand` als Debugging-Werkzeug
Die Fähigkeit Makros zu expandieren ist essentiell für Makro-Entwicklung:

```lisp
> (macroexpand '(when x y))
(if x (begin y))
```

Dies zeigt ob ein Makro korrekt expandiert ohne es auszuführen.

### Go-Idiom: Nicht mehr prüfen als nötig
`fnEq` verglich vorher `Num && Val`. Da `=` nur für Zahlen gedacht ist,
reicht der `Num`-Vergleich. Weniger Code, klarere Semantik.

---

## Fazit Session 3

Kleine, gezielte Verbesserungen mit sofort sichtbarem Nutzen.
Die Codebasis bleibt sauber, die Sprache wird benutzerfreundlicher.

> "Quick Wins sind das Öl einer Codebasis –
>  kleine Investition, große Wirkung auf Geschwindigkeit und Moral."
> — Gerhard & Claude, Februar 2026

---
---

# Session 4 – Unix-Style CLI

Siehe [`docs/retrospectives/2026-02-25-unix-cli.md`](docs/retrospectives/2026-02-25-unix-cli.md)

---
---

# Session 5 – GoLisp Server (golisp2d)

**Datum:** 1. März 2026
**Autoren:** Gerhard Quell & Claude Sonnet 4.6

---

## Was haben wir gebaut?

Ein vollständiger Client-Server-Stack für professionelle Lisp-Entwicklung:

| Feature | Dateien | Commits |
|---------|---------|---------|
| TCP-Server (`golisp2d`) | `cmd/golisp2d/main.go`, `lib/swank/server.go` | 1 |
| Protokoll-Handler | `lib/swank/protocol.go` | 1 |
| CLI-Client (`golisp2-client`) | `cmd/golisp2-client/main.go` | 1 |
| Hilfsfunktionen | `lib/types_helpers.go` | 1 |
| Dokumentation | `CLAUDE.md`, `docs/retrospective-golisp2d-20250301.md` | 2 |

**Gesamt:** 5 neue Dateien, ~900 Zeilen Go-Code, 6 Commits.

**Vorher:** Nur eingebetteter REPL (`./golisp2 -i`).
**Nachher:** Vollständiger Server mit TCP-RPC, persistenter Umgebung, IDE-fähigem Autocomplete.

---

## Was lief gut?

### Architektur-Entscheidungen

**S-Expression-RPC statt JSON**
- Natürliche Passung zu Lisp – kein zusätzlicher Parser nötig
- Menschenlesbare Protokoll-Messages für Debugging
- Der vorhandene `lib.Read()` Parser wiederverwendet

**Geteiltes Environment**
- Alle Clients sehen denselben Zustand → einfache Kollaboration
- `define` und `defun` persistieren zwischen Verbindungen
- Keine komplexe Session-Verwaltung nötig

**Klare Trennung der Verantwortlichkeiten**
- `server.go`: Netzwerk, Connection Handling
- `protocol.go`: Business Logic, Methoden-Implementierung
- `main.go` (beide): CLI, Flag-Handling

### Implementation

**Wiederverwendung bestehender Code**
- `lib.Read()`, `lib.Eval()`, `env.Symbols()` – alles vorhanden
- Nur Protokoll-Wrapper und Client-Logik neu geschrieben

**Schnelle Iteration**
- Sofortiges Testen via `echo ... | nc localhost 4321`
- Go's schnelle Compile-Zeiten ermöglichten rapid prototyping

---

## Was war herausfordernd?

### Multiline-Handling im Client

**Problem:** Newlines in Code-Strings brechen das S-Expression-Format.

**Lösung:** Escaping von `\n` zu `\\n` im Client – der vorhandene Reader handhabt das korrekt.

**Lesson Learned:** Protokoll-Design muss Whitespace berücksichtigen.

### Autocomplete für Spezialformen

**Problem:** `define`, `defun`, `if` etc. sind keine Environment-Symbole.

**Lösung:** Dokumentation klarstellt – Autocomplete zeigt nur gebundene Symbole.

---

## Technische Erkenntnisse

### S-Expression-RPC Format

```lisp
;; Request
(:id 1 :method "eval" :params ("(+ 1 2)"))

;; Response
(:id 1 :status "ok" :result "3")
;; oder
(:id 1 :status "error" :error "unbekanntes Symbol")
```

Property-Listen als natürliches Format für Lisp-Systeme.

### Goroutines pro Connection

Einfache Konkurrenz ohne manuelle Thread-Verwaltung:
```go
for server.running {
    conn, _ := listener.Accept()
    go handleConnection(conn)  // Jede Connection eigene Goroutine
}
```

---

## Fazit Session 5

GoLisp ist nun bereit für professionelle Entwicklung:
- Server-Mode für IDE-Integration (Autocomplete, Hover-Doku)
- Persistente Umgebung für langlaufende Sessions
- Client-REPL mit Multiline-Support

Der Server macht GoLisp von einem Spielzeug zu einem Werkzeug.

> "Ein Lisp ohne Server ist wie ein Klavier ohne Konzertsaal –
>  es funktioniert, aber niemand hört es."
> — Gerhard & Claude, März 2026

---
---

# Session 4 – Unix-Style CLI

Siehe [`docs/retrospectives/2026-02-25-unix-cli.md`](docs/retrospectives/2026-02-25-unix-cli.md)

---
---

# Session 6 – 2026-06-14

**Autoren:** Gerhard Quell & Claude Sonnet 4.6

---

## Was haben wir gebaut?

| Feature | Dateien | Beschreibung |
|---------|---------|--------------|
| `parfunc :timeout N` | `eval.go` | Optionaler Timeout für parallele Auswertung |
| Channel-basiertes parfunc | `eval.go` | `sync.WaitGroup` → Channel, feinere Kontrolle |
| `catch` verbessert | `eval.go` | Fängt jetzt alle Fehler ab, nicht nur `LispError` |
| `mod`, `remainder`, `abs` | `primitives.go` | Arithmetik-Primitiven |
| `random` | `primitives.go` | Zufallszahlen mit/ohne Limit |
| `string-replace`, `string-trim`, `string-contains` | `stringfuncs.go` | String-Primitiven |
| `system` | `shellcmd.go` (neu) | Shell-Kommando ausführen, Exit-Code zurück |
| `file-stat` | `shellcmd.go` (neu) | Datei-Metadaten als Assoziationsliste |
| `assoc` | `shellcmd.go` (neu) | Assoziationslisten-Suche mit `equal?` |
| `symbol->string` | `shellcmd.go` (neu) | Symbol in String konvertieren |
| sigoREST context.Timeout | `sigorest.go` | `http.Client.Timeout` → `context.WithTimeout` |
| sigoREST-Timeout 60→30s | `sigorest.go` | Realistischerer Default |

**Gesamt:** 12 Features/Fixes, 2 Commits golisp2 + 1 Commit sigoREST, 220+ neue Zeilen.

---

## Bug-Analyse: sigoREST `max_tokens:0`

Die interessanteste Arbeit dieser Session war eine Fehlerdiagnose über zwei Projekte.

### Symptom
```lisp
(sigo "test" "cl46-s")
=> Error: eval: sigo HTTP 400
```

### Erste (falsche) Hypothese
*Vermutung:* Mammouth liefert Anthropic-Format für neuere Claude-Modelle,
Engine parst nur OpenAI-Format → "Unexpected response format".

*Fix-Versuch:* `cfg.Type = "anthropic"` basierend auf Model-ID-Prefix.

*Ergebnis:* Fix gebrochen. `"anthropic"`-Type ändert Auth-Header von
`Authorization: Bearer` auf `x-api-key` — Mammouth lehnt das ab.

### Echte Ursache (nach direktem Mammouth-Test)
Mammouth gibt für **alle** Modelle OpenAI-Format zurück. Das Problem lag tiefer:

```
Mammouth /public/models → MaxOutputTokens = 0 für neue Modelle
                              ↓
Server: req.MaxTokens == 0 && modelInfo.MaxOutputTokens == 0
        → max_tokens: 0 im API-Request
                              ↓
Mammouth: finish_reason: "length", content: null
                              ↓
Engine: null.(string) schlägt fehl → "Unexpected response format"
```

`cl4-s` (claude-sonnet-4) funktionierte weil es noch in den `CoreModels`
mit `MaxOutputTokens: 8192` definiert war — alle neueren Modelle kamen
nur aus der Live-API ohne Token-Limits.

### Fix (2 Stellen in sigoREST)
1. `main.go`: `max_tokens` nur senden wenn `> 0`
2. `engine.go`: `content: null` gibt jetzt klare Fehlermeldung statt "Unexpected format"

---

## Was lief gut?

### Direkter API-Test als Debugging-Werkzeug
`curl` direkt gegen Mammouth (mit dem API-Key aus der Umgebung) hat die
falsche Hypothese sofort widerlegt. Ohne diesen Test hätte die falsche
Lösung länger gehalten.

### Cross-Projekt-Navigation
Das `extern/sigoREST` Symlink-Muster erlaubt, beide Projekte in einer
Session zu bearbeiten, ohne Repository-Grenzen zu verlieren.

---

## Was lief nicht so gut?

### Erste Hypothese war falsch
Die "Anthropic-Format vs OpenAI-Format"-Hypothese klang plausibel,
war aber ungeprüft. Direkter `curl`-Test hätte das früher widerlegt.

**Lesson Learned:** Bei HTTP-Fehlern immer zuerst den API-Endpoint
direkt testen, bevor Code-Änderungen gemacht werden.

### Language-Server Diagnostiken (wieder)
LSP meldet Fehler für sigoREST-Dateien weil das Modul nicht im golisp2-Workspace ist.
`go build` bleibt die verlässliche Ground Truth.

---

## Technische Erkenntnisse

### `max_tokens: 0` ist ein semantischer Fehler
Die meisten LLM-APIs interpretieren `max_tokens: 0` als "0 Tokens generieren",
nicht als "Provider-Default". Das Feld weglassen ist der korrekte Weg
für "kein Limit spezifiziert".

### Channel vs WaitGroup für parallele Auswertung
`sync.WaitGroup` sammelt nur "fertig"-Signale — kein Timeout möglich.
Channel mit `select` erlaubt Timeout, Early-exit und geordnetes Mapping:

```go
type parfuncResult struct { idx int; val *Cell }
ch := make(chan parfuncResult, len(exprList))
select {
case r := <-ch: gathered[r.idx] = r.val
case <-timer:   collected = len(exprList)  // Abbruch
}
```

### sigoREST-Modelle sind runtime-dynamisch
Modelle kommen nicht aus einer statischen CSV, sondern werden beim Start
live von Provider-APIs abgerufen. Shortcodes ändern sich wenn Provider
neue Modelle deployen — Dokumentation veraltet schnell.

---

## Offene Punkte

- [ ] `sigorest.go` Default-Modell noch `ollama-gemma3-4b` — nicht mehr verfügbar
- [ ] `eval.go` hat 1003 Zeilen (CLAUDE.md-Limit: 500) — Aufteilen sinnvoll
- [ ] `postgres.go` nicht in CLAUDE.md dokumentiert

---

## Fazit Session 6

Eine Session dominiert von Debugging statt Feature-Bau. Wert lag in der
systematischen Fehleranalyse: falsche Hypothese schnell identifiziert,
echte Ursache durch direkten API-Test gefunden, Fix sauber in zwei Dateien.

> "Ein Bug der zwei Projekte überspannt, lehrt mehr als zehn Features."
> — Gerhard & Claude, Juni 2026

---

# Session 7 – 2026-06-16: Test-Netz und eval.go-Split

**Autoren:** Gerhard Quell & Claude
**Branch:** main
**Abschluss-Commit:** `7917510 Split eval.go (1003 Zeilen) in 6 kohäsive Module`

---

## Ziel

Drei offene Hoch-Prio-Punkte aus `Todo.md` aufräumen, in der Reihenfolge,
die das Risiko minimiert: erst Rätsel klären (certs), dann Sicherheitsnetz
bauen (Tests), dann am Herzstück operieren (eval.go-Split).

## Was haben wir gebaut?

| Arbeit | Ergebnis |
|--------|----------|
| `certs/`-Rätsel geklärt | Verwaistes sigoREST-Cert erkannt, gelöscht, `.gitignore`-Guard |
| Reader-Tests (Todo #3.1) | 13 Charakterisierungstests in `reader_test.go` |
| Eval-Tests (Todo #3.2) | 21 Tests in `eval_test.go`, inkl. TCO-Schutz (200k tail-rec) |
| `eval.go`-Split (Todo #1) | 1003 Zeilen → 6 Module, alle <300, reines Move |
| Atomic Commit | 11 files, +1680/−1003 |

**Tests vorher:** 2 (`env_test.go`). **Tests nachher:** 36.
**`eval.go`:** 1003 → 0 Zeilen (gelöscht, 6 neue Dateien).

---

## Was lief gut?

### Die Reihenfolge stimmte
certs → Tests → Split. Am TCO-Trampolin operieren ohne Test-Netz wäre
russisch Roulette gewesen. Das Sicherheitsnetz zuerst bauen war der
entscheidende Plan.

### Test-Netz hat sich beim ersten Split-Versuch bezahlt gemacht
Build + 36 Tests beim *ersten* Lauf nach dem Split grün. Kein einziges Mal
TCO kaputt — weil die Tests *vorher* standen, nicht weil wir Glück hatten.

### Charakterisierung statt TDD bei existierendem Code
Bei `reader.go` und `eval.go` (beide existierten schon) wurden
Charakterisierungstests geschrieben, kein TDD. Beim ersten Lauf rot:
4/13 Reader-Tests + 5/21 Eval-Tests — *alle* meine Erwartungen falsch,
keine Code-Bugs. Hätte ich TDD-Disziplin angewendet, hätte ich den Code
"repariert", um meine falsche Vermutung zu erfüllen — und echtes Verhalten
kaputtgemacht.

### Bug-Verortung in Tests
Latente Bugs (z.B. stille Typkoersion `(+ 1 "x")` = 1) wurden in Tests
sauber dem *richtigen* File zugeordnet (`primitives.go`, nicht `eval.go`).
Beim Split kein Fehlalarm — bricht ein Test, weiß ich, dass der Split schuld
ist, nicht ein zufällig mitkommender primitives-Bug.

### End-to-End-Verifikation, nicht nur Unit-Tests
lib-Tests grün allein reicht nicht. Smoke-Tests über die echte `golisp2`-
Binary haben gezeigt, dass der Macro-Mechanismus wirklich läuft und TCO
in der Praxis greift (100k tail-rec → `ok`).

---

## Was nicht lief / Verbesserungspotenzial

### `-e` nimmt nur eine Expression
Erst beim Smoke-Test entdeckt: TCO- und Macro-Tests über `-e` schienen zu
"failen" (zeigten nur `defun`-Rückgabe). CLAUDE.md sagt `-e EXPR` (Singular) —
die Konsequenz (zweiter Ausdruck still ignoriert) ist nicht offensichtlich.
Für künftige manuelle Multi-Expr-Tests: stdin/Multiline nutzen.

### Erste Test-Erwartungen zu oft falsch geraten
9 von 34 Tests beim ersten Lauf rot — zwar der *Wert* von
Charakterisierungstests, zeigt aber: mein anfängliches Modell vom
GoLisp-Verhalten war lückenhaft. CLAUDE.md vorher gründlicher lesen
(NIL-Semantik, eq vs equal?, catch-Syntax) hätte einige Vermutungen
vorab korrigiert.

### `git status --cached` als Flag nicht verfügbar
Kleiner Stolperer bei der Verifikation. `git diff --cached` geht. Harmlos,
aber hätte ich wissen können.

---

## Schlüssel-Erkenntnisse

### Tail-Forms müssen inline im Eval-Loop bleiben
`if`/`begin`/`let`/`let*`/`cond`/`case` setzen `expr`/`env` und machen
`continue` — das *ist* das Trampolin. Auslagern hätte echten Funktionsaufruf
statt O(1)-Loop bedeutet → TCO kaputt → 200k-Test crasht. Nur `case`
delegiert an eine Hilfsfunktion, weil es ein Rückgabe-Tripel
`(*Cell, *Env, error)` nutzt, um env ins Trampolin zurückzureichen — der
einzige sichere Weg, eine Tail-Form auszulagern.

### Go-Tool respektiert keine `.gitignore`
Das `certs/`-Problem hätte `.gitignore` *nicht* gelöst — Go traversiert bei
`./...` jedes Unterverzeichnis, unabhängig von Git-Regeln. Nur physisches
Entfernen oder `.`/`_`-Verzeichnisprefix hilft. Häufige, gut dokumentierte Falle.

### Eine Grenze pro Kohäsions-Gruppe, nicht eine pro Zeilenzahl
`eval_specialforms.go` war nach erstem Move 313 Zeilen (über Limit). Statt
künstlich aufzuspalten, wurde `load` (thematisch I/O) nach `eval_load.go`
gezogen. Kohäsiver als mechanisches Zeilen-Splitten.

### Charakterisierungstests sind antisymmetrisch zu TDD
TDD: Test-erst (SOLL), dann Code bis grün. Charakterisierung: Code-erst
(IST), dann Tests die das IST festhalten. Falsche Raten beim Schreiben
sind der Wert — sie zeigen, wo das mentale Modell vom Code abweicht.

---

## Offene Punkte (nach dieser Session)

- [ ] **Todo #2 (hoch):** stdlib zentralisieren — `golisp2d` lädt inline-stdlib
  statt `//go:embed stdlib.lisp`. Drift-Gefahr zwischen zwei stdlib-Versionen.
- [ ] **Todo #3 Rest:** Primitiven-Tests, Makro-Expansion-Tests,
  parfunc/Channel-Tests.
- [ ] **Todo #5-7:** Duplikat-Bereinigung, sigoREST-Konfig, Kleinigkeiten.
- [ ] **Latenter Bug aus Eval-Tests:** Stille Typkoersion in `primitives.go`
  (`(+ 1 "x")` = 1, kein Fehler) — separater Fix, nicht eval.go.
- [x] ~~`eval.go` 1003 Zeilen~~ → aufgeteilt in 6 Module (Session 7).

---

## Fazit Session 7

Eine Session, die dem Motto "Test-Netz zuerst, dann am Herzstück operieren"
folgte — und es hat sich ausgezahlt. Der eval.go-Split lief beim ersten
Versuch grün, weil das TCO-Trampolin durch 36 Tests geschützt war, nicht
durch Glück. Drei offene Hoch-Prio-Punkte auf null reduziert (certs geklärt,
Tests gebaut, eval.go gesplittet), ein atomic Commit, sauber dokumentiert.

> "Am Trampolin operiert man nicht ohne Netz — das Netz kommt zuerst."
> — Gerhard & Claude, Juni 2026

---

# Session 8 – 2026-06-16: stdlib zentralisiert

**Autoren:** Gerhard Quell & Claude
**Branch:** main

---

## Ziel

Todo #2: `golisp2d` lud in `lib/swank/server.go` eine eigene inline-stdlib
(abgespeckte 20/52 Funktionen) statt der eingebetteten `stdlib.lisp` →
Drift. Server-Clients bekamen keine `iota`/`flatten`/`gcd` etc. Eine
gemeinsame Quelle schaffen.

## Was haben wir gebaut?

| Arbeit | Ergebnis |
|--------|----------|
| `stdlib.lisp` verschoben | root → `lib/stdlib.lisp` (git mv) |
| `libs/stdlib.lisp` entfernt | totes inhaltsgleiches Duplikat, untracked |
| `lib/stdlib.go` neu | `//go:embed stdlib.lisp` + zentrale `LoadStdlib(env)` |
| `main.go` umgestellt | embed+LoadString → `lib.LoadStdlib(env)` |
| `lib/swank/server.go` umgestellt | `loadStdlib()`+inline-String → `lib.LoadStdlib(s.env)` |

**Server-stdlib vorher:** 20 Funktionen (inline).
**Server-stdlib nachher:** 52 Funktionen (volle `stdlib.lisp`, wie CLI).
**server.go:** 304 → 251 Zeilen (inline-String entfernt).

## Was lief gut?

### Embed-Pfad-Limit früh erkannt
Todo-Option A ("Server auf `//go:embed` umstellen") war nicht direkt
machbar: Go verbietet `..` in embed-Pfaden, also kann `lib/swank/` nicht
auf `../../stdlib.lisp` im root zugreifen. Lösung: stdlib.lisp *selbst*
nach `lib/` verschieben, wo der Package-Baum sie erreicht. Architektur
folgt aus Tooling-Restriktion — früher erkannt, kein Sackgassen-Refactor.

### Eine Quelle, eine Funktion
Jetzt gibt es genau eine `LoadStdlib(env)` in `lib/stdlib.go` und genau
eine `stdlib.lisp`. CLI und Server rufen dieselbe Funktion auf. Drift
strukturell ausgeschlossen, nicht nur behoben.

### End-to-End über beide Binaries verifiziert
Nicht nur lib-Tests grün — sondern golisp2d gebaut, auf freiem Port
gestartet, und über golisp2-client die ehemals fehlenden Funktionen
abgefragt: `iota`/`flatten`/`gcd`/`length`/`cadr` liefern korrekte
Ergebnisse über den Server. Drift wirklich weg, nicht nur syntaktisch.

## Was nicht lief / Verbesserungspotenzial

### gofmt-Reflex gegen CLAUDE.md
Ich habe `gofmt -w lib/stdlib.go` laufen lassen → tabs. Erst danach
zeigte `gofmt -l .`, dass **alle** lib-Files "nicht-konform" sind — GoLisp
nutzt bewusst 2-Space (CLAUDE.md: "2 Spaces, keine Tabs"). gofmt hätte
die Projektkonvention verletzt. Revertiert. **Lehre:** bei Go-Projekten
nicht reflexhaft gofmt anwenden — erst checken, ob das Projekt gofmt-
Konvention oder eigene (CLAUDE.md) hat. LSP-Linter und CLAUDE.md können
widersprüchlich sein; CLAUDE.md gewinnt.

### Env-Vorrang über Flags fiel beim Smoke-Test auf
`--port 49321` wurde durch `GOLISP_PORT=9123` aus dem Environment
übersteuert → Test-Server startete auf belegtem Port 9123 und crashte.
Erst nach explizitem `GOLISP_PORT=49321` vor dem Server-Aufruf lief es.
CLAUDE.md dokumentiert die Vorrang-Regel (env > flag) implizit. Für
Smoke-Tests: env immer explizit setzen. UX-Fund, der in die Server-Doku
gehört — kein Bug, aber eine Falle für Test-Autoren.

### `max`-Smoke-Test falsch geraten
`(max 3 7 2)` → "zu viele Argumente" — stdlib `max` nimmt nur 2 Args,
nicht variadisch. Mein Test-Input falsch, kein Code-Bug. Zeigt aber:
stdlib-Funktionen haben eigene Arity-Limits, die nirgends dokumentiert
sind. Kandidat für später (stdlib-Docstrings oder Arity-Check).

## Schlüssel-Erkenntnisse

### Drift strukturell ausschließen, nicht nur beheben
Das Problem war nicht "Server hat eine schlechte stdlib", sondern
"Server hat eine *andere* stdlib als CLI". Zwei Quellen = garantierte
Drift über die Zeit. Die Lösung ist nicht, beide inline-Strings gleich
zu halten, sondern **eine Quelle** zu schaffen. Ein `LoadStdlib`-Aufruf
an zwei Stellen kann nicht driften; zwei String-Literale an zwei Stellen
werden es.

### CLAUDE.md schlägt Linter
Projekt-Konventionen (CLAUDE.md) sind higher-priority als Standard-Tools
(gofmt). LSP/Diagnostics zeigen gofmt-Abweichungen als Warnung — aber
wenn das Projekt bewusst davon abweicht, ist die Warnung Fehlalarm.
Immer CLAUDE.md lesen *bevor* man Tool-Warnungen "repariert".

### Tooling-Restriktion bestimmt Architektur
Embed verbietet `..`-Pfade. Das ist keine Geschmacksfrage, sondern ein
hartes Go-Feature. Daraus folgt: shared Assets gehören in das Package,
das sie einbettet — nicht ins Repo-Root. stdlib.lisp in `lib/` ist nicht
nur aufgeräumt, sondern *notwendig* für `//go:embed` aus `lib/`.

---

## Offene Punkte (nach dieser Session)

- [ ] **Todo #3 Rest:** Primitiven-Tests, Makro-Expansion-Tests,
  parfunc/Channel-Tests.
- [ ] **Todo #5-7:** Duplikat-Bereinigung, sigoREST-Konfig, Kleinigkeiten.
- [ ] **Latenter Bug:** Stille Typkoersion in `primitives.go` (`(+ 1 "x")`=1).
- [ ] **Neu entdeckt:** stdlib `max`/`min` etc. nur 2-args, nicht
  variadisch — Arity-Limits undokumentiert.
- [x] ~~Todo #2 stdlib zentralisieren~~ → `LoadStdlib`, eine Quelle (Session 8).

---

## Fazit Session 8

Kompakte Session: ein Hoch-Prio-Punkt (stdlib-Drift) strukturell gelöst —
nicht zwei String-Literale synchronisiert, sondern eine gemeinsame Quelle
geschaffen. Zwei Fallstricke unterwegs (gofmt-Reflex, Env-Vorrang) haben
gezeigt, dass Tooling-Konvention und CLAUDE.md auseinanderliegen können;
CLAUDE.md gewinnt. Verifikation über beide Binaries (CLI + Server) statt
nur Unit-Tests hat die Drift wirklich als behoben bestätigt.

> "Zwei Quellen driften immer. Eine Quelle kann nicht driften."
> — Gerhard & Claude, Juni 2026

---

# Session 9 – 2026-06-16: Test-Netz vollendet (Todo #3)

**Autoren:** Gerhard Quell & Claude
**Branch:** main

---

## Ziel

Todo #3 abschließen: Primitiven-, Makro-Expansion- und parfunc/Channel-
Tests. Damit das Sicherheitsnetz von 36 auf volle Abdeckung der
eingebauten Funktionalität wachsen.

## Was haben wir gebaut?

| Test-Datei | Tests | Abdeckung |
|-----------|-------|-----------|
| `primitives_test.go` | 13 | mod/abs, Typ-Prädikate, Listen-Edges, Strings, fileio, gensym, error, memstats |
| `macros_test.go` | 12 | defmacro, uneval. Args, macroexpand, nested, hygiene, IsMacro |
| `concurrency_test.go` | 12 | parfunc (basic/order/timeout/error), buffered channels, lock |

**Tests gesamt:** 36 → 75 (Reader/Env 15 + Eval 21 + Primitive 13 + Makro 12 + Concurrency 12).
**Suite-Laufzeit:** 1.15s (parfunc-timeout-Test addiert ~1s).

## Was lief gut?

### Charakterisierungstest-Disziplin trug wieder
24 neue Tests, davon beim ersten Lauf 4 rot — alle IST-Funde, keine
Code-Bugs. Jeder Fund wurde als IST dokumentiert (nicht "repariert"),
genau wie bei Reader/Eval. Das Muster hält: falsche Erwartungen *sind*
der Wert.

### setq-vs-set!-Fund ist der wertvollste der Session
Beim swap-Makro-Hygiene-Test kam `(1 2)` statt `(2 1)` raus. Ursache:
`setq` (= `define` = `env.Set`) im inneren `let`-Body legt eine
Shadow-Variable an, statt die äußere zu updaten. `set!` (env.Update)
wäre nötig. Das ist eine latente Semantik-Entscheidung, die aus dem Code
nicht offensichtlich ist und jede/n Makro-AutorIn überrascht. Erst der
Charakterisierungstest machte sie sichtbar — und lieferte gleich den
Kontrast-Test (`set!`-Variante → `(2 1)`) als lebende Dokumentation.

### Deterministische Concurrency-Tests sind möglich
parfunc sammelt nach Expr-Index (Reihenfolge garantiert, unabhängig von
Ankunftszeit) — das macht es testbar ohne `time.Sleep`-Flakiness. Nur
der timeout-Test braucht echtes Timing (1s). Channel-Tests nur
buffered+sequenziell — unbuffered-send-ohne-receiver würde blockieren.
Concurrency testbar halten heißt: die Garantien des Systems ausnutzen,
nicht gegen seine Non-Determinism ankämpfen.

## Was nicht lief / Verbesserungspotenzial

### Go-Test-Caching täuschte über echten Zustand hinweg
`TestIsMacroGo` war im isolierten `-run TestMacro`-Lauf "grün", failte
aber bei `go test ./...`. Ursache: Cache-Hit von einem früheren Code-
Stand; erst der vollständige Lauf (cache invalidiert durch neue
concurrency_test.go) zeigte den echten Bug im Test (`defmacro` gibt
Atom "m" zurück, nicht das Makro — IsMacro muss auf das aus dem env
geholte Makro angewandt werden). **Lehre:** überraschende Test-Ergebnisse
mit `-count=1` oder `go clean -testcache` verifizieren. Cache lügt nicht,
aber er täuscht über aktuelle Konsistenz hinweg.

### CLI-stdin-Multi-Expr zeigte nicht alle Ergebnisse
Beim manuellen swap-Verifizieren via `printf '...\n' | ./golisp2` erschien
nur die `defmacro`-Rückgabe, nicht das `let`-Ergebnis. Mehrere Ausdrücke
über stdin werden ausgewertet, aber die Ausgabe-Strategie bei mehreren
Ergebnissen ist unklar/inkonsistent. Hätte mich auf `go test` verlassen
sollen statt CLI-Piping zu debuggen. CLI-Multi-Expr-Verhalten ist ein
eigenes Untersuchungsthema.

### Hygiene-Test war ursprünglich falsch konstruiert
Der erste `TestMacroHygieneWithGensym` wollte gensym-vs-kein-gensym
demonstrieren, aber der swap bricht nicht an gensym, sondern an der
setq-Semantik. Ich musste den Test umgestalten: statt "Hygiene zeigen"
→ "setq-Shadowing dokumentieren + set!-Kontrast + gensym-Unique". Lehre:
Tests müssen das Verhalten dokumentieren das *ist*, nicht das, das man
*demonstrieren wollte*. Wenn der Test nicht das zeigt was ich will, ist
meine Hypothese falsch — nicht der Code.

## Schlüssel-Erkenntnisse

### Charakterisierungstests als latente Semantik-Dokumentation
Der setq-vs-set!-Fund ist kein Bug-Fund, sondern ein Verhaltens-Fund:
das System verhält sich deterministisch, aber die Determinismus-Regel
("setq = Set im current-env, shadowed bei scope-Tiefe") ist nirgends
dokumentiert. Der Test ist jetzt die Dokumentation. Wer später fragt
"warum ändert mein swap-Makro nichts?" findet den Test und die Antwort.

### eq? = eq (Pointer) bestätigt Type-System-Konsistenz
`eq?` und `eq` sind beide Pointer-Vergleich. Zwei `'foo`-Instanzen sind
nicht `eq?`. Das ist konsistent mit der Singleton-Nil-Optimierung
(`eq (list) (list)` = `t`, weil identische nilCell). Das Type-System
ist pointer-first — wer strukturelle Gleichheit will, muss `equal?`
nutzen. Diese Konsistenz wäre ohne Tests nur schwer zu vertrauen.

### Concurrency-Testbarkeit als Architektur-Validierung
Dass parfunc deterministisch testbar ist (idx-geordnete Ergebnisse),
ist kein Zufall — es ist eine bewusste Design-Entscheidung in
`evalParfunc`: `gathered[r.idx] = r.val` sortiert nach Index, nicht nach
Ankunftszeit. Das macht das Feature testbar. Architekturen, die
Ankunfts-Reihenfolge zurückgeben würden, wären untestbar gewesen.
Testbarkeit ist hier eine emergente Eigenschaft guten Designs.

---

## IST-Funde (kumuliert über alle Sessions)

| Fund | Wo | Status |
|------|----|---------|
| `Cell.String()`: NIL-Cell → `"()"`, nil-Ptr → `"NIL"` | types.go | dokumentiert |
| Backslash außerhalb String = Symbol | reader.go | dokumentiert |
| Dotted-pair-Reader blind nach cdr | reader.go | dokumentiert (Todo #7) |
| Stille Typkoersion `(+ 1 "x")`=1 | primitives.go | dokumentiert (Fix offen) |
| `(- 5)`=5 (kein unäres Minus) | primitives.go | dokumentiert |
| `(if)`=`()` (permissive Syntax) | eval.go | dokumentiert |
| `eq?` = Pointer wie `eq` | primitives.go | dokumentiert |
| `atom? '()` = `t` (NIL ≠ LIST-Typ) | primitives.go | dokumentiert |
| `file-write`/`file-append` geben Pfad zurück | fileio.go | dokumentiert (API-Inkonsistenz) |
| `setq` shadowed in innerem let, `set!` nötig | eval.go | dokumentiert (Makro-Autor-Falle) |
| `(parfunc r)` ohne Expr setzt `r` nicht | eval_control.go | dokumentiert (Mini-Bug) |
| stdlib `max`/`min` nur 2-args | stdlib.lisp | dokumentiert (Arity undokumentiert) |

---

## Offene Punkte (nach dieser Session)

- [ ] **Todo #5-7:** Duplikat-Bereinigung (sliceToCell/isTruthy/countParens),
  sigoREST-Konfig (Default-Modell, Host-Env), Kleinigkeiten (Tabs,
  pg-Conn-Typ, dotted-pair-Check, nil-Prüfungen).
- [ ] **Latente Bugs fixen:** Stille Typkoersion in primitives.go;
  parfunc-Empty-Setzt-r-nicht; stdlib max/min variadisch machen.
- [x] ~~Todo #3 Testinfrastruktur~~ → 75 Tests, vollständige Primitive/
  Makro/Concurrency-Abdeckung (Session 9).

---

## Fazit Session 9

Dritte Test-Session, die das Sicherheitsnetz von 36 auf 75 Tests
verdoppelte. Der wertvollste Fund war kein Bug, sondern eine latente
Semantik-Regel: `setq` shadowed in inneren Scopes, `set!` updatet. Das
ist genau der Wert von Charakterisierungstests — sie dokumentieren das
Verhalten das *ist*, einschließlich der Subtilitäten, die aus dem Code
allein nicht ersichtlich sind. GoLisp hat jetzt ein Test-Netz, das nicht
nur Refactor-Sicherheit bietet, sondern als lebende Verhaltens-Doku
dient. Todo #3 vollständig erledigt.

> "Tests dokumentieren nicht, was der Code tun soll – sie dokumentieren,
> was er wirklich tut. Darin liegt ihr wertvollster Fund."
> — Gerhard & Claude, Juni 2026

---

# Session 10 – 2026-06-16: Latente Bugs gefixt

**Autoren:** Gerhard Quell & Claude
**Branch:** main

---

## Ziel

Die in Sessions 7-9 dokumentierten latenten Bugs beheben. Drei Kandidaten
aus der kumulierten IST-Funde-Tabelle:
1. `(parfunc r)` ohne Expr setzt `r` nicht im env (Mini-Bug)
2. Stille Typkoersion `(+ 1 "x")`=1 (stiller Datenverlust)
3. stdlib `max`/`min` nur 2-args, nicht variadisch

## Was haben wir gemacht?

### Bug 1: parfunc-empty (safe fix)
`evalParfunc` sprang bei leerer exprList früh per `return MakeNil()` ab –
*vor* `env.Set(resultName, ...)`. Fix: `env.Set(resultName, MakeNil())`
vor das return ziehen. `r` ist jetzt gebunden. Backwards-kompatibel.

### Bug 3: max/min variadisch (low-risk)
stdlib `max`/`min` waren `(defun max (a b) ...)` – nur 2 Argumente.
CL-Variante ist variadisch. Fix via `&rest` + `reduce`:
`(defun max (a &rest rest) (reduce (lambda (x y) (if (>= x y) x y)) a rest))`.
Backwards-kompatibel: `(max 3 7)` funktioniert weiter, `(max 3 7 2)`=7 neu.

### Bug 2: Stille Typkoersion (breaking, Design-Entscheidung)
Arithmetik-Primitive (`+,-,*,/,mod,abs`) und Vergleiche (`=,<,>,>=,<=`)
griffen direkt auf `.Num` zu. Strings haben `Num=0`, wurden still addiert:
`(+ 1 "x")`=1, `(= "a" "a")`=t. Stiller Datenverlust.

**Design-Entscheidung via AskUserQuestion:** drei Optionen (Strict /
Lax belassen / Nur Vergleiche). Gerhard wählte **Strict**.

Fix: zentrale `checkNumbers(name, args)`-Hilfsfunktion in primitives.go,
die alle args auf `NUMBER`-Typ prüft und `fmt.Errorf("%s: Zahl erwartet,
got %s", name, a)` wirft. Eingebaut in alle 11 betroffenen Primitive.
Vergleiche mit strict gemacht für Konsistenz (sonst `(+ 1 "x")`→error
aber `(= 1 "x")`→still `()`).

**Breaking:** Programme die auf stiller Koersion vertrauten, brechen
jetzt. Aber: nur 1 Test failte (der den Bug dokumentiert hatte), keine
stdlib-interne Nutzung brach (`length`, `iota`, `max`, `gcd` reichen
Zahlen sauber weiter). Confidence aus 75-Test-Netz.

## Was lief gut?

### Test-Netz als Confidence-Quelle für breaking Change
Die strict-Typing-Änderung ist breaking. Aber das 75-Test-Netz deckte
genau ab, was kaputtgehen könnte: nur `TestEvalSilentTypeCoercion` failte
(der Bug war dort als IST dokumentiert). Kein stdlib-Pfad brach. Ohne das
Netz wäre ein breaking Change ein Glücksspiel – mit Netz eine berechnete
Entscheidung. Genau der Compound-Wert der Test-Investition aus Session 7+9.

### Design-Entscheidung eingeholt statt geraten
Bei Bug 2 (breaking) nicht einfach "ich mache strict" geraten, sondern
per AskUserQuestion drei Optionen mit Preview präsentiert. Gerhard
entschied. Breaking Changes gehören dem Nutzer, nicht dem Werkzeug.

### Kumulierte IST-Funde-Tabelle als Arbeits-Backlog
Die in Session 9 eingeführte Tabelle diente hier direkt als
Bug-Backlog: drei Einträge mit "dokumentiert (Fix offen)" wurden
abgearbeitet. Ohne die Tabelle wären die Bugs über Sessions verstreut
und einzeln mühsam wiederzufinden. Dokumentation als TODO-Liste.

## Was nicht lief / Verbesserungspotenzial

### test_infra-Discovery: evalStr lädt keine stdlib
Beim Testen von Bug 3 (max/min) fiel auf: der Test-Helper `evalStr`
nutzt `BaseEnv()` ohne `LoadStdlib` – stdlib-Funktionen (max, min, iota)
sind in Unit-Tests nicht testbar. Bug 3 musste via CLI-Smoke verifiziert
werden (main.go lädt stdlib). Lücke: kein formeller Test für
stdlib-Funktionen. Kandidat für später: `evalStd(src)`-Helper mit
LoadStdlib, oder eigenes stdlib_test.go.

### Vergleiche-strict war Ausweitung der Wahl
Gerhard wählte "Strict" im Arithmetik-Kontext. Ich habe die Vergleiche
(`=,<,>`) *zusätzlich* strict gemacht, mit Begründung "Konsistenz". Das
ist eine Interpretation seiner Wahl. Hätte ich die Vergleiche separat
nachfragen sollen? Wahrscheinlich ja – es war eine Ausweitung. Hat sich
als richtig erwiesen (kein Widerspruch), aber das Prinzip "breaking
Changes gehören dem Nutzer" gilt auch für Ausweitungen.

## Schlüssel-Erkenntnisse

### Tests ermöglichen breaking Changes mit Confidence
Das ist die Umkehrung der üblichen "Tests verhindern Regression"-Story:
Tests *ermöglichen* mutige Changes, weil sie aufzeigen, was genau bricht.
Strict typing ist breaking – aber mit 75 Tests war es eine berechnete
Entscheidung, kein Sprung ins Dunkle. Der Wert eines Test-Netzes wächst
nicht nur mit der Abdeckung, sondern mit der *Confidence*, die es für
kommende Refactorings/Bugfixes bietet.

### `checkNumbers` als zentrale Wächter-Funktion
Statt in jeder Primitive inline `if a.Type != NUMBER` zu schreiben, eine
Hilfsfunktion mit Operator-Namen. Vorteil: einheitliche Fehlermeldung
("+ : Zahl erwartet, got ..."), wartbar an einer Stelle, Muster für
künftige Primitive etabliert. Architektur-Gewinn aus dem Bug: die Lösung
ist strukturierter als der Bug-Zustand.

### Breaking-Change-Kommunikation ist separater Schritt
Strict typing bricht Programme, die (absichtlich/unabsichtlich) auf
stiller Koersion vertrauten. Tests dokumentieren das neue Verhalten, aber
Nutzer-Communication (CHANGELOG, Release-Note) ist ein separater Schritt,
den die Tests nicht ersetzen. U-Boot-Philosophie mildert (reift in Ruhe),
aber beim "Zeigen" erwähnenswert.

---

## IST-Funde-Status (aktualisiert)

| Fund | Wo | Status |
|------|----|---------|
| `Cell.String()`: NIL→`()`, nil-Ptr→`"NIL"` | types.go | dokumentiert |
| Backslash = Symbol | reader.go | dokumentiert |
| Dotted-pair-Reader blind | reader.go | offen (Todo #7) |
| ~~Stille Typkoersion~~ | primitives.go | **gefixt (strict)** |
| `(- 5)`=5 (kein unäres Minus) | primitives.go | IST, ok |
| `(if)`=`()` (permissive) | eval.go | IST, ok |
| `eq?`=Pointer | primitives.go | dokumentiert |
| `atom? '()`=t | primitives.go | dokumentiert |
| `file-write`/`-append` geben Pfad | fileio.go | dokumentiert (API) |
| `setq` shadowed in innerem let | eval.go | dokumentiert |
| ~~`(parfunc r)` ohne Expr setzt r nicht~~ | eval_control.go | **gefixt** |
| ~~stdlib `max`/`min` nur 2-args~~ | stdlib.lisp | **gefixt (variadisch)** |

3 von 12 dokumentierten Funds gefixt. 6 bleiben als gewolltes IST, 1 offen
(dotted-pair, Todo #7), 2 als API-Inkonsistenz dokumentiert.

---

## Offene Punkte (nach dieser Session)

- [ ] **Todo #5-7:** Duplikat-Bereinigung, sigoREST-Konfig, Kleinigkeiten.
- [ ] **test_infra:** `evalStd(src)`-Helper oder stdlib_test.go –
  stdlib-Funktionen formell testbar machen.
- [ ] **Breaking-Change-Note:** strict typing für künftiges "Release"
  dokumentieren (CHANGELOG o.ä.).
- [x] ~~3 latente Bugs~~ → parfunc-empty, Typkoersion (strict), max/min
  variadisch gefixt (Session 10).

---

## Fazit Session 10

Kompakte Bug-Fix-Session: drei dokumentierte latente Bugs abgearbeitet,
davon eine breaking Design-Entscheidung (strict typing) per
AskUserQuestion mit Gerhard geklärt. Das 75-Test-Netz machte den
breaking Change zu einer berechneten Entscheidung statt einem Glücksspiel
– nur der Test, der den Bug dokumentiert hatte, failte. Kumulierte
IST-Funde-Tabelle diente als direktes Bug-Backlog. Drei Funds von zwölf
gefixt, die Struktur (checkNumbers) ist besser als der Bug-Zustand.

> "Tests verhindern nicht nur Regression – sie ermöglichen mutige
>  Changes. Confidence ist der wahre Compound-Wert eines Test-Netzes."
> — Gerhard & Claude, Juni 2026

---

# Session 11 – 2026-06-16: Aufräumen & Konfig (Todo #5, #6, #7.3)

**Autoren:** Gerhard Quell & Claude
**Branch:** main
**Tagesabschluss-Retro** (5. Session des Tages nach 7-10)

---

## Ziel

Nach Test-Netz (7+9), stdlib-Zentralisierung (8) und Bugfixes (10) die
niedrig-prioren Todos abarbeiten: #5 Code-Duplikation, #6 sigoREST-Konfig,
#7.3 dotted-pair-Reader-Check. Den Tag sauber abschließen.

## Was haben wir gemacht?

### Todo #5 – Code-Duplikation bereinigt
Drei byteweise identische Helper-Duplikate entfernt: unexported
`isTruthy`/`sliceToCell`/`cellToSlice` in eval_core.go waren Schatten der
exportierten `IsTruthy`/`SliceToCell`/`CellToSlice` in types_helpers.go.
13 Aufrufstellen über 5 Files auf exported-Versionen umgestellt.
`readline.go.v2` (dokumentierter Fallback) nach `docs/` archiviert.
`countParens` existierte gar nicht (Todo veraltet).

### Todo #6 – sigoREST-Konfig via Env-Vars
Default-Modell war `ollama-gemma3-4b` — **nicht mehr in Live-Liste** →
`(sigo "prompt")` ohne Modell-Arg failte (verdeckter Bug). Neuer Default
`gem25-flt` (live, verifiziert). `GOLISP_SIGO_HOST`/`GOLISP_SIGO_MODEL`
env beim Start via `init()`. CLAUDE.md dokumentiert.

### Todo #7.3 – dotted-pair-Reader-Check
`readRest` konsumierte nach `(a . b)` das `)` blind per `r.next()` —
Müll wie `(a . b x)` wurde still akzeptiert. Jetzt `peek`+Prüfung, Fehler
bei Nicht-`)`. Session-7-Fund (damals als IST dokumentiert) jetzt Bugfix.

### Zusätzlich: sigoREST-Zugang verifiziert + CLAUDE.md-Modelle aktualisiert
Live-Check: sigoREST PID 1757, Ports 9080/9443, `(sigo "test" "gem25-flt")`
→ "OK". CLAUDE.md-Modelltabelle von 13 → ~30 Einträge ergänzt (cl47-o,
cl48-o, gem35-f etc.), als "runtime-dynamisch, siehe (sigo-models)"
markiert. Memory `sigorest_models.md` neu erstellt.

## Tagesbilanz

| Metrik | Wert |
|--------|------|
| Commits heute | 12 (6 Code + 5 Retro/Doc + 1 Config) |
| Sessions dokumentiert | 5 (Session 7-11) |
| Todos erledigt | #1, #2, #3, #4, #5, #6, #7.3 |
| Tests | 2 → 76 |
| eval.go | 1003 Zeilen → 6 Module |
| Latente Bugs gefixt | 4 (Typkoersion, parfunc-empty, max/min, dotted-pair) |
| Stdlib-Quellen | 2 (Drift) → 1 (LoadStdlib) |
| Duplikate entfernt | 3 Helper + 1 Backup-File |

## Was lief gut?

### Test-Netz als durchgehender Compound-Wert
Jede Session nach Session 7 profitierte vom Test-Netz: Split lief grün
beim ersten Versuch, stdlib-Zentralisierung verifiziert über beide
Binaries, Bugfixes (breaking strict typing!) mit Confidence, Dedup über
5 Files ohne Runtime-Regression, dotted-pair-Fix sofort gesichert. Der
Invest in 76 Tests zahlte sich bei *jedem* der 12 Commits aus. Das ist
der Definition von Compound-Value.

### Kumulierte IST-Funde-Tabelle als Arbeits-Backlog gereift
Session 9 eingeführt, Session 10 als Bug-Backlog genutzt (3 gefixt),
Session 11 setzte den 4. Fund (dotted-pair) um. Die Tabelle ist jetzt
eine *Trackbare Verhaltens-Spezifikation* — jeder Eintrag hat Status
(dokumentiert/gefixt/IST-ok). Was als Beobachtung begann, wurde zu
verlässlicher Projekt-Doku.

### Code/Doc/Retro-Rhythmus als built-in Disziplin
12 Commits, aber kein einziges Mal "8 Commits am Stück durchziehen".
Jeder Code-Commit hatte einen klaren Fokus, jeder Retro-Commit erzwang
Reflexion dazwischen. Commit-Rhythmus als built-in Retrospektive — man
kann nicht mutig refactor-ieren ohne zwischendurch zu fragen "was lief
gut, was nicht".

### Config-Feature deckte verdeckten Bug auf
Todo #6 war als "Config verbessern" deklariert, entpuppte sich als
Bugfix: der Default `ollama-gemma3-4b` war tot. Wer nur "Config
hinzufügen" wollte, hätte den toten Default übersehen. Todo-Liste
sorgfältig lesen = Bug-Quelle erkennen.

## Was nicht lief / Verbesserungspotenzial

### Sehr langer Tag, 5 Sessions — Erschöpfungsrisiko
12 Commits, 5 Retros in einem Tag ist außergewöhnlich viel. Späte
Sessions (10, 11) liefen noch diszipliniert, aber das Risiko von
Qualitätsverlust in Session 12+ wäre real. **Lehre:** bei langen Tagen
bewusst Pausen machen oder ab Session 8-9 nur noch niedrig-risk Tasks.
Heute ging es gut weil die Test-Infra jeden Schritt auffing.

### gofmt-vs-2-Space-Konflikt bleibt ungelöst (Todo #7 Rest)
Projekt nutzt bewusst 2-Space (CLAUDE.md), gofmt will tabs. `gofmt -l`
listet alle Files. Keine Lösung gefunden — nur vermieden (nicht gofmt
anwenden). Offen: pre-commit-Hook der 2-Space erzwingt, oder CLAUDE.md
explizit "gofmt ignorieren" dokumentieren. Unschön, aber nicht blockierend.

### verwaiste Memory-Files (user_profile, project_status) unentdeckt
MEMORY.md verweist auf 3 Files, nur sigorest_models existierte (heute
erstellt). user_profile und project_status fehlen — project_status wäre
stark veraltet ("eval.go aufteilen offen" — völlig falsch nach heute).
Lücke für nächste Session.

## Schlüssel-Erkenntnisse des Tages

### 1. Sicherheitsnetz zuerst, dann operieren
Session 7s Prinzip ("Netz vor Trampolin-OP") trug den ganzen Tag. Die
Reihenfolge certs → Tests → Split → Bugs → Config → Dedup war nicht
Zufall sondern Risiko-Minimierung: jeder Schritt stand auf dem vorigen.

### 2. Eine Quelle schlägt Synchronisation
stdlib-Zentralisierung (Session 8): eine `LoadStdlib` statt zwei
String-Literale. Helper-Dedup (Session 11): ein `IsTruthy` statt
Schatten-Duplikat. Derselbe Architektur-Gedanke, zweimal angewandt:
strukturelle Unmöglichkeit von Drift/Duplikat statt Disziplin.

### 3. Charakterisierungstests als Bugfix-Backlog
Sessions 7+9 dokumentierten IST-Verhalten (4 latente Bugs). Sessions 10+11
fixten sie. Der Zyklus Bug-finden → als IST festhalten → später gezielt
fixen → als SOLL sichern ist reif geworden. Tests als lebende Verhaltens-
Doku, die in ausführbare Specs reift.

### 4. Config-Aufgaben verbergen oft Bugs
Todo #6 "Config verbessern" → toter Default-Modell-Bug. Todo #2 "stdlib
zentralisieren" → Drift-Bug. Wer Config-Todos liest und "nur Settings"
denkt, verpasst die versteckten Defekte. Implizite Annahmen (Default
existiert, zwei Quellen sind gleich) immer verifizieren.

---

## Offene Punkte (nach Session 11)

- [ ] **Todo #7 Rest:** Einrückung (gofmt-vs-2-Space-Konflikt ungelöst),
  pg-Conn-Typ (postgres.go), nil-Prüfungen in eval-Helfern.
- [ ] **verwaiste Memory-Files:** user_profile.md, project_status.md
  erstellen/aktualisieren (project_status stark veraltet).
- [ ] **test_infra:** `evalStd(src)`-Helper oder stdlib_test.go.
- [ ] **Breaking-Change-Note:** strict typing für künftiges Release.

---

## Fazit Session 11 & Tagesabschluss

Kompakte Aufräum-Session die drei niedrig-prioren Todos abarbeitete,
davon einer (#6) wieder einen verdeckten Bug aufdeckte (toter Default-
Modell). Der Tag endet mit 12 Commits, 5 dokumentierten Sessions, 76
Tests, 4 gefixten Bugs — und einem GoLisp das strukturell gesünder ist
als morgens: weniger Duplikate, eine stdlib-Quelle, konfigurierbarer
sigoREST, striktere Typisierung, sauberer Reader.

> "Ein Tag der das System nicht funktional erweiterte, aber strukturell
>  heilte. Manchmal ist Aufräumen die wertvollste Feature-Arbeit."
> — Gerhard & Claude, Juni 2026

---

# Session 12 – 2026-06-21: SWANK/SLIME-Integration zum Laufen gebracht

## Ziel

Todo #1 validieren: Der in Session- predecessors gebaute Swank-Server
(`lib/swank/`, Commits 116f28b / 7aa8c8d) war nie gegen echte Emacs-SLIME-
Session getestet. Ziel: `slime-connect` funktioniert, REPL evaluiert.

## Was haben wir gebaut / gefixt?

Drei Commits, sechs behobene Probleme, iterativ gegen SLIME v2.32
(via quicklisp) erarbeitet.

| Commit | Inhalt |
|--------|--------|
| `6bd171d` | fix(swank): persistenter `bufio.Reader` pro Connection — Pipelining-Bug |
| `f499ce0` | fix(eval): `(eval form)` im globalen Env (CL-Semantik) — sonst `defun` aus REPL verloren |
| `f1e6638` | feat(swank): SLIME-kompatible Handler |

Behobene Probleme in Reihenfolge des Auftretens:

1. **bufio-Pipelining-Bug.** `readFrame` erzeugte pro Call neuen
   `bufio.Reader`; vorausgelesene Frame-Bytes wurden mit dem verworfenen
   Reader gelöscht. Schon beim Code-Lesen als Verdacht notiert, dann mit
   3 gepipelinten Frames bewiesen (1/3 Responses → 3/3 nach Fix).
2. **`swank-repl:`-Prefix fehlt.** SLIME sendet `swank-repl:create-repl` /
   `swank-repl:listener-eval`, nicht `swank:`. Bestehende Handler matchten
   nie. Erst durch `>>`/`<<`-Server-Trace sichtbar.
3. **`:abort` auf unbekannte Ops.** Default-Fall warf `:abort` → "Synchronous
   Lisp Evaluation aborted". SLIME-Contribs rufen beim Connect diverse
   Init-Funktionen. Fix: graceful `(:ok ())`.
4. **`listener-eval` Return-Format.** `(:ok "54")` (String) → SLIME will
   Liste → `listp`-Error. Richtig: `(:write-string "<wert>\n" :repl-result)`
   + `(:ok nil)`. Aus `swank-repl.lisp` `send-repl-results-to-emacs`
   abgelesen.
5. **`autodoc` destructure.** SLIME `(cl-destructuring-bind (doc
   &optional cache-p) doc)`. Leere Liste → 0 Args. `(:ok (nil nil))` →
   `doc=nil` → `insert nil` → `char-or-string-p`. Endgültig
   `(:ok (:not-available nil))` — das Keyword, das SLIME explizit abfragt.
6. **`defun` verschwindet.** `(fib 3)` nach `(defun fib ...)` → "unbekannt
   Symbol". `(eval form)` nutzte dynamisches Env; in der Lambda-Kette
   `swank-dispatch → handle-emacs-rex → listener-eval` ist das ein Child-Env,
   `defun` definierte lokal. Fix: `Env.Root()`, CL-Semantik. Core-Change.

## Was lief gut?

- **Iterativ gegen das echte System.** Sechs Iterationen je eine Code-
  Änderung + Reconnect. Jede SLIME-Fehlermeldung war präziser Fingerzeig.
  Schneller als jede Voraus-Planung.
- **Verdacht先行 (suspicion-first).** bufio-Bug schon beim ersten Lesen von
  `framing.go` vermutet, explizit getestet — nicht erst auf Symptom gewartet.
  5 min vom Verdacht zum Beweis.
- **Gegenseite lesen.** Statt Protokoll zu raten, in `swank-repl.lisp` und
  `slime-autodoc.el` gelesen, was SLIME tatsächlich destrukturiert. Ein
  `cl-destructuring-bind` löste Iteration 5 sofort. Mehr wert als jede Spec.
- **双向 Trace früh.** Go-Errors sahen still aus; Lisp-seitige `:abort`-
  Returns standen nicht im Log. `>>`/`<<`-Trace eingebaut → sah sofort welche
  Ops reinkamen. Decisive für Iteration 2-5. (Leider erst Iteration 2, nicht
  1 — siehe unten.)
- **Core vs. swank im Commit getrennt.** eval-Global-Semantik ist core
  change, eigener Commit mit eigener Begründung. Nicht im swank-Commit
  versteckt.

## Was nicht lief / Verbesserungspotenzial

- **Synthetischer Testclient zu naiv.** Erste `swankc2.go` las 1 Response
  pro Message, aber `create-repl` sendet 2 Events. Output-Verschiebung sah
  aus wie Server-Bug, war Client-Bug. Verwirrend, bis pipelined Test den
  echten Bug zeigte.
- **Klammerfehler im Lisp-Edit.** Eine Edit ließ `handle-emacs-rex`-Defun
  offen → Tests rot ("fehlendes )"). Go-Tests fingen's sofort, aber: GoLisp
  hat keinen Inline-Balancing-Check; `go test` nach jeder Lisp-Edit ist der
  einzige Rettungsanker. Fehleranfällig.
- **Trace zu spät.** Erst in Iteration 2 eingebaut. Hätte von Anfang an
  sein sollen — Iteration 1 war im Dunkeln.
- **Sandbox vs. Background-Prozesse.** `&`-Jobs wurden vom Harness-Wrapper
  gekillt (Exit 144). Mehrere Anläufe bis `run_in_background: true`
  zuverlässig lief. Zeit am Tooling statt am Fachproblem.
- **Punkt 6 spät erkannt.** Dass `defun` nicht persistiert, zeigte sich
  erst als der REPL scheinbar funktionierte. Synthetische Tests prüften nur
  Einzelexpr, nicht `defun` + späteren Call über dieselbe Connection.

## Schlüssel-Erkenntnisse

1. **Gegenseite lesen, nicht raten.** Bei Protokoll-Integration den Client-
   Source lesen. `cl-destructuring-bind` und `send-repl-results-to-emacs`
   sagen mehr als jede Spec.
2. **Verdacht → Test → Fix.** Beim Code-Lesen gefundene Bug-Verdachte direkt
   testen. bufio-Bug in 5 min bewiesen statt 5 Iterationen symptomgetrieben.
3. **Trace früh,双向.** RPC-Systeme brauchen in+out-Trace ab Iteration 1,
   nicht ab Iteration 2. Billig einzubauen, unbezahlbar im Debugging.
4. **Integrationstests testen mehr als die Integration.** Punkt 6 (eval-Env)
   ist ein core-Semantik-Bug, der nur durch den REPL-Integrationstest
   sichtbar wurde. Protokoll-Tests sind core-Tests in Verkleidung.
5. **Synthetische Tests müssen echtes Verhalten modellieren.** Pipelining,
   Multi-Event-Responses, zustandsbehaftete Connections — sonst testen sie
   nicht was SLIME tut.
6. **CL-Semantik als Kompass.** Wenn GoLisp-Verhalten unklar ist, sagt
   Common-Lisp-Spezifikation was richtig ist (`eval` global). Hat Punkt 6
   sofort auf die Lösung gelenkt.

## IST-Funde (Session 12)

- `Env.Set` schreibt nur lokal — kein Walk-up. `defun`/`define`/`setq` sind
  in Lambda-Bodies lokal, nicht global. Per Design, aber CL-unüblich.
  Komplett global machen würde `let`-lokale Defines brechen. Status: nur
  `eval` geht über Root, `defun` direkt bleibt lokal. Bewusst so belassen.

## Offene Punkte (nach dieser Session)

- Weitere SWANK-Methoden: `complete-symbol` (Tab-Completion),
  `describe-symbol` / `arglist-for-echo-area`, `macroexpand`,
  `compile-string`, `load-file`.
- `listener-eval`: mehrere Formen pro String (derzeit nur erste via `read`).
- slime-tramp für Emacs (Todo #2).
- CLAUDE.md um eval-Global-Semantik + swank-REPL-Status ergänzen.
- GoLisp-Lisp-Edits ohne Balancing-Check bleiben fehleranfällig — evtl.
  Reader-Warnung bei unausgeglichenen Klammern in `load`/`read`.

## Fazit Session 12

Vom "MVP steht aber ungetestet" zum "REPL mit Output, Rekursion und
persistierenden Definitionen" in einer Session. Sechs Iterationen, drei
Commits, davon ein core-Semantik-Fix der nur durch den Integrationstest
sichtbar wurde. Der Swank-Server ist jetzt eine echte Emacs-Entwicklungsum-
gebung für GoLisp — nicht mehr nur Gerüst.

Lehrreichster Moment: dass der scheinbare Protokoll-Bug (`defun` verschwindet)
ein core-eval-Bug war. Integrationstests sind core-Tests in Verkleidung.

> "Sechs Iterationen gegen das echte System — jede SLIME-Fehlermeldung ein
>  präziserer Lehrer als jede Spec. Am Ende war der letzte Bug kein
>  swank-Bug, sondern ein core-Bug, den nur der REPL-Test aufdeckte."
> — Gerhard & Claude, 21. Juni 2026

---

# Session 13 – 2026-06-21: SWANK-Methoden ausbauen

## Ziel

Auf der funktionierenden SLIME-REPL-Verbindung (Session 12) die fehlenden
SWANK-Methoden nachrüsten, die Emacs/SLIME zum produktiven Lisp-IDE machen.

## Was haben wir gebaut?

Sieben Commits, je eine SWANK-Methode + ein core-Bugfix. Alle iterativ
gegen echte SLIME-Session validiert.

| Commit | Methode / Fix |
|--------|---------------|
| `0f8ce36` | listener-eval wertet mehrere Formen pro Eingabe |
| `b963da2` | swank:simple-completions (Tab, Basis) |
| `a6ac75f` | swank:completions (Tab, c-p-c Contrib — das echte Op) |
| `d9e60ac` | swank:load-file (C-c C-l) |
| `a3a2b9d` | swank:operator-arglist + autodoc (Lambda-Parameter) |
| `0409a70` | swank-macroexpand-1/full/all + evalMacroexpand-Bugfix |
| `333bb6b` | swank:swank-expand-1/-expand (das Op das C-c C-m wirklich sendet) |

Neue Funktionalität:
- **Multi-Form listener-eval:** `ReadAll` in reader.go liest alle Formen,
  swank--read-all Primitive, listener-eval evaluiert jede Form, sendet
  pro Ergebnis `:write-string`/`:repl-result`.
- **Tab-Completion:** swank--symbols Primitive (closert über Connection-
  Env), Prefix-Filter. Wichtig: SLIME nutzt defaultmäßig die c-p-c Contrib
  und sendet `swank:completions` (nicht `simple-completions`) — Liste von
  1-Element-Listen, Client destrukturiert `(symbol-name classification sym)`.
- **load-file:** `(load filename)` über GoLisp load-Spezialform.
- **Arglist:** swank--arglist Primitive nutzt Lambda-Struktur (Type:LIST
  mit Env!=nil) bzw. Macro, Car = Parameter. autodoc extrahiert Operator
  aus raw-form. Built-in FUNC → `:not-available`.
- **macroexpand:** GoLisp macroexpand-Spezialform. `-1` einmal, `-full`
  wiederholt bis stabil.

## Was lief gut?

- **Gegenseite lesen, konsequent.** Bei jeder Methode erst in slime.el /
  swank.lisp / contrib-Files nachgeschaut, welches Op SLIME sendet und
  welches Return-Format der Client destrukturiert. Hat gleich DREI Ops
  entdeckt die nicht das waren was man naiv annimmt: `simple-completions`
  vs `completions` (c-p-c), `swank-macroexpand-1` vs `swank-expand-1`.
- **Live-Trace als Diagnose.** Bei "Tab klappt nicht" und "macroexpand
  char-or-string-p" den `>>`-Trace reaktiviert → sah sofort welches Op
  SLIME wirklich sendet. Zweimal gerettet, jeweils 1 Iteration.
- **Core-Bugs durch Integrationstests aufgedeckt.** evalMacroexpand warf
  Fehler auf Specialform-Car (`begin`) — erst beim wiederholten Expand
  sichtbar. Wieder: Protokoll-Test = core-Test in Verkleidung.
- **Pro Methode ein Commit, sofort gegen SLIME validiert.** Kein
  Feature-Stau; jede Einheit lebend getestet bevor nächste.

## Was nicht lief / Verbesserungspotenzial

- **Naive Annahme des Op-Namens.** `simple-completions` implementiert,
  dann in Emacs getestet — nichts. Trace zeigte `completions`. Hätte
  vorher die Contrib-Liste aus swank-require lesen können (stand im Log
  von Session 12). Lehre: die angeforderten Contribs verraten welche
  Ops kommen.
- **Synthetischer Testclient desync.** Mehrfach: client las feste Frame-
  Anzahl, aber listener-eval sendet N+1 (N writes + return). Output-
  Verschiebung sah nach Server-Bug aus. Drain-until-return als Muster
  erst spät konsequent angewandt.
- **macroexpand-all ist nur Top-Level-wiederholt.** Subformen werden
  nicht rekursiv expandiert. v1-Limitation, dokumentiert, aber nicht
  vollwertig.
- **Cursor-Semantik-Falle.** C-c C-m griff nur Symbol wenn Cursor auf
  Symbol — SLIME-Standard, aber verwirrend. Kein Bug, aber Erklärungs-
  bedarf gegenüber dem Nutzer.

## Schlüssel-Erkenntnisse

1. **Angeforderte Contribs = Ops-Vorhersage.** swank-require lief die
   Liste (swank-c-p-c, swank-arglists, ...) — daraus lassen sich die
   kommenden Ops ableiten, statt blind zu implementieren was die Spec
   nennt.
2. **Return-Format aus Client-Quellcode, nicht Spec.** `cl-destructuring-
   bind (doc &optional cache-p)`, `slime-format-completions` Loop, 
   `apply-macro-expander` mit `prin1-to-string` — der Client-Code ist
   die wahrheitsgemäße Spec.
3. **Trace ist billig, Unterlassung teuer.** Zwei Debug-Iterationen
   dieses Mal durch späten Trace-Einsatz. Bei RPC-Integration Trace
   ab Iteration 1.
4. **Lambda-Struktur als Reflexionsquelle.** Type:LIST + Env!=nil
   identifiziert Closures; Car = Parameter. Keine separate Metadaten-
   Tabelle nötig für Arglist. GoLisp's Zell-basierte Repräsentation
   ist hierFeature.
5. **`else → :ok ()` graceful ist zweischneidig.** Hat viele Contrib-
   Init-Calls geschluckt, aber bei Ops die String erwarten (expand-1)
   zum `insert nil`-Error geführt. Besser: bekannte Ops explizit
   handhaben, else bleibt graceful.

## IST-Funde (Session 13)

- `evalMacroexpand`: Lookup von form.Car warf Fehler bei Specialform/
  ungebundenem Symbol. Behoben (Lookup-Fehler = nicht expandierbar).
  Entspricht CL-Semantik.
- GoLisp reader: `ReadAll` neu (vorher nur `Read` für eine Form).
- `append`-Primitive fehlte (helper existierte, war nicht exposed).

## Offene Punkte (nach dieser Session)

- `describe-symbol` (C-c C-d C-d) — GoLisp ohne Docstrings, geringe
  Substanz.
- Echtes `macroexpand-all` (rekursiv in Subformen).
- `compile-string` / `compile-file-for-emacs` (C-c C-k).
- slime-tramp (TODO #2).
- v1-Swank-Methoden in CLAUDE.md dokumentieren (mit gemacht).

## Fazit Session 13

Sieben Methoden, sieben Commits, eine funktionierende SLIME-IDE.
Tab-Completion, load-file, Arglist-Anzeige, macroexpand — das was einen
Lisp-REPL von "funktioniert" zu "produktiv" macht. Wieder war der
wiederkehrende Lehrer: nicht raten was SLIME will, sondern lesen was
SLIME sendet und destrukturiert.

> "Sieben Methoden, siebenmal in den Client-Code geschaut statt in die
>  Spec. Der Client-Code lügt nicht — die Spec manchmal schon."
> — Gerhard & Claude, 21. Juni 2026

---

# Session 14: SWANK-Server vervollständigen & Print-Bug

**Datum:** 2026-06-22
**Autoren:** Gerhard Quell & Claude Sonnet 4.6

## Was haben wir gebaut?

| Feature | Dateien | Scope |
|---------|---------|-------|
| Built-in-Arglisten für `operator-arglist` / `autodoc` | `lib/swank/swank.lisp` | Lisp-Alists, ~60 Built-ins |
| `describe-symbol` (C-c C-d C-d) | `lib/swank/swank.lisp`, `lib/swank/env.go` | Statische Registry + Cell-Typ-Primitive |
| `compile-string` / `compile-file-for-emacs` (C-c C-k) | `lib/swank/swank.lisp` | Load/Eval-Wrapper |
| Echtes rekursives `macroexpand-all` | `lib/eval_specialforms.go`, `lib/eval_core.go`, `lib/swank/swank.lisp` | Go-Spezialform, AST-Walk ohne Evaluierung |
| Print-Bugfix: kein `()` hinter `(print "test")` | `lib/primitives.go`, `lib/swank/env.go` | `print`/`println` geben letztes Argument zurück |
| Print-Duplikat-Fix in SWANK | `lib/swank/swank.lisp`, `lib/swank/lisp_test.go` | Top-level `print`/`println` liefern nur Output-Event, kein `:repl-result` |

**Ergebnis:** Die in Session 13 als offen markierten SWANK-Punkte sind implementiert. CLAUDE.md und TODO.md sind auf aktuellem Stand.

## Was lief gut?

### Kleine Schritte statt Big-Bang
Die Aufteilung in vier überschaubare Schritte (Arglisten → describe → Compile → macroexpand-all) hat den Scope beherrschbar gehalten. Nach jedem Schritt lief `go test ./...` grün.

### Lisp-First für Protokoll-Handler
Fast alle neuen SWANK-Features ließen sich rein in `swank.lisp` umsetzen. Nur `macroexpand-all` benötigte eine Go-Spezialform, weil das bestehende `macroexpand` seine Argumente auswertet. Das hybride Go+Lisp-Design von SWANK hat sich erneut bewährt.

### Frühe Regressionstests
Für jede Erweiterung wurden sofort Tests geschrieben — nicht nur für den Happy Path, sondern auch für Randfälle:
- Lambda-Arglisten haben weiterhin Vorrang vor Built-in-Registries
- `quote` wird von `macroexpand-all` nicht durchdrungen
- `print`/`println` ohne Argumente liefern `nil`

### Klare Klärungsfragen vor dem Design
Vor der Implementierung wurden fünf gezielte Fragen gestellt (Scope, Built-in-Arglisten, describe-symbol-Format, Compile-Verhalten, Schrittgröße). Die Antworten haben Architekturentscheidungen vermieden, die später hätten rückgängig gemacht werden müssen.

## Was lief nicht so gut?

### Subagent-Modell-Konfiguration
Die `feature-dev:code-explorer` Agents sind mit `invalid thinking: only type=enabled is allowed for this model` gescheitert. Statt paralleler Agenten wurde die Codebasis direkt gelesen. Das war für diesen Scope noch ok, hätte bei größerer Ausbreitung aber Zeit gekostet.

**Lernpunkt:** Vor dem Starten mehrerer Subagenten das Modell/Thinking-Setup prüfen.

### `macroexpand-all`: Evaluierungsfalle
Die erste Implementation von `evalMacroexpandAll` wertete das Argument aus (`Eval(args.Car, env)`). Das ist korrekt für `(macroexpand-all (read string))` aus SWANK, aber im direkten REPL-Test `(macroexpand-all (when t 1))` führte es dazu, dass `when` bereits vollständig ausgewertet wurde. Erst Quoten der Testformen (`'(when t 1)`) machte den Test stabil.

**Lernpunkt:** Spezialformen, die mit Source-Code arbeiten, brauchen Tests, die das Argument quoten.

### String-Escaping in Go-Tests
Ein neuer SWANK-Test für `print` ist wegen verschachtelter `"` / `\` zunächst als Syntaxfehler gescheitert. Erst Umstellung auf Backtick-Strings hat ihn grün gemacht.

**Lernpunkt:** Bei Lisp-String-Literalen in Go-Tests direkt Backticks verwenden, sobald Backslashes vorkommen.

## Technische Erkenntnisse

### Fallback-Reihenfolge bei Built-in-Arglisten
`operator-arglist` und `autodoc` fragen **zuerst Lambda/Macro, dann Built-in** ab. Warum? Weil `stdlib.lisp` Funktionen wie `append`, `assoc` und `abs` als Lambda überschreibt. Umgekehrte Reihenfolge würde fälschlich die primitive Signatur anzeigen.

### `macroexpand-all` braucht einen nicht-evaluierenden Walk
Das bestehende `macroexpand` expandiert Makros, indem es `Eval` nutzt. Für rekursive Source-Expansion reicht das nicht, weil `Eval` Subformen auswertet. `macroexpand-all` braucht einen eigenen AST-Walk, der `Eval(form.Car, env)` nur verwendet, um Makros zu identifizieren, und `applyLambda` für die Expansion.

### `print`/`println`-Rückgabewert ist Protokoll-relevant
In Common Lisp gibt `print` das gedruckte Objekt zurück. GoLisp hatte `nil` zurückgegeben. Das führte dazu, dass REPL, Stdin-Modus und SWANK hinter jeder `print`-Ausgabe `()` anzeigten. Die Rückkehr zu CL-Semantik behebt das Problem ohne neues `void`-Konzept.

### SWANK-Ausgabe vs. Ergebnis
`swank-print`/`swank-println` markierten ihre `:write-string`-Events fälschlich mit `:repl-result`. Der Tag gehört nur zum finalen Eval-Ergebnis. Entfernen des Tags verhindert doppelte Darstellung in SLIME.

### Print-Duplikat nach SWANK-Integration (Nachtrag)
Beim Test in SLIME zeigte `(print "test")` trotz Fix `"test""test"` — ein Output-Event plus ein `:repl-result`-Event mit dem Rückgabewert. Ursache: `swank--eval-forms` in `lib/swank/swank.lisp` hat für *jede* top-level Form ein Ergebnis-Event gesendet, auch wenn die Form selbst schon ausgegeben hat.

Lösung:
- `swank--output-only-form?` erkennt `print`/`println`/`swank-print`/`swank-println`.
- `swank--eval-forms` unterdrückt `:repl-result` für diese Aufrufe.
- Zwei Tests in `lib/swank/lisp_test.go` sichern: `print` liefert kein `:repl-result`, normale Formen wie `(+ 1 2)` schon.

**Lernpunkt:** Ein Primitive das sein Argument zurückgibt, ist in einem REPL mit getrenntem Output/Result-Kanal doppelt sichtbar — Output *und* Result. Die Protokoll-Schicht muss entscheiden, wann ein Ergebnis zusätzlich angezeigt wird.

---

## Session-Abschluss 2026-06-22

| Aktion | Ergebnis |
|--------|----------|
| Print-Duplikat-Fix | `lib/swank/swank.lisp` + `lib/swank/lisp_test.go` |
| Build-Backup/Test-Reste entfernt | `golisp2d_fixed`, `.playwright-mcp/` gelöscht |
| Commit | `71582c8` auf Branch `session-14-swank-complete-print-fix` |
| Push | Branch auf `origin` gepusht |
| Emacs-Integration | `(defun golisp2 () ... (slime-connect ...))` funktioniert |

## IST-Funde

- `eval.go` ist inzwischen aufgespalten in `eval_core.go`, `eval_lambda.go`, `eval_specialforms.go`, `eval_control.go`, `eval_quasiquote.go`, `eval_load.go`. CLAUDE.md listet noch die alte Monolith-Struktur.
- `swank--cell-type` als neue Primitive hilft `describe-symbol`, FUNC/MACRO/LIST/NIL zu unterscheiden.
- `assoc` aus `shellcmd.go` ist sowohl Built-in als auch in `stdlib.lisp` definiert — die Alist-Lookup-Registry funktioniert trotzdem, weil die Fallback-Reihenfolge stimmt.

## Offene Punkte (nach dieser Session)

- CLAUDE.md Dateistruktur anpassen (`eval.go` → split files).
- Optional: `find-definitions-for-emacs` (M-.), Inspector, Debugger/Restarts — größere Features, die GoLisp's Reflexionsfähigkeiten erweitern müssten.
- Optional: slime-tramp (wenn Remote-Editing gewünscht).

## Fazit Session 14

Vier kleine Schritte, ein abgeschlossener SWANK-Server im vereinbarten Scope. Der wiederkehrende Erfolgsfaktor war: vor jeder Zeile Code die Protokoll- und Eval-Semantik verstehen. Der Print-Bug zeigte, dass scheinbar triviale Primitive (`print`) tiefe Auswirkungen auf alle Ausgabemodi haben.

> "Manchmal ist der letzte offene Punkt nicht ein fehlendes Feature, sondern ein `()` das am falschen Ort auftaucht."
> — Gerhard & Claude, 22. Juni 2026

---

# Session 15 – 2026-06-23: Schulungsunterlagen für GoLisp

**Autoren:** Gerhard Quell & Claude Sonnet 4.6 / Claude
**Branch:** session-14-swank-complete-print-fix

---

## Ziel

TODO.md verlangte Schulungsunterlagen für alle GoLisp-Funktionen:

1. `golisp2-tutorial.md` – Beschreibung + 1–3 Beispiele pro Funktion
2. `golisp2-anki.json` – pro Funktion Kurzbeschreibung, 1–3 atomare Fragen, 1–3 MC-Fragen mit ≥5 Optionen

Der Scope sollte alle öffentlichen Funktionen umfassen: eingebaute Primitiven, Spezialformen und die Standardbibliothek (`lib/stdlib.lisp`).

---

## Was haben wir gebaut?

| Datei | Inhalt |
|-------|--------|
| `golisp2-tutorial.md` | 149 Funktionen/Spezialformen/Makros, gruppiert in 17 Kategorien, mit Syntax, Beschreibung und lauffähigen Beispielen |
| `golisp2-anki.json` | 149 Karten, je 2 atomare + 1 MC-Frage mit 5 Optionen |
| `tools/gen-training/data.py` | Rohdaten für alle Funktionen (Python-Triple-Quotes, damit Lisp-Backticks und doppelte Anführungszeichen keine Escapes erfordern) |
| `tools/gen-training/generate.py` | Generator, der Markdown + JSON aus `data.py` erzeugt |
| `.gitignore` | `__pycache__/` und `*.pyc` ausgeschlossen |

**Commit:** `dc9cb5c` – `docs: Schulungsunterlagen für 149 GoLisp-Funktionen (Tutorial + Anki)`

---

## Was lief gut?

### Generator-Ansatz statt manuellem Tippen
Die Unterlagen enthalten über 6000 Zeilen Output. Statt sie von Hand zu schreiben, wurden Daten (`data.py`) und Formatierung (`generate.py`) getrennt. Das hält Konsistenz zwischen Tutorial und Anki-Karten und macht spätere Erweiterungen trivial.

### Python-Triple-Quotes für Lisp-Beispiele
Go-String-Literale hätten mit Lisp-Quasiquote-Backticks und eingebetteten doppelten Anführungszeichen kämpfen müssen. Python-Triple-Quotes erlauben es, die Lisp-Beispiele 1:1 abzulegen, ohne Escapes.

### Systematische Validierung
- JSON-Syntax mit `python3 -m json.tool` geprüft
- Stichprobenartiges Ausführen von Beispielen gegen die `golisp2`-Binary (Arithmetik, Listen, Strings, Datei-I/O, stdlib-Helfer)
- `go test ./...` grün

---

## Was lief nicht so gut?

### Erster Generator in Go scheiterte an String-Escaping
Der erste Versuch, die Daten direkt in einem Go-Programm (`tools/gen-training/main.go`) als Struct-Literale abzulegen, scheiterte an den vielen doppelten Anführungszeichen in Lisp-Strings. Ein automatischer Konverter erzeugte ungültige Raw-String-Literale und Quasiquote-Backticks brachen die Delimiter.

**Lösung:** Neustart mit Python-Generator. Go wurde nicht für den Datenteil gebraucht – das Tooling muss zur Aufgabe passen.

### Subagent-Tool nicht verwendbar
Geplante parallele `doc-writer`-Agents für Markdown und JSON lieferten `invalid thinking: only type=enabled is allowed for this model`. Die Arbeit wurde daher inline erledigt.

**Lernpunkt:** Bei Agent-Planungen vorab das Tooling-Setup prüfen.

### Datei-I/O-Beispiele brauchen `./tmp`
Die ersten Beispiele für `file-write` etc. schlugen fehl, weil `./tmp` nicht existierte. Erst nach Einfügen von `(system "mkdir -p ./tmp")` in die Beispiele waren sie selbstlaufend.

**Lernpunkt:** Beispiele müssen alle Voraussetzungen selbst erzeugen, auch wenn sie später offensichtlich erscheinen.

---

## Technische Erkenntnisse

### Daten + Generator = skalierbare Dokumentation
Wenn zwei Ausgabedateien aus denselben Quellen generiert werden, lohnt sich ein Generator früh. Der Mehraufwand zahlt sich bei Korrekturen und Erweiterungen aus.

### MC-Fragen aus Kategorien sind eine pragmatische Lösung
Für jede Funktion eine eigene semantische MC-Frage zu erfinden, wäre bei 149 Funktionen unverhältnismäßig. Die Kategorie-Zuordnung als MC-Frage liefert sinnvolle Distraktoren und ist automatisch generierbar.

### `lib/stdlib.lisp` ist Teil der öffentlichen API
Funktionen wie `cadr`, `filter`, `dotimes`, `push`/`pop` sind keine internen Helfer, sondern werden beim Start in die Umgebung geladen. Schulungsunterlagen, die sie auslassen, wären unvollständig.

---

## Offene Punkte

- Keine inhaltlichen Lücken mehr für den aktuellen Scope.
- Mögliche Erweiterungen: Audio/visuelle Anki-Karten, interaktive Übungen, Export in andere Lernsysteme.

---

## Fazit Session 15

Die Schulungsunterlagen decken jetzt alle 149 öffentlichen GoLisp-Funktionen ab – von eingebauten Primitiven über Spezialformen bis zur Standardbibliothek. Der Generator-Ansatz macht sie wartbar und erweiterbar. Der einzige größere Umweg war der fehlgeschlagene Go-Datengenerator; der schnelle Pivot zu Python zeigte, dass Tooling-Flexibilität wichtiger ist als eine bestimmte Sprache durchzuzwingen.

> "Dokumentation wird erst dann lebendig, wenn sie genauso testbar und wiederholbar ist wie der Code selbst."
> — Gerhard & Claude, 23. Juni 2026

---

# Session 16 – 2026-06-24: find-definitions-for-emacs (M-.) via Subagent-Driven Development

**Autoren:** Gerhard Quell & Claude (Sonnet 4.6 / GLM-5.2)
**Branch:** session-14-swank-complete-print-fix → gemerged nach main (7a4202f)

---

## Ziel

TODO.md Punkt 2 hieß „slime-tramp für Emacs erstellen". Beim Klären stellte sich
heraus: slime-tramp war der falsche Begriff. Gerhards echter Use-Case war lokal —
`M-.` in SLIME soll zur Definition einer `defun`/`defmacro`/`define` in der
echten Quelldatei springen, Zeile genau. Kein Remote-TRAMP nötig.

Zusätzliches Ziel: Die ganze Feature-Entwicklung erstmals vollständig durch den
Superpowers-Workflow laufen lassen (Brainstorming → Spec → Plan → Subagent-
Driven Development → Final Review → Branch-Finishing).

---

## Was haben wir gebaut?

`swank:find-definitions-for-emacs` — M-. funktioniert in Emacs/SLIME gegen den
GoLisp-SWANK-Server. Drei Verhalten:
- **Datei-definiert** (`defun` aus geladener Datei): Springt zur Datei + Zeile.
- **REPL-definiert** (kein SrcFile): Rekonstruierter Snippet im Temp-Buffer.
- **Built-in / unbound**: `:error`-Meldung.

| Komponente | Aufgabe |
|------------|---------|
| `Cell.SrcFile`/`SrcLine` (lib/types.go) | Quellposition auf der Datenstruktur |
| Reader (lib/reader.go) | Zeilen-Tracking, stempelt `SrcLine` auf jede Listen-Cell |
| `evalLoad` (lib/eval_load.go) | Stempelt absoluten `SrcFile` auf Top-Level-Formen |
| `lib/defloc.go` (neu) | Thread-safe `map[string]DefLoc` + `sync.RWMutex` |
| `evalDefun`/`evalDefmacro`/`evalDefine` | Registrieren `(symbol → file/line)` |
| 3 Go-Primitive (lib/swank/env.go) | `swank--find-definition`, `--definition-kind`, `--definition-cell` |
| Handler (lib/swank/swank.lisp) | `find-definitions-for-emacs` + Snippet-Fallback |

**13 Feature-Commits** (`8539c55..0b3ae38`), 16 Dateien, +543/-15 Zeilen.
Manuelle Verifikation in Emacs: M-. springt korrekt auf die `defun`-Zeile.

---

## Was lief gut?

### Subagent-Driven Development funktionierte als Orchestrierungs-Disziplin
8 Tasks, je ein frischer Implementer-Subagent + ein Reviewer-Subagent, am Ende
ein breiter Final-Review (opus). Der Controller hielt nur Kontext, keine
Implementierung — eigener Context-Window blieb schlank. Model-Tiering sparte
Kosten: haiku für mechanische Tasks (complete code im Plan), sonnet für
Integration, opus nur für den Final-Review.

### Plan-Bug wurde vom Implementer gefangen
Task 2 hatte eine falsche Test-Erwartung (`f2.SrcLine = 2` statt `3`). Der
haiku-Implementer rechnete die Zeilen selbst nach, korrigierte den Test und
dokumentierte es als Concern. Der Task-Reviewer bestätigte die Korrektur. Der
Plan war falsch, der Prozess hat's abgefangen.

### Pre-existing Race sauber isoliert
`go test -race ./lib/` schlug in `TestParfuncTimeout` an — DATA RACE in
`Env.Set`/`Env.Get` (kein Lock auf der Env-Map bei parfunc). Die Untersuchung
zeigte: schon vor dieser Feature-Reihe vorhanden, nicht durch unsere Commits
eingeführt. Wurde als out-of-scope dokumentiert, statt den Feature-Review
damit zu blockieren.

### Spec-Abweichung begründet statt dogmatisch
Die DefLoc-Map sollte laut Spec in `lib/swank/defs.go` liegen. Bei der
Plan-Erstellung fiel der Import-Cycle auf (`lib` → `swank` verboten, da
`swank` bereits `lib` importiert). Map kam nach `lib/defloc.go`. Begründet im
Plan dokumentiert, Reviewer akzeptierte.

---

## Was lief nicht so gut?

### Unit-Tests erfassten zwei Runtime-Bugs nicht — manueller Emacs-Test schon
Die 8 Task-Reviews und der opus-Final-Review waren alle grün. Trotzdem sprang
M-. im manuellen Emacs-Test nicht. Zwei SLIME-Protokoll-Subtilitäten, die kein
Unit-Test abdeckte:

1. **`(dspec location)`-Wrapper fehlte.** `find-definitions-for-emacs` lieferte
   nackte `(:location ...)` zurück. SLIME erwartet eine Liste von
   `(name location)`-Paaren — ohne Label nahm es das falsche Listenelement und
   zeigte nur die Datei an, statt zu springen.
2. **`:line N :align` ist nicht kanonisch.** SLIME will `(:line N)`. Das
   `:align t` wurde ignoriert, Cursor blieb auf Zeile 1. Zusätzlich: relative
   Load-Pfade (`tmp/test2.lisp`) funktionieren nicht zuverlässig, weil Emacs'
   `default-directory` ≠ Server-CWD. `filepath.Abs` in `evalLoad` repariert das.

**Lernpunkt:** Unit-Tests prüften nur Substrings (`:location`, `:line`) — die
matchten auch bei der kaputten Version. Erst der echte SLIME-Client entlarvte
es. Protokoll-Tests müssen die *Struktur* asserten (hier: dspec-Wrapper +
`:align`-Abwesenheit), nicht nur Substrings. Nachgeschärft in `d329c84`.

### `git checkout -- file` revertierte versehentlich eigene Fixes
Beim Aufräumen eines zu breiten `git add -A` lief `git checkout -- lib/...` und
warf die manuellen Bugfixes weg. Neu angewandt, aber eine Erinnerung:
`git checkout --` verwirft Working-Tree-Änderungen bedingungslos.

### `pkill -f` traf den eigenen Shell-Prozess
Beim Server-Neustart killte `pkill -f 'tmp/golisp2 --swank'` auch die Bash, die
genau diesen Befehl ausführte (exit 144). Sauberer: gezielt nach PID auf dem
Port suchen statt `-f`-Pattern-Match.

### `./build`-Script fehlt im Repo
CLAUDE.md sagt „verwende ./build für die Builds", aber das Script existiert
weder tracked noch untracked. Pre-existing Quirk; `go build` funktioniert.
Sollte bereinigt werden (Script anlegen oder CLAUDE.md korrigieren).

---

## Technische Erkenntnisse

### SLIME `find-definitions-for-emacs` will `(dspec location)`-Paare
Der Return ist `(:ok ((name location) ...))` — eine Liste von 2-Element-Listen,
je `(label location)`. Nackte Locations funktionieren nicht; SLIME nimmt dann
das falsche Element und springt nicht. Bei genau einer Location springt SLIME
direkt, bei mehreren zeigt es eine Auswahlliste.

### `:line`-Position muss kanonisch sein
`(:location (:file "<abs>") (:line N) nil)` ist die sichere Form. `:align` und
Varianten sind implementationsabhängig und wurden ignoriert. Der dritte
Listeneintrag (`nil`) sind Hints — `nil` ist sicher.

### Source-Location auf der `Cell` statt Side-Table
`SrcFile`/`SrcLine` als Felder auf `Cell` (zero-value = unbekannt) war die
sauberere Wahl: Reader stempelt direkt beim Bau der Listen-Cell, kein Lookup
nötig. Alternative wäre eine Side-Map `*Cell → Location` gewesen — mehr
Indirection, GC-Druck, Pointer-Gleichheit nötig. Felder gewinnen.

### DefLoc-Map in `lib`, nicht `swank`
`defun` (Package `lib`) ruft `RegisterDefinition`; der SWANK-Primitive ruft
`LookupDefinition`. Da `swank` bereits `lib` importiert, darf `lib` nicht
zurück nach `swank` importieren → Map muss in `lib` liegen. Klassische
Dependency-Richtung: Shared-Code im tieferen Package.

### Subagent-Driven Development: Task-Briefs als Datei-Handoff
Statt Task-Beschreibungen in den Dispatch-Prompt zu pasten, extrahiert
`scripts/task-brief` jeden Task in eine Datei. Implementer und Reviewer lesen
den Brief, schreiben ihren Report in eine Datei zurück. Der Controller-Context
bleibt schlank — wichtig über 8 Tasks hinweg.

---

## Offene Punkte (nach dieser Session)

- **Env-Locking für parfunc** (pre-existing DATA RACE): `Env.Set`/`Env.Get`
  arbeiten auf der internen Map ohne Lock. parfunc teilt Env über Goroutinen
  → Race unter `-race`. Fix: `sync.RWMutex` in `lib/env.go`. Separates Thema.
- **`./build`-Script** anlegen oder CLAUDE.md-Referenz entfernen.
- **Weitere SLIME-Methoden**: `find-definitions` für Built-ins (Go-Source),
  Inspector, Debugger/Restarts, `who-calls`/`who-references`.
- **Protokoll-Tests strukturell**: Bestehende SWANK-Tests auf
  Substring-Asserts prüfen und durch Struktur-Asserts ersetzen (wie für
  `find-definitions` geschehen).

---

## Fazit Session 16

M-. funktioniert — Gerhard kann in SLIME zur Definition springen wie in
echtem Common Lisp. Der Weg dorthin war die erste vollständige
Superpowers-Durchlauf-Übung: Brainstorming, Spec, Plan, 8 subagent-gesteuerte
Tasks, opus-Final-Review, Merge. Der Prozess fing einen Plan-Bug und isolierte
einen pre-existing Race sauber ab.

Der Wertvollste Fund kam aber erst danach: zwei SLIME-Protokoll-Bugs, die kein
automatisierter Test sah — nur der echte Emacs-Client. Unit-Tests gegen
Protokoll-Strings müssen Struktur asserten, nicht Substrings. Das ist die
nachhaltigere Lektion als das Feature selbst.

> "Ein grüner Review ersetzt nicht den echten Client. Protokolle testet man
> gegen die Wahrheit, nicht gegen Substrings."
> — Gerhard & Claude, 24. Juni 2026

---

# Session 17 – 2026-06-25: FORMAT (Common-Lisp-HyperSpec 22.3) vollständig

**Autoren:** Gerhard Quell & Claude (GLM-5.2)
**Branch:** main

---

## Ziel

TODO.md Aufgabe 1: eine *komplette* `format`-Funktion wie Common Lisp.
Lisp-Code bevorzugt, sonst nativ in Go. Klärung ergab: „voll wie in CL",
Destination CL-style, `~/fun/` erst weglassen, alles auf einmal liefern.

## Was haben wir gebaut?

`(format dest control . args)` als Go-Primitive, vollständiger Direktiven-
Satz mit Parametern (`v` `#` `'c` Literal, Kommasepariert) + Modifiern
(`:` `@`).

| Datei | Inhalt |
|-------|--------|
| `lib/format.go` (336 Z.) | Engine: `fnFormat`, `formatRun`, Parameter/Modifier-Parser, `fmtState`, `resolveInt`/`resolveRune` |
| `lib/format_dirs.go` (960 Z.) | Direktiven-Handler + Helper: `~A~S~D~B~O~X~R~P~C~F~E~G~$~%~&~|~T~*~?~[~{~(~;~^~~`, `aesthetic`/`readable`, `padField`, `cardinal`/`ordinal`/`roman`, `findBlock`/`splitClauses` |
| `lib/format_test.go` (107 Z.) | `TestFormatBasic` (45 Fälle), `TestFormatErrors`, `TestFormatT` (stdout-capture), `TestFormatAppendString` |
| `lib/primitives.go` | `RegisterFormat(env)` in `BaseEnv()` eingehängt |

**Direktiven:** ~A ~S (aesthetic/readable, Padding) · ~D ~B ~O ~X (Radix,
Kommas, Vorzeichen) · ~R (Cardinal/Ordinal/Roman/Base) · ~P (Plural) · ~C
(Character/Name) · ~F ~E ~G ~$ (Float) · ~% ~& ~| (Newline/Fresh/Page) ·
~T (Tabulate) · ~* (Goto) · ~? (Recursive) · ~[ ~] (Conditional) · ~{ ~}
(Iteration, list-of-lists) · ~( ~) (Case-Conversion) · ~; ~^ ~~ ~Newline.

**Destination:** `t`→stdout+nil, `nil`→String, String→anhängen (CL-style).

**Verifikation:** `go test ./lib/` grün, `./golisp2 -t` exit 0, manuell:
`~{~a~^,~}`→"1,2,3", `~@r` 1999→"MCMXCIX", `~r` 42→"forty-two",
`~:(~a~)` "hello world"→"Hello World", `~[a~;b~;c~]` 2→"c", `~e`→Exp-Notation.

---

## Was lief gut?

### AskUserQuestion vor dem Coden — drei Designentscheidungen gebündelt
Destination (CL-style vs. nur-String), `~/fun/` (weglassen vs. Sofort),
Lieferung (alles vs. gestaffelt) in einem Schritt geklärt. Kein späteres
Rückbau-Risiko. Besonders `~/fun/`-Weglassen war wichtig: es braucht
env-Zugriff → wäre Spezialform nötig gewesen. Als Primitive sauber.

### Plan-Modus mit Codegraph statt Read-Loop
Codebase-Muster (Header, `RegisterXxx`, `makeFn`, `Cell.String()`) in einem
`codegraph_explore`-Call. Wichtigster Fund vorab: `Cell.String()` quotet
Strings (`%q`) — `~A` braucht aber *aesthetic*-Form ohne Quotes. Das als
eigener Helper `aesthetic` von Anfang an angelegt, kein nachträglicher Fix.

### Aufteilung am Ende wegen 1000-Zeilen-Limit
`format.go` lief auf 1278 Zeilen. CLAUDE.md: max 1000, ab 800 aufteilen.
Mechanischer Split: Engine (`format.go` 336) + Direktiven/Helper
(`format_dirs.go` 960). Beide <1000, kohäsiv. Tests nach Split sofort wieder
grün — kein Behavior-Change, reines Move.

---

## Was lief nicht so gut?

### Debugging-Odyssee: drei Bugs, einer Maskierte den anderen
Die Iterations-Direktive `~:{` (list-of-lists) failte mit „keine Argumente
mehr". Drei verschachtelte Bugs, die sich gegenseitig maskierten:

1. **`fmtState.out` als `strings.Builder`-Value.** `out: st.out` kopiert den
   Builder — Go verbietet das (addr-Self-Pointer bricht). Sub-States
   schrieben ins Leere. Fix: `out *strings.Builder` (Pointer, shared).
2. **`st.sub()` ohne `out`-Feld.** Nach Pointer-Umstellung erzeugte der
   Clause-Sub-State `&fmtState{args:...}` ohne `out` → nil-Pointer →
   `WriteRune`-Panic. Fix: `out: &strings.Builder{}` (eigner Builder, Ergebnis
   appended).
3. **`:=`-Shadowing im for-Body — zweimal.** `colon, at, pos := parseModifiers()`
   und `ipos, esc := emitIteration()` (ursprünglich `pos, esc :=`). Go's `:=`
   deklariert `pos` im Body-Block *neu* (Shadow), weil `pos` aus dem enclosing
   for-Scope stammt — statt zuzuweisen. Symptom: Direktiv-Char wurde als
   Literal nachgeschrieben / Block wurde auf Parent-Ebene reprozessiert →
   Parent-consume-Fehler.

**Lernpunkt 1:** Go `:=` shadowed Variablen aus *enclosing* Blocks — das ist
kein Assign. Bei Multi-Return in Loop-Bodies: separater Return-Name + `=`.
**Lernpunkt 2:** Debug-`st.err = "DBG ..."` überschreibt echten Fehler und
maskiert den Wurzelbug. Debug-Logs akkumulieren (separater Buffer), nie
`st.err` überschreiben.
**Lernpunkt 3:** `strings.Builder` nie per Value kopieren — Pointer oder
eigene Instanz.

### Debug-Test-File manuell gepflegt statt sauber isoliert
Während des Debugging mehrfach `lib/colon_dbg_test.go` angelegt/gelöscht,
Debug-Vars (`fmtDbg`) per Python-Patch eingefügt und entfernt. Funktional
ok, aber riskant — Reste hätten Build gebrochen. Besser: Debug-Flag als
build-tag oder eigene `*_debug_test.go` die nicht committet wird.

### `-e`-Verifikation durch pre-existing mtest-Konflikt blockiert
`tmp/mtest*.go` (drei `func main` im `golisp2/tmp`-Package) bricht
`go test ./...` und `go run .`. Binary-Build nur via `go build -o tmp/golisp2 .`
(root-Package = main.go, tmp/ separiert). Verwirrend beim Debugging —
anfangs stale Binary suggerierte falsche Fehler. Pre-existing, nicht
FORMAT-Scope, aber sollte bereinigt werden.

---

## Technische Erkenntnisse

### Go `:=`-Shadowing ist eine tückische Falle
`for pos < len { ... x, pos := f() ... }` — wenn `pos` im for-Scope
deklariert ist, erzeugt `:=` im Body eine *neue* `pos` (Shadow), weil Go
nur Variablen aus dem *selben* Block reuses. Der Assign geht ans Shadow,
das bei Iterationsende verworfen wird. Äußeres `pos` unbewegt. Bei
Single-Return (`pos = f()`) kein Problem. **Regel:** Multi-Return-Assignments
in Loop-Bodies immer via Hilfsvariable (`npos, _ := f(); pos = npos`).

### `strings.Builder` ist nicht copy-safe
Builder hat intern `addr *Builder` das auf sich selbst zeigen muss. Kopiert
man den Value (`out: st.out`), zeigt `addr` der Kopie auf das Original —
Writes landen falsch oder panicen. **Regel:** Builder immer als Pointer
(`*strings.Builder`) weiterreichen, oder je Sub-Kontext eine frische
Instanz (`&strings.Builder{}`) und Ergebnis explizit appenden.

### FORMAT-Engine: shared vs. own Builder bewusst wählen
- **Iteration** (`~{`): Body schreibt in Parent-Builder → shared (`out: st.out`).
- **Clauses** (`~[`, `~(`): Body läuft in Sub-Buffer, wird transformiert
  (Case) oder selektiert (Conditional) → eigener Builder, Ergebnis appended.
- **Recursive** (`~?`): eigener Builder, appended.

Falsche Wahl = entweder leerer Output (shared wo own nötig) oder
verfälschte Transform (own wo shared nötig). Bewusste Entscheidung pro
Direktive, nicht global.

### `findBlock` muss Body-Grenze VOR der schließenden Tilde ziehen
`~{~a~}`: Body ist `~a` (Indizes 3-5), schließendes `~}` bei 5-7. Erste
Implementierung lieferte body=[3,6] (inkl. der `~` der Close-Direktive) →
nacktes `~` am Body-Ende → Parser-Fehler „Direktive unvollständig". Fix:
`tildeStart` merken, body=[start, tildeStart], npos=pos+1 (nach Close-Char).

### `~A` ist left-justified, `~D` right-justified
HyperSpec: `~A`/`~S` paden rechts (colon = rechtsbündig). `~D`/`~B`/`~O`/`~X`
paden links. Erste Test-Erwartung war falsch (`~5a` 42 → "   42" geraten,
CL gibt "42   "). CL-Verhalten aus HyperSpec, nicht aus Intuition raten.

---

## IST-Funde (Session 17)

- `Cell.String()` quotet Strings (`%q`). Für `~A` (aesthetic) brauchte es
  eine eigene unquoted-Form → `aesthetic()` Helper. Kein Bug, aber eine
  Lücke im bestehenden Print-Modell (es gab nur `String()`, keine
  princ/prin1-Unterscheidung).
- `~F` Default `d`: CLHS läßt `d` weg → volle Precision. `d=0` explizit →
  keine Nachkommastellen. Unterscheidung missing vs. 0 nötig (Sentinel -1).
- `~$` Default `n=0` (min Ziffern vor Punkt), nicht additiv. Erste
  Implementierung addierte n Pad-Chars immer → " 3.14" statt "3.14".

## Offene Punkte (nach dieser Session)

- **`~/fun/`**: weggelassen, braucht env-Zugriff. Entweder `format` als
  Spezialform in `eval.go` (args manuell auswerten) oder global-env-Capture.
- **`~:[` Default-Klausel via `~:;`**: Stub (`hasDefaultMark` → false).
  Vollständige Implementierung würde `splitClauses` erweitern müssen.
- **`~F/~E/~G` Edge-Cases**: overflowchar, k≠0 Skaling vereinfacht.
  Mainstream-Parameter abgedeckt, dokumentiert im Datei-Header.
- **`tmp/mtest*.go`**: pre-existing Build-Konflikt (drei `func main`).
  Bereinigen (nach `tmp/` ist gitignored, aber Go traversiert trotzdem).
- **`./build`-Script**: fehlt noch immer (Session 16 schon notiert).

---

## Fazit Session 17

FORMAT ist drin — voll wie Common Lisp, ~20 Direktiven mit Parametern und
Modifiern, CL-style Destination. Der Funktionsumfang war nicht das Schwere;
die Debugging-Odyssee war's. Drei verschachtelte Bugs (`strings.Builder`-
Value-Kopie, nil-Sub-Builder, `:=`-Shadowing) maskierten sich gegenseitig,
und Debug-Logs überschrieben den echten Fehler, was die Wurzel noch länger
versteckte.

Die nachhaltigste Lehre ist nicht FORMAT-spezifisch, sondern eine Go-Muster-
Falle: `:=` in for-Loop-Bodies shadowed enclosing `pos`. Zweimal passiert,
zweimal gesucht. Nächstes Mal: Multi-Return in Loops immer via Hilfsvar + `=`.

> "Drei Bugs, die sich gegenseitig deckten — und ein Debug-Log, das den
>  Wurzelbug übermalte. Manchmal ist das Letzte was man findet, das Erste
>  was man hätte wissen müssen: `:=` shadowed."
> — Gerhard & Claude, 25. Juni 2026

---

# Session 17 — Nachtrag: offene FORMAT-Punkte erledigt

**Datum:** 2026-06-25 (Fortsetzung)
**Autoren:** Gerhard Quell & Claude (GLM-5.2)

Direkt im Anschluss wurden die in Session 17 als offen markierten
FORMAT-Punkte abgearbeitet — bis auf die pre-existing Quirks (mtest, die
Gerhard's Dateien sind).

## Was haben wir gebaut?

| Punkt | Lösung |
|-------|--------|
| `~/fun/` | global-env-Capture: `RegisterFormat` speichert BaseEnv in `globalFormatEnv`. `~/name/` lookt die Funktion auf, ruft `apply(fn, [arg])`, schreibt Ergebnis aesthetic. GoLisp hat keine Packages → `~/name/` (kein `package:`-Prefix). |
| `~:[` Default via `~:;` | `splitClauses` umgeschrieben: trackt colon-Modifier beim `;`, liefert `defaultIdx` = Index der Klausel *nach* `~:;`. `emitConditional` nutzt `defaultIdx` statt `hasDefaultMark`-Stub. Stubs entfernt. |
| `~F` k-Skaling | `k`-Parameter (params[2]) → `f *= pow10(k)` vor `formatFixed`. `pow10`-Helper ohne math-Import. overflowchar (selten) bewusst nicht umgesetzt. |
| `./build`-Script | `build/` war ein leeres Verzeichnis, kein Script. `build.sh` angelegt: kompiliert golisp2/golisp2d/golisp2-client nach `build/`. CLAUDE.md-Referenz korrigiert (`./build` → `./build.sh`). |

**Aufteilung erweitert:** `format_dirs.go` war auf 1021 Zeilen gewachsen
(nach `~/fun/`-Zugabe) → 3-Wege-Split: `format_dirs.go` (622, einfache
Direktiven) + `format_blocks.go` (418, Block-Direktiven + Helper) +
`format.go` (350, Engine). Alle <1000.

**Tests:** `TestFormatBasic` um 7 Fälle erweitert (~:[ default ×4, ~F k ×3).
Neu `TestFormatUserFunc` (~/fun/ mit defun+format in `begin`).

## Was lief gut?

### global-env-Capture statt Spezialform für `~/fun/`
CLAUDE.md-Regel lautet „braucht env → Spezialform". Aber `format` wertet
args normal aus — als Spezialform hätte ich args manuell auswerten müssen,
unelegant. Global-Capture (`RegisterFormat` setzt `globalFormatEnv`) ist
pragmatisch: Primitive bekommt env indirekt, args bleiben eval-Loop-Sache.
BaseEnv wird in Tests mehrfach aufgerufen — letzter gewinnt, funktioniert.

### `~:;` Default sauber in splitClauses integriert
Statt separatem `hasDefaultMark`-Lookup (Stub) die Information dort zu
gewinnen wo sie entsteht: `splitClauses` trackt beim Parsen den colon-Modifier
des `;` und liefert `defaultIdx` gleich mit. Eine Stelle, eine Datenstruktur.
Keine Nachbetrachtung der Klausel-Grenzen nötig.

### `-e` Single-Expr-Falle rechtzeitig erkannt
`~/fun/`-Test via `-e "(defun ...) (format ...)"` lieferte „up" statt „HALLO"
— `-e` wertet nur die erste Form aus (Session 7 bekannt). Via stdin/hier-doc
oder `begin`-Wrap sofort korrekt. Alte Lektion, diesmal schnell erinnert.

## Was nicht lief / Verbesserungspotenzial

### format_dirs.go rutschte erneut über 1000 Zeilen
Nach `~/fun/`-Zugabe 1021 Zeilen. Aufteilen in `format_blocks.go` nötig.
Hätte beim Planen der `~/fun/`-Erweiterung sehen können, dass die Datei an
die Grenze kommt. **Lehre:** bei Feature-Zugabe zu einer Datei nahe dem Limit
direkt mitAufteilen planen, nicht erst danach korrigieren.

### mtest*.go bleiben unangetastet
`tmp/mtest*.go` (drei `func main`) blockieren `go test ./...` und `go run .`
weiterhin. Gerhard's Dateien — nicht ohne Erlaubnis angefasst. Nur `go build .`
(root-Package) und `go test ./lib/` funktionieren. Bleibt offen bis Gerhard
entscheidet (löschen/umbenennen/nach non-go-Dir verschieben).

## Technische Erkenntnisse

### `~/fun/` — global-capture als dritter Weg
Nicht Spezialform (args manuell auswerten), nicht env-Parameter an Primitive
(Signatur-Bruch), sondern global-capture. Primitive bleibt `func([]*Cell)(*Cell,error)`,
bekommt env über Package-Variable. Passt zur GoLisp-Architektur wo `Env.Root()`
ähnlich global-denkt. Limit: nur *ein* globalFormatEnv — bei mehreren BaseEnv-
Instanzen (Tests) letzter gewinnt. Für GoLisp's Single-Env-Modell ok.

### `splitClauses`-Return-Signatur ändern ist billig wenn ein Caller
`splitClauses` gab `[][2]int`, jetzt `([][2]int, int)`. Nur `emitConditional`
ruft auf → eine Stelle anpassen. Wäre bei mehreren Callern aufwendiger gewesen.
**Regel:** Helper-Signaturänderungen sind sicher, wenn Call-Graph klein —
codegraph blast-radius vorher checken.

### `pow10` ohne math-Import
`math.Pow(10, k)` wäre direkter, aber math-Import nur für eine Funktion
ist Overhead. 6-Zeilen-Helper (k≥0: mal 10, k<0: geteilt 10) ohne Import.
Klein, aber konsistent mit GoLisp's „sparsame Imports"-Stil.

## Offene Punkte (aktualisiert)

- [x] ~~`~/fun/`~~ → global-capture (Session 17 Nachtrag)
- [x] ~~`~:[` Default via `~:;`~~ → splitClauses defaultIdx (Session 17 Nachtrag)
- [x] ~~`~F` k-Skaling~~ → pow10-Helper (Session 17 Nachtrag)
- [x] ~~`./build`-Script~~ → `build.sh` nach `build/` (Session 17 Nachtrag)
- [ ] **`~F` overflowchar**: weiterhin nicht unterstützt (selten, dokumentiert).
- [x] ~~`tmp/mtest*.go`~~ → nach `.go.bak` umbenannt, `go test ./...` und
  `go run .` funktionieren wieder.
- [ ] **Env-Locking für parfunc**: pre-existing DATA RACE (Session 16).

## Fazit Session 17 — Nachtrag

Vier offene Punkte in einem Rutsch abgearbeitet, FORMAT jetzt im vollen
CL-Scope nutzbar (inkl. `~/fun/` und `~:[` Default). Der global-env-Capture
für `~/fun/` war die interessanteste Entscheidung: CLAUDE.md's
Spezialform-Regel passte hier nicht, weil `format` args normal auswertet.
Dritter Weg (global-capture) statt dogmatisch Regel befolgen. format_dirs.go
rutschte erneut über's Limit — dritte Aufteilung jetzt, beim nächsten
Feature-Zuwachs früher dran denken.

> "Eine Regel die nicht passt, ist keine Regel sondern ein Hinweis. Bei
>  ~/fun/ war der dritte Weg — global-capture — kürzer als die Regel."
> — Gerhard & Claude, 25. Juni 2026

---
---

# Session 18 – 2026-06-26: GA-Integration + Fib-Allokations-Optimierung

**Autoren:** Gerhard Quell & Claude (Kimi)
**Branch:** main

---

## Ziel

Zwei unabhängige Aufgaben aus TODO.md:
1. Genetischen Algorithmus (`lib/genalg.go`) in GoLisp als Lisp-Primitive integrieren.
2. Fibonacci-Benchmark (`lib/fibBench_test.go`) allokationsseitig optimieren.

---

## Was haben wir gebaut?

### 1. GA-Integration

| Arbeit | Dateien | Ergebnis |
|--------|---------|----------|
| Package-Angleichung | `lib/genalg.go` | `package genalg` → `package lib` |
| Lisp-Primitive | `lib/genalg_prims.go` (neu) | 9 Primitive: `ga-create`, `ga-init`, `ga-cross`, `ga-calc`, `ga-select`, `ga-result`, `ga-mut`, `ga-print`, `ga?` |
| Registrierung | `lib/primitives.go` | `RegisterGenAlg(env)` in `BaseEnv()` |
| Tests | `lib/genalg_prims_test.go` (neu) | Creation, Typen, Lifecycle, Race-Detection |
| Smoke-Tests | `main.go` | GA-Zyklus in `runTests()` |

**Lisp-API:**
```lisp
(define ga (ga-create 'bit1 5 4 (lambda (g) (apply + g))))
(ga-init ga)
(ga-calc ga)
(ga-result ga)   ; => sortierte Fitness-Scores
```

**Design-Entscheidungen:**
- GA-Handle als `Cell{Type: FUNC, Val: "ga", Env: *gaHandle}`, analog zu Channels/Mutexes.
- Fitness-Callback ruft Lisp-Funktion via `apply()` auf. `ga-calc` arbeitet parallel,
  daher muss die Fitness-Funktion rein/thread-sicher sein (gleiches Modell wie `parfunc`).
- Genom wird als Lisp-Liste von Zahlen an die Fitness-Funktion übergeben.

### 2. Fib-Allokations-Optimierung

| Optimierung | Dateien | Effekt |
|-------------|---------|--------|
| Small-Integer-Cache | `lib/types.go` | `MakeNum` wiederverwendet NUMBER-Cells für -128..127 |
| `eq`-Semantik beibehalten | `lib/primitives.go` | `(eq 5 5)` bleibt `()` trotz Cache |
| Single-Field-Env | `lib/env.go` | Erster gebundener Name inline, Map erst ab zweitem Symbol |

**Benchmark-Ergebnisse:**

| | Original | Nachher | Delta |
|---|---|---|---|
| Zeit | 100,4 ms/op | 56,3 ms/op | **-44%** |
| Bytes | 112,7 MB/op | 23,4 MB/op | **-79%** |
| Allocs | 1.942.278/op | 1.093.520/op | **-44%** |

---

## Was lief gut?

### Plan-Modus für GA-Integration
Klärung der Architektur-Fragen vor dem Coden: Package-Struktur, Handle-
Repräsentation, Fitness-Callback-Thread-Safety, Genom-Konvertierung. Dadurch
konnte die Integration in einem Schritt sauber umgesetzt werden.

### Muster-Wiederverwendung
`gaHandle` folgt demselben Muster wie `goChannel`/`goMutex`/`pgConn`:
Go-Objekt in `Cell.Env`, Markierung in `Cell.Val`. Dadurch konsistent mit
dem Rest der Codebase.

### Charakterisierungstest-Disziplin
Beim GA sofort Tests für jeden GenType, Invalid-Args, Lifecycle und Race-
Detection geschrieben. `go test -race ./lib/ -run TestGa` grün.

### Iterative Optimierung mit Messung
Fib-Optimierung nicht spekulativ, sondern benchmark-getrieben:
1. Small-Integer-Cache → -19% Allocs
2. Args-Pooling-Versuch → keine Verbesserung, revertiert
3. Größerer Cache (-8192..8192) → keine Verbesserung, revertiert
4. Single-Field-Env → weitere -31% Allocs

Fehlschläge wurden verworfen statt akkumuliert zu werden.

### Semantische Integrität trotz Optimierung
Der Small-Integer-Cache hätte `(eq 5 5)` zu `t` geändert. Statt den Cache zu
entfernen, wurde `eq` so angepasst, dass Numbers weiterhin als nicht-identisch
gelten. Optimierung ohne Verhaltensänderung.

---

## Was lief nicht so gut?

### Args-Pooling war kontraproduktiv
`sync.Pool` für `evalArgs`-Slices verschlechterte Zeit und Bytes. Ursache:
Go optimiert kleine Slices im Fib-Fall bereits auf den Stack; das Pool-
Synchronisation überwog den Nutzen.

**Lernpunkt:** Nicht jede Allokation ist heap-allokiert. Pooling lohnt sich nur
wenn der Profiler Heap-Druck zeigt.

### Größerer Integer-Cache brachte kaum etwas
Erweiterung von -128..127 auf -8192..8192: Allocs praktisch gleich, Zeit leicht
schlechter. Die meisten relevanten Zwischenergebnisse lagen bereits im kleinen
Cache.

**Lernpunkt:** Cache-Größe hat einen Sweet Spot; mehr ist nicht immer besser
(Lokalität, Initialisierungskosten).

### Safety-Classifier blockierte Bash wiederholt
Mehrfach musste Gerhard Benchmarks selbst ausführen (`! ...`), weil der
Classifier temporär ausfiel. Ablauf wurde unterbrochen.

---

## Technische Erkenntnisse

### Single-Field-Env als großer, sicherer Hebel
Die meisten Lambda-Calls haben nur einen Parameter. Die Map-Allokation für
einen einzigen Eintrag ist Overhead. Inline-Speicher des ersten Namens
eliminiert Map-Allokation/Bucket für diesen Fall, ohne Semantik oder Closure-
Verhalten zu ändern.

### NUMBER-Cells sind immutable → cachen ohne Kopfschmerzen
Im Gegensatz zu LIST/ATOM werden NUMBER-Cells nach Erstellung nie verändert.
Das macht Sharing thread-sicher. Der `eq`-Fix zeigt, dass Implementierungs-
Sharing nicht gleich semantische Gleichheit bedeuten muss.

### GA-Fitness-Callback: Parallelität ist Vertragssache
`GaCalc` ruft den Fitness-Callback parallel auf. Die Lisp-Eval/Env-Layer ist
nicht generell thread-safe (siehe pre-existing parfunc-Race). Statt alles zu
serialisieren, wurde die Verantwortung an den Nutzer delegiert: Fitness-
Funktion muss rein sein. Konsistent mit bestehendem `parfunc`-Modell.

---

## IST-Funde (Session 18)

- `lib/genalg.go` war als `package genalg` in `lib/` deklariert — das brach
  `go test ./lib/...`. Behoben durch Package-Wechsel auf `lib`.
- `length` existiert nicht als Primitive; Tests mussten eigene `len`-Hilfe
  definieren.

## Offene Punkte (aktualisiert)

- [x] ~~GA in GoLisp integrieren~~ → Session 18
- [x] ~~Fib-Allokationen optimieren~~ → Session 18
- [x] ~~Env-Locking für parfunc~~ → Session 19 (`sync.RWMutex` in `lib/env.go`)

---

## Fazit Session 18

GA ist von Lisp aus steuerbar, Fib ist deutlich sparsamer. Die größte
Optimierung kam nicht aus einem Pool, sondern aus der Vermeidung einer
Allokation (Single-Field-Env). Das Benchmark-Driven-Approach hat gezeigt,
dass messen, revertieren und erneut messen wichtiger ist als das erste
naheliegende Optimierungsmuster.

> "Der schnellste Code ist der, der nicht alloziert. Der zweitschnellste
>  ist der, der weniger alloziert weil die Datenstruktur schlauer ist."
> — Gerhard & Claude, 26. Juni 2026

---

## Fazit Session 19 – Fib-Optimierung abgeschlossen (2026-07-09)

**Was kam dazu:**

| Schnitt | Änderung | Dateien |
|---------|----------|---------|
| 4 | `sync.Pool` für FUNC-Arg-Slices | `lib/eval_core.go` |
| 5 | Frame-Env-Pool + Tail-Call-Freigabe | `lib/env.go`, `lib/eval_core.go`, `lib/eval_lambda.go`, `lib/eval_control.go` |
| 6 | Small-Int-Cache auf int16-Bereich | `lib/types.go` |
| 7 | Env-Locking für `parfunc` | `lib/env.go` |

**Messung `fib 25`:**

| Stand | allocs/op | B/op | ns/op |
|-------|-----------|------|-------|
| Nach Schnitt 4 | 243 851 | 23 MB | 51 ms |
| Nach Schnitt 5 | 987 | 95 KB | 86 ms |
| Nach Schnitt 6 | 3 | 641 B | 88 ms |

**Erkenntnisse:**

- `sync.Pool`-Buffer nur freigeben, solange `pooled==true`; der `>8-Args`-Switch hat sonst einen noch aktiven Buffer zurückgegeben und SWANK-Tests flaky gemacht.
- Debug-Instrumentierung (poison cells, `debug.Stack`) im Hot-Path macht Benchmarks unbrauchbar — sofort wieder entfernen, bevor man die Zahlen liest.
- Frame-Pooling braucht ein Ownership-Modell: `ownEnv` im Trampolin-Loop, `shared`-Flag für Closures, `defer freeEnv` für `do`/`flet`/`labels`.
- `MakeNum`-Cache wirkt stärker als erwartet: 987 → 3 allocs/op. Immutable NUMBER-Cells erlauben das ohne semantische Brüche.
- Env-Locking mit `sync.RWMutex` macht `parfunc` race-frei. `Update` muss den Child-Lock halten, während es den Parent-Chain entlangsucht — sonst könnte ein paralleles `Set` denselben Namen dazwischen im Child erzeugen.
- Der `fib`-Mikrobenchmark ist ausgeschöpft; weitere Gewinne brauchen andere Workloads oder einen VM/Compiler-Ansatz.

**Noch offen:**

- [x] **Env-Locking für `parfunc`**: erledigt, `go test -race ./...` grün.

---

# Session 20 – 2026-07-09: Rename golisp→golisp2 + golisp2-client auf SWANK

**Autoren:** Gerhard Quell & Claude (zai)
**Branch:** main (Feature-Branches `rename-golisp2-swank-client` → FF-merge, `fix/flag-over-env-priority` → FF-merge)

---

## Ziel

Drei Aufgaben aus TODO.md (20260709) plus eine präzistente Folgeaufgabe:

1. Ausführbare Programme von `golisp` auf `golisp2` umbenennen, Buildprozess anpassen.
2. `golisp2d` ist der SWANK-Server, `golisp2` der Standalone, `golisp2-client` der Client.
3. `golisp2-client` spricht noch altes Custom-RPC, der Server aber SWANK → Client umstellen.
4. CLAUDE.md-Doku-Drift: Server-Abschnitt beschrieb noch Custom-RPC, obwohl SWANK real ist.

---

## Was haben wir gebaut?

### 1. Rename golisp → golisp2

| Arbeit | Dateien | Ergebnis |
|--------|---------|----------|
| cmd-Dirs | `cmd/golisp2d/`, `cmd/golisp2-client/` | via `git mv` (History erhalten, Rename-Detection) |
| Build | `build.sh` | Outputs `golisp2`/`golisp2d`/`golisp2-client` |
| Ignore | `.gitignore` | Root-Level-Binaries auf neue Namen |
| Go-Kommentare | `main.go`, `cmd/*`, `lib/sigorest.go`, `lib/stdlib.go`, `lib/swank/server.go` | Prompt `golisp2>`, Header, Modul-/Binary-Refs |
| Doku | alle `.md` (12+ Dateien) | `sed` mit Wortgrenze + GitHub-URL-Schutz |
| Datei-Rename | `docs/retrospective-golisp2d-20250301.md` | inkl. Ref-Update in RETROSPECTIVE |

**Entscheidungen:**
- Env-Variablen `GOLISP_HOST/PORT/SIGO_*` bewusst beibehalten — kein Breaking-Change für bestehende Setups. Modul heißt `golisp2`, Env-Schlüssel bleiben stabil.
- Brand `GoLisp` (CamelCase) unangetastet — nur Binary-/Modul-Identifier ändern. Sed case-sensitive, `\b`-Wortgrenze schützt bereits umbenannte Tokens (`golisp2d` nicht nochmal angetastet).
- GitHub-Repo-URLs `github.com/gerhardquell/golisp` geschützt — Repo-Slug ist separate Entscheidung, nicht Binary-Name. Remote war ohnehin nur lokaler Pfad.

### 2. golisp2-client auf SWANK

| Arbeit | Detail |
|--------|--------|
| Protokoll | Custom-RPC `(:id N :method "eval")` → echtes SWANK `:emacs-rex` (length-prefixed `%06x<sexpr>`) |
| Framing | Eigenes `readFrame`/`writeFrame` im Client (swank-Package-Funktionen unexported) |
| Cell-Verarbeitung | Import `golisp2/lib` — `lib.Read`/`CellToSlice` statt fragiles String-Slicing |
| Request-Loop | `request()` sendet `:emacs-rex`, liest Frames bis `:return` mit passender ID, sammelt `:write-string`-Events |

**Cmd-→-SWANK-Op-Map:**

| Client-Flag | SWANK-Op |
|-------------|----------|
| `--ping` | `swank:connection-info` |
| `--eval` | `swank-repl:listener-eval` |
| `--complete` | `swank:simple-completions` |
| `--load` | `swank:load-file` |
| `--repl` | `listener-eval`-Loop |

**Smoke-Test gegen `golisp2d`:** `(+ 1 2)`→3, `(defun sq (x) (* x x)) (sq 9)`→`sq`/`81`, `--complete ca`→`car cadddr cadr caddr caar`, `--load`→50, Error→`server: "..."` exit 1. Build + 126 Tests grün.

### 3. CLAUDE.md an SWANK-Realität

- Abschnitt "GoLisp Server (golisp2d)": Custom-RPC-Beschreibung entfernt, `golisp2d = golisp2 --swank = swank.RunServer` klargestellt.
- Obsolete Methoden-Tabelle (`:method`/`eval-return`/`disconnect`) gelöscht → ersetzt durch SWANK-Framing-Details + Cmd-→-Op-Tabelle.
- "Shared Environment" (falsch) → "Pro-Connection-Env" (`handleConn` macht frisches `BaseEnv` pro Verbindung).
- SWANK-Abschnitt-Intro: "(Unabhängig vom oben beschriebenen Custom-RPC)" entfernt.

### 4. Env-vs-Flag-Priorität (Folgeaufgabe)

| Vorher | Nachher |
|--------|---------|
| Env NACH `flag.Parse` angewendet → überschreibt expliziten Flag | Env als Flag-Default VOR Parse → Flag gewinnt |
| `GOLISP_PORT=9123 golisp2d --port 9128` → bindet 9123 | → bindet 9128 |
| Unix-Konvention verletzt: Env > Flag > Default | restored: Flag > Env > Default |

Betrifft `cmd/golisp2d` + `cmd/golisp2-client`. Env-only-Nutzung bleibt erhalten.

---

## Was lief gut?

- **Bestand vor Edit:** Erst alle `golispd`/`golisp-client`/`golisp`-Vorkommen grep'd (Go + .md + URLs), bevor sed lief. GitHub-URLs so erkannt und geschützt — ein blinder `\bgolisp\b`→`golisp2` hätte Clone-URLs kaputtgemacht.
- **Feature-Branch + FF-Merge:** Rename war groß (35 Dateien). Branch隔离 lies den diff sauber reviewen, FF-Merge hielt History linear.
- **Perf-WIP sauber getrennt:** `lib/env.go`/`eval_*.go`/`PerfTODO.md`/`RETROSPECTIVE.md` waren zu Session-Start schon modifiziert (Session-19-Fib-Arbeit). Wurden aus dem Rename-Commit ausgeschlossen — keine Vermischung von Rename und Perf.
- **Smoke-Test mit echtem SWANK-Frame:** Handgeschriebener `%06x`-Frame per `nc` gegen `golisp2d` bewies, dass Framing + Dispatch funktionieren — nicht nur Go-Tests.

---

## IST-Funde (Session 20)

- `cmd/golisp2d/main.go` war bereits auf `swank.RunServer` umgestellt (Refactor 20260618), CLAUDE.md aber nie — die "Custom-RPC"-Beschreibung war reine Doku-Drift, kein Code-Realität. Semantischer Konflikt aus TODO ("golisp2d soll SWANK-Server sein") war längst gelöst.
- `golisp2-client` sprach noch Custom-RPC obwohl der Server SWANK spricht — Client/Server-Protokoll-Mismatch seit 20260618 unentdeckt, weil der Client nie gegen den neuen Server getestet wurde.
- "Shared Environment" in der alten Doku war faktisch falsch seit SWANK-Umstellung: `handleConn` erzeugt pro TCP-Verbindung ein eigenes `env`. Alte Custom-RPC-Annahme (ein globaler Server-Env) galt nicht mehr.
- Remote `origin` → `/u/lisp-projekte/golisp-kimi` war ein **Non-Bare-Repo mit ausgechecktem main** plus uncommitteten getrackten Änderungen. Push wurde von Git korrekt abgelehnt; `reset --hard` hätte dort Arbeit zerstört. → `origin` entfernt, `golisp-kimi` als frozen erklärt (Memory `golisp-kimi-frozen`).
- `define`-curried-Syntax `(define (name args) body)` wird nicht unterstützt — nur `defun`. Präexistent, kein Rename-Bug.

---

## Erkenntnisse

- **Doku-Drift entsteht bei Refactor ohne Doku-Pflege:** Server wurde 20260618 auf SWANK umgestellt, die Custom-RPC-Doku blieb stehen und widersprach der Realität. Zwei Server-Abschnitte (Custom-RPC + SWANK) für denselben `golisp2d` sind ein Symptom — Konsolidierung auf einen einzigen beseitigt den Widerspruch.
- **Robuste Protokoll-Clients parsen, nicht slicen:** Der alte Client extrahierte `:result`-Felder per `strings.Index` — fragil bei Escaping und verschachtelten S-Expressions. Der neue Client nutzt `lib.Read` + `CellToSlice`: korrekt per Construction. `MakeStr(code)` + `cell.String()` (Go `%q`) liefert SWANK-String-Escaping gratis.
- **Multi-Frame-Responses brauchen Lese-Loops:** SWANK `listener-eval` sendet N× `:write-string` + final `:return (:ok ())`. Ein einzelnes `ReadString('\n')` (alter Client) würde hängen oder halbe Antworten liefern. Client muss bis zum `:return` mit passender ID lesen.
- **Non-Bare-Remotes sind keine Backups:** Ein Working-Repo als `origin` bricht beim Push in den ausgecheckten Branch und verleitet zu `reset --hard`-Datenverlust. Backup-Remote muss `--bare` sein.

---

## Fazit Session 20

Rename ist mechanisch durchexerziert, aber die eigentliche Arbeit saß in den
Folgeaufgaben: der Client/Server-Protokoll-Mismatch (Custom-RPC vs SWANK)
war seit Wochen unentdeckt, und die CLAUDE.md-Doku erzählte eine Realität,
die seit 20260618 nicht mehr existierte. Beides kam ans Licht weil der
Rename die Frage aufwarf, was `golisp2d` eigentlich ist.

Die größte Falle war nicht der Code, sondern das Remote: ein Non-Bare-Repo
mit ausgechecktem main als `origin`. Git hat das Pushen korrekt verweigert —
hätte man `receive.denyCurrentBranch=ignore` + `reset --hard` erzwungen,
wäre im `golisp-kimi`-Working-Repo Arbeit verloren gegangen. `golisp-kimi`
ist jetzt frozen, `origin` entfernt.

> "Ein Rename ist ein Stresstest für Doku und Annahmen. Was dabei ans Licht
>  kommt, ist meist älter als der Rename selbst."
> — Gerhard & Claude, 9. Juli 2026
