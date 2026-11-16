//**********************************************************************
//  lib/swank/integration_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260618
//**********************************************************************
// End-to-end test für SWANK connection-info.
//**********************************************************************

package swank

import (
  "bufio"
  "net"
  "os"
  "path/filepath"
  "strings"
  "testing"
  "time"

  "golisp2/src/lib"
)

func TestSwankServerConnectionInfo(t *testing.T) {
  listener, err := net.Listen("tcp", "127.0.0.1:0")
  if err != nil {
    t.Fatalf("listen: %v", err)
  }
  defer listener.Close()

  go serveListener(listener)

  conn, err := net.Dial("tcp", listener.Addr().String())
  if err != nil {
    t.Fatalf("dial: %v", err)
  }
  defer conn.Close()

  // Send connection-info request
  msg := lib.Cons(lib.MakeAtom(":emacs-rex"),
    lib.Cons(lib.Cons(lib.MakeAtom("swank:connection-info"), lib.MakeNil()),
      lib.Cons(lib.MakeNil(),
        lib.Cons(lib.MakeAtom("t"),
          lib.Cons(lib.MakeInt(1), lib.MakeNil())))))
  if err := writeFrame(conn, msg); err != nil {
    t.Fatalf("writeFrame: %v", err)
  }

  // Set read deadline to avoid hanging
  conn.SetReadDeadline(time.Now().Add(2 * time.Second))

  resp, err := readFrame(bufio.NewReader(conn))
  if err != nil {
    t.Fatalf("readFrame: %v", err)
  }
  s := resp.String()
  if !strings.Contains(s, ":return") || !strings.Contains(s, "GoLisp") {
    t.Fatalf("unexpected response: %s", s)
  }
}

func TestIntegrationFindDefinitionsLoadFile(t *testing.T) {
  lib.ClearDefinitions()
  // Testdatei mit defun an bekannter Zeile
  dir := t.TempDir()
  path := filepath.Join(dir, "mod.lisp")
  content := ";; comment\n(defun loaded-fn (x) (* x x))\n"
  if err := os.WriteFile(path, []byte(content), 0644); err != nil {
    t.Fatalf("write: %v", err)
  }
  env := lib.BaseEnv()
  if err := lib.LoadStdlib(env); err != nil {
    t.Fatalf("LoadStdlib: %v", err)
  }
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  // Datei via swank:load-file laden (stempelt SrcFile)
  loadMsg, err := lib.Read(`(:emacs-rex (swank:load-file "` + path + `") nil t 1)`)
  if err != nil {
    t.Fatalf("read load: %v", err)
  }
  if _, err := HandleMessage(env, loadMsg); err != nil {
    t.Fatalf("load-file: %v", err)
  }
  // M-. auf loaded-fn
  findMsg, err := lib.Read(`(:emacs-rex (swank:find-definitions-for-emacs "loaded-fn") nil t 1)`)
  if err != nil {
    t.Fatalf("read find: %v", err)
  }
  result, err := HandleMessage(env, findMsg)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  s := result.String()
  if !strings.Contains(s, path) {
    t.Fatalf("expected path %s in result, got: %s", path, s)
  }
  if !strings.Contains(s, ":line") {
    t.Fatalf("expected :line in result, got: %s", s)
  }
}

// swankClient: eine offene Verbindung zum Test-Server.
type swankClient struct {
  conn net.Conn
  br   *bufio.Reader
}

func dialSwank(t *testing.T, addr string) *swankClient {
  t.Helper()
  conn, err := net.Dial("tcp", addr)
  if err != nil {
    t.Fatalf("dial: %v", err)
  }
  t.Cleanup(func() { conn.Close() })
  return &swankClient{conn: conn, br: bufio.NewReader(conn)}
}

// eval schickt src als listener-eval und liefert alle Frames bis :return.
func (c *swankClient) eval(t *testing.T, src string, id int) []string {
  t.Helper()
  if err := writeFrame(c.conn, rexEval(src, id)); err != nil {
    t.Fatalf("writeFrame: %v", err)
  }
  c.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
  var frames []string
  for {
    resp, err := readFrame(c.br)
    if err != nil {
      t.Fatalf("readFrame nach %q (bisher %v): %v", src, frames, err)
    }
    frames = append(frames, resp.String())
    if strings.HasPrefix(resp.String(), "(:return") {
      return frames
    }
  }
}

// Zwei Verbindungen an einem --swank-Server (RunServer: eigenes Env je
// Verbindung): die Ausgabe einer Auswertung geht an die Verbindung, die
// sie ausgelöst hat — nicht an die zuletzt verbundene, und auch dann
// noch, wenn die andere inzwischen getrennt ist.
func TestSwankOutputGoesToEvaluatingConnection(t *testing.T) {
  listener, err := net.Listen("tcp", "127.0.0.1:0")
  if err != nil {
    t.Fatalf("listen: %v", err)
  }
  defer listener.Close()
  defer lib.ResetOutputWriter()
  defer lib.ResetErrorWriter()
  go serveListener(listener)

  a := dialSwank(t, listener.Addr().String())
  a.eval(t, "(+ 1 1)", 1) // Verbindung A ist fertig eingerichtet
  b := dialSwank(t, listener.Addr().String())
  b.eval(t, "(+ 2 2)", 2) // B verbindet später

  hat := func(frames []string, text string) bool {
    for _, f := range frames {
      if strings.HasPrefix(f, "(:write-string") && strings.Contains(f, text) {
        return true
      }
    }
    return false
  }
  if frames := a.eval(t, `(format t "von-a~%")`, 3); !hat(frames, "von-a") {
    t.Errorf("A sieht die eigene Ausgabe nicht: %v", frames)
  }
  if frames := b.eval(t, `(format t "von-b~%")`, 4); !hat(frames, "von-b") {
    t.Errorf("B sieht die eigene Ausgabe nicht: %v", frames)
  }

  b.conn.Close()
  time.Sleep(50 * time.Millisecond)
  if frames := a.eval(t, `(format t "a-nach-b~%")`, 5); !hat(frames, "a-nach-b") {
    t.Errorf("A verliert die Ausgabe, nachdem B getrennt hat: %v", frames)
  }
}
