# Task 6 Report — Doku, Arglisten, Referenz, Gesamtverifikation

## Was implementiert wurde

Alle 8 Schritte aus `task-6-brief.md` umgesetzt, reine Doku-Arbeit (kein Go-/Lisp-Code
geändert außer Beschreibungstexten in `tools/gen-reference.lisp`).

### Step 1 — SWANK-Arglisten (`src/embed/swank.lisp`)
Nach `("sigo-usage-reset" . …)` ergänzt:
```lisp
("type-of" . "(type-of x)")
("typep" . "(typep x type-spec)")
```

### Step 2 — Referenz-Beschreibungen (`tools/gen-reference.lisp`, `*ref-docs*`)
Ermittelte neuen Symbole per `./build/golisp2 -e "(filter (lambda (s) (string-contains
(symbol-name s) \"type\")) (env-symbols))"` plus `rg -n '^\(defun' src/embed/types.lisp`
(die Filtersuche fand nicht alle Helfer, da z. B. `%bad-spec`, `%struct-instance?`,
`%valid-bound?`, `%lower-ok?`, `%upper-ok?`, `%one-arg?` kein "type" im Namen tragen).
Alphabetisch passend eingefügt (18 Intern-Helfer + `%cell-type` + `*struct-types*` +
`*type-parents*` + `type-of` + `typep` = 22 neue Zeilen):

- `%bad-spec`, `%builtin-subtype?`, `%builtin-type?`, `%cell-type`, `%cons-type`,
  `%lower-ok?`, `%number-type`, `%one-arg?`, `%struct-instance?`, `%symbol-type`,
  `%type-name-warnings`, `%type-start`, `%typep-name`, `%typep-range`,
  `%typep-satisfies`, `%typep-spec`, `%upper-ok?`, `%valid-bound?`
- `*struct-types*`, `*type-parents*`
- `type-of`, `typep` (Texte aus dem Brief wörtlich übernommen)

`%cond-subtype?`, `%cond-type?`, `*condition-types*` waren schon dokumentiert (Vorarbeit
aus condition.lisp) — nicht doppelt eingetragen.

### Step 3 — Bauen und Referenz generieren
```
./build.sh                          → ✓ (go vet, beide Binaries)
./build/golisp2 tools/gen-reference.lisp
  → "309 Symbole nach docs/referenz-generiert.md geschrieben (306 mit Beschreibung, 3 leer)"
git diff --stat docs/referenz-generiert.md
  → 1 file changed, 22 insertions(+)
```
Die 3 verbleibenden leeren Zeilen (`ash`, `ash-left`, `ash-right`) sind Alt-Lücken aus
einem früheren Commit, nicht Teil dieses Tasks.

### Step 4 — `docs/lisp-semantik.md`
Abschnitt `## Typen: type-of und typep` unmittelbar vor `## \`loop\` — Iteration`
eingefügt, Text aus dem Brief wörtlich. Jede Aussage live geprüft (siehe unten).

### Step 5 — `src/embed/ki-referenz.md`
- Neuer Unterabschnitt `### Typen` in `## 4. Stdlib` (nach `### Strukturen`) mit dem
  Bullet-Text aus dem Brief.
- Neue Zeile in der Tabelle `## 9. Gotchas / Abweichungen CL` (Tabellenformat statt
  Bullet, da der Abschnitt dort eine Tabelle ist — Inhalt unverändert zum Brief):
  `| Structs sind Listen | (typep p 'cons) → t; (type-of 3.0) → integer | Structs sind eigener Typ, kein float-Subtyp |`

### Step 6 — `docs/golisp2-cheatsheet.md`
Abschnitt `### Typ-Prädikate` ist ein Codeblock, keine Tabelle → zwei Zeilen im
dortigen Format ergänzt (analog `docs/lisp-semantik.md`/Brief-Wortlaut, an Codeblock-Stil
angepasst):
```lisp
(type-of x)                       ; Typ als Symbol: integer, string, cons, Struct-Name …
(typep x 'typ)                    ; Typprüfung inkl. (or …), (and …), (not …), (integer lo hi), (satisfies f)
```

### Step 7 — Gesamtverifikation
```
go build ./...                      → ✓ (kein Output = ok)
go test ./... -count=1               → 410 passed in 6 packages (rtk-Zusammenfassung);
  rtk proxy go test ./... -count=1 (ungefiltert) → alle Pakete "ok", keine FAIL-Zeile
./build/golisp2 -t                   → "=== Report: 141 PASS, 0 FAIL, 0 XFAIL, 0 XPASS ==="
```
410 > geforderte 393 Go-Tests; 141 PASS / 0 FAIL Lisp-Suite exakt wie erwartet.

