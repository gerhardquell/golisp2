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

func TestSigoCallToHost_SendsBareAndSystemPrompt(t *testing.T) {
  var got map[string]interface{}
  srv := fakeSigo(t, &got)
  res, err := sigoCallToHost("hi", "m", "", srv.URL, "VORSPANN")
  if err != nil {
    t.Fatal(err)
  }
  if got["bare"] != true {
    t.Errorf("bare: erwartet true, bekommen %v", got["bare"])
  }
  if got["system_prompt"] != "VORSPANN" {
    t.Errorf("system_prompt: erwartet VORSPANN, bekommen %v", got["system_prompt"])
  }
  msgs, _ := got["messages"].([]interface{})
  if len(msgs) != 1 {
    t.Fatalf("erwartet 1 Message, bekommen %v", got["messages"])
  }
  m, _ := msgs[0].(map[string]interface{})
  if m["role"] != "user" || m["content"] != "hi" {
    t.Errorf("Message falsch: %v", m)
  }
  want := sigoResult{Text: "(+ 1 2)", Model: "test-m", FinishReason: "stop",
    Usage: sigoUsage{PromptTokens: 5210, CompletionTokens: 115,
      CachedTokens: 5120, ReasoningTokens: 98, CostUSD: 0.0031}}
  if res != want {
    t.Errorf("Ergebnis:\n got  %+v\n want %+v", res, want)
  }
}

func TestSigoCallToHost_EmptySystemPromptOmitsField(t *testing.T) {
  var got map[string]interface{}
  srv := fakeSigo(t, &got)
  if _, err := sigoCallToHost("hi", "m", "", srv.URL, ""); err != nil {
    t.Fatal(err)
  }
  if _, present := got["system_prompt"]; present {
    t.Errorf("system_prompt darf bei leerem Vorspann nicht gesendet werden: %v", got)
  }
  if got["bare"] != true {
    t.Errorf("bare muss auch bei leerem Vorspann true sein: %v", got)
  }
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

// alistGet sucht key in einer Assoc-Liste ((key . val) …).
func alistGet(t *testing.T, alist *Cell, key string) *Cell {
  t.Helper()
  for c := alist; c != nil && c.Type == LIST; c = c.Cdr {
    if c.Car != nil && c.Car.Car != nil && c.Car.Car.Val == key {
      return c.Car.Cdr
    }
  }
  t.Fatalf("Schlüssel %s fehlt in %s", key, alist.String())
  return nil
}

func sigoArgs(url string) []*Cell {
  return []*Cell{MakeStr("hi"), MakeStr("m"), MakeStr(""), MakeStr(url)}
}

func TestSigoStar_ReturnsAlist(t *testing.T) {
  withSigoState(t)
  srv := fakeSigo(t, nil)
  r, err := fnSigoStar(sigoArgs(srv.URL))
  if err != nil {
    t.Fatal(err)
  }
  for key, want := range map[string]string{"text": "(+ 1 2)", "model": "test-m", "finish-reason": "stop"} {
    if got := alistGet(t, r, key).Val; got != want {
      t.Errorf("%s: erwartet %q, bekommen %q", key, want, got)
    }
  }
  for key, want := range map[string]float64{"prompt-tokens": 5210, "completion-tokens": 115,
    "cached-tokens": 5120, "reasoning-tokens": 98, "cost-usd": 0.0031} {
    if got := alistGet(t, r, key).Num; got != want {
      t.Errorf("%s: erwartet %v, bekommen %v", key, want, got)
    }
  }
}

func TestSigoStar_Lisp(t *testing.T) {
  withSigoState(t)
  srv := fakeSigo(t, nil)
  evalEq(t, `(car (car (sigo* "hi" "m" "" "`+srv.URL+`")))`, "text")
}

func TestSigoUsage_SumsAndResets(t *testing.T) {
  withSigoState(t)
  srv := fakeSigo(t, nil)
  if _, err := fnSigoUsageReset(nil); err != nil {
    t.Fatal(err)
  }
  for i := 0; i < 2; i++ {
    if _, err := fnSigo(sigoArgs(srv.URL)); err != nil {
      t.Fatal(err)
    }
  }
  u, _ := fnSigoUsage(nil)
  if got := alistGet(t, u, "calls").Num; got != 2 {
    t.Errorf("calls: erwartet 2, bekommen %v", got)
  }
  if got := alistGet(t, u, "cached-tokens").Num; got != 10240 {
    t.Errorf("cached-tokens: erwartet 10240, bekommen %v", got)
  }
  if got := alistGet(t, u, "cost-usd").Num; got < 0.0062-1e-9 || got > 0.0062+1e-9 {
    t.Errorf("cost-usd: erwartet 0.0062, bekommen %v", got)
  }
  if _, err := fnSigoUsageReset(nil); err != nil {
    t.Fatal(err)
  }
  u, _ = fnSigoUsage(nil)
  if got := alistGet(t, u, "calls").Num; got != 0 {
    t.Errorf("calls nach Reset: erwartet 0, bekommen %v", got)
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
  if got := alistGet(t, u, "calls").Num; got != 4 {
    t.Errorf("calls: erwartet 4, bekommen %v", got)
  }
}
