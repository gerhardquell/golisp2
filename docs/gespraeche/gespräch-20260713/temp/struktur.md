# GoLisp2 – Dateistruktur

Referenz. Bei Widerspruch zum Code gewinnt der Code.

```
golisp2/
  build/                 Build-Artefakte (via ./build.sh)
    golisp2              Hauptbinary: CLI + REPL + --swank-Modus
    golisp2d             Server-Daemon (SWANK)
    golisp2-client       CLI-Client mit REPL
  main.go                Unix-Style CLI: stdin / -i / -e / -t / --swank / Datei + Exit-Codes
  build.sh               baut alle drei Binaries nach ./build/
  tmp/                   temporäres Verzeichnis des Projekts (nicht /tmp !)

  cmd/
    golisp2d/
      main.go            SWANK-Server Entry Point
    golisp2-client/
      main.go            CLI-Client mit REPL (spricht SWANK, nutzt golisp2/lib)

  lib/
    types.go             Cell-Datenstruktur (LispType, Cons, MakeAtom …) + Small-Int-Cache
    types_helpers.go     SliceToCell, Append, CellToSlice, IsTruthy
    reader.go            Parser: String → Cell-Baum (NewReader, Read, ReadAll)
    env.go               Environment: Get, Set, Update, Symbols (verkettete Scopes, RWMutex)
    env_test.go          Go-Tests für Env.Symbols()

    eval_core.go         Eval-Trampolin, apply, evalArgs
    eval_lambda.go       Lambda/Closure-Aufrufe
    eval_specialforms.go quote, if, define, defun, lambda, let*, set!, defmacro, mapcar …
    eval_control.go      while, do, catch, cond, case
    eval_quasiquote.go   quasiquote / unquote / unquote-splicing
    eval_load.go         load-Spezialform + Source-Locations (SrcFile/SrcLine)
    eval_exec.go         exec-Spezialform

    primitives.go        Eingebaute Funktionen + BaseEnv()
    stringfuncs.go       String-Primitiven (RegisterStringFuncs)
    format.go            FORMAT-Engine (fnFormat, formatRun, Parameter-Parser)
    format_dirs.go       FORMAT-Direktiven + Helper
    format_blocks.go     FORMAT-Block-Direktiven
    output.go            print/println Rückgabewerte

    goroutine.go         parfunc, chan-make/send/recv, lock-make
    fileio.go            file-write, file-append, file-read, file-exists?, file-delete
    shellcmd.go          system, file-stat, assoc, symbol->string
    postgres.go          PostgreSQL-Primitiven
    genalg.go            Genetischer Algorithmus (Core)
    genalg_prims.go      GA-Lisp-Primitiven
    shm_lisp.go          Shared-Memory-Primitiven
    sigorest.go          sigo, sigo-models, sigo-host (HTTP zu sigoREST)

    defloc.go            Definition-Locations für M-. (SLIME)
    stdlib.go            //go:embed stdlib.lisp + LoadStdlib
    readline.go          REPL: go-prompt, Syntax-Highlighting, History, Multiline

    swank/               SWANK-Server für Emacs/SLIME
      server.go          TCP-Listener, per-Connection Handling, Env-Setup
      framing.go         Length-prefixed Framing (readFrame/writeFrame)
      dispatch.go        (swank-dispatch msg) Wrapper
      env.go             per-Connection Primitiven (send-event, value-string)
      lisp.go            //go:embed swank.lisp, LoadSwankLisp
      swank.lisp         Semantische Handler (connection-info, listener-eval …)
```

## Cell – die Grundstruktur

```go
type Cell struct {
  Type LispType        // ATOM, NUMBER, STRING, LIST, FUNC, MACRO, NIL
  Val  string          // für ATOM und STRING
  Num  float64         // für NUMBER
  Car  *Cell           // Kopf einer Liste
  Cdr  *Cell           // Rest einer Liste
  Fn   func([]*Cell) (*Cell, error)  // für FUNC
  Env  interface{}     // für Lambda-Closures (*Env) und Go-Objekte
}
```

Lambda/Closure ist eine `Cell{Type: LIST, Car: params, Cdr: body, Env: closureEnv}`.
Makro identisch, aber `Type: MACRO`.
