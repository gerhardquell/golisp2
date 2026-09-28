# sigo-Kontext Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** golisp2 schickt jedem `(sigo …)`-Call eine eingebettete golisp2-Referenz als Vorspann, der Vorspann ist aus Lisp les-, setz- und leerbar, und Token-/Cache-/Kostenangaben kommen bis in golisp2 durch.

**Architecture:** Zwei Repos. sigoREST bekommt ein Request-Flag `bare` (keine Server-Injektion von Memory/System-Prompt) und reicht `cached_tokens`, `reasoning_tokens` und `cost_usd` in `usage` durch. golisp2 bettet `docs/ki/referenz.md` (verschoben nach `src/embed/ki-referenz.md`) ein, schickt sie über den einzigen HTTP-Pfad `sigoCallToHost` immer mit `bare:true`, hält Vorspann/Usage-Summe/Host unter einem Mutex und bietet neue Primitiven.

**Tech Stack:** Go 1.26 (beide Repos), `net/http/httptest` für Fake-Server, `go test -race`, golisp2-Lisp.

**Spec:** `docs/superpowers/specs/2026-09-28-sigo-kontext-design.md`

## Global Constraints

- **golisp2** (`/u/lisp-projekte/golisp2`): Einrückung **2 Spaces, keine Tabs** in `src/lib/sigorest.go` und `src/lib/sigorest_test.go`. **Kein `gofmt`** auf diese Dateien laufen lassen (CLAUDE.md: „nicht blind gofmt folgen“). Kommentare deutsch, sparsam.
- **sigoREST** (`/u/go-projekte/sigoREST`): dortige Konventionen gelten — die Dateien sind gofmt-formatiert (Tabs). Stil der umgebenden Datei übernehmen. Dortige `CLAUDE.md` vor Task 1 lesen.
- Fehlermeldungen: `fmt.Errorf("funktionsname: beschreibung")`.
- Neue Datei-Header (nur für neu angelegte Dateien): Autor Gerhard Quell, CoAutor `claude-opus-5.5`, Copyright 2026 Gerhard Quell - SKEQuell, Erstellt `20260928`. Bestehende Header nie ändern.
- Commits enden mit `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Temporäres in golisp2 nach `./tmp/` (gitignored), **nie** `/tmp`.
- `sigorest.go` bleibt der **einzige** Ort für HTTP gegen sigoREST (Chokepoint).
- Neue Primitiven werden in `RegisterSigo` registriert (wird von `BaseEnv()` aufgerufen, `src/lib/primitives.go:119`).
- Keine Datei über 1000 Zeilen.

---

## File Structure

**sigoREST**
- Modify: `sigoengine/engine.go` — `UsageData` um `CachedTokens`, `ReasoningTokens`; `extractUsage` liest sie.
- Modify: `sigoengine/usage_test.go` — Tests für die Detailfelder.
- Modify: `sigoREST/main.go` — `ChatRequest.Bare`; Kontextaufbau in `handleChatCompletions`; `ChatUsage` erweitert; `cost_usd`.
- Modify: `sigoREST/main_test.go` — Handler-Tests mit Fake-Upstream.

**golisp2**
- Move: `docs/ki/referenz.md` → `src/embed/ki-referenz.md`
- Modify: `src/embed/assets.go` — `KiReferenz`.
- Modify: `src/lib/sigorest.go` — Request-/Ergebnis-Typen, Zustand + Mutex, neue Primitiven.
- Create: `src/lib/sigorest_test.go` — alle Go-Tests für sigo.
- Modify: `src/embed/swank.lisp`, `tools/gen-reference.lisp`, `docs/sigo.md`, `CLAUDE.md`, `docs/golisp2-cheatsheet.md`, `TODO.md`; regeneriert: `docs/referenz-generiert.md`.

---

### Task 1: sigoREST — Cache-/Reasoning-Tokens aus Provider-Antwort lesen

**Files:**
- Modify: `/u/go-projekte/sigoREST/sigoengine/engine.go:556-560` (`UsageData`), `:1385-1426` (`extractUsage`)
- Test: `/u/go-projekte/sigoREST/sigoengine/usage_test.go`

**Interfaces:**
- Produces: `sigoengine.UsageData` mit neuen Feldern `CachedTokens int`, `ReasoningTokens int` (Task 3 liest sie).

- [ ] **Step 1: Failing Tests schreiben** — an `sigoengine/usage_test.go` anhängen:

```go
func TestExtractUsageOpenAIDetails(t *testing.T) {
	result := map[string]interface{}{
		"usage": map[string]interface{}{
			"prompt_tokens":     float64(5210),
			"completion_tokens": float64(115),
			"total_tokens":      float64(5325),
			"prompt_tokens_details": map[string]interface{}{
				"cached_tokens": float64(5120),
			},
			"completion_tokens_details": map[string]interface{}{
				"reasoning_tokens": float64(98),
			},
		},
	}
	u := extractUsage(result, "openai")
	if u == nil {
		t.Fatal("expected usage, got nil")
	}
	if u.CachedTokens != 5120 || u.ReasoningTokens != 98 {
		t.Fatalf("unexpected details: %+v", u)
	}
	if u.InputTokens != 5210 || u.OutputTokens != 115 {
		t.Fatalf("unexpected base tokens: %+v", u)
	}
}

func TestExtractUsageAnthropicCacheRead(t *testing.T) {
	result := map[string]interface{}{
		"usage": map[string]interface{}{
			"input_tokens":            float64(10),
			"output_tokens":           float64(5),
			"cache_read_input_tokens": float64(4000),
		},
	}
	u := extractUsage(result, "anthropic")
	if u == nil {
		t.Fatal("expected usage, got nil")
	}
	if u.CachedTokens != 4000 || u.ReasoningTokens != 0 {
		t.Fatalf("unexpected details: %+v", u)
	}
}

