//**********************************************************************
//  lib/swank/inspector_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261029
//**********************************************************************
// SWANK-Inspector (swank.lisp): Antworten werden strukturell geprüft —
// Plist-Felder und (:value "text" n)-ispecs, keine Substrings.
// Spec: docs/superpowers/specs/2026-10-29-swank-inspector-design.md
//**********************************************************************

package swank

import (
  "strconv"
  "testing"

  "golisp2/src/lib"
)

func inspEnv(t *testing.T, setup string) *lib.Env {
  t.Helper()
  env := lib.BaseEnv()
  lib.LoadStdlib(env)
  RegisterSwankEnv(env, func(c *lib.Cell) error { return nil })
  if err := LoadSwankLisp(env); err != nil {
    t.Fatalf("LoadSwankLisp: %v", err)
  }
  if setup != "" {
    if _, err := lib.LoadString(setup, env); err != nil {
      t.Fatalf("setup: %v", err)
    }
  }
  return env
}

// rex schickt op als :emacs-rex und liefert Status (":ok"/":abort") und Wert
// aus dem letzten Event (:return (status wert) id).
func rex(t *testing.T, env *lib.Env, op string) (string, *lib.Cell) {
  t.Helper()
  msg, err := lib.Read(`(:emacs-rex ` + op + ` "USER" t 1)`)
  if err != nil {
    t.Fatalf("read: %v", err)
  }
  res, err := HandleMessage(env, msg)
  if err != nil {
    t.Fatalf("HandleMessage: %v", err)
  }
  events := lib.CellToSlice(res)
  ret := lib.CellToSlice(events[len(events)-1])
  if len(ret) != 3 || ret[0].Val != ":return" {
    t.Fatalf("kein :return: %s", res)
  }
  st := lib.CellToSlice(ret[1])
  return st[0].Val, st[1]
}

func plist(c *lib.Cell, key string) *lib.Cell {
  parts := lib.CellToSlice(c)
  for i := 0; i+1 < len(parts); i += 2 {
    if parts[i].Type == lib.ATOM && parts[i].Val == key {
      return parts[i+1]
    }
  }
  return nil
}

// chunk zerlegt (ispecs len start end).
func chunk(t *testing.T, c *lib.Cell) (ispecs []*lib.Cell, n, start, end int) {
  t.Helper()
  p := lib.CellToSlice(c)
  if len(p) != 4 {
    t.Fatalf("chunk erwartet (ispecs len start end): %s", c)
  }
  return lib.CellToSlice(p[0]), int(p[1].Num), int(p[2].Num), int(p[3].Num)
}

// valueAfter: Text und Nummer des (:value …) direkt hinter dem String-ispec label.
func valueAfter(t *testing.T, insp *lib.Cell, label string) (string, int) {
  t.Helper()
  ispecs, _, _, _ := chunk(t, plist(insp, ":content"))
  for i, s := range ispecs {
    if s.Type == lib.STRING && s.Val == label && i+1 < len(ispecs) {
      v := lib.CellToSlice(ispecs[i+1])
      if len(v) == 3 && v[0].Val == ":value" {
        return v[1].Val, int(v[2].Num)
      }
    }
  }
  t.Fatalf("label %q ohne folgenden :value in %s", label, insp)
  return "", 0
}

func openOK(t *testing.T, env *lib.Env, op string) *lib.Cell {
  t.Helper()
  st, v := rex(t, env, op)
  if st != ":ok" {
    t.Fatalf("%s: Status %s, Wert %s", op, st, v)
  }
  return v
}

const schiffSetup = `(defclass schiff (name (energie 100) heimat))
(define s (make-schiff :name "Beagle"))`

func TestInspectorKlasse(t *testing.T) {
  env := inspEnv(t, schiffSetup)
  v := openOK(t, env, `(swank:init-inspector "s")`)
  if got := plist(v, ":title").Val; got != "#<schiff>" {
    t.Fatalf("title = %q", got)
  }
  if typ, _ := valueAfter(t, v, "Typ: "); typ != "schiff" {
    t.Errorf("Typ = %q", typ)
  }
  if name, _ := valueAfter(t, v, "  name = "); name != `"Beagle"` {
    t.Errorf("name = %q", name)
  }
  if e, _ := valueAfter(t, v, "  energie = "); e != "100" {
    t.Errorf("energie = %q", e)
  }
  _, n, start, end := chunk(t, plist(v, ":content"))
  if start != 0 || end != n {
    t.Errorf("chunk start/end/len = %d/%d/%d", start, end, n)
  }
  // :id zeigt auf das Objekt selbst (Teil 0)
  if id := plist(v, ":id"); id == nil || id.Num != 0 {
    t.Errorf(":id = %v", id)
  }
}

