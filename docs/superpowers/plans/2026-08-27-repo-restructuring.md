# Repo-Restructuring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Repo-Layout gemäß TODO.md (20260827-1) umbauen: ungenutzte
Verzeichnisse nach `unused/`, `doc/`+`docs/` zu `docs/` verschmelzen,
Go-Code nach `src/`, und eine automatisch generierte Referenz-Doku (Mensch
+ KI) für alle golisp2-Funktionen.

**Architecture:** Reine Dateisystem-Umzüge per `git mv` (Historie bleibt
erhalten) in drei sequenziellen Schritten, gefolgt von einer neuen
Go-Primitive (`env-symbols`) und einem Lisp-Generator-Skript, das darauf
aufbaut. Jeder Schritt ist ein eigener Commit mit eigenem Build/Test-Gate.

**Tech Stack:** Go 1.26 (Standardbibliothek, kein neues Dependency), GoLisp2
selbst für das Generator-Skript.

**Spec:** `docs/superpowers/specs/2026-08-27-repo-restructuring-design.md`

## Global Constraints

- Jeder Schritt = ein Commit, Rollback pro Schritt muss möglich sein, ohne
  andere Schritte zu berühren (aus Spec, Abschnitt "Schritte").
- `golisp2web/` (eigenes Git-Repo), `extern/sigoREST` (Symlink), `chinese/`
  bleiben unangetastet (Spec, "Bewusst nicht").
- `docs/gespraeche/` (archivierte Konversationen) und alle datierten
  Retrospektiven/Pläne/Specs unter `docs/superpowers/` werden NICHT auf
  neue Pfade umgeschrieben — sie sind historischer Stand, kein aktiver
  Verweis. Nur `CLAUDE.md` und `tests/conformance/README.md` sind lebende
  Dokumente und werden korrigiert.
- Go-Modulname `golisp2` in `go.mod` bleibt unverändert, nur interne
  Importpfade ändern sich.
- Nach jedem Schritt: `go build ./...` muss fehlerfrei sein, bevor der
  nächste Schritt beginnt.

---

### Task 1: Cleanup — unused/

**Files:**
- Move: `experiment/` → `unused/experiment/`
- Move: `images/` → `unused/images/`
- Move: `libs/` → `unused/libs/`
- Move: `pn-gps1/` → `unused/pn-gps1/`
- Move: `tools/gen-training/` → `unused/gen-training/`
- Move: `doc/files.zip` → `unused/files.zip`
- Finalize: `PerfTODO.md` → `todos/PerfTODO.md` (bereits im Working Tree
  als `D`/`??` sichtbar — Verschiebung nur noch stagen)
- Modify: `CLAUDE.md` (Doku-Tabelle, PerfTODO-Zeile)

**Interfaces:** keine (reine Dateisystem-Operation, kein Code betroffen)

- [ ] **Step 1: Cleanup-Kandidaten verschieben**

```bash
mkdir -p unused
git mv experiment unused/experiment
git mv images unused/images
git mv libs unused/libs
git mv pn-gps1 unused/pn-gps1
git mv tools/gen-training unused/gen-training
git mv doc/files.zip unused/files.zip
```

- [ ] **Step 2: PerfTODO-Verschiebung nachziehen**

`git status --short PerfTODO.md todos/PerfTODO.md` zeigt `D PerfTODO.md`
und `?? todos/PerfTODO.md` (von Gerhard bereits im Working Tree begonnen).
Stagen, damit Git die Umbenennung als Rename erkennt:

```bash
git add PerfTODO.md todos/PerfTODO.md
git status --short PerfTODO.md todos/PerfTODO.md
```

Erwartung: `R  PerfTODO.md -> todos/PerfTODO.md` (Rename erkannt).

- [ ] **Step 3: CLAUDE.md — PerfTODO-Zeile korrigieren**

In der Tabelle "Weiterführende Doku" steht (Tippfehler in der Groß-/
Kleinschreibung, Pfad veraltet):

```
| `perfTodo.md` | Offene Performance-Arbeit | du optimierst |
```

