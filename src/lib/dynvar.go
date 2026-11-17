//**********************************************************************
//  lib/dynvar.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5-5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261117
//**********************************************************************
// Dynamische Variablen (Spec 2026-11-17): Menge der special-Namen.
// Schlüssel ist die internierte Symbol-Cell; die Cell selbst bleibt
// unverändert (Interning-Invariante). Special gilt global und für immer.
//**********************************************************************
package lib

import (
  "fmt"
  "strings"
  "sync"
)

var (
  specialMu   sync.RWMutex
  specialSyms = map[*Cell]struct{}{}
)

// isSpecial: Vorfilter ohne Lock, weil nur *…*-Namen in die Menge kommen
// (declareSpecial nur über checkSpecialName); progv fragt hier nicht.
// Nur das erste Zeichen: mit *…*-Prüfung fiele isSpecial aus dem Inlining.
func isSpecial(sym *Cell) bool {
  s := sym.Val
  return s != "" && s[0] == '*' && isSpecialLocked(sym)
}

func isSpecialLocked(sym *Cell) bool {
  specialMu.RLock()
  _, ok := specialSyms[sym]
  specialMu.RUnlock()
  return ok
}

// declareSpecial markiert sym und meldet, ob der Name neu war.
func declareSpecial(sym *Cell) bool {
  specialMu.Lock()
  defer specialMu.Unlock()
  if _, ok := specialSyms[sym]; ok {
    return false
  }
  specialSyms[sym] = struct{}{}
  return true
}

// dynBinding ist die unveränderliche Kette der dynamischen Bindungen eines
// evalCtx (Spec §4). Äußerer und innerer ectx teilen sie; beim Verlassen
// eines let gilt einfach wieder die alte Kette. Veränderlich ist nur die
// Box (setq) — darum der Mutex, Worker können sie später teilen.
type dynBinding struct {
  sym  *Cell
  box  *dynBox
  next *dynBinding
}

type dynBox struct {
  mu  sync.Mutex
  val *Cell
}

func (b *dynBox) get() *Cell {
  b.mu.Lock()
  v := b.val
  b.mu.Unlock()
  return v
}

func (b *dynBox) set(v *Cell) {
  b.mu.Lock()
  b.val = v
  b.mu.Unlock()
}

func (d *dynBinding) find(sym *Cell) *dynBox {
  if n := d.findNode(sym); n != nil {
    return n.box
  }
  return nil
}

// bindDyn hängt sym=val vorn an d. Ein älterer Eintrag für sym wäre danach
// verdeckt und fällt aus der neuen Kette (Präfix kopiert, Rest geteilt):
// Tail-Rekursion durch ein dynamisches let wächst so weder in Speicher noch
// Lesezeit. d selbst bleibt unverändert.
func bindDyn(d *dynBinding, sym, val *Cell) *dynBinding {
  if old := d.findNode(sym); old != nil {
    d = d.without(old)
  }
  return &dynBinding{sym: sym, box: &dynBox{val: val}, next: d}
}

func (d *dynBinding) findNode(sym *Cell) *dynBinding {
  for ; d != nil; d = d.next {
    if d.sym == sym {
      return d
    }
  }
  return nil
}

// without kopiert die Knoten vor old; Tiefe ≤ Zahl verschiedener Namen.
func (d *dynBinding) without(old *dynBinding) *dynBinding {
  if d == old {
    return old.next
  }
  return &dynBinding{sym: d.sym, box: d.box, next: d.next.without(old)}
}

// lookupSym ist der einzige Lese-Weg für Variablen (Spec §4): lexikalische
// Frames, dann dynamische Bindungen, dann Root-Env. Frames zuerst, sonst
// sähe (progv '(x) '(1) (let ((x 2)) x)) die 1. Der Weg selbst steckt in
// env.getSymDyn; lookupSym wird geinlined (ein Aufruf wie früher GetSym).
func lookupSym(sym *Cell, env *Env, ectx evalCtx) (*Cell, error) {
  return env.getSymDyn(sym, ectx.dyn)
}

// assignLexOrDyn ist der Zuweisungs-Weg in derselben Reihenfolge wie
// lookupSym: lexikalischer Frame, dann dynamische Bindung. false heißt:
// keins von beiden — der Aufrufer macht weiter wie bisher (global).
func assignLexOrDyn(env *Env, name string, val *Cell, ectx evalCtx) bool {
  if ectx.dyn == nil {
    return false
  }
  sym := MakeAtom(name)
  return env.updateFrames(sym, val) || assignDyn(sym, val, ectx)
}

