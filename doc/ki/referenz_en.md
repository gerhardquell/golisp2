# GoLisp2 — AI Quick Reference (token-optimized)

> **Purpose:** Give other AIs this file as *initial context* to write/understand
> GoLisp2 code without running `rg` across 50 files.
> **Format:** tables, prefixes, no fluff. Human companion:
> `doc/golisp2-cheatsheet.md` (German).
> **Source:** `eval_core.go`, `lib/primitives.go`, `embed/stdlib.lisp` (as of 20260730).

---

## 1. Eval Order (always check)

```
1. ATOM   → Env lookup (nil = singleton nil)
2. LIST   → check car:
   a. Special form (case in eval_core.go)  → direct, no arg eval
   b. MACRO                               → expand, re-evaluate
   c. FUNC/LAMBDA                         → eval args, apply
```

Tail calls (`if`, `begin`, `let`, `lambda`, `case`, `cond`, `prog1/2`, `catch`,
`throw`, `tagbody/go`, ...) set `expr`/`env` and `continue` in the eval loop —
**no new stack frame**. Deep recursion O(1) stack.

---

## 2. Special Forms (55) + 2 stdlib macros (`dotimes`, `dolist`)

| Form | Semantics | Note |
|------|-----------|------|
| `(quote x)` | `x` unevaluated | `'x` reader sugar |
| `(if c t [e])` | Conditional | Tail-capable |
| `(begin . body)` | Sequence | Tail; multi-body via `wrapBegin` |
| `(let (bind) . body)` | Parallel binding | `let*` sequential (stdlib) |
| `(lambda (p) . body)` | Closure | `&optional`, `&key`, `&rest` |
| `(defun f (p) . body)` | Global function | Multi-body via `wrapBegin` |
| `(defmacro m (p) . body)` | Global macro | |
| `(define sym val)` | Var definition | Global or local |
| `(set! sym val)` | Update | `setq` alias |
| `(setq* s1 v1 s2 v2 ...)` | Sequential set | |
| `(psetq s1 v1 s2 v2 ...)` | Parallel set | |
| `(macrolet ((m . spec)) . body)` | Local macro | Non-recursive |
| `(symbol-macrolet ((s expansion) . body)` | Symbol macro | |
| `(flet ((f . spec)) . body)` | Local function | Non-recursive |
| `(labels ((f . spec)) . body)` | Local function | Recursive (mutual) |
| `(block name . body)` | Named block | Lexical |
| `(return-from name [val])` | Non-local exit | |
| `(return [val])` | `(return-from nil [val])` | |
| `(tagbody stmt ...)` | Jump tags | Atoms = tags |
| `(go tag)` | Jump | Lexical, not evaluated |
| `(catch tag . body)` | Dynamic catch | Tag is EVALUATED |
| `(throw tag val)` | Dynamic throw | |
| `(trap expr handler)` | Simple catch | `(trap expr (lambda (e) ...))`, yields `(error . msg)` |
| `(unwind-protect protected cleanup)` | Cleanup always | |
| `(eval form)` | Global eval | Always `Env.Root()` |
| `(load "file")` | Load file | **Caution:** inside `defun` → locally bound! |
| `(progn . body)` | Sequence | Tail |
| `(prog1 first . rest)` | Return first value | |
| `(prog2 a b . rest)` | Return b | |
| `(cond (test result) ...)` | Conditional | `else`/`t` = default |
| `(case key (vals result) ...)` | Structural dispatch | `equal?` comparison |
| `(the type form)` | Declaration | Type **ignored** (no type system) |
| `(declare . decls)` | Declaration | No-op |
| `(eval-when (situation) . body)` | Situation control | |
| `(progv syms vals . body)` | Dynamic binding | **Caution:** no lex/dyn separation |
| `(and . exprs)` | Short-circuit | |
| `(or . exprs)` | Short-circuit | |
| `(not x)` | Negation | |
| `(parfunc expr . opts)` | Parallel eval | `:timeout N`, `:workers N` |
| `(while test . body)` | Loop | |
| `(do ((var step) ...) (test result) . body)` | Scheme iteration | Parallel step |
| `(do* ((var step) ...) (test result) . body)` | Scheme iteration | Sequential step |
| `(dotimes (var n) . body)` | Counting loop | Stdlib macro |
| `(dolist (var lst) . body)` | List iteration | Stdlib macro |
| `(multiple-value-list form)` | Values→list | |
| `(multiple-value-bind (vars) form . body)` | Bind values | |
| `(multiple-value-call fn . forms)` | Pass values | |
| `(multiple-value-prog1 form . rest)` | Keep first value | |
| `(multiple-value-setq vars form)` | Set values | |
| `(nth-value n form)` | n-th value | |
| `(function fn)` | Function literal | `#'` reader sugar |
| `(macroexpand form)` | Expand macro | |
| `(macroexpand-all form)` | Expand fully | |
| `(bound? sym)` | Bound? | |
| `(makunbound sym)` | Remove binding | |
| `(exec shell-cmd)` | Shell command | |
| `(quasiquote x)` | Quasi-quote | `` `x ``, `,x`=unquote, `,@x`=splice |

**Eval keywords (tail position):** `if`, `begin`, `let`, `cond`, `case`,
`prog1`, `prog2`, `catch`, `throw`, `tagbody/go`, `block/return-from`, `do`.

---

## 3. Important Primitives (FUNC, Go side)

### Arithmetic/Comparison
`+ - * / mod remainder abs floor random values`
`= < > >= <= equal? eq eq?`  — `eq`=pointer, `equal?`=structural

### Lists (classic 7)
`car cdr cons atom null list append`
`atom? null? string? number? list? symbol?` — type predicates
`mapcar` — primitive (first-class: `funcall`/`apply` ok)

### Symbol/Atom
`gensym intern symbol-name symbol->string`

### Output
`print println read warn`

### Control/Errors
`error apply funcall`  — `error` yields **string only**, no condition object

### Environment/Introspection
`memstats sleep`

### Domains (own Register-Xxx)
- **sigoREST:** `sigo sigo-models sigo-host`
- **Goroutines:** `chan-make chan-send chan-recv lock-make`
- **Shared memory:** `shm-alloc shm-free shm-write shm-read shm-status shm-cleanup`
- **File I/O:** `file-write file-append file-read file-exists? file-delete set-working-directories get-working-directories get-file-path`
- **Shell:** `system file-stat shell-assoc`
- **Strings:** `string-length string-append substring string-upcase string-downcase string->number number->string string->list list->string string-replace string-trim string-contains`
- **Hashtable:** `make-hash-table gethash puthash remhash clrhash hash-table-count hash-table-p maphash`
- **FORMAT:** `format` — CL HyperSpec 22.3, `~A ~S ~D ~B ~O ~X ~R ~P ~C ~F ~E ~G ~$ ~% ~& ~| ~T ~* ~? ~[ ~{ ~( ~; ~^ ~/fun/ ~~`
- **PostgreSQL:** `pg-connect pg-query pg-exec pg-close`
- **GenAlg:** `ga-create ga-init ga-cross ga-calc ga-select ga-result ga-mut ga-print ga?`
- **Redefine:** `redefine-policy redef-log redef-log-clear defined-in`
- **Trace:** `trace untrace trace?`

---

## 4. Stdlib (embed/stdlib.lisp, ~50 definitions)

### Accessors/Shortcuts
`cadr caddr cadddr cddr cdar caar first second third fourth rest`
`zero? positive? negative? pair?`

### Higher-Order/Functional
`identity constantly complement compose reverse length nth last member assoc filter drop take reduce for-each any every flatten zip list-tail iota max min square expt gcd`

### List helpers
`alist-set alist-get union set-difference find-all set-nth`

### Macros
`when unless let* dotimes dolist push pop defvar setf defstruct`

### Iterators
`dotimes (var n) body` — `(dotimes (i 10) ...)`
`dolist (var lst) body` — `(dolist (x xs) ...)`

### Structures
`(defstruct name (slot default) ...)` — generates: `make-name`, `name-slot`, `name?`
`(setf place val)` — generic; `(defstruct ...)` registers accessors automatically

---

## 5. Truth Values / Nil

- `()` / `nil` / `NIL` → singleton nil (pointer identity!)
- `t` → true, but *not* the only truthy value — everything except nil is true
- `(eq '() '())` → `t` (singleton)
- `(eq 5 5)` → `()` — **by design:** `eq` on numbers always yields `()` (see 10.6). When in doubt use `equal?`

---

## 6. Quasiquote Patterns

```lisp
`(a b c)           ; pure quote
`(a ,x c)          ; unquote
`(a ,@xs c)        ; unquote-splice
```

---

## 7. Error Handling (today)

```lisp
; Raising
(error "message")             ; aborts, yields string only

; Catching
(trap expr (lambda (e) ...))  ; e = (error . "msg")

; Dynamic
(catch 'tag body ...)
(throw 'tag value)
```

**No condition system** — no hierarchy, no slots, no restarts.

---

## 8. Eval Environment

- `(eval form)` → **always `Env.Root()`**, never lambda scope
- `load` → **EXCEPTION:** inside `defun` body → binding is local, not global
  - Workaround: `(eval '(load "file"))` for global loading
- `redefine-policy` → `'allow` / `'warn` / `'error` (default `warn`)

---

## 9. Gotchas / CL Deviations

| Case | GoLisp2 | CL |
|------|---------|-----|
| `(eq 5 5)` | `()` — `eq` on numbers always `()` (by design) | often `t` (small-int) |
| `load` in `defun` | Locally bound | Global |
| `progv` | No lex/dyn separation | Dynamic |
| `declare` | No-op | Type checks |
| `the` | Type ignored | Type checks |
| `(eval form)` | Global | Global (ok) |
| `macrolet` | Non-recursive | Recursive |

---

## 10. Weaknesses (deliberate, own section)

These limitations are **design decisions**, not bugs. An AI must know them
to avoid guessing:

### 10.1 No Package System
- All symbols in one global namespace.
- No `defpackage`, `in-package`, `export`, `import`.
- Avoid collisions by naming convention only (prefixes like `ga-`, `shm-`).

### 10.2 No CLOS
- Only `defstruct` (constructor, accessors, predicate).
- No classes, no multi-methods, no method combination.
- `defmethod`, `defgeneric`, `defclass`, `call-method` — **not present**.

### 10.3 No Condition System
- `error` yields a string only, no object with slots.
- No `define-condition`, `handler-case`, `handler-bind`, `restart-case`.
- No programmatic distinction between "file not found" and "parse error".
- Recovery only via `trap`/`catch`/`throw` with strings.

### 10.4 No Lex/Dyn Separation in progv
- `progv` binds like `let` — lexical shadowings see the progv value.
- CL: lexical binding protects against progv.

### 10.5 No Compile-File
- Pure interpreter. No `compile-file`, no loading FASLs.
- No `eval-when` setup for compiler/compile time.

### 10.6 `eq` on numbers always yields `()`
- `(eq 5 5)` → `()`. `(eq 1000 1000)` → `()`. Even for identical values.
- Internally a small-int cache exists (-32768..32767, `MakeNum` in
  `lib/types.go`) to avoid allocations — but `eq` deliberately treats numbers
  as never identical (`fnEqPtr` in `lib/primitives.go`).
- Always use `equal?` or `=` for number comparison, never `eq`.

### 10.7 Macros Non-Recursive (macrolet)
- `macrolet` bodies do not see the other macros of the same level.
- `labels` for functions is recursive — asymmetry vs. CL.

### 10.8 No Continuations, No MOP
- No `call/cc`.
- `catch`/`throw` exist (section 2), but without restart semantics.
- No CLOS Meta-Object Protocol.

### 10.9 `load` in `defun` binds locally
- Workaround: `(eval '(load "file"))` for global loading.
- Rationale: `eval` runs in `Env.Root()`.

### 10.10 No GC Fine-Tuning
- `memstats` yields Go runtime stats.
- No `tweak`, `make-hash-table` without weak refs.

### 10.11 No Type System
- `declare` and `the` are no-ops.
- No `check-type`, `typecase`, `ctypecase`, `etypecase`.

### 10.12 No LOOP, No Series, No Iterate
- Iteration only via `dolist`, `dotimes`, `do`, `do*`, `mapcar`, `reduce`.
- No `loop` macro, no `series`, no `iterate`.

---

## 11. File Structure (AI scan)

```
lib/
  types*.go          Cell data structure
  reader.go          Read / ReadAll
  env.go             Env (RWMutex, Pool)
  eval_core.go       Eval trampoline + special-form dispatch
  eval_*.go          Special forms, lambda, control, quasiquote
  primitives.go      BaseEnv() — all Go primitives
  *.go               goroutine, fileio, shellcmd, postgres, genalg, shm, sigorest
  stdlib.go          //go:embed stdlib.lisp
  format*.go         CL HyperSpec 22.3
  trace.go           trace/untrace
main.go              CLI
```

---

## 12. Quick Lookup (AI cheatsheet)

| I need... | Use |
|-----------|-----|
| Arithmetic | `+ - * / mod` |
| Comparison | `equal?` (numbers!), `eq` (pointer) |
| Build list | `cons list append` |
| Iteration | `dolist dotimes mapcar reduce` |
| Parallel | `parfunc` |
| Errors | `error` + `trap` |
| Dynamic | `catch/throw` |
| Structure | `defstruct` |
| String | `string-append substring string-replace` |
| File | `file-read file-write` |
| DB | `pg-connect pg-query` |
| AI | `sigo` |
| Debug | `trace` |

---

**End of AI reference.** Human version (German): `doc/golisp2-cheatsheet.md`.
German version: `doc/ki/referenz.md`. Chinese version: `doc/ki/referenz_cn.md`.
