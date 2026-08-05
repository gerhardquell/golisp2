# Retrospektive: Env-Lebensdauer, eq-Semantik, Allokationen

**Datum:** 5. August 2026
**Autor:** Gerhard Quell & claude-opus-5
**Feature:** Aus einer offenen Code-Analyse: zwei Korrektheitsfehler beseitigt, `eq` CL-konform gemacht, Allokationen um 82 % gesenkt

---

## Was wurde gebaut?

Der Tag begann mit einer TODO.md ohne Ziel: *„Wir wollen heute den Code von
golisp2 analysieren."* Analyse → grill-me → Spec → TODO. Daraus wurden zehn
Commits.

**Korrektheit (2 Fehlerklassen, 5 `eq`-Divergenzen):**

- **Env-Pooling entfernt** (`0bf8caf`). `Env.shared` war ein Bit *pro Frame*,
  die Erreichbarkeit einer Closure läuft aber transitiv über `parent`.
  **10 von 11** frame-besitzenden Formen verloren den lexikalischen Scope:
  `let`, `let*`, Lambda-TCO, `applyLambda`, `do`, `do*`, `flet`, `macrolet`,
  `symbol-macrolet`, `progv`, `multiple-value-bind`. Zwei Fehlerbilder —
  verlorene Bindung, oder ein `parent`-Zyklus, der `Env.Get` in einen
  **unrecoverbaren** `fatal error: stack overflow` schickt.
- **Symbol-Interning** (`56e935d`). `MakeAtom` gab pro Aufruf eine frische
  Cell zurück, also war `eq` auf *jedem* Symbol kaputt: `(eq 'foo 'foo)` →
  `()`, CL sagt `T`. `cellT`/`cellNil` waren ein Ad-hoc-Interning für zwei
  Werte und machten das Verhalten inkonsistent, nicht korrekt.
- **`(and)` nachgezogen** (`e396f4e`). Letzte nicht-internierte ATOM-Cell,
  bei `56e935d` übersehen, nur über `(and)` ohne Argumente erreichbar.
- **`let*`-Duplikat entfernt** (`6f2d920`). Go-Spezialform *und* Lisp-Makro;
  `eval` erreichte das Makro nie, `macroexpand` schon.

**Performance (fib 25):**

| | allocs/op | B/op | ns/op |
|---|---|---|---|
| Nachmessung morgens | 1 335 430 | 32 MB | 139 ms |
| nach Schnitt 7+8 (Korrektheit) | 1 578 205 | 59 MB | 148 ms |
| Schnitt 9 `evalCtx` per Value | 242 833 | 27 MB | 115 ms |
| Schnitt 10 `Env` 112→88 B | 242 828 | 23 MB | 113 ms |
| Schnitt 11 `Env` 80 B + Symbol-Lookup | **242 819** | **19,4 MB** | **107 ms** |
| | **−81,8 %** | **−39,4 %** | **−23,0 %** |

**Fünf neue Test-Netze**, 265 → 284 Tests: `env_closure_test.go` (Closure ×
Frame-Form), `intern_test.go` (`eq`-Semantik + Nebenläufigkeit),
`specialform_shadow_test.go` (liest die Namen aus `eval_core.go`),
`eval_depth_test.go` und `env_frame_test.go` (jeweils vor einem Refactoring).
Dazu `evalBench_test.go` — sechs Benchmarks für die Pfade, für die `fib` blind
ist.

**Repo-Hygiene:** kaputter Gitlink entfernt, Default-Branch auf `main`
korrigiert, vier Altbranches aufgeräumt.

---

## Was lief schief? (⚫ Schwarz)

