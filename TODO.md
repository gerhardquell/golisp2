# Aufgabe: 20260813 (Fortsetzung)

**Status:** 1 + 2.1 erledigt (20260815) — Rest offen

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
**brainstorming**

- Einbau von stdin,stdout,stderr als
  sys-stdin, sys-stdout, sys-stderr
  die als Filenamen genutzt werden können. 
ich möchte folgendes machen können 
gets() ==> fgets(sys-stdin) oder
read() ==> fread(sys-stdin) oder
err-write ==> fwrite(sys-stderr)

### 2.3 Formatierte Ausgabe
**brainstorming**

Ich hätte gern Macros, die die C-Funktionen emulieren:
- printf, fprintf, sprintf
- scanf , fscanf, scanf

### 2.4. CLI
**brainstorming**

Da wir shebang einsetzen können, hätte ich gern  eine Funktion, mit der
ich die Kommandozeile und das Enviroment lesen kann.

### 2.5. CLOS-light
**brainstorming**

Lass uns über eine leichtgewichtige Objektorientierung nachdenken
**nur diskutieren**


