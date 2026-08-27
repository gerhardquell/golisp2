### Task 4: env-symbols-Primitiv

**Files:**
- Modify: `src/lib/primitives.go` (Registrierung in `BaseEnv()`)
- Test: `src/lib/primitives_test.go`

**Interfaces:**
- Produces: `(env-symbols)` — Lisp-Primitiv, 0 Argumente, gibt eine Liste
  von Strings zurück (alle im aufrufenden Env sichtbaren Symbolnamen,
  Root-Scope eingeschlossen). Wird von Task 5 (Doku-Generator) konsumiert.

- [ ] **Step 1: Failing Test schreiben**

An `src/lib/primitives_test.go` anhängen:

```go
func TestEnvSymbols(t *testing.T) {
	env := BaseEnv()
	result, err := Eval(mustRead(t, "(env-symbols)"), env)
	if err != nil {
		t.Fatalf("(env-symbols): %v", err)
	}
	names := map[string]bool{}
	for _, c := range CellToSlice(result) {
		if c.Type != STRING {
			t.Fatalf("env-symbols: String erwartet, got %v", c.Type)
		}
		names[c.Val] = true
	}
	if len(names) < 10 {
		t.Fatalf("env-symbols: zu wenige Symbole (%d), BaseEnv() liefert deutlich mehr", len(names))
	}
	for _, want := range []string{"car", "cons", "defun", "+"} {
		if !names[want] {
			t.Errorf("env-symbols: erwartetes Symbol %q fehlt", want)
		}
	}
}
```

`mustRead(t *testing.T, code string) *Cell` existiert bereits in
`src/lib/wsbridge_test.go:149` (gleiches Package `lib`) — nicht neu
anlegen, nur verwenden.

- [ ] **Step 2: Test ausführen, Fehlschlag verifizieren**

```bash
go test ./src/lib/ -run TestEnvSymbols -v
```

Erwartung: FAIL — `(env-symbols)` ist unbekanntes Symbol
(`eval: unbound symbol: env-symbols` o. ä.).

- [ ] **Step 3: Primitiv implementieren**

In `src/lib/primitives.go`, in `BaseEnv()` nach dem Block
`// Memory-Profiling` (vor `// Zeitfunktionen`) einfügen:

```go
	// Introspektion
	_ = env.Set("env-symbols", makeFn(func(args []*Cell) (*Cell, error) {
		names := env.Symbols()
		sort.Strings(names)
		cells := make([]*Cell, len(names))
		for i, n := range names {
			cells[i] = MakeStr(n)
		}
		return SliceToCell(cells), nil
	}))
```

`sort` zum Import-Block am Dateikopf hinzufügen (aktuell nicht importiert):

```go
import (
	"bufio"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	...
)
```

(Reihenfolge alphabetisch einsortiert, restliche Imports unverändert
belassen.)

- [ ] **Step 4: Test ausführen, Erfolg verifizieren**

```bash
go test ./src/lib/ -run TestEnvSymbols -v
```

Erwartung: PASS.

- [ ] **Step 5: Vollen Testlauf gegen Regressionen prüfen, Binary neu bauen**

```bash
go build ./...
go test ./... -count=1
./build.sh
```

Erwartung: keine Regressionen (sortierte Ausgabe von `env-symbols`
beeinflusst kein bestehendes Verhalten, da neues Symbol). `./build.sh` ist
hier zwingend — Task 5 ruft `./build/golisp2 tools/gen-reference.lisp`
auf und braucht dafür ein Binary, das `(env-symbols)` bereits enthält;
ohne Neubau liefe dort noch das alte Binary vom Worktree-Baseline-Build.

- [ ] **Step 6: Commit**

```bash
git add src/lib/primitives.go src/lib/primitives_test.go
git commit -m "$(cat <<'EOF'
feat(primitives): env-symbols liefert alle sichtbaren Symbolnamen

Loest das in CLAUDE.md ("Was es bewusst nicht gibt") bereits erwaehnte
(env-symbols) real ein -- bisher gab es nur Env.Symbols() als interne
Go-Methode (genutzt von swank--symbols), kein Lisp-Primitiv. Grundlage
fuer den Doku-Generator (naechster Commit).

Co-Authored-By: claude-sonnet-5 <noreply@anthropic.com>
EOF
)"
```

---

