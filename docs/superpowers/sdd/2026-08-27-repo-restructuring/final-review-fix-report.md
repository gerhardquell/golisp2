# Final-Review Fix Report — repo-restructuring

Date: 2026-08-27
Worktree: `/u/lisp-projekte/golisp2/.worktrees/repo-restructuring` (branch `repo-restructuring`)
Commit: `387a169` — "fix(repo): Final-Review-Funde — Test-Regression, Doku-Pfade, Generator-Header"

## Status: DONE

All four fixes applied, verified individually and together, single commit created.

---

## Fix 1 — env_closure_test.go regression (SKIP -> PASS)

File: `src/lib/env_closure_test.go`, `TestClosureScopeNoStackOverflow`.

Verified the old text matched exactly (2-space indentation, not tabs):
```go
  bin, err := filepath.Abs("../build/golisp2")
```
Changed to:
```go
  bin, err := filepath.Abs("../../build/golisp2")
```

Verification (raw, unfiltered via `rtk proxy` since the RTK hook otherwise
collapses `go test -v` output):

```
$ rtk proxy go test ./src/lib/ -run TestClosureScopeNoStackOverflow -v
=== RUN   TestClosureScopeNoStackOverflow
--- PASS: TestClosureScopeNoStackOverflow (0.01s)
PASS
ok  	golisp2/src/lib	0.013s
```

Confirmed: no longer `--- SKIP`, the regression guard for the stack-overflow
bug (frame-pooling / closure-scope bug documented in the file's own header
comment) is active again.

---

## Fix 2 — broken build/run commands in top-level docs

**README.md** (line ~125-126): exact match found and replaced.
```
go build -o build/golisp2 .
go build -o build/golisp2-client ./cmd/golisp2-client/
```
->
```
go build -o build/golisp2 ./src/
go build -o build/golisp2-client ./src/cmd/golisp2-client/
```

**README_CN.md** (line 118, in the 安装方法/installation section right after
`cd golisp2`): exact match found and replaced.
```
go build .
```
->
```
go build ./src/
```

**BESCHREIBUNG.md** — two spots, both exact matches found and replaced:

1. Near top (lines 4-5):
```
Alle Features sind im interaktiven REPL (`go run .`) und in Skript-Dateien
(`go run . skript.lisp`) verfügbar.
```
->
```
Alle Features sind im interaktiven REPL (`go run ./src`) und in Skript-Dateien
(`go run ./src skript.lisp`) verfügbar.
```

2. REPL code block (lines 276-278):
```
go run .              # REPL starten
go run . skript.lisp  # Skript ausführen
go run . -t           # Testmodus
```
->
```
go run ./src              # REPL starten
go run ./src skript.lisp  # Skript ausführen
go run ./src -t           # Testmodus
```

**`chinese/ABOUT.md` and `chinese/ABOUT_CN.md` were NOT touched**, per the
plan's explicit instruction — `chinese/` is frozen for the whole
restructuring per a Global Constraint. These two files still contain the
stale `go build .` pattern and will produce a broken build command if
followed literally.

**Concern / follow-up for Gerhard:** `chinese/ABOUT.md` and
`chinese/ABOUT_CN.md` have the same broken `go build .` command as
README.md had. Since `chinese/` is frozen for this restructuring, these
were left untouched as instructed, but they are now stale relative to the
actual repo layout and should be raised as a separate follow-up item
(outside the scope of this fix wave and this restructuring project).

---

## Fix 3 — docs/struktur.md, docs/swank.md, docs/cli.md path staleness

### docs/struktur.md

Read the file first to confirm exact content. Rewrote the tree diagram:
- `build/` (with its two binary entries) stays at the repo root, unchanged.
- `build.sh` and `tmp/` stay at the repo root, unchanged (they were not
  moved by the restructuring).
- `main.go`, `cmd/`, and `lib/` (and everything under them, including the
  `swank/` subtree) are now nested one level deeper under a new `src/`
  entry, with indentation increased by 2 spaces throughout that subtree.
- Updated the `cmd/golisp2-client/main.go` line's parenthetical from
  `(spricht SWANK, nutzt golisp2/lib)` to
  `(spricht SWANK, nutzt golisp2/src/lib)`.
- Every individual file's one-line description text was left byte-for-byte
  identical — only nesting/indentation and the one `golisp2/lib` mention
  changed.

Verified via `git diff docs/struktur.md` (post-commit `git show`): the diff
is exactly the tree reflow (50 insertions / 48 deletions, all indentation
and the `src/` insertion line) plus the single `golisp2/lib` ->
`golisp2/src/lib` text change. No file description text was altered.

### docs/swank.md

Exact match found and replaced:
```
`cmd/golisp2-client/main.go`, nutzt `golisp2/lib` für robuste Cell-Verarbeitung.
```
->
```
`src/cmd/golisp2-client/main.go`, nutzt `golisp2/src/lib` für robuste Cell-Verarbeitung.
```

