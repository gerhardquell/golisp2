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
  "io"
  "net"
  "os"
  "strings"
  "sync/atomic"
  "syscall"
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

// evalOverFrames sendet src und liest alle Frames bis einschließlich der
// :return-Antwort zurück (z. B. um :write-string-Events davor zu prüfen).
func evalOverFrames(t *testing.T, addr, src string) []string {
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
  var frames []string
  for {
    resp, err := readFrame(br)
    if err != nil {
      t.Fatalf("readFrame: %v", err)
    }
    s := resp.String()
    frames = append(frames, s)
    if strings.HasPrefix(s, "(:return") {
      return frames
    }
  }
}

// captureStderr leitet Datei-Deskriptor 2 für die Dauer von fn per dup2 auf
// OS-Ebene in eine Pipe um und gibt das Mitgeschnittene zurück. Bewusst
// KEINE Zuweisung an die Paketvariable os.Stderr: ServeEnv läuft in einer
// eigenen Goroutine weiter und schreibt währenddessen mit
// fmt.Fprintf(os.Stderr, …) — ein Zuweisen von os.Stderr wäre ein von
// -race erkannter Data Race auf dieser Variable. dup2 ändert nur, wohin
// Deskriptor 2 zeigt; os.Stderr (die Variable) wird nie angefasst, die
// anderen Goroutinen schreiben unverändert über denselben Deskriptor.
// fn muss blockieren, bis alles, was geprüft werden soll, bereits
// geschrieben wurde.
func captureStderr(t *testing.T, fn func()) string {
  t.Helper()
  r, w, err := os.Pipe()
  if err != nil {
    t.Fatalf("pipe: %v", err)
  }
  savedFd, err := syscall.Dup(2)
  if err != nil {
    t.Fatalf("dup stderr fd: %v", err)
  }
  if err := syscall.Dup2(int(w.Fd()), 2); err != nil {
    t.Fatalf("dup2 stderr -> pipe: %v", err)
  }
  w.Close()

  restored := false
  restore := func() {
    if restored {
      return
    }
    restored = true
    syscall.Dup2(savedFd, 2)
    syscall.Close(savedFd)
  }
  defer restore()

  fn()
  restore()

  out, err := io.ReadAll(r)
  r.Close()
  if err != nil {
    t.Fatalf("read captured stderr: %v", err)
  }
  return string(out)
}

// TestServeEnvNoRedefOnReconnect belegt Fix-Round 1, Finding 1: eine zweite
// Verbindung auf dieselbe ServeEnv-Umgebung darf swank-send-event & Co.
// nicht erneut registrieren (das würde "REDEF: ..." auf stderr ausgeben),
// und muss trotzdem ihre eigene Ausgabe erhalten (last-connected-wins).
func TestServeEnvNoRedefOnReconnect(t *testing.T) {
  env := lib.BaseEnv()
  if err := lib.LoadStdlib(env); err != nil {
    t.Fatalf("LoadStdlib: %v", err)
  }
  defer lib.ResetOutputWriter()

  exec := func(f func()) { f() }

  l, err := net.Listen("tcp", "127.0.0.1:0")
  if err != nil {
    t.Fatalf("listen: %v", err)
  }
  defer l.Close()
  go ServeEnv(l, env, exec)

  addr := l.Addr().String()

  stderr := captureStderr(t, func() {
    evalOver(t, addr, "(define reconnect-a 1)")
    evalOver(t, addr, "(define reconnect-b 2)")
  })
  if strings.Contains(stderr, "REDEF") {
    t.Fatalf("2. Verbindung hat REDEF ausgelöst, stderr:\n%s", stderr)
  }

  frames := evalOverFrames(t, addr, `(println "x")`)
  found := false
  for _, f := range frames {
    if strings.Contains(f, "x") {
      found = true
      break
    }
  }
  if !found {
    t.Fatalf("erwarte Frame mit \"x\" für die 3. (aktuelle) Verbindung, bekam: %v", frames)
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
