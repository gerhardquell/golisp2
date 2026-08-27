# SDD ledger — plan: docs/superpowers/plans/2026-08-27-repo-restructuring.md

Worktree: /u/lisp-projekte/golisp2/.worktrees/repo-restructuring (branch repo-restructuring)
Spec: docs/superpowers/specs/2026-08-27-repo-restructuring-design.md
Worktree baseline: go build ./... OK, go test ./... -count=1 377 passed / 1 skipped
(1 fail on first run was TestSwankSurvivesNorvigBugs — missing build/golisp2
binary in fresh worktree, not a real failure; `./build.sh` then `-run` alone: PASS)

## Preflight conflict scan

| Task pair | Shared file/interface | Finding |
|---|---|---|
| 1 & 2 | CLAUDE.md | Different lines (PerfTODO row vs. doc/-table rows) — no overlap. Clean. |
| 2 & 3 | CLAUDE.md doc-table rows | Task 3's old_string for the table is written against the *post-Task-2* (docs/-prefixed) state — intentional sequencing, already correct in plan text. Clean. |
| 3 & 4 | src/lib/primitives.go | Task 4 assumes src/ already exists (Task 3 output) — consistent. Clean. |
| 4 & 5 | `(env-symbols)` interface | Task 4 Produces / Task 5 Consumes both specify: 0 args, returns sorted string list. Consistent. |
| 4 & 5 | build/golisp2 binary | **Conflict found**: Task 5 Step 2 runs `./build/golisp2 tools/gen-reference.lisp`, which needs the binary to contain Task 4's new primitive — but Task 4's original Step 5 only ran `go build ./...` / `go test ./...`, never rebuilding `build/golisp2` itself. Ruling below. |
| 5 & 6 | docs/referenz-generiert.md, build/golisp2 | Task 6 re-verifies via `./build.sh` (idempotent rebuild) + `./build/golisp2 -t` — fine given the Task 4 fix. Clean. |

**Ruling:** Added `./build.sh` to Task 4 Step 5 (plan file amended, commit
`681e81e` on main, cherry-picked as `06333e6` into this worktree) — Task 5
now always runs against a binary that actually contains `(env-symbols)`.
Cost if wrong: negligible, `./build.sh` is idempotent and the alternative
(Task 5 silently using a stale binary) would have produced a broken
generator run that's obvious immediately (missing/short output file).

No other conflicts found. Global Constraints re-read: one-commit-per-task,
golisp2web/extern/chinese untouched, docs/gespraeche + dated
retrospectives/plans/specs left alone, go.mod module name unchanged,
`go build ./...` must pass after every task — all carried into task briefs.

## Task log

Task 1: implementer (haiku) reported DONE, commit dfe9169. Controller
manual spot-check (not part of brief's Step 4 grep) found a likely
regression: brief's Step 4 verification grep excluded test-code hits;
running `go test ./lib/swank/ -run TestSwankSurvivesNorvigBugs -v` fails
with `load: 'pn-gps1/gps-norvig-bugs.lisp' nicht gefunden in Suchpfaden`
after pn-gps1/ moved to unused/. `main.go:103,105` also loads
`pn-gps1/gps-norvig-bugs.lisp` / `pn-gps1/gps2-tests.lisp` for the
`-t` Lisp-Testsuite. This falsifies the spec's own claim ("pn-gps1: kein
Bezug zum golisp2-Kern") — pn-gps1/gps-norvig-bugs.lisp is a regression
fixture wired into both the Go test suite and the `-t` CLI flag, not a
dead side-project.
Dispatched task-reviewer (sonnet) with this as a named risk to
independently confirm before ruling. Review package:
review-06333e6..dfe9169.diff

Review verdict: Critical, confirmed (test run FAIL, main.go:103,105 also
affected). Task quality: Needs fixes.

Ruling: pn-gps1/ stays at root (Gerhard, via AskUserQuestion, 2026-08-27)
— CLAUDE.md's breaking-change-decision rule applied (this falsifies the
spec's own premise, not just an implementation slip). Spec + plan
corrected on main (commit af049de) and cherry-picked into this worktree
(fc5ca08). Cost if wrong: none identified — reverting one directory move
is trivially safe and matches the verified-working pre-task state.

Task 1: fix round 1/5 dispatched to original implementer (resume
ad879720b87a6c936) — revert pn-gps1 move only, keep rest of commit dfe9169
untouched.
Task 1: fix round 1/5 (2 addressed, 0 open; commits dfe9169..a2ebf57)
Task 1: complete (commits 06333e6..a2ebf57, review clean after 1 fix round)

Task 2: implementer (haiku) reported DONE, commit e7e083f. Report mentions
"internal cross-references in moved docs/ files updated for consistency"
— NOT in the brief's closed edit list. Flagged as named risk (possible
unrequested scope) for the task-reviewer (sonnet) to verify. Review
package: review-a2ebf57..e7e083f.diff

