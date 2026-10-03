# sigo-request, sigo*/sigo-usage als Hash-Tabelle, Kostenabfragen — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** sigoREST aus golisp2 frei ansprechbar machen (`sigo-request`), `sigo*`/`sigo-usage` als Hash-Tabelle mit Antwortzeit liefern und Budget, Kosten und Preise aus Lisp abfragbar machen.

**Architecture:** `src/lib/sigorest.go` bleibt der einzige HTTP-Weg zu sigoREST: ein gemeinsamer Helfer `sigoDo` macht jeden HTTP-Aufruf, `sigoRequestCell` ist der gemeinsame Chat-Pfad für `sigo-request`, `sigo` und `sigo*`. Request und Antwort laufen über die vorhandene JSON-Abbildung `jsoncell.go` (Objekt ↔ Hash-Tabelle mit `:test 'equal`). Kleine Hash-Helfer kommen nach `hashtable.go`.

**Tech Stack:** Go 1.x (`net/http`, `net/http/httptest`, `context`, `net/url`), golisp2-Cells.

**Spec:** `docs/superpowers/specs/2026-10-03-sigo-request-design.md`

## Global Constraints

- Einrückung 2 Spaces, keine Tabs; Kommentare deutsch, sparsam.
- Fehlertexte: `fmt.Errorf("funktionsname: beschreibung")`.
- Kein zweiter HTTP-Client gegen sigoREST außerhalb `src/lib/sigorest.go`.
- Keine zweite JSON-Abbildung: nur `CellToJSON`/`JSONToCell` aus `src/lib/jsoncell.go`.
- Request-Whitelist exakt: `model messages temperature max_tokens session_id timeout retries system_prompt bare channel`; `stream` gesperrt.
- Datei-Header neuer Dateien: Autor Gerhard Quell, CoAutor = **das schreibende Modell selbst**, Erstellt 20261003. Bestehende Header nie ändern.
- Commits: Trailer `Co-Authored-By: <exakte eigene Modellbezeichnung> <noreply@anthropic.com>` — das schreibende Modell trägt sich **selbst** ein, nichts aus diesem Plan abschreiben.
- Temporäre Dateien nur unter `./tmp/`, nie `/tmp`.
- Build: `./build.sh`; Tests: `cd src && go test ./... -count=1`; Lisp-Suite: `./build/golisp2 -t`.
- Hinweis Laufzeit: jeder erfolgreiche Chat-Call wartet am Rate-Limiter (`sigoRateLimiter`, Tick 2 s). Tests mit echten Fake-Calls dauern daher einige Sekunden — das ist erwartet.

---

### Task 1: `sigo-request` — gemeinsamer Chat-Pfad, Whitelist, Defaults, `elapsed`

**Files:**
- Modify: `src/lib/hashtable.go` (drei Methoden + Konstruktor ans Dateiende)
- Modify: `src/lib/jsoncell.go` (nutzt die neuen Helfer)
- Modify: `src/lib/sigorest.go` (neue Funktionen, Registrierung; Altcode bleibt in dieser Task stehen)
- Test: `src/lib/sigorest_request_test.go` (neu)

**Interfaces:**
- Produces (hashtable.go):
  - `func newStringHash() *Cell` — leere HASHTABLE-Cell, `test: "equal"`
  - `func (ht *HashTable) getStr(key string) (*Cell, bool)`
  - `func (ht *HashTable) putStr(key string, val *Cell)`
  - `func (ht *HashTable) snapshot() []hashEntry`
- Produces (sigorest.go):
  - `func hashAt(c *Cell, key string) *Cell` — Wert oder nil
  - `func numAt(c *Cell, key string) float64` — Zahl oder 0
  - `func sigoDo(fname, method, url string, body []byte, timeout time.Duration) ([]byte, time.Duration, error)`
  - `func sigoRequestCell(fname string, h *Cell, host string) (*Cell, error)`
  - `func sigoUsageFromResponse(resp *Cell) sigoUsage`
  - `sigoUsage` bekommt Feld `Elapsed float64`
  - Lisp: `(sigo-request h [host])`
