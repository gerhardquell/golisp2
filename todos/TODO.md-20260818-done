# Aufgabe: 20260813 (Fortsetzung)

**Status:** komplett erledigt (20260815) — 1, 2.1–2.5

Gefunden beim Live-Testen des sixhat-Projekts über SWANK/Emacs.

## 1. `file-write`/`file-append` respektieren `set-working-directories` nicht

**ERLEDIGT (20260815):** Gelöst über 2.1 — Plural-Funktionen entfernt,
`set-working-directory`/`get-working-directory` (Singular) eingeführt.
`file-write`/`file-append` nutzen neuen `resolveWritePath` (keine
Existenzprüfung, join mit working-directory), `file-delete` nutzt
`resolvePath` wie die Lese-Funktionen. Semantik-Änderung: gesetztes
working-directory schlägt das Prozess-cwd beim Lesen. Tests:
`lib/fileio_test.go:TestWorkingDirectory`.

**Beobachtung:** `file-read`, `file-exists?`, `get-file-path` und `load`
lösen relative Pfade über `resolvePath` auf (aktuelles Verz. →
`working-directories` → System-Suchpfade `/lib/golib` etc.). `file-write`
und `file-append` (`lib/fileio.go`) tun das NICHT — sie schreiben direkt
mit dem rohen, unveränderten Pfad gegen `os.WriteFile`/`os.OpenFile`.

**Auswirkung:** Ein Programm, das mit relativen Pfaden schreibt (z.B.
`(file-write "doc/bericht.md" ...)`), funktioniert nur, wenn der
golisp2-Prozess zufällig im richtigen Arbeitsverzeichnis läuft. Bei SWANK
startet Emacs/Sly den Prozess oft nicht im Projektverzeichnis —
`(set-working-directories "/pfad/zum/projekt")` vorher aufzurufen ändert
am Verhalten von `file-write` nichts, obwohl das naheliegend wirkt (`load`
und `file-read` funktionieren ja damit).

**Konkret aufgetreten:** sixhat-Projekt, `bericht-schreiben` ruft
`(file-write "doc/sitzung_..." ...)` relativ auf — schlägt fehl, wenn der
SWANK-Server nicht im sixhat-Verzeichnis gestartet wurde. Workaround
aktuell: absoluter `/tmp`-Pfad statt `doc/...` im sixhat-Code (nicht hier
gefixt, nur Symptom-Vermeidung).

**Zu klären (Design, kein trivialer Fix):** `resolvePath` für `file-read`
prüft bei jedem Suchpfad-Kandidaten `os.Stat`, ob die Datei DORT bereits
EXISTIERT — das passt für Lesen, aber nicht 1:1 für Schreiben einer NEUEN
Datei (die existiert per Definition noch nirgends). Für `file-write`
bräuchte es eher: Verzeichnis-Anteil des Pfads gegen
`working-directories`/aktuelles Verzeichnis auflösen (erstes existierendes
Verzeichnis gewinnt), Dateiname anhängen, dann schreiben — nicht die
ganze-Datei-existiert-Prüfung von `resolvePath` wiederverwenden. Eigene
Funktion (z.B. `resolveWriteDir`) vermutlich sauberer als `resolvePath`
zu verbiegen. Chokepoint-Frage: eine gemeinsame Basis für Lese- und
Schreib-Pfadauflösung, oder zwei bewusst getrennte Funktionen mit
unterschiedlicher Semantik?

---

## 2. Lösungsansatz Gerhard
**brainstorming**

### 2.1 xxx-working-directories
**ERLEDIGT (20260815):** `set/get-working-directories` entfernt,
`set/get-working-directory` eingeführt (siehe Punkt 1).
`cellToPathList`/`splitPathString` (toter Code) entfernt.
Doku angepasst: cheatsheet, referenz.md/en/cn, cl-inventar.md.

### 2.2 IO-Streams
**ERLEDIGT (20260815):** Pseudodateien `sys-stdin`/`sys-stdout`/`sys-stderr`
in `file-read`/`file-write`/`file-append`. Neue Primitiven:
`(gets)` (Zeile von stdin), `(slurp)` (stdin bis EOF), `(err-write ...)`.
Alle Streams laufen über die Chokepoints: gemeinsamer `stdinReader`
(geteilt mit `read-line` — `readLineFromStdin`/`slurpStdin` in
primitives.go), `WriteOutput`/`WriteError` (SWANK-sichtbar).
Tests: `lib/fileio_test.go:TestSysStreams`.

### 2.3 Formatierte Ausgabe
**ERLEDIGT (20260815):** `lib/cformat.go` — `printf`/`sprintf`/`fprintf`/
`sscanf` als Primitiven. C-Formatstring wird nach CL-format übersetzt
(`cformatToCL`), Ausgabe läuft über die FORMAT-Engine (eine Quelle).
Kern-Set: `%d %i %s %f %e %g %x %o %c %%`, Flags `- 0 +`, width,
.precision. Alles rune-basiert (Unicode-sicher, `%c` = 1 Rune).
Lücken (laut, nicht still): `%.Ns`, linksbündige Floats → Fehler.
Tests: `lib/cformat_test.go`.

### 2.4. CLI
**ERLEDIGT (20260815):** `lib/sysinfo.go` — `(argv)` (rohe os.Args als
String-Liste), `(getenv "NAME")` (String oder `()` wenn unset, leer ≠
unset via LookupEnv), `(environ)` (Alist). Registriert via
`RegisterSysinfo` in `BaseEnv`. Tests: `lib/sysinfo_test.go`.

### 2.5. CLOS-light
**ERLEDIGT (20260815):** `defgeneric`/`defmethod` in `embed/stdlib.lisp`
(rein Lisp, kein Kernel-Eingriff). Single-Dispatch auf Struct-Tag,
`t` = Default-Methode, Extra-Parameter möglich, Hot-Redefinition via
Registry-Hashtabelle `%generic-registry`. Explizit nicht dabei:
Vererbung, `call-next-method`, `:before`/`:after`, Multi-Dispatch.
Tests: `tests/stdlib-test.lisp` Suite `clos-light`.
Doku: `doc/lisp-semantik.md` Abschnitt "Generische Funktionen".