func TestExtractUsageNoDetails(t *testing.T) {
	result := map[string]interface{}{
		"usage": map[string]interface{}{
			"prompt_tokens": float64(10), "completion_tokens": float64(5),
		},
	}
	u := extractUsage(result, "openai")
	if u.CachedTokens != 0 || u.ReasoningTokens != 0 {
		t.Fatalf("expected zero details, got %+v", u)
	}
}
```

- [ ] **Step 2: Test laufen lassen, muss scheitern**

Run: `cd /u/go-projekte/sigoREST && go test ./sigoengine/ -run TestExtractUsage -count=1`
Expected: FAIL — Compilerfehler `u.CachedTokens undefined`.

- [ ] **Step 3: `UsageData` erweitern** (`sigoengine/engine.go:556`):

```go
type UsageData struct {
	InputTokens     int `json:"input_tokens"`
	OutputTokens    int `json:"output_tokens"`
	TotalTokens     int `json:"total_tokens"`
	CachedTokens    int `json:"cached_tokens,omitempty"`    // aus Provider-Cache gelesene Input-Tokens
	ReasoningTokens int `json:"reasoning_tokens,omitempty"` // Thinking-Anteil der Output-Tokens
}
```

- [ ] **Step 4: `extractUsage` erweitern** — direkt vor dem Kommentar `// Wenn Input/Output immer noch 0, versuche total_tokens direkt` einfügen:

```go
	// Cache-/Reasoning-Details: OpenAI-kompatibel (prompt_tokens_details /
	// completion_tokens_details), Anthropic liefert cache_read_input_tokens.
	if d, ok := u["prompt_tokens_details"].(map[string]interface{}); ok {
		usage.CachedTokens = toInt(d["cached_tokens"])
	}
	if usage.CachedTokens == 0 {
		usage.CachedTokens = toInt(u["cache_read_input_tokens"])
	}
	if d, ok := u["completion_tokens_details"].(map[string]interface{}); ok {
		usage.ReasoningTokens = toInt(d["reasoning_tokens"])
	}
```

- [ ] **Step 5: Tests laufen lassen**

Run: `cd /u/go-projekte/sigoREST && go test ./sigoengine/ -count=1`
Expected: PASS (alle, auch die bestehenden).

- [ ] **Step 6: Commit** (im sigoREST-Repo)

```bash
cd /u/go-projekte/sigoREST
git add sigoengine/engine.go sigoengine/usage_test.go
git commit -m "feat(usage): cached_tokens und reasoning_tokens aus Provider-Antwort lesen

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: sigoREST — Request-Flag `bare`

**Files:**
- Modify: `/u/go-projekte/sigoREST/sigoREST/main.go:513-524` (`ChatRequest`), `:870-907` (Kontextaufbau in `handleChatCompletions`)
- Test: `/u/go-projekte/sigoREST/sigoREST/main_test.go`

**Interfaces:**
- Produces: JSON-Feld `"bare": true` in `POST /v1/chat/completions`. golisp2 (Task 5) schickt es immer.
- Produces (Test-Helfer, Task 3 nutzt sie): `newContextTestUpstream(t, *[]map[string]interface{}) *httptest.Server`, `postChat(t, *Server, string) *httptest.ResponseRecorder`, `newContextTestServer(t, upstreamURL string) *Server`.

- [ ] **Step 1: Test-Helfer + Failing Tests schreiben** — an `sigoREST/main_test.go` anhängen (Imports `encoding/json`, `fmt`, `net/http`, `net/http/httptest`, `strings`, `sigoengine` sind in der Datei schon vorhanden; prüfen und ggf. ergänzen):

```go
// contextTestResponse: feste OpenAI-kompatible Upstream-Antwort inkl.
// Cache-/Reasoning-Details.
const contextTestResponse = `{"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],` +
	`"usage":{"prompt_tokens":5210,"completion_tokens":115,"total_tokens":5325,` +
	`"prompt_tokens_details":{"cached_tokens":5120},"completion_tokens_details":{"reasoning_tokens":98}}}`

// newContextTestUpstream legt die beim Provider ankommenden messages in *got ab.
func newContextTestUpstream(t *testing.T, got *[]map[string]interface{}) *httptest.Server {
	t.Helper()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		var body struct {
			Messages []map[string]interface{} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("upstream decode: %v", err)
		}
		*got = body.Messages
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, contextTestResponse)
	}))
	t.Cleanup(upstream.Close)
	return upstream
}

// newContextTestServer: Server mit globalem Memory und System-Prompt, damit
// sichtbar wird, ob bare sie unterdrückt.
func newContextTestServer(t *testing.T, upstreamURL string) *Server {
	t.Helper()
	srv, _ := newTestServer(t)
	srv.memory = sigoengine.MemoryBlock{Content: "MEMORY"}
	srv.systemPrompt = "GLOBAL"
	srv.models = map[string]ModelInfo{
		"claude-h": {ID: "claude-h", Endpoint: upstreamURL, MaxOutputTokens: 100,
			MaxTemperature: 1, MinTemperature: 0, InputCost: 1.0, OutputCost: 2.0},
	}
	return srv
}

func postChat(t *testing.T, srv *Server, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	rr := httptest.NewRecorder()
	srv.handleChatCompletions(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	return rr
}

// assertMessages prüft Rolle und Inhalt jeder Upstream-Message in Reihenfolge.
func assertMessages(t *testing.T, got []map[string]interface{}, want [][2]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d messages, got %d: %+v", len(want), len(got), got)
	}
	for i, w := range want {
		if got[i]["role"] != w[0] || got[i]["content"] != w[1] {
			t.Fatalf("message %d: expected %s/%q, got %+v", i, w[0], w[1], got[i])
		}
	}
}

func TestChatCompletions_BareWithSystemPrompt(t *testing.T) {
	var got []map[string]interface{}
	srv := newContextTestServer(t, newContextTestUpstream(t, &got).URL)
	postChat(t, srv, `{"model":"claude-h","bare":true,"system_prompt":"VORSPANN","messages":[{"role":"user","content":"hi"}]}`)
	assertMessages(t, got, [][2]string{{"system", "VORSPANN"}, {"user", "hi"}})
}

