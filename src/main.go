//**********************************************************************
//  main.go  - GoLisp REPL
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260224
//**********************************************************************

package main

import (
  "os"

  "golisp2/src/cli"
)

func main() {
  os.Exit(cli.Run(os.Args[1:]))
}
