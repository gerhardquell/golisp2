//**********************************************************************
//  lib/wsbridge.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260807
//**********************************************************************
// WebSocket-Bridge der Web-Bridge (Spec TODO.md §3.2, §4, §5).
// Jede eingehende op-Nachricht laeuft in einer eigenen Goroutine — kein
// globaler Eval-Mutex. Schreibzugriff pro Client nur ueber eine eigene
// Writer-Goroutine mit gepuffertem Channel (gorilla erlaubt keine
// nebenläufigen Writes). Voller Channel → Nachricht verwerfen + warn,
// nie blockieren.
//**********************************************************************

package lib

import (
  "encoding/json"
  "fmt"
  "net/http"
  "net/url"
  "os"
  "time"

  "github.com/gorilla/websocket"
)

// wsClient ist eine verbundene WebSocket-Verbindung.
type wsClient struct {
  id      int
  conn    *websocket.Conn
  send    chan []byte        // Kapazitaet 64, einziger Schreibweg
  callIDs map[int]struct{}   // offene ws-call-IDs dieses Clients
}

// wsCallResult transportiert Wert ODER Fehler an einen wartenden ws-call.
type wsCallResult struct {
  val *Cell
  err error
}

// wsInMsg: Browser → Lisp. Zwei Formen: op-Request und call-Antwort.
type wsInMsg struct {
  ID   *int            `json:"id"`
  Op   string          `json:"op"`
  Args json.RawMessage `json:"args"`
  Call *int            `json:"call"`
  OK   json.RawMessage `json:"ok"`
  Err  *string         `json:"err"`
}

// wsOutMsg: Lisp → Browser. Je nach gesetzten Feldern Antwort, Event,
// ws-call- oder ws-eval-Nachricht.
type wsOutMsg struct {
  ID    int             `json:"id,omitempty"`
  OK    json.RawMessage `json:"ok,omitempty"`
  Err   string          `json:"err,omitempty"`
  Event string          `json:"event,omitempty"`
  Data  json.RawMessage `json:"data,omitempty"`
  Call  int             `json:"call,omitempty"`
  JS    string          `json:"js,omitempty"`
}

// RegisterWSFuncs registriert die WebSocket-Primitiven der Web-Bridge.
func RegisterWSFuncs(env *Env) {
  _ = env.Set("ws-export",   makeFn(fnWSExport))
  _ = env.Set("ws-unexport", makeFn(fnWSUnexport))
  _ = env.Set("ws-emit",     makeFn(fnWSEmit))
  _ = env.Set("ws-emit-to",  makeFn(fnWSEmitTo))
  _ = env.Set("ws-eval",     makeFn(fnWSEval))
  _ = env.Set("ws-call",     makeFn(fnWSCall))
  _ = env.Set("ws-clients",  makeFn(fnWSClients))
}

// registerWSRoute haengt den WebSocket-Endpunkt an den Server-Mux.
// Wird von fnHTTPServe aufgerufen.
func (ws *WebServer) registerWSRoute() {
  ws.mux.HandleFunc("/_golisp/ws", ws.handleWS)
}

// ---- Verbindungs-Hub ----

// handleWS nimmt eine WebSocket-Verbindung entgegen (Origin-Check nach
// Spec §9) und betreibt deren Reader-Loop, bis sie abbricht.
func (ws *WebServer) handleWS(w http.ResponseWriter, r *http.Request) {
  up := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
    return ws.originOK(r)
  }}
  conn, err := up.Upgrade(w, r, nil)
  if err != nil {
    return // Upgrade hat bereits geantwortet (z. B. 403)
  }
  ws.mu.Lock()
  ws.nextCid++
  client := &wsClient{
    id:      ws.nextCid,
    conn:    conn,
    send:    make(chan []byte, 64),
    callIDs: make(map[int]struct{}),
  }
  ws.clients[client.id] = client
  ws.mu.Unlock()

  go client.writer()
  defer ws.removeClient(client)

  for {
    _, data, err := conn.ReadMessage()
    if err != nil {
      return
    }
    var msg wsInMsg
    if err := json.Unmarshal(data, &msg); err != nil {
      fmt.Fprintf(os.Stderr, "warn: wsbridge: kaputtes JSON verworfen: %v\n", err)
      continue // Verbindung offen lassen (Spec §4)
    }
    switch {
    case msg.Call != nil:
      ws.deliverCallResult(client, &msg)
    case msg.ID != nil && msg.Op != "":
      // Jede op-Nachricht in eigener Goroutine (Spec §5) — ein
      // blockierender Handler (sigo!) darf andere Clients nicht bremsen.
      go ws.handleOp(client, msg.ID, msg.Op, msg.Args)
    default:
      fmt.Fprintf(os.Stderr, "warn: wsbridge: Nachricht ohne id/call verworfen\n")
    }
  }
}

