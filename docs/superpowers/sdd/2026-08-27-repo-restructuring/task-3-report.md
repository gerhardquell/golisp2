# Task 3 Report: Src-Umzug — main.go, cmd/, lib/, embed/ → src/

## Status: DONE (fix round 1/5 applied, see below)

## What was done

Followed the brief's Steps 1-9 in order.

1. **Move** (`git mv`): `main.go`, `main_test.go`, `cmd/`, `lib/`, `embed/` → `src/`.
2. **Import-path rewrite**: `grep -rlE '"golisp2/(lib|embed)' --include=*.go .` found
   exactly the 16 files listed in the brief. Ran the exact sed command
   (`s#"golisp2/(lib|embed)#"golisp2/src/\1#g'`) on all of them.
3. **build.sh**: updated `go vet`, `go build -o build/golisp2`, and
   `go build -o build/golisp2-client` target paths to `./src/...` exactly as
   specified (old/new text matched verbatim).
4. **go build ./...** verified clean after step 2, before touching CLAUDE.md.
5-7. **CLAUDE.md**: applied all 4 edit points from the brief verbatim —
   Orientierung-Block, Chokepoint-Tabelle, `specialform_shadow_test.go`
   reference, Doku-Tabelle (3-line block + separate sigo.md line), and the
   `rg 'env\.Set\("' lib/` → `src/lib/` hint. All old_string blocks matched
   the file exactly, no deviation needed.
8. **Full verification** (see output below) — found and fixed one regression
   not covered by the brief's file list (see "Deviation" below).
9. **Commit** created with the exact message from the brief, plus an added
   paragraph documenting the extra `gps_bug_test.go` fix (see below) — no
   other content changed from the brief's template.

## Deviation from the brief (flagged, not improvised past a wall)

`go test ./... -count=1` initially reported:

```
[FAIL] TestSwankSurvivesNorvigBugs
   gps_bug_test.go:58: start server: fork/exec ...: no such file or directory
```

Root cause: `src/lib/swank/gps_bug_test.go` computes `repoRoot` as
`filepath.Abs(filepath.Join("..", ".."))` relative to the test package's
directory. Before the move, the package lived at `lib/swank/` (two levels
below repo root — `../..` was correct). After the move to `src/lib/swank/`,
the package is three levels below repo root, so `../..` resolved to `src/`
instead of the actual repo root, and the test could not find
`build/golisp2` to exec.

This file was **not** in the brief's 16-file import-rewrite list (correctly
so — it has no `"golisp2/lib"` or `"golisp2/embed"` import; this is a
relative-filesystem-path issue, not an import-path issue), so the sed step
legitimately didn't touch it. It is exactly the same class of problem the
task instructions warned about ("this got broken once already in Task 1 by
an unrelated directory move"). Fixed with a single one-line change:

```go
-  repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
+  repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
```

Verified: `grep -rn 'filepath.Join("\.\.", "\.\.")' src/ --include=*.go`
found no other occurrences of this pattern — this was the only file
affected. After the fix, `go test ./... -count=1` passes fully (377/377),
and the test's own log correctly lands in the true repo-root `tmp/`
(gitignored) rather than a stray `src/tmp/` (which I removed before
committing — it was a leftover artifact from the pre-fix failing test run).

I judged this within the "you do not dispatch subagents... do this task's
work yourself" and "no regressions" mandate rather than a case to stop and
ask, since it's a single, obviously-scoped one-line fix directly required
to satisfy the brief's own Step 8 gate, and the brief anticipated exactly
this failure mode in general terms. Flagging it clearly here per
self-review discipline.

## Step 8 verification — full output

### `go build ./...`
```
(no output — clean)
```

### `go test ./... -count=1`
```
Go test: 377 passed in 6 packages
```
(Before the gps_bug_test.go fix: `376 passed, 1 failed, 1 skipped in 6
packages` — the 1 skip is `TestSwankSurvivesNorvigBugs`'s Linux-only guard
on non-Linux, unrelated; this env is Linux so it ran and failed pre-fix,
passed post-fix.)

### `./build.sh`
```
→ go vet
→ build golisp2
→ build golisp2-client
✓ Binaries in ./build/:
golisp2
golisp2-client
```

### `./build/golisp2 -t`
```
=== Report: 104 PASS, 0 FAIL, 0 XFAIL, 0 XPASS ===
```

## Commit

SHA: `8b859d5`
Subject: `refactor(repo): Go-Code nach src/ verschoben`

109 files changed, 51 insertions(+), 49 deletions(-) — the bulk are pure
`git mv` renames (100% similarity) except:
- `src/lib/stdlib.go` (98% — import path)
- `src/lib/shm_lisp.go`, `src/lib/shm_lisp_test.go` (99% — import path)
- `src/lib/swank/{dispatch,dispatch_test,env,env_test,framing,framing_test,
  integration_test,lisp,lisp_test,server}.go` (93-99% — import path)
- `src/lib/swank/gps_bug_test.go` (97% — the repoRoot fix above)
- `src/cmd/golisp2-client/main.go` (99% — import path)
- `src/main.go`, `src/main_test.go` (99% — import path)
- `CLAUDE.md`, `build.sh` (modified in place, not renamed)

## Self-review findings

- **Move completeness**: confirmed via `ls main.go main_test.go cmd lib
  embed` (all "not found" — nothing left behind at root) and `ls src/`
  (all five present, plus `tmp/` was a stray test artifact, removed before
  commit).
- **Import rewrite completeness**: `grep -rn '"golisp2/lib\|"golisp2/embed'
  --include=*.go .` after the sed → **zero hits outside `golisp2/src/`**
  (i.e. zero un-prefixed hits). Confirmed the file list matched the brief's
  16 files exactly, no more, no fewer.
- **CLAUDE.md**: all 4 edit points verified via `git diff CLAUDE.md` —
  matches the brief's exact new text for all blocks (Orientierung,
  Chokepoint-Tabelle, specialform_shadow_test.go line, Doku-Tabelle 3-line
  block + separate sigo.md line, rg-Hinweis).