- Produces (Test): `func fakeSigoWith(t *testing.T, status int, delay time.Duration, got *map[string]interface{}) *httptest.Server`, `func reqHash(kv ...interface{}) *Cell`, `func userMsgs(text string) *Cell`

- [ ] **Step 1: Hash-Helfer in `src/lib/hashtable.go` ans Dateiende anfügen**

```go
// newStringHash: leere Hash-Tabelle mit :test equal — Form der
// JSON-Objekte (jsoncell.go) und der sigo-Ergebnisse.
func newStringHash() *Cell {
  return &Cell{Type: HASHTABLE, Ht: &HashTable{m: make(map[string]hashEntry), test: "equal"}}
}

// getStr liest den Wert zum String-Key.
func (ht *HashTable) getStr(key string) (*Cell, bool) {
  k := MakeStr(key)
  ht.mu.RLock()
  e, ok := ht.m[ht.keyOf(k)]
  ht.mu.RUnlock()
  if !ok {
    return nil, false
  }
  return e.val, true
}

// putStr setzt den Wert zum String-Key.
func (ht *HashTable) putStr(key string, val *Cell) {
  k := MakeStr(key)
  ht.mu.Lock()
  ht.m[ht.keyOf(k)] = hashEntry{key: k, val: val}
  ht.mu.Unlock()
}

// snapshot: alle Einträge, unter Lock kopiert.
func (ht *HashTable) snapshot() []hashEntry {
  ht.mu.RLock()
  defer ht.mu.RUnlock()
  entries := make([]hashEntry, 0, len(ht.m))
  for _, e := range ht.m {
    entries = append(entries, e)
  }
  return entries
}
```

- [ ] **Step 2: `src/lib/jsoncell.go` auf die Helfer umstellen**

In `hashToJSONObject` die sechs Zeilen

```go
  ht.mu.RLock()
  entries := make([]hashEntry, 0, len(ht.m))
  for _, e := range ht.m {
    entries = append(entries, e)
  }
  ht.mu.RUnlock()
```

ersetzen durch

```go
  entries := ht.snapshot()
```

In `jsonValueToCell` den Zweig `case map[string]interface{}:` ersetzen durch

```go
  case map[string]interface{}:
    obj := newStringHash()
    for k, raw := range x {
      val, err := jsonValueToCell(raw, depth+1)
      if err != nil {
        return nil, err
      }
      obj.Ht.putStr(k, val)
    }
    return obj, nil
```

Run: `cd src && go test ./lib/ -run JSON -count=1`
Expected: PASS (reine Umstellung, Verhalten gleich).

- [ ] **Step 3: Failing Tests schreiben — `src/lib/sigorest_request_test.go`**

```go
//**********************************************************************
//  lib/sigorest_request_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : <schreibendes Modell>
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
```

`fakeSigoResponse` und `withSigoState` stehen schon in `src/lib/sigorest_test.go`.

- [ ] **Step 4: Tests laufen lassen — müssen scheitern**

Run: `cd src && go test ./lib/ -run TestSigoRequest -count=1`
Expected: Build-Fehler `undefined: sigoRequestCell` (bzw. `hashAt`, `numAt`).

- [ ] **Step 5: Implementierung in `src/lib/sigorest.go`**

Import-Block ergänzen um `"math"`.

`sigoUsage` und `add` ersetzen durch:

```go
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
```

In `RegisterSigo` ergänzen:

```go
  _ = env.Set("sigo-request",       makeFn(fnSigoRequest))
```

Den Drossel-Block aus `sigoRequest` (Rate-Limiter + 500 ms) in eine Funktion ziehen und in `sigoRequest` durch `sigoThrottle()` ersetzen:

```go
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
```