// assignDyn setzt nur die innerste dynamische Bindung (CL symbol-value);
// false: keine da. defparameter nutzt es direkt (Spec §3.1).
func assignDyn(sym, val *Cell, ectx evalCtx) bool {
  if ectx.dyn == nil {
    return false
  }
  if box := ectx.dyn.find(sym); box != nil {
    box.set(val)
    return true
  }
  return false
}

// lexBound: hält ein lexikalischer Frame sym (Parameter von vor dem
// defvar)? Dann bindet let/let* lexikalisch weiter (Spec §3.2). Nur auf
// dem special-Pfad aufgerufen.
func lexBound(sym *Cell, env *Env) bool {
  _, ok, _ := env.getFrames(sym)
  return ok
}

// rejectSpecial: Go-Binder binden lexikalisch und weisen special-Namen ab
// (Spec §3.3). what ist "Parameter" oder "Variable".
func rejectSpecial(who, what string, sym *Cell) error {
  if sym != nil && sym.Type == ATOM && isSpecial(sym) {
    return fmt.Errorf("%s: %s %s ist special — mit let binden", who, what, sym.Val)
  }
  return nil
}

// checkLambdaList prüft alle Namen einer Lambda-Liste bei der Definition
// (makeLambda), nie beim Aufruf: name, (name default supplied-p),
// &rest/&body-Name und Dotted-Rest. Ein späteres defvar wirkt nicht zurück.
func checkLambdaList(who string, params *Cell) error {
  for p := params; p != nil; p = p.Cdr {
    if p.Type == ATOM {
      return rejectSpecial(who, "Parameter", p)  // (a . rest)
    }
    if p.Type != LIST || p.Car == nil {
      return nil
    }
    param := p.Car
    if param.Type != LIST {
      if err := rejectSpecial(who, "Parameter", param); err != nil { return err }
      continue
    }
    if err := rejectSpecial(who, "Parameter", param.Car); err != nil { return err }
    if s := param.Cdr; s != nil && s.Type == LIST && s.Cdr != nil && s.Cdr.Type == LIST {
      if err := rejectSpecial(who, "Parameter", s.Cdr.Car); err != nil { return err }
    }
  }
  return nil
}

func RegisterDynvar(env *Env) {
  _ = env.Set("%check-special", makeFn(fnCheckSpecial))
  _ = env.Set("%declare-special", makeFn(fnDeclareSpecial))
}

// checkSpecialName: darf sym special werden? Ein Name ohne *…* wäre ein
// Fehler-Magnet: stdlib-Lambdas wie (lambda (x) …) schlügen danach fehl.
func checkSpecialName(who string, sym *Cell) error {
  if sym == nil || sym.Type != ATOM {
    return fmt.Errorf("%s: Symbol erwartet, got %s", who, sym)
  }
  name := sym.Val
  if name == "t" {
    return fmt.Errorf("%s: t ist eine Konstante", who)
  }
  if strings.HasPrefix(name, ":") {
    return fmt.Errorf("%s: Keyword %s ist eine Konstante", who, name)
  }
  if !(len(name) >= 3 && strings.HasPrefix(name, "*") && strings.HasSuffix(name, "*")) {
    return fmt.Errorf("%s: %s ohne *…* — special-Namen brauchen *%s*", who, name, name)
  }
  return nil
}

// (%check-special 'defvar 'name) — defvar/defparameter prüfen den Namen
// VOR dem globalen Binden (kein Teil-Effekt), markiert wird danach.
func fnCheckSpecial(args []*Cell) (*Cell, error) {
  if len(args) != 2 || args[0].Type != ATOM {
    return nil, fmt.Errorf("%%check-special: (%%check-special 'wer 'name)")
  }
  if err := checkSpecialName(args[0].Val, args[1]); err != nil {
    return nil, err
  }
  return args[1], nil
}

// (%declare-special 'name) — markiert name; von defvar/defparameter erst
// nach erfolgreichem %defglobal (Spec §3.1).
func fnDeclareSpecial(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("%%declare-special: 1 Argument nötig, got %d", len(args))
  }
  if err := checkSpecialName("%declare-special", args[0]); err != nil {
    return nil, err
  }
  declareSpecial(args[0])
  return args[0], nil
}
