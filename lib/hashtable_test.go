//**********************************************************************
//  lib/hashtable_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : kimi-k3
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260723
//**********************************************************************
// Hashtables: gethash-MV-Vertrag, Testmodi (eql/equal), Mutation,
// maphash. evalEq nutzt nur BaseEnv — die Primitive brauchen keine
// Stdlib. setf-gethash wird in der Konformitäts-Suite abgedeckt
// (dort lädt die Stdlib).
//**********************************************************************

package lib

import "testing"

func TestHashTableBasis(t *testing.T) {
  evalEq(t, `(hash-table-p (make-hash-table))`, "t")
  evalEq(t, `(hash-table-p 'x)`, "()")
  evalEq(t, `(hash-table-count (make-hash-table))`, "0")
  // gethash liefert MV: Wert + gefunden?; Nicht-MV-Kontext sieht Primärwert
  evalEq(t, `(gethash 'a (make-hash-table))`, "()")
  evalEq(t, `(multiple-value-list (gethash 'a (make-hash-table)))`, "(() ())")
  evalEq(t, `(gethash 'a (make-hash-table) 'def)`, "def")
  evalEq(t, `(multiple-value-list (gethash 'a (make-hash-table) 'def))`, "(def ())")
}

func TestHashTableMutation(t *testing.T) {
  evalEq(t, `
    (let ((h (make-hash-table)))
      (puthash 'a h 1)
      (puthash 'b h 2)
      (list (gethash 'a h) (gethash 'b h) (hash-table-count h)))`, "(1 2 2)")
  // Überschreiben zählt nicht doppelt
  evalEq(t, `
    (let ((h (make-hash-table)))
      (puthash 'a h 1)
      (puthash 'a h 9)
      (list (gethash 'a h) (hash-table-count h)))`, "(9 1)")
  // remhash: t bei Treffer, nil bei Fehlschlag (CL)
  evalEq(t, `
    (let ((h (make-hash-table)))
      (puthash 'a h 1)
      (list (remhash 'a h) (remhash 'a h) (hash-table-count h)))`, "(t () 0)")
  // clrhash liefert die (leere) Tabelle
  evalEq(t, `
    (let ((h (make-hash-table)))
      (puthash 'a h 1)
      (clrhash h)
      (hash-table-count h))`, "0")
}

func TestHashTableTestModi(t *testing.T) {
  // eql (default): Zahlen nach Wert, Symbole nach Name
  evalEq(t, `
    (let ((h (make-hash-table)))
      (puthash 7 h 'sieben)
      (puthash 'k h 'ka)
      (list (gethash 7 h) (gethash 'k h)))`, "(sieben ka)")
  // equal: Strings und Listen strukturell
  evalEq(t, `
    (let ((h (make-hash-table :test 'equal)))
      (puthash "s" h 5)
      (puthash '(1 2) h 7)
      (list (gethash "s" h) (gethash (list 1 2) h)))`, "(5 7)")
  // eq ist Alias für eql-Keying
  evalEq(t, `
    (let ((h (make-hash-table :test 'eq)))
      (puthash 'k h 9)
      (gethash 'k h))`, "9")
  // nil ist ein gültiger Schlüssel
  evalEq(t, `
    (let ((h (make-hash-table)))
      (puthash () h 'leer)
      (gethash () h))`, "leer")
}

func TestHashTableMaphash(t *testing.T) {
  evalEq(t, `
    (let ((h (make-hash-table)) (sum 0))
      (puthash 'x h 10)
      (puthash 'y h 20)
      (maphash (lambda (k v) (set! sum (+ sum v))) h)
      sum)`, "30")
  // maphash liefert nil
  evalEq(t, `(maphash (lambda (k v) k) (make-hash-table))`, "()")
  // Callback darf die Tabelle modifizieren (Snapshot-Iteration)
  evalEq(t, `
    (let ((h (make-hash-table)))
      (puthash 'x h 1)
      (maphash (lambda (k v) (remhash k h)) h)
      (hash-table-count h))`, "0")
}

func TestHashTableFehler(t *testing.T) {
  evalErr(t, `(gethash 'a 'keine-tabelle)`)
  evalErr(t, `(puthash 'a 'keine-tabelle 1)`)
  evalErr(t, `(make-hash-table :test 'equalp)`)
  evalErr(t, `(maphash 'keine-fn (let ((h (make-hash-table))) (puthash 'a h 1) h))`)
}