Ersetzen durch:

```
| `todos/PerfTODO.md` | Offene Performance-Arbeit | du optimierst |
```

- [ ] **Step 4: Verifizieren — keine toten Referenzen auf verschobene Pfade**

```bash
grep -rn "experiment/\|libs/\|pn-gps1/\|tools/gen-training" --include=*.go --include=*.lisp --include=*.md . \
  | grep -v "^./unused/" | grep -v "^./docs/retrospectives/" | grep -v "^./docs/gespraeche/"
```

Erwartung: keine Treffer außerhalb der bewusst ausgenommenen Verzeichnisse
(Retrospektiven sind historisch, siehe Global Constraints).

- [ ] **Step 5: Commit**

```bash
git add unused CLAUDE.md
git commit -m "$(cat <<'EOF'
chore(repo): ungenutzte Verzeichnisse nach unused/ verschoben

experiment/, images/, libs/, pn-gps1/, tools/gen-training/, doc/files.zip
seit Juli 2026 unangetastet und ohne aktive Referenz — siehe
docs/superpowers/specs/2026-08-27-repo-restructuring-design.md.
PerfTODO.md-Verschiebung nach todos/ (Gerhard) mit committet.

Co-Authored-By: claude-sonnet-5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Doc-Merge — doc/ + docs/ → docs/

**Files:**
- Move: `doc/AUTHORS.md`, `doc/cl-inventar.md`, `doc/cli.md`,
  `doc/emacs-golisp2web.md`, `doc/golisp2-cheatsheet.md`,
  `doc/lisp-semantik.md`, `doc/memory.md`, `doc/sigo.md`,
  `doc/struktur.md`, `doc/swank.md` → `docs/<gleicher-name>`
- Move: `doc/ki/` → `docs/ki/`
- Modify: `CLAUDE.md` (Doku-Tabelle-Pfade, Struktur-Verweis)
- Modify: `tests/conformance/README.md:54`

**Interfaces:** keine

- [ ] **Step 1: Dateien verschieben**

```bash
git mv doc/AUTHORS.md docs/AUTHORS.md
git mv doc/cl-inventar.md docs/cl-inventar.md
git mv doc/cli.md docs/cli.md
git mv doc/emacs-golisp2web.md docs/emacs-golisp2web.md
git mv doc/golisp2-cheatsheet.md docs/golisp2-cheatsheet.md
git mv doc/lisp-semantik.md docs/lisp-semantik.md
git mv doc/memory.md docs/memory.md
git mv doc/sigo.md docs/sigo.md
git mv doc/struktur.md docs/struktur.md
git mv doc/swank.md docs/swank.md
git mv doc/ki docs/ki
rmdir doc
```

- [ ] **Step 2: CLAUDE.md — Struktur-Verweis korrigieren**

Alt:
```
Vollständige Datei-für-Datei-Beschreibung: `doc/struktur.md`.
```
Neu:
```
Vollständige Datei-für-Datei-Beschreibung: `docs/struktur.md`.
```

- [ ] **Step 3: CLAUDE.md — Doku-Tabelle auf docs/ umstellen**

Alt:
```
| `doc/struktur.md` | Datei-für-Datei-Beschreibung von `lib/` | du dich neu orientierst |
| `doc/cli.md` | Flags, Exit-Codes, Multiline-stdin, `exec`-Syntax | du an `main.go` arbeitest |
| `doc/swank.md` | SWANK-Protokoll, Framing, Op-Tabelle, SLIME-Details | du an `lib/swank/` arbeitest |
| `doc/emacs-golisp2web.md` | golisp2web aus dem SLIME-REPL starten/steuern, `parfunc`+`system`-Muster, Beispiele | du golisp2web aus Emacs heraus benutzen willst |
| `doc/sigo.md` | sigoREST: Env-Vars, Rate-Limiting, Multi-Host, Muster | du an `lib/sigorest.go` arbeitest |
| `doc/lisp-semantik.md` | `eq`/`equal?`, `let`/`let*`, `setq*`, `case`, FORMAT | Semantik unklar ist |
| `doc/memory.md` | GC-Verhalten, `(memstats)`, Best Practices | du Speicher untersuchst |
```
Neu (nur `doc/` → `docs/`, `lib/`- und `main.go`-Anteile bleiben für Task 3):
```
| `docs/struktur.md` | Datei-für-Datei-Beschreibung von `lib/` | du dich neu orientierst |
| `docs/cli.md` | Flags, Exit-Codes, Multiline-stdin, `exec`-Syntax | du an `main.go` arbeitest |
| `docs/swank.md` | SWANK-Protokoll, Framing, Op-Tabelle, SLIME-Details | du an `lib/swank/` arbeitest |
| `docs/emacs-golisp2web.md` | golisp2web aus dem SLIME-REPL starten/steuern, `parfunc`+`system`-Muster, Beispiele | du golisp2web aus Emacs heraus benutzen willst |
| `docs/sigo.md` | sigoREST: Env-Vars, Rate-Limiting, Multi-Host, Muster | du an `lib/sigorest.go` arbeitest |
| `docs/lisp-semantik.md` | `eq`/`equal?`, `let`/`let*`, `setq*`, `case`, FORMAT | Semantik unklar ist |
| `docs/memory.md` | GC-Verhalten, `(memstats)`, Best Practices | du Speicher untersuchst |
```

- [ ] **Step 4: tests/conformance/README.md korrigieren**

Alt (Zeile 54):
```
Details: `doc/cl-inventar.md`, Befundliste in TODO.md.
```
Neu:
```
Details: `docs/cl-inventar.md`, Befundliste in TODO.md.
```

- [ ] **Step 5: Verifizieren**

```bash
test ! -d doc && echo "doc/ entfernt: OK"
grep -rn '`doc/\|"doc/\|(doc/' --include=*.md . \
  | grep -v "^./docs/retrospectives/" | grep -v "^./docs/gespraeche/" \
  | grep -v "^./docs/superpowers/plans/" | grep -v "^./docs/superpowers/specs/"
