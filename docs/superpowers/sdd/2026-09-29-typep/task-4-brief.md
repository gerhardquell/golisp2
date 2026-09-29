### Task 4: `typep` — Namen, Kombinatoren, Bereiche, Fehler

**Files:**
- Modify: `src/embed/types.lisp` (am Ende anhängen)
- Modify: `src/lib/types_typep_test.go`

**Interfaces:**
- Consumes: aus Task 3 `type-of`, `*type-parents*`, `%builtin-type?`, `%builtin-subtype?`, `%struct-instance?`; aus `condition.lisp` `%cond?`, `%cond-type?`, `*condition-types*`.
- Produces: `(typep x spec)` → `t` oder `()`; Fehler laut Spec. Hilfsfunktion für Task 5: keine.

- [ ] **Step 1: Failing tests ergänzen** — an `src/lib/types_typep_test.go` anhängen:

```go
func TestTypepHierarchie(t *testing.T) {
  ja := []string{
    `(typep 3 'integer)`, `(typep 3 'rational)`, `(typep 3 'real)`, `(typep 3 'number)`, `(typep 3 'atom)`, `(typep 3 t)`,
    `(typep 3.5 'float)`, `(typep 3.5 'real)`,
    `(typep "a" 'string)`, `(typep "a" 'sequence)`, `(typep "a" 'atom)`,
    `(typep 'foo 'symbol)`, `(typep :k 'keyword)`, `(typep :k 'symbol)`,
    `(typep t 'boolean)`, `(typep t 'symbol)`,
    `(typep '() 'null)`, `(typep '() 'boolean)`, `(typep '() 'symbol)`, `(typep '() 'list)`, `(typep '() 'sequence)`, `(typep '() 'atom)`,
    `(typep '(1) 'cons)`, `(typep '(1) 'list)`, `(typep '(1) 'sequence)`,
    `(typep car 'compiled-function)`, `(typep car 'function)`, `(typep (lambda (x) x) 'function)`,
    `(typep (make-hash-table) 'hash-table)`,
  }
  for _, src := range ja {
    evalStdlibEq(t, src, "t")
  }
  nein := []string{
    `(typep 3 'float)`, `(typep 3.5 'integer)`, `(typep "a" 'list)`, `(typep 'foo 'keyword)`,
    `(typep t 'null)`, `(typep '() 'cons)`, `(typep '(1) 'atom)`, `(typep (lambda (x) x) 'compiled-function)`,
    `(typep (make-hash-table) 'list)`, `(typep 3 nil)`,
  }
  for _, src := range nein {
    evalStdlibEq(t, src, "()")
  }
}

func TestTypepStruct(t *testing.T) {
  evalStdlibEq(t, `(defstruct punkt x y) (typep (make-punkt :x 1 :y 2) 'punkt)`, "t")
  evalStdlibEq(t, `(defstruct punkt x y) (typep (make-punkt :x 1 :y 2) 'structure-object)`, "t")
  evalStdlibEq(t, `(defstruct punkt x y) (typep (make-punkt :x 1 :y 2) 'cons)`, "t")
  evalStdlibEq(t, `(defstruct punkt x y) (typep '(punkt 1) 'punkt)`, "()")
  evalStdlibEq(t, `(typep '(1 2) 'structure-object)`, "()")
}

func TestTypepCondition(t *testing.T) {
  pre := `(define-condition datei-fehler (error) (pfad)) `
  for _, typ := range []string{"datei-fehler", "error", "condition", "cons"} {
    evalStdlibEq(t, pre+`(handler-case (signal 'datei-fehler :pfad "x") (error (e) (typep e '`+typ+`)))`, "t")
  }
  evalStdlibEq(t, pre+`(handler-case (signal 'datei-fehler :pfad "x") (error (e) (typep e 'lisp-error)))`, "()")
  evalStdlibEq(t, `(typep '(1) 'error)`, "()")
}

func TestTypepKombinatoren(t *testing.T) {
  evalStdlibEq(t, `(typep 'a '(or string symbol))`, "t")
  evalStdlibEq(t, `(typep 3 '(or string symbol))`, "()")
  evalStdlibEq(t, `(typep 3 '(and number (not float)))`, "t")
  evalStdlibEq(t, `(typep 3 '(or))`, "()")
  evalStdlibEq(t, `(typep 3 '(and))`, "t")
  evalStdlibEq(t, `(typep 3 '(not string))`, "t")
  evalStdlibEq(t, `(typep 'b '(member a b))`, "t")
  evalStdlibEq(t, `(typep 'c '(member a b))`, "()")
  evalStdlibEq(t, `(typep 2 '(eql 2))`, "t")
  evalStdlibEq(t, `(typep 3 '(satisfies number?))`, "t")
  evalStdlibEq(t, `(typep "a" '(satisfies number?))`, "()")
  evalStdlibEq(t, `(defun gerade? (n) (= 0 (mod n 2))) (typep 4 '(and integer (satisfies gerade?)))`, "t")
  evalStdlibEq(t, `(typep '((a 1)) '(or null (and cons (satisfies list?))))`, "t")
}

func TestTypepBereiche(t *testing.T) {
  cases := []struct{ src, want string }{
    {`(typep 5 '(integer 0 10))`, "t"},
    {`(typep 11 '(integer 0 10))`, "()"},
    {`(typep 0 '(integer (0) 10))`, "()"},
    {`(typep 10 '(integer 0 (10)))`, "()"},
    {`(typep 99 '(integer 0 *))`, "t"},
    {`(typep -1 '(integer 0))`, "()"},
    {`(typep 5 '(integer))`, "t"},
    {`(typep 5.5 '(integer 0 10))`, "()"},
    {`(typep 5.5 '(real 0 10))`, "t"},
    {`(typep 5.5 '(float 5 6))`, "t"},
    {`(typep "a" '(integer 0 10))`, "()"},
  }
  for _, c := range cases {
    evalStdlibEq(t, c.src, c.want)
  }
}

