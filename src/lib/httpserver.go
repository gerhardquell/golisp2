//**********************************************************************
//  lib/httpserver.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260807
//**********************************************************************
// HTTP-Server-Primitiven fuer die Web-Bridge (Spec TODO.md §3.1).
// Bindet standardmaessig an 127.0.0.1, per :host konfigurierbar.
// WebSocket-Teil: wsbridge.go.
//**********************************************************************

package lib

import (
  "context"
  "crypto/tls"
  "fmt"
  "io"
  "net"
  "net/http"
  "os"
  "os/exec"
  "os/signal"
  "path/filepath"
  "strings"
  "sync"
  "syscall"
  "time"
)

// WebServer haelt den Zustand eines http-serve-Servers (Spec §2).
type WebServer struct {
  mu       sync.RWMutex
  srv      *http.Server
  mux      *http.ServeMux
  ln       net.Listener
  port     int
  handlers map[string]*Cell   // ws-export
  clients  map[int]*wsClient  // wsbridge.go
  nextCid  int
  pending  map[int]chan wsCallResult // ws-call, Key = call-id (Spec: chan *Cell — erweitert um Fehler-Transport)
  nextCall int
  env      *Env               // Root-Env fuer Handler-Aufrufe
  done     chan struct{}
  stopOnce sync.Once
  idleAt   time.Time
}

// RegisterHTTPFuncs registriert die HTTP-Primitiven der Web-Bridge.
func RegisterHTTPFuncs(env *Env) {
  _ = env.Set("http-serve",   makeFn(func(args []*Cell) (*Cell, error) { return fnHTTPServe(env, args) }))
  _ = env.Set("http-static",  makeFn(fnHTTPStatic))
  _ = env.Set("http-upload",  makeFn(fnHTTPUpload))
  _ = env.Set("http-port",    makeFn(fnHTTPPort))
  _ = env.Set("http-wait",    makeFn(fnHTTPWait))
  _ = env.Set("http-stop",    makeFn(fnHTTPStop))
  _ = env.Set("browser-open", makeFn(fnBrowserOpen))
}

// asServer prueft, ob c ein gueltiges Server-Objekt ist.
func asServer(name string, c *Cell) (*WebServer, error) {
  obj, _ := ForeignObj(c, "webserver")
  ws, ok := obj.(*WebServer)
  if !ok {
    return nil, fmt.Errorf("%s: kein Server-Objekt", name)
  }
  return ws, nil
}

// handle registriert eine Route. ServeMux meldet Kollisionen per Panic —
// hier als Lisp-Fehler mit Funktionsnamen statt Go-Quellpfaden.
func (ws *WebServer) handle(name, pattern string, h http.Handler) (err error) {
  defer func() {
    if r := recover(); r != nil {
      if strings.Contains(fmt.Sprint(r), "conflicts") {
        err = fmt.Errorf("%s: Route %q ist schon belegt", name, pattern)
      } else {
        err = fmt.Errorf("%s: Route %q: %v", name, pattern, r)
      }
    }
  }()
  ws.mux.Handle(pattern, h)
  return nil
}

// http-serve: (http-serve port &key host tls) → Server-Cell. port=0 →
// freier Port vom OS. :host Default "127.0.0.1". :tls t bindet mit einem
// ephemeren, selbstsignierten Zertifikat (SAN: host + localhost/127.0.0.1,
// 24h gueltig, nur im Speicher) — Vertrauensentscheidung liegt beim
// Client. Startet Goroutine, kehrt sofort zurueck.
func fnHTTPServe(env *Env, args []*Cell) (*Cell, error) {
  if len(args) < 1 || args[0].Type != NUMBER {
    return nil, fmt.Errorf("http-serve: Port (NUMBER) erwartet")
  }
  host := "127.0.0.1"
  useTLS := false
  for i := 1; i+1 < len(args); i += 2 {
    if args[i].Type != ATOM {
      return nil, fmt.Errorf("http-serve: Keyword erwartet, bekam %s", args[i])
    }
    switch args[i].Val {
    case ":host":
      if args[i+1].Type != STRING {
        return nil, fmt.Errorf("http-serve: :host muss STRING sein")
      }
      host = args[i+1].Val
    case ":tls":
      useTLS = IsTruthy(args[i+1])
    default:
      return nil, fmt.Errorf("http-serve: unbekanntes Keyword %s", args[i])
    }
  }
  port, err := intArg("http-serve", args[0])
  if err != nil {
    return nil, err
  }
  ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
  if err != nil {
    return nil, fmt.Errorf("http-serve: %v", err)
  }
  if useTLS {
    cert, err := generateSelfSignedCert(host)
    if err != nil {
      return nil, fmt.Errorf("http-serve: %v", err)
    }
    ln = tls.NewListener(ln, &tls.Config{Certificates: []tls.Certificate{cert}})
  }
  ws := &WebServer{
    mux:      http.NewServeMux(),
    ln:       ln,
    port:     ln.Addr().(*net.TCPAddr).Port,
    handlers: make(map[string]*Cell),
    clients:  make(map[int]*wsClient),
    pending:  make(map[int]chan wsCallResult),
    env:      env,
    done:     make(chan struct{}),
    idleAt:   time.Now(),
  }
  ws.srv = &http.Server{Handler: ws.mux}
  ws.registerWSRoute()
  go ws.srv.Serve(ln) //nolint:errcheck // Fehler landen im done-Kanal via http-stop
  return MakeForeign("webserver", ws), nil
}

