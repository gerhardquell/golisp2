//**********************************************************************
//  lib/swank/serve_env_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261001
//**********************************************************************

package swank

import (
  "bufio"
  "net"
  "strings"
  "sync/atomic"
  "testing"
  "time"

  "golisp2/src/lib"
)

// rexEval baut (:emacs-rex (swank:listener-eval "src") nil t id).
func rexEval(src string, id int) *lib.Cell {
  form := lib.List(lib.MakeAtom("swank:listener-eval"), lib.MakeStr(src))
  return lib.List(lib.MakeAtom(":emacs-rex"), form, lib.MakeNil(),
    lib.MakeAtom("t"), lib.MakeNum(float64(id)))
}

// evalOver sendet src und liest Frames bis zur :return-Antwort.
func evalOver(t *testing.T, addr, src string) {
  t.Helper()
  conn, err := net.Dial("tcp", addr)
  if err != nil {
    t.Fatalf("dial: %v", err)
  }
  defer conn.Close()
  if err := writeFrame(conn, rexEval(src, 1)); err != nil {
    t.Fatalf("writeFrame: %v", err)
  }
  conn.SetReadDeadline(time.Now().Add(3 * time.Second))
  br := bufio.NewReader(conn)
  for {
    resp, err := readFrame(br)
    if err != nil {
      t.Fatalf("readFrame: %v", err)
    }
    if strings.HasPrefix(resp.String(), "(:return") {
      return
    }
  }
}

func TestServeEnvSharesEnvAndUsesExec(t *testing.T) {
  env := lib.BaseEnv()
  if err := lib.LoadStdlib(env); err != nil {
    t.Fatalf("LoadStdlib: %v", err)
  }
  defer lib.ResetOutputWriter()

  var calls atomic.Int32
  exec := func(f func()) { calls.Add(1); f() }

  l, err := net.Listen("tcp", "127.0.0.1:0")
  if err != nil {
    t.Fatalf("listen: %v", err)
  }
  defer l.Close()
  go ServeEnv(l, env, exec)

  evalOver(t, l.Addr().String(), "(define shared-x 7)")
  evalOver(t, l.Addr().String(), "(define shared-y (+ shared-x 1))")

  y, err := env.Get("shared-y")
  if err != nil {
    t.Fatalf("shared-y fehlt in gemeinsamer Env: %v", err)
  }
  if y.Num != 8 {
    t.Fatalf("shared-y = %s, want 8", y)
  }
  if calls.Load() < 2 {
    t.Fatalf("exec %d-mal aufgerufen, want >= 2", calls.Load())
  }
}
