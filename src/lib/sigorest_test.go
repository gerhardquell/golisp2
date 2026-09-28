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
  host, sp := sigoHost, sigoSystemPrompt
  sigoStateMu.Unlock()
  t.Cleanup(func() {
    sigoStateMu.Lock()
    sigoHost, sigoSystemPrompt = host, sp
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