// http-static: (http-static srv urlpath dir &key headers) → t. Mountet dir
// unter urlpath, mehrfach aufrufbar. Kein Directory-Listing, keine Symlinks.
// :headers '(("Cache-Control" "no-cache") ...) setzt diese Antwort-Header
// bei jeder Antwort der Route (auch 304/404); Go-eigene Header wie
// Content-Type/Last-Modified bleiben, sofern nicht gleichnamig überschrieben.
func fnHTTPStatic(args []*Cell) (*Cell, error) {
  if len(args) < 3 {
    return nil, fmt.Errorf("http-static: 3 Argumente (srv urlpath dir) nötig")
  }
  ws, err := asServer("http-static", args[0])
  if err != nil {
    return nil, err
  }
  if args[1].Type != STRING || args[2].Type != STRING {
    return nil, fmt.Errorf("http-static: urlpath und dir müssen Strings sein")
  }
  urlpath, dir := args[1].Val, args[2].Val
  if !strings.HasPrefix(urlpath, "/") {
    return nil, fmt.Errorf("http-static: urlpath muss mit / beginnen")
  }
  if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
    return nil, fmt.Errorf("http-static: '%s' ist kein Verzeichnis", dir)
  }
  var headers [][2]string
  if len(args)%2 == 0 {
    return nil, fmt.Errorf("http-static: Keyword %s ohne Wert", args[len(args)-1])
  }
  for i := 3; i+1 < len(args); i += 2 {
    if args[i].Type != ATOM {
      return nil, fmt.Errorf("http-static: Keyword erwartet, bekam %s", args[i])
    }
    switch args[i].Val {
    case ":headers":
      if headers, err = staticHeaders(args[i+1]); err != nil {
        return nil, err
      }
    default:
      return nil, fmt.Errorf("http-static: unbekanntes Keyword %s", args[i])
    }
  }
  pattern := urlpath
  if pattern != "/" && !strings.HasSuffix(pattern, "/") {
    pattern += "/"
  }
  prefix := strings.TrimSuffix(pattern, "/") // "/" → ""
  if err := ws.handle("http-static", pattern, http.StripPrefix(prefix, staticHandler(dir, headers))); err != nil {
    return nil, err
  }
  return cellT, nil
}

const maxUploadBytes = 32 << 20 // 32 MB

// http-upload: (http-upload srv urlpath handler) → t. Registriert eine
// POST-only-Route unter urlpath (multipart/form-data). Fuer jede
// hochgeladene Datei wird (handler filename content) aufgerufen — content
// als STRING mit rohen Bytes (wie file-read). Erst nach allen Handler-
// Aufrufen antwortet der Server 200 "ok"; schlaegt ein Handler fehl oder
// ist das Formular ungueltig, antwortet er mit einem Fehlercode und dem
// Fehlertext, ohne weitere Dateien zu verarbeiten.
func fnHTTPUpload(args []*Cell) (*Cell, error) {
  if len(args) < 3 {
    return nil, fmt.Errorf("http-upload: 3 Argumente (srv urlpath handler) nötig")
  }
  ws, err := asServer("http-upload", args[0])
  if err != nil {
    return nil, err
  }
  if args[1].Type != STRING {
    return nil, fmt.Errorf("http-upload: urlpath muss STRING sein")
  }
  urlpath := args[1].Val
  if !strings.HasPrefix(urlpath, "/") {
    return nil, fmt.Errorf("http-upload: urlpath muss mit / beginnen")
  }
  handler := args[2]
  if handler.Type != LAMBDA && handler.Type != FUNC {
    return nil, fmt.Errorf("http-upload: handler muss Lambda sein")
  }

  upload := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
      http.Error(w, "http-upload: nur POST erlaubt", http.StatusMethodNotAllowed)
      return
    }
    r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
    if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
      http.Error(w, fmt.Sprintf("http-upload: %v", err), http.StatusBadRequest)
      return
    }
    defer r.MultipartForm.RemoveAll() //nolint:errcheck

    for _, headers := range r.MultipartForm.File {
      for _, fh := range headers {
        f, err := fh.Open()
        if err != nil {
          http.Error(w, fmt.Sprintf("http-upload: %v", err), http.StatusInternalServerError)
          return
        }
        data, err := io.ReadAll(f)
        f.Close() //nolint:errcheck
        if err != nil {
          http.Error(w, fmt.Sprintf("http-upload: %v", err), http.StatusInternalServerError)
          return
        }
        if _, err := apply(handler, []*Cell{MakeStr(fh.Filename), MakeStr(string(data))}); err != nil {
          http.Error(w, fmt.Sprintf("http-upload: %v", err), http.StatusInternalServerError)
          return
        }
      }
    }
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.Write([]byte("ok")) //nolint:errcheck
  })
  if err := ws.handle("http-upload", urlpath, upload); err != nil {
    return nil, err
  }
  return cellT, nil
}