func TestChatCompletions_BareWithoutSystemPrompt(t *testing.T) {
	var got []map[string]interface{}
	srv := newContextTestServer(t, newContextTestUpstream(t, &got).URL)
	postChat(t, srv, `{"model":"claude-h","bare":true,"messages":[{"role":"user","content":"hi"}]}`)
	assertMessages(t, got, [][2]string{{"user", "hi"}})
}

func TestChatCompletions_NotBareKeepsServerContext(t *testing.T) {
	var got []map[string]interface{}
	srv := newContextTestServer(t, newContextTestUpstream(t, &got).URL)
	postChat(t, srv, `{"model":"claude-h","messages":[{"role":"user","content":"hi"}]}`)
	assertMessages(t, got, [][2]string{{"system", "MEMORY"}, {"system", "GLOBAL"}, {"user", "hi"}})
}
```

- [ ] **Step 2: Tests laufen lassen, müssen scheitern**

Run: `cd /u/go-projekte/sigoREST && go test ./sigoREST/ -run 'TestChatCompletions_(Bare|NotBare)' -count=1 -v`
Expected: `NotBareKeepsServerContext` PASS (Charakterisierung des Ist-Zustands); beide `Bare…` FAIL (MEMORY/GLOBAL stehen noch drin). Falls `NotBare…` scheitert: Helfer falsch, nicht weitermachen, erst Ursache klären.

- [ ] **Step 3: `ChatRequest` erweitern** (`main.go:521`, nach `SystemPrompt`):

```go
	SystemPrompt string        `json:"system_prompt"` // per-Request Override
	Bare         bool          `json:"bare"`          // sigoREST-Erweiterung: kein Memory, kein Server-System-Prompt
```

- [ ] **Step 4: Kontextaufbau umbauen** — in `handleChatCompletions` den Block von `// Globaler Memory-Block als System-Message (immer zuerst)` bis einschließlich `if effectiveSystemPrompt != "" { … }` ersetzen durch:

```go
	// Memory und Server-System-Prompt nur ohne bare. Mit bare bestimmt allein
	// der Client den Kontext: einzig ein nicht-leeres req.SystemPrompt.
	effectiveSystemPrompt := ""
	if !req.Bare {
		// Globaler Memory-Block als System-Message (immer zuerst)
		if mem.Content != "" {
			messages = append(messages, map[string]interface{}{
				"role":    "system",
				"content": mem.Content,
			})
		}

		// Kanal-spezifischer Memory-Block
		channelMemPath := sigoengine.ChannelMemoryPath(s.baseDir, ch.Provider, ch.Name)
		if data, err := os.ReadFile(channelMemPath); err == nil {
			var channelMem sigoengine.MemoryBlock
			if err := json.Unmarshal(data, &channelMem); err == nil && channelMem.Content != "" {
				messages = append(messages, map[string]interface{}{
					"role":    "system",
					"content": channelMem.Content,
				})
			}
		}

		// System-Prompt: Kanal vor globalem Default
		effectiveSystemPrompt = globalSystemPrompt
		channelPromptPath := sigoengine.ChannelSystemPromptPath(s.baseDir, ch.Provider, ch.Name)
		if data, err := os.ReadFile(channelPromptPath); err == nil {
			if prompt := strings.TrimSpace(string(data)); prompt != "" {
				effectiveSystemPrompt = prompt
			}
		}
	}
	// Request-Wert hat immer Vorrang
	if req.SystemPrompt != "" {
		effectiveSystemPrompt = req.SystemPrompt
	}
	if effectiveSystemPrompt != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": effectiveSystemPrompt,
		})
	}
```

Vorher die Originalzeilen lesen (`sed -n 865,910p sigoREST/main.go`) und sicherstellen, dass der ersetzte Block exakt diesen Umfang hat — nichts davor/danach mitnehmen.

- [ ] **Step 5: Tests laufen lassen**

Run: `cd /u/go-projekte/sigoREST && go test ./... -count=1`
Expected: PASS (alle).

- [ ] **Step 6: Commit**

```bash
cd /u/go-projekte/sigoREST
git add sigoREST/main.go sigoREST/main_test.go
git commit -m "feat(chat): Request-Flag bare — kein Memory, kein Server-System-Prompt

Mit bare:true bestimmt allein der Client den Kontext; nur ein
nicht-leeres system_prompt aus dem Request wird gesetzt. Ohne bare
bleibt das Verhalten unverändert.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: sigoREST — Usage-Details und `cost_usd` an den Client

**Files:**
- Modify: `/u/go-projekte/sigoREST/sigoREST/main.go:532-536` (`ChatUsage`), Block `// Usage akkumulieren` in `handleChatCompletions` (~Zeile 1202)
- Test: `/u/go-projekte/sigoREST/sigoREST/main_test.go`

**Interfaces:**
- Consumes: `sigoengine.UsageData.CachedTokens/ReasoningTokens` (Task 1), Test-Helfer aus Task 2, `sigoengine.CalcCostUSD(inputTokens, outputTokens int64, inputCostPerM, outputCostPerM float64) (inCost, outCost, total float64)`.
- Produces: Antwort-JSON `usage.prompt_tokens_details.cached_tokens`, `usage.completion_tokens_details.reasoning_tokens`, `usage.cost_usd` (golisp2 Task 5 parst genau diese Pfade).

- [ ] **Step 1: Failing Test schreiben** — an `main_test.go` anhängen:

```go
func TestChatCompletions_UsageDetailsPassedThrough(t *testing.T) {
	var got []map[string]interface{}
	srv := newContextTestServer(t, newContextTestUpstream(t, &got).URL)
	rr := postChat(t, srv, `{"model":"claude-h","bare":true,"messages":[{"role":"user","content":"hi"}]}`)

	var resp struct {
		Usage struct {
			PromptTokens        int `json:"prompt_tokens"`
			PromptTokensDetails struct {
				CachedTokens int `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
			CompletionTokensDetails struct {
				ReasoningTokens int `json:"reasoning_tokens"`
			} `json:"completion_tokens_details"`
			CostUSD float64 `json:"cost_usd"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Usage.PromptTokens != 5210 {
		t.Fatalf("prompt_tokens: expected 5210, got %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.PromptTokensDetails.CachedTokens != 5120 {
		t.Fatalf("cached_tokens: expected 5120, got %d", resp.Usage.PromptTokensDetails.CachedTokens)
	}
	if resp.Usage.CompletionTokensDetails.ReasoningTokens != 98 {
		t.Fatalf("reasoning_tokens: expected 98, got %d", resp.Usage.CompletionTokensDetails.ReasoningTokens)
	}
	// 5210/1e6*1.0 + 115/1e6*2.0 = 0.00544 (ohne Cache-Rabatt)
	if d := resp.Usage.CostUSD - 0.00544; d > 1e-9 || d < -1e-9 {
		t.Fatalf("cost_usd: expected 0.00544, got %v", resp.Usage.CostUSD)
	}
}

func TestChatCompletions_UsageDetailsOmittedWhenZero(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
	}))
	defer upstream.Close()
	srv := newContextTestServer(t, upstream.URL)
	rr := postChat(t, srv, `{"model":"claude-h","messages":[{"role":"user","content":"hi"}]}`)
	body := rr.Body.String()
	if strings.Contains(body, "prompt_tokens_details") || strings.Contains(body, "completion_tokens_details") {
		t.Fatalf("expected no detail objects for zero values, got %s", body)
	}
}
```

- [ ] **Step 2: Tests laufen lassen, müssen scheitern**

Run: `cd /u/go-projekte/sigoREST && go test ./sigoREST/ -run TestChatCompletions_Usage -count=1 -v`
Expected: `UsageDetailsPassedThrough` FAIL (cached_tokens 0); `OmittedWhenZero` PASS (Charakterisierung).

- [ ] **Step 3: Vorher prüfen, wer `ChatUsage` noch nutzt**

Run: `cd /u/go-projekte/sigoREST && rg -n 'ChatUsage' --type go`
Expected: nur `main.go` (Typ + Handler). Taucht es in der Anthropic-Bridge oder im Streaming auf, dort **nichts** ändern — neue Felder sind `omitempty`.

- [ ] **Step 4: `ChatUsage` erweitern** (`main.go:532`):

```go
type ChatUsage struct {
	PromptTokens            int                      `json:"prompt_tokens"`
	CompletionTokens        int                      `json:"completion_tokens"`
	TotalTokens             int                      `json:"total_tokens"`
	PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
	CostUSD                 float64                  `json:"cost_usd,omitempty"` // sigoREST-Erweiterung, ohne Cache-Rabatt (obere Schranke)
}

type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}
```

- [ ] **Step 5: Handler füllen** — im Block `// Usage akkumulieren` direkt nach der Konstruktion von `chatUsage := &ChatUsage{…}` einfügen:

```go
	if responseUsage.CachedTokens > 0 {
		chatUsage.PromptTokensDetails = &PromptTokensDetails{CachedTokens: responseUsage.CachedTokens}
	}
	if responseUsage.ReasoningTokens > 0 {
		chatUsage.CompletionTokensDetails = &CompletionTokensDetails{ReasoningTokens: responseUsage.ReasoningTokens}
	}
	_, _, chatUsage.CostUSD = sigoengine.CalcCostUSD(
		int64(responseUsage.InputTokens), int64(responseUsage.OutputTokens),
		modelInfo.InputCost, modelInfo.OutputCost,
	)
```

Vorher prüfen, dass `modelInfo` an dieser Stelle im Scope ist und den `ModelInfo` des angefragten Modells hält (`rg -n 'modelInfo' sigoREST/main.go` im Bereich von `handleChatCompletions`). Falls nicht: denselben Lookup verwenden, den `recordUsageWithSession` nutzt (`s.mu.RLock(); info, exists := s.models[modelID]; s.mu.RUnlock()`), und nur bei `exists` rechnen.

- [ ] **Step 6: Tests laufen lassen**

Run: `cd /u/go-projekte/sigoREST && go test ./... -count=1 && make build`
Expected: PASS, Build ok.

- [ ] **Step 7: Commit**

```bash
cd /u/go-projekte/sigoREST
git add sigoREST/main.go sigoREST/main_test.go
git commit -m "feat(chat): usage um cached/reasoning_tokens und cost_usd erweitern

OpenAI-kompatible Detailobjekte (nur wenn > 0) plus sigoREST-eigenes
cost_usd aus input_cost/output_cost, ohne Cache-Rabatt.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

- [ ] **Step 8: Gerhard informieren** — Live-Deployment (root-Service, `sudo cp build/sigoREST /usr/local/sbin/sigoREST && systemctl restart sigorest`) macht **Gerhard** aus seinem Terminal. Nicht selbst versuchen. golisp2-Tasks laufen unabhängig davon weiter.

---

### Task 4: golisp2 — Referenz verschieben und einbetten

**Files:**
- Move: `docs/ki/referenz.md` → `src/embed/ki-referenz.md`
- Modify: `src/embed/assets.go`
- Modify: `CLAUDE.md` (Zeile mit `docs/ki/referenz.md`), `docs/golisp2-cheatsheet.md` (Zeilen 4 und 856)
- Create: `src/lib/sigorest_test.go`

**Interfaces:**
- Produces: `assets.KiReferenz string` (Import `"golisp2/src/embed"`, Paketname `assets`) — Task 5 nutzt es als Default-Vorspann.

- [ ] **Step 1: Failing Test anlegen** — `src/lib/sigorest_test.go`:

```go
//**********************************************************************
//  lib/sigorest_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260928
//**********************************************************************
// Tests für die sigoREST-Anbindung gegen einen Fake-sigoREST
// (httptest). Kein Live-Dienst nötig.
//**********************************************************************

package lib

import (
  "strings"
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
```

- [ ] **Step 2: Test laufen lassen, muss scheitern**

Run: `go test ./src/lib/ -run TestKiReferenzEmbedded -count=1`
Expected: FAIL — `undefined: assets.KiReferenz`.

- [ ] **Step 3: Datei verschieben**

```bash
git mv docs/ki/referenz.md src/embed/ki-referenz.md
```

- [ ] **Step 4: Dateikopf korrigieren** — in `src/embed/ki-referenz.md` die Zeilen 1–8 (Titel bis `> generiert via \`tools/gen-reference.lisp\` (Stand 20260827).`) ersetzen durch:

```markdown
# GoLisp2 — KI-Kurzreferenz (tokenoptimiert)

> **Ziel:** Initial-Context für KIs, die GoLisp2-Code schreiben oder verstehen.
> Wird per `go:embed` als Default-Vorspann jedes `(sigo …)`-Calls
> mitgeschickt (`(sigo-reference)`, `(sigo-system-prompt)`).
> **Cache-Regel:** Der Anfang dieser Datei muss stabil bleiben — kein Datum,
> keine Version, nichts Wechselndes in den ersten Zeilen, sonst verfällt der
> Prompt-Cache der Provider.
> **Format:** Tabellen, Präfixe, kein Fluff. Handgepflegt, nicht generiert;
> Vollständigkeitsliste: `docs/referenz-generiert.md`.
```

- [ ] **Step 5: Einbetten** — an `src/embed/assets.go` anhängen (Stil der Datei: Tabs/Spaces wie vorhanden übernehmen):

```go

//go:embed ki-referenz.md
var KiReferenz string
```

- [ ] **Step 6: Verweise nachziehen**

Run: `rg -n 'ki/referenz\.md' --glob '!docs/superpowers/**' .`
Jeden Treffer auf `src/embed/ki-referenz.md` umstellen. Erwartet: `CLAUDE.md` (1×), `docs/golisp2-cheatsheet.md` (2×); ggf. `docs/ki/referenz_en.md`/`referenz_cn.md`, falls sie auf die deutsche Datei zeigen. Historische Specs/Pläne unter `docs/superpowers/` **nicht** ändern.

- [ ] **Step 7: Tests + Build**

Run: `go build ./... && go test ./src/lib/ -run TestKiReferenzEmbedded -count=1`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add src/embed/ki-referenz.md src/embed/assets.go src/lib/sigorest_test.go CLAUDE.md docs/golisp2-cheatsheet.md docs/ki/
git commit -m "refactor(embed): KI-Kurzreferenz nach src/embed verschoben und eingebettet

docs/ki/referenz.md -> src/embed/ki-referenz.md, als assets.KiReferenz
per go:embed verfügbar (Default-Vorspann für sigo). Falschen Hinweis
'generiert via gen-reference.lisp' entfernt, Cache-Regel ergänzt.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: golisp2 — Vorspann, `bare`, Host-Mutex, `sigo-system-prompt`, `sigo-reference`

**Files:**
- Modify: `src/lib/sigorest.go` (fast vollständig)
- Test: `src/lib/sigorest_test.go`

**Interfaces:**
- Consumes: `assets.KiReferenz` (Task 4); sigoREST-Antwortformat aus Task 3.
- Produces (Task 6 nutzt sie):
  - `type sigoUsage struct { PromptTokens, CompletionTokens, CachedTokens, ReasoningTokens int; CostUSD float64 }`
  - `type sigoResult struct { Text, Model, FinishReason string; Usage sigoUsage }`
  - `func sigoCallToHost(prompt, model, sessionID, host, systemPrompt string) (sigoResult, error)`
  - `func sigoRequest(args []*Cell, fname string) (sigoResult, error)` — Argumente parsen, drosseln, Call
  - `sigoStateMu sync.Mutex`, `sigoSystemPrompt string`, `sigoGetHost() string`, `sigoGetSystemPrompt() string`
  - Test-Helfer `fakeSigo(t, *map[string]interface{}) *httptest.Server`, `withSigoState(t)`, Konstante `fakeSigoResponse`

- [ ] **Step 1: Failing Tests schreiben** — `src/lib/sigorest_test.go` Import-Block ersetzen und Tests anhängen:

```go
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
```

Hinweis zu `evalEq(t, `(sigo-system-prompt "VS")`, `"VS"`)`: prüft die Druckdarstellung eines Strings. Vorher mit `./build/golisp2 -e '"VS"'` verifizieren, wie golisp2 Strings druckt (`"VS"` mit Anführungszeichen erwartet). Weicht es ab, die Erwartung an das tatsächliche Format anpassen — nicht den Code.

- [ ] **Step 2: Tests laufen lassen, müssen scheitern**

Run: `go test ./src/lib/ -run 'Sigo' -count=1`
Expected: FAIL — Compilerfehler (`sigoResult`, `sigoStateMu`, `fnSigoSystemPrompt` … undefiniert).

- [ ] **Step 3: `sigorest.go` umbauen** — Header und `init()` bleiben unverändert. Import-Block um `"golisp2/src/embed"` ergänzen. `var (…)`-Block, `RegisterSigo`, `fnSigo`, `fnSigoModels` (nur Host-Zugriff), `fnSigoHost`, `sigoCall` und `sigoCallToHost` ersetzen durch:

```go
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
```

(Danach folgt unverändert `func init() { … }`.)

```go
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
```

In `fnSigoModels` die Zeile `resp, err := http.Get(sigoHost + "/v1/models")` ersetzen durch `resp, err := http.Get(sigoGetHost() + "/v1/models")`. Rest von `fnSigoModels` unverändert.

```go
// fnSigoHost: (sigo-host "http://192.168.1.10:9080") → Host ändern
func fnSigoHost(args []*Cell) (*Cell, error) {
  sigoStateMu.Lock()
  defer sigoStateMu.Unlock()
  if len(args) >= 1 {
    sigoHost = strings.TrimRight(args[0].Val, "/")
  }
  return MakeStr(sigoHost), nil
}
```

`func sigoCall(…)` **löschen** — keine Aufrufer (vorher bestätigen: `rg -n 'sigoCall\(' src/` darf nur die Definition finden).

```go
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
```

Prüfen, dass kein Tab in der Datei gelandet ist: `grep -c $'\t' src/lib/sigorest.go` → `0`.

- [ ] **Step 4: Tests laufen lassen**

Run: `go build ./... && go test ./src/lib/ -run 'Sigo|KiReferenz' -count=1 -race -v`
Expected: PASS, kein `DATA RACE`. (Dauert einige Sekunden wegen des 2-s-Rate-Limiters.)

