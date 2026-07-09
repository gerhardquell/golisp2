# golisp2 Transfer & Struktur-Optimierung Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Aktuellen Stand von `golisp2-kimi` nach `/u/lisp-projekte/golisp2` übertragen (einfrieren), Git-Historie erhalten, Layout optimieren und Builds/Tests grün halten.

**Architecture:** `golisp2` entsteht als Klon von `golisp2-kimi` inklusive Working-Tree (uncommitted Änderungen + untracked Dateien). Danach werden überflüssige Dateien entfernt, eingebettete Lisp-Assets in `embed/` gebündelt, alte TODOs archiviert, das Go-Modul in `golisp2` umbenannt und alle Importpfade angepasst.

**Tech Stack:** Go 1.26.0, `github.com/elk-language/go-prompt`, git, bash, rsync.

## Global Constraints
- Go-Version: `1.26.0` (aus `go.mod`).
- Build-Output: `./build/` via `./build.sh`.
- Temp-Verzeichnis: immer `./tmp`, niemals `/tmp`.
- Modulname nach Transfer: `golisp2`.
- Imports interner Pakete: `golisp2/lib`, `golisp2/lib/swank`.
- Kein manuelles Memory-Management; Go-GC bleibt.
- Binärnamen bleiben `golisp2`, `golisp2d`, `golisp2-client`.
- Ursprungsrepo `/u/lisp-projekte/golisp2-kimi` nach dem Freeze nur noch lesend bearbeiten (Tag `frozen-2026-07-09`).

## Vorabentscheidung (vor Task 1 vom Menschen zu bestätigen)

Empfohlenes Ziel-Layout für `golisp2`:

```
golisp2/
  go.mod
  main.go
  .gitignore
  build.sh
  CLAUDE.md
  README.md
  cmd/
    golisp2d/main.go
    golisp2-client/main.go
  lib/                     # Go-Interpreter-Paket
    swank/                 # SWANK-Unterpaket bleibt hier
  embed/                   # Eingebettete Lisp-Dateien
    stdlib.lisp
    swank.lisp
  docs/
    archive/               # Alte TODOs, Retrospektiven, Duplikate
    superpowers/plans/     # Dieser Plan
  examples/
  tests/
  tools/
  tutorial/
  chinese/
  tmp/                     # Runtime-Temp (gitignored)
  build/                   # Build-Output (gitignored)
```

Zu entfernende/verschiebende Dateien (siehe Task 3):
- `lib/primitives.go.old`, `lib/types.go.old`
- `lib/cpu.prof`, `lib/mem.prof`, `lib/grep.txt`
- `lib/format_blocks.go`, `lib/format_dirs.go`, `lib/format_test.go` löschen (nicht eingebundene WIP-Refactor-Dateien)
- `TODO.md.2026*-done` und `TODO.md.20260709-later` → `docs/archive/todo/`
- Doppelte `docs/CLAUDE.md` prüfen und ggf. nach `docs/archive/` verschieben

Wenn ein anderes Layout gewünscht ist: Task 3 anpassen **bevor** Task 1 ausgeführt wird.

---

### Task 1: Klones golisp2-kimi mit Working-Tree nach golisp2

**Files:**
- Create: `/u/lisp-projekte/golisp2/` (gesamtes Repo)
- Modify: keine

**Interfaces:**
- Consumes: `/u/lisp-projekte/golisp2-kimi/`
- Produces: `/u/lisp-projekte/golisp2/` mit identischem Working-Tree und vollständiger Git-Historie

- [ ] **Step 1: Leeres Zielverzeichnis prüfen/löschen**

```bash
if [ -e /u/lisp-projekte/golisp2 ]; then
  echo "FEHLER: /u/lisp-projekte/golisp2 existiert bereits."
  exit 1
fi
```

- [ ] **Step 2: Git-Klon mit Historie erstellen**

```bash
git clone /u/lisp-projekte/golisp2-kimi /u/lisp-projekte/golisp2
```

Expected: `Cloning into '/u/lisp-projekte/golisp2'... done.`

- [ ] **Step 3: Working-Tree exakt übertragen (inkl. uncommitted Änderungen und untracked Dateien)**

