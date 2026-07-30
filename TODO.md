# TODO

## Erledigt 2026-07-25 — GOLISP2 Semantik/Syntax-Kurzform (Option B: zwei Dateien)

**Ziel:** Wenn wir andere KIs einsetzen wollen, brauchen diese einen Überblick über die Semantik und Syntax von GOLISP2, aber auch Erläuterungen zu den Schwächen.

**Motivation:** Wenn GOLISP2 erst analysiert werden muß, wird ein Vorgang häufig wiederholt, was nicht notwendig sein sollte.

**Ergebnis (Option B + eigener Schwächen-Abschnitt):**
- `doc/ki/referenz.md` (316 Zeilen) — KI-Form: Tabellen, Präfixe, tokenoptimiert
- `doc/golisp2-cheatsheet.md` (771 Zeilen) — Mensch-Form: Beispiele, Erklärungen
- Beide haben §10 "Schwächen (bewusst)" als eigenen Abschnitt (12 Punkte:
  kein Package, kein CLOS, nur Condition-lite, progv lex/dyn, kein Compile-File,
  kein Small-Int-Cache, macrolet nicht-rekursiv, kein Continuations/MOP,
  load-in-defun, kein GC-Tuning, kein Typ-System, kein LOOP)

## Nächste Aufgaben

### Erledigt 2026-07-30 — Mini-Test-Framework (Nachfolger von `assert=`)

`tests/test-framework.lisp`: `defsuite`, `deftest` (`:suite`,
`:expected-failure`), `is` (sammelt statt abzubrechen), `run-tests`
(Report PASS/FAIL/XFAIL/XPASS, liefert FAIL-Anzahl). Neu: `(exit n)`-Primitiv
→ `(exit (run-tests))` macht `golisp2 -t` CI-tauglich (Exit = FAILs).
Migriert: `stdlib-test.lisp`, `defsystem-tests.lisp`, `pn-gps1/gps2-tests.lisp`
(71 Checks grün). `assert=` bleibt in `test-helpers.lisp` für Altdateien.
Fallstrick dokumentiert: `define`/`defstruct` im deftest-Rumpf bindet lokal —
globale Definitionen via `(eval '(...))` (wie load-in-defun).

### Erledigt 2026-07-30 — Condition-lite (Fehler mit Kontext)

`embed/condition.lisp` (via `LoadStdlib` geladen): `define-condition`
(Typ + Eltern + Slots, Reader automatisch als `typ-slot`), `signal`
(Keyword-Slots, unwindet immer — bewusste CL-Abweichung), `handler-case`
(Vererbungs-Dispatch, erste passende Klausel, kein Match → Re-Signal,
Klausel-Var darf `()` sein). Go-Fehler werden in `handler-case` zur
`lisp-error`-Condition mit `lisp-error-msg`-Reader. Neudefinition ersetzt
still (Reload). Bewusst nicht: Restarts, `handler-bind`, MOP.
Tests: `tests/condition-tests.lisp` (23 Checks, Suite `condition`),
in `-t` eingehängt → 94 PASS, 0 FAIL. Doku: referenz de/en/cn §7 + 10.3,
cheatsheet §10.3.

## Hinweise von Opus — erledigt 2026-07-30

- **10.6 gefixt + erweitert:** Überschrift war doppelt falsch — Small-Int-Cache
  **existiert** (-32768..32767, `MakeNum` in `lib/types.go`), aber `fnEqPtr`
  behandelt Zahlen bewusst nie als identisch. Neuer Abschnitt: „`eq` auf
  Zahlen liefert immer `()`". Auch §5/§9-Begründung „jede Zahl neue Cell"
  korrigiert (referenz.md + cheatsheet.md).
- **10.8 umformuliert:** „`catch`/`throw` vorhanden, aber ohne Restart-Semantik."
- **Zählerei:** 56 Spezialformen + 2 Stdlib-Makros — dann 55, weil:
- **mapcar-Frage beantwortet durch Umbau:** `mapcar` war Spezialform ohne
  Grund (evaluierte beide Args normal — Spezialform-Preis ohne Gegenleistung).
  Jetzt Primitiv (`fnMapcar` in `primitives.go`) → first-class:
  `(funcall mapcar ...)`, `(apply mapcar ...)`, als Wert übergebbar.
  CL-konform. Aufrufsyntax unverändert, kein Breaking Change.
- **Übersetzungen:** `doc/ki/referenz_en.md` + `doc/ki/referenz_cn.md`
  erstellt (gef-fixte Fassung als Basis, alle drei verlinkt).
