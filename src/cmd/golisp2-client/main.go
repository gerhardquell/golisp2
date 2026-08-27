//**********************************************************************
//  cmd/golisp2-client/main.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260301 (renamed 20260709, SWANK-Protokoll 20260709)
//**********************************************************************
// GoLisp2 Client - CLI-Client für den golisp2-SWANK-Server (golisp2 --swank).
// Spricht das echte SWANK-Protokoll (length-prefixed :emacs-rex RPC),
// nicht mehr das alte Custom-RPC. Unterstützt eval, repl, complete, load.
//**********************************************************************

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"golisp2/src/lib"
)

// Client repräsentiert die SWANK-Verbindung zum golisp2-SWANK-Server.
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	host   string
	port   string
	msgID  int
}

// NewClient erstellt einen neuen Client.
func NewClient(host, port string) *Client {
	return &Client{host: host, port: port, msgID: 0}
}

// Connect stellt die TCP-Verbindung zum Server her.
func (c *Client) Connect() error {
	addr := c.host + ":" + c.port
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("kann nicht zu %s verbinden: %w", addr, err)
	}
	c.conn = conn
	c.reader = bufio.NewReader(conn)
	return nil
}

// Close schließt die Verbindung.
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// writeFrame sendet eine S-expression als SWANK-Frame: 6-stellige
// Hex-Länge + Payload. cell.String() escaped Strings via Go %q korrekt.
func (c *Client) writeFrame(cell *lib.Cell) error {
	payload := cell.String()
	frame := fmt.Sprintf("%06x%s", len(payload), payload)
	if _, err := c.conn.Write([]byte(frame)); err != nil {
		return fmt.Errorf("send: %w", err)
	}
	return nil
}

// readFrame liest einen SWANK-Frame und parst ihn zu einer Cell.
func (c *Client) readFrame() (*lib.Cell, error) {
	lenBuf := make([]byte, 6)
	if _, err := io.ReadFull(c.reader, lenBuf); err != nil {
		return nil, fmt.Errorf("recv: %w", err)
	}
	n, err := strconv.ParseInt(string(lenBuf), 16, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid frame length %q: %w", string(lenBuf), err)
	}
	payload := make([]byte, n)
	if _, err := io.ReadFull(c.reader, payload); err != nil {
		return nil, fmt.Errorf("recv short: %w", err)
	}
	cell, err := lib.Read(string(payload))
	if err != nil {
		return nil, fmt.Errorf("parse frame: %w", err)
	}
	return cell, nil
}

// request sendet eine :emacs-rex-Form und liest Frames bis das
// zugehörige :return mit passender id eintrifft. Unterwegs gesehene
// :write-string-Events (REPL-Ausgabe) werden gesammelt.
// Liefert die gesammelten Texte und den :ok-Wert der Antwort.
func (c *Client) request(opForm *lib.Cell) (writes []string, result *lib.Cell, err error) {
	c.msgID++
	id := c.msgID
	// (:emacs-rex opForm "USER" t id)
	msg := lib.Cons(lib.MakeAtom(":emacs-rex"),
		lib.Cons(opForm,
			lib.Cons(lib.MakeStr("USER"),
				lib.Cons(lib.MakeAtom("t"),
					lib.Cons(lib.MakeNum(float64(id)), lib.MakeNil())))))
	if err := c.writeFrame(msg); err != nil {
		return nil, nil, err
	}

	for {
		frame, ferr := c.readFrame()
		if ferr != nil {
			return writes, nil, ferr
		}
		parts := lib.CellToSlice(frame)
		if len(parts) == 0 {
			continue
		}
		head := parts[0]
		if head.Type != lib.ATOM {
			continue
		}

		switch head.Val {
		case ":write-string":
			// (:write-string "text" [:repl-result|nil])
			if len(parts) >= 2 && parts[1].Type == lib.STRING {
				writes = append(writes, parts[1].Val)
			}
		case ":return":
			// (:return (:ok val) id) | (:return (:abort val) id)
			if len(parts) < 2 {
				return writes, nil, fmt.Errorf("malformed :return: %s", frame.String())
			}
			rparts := lib.CellToSlice(parts[1])
			if len(rparts) < 2 {
				return writes, nil, fmt.Errorf("malformed :return-status: %s", parts[1].String())
			}
			status, val := rparts[0], rparts[1]
			if status.Type == lib.ATOM && status.Val == ":abort" {
				return writes, val, fmt.Errorf("server: %s", cellText(val))
			}
			return writes, val, nil
		}
		// :new-package u.ä. ignorieren
	}
}

// cellText liefert den rohen Text einer Cell: STRING -> Val (ungequotet),
// sonst die String-Darstellung. Für Fehlermeldungen an :abort.
func cellText(c *lib.Cell) string {
	if c == nil {
		return "()"
	}
	if c.Type == lib.STRING {
		return c.Val
	}
	return c.String()
}

// Ping prüft via connection-info, ob der Server SWANK spricht.
func (c *Client) Ping() error {
	op := lib.Cons(lib.MakeAtom("swank:connection-info"), lib.MakeNil())
	_, _, err := c.request(op)
	return err
}

// Eval wertet Lisp-Code via listener-eval aus. Sammelt alle
// :write-string-Ausgaben (Ergebnisse + print-Output) und liefert sie
// vereint zurück.
func (c *Client) Eval(code string) (string, error) {
	op := lib.Cons(lib.MakeAtom("swank-repl:listener-eval"),
		lib.Cons(lib.MakeStr(code), lib.MakeNil()))
	writes, _, err := c.request(op)
	if err != nil {
		return "", err
	}
	return strings.Join(writes, ""), nil
}