- **go vet in build.sh**: points at `./src/lib/` `./src/lib/swank/` — correct.
- **Untouched directories**: confirmed `golisp2web/`, `extern/sigoREST`,
  `chinese/`, `pn-gps1/` were not part of this commit (not in the diff
  above; `pn-gps1/*.lisp` is loaded by the Lisp test suite, which passed
  104/104, confirming no path regression there either).
- **Not fixed / out of scope**: `CLAUDE.md` line ~169 (Invarianten section,
  "Singleton-Nil" paragraph) still contains a bare reference to
  `` `lib/env.go` `` in prose ("Pointer-Vergleich nach (`lib/env.go`)").
  This occurrence was **not** among the brief's 4 explicit edit points
  (Orientierung-Block, Chokepoint-Tabelle, `rg`-Hinweis, and the
  specialform_shadow_test.go / Doku-Tabelle / sigo.md lines under Step 7),
  so per "do exactly what the brief specifies... a mismatch usually means
  something upstream changed and the controller needs to know," I left it
  untouched rather than expanding scope. Flagging as a minor stale
  reference for the controller to decide on (likely a one-line follow-up:
  `lib/env.go` → `src/lib/env.go`).

## Concerns

- The one CLAUDE.md line noted above (`` `lib/env.go` `` in the Singleton-Nil
  paragraph) is now stale but was outside the brief's specified edit
  points. Low risk, low value to fix without being asked, so left as-is.
- No other concerns. All four verification gates (build, test, build.sh,
  -t) pass cleanly and the working tree is clean post-commit.

---

## Fix round 1/5

Coordinator review confirmed Concern 2 (the stale `` `lib/env.go` `` prose
reference in CLAUDE.md's Singleton-Nil + Symbol-Interning Invarianten
paragraph, line 169) as should-fix, citing CLAUDE.md's own stated policy
("wenn hier etwas dem Code widerspricht, gewinnt der Code — und die Datei
gehört korrigiert") and that the brief's Step 7 heading ("restliche
lib/-Erwähnungen") signals intent to catch exactly this.

### What was done

1. Confirmed scope with `grep -n '\`lib/\|\`main\.go\`\|\`cmd/' CLAUDE.md`:
   two hits — line 55 (`` `main.go` `` inside a generic filename-convention
   sentence, not a path reference, not stale) and line 169 (the confirmed
   stale one). No other bare `lib/`/`main.go`/`cmd/` prose mentions found,
   so the edit set was not expanded beyond the one line.
2. Made the exact one-line edit:
   ```
   -Pointer-Vergleich nach (`lib/env.go`). Eine nicht-internierte Symbol-Cell
   +Pointer-Vergleich nach (`src/lib/env.go`). Eine nicht-internierte Symbol-Cell
   ```
3. `go build ./...` — clean, no output (doc-only change, unaffected as
   expected).
4. Committed as a new commit (not amended):
   SHA `d7a259c`, subject `fix(docs): CLAUDE.md — letzter lib/env.go-Pfad
   auf src/ korrigiert`, with the standard `Co-Authored-By: claude-sonnet-5
   <noreply@anthropic.com>` trailer. 1 file changed, 1 insertion(+), 1
   deletion(-).

### Note on Concern 1 and `lib/mem.prof`

Per coordinator instruction: Concern 1 (`gps_bug_test.go` repoRoot path fix
from the original task) was judged Minor/correct-as-is — no action taken.
`src/lib/mem.prof` (a tracked binary artifact that rode along unchanged in
the original rename) was explicitly called out as out of scope for this
fix round — not touched.

### Verification

- `go build ./...`: clean.
- Working tree clean after commit (`git status --short` empty).
- No further regressions expected — this was a pure doc-string edit inside
  a Go build comment/prose block, no code or import paths touched.
