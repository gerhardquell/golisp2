# Aufgaben 20260805 — Env-Pooling entfernen (Variante B)

Ergebnis der Code-Analyse vom 20260805. Analyse-Brief archiviert unter
`todos/TODO.md-20260730-done`.

## ✅ Hauptaufgabe ERLEDIGT (Abschnitte 1-3)

Commit `0bf8caf fix(env): Env-Pooling entfernen — Closures verloren ihren
Scope`. Alle fünf Akzeptanzkriterien aus Abschnitt 2 erfüllt:

1. Alle zehn Repros SBCL-identisch — `lib/env_closure_test.go`,
   `TestClosureKeepsScopeAcrossFrameReuse` (10 Fälle) +
   `TestClosureScopeNoStackOverflow` (Subprozess-Test), 21 Tests grün.
2. `go build ./...`, `go test ./... -count=1`, `./build/golisp2 -t` →
   94 PASS / 0 FAIL — alle grün.
3. `go test -race` auf den Concurrency-Tests — grün.
4. `shared`/`freeEnv`/`envPool`/`takeEnv` aus `lib/` entfernt — per `rg`
   bestätigt, keine Treffer mehr.
5. Benchmark gemessen und dokumentiert: `PerfTODO.md` §4.5b —
   139,1 ms/op → 142 ms/op (+2,1 %, unter der 10-%-Schwelle aus Abschnitt 3
   Schritt 5) — **Entscheidung Gerhard: Pooling bleibt draußen.** Spätere,
   unabhängige Env-Optimierungen (Schnitt 8) drückten den Wert danach
   weiter auf 109 ms/op.

Abschnitte 1-3 unten sind die ursprüngliche Analyse/Plan — als Referenz
stehen gelassen, nicht mehr aktuell im Sinne von "offen". Die Repro-Tabelle
in Abschnitt 5 lebt jetzt als Testfälle in `lib/env_closure_test.go` weiter.

---

## 1. Befund

**Ein Designfehler, zehn Symptome.**

`Env.shared` (`lib/env.go:39`) ist ein Bit pro Frame. Erreichbarkeit einer
Closure ist aber transitiv über `parent`. `makeLambda` (`lib/eval_lambda.go:19`)
markiert nur den *direkt* gefangenen Frame. Fängt eine Closure einen
**Nachkommen**-Frame, bleibt der Zwischen-Frame unmarkiert, wandert über
`freeEnv` in `envPool` und wird an einen fremden Frame weitergegeben. Die
Closure zeigt danach über `parent` in einen recycelten Frame.

Zwei Fehlerbilder, beide reproduziert:

- `env: unbekanntes Symbol '<name>'` — Bindung verloren
- `fatal error: stack overflow` — der recycelte Frame zeigt zurück in die
  eigene Kette, `Env.Get` (`lib/env.go:176`) pendelt endlos. **Nicht** vom
  `recover()` in `evalWithCtx` (`lib/eval_core.go:92`) abfangbar. Prozess weg.

### Betroffene Formen

Alle neun Stellen geprüft, die einen Frame anlegen und freigeben.
Repro-Muster überall gleich: Closure über Nachkommen-Frame, dann 300 fremde
Frames erzeugen, damit der Pool ausliefert. Referenz: SBCL.

| Form | Frame-Stelle | Status |
|---|---|---|
| `let` / `let*` / `case` | `eval_core.go:221,248` (Freigabe via `takeEnv`) | **kaputt** |
| Lambda-Call, TCO-Pfad | `eval_core.go:337` | **kaputt** |
| `funcall`/`apply` (`applyLambda`) | `eval_lambda.go:30` | **kaputt** |
| `do` | `eval_control.go:78` | **kaputt** |
| `do*` | `eval_control.go:143` | **kaputt** |
| `flet` | `eval_control.go:223` | **kaputt** |
| `macrolet` | `eval_specialforms.go:370` | **kaputt** |
| `symbol-macrolet` | `eval_specialforms.go:396` | **kaputt** |
| `progv` | `eval_specialforms.go:465` | **kaputt** |
| `multiple-value-bind` | `eval_mv.go:46` | **kaputt** |
| `labels` | `eval_control.go:242` | zufällig sicher (s. u.) |
| `tagbody` `block` `while` `catch` `unwind-protect` `if` `begin` `cond` | kein eigener Frame | sicher |

