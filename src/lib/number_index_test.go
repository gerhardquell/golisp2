//**********************************************************************
//  lib/number_index_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-fable-5-2
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261116
//**********************************************************************
// Ganzzahl-Typ, Spec 2026-11-16 §3.5: Indizes und Anzahlen müssen
// Ganzzahlen sein (auch 2.0 ist ein Fehler), Zähl-Primitiven liefern
// Ganzzahlen.
//**********************************************************************

package lib

import "testing"

func TestIntIndexFloatIstFehler(t *testing.T) {
  const e = "ganze Zahl erwartet"
  runNumCases(t, []struct{ src, want string }{
    {`(nth 1 (list 1 2 3))`, "2"},
    {`(nth 5 (list 1 2 3))`, "()"},
    {`(nth 1.7 (list 1 2 3))`, "ERR:nth: " + e},
    {`(nth 2.0 (list 1 2 3))`, "ERR:nth: " + e},
    {`(take 2 (list 1 2 3))`, "(1 2)"},
    {`(take 2.0 (list 1 2 3))`, "ERR:take: " + e},
    {`(drop 1 (list 1 2 3))`, "(2 3)"},
    {`(drop 1.0 (list 1 2 3))`, "ERR:drop: " + e},
    {`(list-tail (list 1 2 3) 2)`, "(3)"},
    {`(list-tail (list 1 2 3) 2.0)`, "ERR:list-tail: " + e},
    {`(iota 3)`, "(0 1 2)"},
    {`(iota 3.0)`, "ERR:iota: " + e},
    {`(make-list 2 :initial-element 'x)`, "(x x)"},
    {`(make-list 2.0)`, "ERR:make-list: " + e},
    {`(let ((s 0)) (dotimes (i 4 s) (setq s (+ s i))))`, "6"},
    {`(dotimes (i 2.0) i)`, "ERR:dotimes: " + e},
    {`(aref (vector 1 2 3) 1.0)`, "ERR:aref: " + e},
    {`(aset (vector 1 2 3) 1.0 9)`, "ERR:aset: " + e},
    {`(elt (list 1 2 3) 1.0)`, "ERR:elt: " + e},
    {`(subseq (list 1 2 3) 1.0)`, "ERR:subseq: " + e},
    {`(subseq (list 1 2 3) 0 2.0)`, "ERR:subseq: " + e},
    {`(make-array 2.0)`, "ERR:make-array: " + e},
    {`(make-array -1)`, "ERR:make-array: Größe muss >= 0"},
    {`(substring "hallo" 1 3)`, `"al"`},
    {`(substring "hallo" 1.0 3)`, "ERR:substring: " + e},
    {`(substring "hallo" 1 3.0)`, "ERR:substring: " + e},
    {`(chan-make 2.0)`, "ERR:chan-make: " + e},
    {`(nth-value 1 (floor 7 2))`, "1"},
    {`(nth-value 1.0 (floor 7 2))`, "ERR:nth-value: " + e},
    {`(gensym 5)`, "G5"},
    {`(gensym 5.0)`, "ERR:gensym: " + e},
    {`(pool-map 2 (lambda (x) (* x x)) (list 1 2 3))`, "(1 4 9)"},
    {`(pool-map 2.0 (lambda (x) x) (list 1 2 3))`, "ERR:pool-map: " + e},
    {`(pool-map 0 (lambda (x) x) (list 1 2 3))`, "ERR:pool-map: Arbeiterzahl muss > 0"},
    {`(evenp 4)`, "t"},
    {`(evenp 2.0)`, "ERR:evenp: " + e},
    {`(oddp 3.0)`, "ERR:oddp: " + e},
    {`(randomlist 3.0)`, "ERR:randomlist: " + e},
    {`(http-serve 8080.0)`, "ERR:http-serve: " + e},
    {`(webserv :port 8080.5 :html "x")`, "ERR:webserv :port: " + e},
    // ash: exakt auf int64, Überlauf unter eigenem Namen
    {`(ash 1 62)`, "4611686018427387904"},
    {`(ash 1 63)`, "ERR:ash: Ganzzahl-Überlauf"},
    {`(ash -1 63)`, minI},
    {`(ash -1 64)`, "ERR:ash: Ganzzahl-Überlauf"},
    {`(ash 0 1000)`, "0"},
    {`(ash 5 -1000)`, "0"},
    {`(ash -5 -1000)`, "-1"},
    {`(ash 2.0 1)`, "ERR:ash: " + e},
    {`(ash 2 1.0)`, "ERR:ash: " + e},
  })
}

func TestIntZaehlerSindGanzzahlen(t *testing.T) {
  env := BaseEnv()
  if err := LoadStdlib(env); err != nil {
    t.Fatalf("stdlib: %v", err)
  }
  for _, src := range []string{
    `(length (list 1 2 3))`,
    `(cdr (assoc 'heapalloc (memstats)))`,
    `(cdr (assoc 'numgc (memstats)))`,
    `(system "true")`,
    `(system "exit 3")`,
    `(cdr (assoc 'size (file-stat ".")))`,
    `(cdr (assoc 'mtime (file-stat ".")))`,
    `(car (iota 3))`,
    `(let ((n 0)) (dotimes (i 3) (setq n i)) n)`,
  } {
    got, err := evalAll(src, env)
    if err != nil {
      t.Errorf("eval(%q) Fehler: %v", src, err)
      continue
    }
    if got = Primary(got); got.Type != NUMBER || got.NumKind != NumInt {
      t.Errorf("eval(%q) = %s, want Ganzzahl", src, got)
    }
  }
}
