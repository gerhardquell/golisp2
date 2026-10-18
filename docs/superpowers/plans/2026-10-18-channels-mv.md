# Channels + Mehrfachwerte Implementation Plan

> **For agentic workers:** Ein frischer Subagent pro Task, Tasks der Reihe nach. Steps mit Checkbox (`- [ ]`).

**Goal:** (0) `golisp2 --version` nach Gerhards Versionsregel. (1) Spezialformen packen Mehrfachwerte (MVALUES) ihres ausgewerteten Arguments korrekt aus (`lock`, `case`, Audit aller übrigen). (2) Channels bekommen nicht-blockierendes Lesen/Schreiben, Lesen mit Timeout und Schließen. (3) `spawn` liefert einen Ergebnis-Channel statt `nil`. (4) `chan-select` wartet auf den ersten von mehreren Channels. (5) Doku, damit KIs nicht mehr in die gefundenen Fallen laufen.

**Anlass:** Bau von golisp2dfs (`/u/ki-projekte/DataFlowSystem/exp/dfs`), Retro `RETROSPECTIVE.md.d/20261018-1705-golisp2dfs-qad.md` dort. Entscheidungen Gerhard 20261018: Rückgabe über Mehrfachwerte; spawn mit Ergebnis-Channel; CL-Namen und `puthash` nur dokumentieren; `select` als Primitiv `chan-select` (nur Empfangen); Versionsstart 0.2.0 (passend zum REPL-Kopf „GoLisp 0.2“), jeder Task zählt hoch.

**Versionen je Task (im selben Commit hochzählen):** Task 0 → 0.2.0 · Task 1 (Fix) → 0.2.1 · Task 2 (Channels) → 0.3.0 · Task 3 (spawn) → 0.4.0 · Task 4 (chan-select) → 0.5.0 · Task 4b (Fix cond/prog1/or) → 0.5.1 · Task 5 (Doku) unverändert.

**Tech Stack:** Go (Modul `golisp2`), Paket `src/lib`, `go test`, eingebettete Stdlib/Doku.

## Global Constraints

- Arbeitsverzeichnis: Worktree `/u/lisp-projekte/golisp2/.worktrees/channels-mv` (Branch `channels-mv`). Nie im Haupt-Checkout arbeiten.
- Frischer Worktree hat keine Build-Artefakte: vor den Tests `./build.sh` (Tests starten `build/golisp2-client`).
- Baseline: `go test ./... -count=1` → **487 passed in 7 packages**; `./build/golisp2 -t` → `=== Report: 141 PASS, 0 FAIL, 0 XFAIL, 0 XPASS ===`.
- Einrückung 2 Spaces, keine Tabs, Stil der Datei beibehalten. **Kein** `gofmt -w` auf ganze Dateien.
- Header bestehender Dateien nie ändern. Neue Dateien: Autor `Gerhard Quell - gquell@skequell.de`, CoAutor = dein eigenes Modell exakt, Copyright `2026 Gerhard Quell - SKEQuell`, Erstellt `20261018`.
- Commit-Trailer: `Co-Authored-By: <dein exaktes Modell> <noreply@anthropic.com>`.
- Fehler: `fmt.Errorf("<primitiv>: …")`. Kommentare/Doku deutsch, sparsam.
- Neues Verhalten → zuerst failing Test, dann Fix.
- TCO-Invariante: kein rekursiver `Eval()`-Aufruf in der Trampolin-Schleife.
- Eine Quelle: Mehrfachwerte immer über `Primary()` (`types.go`), nie eigene Auspack-Logik. Primitiven registrieren nur in `RegisterGoroutines` (über `BaseEnv()`).
- tmp-Dateien unter `./tmp/`, nicht `/tmp` (`t.TempDir()` in Go-Tests ist ok).

## Dateistruktur

