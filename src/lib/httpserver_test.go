//**********************************************************************
//  lib/httpserver_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Sonnet 5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260821
//**********************************************************************
// Tests fuer http-serve: :host-Keyword (Default 127.0.0.1, explizit
// gesetzt, Fehlerfaelle), :tls-Keyword (TLS-Handshake, Klartext-http
// unveraendert ohne).
//**********************************************************************

package lib

import (
  "crypto/tls"
  "net"
  "net/http"
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

func TestHTTPServeTLS(t *testing.T) {
  env := BaseEnv()
  args := []*Cell{MakeNum(0), MakeAtom(":host"), MakeStr("127.0.0.1"),
    MakeAtom(":tls"), MakeAtom("t")}
  srvCell, err := fnHTTPServe(env, args)
  if err != nil {
    t.Fatalf("http-serve: %v", err)
  }
  ws, err := asServer("http-serve", srvCell)
  if err != nil {
    t.Fatal(err)
  }
  t.Cleanup(func() { fnHTTPStop([]*Cell{srvCell}) }) //nolint:errcheck

  // InsecureSkipVerify hier bewusst: Test gegen den eigenen, gerade erst
  // ephemer erzeugten Prozess -- kein Netzwerk, kein MITM-Risiko. Prueft
  // nur, ob der TLS-Handshake ueberhaupt zustande kommt. Die eigentliche
  // Vertrauensentscheidung fuer selbstsignierte Zertifikate liegt beim
  // Client (golisp2web netGuard-Whitelist), nicht hier.
  addr := ws.ln.Addr().String()
  client := &http.Client{Transport: &http.Transport{
    TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
  }}
  resp, err := client.Get("https://" + addr + "/")
  if err != nil {
    t.Fatalf("https-Request fehlgeschlagen (TLS-Handshake?): %v", err)
  }
  resp.Body.Close() //nolint:errcheck

  // net/http erkennt Klartext-Bytes gegen einen TLS-Listener und
  // antwortet mit einer eingebauten 400-Antwort statt die Verbindung
  // zu droppen -- err bleibt nil, StatusCode ist der Signal.
  plainClient := &http.Client{Timeout: 2 * 1e9}
  plainResp, err := plainClient.Get("http://" + addr + "/")
  if err != nil {
    t.Fatalf("unerwarteter Fehler bei Klartext-Request: %v", err)
  }
  defer plainResp.Body.Close() //nolint:errcheck
  if plainResp.StatusCode != http.StatusBadRequest {
    t.Fatalf("Klartext-http gegen TLS-Listener: StatusCode = %d, erwartet 400",
      plainResp.StatusCode)
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
