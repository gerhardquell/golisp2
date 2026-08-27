# Repo-Restructuring — Design

**Datum:** 2026-08-27 · **Status:** genehmigt (Brainstorming mit Gerhard)
**Autor:** Gerhard Quell · **CoAutor:** claude-sonnet-5

## Ziel

`TODO.md` (20260827-1) fordert vier zusammenhängende Änderungen am
Repo-Layout: Cleanup ungenutzter Verzeichnisse, Merge `doc/`+`docs/` →
`docs/`, ein `src/`-Verzeichnis für den Go-Code, sowie eine Referenz-Doku
(Mensch + KI) für alle golisp2-Funktionen. Dieses Dokument fixiert die
Reihenfolge, die betroffenen Dateien und die Auflösung eines Konflikts mit
einem bestehenden CLAUDE.md-Prinzip.

## Ausgangslage (Rechercheergebnis)

Das Repo ist größer als die `CLAUDE.md`-Orientierung beschreibt. Root enthält
zusätzlich `chinese/`, `experiment/`, `extern/`, `images/`, `libs/`,
`pn-gps1/`, `public/`, `tools/`, `tutorial/`, `golisp2web/` (eigenes
verschachteltes Git-Repo, siehe Memory `golisp2web-v1-grundgeruest` — bleibt
in diesem Umbau unangetastet).

`doc/ki/referenz.md` existiert bereits — eine manuell gepflegte
KI-Kurzreferenz, Stand 20260730 laut Datei-Header. Deckt sich mit dem
TODO-Wunsch nach "kompakter Dokumentation für KIs", ist aber bereits vier
Wochen alt und wird durch dieses Vorhaben ersetzt statt neu erfunden.

## Entscheidungen (aus Brainstorming)

1. **Cleanup-Kandidaten** (einzeln geprüft auf Referenzen, siehe Tabelle
   unten) wandern nach `unused/`. `chinese/` (von README.md verlinkt) und
   `extern/sigoREST` (lebender Symlink auf Schwester-Repo
   `/u/go-projekte/sigoREST`, keine tote Datei) bleiben unangetastet.
2. **`src/`-Scope:** `main.go`, `main_test.go`, `cmd/`, `lib/`, `embed/`
   wandern nach `src/`. Modulname in `go.mod` bleibt `golisp2`, nur der
   Importpfad ändert sich von `golisp2/lib` auf `golisp2/src/lib`.