- [ ] **Step 5: Gesamte Suite**

Run: `go test ./... -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add src/lib/sigorest.go src/lib/sigorest_test.go
git commit -m "feat(sigo): Vorspann per bare/system_prompt, sigo-system-prompt, sigo-reference

Jeder sigo-Call schickt bare:true und den Vorspann (Default: eingebettete
KI-Kurzreferenz). (sigo-system-prompt) liest/setzt/leert ihn.
sigoHost und Vorspann liegen unter sigoStateMu (parfunc-Race behoben).
sigoCallToHost liefert sigoResult inkl. Usage; toter sigoCall entfernt.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: golisp2 — `sigo*`, `sigo-usage`, `sigo-usage-reset`

**Files:**
- Modify: `src/lib/sigorest.go`
- Test: `src/lib/sigorest_test.go`

**Interfaces:**
- Consumes: `sigoRequest`, `sigoResult`, `sigoUsage`, `sigoStateMu`, `fakeSigo`, `withSigoState` (Task 5).
- Produces: Lisp-Primitiven `sigo*`, `sigo-usage`, `sigo-usage-reset`; Go: `sigoUsageSum sigoUsage`, `sigoCalls int`, `func (u *sigoUsage) add(o sigoUsage)`.

- [ ] **Step 1: Failing Tests schreiben** — in `sigorest_test.go` zuerst `withSigoState` durch diese Fassung **ersetzen** (sichert jetzt auch Summe und Zähler):

```go
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
```

Dann anhängen:

```go
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
```

- [ ] **Step 2: Tests laufen lassen, müssen scheitern**

Run: `go test ./src/lib/ -run 'SigoStar|SigoUsage' -count=1`
Expected: FAIL — `undefined: fnSigoStar`, `sigoUsageSum`, `sigoCalls` …

- [ ] **Step 3: Implementieren** — in `sigorest.go`:

Im `var (…)`-Block nach `sigoSystemPrompt = assets.KiReferenz` ergänzen und den Kommentar von `sigoStateMu` anpassen:

```go
  // sigoStateMu schützt sigoHost, sigoSystemPrompt, sigoUsageSum und
  // sigoCalls — parfunc ruft sigo aus mehreren Goroutinen.
  sigoStateMu      sync.Mutex
  // Vorspann jedes Calls (system_prompt). Leer = Modell sieht nur den Prompt.
  sigoSystemPrompt = assets.KiReferenz
  // Summe aller erfolgreichen Calls seit Start bzw. (sigo-usage-reset)
  sigoUsageSum     sigoUsage
  sigoCalls        int
```

Nach der Typdefinition `sigoUsage`:

```go
func (u *sigoUsage) add(o sigoUsage) {
  u.PromptTokens     += o.PromptTokens
  u.CompletionTokens += o.CompletionTokens
  u.CachedTokens     += o.CachedTokens
  u.ReasoningTokens  += o.ReasoningTokens
  u.CostUSD          += o.CostUSD
}
```

`RegisterSigo` um drei Zeilen erweitern:

```go
  _ = env.Set("sigo*",              makeFn(fnSigoStar))
  _ = env.Set("sigo-usage",         makeFn(fnSigoUsage))
  _ = env.Set("sigo-usage-reset",   makeFn(fnSigoUsageReset))
```

In `sigoRequest` die letzte Zeile `return sigoCallToHost(…)` ersetzen durch:

```go
  res, err := sigoCallToHost(prompt, model, sessionID, host, sigoGetSystemPrompt())
  if err != nil { return sigoResult{}, err }

  sigoStateMu.Lock()
  sigoUsageSum.add(res.Usage)
  sigoCalls++
  sigoStateMu.Unlock()
  return res, nil
```

Neue Funktionen (nach `fnSigo`):

```go
// sigoAlist baut ((key . val) …) in der gegebenen Reihenfolge.
func sigoAlist(keys []string, vals []*Cell) *Cell {
  result := MakeNil()
  for i := len(keys) - 1; i >= 0; i-- {
    result = Cons(Cons(MakeAtom(keys[i]), vals[i]), result)
  }
  return result
}

func sigoUsageCells(u sigoUsage) ([]string, []*Cell) {
  return []string{"prompt-tokens", "completion-tokens", "cached-tokens", "reasoning-tokens", "cost-usd"},
    []*Cell{
      MakeNum(float64(u.PromptTokens)),
      MakeNum(float64(u.CompletionTokens)),
      MakeNum(float64(u.CachedTokens)),
      MakeNum(float64(u.ReasoningTokens)),
      MakeNum(u.CostUSD),
    }
}

// fnSigoStar: (sigo* "prompt" [model] [session-id] [host]) → Assoc-Liste
// ((text . "…") (model . "…") (finish-reason . "…") (prompt-tokens . n) …)
func fnSigoStar(args []*Cell) (*Cell, error) {
  res, err := sigoRequest(args, "sigo*")
  if err != nil { return nil, err }
  keys, vals := sigoUsageCells(res.Usage)
  keys = append([]string{"text", "model", "finish-reason"}, keys...)
  vals = append([]*Cell{MakeStr(res.Text), MakeStr(res.Model), MakeStr(res.FinishReason)}, vals...)
  return sigoAlist(keys, vals), nil
}

// fnSigoUsage: (sigo-usage) → Summen seit Start/Reset plus (calls . n)
func fnSigoUsage(args []*Cell) (*Cell, error) {
  sigoStateMu.Lock()
  u, n := sigoUsageSum, sigoCalls
  sigoStateMu.Unlock()
  keys, vals := sigoUsageCells(u)
  return sigoAlist(append(keys, "calls"), append(vals, MakeNum(float64(n)))), nil
}

