# TODO 20260818b — Lispbuch-Lückenanalyse (Gruppe A+B)

**Status:** ERLEDIGT — 20260818

Fortsetzung der Buchüberarbeitung. Gerhard verwies auf das eigenständige
Buchprojekt `/u/golisp2-projekte/lispbuch/src/chapters/` (27 Kapitel,
`chapt-0001.md` … `chapt-0027.md`, golisp2-adaptierte Fassung von
„LISP – Eine unendliche Geschichte"). Dort wird wiederholt festgehalten,
dass CL/CLISP-Features in golisp2 fehlen oder erst gebaut werden müssen.
Ziel: eine Standardbibliotheks-Erweiterung, die diese Lücken schließt.

## Analyse — ERLEDIGT

Fork durchsuchte alle 27 Kapitel nach „fehlt/muss noch gebaut werden"-
Aussagen und prüfte jede gegen den aktuellen golisp2-Stand (Code +
`doc/`). Ergebnis: 5 Buch-Aussagen waren veraltet (`push`/`pop`,
`dotimes`, `intern`, `puthash`/`gethash`, `macroexpand` existieren
längst). Rest in zwei Gruppen sortiert — reine stdlib-Ergänzungen
(Gruppe A) vs. Go-Primitiven-Kandidaten (Gruppe B).

## Gruppe A — ERLEDIGT (Commit 3b3c1cd)

`embed/stdlib.lisp`: `remove-if-not`, `remove-if`, `remove`,
`remove-duplicates`, `butlast`, `copy-list`, `copy-tree`, `make-list`,
`getf`, `zerop`, `incf`/`decf`, `eql` (wertbasiert bei Zahlen, sonst
`eq`-Semantik — nicht `equal?`), `assert`, `macroexpand-1` (Alias),
`coerce` (list↔string), `string-find`, `destructuring-bind` (nur
flaches Pattern — verschachtelt hat dieselbe Symbol-Rebind-Grenze wie
nested-setf). Tests: `lib/stdlib_todo20260818b_test.go`.

## Gruppe B — ERLEDIGT (Commit f36922d)

Neue Datei `lib/clcompat_prims.go`, Registrierung in `BaseEnv()`:
- `sort` — `(sort liste pred &key key)`, nicht-destruktiv, `pred` als
  Lisp-Callback (Muster wie `ga-create`-Fitness-Fn)
- `sqrt` — `math.Sqrt`; negative Zahl → Fehler (keine komplexen Zahlen)
- `get-universal-time` — Sekunden seit CL-Epoche 1900-01-01 UTC
  (CLHS 25.1.4), nicht Unix-Epoche

Tests: `lib/clcompat_prims_test.go`.

## Zurückgestellt (Gerhards Entscheidung bei Bedarf)

`defstruct :include` (Vererbung), `setf` auf verschachtelten Places,
`claude-h`-Bereinigung in User-Docs (README/BESCHREIBUNG/Artikel).

## Ergebnis

`go test ./... -count=1` → 338 Tests grün. `./build/golisp2 -t` →
104 PASS, 0 FAIL.
