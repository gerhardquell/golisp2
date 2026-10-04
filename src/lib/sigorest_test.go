//**********************************************************************
//  lib/sigorest_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Haiku 4.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260928
//**********************************************************************
// Tests für die sigoREST-Anbindung gegen einen Fake-sigoREST
// (httptest). Kein Live-Dienst nötig.
//**********************************************************************

package lib

import (
  "encoding/json"
  "io"
  "net/http"
  "net/http/httptest"
  "strings"
  "sync"
  "testing"

  "golisp2/src/embed"
)

func TestKiReferenzEmbedded(t *testing.T) {
  if !strings.HasPrefix(assets.KiReferenz, "# GoLisp2 — KI-Kurzreferenz") {
    t.Fatalf("KiReferenz beginnt falsch: %.60q", assets.KiReferenz)
  }
  if !strings.Contains(assets.KiReferenz, "sigo") {
    t.Fatal("KiReferenz erwähnt sigo nicht")
  }
}

// fakeSigoResponse: Antwort im Format von sigoREST nach Task 3.
const fakeSigoResponse = `{"model":"test-m","choices":[{"message":{"role":"assistant","content":"(+ 1 2)"},"finish_reason":"stop"}],` +
  `"usage":{"prompt_tokens":5210,"completion_tokens":115,"total_tokens":5325,` +
  `"prompt_tokens_details":{"cached_tokens":5120},"completion_tokens_details":{"reasoning_tokens":98},"cost_usd":0.0031}}`

// fakeSigo startet einen Fake-sigoREST; der letzte Request-Body landet in
// *got (nil = nicht aufzeichnen, z. B. bei parallelen Calls).
func fakeSigo(t *testing.T, got *map[string]interface{}) *httptest.Server {
  t.Helper()
  var mu sync.Mutex
  srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    var body map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
      t.Errorf("fakeSigo: %v", err)
    }
    if got != nil {
      mu.Lock()
      *got = body
      mu.Unlock()
    }
    w.Header().Set("Content-Type", "application/json")
    io.WriteString(w, fakeSigoResponse)
  }))
  t.Cleanup(srv.Close)
  return srv
}

// withSigoState sichert den globalen sigo-Zustand und stellt ihn nach dem
// Test wieder her.
func withSigoState(t *testing.T) {
  t.Helper()
  sigoStateMu.Lock()
  host, sp, sum, calls := sigoHost, sigoSystemPrompt, sigoUsageSum, sigoCalls
  sigoStateMu.Unlock()
  t.Cleanup(func() {
    sigoStateMu.Lock()
    sigoHost, sigoSystemPrompt, sigoUsageSum, sigoCalls = host, sp, sum, calls
    sigoStateMu.Unlock()
  })
}

func TestSigoSystemPrompt_Lisp(t *testing.T) {
  withSigoState(t)
  evalEq(t, `(equal? (sigo-system-prompt) (sigo-reference))`, "t")
  evalEq(t, `(string-length (sigo-system-prompt "abc"))`, "3")
  evalEq(t, `(string-length (sigo-system-prompt))`, "3")
  evalEq(t, `(string-length (sigo-system-prompt ""))`, "0")
  evalEq(t, `(equal? (sigo-system-prompt (sigo-reference)) (sigo-reference))`, "t")
  evalErr(t, `(sigo-system-prompt 42)`)
}

func TestSigo_LispSendsCurrentSystemPrompt(t *testing.T) {
  withSigoState(t)
  var got map[string]interface{}
  srv := fakeSigo(t, &got)
  evalEq(t, `(sigo-system-prompt "VS")`, `"VS"`)
  evalEq(t, `(string-length (sigo "hi" "m" "" "`+srv.URL+`"))`, "7")
  if got["system_prompt"] != "VS" || got["bare"] != true {
    t.Errorf("Request falsch: %v", got)
  }
}

func TestSigoState_ConcurrentAccess(t *testing.T) {
  withSigoState(t)
  var wg sync.WaitGroup
  for i := 0; i < 8; i++ {
    wg.Add(4)
    go func() { defer wg.Done(); _, _ = fnSigoHost([]*Cell{MakeStr("http://h:1/")}) }()
    go func() { defer wg.Done(); _ = sigoGetHost() }()
    go func() { defer wg.Done(); _, _ = fnSigoSystemPrompt([]*Cell{MakeStr("x")}) }()
    go func() { defer wg.Done(); _ = sigoGetSystemPrompt() }()
  }
  wg.Wait()
  if h := sigoGetHost(); h != "http://h:1" {
    t.Fatalf("Host: erwartet http://h:1, bekommen %s", h)
  }
}

func sigoArgs(url string) []*Cell {
  return []*Cell{MakeStr("hi"), MakeStr("m"), MakeStr(""), MakeStr(url)}
}

// hashGet liest key aus einer Hash-Tabelle oder bricht den Test ab.
func hashGet(t *testing.T, h *Cell, key string) *Cell {
  t.Helper()
  if h == nil || h.Type != HASHTABLE {
    t.Fatalf("Hash-Tabelle erwartet, got %s", h)
  }
  v, ok := h.Ht.getStr(key)
  if !ok {
    t.Fatalf("Schlüssel %s fehlt", key)
  }
  return v
}