### Step 8 — Commit
```
b8ac122 docs(types): type-of/typep in Referenz, Semantik, Cheatsheet, SWANK-Arglisten
```
Enthält genau die 6 im Brief gelisteten Dateien (`git status` vor dem Commit zeigte
keine weiteren Änderungen, insbesondere keine build/- oder tmp/-Artefakte).

## Live-Verifikation jeder Doku-Aussage (ki-referenz.md ist sigo-Vorspann — Pflicht)

Alle Checks mit `./build/golisp2 -e "…"` bzw. kurzen Skripten in `./tmp/` (danach
gelöscht):

| Aussage | Befehl | Ergebnis |
|---|---|---|
| `(type-of 3)` → `integer` | `(type-of 3)` | `integer` |
| `(type-of 3.5)` → `float` | `(type-of 3.5)` | `float` |
| `(type-of :k)` → `keyword` | `(type-of :k)` | `keyword` |
| `(typep 3 'number)` → `t` | `(typep 3 'number)` | `t` |
| `(typep x '(or string symbol))` | `(typep 'a '(or string symbol))` | `t` |
| `(typep n '(integer 0 (10)))` Grenzfälle | `(list (typep 5 …) (typep 10 …) (typep 0 …) (typep -1 …))` | `(t () t ())` |
| `(typep x '(satisfies gerade?))` | eigenes Prädikat definiert, `(typep 4 '(satisfies gerade?))` | `t` |
| Kollisionswarnung `defstruct` vs. eingebauter Typ | `(defstruct string a b)` | `"WARN: defstruct string: Name ist ein eingebauter Typ …"` |
| Struct vor Condition (Precedence-Text) | `define-condition myerr` dann `defstruct myerr`, danach `(typep (make-myerr 1 2) 'myerr)` und `(type-of …)` | WARN-Text passend, `typep` → `t`, `type-of` → `myerr` (Struct gewinnt) |
| `type-of` aller Kern-Typen aus dem ki-referenz-Satz | `(list (type-of "s") (type-of 'sym) (type-of t) (type-of ()) (type-of (cons 1 2)) (type-of (lambda (x) x)) (type-of (make-hash-table)))` | `(string symbol boolean null cons function hash-table)` |
| `(type-of (chan-make))` → `compiled-function` | separat | `compiled-function` |
| `(type-of macro)` → `macro` | `defmacro m1`, `(type-of (eval 'm1))` | `macro` |
| `(typep … '(member …))`, `(eql …)`, `(integer * 10)`, `(integer 0 *)`, `(integer 0 (10))` | kombinierter `list`-Aufruf | `(t t t t ())` — alle wie erwartet |
| `deftype`/`typecase`/`check-type`/`subtypep` nicht vorhanden | `(list (bound? 'deftype) (bound? 'typecase) (bound? 'check-type) (bound? 'subtypep))` | `(() () () ())` |
| `3.0` ist `integer` | `(type-of 3.0)` | `integer` |
| Structs sind Listen: `(typep p 'cons)` → `t` | `defstruct punkt`, `(typep (make-punkt 1 2) 'cons)` | `t` |
| Brief-Beispiel Step 5 gesamt | `(list (type-of 3.0) (typep 5 '(integer 0 (10))) (typep 'a '(or string symbol)))` | `(integer t t)` — exakt erwartet |

Kein Widerspruch gefunden; jede Zeile, die in `ki-referenz.md` ergänzt wurde, ist live
bestätigt.

## Selbst-Review

- Diff enthält nur die 6 im Brief gelisteten Dateien; keine build/- oder tmp/-Artefakte
  gestaged (`./tmp/verify_typep.lisp` etc. wurden nach jedem Check gelöscht).
- `tools/gen-reference.lisp`: neue Einträge alphabetisch eingefügt (ASCII-Ordnung nach
  Präfix `%`/`*`/regulär), Stil (Kommentar-Suffix "Intern: … (types.lisp)") konsistent
  mit bestehenden Einträgen wie `%cond-*`. Eine vorbestehende Abweichung von der
  strikten Alphabetik im File (`%make-struct` vor `%loop-acc`) wurde nicht angefasst —
  nicht Teil dieses Tasks.
- `docs/referenz-generiert.md` ist vollständig aus dem Generator-Lauf, keine manuelle
  Bearbeitung.