`labels` kommt aus zwei unabhängigen Zufällen durch: mit Definitionen ruft
Zeile 250 `makeLambda(…, localEnv)` und markiert den eigenen Frame als
Nebeneffekt; ohne Definitionen ist der Frame leer. `flet` übergibt in Zeile 231
korrekterweise `env` statt `localEnv` (CL: flet-Funktionen sehen sich nicht) —
und ist genau deshalb kaputt. **Ein Refactoring, das flet und labels
zusammenlegt, macht labels kaputt.**

`macrolet` / `symbol-macrolet` / `progv` zeigen die wahre Reichweite: dort geht
nicht ein eingefangener *Wert* verloren, sondern lexikalisch sichtbare Makros
und dynamische Bindungen. Der Fehler ist „Closure verliert Scope", nicht
„Closure verliert Variable".

### Warum kein Test es fand

312 grüne Tests (218 Go, 94 Lisp), null Fehler. Die Suiten testen Closures und
sie testen Frame-Formen — nie das Kreuzprodukt. CodeGraph meldet für
`evalFlet`, `evalDo`, `evalDoStar`, `evalBlock` zusätzlich *keine* abdeckenden
Tests.

---

## 2. Spec

**Frame-Lebensdauer gehört dem Go-GC.** Der GC kennt Erreichbarkeit exakt —
`shared` versucht sie zu approximieren und scheitert an der Transitivität.

### Raus

- `envPool` (`lib/env.go:19-21`)
- `freeEnv` (`lib/env.go:137-149`) samt aller 20 Aufrufstellen
- `Env.shared` (`lib/env.go:39`) samt `makeLambda`-Zeile `env.shared = true`
- `takeEnv`-Closure und `ownEnv`-Tracking (`lib/eval_core.go:82-89`)
- `NewEnv`-Recycling-Zweig (`lib/env.go:122-131`) → schlichtes
  `&Env{parent: parent}`
- die tote Nullungsschleife `lib/env.go:127-129` (läuft 0 Iterationen, weil
  `e.vals` beim Eintritt bereits `[:0]` ist) fällt damit mit weg

### Bleibt unangetastet

- **TCO-Trampolin.** `expr`/`env` setzen + `continue` bleibt exakt wie es ist.
  Nur der `freeEnv`-Aufruf im Übergang fällt weg — die Tail-Position selbst
  ändert sich nicht. O(1) Stack bleibt O(1) Stack.
- Root-Env mit Map, Frame-Env mit `singleName`/`singleVal` + Slices. Die
  inline-Repräsentation ist unabhängig vom Pooling und bleibt.
- `Env.mu` / RWMutex-Verhalten für `parfunc`.
- Singleton-Nil, `MakeNil()`, `IsTruthy`, alle Chokepoints.

### Akzeptanzkriterien

1. Alle zehn Repros aus Abschnitt 5 liefern SBCL-identische Ergebnisse.
2. `go build ./...` grün, `go test ./... -count=1` grün, `./build/golisp2 -t`
   → 94 PASS / 0 FAIL.
3. `go test -race` grün auf den Concurrency-Tests.
4. Kein Vorkommen von `shared`, `freeEnv`, `envPool`, `takeEnv` mehr in `lib/`.
5. Benchmark gemessen und gegen die Baseline unten dokumentiert.

### Baseline (gemessen 20260805, AMD Ryzen 5 5500, go1.26.0)

```
go test -run '^$' -bench BenchmarkFib -benchmem -count=3 ./lib/

BenchmarkFib-12   8   139107581 ns/op   32077111 B/op   1335432 allocs/op
BenchmarkFib-12   8   139137495 ns/op   32075181 B/op   1335425 allocs/op
BenchmarkFib-12   8   139455978 ns/op   32075253 B/op   1335426 allocs/op
```

139,1 ms/op · 32,08 MB/op · 1.335.430 allocs/op · Streuung ±0,25 %.

Caveat: `fib` enthält **keine** Closures, der Pool trifft dort immer. Das ist
der Best Case für Pooling — die hier gemessene Regression ist die Obergrenze,
nicht der Alltagswert.

