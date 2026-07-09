//**********************************************************************
//  cmd/golisp2d/main.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260301 (refactored 20260618, renamed 20260709)
//**********************************************************************
// GoLisp2 Server (Daemon) - SWANK TCP-Server
//**********************************************************************

package main

import (
  "flag"
  "fmt"
  "os"

  "golisp2/lib/swank"
)

func main() {
  // Env-Variablen dienen als Flag-Default; ein expliziter --host/--port
  // gewinnt über die Env (Unix-Konvention: Flag > Env > Default).
  hostDefault := os.Getenv("GOLISP_HOST")
  if hostDefault == "" {
    hostDefault = "localhost"
  }
  portDefault := os.Getenv("GOLISP_PORT")
  if portDefault == "" {
    portDefault = "4321"
  }

  host := flag.String("host", hostDefault, "Host für den Server (env: GOLISP_HOST, default: localhost)")
  port := flag.String("port", portDefault, "Port für den Server (env: GOLISP_PORT, default: 4321)")
  flag.Parse()

  addr := *host + ":" + *port
  if err := swank.RunServer(addr); err != nil {
    fmt.Fprintf(os.Stderr, "Fehler: %v\n", err)
    os.Exit(1)
  }
}
