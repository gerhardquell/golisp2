# GoLisp Eval-Performance — Handoff für Claude Code

> Übergabe aus einem Tee-Gespräch (Gerhard + Claude Opus 4.8), 2026-06-26.
> Ziel: den Tree-Walking-Interpreter schneller machen. `fib` ist das
> Mikroskop, nicht das Problem.

---

## 1. Worum es geht

Rekursives `(fib 25)` dient als **Mikrobenchmark für den Eval-Overhead pro
Funktionsaufruf**. fib ist baumrekursiv → kein TCO → jeder der ~242 000
Aufrufe läuft den vollen Funktionsanwendungs-Pfad ganz unten in `Eval`.
Damit misst fib in Reinform, was *ein Aufruf* kostet. Wir wollen NICHT fib
selbst beschleunigen (Memoization wäre trivial), sondern den **Interpreter**.

Vorgehen, strikt: **ein Schnitt → bauen → testen → eine Zahl**. Keine
gebündelten Änderungen, sonst ist die Attribution verschmiert.

---

## 2. Zahlen-Verlauf (fib 25, Ryzen 5 5500)

| Stand | allocs/op | B/op | ns/op |
|-------|-----------|------|-------|
| Baseline (unverändert) | 2 063 673 | 124 MB | 108 ms |
| **Schnitt 1** (t/nil-Singletons) ✅ | 1 942 278 | 113 MB | 98 ms |
| evalArgs vorzählen ❌ (Phantom, siehe §5) | 1 942 278 | 113 MB | 93 ms |
| **Schnitt 2** (slice-Frame-Env mit map-Root) ✅ | ~1 000 000 | ~50 MB | ~70 ms |
| **Schnitt 3** (direktes Argument-Binding, kein evalArgs-Slice) ✅ | 850 733 | 33 MB | 53 ms |
| **Schnitt 4** (sync.Pool für FUNC-Arg-Slices) ✅ | 243 851 | 23 MB | 51 ms |
| **Schnitt 5** (Frame-Env-Pool + Tail-Call-Freigabe) ✅ | 987 | 95 KB | 86 ms |
| **Schnitt 6** (Small-Int-Cache auf int16-Bereich) ✅ | 3 | 641 B | 88 ms |
| ⚠️ Nachmessung 2026-08-05 (unverändert) | 1 335 430 | 32 MB | 139 ms |
| **Schnitt 7** (Env-Pool entfernt — Korrektheit) ✅ | 1 578 206 | 59 MB | 142 ms |

**Aktuelle Baseline: `1 578 206 allocs/op`, 59 MB/op, ~142 ms/op.**

⚠️ **Die „3 allocs/op" aus Schnitt 6 sind Geschichte, nicht Zustand.**
Zwischen Schnitt 6 (2026-06-26) und der Nachmessung am 2026-08-05 ist das
Depth-Limit-/Cancellation-Feature (`evalCtx`) dazugekommen und hat die
Allokationsgewinne der Schnitte 3–6 überschrieben — ohne dass es hier
dokumentiert wurde. Wer diese Datei liest und „3" als Ausgangspunkt nimmt,
plant gegen eine Zahl, die es seit sechs Wochen nicht mehr gibt.

Faustregel zum Einordnen: `allocs/op ÷ 242 000 ≈ Allokationen pro Aufruf`.
Aktuell ~0,004. Jede Allokation ist GC-Last; weniger Objekte = nichtlinear
weniger GC-Druck.

---

## 4.5 Schnitt 5: Frame-Env-Pool + Tail-Call-Freigabe ✅

**Ziel:** die verbleibenden `NewEnv`-Allokationen eliminieren, nachdem
Schnitt 2 die Frame-Envs bereits auf Slice-Frames umgestellt hat.

**Änderungen:**
- `lib/env.go`: `envPool` für Frame-Envs, `shared`-Flag.
  `NewEnv(parent!=nil)` holt aus dem Pool und resettet inline + Slices.
  `freeEnv` gibt zurück, wenn nicht `shared` und nicht Root.
- `lib/eval_lambda.go`: `makeLambda` markiert `env.shared = true`, damit
  Closures nicht in den Pool zurückgegeben werden.
  `applyLambda` behält `defer freeEnv(localEnv)`.
