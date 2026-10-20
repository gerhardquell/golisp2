# TODO golisp2 – Aufgaben und Fragen

Zusammengeführt am 19.10.2026 aus `TODO.md` (Stand 18.10.) und den drei Dateien in `TODO.md.d/`
(`TODO-20260928.md`, `TODO-20261003-luecken.md`, `TODO-20261011.md`). Die alten Dateien bleiben
als Verlauf dort. Ihre offenen Punkte sind mit `- [>]` als „übertragen“ markiert.
Geprüft mit golisp2 0.6.1 (3dc0f75); „geprüft 19.10.“ heißt: im Interpreter ausprobiert.

## 1. Entscheidungen (Gerhard)

- [x] `swank--cell-type` (`src/lib/swank/env.go:116`) ist Beinahe-Zwilling
      von `cellTypeName` (`src/lib/celltype.go`). Erledigt 20.10.: Go-Zwilling entfernt,
      `describe-symbol` nutzt `type-of` (Labels jetzt CL: `symbol`, `cons`, `compiled-function` …).
- [ ] Lispbuch: Remote anlegen? (Repo hat keinen — 6 Commits nur lokal)
- [ ] Lispbuch: `src/ch2/` und `src/chapters/` driften — Kap. 8/12/16/24/27
      in ch2 älter (ohne sort/sqrt-Fixes), Kap. 1 in ch2 neuer. Richtung?