Review verdict: Approved, no Critical/Important. 2 Minor findings (both
the named-risk scope creep: 2 extra CLAUDE.md doc/-line fixes outside the
brief's closed set, and doc/-internal-cross-reference rewrites inside 8 of
the 13 moved files) — judged correct in content but unrequested; the
reviewer notes the brief's own Step 5 grep would have failed without
these fixes (brief gap, not implementer error). No fix loop (Minor only).
Task 2: minor (deferred): CLAUDE.md 2 extra doc/->docs/ line fixes outside brief's Step 2/3 (diff e7e083f)
Task 2: minor (deferred): 8 moved docs/ files got internal doc/->docs/ cross-ref rewrites, not in brief's git-mv-only list (diff e7e083f) — content correct, brief itself was incomplete here
Task 2: complete (commits a2ebf57..e7e083f, review clean, 2 minor deferred)

Task 3: implementer (sonnet) reported DONE_WITH_CONCERNS, commit 8b859d5.
go build/go test(377/377)/build.sh/-t(104/0) all reported passing.
Concern 1: gps_bug_test.go had hardcoded ../.. repoRoot broken by the
extra directory level, fixed to ../../.. (1-line, not in brief's sed-target
list since no golisp2/lib import). Concern 2 (implementer flagged,
controller confirmed): CLAUDE.md:169 still says `lib/env.go` (missing
src/ prefix), not among brief's 4 edit points, left untouched.
Controller verified: go.mod's gorilla/websocket "// indirect" lint
diagnostic pre-dates this task (present at e7e083f already) — out of
scope, noted for final review only, not this task's regression.
Dispatched task-reviewer (sonnet) with both concerns as named risks.
Review package: review-e7e083f..8b859d5.diff

Review verdict: Approved overall, 1 Important (Concern 2 confirmed
should-fix), Concern 1 judged Minor/correct-as-is. Also noted:
lib/mem.prof (tracked binary artifact) rode along to src/lib/mem.prof
unchanged — pre-existing, out of scope, deferred to final review.
Task 3: fix round 1/5 dispatched (resume a45b156d277bdccba) — CLAUDE.md:169
lib/env.go -> src/lib/env.go, one line only.
Task 3: fix round 1/5 (1 addressed, 0 open; commits 8b859d5..d7a259c)
Task 3: complete (commits e7e083f..d7a259c, review clean after 1 fix round;
minor deferred: lib/mem.prof tracked-binary-artifact rode along unchanged
to src/lib/mem.prof — pre-existing, separate future cleanup)

Task 4: implementer (haiku) reported DONE, commit 3354403. 378/378 tests
pass. Deviation: brief's test checked for "defun" among env-symbols, but
implementer says defun is a special form (eval_core.go switch), not an
env binding — substituted "print" instead. Flagged as named risk for
task-reviewer (sonnet) to independently verify against eval_core.go.
Review package: review-d7a259c..3354403.diff

Review verdict: Approved, no findings. "defun" deviation independently
verified against eval_core.go:140 (case "defun" -> evalDefun, special
form, never env.Set) and env.go:280 (Symbols() only enumerates e.vars) —
confirmed correct, brief's literal test text was itself wrong.
Task 4: complete (commits d7a259c..3354403, review clean, no findings)

