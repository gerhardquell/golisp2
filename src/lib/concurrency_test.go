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
  "math"
  "runtime"
  "strings"
  "sync"
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

// --- Channels: try, Timeout, close (0.3.0) ---

func TestChanRecvValues(t *testing.T) {
  evalEq(t, `(define ch (chan-make 1)) (chan-send ch 5)
    (multiple-value-list (chan-recv ch))`, "(5 t)")
  evalEq(t, `(define ch (chan-make 1)) (chan-send ch 5) (chan-recv ch)`, "5")
}

func TestChanRecvClosedEmptyReturnsAtOnce(t *testing.T) {
  start := time.Now()
  evalEq(t, `(define ch (chan-make 1)) (chan-close ch)
    (multiple-value-list (chan-recv ch))`, "(() ())")
  if d := time.Since(start); d > 500*time.Millisecond {
    t.Errorf("chan-recv auf geschlossenem Channel dauerte %v", d)
  }
}

func TestChanCloseDrainsBuffer(t *testing.T) {
  evalEq(t, `(define ch (chan-make 2)) (chan-send ch 1) (chan-send ch 2)
    (chan-close ch)
    (list (multiple-value-list (chan-recv ch))
          (multiple-value-list (chan-recv ch))
          (multiple-value-list (chan-recv ch)))`, "((1 t) (2 t) (() ()))")
}

func TestChanRecvTimeout(t *testing.T) {
  start := time.Now()
  evalEq(t, `(define ch (chan-make 1))
    (multiple-value-list (chan-recv ch :timeout 0.05))`, "(() ())")
  d := time.Since(start)
  if d < 40*time.Millisecond || d > time.Second {
    t.Errorf("Timeout nach %v, erwartet 40 ms .. 1 s", d)
  }
}

func TestChanRecvTimeoutGetsValue(t *testing.T) {
  evalEq(t, `(define ch (chan-make 1)) (chan-send ch 9)
    (multiple-value-list (chan-recv ch :timeout 0.05))`, "(9 t)")
}

func TestChanRecvTimeoutZero(t *testing.T) {
  evalEq(t, `(define ch (chan-make 1))
    (multiple-value-list (chan-recv ch :timeout 0))`, "(() ())")
}

func TestChanRecvSyntax(t *testing.T) {
  for _, src := range []string{
    `(chan-recv)`,
    `(define ch (chan-make 1)) (chan-recv ch :timeout)`,
    `(define ch (chan-make 1)) (chan-recv ch :wait 1)`,
    `(define ch (chan-make 1)) (chan-recv ch :timeout "x")`,
    `(define ch (chan-make 1)) (chan-recv ch :timeout -1)`,
    `(define ch (chan-make 1)) (chan-recv ch :timeout 1 2)`,
  } {
    _, err := evalStr(src)
    if err == nil || !strings.Contains(err.Error(), "chan-recv: Syntax: (chan-recv ch [:timeout sek])") {
      t.Errorf("eval(%q) Fehler = %v", src, err)
    }
  }
}

func TestChanTryRecv(t *testing.T) {
  evalEq(t, `(define ch (chan-make 1))
    (multiple-value-list (chan-try-recv ch))`, "(() ())")
  evalEq(t, `(define ch (chan-make 1)) (chan-send ch 3)
    (multiple-value-list (chan-try-recv ch))`, "(3 t)")
}

func TestChanTrySend(t *testing.T) {
  evalEq(t, `(define ch (chan-make 1))
    (list (chan-try-send ch 1) (chan-try-send ch 2) (chan-recv ch))`, "(t () 1)")
}

func TestChanTrySendNoReceiver(t *testing.T) {
  evalEq(t, `(chan-try-send (chan-make) 1)`, "()")
}

