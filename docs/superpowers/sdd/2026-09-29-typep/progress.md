# SDD ledger — plan: docs/superpowers/plans/2026-09-29-typep.md
Spec: docs/superpowers/specs/2026-09-29-typep-design.md
Worktree: .worktrees/typep, branch typep, from main 4408106 (local HEAD, main ahead of origin by 12)

## Preflight scan
| Tasks | shared file/interface | produces vs consumes | finding |
|---|---|---|---|
| T1→T3,T4 | %cell-type | T1 symbols number/string/symbol/cons/null/lambda/func/macro/hash-table/t; T3 type-of checks 'number 'symbol 'lambda 'func 'cons; T4 satisfies checks '(lambda func) | consistent |
| T2→T3,T4,T5 | *struct-types* ((name n)) | T3 %struct-instance? uses (cadr entry); T4/T5 assoc | consistent |
| T2↔T5 | stdlib.lisp defstruct `(begin` | T5 replaces exactly T2's inserted lines | consistent |
| T2,T3,T4,T5 | types_typep_test.go | T2 creates w/ strings import + `var _ = strings.Contains`; T3/T4 append; T5 removes placeholder + uses strings | consistent |
| T3→T4,T5 | types.lisp helpers | %builtin-type?, %builtin-subtype?, %struct-instance?, type-of; T5 needs %builtin-type? | consistent |
| T3 | LoadStdlib | types loaded last (after loop) — spec says "nach condition.lisp"; last is also after condition | consistent |
| T6 | all | docs only | consistent |
| T1 self | test evalStr/evalEq exist (eval_test.go) | ok |
| T2 self | expects ((punkt 2)); no defstruct in embedded lisp | ok |
| T3 self | macro value via symbol eval verified live | ok |
| T4 self | all expectations verified in prototype tmp/typep-proto | ok |
| T5 self | captureStderr in redeflog_test.go; warn→WriteError→os.Stderr at call time | ok |
| T6 self | needs ./build.sh in worktree (build/ absent) | ok, step 3 builds |
Rubric check: T2 `var _ = strings.Contains` placeholder could be flagged. Ruling: keep — removed in T5, avoids unused-import compile error in T2–T4 — cost if wrong: one cosmetic line for 3 commits.
Ruling: implementers T1–T5 on haiku (plan contains complete code), T6 on sonnet (prose docs + judgment); reviewers sonnet; final review opus — cost if wrong: extra fix rounds.

## Progress
Task 1: dispatched (BASE 4408106, haiku)
Task 1: implementer aafb0de6688d71e22 DONE 1581d50; review dispatched (sonnet)
Task 1: Ruling: TestSwankSurvivesNorvigBugs failed only because fresh worktree had no build/golisp2 — ran ./build.sh, full suite 396/396 — cost if wrong: none (env only)
Task 1: parked — Important plan-mandated: src/lib/swank/env.go:116 swank--cell-type is a near-twin Cell.Type→name switch (strings, other vocab, no HASHTABLE) — Ruling: keep %cell-type as planned (types.lisp needs symbols + hash-table); consolidating swank--cell-type onto cellTypeName would change describe-symbol labels = Gerhard's decision per CLAUDE.md "Fragen, nicht anbauen", surface at finish — cost if wrong: one small follow-up refactor
Task 1: minor (deferred): SYMMACRO falls to default "t" without comment; default branch untested
Task 1: complete (commits 4408106..1581d50, review clean, 1 parked)
Task 2: dispatched (BASE 1581d50, haiku, implementer a696b8812f3d6652c)
Task 2: implementer DONE a5931c6
Task 2: minor (deferred): *struct-types* defvar sits before defstruct-resolve-name, not directly before defstruct comment (cosmetic)
Task 2: complete (commits 1581d50..a5931c6, review clean)
Task 3: dispatched (BASE a5931c6, haiku)
Task 3: implementer a384322df1dc276a2 DONE 944da4f
Task 3: controller check: reviewer line refs are diff-file lines; types.lisp verified identical to plan, smoke (integer keyword compiled-function p) ok, binary rebuilt
Task 3: complete (commits a5931c6..944da4f, review clean)
Task 4: dispatched (BASE 944da4f, haiku)
Task 4: implementer a6cdbe7c302c44cff DONE ef901a2
Task 4: complete (commits 944da4f..ef901a2, review clean)
Task 5: dispatched (BASE ef901a2, haiku)
Task 5: implementer a79552819449faf89 DONE a7f9d1e
Task 5: complete (commits ef901a2..a7f9d1e, review clean)
Task 6: dispatched (BASE a7f9d1e, sonnet)
Task 6: implementer ae4c1f9b46c1ae137 DONE b8ac122
Task 6: review: Important plan-mandated (lisp-semantik.md: Alist/Plist + "noch nicht vorhanden" listed under "Bewusste Abweichungen") — Ruling: fix, move both under own heading; spec separates Abweichungen vs Nicht-Ziele — cost if wrong: doc churn only
Task 6: Ruling: also fix Minor ki-referenz.md Gotchas row (struct + 3.0 conflated, CL column vague) in same round, because file is LLM system prompt — cost if wrong: one extra table row
Task 6: ⚠️ second type table: resolved by controller — only near-twin is swank--cell-type (parked Task 1), no second hierarchy
Task 6: fix round 1/5 dispatched fix da93d92, re-review running (haiku)
Task 6: fix round 1/5 (2 addressed, 0 open; commits b8ac122..da93d92)
Task 6: complete (commits a7f9d1e..da93d92, review clean)
Final review (opus): With fixes — Important: dotted-pair crash in type-of/typep; satisfies bound?/eval env mismatch. Minors 3–8, 10. Controller reproduced 1–4 live.
Final: Ruling: fix 1–8 + 10 in one wave (sonnet) — 3/4/7 are one-liners and 7 is LLM prompt — cost if wrong: small extra diff
Final: Ruling: #9 (lispbuch chapt-0011.md:221 "kein type-of") out of this repo — report to Gerhard, not fixed here
Final: triage: T1 parked swank--cell-type → can wait (Gerhard decision); T1/T2 deferred minors → can wait
Final: fix wave ff86d72 (sonnet, 417 go tests, 141 lisp); scoped re-review running
Final: re-review: all 8 addressed, no new breakage (live-checked trap scope: predicate errors still propagate)
Final: parked — trap around (eval f) swallows any error during symbol resolution (only global symbol-macro edge) — Ruling: acceptable, not a realistic pattern — cost if wrong: misleading error text in rare case
Final: complete (branch 4408106..ff86d72)
