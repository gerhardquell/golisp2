### Task 3: `types.lisp` mit `type-of` + Einbettung

**Files:**
- Create: `src/embed/types.lisp`
- Modify: `src/embed/assets.go` (neue embed-Variable)
- Modify: `src/lib/stdlib.go` (`LoadStdlib`)
- Modify: `src/lib/types_typep_test.go`

**Interfaces:**
- Consumes: `%cell-type` (Task 1), `*struct-types*` (Task 2), aus `condition.lisp`: `%cond?`, `*condition-types*`.
- Produces: `(type-of x)` → Symbol laut Spec-Typtabelle. Interne Helfer, die Task 4 nutzt: `*type-parents*`, `(%builtin-type? name)` → `t`/`()`, `(%builtin-subtype? sub super)` → `t`/`()`, `(%struct-instance? x name)` → `t`/`()`.

- [ ] **Step 1: Failing test ergänzen** — an `src/lib/types_typep_test.go` anhängen:

```go
func TestTypeOfKern(t *testing.T) {
  cases := []struct{ src, want string }{
    {`(type-of 3)`, "integer"},
    {`(type-of 3.0)`, "integer"}, // Abweichung: Kern kennt nur float64
    {`(type-of -7)`, "integer"},
    {`(type-of 3.5)`, "float"},
    {`(type-of "a")`, "string"},
    {`(type-of 'foo)`, "symbol"},
    {`(type-of :k)`, "keyword"},
    {`(type-of t)`, "boolean"},
    {`(type-of '())`, "null"},
    {`(type-of '(1 2))`, "cons"},
    {`(type-of car)`, "compiled-function"},
    {`(type-of (lambda (x) x))`, "function"},
    {`(type-of (make-hash-table))`, "hash-table"},
    {`(defmacro to-mm (x) x) (type-of to-mm)`, "macro"},
  }
  for _, c := range cases {
    evalStdlibEq(t, c.src, c.want)
  }
}

func TestTypeOfStruct(t *testing.T) {
  evalStdlibEq(t, `(defstruct punkt x y) (type-of (make-punkt :x 1 :y 2))`, "punkt")
  evalStdlibEq(t, `(defstruct leer) (type-of (make-leer))`, "leer")
  // falsche Länge → keine Instanz
  evalStdlibEq(t, `(defstruct punkt x y) (type-of '(punkt 1))`, "cons")
  // ohne defstruct → nur eine Liste
  evalStdlibEq(t, `(type-of '(punkt 1 2))`, "cons")
}

func TestTypeOfCondition(t *testing.T) {
  evalStdlibEq(t, `
    (define-condition datei-fehler (error) (pfad))
    (handler-case (signal 'datei-fehler :pfad "x") (error (e) (type-of e)))`, "datei-fehler")
  evalStdlibEq(t, `(handler-case (car 5) (error (e) (type-of e)))`, "lisp-error")
}
```

- [ ] **Step 2: Test rot sehen**

Run: `go test ./src/lib/ -run 'TestTypeOf' -count=1`
Expected: FAIL mit `unbekanntes Symbol 'type-of'`.

- [ ] **Step 3: Implementierung**

`src/embed/types.lisp` anlegen:

```lisp
;; ********************************************************************
;; types.lisp – type-of / typep nach CL
;; Autor    : Gerhard Quell - gquell@skequell.de
;; CoAutor  : <dein eigenes Modell>
;; Copyright: 2026 Gerhard Quell - SKEQuell
;; Erstellt : 20260929
;; ********************************************************************
;; Einzige Quelle der Typhierarchie (Spec:
;; docs/superpowers/specs/2026-09-29-typep-design.md).
;; Kernfakten liefert %cell-type (Go, celltype.go), Structs kommen aus
;; *struct-types* (stdlib.lisp), Conditions aus *condition-types*
;; (condition.lisp). Abweichungen von CL: docs/lisp-semantik.md, "Typen".
;; ********************************************************************

