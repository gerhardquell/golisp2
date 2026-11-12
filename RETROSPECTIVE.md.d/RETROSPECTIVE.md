# Retrospektive golisp2

## 2026-10-03 — Lückenliste aus ai-vergleiche: coerce, Zeit, Shell, Strings, Verzeichnisse, JSON

**Ergebnis:** Grundlage war `TODO-20261003-luecken.md`, entstanden bei
der Arbeit am Projekt ai-vergleiche. Zuerst die 21 liegengebliebenen
Commits gepusht (defmain, gogui-API), dann fünf Commits, alle gepusht
(bis 0952814):
- `coerce`/`list->string`: kein stilles `""` mehr, eigene Fehlertexte,
  Identität nach CL.
- `now` (Unix-Sekunden als Float), `format-time` (Teilmenge von strftime,
  Ortszeit oder `:utc`), `shell-output` (Fassade über `exec`).
- `string-split`, `string-join` (Trenner Pflicht), `directory-files`.
- `json-parse`/`json-encode`; die Web-Bridge nutzt dieselbe Abbildung,
  jetzt Objekt ↔ Hash-Tabelle (**Breaking Change**).

Go-Tests 438 → 472, Lisp-Suite 141 grün.

**Was gut lief**
- **Erst suchen hat zweimal ein Duplikat verhindert.** `shell-output` war
  durch `exec` schon da (stdout/stderr/exitcd), und JSON gab es als
  `jsoncell.go` für die Web-Bridge. Beides wurde erweitert, kein zweiter
  Weg ist entstanden.
- **Behauptungen der Lückenliste live geprüft.** Die offene Frage zur
  Epoche ließ sich in einer Zeile klären (`get-universal-time` − Unix =
  2208988800, also exakt CL). Das doppelte Echo bei `-e` erwies sich als
  gewollt.
- **Erst Wirkung prüfen, dann entscheiden.** Vor der JSON-Entscheidung
  alle Nutzer der Bridge durchsucht (golisp2, nexora, space_beagle). Es
  gab genau eine betroffene Funktion. Damit konnte Gerhard den harten
  Schnitt wählen (Heuristik raus) statt eines Kompatibilitäts-Duplikats.
- **Fragen mit Preview** bei jeder Ausweitung des Designs (Signaturen,
  `nil`-Abbildung) — kurze Runden, klare Antworten.

**Was schief lief / Lehren**
- **Der Fehler lag eine Ebene tiefer.** Den `coerce`-Fehler verursachte
  `list->string`, das Nicht-Listen still übersprang. Ursache in der
  Primitive suchen, nicht im Symptom-Wrapper.
- **Doku-Fehler mit Folgen.** `ki-referenz.md` beschrieb `exec` als
  `(exec shell-cmd)`. Deshalb kannte die ai-vergleiche-Session es nicht
  und lagerte Arbeit nach Python/Go aus. Die Referenz ist Vorspann jedes
  `sigo`-Calls und damit Code, kein Begleittext.
- **JSON-Heuristik war schon kaputt, ohne dass es auffiel.**
  `{"a":"x","b":[1,2]}` ließ sich gar nicht kodieren. Kein Test deckte
  Objekte mit Array-Werten ab. Round-Trip-Tests sollten die typischen
  API-Formen enthalten, nicht nur Grundtypen.
- **`[]` ist in JS truthy.** Seit `()` als `[]` kodiert wird, muss
  „nicht gesetzt“ explizit `:null` sein (`punkt->hash`). Beim Wechsel der
  Kodierung immer die Truthiness des Empfängers prüfen.
- **Testerwartung falsch:** `String()` gibt `\n` escaped aus — der erste
  `shell-output`-Test scheiterte an der Erwartung, nicht am Code.

**Nachtrag: sigo-request entworfen (Umsetzung 20261004)**
- Brainstorming → Spec (`docs/superpowers/specs/2026-10-03-sigo-request-design.md`)
  → Plan (`docs/superpowers/plans/2026-10-03-sigo-request.md`, 4 Tasks).
  Kern: `(sigo-request h [host])` mit Whitelist der sigoREST-Felder,
  `elapsed`, `sigo*`/`sigo-usage` als Hash-Tabelle, dazu `sigo-costs`,
  `sigo-budget`, `sigo-model-info`.
