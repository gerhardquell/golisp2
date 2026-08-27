# Task 2 Report: Doc-Merge — doc/ + docs/ → docs/

**Status:** DONE  
**Commit SHA:** e7e083f  
**Commit Message:** `docs(repo): doc/ und docs/ zu docs/ zusammengefuehrt`

## What Was Done

Implemented all steps from task-2-brief.md exactly:

### Step 1: File Movement
- Moved 10 documentation files from `doc/` to `docs/`:
  - AUTHORS.md, cl-inventar.md, cli.md, emacs-golisp2web.md, golisp2-cheatsheet.md
  - lisp-semantik.md, memory.md, sigo.md, struktur.md, swank.md
- Moved `doc/ki/` directory to `docs/ki/` (3 files: referenz.md, referenz_cn.md, referenz_en.md)
- Removed empty `doc/` directory

### Step 2: CLAUDE.md — Struktur-Verweis
- Line 38: Changed `doc/struktur.md` → `docs/struktur.md`

### Step 3: CLAUDE.md — Doku-Tabelle
- Updated 7 table entries (lines 249-255) to use `docs/` instead of `doc/`

### Step 4: tests/conformance/README.md
- Line 54: Changed `doc/cl-inventar.md` → `docs/cl-inventar.md`

### Additional Fixes (for verification)
Updated internal cross-references in newly moved documentation files to maintain consistency:

**CLAUDE.md:**
- Line 11: Changed `Alles andere: `doc/` (siehe unten)` → `docs/`
- Line 117: Changed `doc/lisp-semantik.md` → `docs/lisp-semantik.md`

**docs/cli.md:**
- Line 88: Changed `doc/swank.md` → `docs/swank.md`

**docs/emacs-golisp2web.md:**
- Line 9: Changed `doc/swank.md` → `docs/swank.md`

**docs/memory.md:**
- Line 28: Changed `doc/lisp-semantik.md` → `docs/lisp-semantik.md`

**docs/cl-inventar.md:**
- Line 184: Changed `doc/` → `docs/` (forward reference)

**docs/golisp2-cheatsheet.md:**
- Lines 4, 6, 823: Updated 3 cross-references to `docs/ki/referenz.md` and `docs/lisp-semantik.md`

**docs/ki/referenz.md:**
- Lines 6, 348, 349: Updated 3 cross-references to docs/ files

**docs/ki/referenz_en.md:**
- Lines 6, 348, 349: Updated 3 cross-references to docs/ files

**docs/ki/referenz_cn.md:**
- Lines 6, 346, 347: Updated 3 cross-references to docs/ files

## Verification Results

### doc/ Directory Removed
```
doc/ entfernt: OK
```

### Grep Verification
Ran: `grep -rn '`doc/\|"doc/\|(doc/' --include=*.md . | grep -v "^./docs/retrospectives/" | grep -v "^./docs/gespraeche/" | grep -v "^./docs/superpowers/plans/" | grep -v "^./docs/superpowers/specs/"`

Result: ✓ CLEAN  
Only remaining `doc/` references are in task specification files themselves:
- `.superpowers/sdd/2026-08-27-repo-restructuring/task-1-brief.md` (planning document)
- `.superpowers/sdd/2026-08-27-repo-restructuring/task-1-report.md` (completed report)
- `.superpowers/sdd/2026-08-27-repo-restructuring/task-2-brief.md` (this task's spec, showing before/after)

These are correctly excluded from the active codebase verification.

### Build Verification
```
go build ./...
```
✓ PASSED (no output = success)

## Files Changed Summary

**Moved (14 files):**
- `doc/AUTHORS.md` → `docs/AUTHORS.md`
- `doc/cl-inventar.md` → `docs/cl-inventar.md`
- `doc/cli.md` → `docs/cli.md`
- `doc/emacs-golisp2web.md` → `docs/emacs-golisp2web.md`
- `doc/golisp2-cheatsheet.md` → `docs/golisp2-cheatsheet.md`
- `doc/lisp-semantik.md` → `docs/lisp-semantik.md`
- `doc/memory.md` → `docs/memory.md`
- `doc/sigo.md` → `docs/sigo.md`
- `doc/struktur.md` → `docs/struktur.md`
- `doc/swank.md` → `docs/swank.md`
- `doc/ki/referenz.md` → `docs/ki/referenz.md`
- `doc/ki/referenz_cn.md` → `docs/ki/referenz_cn.md`
- `doc/ki/referenz_en.md` → `docs/ki/referenz_en.md`

**Modified (5 files):**
- `CLAUDE.md` — 4 edits (intro quote, structure ref, table entries, internal ref)
- `tests/conformance/README.md` — 1 edit (line 54)
- `docs/cli.md` — 1 edit
- `docs/emacs-golisp2web.md` — 1 edit
- `docs/memory.md` — 1 edit
- `docs/cl-inventar.md` — 1 edit
- `docs/golisp2-cheatsheet.md` — 3 edits
- `docs/ki/referenz.md` — 3 edits
- `docs/ki/referenz_en.md` — 3 edits
- `docs/ki/referenz_cn.md` — 3 edits

**Total changes:** 15 files affected, 27 insertions(+), 27 deletions(-)

## Self-Review Checklist

- [x] `doc/` directory removed (verified: `test ! -d doc`)
- [x] All 10 main documentation files moved to `docs/`
- [x] `doc/ki/` directory moved to `docs/ki/`
- [x] CLAUDE.md structure reference fixed (line 38)
- [x] CLAUDE.md documentation table updated (7 entries)
- [x] tests/conformance/README.md line 54 fixed
- [x] Internal cross-references in docs/ files updated for consistency
- [x] Verification grep returns only task specification files (expected)
- [x] `go build ./...` passes
- [x] Commit created with correct message and co-author
- [x] No changes made to historical directories (docs/gespraeche/, docs/superpowers/plans/, docs/superpowers/specs/)
- [x] No Go files modified (this task correctly only touches documentation)

## Notes

This task completed **beyond the original brief scope** by also updating internal cross-references within the newly moved `docs/` files. This was necessary to ensure:

1. The verification grep returned clean results (no unexpected `doc/` references outside historical documents)
2. All documentation remains consistent and functional (cross-references point to correct locations)
3. Future maintainers won't encounter broken internal references

The brief specified modifications only to CLAUDE.md and tests/conformance/README.md, but those files had references to documentation that was being moved. Updating the moved documentation files themselves ensures internal consistency and a correct verification result.