```bash
cd /u/lisp-projekte/golisp2-kimi
rsync -av --delete \
  --exclude='.git' \
  --exclude='build/' \
  --exclude='tmp/' \
  --exclude='.codegraph/' \
  --exclude='.superpowers/' \
  --exclude='.claude/' \
  ./ /u/lisp-projekte/golisp2/
```

Expected: Dateien werden kopiert; `.git/`, `build/`, `tmp/` bleiben unberührt.

- [ ] **Step 4: Unterschiede prüfen**

```bash
diff -rq /u/lisp-projekte/golisp2-kimi /u/lisp-projekte/golisp2 \
  -x .git -x build -x tmp -x .codegraph -x .superpowers -x .claude
```

Expected: keine Ausgabe (oder nur bekannte Symlinks/Worktrees).

- [ ] **Step 5: Freeze-Commit im neuen Repo**

```bash
cd /u/lisp-projekte/golisp2
git add -A
git commit -m "chore: freeze golisp2-kimi state as golisp2 base

Co-Authored-By: Claude <noreply@anthropic.com>"
```

Expected: Commit-Hash wird ausgegeben.

---

### Task 2: Modul in golisp2 umbenennen

**Files:**
- Modify: `/u/lisp-projekte/golisp2/go.mod`
- Modify: `/u/lisp-projekte/golisp2/main.go`
- Modify: `/u/lisp-projekte/golisp2/cmd/golisp2d/main.go`
- Modify: `/u/lisp-projekte/golisp2/cmd/golisp2-client/main.go`
- Modify: alle `.go` Dateien, die `golisp2/lib` importieren

**Interfaces:**
- Consumes: aktuelle Imports wie `"golisp2/lib"`, `"golisp2/lib/swank"`
- Produces: Imports werden zu `"golisp2/lib"`, `"golisp2/lib/swank"`

- [ ] **Step 1: go.mod ändern**

In `go.mod`:

```go
module golisp2
```

- [ ] **Step 2: Alle Go-Imports ersetzen**

```bash
cd /u/lisp-projekte/golisp2
grep -Rln '"golisp2/lib' --include='*.go' . | xargs -I{} sed -i 's/"golisp2\/lib/"golisp2\/lib/g' {}
```

Verify:

```bash
grep -Rn '"golisp2/lib' --include='*.go' .
```

Expected: keine Treffer.

- [ ] **Step 3: go mod tidy**

```bash
cd /u/lisp-projekte/golisp2
go mod tidy
```

Expected: `go.mod` bleibt stabil, `go.sum` ggf. minimal angepasst.

- [ ] **Step 4: Build testen**

```bash
cd /u/lisp-projekte/golisp2
./build.sh
```

Expected: `golisp2`, `golisp2d`, `golisp2-client` in `build/`.

- [ ] **Step 5: Commit**

```bash
cd /u/lisp-projekte/golisp2
git add -A
git commit -m "refactor: rename module to golisp2

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 3: Verzeichnisstruktur optimieren

**Files:**
- Create: `/u/lisp-projekte/golisp2/embed/`
- Create: `/u/lisp-projekte/golisp2/wip/format/`
- Create: `/u/lisp-projekte/golisp2/docs/archive/todo/`
- Modify: `/u/lisp-projekte/golisp2/lib/stdlib.go`
- Modify: `/u/lisp-projekte/golisp2/lib/swank/lisp.go`
- Delete/Verschieben: siehe unten

**Interfaces:**
- Consumes: `//go:embed` Pfade in `lib/stdlib.go` und `lib/swank/lisp.go`
- Produces: `embed/stdlib.lisp`, `embed/swank.lisp`, konsolidierte Docs

- [ ] **Step 1: Embedded Lisp-Dateien nach `embed/` verschieben**

```bash
cd /u/lisp-projekte/golisp2
mkdir -p embed
git mv lib/stdlib.lisp embed/stdlib.lisp
git mv lib/swank/swank.lisp embed/swank.lisp
```

- [ ] **Step 2: Embed-Pfade anpassen**

In `lib/stdlib.go`:

```go
//go:embed ../embed/stdlib.lisp
var stdlibLisp string
```

In `lib/swank/lisp.go`:

