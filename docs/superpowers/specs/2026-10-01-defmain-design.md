# defmain — Design

**Datum:** 2026-10-01 · **Status:** genehmigt (Brainstorming mit Gerhard)
**Autor:** Gerhard Quell · **CoAutor:** claude-opus-5.5

## Ziel

Ein Lisp-Skript soll einen expliziten Einstiegspunkt haben, der **nur**
läuft, wenn die Datei als Hauptprogramm gestartet wird (Shebang oder
`golisp2 datei.lisp`), und wirkungslos bleibt, wenn dieselbe Datei per
`(load …)` als Bibliothek eingebunden wird. Gegenstück zu C-`main` bzw.
Pythons `if __name__ == "__main__":`, aber als eigene Form statt Idiom.

## Ausgangslage (live geprüft, Stand 7894e53)

- Shebang `#!…` ist für den Reader nur ein Kommentar (`src/lib/reader.go:83`).
- Der Kernel macht aus `./skript.lisp a b` den Aufruf
  `golisp2 ./skript.lisp a b`. **Shebang-Start und `golisp2 datei.lisp`
  sind ununterscheidbar** — maßgeblich ist deshalb „diese Datei ist das
  Hauptprogramm“ (die Datei, die golisp2 als Argument bekam).
- `src/main.go:364–384` lädt die Datei über den String
  `(load "` + filename + `")` — bricht bei `"` oder `\` im Dateinamen.
  Danach wird der Wert der letzten Form auf stdout ausgegeben, Exit 0
  (Fehler: `ERR: …` auf stderr, Exit 1).
- `(argv)`, `(getenv)`, `(environ)` existieren (`src/lib/sysinfo.go`).
  `(argv)` liefert die rohe Kommandozeile inkl. Binary und Skriptpfad.
- `evalCtx` (`src/lib/eval_core.go:46`) trägt `depth` und `ctx`;
  `child()` reicht `ctx` durch. `(eval …)` (`eval_control.go:204`) und
  Goroutinen (`eval_control.go:528`) starten mit frischem `evalCtx`.
- `evalLoad` stempelt `SrcFile` nur auf Top-Level-Formen
  (`eval_load.go:56`).
- Kein `defmain` im Repo (weder Go noch Lisp).

## Entscheidungen (aus dem Brainstorming)

| Frage | Entscheidung |
|---|---|
| Wann läuft der Körper? | **Nach** dem Laden der ganzen Datei — Position in der Datei egal |
| Argumente/Environment speichern? | **Nein** — `(argv)`/`(getenv)`/`(environ)` existieren bereits |
| Parameterliste? | **Ja:** `(defmain (args) …)` bzw. `(defmain () …)`; `args` = reine Skript-Argumente |
| Rückgabewert? | **Exit-Code** (wie C-`main`), keine stdout-Ausgabe |
| Doppeltes `defmain`? | **Fehler**, Exit 1 |
| Beim Laden aufrufbar (Testbarkeit)? | **Nein** — keine Zusatzbindung `main`, keine neue Fehlerfläche |
| Erkennung der Hauptdatei | **Merker im Ladekontext** (`evalCtx`), nicht Pfadvergleich, nicht global |

## 1. Semantik (Nutzersicht)

```lisp
#!/usr/local/bin/golisp2
(defun greet (name) (println (format nil "Hallo, ~a!" name)))

(defmain (args)
  (if (null args)
      (begin (warn "Aufruf: greet.lisp NAME") 2)
      (begin (greet (car args)) 0)))
