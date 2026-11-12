# Retrospektive 2026-10-29 — defclass, SWANK-Inspector, string<, Sequenzen

**Ergebnis:** 0.19.0 → 0.24.0, 17 Commits auf main, alle gepusht (bis 1588082).
Go-Tests: 634 grün, Lisp-Suite 168 → 174 PASS.

- **defclass** (0.20.0): Objekte mit Identität als eql-Hashtabelle mit
  `:class`; `setf` ändert in place über neues `*setf-mutators*`; Dispatch
  in `%generic-dispatch`; `type-of`/`typep`/`subtypep` kennen Klassen;
  Druckform `#<klasse>`; describe-Texte in SWANK.
- **Hashtabellen:** zwei Bugs in equal-Tabellen — alle Tabellen/Lambdas
  kollidierten über ihre Druckform; mein Fix hatte selbst eine Kollision
  (`"a s:b"` vs. `"a" "b"`), gefunden vom automatischen Security-Review.
- **SWANK-Inspector** (0.21.0): Spec, Plan, drei Agenten, Doku. Reines
  Lisp in `swank.lisp`, `*inspectee*` statt `*`.
- **string=/string</…** (0.22.0), damit Inspector-Hashtabellen sortiert.
- **Vektoren** Stufe 1 (0.23.0): `mapcar`/`apply`/`append`/`shell-assoc`
  lieferten bei Vektoren still `()`/`0` — jetzt laut; `sort` nimmt Vektoren.
  Stufe 2 (0.24.0): `#(…)`, `vectorp`/`arrayp`/`vector-pop`, `json-encode`,
  `loop across`, Sequenz-Kern (`elt subseq map find position count …`),
  `eql` als Go-Primitive.
- **Fixes nebenbei:** `defstruct`/`defclass` mit Listen-Namen (CL-Gewohnheit
  `(:include …)`) legten still kaputte Typen an — jetzt Fehler.
- **Doku-Wahrheit:** `ki-referenz.md` Abschnitt 10 (DE/EN/CN) behauptete
  fehlendes `loop`, `typecase`, `define`-Zucker; Lispbuch Kap. 2, 3, 4, 15,
  17, 25, 27 und cltl2-CLOS-Tabellen korrigiert.

**Was gut lief**
- **Live prüfen statt Text glauben — wieder bestätigt.** Ein Lauf durch
  die „Kein …“-Liste der `ki-referenz.md` fand vier falsche Aussagen;
  Lispbuch Kap. 15 erklärte drei vorhandene Features für fehlend; cltl2
  markierte `make-instance` als vorhanden. Jede Korrektur stützte sich auf
  einen Lauf gegen das aktuelle Binary.
- **Rauchtest vor dem Test-Schreiben fand den Design-Fehler.** Der erste
  `defclass`-Versuch scheiterte, weil `setf` registrierte Accessoren mit
  Wertsemantik behandelt (`(set! obj (setter obj v))`) — `s` wurde an 80
  gebunden. Das führte zu `*setf-mutators*`, das zugleich verschachtelte
  Places erlaubt.
- **Client-Quelle statt Spec.** Das Inspector-Protokoll kam aus `slime.el`
  2.27 (`slime-open-inspector`, `slime-inspector-insert-ispec`); es
  funktionierte im ersten Anlauf.
- **Batch-Emacs ersetzt Gerhards Terminal.** `emacs --batch` mit
  `slime-repl` hat den Inspector als echter SLIME-Client getestet
  (`C-c I`, `RET`, `l`, `e`, `g`, `[--more--]`). Skript:
  `./tmp/insp-test.el`. Nötig: `slime-setup '(slime-repl)` (sonst
  `slime-dcase failed` bei `:write-string`) und `y-or-n-p` abfangen
  (Versionsabfrage).
- **Agenten mit geprüftem Fakten-Brief.** Wer dem Agenten die live
  verifizierten Tatsachen mitgibt (Tabelle „Behauptung | Wahrheit“), bekommt
  Ergebnisse ohne Abweichung; Stichproben danach fanden nur Kleinigkeiten
  (`; -> nil` statt `()`).
- **Optionen mit Preview** (AskUserQuestion) für den Sequenz-Umfang: eine
  Antwort, keine Nachfragen.

**Was schwierig war / Fehler**
- **Eigener Fix mit neuem Bug.** String-Schlüssel aus zusammengesetzten
  Werten brauchen eindeutige Kodierung der Blätter (`strconv.Quote`), sonst
  wird Inhalt als Struktur gelesen. Gefunden hat es das Security-Review,
  nicht ich.
- **`pkill -f`/`pgrep -f` traf die eigene Shell** (Exit 144, zweimal), weil
  das Muster in der Kommandozeile der Shell selbst steht. Lösung: PID-Datei
  (`echo $! > tmp/x.pid`) oder verankertes Muster `^./build/golisp2 …`.
- **grep auf Meldungstexte versteckt Fehlschläge.** `grep 'FAIL|want'`
  verschluckte `t.Errorf("Warnung fehlt …")`; ein roter Test sah grün aus.
  Auf `--- FAIL` greppen, nicht auf Inhalte.
- **RTK filtert `go test -v`** — Zählungen brauchen `rtk proxy`.
- **`kh search` gibt es nicht**, der Befehl heißt `kh suche`.
- **CL-Konvention trifft Lisp-1:** SLIME bindet `*` ans inspizierte Objekt —
  in golisp2 die Multiplikation. Erst beim Spec-Schreiben bemerkt.

**Lektionen**
1. Jede Schleife `for x != nil && x.Type == LIST` über **Nutzerdaten** ist
   ein stiller Fehler, sobald ein neuer Datentyp kommt. Für Primitiven
   `properList` (primitives.go) nehmen; Spezialformen über Code sind
   unkritisch.
2. Neuer Container-Typ ⇒ sofort fragen: Wer iteriert über Listen und sieht
   den Typ still als `()`? (Vektoren kamen 0.17, die stillen `()` blieben
   bis 0.23.)
3. Eingebaute Namen gegen *.lisp prüfen: `gps2.lisp` hatte ein eigenes
   `some`, `stdlib.lisp` ein eigenes `eql` — Redefine-Guard hätte gewarnt,
   besser vorher entfernen.
4. Doku-Aussagen der Form „kein X“ altern am schnellsten. Bei jedem neuen
   Feature `rg -i 'kein.*X|nicht vorhanden|❌'` über Doku und Lispbuch.

**Offen**
- Lispbuch: `src/ch2/` (lispbuch2-Kopie) hat alle alten Aussagen; Repo
  bleibt bewusst lokal (5 Commits). Kap. 15 Header-Zeile „Inhalt“ ohne
  `handler-case` (Header-Regel).
- Zutat-Kopien `golisp2.md` in artemis, DataFlowSystem, bend2: „10.2 Kein CLOS“.
- cltl2: `docs/` nie committet, Korrekturen nur auf Platte.
- Vektoren Stufe 3 (mehrdimensional, Fill-Pointer) erst bei Bedarf;
  Laufzeit der Sequenzfunktionen (kopieren je einmal) nicht gemessen.
- Inspector im echten Emacs-Fenster (Tastatur) nicht getestet, nur Batch.
- `RETROSPECTIVE.md` steht im Arbeitsbaum als gelöscht, Inhalt liegt
  unverändert in `RETROSPECTIVE.md.d/` (Verschiebung vom 20261020) —
  uncommittet, Gerhards Entscheidung.