// staticHeaders prüft den :headers-Wert von http-static: echte Liste von
// (name wert)-Paaren aus Strings, name ein HTTP-Token, wert ohne CR/LF/NUL.
func staticHeaders(c *Cell) ([][2]string, error) {
  items, err := properList("http-static", c)
  if err != nil {
    return nil, fmt.Errorf("http-static: :headers muss Liste von (name wert) sein, bekam %s", c)
  }
  out := make([][2]string, 0, len(items))
  for _, it := range items {
    pair, err := properList("http-static", it)
    if err != nil || len(pair) != 2 || pair[0].Type != STRING || pair[1].Type != STRING {
      return nil, fmt.Errorf("http-static: :headers-Eintrag muss (name wert) aus zwei Strings sein, bekam %s", it)
    }
    name, val := pair[0].Val, pair[1].Val
    if !isHTTPToken(name) {
      return nil, fmt.Errorf("http-static: ungültiger Header-Name %q", name)
    }
    if strings.ContainsAny(val, "\r\n\x00") {
      return nil, fmt.Errorf("http-static: Header-Wert von %s enthält CR, LF oder NUL", name)
    }
    out = append(out, [2]string{name, val})
  }
  return out, nil
}

// isHTTPToken: RFC 9110 token (tchar) — eigene Prüfung statt x/net/httpguts.
func isHTTPToken(s string) bool {
  if s == "" {
    return false
  }
  for _, r := range s {
    ok := r < 0x80 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' ||
      r >= '0' && r <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", r))
    if !ok {
      return false
    }
  }
  return true
}

// staticHandler liefert Dateien aus root: kein Directory-Listing (fehlendes
// index.html → 404), keine Symlinks, Path-Traversal-sicher. headers werden
// vor jeder Antwort gesetzt.
func staticHandler(root string, headers [][2]string) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    for _, h := range headers {
      w.Header().Set(h[0], h[1])
    }
    // filepath.Clean mit fuehrendem Slash: ".." kann nicht entkommen.
    clean := filepath.Clean("/" + r.URL.Path)
    if pathHasSymlink(root, clean) {
      http.NotFound(w, r)
      return
    }
    full := filepath.Join(root, clean)
    fi, err := os.Stat(full)
    if err != nil {
      http.NotFound(w, r)
      return
    }
    if fi.IsDir() {
      full = filepath.Join(full, "index.html")
      if fi, err = os.Stat(full); err != nil || fi.IsDir() {
        http.NotFound(w, r)
        return
      }
    }
    f, err := os.Open(full)
    if err != nil {
      http.NotFound(w, r)
      return
    }
    defer f.Close()
    http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
  })
}

// pathHasSymlink prueft jede Komponente von root+clean per Lstat.
func pathHasSymlink(root, clean string) bool {
  cur := root
  for _, part := range strings.Split(strings.TrimPrefix(clean, "/"), "/") {
    if part == "" {
      continue
    }
    cur = filepath.Join(cur, part)
    fi, err := os.Lstat(cur)
    if err != nil {
      return false // nicht vorhanden → unten normaler 404
    }
    if fi.Mode()&os.ModeSymlink != 0 {
      return true
    }
  }
  return false
}