func TestChanSendClosedError(t *testing.T) {
  _, err := evalStr(`(define ch (chan-make 1)) (chan-close ch) (chan-send ch 1)`)
  if err == nil || !strings.Contains(err.Error(), "chan-send: Channel geschlossen") {
    t.Errorf("Fehler = %v", err)
  }
  _, err = evalStr(`(define ch (chan-make 1)) (chan-close ch) (chan-try-send ch 1)`)
  if err == nil || !strings.Contains(err.Error(), "chan-try-send: Channel geschlossen") {
    t.Errorf("Fehler = %v", err)
  }
}

func TestChanClose(t *testing.T) {
  evalEq(t, `(chan-close (chan-make))`, "t")
  _, err := evalStr(`(define ch (chan-make)) (chan-close ch) (chan-close ch)`)
  if err == nil || !strings.Contains(err.Error(), "chan-close: Channel bereits geschlossen") {
    t.Errorf("Fehler = %v", err)
  }
}

func TestChanClosedP(t *testing.T) {
  evalEq(t, `(chan-closed? (chan-make 1))`, "()")
  evalEq(t, `(define ch (chan-make 1)) (chan-send ch 1) (chan-close ch)
    (list (chan-closed? ch) (chan-recv ch) (chan-closed? ch))`, "(() 1 t)")
}

func TestChanNoChannelArg(t *testing.T) {
  for _, f := range []string{"chan-try-recv", "chan-close", "chan-closed?"} {
    evalErr(t, "("+f+" 1)")
  }
  evalErr(t, `(chan-try-send 1 2)`)
}

func TestChanCloseWakesReceiver(t *testing.T) {
  start := time.Now()
  evalEq(t, `(define ch (chan-make))
    (spawn (lambda () (sleep 50) (chan-close ch)))
    (multiple-value-list (chan-recv ch :timeout 5))`, "(() ())")
  if d := time.Since(start); d > time.Second {
    t.Errorf("Empfänger nicht durch chan-close geweckt, Dauer = %v", d)
  }
}

// blockedSenders zählt Goroutinen, die gerade in chanPut stecken
func blockedSenders() int {
  buf := make([]byte, 1<<20)
  n := runtime.Stack(buf, true)
  return strings.Count(string(buf[:n]), "lib.chanPut(")
}

func TestChanCloseWakesBlockedSender(t *testing.T) {
  env := spawnEnv(t)
  before := blockedSenders()
  if _, err := LoadString(`(define ch (chan-make))
    (define r (spawn (lambda () (chan-send ch 1))))`, env); err != nil {
    t.Fatalf("LoadString: %v", err)
  }
  deadline := time.Now().Add(2 * time.Second)
  for blockedSenders() <= before {
    if time.Now().After(deadline) { t.Fatalf("Sender blockiert nie") }
    time.Sleep(5 * time.Millisecond)
  }
  res, err := LoadString(`(chan-close ch) (chan-recv r :timeout 5)`, env)
  if err != nil {
    t.Fatalf("LoadString: %v", err)
  }
  if got, want := Primary(res).String(), `(:error "chan-send: Channel geschlossen")`; got != want {
    t.Fatalf("got %s, want %s", got, want)
  }
}

func TestChanSendCloseParallel(t *testing.T) {
  for i := 0; i < 50; i++ {
    ch := makeChannelCell(1)
    var wg sync.WaitGroup
    for j := 0; j < 4; j++ {
      wg.Add(2)
      go func() { defer wg.Done(); _, _ = fnChanSend([]*Cell{ch, MakeNum(1)}) }()
      go func() { defer wg.Done(); _, _ = fnChanTrySend([]*Cell{ch, MakeNum(2)}) }()
    }
    wg.Add(2)
    go func() { defer wg.Done(); _, _ = fnChanTryRecv([]*Cell{ch}) }()
    go func() { defer wg.Done(); _, _ = fnChanClose([]*Cell{ch}) }()
    done := make(chan struct{})
    go func() { wg.Wait(); close(done) }()
    select {
    case <-done:
    case <-time.After(5 * time.Second):
      t.Fatalf("Durchlauf %d: Sender hängt nach chan-close", i)
    }
  }
}

// --- chan-select ---

