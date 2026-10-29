# Vektor-Alltag und Sequenz-Kern — Implementation Plan

> **For agentic workers:** Tasks der Reihe nach, Test zuerst (failing), dann Code, dann volle Suiten, dann Commit. Steps als Checkboxen.

**Goal:** `#(…)`, `vectorp`/`arrayp`, `vector-pop`, `json-encode` für Vektoren, `loop … across`; dazu `elt subseq copy-seq concatenate map some find find-if position position-if count count-if` für Listen und Vektoren.

**Spec:** `docs/superpowers/specs/2026-10-29-sequenzen-design.md` — bindend, vor Beginn lesen. Kommt dir etwas falsch vor: melden, nicht umgehen.

## Global Constraints

- CLAUDE.md im Repo-Root lesen (Chokepoints, TCO-Invariante, Symbol-Interning: ATOM-Cells nur via `MakeAtom`).
- Einrückung: Stil der Datei (`primitives.go`, `arith_prims.go`, `types.go` = Tabs; `vector.go`, `jsoncell.go`, `reader.go`, `eval_core.go`, `clcompat_prims.go` = 2 Spaces — vor dem Edit prüfen). Neue Dateien: 2 Spaces. Kein `gofmt -w` auf ganze Dateien.
- Header neuer Dateien: Autor `Gerhard Quell - gquell@skequell.de`, CoAutor = **dein exaktes Modell**, Copyright `2026 Gerhard Quell - SKEQuell`, Erstellt `20261029`. Bestehende Header nie ändern.
- Fehler: `fmt.Errorf("<funktionsname>: …")`, deutsch.
- Neue Primitiven in `BaseEnv()` registrieren (eigene Datei → `RegisterXxx(env)` dort aufrufen).
- Vor jedem Commit: `./build.sh && go test ./... -count=1 && ./build/golisp2 -t 2>&1 | tail -1` (Baseline 168 PASS). Der Tutorial-Abdeckungstest (`TestTutorialDecktAlleSymbole`) wird durch neue Symbole rot — das ist **erwartet** und wird in Task 4 behoben; in Task 1–3 diesen einen Test als bekannt-rot melden, nicht selbst Tutorial-Einträge schreiben.
- Commits nur mit expliziten Pfaden; nie `zeitstempel.txt`, `RETROSPECTIVE.md`, `RESUME.md`, `*.md.d/`. Trailer `Co-Authored-By: <dein exaktes Modell> <noreply@anthropic.com>`. Nicht pushen. Kein `rm -r`; tmp unter `./tmp/`.
- Testhelfer: `evalEq(t, src, want)`, `evalErr(t, src)` (BaseEnv, `eval_test.go`), `evalStdlibEq`/`evalStdlib` (mit Stdlib, `stdlib_reduce_setf_test.go`).

---

### Task 1: Vektor-Alltag (Go)

**Files:** `src/lib/vector.go`, `src/lib/jsoncell.go`, `src/lib/eval_core.go` (Zeile ~108), `src/lib/reader.go` (`readDispatch`, ~Z. 220), neue Tests in `src/lib/vector_alltag_test.go`.

- [ ] Tests zuerst, u. a.:
  - `(vectorp (vector 1))` → `t`, `(vectorp '(1))` → `()`, `(arrayp (vector))` → `t`
  - `(let ((v (vector 1 2))) (list (vector-pop v) v))` → `(2 #(1))`; `(vector-pop (vector))` → Fehler; `(vector-pop '(1))` → Fehler
  - `(json-encode (vector 1 "a" (vector 2)))` → `"[1,\"a\",[2]]"` (genaue Ausgabeform gegen `(json-encode (list 1 "a" (list 2)))` abgleichen — gleiche Form erwartet)
  - `#(1 2 3)` → `#(1 2 3)`; `#()` → `#()`; `#(a (b c) "s")` → `#(a (b c) "s")` (Elemente nicht ausgewertet, `a` ungebunden ist ok); `(vectorp '#(1))` → `t`; `(aref #(1 2) 1)` → `2`; `(eval (vector 1))` → `#(1)`; verschachtelt `#(#(1))` → `#(#(1))`; `#(1 2` → Reader-Fehler
- [ ] Fehlschlag bestätigen.
- [ ] Umsetzen:
  - `vectorp`, `arrayp` (gleiche Funktion), `vector-pop` in `vector.go` (`RegisterVector`), Locking wie `vector-push-extend`.
  - `jsoncell.go` `cellToJSONValue`: `case VECTOR` wie LIST (Elemente unter `RLock` kopieren).
  - `eval_core.go:108`: `VECTOR` zu den selbstauswertenden Typen.
  - `reader.go` `readDispatch`: `#(` → Elemente bis `)` lesen (vorhandene Listen-Lese-Logik wiederverwenden, nicht duplizieren — z. B. die Liste lesen und mit `makeVector(CellToSlice(...))` umwandeln; Punktnotation `#(1 . 2)` → Fehler).
- [ ] Grün, volle Suiten, Commit `feat: #(…), vectorp/arrayp, vector-pop, json-encode für Vektoren`.

### Task 2: `loop … across`

**Files:** `src/embed/loop.lisp`, `tests/loop-tests.lisp`.

