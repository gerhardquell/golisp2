# Task-Brief: Redefine-Guard + trace

Handoff für Claude Code. Zwei unabhängige Aufgaben. **Erst 1, dann 2.**
Keine externen Abhängigkeiten. Kein SQLite. Keine Datenbank.

**Motivation:** In einem homoikonischen System kann eine Definition still
überschrieben werden — aus `stdlib.lisp`, aus dem SWANK-REPL, aus
`(eval (read (sigo …)))`. Kein Compilerfehler, kein failing Test. Der Fehler
fällt erst Wochen später auf. Der Guard macht ihn laut.

---

## Aufgabe 1: Redefine-Guard

### Ziel

`Env.Set` auf dem **Root-Env** meldet bzw. verhindert das Überschreiben
existierender Go-Primitiven.

### Chokepoint

`lib/env.go`, `Env.Set`. Das ist der **einzige** Weg ins globale Environment —
aus Go (`BaseEnv`), aus Lisp (`define`/`defun`/`setq`), aus SWANK, aus `eval`.
Kein zweiter Hook, kein zweiter Ort.

### Regel

Greift **nur**, wenn alle drei Bedingungen gelten:

1. Env ist Root (kein Parent)
2. Binding existiert bereits
3. Altes Binding hat `Type == FUNC` (= Go-Primitiv aus `BaseEnv()`)

Lambdas (`Type == LIST` mit `Env != nil`) sind **nicht** geschützt — eigene
`defun`s im REPL neu zu definieren ist normal und darf nicht nerven.

### Policy

Go-Package-Variable, per Lisp-Primitiv setzbar:

```lisp
(redefine-policy 'allow)   ; still durchlassen
(redefine-policy 'warn)    ; DEFAULT: nach stderr melden, durchlassen
(redefine-policy 'error)   ; Fehler zurückgeben, Binding bleibt
(redefine-policy)          ; aktuelle Policy zurückgeben
```

Meldung: `REDEF: car (war FUNC)` → **stderr**, nie stdout.
(CLI-Vertrag: Ergebnisse → stdout, alles andere → stderr.)

### Performance — kritisch

`NewEnv+Set` steht mit **35 %** im Profil (siehe `perfTodo.md`). Der Hot Path
sind **Frame-Envs** (Lambda-Argumente), nicht der Root-Env.

Die Prüfung darf den Frame-Pfad **nicht** berühren:

```go
func (e *Env) Set(k string, v *Cell) {
  if e.parent != nil {          // Frame: unverändert, kein Overhead
    e.vars[k] = v
    return
  }
  // ... Root: Guard-Prüfung
}
```

Feldnamen (`parent`, `vars`) **im Code verifizieren**, nicht raten.

`go test -bench=Fib -count=1` vor und nach der Änderung. **Keine messbare
Regression erlaubt.** Bei Regression: melden, nicht kaschieren.

### Hook-Struktur

Die Prüfung ruft eine Funktion `onRootRedefine(name string, old, new *Cell)`.
Später hängt das Define-Log am selben Hook. **Ein Chokepoint, mehrere Nutzer.**

### Tests (`lib/env_test.go`)

| Fall | Erwartung |
|------|-----------|
| `(define car ...)`, Policy `error` | Fehler, `car` bleibt Primitiv |
| `(define car ...)`, Policy `warn` | stderr-Meldung, Überschreiben klappt |
| `(define car ...)`, Policy `allow` | still, klappt |
| `(define neuesSymbol ...)` | still — kein existierendes Binding |
| `(defun f ...)` zweimal | still — Lambda, kein FUNC |
| `(lambda (car) car)` aufrufen | **darf NICHT feuern** — Frame-Env, lokales Shadowing ist legal |
| Benchmark `fib` | keine Regression |

Der Lambda-Parameter-Fall ist der wichtigste. Wenn der feuert, ist der
Frame-Pfad falsch verdrahtet.

---

## Aufgabe 2: `(trace fn)` / `(untrace fn)`

### Ziel

Live-Sicht auf Aufruf und Rückgabe einer **gezielt ausgewählten** Funktion.
Kein globales Tracing.

