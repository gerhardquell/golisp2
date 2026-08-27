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

