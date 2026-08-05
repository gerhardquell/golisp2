# GoLisp2 – CLAUDE.md

Lisp-Interpreter in Go mit nativer KI-Anbindung (sigoREST).
Ziel: ein selbsterweiterndes System, das Goroutinen, Channels und KI-Calls
als eingebaute Lisp-Primitiven beherrscht.

**Autor:** Gerhard Quell – gquell@skequell.de · **CoAutor:** Claude
**Modul:** `golisp2` · **Sprache (Kommentare/Doku):** deutsch

> Diese Datei ist ein **Briefing**, kein Handbuch. Hier steht nur, was du
> nicht aus dem Code ableiten kannst. Alles andere: `doc/` (siehe unten).
> Wenn hier etwas dem Code widerspricht, gewinnt der Code — und die Datei
> gehört korrigiert.

---

## Orientierung

```
main.go              CLI: stdin / -i / -e / -t / --swank / Datei
cmd/golisp2d/        Server-Binary (SWANK-Daemon)
cmd/golisp2-client/  CLI-Client mit REPL
build/               Build-Artefakte (golisp2, golisp2d, golisp2-client)
lib/
  types*.go          Cell-Datenstruktur, Small-Int-Cache, Helfer
  reader.go          Parser: String → Cell-Baum
  env.go             Environment: verkettete Scopes, RWMutex
  eval_core.go       Eval-Trampolin, apply, evalArgs
  eval_*.go          Spezialformen, Lambda, Control, Quasiquote, load, exec
  primitives.go      Eingebaute Funktionen + BaseEnv()
  format*.go         FORMAT-Engine (CL-HyperSpec 22.3)
  <domäne>.go        goroutine, fileio, shellcmd, postgres, genalg, shm, sigorest
  stdlib.go          //go:embed stdlib.lisp
  readline.go        REPL (go-prompt, Highlighting, History)
  swank/             SWANK-Server für Emacs/SLIME
```


Vollständige Datei-für-Datei-Beschreibung: `doc/struktur.md`.
Im Zweifel: `rg` statt raten.

---

## Konventionen

- **Codierung:**
```
  ~/.claude/zutaten/sprachen/lisp.md
  ~/.claude/zutaten/sprachen/golisp2.md 
```

- **Sprache:** Go für den Kern, Lisp für Erweiterungen
- **Einrückung:** 2 Spaces, keine Tabs
- **Dateinamen:** camelCase bzw. `snake_case` für Domänen-Suffixe (außer `main.go`)
- **Kommentare:** sparsam — sprechende Namen bevorzugt
- **Datei-Header:** Autor, CoAutor, Copyright, Erstellt (YYYYMMDD)
- **Fehler:** `fmt.Errorf("funktionsname: beschreibung")`
- **Build:** immer `./build.sh` (baut alle drei Binaries nach `./build/`)
- **tmp:** `./tmp/` verwenden — **nicht** `/tmp` !
- **Dateigröße:** Richtwert 800, hart bei 1000 Zeilen.
  *Bewusste Projektausnahme zur globalen 300/500-Regel:* Kohäsion schlägt
  Zeilenzahl. Zusammenhängender Code (z. B. die FORMAT-Engine) wird nicht
  auseinandergerissen, nur um eine Zahl zu treffen. Erst splitten, wenn ein
  echter Schnitt existiert.

### Attribution

Mehrere Modelle arbeiten an diesem Projekt (siehe `AUTHORS.md`).
Falsche Zuschreibung ist ein Bug wie jeder andere.

- **Datei-Header:** nennt nur, wer die Datei *angelegt* hat. Nie nachträglich
  ändern — alte Einträge (`claude 3.7 sonnet` u. ä.) bleiben stehen.
- **Commits:** das schreibende Modell trägt sich **selbst** ein, mit exakter
  Bezeichnung. Nicht raten, nicht das vorige Modell abschreiben:
  `Co-Authored-By: claude-opus-4.8 <noreply@anthropic.com>`

### Bevor du neuen Code schreibst

**Erst suchen, dann schreiben.** `rg` vor jeder neuen Funktion — gibt es das
schon? Suchen kostet Tokens, Neuschreiben fühlt sich produktiv an. Der Anreiz
zeigt in die falsche Richtung. Widerstehe.

