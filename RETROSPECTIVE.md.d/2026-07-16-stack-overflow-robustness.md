# Retrospektive: Stack-Overflow-Robustheit

**Datum:** 16. Juli 2026  
**Autor:** Gerhard Quell & claude-opus-4.8  
**Feature:** Evaluator gegen Stack-Overflow-Panics härten, `parfunc`-Cancellation, SWANK-Panic-Recovery

---

## Was wurde gebaut?

- `evalCtx` mit `depth int` und `context.Context` durch den gesamten Evaluator gezogen:
  `lib/eval_core.go`, `lib/eval_specialforms.go`, `lib/eval_control.go`,
  `lib/eval_lambda.go`, `lib/eval_quasiquote.go`, `lib/eval_exec.go`.
- Öffentliches Rekursionslimit `MaxEvalDepth` (Default 100000), thread-sicher via
  `GetMaxEvalDepth()` / `SetMaxEvalDepth()`.
- Periodischer Depth-/Cancellation-Check innerhalb des TCO-Trampolin-Loops,
  damit reine Tail-Endlosschleifen abbrechen.
- `parfunc` bricht Worker bei `:timeout N` über `context.WithCancel` ab.
- `freeEnv` toleriert `nil` (Regressionstest in `lib/env_test.go`).
- SWANK `handleConn` mit Panic-Recovery und sauberem `conn.Close()`.
- Integrationstest `lib/swank/gps_bug_test.go`: `golisp2d` überlebt das Laden von
  `pn-gps1/gps-norvig-bugs.lisp`.
- Dokumentation in `doc/lisp-semantik.md` erweitert.

---

## Was lief schief? (⚫ Schwarz)

| Problem | Ursache | Auswirkung |
|---------|---------|------------|
| `worktree-golisp2-stack-overflow-robustness` enthielt viele Features, die nicht zum Plan gehörten | Branch war vor dem Plan schon mit `build-output-dir`-Arbeit vorgebaut | Finaler Reviewer meldete Scope Creep; Merge umfasst weit mehr als das eigentliche Feature |
| `main` hatte keine Remote-Tracking-Informationen | Auf origin existierte nur `build-output-dir`, nicht `main` | `git pull` scheiterte; Merge wurde ohne aktuellen Remote-Stand durchgeführt |
| Lokale Änderungen in `main` blockierten Checkout | `CLAUDE.md`, `TODO.md`, `pn-gps1/gps-norvig-bugs.lisp` waren vor dem Merge schon modifiziert | Mussten vorher gestashed und danach wiederhergestellt werden |
| Task-11-Reviewer behauptete, `parfunc` würde nicht via Context cancellen | Reviewer las den Diff isoliert und übersah `context.WithCancel` in `lib/eval_control.go:268` | Zeit für Verifikation nötig; Finding war ein False Positive |
| Finaler Reviewer merkte: TCO-Loop prüft Depth/Cancellation nicht pro Iteration | Tail-Calls inkrementieren `depth` nicht; Loop hatte keinen periodischen Check | Reine Tail-Endlosschleife wäre nicht gestoppt worden; Nachbesserung nötig |

---

## Was haben wir gelernt? (🔵 Blau)

1. **Der Merge-Base zählt.** Wenn der Branch vor dem eigentlichen Feature schon
   umfangreiche Arbeit enthält, ist der finale Diff viel größer als der Plan.
   Lieber Feature-Branch vom aktuellen `main` abzweigen.
2. **TCO-Trampoline brauchen periodische Checks.** Tail-Rekursion vermeidet
   Go-Stackframes, heißt aber nicht, dass die Schleife ewig laufen darf.
3. **Globale Testvariablen brauchen atomaren Zugriff.** `MaxEvalDepth` wird in
   Tests temporär gesenkt; parallele `Eval`-Aufrufe erzeugen sonst Data-Races.
4. **Review-Findings erst verifizieren, dann fixen.** Ein formuliertes "müsste so
   sein" kann gegen den tatsächlichen Code falsch liegen.
5. **Stash statt Wegwerfen.** Vor einem Branch-Wechsel auf `main` lokale
   Änderungen mit eindeutigem Tag stashen, nicht verwerfen.

---

## Action Items

| # | Aufgabe | Priorität |
|---|---------|-----------|
| 1 | `main` auf origin als Default-Branch etablieren oder Workflow an `build-output-dir` anpassen | Hoch |
| 2 | Bei zukünftigen Features frisch von `main` abzweigen, nicht auf einem vorbeladenen Branch aufsetzen | Hoch |
| 3 | `go test -race` regelmäßig laufen lassen, um Race-Conditions früh zu sehen | Mittel |
| 4 | Review-Prozess: False-Positive-Findings explizit dokumentieren, bevor sie verworfen werden | Niedrig |

---