**Drei Analysebefunde waren falsch — alle drei aus derselben Ursache.**
`Env.Update`-Locking, `argSlicePool`-Nullung, Frame-Mutex: jedes Mal wurde
eine Beobachtung aus dem Code (*„hält Lock über Rekursion"*, *„nullt nicht"*,
*„Frame gehört einer Goroutine"*) zur Behauptung über das System, ohne den
einen Aufrufer zu prüfen, der sie widerlegt. Alle drei standen in der
Analyse als „niedrig, code-reading, kein Repro" — diese Markierung hat den
Schaden begrenzt, aber die Behauptungen hätten so nicht dastehen sollen.

**Ein Test war grün und bewies nichts.** Die `let`/`let*`-Fälle im
Closure-Netz reproduzierten zunächst nicht, weil dem Ausdruck der Tail-Call
fehlte, der `takeEnv` zur Freigabe bringt. Zwei von zehn Formen wären als
„bereits korrekt" durchgegangen — an den beiden heißesten Stellen des
Interpreters. Nur die RED-Verifikation hat es gefangen.

**Ein Kalibrier-Probe hat sich selbst neutralisiert.** Beim Festnageln der
Tiefen-Semantik baute ich die Verschachtelung mit `funcall` — und `funcall`
setzt genau das Depth-Budget zurück, das gemessen werden sollte. Alle Fälle
grün, Aussagekraft null.

**Ein Performance-Modell lag um 2 MB daneben.** Bei Schnitt 10 gegen die
Struct-Größe gerechnet statt gegen die Size-Class des Allocators: 88 B
landen in der 96-B-Klasse, die Ersparnis war 16 statt 24 Byte pro
Allokation. Dass die Rechnung bei 112 B vorher aufging, war Zufall — 112 ist
selbst eine Klasse.

**Eine Optimierung gemessen, bestätigt, und trotzdem falsch.** Die feste
`entries`-Startkapazität gewinnt bei vierstelligen Funktionen deutlich
(−33 % Allokationen). Der Benchmark, der den Posten aufgedeckt hatte, hätte
den Fix auch bestätigt. Erst fünf weitere Benchmarks zeigten den Verlust im
Alltag — zweistellige Funktionen, das `(n acc)`-Idiom. Revertiert.

**Die Antwort auf einen der Fehlbefunde stand schon im Repo.**
`RETROSPECTIVE.md:2467` erklärt seit Juli, warum `Env.Update` sein Lock über
die Parent-Kette hält: *„sonst könnte ein paralleles `Set` denselben Namen
dazwischen im Child erzeugen."* Genau diese Begründung habe ich heute
unabhängig neu hergeleitet — nachdem ich es zuvor als Mangel in die Analyse
geschrieben hatte. Die Analyse las `lib/`, `CLAUDE.md`, `doc/` und
`PerfTODO.md`, aber nicht die Retrospektiven. Bei einem Befund der Form
*„warum ist das so gebaut?"* ist das Protokoll die erste Quelle, nicht die
letzte.

**Ein `+4 %` bleibt unerklärt.** Symbol-Interning kostet messbar Zeit auf
`fib`, obwohl `allocs/op` bit-identisch bleibt und `MakeAtom` im CPU-Profil
nicht auftaucht. Vermutlich Code-Layout. Steht als offene Frage in PerfTODO,
nicht als Erklärung.

---

## Was haben wir gelernt? (🔵 Blau)

**Ein Pointer ist eine Aussage über den Zustand.** `evalCtx` war
write-once und pro Goroutine exklusiv — beides Gegenteil von „geteilt,
veränderlich". Als Pointer kostete es 84,9 % aller Allokationen des
Interpreters. Wer einen Pointer nimmt, wo ein Wert genügt, bezahlt eine
Heap-Allokation für eine unwahre Aussage.

**Ein Bit pro Objekt kann keine transitive Eigenschaft ausdrücken.** Das war
der Env-Pool-Bug in einem Satz. Erreichbarkeit ist transitiv über `parent`,
`shared` war lokal. Deshalb waren es nicht zehn Bugs, sondern ein
Designfehler — und deshalb hat ein einziger Fix alle zehn behoben.

**Ein manueller Pool über einer GC-Sprache übernimmt die Beweislast für die
Lebensdauer.** Der GC kennt Erreichbarkeit exakt. Ihn für einen Objekttyp
abzuschalten heißt, diese Kenntnis selbst nachzubauen — und die Kosten
dafür lagen zuletzt bei 2,3 %, während der Fehler ein unrecoverbarer
Prozessabbruch war.

**Der Benchmark, der eine Optimierung motiviert, ist nicht ihr Beleg.**
Zweimal an einem Tag: `fib` hat Schnitt 5 motiviert und dessen
Korrektheitsloch nicht gesehen (es enthält keine Closures);
`MultiArgLambda` hat die `entries`-Kapazität motiviert und hätte sie
bestätigt (es enthält keine zweistelligen Funktionen).

**`unsafe.Sizeof` ist nicht die allozierte Größe.** Go rundet auf
Size-Classes (… 64, 80, 96, 112 …). Ein Struct um 8 Byte zu verkleinern
bringt nur etwas, wenn es dabei eine Klassengrenze unterschreitet. An drei
Stellen dieses Tages entscheidend geworden, inklusive der Erklärung, warum
`cap=3` besser war als `cap=4`.

**Gegenläufige Änderungen getrennt messen.** Schnitt 11 bestand aus zwei
Teilen: Feld verkleinern (allein **+4,2 %**, weil `Set` dann internieren
muss) und Hot-Paths auf Pointer-Lookup (dreht es auf −3,3 %). Ohne die
Zwischenmessung wäre der Schnitt nach der falschen Hälfte verworfen worden.

**Ein Kommentar über eine *nicht* gemachte Änderung ist oft wertvoller als
Code.** Drei Sackgassen sind jetzt an ihrer Fundstelle begründet
(`Env.Update`, `putArgSlice`, Frame-Mutex). Der nächste Bearbeiter kommt mit
derselben Beobachtung und derselben plausiblen Ableitung an.

**Halb-korrekte Identität ist schlechter als konsistent falsche.**
`cellT`/`cellNil` machten `eq` an 14 Stellen richtiger und überall sonst
nicht. Konsistent falsch kann man dokumentieren; inkonsistent nicht, und die
Tests beruhigt es trotzdem.

**Ein Test, der die Wahrheit aus dem Produktivcode liest, veraltet nicht.**
`TestNoLispDefineShadowsSpecialForm` extrahiert die 58 Spezialform-Namen aus
`eval_core.go`. Eine gepflegte Liste wäre nach dem nächsten `case` falsch —
und hätte dann falsche Sicherheit gegeben. Ein zweiter Test prüft die
Extraktion, damit sie nicht still leer läuft.

**Retrospektiven sind Protokoll, keine Doku.** Die Lektion von §4.5
(*„Frame-Pooling braucht ein Ownership-Modell: `ownEnv` + `shared`-Flag"*)
war korrekt formuliert und trotzdem falsch angewandt. Sie wurde **nicht**
korrigiert — sie zeigt, was man damals dachte. Die Korrektur gehört dorthin,
wo der aktuelle Stand steht.

---

## Action Items

- [ ] **`Cell` verkleinern (104 B).** Das Struct trägt Felder für jeden Typ
  gleichzeitig (`Fn`, `Env`, `Ht`, `SrcFile`, `SrcLine`, `Car`, `Cdr`, `Val`,
  `Num`), obwohl jede Cell nur einen Typ ist. `SrcFile`/`SrcLine` (24 B)
  werden ausschließlich auf LIST-Cells gestempelt. Benchmarks dafür
  existieren jetzt (`ListBuild`, `MacroExpand`). Size-Class vorher rechnen.
- [ ] **Tiefenbegrenzung ist kein hartes Limit.** `funcall`/`apply` und
  `eval` setzen das Budget zurück, obwohl beide echten Go-Stack verbrauchen.
  Lisp-Code kann die Grenze durch Bouncen beliebig weit umgehen. Als
  Charakterisierungstest festgehalten, nicht gefixt — Design-Entscheidung
  offen.
- [ ] **Die +4 % beim Interning klären** oder als nicht klärbar abschließen.
  `fib` erzeugt keine Symbole, ist also das falsche Instrument. Ein
  lese-/makrolastiger Benchmark wäre nötig.
- [ ] **`churnCount` im Closure-Netz ist probabilistisch.** `sync.Pool`
  konnte seine Einträge bei GC verwerfen; nach dem Entfernen des Pools ist
  der Churn nur noch Stressor. Bewusst so belassen, dokumentiert.
- [ ] **Retrospektiven in die Analyse-Quellen aufnehmen.** Ein Befund der
  Form *„warum ist das so gebaut?"* gehört zuerst gegen
  `docs/retrospectives/` geprüft. Heute hätte das einen von drei
  Fehlbefunden verhindert. Kandidat für einen Satz in CLAUDE.md unter
  „Bevor du neuen Code schreibst".
- [ ] **CL-Konformität:** 132/978 Symbole. Die Quote gegen ANSI-Vollumfang
  ist wenig aussagekräftig, solange CLOS und Packages abgelehnt sind — eine
  Quote gegen den *gewollten* Umfang wäre die ehrlichere Kennzahl.
