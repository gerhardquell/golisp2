# Retrospektive golisp2

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
