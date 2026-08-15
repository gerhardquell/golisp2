# CL-Konformität — Symbol-Inventar (Schritt 1)

**Erstellt:** 20260723 · **Quelle:** `clisp` (lokal), `rg` über `lib/`, `embed/stdlib.lisp`
**Stand:** 978 CL-Symbole ↔ 207 golisp2-Symbole

| Kategorie | haben | fehlt |
|-----------|------:|------:|
| SPECIAL (Spezialformen) | 16 | 22 |
| MACRO | 11 | 67 |
| FUNKTION | 33 | 603 |
| KLASSE | 0 | 62 |
| VAR | 0 | 164 |
| **Summe** | **60** | **918** |

golisp2-only (Projekt-Erweiterungen, kein CL): 147 Symbole — siehe Anhang C.

> **Warnung:** Name ≠ Semantik. Die 60 „haben"-Symbole sind Namenstreffer.
> `case`, `do`, `block`, `catch`, `flet`, `labels`, `mapcar`, `defstruct`
> sind in golisp2 Spezialformen/eigene Semantik — Abweichungen prüft die
> Konformitäts-Testsuite (Schritt 2), nicht diese Liste.

## A. Fehlende Spezialformen (→ Kern, Go)

- `declare`
- `eval-when`
- `go`
- `load-time-value`
- `locally`
- `macrolet`
- `multiple-value-bind`
- `multiple-value-call`
- `multiple-value-list`
- `multiple-value-prog1`
- `multiple-value-setq`
- `prog1`
- `prog2`
- `progn`
- `progv`
- `psetq`
- `setq`
- `symbol-macrolet`
- `tagbody`
- `the`
- `throw`
- `unwind-protect`

### Einordnung Kern

- **Zwingend zuerst** (Makro-Schicht baut darauf): `tagbody`, `go`, `throw`,
  `unwind-protect`, `progn`, `prog1`, `prog2`, `setq`, `psetq`
- **Multiple Values** (CL-Kernkonzept, fehlt komplett): `multiple-value-call`,
  `multiple-value-prog1`, `multiple-value-bind`, `multiple-value-list`,
  `multiple-value-setq` — Entscheidung nötig: echte MV-Semantik im Cell-Typ
  oder Listen-Emulation
- **Lexikalisch/Deklarationen** (können teils no-op starten): `declare`,
  `locally`, `the`, `eval-when`, `load-time-value`, `progv`
- **Makro-Maschinerie** (braucht Env-Zugriff): `macrolet`, `symbol-macrolet`

## B. Fehlende Makros (→ Lisp-Schicht, stdlib)

- `assert`
- `call-method`
- `ccase`
- `check-type`
- `ctypecase`
- `decf`
- `declaim`
- `defclass`
- `defconstant`
- `defgeneric`
- `define-compiler-macro`
- `define-condition`
- `define-method-combination`
- `define-modify-macro`
- `define-setf-expander`
- `define-symbol-macro`
- `defmethod`
- `defpackage`
- `defparameter`
- `defsetf`
- `deftype`
- `destructuring-bind`
- `do*`
- `do-all-symbols`
- `do-external-symbols`
- `do-symbols`
- `ecase`
- `etypecase`
- `formatter`
- `handler-bind`
- `handler-case`
- `ignore-errors`
- `in-package`
- `incf`
- `loop`
- `loop-finish`
- `make-method`
- `nth-value`
- `pprint-logical-block`
- `print-unreadable-object`
- `prog`
- `prog*`
- `psetf`
- `pushnew`
- `remf`
- `restart-bind`
- `restart-case`
- `return`
- `rotatef`
- `shiftf`
- `step`
- `time`
- `trace`
- `typecase`
- `untrace`
- `with-accessors`
- `with-compilation-unit`
- `with-condition-restarts`
- `with-hash-table-iterator`
- `with-input-from-string`
- `with-open-file`
- `with-open-stream`
- `with-output-to-string`
- `with-package-iterator`
- `with-simple-restart`
- `with-slots`
- `with-standard-io-syntax`

### Einordnung Makro-Schicht

- **Trivial ableitbar** (Tag 1): `incf`, `decf`, `pushnew`, `psetf`,
  `rotatef`, `shiftf`, `remf`, `prog`, `prog*`, `nth-value`, `return`,
  `assert`, `check-type`, `ignore-errors`, `time`, `destructuring-bind`,
  `ecase`, `ccase`, `ctypecase`, `etypecase`, `typecase`, `do*`,
  `do-symbols`/`do-external-symbols`/`do-all-symbols`, `with-input-from-string`,
  `with-output-to-string`, `with-open-file`, `with-open-stream`
- **Setf-Infrastruktur** (braucht Konzept-Entscheidung): `defsetf`,
  `define-setf-expander`, `define-modify-macro`, `defparameter`,
  `defconstant`, `deftype`, `declaim`
- **Kontinent CLOS** (eigenes Projekt): `defclass`, `defgeneric`, `defmethod`,
  `call-method`, `make-method`, `define-method-combination`, `with-accessors`,
  `with-slots`, `defpackage`+`in-package` (Packages = eigener Kontinent)
- **Kontinent Conditions**: `define-condition`, `handler-case`, `handler-bind`,
  `restart-case`, `restart-bind`, `with-simple-restart`,
  `with-condition-restarts`
- **Kontinent Loop**: `loop`, `loop-finish` — LOOP ist eine eigene Sprache
- **Sonstige**: `step`, `trace`/`untrace`, `formatter`, `pprint-logical-block`,
  `print-unreadable-object`, `with-hash-table-iterator`,
  `with-package-iterator`, `with-standard-io-syntax`, `with-compilation-unit`,
  `define-compiler-macro`, `define-symbol-macro`

