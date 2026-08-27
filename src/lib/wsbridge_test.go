//**********************************************************************
//  lib/wsbridge_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260807
//**********************************************************************
// Integrationstests der WebSocket-Bridge mit echtem gorilla-Client
// gegen einen echten http-serve-Server (Spec TODO.md §4, §5, §8.3).
//**********************************************************************

package lib

import (
  "encoding/json"
  "fmt"
  "net/http"
  "strings"
  "testing"
  "time"

  "github.com/gorilla/websocket"
)

// wsTestServer startet einen echten Server auf Port 0.
func wsTestServer(t *testing.T) (*WebServer, *Env) {
  t.Helper()
  env := BaseEnv()
  cell, err := fnHTTPServe(env, []*Cell{MakeNum(0)})
  if err != nil {
    t.Fatalf("http-serve: %v", err)
  }
  ws := cell.Env.(*WebServer)
  t.Cleanup(func() {
    fnHTTPStop([]*Cell{cell}) //nolint:errcheck
  })
  return ws, env
}

// wsDial verbindet einen Test-Client.
func wsDial(t *testing.T, ws *WebServer) *websocket.Conn {
  t.Helper()
  conn, _, err := websocket.DefaultDialer.Dial(
    fmt.Sprintf("ws://127.0.0.1:%d/_golisp/ws", ws.port), nil)
  if err != nil {
    t.Fatalf("Dial: %v", err)
  }
  t.Cleanup(func() { conn.Close() })
  return conn
}

// evalCode wertet Lisp-Code im env aus.
func evalCode(t *testing.T, env *Env, code string) *Cell {
  t.Helper()
  forms, err := ReadAll(code)
  if err != nil {
    t.Fatalf("ReadAll(%q): %v", code, err)
  }
  var res *Cell
  for f := forms; f != nil && f.Type == LIST; f = f.Cdr {
    res, err = Eval(f.Car, env)
    if err != nil {
      t.Fatalf("Eval(%q): %v", code, err)
    }
  }
  return res
}

// readMsg liest eine WS-Nachricht mit Timeout.
func readMsg(t *testing.T, conn *websocket.Conn) map[string]json.RawMessage {
  t.Helper()
  conn.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:errcheck
  var msg map[string]json.RawMessage
  if err := conn.ReadJSON(&msg); err != nil {
    t.Fatalf("ReadJSON: %v", err)
  }
  return msg
}

func TestWSOpCallAndLiveRedefine(t *testing.T) {
  ws, env := wsTestServer(t)
  conn := wsDial(t, ws)

  srv := &Cell{Type: LIST, Env: ws}

  // ws-export mit Handler, der die Client-ID mitbekommt (Spec §3.2)
  handler, err := Read(`(lambda (c frage) frage)`)
  if err != nil {
    t.Fatal(err)
  }
  fn, err := Eval(handler, env)
  if err != nil {
    t.Fatal(err)
  }
  if _, err := fnWSExport([]*Cell{srv, MakeStr("echo"), fn}); err != nil {
    t.Fatalf("ws-export: %v", err)
  }

  send := map[string]interface{}{"id": 1, "op": "echo", "args": []string{"hallo"}}
  if err := conn.WriteJSON(send); err != nil {
    t.Fatal(err)
  }
  msg := readMsg(t, conn)
  var ok string
  if err := json.Unmarshal(msg["ok"], &ok); err != nil || ok != "hallo" {
    t.Fatalf("Antwort ok = %q (err: %s), erwartet \"hallo\"", ok, msg["err"])
  }

  // Live-Redefinition (Spec §8.4 A): Handler austauschen, ohne Reconnect
  fn2, err := Eval(mustRead(t, `(lambda (c frage) "zwei")`), env)
  if err != nil {
    t.Fatal(err)
  }
  if _, err := fnWSExport([]*Cell{srv, MakeStr("echo"), fn2}); err != nil {
    t.Fatal(err)
  }
  if err := conn.WriteJSON(map[string]interface{}{"id": 2, "op": "echo", "args": []string{"x"}}); err != nil {
    t.Fatal(err)
  }
  msg = readMsg(t, conn)
  if err := json.Unmarshal(msg["ok"], &ok); err != nil || ok != "zwei" {
    t.Fatalf("nach Redefinition ok = %q, erwartet \"zwei\"", ok)
  }

  // unbekannte Operation → err, Verbindung bleibt offen
  if err := conn.WriteJSON(map[string]interface{}{"id": 3, "op": "gibtsnicht"}); err != nil {
    t.Fatal(err)
  }
  msg = readMsg(t, conn)
  var errStr string
  if err := json.Unmarshal(msg["err"], &errStr); err != nil ||
    !strings.Contains(errStr, "unbekannte Operation") {
    t.Fatalf("err = %q, erwartet 'unbekannte Operation'", errStr)
  }

  // kaputtes JSON → verworfen, Verbindung bleibt offen (Spec §4)
  if err := conn.WriteMessage(websocket.TextMessage, []byte("{kaputt")); err != nil {
    t.Fatal(err)
  }
  if err := conn.WriteJSON(map[string]interface{}{"id": 4, "op": "echo", "args": []string{"lebt"}}); err != nil {
    t.Fatal(err)
  }
  msg = readMsg(t, conn)
  if err := json.Unmarshal(msg["ok"], &ok); err != nil || ok != "zwei" {
    t.Fatalf("nach kaputtem JSON ok = %q, erwartet \"zwei\" (Verbindung tot)", ok)
  }
}

