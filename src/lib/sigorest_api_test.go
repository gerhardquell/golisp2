//**********************************************************************
//  lib/sigorest_api_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-sonnet-5.5
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
  // length liegt in der Stdlib, nicht in BaseEnv: Anzahl über car/cdr prüfen
  evalEq(t, `(gethash "id" (car (cdr (sigo-model-info))))`, `"minimax-m3"`)
  evalEq(t, `(cdr (cdr (sigo-model-info)))`, "()")
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