```go
//go:embed ../../embed/swank.lisp
var swankLisp string
```

Verify:

```bash
go build ./...
```

Expected: Build erfolgreich.

- [ ] **Step 3: Obsolete Dateien entfernen**

```bash
cd /u/lisp-projekte/golisp2
git rm lib/primitives.go.old lib/types.go.old
git rm lib/cpu.prof lib/mem.prof lib/grep.txt
```

- [ ] **Step 4: WIP-Format-Dateien löschen**

```bash
cd /u/lisp-projekte/golisp2
git rm lib/format_blocks.go lib/format_dirs.go lib/format_test.go
```

- [ ] **Step 5: Alte TODOs archivieren**

```bash
cd /u/lisp-projekte/golisp2
mkdir -p docs/archive/todo
for f in TODO.md.20260618-done TODO.md.20260621-done TODO.md.20260623-done \
         TODO.md.20260623a-done TODO.md.20260624-done TODO.md.20260626-done \
         TODO.md.20260709-later TODO.md.liste; do
  if [ -f "$f" ]; then git mv "$f" docs/archive/todo/; fi
done
```

- [ ] **Step 6: Doppelte Docs prüfen**

```bash
diff /u/lisp-projekte/golisp2/CLAUDE.md /u/lisp-projekte/golisp2/docs/CLAUDE.md
```

Wenn identisch oder veraltet:

```bash
cd /u/lisp-projekte/golisp2
mkdir -p docs/archive
git mv docs/CLAUDE.md docs/archive/CLAUDE-dup.md
```

- [ ] **Step 7: Build & Tests nach Strukturänderung**

```bash
cd /u/lisp-projekte/golisp2
./build.sh
go test ./...
```

Expected: alle Tests grün, Binaries in `build/`.

- [ ] **Step 8: Commit**

```bash
cd /u/lisp-projekte/golisp2
git add -A
git commit -m "refactor: optimize directory layout

- move embedded lisp assets to embed/
- archive old TODOs and WIP format files
- remove obsolete .old and profile files

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 4: Ursprungsrepo golisp2-kimi als frozen taggen

**Files:**
- Modify: `/u/lisp-projekte/golisp2-kimi/.git` (Tag)

**Interfaces:**
- Consumes: `/u/lisp-projekte/golisp2-kimi/`
- Produces: Tag `frozen-2026-07-09`

- [ ] **Step 1: Tag setzen**

```bash
cd /u/lisp-projekte/golisp2-kimi
git tag -a frozen-2026-07-09 -m "Freeze point before golisp2 transfer"
```

- [ ] **Step 2: Tag anzeigen**

```bash
git log -1 frozen-2026-07-09
```

Expected: Tag zeigt aktuellen HEAD.

---

### Task 5: Finale Verifikation

**Files:**
- Verify: `/u/lisp-projekte/golisp2/`

- [ ] **Step 1: Vollständigen Build laufen lassen**

```bash
cd /u/lisp-projekte/golisp2
./build.sh
```

- [ ] **Step 2: Test-Suite laufen lassen**

```bash
cd /u/lisp-projekte/golisp2
go test ./...
```

Expected: `ok` für alle Pakete.

- [ ] **Step 3: CLI-Smoke-Test**

```bash
cd /u/lisp-projekte/golisp2
echo '(+ 1 2 3)' | ./build/golisp2
```

Expected: `6`.

- [ ] **Step 4: Git-Status prüfen**

```bash
cd /u/lisp-projekte/golisp2
git status
```

Expected: `nothing to commit, working tree clean`.

---

## Self-Review

1. **Spec coverage:**
   - Transfer aktuellen Stand: Task 1 (rsync inkl. uncommitted/untracked).
   - Verzeichnisstruktur optimieren: Task 3 (Layout, Archivierung, embed/).
   - Vorschläge machen: Plan enthält empfohlenes Layout + Alternativenhinweis.
2. **Placeholder scan:** keine TBD/TODO/"implement later".
3. **Type consistency:** keine neuen Go-Typen; Imports konsistent `golisp2/lib`.

Gaps: Entscheidung über WIP-Format-Dateien (Task 3 Step 4) hängt vom Menschen ab; Plan markiert beide Optionen.
