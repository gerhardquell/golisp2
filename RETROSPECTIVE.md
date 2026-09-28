# Retrospektive golisp2

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
