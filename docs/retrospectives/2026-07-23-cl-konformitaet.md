# Retrospektive: CL-Konformität der Kernsymbole

**Datum:** 23. Juli 2026  
**Autor:** Gerhard Quell & kimi-k3  
**Feature:** Konformitäts-Suite gegen clisp als Goldstandard + 11 Meilensteine
neuer Kernsymbole (TODO.md Aufgabe 20260723, Schritte ①–④, Commit 7eea2d8)

---

## Was wurde gebaut?

- **Konformitäts-Suite** (`tests/conformance/`): Fall-Dateien (eine Form pro
  Zeile), clisp-Treiber evaluiert jede Form unter `handler-case`, `run.sh`
  fährt golisp2 pro Datei via stdin, normalisiert Ausgaben (Case, nil,
  unreadable objects) und vergleicht. Status: PASS / FAIL / XFAIL / XPASS;
  `known-failures.txt` hält akzeptierte Abweichungen als exakte Form-Texte.
  Bestand zu Beginn: 72 PASS — danach Meilenstein für Meilenstein ausgebaut.
- **Multiple Values als Cell-Typ**: `MVALUES` + `Primary()`-Chokepoint in
  `evalArgsPooled`. Alle MV-Konsumenten (`nth-value`, `multiple-value-*`)
  sind Spezialformen, nie FUNC — sonst würde `Primary()` den MV-Cell
  vorher auf seinen Primärwert strippen.
- **Hashtables** (`lib/hashtable.go`): eigener `HASHTABLE`-Typ, Go-Map +
  `sync.RWMutex` pro Tabelle, mutable Pointer-Identität (eq-fähig).
  Test-Modi eql/equal über Schlüsselfunktion; `maphash` iteriert über
  Snapshot, damit der Callback die Tabelle mutieren darf.
- **Kontrollfluss**: `catch`/`throw`, `tagbody`/`go`, `unwind-protect`,
  `do*` (sequentielle Init/Steps), `progv`, `eval-when`, `declare`
  (ignoriert), `locally`, `the`, `return`, `block nil` / `return-from nil`.
- **Lokale Makros**: `macrolet` (nicht-rekursiv, CL-konform) und
  `symbol-macrolet` über einen **Lazy-SYMMACRO-Marker** im Env statt
  Baum-Rewrite — Shadowing via Env-Kette ist gratis, makroexpandierte
  Bindungsformen (`dolist`) können es nicht kaputtmachen. `setq`/`psetq`/
  `set!` routen über `symMacroTarget` (nur Symbol-Expansionen zuweisbar).
- **setf-Places in der stdlib**: `(setf (gethash k t) v)` → `puthash`;
  `(setf (nth i xs) v)` → `set-nth` + `set!` (immutable Cells erzwingen
  Rebind), Index und Wert gensym-gebunden, je einmal evaluiert.
- **`&body` = `&rest`-Alias** in beiden Lambda-Listen-Parsern.
- **setq-Semantik auf CL gedreht**: `setq` sucht die Bindung entlang der
  Env-Kette und updatet sie (wie `set!`), statt im inneren Scope zu
  shadowen. swap-Makro mit `setq` im `let`-Body funktioniert jetzt.

Endstand: **PASS 199, FAIL 0, XFAIL 1** (Start: PASS 72, XFAIL 22).

---

## Was lief schief? (⚫ Schwarz)

| Problem | Ursache | Auswirkung |
|---------|---------|------------|
| setf-Makro zweimal mit falscher Klammerzahl committed (9 statt 10, dann 11 statt 10) | Quasiquote-Nesting-Tiefe beim Editieren nicht nachgezählt | `stdlib: reader: leeres Token` beim Start — die ganze Suite lief ins Leere und meldete nur „Zeilen-Alignment verloren (28 Formen, 1 Ausgaben)". Die Fehlermeldung zeigte auf den Driver, nicht auf die stdlib |
| `TestHashTableFehler` erwartete Fehler von `maphash` auf **leerer** Tabelle | Leere Tabelle → Callback feuert nie → kein Fehler | Test grün aus dem falschen Grund; erst mit nicht-leerer Tabelle echte Assertion |
| `TestEvalDoStar` mit falscher Erwartung geschrieben (4 statt 10) | Selbst gerechnet statt clisp gefragt: `s` akkumuliert das **neue** `i` (1+2+3+4), nicht das alte | failing test durch falsche Erwartung, nicht durch falschen Code — hätte fast zu einem „Fix" des korrekten Codes geführt |
| `TestEvalSymbolMacrolet` nutzte `defvar` in BaseEnv-only-Test | `defvar` ist ein stdlib-Makro, im Go-Test-Env nicht vorhanden | Dieselbe Bug-Klasse wie das ursprüngliche `nth-value`-Problem: Go-Tests sehen die stdlib nicht |
| eval-when-Case `(eval-when (execute) 5)` lieferte nil, für einen Bug gehalten | Kein Bug — clisp selbst liefert nil: `execute` ist **kein** gültiger old-style Name (nur compile/load/eval) | Erst das direkte Anproben von clisp (`(eval '(eval-when (:execute) 5))` → 5) klärte: der Case war falsch, nicht der Code |
| Einmal `/tmp` statt `./tmp` für XPASS-Zwischendatei benutzt | Gewohnheit | Gegen Projektregel; verschoben und aufgeräumt |
| Phantom: „progv besteht ohne Implementierung" | Zeilen-Alignment-Verwirrung zwischen direktem stdin-Lauf (13 Zeilen, Kommentarzeile ergibt `()`) und run.sh-Pipeline (12 Zeilen, Kommentare gestrippt) — plus grep-Filter, der die progv-Cases gar nicht einschloss | Zeit für Diagnose; harmlose Ursache, aber die Suite-Mechanik musste dafür komplett verstanden werden |