```

Erwartung: `doc/` existiert nicht mehr; zweiter Befehl liefert keine
Treffer (alle verbleibenden `doc/`-Erwähnungen sind historische
Dokumente, siehe Global Constraints).

- [ ] **Step 6: Commit**

```bash
git add docs CLAUDE.md tests/conformance/README.md
git commit -m "$(cat <<'EOF'
docs(repo): doc/ und docs/ zu docs/ zusammengefuehrt

Alle aktiven Doku-Querverweise (CLAUDE.md, tests/conformance/README.md)
auf docs/ korrigiert. Historische Retrospektiven/Pläne/Specs bleiben mit
ihren ursprünglichen doc/-Erwähnungen unangetastet (dokumentieren ihren
damaligen Stand).

Co-Authored-By: claude-sonnet-5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Src-Umzug — main.go, cmd/, lib/, embed/ → src/

**Files:**
- Move: `main.go`, `main_test.go`, `cmd/`, `lib/`, `embed/` → `src/`
- Modify: 16 `.go`-Dateien mit Importpfad `golisp2/lib*` oder
  `golisp2/embed` (Liste in Step 2)
- Modify: `build.sh`
- Modify: `CLAUDE.md` (Orientierung-Block, Chokepoint-Tabelle,
  Doku-Tabelle, `rg`-Hinweis)

**Interfaces:** keine neuen — reine Pfadverschiebung, alle Go-Symbole
bleiben unverändert.

- [ ] **Step 1: Dateien/Verzeichnisse verschieben**

```bash
mkdir -p src
git mv main.go src/main.go
git mv main_test.go src/main_test.go
git mv cmd src/cmd
git mv lib src/lib
git mv embed src/embed
```

- [ ] **Step 2: Importpfade korrigieren**

Betroffene Dateien (ermittelt via
`grep -rlE '"golisp2/(lib|embed)' --include=*.go .`):

