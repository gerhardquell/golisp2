//**********************************************************************
//  lib/redeflog_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260724
//**********************************************************************

package lib

import (
  "fmt"
  "testing"
)

func TestRedefLogAppendAndOrder(t *testing.T) {
  ClearRedefLog()
  logRedef(RedefEvent{Name: "a", OldKind: "lambda", Action: "reload"})
  logRedef(RedefEvent{Name: "b", OldKind: "func", Action: "warn"})
  events := RedefLog()
  if len(events) != 2 {
    t.Fatalf("2 Events erwartet, got %d", len(events))
  }
  if events[0].Name != "a" || events[1].Name != "b" {
    t.Fatalf("Reihenfolge älteste→neueste verletzt: %+v", events)
  }
}

func TestRedefLogRingOverflow(t *testing.T) {
  ClearRedefLog()
  for i := 0; i < redefLogSize+10; i++ {
    logRedef(RedefEvent{Name: fmt.Sprintf("n%d", i)})
  }
  events := RedefLog()
  if len(events) != redefLogSize {
    t.Fatalf("Ring muss bei %d kappen, got %d", redefLogSize, len(events))
  }
  if events[0].Name != "n10" {
    t.Fatalf("ältestes Event muss n10 sein, got %q", events[0].Name)
  }
  want := fmt.Sprintf("n%d", redefLogSize+9)
  if events[len(events)-1].Name != want {
    t.Fatalf("neuestes Event muss %q sein, got %q", want, events[len(events)-1].Name)
  }
}

func TestRedefLogReturnsCopy(t *testing.T) {
  ClearRedefLog()
  logRedef(RedefEvent{Name: "x"})
  events := RedefLog()
  events[0].Name = "mutiert"
  if RedefLog()[0].Name != "x" {
    t.Fatal("RedefLog muss eine Kopie liefern")
  }
}

func TestKindOf(t *testing.T) {
  cases := map[*Cell]string{
    {Type: FUNC}:   "func",
    {Type: LAMBDA}: "lambda",
    {Type: MACRO}:  "macro",
    {Type: NUMBER}: "value",
    {Type: ATOM}:   "value",
  }
  for c, want := range cases {
    if got := kindOf(c); got != want {
      t.Errorf("kindOf(%v) = %q, want %q", c.Type, got, want)
    }
  }
}
