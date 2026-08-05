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
| **Schnitt 8** (Symbol-Interning — Korrektheit) ✅ | 1 578 205 | 59 MB | 148 ms |
| **Schnitt 9** (evalCtx per Value statt Pointer) ✅ | 242 833 | 27 MB | 115 ms |
| **Schnitt 10** (Env-Struct 112 → 88 B) ✅ | 242 828 | 23 MB | 113 ms |
| **Schnitt 11** (Env 80 B + Symbol-Keyed Lookup) ✅ | 242 819 | 19 MB | 109 ms |
| entries-Startkapazität ❌ (netto negativ, siehe §3) | — | — | — |

**Aktuelle Baseline: `242 819 allocs/op`, 19,4 MB/op, ~109 ms/op.**

Die 242 819 sind **ein `NewEnv` pro fib-Aufruf** (242 785 Aufrufe) und machen
95,8 % der Allokationen aus. Die Zahl der Allokationen ist damit am Boden für
diese Architektur — ein Frame pro Aufruf ist das Minimum, solange Frames auf
dem Heap liegen. Was noch geht, ist ihre **Größe**: siehe §4.5e und §6.

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

## 3. Zweiter Benchmark-Satz (evalBench_test.go, 2026-08-05)

`fib` misst genau einen Fall: einstelliger Lambda-Aufruf, Zahlen aus dem
Small-Int-Cache, keine Cell-Allokation, keine Strings, keine Makros, keine
Closures. Es ist damit blind für die Pfade, in denen die Korrektheitsfehler
dieses Tages sassen (Closure × Frame, §4.5b) und für `Cell` (104 B), den
nächsten Größen-Kandidaten.

Baseline nach Schnitt 11, Ryzen 5 5500:

| Benchmark | ns/op | B/op | allocs/op | allocs pro Lisp-Op |
|---|---|---|---|---|
| `ListBuild` (1000 cons + traverse) | 1 183 257 | 398 543 | 4 971 | ~2,5 |
| `MultiArgLambda` (2000 × 4-stellig) | 2 270 655 | 576 258 | 12 002 | 6,0 |
| `ClosureCreate` (2000 Closures) | 1 956 416 | 958 563 | 11 986 | 6,0 |
| `MacroExpand` (2000 Expansionen) | 1 907 175 | 1 040 325 | 14 002 | 7,0 |
| `StringOps` (500 string-append) | 351 868 | 244 009 | 2 027 | ~4,0 |
| `LetChain` (2000 × 2 verschachtelte let) | 1 727 591 | 512 210 | 8 002 | 4,0 |

**Erster Befund, den `fib` nie zeigen konnte:** der `entries`-Slice eines
Frame-Envs startet bei `nil`, und Gos `append`-Wachstum (1→2→4→8) kostet
⌈log₂(n)⌉+1 Allokationen für n Einträge statt einer. Gemessen über die
Parameterzahl:

| params | entries | allocs/op | B/op |
|---|---|---|---|
| 1 | 0 | 1 | 80 |
| 2 | 1 | 2 | 96 |
| 3 | 2 | 3 | 128 |
| 4 | 3 | 4 | 192 |
| 5 | 4 | 4 | 192 |
| 8 | 7 | 5 | 320 |
| 9 | 8 | 5 | 320 |

`fib` hat genau einen Parameter, also null `entries` — deshalb war der
Posten elf Schnitte lang unsichtbar. Der Aufrufer kennt die Parameterzahl
(`bindArgs`/`bindEvalArgs`) bzw. die Bindungszahl (`let`/`let*`), also
schien es vermeidbar. **Ist es nicht** — der Versuch ist gemessen und
abgelehnt, siehe direkt unten.

---

### ❌ Abgelehnt: feste Startkapazität für `entries` (gemessen 2026-08-05)

**Hypothese:** der `entries`-Slice wächst aus `nil` (1→2→4→8) und kostet
⌈log₂(n)⌉+1 Allokationen für n Einträge. Eine feste Startkapazität sollte
das auf eine Allokation drücken.