func TestSigo_SendsOneUserMessage(t *testing.T) {
  withSigoState(t)
  var got map[string]interface{}
  srv := fakeSigo(t, &got)
  r, err := fnSigo(sigoArgs(srv.URL))
  if err != nil {
    t.Fatal(err)
  }
  if r.Val != "(+ 1 2)" {
    t.Errorf("Text: got %q", r.Val)
  }
  msgs, _ := got["messages"].([]interface{})
  if len(msgs) != 1 {
    t.Fatalf("erwartet 1 Message, bekommen %v", got["messages"])
  }
  m, _ := msgs[0].(map[string]interface{})
  if m["role"] != "user" || m["content"] != "hi" || got["model"] != "m" {
    t.Errorf("Request falsch: %v", got)
  }
  if _, present := got["session_id"]; present {
    t.Errorf("leere session-id darf nicht gesendet werden: %v", got)
  }
}

func TestSigo_TypeErrors(t *testing.T) {
  evalErr(t, `(sigo)`)
  evalErr(t, `(sigo 5)`)
  evalErr(t, `(sigo "x" 5)`)
  evalErr(t, `(sigo "a" "b" "c" "d" "e")`)
  evalErr(t, `(sigo* 5)`)
}

func TestSigoStar_ReturnsHash(t *testing.T) {
  withSigoState(t)
  srv := fakeSigo(t, nil)
  r, err := fnSigoStar(sigoArgs(srv.URL))
  if err != nil {
    t.Fatal(err)
  }
  for key, want := range map[string]string{"text": "(+ 1 2)", "model": "test-m", "finish-reason": "stop"} {
    if got := hashGet(t, r, key).Val; got != want {
      t.Errorf("%s: erwartet %q, bekommen %q", key, want, got)
    }
  }
  for key, want := range map[string]float64{"prompt-tokens": 5210, "completion-tokens": 115,
    "cached-tokens": 5120, "reasoning-tokens": 98, "cost-usd": 0.0031} {
    if got := hashGet(t, r, key).Num; got != want {
      t.Errorf("%s: erwartet %v, bekommen %v", key, want, got)
    }
  }
  if e := hashGet(t, r, "elapsed").Num; e <= 0 {
    t.Errorf("elapsed muss > 0 sein, got %v", e)
  }
}

func TestSigoStar_Lisp(t *testing.T) {
  withSigoState(t)
  srv := fakeSigo(t, nil)
  evalEq(t, `(gethash "text" (sigo* "hi" "m" "" "`+srv.URL+`"))`, `"(+ 1 2)"`)
}

func TestSigoUsage_SumsAndResets(t *testing.T) {
  withSigoState(t)
  srv := fakeSigo(t, nil)
  bad := fakeSigoWith(t, http.StatusInternalServerError, 0, nil)
  if _, err := fnSigoUsageReset(nil); err != nil {
    t.Fatal(err)
  }
  for i := 0; i < 2; i++ {
    if _, err := fnSigo(sigoArgs(srv.URL)); err != nil {
      t.Fatal(err)
    }
  }
  if _, err := fnSigo(sigoArgs(bad.URL)); err == nil {
    t.Fatal("HTTP 500 muss Fehler liefern")
  }
  u, _ := fnSigoUsage(nil)
  if got := hashGet(t, u, "calls").Num; got != 2 {
    t.Errorf("calls: erwartet 2 (Fehlschlag zählt nicht), bekommen %v", got)
  }
  if got := hashGet(t, u, "cached-tokens").Num; got != 10240 {
    t.Errorf("cached-tokens: erwartet 10240, bekommen %v", got)
  }
  if got := hashGet(t, u, "cost-usd").Num; got < 0.0062-1e-9 || got > 0.0062+1e-9 {
    t.Errorf("cost-usd: erwartet 0.0062, bekommen %v", got)
  }
  if got := hashGet(t, u, "elapsed").Num; got <= 0 {
    t.Errorf("elapsed: erwartet > 0, bekommen %v", got)
  }
  if _, err := fnSigoUsageReset(nil); err != nil {
    t.Fatal(err)
  }
  u, _ = fnSigoUsage(nil)
  if hashGet(t, u, "calls").Num != 0 || hashGet(t, u, "elapsed").Num != 0 {
    t.Errorf("nach Reset nicht 0: calls=%v elapsed=%v", hashGet(t, u, "calls"), hashGet(t, u, "elapsed"))
  }
}

func TestSigoStar_ParallelCallsAreRaceFree(t *testing.T) {
  withSigoState(t)
  srv := fakeSigo(t, nil)
  _, _ = fnSigoUsageReset(nil)
  var wg sync.WaitGroup
  for i := 0; i < 4; i++ {
    wg.Add(3)
    go func() { defer wg.Done(); _, _ = fnSigoStar(sigoArgs(srv.URL)) }()
    go func() { defer wg.Done(); _, _ = fnSigoUsage(nil) }()
    go func() { defer wg.Done(); _, _ = fnSigoSystemPrompt([]*Cell{MakeStr("p")}) }()
  }
  wg.Wait()
  u, _ := fnSigoUsage(nil)
  if got := hashGet(t, u, "calls").Num; got != 4 {
    t.Errorf("calls: erwartet 4, bekommen %v", got)
  }
}