Neue Funktionen (vor `sigoCallToHost` einfügen):

```go
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
func sigoBuildRequest(h *Cell, systemPrompt string) (*Cell, time.Duration, error) {
  if h == nil || h.Type != HASHTABLE {
    return nil, 0, fmt.Errorf("sigo-request: Hash-Tabelle erwartet, got %s", h)
  }
  req := newStringHash()
  for _, e := range h.Ht.snapshot() {
    if e.key.Type != STRING {
      return nil, 0, fmt.Errorf("sigo-request: Key muss String sein, got %s", e.key)
    }
    k := e.key.Val
    if k == "stream" {
      return nil, 0, fmt.Errorf("sigo-request: 'stream' ist gesperrt (sigoREST bucht Streaming-Kosten falsch)")
    }
    if !sigoRequestFields[k] {
      return nil, 0, fmt.Errorf("sigo-request: Feld '%s' kennt sigoREST nicht", k)
    }
    req.Ht.putStr(k, e.val)
  }
  if m, ok := req.Ht.getStr("model"); !ok || m.Type != STRING {
    return nil, 0, fmt.Errorf("sigo-request: 'model' (String) fehlt")
  }
  if m, ok := req.Ht.getStr("messages"); !ok || m.Type != LIST {
    return nil, 0, fmt.Errorf("sigo-request: 'messages' (Liste) fehlt")
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
      return nil, 0, fmt.Errorf("sigo-request: 'timeout' muss Zahl > 0 sein, got %s", v)
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

  req, timeout, err := sigoBuildRequest(h, systemPrompt)
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
```

- [ ] **Step 6: Tests laufen lassen — müssen grün sein**

Run: `cd src && go build ./... && go test ./lib/ -run 'TestSigo|JSON' -count=1`
Expected: PASS (neue und alte sigo-Tests).

- [ ] **Step 7: Volle Suite**

Run: `cd src && go test ./... -count=1`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add src/lib/hashtable.go src/lib/jsoncell.go src/lib/sigorest.go src/lib/sigorest_request_test.go
git commit -m "feat(sigo): sigo-request mit Whitelist, Defaults und elapsed