```
src/main.go
src/main_test.go
src/cmd/golisp2-client/main.go
src/lib/shm_lisp.go
src/lib/shm_lisp_test.go
src/lib/stdlib.go
src/lib/swank/dispatch.go
src/lib/swank/dispatch_test.go
src/lib/swank/env.go
src/lib/swank/env_test.go
src/lib/swank/framing.go
src/lib/swank/framing_test.go
src/lib/swank/integration_test.go
src/lib/swank/lisp.go
src/lib/swank/lisp_test.go
src/lib/swank/server.go
```

```bash
grep -rlE '"golisp2/(lib|embed)' --include=*.go . | \
  xargs sed -i -E 's#"golisp2/(lib|embed)#"golisp2/src/\1#g'
```

- [ ] **Step 3: build.sh anpassen**

Alt:
```bash
echo "→ go vet"
go vet ./lib/ ./lib/swank/ 2>/dev/null || true

echo "→ build golisp2"
go build -o build/golisp2 .

echo "→ build golisp2-client"
go build -o build/golisp2-client ./cmd/golisp2-client/
```
Neu:
```bash
echo "→ go vet"
go vet ./src/lib/ ./src/lib/swank/ 2>/dev/null || true

echo "→ build golisp2"
go build -o build/golisp2 ./src/

echo "→ build golisp2-client"
go build -o build/golisp2-client ./src/cmd/golisp2-client/
```

- [ ] **Step 4: go build verifizieren**

```bash
go build ./...
```

Erwartung: keine Fehler. Falls `imported and not used` o. ä. auftritt: ein
Importpfad wurde nicht erfasst — erneut mit
`grep -rn '"golisp2/lib"' --include=*.go .` (ohne Subpfad) nach
Restvorkommen suchen.

- [ ] **Step 5: CLAUDE.md — Orientierung-Block**

Alt:
```
main.go              CLI: stdin / -i / -e / -t / --swank / Datei
cmd/golisp2-client/  CLI-Client mit REPL
build/               Build-Artefakte (golisp2, golisp2-client)
lib/
  types*.go          Cell-Datenstruktur, Small-Int-Cache, Helfer
  reader.go          Parser: String → Cell-Baum
  env.go             Environment: verkettete Scopes, RWMutex
  eval_core.go       Eval-Trampolin, apply, evalArgs
  eval_*.go          Spezialformen, Lambda, Control, Quasiquote, load, exec
  primitives.go      Eingebaute Funktionen + BaseEnv()
  format*.go         FORMAT-Engine (CL-HyperSpec 22.3)
  <domäne>.go        goroutine, fileio, shellcmd, postgres, genalg, shm, sigorest
  stdlib.go          //go:embed stdlib.lisp
  readline.go        REPL (go-prompt, Highlighting, History)
  swank/             SWANK-Server für Emacs/SLIME
```
Neu:
```
src/
  main.go              CLI: stdin / -i / -e / -t / --swank / Datei
  cmd/golisp2-client/  CLI-Client mit REPL
  embed/               //go:embed Assets (stdlib.lisp, swank.lisp, ...)
  lib/
    types*.go          Cell-Datenstruktur, Small-Int-Cache, Helfer
    reader.go          Parser: String → Cell-Baum
    env.go             Environment: verkettete Scopes, RWMutex
    eval_core.go       Eval-Trampolin, apply, evalArgs
    eval_*.go          Spezialformen, Lambda, Control, Quasiquote, load, exec
    primitives.go      Eingebaute Funktionen + BaseEnv()
    format*.go         FORMAT-Engine (CL-HyperSpec 22.3)
    <domäne>.go        goroutine, fileio, shellcmd, postgres, genalg, shm, sigorest
    stdlib.go          //go:embed stdlib.lisp
    readline.go        REPL (go-prompt, Highlighting, History)
    swank/             SWANK-Server für Emacs/SLIME
build/               Build-Artefakte (golisp2, golisp2-client)
```

- [ ] **Step 6: CLAUDE.md — Chokepoint-Tabelle**

