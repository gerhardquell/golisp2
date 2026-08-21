//**********************************************************************
//  lib/webserv.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Sonnet 5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260808
//**********************************************************************
// webserv: Ein-Aufruf-Bootstrap fuer die Web-Bridge (Spec
// docs/superpowers/specs/2026-08-08-webserv-design.md). Fasst
// http-serve + Content-Serving + boot.js-Injektion + browser-open zu
// einem Primitiv zusammen. ws-export/-emit/-call u.a. bleiben unveraendert
// nutzbar auf dem zurueckgegebenen Server-Objekt — kein zweiter
// Server-Aufbau-Pfad, webserv ruft intern fnHTTPServe.
//**********************************************************************

package lib

import (
  "fmt"
  "net/http"
  "os"
  "regexp"
  "strings"
)

const bootScriptTag = `<script src="/_golisp/boot.js"></script>`

var headCloseRe = regexp.MustCompile(`(?i)</head\s*>`)

// RegisterWebservFuncs registriert das webserv-Primitiv.
func RegisterWebservFuncs(env *Env) {
  _ = env.Set("webserv", makeFn(func(args []*Cell) (*Cell, error) { return fnWebServ(env, args) }))
}

// fnWebServ: (webserv &key port host html htmlpath open) → Server-Cell.
// Genau eines von :html/:htmlpath ist Pflicht. :port Default 0 (freier
// Port). :host Default "127.0.0.1". :open Default t (Browser automatisch
// oeffnen).
func fnWebServ(env *Env, args []*Cell) (*Cell, error) {
  if len(args)%2 != 0 {
    return nil, fmt.Errorf("webserv: gerade Anzahl Argumente (Keyword-Paare) erwartet")
  }

  port := 0.0
  host := "127.0.0.1"
  var html, htmlpath string
  haveHTML, haveHTMLPath := false, false
  open := true

  for i := 0; i+1 < len(args); i += 2 {
    if args[i].Type != ATOM {
      return nil, fmt.Errorf("webserv: Keyword erwartet, bekam %s", args[i])
    }
    switch args[i].Val {
    case ":port":
      if args[i+1].Type != NUMBER {
        return nil, fmt.Errorf("webserv: :port muss NUMBER sein")
      }
      port = args[i+1].Num
    case ":host":
      if args[i+1].Type != STRING {
        return nil, fmt.Errorf("webserv: :host muss STRING sein")
      }
      host = args[i+1].Val
    case ":html":
      if args[i+1].Type != STRING {
        return nil, fmt.Errorf("webserv: :html muss STRING sein")
      }
      html = args[i+1].Val
      haveHTML = true
    case ":htmlpath":
      if args[i+1].Type != STRING {
        return nil, fmt.Errorf("webserv: :htmlpath muss STRING sein")
      }
      htmlpath = args[i+1].Val
      haveHTMLPath = true
    case ":open":
      open = IsTruthy(args[i+1])
    default:
      return nil, fmt.Errorf("webserv: unbekanntes Keyword %s", args[i].Val)
    }
  }

  if haveHTML == haveHTMLPath {
    if haveHTML {
      return nil, fmt.Errorf("webserv: :html und :htmlpath schliessen sich aus")
    }
    return nil, fmt.Errorf("webserv: :html oder :htmlpath erforderlich")
  }

  srvCell, err := fnHTTPServe(env, []*Cell{MakeNum(port), MakeAtom(":host"), MakeStr(host)})
  if err != nil {
    return nil, err
  }
  ws, err := asServer("webserv", srvCell)
  if err != nil {
    return nil, err
  }

  if haveHTML {
    body := injectBootScript(html)
    ws.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
      w.Header().Set("Content-Type", "text/html; charset=utf-8")
      w.Write([]byte(body)) //nolint:errcheck
    })
  } else {
    path := htmlpath
    ws.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
      resolved, err := resolvePath(path)
      if err != nil {
        http.NotFound(w, r)
        return
      }
      data, err := os.ReadFile(resolved)
      if err != nil {
        http.NotFound(w, r)
        return
      }
      w.Header().Set("Content-Type", "text/html; charset=utf-8")
      w.Write([]byte(injectBootScript(string(data)))) //nolint:errcheck
    })
  }

  if open {
    url := fmt.Sprintf("http://%s:%d/", host, ws.port)
    _, _ = fnBrowserOpen([]*Cell{MakeStr(url)})
  }

  return srvCell, nil
}

// injectBootScript fuegt den boot.js-Script-Tag ein, falls er im HTML
// noch fehlt — vor </head> (case-insensitiv, optionale Whitespace vor
// dem Schluss-Tag), sonst am Dokumentanfang.
func injectBootScript(html string) string {
  if strings.Contains(html, "/_golisp/boot.js") {
    return html
  }
  if loc := headCloseRe.FindStringIndex(html); loc != nil {
    return html[:loc[0]] + bootScriptTag + html[loc[0]:]
  }
  return bootScriptTag + html
}