Co-Authored-By: <eigenes Modell> <noreply@anthropic.com>"
```

---

### Task 2: `sigo`, `sigo*`, `sigo-usage` auf `sigo-request`; Altpfad entfernen

**Files:**
- Modify: `src/lib/sigorest.go` (`sigoRequest`, `fnSigo`, `sigoAlist`, `sigoUsageCells`, `fnSigoStar`, `fnSigoUsage`, `sigoResult`, `sigoCallToHost` ersetzen/entfernen)
- Modify: `src/lib/sigorest_test.go` (Tests auf Hash-Tabelle umstellen, Altpfad-Tests ersetzen)

**Interfaces:**
- Consumes (Task 1): `sigoRequestCell`, `hashAt`, `numAt`, `sigoUsageFromResponse`, `newStringHash`, `getStr`, `putStr`, `fakeSigoWith`
- Produces:
  - `func sigoPromptRequest(fname string, args []*Cell) (*Cell, string, error)` — Antwort-Hash und angefragtes Modell
  - `func sigoFirstChoice(resp *Cell) (text, finish string)`
  - `func sigoUsageHash(u sigoUsage) *Cell`
  - Lisp: `sigo*` → Hash mit Keys `text model finish-reason prompt-tokens completion-tokens cached-tokens reasoning-tokens cost-usd elapsed`; `sigo-usage` → dieselben Token-/Kosten-Keys plus `elapsed` und `calls`

- [ ] **Step 1: Tests in `src/lib/sigorest_test.go` umstellen (failing)**

Diese Funktionen **löschen**: `TestSigoCallToHost_SendsBareAndSystemPrompt`, `TestSigoCallToHost_EmptySystemPromptOmitsField`, `alistGet`, `TestSigoStar_ReturnsAlist`, `TestSigoStar_Lisp`, `TestSigoUsage_SumsAndResets`, `TestSigoStar_ParallelCallsAreRaceFree`.

Und an ihrer Stelle einfügen:

```go
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
```

`TestSigo_LispSendsCurrentSystemPrompt`, `TestSigoSystemPrompt_Lisp`, `TestSigoState_ConcurrentAccess`, `TestKiReferenzEmbedded`, `fakeSigo`, `withSigoState`, `sigoArgs`, `fakeSigoResponse` bleiben unverändert.

- [ ] **Step 2: Tests laufen lassen — müssen scheitern**

Run: `cd src && go test ./lib/ -run 'TestSigo' -count=1`
Expected: FAIL (`TestSigoStar_ReturnsHash`: „Hash-Tabelle erwartet“, `TestSigo_TypeErrors`: `(sigo 5)` liefert keinen Fehler; `TestSigoUsage_*`: Alist statt Hash).

- [ ] **Step 3: Implementierung in `src/lib/sigorest.go`**

**Entfernen:** `type sigoResult`, `func sigoRequest`, `func sigoAlist`, `func sigoUsageCells`, `func sigoCallToHost` (komplett).

`fnSigo`, `fnSigoStar`, `fnSigoUsage` ersetzen und neue Helfer einfügen:

```go
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
```

`fnSigoUsageReset` bleibt (setzt `sigoUsageSum = sigoUsage{}`, damit auch `Elapsed`). Nach dem Entfernen ungenutzte Imports (`encoding/json` wird noch von `fnSigoModels` gebraucht — erst Task 3 entfernt ihn) per `go build` prüfen.

- [ ] **Step 4: Tests laufen lassen — müssen grün sein**

Run: `cd src && go build ./... && go test ./lib/ -run 'TestSigo' -count=1`
Expected: PASS.

- [ ] **Step 5: Volle Suite + Lisp-Suite**

Run: `./build.sh && (cd src && go test ./... -count=1) && ./build/golisp2 -t 2>&1 | tail -1`
Expected: Go PASS; Lisp-Report `0 FAIL`.

- [ ] **Step 6: Commit**

```bash
git add src/lib/sigorest.go src/lib/sigorest_test.go
git commit -m "feat(sigo)!: sigo/sigo*/sigo-usage über sigo-request, Hash-Tabellen

BREAKING CHANGE: sigo* und sigo-usage liefern Hash-Tabellen statt
Alists (neu: elapsed). sigo/sigo* verlangen String-Argumente.

Co-Authored-By: <eigenes Modell> <noreply@anthropic.com>"
```

---

### Task 3: `sigo-budget`, `sigo-costs`, `sigo-model-info`; `sigo-models` über `sigoDo`

**Files:**
- Modify: `src/lib/sigorest.go` (neue Funktionen, Registrierung, `fnSigoModels` ersetzen, Imports)
- Test: `src/lib/sigorest_api_test.go` (neu)

**Interfaces:**
- Consumes (Task 1): `sigoDo`, `hashAt`, `sigoGetHost` (vorhanden), `unixFloatToTime` (aus `src/lib/timefuncs.go`)
- Produces:
  - `func sigoGetJSON(fname, path string) (*Cell, error)`
  - Lisp: `(sigo-budget)`, `(sigo-costs [seit [bis]])`, `(sigo-model-info [modell])`, `(sigo-models)` (Verhalten wie bisher, aber HTTP-Status geprüft)

- [ ] **Step 1: Failing Tests — `src/lib/sigorest_api_test.go`**

```go
//**********************************************************************
//  lib/sigorest_api_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : <schreibendes Modell>
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261003
//**********************************************************************
// Tests für sigo-budget, sigo-costs, sigo-model-info, sigo-models gegen
// einen Fake-sigoREST (GET-Endpunkte).
//**********************************************************************

package lib