---

## 3. Plan

Reihenfolge ist bindend: Tests zuerst, sonst ist nicht belegbar, dass der Fix
den Fehler behebt und nicht nur verschiebt.

### Schritt 1 — Failing Tests (vor jeder Änderung)

Neue Datei `lib/env_closure_test.go`. Ein Testfall pro betroffene Form aus
Abschnitt 5, plus die `labels`- und `tagbody`-Kontrollen. Jeder Test:
Closure bauen → 300 fremde Frames erzeugen → Closure aufrufen → Ergebnis
gegen den SBCL-Wert asserten.

`go test ./lib/ -run Closure` muss **rot** sein, mit 10 Fehlern. Erst wenn
das belegt ist, weiter. Ein Test, der vorher nicht rot war, beweist nach dem
Fix nichts.

Zusätzlich: den Stack-Overflow-Fall als eigenen Test. Er crasht den
Testprozess unrecovert, muss also über ein Subprozess-Harness laufen
(`exec.Command` auf `./build/golisp2 -e …`, Exit-Code prüfen) — nicht als
In-Process-Test, sonst reißt er die ganze Suite mit.

### Schritt 2 — Pooling entfernen

In dieser Reihenfolge, weil jeder Schritt für sich kompiliert:

1. `lib/env.go`: `freeEnv` auf No-op reduzieren, `NewEnv`-Recycling-Zweig
   durch `&Env{parent: parent}` ersetzen. → Tests aus Schritt 1 müssen jetzt
   **grün** sein. Das ist der Beweis, dass Pooling die Ursache war und nichts
   anderes.
2. `lib/eval_core.go`: `takeEnv`/`ownEnv` entfernen, Tail-Übergänge auf
   direktes `env = localEnv` umstellen.
3. Die verbleibenden 18 `freeEnv`-Aufrufe in `eval_control.go`,
   `eval_specialforms.go`, `eval_mv.go`, `eval_lambda.go` löschen.
4. `freeEnv`, `envPool`, `Env.shared`, `env.shared = true` in `makeLambda`
   löschen.

### Schritt 3 — Verifikation

Per CLAUDE.md **frischer Subagent** für den Build:

```
./build.sh
go build ./...
go test ./... -count=1
go test -race -run 'Conc|Par|Goroutine|Shm' ./lib/
./build/golisp2 -t
```

Alle fünf grün, sonst zurück zu Schritt 2.

### Schritt 4 — Messen

```
go test -run '^$' -bench BenchmarkFib -benchmem -count=3 ./lib/
```

Ergebnis gegen die Baseline in `perfTodo.md` eintragen — mit dem Hinweis, dass
`fib` closure-frei ist und damit den Worst Case für den Vergleich darstellt.

### Schritt 5 — Entscheidung (Gerhard)

- Regression unter ~10 %: fertig, Pooling bleibt draußen.
- Regression darüber: Ticket für eine gezielte Re-Optimierung. Dann aber mit
  transitivem Markieren *und* der Testmatrix aus Schritt 1 als Netz — nicht
  wieder mit einem Bit pro Frame.

### Schritt 6 — Commit

Ein Commit für Schritt 1 (Tests, rot), einer für Schritt 2-4 (Fix, grün).
Trennung, damit im Log sichtbar bleibt, dass die Tests den Fehler vorher
gefangen haben.

---

## 4. Danach — Restbefunde aus der Analyse

Bewusst getrennt, nicht in denselben Commit. Reihenfolge nach Risiko.

### 4.1 Symbol-Interning ✅ ERLEDIGT (Schnitt 8, Commit folgt)

**Der Umfang war größer als hier geschätzt.** Nicht „zwei Variablen löschen":
golisp2 hat Symbole überhaupt nicht interniert. `MakeAtom` gab bei jedem
Aufruf eine frische Cell zurück, also war `eq` auf **jedem** Symbol kaputt:

```
                 (eq 'foo 'foo)  (eq 't 't)  (eq ':k ':k)
vorher:               ()             ()           ()
SBCL:                 T              T            T
```