// originOK: erlaubt sind http(s)://127.0.0.1:<port> und localhost:<port>;
// fehlender Origin-Header (Nicht-Browser) → erlaubt (Spec §9).
func (ws *WebServer) originOK(r *http.Request) bool {
  origin := r.Header.Get("Origin")
  if origin == "" {
    return true
  }
  u, err := url.Parse(origin)
  if err != nil {
    return false
  }
  if u.Scheme != "http" && u.Scheme != "https" {
    return false
  }
  host := u.Hostname()
  port := u.Port()
  if port == "" {
    port = "80"
  }
  return (host == "127.0.0.1" || host == "localhost") && port == fmt.Sprintf("%d", ws.port)
}

// writer ist die einzige schreibende Goroutine pro Client.
func (c *wsClient) writer() {
  for msg := range c.send {
    if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
      return
    }
  }
  c.conn.Close()
}

// enqueue wirft eine Nachricht in den Writer-Channel. Voller Channel →
// verwerfen und warnen, nie blockieren (Spec §5). Liefert false bei Drop.
func (c *wsClient) enqueue(msg []byte) bool {
  select {
  case c.send <- msg:
    return true
  default:
    fmt.Fprintf(os.Stderr, "warn: wsbridge: Client %d langsam, Nachricht verworfen\n", c.id)
    return false
  }
}

// removeClient meldet einen Client ab: aus der Map, offene ws-calls mit
// Fehler (nicht Timeout!) beenden, Writer-Channel schliessen, idle-Uhr
// stellen wenn der letzte Client ging.
func (ws *WebServer) removeClient(c *wsClient) {
  ws.mu.Lock()
  if _, ok := ws.clients[c.id]; !ok {
    ws.mu.Unlock()
    return
  }
  delete(ws.clients, c.id)
  for callID := range c.callIDs {
    if ch, ok := ws.pending[callID]; ok {
      ch <- wsCallResult{err: fmt.Errorf("ws-call: Client %d hat die Verbindung getrennt", c.id)}
      delete(ws.pending, callID)
    }
  }
  if len(ws.clients) == 0 {
    ws.idleAt = time.Now()
  }
  ws.mu.Unlock()
  close(c.send)
}

// deliverCallResult rueckt eine Browser-Antwort an den wartenden ws-call.
func (ws *WebServer) deliverCallResult(client *wsClient, msg *wsInMsg) {
  ws.mu.Lock()
  ch, ok := ws.pending[*msg.Call]
  if ok {
    delete(ws.pending, *msg.Call)
    delete(client.callIDs, *msg.Call)
  }
  ws.mu.Unlock()
  if !ok {
    fmt.Fprintf(os.Stderr, "warn: wsbridge: Antwort auf unbekannten call %d\n", *msg.Call)
    return
  }
  if msg.Err != nil {
    ch <- wsCallResult{err: fmt.Errorf("%s", *msg.Err)}
    return
  }
  val, err := JSONToCell(msg.OK)
  if err != nil {
    ch <- wsCallResult{err: err}
    return
  }
  ch <- wsCallResult{val: val}
}

// handleOp ruft einen exportierten Handler auf und antwortet dem Browser.
func (ws *WebServer) handleOp(client *wsClient, id *int, op string, rawArgs json.RawMessage) {
  reply := func(out wsOutMsg) {
    out.ID = *id
    data, err := json.Marshal(out)
    if err != nil {
      fmt.Fprintf(os.Stderr, "warn: wsbridge: Antwort nicht kodierbar: %v\n", err)
      return
    }
    client.enqueue(data)
  }

  ws.mu.RLock()
  handler, ok := ws.handlers[op]
  ws.mu.RUnlock()
  if !ok {
    reply(wsOutMsg{Err: fmt.Sprintf("unbekannte Operation: %s", op)})
    return
  }

  var argCells []*Cell
  if len(rawArgs) > 0 {
    var rawList []json.RawMessage
    if err := json.Unmarshal(rawArgs, &rawList); err != nil {
      reply(wsOutMsg{Err: fmt.Sprintf("%s: args ist kein Array", op)})
      return
    }
    for _, raw := range rawList {
      c, err := JSONToCell(raw)
      if err != nil {
        reply(wsOutMsg{Err: fmt.Sprintf("%s: %v", op, err)})
        return
      }
      argCells = append(argCells, c)
    }
  }

  // Client-ID immer als erstes Argument (Spec §3.2), Aufruf wie funcall.
  result, err := apply(handler, append([]*Cell{MakeNum(float64(client.id))}, argCells...))
  if err != nil {
    reply(wsOutMsg{Err: err.Error()})
    return
  }
  js, err := CellToJSON(Primary(result))
  if err != nil {
    reply(wsOutMsg{Err: err.Error()})
    return
  }
  reply(wsOutMsg{OK: js})
}