### API

```lisp
(trace fib)       ; einschalten
(untrace fib)     ; ausschalten
(trace)           ; Liste der aktuell getracten Namen
(untrace)         ; alle aus
```

### Ausgabe — nach stderr, eingerückt nach Tiefe

```
0: (fib 3)
  1: (fib 2)
    2: (fib 1) => 1
    2: (fib 0) => 0
  1: (fib 2) => 1
  1: (fib 1) => 1
0: (fib 3) => 2
```

### Mechanik

Homoikonisch fast geschenkt: Binding aus dem Root-Env holen, in eine
Wrapper-Cell packen, zurückschreiben. `untrace` legt das Original zurück.
Original **immer** in einer Map aufbewahren — sonst ist es weg.

Muss für **beide** Typen funktionieren:
- `Type == FUNC` (Go-Primitiv) → `Fn` direkt aufrufen
- `Type == LIST` mit `Env != nil` (Lambda) → über `apply` aufrufen

Ist `apply` aus dem Trace-Modul erreichbar (`lib/eval_core.go`, gleiches
Package)? **Verifizieren.** Bei Import-Cycle: **stoppen und fragen**, nicht
umbauen.

### ⚠ Gotcha: Trace bricht TCO

Ein Lambda in eine Go-Wrapper-Funktion zu packen, unterbricht das
Eval-Trampolin — Tail-Calls der getracten Funktion bekommen wieder einen echten
Stack-Frame.

**Konsequenz:** Eine tail-rekursive Funktion, die untraced beliebig tief läuft,
kann getraced den Stack sprengen.

Das ist **akzeptiert** (CL verhält sich ähnlich), aber es muss:
- in `doc/lisp-semantik.md` dokumentiert werden
- als Kommentar über der Wrapper-Funktion stehen

Nicht versuchen, das zu „reparieren". Es ist der Preis.

### ⚠ Gotcha: parfunc

Der Tiefenzähler ist global. Unter `parfunc` verschachteln sich die Ausgaben
mehrerer Goroutinen ineinander.

**Akzeptiert.** Mutex ums Schreiben (damit keine Zeile zerreißt), Verschachtelung
dokumentieren. **Nicht** über-engineeren — das ist ein Diagnosewerkzeug, kein
Profiler.

### Tests

| Fall | Erwartung |
|------|-----------|
| `(trace fib)`, `(fib 3)` | Ein/Aus-Zeilen, korrekte Einrückung, korrektes Ergebnis |
| `(untrace fib)`, `(fib 3)` | keine Ausgabe, Original wiederhergestellt |
| `(trace car)` | Go-Primitiv wird getraced |
| `(trace)` nach 2× trace | beide Namen |
| Ergebniswert | durch Tracing **unverändert** |
| Ausgabe | **stderr**, nie stdout |

---

## Grenzen — hier ist Schluss

- **Kein** SQLite, kein Treiber, keine DB.
- **Kein** globales Auto-Tracing aller Funktionen.
- **Kein** Call-Graph, keine Aggregation, kein Export. Kommt später, separat.
- **Kein** Refactoring von `Env`, `Eval` oder `apply` über die genannten Stellen
  hinaus. Blast-Radius klein halten.

Wenn die Spec unvollständig oder falsch wirkt: **sagen, nicht umgehen.**

## Konventionen

`CLAUDE.md` gilt. Insbesondere:
2 Spaces · camelCase · Datei-Header · `./tmp/` statt `/tmp` · `./build.sh` ·
Fehler als `fmt.Errorf("funktion: beschreibung")`.

`go build ./...` ist die Ground Truth, nicht der Language Server.

**Attribution:** Das schreibende Modell trägt sich selbst als
`Co-Authored-By:` in den Commit ein — mit exakter Modellbezeichnung.

## Definition of Done

1. `go build ./...` sauber
2. `go test ./... -count=1` grün
3. `go test -bench=Fib -count=1` ohne Regression
4. `doc/lisp-semantik.md` ergänzt: `redefine-policy`, `trace`, TCO-Gotcha
5. Zwei getrennte Commits — Guard und Trace sind unabhängig