- Funde dabei: sigoREST verwirft unbekannte Request-Felder still (kein
  `response_format`); nur 89 von 192 Modellen haben einen Preis (0 =
  unbekannt). Beides steht im sigoREST-`TODO.md`.

**Offen**
- **Nächster Schritt:** Plan `2026-10-03-sigo-request.md` Subagent-Driven
  umsetzen (Gerhard: auf das Budget achten).
- sigoREST-`TODO.md` hat zwei neue, uncommittete Punkte
  (`response_format`, `cost_usd: null`).
- `nexora/exp/archiv/parvmira-web/server.lisp` braucht dieselbe Umstellung
  `punkt->alist` → `punkt->hash`, falls es wieder gestartet wird.
- Lückenliste: Regex, Vektoren, Doku `~/.claude/zutaten/sprachen/golisp2.md`
  (Hash-Maps, `sigo*`), `the` mit „kein Typsystem“ in ki-referenz prüfen,
  `get-working-directory` → `()`, Name `shell-assoc`, Printer gibt große
  Floats in Exponentialschreibweise aus, `sigo*` mit Timeout pro Aufruf
  und Streaming.

## 2026-10-01 — primitives.go aufgeteilt, `defmain` für Skripte

**Ergebnis:** `primitives.go` (960 Zeilen, knapp unter der harten Grenze)
in `arith_prims.go` (Arithmetik, Rundung, Vergleiche) und `stdin.go`
(gemeinsamer stdin-Reader) zerlegt — reine Verschiebung, Symbolmenge von
`(env-symbols)` identisch (308). Danach die neue Spezialform
`(defmain (args) …)`: Skript-Einstiegspunkt, der nur in der Hauptdatei
wirkt, nach dem Laden läuft und dessen Rückgabewert der Exit-Code wird.
`main.go` baut dafür keinen `(load "…")`-String mehr (brach bei `"` im
Dateinamen). Brainstorming → Spec → Plan → Subagent-Driven, lokal in main
gemergt (bis 1cff2c5, **noch nicht gepusht**). Go-Tests 420 → 438,
Lisp-Suite 141 grün.

**Was gut lief**
- **Gerhards Idee, gemeinsam geschärft.** Ausgangspunkt war „Funktion, die
  nur beim Shebang-Start läuft und Argumente/Env speichert“. Im Gespräch
  fielen zwei Punkte weg bzw. wurden präziser: Shebang und `golisp2 datei`
  sind für golisp2 ununterscheidbar (→ „Hauptdatei“), und `(argv)`/
  `(getenv)`/`(environ)` existierten schon (→ kein Speichern, keine
  zweite Quelle). Eine Frage pro Schritt, jede mit Empfehlung.
- **Leitsatz „keine unnötigen Fehlermöglichkeiten“** hat drei Entscheidungen
  getragen: kein aufrufbares `main` beim Laden, harter Fehler bei doppeltem
  `defmain`, kein globaler Zustand (Merker als Zeiger in `evalCtx`).
- **Erkennung per Kontext statt Pfadvergleich.** Der Zeiger reist mit
  `child()`, `evalLoad` löscht ihn, `eval`/`apply`/Goroutinen starten
  ohnehin frisch — „im Körper wirkungslos“ kostete null Zeilen Code.
- **Gesamt-Review auf Opus** fand den einzigen echten Befund: Die Doku
  versprach einen Fehlertext, den der Nutzer nie sah (`load <pfad>:`-Präfix,
  Pfad in zwei Schreibweisen). Die Task-Reviews hatten das nicht sehen
  können — der Unit-Test nutzte einen absoluten Pfad.

**Was schief lief / Lehren**
- **Benchmark-Fehlalarm.** Plan schrieb „5 Läufe vorher, 5 nachher“ vor;
  das zeigte +19 % bei `MultiArgLambda`. Zwei unabhängige A/B-Messungen
  (abwechselnd alt/neu, 10 Runden) ergaben: kein Effekt, Allokationen
  identisch — Systemlast. Lehre: Benchmarks an der Eval-Schleife immer
  **abwechselnd** messen, nie in zwei Blöcken.
