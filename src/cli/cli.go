//**********************************************************************
//  cli/cli.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261001
//**********************************************************************
// Kommandozeile von golisp2 als Bibliothek: Run(args, ext...) liefert
// den Exit-Code. Extensions erweitern die Env vor dem Modus (gogui).
//**********************************************************************

package cli

import (
  "errors"
  "flag"
  "fmt"
  "io"
  "os"
  "path/filepath"
  "strings"

  "golisp2/src/lib"
  "golisp2/src/lib/swank"
)

// Extension ergänzt die Env nach BaseEnv und LoadStdlib.
type Extension func(env *lib.Env) error

// Run führt golisp2 mit args (ohne Programmnamen) aus und liefert den
// Exit-Code: 0 ok, 1 Fehler, 2 ungültige Flags.
//
// --swank startet RunServer: jede Verbindung bekommt eine eigene, frische
// Env (BaseEnv + LoadStdlib) — ext wird dort NICHT angewendet, Extensions
// sind in diesem Modus also unsichtbar. Für eine geteilte Env mit
// Extensions siehe swank.ServeEnv/RunServerEnv (gogui).
func Run(args []string, ext ...Extension) int {
  return run(args, os.Stdout, os.Stderr, ext...)
}

func run(args []string, out, errOut io.Writer, ext ...Extension) int {
  fs := flag.NewFlagSet(filepath.Base(os.Args[0]), flag.ContinueOnError)
  fs.SetOutput(errOut)
  interactiveFlag := fs.Bool("i", false, "Interaktiver REPL-Modus")
  exprFlag := fs.String("e", "", "Expression direkt ausführen")
  testFlag := fs.Bool("t", false, "Tests ausführen")
  swankFlag := fs.String("swank", "", "SWANK-Server starten (Format: host:port, z.B. 127.0.0.1:4005)")
  if err := fs.Parse(args); err != nil {
    if errors.Is(err, flag.ErrHelp) {
      return 0
    }
    return 2
  }

  env := lib.BaseEnv()
  if err := lib.LoadStdlib(env); err != nil {
    fmt.Fprintln(errOut, "stdlib Fehler:", err)
    return 1
  }
  for _, e := range ext {
    if err := e(env); err != nil {
      fmt.Fprintln(errOut, "Erweiterung Fehler:", err)
      return 1
    }
  }

  if *swankFlag != "" {
    addr := *swankFlag
    if !strings.Contains(addr, ":") {
      addr = "127.0.0.1:" + addr
    }
    if err := swank.RunServer(addr); err != nil {
      fmt.Fprintln(errOut, "swank server error:", err)
      return 1
    }
    return 0
  }

  if *testFlag {
    runTests(env)
    return 0
  }

  if *exprFlag != "" {
    return runExpression(*exprFlag, env, out, errOut)
  }

  if *interactiveFlag {
    return runREPL(env)
  }

  if fs.NArg() > 0 {
    exitCode, result, hasMain, err := lib.RunScript(fs.Arg(0), fs.Args()[1:], env)
    if err != nil {
      var le *lib.LispError
      if errors.As(err, &le) {
        fmt.Fprintln(errOut, "ERR:", le.Msg)
      } else {
        fmt.Fprintln(errOut, "ERR:", err)
      }
      return 1
    }
    if !hasMain {
      fmt.Fprintln(out, result)
    }
    return exitCode
  }

  return runStdin(env)
}
