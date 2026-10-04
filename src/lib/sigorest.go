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
  "fmt"
  "io"
  "math"
  "net/http"
  "net/url"
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

  // sigoStateMu schützt sigoHost, sigoSystemPrompt, sigoUsageSum und
  // sigoCalls — parfunc ruft sigo aus mehreren Goroutinen.
  sigoStateMu      sync.Mutex
  // Vorspann jedes Calls (system_prompt). Leer = Modell sieht nur den Prompt.
  sigoSystemPrompt = assets.KiReferenz
  // Summe aller erfolgreichen Calls seit Start bzw. (sigo-usage-reset)
  sigoUsageSum     sigoUsage
  sigoCalls        int
)

// sigoUsage: Token- und Kostenangaben eines Calls (bzw. deren Summe).
// CostUSD ist eine obere Schranke ohne Cache-Rabatt (von sigoREST), gilt
// für OpenAI-kompatible Provider — kein Anthropic-Kanal konfiguriert.
type sigoUsage struct {
  PromptTokens     int
  CompletionTokens int
  CachedTokens     int
  ReasoningTokens  int
  CostUSD          float64
  Elapsed          float64 // Sekunden, nur HTTP-Aufruf
}

func (u *sigoUsage) add(o sigoUsage) {
  u.PromptTokens     += o.PromptTokens
  u.CompletionTokens += o.CompletionTokens
  u.CachedTokens     += o.CachedTokens
  u.ReasoningTokens  += o.ReasoningTokens
  u.CostUSD          += o.CostUSD
  u.Elapsed          += o.Elapsed
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
  _ = env.Set("sigo*",              makeFn(fnSigoStar))
  _ = env.Set("sigo-usage",         makeFn(fnSigoUsage))
  _ = env.Set("sigo-usage-reset",   makeFn(fnSigoUsageReset))
  _ = env.Set("sigo-request",       makeFn(fnSigoRequest))
  _ = env.Set("sigo-budget",        makeFn(fnSigoBudget))
  _ = env.Set("sigo-costs",         makeFn(fnSigoCosts))
  _ = env.Set("sigo-model-info",    makeFn(fnSigoModelInfo))
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

// sigoPromptRequest: (fname "prompt" [model] [session-id] [host]) über
// sigoRequestCell. Liefert Antwort-Hash und angefragtes Modell.
func sigoPromptRequest(fname string, args []*Cell) (*Cell, string, error) {
  if len(args) < 1 || len(args) > 4 {
    return nil, "", fmt.Errorf("%s: 1 bis 4 Argumente erwartet (prompt [model] [session-id] [host])", fname)
  }
  for i, a := range args {
    if a.Type != STRING {
      return nil, "", fmt.Errorf("%s: Argument %d muss String sein, got %s", fname, i+1, a)
    }
  }
  model := sigoDefaultModel
  if len(args) >= 2 { model = args[1].Val }
  msg := newStringHash()
  msg.Ht.putStr("role", MakeStr("user"))
  msg.Ht.putStr("content", args[0])
  h := newStringHash()
  h.Ht.putStr("model", MakeStr(model))
  h.Ht.putStr("messages", List(msg))
  if len(args) >= 3 && args[2].Val != "" {
    h.Ht.putStr("session_id", args[2])
  }
  host := ""
  if len(args) >= 4 { host = args[3].Val }
  resp, err := sigoRequestCell(fname, h, host)
  return resp, model, err
}

// sigoFirstChoice: content und finish_reason der ersten Antwort.
func sigoFirstChoice(resp *Cell) (text, finish string) {
  choices := hashAt(resp, "choices")
  if choices == nil || choices.Type != LIST {
    return "", ""
  }
  first := choices.Car
  if c := hashAt(hashAt(first, "message"), "content"); c != nil && c.Type == STRING {
    text = c.Val
  }
  if f := hashAt(first, "finish_reason"); f != nil && f.Type == STRING {
    finish = f.Val
  }
  return text, finish
}

// sigoUsageHash: Token-/Kosten-/Zeit-Keys mit Bindestrich (sigo*, sigo-usage).
func sigoUsageHash(u sigoUsage) *Cell {
  h := newStringHash()
  h.Ht.putStr("prompt-tokens", MakeNum(float64(u.PromptTokens)))
  h.Ht.putStr("completion-tokens", MakeNum(float64(u.CompletionTokens)))
  h.Ht.putStr("cached-tokens", MakeNum(float64(u.CachedTokens)))
  h.Ht.putStr("reasoning-tokens", MakeNum(float64(u.ReasoningTokens)))
  h.Ht.putStr("cost-usd", MakeNum(u.CostUSD))
  h.Ht.putStr("elapsed", MakeNum(u.Elapsed))
  return h
}

// fnSigo: (sigo "prompt" [model] [session-id] [host]) → Antworttext
func fnSigo(args []*Cell) (*Cell, error) {
  resp, _, err := sigoPromptRequest("sigo", args)
  if err != nil { return nil, err }
  text, _ := sigoFirstChoice(resp)
  return MakeStr(text), nil
}

// fnSigoStar: (sigo* "prompt" [model] [session-id] [host]) → Hash-Tabelle
// text model finish-reason prompt-/completion-/cached-/reasoning-tokens
// cost-usd elapsed
func fnSigoStar(args []*Cell) (*Cell, error) {
  resp, model, err := sigoPromptRequest("sigo*", args)
  if err != nil { return nil, err }
  if m := hashAt(resp, "model"); m != nil && m.Type == STRING && m.Val != "" {
    model = m.Val
  }
  text, finish := sigoFirstChoice(resp)
  u := sigoUsageFromResponse(resp)
  u.Elapsed = numAt(resp, "elapsed")
  out := sigoUsageHash(u)
  out.Ht.putStr("text", MakeStr(text))
  out.Ht.putStr("model", MakeStr(model))
  out.Ht.putStr("finish-reason", MakeStr(finish))
  return out, nil
}

// fnSigoUsage: (sigo-usage) → Summen seit Start/Reset plus calls
func fnSigoUsage(args []*Cell) (*Cell, error) {
  sigoStateMu.Lock()
  u, n := sigoUsageSum, sigoCalls
  sigoStateMu.Unlock()
  h := sigoUsageHash(u)
  h.Ht.putStr("calls", MakeNum(float64(n)))
  return h, nil
}

// fnSigoUsageReset: (sigo-usage-reset) → Summen auf 0
func fnSigoUsageReset(args []*Cell) (*Cell, error) {
  sigoStateMu.Lock()
  sigoUsageSum = sigoUsage{}
  sigoCalls = 0
  sigoStateMu.Unlock()
  return MakeNil(), nil
}

// sigoGetJSON: GET gegen sigoREST, Antwort per JSONToCell.
func sigoGetJSON(fname, path string) (*Cell, error) {
  data, _, err := sigoDo(fname, "GET", sigoGetHost()+path, nil, sigoTimeout)
  if err != nil { return nil, err }
  c, err := JSONToCell(data)
  if err != nil { return nil, fmt.Errorf("%s: Antwort kein JSON: %v", fname, err) }
  return c, nil
}

// fnSigoModels: (sigo-models) → Liste der Modell-IDs aus /v1/models
func fnSigoModels(args []*Cell) (*Cell, error) {
  resp, err := sigoGetJSON("sigo-models", "/v1/models")
  if err != nil { return nil, err }
  var ids []*Cell
  for c := hashAt(resp, "data"); c != nil && c.Type == LIST; c = c.Cdr {
    if id := hashAt(c.Car, "id"); id != nil && id.Type == STRING {
      ids = append(ids, id)
    }
  }
  return SliceToCell(ids), nil
}

// fnSigoBudget: (sigo-budget) → /api/budget als Hash-Tabelle
func fnSigoBudget(args []*Cell) (*Cell, error) {
  if len(args) != 0 {
    return nil, fmt.Errorf("sigo-budget: keine Argumente erwartet")
  }
  return sigoGetJSON("sigo-budget", "/api/budget")
}

// fnSigoCosts: (sigo-costs [seit [bis]]) → /api/costs als Hash-Tabelle.
// seit/bis: Unix-Sekunden wie (now), gesendet als RFC3339 (Ortszeit).
func fnSigoCosts(args []*Cell) (*Cell, error) {
  if len(args) > 2 {
    return nil, fmt.Errorf("sigo-costs: 0 bis 2 Argumente erwartet ([seit [bis]])")
  }
  q := url.Values{}
  for i, name := range []string{"since", "until"}[:len(args)] {
    if args[i].Type != NUMBER {
      return nil, fmt.Errorf("sigo-costs: Zeit muss Zahl sein (Unix-Sekunden), got %s", args[i])
    }
    q.Set(name, unixFloatToTime(args[i].Num).Format(time.RFC3339))
  }
  path := "/api/costs"
  if len(q) > 0 {
    path += "?" + q.Encode()
  }
  return sigoGetJSON("sigo-costs", path)
}

// fnSigoModelInfo: (sigo-model-info [modell]) → alle Modelle aus
// /api/models oder das eine mit passender id bzw. shortcode.
// Preise input_cost/output_cost in USD pro 1 Mio. Tokens; 0 = unbekannt.
func fnSigoModelInfo(args []*Cell) (*Cell, error) {
  if len(args) > 1 {
    return nil, fmt.Errorf("sigo-model-info: 0 oder 1 Argument erwartet ([modell])")
  }
  if len(args) == 1 && args[0].Type != STRING {
    return nil, fmt.Errorf("sigo-model-info: Modell muss String sein, got %s", args[0])
  }
  all, err := sigoGetJSON("sigo-model-info", "/api/models")
  if err != nil || len(args) == 0 { return all, err }
  for c := all; c != nil && c.Type == LIST; c = c.Cdr {
    for _, k := range []string{"id", "shortcode"} {
      if v := hashAt(c.Car, k); v != nil && v.Type == STRING && v.Val == args[0].Val {
        return c.Car, nil
      }
    }
  }
  return nil, fmt.Errorf("sigo-model-info: Modell '%s' unbekannt", args[0].Val)
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

// sigoThrottle: Rate-Limiter plus mindestens 500 ms zwischen Calls.
func sigoThrottle() {
  <-sigoRateLimiter
  sigoCallMutex.Lock()
  sinceLast := time.Since(sigoLastCall)
  if sinceLast < 500*time.Millisecond {
    time.Sleep(500*time.Millisecond - sinceLast)
  }
  sigoLastCall = time.Now()
  sigoCallMutex.Unlock()
}

// sigoRequestFields: genau die Felder von sigoREST ChatRequest
// (sigoREST/main.go:513). sigoREST verwirft andere still — hier Fehler.
var sigoRequestFields = map[string]bool{
  "model": true, "messages": true, "temperature": true, "max_tokens": true,
  "session_id": true, "timeout": true, "retries": true,
  "system_prompt": true, "bare": true, "channel": true,
}

// hashAt: Wert unter key, wenn c eine Hash-Tabelle ist, sonst nil.
func hashAt(c *Cell, key string) *Cell {
  if c == nil || c.Type != HASHTABLE {
    return nil
  }
  v, _ := c.Ht.getStr(key)
  return v
}

// numAt: Zahl unter key, sonst 0 (fehlende Usage-Felder zählen als 0).
func numAt(c *Cell, key string) float64 {
  v := hashAt(c, key)
  if v == nil || v.Type != NUMBER {
    return 0
  }
  return v.Num
}

// sigoBuildRequest prüft h gegen die Whitelist und liefert eine Kopie mit
// Defaults (bare, system_prompt, timeout) plus den HTTP-Timeout.
func sigoBuildRequest(fname string, h *Cell, systemPrompt string) (*Cell, time.Duration, error) {
  if h == nil || h.Type != HASHTABLE {
    return nil, 0, fmt.Errorf("%s: Hash-Tabelle erwartet, got %s", fname, h)
  }
  req := newStringHash()
  for _, e := range h.Ht.snapshot() {
    if e.key.Type != STRING {
      return nil, 0, fmt.Errorf("%s: Key muss String sein, got %s", fname, e.key)
    }
    k := e.key.Val
    if k == "stream" {
      return nil, 0, fmt.Errorf("%s: 'stream' ist gesperrt (sigoREST bucht Streaming-Kosten falsch)", fname)
    }
    if !sigoRequestFields[k] {
      return nil, 0, fmt.Errorf("%s: Feld '%s' kennt sigoREST nicht", fname, k)
    }
    req.Ht.putStr(k, e.val)
  }
  if m, ok := req.Ht.getStr("model"); !ok || m.Type != STRING {
    return nil, 0, fmt.Errorf("%s: 'model' (String) fehlt", fname)
  }
  if m, ok := req.Ht.getStr("messages"); !ok || m.Type != LIST {
    return nil, 0, fmt.Errorf("%s: 'messages' (Liste) fehlt", fname)
  }
  if _, ok := req.Ht.getStr("bare"); !ok {
    req.Ht.putStr("bare", cellT)
  }
  if _, ok := req.Ht.getStr("system_prompt"); !ok && systemPrompt != "" {
    req.Ht.putStr("system_prompt", MakeStr(systemPrompt))
  }
  timeout := sigoTimeout
  if v, ok := req.Ht.getStr("timeout"); ok {
    if v.Type != NUMBER || v.Num <= 0 {
      return nil, 0, fmt.Errorf("%s: 'timeout' muss Zahl > 0 sein, got %s", fname, v)
    }
    timeout = time.Duration(v.Num * float64(time.Second))
  }
  req.Ht.putStr("timeout", MakeNum(math.Ceil(timeout.Seconds())))
  return req, timeout, nil
}

// sigoDo ist der einzige HTTP-Aufruf gegen sigoREST. Liefert Body und
// Dauer (Senden bis Antwort gelesen); HTTP ≠ 200 ist ein Fehler mit Body.
func sigoDo(fname, method, url string, body []byte, timeout time.Duration) ([]byte, time.Duration, error) {
  ctx, cancel := context.WithTimeout(context.Background(), timeout)
  defer cancel()
  var rd io.Reader
  if body != nil {
    rd = bytes.NewReader(body)
  }
  req, err := http.NewRequestWithContext(ctx, method, url, rd)
  if err != nil { return nil, 0, fmt.Errorf("%s: %v", fname, err) }
  if body != nil {
    req.Header.Set("Content-Type", "application/json")
  }
  timedOut := func() error { return fmt.Errorf("%s: Timeout nach %g s", fname, timeout.Seconds()) }
  start := time.Now()
  resp, err := http.DefaultClient.Do(req)
  if err != nil {
    if ctx.Err() == context.DeadlineExceeded { return nil, 0, timedOut() }
    return nil, 0, fmt.Errorf("%s: %v", fname, err)
  }
  defer resp.Body.Close()
  data, err := io.ReadAll(resp.Body)
  elapsed := time.Since(start)
  if err != nil {
    if ctx.Err() == context.DeadlineExceeded { return nil, 0, timedOut() }
    return nil, 0, fmt.Errorf("%s: Antwort lesen: %v", fname, err)
  }
  if resp.StatusCode != http.StatusOK {
    return nil, elapsed, fmt.Errorf("%s: HTTP %d: %s", fname, resp.StatusCode, strings.TrimSpace(string(data)))
  }
  return data, elapsed, nil
}

// sigoUsageFromResponse liest usage aus der Antwort (fehlend → 0).
func sigoUsageFromResponse(resp *Cell) sigoUsage {
  usage := hashAt(resp, "usage")
  return sigoUsage{
    PromptTokens:     int(numAt(usage, "prompt_tokens")),
    CompletionTokens: int(numAt(usage, "completion_tokens")),
    CachedTokens:     int(numAt(hashAt(usage, "prompt_tokens_details"), "cached_tokens")),
    ReasoningTokens:  int(numAt(hashAt(usage, "completion_tokens_details"), "reasoning_tokens")),
    CostUSD:          numAt(usage, "cost_usd"),
  }
}

// sigoRequestCell: gemeinsamer Chat-Pfad für sigo-request, sigo und sigo*.
// host "" = (sigo-host). Fehlgeschlagene Calls zählen nicht in sigo-usage.
func sigoRequestCell(fname string, h *Cell, host string) (*Cell, error) {
  sigoStateMu.Lock()
  defHost, systemPrompt := sigoHost, sigoSystemPrompt
  sigoStateMu.Unlock()
  if host == "" {
    host = defHost
  }
  host = strings.TrimRight(host, "/")

  req, timeout, err := sigoBuildRequest(fname, h, systemPrompt)
  if err != nil { return nil, err }
  body, err := CellToJSON(req)
  if err != nil { return nil, fmt.Errorf("%s: %v", fname, err) }

  sigoThrottle()
  data, elapsed, err := sigoDo(fname, "POST", host+"/v1/chat/completions", body, timeout)
  if err != nil { return nil, err }

  resp, err := JSONToCell(data)
  if err != nil { return nil, fmt.Errorf("%s: Antwort kein JSON: %v", fname, err) }
  if resp.Type != HASHTABLE { return nil, fmt.Errorf("%s: Antwort ist kein Objekt", fname) }
  if c := hashAt(resp, "choices"); c == nil || c.Type != LIST {
    return nil, fmt.Errorf("%s: leere Antwort (keine choices)", fname)
  }

  secs := elapsed.Seconds()
  resp.Ht.putStr("elapsed", MakeNum(secs))
  u := sigoUsageFromResponse(resp)
  u.Elapsed = secs
  sigoStateMu.Lock()
  sigoUsageSum.add(u)
  sigoCalls++
  sigoStateMu.Unlock()
  return resp, nil
}

// fnSigoRequest: (sigo-request h [host]) → Antwort als Hash-Tabelle
// plus "elapsed".
func fnSigoRequest(args []*Cell) (*Cell, error) {
  if len(args) < 1 || len(args) > 2 {
    return nil, fmt.Errorf("sigo-request: 1 oder 2 Argumente erwartet (h [host])")
  }
  host := ""
  if len(args) == 2 {
    if args[1].Type != STRING {
      return nil, fmt.Errorf("sigo-request: Host muss String sein, got %s", args[1])
    }
    host = args[1].Val
  }
  return sigoRequestCell("sigo-request", args[0], host)
}
