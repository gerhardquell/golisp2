# Retrospektive: Redefinition-Guard, Redef-Log und makunbound

**Datum:** 24. Juli 2026  
**Autor:** Gerhard Quell & kimi-k3  
**Feature:** A+B+C aus TODO 20260724 (das „define-Problem"): kontextbewusster
Redefinition-Guard, beobachtbares Redef-Log, `makunbound` — dazu zwei
Alt-Bug-Fixes (`runStdin` Silent-Drop, `break` statt `continue`) und die
Zentralisierung von `assert=` (Commits d14d18b..a7e0c75, 893a260)

---

## Was wurde gebaut?

- **Kontext-Guard** (`lib/redefguard.go: checkRootRedefine`): wird in
  `define`/`defun`/`defmacro` **vor** `env.Set` gerufen. Alt-Bindung
  LAMBDA/MACRO + gleiche Quelle (`DefLoc.File`) → stiller Reload (der
  normale Entwicklungs-Workflow bleibt warnfrei); fremde Quelle → Policy
  `allow|warn|error`. FUNC (Go-Primitiven) bleibt beim bestehenden
  `Env.Set`-Hook als Backstop — so sind auch `setq`/`progv`-Pfade
  abgedeckt, ohne dass sich die beiden Guards doppelt melden.
- **Redef-Log** (`lib/redeflog.go`): Ringpuffer (256 Events) aller
  Root-Redefinitionen inkl. stiller Reloads, Lisp-Zugang via
  `(redef-log)` / `(redef-log-clear)`. Beobachtbarkeit statt Verbot:
  der selbsterweiternde Pfad `(eval (read (sigo …)))` bleibt gewollt
  erlaubt, ist jetzt aber auditierbar.
- **`makunbound`** als Spezialform (Primitiven bekommen kein `env`):
  entfernt Root-Bindung + DefLoc-Eintrag, Policy greift auf Callables,
  ungebundenes Symbol ist ein lauter Fehler.
- **Policy einmal pro Guard-Aufruf** (`currentPolicy()` + Parameter
  statt doppeltem Atomic-Read) — Fix aus der Final-Review.
- **Beifang:** `runStdin` verwarf bisher still alle Forms außer der
  ersten pro Pipe-Batch (`lib.Read` → `lib.ReadAll`); `assert=`
  zentralisiert nach `tests/test-helpers.lisp`; gps2-Load explizit in
  `(redefine-policy 'allow)` gehüllt.
- **Prozess:** Plan (writing-plans) → Subagent-Driven Development
  (6 Tasks à Implementer + Reviewer, haiku/sonnet) → Final-Review
  (opus) → Merge. TODO.md-Aufgabe abgehakt und archiviert.

Endstand: **212/212 Go-Tests, Testsuite 0 REDEF (vorher 4), 0 FAIL,
39 PASS.**

---

## Was lief schief? (⚫ Schwarz)

| Problem | Ursache | Auswirkung |
|---------|---------|------------|
| Plan-Tests für Cross-File-Warnungen funktionierten nicht wie geschrieben | `evalStr` baut pro Aufruf ein frisches `BaseEnv` — der `RegisterDefinition`-Trick aus dem Plan überlebte den zweiten Aufruf nicht | Task-3-Implementer musste `evalInEnv`-Helfer einführen (abweichend vom Brief, vom Review verifiziert). Dieselbe Bug-Klasse war im Plan bei makunbound schon aufgefallen — und trotzdem in Task 3 wieder drin |
| Fix-Subagent committed das **gesamte** Arbeitsverzeichnis | `git add` ohne explizite Dateiliste — unzusammenhängende Dateien (zeitstempel.txt, epub-Binary!) landeten im Fix-Commit | Commit 999f6b8 musste verworfen und als 50aa111 sauber neu gebaut werden |
| `break` statt `continue` in der neuen runStdin-Forms-Schleife | Der Task-6-Implementer fixte den Silent-Drop, änderte dabei aber das dokumentierte Fehlerverhalten (`doc/cli.md`: „weitere Expressions werden trotzdem verarbeitet") | Task-Review fing es; ohne den Doku-Abgleich wäre es durchgegangen |
| Policy-Atomic zweimal gelesen (Log-Action ≠ Enforcement) | `policyAction()` und `applyRedefPolicy()` lasen `redefinePolicyAtomic` unabhängig | Unter parfunc könnte das Log „warn" zeigen, während „error" blockierte — Final-Review (opus) fing es, Task-Reviews (sonnet) nicht |
| `runStdin` Silent-Drop unentdeckt seit Bestehen | `lib.Read` las nur die erste Form pro Batch | Erst der Rauchtest des Plans (`echo '(a)(b)(c)' \| golisp2`) machte ihn sichtbar — ein Feature-Plan als Metalldetektor für einen Alt-Bug |
| Testsuite warnte 4× REDEF gegen sich selbst | `assert=` identisch in zwei Testdateien; gps2 ersetzt gps-v1 ohne Markierung | Der Guard validierte sich am eigenen Projekt sofort selbst — aber die Koexistenz von gps-v1/v2 im selben Env war vorher unsichtbar |

---

## Was haben wir gelernt? (🔵 Blau)

1. **Kontext schlägt Strenge.** Der naive Guard (jede Redefinition
   warnen) wäre am ersten Tag wegkonfiguriert worden — `defun`-Reload
   ist Arbeit, kein Angriff. Erst `DefLoc.File` als Kontext macht die
   Unterscheidung „Entwicklung" vs. „fremde Quelle" möglich. Dieselbe
   Lektion wie „Ausweichregel ohne Meldung ist der schlechtere Handel":
   Mechanismen brauchen Semantik, nicht nur Schwellen.
2. **Beobachtbarkeit statt Verbot für gewollte Muster.** Der
   sigo-Pfad `(eval (read …))` soll ins globale Env schreiben dürfen —
   das Redef-Log macht ihn nachvollziehbar, ohne ihn zu bremsen.
   Ringpuffer + `(redef-log)` sind 110 Zeilen Go für dauerhafte
   Auditierbarkeit.
3. **Primitiven haben kein `env` — das entscheidet Spezialform vs.
   Primitiv.** `makunbound` sah aus wie ein Primitiv (CL: Funktion),
   musste aber Spezialform werden (wie `bound?`). Die Signatur
   `func([]*Cell)` ist die harte Grenze; `rg 'func makeFn'` vor dem
   Design spart den Umweg.
4. **`evalStr`-Fresh-Env ist eine Testdesign-Falle.** Zwei `evalStr`-
   Aufrufe teilen **nichts** außer Package-Globals (DefLoc, Redef-Log).
   Tests, die Bindungen über Aufrufe hinweg brauchen, gehören in einen
   String oder einen expliziten Env-Helfer. Beim Plan-Schreiben gilt:
   Test-Helfer-Semantik zuerst prüfen, dann Testcode.
5. **Der beste Zeitpunkt für einen Guard ist vor dem ersten
   Schaden.** Die 4 REDEF-Warnungen fielen sofort auf reale Duplikate
   (`assert=`-Kopie) und eine unmarkierte Versions-Koexistenz (gps v1/v2
   mit verschiedener Arity). Das Werkzeug bezahlte sich in der ersten
   Stunde selbst.
6. **Review-Tiefe skaliert mit Modell — und das ist ok.** Task-Reviews
   (sonnet) fingen Spec- und Doku-Verstöße (`break`/`continue`), der
   subtile Atomic-Race fiel erst der Final-Review (opus) auf. Genau so
   ist die Modellleiter gedacht: billig für Konformität, stark für
   Architektur.
7. **Fix-Subagents brauchen eine explizite Commit-Whitelist.**
   „Commite den Fix" heißt bei dreckigem Arbeitsverzeichnis „commite
   alles". Ab jetzt steht in jedem Fix-Dispatch: `git add <exakte
   Dateien>`, nichts anderes. (Steht bereits so in der
   SDD-Grundausrüstung — dieser Fall zeigt warum.)
8. **Ein guter Rauchtest ist ein Alt-Bug-Detektor.** Der Plan-Rauchtest
   `echo '(a)(b)(c)' | golisp2` war als Feature-Demo gedacht und
   entlarvte den stdin-Silent-Drop. Rauchtests, die *mehrere* Dinge
   auf einmal tun (mehrere Forms, Redefinition, Log-Abfrage), finden
   mehr als drei einzelne.

---

## Action Items

| # | Aufgabe | Priorität | Status |
|---|---------|-----------|--------|
| 1 | CLAUDE.md „kein `undefine`"-Märchen durch Ist-Zustand ersetzen | Hoch | ✅ erledigt (9ac5aa4) |
| 2 | Prüfen, ob gps-v1-Code nach gps2-Load im selben Env noch läuft (Arity-Bruch v1→v2 wäre laut, aber spät) | Mittel | offen |
| 3 | Detail-String `aus :0` bei quelllosen Symbolen hübschen („interaktiv" statt `:0`) | Niedrig | offen |
| 4 | swank-Env (`lib/swank/env.go`) an Guard/Log anschließen — aktuell bewusste Grenze | Niedrig | offen |
| 5 | Ringpuffer O(n)-Shift nur anfassen, wenn Profiler zeigt, dass 256er-Shifts wehtun (YAGNI) | Niedrig | bewusst zurückgestellt |

---
