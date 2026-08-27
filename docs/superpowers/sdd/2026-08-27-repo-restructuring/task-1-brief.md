### Task 1: Cleanup — unused/

**Files:**
- Move: `experiment/` → `unused/experiment/`
- Move: `images/` → `unused/images/`
- Move: `libs/` → `unused/libs/`
- Move: `tools/gen-training/` → `unused/gen-training/`
- Move: `doc/files.zip` → `unused/files.zip`
- Finalize: `PerfTODO.md` → `todos/PerfTODO.md` (bereits im Working Tree
  als `D`/`??` sichtbar — Verschiebung nur noch stagen)
- Modify: `CLAUDE.md` (Doku-Tabelle, PerfTODO-Zeile)

**Korrektur (Task-1-Review, 2026-08-27):** `pn-gps1/` bleibt am Root —
ursprünglich fälschlich als unreferenziert eingestuft, tatsächlich
referenziert von `main.go:103,105` (`-t`-Testsuite) und
`lib/swank/gps_bug_test.go` (`TestSwankSurvivesNorvigBugs`). Gerhards
Ruling: nicht verschieben. Siehe Spec-Korrektur in der Cleanup-Tabelle.

**Interfaces:** keine (reine Dateisystem-Operation, kein Code betroffen)

- [ ] **Step 1: Cleanup-Kandidaten verschieben**

```bash
mkdir -p unused
git mv experiment unused/experiment
git mv images unused/images
git mv libs unused/libs
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
grep -rn "experiment/\|libs/\|tools/gen-training" --include=*.go --include=*.lisp --include=*.md . \
  | grep -v "^./unused/" | grep -v "^./docs/retrospectives/" | grep -v "^./docs/gespraeche/"
```

Erwartung: keine Treffer außerhalb der bewusst ausgenommenen Verzeichnisse
(Retrospektiven sind historisch, siehe Global Constraints). `pn-gps1/`
bleibt bewusst außen vor (siehe Korrektur oben) — dafür stattdessen
verifizieren, dass die bestehende Testreferenz noch funktioniert:

```bash
go test ./lib/swank/ -run TestSwankSurvivesNorvigBugs -v
```

Erwartung: PASS (unverändert, da `pn-gps1/` nicht angefasst wird).

- [ ] **Step 5: Commit**

```bash
git add unused CLAUDE.md
git commit -m "$(cat <<'EOF'
chore(repo): ungenutzte Verzeichnisse nach unused/ verschoben

experiment/, images/, libs/, tools/gen-training/, doc/files.zip
seit Juli 2026 unangetastet und ohne aktive Referenz — siehe
docs/superpowers/specs/2026-08-27-repo-restructuring-design.md.
PerfTODO.md-Verschiebung nach todos/ (Gerhard) mit committet.

Co-Authored-By: claude-sonnet-5 <noreply@anthropic.com>
EOF
)"
```

---

