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

// waitForSetup wartet, bis ServeEnvs einmaliges Setup (RegisterSwankEnv)
// gelaufen ist: swank-send-event ist erst danach in env gebunden. Env.Get
// ist mutex-geschützt (env.go), daher kein Data Race mit der parallel
// laufenden ServeEnv-Goroutine.
func waitForSetup(t *testing.T, env *lib.Env) {
  t.Helper()
  deadline := time.Now().Add(3 * time.Second)
  for {
    if _, err := env.Get("swank-send-event"); err == nil {
      return
    }
    if time.Now().After(deadline) {
      t.Fatal("ServeEnv-Setup (swank-send-event) nicht rechtzeitig sichtbar")
    }
    time.Sleep(time.Millisecond)
  }
}

// captureStdout leitet Datei-Deskriptor 1 für die Dauer von fn per dup2 auf
// OS-Ebene in eine Pipe um (gleiche Technik wie captureStderr, s.u.).
func captureStdout(t *testing.T, fn func()) string {
  t.Helper()
  r, w, err := os.Pipe()
  if err != nil {
    t.Fatalf("pipe: %v", err)
  }
  savedFd, err := syscall.Dup(1)
  if err != nil {
    t.Fatalf("dup stdout fd: %v", err)
  }
  if err := syscall.Dup2(int(w.Fd()), 1); err != nil {
    t.Fatalf("dup2 stdout -> pipe: %v", err)
  }
  w.Close()

  restored := false
  restore := func() {
    if restored {
      return
    }
    restored = true
    syscall.Dup2(savedFd, 1)
    syscall.Close(savedFd)
  }
  defer restore()

  fn()
  restore()

  out, err := io.ReadAll(r)
  r.Close()
  if err != nil {
    t.Fatalf("read captured stdout: %v", err)
  }
  return string(out)
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

// TestServeEnvOutputFallsBackToStdoutBeforeConnect belegt Finding 1: solange
// noch keine Emacs-Verbindung angenommen wurde (sender.current == nil),
// muss (println ...) trotzdem sichtbar sein — über den os.Stdout-Fallback
// in swankSender.send, nicht stillschweigend verschluckt werden.
func TestServeEnvOutputFallsBackToStdoutBeforeConnect(t *testing.T) {
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

  waitForSetup(t, env)

  var loadErr error
  stdout := captureStdout(t, func() {
    _, loadErr = lib.LoadString(`(println "pre-connect")`, env)
  })
  if loadErr != nil {
    t.Fatalf("LoadString: %v", loadErr)
  }
  if !strings.Contains(stdout, "pre-connect") {
    t.Fatalf("stdout = %q, want it to contain %q", stdout, "pre-connect")
  }
}

// TestServeEnvOutputFallsBackToStdoutAfterDisconnect belegt Finding 1b:
// nach dem Schließen der einzigen Verbindung darf (println ...) nicht mehr
// mit "use of closed network connection" fehlschlagen — sender.current
// muss auf nil zurückgesetzt worden sein, Ausgabe fällt auf stdout zurück.
func TestServeEnvOutputFallsBackToStdoutAfterDisconnect(t *testing.T) {
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
  evalOver(t, addr, "(define discon-probe 1)")

  conn, err := net.Dial("tcp", addr)
  if err != nil {
    t.Fatalf("dial: %v", err)
  }
  if err := writeFrame(conn, rexEval("(define discon-probe-2 2)", 2)); err != nil {
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
      break
    }
  }
  conn.Close()

  deadline := time.Now().Add(3 * time.Second)
  var stdout string
  for {
    var loadErr error
    stdout = captureStdout(t, func() {
      _, loadErr = lib.LoadString(`(println "post-disconnect")`, env)
    })
    if loadErr == nil {
      break
    }
    if time.Now().After(deadline) {
      t.Fatalf("println blieb nach Disconnect fehlerhaft: %v", loadErr)
    }
    time.Sleep(10 * time.Millisecond)
  }
  if !strings.Contains(stdout, "post-disconnect") {
    t.Fatalf("stdout = %q, want it to contain %q", stdout, "post-disconnect")
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
  if calls.Load() != 3 {
    t.Fatalf("exec %d-mal aufgerufen, want 3 (1 Setup + 2 Nachrichten)", calls.Load())
  }
}
