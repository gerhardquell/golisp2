# defsystem-lite — Design

**Datum:** 2026-07-25 · **Status:** genehmigt (Brainstorming mit Gerhard)
**Autor:** Gerhard Quell · **CoAutor:** kimi-k3

## Ziel

Deklarative Systemdefinition + dependency-geordnetes, idempotentes Laden als
Fundament eines eigenen GoLisp2-Ökosystems. Orientierung an ASDF-Konzepten,
**keine** ASDF-Kompatibilität (kein Compile-File, kein CLOS, keine Packages).

## Entscheidungen (aus Brainstorming)

1. **Platzierung:** eigene Embed-Datei `embed/defsystem.lisp`, von `LoadStdlib` mitgeladen.
2. **Fehler beim Laden:** Abbruch, Teilzustand bleibt stehen; erneuter Aufruf setzt an Fehlerstelle fort.
3. **Go-Zugriff auf DefLoc:** genau ein neues Primitiv `(defined-in "pfad")`.

## Architektur

Reines Lisp (~150–200 Zeilen) auf vorhandener Infrastruktur:
DefLoc (`lib/defloc.go`), `makunbound`, Redef-Log, `load`, `get-file-path`.
Einziger Go-Zuwachs: `defined-in`.

## Datenstrukturen

Drei Root-Variablen, reine Daten:

```lisp
(define *systems* '())         ; Alist: (name . (:depends-on (...) :components (...)))
(define *loaded-files* '())    ; normalisierte Pfade (via get-file-path)
(define *loaded-systems* '())  ; Namen explizit via load-system geladener Systeme
```

**Nachtrag 2026-07-25 (Implementierung Task 6):** Ursprünglich sollte der
System-Status rein aus `*loaded-files*` berechnet werden. Das ist ein Defekt:
beim Shared-File-Fall gilt ein entladenes System weiter als geladen (seine
Datei steht ja noch drin) — jeder unload des anderen Systems sieht die Datei
als shared und überspringt sie. Shared Files wären **niemals** entladbar
(Deadlock). Deshalb: `*loaded-systems*` führt den System-Lifecycle explizit
(load-system trägt ein, unload-system trägt aus). Getrennte Verantwortung:
`*loaded-files*` = Datei-Idempotenz, `*loaded-systems*` = System-Status.
Gerhard genehmigt 2026-07-25.

## API

### defsystem (Makro)

```lisp
(defsystem gps2
  :depends-on (test-helpers)
  :components ("pn-gps1/gps2.lisp"))
```

- Name unquotiert; `:depends-on` optional (Default `'()`).
- Registry-Update in `*systems*`; Neudefinition gleichen Namens ersetzt still
  (analog Reload-Semantik).
- Validierung **zur Expansionszeit**: nur `:depends-on` + `:components`;
  Deps = Symbolliste, Components = Stringliste. Verstoß → `error` bei Expansion.

### load-system

```lisp
(load-system 'gps2)
```

1. **Topo-Sort** (DFS über `:depends-on`, Deps zuerst).
2. **Zyklus-Erkennung** (graue Markierung): `(error "defsystem: Abhängigkeitszyklus: a -> b -> a")`.
3. Unbekannte Dep → `(error "defsystem: unbekanntes System 'xyz'")`.
4. Pro System in Topo-Reihenfolge, pro Komponente: Pfad via `get-file-path`
   normalisieren; wenn nicht `member` in `*loaded-files*` → `(load ...)` + pushen,
   sonst skip (idempotent).
5. Fehler in Komponente → Abbruch, Teilzustand bleibt.

Idempotenz auf **Datei-Ebene** (Shared Files zwischen Systemen), nicht System-Ebene.

**Limitation (dokumentiert):** zwei Schreibweisen derselben Datei
(`./x.lisp` vs `x.lisp`) gelten als verschiedene Einträge.

### Introspection & unload

```lisp
(loaded-systems)        ; → (gps2 test-helpers ...)   — berechnet
(system-symbols 'gps2)  ; → (gps gps-op ...)          — mapcan defined-in
(unload-system 'gps2)   ; → Liste entfernter Symbole
```

- `loaded-systems`: Mitgliedschaft in `*loaded-systems*` **und** alle
  Komponenten in `*loaded-files*` (defensive Doppelprüfung).
- `system-symbols`: `mapcan` von `defined-in` über normalisierte Komponenten-Pfade.
- `unload-system`: entfernt den Systemnamen zuerst aus `*loaded-systems*`,
  dann pro Komponente nur dann `makunbound` aller `defined-in`-Symbole
  + Streichen aus `*loaded-files*`, wenn **kein anderes geladenes System** die
  Datei mitlistet (Shared Files bleiben unangetastet). Deps werden **nicht**
  mit-entladen. Nicht-geladenes System → no-op, leere Liste.
- `makunbound` loggt ins Redef-Log → Nachvollziehbarkeit gratis.
- Fehler während `unload-system`: analog load-system bleibt der Teilzustand
  stehen. `*loaded-systems*` wurde bereits angepasst (zwingend vor der
  Shared-Prüfung, sonst Deadlock) und wird **nicht** zurückgerollt — der
  Aufrufer bewertet den Zustand neu.

## Fehlerfälle

| Situation | Verhalten |
|---|---|
| `load-system` unbekanntes System | `error` mit Name |
| Dep-Zyklus | `error` mit Zykluspfad |
| Dep zeigt auf undefiniertes System | `error` mit Name |
| Komponente nicht auffindbar | `load`-Fehler durchgereicht, Teilzustand bleibt |
| `defsystem` unbekanntes Keyword | `error` |
| `system-symbols`/`unload-system` unbekannt | `error` |
| `unload-system` nicht-geladen | no-op, leere Liste |

## Go-Teil

`(defined-in "pfad")` in `lib/defloc.go` (~40 Zeilen):

- Argument durch denselben Normalisierer wie `load` (`resolvePath` +
  `filepath.Abs`) → Registry-Scan (`LookupDefinition`-Pendants) → **sortierte**
  Symbolliste (deterministisch, diffbar).
- Registrierung in `BaseEnv()` (`lib/primitives.go`).
- Go-Test in `lib/defloc_test.go`: Treffer, Nicht-Treffer, Abs-Normalisierung.

Hintergrund: `load` speichert `SrcFile` als absoluten Pfad (`filepath.Abs` in
`lib/eval_load.go`), deshalb muss `defined-in` identisch normalisieren.

## Dateien

| Datei | Änderung |
|---|---|
| `embed/defsystem.lisp` | neu, ~150–200 Zeilen, deutsch, Datei-Header |
| `embed/assets.go` | `//go:embed defsystem.lisp` + `var Defsystem string` |
| `lib/stdlib.go` | `LoadStdlib` lädt Stdlib, dann Defsystem |
| `lib/defloc.go` | Primitiv `defined-in` |
| `lib/primitives.go` | Registrierung in `BaseEnv()` |
| `lib/defloc_test.go` | Go-Test |
| `tests/defsystem-tests.lisp` | Lisp-Suite (nutzt `assert=` aus test-helpers) |
| `tests/fixtures/` | Mini-Systeme: Kette a→b→c, Shared-File, Zyklus |

## Verifikation

`./build.sh` · `go test ./...` · `./build/golisp2 -t` · REPL-Smoke-Test.

## Bewusst nicht (YAGNI)

- Quicklisp-Analogon, ASDF-Kompat, Compile-File
- `:redefine :allow`-Hülle (TODO-Option, erst bei konkretem Bedarf)
- `:force`-Flag für load-system
- Entladen von Deps, Restore vorheriger Bindungen