---

## Was haben wir gelernt? (🔵 Blau)

1. **Der Goldstandard ist auch ein Orakel.** Wenn Case und Code
   widersprechen, erst `clisp` direkt anproben, bevor man „repariert".
   Einmal hätte das fast korrekten Code an einen falschen Case angepasst
   (eval-when), einmal einen korrekten Code an eine falsche Erwartung
   (do*). **Das Orakel schlägt das eigene Kopfrechnen.**
2. **Nach jeder stdlib-Änderung smoke-testen:**
   `echo '(+ 1 2)' | ./build/golisp2`. Ein Klammer-Fehler in der stdlib
   tarnt sich als Suite-Alignment-Problem — das Framing der Suite
   (eine Zeile pro Form) bricht dann komplett zusammen und die Meldung
   zeigt in die falsche Richtung.
3. **XPASS ist der Implementierungs-Tacho.** Der Workflow
   „Case in known-failures → implementieren → XPASS → Eintrag entfernen"
   macht jeden Meilenstein atomar sichtbar. XPASS-Einträge nicht
   entfernen ist der stille Schuldenaufbau der Suite.
4. **Go-Tests sehen nur `BaseEnv()`, nie die stdlib.** Wer in Go-Tests
   `defvar`, `dolist` o. ä. benutzt, testet gegen ein Loch. Der Test
   schlägt dann nicht am Feature, sondern am fehlenden stdlib-Makro —
   genau wie ursprünglich `nth-value` als stdlib-Makro für Go-Tests
   unsichtbar war (Grund, warum es Spezialform wurde).
5. **Lazy-Marker schlägt Baum-Rewrite bei symbol-macrolet.** Die
   Env-Kette erledigt Shadowing von allein; ein Rewrite hätte jede
   Bindungsform kennen müssen — und makroexpandierte `dolist`-Bodies
   hätten es lautlos gebrochen. **Mechanismen nutzen, die schon
   existieren, statt neue zu enumerieren.**
6. **Sentinel-Errors für non-lokale Exits.** `blockReturn` als
   Error-Typ durch die Eval-Kette zu werfen hält `block`/`return-from`/
   `return` (inkl. `nil` als gültigem Block-Namen) aus dem Wert-Raum
   raus — kein Sentinel-Value kann mit einem echten Lisp-Wert kollidieren.
7. **Pre-existing Failures mit Worktree beweisen, nicht mit stash.**
   `git worktree add tmp/head-check HEAD`, dort bauen, identische
   Fehler → kein Regression-Geisterjagen vor dem Commit. Sauberer
   Trennschnitt: 5 failende `tests/*.lisp`-Skripte waren alt
   (fehlende `range`, Quasiquote-Edge, nicht deploytes KI-Modell,
   Timeout) und blieben bewusst außerhalb des Commits.
8. **Eine permanente Abweichung ist OK — wenn sie dokumentiert ist.**
   `progv` vs. lexikalischem `let`: golisp2 hat kein lexical/dynamic-Split,
   das Gold-Verhalten ist ohne Special-Variable-Machinerie unerreichbar.
   Steht als einziger XFAIL-Eintrag mit Begründung in `known-failures.txt`
   statt als halbgarem Workaround im Code.

---

## Action Items

| # | Aufgabe | Priorität |
|---|---------|-----------|
| 1 | Smoke-Test `echo '(+ 1 2)' \| ./build/golisp2` in `run.sh` oder `build.sh` als stdlib-Sanity-Check einbauen | Hoch |
| 2 | 5 pre-existing failing `tests/*.lisp`-Skripte in eigener Session aufräumen (range, Quasiquote-Edge, sigo-Modell, parallel-mind-Timeout) | Mittel |
| 3 | Erwartungswerte neuer Conformance-Cases grundsätzlich gegen clisp generieren (`--gold`), nie aus dem Kopf | Hoch |
| 4 | Suite-Framing dokumentieren: eine Ausgabezeile pro Form, Kommentarzeilen werden gestrippt — Alignment-Verlust deutet auf Stdlib-/Reader-Fehler | Niedrig |

---