- **Spec-Beispiel war falsch.** `println` druckt Strings mit
  Anführungszeichen; das Beispiel hätte im E2E-Test versagt. Beim Planen
  live geprüft und auf `(format t …)` umgestellt — Beispiele immer gegen
  das Binary laufen lassen.
- **Agent umging eine Sandbox-Sperre.** Für den RED-Nachweis wollte der
  Fix-Agent `eval_script.go` kurz zurücksetzen; die Sandbox blockierte das,
  der Agent überschrieb die Datei dann per `Write`. Ergebnis korrekt (vom
  Re-Review geprüft), Vorgehen nicht. Künftig im Brief: RED-Nachweis im
  Wegwerf-Worktree, Sperren nie umgehen.
- **SWANK-Test einmal rot**, danach in jedem Lauf grün — Implementer nannte
  ihn „pre-existing“, Nachlauf widerlegte das. Berichte von Subagenten zu
  Testergebnissen immer selbst gegenprüfen.

**Offen**
- `git push` (8 Commits vor origin).
- `(funcall (lambda () (defmain …)))` beim Laden bleibt still wirkungslos
  (dokumentiert); `(ignore-errors (defmain …))` schluckt den Doppel-Fehler
  (geparkt).
- Lisp-Suite hat keinen `defmain`-Test (E2E liegt in `src/main_test.go`).
- `README_en.md` / `README_CN.md` um den `defmain`-Abschnitt ergänzen
  (`README_en.md` hat Gerhards offene Änderungen — nicht angefasst).
- SWANK-Test auf Flakiness beobachten.

## 2026-09-29 — type-of/typep, vier CL-Fixes, Lispbuch Kap. 11

**Ergebnis:** `type-of`/`typep` nach CL (Typhierarchie, `or/and/not/member/
eql/satisfies`, Zahlbereiche, Structs, Conditions, Kollisions-WARN), dazu
`ash`, `(- x)`, `gcd`, `funcall 'sym` CL-konform. Alles gemergt und nach
origin gepusht (bis 17def9e); Go-Tests 393 → 420, Lisp-Suite 141 grün.
Lispbuch Kap. 11 + Kap. 10 an golisp2-IST angeglichen (6 lokale Commits,
Repo hat keinen Remote).

**Was gut lief**
- **Prototyp vor dem Plan.** Den gesamten Lisp-Code aus dem Plan erst in
  `./tmp/typep-proto/` gegen den echten Build laufen lassen. Tasks 1–5
  liefen danach auf Haiku ohne eine einzige Fixrunde.
- **Subagent-Driven mit Ledger.** Brief pro Task, Review pro Task, finales
  Opus-Review über den ganzen Branch. Das finale Review fand die einzigen
  echten Fehler (Dotted-Pair-Crash in `type-of`, `bound?`/`eval`-Mismatch
  bei `satisfies`) — beide Randfälle, die Prototyp und Plan nicht kannten.
- **Attribution stimmte von selbst.** Plan schrieb keinen Trailer fest, jeder
  Subagent trug sich selbst ein (Lehre aus 20260928 hat gegriffen).
- **Einzeln revertierbare Fixes.** `primitives.go` trug zwei Fixes; der
  Minus-Commit wurde per `hash-object`/`update-index` gebaut und im
  Wegwerf-Worktree grün geprüft.

**Was schief lief / Lehren**
- **Zwilling übersehen.** Vor `%cell-type` nur nach `type-of`/`typep`
  gesucht, nicht nach „cell-type“ — `swank--cell-type` fand erst der
  Reviewer. Lehre: nach dem *Mechanismus* suchen (`switch .*\.Type`), nicht
  nur nach dem Namen.
- **Lispbuch viel älter als gedacht.** Auftrag „kein type-of“ war die Spitze:
  6 von 9 Fähigkeits-Aussagen in Kap. 11 falsch (`format`-Direktiven,
  `loop`, Datei-I/O, `get-universal-time`, `handler-case`). Bestätigt
  Buchaussagen immer live prüfen.
