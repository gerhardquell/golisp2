# GoLisp2 — AI Quick Reference (token-optimized)

> **Purpose:** initial context for AIs that write or read GoLisp2 code.
> **Translation** of `src/embed/ki-referenz.md` (German) — that file is the
> source of truth and is embedded via `go:embed` as the default preamble of
> every `(sigo …)` call. When in doubt, the German file wins.
> **Error messages** of the interpreter are German; they are quoted verbatim
> here, with an English gloss in parentheses.
> **Format:** tables, prefixes, no fluff. Hand-maintained, not generated;
> completeness list: `docs/referenz-generiert.md`.

---

## 1. Eval Order (always check)

```
1. ATOM   → env lookup (nil = singleton nil)
2. LIST   → check car:
   a. special form (case in eval_core.go)  → directly, no arg eval
   b. MACRO                               → expand, evaluate again
   c. FUNC/LAMBDA                         → eval args, apply
```

Tail calls (`if`, `begin`/`progn`/`locally`, `let`, `let*`, `cond`, `case`)
set `expr`/`env` and `continue` in the eval loop — **no new stack frame**.
Deep recursion in O(1) stack.

---

## 2. Special Forms (60 case branches, 62 keywords) + 2 stdlib macros (`dotimes`, `dolist`)

Counted directly from the `switch expr.Car.Val` in `eval_core.go` (as of
20261001) — `begin`/`progn`/`locally` share one case branch (3 names,
1 implementation), hence 60 branches but 62 names.