func TestInspectorStructHashListe(t *testing.T) {
  env := inspEnv(t, `(defstruct punkt x y) (define p (make-punkt :x 1 :y 2))
(define h (make-hash-table)) (puthash 'k h 7)`)
  v := openOK(t, env, `(swank:init-inspector "p")`)
  if x, _ := valueAfter(t, v, "  x = "); x != "1" {
    t.Errorf("struct x = %q", x)
  }
  v = openOK(t, env, `(swank:init-inspector "h")`)
  if k, _ := valueAfter(t, v, "  "); k != "k" {
    t.Errorf("hash key = %q", k)
  }
  v = openOK(t, env, `(swank:init-inspector "(list 'a 'b)")`)
  if b, _ := valueAfter(t, v, "  1: "); b != "b" {
    t.Errorf("liste[1] = %q", b)
  }
  v = openOK(t, env, `(swank:init-inspector "(cons 1 2)")`)
  if d, _ := valueAfter(t, v, "  cdr: "); d != "2" {
    t.Errorf("cdr = %q", d)
  }
  v = openOK(t, env, `(swank:init-inspector "(vector 5 6)")`)
  if e, _ := valueAfter(t, v, "  0: "); e != "5" {
    t.Errorf("vector[0] = %q", e)
  }
  v = openOK(t, env, `(swank:init-inspector "'p")`)
  if w, _ := valueAfter(t, v, "Wert: "); w != "(punkt 1 2)" {
    t.Errorf("symbol wert = %q", w)
  }
}

func TestInspectorFehler(t *testing.T) {
  env := inspEnv(t, "")
  if st, _ := rex(t, env, `(swank:init-inspector "(car 5)")`); st != ":abort" {
    t.Errorf("Status = %s, want :abort", st)
  }
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestInspectorNavigation(t *testing.T) {
  env := inspEnv(t, schiffSetup+`
(define heim (make-schiff :name "Erde"))
(setf (schiff-heimat s) heim)`)
  v := openOK(t, env, `(swank:init-inspector "s")`)
  _, n := valueAfter(t, v, "  heimat = ")
  v = openOK(t, env, `(swank:inspect-nth-part `+itoa(n)+`)`)
  if name, _ := valueAfter(t, v, "  name = "); name != `"Erde"` {
    t.Fatalf("nth-part: name = %q", name)
  }
  v = openOK(t, env, `(swank:inspector-pop)`)
  if name, _ := valueAfter(t, v, "  name = "); name != `"Beagle"` {
    t.Fatalf("pop: name = %q", name)
  }
  v = openOK(t, env, `(swank:inspector-next)`)
  if name, _ := valueAfter(t, v, "  name = "); name != `"Erde"` {
    t.Fatalf("next: name = %q", name)
  }
  // Vorwärts-Stack ist jetzt leer
  if _, v := rex(t, env, `(swank:inspector-next)`); v.Type != lib.NIL {
    t.Errorf("next ohne Ziel = %s, want ()", v)
  }
}

func TestInspectorLeereHistoryUndUnbekannterTeil(t *testing.T) {
  env := inspEnv(t, schiffSetup)
  openOK(t, env, `(swank:init-inspector "s")`)
  if _, v := rex(t, env, `(swank:inspector-pop)`); v.Type != lib.NIL {
    t.Errorf("pop ohne History = %s, want ()", v)
  }
  if st, _ := rex(t, env, `(swank:inspect-nth-part 999)`); st != ":abort" {
    t.Errorf("unbekannter Teil: Status %s, want :abort", st)
  }
}

// Kernnutzen: Reinspect zeigt Änderungen am geteilten Zustand.
func TestInspectorReinspectZeigtMutation(t *testing.T) {
  env := inspEnv(t, schiffSetup)
  openOK(t, env, `(swank:init-inspector "s")`)
  if _, err := lib.LoadString(`(decf (schiff-energie s) 30)`, env); err != nil {
    t.Fatal(err)
  }
  v := openOK(t, env, `(swank:inspector-reinspect)`)
  if e, _ := valueAfter(t, v, "  energie = "); e != "70" {
    t.Errorf("energie nach Mutation = %q", e)
  }
  if _, v := rex(t, env, `(swank:quit-inspector)`); v.Type != lib.NIL {
    t.Errorf("quit = %s", v)
  }
}
