# TODO 20260818

**Status:** ERLEDIGT — 20260818

Bei der Überarbeitung des Buches LISP-Eine unendliche Geschichte zeigte
golisp2 einig Fehler und Probleme.

## Quellen:
 ./docs/lispbuch1.epub
 ./docs/golisp2-fehler.md
 ./docs/buch_resultate.md

## Aufgabenextraktion — ERLEDIGT

9 Punkte aus den Quellen extrahiert und mit Gerhard besprochen (Doku 5–9,
Code 1–4). Recherche ergab: #5 (ga-create-Signatur), #7 (pg-*-Primitive)
und #9 (System-/Nebenläufigkeits-Primitive) waren bereits in `doc/` aktuell
dokumentiert — CLAUDE.md dokumentiert bewusst keine Primitivenliste mehr.
#8 (`claude-h` veraltet in README/BESCHREIBUNG/Artikeln) auf Gerhards
Entscheidung hin zurückgestellt (nur Beispieltext, kein Bug).

## Umsetzen — ERLEDIGT

- **#6 Fehlermodell:** `doc/lisp-semantik.md` um Abschnitt „Fehlermodell:
  catch/throw vs. trap" ergänzt — bisher einzige echte Doku-Lücke.
- **#1 `defstruct`-Prädikat:** generiert jetzt zusätzlich `<name>-p` als
  CL-Alias zu `<name>?` (gleiche Kollisionsvermeidung wie Accessoren).
- **#3 `setf` Multi-Place:** `(setf a 1 b 2 …)` über `&rest` unterstützt.
- **#4 `ignore-errors`:** dünnes Makro über `trap`.
- Zurückgestellt (Design-Ausweitung, Gerhards Entscheidung bei Bedarf):
  `defstruct :include` (Vererbung), `setf` auf verschachtelten Places.
- Tests: `lib/stdlib_todo20260818_test.go` (TDD, 4 neue Testfälle).
  `go test ./... -count=1` → 328 Tests grün. `./build/golisp2 -t` →
  104 PASS, 0 FAIL.


