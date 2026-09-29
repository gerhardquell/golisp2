# Implementer instructions (common to every task of this plan)

- Work ONLY in /u/lisp-projekte/golisp2/.worktrees/typep (git worktree, branch `typep`). Never cd to /u/lisp-projekte/golisp2 itself.
- Read your task brief first — it is your requirements, with exact code and values to use verbatim. Also read global-constraints.md in the same directory.
- Go commands from the worktree root, e.g. `go test ./src/lib/ -run 'TestX' -count=1`. Files in src/embed/ are go:embed'ed — `go test` picks up edits automatically.
- A built ./build/golisp2 exists (SWANK tests need it). If you change embedded .lisp files, the binary is stale — that is fine for `go test`; do not rebuild unless the brief says so.
- Placeholders `<dein eigenes Modell>` in file headers and commit trailers: replace with YOUR OWN exact model name. Never copy a model name from the plan.
- New files: 2 spaces, no tabs. Existing files: match surrounding indentation. Never run gofmt.
- Temp files only under ./tmp/ in the worktree, never /tmp.
- TDD exactly per brief: write test, run it and see it fail for the expected reason, implement, see it pass. Run the full suite `go test ./... -count=1` once before committing.
- Commit, self-review your own diff (completeness, YAGNI, tests verify behavior, output pristine), fix issues before reporting.
- Never spawn subagents or reviewers. Review is the controller's job.
- If unclear or stuck: stop and report NEEDS_CONTEXT or BLOCKED with specifics. Bad work is worse than no work.
- If the brief's code does not behave as the brief expects, do NOT improvise a redesign: report DONE_WITH_CONCERNS or BLOCKED with the exact failing output.

## Report
Write the full report to the report file named in your dispatch:
- what you implemented; files changed
- TDD evidence: RED (command + relevant failing output, why expected) and GREEN (command + passing output)
- full-suite result
- self-review findings, concerns

Then reply with ONLY (under 15 lines): Status (DONE | DONE_WITH_CONCERNS | BLOCKED | NEEDS_CONTEXT), commits (short SHA + subject), one-line test summary, concerns, report file path.

If resumed with review findings: fix, re-run the covering tests, append a fix report (changes, tests, command, output) to the same report file, reply with the same short contract.