Eine zweite Implementierung von etwas Vorhandenem ist **immer** eine
Design-Entscheidung — auch wenn sie nichts bricht. **Additive Duplikate sind
still:** kein Compilerfehler, kein failing Test, kein Absturz. Sie fallen erst
bei der nächsten Änderung auf, wenn niemand mehr weiß, welche Hälfte läuft.
Ein Absturz wäre ein Geschenk. **Fragen, nicht anbauen.**

**Die Spec ist bindend.** Kommt dir eine Vorgabe falsch vor: sagen, nicht
umgehen.

### Chokepoints — genau eine Quelle, nie zwei

| Aufgabe | Einzige Quelle |
|---------|----------------|
| HTTP gegen sigoREST | `lib/sigorest.go` |
| Stdlib laden | `LoadStdlib` (`lib/stdlib.go`) |
| Truthiness | `IsTruthy` (`lib/types_helpers.go`) |
| Primitiven registrieren | `BaseEnv()` (`lib/primitives.go`) |
| Parsen | `lib/reader.go` (`Read` / `ReadAll`) |
| SWANK-Framing | `lib/swank/framing.go` |
| Eval-Schleife | `lib/eval_core.go` |

Wer anderswo einen HTTP-Client gegen `:9080` aufmacht, einen zweiten Parser
baut oder eine eigene Truthiness-Prüfung schreibt, macht es falsch — auch wenn
es funktioniert.

### Homoikonizität — Duplikate verstecken sich auch im Lisp

Das gefährlichste Duplikat ist keine Go-Datei. Es ist ein `define` in
`stdlib.lisp`, `swank.lisp` oder in zur Laufzeit evaluiertem Code.

- **Der letzte Schreiber gewinnt — aber nicht mehr lautlos.** Das Root-Env
  bewacht Redefinitionen per `(redefine-policy 'allow|'warn|'error)` (Default
  `warn`): Go-Primitiven (FUNC) immer, Lisp-Definitionen (LAMBDA/MACRO) bei
  fremder Quelle — Reload derselben Datei bleibt still. Alle Redefinitionen
  landen im Ringpuffer, abfragbar via `(redef-log)`. `(makunbound 'sym)`
  entfernt eine Root-Bindung. Details: `doc/lisp-semantik.md`.
- **Deshalb: `rg` muss `*.lisp` einschließen.** Eine Suche nur über `*.go`
  findet die halbe Wahrheit.
- **Spezialformen werden vor Makros geprüft** (siehe Eval-Reihenfolge). Ein
  `defmacro` mit dem Namen einer Spezialform wird von `eval` nie erreicht —
  bleibt aber im Root-Env und ist über `macroexpand` sichtbar. Zwei
  Implementierungen derselben Form, die still auseinanderlaufen. Der
  Redefine-Guard fängt das *nicht*: er warnt nur bei bestehenden
  FUNC-Bindungen, und Spezialformen sind gar keine Env-Bindungen.
  Bewacht von `TestNoLispDefineShadowsSpecialForm` (`lib/specialform_shadow_test.go`),
  der die Namen aus `eval_core.go` liest — Liste pflegen ist nicht nötig.
- **`(eval (read (sigo …)))` schreibt zur Laufzeit ins globale Env.** Das ist
  das selbsterweiternde Muster und ausdrücklich gewollt — aber es ist auch der
  direkteste Weg, eine Definition still zu überschreiben. Bei Arbeit an diesem
  Pfad: doppelt hinschauen.

### Wo kommt neuer Code hin?

- Braucht die Funktion `env`? → **Spezialform** in der passenden `eval_*.go`
  (`eval_specialforms.go`, `eval_control.go`, `eval_quasiquote.go`, …)
- Reine Berechnung ohne `env`? → **Primitiv** in `primitives.go`
- Gruppe verwandter Primitiven? → eigene Datei mit `RegisterXxx(env *Env)`
- **Neue Primitiven immer in `BaseEnv()` registrieren** — sonst existieren sie nicht.

---

## Invarianten — nicht versehentlich kaputtmachen

