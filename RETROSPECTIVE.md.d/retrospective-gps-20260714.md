# Retrospektive: GPS-Port und Spracherweiterungen

**Datum:** 14. Juli 2026  
**Autor:** Gerhard Quell & kimi-k2.7-code  
**Feature:** Norvig GPS (PAIP Kap. 4) nach golisp2, inkl. Version 2 (State-Passing)

---

## Was wurde gebaut?

- Port von Peter Norvigs **GPS Version 1** (`pn-gps1/gps.lisp`) mit globaler Zustandsmutation.
- Vollständige Testsuite für Norvigs drei bekannte Version-1-Fehler (`pn-gps1/gps-norvig-bugs.lisp`).
- **GPS Version 2** (`pn-gps1/gps2.lisp`) mit explizitem State-Passing, `goal-stack`-Zykluserkennung und lexikalisch durchgereichten Operatoren.
- Begleitende Spracherweiterungen/Fixes in `embed/stdlib.lisp`:
  - `defstruct` kollisionsfrei und idempotent via `%make-struct`.
  - `setf` gibt den zugewiesenen Wert zurück und wertet ihn nur einmal aus.
  - `bound?` wertet sein Argument aus (wichtig für Makros).
- Panic-Abwehr im Go-Kern (`eval_core.go`, `eval_lambda.go`, `eval_specialforms.go`, `eval_control.go`, `primitives.go`) mit `recover()` an der Auswertungsgrenze.

---

## Was lief schief? (⚫ Schwarz)

| Problem | Ursache | Auswirkung |
|---------|---------|------------|
| `defstruct box (list nil)` brach | Slot-Name `list` shadowte das Primitiv im Konstruktor | Panic in `eval_core.go:215`, Prozess tot |
| `defstruct` nicht idempotent | Zweites Laden wich auf `make--pt` / `pt--x` aus | Alte Aufrufer verweisen auf tote Namen |
| `setf` lieferte Objekt statt Wert | Makro gab das aktualisierte Objekt zurück | CL-Inkompatibilität |
| `bound?` wertete Argument nicht aus | Makro sah immer die lokale Variable als gebunden | Kollisionserkennung in `defstruct` wirkte nie |
| GPS Version 1 hatte bekannte Fehler | Globale Mutation + fehlende finale Konsistenzprüfung | Clobbered Sibling Goal, Leaping before you look, rekursive Zyklen |

**Erkenntnis:** Der GPS-Port war lange "grün", weil keine Frage gestellt wurde, die die darunterliegenden Sprachlücken offengelegt hätte.

---

## Was haben wir gelernt? (🔵 Blau)

1. **Grün heißt nicht richtig.** Grün heißt: noch keine passende Frage gestellt. Ein Feature-Port muss aktiv auf bekannte Grenzfälle geprüft werden.
2. **Ein Feature-Port zieht Spracherweiterungen nach.** Der Anwendungsfall testet sie nur entlang *eines* Pfades. Neue stdlib-Funktion → eigener Test.
3. **Namensgenerierung braucht Kollisionsregeln *und* eine Warnung.** Eine Ausweichregel ohne Meldung tauscht einen lauten Fehler gegen einen leisen.
4. **"Unbekanntes Symbol" im REPL beweist nichts über Code in der Datei.** Erst laden, dann schließen.

---

## Action Items

| # | Aufgabe | Priorität |
|---|---------|-----------|
| 1 | Eigener Cell-Typ `LAMBDA`/`CLOSURE` statt LIST + optionales Env evaluieren | Mittel |
| 2 | Regressions-Unit-Test für `(define xs '(1 2 3)) (xs 0)` → Fehler, kein Absturz | Niedrig |
| 3 | `remove-if` / `some` / `subsetp` bei Bedarf in stdlib heben, nicht nur in gps2.lisp | Niedrig |
| 4 | GPS Version 2 mit goroutine-paralleler Suche erweitern (wenn gewünscht) | Niedrig |

---