- `lib/eval_core.go`: `Eval` trackt `ownEnv` – den letzten im Tail-Call
  angelegten Frame. Bei Tail-Call-Uebergang (`let`/`let*`/`lambda-TCO`)
  wird der Vorgaenger freigegeben, am Ende der Aufruf der letzte Frame.
  `ownEnv == newEnv.parent` → nicht freigeben (neuer Frame braucht ihn).
- `lib/eval_control.go`: `do`/`flet`/`labels` haben `defer freeEnv(localEnv)`
  bekommen, damit diese lokalen Frames ebenfalls zurückfließen.

**Ergebnis (fib 25):**
```
987 allocs/op, 95 KB/op, ~86 ms/op
```

**Verbleibende Allokationen:** fast ausschließlich `MakeNum` für
Zwischenergebnisse > 127 (das kleine Int-Cache-Fenster). Siehe Profil
weiter unten.

**Lektion:** sync.Pool für Envs funktioniert, verlangt aber klare
Ownership. Einmal falsch freigegebener Frame, der noch Parent eines
späteren Tail-Calls ist, produziert sofort wilde Pointer. Die Lösung:
`ownEnv`-Tracking im Trampolin-Loop plus `shared`-Flag für Closures.

---

## 4.5b Schnitt 7: Frame-Env-Pool entfernt — Rücknahme von Schnitt 5 ✅

**Anlass:** kein Performance-Ziel, sondern ein Korrektheitsloch. Schnitt 5
hat `envPool` + `shared` + `ownEnv`-Tracking eingeführt. `shared` ist ein Bit
**pro Frame**, die Erreichbarkeit einer Closure läuft aber transitiv über
`parent`. Fing eine Closure einen *Nachkommen*-Frame, blieb der Zwischen-Frame
unmarkiert, wanderte in den Pool und wurde recycelt.

**10 von 11 frame-besitzenden Formen waren betroffen:** `let`, `let*`,
Lambda-TCO, `applyLambda`, `do`, `do*`, `flet`, `macrolet`,
`symbol-macrolet`, `progv`, `multiple-value-bind`. Nur `labels` kam durch —
weil es `makeLambda` den `localEnv` übergibt und ihn dadurch als Nebeneffekt
markiert. Zwei Fehlerbilder: verlorene Bindung, oder ein `parent`-Zyklus, der
`Env.Get` in einen **unrecoverbaren** `fatal error: stack overflow` schickt
(kein `panic` → `recover()` in `evalWithCtx` sieht ihn nie).

