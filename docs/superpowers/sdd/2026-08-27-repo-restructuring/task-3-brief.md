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