| Form | Semantics | Note |
|------|-----------|------|
| `(quote x)` | `x` unevaluated | `'x` reader sugar |
| `(if c t [e])` | conditional | tail-capable |
| `(begin . body)` | sequence | tail; aliases: `progn`, `locally`; multi-body via `wrapBegin` |
| `(let (bind) . body)` | parallel binding | tail-capable |
| `(let* (bind) . body)` | sequential binding | tail-capable; **native special form** (not stdlib — each binding sees the previous ones) |
| `(lambda (p) . body)` | closure | `&optional`, `&key`, `&rest` |
| `(defun f (p) . body)` | global function | multi-body via `wrapBegin` |
| `(defmacro m (p) . body)` | global macro | |
| `(defmain (args) . body)` | script entry point | only if the file is the main program (shebang/`golisp2 f.lisp`); runs after loading; return value = exit code 0–255; otherwise `nil`. `args` without binary/file name; only works at top level/inside special forms of the main file, not via `funcall`/`apply`/`mapcar` |
| `(define sym val)` | variable definition | global or local; `(define (f p) body)` = shorthand for `(define f (lambda (p) body))` |
| `(set! sym val)` | update one pair | only `env.Update` — **error if unbound**, creates nothing |
| `(setq v1 val1 v2 val2 ...)` | sequential assignment (CL) | several pairs; unbound → created **globally** + `WARN` on stderr (like SBCL; also applies to `setq*`, `setf`, `multiple-value-setq`) |
| `(setq* v1 val1 v2 val2 ...)` | sequential assignment, unbound → global + `WARN` | **returns:** the last **symbol** (not its value) — unlike `setq`, which returns the last **value** |
| `(psetq v1 val1 v2 val2 ...)` | parallel assignment | evaluate all values first, then assign |
| `(macrolet ((m . spec)) . body)` | local macro | non-recursive |
| `(symbol-macrolet ((s expansion)) . body)` | symbol macro | |
| `(flet ((f . spec)) . body)` | local function | non-recursive |
| `(labels ((f . spec)) . body)` | local function | recursive (mutually) |
| `(block name . body)` | named block | lexical |
| `(return-from name [val])` | non-local exit | |
| `(return [val])` | `(return-from nil [val])` | |
| `(tagbody stmt ...)` | jump tags | atoms = tags |
| `(go tag)` | jump | lexical, not evaluated |
| `(catch tag . body)` | dynamic catch | tag IS EVALUATED |
| `(throw tag val)` | dynamic throw | |
| `(trap expr handler)` | simple catch | `(trap expr (lambda (e) ...))`, e = message string |
| `(unwind-protect protected cleanup)` | cleanup always | |
| `(lock mutex . body)` | critical section | `mutex` from `lock-make`; body like `begin` |
| `(eval form)` | global eval | always `Env.Root()` |
| `(load "file")` | load file | **caution:** inside `defun` → bound locally! |
| `(prog1 first . rest)` | return first value | |
| `(prog2 a b . rest)` | return b | |
| `(cond (test result) ...)` | conditional | `else`/`t` = default |
| `(case key (vals result) ...)` | structural dispatch | `equal?` comparison |
| `(the type form)` | declaration | type **ignored** (no type system) |
| `(declare . decls)` | declaration | no-op |
| `(eval-when (situation) . body)` | situation control | |
| `(progv syms vals . body)` | dynamic binding | **caution:** no lexical/dynamic separation |
| `(and . exprs)` | short-circuit | |
| `(or . exprs)` | short-circuit | |
| `(not x)` | negation | |
| `(parfunc name [:timeout N] . exprs)` | fork-join parallel eval | `name` = symbol (not a combiner!), bound to the result list in the current env and returned; `:timeout` in seconds, branch over time → `()` |
| `(while test . body)` | loop | |
| `(do ((var step) ...) (test result) . body)` | Scheme iteration | parallel step |
| `(do* ((var step) ...) (test result) . body)` | Scheme iteration | sequential step |
| `(multiple-value-list form)` | values → list | |
| `(multiple-value-bind (vars) form . body)` | bind values | |
| `(multiple-value-call fn . forms)` | pass values on | |
| `(multiple-value-prog1 form . rest)` | keep first value | |
| `(multiple-value-setq vars form)` | assign values | |
| `(nth-value n form)` | n-th value | |
| `(function fn)` | function literal | `#'` reader sugar |
| `(macroexpand form)` | expand macro | non-tail |
| `(macroexpand-all form)` | expand fully | non-tail |
| `(bound? sym)` | bound? | sym is evaluated |
| `(makunbound sym)` | remove binding | |
| `(exec "prog" param: "arg" … stdout: var stderr: var exitcd: var)` | start a program (no shell!) | `stdout:`/`stderr:`/`exitcd:` are variable names; `timeout: N` s (default 60); exit ≠ 0 is no error |
| `(quasiquote x)` | quasi-quote | `` `x ``, `,x`=unquote, `,@x`=splice |
| `(unquote x)` | only inside quasiquote | error outside |
| `(unquote-splice x)` | only inside quasiquote | error outside |

**Tail positions** (set `expr`/`env`, `continue` in the loop): `if`,
`begin`/`progn`/`locally`, `let`, `let*`, `cond`, `case`. All other case
branches return immediately or delegate to an `eval*` helper (e.g.
`catch`, `unwind-protect`, `do`/`do*` — no TCO, but bounded by
`ectx.child()`).

---

## 3. Important Primitives (FUNC, Go side)

### Arithmetic/Comparison
`+ - * / mod rem remainder abs floor ceiling truncate round ash random values sqrt exp log expt` — `(log x)` natural only, x > 0; `expt` also negative/fractional (`(expt 0.9 -2)`), NaN/Inf → error; `mod` floored like CL (`(mod -3 2)` → 1), `rem`/`remainder` toward zero (→ -1)
`(floor x [d])` `ceiling` `truncate` `round` like CL: two values (quotient, remainder) — `(truncate 7 2)` → `3 1`; `round` to even (`(round 2.5)` → 2, `(round 3.5)` → 4); `(ash x n)` bit shift, n < 0 shifts right
`(parse-int "42")` `(parse-float "4.5")` — decimal only, no radix; invalid → error (not `nil`)
`= /= < > >= <= equal? eql eq eq?` — numeric comparisons as a CL chain: `(< 1 2 3)` → `t`, `(/= 1 2 3)` → `t` (all pairwise different); `eq`=pointer, `eql` like CL (numbers by value, strings never), `equal?`=structural (hash tables, vectors, channels by identity only, like CL `equal`)

### Lists (classic 7)
`car cdr cons atom null list append`
`atom? null? string? number? list? symbol?` — type predicates
`mapcar sort` — primitives (first-class: `funcall`/`apply` ok)

### Symbol/Atom
`gensym intern symbol-name symbol->string`

### Output
`print println read warn`

### Control/Errors
`error apply funcall exit` — `error` yields **only a string**, no condition object
`exit` — end the process immediately, code as number (no cleanup!)

### Environment/Introspection
`memstats sleep env-symbols lisp-implementation-version` — `(sleep ms)` in milliseconds; `(env-symbols)` returns all root env names
sorted (basis of `tools/gen-reference.lisp`); `(lisp-implementation-version)` → `"0.24.0"`

### Domains (own Register-Xxx)
- **sigoREST:** `sigo sigo* sigo-request sigo-models sigo-model-info sigo-costs sigo-budget sigo-host sigo-system-prompt sigo-reference sigo-usage sigo-usage-reset` — `(sigo "prompt" [model] [session-id] [host])` → text; `sigo*` → hash table (`text model finish-reason prompt-tokens completion-tokens cached-tokens reasoning-tokens cost-usd elapsed`); `(sigo-request h [host])`: hash with exactly `model messages temperature max_tokens session_id timeout retries system_prompt bare channel` (anything else → error; without `temperature` golisp2 sends 0.2, `0` is an error; without `max_tokens` the model maximum applies), result = full answer as hash plus `elapsed` (s); `(sigo-usage)` totals incl. `calls`/`elapsed`; `(sigo-costs [since [until]])` (Unix seconds like `(now)`), `(sigo-budget)`, `(sigo-model-info [model])` (prices USD/1M tokens, **0 = unknown, not free**); `(sigo-system-prompt "")` → model sees only the prompt (plus session history, if session-id)
- **Goroutines:** `chan-make chan-send chan-recv chan-try-recv chan-try-send chan-close chan-closed? chan-select spawn lock-make`
  - `(chan-recv ch [:timeout sec])` → `(values x t)`; closed+empty or timeout → `(values nil nil)` (`sec` = seconds, fractions ok). `(chan-try-recv ch)` like `chan-recv`, never blocking
  - `(chan-send ch x)` blocks; error on a closed channel. `(chan-try-send ch x)` → `t` / `nil` (buffer full); `(chan-close ch)` → `t`, second time error; `(chan-closed? ch)` → `t` only if closed **and** empty
  - `(chan-select chans [:timeout sec])` → `(values x t i)` (`i` = index in `chans`, from 0); timeout or all closed+empty → `(values nil nil nil)`
  - `(spawn fn)` → immediately a result channel with exactly one message `(:ok value)` or `(:error "text")`; errors also go to stderr. Counterpart to `parfunc` (which waits)
  - `(pool-map n fn list)` → like `mapcar`, but at most `n` calls at the same time (worker pool, e.g. for many `sigo` calls); results in input order, a failed element becomes `(:error "text")`, the others keep running
- **Shared memory:** `shm-alloc shm-free shm-write shm-read shm-status shm-cleanup`
- **File I/O:** `file-write file-append file-read file-exists? file-delete set-working-directory get-working-directory get-file-path directory-files gets slurp err-write printf sprintf fprintf sscanf argv getenv environ`
- **Shell:** `system shell-output file-stat shell-assoc` — `(system "cmd")` → exit code only; `(shell-output "cmd")` → stdout as string (like `$(…)`), exit ≠ 0 → error; `shell-assoc` is just `assoc`, no shell call
- **Time:** `now format-time get-universal-time sleep` — `(now)` → Unix seconds as float; `(format-time "%F %T" [time] [:utc])` strftime-like (`%Y %y %m %d %H %M %S %j %F %T %a %A %b %B %z %Z %s %N %3N %%`, English names); `get-universal-time` counts from 1900
- **Strings:** `string-length string-append substring string-upcase string-downcase string->number number->string string->list list->string string-replace string-trim string-contains string-find string-split string-join string= string/= string< string> string<= string>=` — comparisons character-wise, strings or symbols: `string=` → t/(), the others → index of the first mismatch or () (CL), `(sort l string<)`; `(string-split s [sep])`: with sep literally (empty fields stay), without sep on whitespace; `(string-join list sep)` sep required; argument order: `string-contains`/`string-split` haystack first, `string-find` needle first
- **Regex (Go RE2, no backreferences, pattern first):** `(regex-match re s)` → `t`/`()`; `(regex-find re s)` → first match or `()`, 2nd value list of groups; `(regex-find-all re s)` → list; `(regex-replace re s new)` replaces all, `$1`/`${name}` for groups (literal `$` as `$$`); `(regex-split re s)` → list. Double the backslash in strings: `"\\d+"`.
- **Hashtable:** `make-hash-table gethash puthash remhash clrhash hash-table-count hash-table-p maphash` — `puthash` is internal: `(puthash key table value)`, table in 2nd position (unlike Emacs Lisp); recommended `(setf (gethash k h) v)`. Same key → value is overwritten. `(gethash k h [default])` → value, 2nd value `t`/`()` = present. `:test` `'eql` (default) or `'equal`: numbers by value, symbols by name, **strings by content — even with `eql`** (CL: identity); lists structurally only with `'equal`; hash tables, vectors, lambdas as keys always by identity
- **Vectors (one-dimensional, O(1)):** `make-array vector aref aset vector-push-extend vector-pop vectorp arrayp` — `(make-array n [:initial-element x])`, `(aref v i)`, `(aset v i x)` or `(setf (aref v i) x)`, `(vector-push-extend x v)` → new index; `length`, `coerce` (`'list`/`'vector`) and `sort` (returns a new vector) know vectors; list functions (`mapcar`, `apply`, `append`, `dolist` …) raise an error on vectors — `(coerce v 'list)` first. Literal/print form `#(1 2 3)` (elements unevaluated, self-evaluating); `(loop for x across v …)`; `json-encode` → JSON array. Mutable and shared; `equal?` only on identity (CL `equal`).
- **Sequences (list or vector, not string):** `elt subseq copy-seq concatenate map some find find-if position position-if count count-if` — `(subseq s start [end])` and `copy-seq` keep the type; `(map 'vector f s…)`, `(concatenate 'list a b)` (type `'list`/`'vector`); `find`/`position`/`count` with `:test` (default `eql`, strings: `:test 'equal?`) and `:key`; non-destructive.
- **JSON:** `json-parse json-encode` — object ↔ hash table (`:test 'equal`, string keys), array ↔ list, `true` ↔ `t`, `null` ↔ `:null`, `false` → `()`, `()` → `[]`, `:false` → `false`. Alists are **not** objects (error when encoding). Also applies to `ws-call`/`ws-export`/`ws-emit`
- **FORMAT:** `format` — CL HyperSpec 22.3, `~A ~S ~D ~B ~O ~X ~R ~P ~C ~F ~E ~G ~$ ~% ~& ~| ~T ~* ~? ~[ ~{ ~( ~; ~^ ~/fun/ ~~`
  - rounding: half-to-even (Go `strconv`), not half-up like C — `%.2f` of `2.25` → `"2.2"`