Die Lektion aus §4.5 („verlangt klare Ownership … die Lösung: `ownEnv`-Tracking
plus `shared`-Flag") war zu optimistisch: `ownEnv` + `shared` deckt genau den
Fall ab, den `fib` benchmarkt — Frames ohne Closures. Sobald eine Closure
einen Zwischen-Frame überlebt, ist die Ownership-Regel falsch, und der
Benchmark merkt davon nichts.

**Änderung:** `envPool`, `freeEnv`, `Env.shared`, `takeEnv`/`ownEnv` entfernt.
`NewEnv(parent != nil)` gibt `&Env{parent: parent}` zurück. Frame-Lebensdauer
gehört dem Go-GC — der kennt transitive Erreichbarkeit exakt, statt sie mit
einem Bit zu approximieren. Das TCO-Trampolin ist unangetastet (3 Mio.
Tail-Calls verifiziert, O(1) Stack).

**Ergebnis (fib 25):**
```
vorher:  1 335 430 allocs/op   32 MB/op   139,1 ms/op
nachher: 1 578 206 allocs/op   59 MB/op   142,3 ms/op
Delta:      +242 776 (+18 %)     +85 %       +2,3 %
```

Die +242 776 sind **exakt ein Env pro fib-Aufruf** (242 785 Aufrufe) — die
Änderung kostet genau das, was sie vorhergesagt hat, und nichts darüber.
Der Zeitpreis von 2,3 % zeigt, dass der Pool zuletzt kaum noch etwas trug:
`sync.Pool`-Atomics plus die `shared`-Prüfungen aßen ihren eigenen Gewinn
weitgehend auf.

**Lektion:** Ein manueller Pool über einer GC-Sprache schaltet den GC für
einen Objekttyp ab und übernimmt die Beweislast für die Lebensdauer. Wer
diese Beweislast mit einem Bit pro Objekt tragen will, muss zeigen, dass die
Eigenschaft nicht transitiv ist. Bei verketteten Environments ist sie es.

**Regressionsnetz:** `lib/env_closure_test.go` — Kreuzprodukt Closure ×
Frame-Form, 10 Bug-Fälle + 7 Kontrollen + ein Subprozess-Test für den
Stack-Overflow. Vor dem Fix rot verifiziert.

---

## 4.6 Schnitt 6: Small-Int-Cache verbreitern ✅

**Ziel:** die verbleibenden ~987 Allokationen pro `fib 25` eliminieren.
Das Profil zeigte `MakeNum` als letzten dominanten Posten.

**Änderung:** `lib/types.go`: Small-Int-Cache von `-128..127` auf
`-32768..32767` (int16-Bereich) verbreitert.

```go
const smallIntMin = -32768
const smallIntMax = 32767
```

**Ergebnis (fib 25):**
```
3 allocs/op, 641 B/op, ~88 ms/op
```

Die verbleibenden 3 Allokationen pro Durchlauf liegen im Test-Harness /
Runtime-Overhead, nicht mehr im Interpreter selbst.

**Kosten:** ca. 3 MB statischer Speicher für 65 536 vorallozierte
Number-Cells. Für typische Workloads akzeptabel; bei Bedarf lässt sich der
Bereich später noch dynamisch oder zweistufig gestalten.

**Lektion:** Caching ist trivial, solange die Werte unveränderlich sind.
NUMBER-Cells werden nach der Erzeugung nie mutiert, daher ist ein
bereits vorhandener Cache-Eintrag semantisch identisch mit einer frischen
Allokation — sogar für `equal?`.

---

## 5. ⚠️ Korrigierte Denkmodelle — NICHT wiederholen

```
go tool pprof -top -sample_index=alloc_objects -nodecount=12 mem.prof
```

| Posten | Anteil | Quelle |
|--------|--------|--------|
| `evalArgs` | **40 %** | ein `[]*Cell`-Slice pro Aufruf |
| `NewEnv` + `Env.Set` | **35 %** | eine `map[string]*Cell` pro Frame + Buckets |
| `MakeNum` + `MakeAtom` | **24 %** | Zahl-/Atom-Boxing in den Primitiven |

Nach Schnitt 1 ist der `MakeAtom`-Anteil (~6 %, die wahren `<`-Fälle) weg.
Die zwei dicken Brocken — `evalArgs` (40 %) und Env (35 %) — sind beide
„**eine** Allokation pro Aufruf". Nur strukturell zu killen, nicht durch
Kosmetik.

---

## 4. Was erledigt ist — Schnitt 1 (t/nil-Singletons) ✅

In `lib/types.go` (package-level):
```go
var cellT   = &Cell{Type: ATOM, Val: "t"}
var cellNil = &Cell{Type: NIL}
```
Alle **boolean-rückgebenden** Primitiven in `lib/primitives.go` geben jetzt
`cellT`/`cellNil` statt frischer `MakeAtom("t")`/`MakeNil()`:
`fnEq, fnLt, fnGt, fnGe, fnLe, fnEqual, fnEqPtr, fnAtom, fnNull, fnStringP,
fnNumberP, fnListP, fnSymbolP`.

**Sicherheit verifiziert:** grep nach `.Val=`/`.Num=`/`.Type=` (echte
Zuweisungen) ergab nur `eval_specialforms.go:243 lam.Type = MACRO` — ein
frisches Lambda, nie ein Singleton. Keine destruktiven Listen-Ops
(`rplacd`/`nconc`) im Code. Singletons sind sicher. Bei `t` macht der
Singleton `eq` sogar *korrekter*.

**Lektion:** Der fib-Bench fängt Fehler in `=`/`>` NICHT (fib nutzt nur `<`).
Beim Umstellen schlichen sich zwei `cellNil`-statt-`cellT`-Tippfehler in
`fnEq`/`fnGt` ein, gefunden durch Drüberschauen + `./golisp2 -t`. Also nach
JEDEM Schnitt **beides**: `./golisp2 -t` (Korrektheit) UND der Bench (Tempo).

---

## 5. ⚠️ Korrigierte Denkmodelle — NICHT wiederholen

**Phantom-Optimierung „evalArgs vorzählen":** Ich (Claude) behauptete,
`append` aus `nil` würde bei 2 Elementen *zweimal* allozieren, und
`make([]*Cell, 0, n)` spare eine Allokation. **Falsch.** `allocs/op` zählt
**Heap-Objekte**, nicht Slice-Wachstumsschritte. Ob `append` intern ein- oder
zweimal umkopiert — am Ende steht **ein** Backing-Array auf dem Heap. Die Zahl
blieb bit-identisch (1 942 278), durch drei Läufe hindurch. → **Vorzähl-Code
wieder entfernt**, `evalArgs` ist zurück auf die schlanke Variante.

> Merksatz fürs nächste Mal: Eine Zahl, die sich auf die letzte Ziffer NICHT
> rührt, ist fast nie ein schwacher Effekt — sie ist nicht-ausgeführter oder
> wirkungsloser Code. Echte Optimierungen rauschen. Bit-Identität = derselbe
> Heap-Footprint. **Das Profil hat recht, nicht die Kopfrechnung** — auch die
> des Assistenten, der die Disziplin predigt.

**Konsequenz:** Der `evalArgs`-Slice ist NUR zu killen, indem man gar keinen
Slice erzeugt (direktes Binden, siehe Schnitt 3).

---

## 6. Plan — Status und nächste Schnitte

Schnitt 2–6 sind umgesetzt und verifiziert. `fib 25` allokiert nur noch
3 Objekte pro Durchlauf (Test-Harness-Overhead).

### Schnitt 2–6 ✅ (erledigt)

- **Schnitt 2:** slice-basiertes Frame-Env mit map-Root (`lib/env.go`).
- **Schnitt 3:** direktes Argument-Binding ohne `[]*Cell`-Zwischenslice
  (`bindEvalArgs` in `lib/eval_lambda.go`, Lambda-Apply in `lib/eval_core.go`).
- **Schnitt 4:** `sync.Pool` für FUNC-Arg-Slices (`evalArgsPooled` in
  `lib/eval_core.go`).
- **Schnitt 5:** Frame-Env-Pool + Tail-Call-Freigabe (`envPool`, `shared`,
  `ownEnv`-Tracking in `lib/eval_core.go`; `defer freeEnv` in
  `lib/eval_control.go` für `do`/`flet`/`labels`).
- **Schnitt 6:** Small-Int-Cache auf int16-Bereich `-32768..32767`
  verbreitert (`lib/types.go`).

### Schnitt 7 ✅ (erledigt, 2026-08-05)

Rücknahme von Schnitt 5: `envPool`/`shared`/`ownEnv` entfernt, weil das
`shared`-Bit die transitive Erreichbarkeit von Closures nicht ausdrücken
kann. Details in §4.5b.

### Was jetzt? — `evalCtx.child()` ist der Elefant

Profil vom 2026-08-05 (`-sample_index=alloc_objects`, nach Schnitt 7):

| Posten | Anteil | Objekte |
|--------|--------|---------|
| `(*evalCtx).child` | **84,9 %** | 21 823 932 |
| `NewEnv` | 14,8 % | 3 810 846 |
| alles andere | 0,3 % | — |

`ectx.child()` allokiert pro **Nicht-Tail**-Auswertung ein `&evalCtx{depth,
ctx}` — 24 Byte, ~5,5 Stück pro fib-Aufruf. Das Depth-Limit- und
Cancellation-Feature ist damit der dominante Allokationsposten des
Interpreters, mit fast sechsfachem Volumen gegenüber den Frame-Envs, deren
Pooling sich Schnitt 5 ein Korrektheitsloch kosten ließ.

**Nächster Schnitt: `evalCtx` von der Halde nehmen.** Kandidaten, in
aufsteigender Invasivität:

1. `evalCtx` **per Value** durchreichen statt per Pointer — `depth int` +
   `ctx context.Context` sind 24 Byte, das kopiert sich billiger als es
   allokiert. `child()` wird `ectx.childVal()` und gibt einen Wert zurück.
   Braucht eine Signaturänderung über den halben Interpreter — Blast-Radius
   vor dem Anfangen prüfen.
2. Nur allokieren, wenn `ctx != nil`: die Tiefe allein könnte als `int`-Parameter
   mitlaufen. Die meisten Läufe haben gar keinen Context.
3. Escape-Analyse prüfen: `go build -gcflags='-m'` auf `child()` — vielleicht
   entkommt es nur wegen einer einzigen Speicherung.

Erst messen, welche der drei greift. Erwartung vor dem Lauf formulieren
(siehe §9), sonst wiederholt sich das Phantom aus §5.

Danach ist der `fib`-Mikrobenchmark am Ende seines Aussagegehalts.
Weitere Reduktionen bräuchten einen anderen Ansatz:

- **Bytecode-VM / Compiler:** Tree-Walking-Overhead pro Aufruf reduzieren.
- **NaN-Boxing:** NUMBER-Werte ohne Heap-Cell direkt im Pointer kodieren.
- **Andere Workloads benchmarken:** z.B. `sum-acc`, String-Operationen,
  Makro-Expansion, KI-Calls — dort dominieren andere Posten.

Empfohlener nächster Schritt: einen weiteren Bench definieren, bevor ein
neuer Schnitt angelegt wird.

---

## 7. Datei-Landkarte

| Datei | Was drin steckt |
|-------|-----------------|
| `lib/eval_core.go` | `Eval`-Trampolin, `evalCtx`/`child()`, `evalArgsPooled`, Lambda-Apply-Pfad |
| `lib/env.go` | `Env`-Struct (Root-Map + Slice-Frames), Redefine-Policy |
| `lib/env_closure_test.go` | Regressionsnetz Closure × Frame-Form (§4.5b) |
| `lib/eval_lambda.go` | `bindArgs`, `bindEvalArgs`, `applyLambda`, `makeLambda` |
| `lib/eval_control.go` | `do`/`while`/`flet`/`labels`/`block`/`return-from`/`catch`/`eval` |
| `lib/primitives.go` | `BaseEnv` (Root-Env-Aufbau), `MakeNum`-Boxing in fnAdd/fnSub… |
| `lib/types.go` | `Make*`-Konstruktoren, `cellT`/`cellNil`-Singletons, small-int-cache |
| `lib/fibBench_test.go` | der Benchmark (liegt in `lib/`) |

`bindArgs`-Signatur aktuell: `bindArgs(params, args []*Cell, closureEnv, localEnv *Env) error`.

---

## 8. Bench- & Test-Kommandos

```bash
cd lib
go build . && ../golisp2 -t                              # Korrektheit (40 Tests)
go build . && go test -count=1 -run=^$ -bench=Fib -benchmem
# Profil bei Bedarf:
go test -count=1 -run=^$ -bench=Fib -benchmem -memprofile=mem.prof
go tool pprof -top -sample_index=alloc_objects -nodecount=12 mem.prof
```

`-count=1` killt den Test-Cache (sonst wird ein gecachter Lauf gezeigt — war
heute kurz ein Verdacht). Projekt war unter `$GOPATH` → `go.mod`-Warnung;
bei Build-Zweifeln `unset GOPATH` oder Projekt aus dem GOPATH ziehen.

---

## 9. Arbeitshaltung (für die neue Instanz)

- **Messen, nicht raten.** Bei jeder Hypothese erst das Profil/den Bench
  fragen. Ich bin heute selbst in die Raten-Falle getappt (§5) — das Profil
  war jedes Mal der Schiedsrichter.
- **Ein Schnitt, ein Test, eine Zahl.** Steenberg-Disziplin: kleinster
  messbarer Schritt zuerst, konstante Velocity.
- **Erwartung vor dem Lauf formulieren** (Modell), dann mit der Messung
  vergleichen. Decken sich Modell und Messung, verstehen wir die Maschine —
  das ist wertvoller als der einzelne Prozentpunkt.
- Korrektheit (`-t`) UND Tempo (Bench) nach jedem Schnitt — der Bench prüft
  keine Semantik.
- Gerhards Stil: 2-Space-Einrückung (kein gofmt-Tab!), camelCase, kurze
  Bezeichner, sparsame Kommentare, sein Datei-Header.

**Nächster konkreter Schritt:** Neuen Benchmark definieren (z.B. `sum-acc`,
String-Operationen, Makro-Expansion), bevor ein weiterer Schnitt angelegt wird.