import (
  "io"
  "net/http"
  "net/http/httptest"
  "net/url"
  "strings"
  "sync"
  "testing"
  "time"
)

// fakeSigoAPI: Fake-sigoREST, Antwort je Pfad; unbekannter Pfad → 404.
// Die letzte Query landet in *q. Setzt (sigo-host) auf den Fake.
func fakeSigoAPI(t *testing.T, routes map[string]string, q *url.Values, mu *sync.Mutex) *httptest.Server {
  t.Helper()
  srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if q != nil {
      mu.Lock()
      *q = r.URL.Query()
      mu.Unlock()
    }
    body, ok := routes[r.URL.Path]
    if !ok {
      http.Error(w, "nicht gefunden", http.StatusNotFound)
      return
    }
    w.Header().Set("Content-Type", "application/json")
    io.WriteString(w, body)
  }))
  t.Cleanup(srv.Close)
  withSigoState(t)
  if _, err := fnSigoHost([]*Cell{MakeStr(srv.URL)}); err != nil {
    t.Fatal(err)
  }
  return srv
}

const fakeModels = `[{"id":"ci-gpt-6-luna","shortcode":"che-gpt6-l88","input_cost":0.065,"output_cost":0.325},` +
  `{"id":"minimax-m3","shortcode":"mam-mmxm3","input_cost":0,"output_cost":0}]`

func TestSigoBudget(t *testing.T) {
  fakeSigoAPI(t, map[string]string{"/api/budget": `{"status":{"daily_spend_usd":1.4,"blocked":false}}`}, nil, nil)
  evalEq(t, `(gethash "daily_spend_usd" (gethash "status" (sigo-budget)))`, "1.4")
  evalErr(t, `(sigo-budget 1)`)
}

func TestSigoCosts(t *testing.T) {
  var q url.Values
  var mu sync.Mutex
  fakeSigoAPI(t, map[string]string{"/api/costs": `{"requests":3,"total_cost_usd":0.5}`}, &q, &mu)

  evalEq(t, `(gethash "requests" (sigo-costs))`, "3")
  mu.Lock()
  if len(q) != 0 {
    t.Errorf("ohne Argument keine Query erwartet, got %v", q)
  }
  mu.Unlock()

  evalEq(t, `(gethash "total_cost_usd" (sigo-costs 1791046147 1791049747))`, "0.5")
  mu.Lock()
  since, until := q.Get("since"), q.Get("until")
  mu.Unlock()
  if want := time.Unix(1791046147, 0).Format(time.RFC3339); since != want {
    t.Errorf("since: erwartet %s, got %s", want, since)
  }
  if want := time.Unix(1791049747, 0).Format(time.RFC3339); until != want {
    t.Errorf("until: erwartet %s, got %s", want, until)
  }

  evalErr(t, `(sigo-costs "heute")`)
  evalErr(t, `(sigo-costs 1 2 3)`)
}

func TestSigoModelInfo(t *testing.T) {
  fakeSigoAPI(t, map[string]string{"/api/models": fakeModels}, nil, nil)
  evalEq(t, `(length (sigo-model-info))`, "2")
  evalEq(t, `(gethash "input_cost" (sigo-model-info "ci-gpt-6-luna"))`, "0.065")
  evalEq(t, `(gethash "id" (sigo-model-info "mam-mmxm3"))`, `"minimax-m3"`)
  evalErr(t, `(sigo-model-info "gibtsnicht")`)
  evalErr(t, `(sigo-model-info 5)`)
}

func TestSigoModels(t *testing.T) {
  fakeSigoAPI(t, map[string]string{"/v1/models": `{"data":[{"id":"a"},{"id":"b"}]}`}, nil, nil)
  evalEq(t, `(sigo-models)`, `("a" "b")`)
}