- [ ] Lisp-Test(s) in `tests/loop-tests.lisp` im vorhandenen Stil: `(loop for x across (vector 1 2 3) collect (* x x))` → `(1 4 9)`; mit `sum`; leerer Vektor → `()`; Liste statt Vektor → Fehler (über `aref`/`length` laut).
- [ ] Fehlschlag (`./build/golisp2 -t`), umsetzen analog zu `for v in liste` (Index-Variable via `gensym`, `length`/`aref`), Kopfkommentar von `loop.lisp` (Klausel-Liste, Z. ~14) ergänzen.
- [ ] Grün, volle Suiten, Commit `feat(loop): for x across vektor`.

### Task 3: Sequenz-Kern (Go) und `eql` als Primitive

**Files:** neu `src/lib/sequence.go` (+ `RegisterSequence(env)` in `BaseEnv()`), neu `src/lib/sequence_test.go`, `src/lib/arith_prims.go` (eql neben `eq`/`equal?`), `src/embed/stdlib.lisp:612-617` (Lisp-`eql` entfernen), `pn-gps1/gps2.lisp:65-70` (eigenes `some` entfernen).

- [ ] Tests zuerst (`sequence_test.go`) — je Funktion: Liste, Vektor, leere Sequenz, Fehlerfall, `:key`/`:test`, Symbol als Funktionsbezeichner. Mindestens:
  - `(elt '(a b c) 1)` → `b`, `(elt (vector 1 2) 1)` → `2`, `(elt '(1) 5)` → Fehler, `(elt "abc" 0)` → Fehler
  - `(subseq '(1 2 3 4) 1 3)` → `(2 3)`, `(subseq (vector 1 2 3) 1)` → `#(2 3)`, `(subseq '(1 2) 1 5)` → Fehler, `(subseq '(1 2) 2)` → `()`
  - `(copy-seq (vector 1 2))` → `#(1 2)` und nicht `eq` zum Original
  - `(concatenate 'list '(1) (vector 2) ())` → `(1 2)`, `(concatenate 'vector '(1) (vector 2))` → `#(1 2)`, `(concatenate 'string "a")` → Fehler
  - `(map 'list + '(1 2 3) (vector 10 20))` → `(11 22)`, `(map 'vector (lambda (x) (* x 2)) '(1 2))` → `#(2 4)`, `(map nil print '(1))` → `()` (Ausgabe egal), `(map 'list '1+ '(1))` → `(2)`
  - `(some (lambda (x) (and (> x 1) (* x 10))) '(1 2 3))` → `20`, `(some evenp (vector 1 3))` → `()`
  - `(find 2 '(1 2 3))` → `2`, `(find "a" '("a"))` → `()` (eql!), `(find "a" '("a") :test 'equal?)` → `"a"`, `(find 3 (vector '(1 . a) '(3 . b)) :key 'car)` → `(3 . b)`
  - `(find-if evenp (vector 1 4 6))` → `4`, `(position 3 '(1 2 3))` → `2`, `(position 9 '(1))` → `()`, `(position-if evenp (vector 1 2))` → `1`
  - `(count 1 '(1 2 1))` → `2`, `(count-if evenp (vector 2 4 5))` → `2`, `(count 1 '(1) :foo 2)` → Fehler, `(count 1 '(1) :test)` → Fehler
  - `eql`: `(eql 1 1)` → `t`, `(eql 1.0 1)` → `t` (golisp2 kennt nur float64 — wie bisher), `(eql "a" "a")` → `()`, `(eql 'a 'a)` → `t`, `(eql (list 1) (list 1))` → `()` — vorher mit dem Lisp-`eql` gegenprüfen, dass das Verhalten identisch bleibt (Charakterisierung).
- [ ] Fehlschlag bestätigen.
- [ ] Umsetzen:
  - `seqItems(name string, c *Cell) ([]*Cell, bool, error)` wie Spec; nutzt `properList` (`primitives.go`) und `asVector`.
  - Keyword-Parser für `:test`/`:key` (Symbole via `resolveFn(env, x)`), unbekannt/ohne Wert → Fehler. Positionale Funktionsargumente via `withFnDesignator`.
  - `eql` als Go-Primitive (Zahlen nach Wert, sonst Pointer), Lisp-Definition löschen; prüfen, dass Lisp-Code, der `eql` nutzt (`types.lisp` `member`/`eql`-Typangaben, Hashtabellen sind unabhängig), grün bleibt.
  - `pn-gps1/gps2.lisp`: `some` entfernen, Kommentar „some ist eingebaut (CL)“; prüfen, wo gps2 geladen wird, und dass dessen Tests grün bleiben.
- [ ] Grün, volle Suiten (bis auf den bekannten Tutorial-Test), Commit `feat: Sequenz-Kern (elt subseq map find position count …), eql in Go`.

### Task 4: Doku und Version (Hauptsession)

Tutorial-Einträge (`### name`) für alle neuen Symbole, Cheatsheet, `ki-referenz.md` (Vektor-Zeile, Sequenzen), `lisp-semantik.md` (Vektor-Absatz: Literal, selbstauswertend), `tools/gen-reference.lisp` + Regenerierung, SWANK-Arglisten, Version 0.24.0. Volle Suiten, Commit.