| Datei | Aktion | Task |
|---|---|---|
| `src/lib/version.go`, `src/lib/version_test.go` | Create | 0 (danach nur `Version` hochzählen) |
| `src/cli/cli.go`, `src/cli/modes.go`, `src/lib/swank/server.go`, `src/cmd/golisp2-client/main.go` | Modify | 0 |
| `src/lib/eval_control.go` | Modify `evalLock` | 1 |
| `src/lib/eval_specialforms.go` | Modify `evalCase` (+ Audit-Funde) | 1 |
| `src/lib/eval_*.go` | Audit, nur Funde ändern | 1 |
| `src/lib/eval_mv_test.go` | Tests MV in Spezialformen | 1 |
| `src/lib/goroutine.go` | Channels + spawn + chan-select | 2, 3, 4 |
| `src/lib/concurrency_test.go` | Channel-Tests | 2, 4 |
| `src/lib/spawn_test.go` | spawn-Tests anpassen/ergänzen | 3 |
| `tools/gen-reference.lisp` | Einträge spawn + neue Primitiven | 3, 4, 5 |
| `src/embed/ki-referenz.md`, `docs/golisp2-fehler.md`, `docs/golisp2-cheatsheet.md`, `docs/referenz-generiert.md`, `docs/cli.md` | Doku | 5 |

---

## Task 0: `--version` (0.2.0)

**Regel (Gerhard, basis.md):** Ausgabe `<name> <MAJOR.MINOR.PATCH> (<commit>, <JJJJ-MM-TT>[, dirty])`. Nummer von Hand, Commit/Datum/dirty automatisch aus git. Daemons zusätzlich in der Startmeldung.

**Ort:** `src/lib/version.go` – nicht `main.go`, weil `cli.Run` auch von gogui als Bibliothek genutzt wird und `swank` sowie `golisp2-client` nur `lib` importieren.

```go
// Version von golisp2 – von Hand gepflegt (SemVer)
const Version = "0.2.0"

// VersionText liefert "<name> <Version> (<commit>, <datum>[, dirty])";
// ohne VCS-Infos (z. B. go test) nur "<name> <Version>".
func VersionText(name string) string {
  info, ok := debug.ReadBuildInfo()
  if !ok {
    return name + " " + Version
  }
  var rev, zeit string
  dirty := false
  for _, s := range info.Settings {
    switch s.Key {
    case "vcs.revision":
      rev = s.Value
    case "vcs.time":
      zeit = s.Value
    case "vcs.modified":
      dirty = s.Value == "true"
    }
  }
  if rev == "" {
    return name + " " + Version
  }
  teile := []string{rev[:min(7, len(rev))]}
  if len(zeit) >= 10 {
    teile = append(teile, zeit[:10])
  }
  if dirty {
    teile = append(teile, "dirty")
  }
  return name + " " + Version + " (" + strings.Join(teile, ", ") + ")"
}
```

- [ ] `version_test.go`: ohne Build-Info beginnt `VersionText("golisp2")` mit `golisp2 0.2.0`; Format-Regex für den Fall mit VCS-Daten über eine kleine Hilfsfunktion testbar machen (Settings als Parameter).
- [ ] `cli.go`: Flag `version` (Go-`flag` akzeptiert `-version` und `--version`) → `VersionText("golisp2")` auf `out`, Exit 0, **vor** BaseEnv/LoadStdlib. Test in `cli_test.go`.
- [ ] `modes.go:184`: REPL-Kopf `GoLisp 0.2` → `VersionText("golisp2")`.
- [ ] `swank/server.go`: beide Startmeldungen `SWANK server on …` → `<VersionText("golisp2")> – SWANK server on …`.
- [ ] `golisp2-client`: Flag `version` → `VersionText("golisp2-client")`, Exit 0, **vor** dem Verbindungsaufbau.
- [ ] Prüfen nach `./build.sh`: `./build/golisp2 --version` → `golisp2 0.2.0 (<7 hex>, <datum>, dirty)` (dirty, solange uncommittet); `./build/golisp2-client --version` analog ohne Server.
- [ ] `docs/cli.md`: Flag ergänzen.
- [ ] Grün + Commit `feat(cli): --version für golisp2 und golisp2-client (0.2.0)`.

---

## Task 1: Mehrfachwerte in Spezialformen (0.2.1)