**TCO-Trampolin in `Eval()`**
Lambda-Calls und alle Tail-Spezialformen (`if`, `begin`, `let`, `cond`) setzen
`expr`/`env` und machen `continue` im `for {}`-Loop. Kein neuer Stack-Frame,
O(1) Stack bei beliebig tiefer Tail-Rekursion. Wer hier einen rekursiven
`Eval()`-Aufruf einbaut, zerstört das lautlos.

**`(eval form)` läuft im globalen Environment (`Env.Root()`)**
Nicht im dynamischen Lambda-Scope — Common-Lisp-Semantik. Notwendig, damit
Definitionen aus `(eval (read ...))` (REPL, `swank-repl:listener-eval`,
selbsterweiterndes Muster) global sichtbar bleiben und nicht im Child-Env der
aufrufenden Lambda-Kette verschwinden.

**Singleton-Nil + Symbol-Interning**
Genau **eine** NIL-Cell (`MakeNil()`), und genau **eine** Cell pro
Symbolname (`MakeAtom`, `internTable` in `types.go`). Es darf keine zweite
geben — ein zweites NIL oder ein nicht-interniertes `t` bricht `eq` still.
Nur lesen, nie modifizieren: eine Mutation trifft `parfunc` *und* jedes
andere Vorkommen desselben Symbols. Quellpositionen werden deshalb
ausschließlich auf `LIST`-Cells gestempelt (`reader.go`, `eval_load.go`).
Konsequenz: `eq` ist Pointer-Identität und für Symbole CL-korrekt —
`(eq 'foo 'foo)` → `t`. `equal?` bleibt struktureller Vergleich.
Das Env baut darauf: `GetSym`/`SetSym` schlagen Bindungen per
Pointer-Vergleich nach (`lib/env.go`). Eine nicht-internierte Symbol-Cell
würde dort still nicht gefunden — deshalb ist `MakeAtom` die **einzige**
erlaubte Quelle für ATOM-Cells, nie `&Cell{Type: ATOM, …}`.
Zwei bewusste Ausnahmen: Zahlen sind ausgenommen (`(eq 5 5)` → `()`,
damit der Small-Int-Cache `eq` nicht durch die Hintertür verändert; CL
lässt das unspezifiziert), Strings werden nicht interniert
(`(eq "a" "a")` → `()`, wie CL).

**Multi-Body via `wrapBegin`**
`defun`/`lambda`/`defmacro` wrappen mehrere Body-Ausdrücke zur *Definitionszeit*
in `(begin ...)`. Einzelausdruck bleibt unverpackt (kein Overhead).

**Eval-Reihenfolge in `evalList`**
1. Spezialformen · 2. Makro-Expansion · 3. Normale Anwendung (Funktion → Args → apply)

---

## Build & Test

```bash
./build.sh                                 # alle Binaries → ./build/
go build ./...                             # Ground Truth für "kompiliert es?"
go test ./...                              # Unit-Tests
go test ./... -count=1                     # bei überraschenden Ergebnissen (Cache!)

./build/golisp2 -t                         # Lisp-Testsuite
./build/golisp2 -i                         # REPL (braucht TTY)
./build/golisp2 -e "(+ 1 2)"               # Expression
echo "(+ 1 2)" | ./build/golisp2           # stdin (Default)
./build/golisp2 skript.lisp                # Datei

./build/golisp2d --port 4321               # SWANK-Server
./build/golisp2-client --repl              # Client-REPL
```

Exit-Codes: `0` = Erfolg, `1` = Fehler. Fehler → stderr, Ergebnisse → stdout.

---

## Engineering-Prinzipien

**Wahrheitsquellen**
- `go build ./...` schlägt Language-Server-Diagnostiken (cross-file false positives).
- Tests sind die objektive Wahrheit: neues Verhalten → failing test zuerst;
  existierender Code → Charakterisierungstest, der das IST festhält.
- Test-Netz **vor** riskanten Refactorings. Tests ermöglichen mutige Änderungen,
  weil sie exakt zeigen, was bricht.
- Common-Lisp-Semantik ist der Kompass, wenn GoLisp-Verhalten unklar ist.