func TestChanSelectOneReady(t *testing.T) {
  evalEq(t, `(define a (chan-make 1)) (define b (chan-make 1))
    (chan-send b 7)
    (multiple-value-list (chan-select (list a b)))`, "(7 t 1)")
  evalEq(t, `(define a (chan-make 1)) (define b (chan-make 1))
    (chan-send a "x")
    (multiple-value-list (chan-select (list a b) :timeout 1))`, `("x" t 0)`)
}

func TestChanSelectBind(t *testing.T) {
  evalEq(t, `(define a (chan-make 1)) (define b (chan-make 1))
    (chan-send b 5)
    (multiple-value-bind (x ok i) (chan-select (list a b))
      (list x ok i))`, "(5 t 1)")
}

func TestChanSelectBothReadyRandom(t *testing.T) {
  evalEq(t, `(define a (chan-make 1)) (define b (chan-make 1))
    (chan-send a 1) (chan-send b 2)
    (multiple-value-bind (x ok i) (chan-select (list a b))
      (list ok (if (= i 0) (= x 1) (= x 2))))`, "(t t)")
}

func TestChanSelectTimeout(t *testing.T) {
  start := time.Now()
  evalEq(t, `(define a (chan-make 1)) (define b (chan-make 1))
    (multiple-value-list (chan-select (list a b) :timeout 0.05))`, "(() () ())")
  if d := time.Since(start); d < 40*time.Millisecond || d > 2*time.Second {
    t.Errorf("Timeout-Dauer = %v", d)
  }
}

func TestChanSelectTimeoutZero(t *testing.T) {
  start := time.Now()
  evalEq(t, `(define a (chan-make 1))
    (multiple-value-list (chan-select (list a) :timeout 0))`, "(() () ())")
  if d := time.Since(start); d > time.Second {
    t.Errorf("sofort erwartet, Dauer = %v", d)
  }
  evalEq(t, `(define a (chan-make 1)) (chan-send a 9)
    (multiple-value-list (chan-select (list a) :timeout 0))`, "(9 t 0)")
}

func TestChanSelectAllClosedEmpty(t *testing.T) {
  evalEq(t, `(define a (chan-make 1)) (define b (chan-make 1))
    (chan-close a) (chan-close b)
    (multiple-value-list (chan-select (list a b)))`, "(() () ())")
}

func TestChanSelectClosedWithBufferDelivers(t *testing.T) {
  evalEq(t, `(define a (chan-make 1)) (define b (chan-make 1))
    (chan-send a 4) (chan-close a) (chan-close b)
    (list (multiple-value-list (chan-select (list a b)))
          (multiple-value-list (chan-select (list a b))))`, "((4 t 0) (() () ()))")
}

func TestChanSelectSkipsClosedWhileWaiting(t *testing.T) {
  evalEq(t, `(define a (chan-make)) (define b (chan-make 1))
    (spawn (lambda () (sleep 50) (chan-close a) (sleep 50) (chan-send b 3)))
    (multiple-value-list (chan-select (list a b) :timeout 5))`, "(3 t 1)")
}

func TestChanSelectBlocksUntilSend(t *testing.T) {
  evalEq(t, `(define a (chan-make)) (define b (chan-make))
    (spawn (lambda () (sleep 50) (chan-send a 8)))
    (multiple-value-list (chan-select (list a b) :timeout 5))`, "(8 t 0)")
}

func TestChanSelectMiddleClosedIndex(t *testing.T) {
  evalEq(t, `(define a (chan-make)) (define b (chan-make)) (define c (chan-make))
    (chan-close b)
    (spawn (lambda () (sleep 50) (chan-send c 9)))
    (multiple-value-list (chan-select (list a b c) :timeout 5))`, "(9 t 2)")
}