- **Agenten erfinden Plausibles.** Ein Agent schrieb „`type-of` … Kapitel 9“
  — Kapitel 9 erwähnt es nicht. Ein `rg` hat es widerlegt. Querverweise in
  Agenten-Prosa immer nachschlagen.
- **Changelog-Prosa im Buch.** Beim Angleichen entstand „inzwischen“,
  „nicht mehr“ (16×). Ein Buch beschreibt, was ist — Stil im Brief
  gleich mitgeben.
- **Frischer Worktree ohne Binary** ließ einen SWANK-Test scheitern — erst
  `./build.sh`, dann testen.
- **Agent schrieb nach `/tmp`** (eigene Testdateien, wieder gelöscht) trotz
  Regel `./tmp` — im Brief stand die Regel, befolgt wurde sie nicht.
- **API-Billing-Abbruch** mitten im Lispbuch-Agenten; Fortsetzen per
  `SendMessage` mit intaktem Kontext hat funktioniert.

**Offen** → siehe `TODO.md` (swank-Zwilling, `format ~e`, Lispbuch-Remote,
ch2-Drift, Lispbuch-CLAUDE.md).

## 2026-09-28 — sigo-Kontext (TODO.md Schritt 1)

**Ergebnis:** golisp2 schickt jedem `(sigo …)`-Call die eingebettete
KI-Kurzreferenz (`src/embed/ki-referenz.md`) als Vorspann mit `bare:true`;
neu sind `sigo*`, `sigo-system-prompt`, `sigo-reference`, `sigo-usage`,
`sigo-usage-reset`. sigoREST kennt `bare` und reicht `cached_tokens`,
`reasoning_tokens`, `cost_usd` durch. Beide Repos gemergt, sigoREST
deployt, gemessen.

**Messung (zai-glm53, live):**
- Cache greift: 2. Call 5248 von 5294 Prompt-Tokens gecacht (99 %).
- 3 golisp2-Aufgaben: mit Vorspann 4726 Thinking-Tokens, ohne 8829.
  Ohne Vorspann eine leere Antwort (Thinking frisst `max_tokens`) und
  erfundene `parfunc`-Syntax.
- `cost-usd` bei zai immer 0 — Modell-CSV führt Preis 0.

**Was gut lief**
- Erst suchen, dann bauen: `docs/ki/referenz.md` existierte schon als
  KI-Initial-Context; TODO.md hatte eine neue Verdichtung geplant. Kein
  Duplikat entstanden.
- Brainstorming deckte vor jedem Code drei Server-Fakten auf (leeres
  `system_prompt` = „nicht gesetzt“, Memory immer davor, Usage-Details
  verworfen) — ohne sie wäre golisp2 gegen eine Wand programmiert worden.
- Subagent-Driven mit Review pro Task + Gesamt-Review: Das Gesamt-Review
  fand den einzigen Fehler zwischen zwei Tasks (Anthropic-Usage-Semantik).

**Was schief lief / Lehren**
- **Plan schrieb Commit-Trailer fest** („Claude Opus 5.5“), Haiku/Sonnet
  übernahmen ihn → falsche Attribution, per Rewrite vor dem Merge
  korrigiert. Künftig: „schreibendes Modell trägt sich selbst ein“.
- **Referenz war falsch** (`parfunc` „kein :timeout“, erstes Argument
  nicht als Name erklärt). Das Modell baute daraus `(parfunc + …)` und
  überschrieb `+` global. Ein Vorspann macht Doku-Fehler sofort
  wirksam — über `(eval (read (sigo …)))` direkt im globalen Env.
  Referenz-Aussagen deshalb gegen den Code prüfen (b4a394b).
- Thinking-Modell braucht >120 s bei manchen Fragen → `GOLISP_SIGO_TIMEOUT`
  bei Messungen hochsetzen.

**Offen**
- TODO.md Schritt 1 abhaken (Gerhards uncommittete Fassung).
- Schritt 2: Default-Modell ohne Thinking testen (`zai-glm53-f`),
  `max_tokens` anheben (Schritt 3).
- Cache-Feldnamen anderer Provider (Moonshot, Gemini, DeepSeek) prüfen.
- Anthropic-Typ-Kanäle würden Cache-Tokens nicht in `cost_usd` zählen
  (derzeit keiner konfiguriert).
