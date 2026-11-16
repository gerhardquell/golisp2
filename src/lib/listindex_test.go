//**********************************************************************
//  lib/listindex_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Fable 5.2
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261116
//**********************************************************************

package lib

import "testing"

// nth/take/drop/list-tail als Go-Primitiven: Verhalten wie die früheren
// Lisp-Fassungen, take ohne Rekursionstiefe.
func TestListIndexPrims(t *testing.T) {
  runNumCases(t, []struct{ src, want string }{
    {`(nth 0 (list 1 2 3))`, "1"},
    {`(nth 2 (list 1 2 3))`, "3"},
    {`(nth 5 (list 1 2 3))`, "()"},
    {`(nth -1 (list 1 2 3))`, "()"},
    {`(nth 0 ())`, "()"},
    {`(take 2 (list 1 2 3))`, "(1 2)"},
    {`(take 0 (list 1 2 3))`, "()"},
    {`(take -1 (list 1 2 3))`, "()"},
    {`(take 9 (list 1 2 3))`, "(1 2 3)"},
    {`(drop 2 (list 1 2 3))`, "(3)"},
    {`(drop 0 (list 1 2 3))`, "(1 2 3)"},
    {`(drop -1 (list 1 2 3))`, "(1 2 3)"},
    {`(drop 9 (list 1 2 3))`, "()"},
    {`(list-tail (list 1 2 3) 1)`, "(2 3)"},
    {`(list-tail (list 1 2 3) 9)`, "()"},
    {`(length (take 200000 (iota 200000)))`, "200000"},
    {`(trap (nth 1.0 (list 1 2)) (lambda (e) :fehler))`, ":fehler"},
    {`(trap (nth 0 5) (lambda (e) :fehler))`, ":fehler"},
    {`(trap (list-tail (list 1 2) -1) (lambda (e) :fehler))`, ":fehler"},
  })
}
