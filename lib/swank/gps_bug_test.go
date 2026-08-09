//**********************************************************************
//  lib/swank/gps_bug_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k2.7-code
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260716
//**********************************************************************
// Integrationstest: der SWANK-Server (golisp2 --swank) überlebt das Laden
// von gps-norvig-bugs.lisp. Lädt das Skript per golisp2-client --load und
// prüft, ob der Server danach noch auf Signal 0 antwortet (also lebt).
//**********************************************************************

package swank

import (
  "fmt"
  "net"
  "os"
  "os/exec"
  "path/filepath"
  "runtime"
  "syscall"
  "testing"
  "time"
)

func TestSwankSurvivesNorvigBugs(t *testing.T) {
  if runtime.GOOS != "linux" {
    t.Skip("nur auf Linux")
  }

  repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
  if err != nil {
    t.Fatalf("repo root: %v", err)
  }

  ln, err := net.Listen("tcp", "127.0.0.1:0")
  if err != nil {
    t.Fatalf("listen: %v", err)
  }
  port := ln.Addr().(*net.TCPAddr).Port
  ln.Close()

  logFile := filepath.Join(repoRoot, "tmp", "swank-norvig-test.log")
  if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
    t.Fatalf("mkdir: %v", err)
  }

  server := exec.Command(filepath.Join(repoRoot, "build", "golisp2"), "--swank", fmt.Sprintf("127.0.0.1:%d", port))
  server.Dir = repoRoot
  f, err := os.Create(logFile)
  if err != nil {
    t.Fatalf("create log: %v", err)
  }
  server.Stdout = f
  server.Stderr = f
  if err := server.Start(); err != nil {
    t.Fatalf("start server: %v", err)
  }
  defer func() {
    _ = server.Process.Kill()
    _ = f.Close()
  }()

  time.Sleep(500 * time.Millisecond)

  client := exec.Command(
    filepath.Join(repoRoot, "build", "golisp2-client"),
    "--host", "127.0.0.1",
    "--port", fmt.Sprintf("%d", port),
    "--load", "pn-gps1/gps-norvig-bugs.lisp",
  )
  client.Dir = repoRoot
  out, err := client.CombinedOutput()
  if err != nil {
    t.Fatalf("client load failed: %v\n%s", err, out)
  }

  deadline := time.Now().Add(5 * time.Second)
  for {
    if err := server.Process.Signal(syscall.Signal(0)); err == nil {
      return
    }
    if time.Now().After(deadline) {
      break
    }
    time.Sleep(200 * time.Millisecond)
  }

  log, _ := os.ReadFile(logFile)
  t.Fatalf("server died after Norvig bugs script:\n%s", log)
}
