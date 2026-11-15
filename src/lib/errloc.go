//**********************************************************************
//  lib/errloc.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-fable-5-2
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261115
//**********************************************************************
// Fehlerort und Aufrufkette. evalWithCtx hängt beides auf dem Fehlerpfad
// an (locate); die Meldung selbst bleibt unverändert, damit trap,
// handler-case und Tests weiter denselben Text sehen. Nur die Ausgabe
// ganz oben (CLI) zeigt den Ort über ErrorTrace.
//**********************************************************************

package lib

import (
  "errors"
  "fmt"
  "strings"
)

const maxLocCalls = 20
const maxLocFormLen = 70

// locError trägt einen Fehler samt Ort. at ist die innerste Listenform,
// in der er auftrat, calls die Aufrufformen von innen nach außen.
type locError struct {
  err   error
  at    *Cell
  calls []*Cell
  more  int // Aufrufe jenseits von maxLocCalls
}

func (e *locError) Error() string { return e.err.Error() }
func (e *locError) Unwrap() error { return e.err }

func (e *locError) note(expr, call, first *Cell) {
  // innerste Listenform gewinnt, auch ohne Zeile (Makro-Expansion):
  // die Form selbst ist dann der Hinweis, die Aufrufkette trägt den Ort
  if e.at == nil && expr != nil && expr.Type == LIST {
    e.at = expr
  }
  e.addCall(call)
  if first != call {
    e.addCall(first)
  }
}

func (e *locError) addCall(call *Cell) {
  if call == nil {
    return
  }
  if len(e.calls) < maxLocCalls {
    e.calls = append(e.calls, call)
  } else {
    e.more++
  }
}

// locate läuft in jedem evalWithCtx-Frame, durch den ein Fehler nach
// außen wandert. expr ist die Form, die dieser Frame gerade auswertete,
// call die Lambda-Anwendung, in deren Rumpf er steckt (nil, wenn keine),
// first die erste des Frames, falls Tail-Calls call überschrieben haben.
// Kontrollfluss-Sentinels (return-from, throw, go) bleiben unberührt.
func locate(err error, expr, call, first *Cell) error {
  switch e := err.(type) {
  case *locError:
    e.note(expr, call, first)
    return err
  case *blockReturn, *throwValue, *goSignal:
    return err
  }
  var le *locError
  if errors.As(err, &le) { // z. B. von loadFile mit %w umhüllt
    le.note(expr, call, first)
    return err
  }
  le = &locError{err: err}
  le.note(expr, call, first)
  return le
}

func srcPosString(c *Cell) string {
  form := c.String()
  if r := []rune(form); len(r) > maxLocFormLen {
    form = string(r[:maxLocFormLen]) + "…"
  }
  switch {
  case c.SrcFile() != "":
    return fmt.Sprintf("%s:%d: %s", c.SrcFile(), c.SrcLine, form)
  case c.SrcLine > 0:
    return fmt.Sprintf("Zeile %d: %s", c.SrcLine, form)
  }
  return form // zur Laufzeit gebaute Form (Makro, eval) — kein Ort
}

// wantsErrorTrace: ein trap-Handler bekommt den Fehlerort nur, wenn er als
// Lambda genau zwei einfache Parameter hat. Alles andere (ein Parameter,
// &optional/&rest, eingebaute Funktion) bleibt beim bisherigen Aufruf mit
// der Meldung allein.
func wantsErrorTrace(h *Cell) bool {
  if h == nil || h.Type != LAMBDA {
    return false
  }
  n := 0
  for p := h.Car; p != nil && p.Type != NIL; p = p.Cdr {
    if p.Type != LIST || p.Car == nil || p.Car.Type != ATOM || strings.HasPrefix(p.Car.Val, "&") {
      return false
    }
    n++
  }
  return n == 2
}

// ErrorTrace liefert Fehlerort und Aufrufkette als Zeilen für die
// Ausgabe unter der Meldung, oder "" wenn err keinen Ort trägt.
func ErrorTrace(err error) string {
  var le *locError
  if !errors.As(err, &le) {
    return ""
  }
  var b strings.Builder
  if le.at != nil {
    fmt.Fprintf(&b, "  in %s\n", srcPosString(le.at))
  }
  for _, c := range le.calls {
    fmt.Fprintf(&b, "  aufgerufen von %s\n", srcPosString(c))
  }
  if le.more > 0 {
    fmt.Fprintf(&b, "  … %d weitere Aufrufe\n", le.more)
  }
  return b.String()
}
