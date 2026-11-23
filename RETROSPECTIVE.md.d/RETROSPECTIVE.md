# Retrospektive golisp2

## 2026-11-23 (Nachmittag) — notany/notevery, Vektor-Zyklen, Conformance in CI, md2xhtml2 1.0.0, Lispbuch-Nacharbeit

**Ergebnis:**
- **golisp2:** zwei Releases, beide gemergt, gepusht und installiert.
  - **0.38.1:** `notany` und `notevery`.
  - **0.38.2:** Ein Vektor auf einem Zyklus druckt `#<…zirkulär>` statt `fatal error: stack overflow`. Bisher riss das den ganzen Prozess samt SWANK mit.
- **Conformance:** Der Runner filtert jetzt die Fehlerort-Zeilen, und ein veralteter XPASS ist bereinigt. Ergebnis: 200 PASS. Die Suite läuft jetzt in CI, der GitHub-Lauf ist grün.
- **md2xhtml2 1.0.0:** `--version`, `--help` und ein generator-meta-Tag, mit Tag `v1.0.0`. Installiert ist es nach `/usr/local/bin`.
- **Lispbuch, lokal:**
  - Kap. 3 korrigiert: Die Scheme-Kurzform geht, `compose` und `square` sind eingebaut.
  - `tools/md2xhtml.py` erzeugt jetzt `<ol>`. Damit haben 38 Dateien echte nummerierte Listen.

**Was gut lief**
- **Den Auftrag am Code prüfen.** Der Agent für die Container-Zyklen fand heraus, dass nur der Drucker betroffen war. `equal?` und Hash-Schlüssel vergleichen Container nach Identität wie CL `equal`. Meine Vorgabe, dort laut zu werden, wäre falsch gewesen. Er hat das begründet abgelehnt statt blind gebaut. Meine eigene frühere Aussage „equal? hängt“ stammte ungeprüft aus einem Agentenbericht.
- **Das Symptom eine Ebene tiefer gesucht.** Bei `02-let-lambda` lag kein Interpreter-Fehler vor. Der Runner erwartete eine Zeile pro Form, aber Fehler haben seit dem Fehlerort-Feature mehrere Zeilen. Eine Probe mit jeder Form einzeln zeigte das sofort.
- **Stichproben gegen das Buch.** Beim Prüfen von Kap. 3 fiel nebenbei die falsche Aussage „golisp2 kennt keine `(define (f x) …)`-Kurzform“ auf.

**Was schief lief / Lehren**
- **Neuerzeugen kann Handarbeit vernichten.** Wir haben die lispbuch1-XHTML heute mehrfach neu erzeugt. Dabei gingen in 7 Dateien `<ol>`-Listen verloren, die nicht aus dem Generator stammten, sondern aus Sigil oder von Hand. Aufgefallen ist das erst beim dritten Lauf, durch einen aufmerksamen Agenten. Lehre: Vor dem ersten Neuerzeugen einen **Basis-Vergleich** machen, also Generator auf den alten Quellen gegen die committete Ausgabe. Jede Abweichung ist entweder veraltet oder Handarbeit, und das muss man vorher wissen.
- **Fehlerort-Ausgabe hatte einen ungeprüften Verbraucher.** Das Feature, das Fehler mehrzeilig macht, hat die Conformance-Suite leise kaputtgemacht. Sie lief nicht in CI, deshalb fiel es niemandem auf. Jetzt läuft sie dort mit.
- **Subagenten halten `./tmp` nicht zuverlässig ein.** Wieder einmal landete etwas unter `/tmp`, und einmal lagen 54 Sections im Repo-Root. Beides haben die Agenten selbst gemeldet und weggeräumt.

**Offen**
- Das md2html-Repo hat keinen Remote. Ob es lokal bleibt, ist noch nicht entschieden.
- Die Sandbox meldet bei Vektor-Zyklen „zirkuläre Liste“ statt „zirkuläre Struktur“, das ist nur der Wortlaut.

## 2026-11-22/23 — Handles, Kleinkram, echte Cons-Mutation, CL-Angleichungen, Lispbuch

**Ergebnis:** Fünf Releases, alle gemergt und gepusht, alle Subagent-Driven. Claude war Teamleiter, Opus und Haiku haben implementiert.
- **0.36.4:** Handles als FOREIGN.
- **0.36.5:** `length`-Meldung, `(/ x)`, `(read "")` meldet end-of-file, `http-static :headers`.
- **0.37.0:** echte Cons-Mutation:
  - `rplaca`/`rplacd`/`nconc`/`nthcdr`; `setf` auf Listen-Places ändert die Zelle.
  - defstruct-Setter mutieren.
  - Ringlisten-Schutz.
  - Sandbox-Grenze kopiert.
