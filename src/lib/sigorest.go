//**********************************************************************
//  lib/sigorest.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260223
//**********************************************************************

package lib

import (
  "bytes"
  "context"
  "encoding/json"
  "fmt"
  "io"
  "net/http"
  "os"
  "strings"
  "sync"
  "time"

  "golisp2/src/embed"
)

var (
  sigoHost    = "http://127.0.0.1:9080"
  // Lokale LLMs (z. B. ollama-qwen3-coder-30b) brauchen oft >30s.
  // Default 120s. Überschreibbar via GOLISP_SIGO_TIMEOUT.
  sigoTimeout = 120 * time.Second
  // Default-Modell wenn (sigo "prompt") ohne Modell aufgerufen wird.
  // Überschreibbar via GOLISP_SIGO_MODEL. Fallback zai-glm53
  // (Thinking-Modell; Wahl wird in TODO.md Schritt 2 neu entschieden).
  sigoDefaultModel = "zai-glm53"
  // Rate-Limiting: max 1 Request pro 2 Sekunden pro Model
  sigoRateLimiter = time.Tick(2 * time.Second)
  // Circuit-Breaker Schutz
  sigoLastCall    time.Time
  sigoCallMutex   sync.Mutex

  // sigoStateMu schützt sigoHost und sigoSystemPrompt — parfunc ruft
  // sigo aus mehreren Goroutinen, (sigo-host …) schreibt parallel.
  sigoStateMu      sync.Mutex
  // Vorspann jedes Calls (system_prompt). Leer = Modell sieht nur den Prompt.
  sigoSystemPrompt = assets.KiReferenz
)

// sigoUsage: Token- und Kostenangaben eines Calls (bzw. deren Summe).
// CostUSD ist eine obere Schranke ohne Cache-Rabatt (von sigoREST).
type sigoUsage struct {
  PromptTokens     int
  CompletionTokens int
  CachedTokens     int
  ReasoningTokens  int
  CostUSD          float64
}

type sigoResult struct {
  Text         string
  Model        string
  FinishReason string
  Usage        sigoUsage
}

// init liest sigoREST-Konfiguration aus Umgebungsvariablen:
//   GOLISP_SIGO_HOST     – sigoREST-Host (default http://127.0.0.1:9080)
//   GOLISP_SIGO_MODEL    – Default-Modell für (sigo "prompt")
//   GOLISP_SIGO_TIMEOUT  – Request-Timeout, z. B. "30s", "5m", "2m30s"
func init() {
  if h := os.Getenv("GOLISP_SIGO_HOST"); h != "" {
    sigoHost = strings.TrimRight(h, "/")
  }
  if m := os.Getenv("GOLISP_SIGO_MODEL"); m != "" {
    sigoDefaultModel = m
  }
  if t := os.Getenv("GOLISP_SIGO_TIMEOUT"); t != "" {
    if d, err := time.ParseDuration(t); err == nil {
      sigoTimeout = d
    }
  }
}

// RegisterSigo registriert die sigoREST-Primitiven
func RegisterSigo(env *Env) {
  _ = env.Set("sigo",               makeFn(fnSigo))
  _ = env.Set("sigo-models",        makeFn(fnSigoModels))
  _ = env.Set("sigo-host",          makeFn(fnSigoHost))
  _ = env.Set("sigo-system-prompt", makeFn(fnSigoSystemPrompt))
  _ = env.Set("sigo-reference",     makeFn(fnSigoReference))
}

func sigoGetHost() string {
  sigoStateMu.Lock()
  defer sigoStateMu.Unlock()
  return sigoHost
}

func sigoGetSystemPrompt() string {
  sigoStateMu.Lock()
  defer sigoStateMu.Unlock()
  return sigoSystemPrompt
}

// sigoRequest: gemeinsamer Pfad für sigo und sigo*
//   (fname "prompt" [model] [session-id] [host])
func sigoRequest(args []*Cell, fname string) (sigoResult, error) {
  if len(args) < 1 {
    return sigoResult{}, fmt.Errorf("%s: mindestens 1 Argument", fname)
  }

  prompt    := args[0].Val
  model     := sigoDefaultModel
  sessionID := ""
  host      := sigoGetHost()

  if len(args) >= 2 { model = args[1].Val }
  if len(args) >= 3 { sessionID = args[2].Val }
  if len(args) >= 4 { host = strings.TrimRight(args[3].Val, "/") }

  // Rate-Limiting: Warte auf Token im Ticker
  <-sigoRateLimiter

  // Circuit-Breaker Schutz: mindestens 500ms zwischen Calls
  sigoCallMutex.Lock()
  sinceLast := time.Since(sigoLastCall)
  if sinceLast < 500*time.Millisecond {
    time.Sleep(500*time.Millisecond - sinceLast)
  }
  sigoLastCall = time.Now()
  sigoCallMutex.Unlock()

  return sigoCallToHost(prompt, model, sessionID, host, sigoGetSystemPrompt())
}

// fnSigo: (sigo "prompt" [model] [session-id] [host]) → Antworttext
func fnSigo(args []*Cell) (*Cell, error) {
  res, err := sigoRequest(args, "sigo")
  if err != nil { return nil, err }
  return MakeStr(res.Text), nil
}