- [ ] Lispbuch-`CLAUDE.md`: Sprachabschnitt veraltet („keine Hash-Maps",
      „KEIN CL-Condition-System").
- [x] Printer: Große Floats erscheinen in Exponentialschreibweise
      (geprüft 19.10.: `(now)` → `1.7924418639679012e+09`). Erledigt 20.10.: Regel wie
      Python-`repr` (Festkomma 1e-4 ≤ |x| < 1e16), eine Quelle `FormatNumber`.
- [x] `golisp2 -e` druckt den Wert zusätzlich zur Ausgabe (geprüft 19.10.:
      `(println 1)` → zwei Zeilen `1`). Erledigt 20.10.: Wert nur, wenn die Form
      nichts selbst nach stdout geschrieben hat (`docs/cli.md`).

## 2. Fehler und Kleinkram

- [x] `format ~e` / `~g` runden falsch: `(format nil "~e" 1.5)` → `"2e+00"`
      (CL: `"1.5e+0"`). Erledigt 20.10.: ohne `d`/`e` so viele Stellen wie nötig;
      `printf "%e"` behält C-Form `e+00`, `%0We` füllt jetzt mit `0`.
      `~g` seit 20.10. nach CLHS 22.3.3.3 (`~f` + 4 Leerzeichen oder `~e`);
      `printf "%g"` ist jetzt ein Fehler (kein FORMAT-Gegenstück).
- [x] `(get-working-directory)` liefert `()`, wenn es nie gesetzt wurde
      (geprüft 19.10.). Erledigt 20.10.: liefert dann das Prozess-Verzeichnis.
- [x] Kommentar an `fnAppend` (`src/lib/primitives.go:261`) sagt „single-item",
      Tests sagen CL-`append` — erledigt 20.10.: Kommentar beschreibt CL-`append`.
- [x] webserv deferred-minor (20260809): `fnBrowserOpen` verschluckt Fehler,
      `/`-Kollision → Go-Panic, `webserv-smoke` nicht in `-t`. Erledigt 20.10.:
      Browser-Startfehler → Fehler (Server gestoppt), kein Browser → Warnung;
      Routen-Kollision → `http-static`/`http-upload`-Fehler; `webbridge.lisp` läuft in `-t`.

- [x] Gequotetes Symbol als Funktion: `(mapcar 'f …)`, `(apply '+ …)`, `(reduce '+ …)`
      → „ist keine Funktion"; nur `funcall` kann es (CL: überall erlaubt). Fiel in der
      sigo-Messung 20261020 auf — KIs schreiben das. Erledigt 20.10.: `resolveFn`
      (Eval-Pfad, `funcall`, `apply`, `mapcar`, `sort`).

## 3. Fehlende Funktionen

- [x] Reguläre Ausdrücke für Lisp-Code — erledigt 20.10.: `regex-match`, `regex-find`,
      `regex-find-all`, `regex-replace`, `regex-split` (`src/lib/regex.go`, Muster zuerst).
- [x] Vektoren/Arrays mit O(1)-Zugriff — erledigt 20.10.: `make-array`, `vector`, `aref`,
      `aset`, `vector-push-extend`; `length`/`coerce`/`setf`/`typep` kennen Vektoren (`src/lib/vector.go`).
- [x] Arbeiterpool — erledigt 20.10.: `(pool-map n fn liste)` in `stdlib.lisp`, Ergebnisse in
      Eingabe-Reihenfolge, Fehler als `(:error text)` an der Stelle. Vorlage war der Eigenbau in
      `/u/ki-projekte/ai-vergleiche/src/steuerung.lisp` (dort könnte er jetzt ersetzt werden).
- [x] typep-Folgen (Spec-Nicht-Ziele): `typecase`/`etypecase`/`check-type`/`deftype`/`subtypep`
      erledigt 20.10. (`src/embed/types.lisp`).

## 4. sigo / sigoREST


- [x] Schritt 2: Default-Modell von `(sigo …)` auf ein Modell ohne Thinking
      (Kandidat `zai-glm53-f`); Thinking-Modelle nur gezielt, dann `max_tokens` großzügig.
      Erledigt 20.10.: Default `zai-glm53-f` (~14× billiger, denkt wenig — 11 Reasoning-
      Tokens gemessen, also nicht ganz „ohne“); Regel in `docs/sigo.md`.
- [x] Schritt 3: Für Code niedrige `temperature` und bewusstes `max_tokens` als Standard
      (pro Aufruf geht es schon über `sigo-request`). Erledigt 20.10.: Default-temperature
      0.2 (`GOLISP_SIGO_TEMPERATURE`), `0` ist Fehler; `max_tokens` bleibt beim
      Modell-Maximum von sigoREST (bewusst, `docs/sigo.md`).
- [x] Schritt 4: Nach Schritt 2–3 neu messen (Vorgehen: `TODO.md.d/TODO-20260928.md` Abschnitt 4).
      Greift `cached_tokens` mit dem Vorspann? Thinking-Anteil? Antworten golisp2-korrekt?
      Erledigt 20.10. (`docs/sigo.md`, „Messung 20261020"): Cache 99,5 % ab 2. Call, beide
      Modelle denken (~90 % der Completion), `zai-glm53-f` 11/12 korrekt bei 1/14 der Kosten.
- [ ] Cache-Feldnamen Moonshot/Gemini/DeepSeek.
- [ ] Streaming für `sigo*`. Achtung: sigoREST bucht bei Streaming die Kosten falsch
      (`/u/go-projekte/sigoREST/TODO-20261003-kosten.md`).
- [x] `cost-usd` von `sigo*` gegen `/api/costs` abgleichen (Werkzeug: `(sigo-costs t0)`).
      Erledigt 20.10.: exakt gleich (4 Calls, 0,009657 USD). Aber beide ohne Cache-Rabatt —
      obere Schranke (`docs/sigo.md`).
- [ ] sigoREST (anderes Repo): `cached_input_cost` für die `zai`-Modelle eintragen
      (heute 0 → gecachte Tokens zum vollen Preis). Echten Cache-Preis bei Z.ai nachsehen.
- [ ] Später, bei Bedarf: Zusammenfassung langer Sessions im Client statt des
      20-Nachrichten-Fensters; Retrieval über `/v1/embeddings` statt Komplett-Referenz;
      sigoREST (anderes Repo): Thinking pro Request abschalten.

## 5. Beim Zusammenführen als erledigt erkannt (19.10.2026)

| Punkt | Beleg |
|---|---|
| 20260928 Schritt 1: Referenz als Vorspann, stabiler Anfang, als `system_prompt` | `(sigo-reference)` ist per `go:embed` der Default-Vorspann; Cache-Regel steht im Kopf der Referenz |
| 20260928 Gerhards Gedanken 1–3: Prompt leeren, lesen/schreiben, Kosten und Cache | `(sigo-system-prompt "")`, `(sigo-system-prompt "text")`, `sigo*` mit `cached-tokens`/`cost-usd`, `sigo-costs`, `sigo-usage` |
| 20261003 §0: Doku „keine Hash-Maps“, „sigo nur Text“ | in golisp2-`CLAUDE.md` und der Zutat nicht mehr enthalten; KI-Referenz §3 beschreibt Hash-Tabellen und `sigo*` |
| 20261003 §1: `parse-time` | entschieden 20261003: nicht gebaut (YAGNI) |
| 20261003 §2: `shell-assoc` irreführend | dokumentiert: „nur `assoc`, kein Shell-Aufruf“ (KI-Referenz §3) |
| 20261003 §4: `string-find` Nadel zuerst | dokumentiert (KI-Referenz §3, Strings) |
| 20261003 §5: `slurp` | dokumentiert: liest stdin (KI-Referenz §9) |
| 20261003 §5: `coerce 1 'string` | war schon erledigt; heute klarer Fehler statt `""` |
| 20261011: Tabs in `examples/golisp2web_demo.lisp` | erledigt (TODO.md vom 18.10.) |
| 20261011 §1: `golisp2 --version` | erledigt 20261018 (`src/lib/version.go`) |