### docs/cli.md

Exact match found and replaced:
```
  `main.go` liest bereits vor der Auswertung das komplette stdin via
```
->
```
  `src/main.go` liest bereits vor der Auswertung das komplette stdin via
```

---

## Fix 4 — tools/gen-reference.lisp header + regenerated artifact

Exact match found in `tools/gen-reference.lisp` (line 18, inside the
`format nil` call) and replaced per the plan:

Old:
```
"# GoLisp2 — Generierte Funktionsreferenz~%~%> Automatisch erzeugt aus `(env-symbols)` — nicht von Hand editieren.~%> Neu generieren: `./build/golisp2 tools/gen-reference.lisp`~%~%| Symbol | Beschreibung |~%|---|---|~%"
```
New:
```
"# GoLisp2 — Generierte Funktionsreferenz~%~%> Automatisch erzeugt aus `(env-symbols)`. Die Beschreibung-Spalte ist bewusst leer (Grundgerüst) und kann von Hand ergänzt werden — ACHTUNG: dieses Skript überschreibt die Datei komplett bei jedem Lauf, von Hand ergänzte Beschreibungen gehen dabei verloren.~%> Neu generieren: `./build/golisp2 tools/gen-reference.lisp`~%~%| Symbol | Beschreibung |~%|---|---|~%"
```

Regenerated the artifact:
```
$ ./build/golisp2 tools/gen-reference.lisp
258 Symbole nach docs/referenz-generiert.md geschrieben
```

New header confirmed (docs/referenz-generiert.md, lines 1-6):
```
# GoLisp2 — Generierte Funktionsreferenz

> Automatisch erzeugt aus `(env-symbols)`. Die Beschreibung-Spalte ist bewusst leer (Grundgerüst) und kann von Hand ergänzt werden — ACHTUNG: dieses Skript überschreibt die Datei komplett bei jedem Lauf, von Hand ergänzte Beschreibungen gehen dabei verloren.
> Neu generieren: `./build/golisp2 tools/gen-reference.lisp`

| Symbol | Beschreibung |
```

---

## Verification (all four fixes together)

```
$ go build ./...
(no output — success)

$ go test ./... -count=1
Go test: 379 passed in 6 packages

$ ./build.sh
→ go vet
→ build golisp2
→ build golisp2-client
✓ Binaries in ./build/:
golisp2
golisp2-client

$ ./build/golisp2 -t
...
=== Report: 104 PASS, 0 FAIL, 0 XFAIL, 0 XPASS ===

$ rtk proxy go test ./src/lib/ -run TestClosureScopeNoStackOverflow -v
=== RUN   TestClosureScopeNoStackOverflow
--- PASS: TestClosureScopeNoStackOverflow (0.01s)
PASS
ok  	golisp2/src/lib	0.013s
```

All five gates green. `TestClosureScopeNoStackOverflow` explicitly confirmed
as `--- PASS`, not `--- SKIP`.

---

## Files changed (git status before commit, all expected, no stray files)

```
 M BESCHREIBUNG.md
 M README.md
 M README_CN.md
 M docs/cli.md
 M docs/referenz-generiert.md
 M docs/struktur.md
 M docs/swank.md
 M src/lib/env_closure_test.go
 M tools/gen-reference.lisp
```

## Commit

```
commit 387a169
fix(repo): Final-Review-Funde — Test-Regression, Doku-Pfade, Generator-Header

- src/lib/env_closure_test.go: ../build -> ../../build (Test lief sonst
  still als SKIP statt zu pruefen, Regressions-Guard war abgeschaltet)
- README.md/README_CN.md/BESCHREIBUNG.md: go build/go run Kommandos auf
  ./src praefigiert (chinese/ bewusst nicht angefasst, siehe Ruling)
- docs/struktur.md, docs/swank.md, docs/cli.md: verbleibende
  main.go/cmd/lib-Pfade auf src/ korrigiert
- tools/gen-reference.lisp: Header klaert, dass die leere
  Beschreibung-Spalte Absicht ist statt "nicht editieren" zu suggerieren;
  docs/referenz-generiert.md neu generiert

Co-Authored-By: claude-sonnet-5 <noreply@anthropic.com>

9 files changed, 63 insertions(+), 61 deletions(-)
```

## Concerns

1. **chinese/ stale build commands (follow-up, not fixed here):**
   `chinese/ABOUT.md` and `chinese/ABOUT_CN.md` contain the same stale
   `go build .` pattern that README.md had before this fix. Left untouched
   per the plan's explicit Global Constraint that `chinese/` is frozen for
   this restructuring. Raise with Gerhard as a separate follow-up item.

2. No other concerns — all four fixes matched the plan's "old" text blocks
   exactly on first read (no whitespace/wording surprises), and all
   verification gates passed cleanly on the first run.