**Befund:** Funktionsargumente, `let`, `setq` usw. packen über `Primary()` aus. `evalLock` und `evalCase` nicht:

```lisp
(let ((h (make-hash-table)))
  (setf (gethash "mu" h) (lock-make))
  (lock (gethash "mu" h) 7))            ; → ERR: kein Mutex     (soll: 7)
(case (values 'a 2) ((a) "ok") (t "nein"))  ; → "nein"           (soll: "ok") – still falscher Zweig!
```

Schon geprüft und korrekt: `dolist`, `dotimes`, `while`, `catch`, `parfunc :timeout`, `handler-case`, `unwind-protect`, `setf`, `do`, `apply`, `funcall`, `typep`, `format`.

- [ ] Failing Tests in `eval_mv_test.go`: die zwei Beispiele oben (mit `evalEq`).
- [ ] Fix: `evalLock` → `getMutex(Primary(muCell))`; `evalCase` → `key = Primary(key)` direkt nach dem Auswerten.
- [ ] Audit: jede Stelle in `src/lib/eval_*.go`, an der das Ergebnis von `evalWithCtx`/`Eval` **inspiziert** wird (Typprüfung, Vergleich, als Zahl/Liste/Symbol/Tag/Mutex/Env benutzt). Nicht ändern: Tail-Rückgaben (MV muss dort durchgereicht werden, CL) und reine Wahrheitsprüfungen (`IsTruthy` packt selbst aus).
- [ ] Je Fund: ein Test, der vorher scheitert, dann `Primary()`. Liste aller geprüften Stellen mit Urteil (ok / gefixt) in den Abschlussbericht.
- [ ] `Version` → `0.2.1`.
- [ ] `go test ./... -count=1` und `./build/golisp2 -t` grün. Commit `fix(eval): Spezialformen packen Mehrfachwerte über Primary aus (0.2.1)`.

**Erfolg:** Beide Beispiele liefern `7` bzw. `"ok"`; keine Regression.

---

## Task 2: Channels – try, Timeout, close (0.3.0)

**API (verbindlich):**

| Form | Ergebnis |
|---|---|
| `(chan-recv ch)` | `(values x t)`; geschlossen und leer → `(values nil nil)` sofort (heute: ewig blockiert) |
| `(chan-recv ch :timeout sek)` | wie oben; nach `sek` Sekunden (Zahl ≥ 0) → `(values nil nil)` |
| `(chan-try-recv ch)` | `(values x t)` oder sofort `(values nil nil)` |
| `(chan-send ch x)` | wie bisher (blockiert); auf geschlossenem Channel Fehler `chan-send: Channel geschlossen` |
| `(chan-try-send ch x)` | `t` gesendet, `nil` wenn Puffer voll / kein Empfänger bereit; geschlossen → Fehler `chan-try-send: Channel geschlossen` |
| `(chan-close ch)` | `t`; zweites Mal Fehler `chan-close: Channel bereits geschlossen` |
| `(chan-closed? ch)` | `t` nur wenn geschlossen **und** leer (dann kommt garantiert nichts mehr), sonst `nil` |

Syntaxfehler wörtlich: `chan-recv: Syntax: (chan-recv ch [:timeout sek])`.
Primärwert bleibt das Datum → bestehender Code `(chan-recv ch)` ändert sich nicht.

**Code-Vorlage** (`goroutine.go`; Typnamen/Zahlen-Zugriff an den Bestand anpassen, z. B. wie `parfunc :timeout` die Zahl liest):

