# Design: `exec` Primitive in GoLisp2

**Datum:** 2026-07-11  
**Status:** Genehmigt  
**Autor:** Gerhard Quell / Claude

## Ziel

GoLisp2 erhält eine Spezialform `exec`, mit der externe Programme direkt (ohne Shell) aufgerufen werden können. Eingabe, Ausgabe, Fehlerausgabe und Exit-Code werden über Keyword-Argumente gesteuert.

## Architektur

- `exec` ist eine **Spezialform** (wie `define`, `let`, …), weil sie Variable im aktuellen Environment setzen muss.
- Implementierung in neuer Datei `lib/eval_exec.go`.
- Dispatch in `lib/eval_core.go` unter den Nicht-Tail-Spezialformen.

## Lisp-API

```lisp
(exec "programm"
      param: "arg1"
      param: "arg2"
      stdin:  eingabe
      stdout: ausgabe-var
      stderr: fehler-var
      exitcd: code-var)
```

| Keyword | Auswertung | Bedeutung |
|---------|------------|-----------|
| `param:` | Ja | Ein Kommandozeilenargument als String. Mehrfach erlaubt. |
| `stdin:` | Ja | String, der in stdin geschrieben wird. |
| `stdout:` | Nein | Name der Variable, in die stdout geschrieben wird. |
| `stderr:` | Nein | Name der Variable, in die stderr geschrieben wird. |
| `exitcd:` | Nein | Name der Variable, in die der Exit-Code geschrieben wird. |

## Semantik

- Rückgabe: `t` wenn der Prozess gestartet und beendet wurde.
- Exit-Code ≠ 0 ist kein Fehler; er landet in `exitcd:`.
- Technischer Fehler (z. B. Programm nicht gefunden) → Rückgabe `nil`, `exitcd:` wird auf `-1` gesetzt (falls angegeben).
- Fehlende Keywords sind optional.
- Nicht angeforderte Streams werden verworfen.

## Go-Implementierung

### Dispatch

In `lib/eval_core.go`:

```go
case "exec": return evalExec(expr.Cdr, env)
```

### Parser

```go
func evalExec(args *Cell, env *Env) (*Cell, error)
```

- Erstes Argument ist der Programmname (wird evaluiert).
- Danach abwechselnd Keyword-Atom und Wert.
- Keywords: `param:`, `stdin:`, `stdout:`, `stderr:`, `exitcd:`.
- Unbekannte Keywords → Fehler.

### Ausführung

- `os/exec.Command(program, params...)`
- `bytes.Buffer` für stdout/stderr
- `strings.NewReader(stdin)` für stdin
- Nach `cmd.Run()` Variablen setzen und `t` bzw. `nil` zurückgeben.

## Fehlerbehandlung

- `fmt.Errorf("exec: ...")` auf Deutsch.
- Fehler bei zu wenig Argumenten, unbekanntem Keyword, falschem Typ.

## Tests

- `lib/eval_exec_test.go`
- Fälle:
  1. Echo-Programm, stdout abfragen.
  2. Exit-Code ≠ 0.
  3. Stderr abfragen.
  4. Stdin an Programm übergeben.
  5. Unbekanntes Programm → `nil`.
  6. Mehrere `param:`-Argumente.

## Dateien

| Datei | Änderung |
|-------|----------|
| `lib/eval_exec.go` | neu |
| `lib/eval_exec_test.go` | neu |
| `lib/eval_core.go` | `case "exec": return evalExec(...)` |