// broadcast wirft msg bei allen Clients in den Channel, liefert Anzahl
// erfolgreicher Enqueues.
func (ws *WebServer) broadcast(msg []byte) int {
  ws.mu.RLock()
  clients := make([]*wsClient, 0, len(ws.clients))
  for _, c := range ws.clients {
    clients = append(clients, c)
  }
  ws.mu.RUnlock()
  n := 0
  for _, c := range clients {
    if c.enqueue(msg) {
      n++
    }
  }
  return n
}

// findClient sucht einen Client per ID.
func (ws *WebServer) findClient(id int) *wsClient {
  ws.mu.RLock()
  defer ws.mu.RUnlock()
  return ws.clients[id]
}

// eventName: event/name darf STRING oder ATOM sein (Spec §3.2).
func eventName(fn string, c *Cell) (string, error) {
  if c == nil || (c.Type != STRING && c.Type != ATOM) {
    return "", fmt.Errorf("%s: Name muss String oder Symbol sein", fn)
  }
  return c.Val, nil
}

// ---- Primitiven ----

// ws-export: (ws-export srv name fn) → t. Ueberschreibt still (Reload-
// Semantik), der Client bleibt verbunden — der Kern der Live-Image-Idee.
func fnWSExport(args []*Cell) (*Cell, error) {
  if len(args) < 3 {
    return nil, fmt.Errorf("ws-export: 3 Argumente (srv name fn) nötig")
  }
  ws, err := asServer("ws-export", args[0])
  if err != nil {
    return nil, err
  }
  name, err := eventName("ws-export", args[1])
  if err != nil {
    return nil, err
  }
  if args[2].Type != LAMBDA && args[2].Type != FUNC {
    return nil, fmt.Errorf("ws-export: fn muss Lambda sein")
  }
  ws.mu.Lock()
  ws.handlers[name] = args[2]
  ws.mu.Unlock()
  return cellT, nil
}

// ws-unexport: (ws-unexport srv name) → t, oder nil wenn nicht registriert.
func fnWSUnexport(args []*Cell) (*Cell, error) {
  if len(args) < 2 {
    return nil, fmt.Errorf("ws-unexport: 2 Argumente (srv name) nötig")
  }
  ws, err := asServer("ws-unexport", args[0])
  if err != nil {
    return nil, err
  }
  name, err := eventName("ws-unexport", args[1])
  if err != nil {
    return nil, err
  }
  ws.mu.Lock()
  _, ok := ws.handlers[name]
  delete(ws.handlers, name)
  ws.mu.Unlock()
  if !ok {
    return MakeNil(), nil
  }
  return cellT, nil
}

// wsEmitMsg baut die Event-Nachricht {"event":name,"data":...}.
func wsEmitMsg(name string, data *Cell) ([]byte, error) {
  js, err := CellToJSON(Primary(data))
  if err != nil {
    return nil, err
  }
  return json.Marshal(wsOutMsg{Event: name, Data: js})
}

// ws-emit: (ws-emit srv event data) → Anzahl Empfaenger (Broadcast).
func fnWSEmit(args []*Cell) (*Cell, error) {
  if len(args) < 3 {
    return nil, fmt.Errorf("ws-emit: 3 Argumente (srv event data) nötig")
  }
  ws, err := asServer("ws-emit", args[0])
  if err != nil {
    return nil, err
  }
  name, err := eventName("ws-emit", args[1])
  if err != nil {
    return nil, err
  }
  msg, err := wsEmitMsg(name, args[2])
  if err != nil {
    return nil, fmt.Errorf("ws-emit: %v", err)
  }
  return MakeNum(float64(ws.broadcast(msg))), nil
}