Alt:
```
| HTTP gegen sigoREST | `lib/sigorest.go` |
| Stdlib laden | `LoadStdlib` (`lib/stdlib.go`) |
| Truthiness | `IsTruthy` (`lib/types_helpers.go`) |
| Primitiven registrieren | `BaseEnv()` (`lib/primitives.go`) |
| Parsen | `lib/reader.go` (`Read` / `ReadAll`) |
| SWANK-Framing | `lib/swank/framing.go` |
| Eval-Schleife | `lib/eval_core.go` |
```
Neu:
```
| HTTP gegen sigoREST | `src/lib/sigorest.go` |
| Stdlib laden | `LoadStdlib` (`src/lib/stdlib.go`) |
| Truthiness | `IsTruthy` (`src/lib/types_helpers.go`) |
| Primitiven registrieren | `BaseEnv()` (`src/lib/primitives.go`) |
| Parsen | `src/lib/reader.go` (`Read` / `ReadAll`) |
| SWANK-Framing | `src/lib/swank/framing.go` |
| Eval-Schleife | `src/lib/eval_core.go` |
```

- [ ] **Step 7: CLAUDE.md — restliche lib/-Erwähnungen**

Alt:
```
Bewacht von `TestNoLispDefineShadowsSpecialForm` (`lib/specialform_shadow_test.go`),
```
Neu:
```
Bewacht von `TestNoLispDefineShadowsSpecialForm` (`src/lib/specialform_shadow_test.go`),
```

Alt (Doku-Tabelle, Rest-Anteile aus Task 2):
```
| `docs/struktur.md` | Datei-für-Datei-Beschreibung von `lib/` | du dich neu orientierst |
| `docs/cli.md` | Flags, Exit-Codes, Multiline-stdin, `exec`-Syntax | du an `main.go` arbeitest |
| `docs/swank.md` | SWANK-Protokoll, Framing, Op-Tabelle, SLIME-Details | du an `lib/swank/` arbeitest |
```
Neu:
```
| `docs/struktur.md` | Datei-für-Datei-Beschreibung von `src/lib/` | du dich neu orientierst |
| `docs/cli.md` | Flags, Exit-Codes, Multiline-stdin, `exec`-Syntax | du an `src/main.go` arbeitest |
| `docs/swank.md` | SWANK-Protokoll, Framing, Op-Tabelle, SLIME-Details | du an `src/lib/swank/` arbeitest |
```

Alt:
```
| `docs/sigo.md` | sigoREST: Env-Vars, Rate-Limiting, Multi-Host, Muster | du an `lib/sigorest.go` arbeitest |
```
Neu:
```
| `docs/sigo.md` | sigoREST: Env-Vars, Rate-Limiting, Multi-Host, Muster | du an `src/lib/sigorest.go` arbeitest |
```

Alt:
```
Wahrheit ist der Code:
  `rg 'env\.Set\("' lib/` bzw. `(env-symbols)` zur Laufzeit.
```
Neu:
```
Wahrheit ist der Code:
  `rg 'env\.Set\("' src/lib/` bzw. `(env-symbols)` zur Laufzeit.
```

- [ ] **Step 8: Vollständige Verifikation**

```bash
go build ./...
go test ./... -count=1
./build.sh
./build/golisp2 -t
```

Erwartung: alle vier Befehle fehlerfrei / alle Tests grün.

- [ ] **Step 9: Commit**

```bash
git add src build.sh CLAUDE.md
git commit -m "$(cat <<'EOF'
refactor(repo): Go-Code nach src/ verschoben

main.go, main_test.go, cmd/, lib/, embed/ -> src/. Importpfad
golisp2/lib(*) -> golisp2/src/lib(*), golisp2/embed -> golisp2/src/embed.
build.sh und CLAUDE.md-Pfadverweise korrigiert. Modulname golisp2
unveraendert.

Co-Authored-By: claude-sonnet-5 <noreply@anthropic.com>
EOF
)"
```

---

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

- [ ] **Step 5: Vollen Testlauf gegen Regressionen prüfen**