```go
type goChannel struct {
  ch     chan *Cell
  closed atomic.Bool
}

// recvValues: (values x t) bei Erfolg, (values nil nil) sonst
func recvValues(v *Cell, ok bool) *Cell {
  if !ok { return MakeValues([]*Cell{MakeNil(), MakeNil()}) }
  return MakeValues([]*Cell{v, MakeAtom("t")})
}

// chanPut: Senden mit Schutz gegen close-Wettlauf (Go panict bei send auf closed)
func chanPut(gc *goChannel, v *Cell, block bool, name string) (sent bool, err error) {
  if gc.closed.Load() { return false, fmt.Errorf("%s: Channel geschlossen", name) }
  defer func() {
    if r := recover(); r != nil { sent, err = false, fmt.Errorf("%s: Channel geschlossen", name) }
  }()
  if block { gc.ch <- v; return true, nil }
  select {
  case gc.ch <- v: return true, nil
  default:         return false, nil
  }
}
// chan-recv mit :timeout: time.NewTimer + select { case v, ok := <-gc.ch / case <-timer.C }
// chan-try-recv: select { case v, ok := <-gc.ch / default }
// chan-close: CompareAndSwap(false, true) sonst Fehler; dann close(gc.ch)
// chan-closed?: gc.closed.Load() && len(gc.ch) == 0
```

- [ ] Failing Tests in `concurrency_test.go`, deterministisch (gepufferte Channels, kurze Timeouts 50 ms, großzügige Zeitgrenzen): je Zeile der API-Tabelle mindestens ein Test, dazu: Puffer wird nach `chan-close` noch geleert; `multiple-value-bind` sieht `(x t)` bzw. `(nil nil)`; `chan-try-send` auf vollem Puffer → `nil`; Timeout kehrt nach ≈ `sek` zurück (nicht sofort, nicht nach 1 s).
- [ ] Implementieren, registrieren in `RegisterGoroutines`, Kopfkommentar von `goroutine.go` um die neuen Formen ergänzen.
- [ ] `go test ./... -count=1 -race` für `src/lib` mindestens einmal ohne Befund.
- [ ] `Version` → `0.3.0`.
- [ ] Grün + Commit `feat(goroutine): chan-try-recv, chan-recv :timeout, chan-try-send, chan-close, chan-closed? (0.3.0)`.

---

## Task 3: spawn mit Ergebnis-Channel (0.4.0)

**Verhalten:** `(spawn fn)` gibt einen Channel (Puffer 1) zurück. Die Goroutine schickt genau eine Nachricht:
- Erfolg: `(:ok wert)` – `wert` = `Primary(Ergebnis)`
- Fehler: `(:error "text")` – Text wie `(lisp-error-msg e)` ihn liefern würde (bei `*LispError` `.Msg`, sonst `err.Error()`)
- panic: `(:error "panic: …")`

Die bisherigen stderr-Meldungen (`spawn: …`, `spawn: panic: …`) bleiben. Wer den Rückgabewert ignoriert, merkt keinen Unterschied. Syntaxprüfungen (`(spawn)`, `(spawn 1)`) bleiben.

- [ ] Prüfen: wer nutzt den Rückgabewert von `spawn` (`rg -n "spawn" --glob '*.lisp' --glob '*.go'`, inkl. `tests/`, `examples/`). Funde im Bericht nennen.
- [ ] Failing Tests in `spawn_test.go`: `(chan-recv (spawn (lambda () 42)))` → `(:ok 42)`; Fehlerfall → `(:error "kaputt")`; Rückgabe ist ein Channel.
- [ ] Implementieren; Kommentar „Fire-and-forget … kehrt mit nil zurück“ an `fnSpawn` und im Dateikopf anpassen; `tools/gen-reference.lisp` Eintrag `spawn` anpassen.
- [ ] `Version` → `0.4.0`.
- [ ] Grün + Commit `feat(goroutine): spawn liefert Ergebnis-Channel (:ok wert) / (:error text) (0.4.0)`.

---

## Task 4: chan-select (0.5.0)

**API (verbindlich):** `(chan-select chans [:timeout sek])` – `chans` ist eine Liste von Channels. Nur Empfangen.

| Lage | Ergebnis |
|---|---|
| ein Channel liefert `x` | `(values x t i)`, `i` = Index (ab 0) in `chans`; bei mehreren bereiten zufällig (wie Go) |
| `:timeout sek` abgelaufen | `(values nil nil nil)` |
| `:timeout 0` und nichts bereit | sofort `(values nil nil nil)` |
| alle Channels geschlossen und leer | sofort `(values nil nil nil)` (auch ohne Timeout – sonst Ewig-Block) |
| ein Channel schließt während des Wartens | wird übersprungen, Warten geht mit den übrigen weiter |
| ohne `:timeout` | blockiert, bis einer liefert oder alle geschlossen sind |