// ws-emit-to: (ws-emit-to srv client event data) → t, nil bei unbekanntem Client.
func fnWSEmitTo(args []*Cell) (*Cell, error) {
  if len(args) < 4 {
    return nil, fmt.Errorf("ws-emit-to: 4 Argumente (srv client event data) nötig")
  }
  ws, err := asServer("ws-emit-to", args[0])
  if err != nil {
    return nil, err
  }
  if args[1].Type != NUMBER {
    return nil, fmt.Errorf("ws-emit-to: Client-ID muss NUMBER sein")
  }
  name, err := eventName("ws-emit-to", args[2])
  if err != nil {
    return nil, err
  }
  client := ws.findClient(int(args[1].Num))
  if client == nil {
    return MakeNil(), nil
  }
  msg, err := wsEmitMsg(name, args[3])
  if err != nil {
    return nil, fmt.Errorf("ws-emit-to: %v", err)
  }
  client.enqueue(msg)
  return cellT, nil
}

// ws-eval: (ws-eval srv js) → Anzahl Empfaenger. Feuern und vergessen,
// keine Rueckgabe aus dem Browser.
func fnWSEval(args []*Cell) (*Cell, error) {
  if len(args) < 2 {
    return nil, fmt.Errorf("ws-eval: 2 Argumente (srv js) nötig")
  }
  ws, err := asServer("ws-eval", args[0])
  if err != nil {
    return nil, err
  }
  if args[1].Type != STRING {
    return nil, fmt.Errorf("ws-eval: js muss String sein")
  }
  msg, err := json.Marshal(wsOutMsg{JS: args[1].Val})
  if err != nil {
    return nil, fmt.Errorf("ws-eval: %v", err)
  }
  return MakeNum(float64(ws.broadcast(msg))), nil
}

// ws-call: (ws-call srv client js &key timeout) → Wert oder Lisp-Error.
// Blockiert; Default-Timeout 5000 ms. Unbekannter Client → Fehler,
// kein Haenger (Spec §8.3).
func fnWSCall(args []*Cell) (*Cell, error) {
  if len(args) < 3 {
    return nil, fmt.Errorf("ws-call: 3 Argumente (srv client js) nötig")
  }
  ws, err := asServer("ws-call", args[0])
  if err != nil {
    return nil, err
  }
  if args[1].Type != NUMBER {
    return nil, fmt.Errorf("ws-call: Client-ID muss NUMBER sein")
  }
  if args[2].Type != STRING {
    return nil, fmt.Errorf("ws-call: js muss String sein")
  }
  timeout := 5000 * time.Millisecond
  for i := 3; i+1 < len(args); i += 2 {
    if args[i].Type != ATOM || args[i].Val != ":timeout" {
      return nil, fmt.Errorf("ws-call: unbekanntes Keyword %s", args[i])
    }
    if args[i+1].Type != NUMBER {
      return nil, fmt.Errorf("ws-call: :timeout muss NUMBER (ms) sein")
    }
    timeout = time.Duration(args[i+1].Num) * time.Millisecond
  }
  clientID := int(args[1].Num)
  client := ws.findClient(clientID)
  if client == nil {
    return nil, fmt.Errorf("ws-call: unbekannter Client %d", clientID)
  }

  ws.mu.Lock()
  ws.nextCall++
  callID := ws.nextCall
  ch := make(chan wsCallResult, 1) // gepuffert: deliverCallResult blockiert nie
  ws.pending[callID] = ch
  client.callIDs[callID] = struct{}{}
  ws.mu.Unlock()

  msg, err := json.Marshal(wsOutMsg{Call: callID, JS: args[2].Val})
  if err != nil {
    return nil, fmt.Errorf("ws-call: %v", err)
  }
  client.enqueue(msg)

  select {
  case res := <-ch:
    if res.err != nil {
      return nil, res.err
    }
    return res.val, nil
  case <-time.After(timeout):
    ws.mu.Lock()
    delete(ws.pending, callID)
    if c := ws.clients[clientID]; c != nil {
      delete(c.callIDs, callID)
    }
    ws.mu.Unlock()
    return nil, fmt.Errorf("ws-call: timeout nach %dms", int(timeout/time.Millisecond))
  }
}

// ws-clients: (ws-clients srv) → Liste der verbundenen Client-IDs.
func fnWSClients(args []*Cell) (*Cell, error) {
  if len(args) < 1 {
    return nil, fmt.Errorf("ws-clients: 1 Argument nötig")
  }
  ws, err := asServer("ws-clients", args[0])
  if err != nil {
    return nil, err
  }
  ws.mu.RLock()
  ids := make([]*Cell, 0, len(ws.clients))
  for id := range ws.clients {
    ids = append(ids, MakeNum(float64(id)))
  }
  ws.mu.RUnlock()
  return SliceToCell(ids), nil
}
