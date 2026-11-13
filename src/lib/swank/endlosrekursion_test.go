//**********************************************************************
//  src/lib/swank/endlosrekursion_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k2.7-code
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260716
//**********************************************************************
// Integrationstest: der SWANK-Server (golisp2 --swank) überlebt das Laden
// von tests/endlosrekursion.lisp. Lädt das Skript per swank:load-file über TCP und
// prüft, ob der Server danach noch auf Signal 0 antwortet (also lebt).
//**********************************************************************

package swank

import (
  "bufio"
  "fmt"
  "net"
  "os"
  "os/exec"
  "path/filepath"
  "runtime"
  "strconv"
  "syscall"
  "testing"
  "time"

  "golisp2/src/lib"
)

func TestSwankSurvivesEndlosrekursion(t *testing.T) {
  if runtime.GOOS != "linux" {
    t.Skip("nur auf Linux")
  }

  repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
  if err != nil {
    t.Fatalf("repo root: %v", err)
  }

  ln, err := net.Listen("tcp", "127.0.0.1:0")
  if err != nil {
    t.Fatalf("listen: %v", err)
  }
  port := ln.Addr().(*net.TCPAddr).Port
  ln.Close()

  logFile := filepath.Join(repoRoot, "tmp", "swank-endlosrekursion-test.log")
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

  addr := fmt.Sprintf("127.0.0.1:%d", port)
  var conn net.Conn
  for i := 0; i < 50; i++ {
    if conn, err = net.Dial("tcp", addr); err == nil {
      break
    }
    time.Sleep(100 * time.Millisecond)
  }
  if err != nil {
    t.Fatalf("dial %s: %v", addr, err)
  }
  defer conn.Close()

  msg, err := lib.Read(`(:emacs-rex (swank:load-file ` + strconv.Quote("tests/endlosrekursion.lisp") + `) "USER" t 1)`)
  if err != nil {
    t.Fatalf("read msg: %v", err)
  }
  if err := writeFrame(conn, msg); err != nil {
    t.Fatalf("writeFrame: %v", err)
  }
  if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
    t.Fatalf("deadline: %v", err)
  }
  br := bufio.NewReader(conn)
  for {
    frame, err := readFrame(br)
    if err != nil {
      t.Fatalf("load failed: %v", err)
    }
    parts := lib.CellToSlice(frame)
    if len(parts) < 2 || parts[0].Type != lib.ATOM || parts[0].Val != ":return" {
      continue
    }
    status := lib.CellToSlice(parts[1])
    if len(status) < 1 || status[0].Type != lib.ATOM || status[0].Val != ":ok" {
      t.Fatalf("load failed: %s", frame.String())
    }
    break
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
  t.Fatalf("server died after endlosrekursion script:\n%s", log)
}