- **PostgreSQL:** `pg-connect pg-query pg-exec pg-close`
- **Maxima (CAS, external process):** `(maxima-open)` → session; `(maxima-eval s "integrate(x^2,x)" [timeout-sec])` → result as string (state stays in the session); `(maxima-close s)` → `t`. Needs `maxima` installed
- **GenAlg:** `ga-create ga-init ga-cross ga-calc ga-select ga-result ga-mut ga-print ga?`
- **Redefine:** `redefine-policy redef-log redef-log-clear defined-in`
- **Trace:** `trace untrace trace?`
- **defsystem-lite:** `defsystem load-system unload-system loaded-systems system-symbols`
- **Web bridge (golisp2web):** `http-serve http-static http-stop http-port http-upload http-wait webserv ws-call ws-clients ws-emit ws-emit-to ws-eval ws-export ws-unexport browser-open`
- **Condition-lite:** `define-condition signal handler-case documentation` (see section 7)

Complete, generated list (not an excerpt): `docs/referenz-generiert.md`.

---

## 4. Stdlib (embed/stdlib.lisp, ~90 definitions)

### Accessors/Shortcuts
`cadr caddr cadddr cddr cdar caar first second third fourth rest`
`zero? positive? negative? pair?` — CL names: `zerop plusp minusp evenp oddp 1+ 1-` (`evenp`/`oddp` for integers only)