Fehler wörtlich: `chan-select: Syntax: (chan-select chans [:timeout sek])`, `chan-select: leere Channel-Liste`, `chan-select: kein Channel an Stelle <i>`.

**Umsetzung:** `reflect.Select` mit einem `reflect.SelectCase` je offenem Channel plus Timer-Case; Index-Abbildung auf `chans` mitführen. Liefert ein Case `ok == false` (geschlossen), Case entfernen und erneut wählen. Ergebnis über `MakeValues`. Zahl für `:timeout` wie in Task 2 lesen (gemeinsame Hilfsfunktion, nicht zweimal).

- [ ] Failing Tests in `concurrency_test.go` je Tabellenzeile (deterministisch: gepufferte Channels, kurze Timeouts), dazu `multiple-value-bind`-Test.
- [ ] Implementieren, in `RegisterGoroutines` registrieren, Dateikopf und `tools/gen-reference.lisp` ergänzen.
- [ ] `-race` einmal ohne Befund.
- [ ] `Version` → `0.5.0`.
- [ ] Grün + Commit `feat(goroutine): chan-select – erster von mehreren Channels (0.5.0)`.

---

## Task 4b: cond ohne Körper, prog1/prog2/or nach CL (0.5.1)

**Befund aus Task 1** (Entscheidung Gerhard 20261018: mit erledigen):

| Form | heute | soll (CL) |
|---|---|---|
| `(cond ('foo))` | Fehler „unbekanntes Symbol 'foo'“ – Testwert wird ein zweites Mal ausgewertet (`expr = testVal; continue`) | `foo` |
| `(cond ((list 1 2)))` | ruft `(1 2)` als Funktion auf | `(1 2)` |
| `(multiple-value-list (cond ((values 1 2))))` | – | `(1)` – Klausel ohne Körper liefert den **Primärwert** des Tests |
| `(multiple-value-list (prog1 (values 1 2) 3))` | `(1 2)` | `(1)` |
| `(multiple-value-list (prog2 0 (values 1 2) 3))` | `(1 2)` | `(1)` |
| `(multiple-value-list (or (values 1 2) 3))` | `(1 2)` | `(1)` – vordere Formen nur Primärwert |
| `(multiple-value-list (or nil (values 1 2)))` | `(1 2)` | `(1 2)` – letzte Form bleibt Tail, MV wird durchgereicht |
| `(multiple-value-list (and 1 (values 1 2)))` | `(1 2)` | `(1 2)` – unverändert (Charakterisierungstest) |
| `(parfunc r :timeout 0.2 (progn (sleep 1000) 1))` | Timeout still ignoriert (`time.Duration(0.2)*time.Second` = 0 → kein Timeout), Ergebnis `(1)` nach 1 s | `(())` nach ≈ 0,2 s – Zahl über `timeoutArg` (Task 2) lesen, keine zweite Lese-Logik |

- [ ] Failing Tests (eine Testfunktion je Form) in `src/lib/eval_mv_test.go` bzw. für `cond` in `eval_control_test.go`/passender Datei; `and`-Zeile als Charakterisierungstest.
- [ ] Fix `cond`: Klausel ohne Körper gibt `Primary(testVal)` direkt zurück, **kein** erneutes Auswerten. TCO der übrigen `cond`-Zweige unverändert lassen.
- [ ] Fix `prog1`/`prog2`/`or` über `Primary()`; letzte `or`-Form bleibt Tail.
- [ ] Fix `parfunc :timeout`: `timeoutArg` aus `goroutine.go` nutzen (Bruchteile, Typprüfung); bestehende `parfunc`-Tests bleiben grün.
- [ ] `Version` → `0.5.1`.
- [ ] Grün + Commit `fix(eval): cond ohne Körper wertet nicht doppelt aus; prog1/prog2/or liefern Primärwert (0.5.1)`.