// fnSigoModels: (sigo-models) → Liste der verfügbaren Modelle
func fnSigoModels(args []*Cell) (*Cell, error) {
  resp, err := http.Get(sigoGetHost() + "/v1/models")
  if err != nil { return nil, fmt.Errorf("sigo-models: %v", err) }
  defer resp.Body.Close()

  body, _ := io.ReadAll(resp.Body)

  var data struct {
    Data []struct{ ID string `json:"id"` } `json:"data"`
  }
  if err := json.Unmarshal(body, &data); err != nil {
    return nil, fmt.Errorf("sigo-models parse: %v", err)
  }

  result := MakeNil()
  for i := len(data.Data) - 1; i >= 0; i-- {
    result = Cons(MakeStr(data.Data[i].ID), result)
  }
  return result, nil
}

// fnSigoHost: (sigo-host "http://192.168.1.10:9080") → Host ändern
func fnSigoHost(args []*Cell) (*Cell, error) {
  sigoStateMu.Lock()
  defer sigoStateMu.Unlock()
  if len(args) >= 1 {
    sigoHost = strings.TrimRight(args[0].Val, "/")
  }
  return MakeStr(sigoHost), nil
}

// fnSigoSystemPrompt: (sigo-system-prompt) lesen, (sigo-system-prompt "…")
// setzen, (sigo-system-prompt "") leeren. Liefert den (neuen) Vorspann.
func fnSigoSystemPrompt(args []*Cell) (*Cell, error) {
  sigoStateMu.Lock()
  defer sigoStateMu.Unlock()
  if len(args) >= 1 {
    if args[0].Type != STRING {
      return nil, fmt.Errorf("sigo-system-prompt: String erwartet")
    }
    sigoSystemPrompt = args[0].Val
  }
  return MakeStr(sigoSystemPrompt), nil
}

// fnSigoReference: (sigo-reference) → eingebettete KI-Kurzreferenz
func fnSigoReference(args []*Cell) (*Cell, error) {
  return MakeStr(assets.KiReferenz), nil
}

// sigoCallToHost sendet einen Chat-Request an einen bestimmten Host.
// Immer bare:true — sigoREST legt dann weder Memory noch Server-Prompt
// davor; der Kontext besteht nur aus systemPrompt (falls nicht leer).
func sigoCallToHost(prompt, model, sessionID, host, systemPrompt string) (sigoResult, error) {
  reqBody := map[string]interface{}{
    "model": model,
    "bare":  true,
    "messages": []map[string]string{
      {"role": "user", "content": prompt},
    },
  }
  if systemPrompt != "" {
    reqBody["system_prompt"] = systemPrompt
  }
  if sessionID != "" {
    reqBody["session_id"] = sessionID
  }

  data, err := json.Marshal(reqBody)
  if err != nil { return sigoResult{}, fmt.Errorf("sigo marshal: %v", err) }

  ctx, cancel := context.WithTimeout(context.Background(), sigoTimeout)
  defer cancel()

  req, err := http.NewRequestWithContext(ctx, "POST",
    host+"/v1/chat/completions",
    bytes.NewReader(data),
  )
  if err != nil { return sigoResult{}, fmt.Errorf("sigo request: %v", err) }
  req.Header.Set("Content-Type", "application/json")

  client := &http.Client{}
  resp, err := client.Do(req)
  if err != nil { return sigoResult{}, fmt.Errorf("sigo connect: %v", err) }
  defer resp.Body.Close()

  body, _ := io.ReadAll(resp.Body)

  if resp.StatusCode != 200 {
    return sigoResult{}, fmt.Errorf("sigo HTTP %d: %s", resp.StatusCode, string(body))
  }

  var result struct {
    Model   string `json:"model"`
    Choices []struct {
      Message struct {
        Content string `json:"content"`
      } `json:"message"`
      FinishReason string `json:"finish_reason"`
    } `json:"choices"`
    Usage struct {
      PromptTokens        int `json:"prompt_tokens"`
      CompletionTokens    int `json:"completion_tokens"`
      PromptTokensDetails struct {
        CachedTokens int `json:"cached_tokens"`
      } `json:"prompt_tokens_details"`
      CompletionTokensDetails struct {
        ReasoningTokens int `json:"reasoning_tokens"`
      } `json:"completion_tokens_details"`
      CostUSD float64 `json:"cost_usd"`
    } `json:"usage"`
  }
  if err := json.Unmarshal(body, &result); err != nil {
    return sigoResult{}, fmt.Errorf("sigo parse: %v", err)
  }
  if len(result.Choices) == 0 {
    return sigoResult{}, fmt.Errorf("sigo: leere Antwort")
  }
  if result.Model == "" {
    result.Model = model
  }
  return sigoResult{
    Text:         result.Choices[0].Message.Content,
    Model:        result.Model,
    FinishReason: result.Choices[0].FinishReason,
    Usage: sigoUsage{
      PromptTokens:     result.Usage.PromptTokens,
      CompletionTokens: result.Usage.CompletionTokens,
      CachedTokens:     result.Usage.PromptTokensDetails.CachedTokens,
      ReasoningTokens:  result.Usage.CompletionTokensDetails.ReasoningTokens,
      CostUSD:          result.Usage.CostUSD,
    },
  }, nil
}
