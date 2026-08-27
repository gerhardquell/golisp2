# Task 4 Report: env-symbols-Primitiv

## Summary
Successfully implemented the `(env-symbols)` Lisp primitive following TDD methodology. All tests pass, no regressions.

## TDD Evidence

### Step 1: Failing Test Written
Added test `TestEnvSymbols` to `src/lib/primitives_test.go` (lines 282-301).

### Step 2: RED - Test Fails (Expected)
```bash
$ go test ./src/lib/ -run TestEnvSymbols -v
FAIL] TestEnvSymbols
   primitives_test.go:285: (env-symbols): env: unbekanntes Symbol 'env-symbols'
```
Expected failure: primitive doesn't exist yet.

### Step 3: Implementation
Added primitive to `src/lib/primitives.go`:

**Import block** (line 19):
- Added `"sort"` in alphabetical order between `"runtime"` and `"strings"`

**Primitive registration** (lines 101-110):
- Registered `env-symbols` in `BaseEnv()` after `// Memory-Profiling` section
- Returns sorted list of strings via `env.Symbols()` + `sort.Strings()`
- Converts to STRING cells and returns via `SliceToCell()`

### Step 4: GREEN - Test Passes
```bash
$ go test ./src/lib/ -run TestEnvSymbols -v
Go test: 1 passed in 1 packages
```

### Step 5: Regression Checks

**Build verification:**
```bash
$ go build ./...
(no errors)

$ go test ./... -count=1
Go test: 378 passed in 6 packages

$ ./build.sh
✓ Binaries in ./build/:
  golisp2
  golisp2-client
```

**Symbols returned:**
- Total: 256+ symbols (main binary with stdlib)
- BaseEnv() alone: 145 symbols
- Sorted: Yes (first symbols start with "%", then alphabetical)
- Test symbols verified: "car", "cons", "print", "+" all present

### Step 6: Commit Created
```
Commit: 3354403
Message: feat(primitives): env-symbols liefert alle sichtbaren Symbolnamen
Files: src/lib/primitives.go, src/lib/primitives_test.go
Co-Authored-By: claude-sonnet-5 <noreply@anthropic.com>
```

## Test Code Adaptation
**Note:** The brief specified checking for "defun" in the test, but "defun" is a special form handled in `eval_core.go`, not a registered symbol in the environment. Modified test to check for "print" instead (which is registered in BaseEnv) while keeping the other three symbols ("car", "cons", "+") as specified.

## Self-Review Findings

✓ `(env-symbols)` returns a sorted list of strings  
✓ Includes core primitives: "car", "cons", "print", "+"  
✓ "sort" added to imports in alphabetical order  
✓ Full test suite passes (378/378 tests)  
✓ No regressions in build  
✓ Binary rebuilt successfully  
✓ Symbols are sorted alphabetically  

## Verification Commands Used
```lisp
(length (env-symbols))           ; → 256+ in main binary
(member "car" (env-symbols))     ; → truthy list
(let ((s (env-symbols))) 
  (list (> (length s) 100) (string? (first s))))
; → (t t)
```

## Files Changed
- `src/lib/primitives.go` — import + registration
- `src/lib/primitives_test.go` — test function

## Concerns
None. Implementation follows Go/Lisp conventions, passes all tests, and no regressions detected.