`cellT`/`cellNil` waren ein Ad-hoc-Interning für genau zwei Werte und haben
das Verhalten **inkonsistent** gemacht, nicht korrekt.

Umgesetzt: `internTable sync.Map` in `lib/types.go`, `cellNil` gelöscht,
`cellT` kommt jetzt aus der Tabelle. Alle sechs Divergenzen der Tabelle
unten behoben, gegen SBCL verifiziert. Zwei bewusste Ausnahmen bleiben:
Zahlen (`(eq 5 5)` → `()`) und Strings (`(eq "a" "a")` → `()`).

Vorher geprüft: kein ATOM wird in-place mutiert (Quellpositionen gehen nur
auf LIST-Cells, keine destruktiven Listen-Ops). Details in PerfTODO §4.5c.
Kosten: +4,0 % ns/op, Ursache unerklärt — `fib` erzeugt keine Symbole, das
Profil zeigt `MakeAtom` gar nicht. Steht als offene Frage in PerfTODO.

Regressionsnetz: `lib/intern_test.go`. Zwei Alt-Tests (`eval_test.go`,
`primitives_test.go`) hielten das alte Verhalten fest und wurden auf die
CL-korrekte Erwartung umgestellt — der Breaking Change ist von dir
freigegeben.

<details>
<summary>Ursprüngliche Analyse (Stand vor dem Fix)</summary>

#### Zwei NIL- und zwei T-Instanzen (hoch)

`lib/types.go:51-53` — `cellNil` ist eine **zweite** NIL-Instanz neben
`nilCell`, das `MakeNil()` liefert. 14 Verwendungen in `primitives.go`.
Verletzt die CLAUDE.md-Invariante „`MakeNil()` gibt immer dieselbe Instanz
zurück".

| Ausdruck | golisp2 | SBCL |
|---|---|---|
| `(eq (= 1 2) '())` | `()` | `T` |
| `(eq (= 1 1) 't)` | `()` | `T` |
| `(eq (null '()) 't)` | `()` | `T` |
| `(eq (atom 1) 't)` | `()` | `T` |
| `(eq (member 9 '(1 2)) '())` | `t` | `T` |
| `(eq (assoc 9 '((1 2))) '())` | `t` | `T` |

Schlimmer als die Divergenz: es ist **inkonsistent**. `member`/`assoc` nutzen
`MakeNil()`, die Vergleichsprädikate `cellNil`. Gleiche Prädikatklasse,
unterschiedliches Identitätsverhalten. `if` und `null` funktionieren, weil
`IsTruthy` auf `Type` prüft — der Bug schläft, bis jemand `eq` benutzt.

Fix: `cellNil` und `cellT` löschen, alle Stellen auf `MakeNil()` bzw. einen
einzigen `t`-Singleton umstellen. **Ändert `eq`-Verhalten → Gerhards Freigabe
nötig** (Breaking Change nach CLAUDE.md).

</details>

### 4.2 `let*`-Duplikat ✅ ERLEDIGT (Variante A: Lisp-Makro gelöscht)

Lisp-Makro aus `embed/stdlib.lisp` entfernt, Go-Spezialform bleibt.
`(bound? 'let*)` → `()` wie bei allen 57 anderen Spezialformen.
`macroexpand`/`macroexpand-all` lassen `let*` jetzt unberührt — konsistent
mit `if`, `let`, `cond` und dem Rest.

**Korrektur zur Analyse unten:** die beiden Hälften haben **nicht**
divergiert. Acht Edge-Cases direkt gegen `(eval (macroexpand-all ...))`
verglichen — überall identisch. Es war Redundanz, kein Fehlverhalten. Das
Problem war der stille Duplikat-Charakter in beide Richtungen: Lisp-Hälfte
reparieren wirkt nicht, Go-Hälfte ändern macht `macroexpand-all` falsch.

**Historie:** Makro kam 2026-02-24 (`c54b7f6`, damals die einzige und
erreichbare Implementierung), Spezialform 2026-02-28 (`680407c`). Vier Tage
Abstand, dann fünf Monate Überrest.

**Warum es durchschlüpfte:** der Redefine-Guard prüft nur bestehende
FUNC-Bindungen; Spezialformen sind keine Env-Bindungen, also gab es nichts
zu überschreiben und keine Warnung.

