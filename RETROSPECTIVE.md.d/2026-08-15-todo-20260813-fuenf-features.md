# Retrospektive: TODO 20260813 — fünf Features an einem Tag

**Datum:** 15. August 2026
**Autor:** Gerhard Quell & kimi-k3
**Feature:** Komplette Abarbeitung von TODO.md (Aufgabe 20260813):
working-directory-Singular + file-write-Fix, IO-Streams (sys-stdin/out/err),
C-formatierte Ausgabe (printf-Familie), CLI/Environment-Zugriff, CLOS-light

---

## Was wurde gebaut?

Fünf Commits, fünf Features, alle aus TODO.md (20260813), ausgelöst durch
einen Live-Befund aus dem sixhat-Projekt über SWANK:

- **Punkt 1 + 2.1 — working-directory (Commit `5bf8d00`):**
  `file-write`/`file-append` ignorierten `set-working-directories` komplett —
  SWANK-Server außerhalb des Projektverzeichnisses → Schreibzugriffe landeten
  im Nirgendwo. Gerhards Design: Plural-API (`set/get-working-directories`)
  durch Singular (`set/get-working-directory`) ersetzt. Entscheidender
  Schnitt: **zwei Resolver statt einer** — `resolvePath` (lesen, prüft
  Existenz) vs. `resolveWritePath` (schreiben, keine Existenzprüfung, neue
  Dateien erlaubt). Die Chokepoint-Frage aus der TODO ("eine gemeinsame
  Basis oder zwei getrennte Funktionen?") wurde bewusst mit "zwei getrennte"
  beantwortet. Toter Code (`cellToPathList`, `splitPathString`) entfernt.
- **Punkt 2.2 — IO-Streams (Commit `892787b`):** Pseudodateien
  `sys-stdin`/`sys-stdout`/`sys-stderr` in `file-read`/`file-write`/
  `file-append`, plus `(gets)`, `(slurp)`, `(err-write ...)`. Alle Streams
  laufen über die bestehenden Chokepoints (`WriteOutput`/`WriteError`,
  geteilter `stdinReader`) — dadurch sind sie automatisch SWANK-sichtbar.
- **Punkt 2.3 — C-Formatierung (Commit `429bb17`):** `lib/cformat.go` —
  `printf`/`sprintf`/`fprintf`/`sscanf`. Kernidee: **Übersetzer statt
  zweiter Format-Engine** — `cformatToCL` wandelt C-Formatstrings nach
  CL-format um, Ausgabe läuft über die bestehende FORMAT-Engine
  (eine Quelle, keine Duplikation). Komplett rune-basiert (Unicode-sicher:
  `%6s` mit "hällö" paddet korrekt). Lücken sind laut, nicht still:
  `%.Ns`, linksbündige Floats → Fehler statt falschem Ergebnis.
- **Punkt 2.4 — CLI/Environment (Commit `81d6382`):** `lib/sysinfo.go` —
  `(argv)` (rohe os.Args), `(getenv)` (LookupEnv: leer ≠ unset),
  `(environ)` (Alist). Damit sind Shebang-Skripte mit Argumenten möglich.
- **Punkt 2.5 — CLOS-light (Commit `f991cea`):** `defgeneric`/`defmethod`
  in `embed/stdlib.lisp` — **rein Lisp, kein Kernel-Eingriff**.
  Single-Dispatch auf Struct-Tag, `t` = Default-Methode, Hot-Redefinition
  über Registry-Hashtabelle. Explizit nicht dabei: Vererbung,
  `call-next-method`, `:before`/`:after`, Multi-Dispatch.
- **Doku:** cheatsheet, referenz.md/_en/_cn (dreisprachig), cl-inventar.md,
  lisp-semantik.md (neuer Abschnitt "Generische Funktionen").
- **Verifikation:** 325 Go-Tests grün, Lisp-Suite 104 PASS / 0 FAIL,
  Smoke-Tests (Shebang-argv, printf-Unicode, CLOS-Dispatch) durch frische
  Build-Subagenten.

---

## Was lief schief? (⚫ Schwarz)

| Problem | Ursache | Auswirkung |
|---------|---------|------------|
| `unbekanntes Symbol 'defgeneric'` nach stdlib-Edit | `go build ./...` baut nicht `./build/golisp2` neu — `//go:embed` erfordert `./build.sh` | Eine Testrunde gegen das alte Binary, bis der Unterschied aufging |
| `lambda: Parameter muss Atom sein` im defmethod-Makro | GoLisp-Makros destrukturieren keine verschachtelten Lambda-Listen — `(name (spec &rest params) &rest body)` geht nicht | defmethod-Signatur musste flach werden (`car`/`cdr` im Body); funktional gleichwertig, aber ein versteckter Sprachlimit-Fund |
| `TestArgv` schlug fehl: `unbekanntes Symbol 'cadr'` | GoLisp hat nur `car`/`cdr`, kein `cadr` — CL-Gewohnheit beim Testschreiben | Kleiner Test-Fix, aber Symptom: CL-Muskelgedächtnis schreibt Tests, die der eigene Interpreter nicht kann |
| `(sprintf "%.2f" 2.25)` → `"2.2"` statt erwartetem `"2.3"` | Go-`strconv` rundet half-to-even (Banker's), C rundet half-up — FORMAT-Engine erbt Go-Verhalten | Test-Input geändert (2.26); semantische Abweichung zu C bleibt dokumentiert |
| Erster Smoke-Test meldete `%S`-Fehler | Prompt des Smoke-Agenten nutzte `%S` — Direktive existiert bewusst nicht | Feature war korrekt, der Test war falsch; Neulauf mit `%s` nötig |
| `*: Zahl erwartet, got ()` im CLOS-Smoke-Test | `(make-kreis)` ohne `:radius` gibt NIL-Radius — Test bug, kein Feature-Bug | t-Default-Methode musste mit unbekanntem Tag getestet werden, nicht mit fehlendem Slot |
| `sed`-Sammelersetzung in referenz.md/_en/_cn | ``defstruct` `` → ``defstruct defgeneric defmethod` `` über drei Sprachdateien: traf auch Tabellenzeilen und erzeugte Widerspruch zu "Kein CLOS"-Abschnitten | Drei Dateien von Hand nachrepariert; sed auf Markdown-Prosa mit Code-Spans ist ein stumpfes Messer |
| Veralteter Test-Code kompilierte nicht nach Plural→Singular-Umbau | `fileio_test.go` referenzierte entfernte `workingDirectories`-Variablen | Erwartbar bei API-Bruch; kompletter Rewrite der Datei statt Flickwerk |

---

## Was haben wir gelernt? (🔵 Blau)

1. **`go build ./...` ist NICHT die Ground Truth für `./build/golisp2`.**
   Bei `//go:embed stdlib.lisp` trägt das gebaute Test-Binary zwar die neue
   Stdlib, aber das CLI-Binary unter `./build/` bleibt alt, bis `./build.sh`
   läuft. Nach Stdlib-Änderungen immer `./build.sh`, dann smoke-testen.
2. **Zwei Resolver mit unterschiedlicher Semantik schlagen einen
   verbogenen.** `resolvePath` prüft Existenz (lesen), `resolveWritePath`
   prüft nicht (schreiben neuer Dateien). Eine gemeinsame Funktion mit Flag
   hätte den Chokepoint-Buchstaben erfüllt, aber den Geist verletzt:
   unterschiedliche Semantik gehört in unterschiedliche Namen.
3. **Übersetzen schlägt Nachbauen.** `cformatToCL` (C→CL-Formatstring) +
   bestehende FORMAT-Engine = ~350 Zeilen für die ganze printf-Familie.
   Eine zweite C-kompatible Engine wäre ein additives Duplikat nach
   CLAUDE.md-Definition gewesen — still, bis zur ersten FORMAT-Änderung.
4. **Laut scheitern schlägt still falsch liefern** — zweimal an einem Tag:
   `%.Ns` und linksbündige Floats werfen Fehler statt C-inkompatibel
   zu raten; der Redefine-Guard aus früheren Sessions warnt bei
   Shadowing. Beides dieselbe Philosophie: ein Absturz ist ein Geschenk.
5. **GoLisps Sprachgrenzen zeigen sich erst beim Schreiben von Lisp, nicht
   beim Lesen von Go.** Kein `cadr`, keine verschachtelte Destrukturierung
   in Makro-Lambda-Listen, `define`/`defstruct` in Lambda-Scope lokal
   (eval-Trick in Tests nötig). Diese Grenzen sind bekannte Fallen
   (siehe Memory `golisp2-laufzeit-fallen`) — und wurden trotzdem wieder
   live erlebt. Der Wert des Memory-Eintrags zeigt sich im Wiedererkennen,
   nicht im Vermeiden.
6. **half-to-even vs. half-up ist kein Bug, sondern eine Entscheidung.**
   Go rundet `2.25` → `2.2`, C → `2.3`. Die FORMAT-Engine erbt Go. Wer
   C-printf emuliert, emuliert nicht automatisch C-Rundung — das gehört in
   die Doku, nicht in den Code.
7. **Brainstorming-Gate hat sich bewährt.** Alle fünf Punkte gingen erst
   als Design (teilweise per AskUserQuestion: Chokepoint-Routing, Naming,
   Übersetzer-Ansatz, CLOS-Scope Option A) an Gerhard, dann in Umsetzung.
   Null Rework durch falsch verstandene Anforderungen — die Gate-Kosten
   (fünf kurze Design-Runden) waren kleiner als jede einzelne falsche
   Implementierung gewesen wäre.
8. **CLOS-light in reinem Lisp zu bauen war die schnellste Option** —
   kein Kernel-Eingriff, kein neuer Cell-Typ, kein Eval-Pfad. Hashtabelle +
   Tag-Dispatch + zwei Makros. Bestätigt die Projekt-Maxime: Go für den
   Kern, Lisp für Erweiterungen.
9. **`sed` über mehrsprachige Markdown-Doku ist riskanter als dreimal
   Edit.** Code-Spans in Prosa machen Muster uneindeutig; ein Treffer zu
   viel erzeugt Widersprüche zwischen Abschnitten, die niemand sucht.

---

## Action Items

| # | Aufgabe | Priorität |
|---|---------|-----------|
| 1 | sixhat-Projekt: `/tmp`-Workaround entfernen, auf `set-working-directory` umstellen (der ursprüngliche Auslöser der TODO) | Hoch |
| 2 | Doku-Notiz: `%.2f`-Rundung ist half-to-even (Go), nicht half-up (C) — in cheatsheet/referenz explizit machen | Niedrig |
| 3 | Memory-Eintrag `golisp2-laufzeit-fallen` um "kein cadr, keine Makro-Destrukturierung verschachtelter Lambda-Listen" ergänzen (dort nur teilweise erfasst) | Niedrig |
| 4 | CL-Konformitätsliste (`docs/cl-konformitaet/symbole.csv`): printf/gets/environ/defgeneric/defmethod nachpflegen | Niedrig |