**Umgesetzt** als `entriesInitCap` in `SetSym`, drei Werte gemessen
(MultiArgLambda / ClosureCreate / LetChain, B/op):

| cap | MultiArg allocs | MultiArg B | Closure B | LetChain B |
|---|---|---|---|---|
| **wachsen aus nil** | 12 002 | 576 258 | **958 563** | **512 210** |
| 2 | 10 002 | 576 270 | 990 588 | 544 231 |
| 3 | **8 002** | **512 267** | 1 022 616 | 576 255 |
| 4 | 8 002 | 576 301 | 1 054 647 | 608 277 |

**Ergebnis: netto negativ, revertiert.** `cap=3` gewinnt bei
`MultiArgLambda` (−33 % allocs, sogar −11 % Bytes, weil 3 × 16 B genau in
die 48-B-Klasse passt), verliert aber bei `ClosureCreate` (+6,7 %) und
`LetChain` (+12,5 %) — und die stehen für den Alltag.

**Warum es strukturell nicht aufgeht:** der häufigste Frame hat **einen**
`entries`-Eintrag — jede zweistellige Funktion, also das gesamte
`(n acc)`-Akkumulator-Idiom. Dieser Fall kostet beim Wachsen aus `nil`
bereits genau eine Allokation; da ist nichts zu holen. Eine feste
Kapazität kann dort nur Bytes verschwenden und erst ab drei Einträgen
(vierstellige Funktionen) Allokationen sparen. Vier von sechs Benchmarks
werden von zweistelligen Funktionen dominiert.

**Was stattdessen gehen würde:** die *exakte* Zahl reservieren. Frei
verfügbar ist sie nur in `applyLambda` (`len(args)`) — dem funcall/apply-Pfad,
nicht dem Trampolin-Hauptpfad. Für `bindEvalArgs` müsste die Parameterliste
ein zweites Mal abgelaufen werden, was der häufigste Fall (ein Parameter,
`entries` bleibt `nil`) ohne Nutzen mitbezahlt. Die Parameterzahl auf der
LAMBDA-Cell zu cachen würde `Cell` (104 B) für alle Cells vergrößern.

**Lektion:** Ein Benchmark, der eine Optimierung motiviert, taugt nicht als
ihr Beleg. `MultiArgLambda` hat den Posten aufgedeckt und hätte den Schnitt
auch bestätigt — erst die fünf anderen zeigten, dass er im Alltag verliert.
Zweite Instanz derselben Regel wie §5.

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

## 4.5c Schnitt 8: Symbol-Interning ✅ — und was Schnitt 1 falsch gemacht hat

**Anlass:** wieder Korrektheit, nicht Tempo. `eq` ist Pointer-Identität,
aber `MakeAtom` gab bei jedem Aufruf eine frische Cell zurück. Damit war
`(eq 'foo 'foo)` → `()`, wo CL `T` sagt — und zwar für **jedes** Symbol.