```

**Syntax:** `(defmain (params) body…)`. Parameterliste ist `()` oder
`(name)` — genau ein Symbol. Alles andere (`(a b)`, `args` ohne Klammern,
fehlende Parameterliste) ist ein Syntaxfehler. Mehrere Body-Ausdrücke
werden wie bei `defun` per `wrapBegin` verpackt.

**In der Hauptdatei** (`golisp2 datei.lisp …` / `./datei.lisp …`):

1. `defmain` speichert den Körper als Closure (im Env der Definition) und
   liefert `nil`; die Datei wird zu Ende geladen.
2. Danach wird der Körper aufgerufen. Bei `(name)` ist `name` an die
   Skript-Argumente gebunden (`./greet.lisp Anna -v` → `("Anna" "-v")`);
   bei `()` wird ohne Argument aufgerufen.
3. Exit-Code aus dem Rückgabewert:
   - ganze Zahl 0–255 → dieser Exit-Code,
   - `nil` oder Nicht-Zahl → 0,
   - Zahl außerhalb 0–255 oder nicht ganzzahlig →
     `ERR: defmain: Exit-Code muss ganze Zahl 0–255 sein, got <wert>`,
     Exit 1 (laut statt stilles `256 → 0` des Kernels).
4. Keine zusätzliche stdout-Ausgabe.
5. Zweites `defmain` in der Hauptdatei →
   `ERR: load <datei>: defmain: bereits definiert in <datei>:<zeile>`, Exit 1;
   kein Körper läuft.
6. Fehler im Körper → `ERR: …` auf stderr, Exit 1.

**Überall sonst** — per `(load …)` nachgeladene Datei, REPL, SWANK,
`-e`, stdin-Modus, `(eval …)`, Goroutinen — liefert `defmain` `nil`
und tut nichts. Die Parameterliste wird dort **nicht** geprüft; Syntax-
fehler fallen erst auf, wenn die Datei selbst Hauptprogramm ist.

`defmain` liefert auch in der Hauptdatei `nil` — der Körper läuft erst
nach dem Laden, an der Definitionsstelle gibt es keinen Wert.

**Ohne `defmain`:** Verhalten wie bisher (Wert der letzten Form auf
stdout, Exit 0 / 1).

Verschachtelte Verwendung in der Hauptdatei (`(when debug (defmain …))`)
wirkt, weil der Merker über `evalCtx.child()` mitwandert.

## 2. Architektur (Go)

**Neu: `src/lib/eval_script.go`**

```go
// mainScriptState existiert nur, während RunScript die Hauptdatei lädt.
type mainScriptState struct {
  body    *Cell  // Closure aus defmain; nil = kein defmain gesehen
  srcFile string // für „bereits definiert in …“
  srcLine int
}

// RunScript lädt path als Hauptprogramm, ruft danach ggf. den
// defmain-Körper mit args auf und liefert den Exit-Code.
func RunScript(path string, args []string, env *Env) (exitCode int, result *Cell, hasMain bool, err error)

