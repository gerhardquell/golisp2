//**********************************************************************
//  lib/redeflog.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260724
//**********************************************************************
// Redef-Log: Ringpuffer aller Root-Redefinitionen und makunbound-Events.
// Beobachtbarkeit statt Verbot: der selbsterweiternde Pfad
// (eval (read (sigo ...))) bleibt erlaubt, aber nachvollziehbar.
//**********************************************************************

package lib

import "sync"

// RedefEvent beschreibt eine Root-Redefinition oder ein makunbound.
type RedefEvent struct {
  Name    string
  OldKind string // "func", "lambda", "macro", "value"
  NewKind string // "" bei makunbound
  OldFile string // "" = interaktiv (REPL/stdin/swank)
  OldLine int
  NewFile string // "" = Quelle unbekannt (Set-Hook) oder interaktiv
  NewLine int
  Action  string // "reload", "redef", "warn", "error", "makunbound"
}

const redefLogSize = 256

var (
  redefMu   sync.Mutex
  redefRing = make([]RedefEvent, 0, redefLogSize)
)

// logRedef haengt ein Event an; bei vollem Ring faellt das aelteste raus.
func logRedef(e RedefEvent) {
  redefMu.Lock()
  defer redefMu.Unlock()
  if len(redefRing) < redefLogSize {
    redefRing = append(redefRing, e)
    return
  }
  copy(redefRing, redefRing[1:])
  redefRing[redefLogSize-1] = e
}

// RedefLog liefert eine Kopie aller Events, aelteste zuerst.
func RedefLog() []RedefEvent {
  redefMu.Lock()
  defer redefMu.Unlock()
  out := make([]RedefEvent, len(redefRing))
  copy(out, redefRing)
  return out
}

// ClearRedefLog leert das Log (Tests).
func ClearRedefLog() {
  redefMu.Lock()
  defer redefMu.Unlock()
  redefRing = redefRing[:0]
}

// kindOf klassifiziert eine Bindung fuers Log.
func kindOf(c *Cell) string {
  switch c.Type {
  case FUNC:
    return "func"
  case LAMBDA:
    return "lambda"
  case MACRO:
    return "macro"
  }
  return "value"
}
