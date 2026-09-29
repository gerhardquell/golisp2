### Task 2: Struct-Registry in `defstruct`

**Files:**
- Modify: `src/embed/stdlib.lisp` (vor `(defmacro defstruct …)` und im `` `(begin `` der Expansion)
- Create: `src/lib/types_typep_test.go`

**Interfaces:**
- Consumes: `alist-set` (stdlib, Format `(key val)`), `defvar`.
- Produces: globale Variable `*struct-types*` — Liste von `(name slot-anzahl)`, z. B. `((punkt 2) (leer 0))`. `defstruct` trägt bei jeder Ausführung ein (Reload ersetzt den Eintrag).

- [ ] **Step 1: Failing test schreiben** — `src/lib/types_typep_test.go` anlegen:

```go
//**********************************************************************
//  lib/types_typep_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : <dein eigenes Modell>
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260929
//**********************************************************************
// type-of / typep (src/embed/types.lisp), Struct-Registry (stdlib.lisp)
// und Kollisionswarnungen. Spec: docs/superpowers/specs/2026-09-29-typep-design.md
//**********************************************************************

package lib

import (
  "strings"
  "testing"
)

var _ = strings.Contains // wird ab Task 5 gebraucht

func TestStructRegistry(t *testing.T) {
  evalStdlibEq(t, `(defstruct punkt x y) *struct-types*`, "((punkt 2))")
  evalStdlibEq(t, `(defstruct leer) *struct-types*`, "((leer 0))")
  // Reload ersetzt, dupliziert nicht
  evalStdlibEq(t, `(defstruct punkt x y) (defstruct punkt x y z) *struct-types*`, "((punkt 3))")
}
```

- [ ] **Step 2: Test rot sehen**

Run: `go test ./src/lib/ -run 'TestStructRegistry' -count=1`
Expected: FAIL mit `unbekanntes Symbol '*struct-types*'`.

- [ ] **Step 3: Implementierung** — in `src/embed/stdlib.lisp` direkt **vor** dem Kommentarblock `;; defstruct: (defstruct name [docstring] slot…)`:

```lisp
;; *struct-types*: Registry (name slot-anzahl) für type-of/typep (types.lisp).
;; defvar: Mehrfach-Load darf registrierte Structs nicht löschen.
(defvar *struct-types* '())

```

Im `defstruct`-Makro die Zeilen

```lisp
    `(begin
       (defun ,mk (&key ,@slots) (%make-struct ',name ,@slot-names))
```

ersetzen durch

```lisp
    `(begin
       (set! *struct-types* (alist-set ',name ,n *struct-types*))
       (defun ,mk (&key ,@slots) (%make-struct ',name ,@slot-names))
```

(`n` ist im `let*` des Makros bereits als `(length slots)` gebunden.)

- [ ] **Step 4: Test grün sehen**

Run: `go test ./src/lib/ -run 'TestStructRegistry' -count=1`
Expected: PASS.
Dann: `go test ./src/lib/ -count=1` — alles grün (defstruct-Bestandstests).

- [ ] **Step 5: Commit**

```bash
git add src/embed/stdlib.lisp src/lib/types_typep_test.go
git commit -m "feat(stdlib): defstruct registriert Structs in *struct-types*

Co-Authored-By: <dein eigenes Modell> <noreply@anthropic.com>"
```

---