func TestSigoAPI_HTTPError(t *testing.T) {
  fakeSigoAPI(t, map[string]string{}, nil, nil) // jeder Pfad → 404
  for _, src := range []string{`(sigo-budget)`, `(sigo-costs)`, `(sigo-model-info)`, `(sigo-models)`} {
    _, err := evalStr(src)
    if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
      t.Errorf("%s: HTTP-404-Fehler erwartet, got %v", src, err)
    }
  }
}
```

- [ ] **Step 2: Tests laufen lassen — müssen scheitern**

Run: `cd src && go test ./lib/ -run 'TestSigoBudget|TestSigoCosts|TestSigoModelInfo|TestSigoModels|TestSigoAPI' -count=1`
Expected: FAIL (`unbekanntes Symbol 'sigo-budget'` u. a.; `TestSigoAPI_HTTPError` für `sigo-models` ohne „HTTP 404“).

- [ ] **Step 3: Implementierung in `src/lib/sigorest.go`**

Import-Block: `"net/url"` ergänzen; `"encoding/json"` entfernen (nach dem Ersetzen von `fnSigoModels` ungenutzt — `go build` bestätigt).

In `RegisterSigo` ergänzen:

```go
  _ = env.Set("sigo-budget",        makeFn(fnSigoBudget))
  _ = env.Set("sigo-costs",         makeFn(fnSigoCosts))
  _ = env.Set("sigo-model-info",    makeFn(fnSigoModelInfo))
```

`fnSigoModels` ersetzen und neue Funktionen anfügen:

```go
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
```

- [ ] **Step 4: Tests laufen lassen — müssen grün sein**

Run: `cd src && go build ./... && go test ./lib/ -run 'TestSigo' -count=1`
Expected: PASS.

- [ ] **Step 5: Volle Suite**

Run: `cd src && go test ./... -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add src/lib/sigorest.go src/lib/sigorest_api_test.go
git commit -m "feat(sigo): sigo-budget, sigo-costs, sigo-model-info; sigo-models prüft HTTP-Status

Co-Authored-By: <eigenes Modell> <noreply@anthropic.com>"
```

---

### Task 4: Doku und Live-Abnahme

**Files:**
- Modify: `src/embed/ki-referenz.md` (sigoREST-Zeile)
- Modify: `tools/gen-reference.lisp` (`*ref-docs*`-Einträge), danach `docs/referenz-generiert.md` neu erzeugen
- Modify: `src/embed/swank.lisp` (Arglisten + Docstrings)
- Modify: `docs/sigo.md` (Abschnitt „Vorspann, Kontext und Kosten“ und neuer Abschnitt „Messen und Kosten“)

**Interfaces:**
- Consumes: alle Lisp-Funktionen aus Task 1–3.

- [ ] **Step 1: `src/embed/ki-referenz.md`** — die Zeile, die mit `- **sigoREST:**` beginnt, komplett ersetzen durch:

```markdown
- **sigoREST:** `sigo sigo* sigo-request sigo-models sigo-model-info sigo-costs sigo-budget sigo-host sigo-system-prompt sigo-reference sigo-usage sigo-usage-reset` — `(sigo "prompt" [model] [session-id] [host])` → Text; `sigo*` → Hash-Tabelle (`text model finish-reason prompt-tokens completion-tokens cached-tokens reasoning-tokens cost-usd elapsed`); `(sigo-request h [host])`: Hash mit genau `model messages temperature max_tokens session_id timeout retries system_prompt bare channel` (anderes → Fehler), Ergebnis = volle Antwort als Hash plus `elapsed` (s); `(sigo-usage)` Summen inkl. `calls`/`elapsed`; `(sigo-costs [seit [bis]])` (Unix-Sekunden wie `(now)`), `(sigo-budget)`, `(sigo-model-info [modell])` (Preise USD/1 Mio. Tokens, **0 = unbekannt, nicht gratis**); `(sigo-system-prompt "")` → Modell sieht nur den Prompt (plus Session-Verlauf, falls session-id)
```

- [ ] **Step 2: `tools/gen-reference.lisp`** — Einträge in `*ref-docs*` (alphabetisch, zwischen den vorhandenen `sigo…`-Zeilen):

Ersetzen:
```lisp
    (sigo* . "Wie sigo, Ergebnis als Assoc-Liste (text, model, finish-reason, prompt-/completion-/cached-/reasoning-tokens, cost-usd)")