func evalDefmain(form *Cell, env *Env, ectx evalCtx) (*Cell, error)
```

**Geändert:**

| Datei | Änderung |
|---|---|
| `eval_core.go` | `evalCtx` + Feld `script *mainScriptState`; `child()` kopiert den Zeiger; Dispatch `case "defmain"` |
| `eval_load.go` | Kern von `evalLoad` → `loadFile(path, env, ectx)`. `evalLoad` ruft ihn mit `script=nil` (nachgeladenes `defmain` wirkungslos), `RunScript` mit Zustand. Weiterhin genau **ein** Dateilade-Pfad |
| `main.go` | Datei-Modus ruft `lib.RunScript(flag.Arg(0), flag.Args()[1:], env)` statt String-Bau. Ohne `defmain`: `result` ausgeben wie bisher. Mit `defmain`: nichts ausgeben, `os.Exit(exitCode)` |

**Ablauf `RunScript`:**

1. `st := &mainScriptState{}`; Pfad wie bisher über `resolvePath` +
   `filepath.Abs` auflösen; `loadFile(path, env, evalCtx{script: st})`.
2. `evalDefmain`:
   - `ectx.script == nil` → `nil`, keine Prüfung.
   - Parameterliste ungültig → Syntaxfehler.
   - `st.body != nil` → Fehler „bereits definiert in `st.srcFile`:`st.srcLine`“.
   - sonst `st.body = makeLambda(params, wrapBegin(body), env)`,
     Quellort merken, `nil` liefern.
3. Nach dem Laden, falls `st.body != nil`: `apply(st.body, argListe)`
   (ein Argument bei `(name)`, keines bei `()`), mit frischem Kontext
   **ohne** Script-Zustand — ein `defmain` im Körper ist wirkungslos.
4. Rückgabewert → Exit-Code nach Abschnitt 1.

**Warum Zeiger statt `bool`:** Zustand gehört genau einem `RunScript`-
Aufruf — kein globaler Zustand, kein Zurücksetzen, unsichtbar für
Goroutinen; Erkennung und Ablage des Körpers in einem.

**Invarianten:** TCO-Trampolin unberührt (der Zeiger reist im
`evalCtx`-Wert, kein rekursiver `Eval()`-Aufruf). `defmain` steht in der
einzigen Spezialform-Liste (`eval_core.go`); `TestNoLispDefineShadowsSpecialForm`
deckt den Namen automatisch ab.

## 3. Tests, Messung, Doku

**Go-Unit-Tests** (`src/lib/eval_script_test.go`, TDD: rot zuerst).
Jeder Test schreibt ein Skript nach `t.TempDir()` und ruft `RunScript`:

| Fall | Erwartung |
|---|---|
| `(defmain (args) (parse-int (car args)))`, args `["7"]` | Exit 7 |
| `(defmain () 0)` | Exit 0, `hasMain=true` |
| `defmain` vor dem `defun`, das es nutzt | läuft (Körper nach dem Laden) |
| `(when t (defmain …))` | wirkt |
| Rückgabe `nil` / String | Exit 0 |
| Rückgabe 256 / -1 / 2.5 | Fehler, Exit 1 |
| zwei `defmain` | Fehler „bereits definiert in …:zeile“, kein Körper läuft |
| `(defmain (a b) …)`, `(defmain args …)` | Syntaxfehler |
| Hauptdatei lädt Bibliothek mit eigenem `defmain` | Bibliotheks-`defmain` → `nil`; Haupt-`defmain` läuft |
| `defmain` via `LoadString`, `Eval`, `(eval …)` | `nil`, keine Wirkung |
| Fehler im Körper | `err` gesetzt, Exit 1 |
| Datei ohne `defmain` | `hasMain=false`, `result` = letzte Form |
| Dateiname mit `"` | lädt korrekt (Regression String-Bau) |

**End-to-End** in der Lisp-Suite: Shebang-Skript (`chmod +x`), per
`exec` als echter Prozess gestartet; geprüft werden stdout (kein
angehängter Rückgabewert), Exit-Code, durchgereichte Argumente. Die
genaue Einhängung (Testverzeichnis, Fixture) wird im Plan festgelegt.

**Benchmark** (CLAUDE.md: Eval-Schleife): `BenchmarkFib` und
`evalBench_test.go` mit `-count=5` vor/nach; Vergleich via `benchstat`
(falls installiert) sonst manuell. Messbare Regression (> 2 % jenseits
des Rauschens) → Gerhard vorlegen, bevor weitergemacht wird.

**Doku:**
- `docs/cli.md`: Abschnitt „Skripte mit `defmain`“ (Beispiel, Exit-Code-
  Regeln, wann `defmain` wirkungslos ist).
- `src/embed/ki-referenz.md`, `docs/golisp2-cheatsheet.md`: je ein Eintrag.
- `docs/struktur.md`: `eval_script.go`.
- `docs/referenz-generiert.md`: prüfen, ob Spezialformen dort einen
  Abschnitt haben (generiert aus `(env-symbols)`; Spezialformen sind
  keine Env-Bindungen).

## Nicht-Ziele

- Keine aufrufbare `main`-Bindung beim Laden (Testbarkeit über echten
  Prozess per `exec`).
- Kein `__name__`-/`*load-truename*`-Äquivalent.
- Kein Argument-Parsing (Flags, Optionen) — Skripte parsen `args` selbst.
- Keine Speicherung von Argumenten/Environment — `(argv)`/`(getenv)`/
  `(environ)` sind die einzige Quelle.