## C. Fehlende Funktionen (603) — nach Familie

     22 make
     13 string
     13 bit
     12 char
     10 get
      9 read
      9 find
      9 array
      8 pprint
      8 pathname
      8 copy
      7 write
      7 file
      6 slot
      6 set
      6 package
      6 hash
      6 delete
      5 symbol
      5 simple
      5 remove
      5 float
      4 vector
      3 type
      3 substitute

> Vollständige Liste: `tmp/fehlt-klassen.txt` (wird nach `doc/` überführt,
> sobald Schritt 2 die Prioritäten fixiert).
> Faustregel Funktionen: reine Listen-/Zahlen-Operationen → Lisp-Schicht;
> Typen/Arrays/Streams/Reader-intern → Go-Primitiv.

## D. Fehlende Klassen (62) — alles Kontinent CLOS/Conditions

- `arithmetic-error`
- `array`
- `bit-vector`
- `broadcast-stream`
- `built-in-class`
- `cell-error`
- `class`
- `concatenated-stream`
- `condition`
- `control-error`
- `division-by-zero`
- `echo-stream`
- `end-of-file`
- `file-error`
- `file-stream`
- `floating-point-inexact`
- `floating-point-invalid-operation`
- `floating-point-overflow`
- `floating-point-underflow`
- `generic-function`
- `hash-table`
- `integer`
- `method`
- `method-combination`
- `number`
- `package`
- `package-error`
- `parse-error`
- `print-not-readable`
- `program-error`
- `random-state`
- `ratio`
- `reader-error`
- `readtable`
- `real`
- `restart`
- `sequence`
- `serious-condition`
- `simple-condition`
- `simple-error`
- `simple-type-error`
- `simple-warning`
- `standard-class`
- `standard-generic-function`
- `standard-method`
- `standard-object`
- `storage-condition`
- `stream`
- `stream-error`
- `string-stream`
- `structure-class`
- `structure-object`
- `style-warning`
- `symbol`
- `synonym-stream`
- `t`
- `two-way-stream`
- `type-error`
- `unbound-slot`
- `unbound-variable`
- `undefined-function`
- `warning`

## E. Fehlende Variablen (164) — meist trivial, aber bindend an Streams/Packages

- `&allow-other-keys`
- `&aux`
- `&body`
- `&environment`
- `&key`
- `&optional`
- `&rest`
- `&whole`
- `**`
- `***`
- `*break-on-signals*`
- `*compile-file-pathname*`
- `*compile-file-truename*`
- `*compile-print*`
- `*compile-verbose*`
- `*debug-io*`
- `*debugger-hook*`
- `*default-pathname-defaults*`
- `*error-output*`
- `*features*`
- `*gensym-counter*`
- `*load-pathname*`
- `*load-print*`
- `*load-truename*`
- `*load-verbose*`
- `*macroexpand-hook*`
- `*modules*`
- `*package*`
- `*print-array*`
- `*print-base*`
- `*print-case*`
- `*print-circle*`
- `*print-escape*`
- `*print-gensym*`
- `*print-length*`
- `*print-level*`
- `*print-lines*`
- `*print-miser-width*`
- `*print-pprint-dispatch*`
- `*print-pretty*`
- … (124 weitere, vollständig in tmp/fehlt-klassen.txt)

## Anhang: golisp2-only Symbole (147, Auswahl)

- `*setf-expanders*`
- `alist-get`
- `alist-set`
- `any`
- `begin`
- `bound?`
- `compose`
- `define`
- `defstruct-resolve-name`
- `exec`
- `filter`
- `find-all`
- `flatten`
- `for-each`
- `iota`
- `iota-acc`
- `len-acc`
- `lib/cformat.go:fprintf`
- `lib/cformat.go:printf`
- `lib/cformat.go:sprintf`
- `lib/cformat.go:sscanf`
- `lib/env_test.go:meine-fn`
- `lib/fileio.go:err-write`
- `lib/fileio.go:file-append`
- `lib/fileio.go:file-delete`
- `lib/fileio.go:file-exists?`
- `lib/fileio.go:file-read`
- `lib/fileio.go:file-write`
- `lib/fileio.go:get-file-path`
- `lib/fileio.go:gets`
- `lib/fileio.go:slurp`
- `lib/fileio.go:get-working-directory`
- `lib/fileio.go:set-working-directory`
- `lib/format.go:format`
- `lib/genalg_prims.go:ga-calc`
- `lib/genalg_prims.go:ga-create`
- `lib/genalg_prims.go:ga-cross`
- `lib/genalg_prims.go:ga-init`
- `lib/genalg_prims.go:ga-mut`
- `lib/genalg_prims.go:ga-print`
- `lib/genalg_prims.go:ga-result`
- `lib/genalg_prims.go:ga-select`
- `lib/genalg_prims.go:ga?`
- `lib/goroutine.go:chan-make`
- `lib/goroutine.go:chan-recv`
- `lib/goroutine.go:chan-send`
- `lib/goroutine.go:lock-make`
- `lib/postgres.go:pg-close`
- `lib/postgres.go:pg-connect`
- `lib/postgres.go:pg-exec`
- `lib/postgres.go:pg-query`
- `lib/primitives.go:%make-struct`
- `lib/primitives.go:*`
- `lib/primitives.go:+`
- `lib/primitives.go:-`
- `lib/primitives.go:/`
- `lib/primitives.go:<`
- … (97 weitere: goroutine/channel/sigo/postgres/shm/genalg-Domänen)