- **0.37.1:** `copy-list` iterativ und Ring-sicher.
- **0.38.0:** `butlast n`, `assert` mit Meldung, `every` mehrstellig, `macroexpand` voll, `block nil` in `loop`/`dotimes`/`dolist`, `member`/`remove` mit `:key`/`:test`.

Daneben:
- nexora an 0.37 angepasst.
- Lispbuch (beide Bücher):
  - auf 0.37 gebracht;
  - rund 40 Dateien mit Altfehlern korrigiert;
  - md2html liefert wohlgeformtes XHTML.

Go-Tests 795 → 806, Lisp-Suite 180 grün.

**Was gut lief**
- **Phasen bei Breaking Changes.** Zuerst Charakterisierungstests, die das IST festhalten (Phase 1). Dann die Spec mit Gerhards Entscheidungen per Fragen mit Optionen. Erst danach der Umbau. Die `TestHeute…`-Tests zeigten beim Umdrehen genau, was sich ändert.
- **Externe Suiten vor dem Install.**
  - nexora brach unter 0.37.0, weil es defstruct-Setter funktional nutzte. Das fiel auf, bevor das neue Binary installiert war.
  - Die Anpassung lief als eigener Agent parallel zum Sandbox-Fix.
- **Messen statt vermuten.**
  - Die Messung vorher/nachher bei `length` zeigte 40× schneller, beim Drucker 1200× schneller. Der Drucker hatte quadratisches `s +=`.
  - Bei `dotimes` mit `block nil` zeigte sie keinen messbaren Unterschied.
- **Agenten mit Urteilsvermögen.**
  - Der Haiku-Agent committete nicht, als `xmllint` trotz Entity-Fix weiter scheiterte (`<hr>`). Er fragte nach, statt „wohlgeformt“ zu behaupten.
  - Der Opus-Agent baute die Ablehnung von Funktionen an der Sandbox-Grenze nicht ein, weil dokumentiertes Verhalten dagegen stand, und legte die Entscheidung vor.
- **Stichproben des Teamleiters fanden Zusatzfehler.** `copy-list` scheiterte nicht nur an Ringen, sondern schon an 200 000 Elementen; die Ursache war nicht-endrekursives Lisp. Der Fix behob beides.

**Was schief lief / Lehren**
- **Mutation macht aus geteiltem Code eine Angriffsfläche.** Ein automatischer Security-Review fand zwei Probleme an der Sandbox-Grenze:
  - Eine in die Sandbox gereichte Host-Lambda ließ sich per `car` + `rplaca` umschreiben. Danach lief fremder Code mit vollen Rechten.
  - Die Baumkopie ohne Memo blähte DAGs exponentiell auf, und das außerhalb des Zeitlimits.

  Lehre: Wird etwas veränderlich, jede Vertrauensgrenze neu prüfen, vor allem auf Code, der als Daten erreichbar ist.
- **Das Buch war weit schlechter als gemeldet.** Statt der gemeldeten rund 15 Punkte waren es etwa 40 Dateien mit Falschaussagen, z. B. „kein push“, „let sequenziell“, „eq für Zahlen“. lispbuch2 enthielt dazu ältere Fassungen mit denselben Fehlern. Das bestätigt die alte Regel: Lispbuch-Aussagen immer live prüfen.
- **Den Generator erst finden, dann neu bauen.** Die Section-XHTML von lispbuch2 hielt der Agent für Handarbeit aus Sigil. Ein Neuerzeugen aus dem alten Commit mit `md2xhtml2` war aber für 51 von 54 Dateien byte-gleich, die drei übrigen waren veraltet. Fertig ist das Werkzeug damit erst, wenn es auffindbar ist; heute liegt es nur unter `/usr/local/bin` ohne Verweis im Repo.
- **Nicht testen, während der Agent noch baut.** Eine Zwischenprüfung von md2html lief, während der Agent baute, und gab Fehlalarm (`<hr>` ungeschlossen). Auf den Bericht warten.
- **Nutzungslimit mitten im Agenten.** Der Lispbuch-Agent brach in Kapitel 24 ab. Ein Fortsetzen per Nachricht nach dem Reset funktionierte, und die schon gemachten Commits blieben erhalten. Kleinere Commit-Schritte machen solche Abbrüche billig.
- **Regeln für Subagenten bleiben Disziplin.** Zweimal legte ein Agent etwas unter `/tmp` an, obwohl die Vorgabe `./tmp/` war, und einmal versuchte einer `rm -rf`; der Hook blockierte es.

**Offen**
- `notany`/`notevery` fehlen.
- Conformance `02-let-lambda` scheitert (alt).
- Ringe über Vektoren/Hash-Tabellen hängen weiter.
- Das neue md2xhtml2 ist noch nicht nach `/usr/local/bin` installiert.
- Kap. 3 des Lispbuchs definiert `compose`/`square` neu (REDEF).

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
