//**********************************************************************
//  lib/wsbridge_lifecycle_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : OpenAI
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261007
//**********************************************************************

package lib

import (
  "encoding/json"
  "sync"
  "testing"
  "time"
)

// ########################################
func waitWSClient(t *testing.T, ws *WebServer) *wsClient {
  t.Helper()
  deadline := time.Now().Add(2 * time.Second)
  for time.Now().Before(deadline) {
    ws.mu.RLock()
    for _, c := range ws.clients {
      ws.mu.RUnlock()
      return c
    }
    ws.mu.RUnlock()
    time.Sleep(time.Millisecond)
  }
  t.Fatal("Client nicht registriert")
  return nil
}

// ########################################
func TestWSLateReplyAfterDisconnect(t *testing.T) {
  ws, _ := wsTestServer(t)
  conn := wsDial(t, ws)
  client := waitWSClient(t, ws)
  started := make(chan struct{})
  release := make(chan struct{})
  var releaseOnce sync.Once
  unblock := func() { releaseOnce.Do(func() { close(release) }) }
  defer unblock()
  ws.mu.Lock()
  ws.handlers["slow"] = makeFn(func(_ []*Cell) (*Cell, error) {
    close(started)
    <-release
    return MakeNum(42), nil
  })
  ws.mu.Unlock()

  done := make(chan any, 1)
  go func() {
    defer func() { done <- recover() }()
    id := 1
    ws.handleOp(client, &id, "slow", nil)
  }()
  select {
  case <-started:
  case <-time.After(2 * time.Second):
    t.Fatal("Handler nicht gestartet")
  }
  if err := conn.Close(); err != nil {
    t.Fatal(err)
  }
  deadline := time.Now().Add(2 * time.Second)
  for ws.findClient(client.id) != nil {
    if time.Now().After(deadline) {
      t.Fatal("Client nicht abgemeldet")
    }
    time.Sleep(time.Millisecond)
  }
  // Sorgt auch dafür, dass das Schließen des Sendekanals abgeschlossen ist.
  ws.removeClient(client)
  unblock()
  select {
  case p := <-done:
    if p != nil {
      t.Fatalf("Verspätete Antwort löst Panic aus: %v", p)
    }
  case <-time.After(2 * time.Second):
    t.Fatal("Verspätete Antwort hängt")
  }
}

// ########################################
func TestWSClientConcurrentSendAndClose(t *testing.T) {
  for i := 0; i < 200; i++ {
    c := &wsClient{send: make(chan []byte, 16)}
    start := make(chan struct{})
    var wg sync.WaitGroup
    for j := 0; j < 16; j++ {
      wg.Add(1)
      go func() {
        defer wg.Done()
        <-start
        c.enqueue([]byte("message"))
      }()
    }
    for j := 0; j < 2; j++ {
      wg.Add(1)
      go func() {
        defer wg.Done()
        <-start
        c.closeSend()
      }()
    }
    close(start)
    wg.Wait()
    if c.enqueue([]byte("late")) {
      t.Fatal("Geschlossener Client akzeptiert Nachricht")
    }
  }
}

// ########################################
func TestWSCallDisconnectCleansPending(t *testing.T) {
  ws, _ := wsTestServer(t)
  conn := wsDial(t, ws)
  client := waitWSClient(t, ws)
  done := make(chan error, 1)
  go func() {
    _, err := fnWSCall([]*Cell{
      {Type: LIST, Env: ws}, MakeNum(float64(client.id)), MakeStr("1"),
      MakeAtom(":timeout"), MakeNum(10000),
    })
    done <- err
  }()
  msg := readMsg(t, conn)
  if len(msg["call"]) == 0 {
    t.Fatalf("ws-call-Nachricht fehlt: %v", msg)
  }
  if err := conn.Close(); err != nil {
    t.Fatal(err)
  }
  select {
  case err := <-done:
    if err == nil {
      t.Fatal("Disconnect muss Fehler liefern")
    }
  case <-time.After(2 * time.Second):
    t.Fatal("ws-call wartet trotz Disconnect weiter")
  }
  ws.mu.RLock()
  pending, calls := len(ws.pending), len(client.callIDs)
  ws.mu.RUnlock()
  if pending != 0 || calls != 0 {
    t.Fatalf("Verwaiste Aufrufe: pending=%d, callIDs=%d", pending, calls)
  }
}

// ########################################
func TestWSCallRejectedSendCleansPending(t *testing.T) {
  c := &wsClient{id: 1, send: make(chan []byte, 1), callIDs: make(map[int]struct{})}
  c.send <- []byte("full")
  ws := &WebServer{
    clients: map[int]*wsClient{1: c},
    pending: make(map[int]chan wsCallResult),
  }
  _, err := fnWSCall([]*Cell{
    {Type: LIST, Env: ws}, MakeNum(1), MakeStr("1"),
  })
  if err == nil {
    t.Fatal("Voller Sendepuffer muss Fehler liefern")
  }
  if len(ws.pending) != 0 || len(c.callIDs) != 0 {
    t.Fatalf("Verwaiste Aufrufe: pending=%d, callIDs=%d", len(ws.pending), len(c.callIDs))
  }
  c.closeSend()
}

// ########################################
func TestWSHandlerPanicIsContained(t *testing.T) {
  ws, _ := wsTestServer(t)
  conn := wsDial(t, ws)
  waitWSClient(t, ws)
  ws.mu.Lock()
  ws.handlers["panic"] = makeFn(func(_ []*Cell) (*Cell, error) {
    panic("test handler panic")
  })
  ws.handlers["ok"] = makeFn(func(_ []*Cell) (*Cell, error) {
    return MakeNum(7), nil
  })
  ws.mu.Unlock()

  if err := conn.WriteJSON(map[string]any{"id": 1, "op": "panic"}); err != nil {
    t.Fatal(err)
  }
  msg := readMsg(t, conn)
  var id int
  var text string
  if err := json.Unmarshal(msg["id"], &id); err != nil || id != 1 {
    t.Fatalf("Falsche Request-ID: %s (%v)", msg["id"], err)
  }
  if err := json.Unmarshal(msg["err"], &text); err != nil || text == "" {
    t.Fatalf("Fehlerantwort fehlt: %v", msg)
  }
  if err := conn.WriteJSON(map[string]any{"id": 2, "op": "ok"}); err != nil {
    t.Fatal(err)
  }
  msg = readMsg(t, conn)
  var n int
  if err := json.Unmarshal(msg["ok"], &n); err != nil || n != 7 {
    t.Fatalf("Verbindung nach Panic nicht verwendbar: %v", msg)
  }
}