func TestTypepFehler(t *testing.T) {
  for _, src := range []string{
    `(typep 1 'gibts-nicht)`,
    `(typep 1 '(not a b))`,
    `(typep 1 '(eql))`,
    `(typep "a" '(integer "x"))`,
    `(typep 1 '(integer 0 1 2))`,
    `(typep 1 '(foo 1))`,
    `(typep 1 '(satisfies gibts-nicht))`,
    `(typep 1 '(satisfies))`,
    `(typep 1 "string")`,
  } {
    evalStdlibErr(t, src)
  }
  // Fehlertexte laut Spec
  evalStdlibEq(t, `(trap (typep 1 'foo) (lambda (e) e))`, `"typep: unbekannter Typ 'foo'"`)
  evalStdlibEq(t, `(trap (typep 1 '(satisfies foo)) (lambda (e) e))`, `"typep: satisfies: 'foo' ist keine Funktion"`)
}
```

(`trap` liefert den Fehlertext als String; `String()` druckt ihn mit Anführungszeichen — geprüft 20260929.)

- [ ] **Step 2: Tests rot sehen**

Run: `go test ./src/lib/ -run 'TestTypep' -count=1`
Expected: FAIL mit `unbekanntes Symbol 'typep'`.

- [ ] **Step 3: Implementierung** — an `src/embed/types.lisp` anhängen:

```lisp
;; === typep ===========================================================

(defun %bad-spec (spec)
  (error (format nil "typep: ungültige Typangabe ~a" spec)))

;; %type-start: Einstieg in *type-parents* für x. Structs starten bei
;; structure-object, Conditions bei cons.
(defun %type-start (x)
  (let ((k (type-of x)))
    (cond ((%builtin-type? k) k)
          ((%cond? x)         'cons)
          (t                  'structure-object))))

;; Auflösung: eingebaut → Struct → Condition → Fehler (Spec).
(defun %typep-name (x name)
  (cond ((eq name t)                    t)
        ((eq name 'atom)                (not (pair? x)))
        ((%builtin-type? name)          (%builtin-subtype? (%type-start x) name))
        ((assoc name *struct-types*)    (%struct-instance? x name))
        ((assoc name *condition-types*) (if (%cond-type? x name) t ()))
        (t (error (format nil "typep: unbekannter Typ '~a'" name)))))

(defun %valid-bound? (b)
  (or (eq b '*)
      (number? b)
      (and (pair? b) (number? (car b)) (null (cdr b)))))

(defun %lower-ok? (x b)
  (cond ((eq b '*)   t)
        ((number? b) (>= x b))
        (t           (> x (car b)))))

(defun %upper-ok? (x b)
  (cond ((eq b '*)   t)
        ((number? b) (<= x b))
        (t           (< x (car b)))))

;; (integer lo hi) u. ä.: Grenze = Zahl (inklusiv), (zahl) (exklusiv),
;; * oder weggelassen (unbegrenzt). Grenzen werden vor dem Typ geprüft,
;; damit eine kaputte Angabe immer auffällt.
(defun %typep-range (x head args spec)
  (if (or (> (length args) 2) (not (every %valid-bound? args)))
      (%bad-spec spec)
      (if (and (%typep-name x head)
               (%lower-ok? x (if (null args) '* (car args)))
               (%upper-ok? x (if (null (cdr args)) '* (cadr args))))
          t
          ())))

;; (satisfies f): f ist Symbol; (eval f) löst auf, weil funcall
;; Symbole nicht auflöst.
(defun %typep-satisfies (x spec)
  (let ((args (cdr spec)))
    (if (not (and (pair? args) (symbol? (car args)) (null (cdr args))))
        (%bad-spec spec)
        (let* ((f  (car args))
               (fn (if (bound? f) (eval f) ())))
          (if (not (member (%cell-type fn) '(lambda func)))
              (error (format nil "typep: satisfies: '~a' ist keine Funktion" f))
              (if (funcall fn x) t ()))))))

(defun %one-arg? (args)
  (and (pair? args) (null (cdr args))))

(defun %typep-spec (x spec)
  (cond ((null spec)        ())
        ((symbol? spec)     (%typep-name x spec))
        ((not (pair? spec)) (%bad-spec spec))
        (t
         (let ((head (car spec))
               (args (cdr spec)))
           (cond ((eq head 'or)
                  (any (lambda (s) (%typep-spec x s)) args))
                 ((eq head 'and)
                  (every (lambda (s) (%typep-spec x s)) args))
                 ((eq head 'not)
                  (if (%one-arg? args) (not (%typep-spec x (car args))) (%bad-spec spec)))
                 ((eq head 'member)
                  (any (lambda (v) (eql x v)) args))
                 ((eq head 'eql)
                  (if (%one-arg? args) (eql x (car args)) (%bad-spec spec)))
                 ((eq head 'satisfies)
                  (%typep-satisfies x spec))
                 ((member head '(integer float real rational))
                  (%typep-range x head args spec))
                 (t (%bad-spec spec)))))))

(defun typep (x spec)
  (%typep-spec x spec))
```

- [ ] **Step 4: Tests grün sehen**

Run: `go test ./src/lib/ -run 'TestTypep|TestTypeOf|TestStructRegistry' -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add src/embed/types.lisp src/lib/types_typep_test.go
git commit -m "feat(types): typep mit Hierarchie, Kombinatoren und Zahlbereichen

Co-Authored-By: <dein eigenes Modell> <noreply@anthropic.com>"
```

---

