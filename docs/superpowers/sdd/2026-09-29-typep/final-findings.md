# Final review findings — fix wave (all in one dispatch)

Spec rule that binds: "type-of wirft nie"; typep errors only with the spec texts
"typep: unbekannter Typ '…'", "typep: ungültige Typangabe …", "typep: satisfies: '…' ist keine Funktion".

## Important
1. src/embed/types.lisp `%struct-instance?`: `length` on improper list crashes.
   `(defstruct punkt x y) (type-of (cons 'punkt 2))` → `ERR: cdr: Liste erwartet`; also `(typep (cons 'punkt 2) 'cons)`.
   Fix: a small proper-length check (e.g. helper `%proper-length` returning -1 for improper lists, or trap around length). Expected after fix: type-of → cons, typep 'cons → t.
2. src/embed/types.lisp `%typep-satisfies`: `(bound? f)` sees the helper's own locals (x, spec, args, f) while `(eval f)` looks in root env.
   `(typep 1 '(satisfies f))` → `ERR: env: unbekanntes Symbol 'f'` instead of spec text.
   Fix: resolve via root env only, e.g. `(fn (trap (eval f) (lambda (e) ())))`. Expected: `typep: satisfies: 'f' ist keine Funktion` for f, x, spec, args.

## Minor (fix in same wave)
3. `%number-type`: +Inf/-Inf → integer. Require finite (e.g. NaN/Inf check: Inf satisfies (= x (* 2 x)) for x≠0 is one way; choose a clear one). Expected `(type-of (* 1e308 10))` → float; NaN stays float.
4. `%symbol-type`: `(type-of (intern ""))` → substring error. Check name length first → symbol.
5. `%cons-type`: malformed condition markers: `(type-of '(%condition))` → `()`, `(type-of (cons '%condition 2))` → car error. Guard with `(pair? (cdr x))` and `(symbol? (cadr x))`; otherwise fall through to cons. Note %cond? lives in condition.lisp — do not change it; guard in types.lisp.
6. Dotted type specs `(or . x)`, `(member . 1)`, `(integer 0 . 5)` → raw car/cdr error. Should be `typep: ungültige Typangabe …` (validate args is a proper list before dispatch in %typep-spec).
7. src/embed/ki-referenz.md (LLM system prompt!): row label "Kein Float-Typ" is wrong ((type-of 3.5) → float exists) → relabel e.g. "Ganzzahlige Floats sind integer". In the stdlib entry for type-of/typep add: accepted names also `rational real number atom list sequence structure-object t`; `fixnum character vector double-float single-float` → "unbekannter Typ"; `check-type`, `subtypep`, `typecase`, `etypecase`, `deftype` fehlen; typnamen sind case-sensitiv (`'INTEGER` unbekannt).
8. Tests (src/lib/types_typep_test.go): regression tests for 1–6, plus missing supertype rows: macro vs 'macro, struct vs 'list and 'sequence, condition vs 'list.

## Controller notes
- Keep changes minimal, in types.lisp only for code; do not touch condition.lisp/stdlib.lisp semantics.
- After fixing: rebuild (./build.sh), go test ./... -count=1, ./build/golisp2 -t, and verify every ki-referenz claim live.
