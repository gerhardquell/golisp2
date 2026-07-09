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

**Aktuelle Baseline, die zu schlagen ist: `1 942 278 allocs/op`.**

Faustregel zum Einordnen: `allocs/op ÷ 242 000 ≈ Allokationen pro Aufruf`.
Aktuell ~8,0. Jede Allokation ist GC-Last; weniger Objekte = nichtlinear
weniger GC-Druck (deshalb fiel die Zeit bei Schnitt 1 um 9 %, nicht nur
um „Mikrosekunden").

---

## 3. Profil-Aufschlüsselung (alloc_objects, am 2,06-M-Baseline gemessen)

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
`fnEq`/`fnGt` ein, gefunden durch Drüberschauen + `./golisp -t`. Also nach
JEDEM Schnitt **beides**: `./golisp -t` (Korrektheit) UND der Bench (Tempo).

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

## 6. Plan — die nächsten Schnitte

### Schnitt 2 (empfohlen als nächstes): slice-basiertes Frame-Env — zielt auf 35 %

Map raus, parallele Slices rein. **Kritische Nuance, sonst wird's LANGSAMER:**

- **Frame-Envs** (Lambda-Locals, `let`, `let*`, …) haben 1–3 Variablen →
  `names []string` + `vals []*Cell`, **lineare Suche**. Schlägt die Hash-Map
  bei so wenigen Einträgen *und* spart die `make(map)`-Allokation.
- **Root-Env** (`BaseEnv`, ~80+ Symbole: `+ - < fib …`) MUSS map-basiert
  bleiben. Linearer Scan über 80 Einträge pro Lookup von `+` wäre fatal.
- Lookup-Pfad bei fib: `n` trifft lokal (1–2 Vergleiche im Slice), `fib/+/-/<`
  verfehlen lokal → fallen durch zur Root-Map. Genau richtig.

→ Design: `Env` bekommt zwei Modi, oder ein Interface mit `frameEnv`
(slice) und `rootEnv` (map). Alternativ: ein Feld, das ab Schwellwert (z.B.
>8 Einträge) von Slice auf Map promotet. Einfacher zuerst: Root explizit
map, Frames explizit slice.

Betroffen: `lib/env.go` (Struct, `NewEnv`, `Get`, `Set`, `Update`, `Root`,
`Symbols`), Aufrufer von `NewEnv` in `eval_core.go` (`let/let*/lambda-apply`)
und `bindArgs` in `eval_lambda.go`.

**Erwartung:** Env ist 35 % der allocs → grob `1,94 M → ~1,25 M`.
Erst messen, dann glauben.

### Schnitt 3 (danach, invasiver): direktes Argument-Binding — zielt auf 40 %

`evalArgs` erzeugt einen `[]*Cell`, der nur dazu da ist, gleich von
`bindArgs` ins Frame-Env kopiert zu werden. Bei Lambda-Aufrufen lässt sich
der Zwischen-Slice vermeiden: Argumente direkt beim Auswerten ins (slice-)
Frame-Env schreiben. Verquickt `evalArgs`/`bindArgs` — größerer Eingriff,
deshalb NACH Schnitt 2, mit eigenem Mess-Punkt.

Achtung: eingebaute `FUNC` brauchen weiterhin einen `[]*Cell` (`fn.Fn(args)`).
Also Sonderweg nur für Lambda-Apply, FUNC-Pfad bleibt wie er ist.

### Optional später: Number-Caching — zielt auf Rest von MakeNum (~18 %)

`MakeNum` boxt jede Zahl frisch (`Num` ist float64). Kleine Ganzzahlen
0..255 als vorab-allozierte Cells cachen (wie CPythons small-int-pool).
Greift nur bei ganzzahligen Kleinwerten — bei fib (n, n-1, n-2) durchaus
relevant. Kleiner, sauberer Schnitt; aufheben für später.

---

## 7. Datei-Landkarte

| Datei | Was drin steckt |
|-------|-----------------|
| `lib/eval_core.go` | `Eval`-Trampolin, `evalArgs` (~Z.285+), Lambda-Apply-Pfad (`NewEnv` + `bindArgs`, ~Z.197) |
| `lib/env.go` | `Env`-Struct (map-basiert), `NewEnv/Get/Set/Update/Root/Symbols` |
| `lib/eval_lambda.go` | `bindArgs`, `applyLambda`, `&optional/&key/&rest`-Logik |
| `lib/primitives.go` | `BaseEnv` (Root-Env-Aufbau), `MakeNum`-Boxing in fnAdd/fnSub…, Vergleichs-Prims (jetzt cellT/cellNil) |
| `lib/types.go` | `Make*`-Konstruktoren, `cellT`/`cellNil`-Singletons |
| `lib/fibBench_test.go` | der Benchmark (liegt in `lib/`) |

`bindArgs`-Signatur aktuell: `bindArgs(params, args []*Cell, closureEnv, localEnv *Env) error`.

---

## 8. Bench- & Test-Kommandos

```bash
cd lib
go build . && ../golisp -t                              # Korrektheit (40 Tests)
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

**Nächster konkreter Schritt:** Schnitt 2 — slice-Frame-Env mit map-Root.
Design zuerst skizzieren, dann `lib/env.go` umbauen, dann messen.