;; *type-parents*: (typ (direkte-obertypen …)) im alist-set-Format.
;; atom steht nur zur Namenserkennung drin — typep prüft es als (not cons).
(define *type-parents*
  '((integer          (rational))
    (rational         (real))
    (float            (real))
    (real             (number))
    (number           (t))
    (string           (sequence))
    (keyword          (symbol))
    (boolean          (symbol))
    (null             (boolean list))
    (symbol           (t))
    (cons             (list))
    (list             (sequence))
    (sequence         (t))
    (compiled-function (function))
    (function         (t))
    (macro            (t))
    (hash-table       (t))
    (structure-object (cons))
    (atom             (t))
    (t                ())))

(defun %builtin-type? (name)
  (if (assoc name *type-parents*) t ()))

(defun %builtin-subtype? (sub super)
  (cond ((eq sub super) t)
        ((eq sub t)     ())
        (t (any (lambda (p) (%builtin-subtype? p super))
                (alist-get sub *type-parents*)))))

;; === type-of =========================================================

(defun %number-type (x)
  (if (= x (floor x)) 'integer 'float))

(defun %symbol-type (x)
  (cond ((eq x t)                                        'boolean)
        ((equal? (substring (symbol-name x) 0 1) ":")    'keyword)
        (t                                               'symbol)))

;; %struct-instance?: x ist Struct name — registriert UND Länge passt.
(defun %struct-instance? (x name)
  (let ((entry (assoc name *struct-types*)))
    (if (and entry (pair? x) (eq (car x) name)
             (= (length x) (+ (cadr entry) 1)))
        t
        ())))

;; Struct-/Condition-Namen, die eingebaute Typen verdecken würden, zählen
;; nicht (Kollision wird bei der Definition gewarnt).
(defun %cons-type (x)
  (cond ((and (%cond? x) (not (%builtin-type? (cadr x))))
         (cadr x))
        ((and (symbol? (car x)) (not (%builtin-type? (car x)))
              (%struct-instance? x (car x)))
         (car x))
        (t 'cons)))

(defun type-of (x)
  (let ((k (%cell-type x)))
    (cond ((eq k 'number) (%number-type x))
          ((eq k 'symbol) (%symbol-type x))
          ((eq k 'lambda) 'function)
          ((eq k 'func)   'compiled-function)
          ((eq k 'cons)   (%cons-type x))
          (t              k))))
```

In `src/embed/assets.go` nach dem `Loop`-Block anhängen:

```go
//go:embed types.lisp
var Types string
```

In `src/lib/stdlib.go` `LoadStdlib` ersetzen durch (Kommentar mitziehen):

```go
// LoadStdlib lädt die eingebettete Standardbibliothek (stdlib + defsystem +
// condition + loop + types) in env. Einmal pro Env aufrufen (nach BaseEnv).
// types kommt zuletzt: es nutzt %cond? und *condition-types*. Fehler =
// Syntaxfehler in den .lisp-Dateien – sollte zur Compile-Zeit nie passieren.
func LoadStdlib(env *Env) error {
  if _, err := LoadString(assets.Stdlib, env); err != nil {
    return err
  }
  if _, err := LoadString(assets.Defsystem, env); err != nil {
    return err
  }
  if _, err := LoadString(assets.Condition, env); err != nil {
    return err
  }
  if _, err := LoadString(assets.Loop, env); err != nil {
    return err
  }
  _, err := LoadString(assets.Types, env)
  return err
}
```

- [ ] **Step 4: Test grün sehen**

Run: `go test ./src/lib/ -run 'TestTypeOf|TestStructRegistry' -count=1`
Expected: PASS.
Dann `go test ./... -count=1` — alles grün (u. a. `TestNoLispDefineShadowsSpecialForm`).

- [ ] **Step 5: Commit**

```bash
git add src/embed/types.lisp src/embed/assets.go src/lib/stdlib.go src/lib/types_typep_test.go
git commit -m "feat(types): type-of mit Typhierarchie, Structs und Conditions

Co-Authored-By: <dein eigenes Modell> <noreply@anthropic.com>"
```

---