Task 5: implementer (sonnet) dispatched — Doku-Generator (tools/gen-reference.lisp,
docs/referenz-generiert.md, docs/ki/referenz.md rewrite, CLAUDE.md
doc-table addition). Instructed to check binary has (env-symbols) before
running generator (Task-4-preflight-fix from earlier ruling).
Task 5: implementer (sonnet) reported DONE, commit e7e224f. 378 tests
pass, 104/0 Lisp suite, docs/referenz-generiert.md 265 lines/258 symbols.
Deviation: brief's script used `(define (f) ...)` function-def sugar this
repo's `define` doesn't support, fixed to `(defun f () ...)` — no new
primitive introduced per implementer. Flagged as named risk for
task-reviewer (sonnet) to verify against eval_core.go/eval_specialforms.go.
Review package: review-3354403..e7e224f.diff

Review verdict: Needs fixes. define/defun deviation independently
verified correct (evalDefine requires ATOM name, live-tested). 2 Important
findings, both verified live against interpreter: (1) set! row claimed it
creates a binding if unbound — false, evalSet only calls env.Update, errors
if unbound; (2) setq* row said "wie setq" but setq* returns the symbol
name, setq returns the value — different return semantics. 2 Minor
(generator self-pollution *ref-out*/write-reference leak into generated
table; empty Beschreibung column) — deferred, acceptable as documented
limitations.
Task 5: fix round 1/5 dispatched (resume a20bb1dd747de17b8) — correct
set!/setq* rows in docs/ki/referenz.md, re-verify live.
Task 5: fix round 1/5 (2 addressed, 0 open; commits e7e224f..9cea6b2)
Task 5: complete (commits 3354403..9cea6b2, review clean after 1 fix round;
minor deferred: generator self-pollution *ref-out*/write-reference leak
into generated table; empty Beschreibung column in referenz-generiert.md
is expected/documented limitation)

Task 6: controller ran directly (no diff produced, per plan — "Dieser
Task verändert nichts"). go build/go test(378 pass)/build.sh/-t(104/0)
all green. Smoke test: (+ 1 2) -> 3, (env-symbols) non-empty. Residue
check clean: doc/ gone, src/lib present, unused/ present, pn-gps1/ at
root, git status clean.
Task 6: complete (no commits, all gates green)

Final whole-branch review dispatched (opus), range 77fe613..9cea6b2 (10
commits). Given: full task summary, both mid-execution rulings to verify,
5 deferred/parked minor findings to triage.

Final review verdict: Ready to merge WITH FIXES. Both rulings verified
against final tree (pn-gps1 at root with both consumers intact;
env-symbols in shipped binary, 256 symbols). 3 must-fix items:
1. (regression) src/lib/env_closure_test.go:235 same ../.. path bug as
   gps_bug_test.go had, but missed -- test silently SKIPs instead of
   running, disabling a real stack-overflow regression guard.
2. README.md/README_CN.md/BESCHREIBUNG.md build/run commands broken
   (need ./src prefix) -- living docs the plan's Global Constraints
   forgot to name.
3. docs/struktur.md still describes pre-move tree; docs/swank.md +
   docs/cli.md have 2 more stale bare main.go/lib mentions.
Plus 1 "fix now, it's cheap" item from the deferred list: generator
header said "nicht von Hand editieren" (opposite of design intent that
the empty column is meant to be hand-fillable).
Triage of the other 4 deferred/parked items: no action (2x "brief was
incomplete, executor did the right thing"), defer (mem.prof tracked
binary, generator self-pollution rows) -- all confirmed acceptable-as-is.
Also surfaced (not fixed, pre-existing/out of scope): stale lib/ in
stdlib.lisp/examples/tests comments, docs/ki/referenz_en.md+_cn.md still
carry the same wrong set!/let* claims Task 5 fixed only in German,
90 unchanged file-header path comments (leave per CLAUDE.md's own
no-retro-edit-headers rule), pre-existing CLAUDE.md nits (bare AUTHORS.md
ref, "drei Binaries" claim -- only two exist since golisp2d retired).
Dispatched ONE fix subagent (sonnet) covering all 4 fix-now items in one
commit, per skill (no per-finding fixers). No second fix wave after this
per skill rule -- residual findings surface to Gerhard via
finishing-a-development-branch.