// fnSigoUsageReset: (sigo-usage-reset) → Summen auf 0
func fnSigoUsageReset(args []*Cell) (*Cell, error) {
  sigoStateMu.Lock()
  sigoUsageSum = sigoUsage{}
  sigoCalls = 0
  sigoStateMu.Unlock()
  return MakeNil(), nil
}
```

`grep -c $'\t' src/lib/sigorest.go` → `0`.

- [ ] **Step 4: Tests laufen lassen**

Run: `go build ./... && go test ./src/lib/ -run 'Sigo|KiReferenz' -count=1 -race -v`
Expected: PASS, kein `DATA RACE` (dauert ~20 s wegen Rate-Limiter).

- [ ] **Step 5: Gesamte Suite**

Run: `go test ./... -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add src/lib/sigorest.go src/lib/sigorest_test.go
git commit -m "feat(sigo): sigo* mit Token-/Cache-/Kostenangaben, sigo-usage(-reset)

sigo* liefert eine Assoc-Liste (text, model, finish-reason, prompt-,
completion-, cached-, reasoning-tokens, cost-usd); sigo bleibt String.
sigo-usage summiert alle Calls mutex-geschützt, sigo-usage-reset nullt.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 7: golisp2 — SWANK, Referenzen, Doku, Verifikation

**Files:**
- Modify: `src/embed/swank.lisp` (Arglisten ~Zeile 78–80, Doku ~Zeile 112)
- Modify: `tools/gen-reference.lisp` (`*ref-docs*` ~Zeile 246–248)
- Modify: `src/embed/ki-referenz.md` (Zeile `- **sigoREST:** …`)
- Modify: `docs/sigo.md` (neuer Abschnitt vor `## Rate-Limiting`)
- Regenerate: `docs/referenz-generiert.md`
- Modify: `TODO.md` (Schritt 1 abhaken)

**Interfaces:**
- Consumes: Primitiven-Namen aus Task 5/6: `sigo*`, `sigo-system-prompt`, `sigo-reference`, `sigo-usage`, `sigo-usage-reset`.

- [ ] **Step 1: SWANK-Arglisten** — in `src/embed/swank.lisp` nach `("sigo-host" . "(sigo-host &optional host)")` einfügen:

```lisp
    ("sigo*" . "(sigo* prompt &optional model session-id host)")
    ("sigo-system-prompt" . "(sigo-system-prompt &optional text)")
    ("sigo-reference" . "(sigo-reference)")
    ("sigo-usage" . "(sigo-usage)")
    ("sigo-usage-reset" . "(sigo-usage-reset)")
```

und nach `("sigo" . "Sendet einen Prompt an den sigoREST-Server.")`:

```lisp
    ("sigo*" . "Wie sigo, liefert Assoc-Liste mit Text, Tokens, Cache und Kosten.")
    ("sigo-system-prompt" . "Liest, setzt oder leert den Vorspann für sigo-Calls.")
```

- [ ] **Step 2: Generator-Doku** — in `tools/gen-reference.lisp` nach `(sigo-models . "Liste der verfuegbaren sigoREST-Modelle")` einfügen:

```lisp
    (sigo* . "Wie sigo, Ergebnis als Assoc-Liste (text, model, finish-reason, prompt-/completion-/cached-/reasoning-tokens, cost-usd)")
    (sigo-reference . "Eingebettete KI-Kurzreferenz (Default-Vorspann von sigo)")
    (sigo-system-prompt . "Vorspann fuer sigo lesen, setzen oder mit \"\" leeren")
    (sigo-usage . "Summe der sigo-Tokens und -Kosten seit Start, plus calls")
    (sigo-usage-reset . "sigo-usage-Summen auf 0 setzen")
```

Vorher prüfen, ob `\"` in den Strings der Datei so verwendet wird (`rg -n '\\\\"' tools/gen-reference.lisp`); sonst `"..."` durch `leerem String` umformulieren.

- [ ] **Step 3: KI-Referenz** — in `src/embed/ki-referenz.md` die Zeile `- **sigoREST:** \`sigo sigo-models sigo-host\`` ersetzen durch:

```markdown
- **sigoREST:** `sigo sigo* sigo-models sigo-host sigo-system-prompt sigo-reference sigo-usage sigo-usage-reset` — `sigo*` → Assoc-Liste mit Tokens/Cache/Kosten; `(sigo-system-prompt "")` → Modell sieht nur den Prompt
```

- [ ] **Step 4: `docs/sigo.md`** — vor `## Rate-Limiting` einfügen:

```markdown
## Vorspann, Kontext und Kosten

Jeder `sigo`-Call schickt `"bare": true` und einen **Vorspann** als
`system_prompt`. Mit `bare` legt sigoREST **nichts** davor — kein
`memory.json`, kein Server-System-Prompt. Was das Modell sieht, bestimmt
allein golisp2.

Default-Vorspann ist die eingebettete KI-Kurzreferenz
(`src/embed/ki-referenz.md`, ~4–5k Token). Sie liegt über der
Cache-Schwelle der Provider; ab dem zweiten Call wird sie typischerweise
aus dem Cache gelesen (`cached-tokens`).

| Aufruf | Wirkung |
|---|---|
| `(sigo-system-prompt)` | aktuellen Vorspann lesen |
| `(sigo-system-prompt "text")` | Vorspann setzen |
| `(sigo-system-prompt "")` | leeren — Modell sieht nur den Prompt |
| `(sigo-reference)` | eingebettete Referenz; `(sigo-system-prompt (sigo-reference))` setzt zurück |
| `(sigo* prompt …)` | wie `sigo`, aber Assoc-Liste: `text model finish-reason prompt-tokens completion-tokens cached-tokens reasoning-tokens cost-usd` |
| `(sigo-usage)` | Summen seit Start/Reset plus `calls` |
| `(sigo-usage-reset)` | Summen auf 0 |

```lisp
(let ((r (sigo* "Schreibe eine Funktion quadrat.")))
  (println (cdr (assoc 'text r)))
  (println "cached: " (cdr (assoc 'cached-tokens r))))
```

**`cost-usd` ist eine obere Schranke:** sigoREST rechnet mit
`input_cost`/`output_cost` ohne Cache-Rabatt. Ob der Cache greift, zeigt
`cached-tokens > 0`.

Der Vorspann ist prozessweit (alle Goroutinen, auch `parfunc`) und
mutex-geschützt. Ein älteres sigoREST ohne `bare`-Unterstützung ignoriert
das Feld: dann kommen Memory und Server-Prompt wieder dazu, und die
Detailfelder bleiben 0.

`finish-reason` `"length"` mit leerem `text` heißt: Das Token-Budget wurde
vom Thinking aufgebraucht.
```

- [ ] **Step 5: Build + Referenz regenerieren**

Run: `./build.sh && ./build/golisp2 tools/gen-reference.lisp && rg -n 'sigo' docs/referenz-generiert.md`
Expected: alle 8 sigo-Symbole mit Beschreibung in `docs/referenz-generiert.md`.

- [ ] **Step 6: Vollverifikation**

Run: `go build ./... && go test ./... -count=1 && go test ./src/lib/ -run 'Sigo' -race -count=1 && ./build/golisp2 -t`
Expected: alles PASS; Lisp-Testsuite ohne neue Fehler (vorher/nachher-Zahl vergleichen: `git stash`-frei, einfach die Zusammenfassungszeile notieren).

Smoke-Test ohne Server (muss sauber mit Verbindungsfehler scheitern, nicht crashen):
Run: `GOLISP_SIGO_HOST=http://127.0.0.1:1 ./build/golisp2 -e '(list (string-length (sigo-system-prompt)) (sigo-usage))'`
Expected: Länge > 0 und `((prompt-tokens . 0) … (calls . 0))`.

- [ ] **Step 7: TODO.md** — in Abschnitt „Schritt 1“ alle vier Checkboxen auf `[x]` setzen und unter der letzten eine Zeile anfügen:

```markdown
      → erledigt 20260928: Spec `docs/superpowers/specs/2026-09-28-sigo-kontext-design.md`
        (Quelle ist die bestehende `docs/ki/referenz.md`, jetzt `src/embed/ki-referenz.md`)
```

In „5. Gerhards Gedanken“ hinter Punkt 1–3 jeweils ` → erledigt (siehe Spec)` anfügen.

**Achtung:** `TODO.md` hatte bei Sessionstart uncommittete Änderungen von Gerhard (Stand 27.09.). Vor dem Commit Gerhard fragen, ob `TODO.md` in diesen Commit darf.

- [ ] **Step 8: Commit**

```bash
git add src/embed/swank.lisp tools/gen-reference.lisp src/embed/ki-referenz.md docs/sigo.md docs/referenz-generiert.md
# TODO.md nur nach Gerhards OK:
# git add TODO.md
git commit -m "docs(sigo): Vorspann/Kosten dokumentiert, SWANK-Arglisten, Referenz neu generiert

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 8: Messung (mit Gerhard, nach Task 3 + 7)

Kein Code. Ergebnis ist ein Befund für Gerhard, kein Commit (außer optional einer Notiz in `TODO.md` Schritt 4).

**Files:**
- Temporär: `./tmp/sigo-mess/` (gitignored) — wird am Ende gelöscht.

- [ ] **Step 1: Test-sigoREST mit neuem Build starten** (Live-Dienst bleibt unberührt)

```bash
cd /u/lisp-projekte/golisp2
mkdir -p tmp/sigo-mess
cp -a /var/sigoREST tmp/sigo-mess/data
/u/go-projekte/sigoREST/build/sigoREST -http-port 19080 -https-port 19443 \
  -data-dir tmp/sigo-mess/data -comm-log tmp/sigo-mess/comm.jsonl
```

(Als Hintergrundprozess starten; bei Leserechte-Problem auf `/var/sigoREST` Gerhard fragen.)

- [ ] **Step 2: Cache-Nachweis** — zwei gleiche Calls in **einem** Prozess:

```bash
GOLISP_SIGO_HOST=http://127.0.0.1:19080 ./build/golisp2 -e '
(begin
  (sigo-usage-reset)
  (define a (sigo* "Schreibe eine golisp2-Funktion quadrat."))
  (define b (sigo* "Schreibe eine golisp2-Funktion quadrat."))
  (println "1: " (cdr (assoc (quote cached-tokens) a)) " / " (cdr (assoc (quote prompt-tokens) a)))
  (println "2: " (cdr (assoc (quote cached-tokens) b)) " / " (cdr (assoc (quote prompt-tokens) b)))
  (println (sigo-usage)))'
```

Expected: Call 2 hat `cached-tokens > 0`. Wenn nicht: im `comm.jsonl` prüfen, ob `usage.prompt_tokens_details` vom Provider überhaupt kommt (Provider-Seite) oder unterwegs verloren geht (sigoREST-Seite).

- [ ] **Step 3: Qualitäts-Stichprobe** — drei golisp2-spezifische Aufgaben, je mit und ohne Vorspann, Ergebnis in eine Datei für Gerhard:

```bash
GOLISP_SIGO_HOST=http://127.0.0.1:19080 ./build/golisp2 -e '
(begin
  (define aufgaben (list
    "Erkläre setq* in golisp2 mit einem kurzen Beispiel."
    "Zeige parfunc in golisp2 mit zwei parallelen Berechnungen."
    "Definiere in golisp2 eine Struktur punkt mit defstruct und lies x aus."))
  (define (lauf label)
    (dolist (a aufgaben)
      (println "=== " label " — " a)
      (println (sigo a))))
  (lauf "MIT Vorspann")
  (sigo-system-prompt "")
  (lauf "OHNE Vorspann")
  (println (sigo-usage)))' > tmp/sigo-mess/vergleich.txt 2>&1
```

Gerhard liest `tmp/sigo-mess/vergleich.txt` und urteilt, ob die Antworten mit Vorspann golisp2-korrekter sind. Zahlen (Thinking-Anteil, Kosten) aus `(sigo-usage)` notieren.

- [ ] **Step 4: Aufräumen** — Test-Server stoppen, dann:

```bash
rm -rf tmp/sigo-mess
```

(Enthält Sessions, DBs und alle Prompts im Klartext — Spec verlangt Löschen.)

- [ ] **Step 5: Befund an Gerhard** — Cache ja/nein mit Zahlen, Kostenvergleich, Qualitätsurteil; Vorschlag für TODO-Schritt 2 (Modellwahl) auf Basis des Thinking-Anteils.