### Higher-Order/Functional
`identity constantly complement compose reverse length nth last member assoc filter drop take reduce for-each any every flatten zip list-tail iota shuffle randomlist max min square gcd` — `(shuffle l)` → new, randomly shuffled list (lists only, original unchanged); `(randomlist n)` = `(shuffle (iota n))`
Function argument: lambda, `#'f`, `f` or quoted symbol `'f` (resolved globally like CL) — `(mapcar 'string->number l)`, `(reduce '+ l)`, `(sort l '<)`.

### List helpers
`alist-set alist-get union set-difference find-all set-nth`
`remove remove-if remove-if-not remove-duplicates butlast copy-list copy-tree make-list getf` — lists only; `(remove x l)` compares with `equal?`, **no** `:test`/`:key` (→ error „zu viele Argumente“ (too many arguments)); `(make-list n :initial-element x)`; `(getf plist key [default])`

### Macros
`when unless let* dotimes dolist push pop incf decf defvar defparameter setf defstruct defgeneric defmethod ignore-errors assert destructuring-bind macroexpand-1`
- `(incf place [d])`/`(decf place [d])` via `setf`; `defparameter` always overwrites, `defvar` only if unbound
- `(ignore-errors . body)` → value or `()` on error (one value only, no condition object; `throw`/`return-from` pass through)
- `(assert form)` → `t` or error `assert failed: form` — one argument only, no custom message
- `(destructuring-bind (a b c) list . body)` — **flat only**: nested patterns, dotted pairs and `&rest`/`&key`/`&optional` → error. For the rest: `(destructuring-bind (a b) l …)` plus `(cddr l)`
- `macroexpand-1` = `macroexpand` (both exactly one step)

