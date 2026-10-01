//**********************************************************************
//  lib/stdin.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261001
//**********************************************************************
// Gemeinsamer gepufferter stdin-Reader für (read-line) und gets.
// Aus primitives.go herausgelöst.
//**********************************************************************

package lib

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// stdinReader/stdinMu: gepufferter Reader über die aktuelle stdin-Quelle
// für (read-line). Package-level und mutex-geschützt, damit bereits
// gepufferte, aber ungelesene Bytes zwischen Aufrufen nicht verloren gehen
// (kein bufio.NewReader pro Call). Default: os.Stdin.
var (
	stdinMu     sync.Mutex
	stdinReader = bufio.NewReader(os.Stdin)
)

// SetStdinReader tauscht die stdin-Quelle für (read-line) aus. Für Tests
// gedacht (os.Pipe()) — Produktionscode ruft das nicht auf.
func SetStdinReader(r io.Reader) {
	stdinMu.Lock()
	defer stdinMu.Unlock()
	stdinReader = bufio.NewReader(r)
}

// ResetStdinReader stellt os.Stdin als Quelle für (read-line) wieder her.
func ResetStdinReader() {
	stdinMu.Lock()
	defer stdinMu.Unlock()
	stdinReader = bufio.NewReader(os.Stdin)
}

// read-line: (read-line) → liest eine Zeile von stdin, liefert String ohne
// Newline. Kein Parsing — wer Lisp-Daten will, kombiniert selbst
// (read (read-line)).
//
// EINSCHRÄNKUNG (TODO.md Punkt 3, 20260813): nur sinnvoll im
// Datei-Argument-Modus (`golisp2 skript.lisp`, auch per Shebang direkt
// ausgeführt, z. B. `./skript.lisp`) nutzbar. main.go liest im
// Default-stdin-Modus (kein Datei-Argument, runStdin) bereits VOR der
// Auswertung das komplette stdin via io.ReadAll(os.Stdin) als Programmquelle
// ein — dort ist stdin zur Laufzeit schon leer (EOF). Über SWANK
// (lib/swank/) ebenfalls nicht nutzbar: kein Reverse-RPC zum Client, Emacs
// leitet Tastatureingaben nicht an golisp2s Stdin weiter. Details: doc/cli.md.
func fnReadLine(args []*Cell) (*Cell, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("read-line: keine Argumente erwartet")
	}
	line, err := readLineFromStdin()
	if err != nil {
		return nil, fmt.Errorf("read-line: %w", err)
	}
	return MakeString(line), nil
}

// readLineFromStdin liest eine Zeile (ohne Newline) vom gemeinsamen
// stdinReader. Geteilt von read-line und gets — nie einen zweiten
// bufio.Reader über os.Stdin legen (Puffer-Verlust).
func readLineFromStdin() (string, error) {
	stdinMu.Lock()
	r := stdinReader
	stdinMu.Unlock()
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line, nil
}

// slurpStdin liest den gemeinsamen stdinReader bis EOF.
func slurpStdin() (string, error) {
	stdinMu.Lock()
	r := stdinReader
	stdinMu.Unlock()
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