func mustRead(t *testing.T, code string) *Cell {
  t.Helper()
  c, err := Read(code)
  if err != nil {
    t.Fatal(err)
  }
  return c
}

func TestWSEmitAndClients(t *testing.T) {
  ws, _ := wsTestServer(t)
  conn := wsDial(t, ws)
  // Reader-Loop muss die Verbindung registriert haben
  deadline := time.Now().Add(2 * time.Second)
  for {
    if n := len(fnWSClientsMust(t, ws)); n == 1 {
      break
    }
    if time.Now().After(deadline) {
      t.Fatal("Client nicht registriert")
    }
    time.Sleep(10 * time.Millisecond)
  }

  srv := &Cell{Type: LIST, Env: ws}
  res, err := fnWSEmit([]*Cell{srv, MakeAtom("tick"), MakeNum(42)})
  if err != nil {
    t.Fatal(err)
  }
  if res.Num != 1 {
    t.Fatalf("ws-emit Empfänger = %v, erwartet 1", res.Num)
  }
  msg := readMsg(t, conn)
  var event string
  var data float64
  json.Unmarshal(msg["event"], &event) //nolint:errcheck
  json.Unmarshal(msg["data"], &data)   //nolint:errcheck
  if event != "tick" || data != 42 {
    t.Fatalf("Event = %q/%v, erwartet tick/42", event, data)
  }

  // ws-emit ohne Clients → 0
  conn.Close()
  deadline = time.Now().Add(2 * time.Second)
  for {
    if len(fnWSClientsMust(t, ws)) == 0 {
      break
    }
    if time.Now().After(deadline) {
      t.Fatal("Client nicht abgemeldet")
    }
    time.Sleep(10 * time.Millisecond)
  }
  res, err = fnWSEmit([]*Cell{srv, MakeAtom("tick"), MakeNum(1)})
  if err != nil {
    t.Fatal(err)
  }
  if res.Num != 0 {
    t.Fatalf("ws-emit ohne Clients = %v, erwartet 0 (Spec §8.3)", res.Num)
  }
}

func fnWSClientsMust(t *testing.T, ws *WebServer) []*Cell {
  t.Helper()
  res, err := fnWSClients([]*Cell{{Type: LIST, Env: ws}})
  if err != nil {
    t.Fatal(err)
  }
  return CellToSlice(res)
}

func TestWSCallRoundtripAndUnknownClient(t *testing.T) {
  ws, _ := wsTestServer(t)
  conn := wsDial(t, ws)
  srv := &Cell{Type: LIST, Env: ws}

  deadline := time.Now().Add(2 * time.Second)
  for len(fnWSClientsMust(t, ws)) != 1 {
    if time.Now().After(deadline) {
      t.Fatal("Client nicht registriert")
    }
    time.Sleep(10 * time.Millisecond)
  }
  clist, err := fnWSClients([]*Cell{srv})
  if err != nil {
    t.Fatal(err)
  }
  cid := CellToSlice(clist)[0].Num

  // ws-call blockiert → in Goroutine; Browser-Seite hier simuliert
  type callRes struct {
    val *Cell
    err error
  }
  done := make(chan callRes, 1)
  go func() {
    val, err := fnWSCall([]*Cell{srv, MakeNum(cid), MakeStr("return window.innerWidth")})
    done <- callRes{val, err}
  }()

  msg := readMsg(t, conn)
  var callID int
  var js string
  json.Unmarshal(msg["call"], &callID) //nolint:errcheck
  json.Unmarshal(msg["js"], &js)       //nolint:errcheck
  if js != "return window.innerWidth" {
    t.Fatalf("js = %q", js)
  }
  if err := conn.WriteJSON(map[string]interface{}{"call": callID, "ok": 1280}); err != nil {
    t.Fatal(err)
  }
  select {
  case r := <-done:
    if r.err != nil {
      t.Fatalf("ws-call: %v", r.err)
    }
    if r.val.Type != NUMBER || r.val.Num != 1280 {
      t.Fatalf("ws-call = %v, erwartet 1280", r.val)
    }
  case <-time.After(2 * time.Second):
    t.Fatal("ws-call hängt")
  }

  // unbekannter Client → Fehler, kein Hänger (Spec §8.3)
  if _, err := fnWSCall([]*Cell{srv, MakeNum(999), MakeStr("1")}); err == nil ||
    !strings.Contains(err.Error(), "unbekannter Client") {
    t.Fatalf("ws-call 999: err = %v, erwartet 'unbekannter Client'", err)
  }
}

