### Task 5: Kollisionswarnungen

**Files:**
- Modify: `src/embed/types.lisp` (am Ende anhängen)
- Modify: `src/embed/stdlib.lisp` (`defstruct`-Expansion)
- Modify: `src/embed/condition.lisp` (`define-condition`-Expansion)
- Modify: `src/lib/types_typep_test.go`

**Interfaces:**
- Consumes: `%builtin-type?`, `*struct-types*`, `*condition-types*`; Test-Helfer `captureStderr(t, fn func()) string` aus `src/lib/redeflog_test.go`.
- Produces: `(%type-name-warnings kind name)` → Liste von `(warn "…")`-Formen, `kind` ist `'defstruct` oder `'define-condition`. Wird zur Makro-Expansionszeit aufgerufen.

- [ ] **Step 1: Failing tests ergänzen** — an `src/lib/types_typep_test.go` anhängen und die Zeile `var _ = strings.Contains …` aus Task 2 entfernen:

```go
func stderrVon(t *testing.T, src string) string {
  t.Helper()
  return captureStderr(t, func() {
    if _, err := evalStdlib(t, src); err != nil {
      t.Fatalf("eval: %v", err)
    }
  })
}

func TestKollisionStructEingebauterTyp(t *testing.T) {
  out := stderrVon(t, `(defstruct integer a)`)
  if !strings.Contains(out, "WARN: defstruct integer: Name ist ein eingebauter Typ") {
    t.Errorf("Warnung fehlt, stderr = %q", out)
  }
  // eingebauter Typ gewinnt
  evalStdlibEq(t, `(defstruct integer a) (type-of (make-integer :a 1))`, "cons")
  evalStdlibEq(t, `(defstruct integer a) (typep (make-integer :a 1) 'integer)`, "()")
}

func TestKollisionStructCondition(t *testing.T) {
  out := stderrVon(t, `(define-condition mein-fehler (error) ()) (defstruct mein-fehler a)`)
  if !strings.Contains(out, "WARN: defstruct mein-fehler: Name ist auch ein Condition-Typ") {
    t.Errorf("Warnung fehlt, stderr = %q", out)
  }
  out = stderrVon(t, `(defstruct punkt x y) (define-condition punkt (error) ())`)
  if !strings.Contains(out, "WARN: define-condition punkt: Name ist auch ein Struct") {
    t.Errorf("Warnung fehlt, stderr = %q", out)
  }
  // Struct gewinnt
  evalStdlibEq(t, `(define-condition mein-fehler (error) ()) (defstruct mein-fehler a) (typep (make-mein-fehler :a 1) 'mein-fehler)`, "t")
}

func TestKollisionConditionEingebauterTyp(t *testing.T) {
  out := stderrVon(t, `(define-condition string (error) ())`)
  if !strings.Contains(out, "WARN: define-condition string: Name ist ein eingebauter Typ") {
    t.Errorf("Warnung fehlt, stderr = %q", out)
  }
}

func TestKeineWarnungOhneKollision(t *testing.T) {
  out := stderrVon(t, `(defstruct punkt x y) (define-condition datei-fehler (error) (pfad))`)
  if strings.Contains(out, "WARN") {
    t.Errorf("unerwartete Warnung: %q", out)
  }
}
```

- [ ] **Step 2: Tests rot sehen**

Run: `go test ./src/lib/ -run 'TestKollision|TestKeineWarnung' -count=1`
Expected: FAIL — Warnungen fehlen (`TestKeineWarnungOhneKollision` darf schon grün sein).

- [ ] **Step 3: Implementierung**

An `src/embed/types.lisp` anhängen:

```lisp
;; === Kollisionen =====================================================

;; %type-name-warnings: (warn …)-Formen für defstruct/define-condition,
;; wenn name einen eingebauten Typ oder die jeweils andere Registry trifft.
;; Wird zur Expansionszeit aufgerufen; kind = 'defstruct | 'define-condition.
(defun %type-name-warnings (kind name)
  (mapcar (lambda (m) (list 'warn m))
          (filter identity
            (list
              (if (%builtin-type? name)
                  (format nil "WARN: ~a ~a: Name ist ein eingebauter Typ → typep/type-of sehen den eingebauten Typ" kind name)
                  ())
              (if (and (eq kind 'defstruct) (assoc name *condition-types*))
                  (format nil "WARN: defstruct ~a: Name ist auch ein Condition-Typ → typep sieht den Struct" name)
                  ())
              (if (and (eq kind 'define-condition) (assoc name *struct-types*))
                  (format nil "WARN: define-condition ~a: Name ist auch ein Struct → typep sieht den Struct" name)
                  ())))))
```

In `src/embed/stdlib.lisp`, `defstruct`-Expansion — die in Task 2 geänderten Zeilen

```lisp
    `(begin
       (set! *struct-types* (alist-set ',name ,n *struct-types*))
```

ersetzen durch

```lisp
    `(begin
       ,@(if (bound? '%type-name-warnings) (%type-name-warnings 'defstruct name) '())
       (set! *struct-types* (alist-set ',name ,n *struct-types*))
```

In `src/embed/condition.lisp`, `define-condition`, die Zeilen

```lisp
  `(progn
     (set! *condition-types*
```

ersetzen durch

```lisp
  `(progn
     ,@(if (bound? '%type-name-warnings) (%type-name-warnings 'define-condition name) '())
     (set! *condition-types*
```

(`bound?`-Schutz: `types.lisp` wird nach `stdlib.lisp`/`condition.lisp` geladen; die Makros expandieren erst bei Benutzung, aber einzeln geladene Dateien sollen nicht brechen.)

- [ ] **Step 4: Tests grün sehen**

Run: `go test ./src/lib/ -run 'TestKollision|TestKeineWarnung|TestTypep|TestTypeOf|TestStructRegistry' -count=1`
Expected: PASS.
Dann `go test ./... -count=1` — alles grün.

- [ ] **Step 5: Commit**

```bash
git add src/embed/types.lisp src/embed/stdlib.lisp src/embed/condition.lisp src/lib/types_typep_test.go
git commit -m "feat(types): WARN bei Namenskollision von Struct, Condition und eingebautem Typ

Co-Authored-By: <dein eigenes Modell> <noreply@anthropic.com>"
```

---

