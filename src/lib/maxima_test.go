//**********************************************************************
//  lib/maxima_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261118
//**********************************************************************
// maxima-close ist idempotent, closed wird nur unter sess.mu gelesen
// (mit -race prüfen). Ohne installiertes maxima: übersprungen.
//**********************************************************************

package lib

import (
  "os/exec"
  "strings"
  "sync"
  "testing"
)

func TestMaximaCloseNebenlaeufigUndIdempotent(t *testing.T) {
  if _, err := exec.LookPath(maximaBinary); err != nil {
    t.Skip("maxima nicht installiert")
  }
  sess, err := fnMaximaOpen(nil)
  if err != nil {
    t.Fatalf("maxima-open: %v", err)
  }
  var wg sync.WaitGroup
  for i := 0; i < 4; i++ {
    wg.Add(1)
    go func() {
      defer wg.Done()
      // vor oder nach dem close — beides erlaubt, nur kein Rennen
      _, _ = fnMaximaEval([]*Cell{sess, MakeStr("1+1")})
    }()
  }
  wg.Add(1)
  go func() {
    defer wg.Done()
    if _, err := fnMaximaClose([]*Cell{sess}); err != nil {
      t.Errorf("maxima-close: %v", err)
    }
  }()
  wg.Wait()

  res, err := fnMaximaClose([]*Cell{sess})
  if err != nil || res != cellT {
    t.Errorf("zweites maxima-close = %v, %v; want t", res, err)
  }
  _, err = fnMaximaEval([]*Cell{sess, MakeStr("1+1")})
  if err == nil || !strings.Contains(err.Error(), "Session bereits geschlossen") {
    t.Errorf("maxima-eval nach close: err = %v", err)
  }
}
