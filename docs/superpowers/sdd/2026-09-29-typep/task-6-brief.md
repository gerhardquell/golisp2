### Task 6: Doku, Arglisten, Referenz, Gesamtverifikation

**Files:**
- Modify: `src/embed/swank.lisp` (Arglisten-Tabelle, Einträge im Format `("name" . "(name args)")`)
- Modify: `tools/gen-reference.lisp` (`*ref-docs*`, Einträge `(sym . "Beschreibung")`)
- Regenerate: `docs/referenz-generiert.md`
- Modify: `src/embed/ki-referenz.md`, `docs/lisp-semantik.md`, `docs/golisp2-cheatsheet.md`

**Interfaces:**
- Consumes: alles aus Tasks 1–5.
- Produces: Doku; keine Code-Schnittstellen.

- [ ] **Step 1: SWANK-Arglisten** — in `src/embed/swank.lisp` bei den übrigen Einträgen (z. B. nach `("sigo-usage-reset" . …)`) ergänzen:

```lisp
    ("type-of" . "(type-of x)")
    ("typep" . "(typep x type-spec)")
```

- [ ] **Step 2: Referenz-Beschreibungen** — in `tools/gen-reference.lisp`, Liste `*ref-docs*`, alphabetisch passend einfügen:

```lisp
    (%cell-type . "Intern: Kern-Typ einer Cell als Symbol (celltype.go)")
    (type-of . "Typ eines Werts als Symbol (integer, string, cons, Struct-/Condition-Name …)")
    (typep . "Typprüfung: (typep x 'number), (or …), (and …), (not …), (member …), (satisfies f), (integer lo hi)")
```

Weitere neue Symbole (`%builtin-type?`, `%type-start`, …) erscheinen automatisch mit leerer Beschreibung — für jedes einen Eintrag `"Intern: … (types.lisp)"` ergänzen. Liste der neuen Symbole: `./build/golisp2 -e "(filter (lambda (s) (string-contains (symbol-name s) \"type\")) (env-symbols))"` (Ausgabe prüfen, Funktionsnamen ggf. über `rg -n 'defun' src/embed/types.lisp` ergänzen).

- [ ] **Step 3: Bauen und Referenz generieren**

```bash
./build.sh
./build/golisp2 tools/gen-reference.lisp
git diff --stat docs/referenz-generiert.md
```

Expected: Build ok, `referenz-generiert.md` enthält `type-of`, `typep`, `%cell-type` mit Beschreibung.

- [ ] **Step 4: `docs/lisp-semantik.md`** — neuen Abschnitt vor `## `loop` — Iteration` einfügen:

````markdown
## Typen: `type-of` und `typep`

```lisp
(type-of 3)                        ; → integer
(type-of 3.5)                      ; → float
(type-of :k)                       ; → keyword
(typep 3 'number)                  ; → t   (integer → rational → real → number)
(typep x '(or string symbol))
(typep n '(integer 0 (10)))        ; 0 ≤ n < 10
(typep x '(satisfies gerade?))     ; eigenes Prädikat
```

Hierarchie und Typnamen stehen an genau einer Stelle: `*type-parents*`
in `src/embed/types.lisp`. Structs (`defstruct`) und Conditions
(`define-condition`) sind eigene Typen; ihre Namen kollidieren laut
(`WARN:`), nie still — der eingebaute Typ gewinnt vor Struct, Struct vor
Condition.

**Bewusste Abweichungen von CL:**
- `3.0` ist `integer` — der Kern kennt nur `float64`.
- `type-of` liefert `integer`/`float`, nicht `fixnum`/`double-float`.
- Structs sind Listen: `(typep p 'cons)` → `t`. Ein Literal `'(punkt 1 2)`
  gilt als `punkt`, wenn `punkt` registriert ist und die Länge passt.
- Channels sind `compiled-function`; `macro` ist ein golisp2-eigener Typ.
- Keine Characters, keine Vektoren.
- Alist/Plist sind keine Typen (auch in CL nicht) → `(satisfies …)`.
- Noch nicht vorhanden: `deftype`, `check-type`, `subtypep`, `typecase`.
````

- [ ] **Step 5: `src/embed/ki-referenz.md`** — dies ist der Vorspann für `sigo`-Calls, jede Aussage muss stimmen. In Abschnitt `## 4. Stdlib` einen Eintrag ergänzen:

```markdown
- `(type-of x)` → `integer` `float` `string` `symbol` `keyword` `boolean` `null` `cons` `function` `compiled-function` `macro` `hash-table`, Struct- oder Condition-Name. `(typep x spec)` mit Typnamen, `(or …)` `(and …)` `(not …)` `(member …)` `(eql x)` `(satisfies f)` `(integer lo hi)` (`(n)` = exklusiv, `*` = offen). `3.0` ist `integer`. Kein `deftype`/`typecase`.
```

und in `## 9. Gotchas / Abweichungen CL` eine Zeile:

```markdown
- Structs sind Listen: `(typep p 'cons)` → `t`; `(type-of 3.0)` → `integer`.
```

Danach jede Aussage live prüfen, z. B.:

```bash
./build/golisp2 -e "(list (type-of 3.0) (typep 5 '(integer 0 (10))) (typep 'a '(or string symbol)))"
```

Expected: `(integer t t)`.

- [ ] **Step 6: `docs/golisp2-cheatsheet.md`** — im Abschnitt zu Prädikaten/Typen (per `rg -n 'number\?' docs/golisp2-cheatsheet.md` finden) ergänzen:

```markdown
| `(type-of x)` | Typ als Symbol: `integer`, `string`, `cons`, Struct-Name … |
| `(typep x 'typ)` | Typprüfung inkl. `(or …)`, `(and …)`, `(not …)`, `(integer lo hi)`, `(satisfies f)` |
```

Falls der Abschnitt keine Tabelle ist, im dortigen Format eintragen.

- [ ] **Step 7: Gesamtverifikation**

```bash
go build ./...
go test ./... -count=1
./build/golisp2 -t
```

Expected: alles grün; Anzahl Go-Tests > 393, Lisp-Suite 141 PASS / 0 FAIL (oder mehr PASS).

- [ ] **Step 8: Commit**

```bash
git add src/embed/swank.lisp tools/gen-reference.lisp docs/referenz-generiert.md src/embed/ki-referenz.md docs/lisp-semantik.md docs/golisp2-cheatsheet.md
git commit -m "docs(types): type-of/typep in Referenz, Semantik, Cheatsheet, SWANK-Arglisten

Co-Authored-By: <dein eigenes Modell> <noreply@anthropic.com>"
```
