//**********************************************************************
//  lib/concurrency_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260616
//**********************************************************************
// Tests für parfunc (eval_control.go) und Channel/Mutex-Primitive
// (goroutine.go). Nur deterministische Muster: parfunc sammelt nach
// Expr-Index (Reihenfolge garantiert), buffered Channels sequenziell.
// Kein unbuffered-send-ohne-receiver (würde Test blockieren).
//**********************************************************************

package lib

import (
  "strings"
  "testing"
  "time"
)

// --- parfunc ---

func TestParfuncBasic(t *testing.T) {
  // Zwei Expr parallel, Ergebnisse als Liste in r, Reihenfolge nach idx.
  src := `(parfunc r (+ 1 1) (* 2 2)) r`
  evalEq(t, src, "(2 4)")
}

func TestParfuncOrder(t *testing.T) {
  // Trotz paralleler Auswertung: Ergebnisreihenfolge = Expr-Reihenfolge.
  src := `(parfunc r 10 20 30) r`
  evalEq(t, src, "(10 20 30)")
}

// TestParfuncErrorBecomesNil: ein Expr wirft Fehler → nil an der Stelle.
func TestParfuncErrorBecomesNil(t *testing.T) {
  // (/ 1 0) wirft Fehler → Goroutine liefert nil.
  src := `(parfunc r (+ 1 1) (/ 1 0) (* 2 2)) r`
  evalEq(t, src, "(2 () 4)")
}

func TestParfuncEmpty(t *testing.T) {
  // Keine Expr → parfunc liefert nil zurück UND setzt resultName im env.
  // (Früher Mini-Bug: frühes return vor env.Set – r blieb ungebunden.
  //  Jetzt gefixt: env.Set(resultName, MakeNil()) vor return.)
  evalEq(t, `(parfunc r)`, "()")
  // r ist jetzt im env gesetzt (auf nil)
  evalEq(t, `(parfunc r2) r2`, "()")
}

func TestParfuncStoresInEnv(t *testing.T) {
  // Ergebnis wird unter resultName im env gespeichert (nur Rückgabe prüfen).
  got, err := evalStr(`(parfunc ergebnis (+ 5 5))`)
  if err != nil {
    t.Fatalf("parfunc Fehler: %v", err)
  }
  if got.String() != "(10)" {
    t.Errorf("parfunc Rückgabe = %q, want (10)", got)
  }
}

// TestParfuncTimeout: :timeout N bricht langsame Goroutinen ab → nil.
// Deterministisch: sleep-Expr läuft länger als timeout, wird nil.
func TestParfuncTimeout(t *testing.T) {
  start := time.Now()
  // sleep 2000ms = 2s, timeout 1s → sleep-Goroutine wird nil.
  // (+ 1 1) läuft instant → 2.
  src := `(parfunc r :timeout 1 (sleep 2000) (+ 1 1)) r`
  got, err := evalStr(src)
  elapsed := time.Since(start)
  if err != nil {
    t.Fatalf("parfunc timeout Fehler: %v", err)
  }
  // idx0 = sleep → nil (abgebrochen), idx1 = 2
  if got.String() != "(() 2)" {
    t.Errorf("parfunc timeout = %q, want (() 2)", got)
  }
  // Sollte ~1s gedauert haben (timeout), nicht 2s (sleep).
  if elapsed > 1500*time.Millisecond {
    t.Errorf("parfunc timeout dauerte %v, expected ~1s", elapsed)
  }
}

// --- Channels (nur buffered, sequenziell – kein Blocking) ---

func TestChannelBuffered(t *testing.T) {
  // Buffered channel: send nicht-blockierend solange Buffer Platz.
  src := `
    (define ch (chan-make 2))
    (chan-send ch "a")
    (chan-send ch "b")
    (list (chan-recv ch) (chan-recv ch))
  `
  evalEq(t, src, `("a" "b")`)
}

func TestChannelSendReturnsValue(t *testing.T) {
  // chan-send liefert den gesendeten Wert zurück.
  src := `
    (define ch (chan-make 1))
    (chan-send ch 42)
  `
  evalEq(t, src, "42")
}

func TestChannelFIFO(t *testing.T) {
  // Buffered channel ist FIFO: Reihenfolge bleibt erhalten.
  src := `
    (define ch (chan-make 3))
    (chan-send ch 1)
    (chan-send ch 2)
    (chan-send ch 3)
    (list (chan-recv ch) (chan-recv ch) (chan-recv ch))
  `
  evalEq(t, src, "(1 2 3)")
}

// --- Mutex / lock ---

func TestLockMake(t *testing.T) {
  // lock-make erzeugt eine Mutex-Cell (nicht nil, kein Fehler).
  got, err := evalStr(`(lock-make)`)
  if err != nil {
    t.Fatalf("lock-make Fehler: %v", err)
  }
  if got == nil {
    t.Fatal("lock-make lieferte nil")
  }
  // Mutex-Cell ist ein Go-Objekt in Env – Stringer gibt etwas zurück,
  // aber kein nil/(). Hier nur prüfen dass es kein Nil ist.
  if got.String() == "()" || got.String() == "NIL" {
    t.Errorf("lock-make = %q, will Mutex-Objekt", got)
  }
}

func TestLockBasic(t *testing.T) {
  // (lock mu expr...) wertet expr unter Lock aus, gibt letzten Wert.
  src := `
    (define mu (lock-make))
    (lock mu (+ 1 2) (* 3 4))
  `
  evalEq(t, src, "12")
}

func TestLockSyntaxError(t *testing.T) {
  // lock ohne Argumente → Syntaxfehler.
  evalErr(t, `(lock)`)
}

// TestParfuncResultIsList sichert den Rückgabetyp.
func TestParfuncResultIsList(t *testing.T) {
  got, err := evalStr(`(parfunc r 1 2 3)`)
  if err != nil {
    t.Fatalf("parfunc: %v", err)
  }
  if got.Type != LIST {
    t.Errorf("parfunc Rückgabe Type = %v, will LIST", got.Type)
  }
  _ = strings.Contains // keep import wenn später gebraucht
}
