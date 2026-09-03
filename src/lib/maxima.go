//**********************************************************************
//  lib/maxima.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-sonnet-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260903
//**********************************************************************
// Maxima-Integration: programmierbarer Taschenrechner via externen
// Maxima-Prozess. Maxima ist selbst ein Lisp-Programm (eigener CL-Runtime,
// z.B. ECL/SBCL/GCL) -- Embedding in golisp2 scheidet aus (Packages/CLOS,
// siehe Oekosystem-Strategie). Stattdessen ein Long-Lived-Subprozess pro
// Session: Zustand (Variablen, Funktionsdefs) bleibt ueber mehrere
// maxima-eval-Aufrufe erhalten.
//
// Synchronisation ueber einen Sentinel-Marker (printf(true,"~%~a~%",TOK)$
// nach jedem Ausdruck), nicht ueber (%iN)-Prompt-Parsing: robust
// unabhaengig von --very-quiet (unterdrueckt Prompts komplett), Fehler-
// meldungen (Maxima setzt danach fort) und Zeilenumbruch (linel:100000
// beim Start deaktiviert 2D-Umbruch).
//**********************************************************************

package lib

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const maximaBinary = "maxima"

// maximaSession haelt einen laufenden Maxima-Prozess mit offenen Pipes.
// mu serialisiert: ein maxima-eval gleichzeitig pro Session.
type maximaSession struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
	n      int64
	closed bool
}

// RegisterMaxima fuegt maxima-open, maxima-eval, maxima-close in env ein
func RegisterMaxima(env *Env) {
	_ = env.Set("maxima-open", makeFn(fnMaximaOpen))
	_ = env.Set("maxima-eval", makeFn(fnMaximaEval))
	_ = env.Set("maxima-close", makeFn(fnMaximaClose))
}

// maxima-open: () → Session-Handle. Startet Maxima im Hintergrund,
// schaltet auf lineare Ausgabe (kein 2D-Pretty-Print, kein Zeilenumbruch).
func fnMaximaOpen(args []*Cell) (*Cell, error) {
	cmd := exec.Command(maximaBinary, "--very-quiet")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("maxima-open: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("maxima-open: %w", err)
	}
	// Startup-Meldungen (ECL-Loading-Zeilen) landen auf stdout -- werden
	// beim ersten evalRaw als Teil des Sentinel-Reads mitkonsumiert.
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("maxima-open: Start fehlgeschlagen (ist 'maxima' installiert?): %w", err)
	}

	sess := &maximaSession{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdoutPipe),
	}

	if _, err := sess.evalRaw("display2d:false$ linel:100000$", defaultExecTimeout); err != nil {
		_ = sess.kill()
		return nil, fmt.Errorf("maxima-open: Init fehlgeschlagen: %w", err)
	}

	return &Cell{Type: LIST, Env: sess}, nil
}

// maxima-eval: (maxima-eval session "expr" [timeout-sekunden]) → String
func fnMaximaEval(args []*Cell) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("maxima-eval: 2 Argumente nötig (session expr)")
	}
	sess, err := asMaximaSession(args[0])
	if err != nil {
		return nil, fmt.Errorf("maxima-eval: %w", err)
	}
	if args[1].Type != STRING {
		return nil, fmt.Errorf("maxima-eval: expr muss ein String sein")
	}
	timeout := defaultExecTimeout
	if len(args) >= 3 {
		if args[2].Type != NUMBER {
			return nil, fmt.Errorf("maxima-eval: timeout muss eine Zahl (Sekunden) sein")
		}
		timeout = time.Duration(args[2].Num * float64(time.Second))
	}

	out, err := sess.evalRaw(args[1].Val, timeout)
	if err != nil {
		return nil, fmt.Errorf("maxima-eval: %w", err)
	}
	return MakeStr(out), nil
}

// maxima-close: (maxima-close session) → t
func fnMaximaClose(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("maxima-close: 1 Argument nötig")
	}
	sess, err := asMaximaSession(args[0])
	if err != nil {
		return nil, fmt.Errorf("maxima-close: %w", err)
	}
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if sess.closed {
		return cellT, nil
	}
	sess.closed = true
	_ = sess.stdin.Close()
	done := make(chan error, 1)
	go func() { done <- sess.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = sess.kill()
		<-done
	}
	return cellT, nil
}

func asMaximaSession(c *Cell) (*maximaSession, error) {
	if c == nil || c.Env == nil {
		return nil, fmt.Errorf("ungültige Session")
	}
	sess, ok := c.Env.(*maximaSession)
	if !ok {
		return nil, fmt.Errorf("keine Maxima-Session")
	}
	if sess.closed {
		return nil, fmt.Errorf("Session bereits geschlossen")
	}
	return sess, nil
}

func (s *maximaSession) kill() error {
	if s.cmd != nil && s.cmd.Process != nil {
		return s.cmd.Process.Kill()
	}
	return nil
}

// evalRaw schreibt expr an Maxima, haengt ein Sentinel-printf an und liest
// stdout bis der Sentinel-Marker als eigene Zeile erscheint. Alles davor
// ist das Ergebnis (Wert oder Fehlertext -- Maxima setzt nach Fehlern fort).
func (s *maximaSession) evalRaw(expr string, timeout time.Duration) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", fmt.Errorf("leerer Ausdruck")
	}
	if !strings.HasSuffix(expr, ";") && !strings.HasSuffix(expr, "$") {
		expr += ";"
	}

	s.n++
	token := fmt.Sprintf("GOLISP2-EOF-%d", s.n)
	cmd := expr + "\n" + `printf(true,"~%~a~%","` + token + `")$` + "\n"

	if _, err := io.WriteString(s.stdin, cmd); err != nil {
		return "", fmt.Errorf("Schreiben an Maxima fehlgeschlagen: %w", err)
	}

	type readResult struct {
		text string
		err  error
	}
	resultCh := make(chan readResult, 1)
	go func() {
		var buf strings.Builder
		for {
			line, err := s.stdout.ReadString('\n')
			if strings.TrimSpace(line) == token {
				resultCh <- readResult{text: buf.String()}
				return
			}
			buf.WriteString(line)
			if err != nil {
				resultCh <- readResult{text: buf.String(), err: err}
				return
			}
		}
	}()

	select {
	case res := <-resultCh:
		if res.err != nil {
			return "", fmt.Errorf("Lesen von Maxima fehlgeschlagen (Prozess beendet?): %w", res.err)
		}
		return strings.TrimSpace(res.text), nil
	case <-time.After(timeout):
		_ = s.kill()
		return "", fmt.Errorf("Timeout nach %v -- Session gekillt", timeout)
	}
}
