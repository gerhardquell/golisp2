//**********************************************************************
//  cmd/golispd/main.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260301 (refactored 20260618)
//**********************************************************************
// GoLisp Server (Daemon) - SWANK TCP-Server
//**********************************************************************

package main

import (
  "flag"
  "fmt"
  "os"

  "golisp2/lib/swank"
)

func main() {
  // Flags parsen
  var (
    host = flag.String("host", "localhost", "Host für den Server (default: localhost)")
    port = flag.String("port", "4321", "Port für den Server (default: 4321)")
  )
  flag.Parse()

  // Umgebungsvariablen prüfen (haben Vorrang vor Flags)
  if envHost := os.Getenv("GOLISP_HOST"); envHost != "" {
    *host = envHost
  }
  if envPort := os.Getenv("GOLISP_PORT"); envPort != "" {
    *port = envPort
  }

  addr := *host + ":" + *port
  if err := swank.RunServer(addr); err != nil {
    fmt.Fprintf(os.Stderr, "Fehler: %v\n", err)
    os.Exit(1)
  }
}
