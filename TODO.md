# TODO - Aufgabenplanung 20260709

## Erledigt 20260709 — Rename golisp → golisp2

- [x] Ausführbare Programme: `golisp`→`golisp2`, `golispd`→`golisp2d`,
      `golisp-client`→`golisp2-client` (build.sh, .gitignore, Go-Kommentare,
      alle .md-Dokumente).
- [x] cmd-Dirs: `cmd/golispd`→`cmd/golisp2d`, `cmd/golisp-client`→`cmd/golisp2-client`.
- [x] `golisp2d` ist der SWANK-Server (`swank.RunServer`), `golisp2` der
      Standalone-Prozess, `golisp2-client` der Client-Prozess.
- [x] REPL-Prompt `golisp2> `.
- [x] Env-Variablen `GOLISP_HOST/PORT/SIGO_*` bewusst beibehalten (kein Breaking-Change).
- [x] Build + Smoke-Test grün (`./build.sh`, `golisp2 -e`, SWANK-Frame-Handshake).

## Offen — Folgeaufgaben aus dem Rename

- [x] **golisp2-client auf SWANK umgestellt** (20260709). Client spricht jetzt
      das echte SWANK-Protokoll (`:emacs-rex` length-prefixed), nicht mehr das
      alte Custom-RPC. Map: ping→`connection-info`, eval→`listener-eval`,
      complete→`simple-completions`, load→`load-file`. Nutzt `golisp2/lib` für
      robuste Cell-Verarbeitung statt String-Slicing. Smoke-Test + 126 Tests grün.
- [ ] **CLAUDE.md-Doku-Drift**: Abschnitt "GoLisp Server (golisp2d)" + 
      "Protokoll (S-Expression-RPC)" beschreiben noch das alte Custom-RPC
      (`(:id … :method …)`), obwohl `golisp2d` SWANK spricht und der Client
      jetzt SWANK nutzt. Doku an SWANK-Realität anpassen.
- [ ] **UX-Frage (präexistent)**: Env-Vars `GOLISP_HOST/PORT` schlagen
      explizite `--host`/`--port`-Flags vor (Server wie Client). Ggf. Flag
      priorisieren über Env — separates Refactor.
