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
  "os"
  "path/filepath"
  "strings"
  "testing"
)

func TestHTTPServeDefaultHost(t *testing.T) {
  env := BaseEnv()
  srvCell, err := fnHTTPServe(env, []*Cell{MakeInt(0)})
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
  args := []*Cell{MakeInt(0), MakeAtom(":host"), MakeStr("127.0.0.1")}
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
  args := []*Cell{MakeInt(0), MakeAtom(":host"), MakeStr("127.0.0.1"),
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
    {"unbekanntes Keyword", []*Cell{MakeInt(0), MakeAtom(":quatsch"), MakeStr("x")}},
    {":host falscher Typ", []*Cell{MakeInt(0), MakeAtom(":host"), MakeNum(1)}},
  }
  for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) {
      if _, err := fnHTTPServe(env, tc.args); err == nil {
        t.Fatalf("%s: erwartet Fehler, bekam keinen", tc.name)
      }
    })
  }
}

func TestHTTPServeIstForeign(t *testing.T) {
  if _, err := fnHTTPPort([]*Cell{keinHandle}); err == nil ||
    err.Error() != "http-port: kein Server-Objekt" {
    t.Errorf("http-port mit Liste: err = %v", err)
  }
  srv, err := fnHTTPServe(BaseEnv(), []*Cell{MakeInt(0)})
  if err != nil {
    t.Fatalf("http-serve: %v", err)
  }
  t.Cleanup(func() { fnHTTPStop([]*Cell{srv}) }) //nolint:errcheck
  pruefeHandle(t, srv, "webserver")
}

// staticServer startet einen Server mit http-static "/s" auf ein Temp-
// Verzeichnis mit a.js; extra sind die Argumente nach dir.
func staticServer(t *testing.T, extra ...*Cell) (string, error) {
  t.Helper()
  dir := t.TempDir()
  if err := os.WriteFile(filepath.Join(dir, "a.js"), []byte("x"), 0o644); err != nil {
    t.Fatal(err)
  }
  srv, err := fnHTTPServe(BaseEnv(), []*Cell{MakeInt(0)})
  if err != nil {
    t.Fatalf("http-serve: %v", err)
  }
  t.Cleanup(func() { fnHTTPStop([]*Cell{srv}) }) //nolint:errcheck
  ws, _ := asServer("http-serve", srv)
  args := append([]*Cell{srv, MakeStr("/s"), MakeStr(dir)}, extra...)
  _, err = fnHTTPStatic(args)
  return "http://" + ws.ln.Addr().String() + "/s/", err
}

func TestHTTPStaticHeaders(t *testing.T) {
  hdrs := List(List(MakeStr("Cache-Control"), MakeStr("no-cache")))
  base, err := staticServer(t, MakeAtom(":headers"), hdrs)
  if err != nil {
    t.Fatalf("http-static: %v", err)
  }
  for _, p := range []string{"a.js", "fehlt.js"} {
    resp, err := http.Get(base + p)
    if err != nil {
      t.Fatal(err)
    }
    resp.Body.Close() //nolint:errcheck
    if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
      t.Errorf("%s: Cache-Control = %q, erwartet no-cache", p, got)
    }
    if p == "a.js" && resp.Header.Get("Last-Modified") == "" {
      t.Errorf("Last-Modified fehlt")
    }
  }
}

func TestHTTPStaticOhneHeaders(t *testing.T) {
  base, err := staticServer(t)
  if err != nil {
    t.Fatalf("http-static: %v", err)
  }
  resp, err := http.Get(base + "a.js")
  if err != nil {
    t.Fatal(err)
  }
  resp.Body.Close() //nolint:errcheck
  if got := resp.Header.Get("Cache-Control"); got != "" {
    t.Errorf("Cache-Control = %q, erwartet leer", got)
  }
}

func TestHTTPStaticHeadersFehler(t *testing.T) {
  s := MakeStr
  kw := MakeAtom(":headers")
  cases := []struct {
    name string
    args []*Cell
  }{
    {"keine Liste", []*Cell{kw, s("Cache-Control")}},
    {"Punktpaar", []*Cell{kw, Cons(List(s("A"), s("b")), s("x"))}},
    {"Eintrag mit 1 Element", []*Cell{kw, List(List(s("A")))}},
    {"Eintrag mit 3 Elementen", []*Cell{kw, List(List(s("A"), s("b"), s("c")))}},
    {"Eintrag kein Liste", []*Cell{kw, List(s("A"))}},
    {"Nicht-String", []*Cell{kw, List(List(s("A"), MakeInt(1)))}},
    {"Name mit Leerzeichen", []*Cell{kw, List(List(s("Cache Control"), s("x")))}},
    {"Name mit Doppelpunkt", []*Cell{kw, List(List(s("X:Y"), s("x")))}},
    {"leerer Name", []*Cell{kw, List(List(s(""), s("x")))}},
    {"Wert mit CRLF", []*Cell{kw, List(List(s("X"), s("a\r\nSet-Cookie: b")))}},
    {"Wert mit NUL", []*Cell{kw, List(List(s("X"), s("a\x00")))}},
    {"unbekanntes Keyword", []*Cell{MakeAtom(":quatsch"), s("x")}},
    {"Keyword ohne Wert", []*Cell{kw}},
    {"kein Keyword", []*Cell{s(":headers"), List()}},
  }
  for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) {
      _, err := staticServer(t, tc.args...)
      if err == nil || !strings.HasPrefix(err.Error(), "http-static: ") {
        t.Fatalf("erwartet http-static-Fehler, bekam %v", err)
      }
    })
  }
}
