//**********************************************************************
//  lib/webserv_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Sonnet 5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260808
//**********************************************************************
// Tests fuer webserv: Inline-HTML, Datei-Modus mit frischem Reload pro
// Request, boot.js-Injektion, Fehlerfaelle. :open wird in jedem Test auf
// nil gesetzt, damit kein echter Browser gestartet wird.
//**********************************************************************

package lib

import (
  "io"
  "net/http"
  "os"
  "path/filepath"
  "strconv"
  "strings"
  "testing"
)

func webservTestEnv(t *testing.T) *Env {
  t.Helper()
  return BaseEnv()
}

func fetchBody(t *testing.T, url string) (int, string) {
  t.Helper()
  resp, err := http.Get(url)
  if err != nil {
    t.Fatalf("GET %s: %v", url, err)
  }
  defer resp.Body.Close()
  body, err := io.ReadAll(resp.Body)
  if err != nil {
    t.Fatalf("ReadAll: %v", err)
  }
  return resp.StatusCode, string(body)
}

func TestWebServInlineHTML(t *testing.T) {
  env := webservTestEnv(t)
  args := []*Cell{
    MakeAtom(":html"), MakeStr("<html><head></head><body>hallo</body></html>"),
    MakeAtom(":open"), MakeNil(),
  }
  srvCell, err := fnWebServ(env, args)
  if err != nil {
    t.Fatalf("webserv: %v", err)
  }
  ws, err := asServer("webserv", srvCell)
  if err != nil {
    t.Fatal(err)
  }
  t.Cleanup(func() { fnHTTPStop([]*Cell{srvCell}) }) //nolint:errcheck

  status, body := fetchBody(t, "http://127.0.0.1:"+strconv.Itoa(ws.port)+"/")
  if status != 200 {
    t.Fatalf("status = %d, erwartet 200", status)
  }
  if !strings.Contains(body, "hallo") {
    t.Fatalf("body fehlt Inhalt: %q", body)
  }
  if !strings.Contains(body, `/_golisp/boot.js`) {
    t.Fatalf("boot.js-Tag fehlt: %q", body)
  }
  if !strings.Contains(body, "</head>") {
    t.Fatalf("Injektion hat </head> zerstoert: %q", body)
  }
  if idx := strings.Index(body, "/_golisp/boot.js"); idx > strings.Index(body, "</head>") {
    t.Fatalf("boot.js-Tag steht nach </head>, erwartet davor: %q", body)
  }
}

func TestWebServInlineHTMLAlreadyHasBootJS(t *testing.T) {
  env := webservTestEnv(t)
  original := `<html><head><script src="/_golisp/boot.js"></script></head><body>x</body></html>`
  args := []*Cell{
    MakeAtom(":html"), MakeStr(original),
    MakeAtom(":open"), MakeNil(),
  }
  srvCell, err := fnWebServ(env, args)
  if err != nil {
    t.Fatal(err)
  }
  ws, _ := asServer("webserv", srvCell)
  t.Cleanup(func() { fnHTTPStop([]*Cell{srvCell}) }) //nolint:errcheck

  _, body := fetchBody(t, "http://127.0.0.1:"+strconv.Itoa(ws.port)+"/")
  if strings.Count(body, "/_golisp/boot.js") != 1 {
    t.Fatalf("boot.js-Tag doppelt eingefuegt: %q", body)
  }
}

func TestWebServHTMLPathFreshReload(t *testing.T) {
  dir := t.TempDir()
  path := filepath.Join(dir, "seite.html")
  if err := os.WriteFile(path, []byte("<html><head></head><body>eins</body></html>"), 0o644); err != nil {
    t.Fatal(err)
  }

  env := webservTestEnv(t)
  args := []*Cell{
    MakeAtom(":htmlpath"), MakeStr(path),
    MakeAtom(":open"), MakeNil(),
  }
  srvCell, err := fnWebServ(env, args)
  if err != nil {
    t.Fatal(err)
  }
  ws, _ := asServer("webserv", srvCell)
  t.Cleanup(func() { fnHTTPStop([]*Cell{srvCell}) }) //nolint:errcheck

  url := "http://127.0.0.1:" + strconv.Itoa(ws.port) + "/"
  _, body := fetchBody(t, url)
  if !strings.Contains(body, "eins") {
    t.Fatalf("erster Request: %q, erwartet 'eins'", body)
  }

  if err := os.WriteFile(path, []byte("<html><head></head><body>zwei</body></html>"), 0o644); err != nil {
    t.Fatal(err)
  }
  _, body = fetchBody(t, url)
  if !strings.Contains(body, "zwei") {
    t.Fatalf("zweiter Request nach Datei-Aenderung: %q, erwartet 'zwei' (kein Caching)", body)
  }
}

func TestWebServHTMLPathAlreadyHasBootJS(t *testing.T) {
  dir := t.TempDir()
  path := filepath.Join(dir, "seite.html")
  original := `<html><head><script src="/_golisp/boot.js"></script></head><body>x</body></html>`
  if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
    t.Fatal(err)
  }

  env := webservTestEnv(t)
  args := []*Cell{
    MakeAtom(":htmlpath"), MakeStr(path),
    MakeAtom(":open"), MakeNil(),
  }
  srvCell, err := fnWebServ(env, args)
  if err != nil {
    t.Fatal(err)
  }
  ws, _ := asServer("webserv", srvCell)
  t.Cleanup(func() { fnHTTPStop([]*Cell{srvCell}) }) //nolint:errcheck

  _, body := fetchBody(t, "http://127.0.0.1:"+strconv.Itoa(ws.port)+"/")
  if strings.Count(body, "/_golisp/boot.js") != 1 {
    t.Fatalf("boot.js-Tag doppelt eingefuegt (htmlpath-Modus): %q", body)
  }
}

func TestWebServHTMLPathMissing404(t *testing.T) {
  env := webservTestEnv(t)
  args := []*Cell{
    MakeAtom(":htmlpath"), MakeStr("/pfad/gibts/nicht.html"),
    MakeAtom(":open"), MakeNil(),
  }
  srvCell, err := fnWebServ(env, args)
  if err != nil {
    t.Fatalf("webserv-Aufruf selbst darf nicht fehlschlagen, auch wenn Datei fehlt: %v", err)
  }
  ws, _ := asServer("webserv", srvCell)
  t.Cleanup(func() { fnHTTPStop([]*Cell{srvCell}) }) //nolint:errcheck

  status, _ := fetchBody(t, "http://127.0.0.1:"+strconv.Itoa(ws.port)+"/")
  if status != 404 {
    t.Fatalf("status = %d, erwartet 404", status)
  }
}

func TestWebServErrors(t *testing.T) {
  env := webservTestEnv(t)

  cases := []struct {
    name string
    args []*Cell
  }{
    {"weder html noch htmlpath", []*Cell{MakeAtom(":open"), MakeNil()}},
    {"beide gesetzt", []*Cell{
      MakeAtom(":html"), MakeStr("<html></html>"),
      MakeAtom(":htmlpath"), MakeStr("/tmp/x.html"),
    }},
    {"unbekanntes Keyword", []*Cell{MakeAtom(":quatsch"), MakeStr("x")}},
    {"ungerade Argumentzahl", []*Cell{MakeAtom(":html")}},
    {":port falscher Typ", []*Cell{
      MakeAtom(":port"), MakeStr("8083"),
      MakeAtom(":html"), MakeStr("<html></html>"),
    }},
  }
  for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) {
      if _, err := fnWebServ(env, tc.args); err == nil {
        t.Fatalf("%s: erwartet Fehler, bekam keinen", tc.name)
      }
    })
  }
}