Statt eines `let*`-Tests bewacht jetzt `TestNoLispDefineShadowsSpecialForm`
(`lib/specialform_shadow_test.go`) die **ganze Klasse**: kein Lisp-Define
darf den Namen einer Spezialform tragen. Der Test liest die 58 Namen aus
`eval_core.go`, ist also selbstaktualisierend — eine gepflegte Liste würde
veralten und dann falsche Sicherheit geben.

Verifiziert: alle vier echten `let*`-Nutzer laufen — `reduce`
(`stdlib.lisp:107`), `defstruct` (`330`, `340`), `swank:autodoc`
(`swank.lisp:275`).

<details>
<summary>Ursprüngliche Analyse (Stand vor dem Fix)</summary>

#### `let*` doppelt implementiert (hoch)

- Go-Spezialform `lib/eval_core.go:247`
- Lisp-Makro `embed/stdlib.lisp:40`

Eval nimmt die Spezialform. Das Makro ist aber nicht tot, sondern über
`macroexpand` erreichbar:

```
(macroexpand '(let* ((a 1) (b a)) b))  →  (let ((a 1)) (let* ((b a)) b))
(macroexpand-all '(let* ((a 1)) a))    →  (let ((a 1)) (begin a))
```

`macroexpand-all` liefert Code, der eine **andere** Implementierung durchläuft
als `eval` ausführt. Für ein System, dessen Kernmuster `(eval (read (sigo …)))`
und Code-Walking ist, sind Werkzeug und Laufzeit damit über `let*` uneinig.

Vorschlag: Lisp-Makro löschen (die Go-Spezialform ist TCO-fähig, das Makro
kann es nicht sein). Preis: `macroexpand-all` kann `let*` nicht mehr auflösen.
**Entscheidung Gerhard** — welche Hälfte fliegt.

*Nachtrag: Das TCO-Argument war zu stark. Die Expansion
`(let ((a 1)) (let ((b a)) (begin b)))` besteht selbst nur aus Tail-Formen,
wäre also tail-transparent. Die Spezialform spart die N Expansionsschritte
bei N Bindungen, nicht den Stack.*

</details>

### 4.3–4.5 ✅ ERLEDIGT — aber nur 4.3 als Codeänderung

Alle drei nachgeprüft. Zwei davon waren **keine Bugs**; sie sind jetzt als
bewusste Entscheidung dokumentiert, statt „behoben" zu werden.

**4.3 — toter `evalArgs`: gelöscht.** 0 Aufrufer bestätigt (die einzige
Nennung war ein Kommentar in `eval_lambda.go:164`, mitkorrigiert). Die
`NewEnv`-Nullungsschleife war mit Schnitt 7 schon weg.

**4.4 — `Env.Update`-Locking: unverändert, mit Begründung im Code.**
Kein Deadlock möglich: alle Env-Methoden laufen ausschließlich Kind→Eltern,
es gibt keinen Pfad, der ein Eltern-Lock hält und ein Kind-Lock nimmt.
Mein Verdacht auf `checkRootRedefine` (`redefguard.go:50` ruft `env.Get`)
war unbegründet — es wird von `define`/`defun`/`defmacro` **vor** `env.Set`
gerufen (`eval_specialforms.go:28,252,355`), hält also kein Lock.

Umstellen auf das `Get`-Muster wäre eine **Verschlechterung**: es öffnet ein
Fenster, in dem eine andere Goroutine die Bindung im Kind-Frame anlegen
kann, während dieser `Update` sie schon im Parent sucht. Das Fenster
existiert heute nicht. Als Kommentar an `Env.Update` festgehalten.

**4.5 — `argSlicePool`-Nullung: bewusst nicht gemacht, gemessen.**
`sync.Pool` wirft seinen Inhalt bei jedem GC weg — die Retention der alten
`*Cell`-Zeiger reicht höchstens bis zum nächsten GC, also genau bis zu dem
Zyklus, in dem sie ohnehin eingesammelt würden. A/B in einer Session
(fib 25, je 5 Läufe):

