# GoLisp2 – Unix-Style CLI

GoLisp2 verhält sich wie ein typisches Unix-Tool: Ergebnisse nach stdout,
Fehler nach stderr, Exit-Code sagt die Wahrheit.

## Flags

| Flag | Beschreibung | Beispiel |
|------|--------------|----------|
| *(default)* | Liest von stdin, gibt nur das Ergebnis aus | `echo "(+ 1 2)" \| ./build/golisp2` |
| `-i` | Interaktiver REPL (go-prompt, braucht TTY) | `./build/golisp2 -i` |
| `-e EXPR` | Expression(en) direkt ausführen | `./build/golisp2 -e "(* 6 7)"` |
| `-t` | Lisp-Testsuite ausführen (Exit-Code = Anzahl FAILs) | `./build/golisp2 -t` |
| `--swank HOST:PORT` | SWANK-Server starten (Emacs/SLIME) | `./build/golisp2 --swank 127.0.0.1:4242` |
| `DATEI` | Lisp-Datei laden und ausführen | `./build/golisp2 script.lisp` |

**Shebang:** Eine Zeile `#!…` (typisch `#!/usr/local/bin/golisp2`) gilt
überall im Quelltext als Kommentar bis Zeilenende (SBCL-Konvention).
Skripte sind damit direkt ausführbar (`chmod +x script.lisp`) und bleiben
gleichzeitig per `(load "script.lisp")` ladbar.

**Hinweis zu `-e`:** Eine einzelne Form gibt ihr Ergebnis aus. Bei mehreren
Formen wird das letzte Ergebnis unterdrückt, damit rein seiteneffektbehaftete
Skripte saubere Ausgabe erzeugen.

## Exit-Codes

- **0** – Erfolg
- **1** – Fehler (Parser, Eval, unbekanntes Symbol, …)
- **-t:** Anzahl fehlgeschlagener Framework-Tests (`run-tests`), 0 = grün
- Lisp-seitig: `(exit n)` beendet den Prozess sofort mit Code `n`

```bash
echo "(+ 1 2)" | ./build/golisp2; echo $?   # → 0
./build/golisp2 -e "(error 'x')"; echo $?   # → 1
```

## Fehlerbehandlung

- Alle Fehler → `stderr`
- Alle Ergebnisse → `stdout`
- Fehler in Pipe/Datei: weitere Expressions werden trotzdem verarbeitet,
  Exit-Code am Ende `1`

## `(read-line)` – Freitext von stdin lesen (TODO.md, 20260813)

`(read-line)` liest eine Zeile von `os.Stdin`, liefert sie als String ohne
Newline (kein Parsing — `(read (read-line))` kombinieren, wer Lisp-Daten
will).

**Nur sinnvoll im Datei-Argument-Modus** (`golisp2 skript.lisp`, auch per
Shebang direkt ausgeführt, z. B. `./skript.lisp`) — dort bleibt stdin
unberührt und mit dem Terminal verbunden.

- **NICHT** im Default-stdin-Modus (kein Datei-Argument): `runStdin` in
  `main.go` liest bereits vor der Auswertung das komplette stdin via
  `io.ReadAll(os.Stdin)` als Programmquelle ein — `(read-line)` trifft dort
  zur Laufzeit auf EOF.
- **NICHT** über `--swank`: kein Reverse-RPC zum Client, Emacs/SLIME leitet
  Tastatureingaben nicht an golisp2s Stdin weiter.

## Multiline-Support (stdin)

Eine Expression wird erst ausgewertet, wenn die Klammern ausgeglichen sind:

```bash
cat <<'EOF' | ./build/golisp2
(defun square (x)
  (* x x))
(square 5)
EOF
# → 25
```

---

## REPL (`golisp2 -i`, `lib/readline.go`)

- **Start:** `./build/golisp2 -i` — benötigt ein TTY; im Skript/CI kommt eine
  Fehlermeldung
- **Syntax-Highlighting:** Klammern nach Tiefe eingefärbt (6 Farben, fett) ·
  Strings grün · Kommentare grau · Quote-Zeichen gelb
- **Multiline:** Enter bei offenem Ausdruck → automatische Einrückung
- **History:** persistent in `~/.golisp_history` (500 Einträge)
- **Library:** `github.com/elk-language/go-prompt`

Der REPL des *Clients* (`golisp2-client --repl`) ist etwas anderes und läuft
über SWANK — siehe `docs/swank.md`.

---

## `exec` – externe Programme ausführen

```lisp
(exec "programm"
      param:  "arg1"
      param:  "arg2"
      stdin:  eingabe
      stdout: ausgabe-var
      stderr: fehler-var
      exitcd: code-var)
```

- `param:` und `stdin:` werden **ausgewertet**.
- `stdout:`, `stderr:`, `exitcd:` sind **Variablennamen** (werden gesetzt, nicht ausgewertet).
- Rückgabe `t` bei erfolgreichem Start/Beenden, `nil` bei technischem Fehler
  (z. B. Programm nicht gefunden).
- Ein Exit-Code ≠ 0 ist **kein** Fehler — er landet in `exitcd:`.
- Standard-Timeout: 60 Sekunden.