3. **Doku-Konflikt-Auflösung:** CLAUDE.md verbietet bewusst eine
   handgepflegte Primitivenliste ("wäre ab dem nächsten `RegisterXxx()`
   falsch"). Auflösung: ein Generator-Lisp-Skript liest `(env-symbols)`
   zur Laufzeit und erzeugt die Referenz automatisch — Vollständigkeit ist
   damit strukturell garantiert, das CLAUDE.md-Prinzip bleibt gültig (die
   Doku *ist* der Code, nur aufbereitet).

## Zielstruktur

```
src/            main.go, main_test.go, cmd/, lib/ (84 Dateien + swank/), embed/
docs/           Merge von doc/+docs/, doc/ki/ → docs/ki/
unused/         experiment/, images/, libs/, tools/gen-training/, doc/files.zip
Root bleibt:    CLAUDE.md, README.md, README_CN.md, BESCHREIBUNG.md, LICENSE,
                TODO.md, go.mod, go.sum, build.sh, zeitstempel.txt, build/,
                examples/, tests/, todos/, tutorial/, public/, chinese/,
                pn-gps1/, extern/ (Symlink), golisp2web/ (fremdes Git-Repo)
```

### Cleanup-Tabelle (Begründung)

| Verzeichnis/Datei | Referenzstatus | Ziel |
|---|---|---|
| `chinese/` | von README.md Z.578ff verlinkt | bleibt |
| `extern/sigoREST` | Symlink, kein Text-Ref, aber lebender Pointer | bleibt |
| `experiment/` | keine Referenz | `unused/` |
| `images/` (1,4M) | keine Referenz mehr in README* | `unused/` |
| `libs/` | nur in einer alten Retrospektive erwähnt, Duplikat-Verdacht zu `lib/` | `unused/` |
| `pn-gps1/` | **Korrektur (Task-1-Review, 2026-08-27):** ursprünglich fälschlich als "kein Bezug zum golisp2-Kern" eingestuft. Tatsächlich referenziert von `main.go:103,105` (`-t`-Testsuite) und `lib/swank/gps_bug_test.go` (`TestSwankSurvivesNorvigBugs`). Gerhards Ruling: bleibt am Root. | bleibt |
| `tools/gen-training/` | nur in Retrospektive erwähnt | `unused/` |
| `doc/files.zip` | Backup-Zip identischer Docs vom 13.07., redundant | `unused/` |

## Schritte (je Schritt ein eigener Commit)

### 1. Cleanup

`git mv` der acht Kandidaten aus obiger Tabelle nach `unused/<name>`.
Bereits im Working Tree begonnene Verschiebung (`PerfTODO.md` →
`todos/PerfTODO.md`, im `git status` als `D`/`??` sichtbar) wird mit diesem
Commit sauber nachgezogen (`git add`).

### 2. Doc-Merge

`doc/*` → `docs/*` (git mv, Datei für Datei; keine Namenskollisionen
zwischen den beiden Verzeichnissen). `doc/ki/` → `docs/ki/`. Danach:

- `CLAUDE.md`: Orientierung-Tabelle und "Weiterführende Doku"-Tabelle am
  Ende auf `docs/...`-Pfade korrigieren.
- Alle `.md`-Dateien mit `doc/`-Querverweisen (u. a.
  `docs/retrospectives/*.md`, `tests/conformance/README.md`) auf `docs/`
  umstellen, soweit sie golisp2-eigene Pfade referenzieren (Zitate
  fremder/historischer Inhalte in `docs/gespraeche/` bleiben unangetastet —
  das sind archivierte Konversationen, keine aktiven Verweise).
- `doc/` wird nach dem Merge entfernt (leer).

### 3. Src-Umzug

`git mv main.go main_test.go cmd embed lib src/`. Danach:

- Importpfad `"golisp2/lib"` → `"golisp2/src/lib"` in allen 15 betroffenen
  `.go`-Dateien (main.go, cmd/golisp2-client/main.go, `lib/swank/*.go` — die
  Swank-Pakete importieren das Elternpaket für Typen).
- `"golisp2/lib/swank"` → `"golisp2/src/lib/swank"` (main.go).
- `build.sh`: Pfad zu `main.go` auf `src/main.go` anpassen (Build-Befehle
  selbst bleiben sonst gleich, `go build` löst Pakete über `go.mod` auf).
- `go:embed`-Direktiven in `embed/assets.go` sind relativ zum Paketverzeichnis
  und wandern mit `embed/` nach `src/embed/` — Verhältnis bleibt erhalten,
  wird in Schritt 5 verifiziert statt vorab angepasst.
- CLAUDE.md-Orientierungsblock (Dateibaum ganz oben) auf `src/`-Präfix
  korrigieren.

### 4. Doku-Generator

Neues Lisp-Skript `tools/gen-reference.lisp` (Name vorläufig, finaler Ort:
`tools/` bleibt bestehen als Heimat für Repo-interne Hilfsskripte):

- Nutzt `(env-symbols)` zur Laufzeit, klassifiziert jedes Symbol nach Typ
  (FUNC/LAMBDA/MACRO — Spezialformen kommen separat aus der Liste in
  `eval_core.go`, analog zum bestehenden Muster in
  `TestNoLispDefineShadowsSpecialForm`).
- Erzeugt zwei Artefakte:
  - `docs/referenz-generiert.md` — vollständige Tabelle (Name, Typ,
    Herkunftsdatei via `defined-in`/Go-Quelle) als Grundgerüst für Menschen.
    Einträge ohne Kurzbeschreibung sind sichtbar markiert (leere Spalte),
    kann von Hand ergänzt werden, ohne dass etwas unbemerkt fehlen kann.
  - `docs/ki/referenz.md` — kompakte Tabellenform nach Vorbild der
    bestehenden (jetzt ersetzten) Handversion: Eval-Reihenfolge,
    Spezialformen-Tabelle, Primitiven-Kurzliste. Ersetzt die alte Datei
    komplett (Git-Historie bleibt über `git mv` + Edit erhalten, kein
    `git rm`).
- Kein automatischer Build-Hook (YAGNI) — Skript wird bei Bedarf manuell
  erneut ausgeführt, wenn die Referenz sichtbar veraltet.

### 5. Verifikation

`go build ./...` · `go test ./... -count=1` · `./build.sh` ·
`./build/golisp2 -t`. Bei Fehlern: Importpfad-Korrekturen aus Schritt 3
nacharbeiten, nicht Struktur zurückrollen.

## Bewusst nicht (YAGNI)

- Keine Umbenennung des Go-Modulnamens (`golisp2` bleibt `golisp2`, nur
  interne Pfade ändern sich).
- Kein CI-Hook, der den Doku-Generator automatisch bei jedem Commit läuft.
- Keine Neu-Indizierung von `.codegraph/` als Teil dieses Tasks — Index
  zeigt danach auf alte Pfade, Neuindizierung ist Gerhards Entscheidung.
- `golisp2web/` (eigenes Git-Repo), `extern/sigoREST` (Symlink), `chinese/`
  bleiben unangetastet.
- Keine inhaltliche Überarbeitung von `docs/gespraeche/` (archivierte
  Konversationen) über die reine Verzeichnis-Existenz hinaus.

## Verifikation der Spec

Alle fünf Schritte sind unabhängig testbar (`go build` nach Schritt 3,
`go test` nach Schritt 4/5) und jeder Schritt ist ein eigener, überschaubarer
Commit — Rollback pro Schritt ist möglich, ohne die anderen zu berühren.
