//**********************************************************************
//  lib/swank/framing.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260618
//**********************************************************************
// SWANK length-prefixed UTF-8 framing.
//**********************************************************************

package swank

import (
  "bufio"
  "fmt"
  "io"
  "strconv"

  "golisp2/lib"
)

// readFrame reads one SWANK length-prefixed S-expression.
// Format: 6-digit hex length followed immediately by the S-expression.
// br MUST be a persistent *bufio.Reader for the connection — creating a
// fresh bufio.Reader per call would discard buffered bytes from pipelined
// frames and lose them.
func readFrame(br *bufio.Reader) (*lib.Cell, error) {
  lenBuf := make([]byte, 6)
  if _, err := io.ReadFull(br, lenBuf); err != nil {
    return nil, fmt.Errorf("readFrame: %w", err)
  }
  n, err := strconv.ParseInt(string(lenBuf), 16, 32)
  if err != nil {
    return nil, fmt.Errorf("readFrame: invalid length %q: %w", string(lenBuf), err)
  }
  payload := make([]byte, n)
  if _, err := io.ReadFull(br, payload); err != nil {
    return nil, fmt.Errorf("readFrame: short read: %w", err)
  }
  cell, err := lib.Read(string(payload))
  if err != nil {
    return nil, fmt.Errorf("readFrame: parse: %w", err)
  }
  return cell, nil
}

// writeFrame writes one SWANK length-prefixed S-expression.
// Format: 6-digit hex length followed immediately by the S-expression.
func writeFrame(w io.Writer, cell *lib.Cell) error {
  payload := cell.String()
  frame := fmt.Sprintf("%06x%s", len(payload), payload)
  _, err := io.WriteString(w, frame)
  if err != nil {
    return fmt.Errorf("writeFrame: %w", err)
  }
  return nil
}