```
ohne Nullung: 144,7 · 144,7 · 145,6 · 145,6 · 146,5   Median 145,6 ms
mit Nullung:  146,7 · 147,0 · 147,2 · 147,8 · 147,9   Median 147,2 ms
                                                       → +1,1 %
```

Ranges überlappen nicht. +1,1 % auf dem heißesten Pfad für einen
GC-Zyklus Retention von ein paar Zeigern. Als Kommentar an `putArgSlice`
festgehalten, damit es niemand erneut „repariert".

<details>
<summary>Ursprüngliche Analyse (Stand vor der Prüfung)</summary>

#### Toter Code (niedrig)

- `evalArgs` (`lib/eval_core.go:401`) — **0 Aufrufer**, ersetzt durch
  `evalArgsPooled`. Löschen.
- `NewEnv`-Nullungsschleife (`lib/env.go:127-129`) — 0 Iterationen. Fällt mit
  Schritt 2 sowieso weg.

### 4.4 `Env.Update` hält Lock über die Kette (niedrig)

`lib/env.go:283-306` hält per `defer` das eigene Lock, während es zum Parent
rekursiert. `Env.Get` (`152-177`) gibt vorher frei und ist damit die richtige
Vorlage. Kein Deadlock nachgewiesen — die Ordnung ist konsistent Kind→Eltern —
aber unnötige Contention und asymmetrisch zu `Get`.

### 4.5 `argSlicePool` nullt nicht (niedrig)

`lib/eval_core.go:379,385,397` geben den Puffer ohne Nullung in den Pool.
Alte `*Cell`-Zeiger bleiben erreichbar und blockieren den GC. Nicht
korrektheitsrelevant, aber Speicher-Retention.

Hinweis: dieser Pool ist eine **andere** Baustelle als der Env-Pool und von
Variante B nicht betroffen. Argument-Slices überleben den `fn.Fn`-Aufruf
nicht — dort ist die Lebensdauer-Annahme tatsächlich gültig.

*Nachtrag: „blockieren den GC" war falsch. `sync.Pool` leert sich bei jedem
GC-Lauf, die Retention endet also mit dem Zyklus, der die Zellen sowieso
einsammelt. Zusätzlich geprüft: kein Primitiv behält den args-Slice
(`SliceToCell` kopiert in frische Cons-Zellen), es gibt also auch kein
Korrektheitsproblem durch Pool-Wiederverwendung.*

</details>

### 4.6 `symbole.csv` ✅ ERLEDIGT (Commit `e9e2169`)

Datei entfernt, `symbole.ods` ist jetzt die Quelle. Restaurierbar aus
`0bf8caf` (979 Zeilen CSV).

<details>
<summary>Ursprüngliche Analyse</summary>

#### `symbole.csv` ist keine CSV (Datenverlust, Working-Tree)

`docs/cl-konformitaet/symbole.csv` ist eine **gzip-komprimierte
Gnumeric-Datei**, keine CSV. Gnumeric hat sein natives Format über die Datei
geschrieben.

- HEAD hat 979 intakte CSV-Zeilen.
- Working-Tree entpackt zu 4035 Zeilen Gnumeric-XML — Inhalt ist da, als CSV
  aber unbrauchbar. Entpackte Kopie liegt in `tmp/symbole_entpackt.csv`.
- Nicht committet. `git checkout` verliert die Zwischenarbeit, blindes
  Committen zerstört die CSV-Historie.

**Entscheidung Gerhard** — ich weiß nicht, ob in der Gnumeric-Datei Arbeit
steckt, die du behalten willst.

</details>

---

## 5. Repros (Copy-Paste-fertig)

Jeder Fall: `BURN` erst definieren, dann Closure bauen, dann `burn` laufen
lassen, dann aufrufen. Ohne `burn` liefert der `let`-Fall statt eines Fehlers
einen `fatal error: stack overflow`.

```lisp
;; Vorspann für alle Fälle
(defun burn (k) (if (> k 0) (let ((a k) (b k)) (burn (- k 1))) 'ok))
```