```
durch:
```lisp
    (sigo* . "Wie sigo, Ergebnis als Hash-Tabelle (text, model, finish-reason, Tokens, cost-usd, elapsed)")
    (sigo-budget . "Tageslimit und -verbrauch von sigoREST (/api/budget) als Hash-Tabelle")
    (sigo-costs . "Kosten seit/bis (Unix-Sekunden) aus /api/costs als Hash-Tabelle")
```
und vor `(sigo-reference . …)` einfügen:
```lisp
    (sigo-model-info . "Modell-Infos aus /api/models (Preise USD/1M Tokens, 0 = unbekannt), alle oder per id/shortcode")
```
und nach `(sigo-reference . …)` einfügen:
```lisp
    (sigo-request . "Chat-Request als Hash-Tabelle (Whitelist), Antwort als Hash-Tabelle plus elapsed")
```
und ersetzen:
```lisp
    (sigo-usage . "Summe der sigo-Tokens und -Kosten seit Start, plus calls")
```
durch:
```lisp
    (sigo-usage . "Summe der sigo-Tokens, -Kosten und -Antwortzeit seit Start/Reset, plus calls")
```

Run: `./build.sh && ./build/golisp2 tools/gen-reference.lisp | head -1`
Expected: Zeile „… Symbole nach docs/referenz-generiert.md geschrieben (…, 3 leer)“ — die Zahl der leeren Einträge darf nicht steigen.

- [ ] **Step 3: `src/embed/swank.lisp`**

Nach `    ("sigo*" . "(sigo* prompt &optional model session-id host)")` einfügen:
```lisp
    ("sigo-request" . "(sigo-request request &optional host)")
    ("sigo-costs" . "(sigo-costs &optional since until)")
    ("sigo-budget" . "(sigo-budget)")
    ("sigo-model-info" . "(sigo-model-info &optional model)")
```
Ersetzen:
```lisp
    ("sigo*" . "Wie sigo, liefert Assoc-Liste mit Text, Tokens, Cache und Kosten.")
```
durch:
```lisp
    ("sigo*" . "Wie sigo, liefert Hash-Tabelle mit Text, Tokens, Cache, Kosten und Antwortzeit.")
    ("sigo-request" . "Freier Chat-Request an sigoREST als Hash-Tabelle; Antwort als Hash-Tabelle.")
```

- [ ] **Step 4: `docs/sigo.md`**

In der Tabelle des Abschnitts „Vorspann, Kontext und Kosten“ die Zeilen für `sigo*` und `sigo-usage` ersetzen durch:
```markdown
| `(sigo* prompt …)` | wie `sigo`, aber Hash-Tabelle: `text model finish-reason prompt-tokens completion-tokens cached-tokens reasoning-tokens cost-usd elapsed` |
| `(sigo-usage)` | Summen seit Start/Reset (Tokens, `cost-usd`, `elapsed`) plus `calls` |
```
Das Beispiel darunter ersetzen durch:
```lisp
(let ((r (sigo* "Schreibe eine Funktion quadrat.")))
  (println (gethash "text" r))
  (println "cached: " (gethash "cached-tokens" r) "  Sekunden: " (gethash "elapsed" r)))