// http-port: (http-port srv) → tatsaechlicher Port (NUMBER).
func fnHTTPPort(args []*Cell) (*Cell, error) {
  if len(args) < 1 {
    return nil, fmt.Errorf("http-port: 1 Argument nötig")
  }
  ws, err := asServer("http-port", args[0])
  if err != nil {
    return nil, err
  }
  return MakeInt(int64(ws.port)), nil
}

// http-wait: (http-wait srv &key idle-exit) → nil. Blockiert bis http-stop
// oder — mit idle-exit (ms) — bis so lange kein Client mehr verbunden war.
// Der Timer laeuft ab Serverstart, wenn nie ein Client kam.
// Bei SIGINT/SIGTERM beendet sich der Prozess (os.Exit) statt nur den Call
// zu entblocken: signal.Notify wirkt prozessweit — ein blockierendes
// http-wait im SWANK-Daemon (golisp2 -swank) wuerde sonst das
// Shutdown-Signal des ganzen Prozesses schlucken, bis systemd nach
// TimeoutStopSec hart mit SIGKILL nachhilft.
func fnHTTPWait(args []*Cell) (*Cell, error) {
  if len(args) < 1 {
    return nil, fmt.Errorf("http-wait: 1 Argument nötig")
  }
  ws, err := asServer("http-wait", args[0])
  if err != nil {
    return nil, err
  }
  var idle time.Duration
  for i := 1; i+1 < len(args); i += 2 {
    if args[i].Type != ATOM || args[i].Val != ":idle-exit" {
      return nil, fmt.Errorf("http-wait: unbekanntes Keyword %s", args[i])
    }
    if args[i+1].Type != NUMBER {
      return nil, fmt.Errorf("http-wait: :idle-exit muss NUMBER (ms) sein")
    }
    idle = time.Duration(args[i+1].Num) * time.Millisecond
  }
  sigCh := make(chan os.Signal, 1)
  signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
  defer signal.Stop(sigCh)
  if idle <= 0 {
    select {
    case <-ws.done:
    case <-sigCh:
      os.Exit(0)
    }
    return MakeNil(), nil
  }
  tick := time.NewTicker(50 * time.Millisecond)
  defer tick.Stop()
  for {
    select {
    case <-ws.done:
      return MakeNil(), nil
    case <-sigCh:
      os.Exit(0)
    case <-tick.C:
      ws.mu.RLock()
      idleSince := time.Since(ws.idleAt)
      noClients := len(ws.clients) == 0
      ws.mu.RUnlock()
      if noClients && idleSince >= idle {
        return MakeNil(), nil
      }
    }
  }
}

// http-stop: (http-stop srv) → t. Graceful Shutdown (2 s), idempotent.
func fnHTTPStop(args []*Cell) (*Cell, error) {
  if len(args) < 1 {
    return nil, fmt.Errorf("http-stop: 1 Argument nötig")
  }
  ws, err := asServer("http-stop", args[0])
  if err != nil {
    return nil, err
  }
  ws.stopOnce.Do(func() {
    close(ws.done)
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    ws.srv.Shutdown(ctx) //nolint:errcheck // best effort
  })
  return cellT, nil
}

// browser-open: (browser-open url) → t, oder nil wenn kein Browser gefunden. Sucht chromium,
// chromium-browser, google-chrome(-stable), xdg-open — in dieser Reihenfolge.
// Chromium-Varianten bekommen --app=URL und ein frisches Profil.
// Prozess detached, kein Warten.
func fnBrowserOpen(args []*Cell) (*Cell, error) {
  if len(args) < 1 || args[0].Type != STRING {
    return nil, fmt.Errorf("browser-open: URL (String) erwartet")
  }
  url := args[0].Val
  chromium := []string{"chromium", "chromium-browser", "google-chrome-stable", "google-chrome"}
  for _, name := range chromium {
    if path, err := exec.LookPath(name); err == nil {
      profile, err := os.MkdirTemp("", "golisp2-browser")
      if err != nil {
        return nil, fmt.Errorf("browser-open: %v", err)
      }
      cmd := exec.Command(path, "--app="+url, "--user-data-dir="+profile)
      if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("browser-open: %v", err)
      }
      go cmd.Wait() //nolint:errcheck // reapt den Zombie, blockiert nicht
      return cellT, nil
    }
  }
  if path, err := exec.LookPath("xdg-open"); err == nil {
    cmd := exec.Command(path, url)
    if err := cmd.Start(); err != nil {
      return nil, fmt.Errorf("browser-open: %v", err)
    }
    go cmd.Wait() //nolint:errcheck
    return cellT, nil
  }
  return MakeNil(), nil
}
