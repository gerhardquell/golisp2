### Task 1: Go-Primitiv `%cell-type`

**Files:**
- Create: `src/lib/celltype.go`
- Create: `src/lib/celltype_test.go`
- Modify: `src/lib/primitives.go` (in `BaseEnv()`, direkt nach `RegisterHashtables(env)`)

**Interfaces:**
- Consumes: `makeFn`, `MakeAtom`, `Env.Set`, Typkonstanten aus `src/lib/types.go` (`NUMBER`, `STRING`, `ATOM`, `LIST`, `NIL`, `LAMBDA`, `FUNC`, `MACRO`, `HASHTABLE`).
- Produces: Lisp-Funktion `(%cell-type x)` → eines der Symbole `number`, `string`, `symbol`, `cons`, `null`, `lambda`, `func`, `macro`, `hash-table`, sonst `t`. Go: `func RegisterCellType(env *Env)`.

- [ ] **Step 1: Failing test schreiben** — `src/lib/celltype_test.go`:

```go
//**********************************************************************
//  lib/celltype_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : <dein eigenes Modell>
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260929
//**********************************************************************

package lib

import "testing"

func TestCellTypeKernTypen(t *testing.T) {
  evalEq(t, `(%cell-type 3.5)`, "number")
  evalEq(t, `(%cell-type "a")`, "string")
  evalEq(t, `(%cell-type 'foo)`, "symbol")
  evalEq(t, `(%cell-type :k)`, "symbol")
  evalEq(t, `(%cell-type '(1 2))`, "cons")
  evalEq(t, `(%cell-type '())`, "null")
  evalEq(t, `(%cell-type (lambda (x) x))`, "lambda")
  evalEq(t, `(%cell-type car)`, "func")
  evalEq(t, `(%cell-type (make-hash-table))`, "hash-table")
}

func TestCellTypeMakro(t *testing.T) {
  evalEq(t, `(progn (defmacro ct-mm (x) x) (%cell-type ct-mm))`, "macro")
}

func TestCellTypeArity(t *testing.T) {
  if _, err := evalStr(`(%cell-type)`); err == nil {
    t.Errorf("(%%cell-type) ohne Argument sollte Fehler geben")
  }
}
```

(`evalStr(src string) (*Cell, error)` und `evalEq` stehen in `src/lib/eval_test.go`, ohne stdlib.)

- [ ] **Step 2: Test rot sehen**

Run: `go test ./src/lib/ -run 'TestCellType' -count=1`
Expected: FAIL mit `unbekanntes Symbol '%cell-type'`.

- [ ] **Step 3: Implementierung** — `src/lib/celltype.go`:

```go
//**********************************************************************
//  lib/celltype.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : <dein eigenes Modell>
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260929
//**********************************************************************
// %cell-type: Kern-Typ einer Cell als Symbol. Einzige Go-Stelle, an der
// types.lisp (type-of/typep) Kernfakten abfragt — die Typhierarchie
// selbst lebt ausschliesslich in types.lisp.
//**********************************************************************

package lib

import "fmt"

func RegisterCellType(env *Env) {
  _ = env.Set("%cell-type", makeFn(fnCellType))
}

func fnCellType(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("%%cell-type: 1 Argument nötig")
  }
  return MakeAtom(cellTypeName(args[0])), nil
}

func cellTypeName(c *Cell) string {
  switch c.Type {
  case NUMBER:
    return "number"
  case STRING:
    return "string"
  case ATOM:
    return "symbol"
  case LIST:
    return "cons"
  case NIL:
    return "null"
  case LAMBDA:
    return "lambda"
  case FUNC:
    return "func"
  case MACRO:
    return "macro"
  case HASHTABLE:
    return "hash-table"
  default:
    return "t"
  }
}
```

In `src/lib/primitives.go` direkt nach dem Block `RegisterHashtables(env)` (Tabs!):

```go
	// Kern-Typ für type-of/typep (types.lisp)
	RegisterCellType(env)
```

- [ ] **Step 4: Test grün sehen**

Run: `go test ./src/lib/ -run 'TestCellType' -count=1`
Expected: PASS. Danach `go build ./...` ohne Fehler.

- [ ] **Step 5: Commit**

```bash
git add src/lib/celltype.go src/lib/celltype_test.go src/lib/primitives.go
git commit -m "feat(types): Primitiv %cell-type liefert Kern-Typ einer Cell

Co-Authored-By: <dein eigenes Modell> <noreply@anthropic.com>"
```

---