// TestWSCallReentranz: Handler ruft ws-call auf denselben Client auf,
// von dem der Request kam (Spec §8.4 B — darf nicht blockieren).
func TestWSCallReentranz(t *testing.T) {
  ws, env := wsTestServer(t)
  conn := wsDial(t, ws)
  srv := &Cell{Type: LIST, Env: ws}
  if err := env.Set("*srv*", srv); err != nil {
    t.Fatal(err)
  }

  deadline := time.Now().Add(2 * time.Second)
  for len(fnWSClientsMust(t, ws)) != 1 {
    if time.Now().After(deadline) {
      t.Fatal("Client nicht registriert")
    }
    time.Sleep(10 * time.Millisecond)
  }

  evalCode(t, env, `(ws-export *srv* "breite" (lambda (c)
    (ws-call *srv* c "return window.innerWidth")))`)

  if err := conn.WriteJSON(map[string]interface{}{"id": 1, "op": "breite"}); err != nil {
    t.Fatal(err)
  }
  // Erste Nachricht: der ws-call des Handlers an denselben Client
  msg := readMsg(t, conn)
  var callID int
  if err := json.Unmarshal(msg["call"], &callID); err != nil || callID == 0 {
    t.Fatalf("erwartete ws-call-Nachricht, bekam %v", msg)
  }
  if err := conn.WriteJSON(map[string]interface{}{"call": callID, "ok": 1440}); err != nil {
    t.Fatal(err)
  }
  // Zweite Nachricht: die op-Antwort mit dem ws-call-Ergebnis
  msg = readMsg(t, conn)
  var ok float64
  if err := json.Unmarshal(msg["ok"], &ok); err != nil || ok != 1440 {
    t.Fatalf("Reentranz ok = %v (msg %v), erwartet 1440", ok, msg)
  }
}

func TestWSOriginCheck(t *testing.T) {
  ws, _ := wsTestServer(t)

  // fremder Origin → 403
  header := http.Header{"Origin": []string{"http://evil.example.com"}}
  _, resp, err := websocket.DefaultDialer.Dial(
    fmt.Sprintf("ws://127.0.0.1:%d/_golisp/ws", ws.port), header)
  if err == nil {
    t.Fatal("fremder Origin muss abgelehnt werden")
  }
  if resp != nil && resp.StatusCode != http.StatusForbidden {
    t.Fatalf("Status = %d, erwartet 403", resp.StatusCode)
  }

  // passender Origin → ok
  header = http.Header{"Origin": []string{fmt.Sprintf("http://127.0.0.1:%d", ws.port)}}
  conn, _, err := websocket.DefaultDialer.Dial(
    fmt.Sprintf("ws://127.0.0.1:%d/_golisp/ws", ws.port), header)
  if err != nil {
    t.Fatalf("passender Origin abgelehnt: %v", err)
  }
  conn.Close()

  // fehlender Origin (Nicht-Browser) → ok (Spec §9)
  conn2, _, err := websocket.DefaultDialer.Dial(
    fmt.Sprintf("ws://127.0.0.1:%d/_golisp/ws", ws.port), nil)
  if err != nil {
    t.Fatalf("fehlender Origin abgelehnt: %v", err)
  }
  conn2.Close()
}

func TestWSUnexport(t *testing.T) {
  ws, env := wsTestServer(t)
  srv := &Cell{Type: LIST, Env: ws}
  fn, err := Eval(mustRead(t, `(lambda (c) 1)`), env)
  if err != nil {
    t.Fatal(err)
  }
  // unexport ohne Registrierung → nil (Spec §8.3)
  res, err := fnWSUnexport([]*Cell{srv, MakeStr("nix")})
  if err != nil || res.Type != NIL {
    t.Fatalf("ws-unexport unbekannt = %v (err %v), erwartet nil", res, err)
  }
  fnWSExport([]*Cell{srv, MakeStr("x"), fn}) //nolint:errcheck
  res, err = fnWSUnexport([]*Cell{srv, MakeStr("x")})
  if err != nil || res.Type == NIL {
    t.Fatalf("ws-unexport registriert = %v (err %v), erwartet t", res, err)
  }
}

func mustCell(t *testing.T, c *Cell, err error) *Cell {
  t.Helper()
  if err != nil {
    t.Fatal(err)
  }
  return c
}
