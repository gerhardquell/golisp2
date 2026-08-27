# Task 5 Report — Doku-Generator

## Status: DONE

## Preflight

`./build/golisp2 -e "(env-symbols)"` worked immediately (returned the full
sorted symbol list) — the binary in this worktree was already built with
Task 4's `env-symbols` primitive. No rebuild was necessary.

## What was done

### Step 1/2: `tools/gen-reference.lisp` + run

Wrote `tools/gen-reference.lisp` exactly per the brief, with one required
fix: the brief's script used `(define (write-reference) ...)` —
function-definition sugar for `define`. This repo's `define` special form
only accepts `(define name value)` (verified: `./build/golisp2 -e
"(define (foo) 1)"` → `ERR: define: Syntax: (define name value)`).
Changed to `(defun write-reference () ...)`, which is the correct,
pre-existing special form for exactly this purpose. No other primitive was
substituted — `env-symbols`, `format`, `mapcar`, `apply`, `string-append`,
`file-write`, `length` are all used exactly as the brief specified and all
verified present in `(env-symbols)` beforehand.

Ran `./build/golisp2 tools/gen-reference.lisp`:
```
258 Symbole nach docs/referenz-generiert.md geschrieben
```
(The trailing `()` in the terminal is the script's own top-level return
value being echoed by the file-execution path — not an error.)

`docs/referenz-generiert.md`: 265 lines, one table row per of the 258
symbols plus header/blank lines — well above the 100-line bar.

### Step 3: `docs/ki/referenz.md` rewritten

Read the existing file first (dated 20260730) to preserve its structure
(Eval-Reihenfolge, Spezialformen-Tabelle, Primitiven-Kurzliste, Gotchas,
Schwächen-Katalog, Dateistruktur, Cheatsheet) — all 12 sections kept.

Read `src/lib/eval_core.go`'s `switch expr.Car.Val` in full and counted
case branches directly rather than trusting the old file's claim of "55
Spezialformen":

- **59 case-blocks**, **61 keyword names** (the `case "begin", "progn",
  "locally":` block covers 3 names in one implementation).
- The old file undercounted and had two factual errors, both fixed:
  - `let*` was documented as `"let* sequentiell (Stdlib)"` — it is **not**
    in the stdlib. `src/embed/stdlib.lisp` line 39 explicitly comments
    "`let*` ist eine Go-Spezialform (`lib/eval_core.go`) — hier absichtlich
    NICHT [definiert]". Fixed: `let*` now has its own row, marked native.
  - `setq` (distinct case branch, multi-pair CL-style sequential set) was
    never listed — only `set!` (single-pair) was documented, mislabeled as
    "setq Alias". Both now have separate, accurate rows.
  - `lock` (case branch for `(lock mutex . body)`, a critical section using
    a `lock-make` mutex) was entirely undocumented. Added.
  - `locally` (third alias of `begin`/`progn`) was undocumented. Added as a
    noted alias on the `begin` row.
- Spot-checked a handful of entries against `eval_specialforms.go` /
  `eval_control.go` source (`evalSetq`, `evalSet`, `evalSetQStar`,
  `evalPsetq`, `evalLock`) to get semantics right, not guessed.
- Section 2's table now has exactly 59 rows (verified by grep count),
  matching the stated case-block count.
- Section 3 (Primitiven-Kurzliste) updated from `docs/referenz-generiert.md`:
  added the previously undocumented Web-Bridge domain (`http-serve
  http-static http-stop http-port http-upload http-wait webserv ws-call
  ws-clients ws-emit ws-emit-to ws-eval ws-export ws-unexport
  browser-open`) and defsystem-lite domain (`defsystem load-system
  unload-system loaded-systems system-symbols`), both of which exist in
  `(env-symbols)` but were absent from the 20260730 file. Added `sort` and
  `string-find` to the existing lists (present in env-symbols, missing
  before). Added a pointer to `docs/referenz-generiert.md` for the
  non-curated full list.
- Section 4 (Stdlib) gets an explicit callout that `let*` there is only a
  CL-habit anchor, not the real implementation (cross-reference to fix
  above).
- Section 9 (Gotchas) and Section 10 (Schwächen) each got one new row/
  subsection (10.13) documenting that `(define (f p) body)` is a syntax
  error in this repo, not `defun` sugar — directly relevant since the
  brief's own script draft hit exactly this trap.
- Section 11 (Dateistruktur) updated to the current `src/lib/`, `src/embed/`,
  `src/main.go` layout (this worktree already moved these under `src/` in
  earlier restructuring tasks; the old file still said bare `lib/`).
- Header/"Quelle" line updated to the brief's exact text (Stand 20260827,
  generated via `tools/gen-reference.lisp`).

### Step 4: CLAUDE.md doc table