- Brief-Texte (Step 1, 2, 4, 5-Bullet, 6) wörtlich übernommen, wo der Brief exakten
  Text vorgab. Für die Gotchas-Tabelle (Step 5, zweiter Teil) und den Cheatsheet-
  Codeblock (Step 6) war der Zielabschnitt kein Bullet-Format — dort wie im Brief
  vorgesehen ("Falls der Abschnitt keine Tabelle ist, im dortigen Format eintragen")
  ans lokale Format angepasst, Inhalt unverändert.
- Keine Datei über 1000 Zeilen betroffen; keine Header geändert (nur bestehende
  Dateien editiert, keine neuen Dateien angelegt).
- Attribution: `Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>` laut
  aktuellem System-Reminder verwendet (überschreibt den generischen Platzhalter aus
  dem Plan-Text).

## Ergebnis

Status: **DONE**. Keine offenen Punkte.

---

## Fix Round 1 (Controller-Findings, beide gefixt)

### Finding 1 (Important) — `docs/lisp-semantik.md`

Die Bullets „Alist/Plist sind keine Typen (auch in CL nicht) → `(satisfies …)`"
und „Noch nicht vorhanden: `deftype`, `check-type`, `subtypep`, `typecase`"
standen unter „**Bewusste Abweichungen von CL:**" — sind aber keine
CL-Abweichungen (Alist/Plist sind in CL selbst auch keine Typen; die
fehlenden Makros sind schlicht nicht implementiert, keine bewusste
Design-Entscheidung gegen CL). Beide Bullets in einen neuen Block
„**Nicht vorhanden / keine Typen:**" direkt nach dem Abweichungs-Block
verschoben, Wortlaut unverändert:

```markdown
**Bewusste Abweichungen von CL:**
- `3.0` ist `integer` — der Kern kennt nur `float64`.
- `type-of` liefert `integer`/`float`, nicht `fixnum`/`double-float`.
- Structs sind Listen: `(typep p 'cons)` → `t`. Ein Literal `'(punkt 1 2)`
  gilt als `punkt`, wenn `punkt` registriert ist und die Länge passt.
- Channels sind `compiled-function`; `macro` ist ein golisp2-eigener Typ.
- Keine Characters, keine Vektoren.

**Nicht vorhanden / keine Typen:**
- Alist/Plist sind keine Typen (auch in CL nicht) → `(satisfies …)`.
- Noch nicht vorhanden: `deftype`, `check-type`, `subtypep`, `typecase`.
```

### Finding 2 (Minor) — `src/embed/ki-referenz.md` §9

Die neue Zeile in der Gotchas-Tabelle vermengte zwei unabhängige Aussagen
(Struct-als-cons und 3.0-ist-integer) in einer Zeile mit vager CL-Spalte
(„Structs sind eigener Typ, kein `float`-Subtyp" beschreibt keine der
beiden Aussagen korrekt). In zwei Zeilen mit je explizitem CL-Wert
gesplittet:

```markdown
| Structs sind Listen | `(typep p 'cons)` → `t` | eigener Typ, nicht `cons` |
| Kein Float-Typ | `(type-of 3.0)` → `integer` | `single-float`/`double-float` (3.0 ist ein Float) |
```

Beide golisp2-Seiten live geprüft (`ki-referenz.md` ist sigo-Systemprompt,
jede Zeile muss stimmen):

```
$ mkdir -p ./tmp && cat > ./tmp/v5.lisp << 'EOF'
(defstruct punkt x y)
(print (typep (make-punkt 1 2) 'cons))
EOF
$ ./build/golisp2 ./tmp/v5.lisp
t
$ ./build/golisp2 -e "(type-of 3.0)"
integer
$ rm -f ./tmp/v5.lisp
```

Beide Ergebnisse bestätigen die neuen Tabellenzeilen exakt.

### Gesamtverifikation nach dem Fix

```
go build ./...                       → ✓ (kein Output)
rtk proxy go test ./... -count=1     → alle 3 getesteten Pakete "ok", keine FAIL-Zeile
./build/golisp2 -t                   → "=== Report: 141 PASS, 0 FAIL, 0 XFAIL, 0 XPASS ==="
```

Betroffene Dateien: nur `docs/lisp-semantik.md` und `src/embed/ki-referenz.md`
(reiner Markdown/Docstring-Text, keine `*ref-docs*`-Änderung nötig, also
kein Re-Run von `tools/gen-reference.lisp` erforderlich — `docs/referenz-generiert.md`
unverändert).

Commit: `da93d92` docs(types): Fix Round 1 — Nicht-Abweichungen aus CL-Block gelöst, Gotchas-Zeile gesplittet

Status: **DONE**.
