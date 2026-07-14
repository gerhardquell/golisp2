# TODO — golisp2, Stand 2026-07-14, 00:45

## Kontext (falls der Kopf leer ist)
GPS-Port (Norvig PAIP Kap. 4) war grün. Beim Nachfragen *warum* er grün ist,
fielen drei Ebenen auf: gps.lisp korrekt → defstruct-Makro fehlerhaft →
Evaluator unterscheidet Code nicht von Daten.

Kette: `(defstruct box (list nil))` → Konstruktor `(list (quote box) list)`
→ Parameter `list` shadowed das Primitiv → Datenliste in Funktionsposition
→ `eval_core.go:215`, `fn.Env.(*Env)` auf nil → Panic, Prozess tot.
GPS lief nur, weil Norvig keinen Slot `list` benutzt.

## Erledigt (gestern)
- [x] eval_core.go:215 — Type Assertion abgesichert

## 1. Evaluator (zuerst — Fundament)
- [x] Regressionstest zum Fix: `(define xs '(1 2 3))` `(xs 0)` → Fehler, kein Absturz
  (manuell verifiziert; persistenter Test wünschenswert)
- [x] `rg -n '\.\(\*Env\)' lib/` — geprüft und abgesichert
  - `eval_core.go:215` / `eval_lambda.go:26` sicher
  - `env.go:118` ist `envPool.Get().(*Env)` — synchronisierter Pool, bleibt sicher
- [x] `recover()` an die Auswertungsgrenze. **Wichtigster Punkt.**
  `Eval()` fängt jetzt jede Panic und wandelt sie in `eval: panic recovered: ...` um.
  Auch `parfunc`-Worker-Goroutinen sind dadurch abgedeckt.
- [x] Nil-Pointer-Validierung in Spezialformen (zusätzliche harte Abwehr):
  `define`, `defun`, `defmacro`, `set!`, `do`, `flet`, `labels`, `block`,
  `return-from`, `parfunc` liefern bei unvollständigem Input Syntax-Fehler
  statt Panic.
- [x] `bound?` wertet sein Argument aus, damit Makros wie `defvar` und
  `defstruct-resolve-name` korrekt mit Symbol-Variablen arbeiten.
- [ ] Grundsatzfrage (nicht heute entscheiden): eigener Typ LAMBDA/CLOSURE,
      statt Lambda = LIST + optionales Env. Macht den Fehler unmöglich,
      statt ihn zu fangen. Guter Zeitpunkt: GPS ist einziger ernsthafter Nutzer.

## 2. defstruct (embed/stdlib.lisp)
- [x] Konstruktor-Rumpf baut über `list` → jeder Slot namens `list` bricht ihn.
      Fix: internes Primitiv (`%make-struct`) oder Name, den der Nutzer nicht wählen kann.
- [x] **Nicht idempotent**: zweites Laden desselben defstruct weicht komplett auf
      `make--pt` / `pt--x` / `pt--?` aus. Alte Definition bleibt aktiv, lautlos.
      Fix: Kollision nur bei *fremdem* Namen ausweichen.
- [x] `defaults` (Zeile 3 im let*) ist toter Code — wird berechnet, nie benutzt. Raus.
- [x] Kollisionsregel schützt nur eine Ebene: `set--difference` wird kommentarlos
      überschrieben, falls belegt.
- [x] **Warnung auf stderr** bei Ausweichnamen:
      `WARN: defstruct set: 'set-difference' existiert → Accessor heißt 'set--difference'`
      Ohne die steht der tatsächliche Name in *keiner* Quelle.
- [x] `eval '(defun ...)` im Makro ist laut macroexpand überflüssig — prüfen:
      an einem Slot rausnehmen, `(pt-x (make-pt :x 7))` → 7? Dann überall weg,
      Makro halbiert sich.

## 3. stdlib-Tests (Nachholschuld)
- [x] `stdlib-test.lisp` angelegt und in `./build/golisp2 -t` eingebunden.
      Testet: defstruct, setf, defvar, union, set-difference, find-all.
- [x] Bekannte Inkonsistenz: `(setf (pt-x p) 9)` liefert jetzt `9`.
      CL-konform: setf gibt den zugewiesenen Wert zurück.
      Zusätzlich: setf wertet den Wert-Ausdruck nur noch einmal aus
      (gensym-Hilfsvariable), wichtig für Places wie `(setf x (f x))`.

## 4. gps.lisp — Kommentare ehrlich machen
- [x] Kopfkommentar Z. 16–20: "Semantik bleibt erhalten" ist falsch.
      GPS hinterlässt den Endzustand **global** (`set!`), Norvigs dynamische
      Bindung nicht. Belegt: `(println *state*)` nach Fall 1.
- [x] Bei `shop-installs-battery`: kein `:del-list` → Zustand bleibt widersprüchlich
      (`car-needs-battery` UND `car-works`). Norvig-Original, harmlos bis
      Version 2 (negierte Ziele). Hinschreiben, nicht reparieren.
- [x] Norvigs drei bekannte Fehler als Tests ergänzt — ein Port ist erst treu,
      wenn er die *gleichen* Fehler macht:
      1. Clobbered Sibling Goal: Ziele `'(have-money son-at-school)` → muss `solved`
         geben, obwohl das Geld weg ist
      2. Leaping before you look: `'(son-at-school have-money)` → `()`, aber Aktionen
         wurden schon ausgeführt
      3. Rekursives Unterziel (`ask-phone-number`) → muss hängen
      Datei: `pn-gps1/gps-norvig-bugs.lisp`, geladen von `./build/golisp2 -t`.

## 5. Danach
- [x] PAIP Kap. 4.11 → GPS Version 2: **State-Passing** statt globaler Mutation.
      `achieve-all` nimmt state und gibt neuen state zurück. Implementiert in
      `pn-gps1/gps2.lisp`, Tests in `pn-gps1/gps2-tests.lisp`, eingebunden in
      `./build/golisp2 -t`. Goroutine-tauglich, weil `ops` explizit durchgereicht
      wird (golisp2 hat keinen dynamischen Scope).

## RETROSPECTIVE (eintragen)
- Grün heißt nicht richtig. Grün heißt: noch keine passende Frage gestellt.
- Ein Feature-Port zieht Spracherweiterungen nach. Der Anwendungsfall testet
  sie nur entlang *eines* Pfades. Neue stdlib-Funktion → eigener Test.
- Namensgenerierung braucht Kollisionsregeln **und** eine Warnung. Eine Ausweich-
  regel ohne Meldung tauscht einen lauten Fehler gegen einen leisen.
- "Unbekanntes Symbol" im REPL beweist nichts über Code in der Datei. Erst laden.