```bash
go build ./...
go test ./... -count=1
```

Erwartung: keine Regressionen (sortierte Ausgabe von `env-symbols`
beeinflusst kein bestehendes Verhalten, da neues Symbol).

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

### Task 5: Doku-Generator

**Files:**
- Create: `tools/gen-reference.lisp`
- Create: `docs/referenz-generiert.md` (generiertes Artefakt, committet)
- Modify: `docs/ki/referenz.md` (ersetzt bestehenden Inhalt komplett)
- Modify: `CLAUDE.md` (Doku-Tabelle: neuen Eintrag ergänzen)

**Interfaces:**
- Consumes: `(env-symbols)` aus Task 4, `(defined-in "pfad")` aus
  `src/lib/defloc.go` (bereits vorhanden, defsystem-lite), `(bound? sym)`
  (bereits vorhanden) zur Typklassifikation.
- Produces: zwei Markdown-Dateien, keine weiteren Konsumenten in diesem
  Plan.

- [ ] **Step 1: Generator-Skript schreiben**

`tools/gen-reference.lisp` — nutzt ausschließlich verifiziert vorhandene
Primitiven (`env-symbols` aus Task 4, `format`, `mapcar`, `apply`,
`string-append`, `file-write`, `length` — kein `open`/`sort`/`string<`,
die gibt es in diesem Repo nicht):

```lisp
;;**********************************************************************
;;  tools/gen-reference.lisp
;;  Autor    : Gerhard Quell - gquell@skequell.de
;;  CoAutor  : claude-sonnet-5
;;  Copyright: 2026 Gerhard Quell - SKEQuell
;;  Erstellt : 20260827
;;**********************************************************************
;; Generiert docs/referenz-generiert.md aus (env-symbols) -- Vollstaendigkeit
;; ist strukturell garantiert, da direkt aus dem Root-Env gelesen wird.
;; Aufruf: ./build/golisp2 tools/gen-reference.lisp
;;**********************************************************************

(define *ref-out* "docs/referenz-generiert.md")

(define (write-reference)
  (let* ((names (env-symbols))
         (rows (mapcar (lambda (n) (format nil "| `~a` | |~%" n)) names))
         (header (format nil "# GoLisp2 — Generierte Funktionsreferenz~%~%> Automatisch erzeugt aus `(env-symbols)` — nicht von Hand editieren.~%> Neu generieren: `./build/golisp2 tools/gen-reference.lisp`~%~%| Symbol | Beschreibung |~%|---|---|~%"))
         (body (apply string-append rows)))
    (file-write *ref-out* (string-append header body))
    (format t "~a Symbole nach ~a geschrieben~%" (length names) *ref-out*)))

(write-reference)
```

`env-symbols` liefert bereits eine sortierte Liste (`sort.Strings` in
Task 4), deshalb kein zusätzlicher Sortierschritt nötig.

- [ ] **Step 2: Skript ausführen**

```bash
./build/golisp2 tools/gen-reference.lisp
```

Erwartung: `docs/referenz-generiert.md` wird erzeugt, Konsolenausgabe
nennt eine Symbolzahl deutlich über 100.

- [ ] **Step 3: docs/ki/referenz.md aktualisieren**

Bestehenden Inhalt vollständig ersetzen durch eine kompakte Fassung nach
dem bisherigen Aufbau (Eval-Reihenfolge, Spezialformen-Tabelle,
Primitiven-Kurzliste) — Spezialformen-Liste manuell aus
`src/lib/eval_core.go` (case-Zweige, siehe Datei-Header-Kommentar
"Eval-Reihenfolge in `evalList`") übernehmen, Primitiven-Kurzliste aus
`docs/referenz-generiert.md`. Datei-Header aktualisieren:

```
> **Quelle:** `eval_core.go`, `primitives.go`, `embed/stdlib.lisp`,
> generiert via `tools/gen-reference.lisp` (Stand 20260827).
```

- [ ] **Step 4: CLAUDE.md — Doku-Tabelle ergänzen**

