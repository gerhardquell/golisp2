# GoLisp2 – Dateistruktur

Referenz. Bei Widerspruch zum Code gewinnt der Code.

```
golisp2/
  build/                 Build-Artefakte (via ./build.sh)
    golisp2              Hauptbinary: CLI + REPL + --swank-Modus (SWANK-Server)
    golisp2-client       CLI-Client mit REPL
  build.sh               baut alle Binaries nach ./build/
  tmp/                   temporäres Verzeichnis des Projekts (nicht /tmp !)

  tools/
    gen-reference.lisp   Generator für docs/referenz-generiert.md
                         (env-symbols + *ref-docs*-Aliste — Beschreibungen
                         hier pflegen, nicht im generierten Markdown!)

  src/
    main.go              Unix-Style CLI: stdin / -i / -e / -t / --swank / Datei + Exit-Codes
    main_test.go         Go-Tests für die CLI

    cmd/
      golisp2-client/
        main.go          CLI-Client mit REPL (spricht SWANK, nutzt golisp2/src/lib)

    embed/               //go:embed Assets — zentraler Ort für Lisp-Quellen
      assets.go          Embed-Deklarationen (trennt Assets vom Go-Code)
      stdlib.lisp        Standardbibliothek (defstruct, defgeneric/defmethod,
                         setf, reduce, assoc/List-Helfer …)
      condition.lisp     Condition-lite: define-condition, signal,
                         handler-case, ignore-errors …
      defsystem.lisp     defsystem/load-system/unload-system (topologisch,
                         mit Zyklenerkennung)
      swank.lisp         SWANK-Semantik-Handler (connection-info,
                         listener-eval …)

    lib/
      types.go           Cell-Datenstruktur (LispType, Cons, MakeAtom, internTable)
                         + Small-Int-Cache; Primary() — MV-Regel lebt hier
      types_helpers.go   SliceToCell, Append, CellToSlice, IsTruthy
      types_test.go      …
      reader.go          Parser: String → Cell-Baum (NewReader, Read, ReadAll)
      env.go             Environment: Get, Set, Update, Symbols (verkettete
                         Scopes, RWMutex); Set-Hook für Redefine-Guard

      eval_core.go       Eval-Trampolin, apply, evalArgs (TCO-Invariante!)
      eval_lambda.go     Lambda/Closure-Logik: makeLambda, applyLambda,
                         bindArgs (Multi-Body via wrapBegin)
      eval_specialforms.go  Spezialformen (nicht-tail): define/setq, defun,
                         lambda, defmacro, set!/setq*, begin, load, flet, labels
      eval_control.go    Control-Flow + Nebenläufigkeit: while, do, prog1/2,
                         block/return-from, catch, eval, parfunc, lock, if,
                         cond, case, let/let* (Tail-Formen → TCO)
      eval_mv.go         Multiple Values: multiple-value-list/-bind/-call/
                         -prog1/-setq; Produktion via (values …)
      eval_quasiquote.go quasiquote / unquote / unquote-splicing
      eval_load.go       load-Spezialform + Source-Locations (SrcFile/SrcLine)
      eval_exec.go       exec-Spezialform (Subprocess: stdout/stderr/exitcd/
                         stdin-Variablen, env-Keys)

      primitives.go      Eingebaute Funktionen + BaseEnv() (Chokepoint:
                         Neues immer hier registrieren)
      stringfuncs.go     String-Primitiven (RegisterStringFuncs)
      hashtable.go       CL-Hashtables: make-hash-table, gethash (MV!),
                         puthash, maphash, remhash, clrhash …
      clcompat_prims.go  sort (Lisp-Callback als Go-Prädikat), sqrt,
                         get-universal-time — Lispbuch-Lücken (TODO 20260818 B)
      cformat.go         C-artige E/A: printf, sprintf, fprintf, sscanf
      format.go          FORMAT-Engine (CL-HyperSpec 22.3): fnFormat,
                         formatRun, Parameter-Parser
      format_dirs.go     FORMAT-Direktiven + Helper
      format_blocks.go   FORMAT-Block-Direktiven (~{ ~[ ~/ …)
      output.go          Zentraler Output-Handler (SWANK lenkt print/println/
                         format-Ausgaben nach Emacs um)
      docstring.go       Docstring-Registry für (documentation 'name 'function)
      trace.go           trace, untrace, trace? (Live-Tracing von Funktionen)
      sysinfo.go         argv, getenv, environ
      defloc.go          Definition-Locations für M-. (SLIME)

      redefguard.go      Redefine-Policy: allow/warn/error für Root-Redefinitionen
      redeflog.go        redef-log: Ringpuffer aller Root-Redefinitionen +
                         makunbound-Events (Beobachtbarkeit statt Verbot)

      goroutine.go       parfunc, chan-make/send/recv, lock-make
      fileio.go          Datei-I/O: file-write/-append/-read/-exists?/-delete
      shellcmd.go        system, file-stat, shell-assoc, symbol->string
      postgres.go        PostgreSQL-Primitiven
      maxima.go          maxima-open/-eval/-close — CAS via externen
                         Maxima-Prozess, Sentinel-Sync statt Prompt-Parsing
      genalg.go          Genetischer Algorithmus (Core)
      genalg_prims.go    GA-Lisp-Primitiven
      shm_lisp.go        Shared-Memory-Primitiven (High-Level Pool-API)
      sigorest.go        sigo, sigo-models, sigo-host — HTTP zu sigoREST
                         (Chokepoint: einziger HTTP-Client gegen :9080)

      httpserver.go      Web-Bridge: http-serve (:host, :tls), http-static,
                         http-upload, http-port/-wait/-stop, browser-open
      tlscert.go         Ephemere selbstsignierte TLS-Zertifikate für :tls
      wsbridge.go        Web-Bridge: ws-export/-emit/-call u. a., WS-Hub,
                         boot.js-Embed
      webserv.go         Web-Bridge: webserv (Ein-Aufruf-Bootstrap:
                         Server+HTML+boot.js+Browser; :host seit 20260821)
      jsoncell.go        JSON ↔ Cell (CellToJSON/JSONToCell)
      embed/boot.js      Browser-Client der Web-Bridge (golisp.call/on/…)

      stdlib.go          //go:embed + LoadStdlib (Chokepoint: eine Quelle)
      readline.go        REPL: go-prompt, Syntax-Highlighting, History, Multiline

      swank/             SWANK-Server für Emacs/SLIME
        server.go        TCP-Listener, per-Connection Handling, Env-Setup
        framing.go       Length-prefixed Framing (readFrame/writeFrame)
        dispatch.go      (swank-dispatch msg) Wrapper
        env.go           per-Connection Primitiven (send-event, value-string)
        lisp.go          //go:embed swank.lisp, LoadSwankLisp
        swank.lisp       Semantische Handler (connection-info, listener-eval …)

      *_test.go          Go-Tests pro Modul (Charakterisierung + Unit);
                         specialform_shadow_test.go bewacht Spezialform-Namen
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