**Caution:** `let*` appears here only as a CL habit anchor — the actual
implementation is the native special form from section 2
(`src/embed/stdlib.lisp` deliberately does **not** define it).

### Iterators
`dotimes (var n) body` — `(dotimes (i 10) ...)`
`dolist (var lst) body` — `(dolist (x xs) ...)`

### Structures
`(defstruct name [doc] slot|(slot default) ...)` — generates: `(make-name :slot v ...)`, `name-slot`, `name?` (+ `name-p`)
`(setf place val)` — generic, `(defstruct ...)` registers accessors automatically

### Types
- `(type-of x)` → `integer` `float` `string` `symbol` `keyword` `boolean` `null` `cons` `function` `compiled-function` `macro` `hash-table` `vector`, struct, class (defclass) or condition name; classes are a subtype of `hash-table`. `(typep x spec)` with type names, `(or …)` `(and …)` `(not …)` `(member …)` `(eql x)` `(satisfies f)` `(integer lo hi)` (`(n)` = exclusive, `*` = open). `3.0` is `integer`.
  Also valid as type names (not returned by `type-of`, but `typep` supertypes): `rational` `real` `number` `atom` `list` `sequence` `structure-object` `t`. Not available → `typep: unbekannter Typ '…'` (unknown type): `fixnum` `character` `double-float` `single-float`. Type names are case-sensitive — `'INTEGER` is unknown, only `'integer`. `(typecase x (integer …) ((or string symbol) …) (otherwise …))` branches by type, `etypecase` raises an error without a match. `(check-type x (integer 0 *) ["description"])` → `()` or error. Both raise a plain `error`, no `type-error`. `(deftype digit () '(integer 0 9))` defines type names, with parameters `(deftype upto (&optional (n '*)) (list 'integer 0 n))` → `(typep x '(upto 5))`; names of built-in types, structs, classes, conditions → error. `(subtypep 'integer 'number)` → two values `t t`; undecidable (`not`, `satisfies`, subranges) → `() ()`.

---

## 5. Truth Values / Nil

- `()` / `nil` / `NIL` → singleton nil (pointer identity!)
- `t` → true, but *not* the only true value — everything except nil is true
- `(eq '() '())` → `t` (singleton)
- `(eq 5 5)` → `()` — **design:** `eq` on numbers always returns `()` (see 10.6). When in doubt, `equal?`

---

## 6. Quasiquote Patterns

```lisp
`(a b c)           ; plain quote
`(a ,x c)          ; unquote
`(a ,@xs c)        ; unquote-splice
```

---

## 7. Error Handling

```lisp
; raise an error
(error "message")             ; aborts, yields only a string

; catch
(trap expr (lambda (e) ...))  ; e = "msg" (message string)