func TestChanSelectErrors(t *testing.T) {
  cases := []struct{ src, want string }{
    {`(chan-select)`, "chan-select: Syntax: (chan-select chans [:timeout sek])"},
    {`(chan-select (list (chan-make)) :timeout)`, "chan-select: Syntax: (chan-select chans [:timeout sek])"},
    {`(chan-select (list (chan-make)) :foo 1)`, "chan-select: Syntax: (chan-select chans [:timeout sek])"},
    {`(chan-select (list (chan-make)) :timeout -1)`, "chan-select: Syntax: (chan-select chans [:timeout sek])"},
    {`(chan-select 5)`, "chan-select: Syntax: (chan-select chans [:timeout sek])"},
    {`(chan-select (list))`, "chan-select: leere Channel-Liste"},
    {`(chan-select (list (chan-make) 5))`, "chan-select: kein Channel an Stelle 1"},
    {`(chan-select (list 5 (chan-make)))`, "chan-select: kein Channel an Stelle 0"},
  }
  for _, c := range cases {
    _, err := evalStr(c.src)
    if err == nil || !strings.Contains(err.Error(), c.want) {
      t.Errorf("eval(%q) Fehler = %v, want %q", c.src, err, c.want)
    }
  }
}

// --- timeoutArg ---

func TestTimeoutGrossLaeuftNichtUeber(t *testing.T) {
  evalEq(t, `(define ch (chan-make))
    (spawn (lambda () (sleep 50) (chan-send ch 4)))
    (multiple-value-list (chan-recv ch :timeout 1e12))`, "(4 t)")
  evalEq(t, `(define a (chan-make)) (define b (chan-make))
    (spawn (lambda () (sleep 50) (chan-send b 6)))
    (multiple-value-list (chan-select (list a b) :timeout 1e12))`, "(6 t 1)")
}

func TestTimeoutArgNaNInf(t *testing.T) {
  for _, f := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
    if _, ok := timeoutArg(MakeNum(f)); ok {
      t.Errorf("timeoutArg(%v) ok, want Fehler", f)
    }
  }
  if d, ok := timeoutArg(MakeNum(1e300)); !ok || d != math.MaxInt64 {
    t.Errorf("timeoutArg(1e300) = %v, %v, want MaxInt64", d, ok)
  }
}

func TestPoolMap(t *testing.T) {
  evalStdlibEq(t, `(pool-map 3 (lambda (x) (* x x)) '(1 2 3 4 5))`, "(1 4 9 16 25)") // Reihenfolge bleibt
  evalStdlibEq(t, `(pool-map 2 (lambda (x) x) '())`, "()")
  evalStdlibEq(t, `(pool-map 10 (lambda (x) (+ x 1)) '(1 2))`, "(2 3)") // n > Länge
  // Fehler bleiben im Ergebnis stehen, die anderen Elemente laufen durch
  evalStdlibEq(t, `(pool-map 2 (lambda (x) (if (= x 2) (error "kaputt") x)) '(1 2 3))`, `(1 (:error "kaputt") 3)`)
}

// Höchstens n gleichzeitig: jeder Auftrag belegt einen Platz in einem
// Channel der Kapazität n; gäbe es mehr gleichzeitige, schlüge try-send fehl.
func TestPoolMapObergrenze(t *testing.T) {
  evalStdlibEq(t, `(let ((sem (chan-make 2)))
                     (pool-map 2 (lambda (x)
                                   (if (chan-try-send sem t)
                                       (begin (sleep 20) (chan-recv sem) 'ok)
                                       'zuviel))
                               '(1 2 3 4 5 6)))`, "(ok ok ok ok ok ok)")
}

func TestPoolMapParallel(t *testing.T) {
  // 4 × 100 ms mit 4 Arbeitern: deutlich unter seriellen 400 ms
  evalStdlibEq(t, `(let ((t0 (now)))
                     (pool-map 4 (lambda (x) (sleep 100)) '(1 2 3 4))
                     (< (- (now) t0) 0.35))`, "t")
}

func TestPoolMapFehler(t *testing.T) {
  for _, src := range []string{
    `(pool-map 0 (lambda (x) x) '(1))`,
    `(pool-map 1.5 (lambda (x) x) '(1))`,
    `(pool-map "2" (lambda (x) x) '(1))`,
    `(pool-map 2 (lambda (x) x) 5)`,
  } {
    evalStdlibErr(t, src)
  }
}