**Arbeitsweise**
- Minimal Changes: kleine, gezielte Änderungen sind robuster als Big-Bang.
- Erkundung vor Read-Loop; Blast-Radius prüfen vor Signaturänderungen.
- Eine Quelle schlägt Synchronisation: eine `LoadStdlib`, ein `IsTruthy`, keine Duplikate.
- **Breaking Changes und Design-Ausweitungen sind Gerhards Entscheidung** —
  fragen, mit Optionen und Preview.
- Diese CLAUDE.md schlägt Standard-Tools: nicht blind `gofmt`/LSP-Warnungen folgen.

**Performance**
- Benchmark-driven: messen, revertieren wenn keine Verbesserung. Nicht spekulieren.
- Nicht jede Allokation ist heap-allokiert. Pooling nur, wenn der Profiler Heap-Druck zeigt.
- Immutable Cells erlauben Sharing ohne semantische Brüche.

**Go-Gotchas (erlebte Bugs, keine Theorie)**
- `:=` im Loop-Body shadowed Variablen aus dem enclosing Block.
  Multi-Return-Assignments in Loops immer via Hilfsvariable + `=`.
- `strings.Builder` nie per Value kopieren — Pointer oder frische Instanz pro Sub-Kontext.
- Debug-Logs akkumulieren; nie die `err`-Variable oder ein Debug-Feld mit einem Log überschreiben.

**Protokoll-Arbeit**
- Client-Source lesen statt Spec raten. Trace (`>>`/`<<`) früh und bidirektional einbauen.
- Unit-Tests gegen Protokolle asserten Struktur, nicht Substrings.
- Synthetische Tests müssen echtes Verhalten modellieren (Pipelining, Multi-Event, Stateful Conns).

**Namensgenerierung**
- Namensgenerierung braucht Kollisionsregeln und eine Warnung. Eine Ausweichregel ohne Meldung tauscht einen lauten Fehler gegen einen leisen — das ist der schlechtere Handel.
---

## Weiterführende Doku — bei Bedarf lesen, nicht vorsorglich

| Datei | Inhalt | Lies das, wenn … |
|-------|--------|------------------|
| `doc/struktur.md` | Datei-für-Datei-Beschreibung von `lib/` | du dich neu orientierst |
| `doc/cli.md` | Flags, Exit-Codes, Multiline-stdin, `exec`-Syntax | du an `main.go` arbeitest |
| `doc/swank.md` | SWANK-Protokoll, Framing, Op-Tabelle, SLIME-Details | du an `lib/swank/` arbeitest |
| `doc/sigo.md` | sigoREST: Env-Vars, Rate-Limiting, Multi-Host, Muster | du an `lib/sigorest.go` arbeitest |
| `doc/lisp-semantik.md` | `eq`/`equal?`, `let`/`let*`, `setq*`, `case`, FORMAT | Semantik unklar ist |
| `doc/memory.md` | GC-Verhalten, `(memstats)`, Best Practices | du Speicher untersuchst |
| `perfTodo.md` | Offene Performance-Arbeit | du optimierst |

### Was es bewusst *nicht* gibt

- **Keine Primitivenliste in der Doku.** Sie wäre ab dem nächsten
  `RegisterXxx()` falsch. Wahrheit ist der Code:
  `rg 'env\.Set\("' lib/` bzw. `(env-symbols)` zur Laufzeit.
- **Keine Modell-/Shortcode-Tabelle für `sigo`.** Provider deployen laufend
  neu. Wahrheit ist `(sigo-models)`.

---

## Philosophie

GoLisp2 ist ein **U-Boot-Projekt** — es reift in Ruhe, bevor es der Welt
gezeigt wird.

- **Nexialistisch:** Go-Effizienz + Lisp-Eleganz + KI-Power
- **Selbsterweiternd:** GoLisp2 vervollständigt sich durch KI-Calls selbst
- **Ensemble-fähig:** mehrere KIs parallel → Synthese
- **Centaur:** Mensch als Meta-Entscheider, KIs als Spezialisten

> „Code = Daten + KI = sich selbst erweiterndes System"
> – Gerhard & Claude, Juli 2026