; dynamic
(catch 'tag body ...)
(throw 'tag value)
```

**Condition-lite** (`embed/condition.lisp`, loaded automatically) —
structured errors with a type hierarchy and slots:

```lisp
(define-condition file-error (io-error) (path))  ; type + parents + slots
(signal 'file-error :path "x.lisp")              ; raises, always unwinds!
(handler-case (load "x.lisp")
  (file-error (e) (file-error-path e))  ; reader generated: type-slot
  (io-error  (e) "some io error")       ; also matches subtypes
  (error     ()  "without var binding")) ; var may be ()
```

- Base hierarchy: `condition` → `error` → `lisp-error`
- **Go errors** (file-read etc.) become `lisp-error` in `handler-case`,
  message via `(lisp-error-msg e)`
- No match → **re-signal** to the outer handler
- Redefining a type replaces it silently (reload semantics)
- **CL deviation:** `signal` always unwinds (behaves like CL's `error`,
  not like CL's `signal`). No restarts, no `handler-bind`.
- Slot names must be unique across the inheritance hierarchy
  (flat plist, no shadowing).

**Test framework:** `tests/test-framework.lisp` — `defsuite`, `deftest`
(`:suite`, `:expected-failure`), `is`, `run-tests` → number of FAILs.
Typical: `(exit (run-tests))` → exit code = FAILs.

---

## 8. Eval Environment

- `(eval form)` → **always `Env.Root()`**, never the lambda scope
- `load` → **EXCEPTION:** inside a `defun` body → binding local, not global
  - workaround: `(eval '(load "file"))` for global loading
- `redefine-policy` → `'allow` / `'warn` / `'error` (default `warn`)

---

## 9. Gotchas / CL Deviations

| Case | GoLisp2 | CL |
|------|---------|-----|
| `(eq 5 5)` | `()` — `eq` on numbers always `()` (design) | often `t` (small int) |
| `load` in `defun` | bound locally | global |
| `progv` | no lexical/dynamic separation | dynamic |
| `declare` | no-op | type checks |
| `the` | type ignored | type checks |
| `(eval form)` | global | global (ok) |
| `macrolet` | non-recursive | recursive |
| `(define (f p) ...)` | shorthand for `(define f (lambda (p) ...))` (Scheme style) | not available — `defun` |
| structs are lists | `(typep p 'cons)` → `t` | own type, not `cons` |
| integral floats are integers | `(type-of 3.0)` → `integer` | `single-float`/`double-float` (3.0 is a float) |
| string keys in `(make-hash-table)` (default `eql`) | by content: `"rot"` twice = one key | `eql` → identity, two `"rot"` literals = two keys; use `:test 'equal` for that |
| `(puthash k h v)` | table in 2nd position; prefer `(setf (gethash k h) v)` | Emacs Lisp: `(puthash k v h)` |
| `stringp` `consp` `equal` | missing → error „unbekanntes Symbol“ (unknown symbol); use `string?`, `(typep x 'cons)`, `equal?` | available |
| `destructuring-bind` with `&rest`/`&key`, nested, dotted pair | error — flat patterns only | fully supported |
| `(member x l)` | no `:test`; compares strings structurally | `eql` (strings only with `:test 'equal`) |
| `(last l)` | returns the **element**, not the cell | last cons cell |
| `(slurp)` | reads **stdin**; file → `(file-read path)` | – |
| `(gets)` at end of input | error `gets: EOF`, not `nil` | `read-line` → `nil`/EOF value |
| `(sleep ms)` vs. `:timeout sec` | `sleep` in **milliseconds**, `:timeout` in **seconds** — `(sleep 0.05)` has no effect | `sleep` in seconds |
| `(parfunc r :timeout N …)` | `N` number literal only (not evaluated), number ≥ 0, fractions ok; `:timeout 0` = **no** timeout (with `chan-recv`/`chan-select`, however: return immediately) | – |

---

## 10. Weaknesses (deliberate, own section)

These limitations are **design decisions**, not bugs. An AI must know them
so it does not guess:

### 10.1 No package system
- All symbols live in one global namespace.
- No `defpackage`, `in-package`, `export`, `import`.
- Avoid collisions only by naming convention (prefixes like `ga-`, `shm-`).

### 10.2 No full CLOS
- `defstruct` (constructor, accessors, predicate).
- CLOS-light: `defgeneric`/`defmethod` — single dispatch on struct tag or defclass class, `t` = default method.
- `(defclass name (slot (slot default) ...))` — objects with identity (hash table, `:class`); `make-name`, `name?`, `name-slot`; `setf` changes in place; prints as `#<name>`. **Not** CL syntax: no superclass list, no `:initarg`.
- No inheritance, no `call-next-method`, no method combination, no `call-method`.

### 10.3 Only condition-lite, no full CL condition system
- `define-condition`/`signal`/`handler-case` available (hierarchy, slots,
  inheritance dispatch, `lisp-error` fallback for Go errors).
- **But:** `signal` always unwinds (like CL's `error`), no restarts,
  no `handler-bind`, no `restart-case`, no MOP.
- Slot names flat — no shadowing across inheritance.

### 10.4 No lexical/dynamic separation in progv
- `progv` binds like `let` — lexical shadowings see the progv value.
- CL: a lexical binding protects against progv.

### 10.5 No compile-file
- Pure interpreter. No `compile-file`, no loading of FASLs.
- No `eval-when` setup for compiler/compile time.

### 10.6 `eq` on numbers always returns `()`
- `(eq 5 5)` → `()`. `(eq 1000 1000)` → `()`. Even with identical value.
- Internally there is a small-int cache (-32768..32767, `MakeNum` in `types.go`)
  to avoid allocations — `eq` still deliberately treats numbers as never
  identical (`fnEqPtr` in `primitives.go`).
- Always use `equal?` or `=` to compare numbers, never `eq`.

### 10.7 Macros non-recursive (macrolet)
- `macrolet` bodies do not see the other macros of the same level.
- `labels` for functions is recursive — asymmetry compared to CL.

### 10.8 No continuations, no MOP
- No `call/cc`.
- `catch`/`throw` available (section 2), but without restart semantics.
- No CLOS meta-object protocol.

### 10.9 `load` in `defun` binds locally
- Workaround: `(eval '(load "file"))` for global loading.
- Reason: `eval` runs in `Env.Root()`.

### 10.10 No fine-grained GC control
- `memstats` returns Go runtime stats.
- No `tweak`, `make-hash-table` without weak refs.

### 10.11 Lean type system
- `declare` and `the` are no-ops.
- `type-of`, `typep`, `typecase`, `etypecase`, `check-type`, `deftype`,
  `subtypep` available (see Types). No `ctypecase`, `ccase`.

### 10.12 LOOP only as a practical core, no Series, no Iterate
- `loop`: `for x in`/`from … to`/`below`/`by`, `repeat`, `while`/`until`,
  `when`/`unless`, `collect`/`sum`/`count`/`maximize`/`minimize`, `do`,
  `finally`, `return`, `for x across vector`. Not available: destructuring, `named`.
- No `series`, no `iterate`.

### 10.13 `define` with function sugar
- `(define (f p) body)` is shorthand for `(define f (lambda (p) body))`.

---

## 11. File Structure (AI scan)

```
src/
  lib/
    types*.go          Cell data structure
    reader.go          Read / ReadAll
    env.go             Env (RWMutex, pool)
    eval_core.go       eval trampoline + special form dispatch
    eval_*.go          special forms, lambda, control, quasiquote
    primitives.go      BaseEnv() — all Go primitives
    *.go               goroutine, fileio, shellcmd, postgres, genalg, shm, sigorest
    stdlib.go          //go:embed stdlib.lisp
    format*.go         CL HyperSpec 22.3
    trace.go           trace/untrace
  embed/               //go:embed assets (stdlib.lisp, condition.lisp, ...)
  main.go              CLI
```

---

## 12. Quick Lookup (AI cheatsheet)

| I need... | Use |
|-----------|-----|
| arithmetic | `+ - * / mod` |
| comparison | `equal?` (numbers!), `eq` (pointer) |
| build a list | `cons list append` |
| iteration | `dolist dotimes loop mapcar reduce` |
| parallel | `parfunc` |
| errors | `error` + `trap` |
| dynamic | `catch/throw` |
| structure | `defstruct` |
| string | `string-append substring string-replace` |
| file | `file-read file-write` |
| DB | `pg-connect pg-query` |
| AI | `sigo` |
| web bridge | `http-serve webserv ws-emit` |
| debug | `trace` |
| all symbols | `(env-symbols)` or `docs/referenz-generiert.md` |

---

**End of AI reference.** Human version: `docs/golisp2-cheatsheet.md` (German).
Complete generated function reference: `docs/referenz-generiert.md`.
Source of truth (German): `src/embed/ki-referenz.md` · 中文: `docs/ki/referenz_cn.md`.
