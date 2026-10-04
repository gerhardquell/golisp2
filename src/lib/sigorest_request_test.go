//**********************************************************************
//  lib/sigorest_request_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-sonnet-5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261003
//**********************************************************************
// Tests für (sigo-request h [host]) gegen einen Fake-sigoREST.
//**********************************************************************

package lib

import (
  "encoding/json"
  "io"
  "math"
  "net/http"
  "net/http/httptest"
  "strings"
  "sync"
  "testing"
  "time"
)

// fakeSigoWith: Fake-sigoREST mit Status und Verzögerung; der letzte
// Request-Body landet in *got (nil = nicht aufzeichnen).
func fakeSigoWith(t *testing.T, status int, delay time.Duration, got *map[string]interface{}) *httptest.Server {
  t.Helper()
  var mu sync.Mutex
  srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    var body map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
      t.Errorf("fakeSigoWith: %v", err)
    }
    if got != nil {
      mu.Lock()
      *got = body
      mu.Unlock()
    }
    time.Sleep(delay)
    w.Header().Set("Content-Type", "application/json")
    if status != http.StatusOK {
      w.WriteHeader(status)
      io.WriteString(w, `{"error":"budget erschoepft"}`)
      return
    }
    io.WriteString(w, fakeSigoResponse)
  }))
  t.Cleanup(srv.Close)
  return srv
}

// reqHash baut eine Request-Hash-Tabelle aus String-Key/Cell-Paaren.
func reqHash(kv ...interface{}) *Cell {
  h := newStringHash()
  for i := 0; i+1 < len(kv); i += 2 {
    h.Ht.putStr(kv[i].(string), kv[i+1].(*Cell))
  }
  return h
}

// userMsgs: messages-Liste mit einer User-Nachricht.
func userMsgs(text string) *Cell {
  m := newStringHash()
  m.Ht.putStr("role", MakeStr("user"))
  m.Ht.putStr("content", MakeStr(text))
  return List(m)
}

func TestSigoRequest_Whitelist(t *testing.T) {
  symKey := reqHash("messages", userMsgs("hi"))
  if _, err := fnPuthash([]*Cell{MakeAtom("model"), symKey, MakeStr("m")}); err != nil {
    t.Fatal(err)
  }
  cases := []struct {
    name string
    h    *Cell
    want string
  }{
    {"kein Hash", MakeStr("x"), "Hash-Tabelle erwartet"},
    {"unbekannt", reqHash("model", MakeStr("m"), "messages", userMsgs("hi"), "response_format", MakeStr("x")), "Feld 'response_format' kennt sigoREST nicht"},
    {"stream", reqHash("model", MakeStr("m"), "messages", userMsgs("hi"), "stream", cellT), "'stream' ist gesperrt"},
    {"Symbol-Key", symKey, "Key muss String sein"},
    {"ohne model", reqHash("messages", userMsgs("hi")), "'model' (String) fehlt"},
    {"ohne messages", reqHash("model", MakeStr("m")), "'messages' (Liste) fehlt"},
    {"timeout Text", reqHash("model", MakeStr("m"), "messages", userMsgs("hi"), "timeout", MakeStr("5")), "'timeout' muss Zahl > 0 sein"},
    {"timeout 0", reqHash("model", MakeStr("m"), "messages", userMsgs("hi"), "timeout", MakeNum(0)), "'timeout' muss Zahl > 0 sein"},
  }
  for _, c := range cases {
    _, err := sigoRequestCell("sigo-request", c.h, "http://127.0.0.1:1")
    if err == nil || !strings.Contains(err.Error(), c.want) {
      t.Errorf("%s: Fehler mit %q erwartet, got %v", c.name, c.want, err)
    }
  }
}

func TestSigoRequest_DefaultsAndOverrides(t *testing.T) {
  withSigoState(t)
  var got map[string]interface{}
  srv := fakeSigoWith(t, http.StatusOK, 0, &got)
  if _, err := fnSigoSystemPrompt([]*Cell{MakeStr("VS")}); err != nil {
    t.Fatal(err)
  }

  h := reqHash("model", MakeStr("m"), "messages", userMsgs("hi"))
  resp, err := sigoRequestCell("sigo-request", h, srv.URL)
  if err != nil {
    t.Fatal(err)
  }
  if got["bare"] != true || got["system_prompt"] != "VS" {
    t.Errorf("Defaults bare/system_prompt fehlen: %v", got)
  }
  if want := math.Ceil(sigoTimeout.Seconds()); got["timeout"] != want {
    t.Errorf("timeout: erwartet %v, got %v", want, got["timeout"])
  }
  if n := len(h.Ht.snapshot()); n != 2 {
    t.Errorf("Eingabetabelle verändert: %d Einträge statt 2", n)
  }
  if m := hashAt(resp, "model"); m == nil || m.Val != "test-m" {
    t.Errorf("Antwort-model: got %v", m)
  }
  if e := numAt(resp, "elapsed"); e <= 0 {
    t.Errorf("elapsed muss > 0 sein, got %v", e)
  }

  h2 := reqHash("model", MakeStr("m"), "messages", userMsgs("hi"),
    "bare", MakeAtom(":false"), "system_prompt", MakeStr("X"), "timeout", MakeNum(2.5))
  if _, err := sigoRequestCell("sigo-request", h2, srv.URL); err != nil {
    t.Fatal(err)
  }
  if got["bare"] != false || got["system_prompt"] != "X" || got["timeout"] != 3.0 {
    t.Errorf("Overrides falsch: %v", got)
  }

  if _, err := fnSigoSystemPrompt([]*Cell{MakeStr("")}); err != nil {
    t.Fatal(err)
  }
  if _, err := sigoRequestCell("sigo-request", reqHash("model", MakeStr("m"), "messages", userMsgs("hi")), srv.URL); err != nil {
    t.Fatal(err)
  }
  if _, present := got["system_prompt"]; present {
    t.Errorf("leerer Vorspann darf nicht gesendet werden: %v", got)
  }
}

func TestSigoRequest_HTTPError(t *testing.T) {
  withSigoState(t)
  srv := fakeSigoWith(t, http.StatusPaymentRequired, 0, nil)
  _, err := sigoRequestCell("sigo-request", reqHash("model", MakeStr("m"), "messages", userMsgs("hi")), srv.URL)
  if err == nil || !strings.Contains(err.Error(), "HTTP 402") || !strings.Contains(err.Error(), "budget") {
    t.Errorf("HTTP-402-Fehler mit Body erwartet, got %v", err)
  }
}

func TestSigoRequest_Timeout(t *testing.T) {
  withSigoState(t)
  srv := fakeSigoWith(t, http.StatusOK, 1500*time.Millisecond, nil)
  h := reqHash("model", MakeStr("m"), "messages", userMsgs("hi"), "timeout", MakeNum(0.2))
  _, err := sigoRequestCell("sigo-request", h, srv.URL)
  if err == nil || !strings.Contains(err.Error(), "Timeout") {
    t.Errorf("Timeout-Fehler erwartet, got %v", err)
  }
}

func TestSigoRequest_Lisp(t *testing.T) {
  withSigoState(t)
  srv := fakeSigoWith(t, http.StatusOK, 0, nil)
  evalEq(t, `(let ((h (make-hash-table :test 'equal)))
                (puthash "model" h "m")
                (puthash "messages" h (list (json-parse "{\"role\":\"user\",\"content\":\"hi\"}")))
                (gethash "model" (sigo-request h "`+srv.URL+`")))`, `"test-m"`)
  evalErr(t, `(sigo-request)`)
  evalErr(t, `(sigo-request (make-hash-table) 5)`)
}