**Schnitt 1 war eine Optimierung, die versehentlich Semantik verändert hat.**
`cellT`/`cellNil` waren ein Ad-hoc-Interning für genau zwei Werte. §4 notiert
den Nebeneffekt sogar positiv („Bei `t` macht der Singleton `eq` sogar
*korrekter*") — aber nur an den 14 Stellen, die die Singletons benutzten.
`member`/`assoc` lieferten weiter `MakeNil()`. Ergebnis: zwei NIL-Instanzen,
und `eq` war je nach Primitiv verschieden. Halb-korrekt ist bei Identität
schlechter als konsistent falsch — konsistent falsch kann man dokumentieren.

**Änderung:** `internTable sync.Map` in `lib/types.go`; `MakeAtom` liefert pro
Namen dieselbe Cell. `cellNil` gelöscht (14 Stellen → `MakeNil()`), `cellT`
bleibt als Direktreferenz für heiße Pfade, kommt aber AUS der Tabelle
(`var cellT = MakeAtom("t")`) — eine Quelle, nicht zwei.

**Vorher geprüft, weil Interning Cells teilt:** kein ATOM wird in-place
mutiert. Quellpositionen stempeln `reader.go:130` (auf einer `Cons`-Zelle) und
`eval_load.go:54` (`if expr.Type == LIST`) ausschließlich auf Listen;
destruktive Listen-Ops (`rplaca`/`nconc`) existieren nicht.

**Ergebnis:** alle sechs `eq`-Divergenzen aus TODO 4.1 behoben, gegen SBCL
verifiziert. Zwei bewusste Ausnahmen bleiben: Zahlen (`(eq 5 5)` → `()`) und
Strings (`(eq "a" "a")` → `()`).

```
allocs/op:  1 578 206 → 1 578 205   (bit-identisch)
B/op:            59 MB → 59 MB
ns/op:          142,4 → 148,1 ms   (+4,0 %)
```

**⚠️ Die +4 % sind NICHT erklärt.** A/B in derselben Session (`git stash`,
je 5 Läufe, Mediane) — die Ranges überlappen kaum, es ist kein Rauschen. Aber:

- allocs/op ist **bit-identisch**. Nach dem Merksatz aus §5 heißt das:
  der neue Code wird auf diesem Pfad nicht ausgeführt.
- Das CPU-Profil bestätigt es. `MakeAtom` erscheint in den Top 40 **nicht**.
  `mapaccess2_faststr` (6,3 %) ist `Env.Get` auf dem Root-Env, die
  `sync.Pool`-Posten sind der Arg-Slice-Pool — beides vorher schon da.
  Keine einzige `sync.Map`-Operation im Profil.

`fib` erzeugt keine Symbole, kann diese Änderung also gar nicht messen. Die
wahrscheinlichste Erklärung ist ein Code-Layout-Effekt (verschobene
Funktionsadressen → i-Cache/Branch-Alignment im Interpreter-Loop), aber das
ist **eine Hypothese, keine Messung**. Wer es wissen will, braucht einen
lese-/makrolastigen Benchmark — steht in §6 sowieso auf der Liste.

Nicht dieselbe Falle wie §5 aufmachen: die Zahl steht hier, die Ursache ist
offen, und sie wird nicht wegerklärt.

**Lektion:** Eine Optimierung, die Identität anfasst, ändert Semantik — auch
wenn sie nur schneller sein will. `cellT`/`cellNil` waren als Allokations-
Ersparnis gedacht und haben `eq` sechs Wochen lang inkonsistent gemacht,
ohne einen einzigen Test zu brechen.

**Regressionsnetz:** `lib/intern_test.go` — 15 Interning-Fälle, 3
Negativfälle, 3 Charakterisierungstests für die bewussten Abweichungen,
plus ein Nebenläufigkeitstest (32 Goroutinen × 16 Symbole, mit `-race`).

---

## 4.5d Schnitt 9: evalCtx per Value ✅ — der Elefant aus §6

**Ziel:** `(*evalCtx).child()` allokierte pro Nicht-Tail-Auswertung ein
`&evalCtx{depth, ctx}` — 24 Byte, ~5,5 Stück pro fib-Aufruf, 84,9 % aller
Allokationen des Interpreters.

**Analyse vor dem Umbau:** `evalCtx` wird nach der Erzeugung **nie**
mutiert (`rg '\.(depth|ctx)\s*=[^=]'` → leer außer den Literalen). `child()`
baut immer eine neue Instanz. Damit ist by-value semantisch identisch, und
der Wert (int + Interface = 24 Byte, 3 Words) passt in Register — die
Go-Register-ABI nimmt bis zu 9 Words, `evalWithCtx` braucht mit 2 Pointern
insgesamt 5.

**Änderung:** `ectx *evalCtx` → `ectx evalCtx` an 63 Stellen in 8 Dateien,
Methoden von Pointer- auf Value-Receiver, 7 `&evalCtx{…}` → `evalCtx{…}`.
Die defensiven `e == nil`-Zweige in `child()`/`check()` fallen weg (ein
Value ist nie nil), ebenso `ectx != nil` in `evalParfunc`.

**Ergebnis (fib 25):**
```
             vorher       nachher      Delta
allocs/op    1 578 205    242 833      −84,6 %
B/op         59,27 MB     27,20 MB     −54,1 %
ns/op        145,6 ms     114,6 ms     −21,3 %
```

Die Rechnung schließt sich: 1 578 205 − 242 833 = 1 335 372, und das ist
praktisch genau die „Nachmessung 2026-08-05" von 1 335 430. Die
undokumentierte Regression zwischen Schnitt 6 und heute **war** `evalCtx`,
vollständig und ausschließlich.

Im Profil erscheint `evalCtx` nicht mehr. Neuer Spitzenposten: `NewEnv` mit
95,84 %.

**Nebeneffekt:** die 3-Mio-Tail-Call-Schleife läuft in 2,05 s statt 2,48 s
(−17 %). Tail-Calls erhöhen die Tiefe nicht, allokierten aber trotzdem einen
Kontext pro Argument-Auswertung.

**Netz:** `lib/eval_depth_test.go`, sieben Fälle, VOR dem Umbau committet
(`f331b24`). Bewusst nur über die öffentliche API, damit die
Signaturänderung an `evalWithCtx`/`child` sie nicht berührt — ein Netz, das
beim Refactoring mitgeändert werden muss, sichert nichts ab. Es hält die
Tiefen-Rücksetzpunkte fest (`eval`, `funcall`/`apply`), die Tail-Freiheit
und die ctx-Weitergabe an `child()`.

**Lektion:** Drei Optimierungen in dieser Datei haben Semantik verändert
(Schnitt 1 Identität, Schnitt 5 Frame-Lebensdauer) oder waren teuer, weil
etwas per Pointer durchgereicht wurde, das per Value gehört. Der
gemeinsame Nenner: ein Pointer signalisiert „geteilter, veränderlicher
Zustand". `evalCtx` war beides nicht. Wer einen Pointer nimmt, wo ein Wert
genügt, bezahlt eine Heap-Allokation für eine Aussage, die nicht stimmt.

---

## 4.5e Schnitt 10: Env-Struct verkleinern ✅ — und die Size-Class-Falle

**Ziel:** `NewEnv` war nach Schnitt 9 mit 95,8 % der dominante Posten. Ein
Frame-Env pro Aufruf, 112 Byte.

**Was NICHT ging.** §6 schlug vor, den `sync.RWMutex` (24 B) aus den
Frame-Envs zu nehmen, weil ein Frame „per Definition genau einer Goroutine
gehört". **Das ist falsch.** `evalParfunc` gibt sein `env` direkt an jede
Worker-Goroutine (`eval_control.go`). Steht ein `parfunc` in einem `let`
oder Lambda-Body, teilen N Goroutinen einen FRAME-Env — und sie schreiben
ihn:

```
(let ((s 42)) (parfunc r (+ s 1) (+ s 2) (+ s 3)) r)  =>  (43 44 45)
(let ((c 0))  (parfunc r (setq c (+ c 1)) (setq c (+ c 1))) c)  =>  2
```

Der Race-Detektor ist genau deshalb sauber, weil der Mutex da ist. Ihn zu
entfernen wäre ein Data Race — die schlechtere Version des Fehlers aus
§4.5b. Der Mutex bleibt, mit dieser Begründung als Kommentar im Struct.

**Was ging:** `names []string` + `vals []*Cell` (2×24 B) zu einem
`entries []envEntry{name, val}` (24 B) zusammenlegen. Halbiert außerdem die
Slice-Allokationen bei Frames mit mehr als einer Bindung.

**Ergebnis (fib 25) — und ein falsches Modell:**

| | Vorhersage | Messung | |
|---|---|---|---|
| `unsafe.Sizeof(Env{})` | 88 B | 88 B | ✓ |
| allocs/op | unverändert | 242 828 | ✓ |
| B/op | 21,4 MB (−21,4 %) | **23,32 MB (−14,3 %)** | ✗ |
| ns/op | −2 bis −5 % | **−1,7 %** | knapp drunter |

Die Bytes lagen 2 MB daneben, weil ich gegen die **Struct-Größe** geplant
habe statt gegen die **Size-Class** des Allocators. Go rundet 88 B auf die
96-B-Klasse:

```
242 828 × 88  = 21,37 MB   (Struct-Groesse — falsches Modell)
242 828 × 96  = 23,31 MB   (Size-Class — passt zu 23,32 gemessen)
242 828 × 112 = 27,20 MB   (vorher; 112 IST eine Size-Class, daher exakt)
```

Reale Ersparnis also 16 B pro Allokation, nicht 24. Dass die Rechnung
vorher aufging, war Zufall: 112 ist selbst eine Size-Class.

**Lektion:** `unsafe.Sizeof` ist nicht die allozierte Größe. Go rundet auf
Size-Classes (… 64, 80, 96, 112, 128 …). Eine Struktur um 8 Byte zu
verkleinern bringt nur etwas, wenn sie dabei eine Klassengrenze
unterschreitet. Vor dem nächsten Schnitt an einem Struct: erst die Klasse
ausrechnen, dann entscheiden.

**Nächster Hebel wäre die 80-B-Klasse** — 8 Byte weniger, dann
242 828 × 80 = 19,4 MB (weitere −17 %). Genau 8 Byte liegen brach: das
`vars`-Map-Feld ist in Frame-Envs immer nil, `singleName`/`singleVal`/
`entries` sind im Root-Env immer leer. Das Struct bezahlt beide Modi. Das
aufzulösen heißt `Env` in zwei Typen zu trennen — `Env` ist exportiert und
wird von `lib/swank` und `main.go` benutzt, also erst Blast-Radius prüfen.

**Netz:** `lib/env_frame_test.go`, sechs Fälle, vor dem Umbau grün.
`env_test.go` deckte nur Root-Env und Redefine-Policy ab — der Slice-Pfad
und der Übergang inline → Slice waren untestet, obwohl jede Lambda mit mehr
als einem Parameter dort landet. Festgehalten werden: der Übergang, Set
überschreibt ohne Duplikat auf beiden Pfaden, Update trifft beide und steigt
zum Parent auf, Shadowing schreibt nicht durch, `Symbols` dedupliziert über
die Kette, und 64 Bindungen gegen einen append-Reallocation-Bug.

---

## 4.5f Schnitt 11: Env auf 80 B + Symbol-Keyed Lookup ✅

**Ziel:** die in §4.5e identifizierten 8 Byte holen, um in die
80-Byte-Size-Class zu kommen. Diesmal die Klasse gerechnet, nicht die
Struct-Größe.

**Änderung:** `singleName string` (16 B) → `singleSym *Cell` (8 B), und
`envEntry.name string` → `envEntry.sym *Cell` (Eintrag 24 → 16 B). Möglich
erst seit dem Symbol-Interning (§4.5c): pro Name existiert genau eine Cell,
also ist Pointer-Gleichheit äquivalent zu Namensgleichheit.

Dazu `GetSym`/`SetSym` als Pointer-Variante. Die öffentliche String-API
(`Get`/`Set`/`Update`) bleibt unverändert — `lib/swank` und `main.go` sind
nicht betroffen. Beide Wege schreiben dasselbe Feld und sind beliebig
mischbar; genau das prüft `TestFrameSymAPIInteropWithStringAPI`.

**Zweistufig gemessen, weil die Effekte gegenläufig sind:**

| | ns/op | B/op | allocs/op |
|---|---|---|---|
| Ausgangspunkt (Schnitt 10) | 112,7 ms | 23,32 MB | 242 828 |
| **A** nur Feld verkleinert | 117,4 ms **(+4,2 %)** | 19,43 MB (−16,7 %) | unverändert |
| **B** + heiße Pfade auf `*Sym` | **109,0 ms (−3,3 %)** | 19,43 MB (−16,7 %) | unverändert |

Schritt A ist **allein eine Verschlechterung**: `Set(name string)` muss den
Namen jetzt internieren, das sind 242k zusätzliche `sync.Map`-Suchen. Erst
die Umstellung der fünf heißen Aufrufstellen dreht das um. Wer nur A
gemessen und aufgehört hätte, hätte den Schnitt verworfen.

Umgestellt wurden genau fünf Stellen — dort liegt die internierte Cell schon
vor, es wurde also bisher unnötig `.Val` extrahiert:

- `eval_core.go` ATOM-Auswertung: `env.Get(expr.Val)` → `env.GetSym(expr)`
- `eval_core.go` `let` und `let*`: `Set(b.Car.Val, …)` → `SetSym(b.Car, …)`
- `eval_lambda.go` `bindArgs` und `bindEvalArgs`, reguläre Parameter

`&optional`/`&key` laufen weiter über die String-API: dort kommt der Name
aus `parseParamSpec` als String, das umzustellen hieße die Hilfsfunktion
mitzuändern. Kein Hot-Path-Fall, bewusst gelassen.

**Modell gegen Messung:** diesmal alle drei getroffen — `Sizeof(Env)` = 80 B
exakt, B/op = 19,43 MB exakt (242 828 × 80), ns/op mit −3,3 % im
geschätzten Band von −1 bis −5 %. Der Unterschied zu §4.5e ist, dass dort
gegen die Struct-Größe gerechnet wurde und hier gegen die Size-Class.

**Lektion:** Zwei Änderungen mit gegenläufigem Vorzeichen gehören getrennt
gemessen, auch wenn sie zusammen committet werden. Die Zwischenmessung ist
der einzige Grund, warum hier belegbar ist, *warum* der Schnitt wirkt —
und sie hätte einen Abbruch nach der falschen Hälfte verhindert.

**Nebeneffekt fürs Interning:** Pointer-Keyed Lookup verlangt, dass JEDES
Symbol interniert ist. Der Scan nach `&Cell{Type: ATOM …}` fand dabei die
letzte Ausnahme — `evalAnd` baute sein `(and)`-Ergebnis direkt. Separat
gefixt (`e396f4e`), war eine echte `eq`-Divergenz gegen SBCL.

**Netz:** `lib/env_frame_test.go` um zwei Fälle erweitert (Sym/String-Interop
in beiden Richtungen, `GetSym`-Durchfall bis ins Root-Env). Die sechs
bestehenden Fälle nutzen die String-API und blieben unverändert grün — genau
das soll ein Netz bei einem Refactoring tun.

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

### Schnitt 7 + 8 ✅ (erledigt, 2026-08-05)

Beide aus Korrektheitsgründen, nicht wegen Tempo:

- **Schnitt 7:** Rücknahme von Schnitt 5 — `envPool`/`shared`/`ownEnv`
  entfernt, weil das `shared`-Bit die transitive Erreichbarkeit von
  Closures nicht ausdrücken kann. §4.5b. (Korrektheit, kostete +2,3 %)
- **Schnitt 8:** Symbol-Interning — Rücknahme des Ad-hoc-Interning aus
  Schnitt 1. §4.5c. (Korrektheit, kostete +4,0 %)
- **Schnitt 9:** `evalCtx` per Value statt Pointer. §4.5d.
  (Performance: −84,6 % allocs, −21,3 % ns)
- **Schnitt 10:** Env-Struct 112 → 88 B. §4.5e.
  (Performance: −14,3 % B/op, −1,7 % ns; Modell lag daneben, s. dort)
- **Schnitt 11:** Env 80 B + Symbol-Keyed Lookup. §4.5f.
  (Performance: −16,7 % B/op, −3,3 % ns; Modell traf alle drei Größen)

Netto gegenüber der Nachmessung vom 2026-08-05: **139 ms → 115 ms** bei
**1 335 430 → 242 819 allocs/op**, **32 → 19,4 MB/op**, **139 → 109 ms/op**.
Zwei Fehlerklassen weg *und* 21,6 % schneller.

### Was jetzt? — `Env` ist ausgereizt, `Cell` ist der nächste Kandidat

Stand nach Schnitt 11: `NewEnv` bleibt mit ~95 % der dominante Posten, aber
sowohl Anzahl (ein Frame pro Aufruf = Minimum) als auch Größe
(80 B = Size-Class-Grenze, exakt getroffen) sind ausgereizt. Weniger geht nur
noch mit einer Typtrennung Root/Frame — `vars` (8 B) liegt in Frames brach,
`singleSym`/`singleVal`/`entries` im Root. Das Struct bezahlt beide Modi,
aber die nächste Klasse darunter ist 64 B, also müssten **16** Byte weg.

**Was NICHT geht — bitte nicht erneut versuchen:**

- **Mutex aus Frame-Envs entfernen.** `evalParfunc` gibt sein `env` direkt an
  jede Worker-Goroutine; ein `parfunc` in einem `let` teilt einen FRAME-Env
  über N Goroutinen, die ihn per `setq` schreiben. Mit `-race` verifiziert,
  §4.5e.
- **Env-Pooling wiederbeleben.** Ursache für zehn kaputte Frame-Formen, §4.5b.
- **Mutex lazy allokieren.** Dieselbe Erreichbarkeitsfrage, an der das
  `shared`-Bit scheiterte — nur ist der Fehlerfall hier ein Data Race.
- **Nur die Struct-Größe rechnen.** Es zählt die Size-Class, §4.5e.
- **Feste Startkapazität für `entries`.** Gemessen, netto negativ, §3.

**Lohnender als weitere Env-Arbeit:** `Cell` ist **104 Byte** und wird
ungleich häufiger allokiert als `Env` — nur nicht in `fib`, weil der
Small-Int-Cache die Zahlen abfängt. Das Struct trägt Felder für jeden Typ
gleichzeitig (`Fn`, `Env`, `Ht`, `SrcFile`, `SrcLine`, `Car`, `Cdr`, `Val`,
`Num`), obwohl jede Cell nur einen Typ ist. `SrcFile`/`SrcLine` (24 B) werden
ausschließlich auf LIST-Cells gestempelt. Vor einem Schnitt dort aber erst
einen Benchmark bauen, der Cells tatsächlich erzeugt — Listenaufbau,
String-Operationen, Makro-Expansion.

**Und der eigentliche Punkt:** `fib` ist am Ende seines Aussagegehalts. Es
misst einen einstelligen Lambda-Aufruf mit Zahlen aus dem Cache. Alles was
diese Datei ab hier noch verbessern kann, verbessert genau das. Der nächste
sinnvolle Schritt ist ein **zweiter Benchmark**, nicht ein zwölfter Schnitt.

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
| `lib/eval_core.go` | `Eval`-Trampolin, `evalCtx` (**per Value**), `evalArgsPooled`, Lambda-Apply-Pfad |
| `lib/eval_depth_test.go` | Netz für Tiefe + Cancellation (§4.5d), nur öffentliche API |
| `lib/env.go` | `Env`-Struct 80 B, `Get`/`Set` (String) + `GetSym`/`SetSym` (Pointer), Redefine-Policy |
| `lib/env_frame_test.go` | Netz für den Frame-Pfad inline↔`entries` (§4.5e) |
| `lib/env_closure_test.go` | Regressionsnetz Closure × Frame-Form (§4.5b) |
| `lib/eval_lambda.go` | `bindArgs`, `bindEvalArgs`, `applyLambda`, `makeLambda` |
| `lib/eval_control.go` | `do`/`while`/`flet`/`labels`/`block`/`return-from`/`catch`/`eval` |
| `lib/primitives.go` | `BaseEnv` (Root-Env-Aufbau), `MakeNum`-Boxing in fnAdd/fnSub… |
| `lib/types.go` | `Make*`-Konstruktoren, `internTable` (Symbol-Interning), `nilCell`, small-int-cache |
| `lib/intern_test.go` | Regressionsnetz Symbol-Interning + `eq`-Semantik (§4.5c) |
| `lib/fibBench_test.go` | fib-Mikrobenchmark (ein Fall: 1-stelliges Lambda, gecachte Zahlen) |
| `lib/evalBench_test.go` | zweiter Satz: Cell, Multi-Arg, Closure, Makro, String, let-Kette (§3) |

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