```

Am Dateiende neuen Abschnitt anfügen:
````markdown
## Messen und Kosten

`(sigo-request h [host])` schickt einen frei gebauten Chat-Request. `h` ist
eine Hash-Tabelle mit **genau** den Feldern, die sigoREST kennt:
`model messages temperature max_tokens session_id timeout retries
system_prompt bare channel`. Jedes andere Feld ist ein Fehler — sigoREST
würde es still verwerfen (z. B. `response_format`). `stream` ist gesperrt,
solange sigoREST Streaming-Kosten falsch bucht. Fehlen `bare`,
`system_prompt` oder `timeout`, setzt golisp2 die Defaults (`t`, aktueller
Vorspann, `GOLISP_SIGO_TIMEOUT`). Ergebnis: die volle Antwort als
Hash-Tabelle plus `"elapsed"` — Sekunden nur für den HTTP-Aufruf, ohne
golisp2s eigene Drossel.

```lisp
(let ((h (make-hash-table :test 'equal)))
  (puthash "model" h "ci-gpt-6-luna")
  (puthash "messages" h (list (json-parse "{\"role\":\"user\",\"content\":\"2+2?\"}")))
  (puthash "max_tokens" h 50)
  (puthash "temperature" h 0)
  (puthash "timeout" h 300)
  (let ((r (sigo-request h)))
    (list (gethash "elapsed" r) (gethash "cost_usd" (gethash "usage" r)))))
```

Kosten und Budget:

| Aufruf | Wirkung |
|---|---|
| `(sigo-usage)` / `(sigo-usage-reset)` | Summen dieses Prozesses (Tokens, `cost-usd`, `elapsed`, `calls`) |
| `(sigo-costs [seit [bis]])` | `/api/costs` — serverweit, `seit`/`bis` in Unix-Sekunden wie `(now)`; ohne Argument: heute |
| `(sigo-budget)` | `/api/budget` — Tageslimit, Verbrauch, `blocked` |
| `(sigo-model-info [modell])` | `/api/models` — alle Modelle oder eins per `id`/`shortcode`; `input_cost`/`output_cost` in USD pro 1 Mio. Tokens |

**Preis 0 heißt „unbekannt“, nicht „gratis“.** Rund die Hälfte der Modelle
hat in sigoREST keinen Preis; für sie ist `cost-usd` immer 0. Wer sparen
will, prüft vorher `(sigo-model-info m)`. Messfenster:

```lisp
(let ((t0 (now)))
  (sigo* "…" "ci-gpt-6-luna")
  (gethash "total_cost_usd" (sigo-costs t0)))
```

HTTP 402 von sigoREST (Budget-Stopp) kommt als Fehler
`sigo-request: HTTP 402: …` an.
````

- [ ] **Step 5: Suiten**

Run: `./build.sh && (cd src && go test ./... -count=1) && ./build/golisp2 -t 2>&1 | tail -1`
Expected: Go PASS; Lisp-Report `0 FAIL`.

- [ ] **Step 6: Live-Abnahme gegen sigoREST** (nur wenn `curl -s -m 3 http://127.0.0.1:9080/ping` antwortet; sonst überspringen und das melden)

Billiges Modell, winziger Prompt (Kosten < 0,001 USD):

```bash
./build/golisp2 -e '(let ((t0 (now)) (r (sigo* "Antworte nur mit: ok" "ci-gpt-6-luna")))
  (list (gethash "text" r) (gethash "elapsed" r) (gethash "cost-usd" r)
        (gethash "input_cost" (sigo-model-info "ci-gpt-6-luna"))
        (gethash "blocked" (gethash "status" (sigo-budget)))))'
```

Expected: Liste mit Text (enthält „ok“), `elapsed` > 0, `cost-usd` > 0, `0.065`, `()`. Ausgabe im Bericht festhalten.

- [ ] **Step 7: Commit**

```bash
git add src/embed/ki-referenz.md tools/gen-reference.lisp docs/referenz-generiert.md src/embed/swank.lisp docs/sigo.md
git commit -m "docs(sigo): sigo-request, Hash-Ergebnisse, Kostenabfragen

Co-Authored-By: <eigenes Modell> <noreply@anthropic.com>"
```
