# TODO golisp2 – Aufgaben und Fragen

Zusammengeführt am 19.10.2026 aus `TODO.md` (Stand 18.10.) und drei älteren TODO-Dateien
vom 28.09., 03.10. und 11.10. (Verlauf, nicht im öffentlichen Repo). Ihre offenen Punkte sind mit `- [>]` als „übertragen“ markiert.
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

- [x] `mod` rechnet wie `remainder` (Richtung Null): `(mod -3 2)` → `-1`, CL → `1`
      (CL-`mod` rundet ab, Vorzeichen des Divisors). `rem` als CL-Name fehlt.
      Erledigt 20.10.: `mod` wie CL, neues `rem`, `remainder` unverändert (Richtung Null).

## 3. Fehlende Funktionen

- [x] CL-Zahlenprädikate fehlen: `evenp`, `oddp`, `plusp`, `minusp`, `1+`, `1-`
      (`zerop` gibt es; geprüft 20.10. beim Tutorial-Ausbau). KIs schreiben sie.
      Erledigt 20.10. (`stdlib.lisp`, `plusp`/`minusp` delegieren an `positive?`/`negative?`).

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
- [x] Schritt 4: Nach Schritt 2–3 neu messen (Vorgehen: TODO vom 28.09., Abschnitt 4).
      Greift `cached_tokens` mit dem Vorspann? Thinking-Anteil? Antworten golisp2-korrekt?
      Erledigt 20.10. (`docs/sigo.md`, „Messung 20261020"): Cache 99,5 % ab 2. Call, beide
      Modelle denken (~90 % der Completion), `zai-glm53-f` 11/12 korrekt bei 1/14 der Kosten.
- [x] Cache-Feldnamen Moonshot/Gemini/DeepSeek. Erledigt 20.10., nichts zu ändern: sigoREST
      normalisiert `usage`; je 2 Calls gemessen, 2. Call meldet Cache bei `moo-kimik26` (6144),
      `che-ds4-f`/`ope-ds4-f-2` (7936), `che-gem3-f` (4082). (`docs/sigo.md`)
- [ ] sigoREST (anderes Repo): kimi-Modelle (`moo-kimik26`, `-27-cod`, `-27-codhs`, `-k3`) stehen
      mit temperature 0–2 in der Registry, Moonshot erlaubt nur 1 → HTTP 400, seit golisp2
      0.2 schickt (Schritt 3). Fix: min = max = 1, dann erzwingt sigoREST den Wert.
      Umgehung bis dahin: `GOLISP_SIGO_TEMPERATURE=1` oder `temperature` 1 per `sigo-request`.
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

## 6. Offen nach dem Ganzzahl-Typ (16.11.2026, Stand 0.31.1)

Ganzzahl-Typ (`int64` neben `float64`) ist gemergt und gepusht. Regeln:
`docs/lisp-semantik.md` (Zahlen), `docs/golisp2-fehler.md` (Abschnitt G).

### Entscheidungen (Gerhard)

- [ ] `ash-left`/`ash-right` (`stdlib.lisp`) haben keine Aufrufer mehr — löschen?
      (wäre Breaking). `MakeNumber` (`types_helpers.go`) am 17.11. gelöscht.
- [ ] Zellgröße 112 → 96 Byte: `LispType` als `uint8`, `SrcLine` als `int32`.
      Eigener Schritt, vorher und nachher messen.
- [x] Worktree `.worktrees/ganzzahl-typ` und Zweig `ganzzahl-typ` entfernen? Erledigt 17.11.

### Fehler (geprüft 16.11., nicht angefasst)

- [x] `(random)` ohne Argument liefert eine große Ganzzahl statt `[0,1)`.
      `tests/genetic-programming-demo.lisp` und `tests/demo-utils.lisp` rechnen mit
      `(< (random) 0.4)` deshalb falsch. Erledigt 17.11. (0.32.0): Float aus `[0,1)`.
- [x] `~va` verbraucht den Breiten-Parameter in falscher Reihenfolge:
      `(format nil "~va|" 5 3)` → `"5  |"` statt `"3    |"`. Bei `~vd` schon behoben.
      Erledigt 17.11.: `~a`/`~s` lösen v-Parameter vor dem Wert auf.
- [x] `(length "abc")` → `cdr: Liste erwartet`. Erledigt 17.11.: `length` kennt Strings.
- [x] `~r` kürzt einen Float still (`~d` druckt ihn seit 0.31.0 wie `~a`). Erledigt 17.11.:
      `~r` ohne Basis → Fehler (CL), `~nr` wie `~d`. `~p` war kein Fehler (1.5 ≠ 1 → Plural, CL).

- [x] `reduce` ohne `:initial-value` wendet `:key` nicht auf das erste Element an:
      `(reduce #'+ '(1 2 3) :key (lambda (e) (* e 2)))` → `11`, CL → `12`
      (`stdlib.lisp:101`, `(car s)` geht ohne key in die Faltung). Gefunden 17.11., behoben 17.11.

### Noch zu prüfen

- [ ] Postgres gegen echte Datenbank: `int2/int4/int8` exakt, Ganzzahl-Parameter als `int64`.
- [x] sigo gegen sigoREST: Token-Zahlen als Ganzzahl. Geprüft 17.11.: `sigo*` liefert
      `prompt-/completion-/cached-/reasoning-tokens` als `integer`, `cost-usd`/`elapsed` als `float`.
- [ ] SLIME/Emacs: Inspector und REPL mit den neuen Druckformen (`3.0`).
- [x] Code außerhalb des Repos auf die Breaking Changes durchsehen (golisp2web,
      Space Beagle, `/u/ki-projekte`): `~N:a`, kleines `~x`, `(eql 1 1.0)`, Druckform
      `4.0`, Float als Index, Überlauf als Fehler. Erledigt 17.11. (Space Beagle ausgenommen,
      eigenständig): Testsuiten von golisp2-lernen, nexora, rag-medizin und 173 Kartentests
      laufen mit 0.32.0 wie mit 0.24.1; statische Suche nach den Mustern ohne echten Treffer.
      Gefunden: nur die Karte `ganzzahl-pruefen` (s. u.).
- [x] Installiert ist noch `/usr/local/bin/golisp2` 0.24.1 — alle externen Projekte laufen
      auf dem alten Stand. Deploy bringt die Breaking Changes seit 0.25.0 dorthin.
      Erledigt 17.11.: 0.33.0 installiert, externe Suiten und Kartentests unverändert.

### Doku in anderen Repos

- [x] Kopien der KI-Referenz sagen noch „Zahl (64-bit Float)“: `golisp2-ide`,
      `ai-vergleiche`, `OpenCEM`, `sixhat`, `lispbuch`. Erledigt 17.11.: Es waren alte
      eingebettete Fassungen der golisp2-Zutat in deren `CLAUDE.md`; ersetzt durch die aktuelle
      Zutat (auch in `cltl2`), Zutat selbst korrigiert. Dort jeweils nicht committet.
- [x] nexora `CLAUDE.md`: alter golisp2-Abschnitt mit eigenen Projekt-Notizen (Records,
      Hash-Tabellen, „Keine Vektoren“ — teils veraltet). Gerhard entscheidet: ersetzen und
      Notizen übernehmen/streichen? Erledigt 17.11.: ersetzt, gültige Notizen übernommen.
- [ ] Lispbuch Kap. 16 („Float-only“) nachziehen.
- [x] golisp2-lernen, Karte `ganzzahl-pruefen` (`karten/terminalUndProzesse.lisp:16`) lehrt
      die alte Semantik: „`integerp` fehlt“, „`(typep 3.0 'integer)` → t, jede Zahl ist float64“.
      Erledigt 17.11., ebenso Karte `zufall` (`(random 1.0)` → 0, „shuffle fehlt“); dort
      nicht committet, Import in die DB steht aus.

### Danach (aus der Analyse vom 15.11.)

- [x] Dynamische Variablen (`defvar` mit echter Bindung über `let`) — braucht Spec.
      Erledigt 17.11. (0.33.0, bis 2ca165f, gepusht, installiert). Regeln: `docs/lisp-semantik.md`,
      `docs/golisp2-fehler.md` Abschnitt H.
- [ ] Eingeschränktes Env für KI-Code (`(eval (read (sigo …)))`) — braucht Spec.
- [ ] `destructuring-bind` vollständig; `setf` auf Listen ändert statt neu zu binden.

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
