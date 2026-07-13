# Session – Redefine-Guard & Live-Tracing

**Datum:** 2026-07-13
**Autoren:** Gerhard Quell & kimi

---

## Was haben wir gebaut?

| Feature | Dateien | Commits |
|---------|---------|---------|
| Redefine-Guard für Root-Env | `lib/env.go`, `lib/primitives.go`, `lib/eval_*.go`, Register-Dateien | offen |
| `redefine-policy` Primitive | `lib/primitives.go` | offen |
| Tests für Redefine-Guard | `lib/env_test.go` | offen |
| Live-Tracing `trace`/`untrace`/`trace?` | `lib/trace.go`, `lib/primitives.go` | offen |
| stderr-Output-Infrastruktur | `lib/output.go`, `lib/output_test.go` | offen |
| Tests für Tracing | `lib/trace_test.go` | offen |
| Doku | `doc/lisp-semantik.md` | offen |

**Gesamt:** 2 Features, ~7 Dateien, ~300 neue Zeilen Code & Tests.

**Vorher:** Go-Primitiven konnten im Root-Env lautlos überschrieben werden; keine Möglichkeit, einzelne Funktionen live zu tracen.
**Nachher:** Redefinitionen von Primitiven sind nach Policy sichtbar/verhinderbar; beliebige Root-Env-Funktionen können mit Einrückung und Return-Wert getraced werden.

---

## Was lief gut?

### Redefine-Guard am richtigen Chokepoint
`Env.Set` in `lib/env.go` ist die einzige Stelle, die Root-Env-Schreibzugriffe kontrolliert. Der Guard fängt hier ab — alle Pfade (Go-Registrierung, Lisp `define`/`setq`/`defun`, SWANK, `eval`) sind abgedeckt, ohne dass jeder Aufrufer eigene Logik braucht.

### `Env.Set` Signatur-Änderung zu `error`
Anfangs unbequem (viele Call-Sites), aber die einzige saubere Lösung für Policy `error`. Frame-Envs brauchen die Rückgabe nicht, Root-Caller propagieren sie. Das vermeidet einen zweiten Chokepoint.

### Wrapper-Ansatz für Tracing
`apply` und `applyLambda` blieben unverändert. Der Wrapper ersetzt nur die Root-Env-Bindung durch eine `FUNC`-Cell, die vor/nach dem Aufruf nach stderr schreibt. TCO für nicht-getracede Aufrufe bleibt erhalten.

### Idempotenz von `(trace 'name)
Durch Vorab-Prüfung in `traceOrigs` wird ein bereits getracedes Symbol nicht doppelt gewrappt. `(trace '+)(trace '+)` verhält sich wie ein einzelner Aufruf.

### Zentraler stderr-Writer
`WriteError`/`SetErrorWriter` erlauben saubere Tests ohne globale `os.Stderr`-Manipulation. Das ist auch für SWANK relevant, falls später stderr separat geroutet werden soll.

---

## Was lief nicht so gut?

### Bash-Classifier temporär nicht verfügbar
Die Lisp-Testsuite `./build/golisp2 -t` konnte nicht ausgeführt werden, weil der Bash-Classifier zwischenzeitlich blockiert war. Unit-Tests (`go test ./...`) waren erfolgreich (164 passed), die eigentliche Lisp-Testsuite bleibt offen.

### Testkomplexität für globale Zustände
`traceOrigs` ist package-level. Tests müssen am Ende `(untrace)` aufrufen, um die Registry zu säubern. Vergisst man das, beeinflussen Tests einander. Eine `withEnv`/`withCleanTrace`-Hilfsfunktion reduziert das Risiko, bleibt aber ein Stolperstein.

### Lambda-Trace-Test ist empfindlich
`TestTraceLambdaNested` prüft die exakte Einrückung der ersten drei Zeilen. Das ist robust genug für die aktuelle Implementierung, aber jede Änderung an der Trace-Formatierung bricht diesen Test.

---

## Technische Erkenntnisse

### Homoikonizität verlangt Chokepoints
In einem Lisp, in dem Code = Daten ist, können Definitionen lautlos überschrieben werden. Der einzige Schutz ist ein klar definierter Schreibpfad. `Env.Set` ist dieser Pfad.

### Thread-Safety bei package-level Zustand
`parfunc` kann mehrere Goroutinen im Root-Env starten. Deshalb:
- Policy über `atomic.Int32`
- Trace-Registry über `sync.RWMutex`
- Output-Writer über `sync.Mutex`

Ohne diese Sicherungen wären Races bei parallelen `(trace ...)`-Aufrufen wahrscheinlich.

### TCO vs. Wrapper sind unvereinbar für getracede Calls
Ein getracede Tail-Call kann nicht mehr TCO-optimiert werden, weil der Wrapper einen Go-Stackframe hinzufügt. Das ist der akzeptierte Kostenfaktor des Designs.

### `apply(original, args)` funktioniert für FUNC und LIST
Der Wrapper muss nicht unterscheiden, ob das Original eine eingebaute Funktion oder ein Lambda ist. `apply` dispatched korrekt. Für Makros funktioniert das nicht, weil Makros vor der normalen Eval-Reihenfolge expandiert werden.

---

## Fazit

Redefine-Guard und Tracing sind zwei Features, die GoLisp2 debugg- und wartbarer machen — ohne den Eval-Kern zu verändern. Beide setzen auf existierende Chokepoints (`Env.Set`, Root-Env-Bindungen) auf, anstatt neue parallele Pfade einzuführen.

Offen bleibt der abschließende Check mit `./build/golisp2 -t`.

> „Stille Überschreibungen sind gefährlicher als laute Fehler.
>  Tracing macht das Unsichtbare sichtbar."
> — Gerhard & kimi, Juli 2026