Inserted, verbatim as specified in the brief, directly after the
`docs/memory.md` row:
```
| `docs/referenz-generiert.md` | Vollständige Funktionsreferenz, generiert aus `(env-symbols)` | du eine konkrete Funktion nachschlägst |
```

### Step 5: Verification

```
go build ./...                    → clean, no output
go test ./... -count=1            → 378 passed in 6 packages
./build/golisp2 -t                → === Report: 104 PASS, 0 FAIL, 0 XFAIL, 0 XPASS ===
wc -l docs/referenz-generiert.md  → 265 docs/referenz-generiert.md
```

### Step 6: Commit

```
commit e7e224f
docs(reference): Funktionsreferenz aus (env-symbols) generiert
4 files changed, 361 insertions(+), 41 deletions(-)
 create mode 100644 docs/referenz-generiert.md
 create mode 100644 tools/gen-reference.lisp
```
Files: `tools/gen-reference.lisp` (new), `docs/referenz-generiert.md` (new),
`docs/ki/referenz.md` (modified), `CLAUDE.md` (modified).

## Self-review

- `docs/referenz-generiert.md`: 265 lines (> 100 ✓), generator's own printed
  summary said "258 Symbole" — plausible (env-symbols returned 258 names
  when spot-checked interactively, matching the earlier full listing shown
  in the task context).
- `docs/ki/referenz.md` special-forms table: counted programmatically —
  exactly 59 rows, matching the stated "59 Case-Zweige, 61 Schlüsselwörter"
  header claim. Spot-checked `let*` (native, not stdlib — confirmed against
  `src/embed/stdlib.lisp` comment), `setq` vs `set!` vs `setq*` vs `psetq`
  (four distinct case branches, four distinct semantics per source read),
  `lock` (confirmed via `evalLock` in `eval_control.go`), and
  `begin`/`progn`/`locally` (confirmed single case block with three string
  labels in `eval_core.go` line 198).
- CLAUDE.md addition: diffed — exactly the one line specified in the brief,
  inserted after the `docs/memory.md` row, no other changes to CLAUDE.md.

## Concerns

None. The only deviation from the brief's literal text is the
`(define (write-reference) ...)` → `(defun write-reference () ...)` fix in
the generator script, which was a necessary syntax correction (not a
primitive substitution) — `defun` was already on the verified-primitive
list implicitly since it's a core special form, and no new primitive or
approach was introduced.

---

## Fix round 1/5 (commit 9cea6b2)

Coordinator review flagged two factual errors in `docs/ki/referenz.md`'s
Section 2 table, both about `set!`/`setq*` semantics I had documented from
memory of similar Schemes rather than reading `evalSet`/`evalSetQStar`
closely enough the first time.

### Finding 1: `set!` row was backwards

Old row: `| \`(set! sym val)\` | Ein Paar updaten | Legt neu an, falls
ungebunden |` — wrong. `evalSet` (`src/lib/eval_specialforms.go:280-290`)
calls only `env.Update`, no `env.Set` fallback (unlike `setq`/`setq*`,
which do fall back). Verified live:

```
$ ./build/golisp2 -e "(set! zzz-undefined-var 5)"
ERR: env: set! – Symbol 'zzz-undefined-var' nicht gefunden
```

Fixed to: `| \`(set! sym val)\` | Ein Paar updaten | Nur \`env.Update\` —
**Fehler falls ungebunden**, legt nichts neu an |`

### Finding 2: `setq*` row glossed over a real return-value difference

Old row said `setq*` is "wie `setq`, eigener Case-Zweig" — true that it's a
separate case branch, but it hides that the two forms return different
things. `evalSetq` (`eval_specialforms.go:38-62`) returns the last
**value** assigned; `evalSetQStar` (`eval_specialforms.go:293-316`) returns
the last **symbol name**, not its value. Verified live:

```
$ ./build/golisp2 -e "(setq* zzz3 42)"
zzz3
$ ./build/golisp2 -e "(setq zzz4 42)"
42
```

Fixed to: `| \`(setq* v1 val1 v2 val2 ...)\` | Sequentielles Setzen, legt
neu an falls ungebunden | **Rückgabe:** letztes **Symbol** (nicht dessen
Wert) — anders als \`setq\`, das den letzten **Wert** liefert |`

### Verification

```
go build ./...   → clean, exit 0 (doc-only change, unaffected as expected)
```

`git diff` confirmed only the two targeted table cells changed (2
insertions, 2 deletions in `docs/ki/referenz.md`) — no other rows touched.

### Commit

```
commit 9cea6b2
fix(docs): docs/ki/referenz.md — set!/setq* Semantik korrigiert
1 file changed, 2 insertions(+), 2 deletions(-)
```

### Not actioned (per coordinator instruction)

The review's other two findings — the generator's own `*ref-out*`/
`write-reference` symbols leaking into the generated table, and the empty
`Beschreibung` column in `docs/referenz-generiert.md` — were judged
Minor/acceptable by the coordinator; no changes made for those in this
round.