Nach der Zeile mit `docs/memory.md` einfügen:

```
| `docs/referenz-generiert.md` | Vollständige Funktionsreferenz, generiert aus `(env-symbols)` | du eine konkrete Funktion nachschlägst |
```

- [ ] **Step 5: Verifizieren**

```bash
go build ./...
go test ./... -count=1
./build/golisp2 -t
wc -l docs/referenz-generiert.md
```

Erwartung: alles grün, generierte Datei hat mehr als 100 Zeilen.

- [ ] **Step 6: Commit**

```bash
git add tools/gen-reference.lisp docs/referenz-generiert.md docs/ki/referenz.md CLAUDE.md
git commit -m "$(cat <<'EOF'
docs(reference): Funktionsreferenz aus (env-symbols) generiert

Ersetzt die manuell gepflegte, bereits 4 Wochen alte docs/ki/referenz.md.
tools/gen-reference.lisp ist bei Bedarf erneut ausfuehrbar (kein
Build-Hook, YAGNI) -- Vollstaendigkeit ist strukturell garantiert, da
direkt aus dem Root-Env gelesen wird statt von Hand gepflegt.

Co-Authored-By: claude-sonnet-5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Abschluss-Verifikation

**Files:** keine Änderungen erwartet — reines Verifikations-Gate.

**Interfaces:** keine

- [ ] **Step 1: Vollständiger Build**

```bash
go build ./...
```

- [ ] **Step 2: Go-Testsuite**

```bash
go test ./... -count=1
```

- [ ] **Step 3: Alle drei Binaries bauen**

```bash
./build.sh
```

- [ ] **Step 4: Lisp-Testsuite**

```bash
./build/golisp2 -t
```

- [ ] **Step 5: Stichprobe REPL-Smoke-Test**

```bash
echo "(+ 1 2)" | ./build/golisp2
./build/golisp2 -e "(env-symbols)" | head -c 200
```

Erwartung: `3`, danach eine nicht-leere Symbolliste.

- [ ] **Step 6: Reste prüfen**

```bash
test ! -d doc && echo "doc/ weg: OK"
test -d src/lib && echo "src/lib vorhanden: OK"
test -d unused && echo "unused/ vorhanden: OK"
git status --short
```

Erwartung: `git status --short` zeigt nur noch, was in vorherigen Tasks
bereits committet wurde (also idealerweise leer, da jeder Task committet
hat).

- [ ] **Step 7: Bei Erfolg — kein weiterer Commit nötig**

Dieser Task verändert nichts; er bestätigt nur, dass Tasks 1–5 zusammen
funktionieren. Falls ein Schritt fehlschlägt: zum jeweiligen Task
zurückgehen, dort nachbessern und dort committen (nicht hier sammeln).

---

## Self-Review

**Spec-Abdeckung:**
- Cleanup → Task 1 ✓
- doc/+docs/-Merge → Task 2 ✓
- src/-Verzeichnis → Task 3 ✓
- Doku-Generator (Mensch: `docs/referenz-generiert.md`, KI:
  `docs/ki/referenz.md`) → Task 5 ✓ (Task 4 liefert die dafür nötige
  Grundlage `(env-symbols)`, die die Spec voraussetzt, aber real noch
  nicht existierte — beim Schreiben dieses Plans entdeckt)
- Testen/Build → Task 6 ✓ (zusätzlich Build/Test-Gates in jedem
  Einzeltask)

**Platzhalter-Scan:** keine TBD/TODO-Marker; jeder Code-Schritt enthält
lauffähigen Text. Die eine bewusste Unsicherheit (Step 1 von Task 5:
`string<`/`intern`/`open` könnten anders heißen) ist explizit als
Prüfschritt mit Fallback-Anweisung markiert, kein stiller Platzhalter.

**Typkonsistenz:** `(env-symbols)` durchgängig als 0-Argumente-Primitiv
mit String-Listen-Rückgabe beschrieben (Task 4 Produces-Block, Task 5
Consumes-Block, CLAUDE.md-Ergänzung).