---

## Task 5: Doku

- [ ] `src/embed/ki-referenz.md` (**Cache-Regel:** Dateianfang nicht ändern, nur die betroffenen Abschnitte):
  - Abschnitt 3, Zeile **Goroutinen**: alle Channel-Formen mit Signatur und Rückgabe (Tabellen aus Task 2 und 4), `spawn` mit Ergebnis-Channel.
  - Zeile **Hashtable**: `puthash` als intern markieren: `(puthash key table value)` – Tabelle an 2. Stelle (anders als Emacs Lisp); empfohlen `(setf (gethash k h) v)`.
  - Abschnitt 9 **Gotchas**, neue Zeilen: `puthash`-Reihenfolge; `stringp`/`consp`/`string=`/`equal` fehlen → `string?`, `(typep x 'cons)`, `equal?`; `member` ohne `:test` (vergleicht Strings strukturell); `(last l)` liefert das Element, nicht die Zelle; `slurp` liest stdin, Datei → `file-read`; `(gets)` am Eingabeende → Fehler `gets: EOF`, nicht `nil`.
- [ ] `README.md`, `README_en.md`, `README_CN.md`: festen REPL-Kopf `GoLisp 0.2  –  Ctrl+D …` an die neue Ausgabe anpassen (`golisp2 <version> (…)  –  Ctrl+D …`), `--version` erwähnen.
- [ ] `docs/golisp2-fehler.md`: Einträge im Stil der Datei – [FIXED] MV in `lock`/`case` (+ 21 Audit-Funde aus Task 1); [FIXED] `cond` ohne Körper wertete doppelt aus; [FIXED] `prog1`/`prog2`/`or` reichten MV durch; [KNOWN] `puthash`-Reihenfolge; [LIMITATION] fehlende CL-Namen, `last`; [FIXED] Channels ohne Timeout/close; [FIXED] spawn-Fehler nur auf stderr.
- [ ] `docs/golisp2-cheatsheet.md`: Abschnitt Nebenläufigkeit um die neuen Formen ergänzen.
- [ ] `docs/referenz-generiert.md` mit `tools/gen-reference.lisp` neu erzeugen (Aufruf steht im Kopf des Skripts).
- [ ] `TODO.md` Punkt 1 (`--version`) **nicht** anfassen – Gerhard hat dort offene Änderungen; der Hauptagent hakt ihn nach dem Merge ab.
- [ ] Commit `docs: Version, Channels, chan-select, spawn-Ergebnis, MV-Fix, Gotchas aus golisp2dfs`.

---

## Task 6: Abschluss (Hauptagent, nicht Subagent)

- [ ] Branch-Review (stärkstes Modell) über `main..channels-mv`.
- [ ] Merge nach `main` (Gerhards offene Änderungen an `TODO.md`/`zeitstempel.txt` nicht anfassen), `./build.sh`, `cp build/golisp2 /usr/local/bin/golisp2`; danach `golisp2 --version` → `golisp2 0.5.0 (<commit>, <datum>)` ohne dirty? (dirty, solange Gerhards Änderungen offen sind – so melden).
- [ ] `TODO.md` Punkt 1 abhaken – erst nach Rückfrage, weil die Datei uncommittete Änderungen von Gerhard hat.
- [ ] Zutat `~/.claude/zutaten/sprachen/golisp2.md` aus `ki-referenz.md` abgleichen (Zutat = eigener Kopf + Referenz mit um eine Ebene tieferen Überschriften).
- [ ] DataFlowSystem `exp/dfs/test.sh` mit neuem Binary: 27/27.
- [ ] kh-Zettel `e72bf7d0` (Channels) und `aec6141d` (Stolpersteine) aktualisieren.

**Nicht in dieser Runde:** `select` mit Sende-Zweigen (Spezialform wie Go); `--version` für `golisp2web` (eigenes Programm); golisp2dfs auf die neuen Channels umstellen (eigener Schritt in DataFlowSystem).