// Complete liefert die Simple-Completions für einen Prefix.
func (c *Client) Complete(prefix string) (string, error) {
	op := lib.Cons(lib.MakeAtom("swank:simple-completions"),
		lib.Cons(lib.MakeStr(prefix), lib.Cons(lib.MakeStr("USER"), lib.MakeNil())))
	_, val, err := c.request(op)
	if err != nil {
		return "", err
	}
	var matches []string
	for _, m := range lib.CellToSlice(val) {
		if m.Type == lib.STRING {
			matches = append(matches, m.Val)
		}
	}
	if len(matches) == 0 {
		return "(keine Treffer)", nil
	}
	return strings.Join(matches, " "), nil
}

// LoadFile lädt eine Lisp-Datei serverseitig via load-file.
func (c *Client) LoadFile(path string) (string, error) {
	op := lib.Cons(lib.MakeAtom("swank:load-file"),
		lib.Cons(lib.MakeStr(path), lib.MakeNil()))
	_, val, err := c.request(op)
	if err != nil {
		return "", err
	}
	if val.Type == lib.STRING {
		return val.Val, nil
	}
	return val.String(), nil
}

// runREPL startet den interaktiven Modus.
func runREPL(client *Client) {
	fmt.Println("GoLisp2 Client REPL (SWANK)")
	fmt.Printf("Verbunden mit %s:%s\n", client.host, client.port)
	fmt.Println("Befehle: :quit, :complete prefix, :load datei")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	multiline := ""
	openParens := 0

	for {
		if multiline == "" {
			fmt.Print("golisp2> ")
		} else {
			fmt.Print("      > ")
		}

		if !scanner.Scan() {
			break
		}
		line := scanner.Text()

		if multiline == "" {
			if line == ":quit" || line == ":q" {
				break
			}
			if strings.HasPrefix(line, ":complete ") {
				prefix := strings.TrimSpace(line[10:])
				result, err := client.Complete(prefix)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Fehler: %v\n", err)
				} else {
					fmt.Println(result)
				}
				continue
			}
			if strings.HasPrefix(line, ":load ") {
				path := strings.TrimSpace(line[6:])
				result, err := client.LoadFile(path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Fehler: %v\n", err)
				} else {
					fmt.Println(result)
				}
				continue
			}
		}

		multiline += line + "\n"
		openParens += countParens(line)

		if openParens <= 0 && strings.TrimSpace(multiline) != "" {
			expr := strings.TrimSpace(multiline)
			result, err := client.Eval(expr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Fehler: %v\n", err)
			} else if result != "" {
				fmt.Print(result)
				if !strings.HasSuffix(result, "\n") {
					fmt.Println()
				}
			}
			multiline = ""
			openParens = 0
		}
	}
}

// countParens zählt offene Klammern (string-sicher).
func countParens(s string) int {
	count := 0
	inString := false
	escape := false
	for _, ch := range s {
		if escape {
			escape = false
			continue
		}
		if ch == '\\' && inString {
			escape = true
			continue
		}
		if ch == '"' && !escape {
			inString = !inString
			continue
		}
		if !inString {
			if ch == '(' {
				count++
			} else if ch == ')' {
				count--
			}
		}
	}
	return count
}

func main() {
	// Env-Variablen dienen als Flag-Default; ein expliziter --host/--port
	// gewinnt über die Env (Unix-Konvention: Flag > Env > Default).
	hostDefault := os.Getenv("GOLISP_HOST")
	if hostDefault == "" {
		hostDefault = "localhost"
	}
	portDefault := os.Getenv("GOLISP_PORT")
	if portDefault == "" {
		portDefault = "4321"
	}

	var (
		host     = flag.String("host", hostDefault, "Server-Host (env: GOLISP_HOST)")
		port     = flag.String("port", portDefault, "Server-Port (env: GOLISP_PORT)")
		evalFlag = flag.String("eval", "", "Expression auswerten")
		replFlag = flag.Bool("repl", false, "Interaktiver REPL-Modus")
		compFlag = flag.String("complete", "", "Autocomplete für Prefix")
		loadFlag = flag.String("load", "", "Datei laden")
		pingFlag = flag.Bool("ping", false, "Server-Ping")
	)
	flag.Parse()

	client := NewClient(*host, *port)
	if err := client.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "Fehler: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	if err := client.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "Server nicht erreichbar: %v\n", err)
		os.Exit(1)
	}

	switch {
	case *pingFlag:
		fmt.Println("Server ist erreichbar: pong (SWANK connection-info ok)")

	case *evalFlag != "":
		result, err := client.Eval(*evalFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fehler: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(strings.TrimRight(result, "\n") + "\n")

	case *compFlag != "":
		result, err := client.Complete(*compFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fehler: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(result)

	case *loadFlag != "":
		result, err := client.LoadFile(*loadFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fehler: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(result)

	case *replFlag:
		runREPL(client)

	default:
		// Default: einfacher Eval-Modus für "golisp2-client 'expr'"
		if flag.NArg() > 0 {
			expr := flag.Arg(0)
			result, err := client.Eval(expr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Fehler: %v\n", err)
				os.Exit(1)
			}
			fmt.Print(strings.TrimRight(result, "\n") + "\n")
		} else {
			runREPL(client)
		}
	}
}