| # | Form | Ausdruck (nach `(define c …)` → `(burn 300)` → `(funcall c)`) | ist | soll |
|---|---|---|---|---|
| 1 | `let` + Tail-Call | `(begin (defun h (f) f) (defun g () (let ((n 5)) (h (let ((m 6)) (lambda () (list m n))))))) ` dann `(funcall (g))` | crash / `'n'` | `(6 5)` |
| 2 | `applyLambda` | `(funcall (lambda (n) (let ((m 6)) (lambda () (list m n)))) 5)` | `'n'` | `(6 5)` |
| 3 | `flet` | `(flet ((f () 1)) (let ((n 5)) (lambda () (list n (f)))))` | `'f'` | `(5 1)` |
| 4 | `do` | `(do ((i 0 (+ i 1))) ((> i 0) (let ((n 5)) (lambda () (list n i)))))` | `'i'` | `(5 1)` |
| 5 | `do*` | `(do* ((i 0 (+ i 1))) ((> i 0) (let ((n 5)) (lambda () (list n i)))))` | `'i'` | `(5 1)` |
| 6 | `multiple-value-bind` | `(multiple-value-bind (a b) (values 1 2) (let ((n 5)) (lambda () (list n a))))` | `'a'` | `(5 1)` |
| 7 | `macrolet` | `(macrolet ((m () 1)) (let ((n 5)) (lambda () (list n (m)))))` | `'m'` | `(5 1)` |
| 8 | `symbol-macrolet` | `(symbol-macrolet ((s 1)) (let ((n 5)) (lambda () (list n s))))` | `'s'` | `(5 1)` |
| 9 | `progv` | `(progv '(*x*) '(1) (let ((n 5)) (lambda () (list n *x*))))` | `'*x*'` | `(5 1)` |
| 10 | `let*` | wie 1, mit `let*` statt `let` | dito | dito |

Kontrollen, die **grün bleiben müssen** (kein eigener Frame bzw. korrekt
markiert):

| Form | Ausdruck | soll |
|---|---|---|
| `labels` mit defs | `(labels ((f () 1)) (let ((n 5)) (lambda () (list n (f)))))` | `(5 1)` |
| `labels` ohne defs | `(labels () (let ((n 5)) (lambda () (list n))))` | `(5)` |
| `tagbody` | `(let ((r ())) (tagbody (let ((n 5)) (setq r (lambda () n)))) r)` | `5` |
| `block` | `(block b (let ((n 5)) (lambda () n)))` | `5` |
| `while` | `(let ((r ()) (i 0)) (while (< i 1) (setq i 1) (let ((n 5)) (setq r (lambda () n)))) r)` | `5` |
| `catch` | `(catch 'c (let ((n 5)) (lambda () n)))` | `5` |
| `unwind-protect` | `(unwind-protect (let ((n 5)) (lambda () n)) 1)` | `5` |

---

## 6. Kontext

Was die Analyse als **sauber** bestätigt hat — nicht anfassen:

- Chokepoints intakt: kein `net/http` außerhalb `lib/sigorest.go`, genau ein
  `IsTruthy`, ein Parser, ein `LoadStdlib`.
- Keine Go↔Lisp-Namenskollisionen. 114 Go-Primitiven gegen 106
  Lisp-Definitionen in `embed/`, Schnittmenge leer. Einzige Ausnahme `let*`
  (4.2), und das ist Spezialform gegen Makro.
- Dateigrößen-Regel gehalten: größte Datei 761 Zeilen (`primitives.go`).
- `go test -race` sauber auf den Concurrency-Tests.
- Env-Locking ist konsistent Kind→Eltern; kein Pfad geht umgekehrt, also
  kein Deadlock-Zyklus. `Env.Get` gibt vor dem Aufstieg frei, `Env.Update`
  nicht — beide bewusst so, siehe 4.4.

CL-Konformität Stand HEAD: **132 / 978 = 13,5 %** (Kern 104/887 = 11,7 %,
Macro 28/91 = 30,8 %). Die Makro-Quote liegt 2,6× über der Kern-Quote — der
billige Hebel via `stdlib.lisp` ist noch nicht ausgeschöpft. Die Zahl gegen
ANSI-Vollumfang ist wenig aussagekräftig, solange CLOS und Packages bewusst
abgelehnt sind; eine Quote gegen den *gewollten* Umfang wäre die ehrlichere
Kennzahl.
