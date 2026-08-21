//**********************************************************************
//  lib/httpserver_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Sonnet 5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260821
//**********************************************************************
// Tests fuer http-serve: :host-Keyword (Default 127.0.0.1, explizit
// gesetzt, Fehlerfaelle).
//**********************************************************************

package lib

import (
  "net"
  "testing"
)

func TestHTTPServeDefaultHost(t *testing.T) {
  env := BaseEnv()
  srvCell, err := fnHTTPServe(env, []*Cell{MakeNum(0)})
  if err != nil {
    t.Fatalf("http-serve: %v", err)
  }
  ws, err := asServer("http-serve", srvCell)
  if err != nil {
    t.Fatal(err)
  }
  t.Cleanup(func() { fnHTTPStop([]*Cell{srvCell}) }) //nolint:errcheck

  host, _, err := net.SplitHostPort(ws.ln.Addr().String())
  if err != nil {
    t.Fatal(err)
  }
  if host != "127.0.0.1" {
    t.Fatalf("host = %q, erwartet 127.0.0.1", host)
  }
}

func TestHTTPServeCustomHost(t *testing.T) {
  env := BaseEnv()
  args := []*Cell{MakeNum(0), MakeAtom(":host"), MakeStr("127.0.0.1")}
  srvCell, err := fnHTTPServe(env, args)
  if err != nil {
    t.Fatalf("http-serve: %v", err)
  }
  ws, err := asServer("http-serve", srvCell)
  if err != nil {
    t.Fatal(err)
  }
  t.Cleanup(func() { fnHTTPStop([]*Cell{srvCell}) }) //nolint:errcheck

  host, _, err := net.SplitHostPort(ws.ln.Addr().String())
  if err != nil {
    t.Fatal(err)
  }
  if host != "127.0.0.1" {
    t.Fatalf("host = %q, erwartet 127.0.0.1", host)
  }
}

func TestHTTPServeErrors(t *testing.T) {
  env := BaseEnv()

  cases := []struct {
    name string
    args []*Cell
  }{
    {"kein Argument", []*Cell{}},
    {"Port falscher Typ", []*Cell{MakeStr("8083")}},
    {"unbekanntes Keyword", []*Cell{MakeNum(0), MakeAtom(":quatsch"), MakeStr("x")}},
    {":host falscher Typ", []*Cell{MakeNum(0), MakeAtom(":host"), MakeNum(1)}},
  }
  for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) {
      if _, err := fnHTTPServe(env, tc.args); err == nil {
        t.Fatalf("%s: erwartet Fehler, bekam keinen", tc.name)
      }
    })
  }
}
