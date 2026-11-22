//**********************************************************************
//  lib/sandbox.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : Claude Opus 5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261117
//**********************************************************************
// Sandbox für KI-Code (Spec 2026-11-17): eigene Root-Umgebung mit
// Whitelist. (make-sandbox &key zeitlimit erlaube), (sandbox-eval sb x).
//
// Die Sandbox-Root registriert ihre Primitiven selbst (BaseEnv +
// LoadStdlib) und verliert danach alles Nicht-Erlaubte. Kopierte Cells
// gingen nicht: withFnDesignator, seqSearchFn und resolveFn fangen das env
// ihrer Registrierung — (funcall 'system …) käme an die volle Umgebung.
//**********************************************************************

package lib

import (
  "context"
  "errors"
  "fmt"
  "os"
  "sync"
  "time"
)

// sandboxGruppen: Register-Gruppen, die ganz zur Grundmenge gehören
// (Spec §3.2) — außer sandboxNichtErlaubt.
var sandboxGruppen = []func(*Env){
  RegisterArith, RegisterStringFuncs, RegisterRegex, RegisterVector,
  RegisterSequence, RegisterHashtables, RegisterJSON, RegisterCellType,
  RegisterCLCompatPrims, RegisterFormat, RegisterCFormat, RegisterTime,
  RegisterDynvar, RegisterConsMutation,
}

// sandboxNichtErlaubt: Ausnahmen in sandboxGruppen. fprintf schreibt Dateien.
var sandboxNichtErlaubt = []string{"fprintf"}

// sandboxEinzeln: Grundmenge außerhalb der Gruppen und Standard-Freigaben
// (Spec §3.2, Entscheidung 3). format t steckt in format (RegisterFormat),
// parfunc und lock sind Spezialformen.
var sandboxEinzeln = []string{
  // Kern: Listen, Typ-Prädikate, Symbole, Fehler
  "t", "nil", "values", "car", "cdr", "cons", "nth", "take", "drop",
  "list-tail", "atom", "null", "list", "%make-struct", "append",
  "string?", "number?", "list?", "symbol?", "atom?", "null?", "eq?",
  "apply", "funcall", "mapcar", "gensym", "intern", "symbol-name", "error",
  "read", "sleep", "documentation", "defined-in", "memstats", "env-symbols",
  "shell-assoc", "symbol->string",
  // Ausgabe
  "print", "println", "printf", "warn", "err-write",
  // Goroutinen
  "spawn", "chan-make", "chan-send", "chan-recv", "chan-try-recv",
  "chan-try-send", "chan-select", "chan-close", "chan-closed?", "lock-make",
  // Datei lesen
  "file-read", "file-exists?", "file-stat", "directory-files",
  "get-working-directory", "get-file-path",
}

// sandboxFormen: gesperrte Spezialformen; prüfen selbst per sandboxSperre.
var sandboxFormen = []string{"exec", "load", "defmain"}

// sandboxNieFreigebbar: auch per :erlaube gesperrt — eine Sandbox in der
// Sandbox könnte sich jede Freigabe selbst geben.
var sandboxNieFreigebbar = map[string]bool{"make-sandbox": true, "sandbox-eval": true}

var (
  sandboxErlaubtOnce sync.Once
  sandboxErlaubtSet  map[string]bool
)

// sandboxErlaubt: Namen der Grundmenge samt Standard-Freigaben. Die
// Gruppen-Namen stammen aus Wegwerf-Envs: so folgt die Liste dem Code der
// Gruppe, statt eine Kopie zu pflegen.
func sandboxErlaubt() map[string]bool {
  sandboxErlaubtOnce.Do(func() {
    m := make(map[string]bool)
    for _, reg := range sandboxGruppen {
      tmp := NewEnv(nil)
      reg(tmp)
      for _, n := range tmp.Symbols() {
        m[n] = true
      }
    }
    for _, n := range sandboxNichtErlaubt {
      delete(m, n)
    }
    for _, n := range sandboxEinzeln {
      m[n] = true
    }
    sandboxErlaubtSet = m
  })
  return sandboxErlaubtSet
}

// sandboxState: Zustand einer Sandbox, Schlüssel ist ihre Root. Nach dem
// Eintragen unveränderlich — Lesen ohne Lock.
type sandboxState struct {
  zeitlimit float64         // Sekunden
  zeitText  string          // zeitlimit wie gedruckt, für die Meldung
  entfernt  map[string]bool // gesperrt: entfernte Bindungen + Spezialformen
}

var (
  sandboxMu sync.RWMutex
  // shortcut: Einträge bleiben bis Prozessende (kein Abbau bei GC), bei
  // vielen kurzlebigen Sandboxen auf weak-Schlüssel umstellen.
  sandboxen = map[*Env]*sandboxState{}
)

// sandboxSperre: Sperrmeldung, wenn root eine Sandbox ist und name dort
// gesperrt; sonst nil.
func sandboxSperre(root *Env, name string) error {
  sandboxMu.RLock()
  st := sandboxen[root]
  sandboxMu.RUnlock()
  if st == nil || !st.entfernt[name] {
    return nil
  }
  return fmt.Errorf("sandbox: %s nicht erlaubt (freigeben mit :erlaube)", name)
}

// unbekanntesSymbol: Fehlerweg der Symbol-Auswertung (env.go).
func unbekanntesSymbol(root *Env, name string) error {
  if err := sandboxSperre(root, name); err != nil {
    return err
  }
  return fmt.Errorf("env: unbekanntes Symbol '%s'", name)
}

// RegisterSandbox: make-sandbox, sandbox-eval.
func RegisterSandbox(env *Env) {
  _ = env.Set("make-sandbox", makeFn(fnMakeSandbox))
  _ = env.Set("sandbox-eval", makeCtxFn(fnSandboxEval))
}

// make-sandbox: (make-sandbox &key zeitlimit erlaube) → #<foreign sandbox>
func fnMakeSandbox(args []*Cell) (*Cell, error) {
  if len(args)%2 != 0 {
    return nil, fmt.Errorf("make-sandbox: Schlüsselwort ohne Wert")
  }
  st := &sandboxState{zeitlimit: 10, zeitText: "10"}
  var erlaube []*Cell
  for i := 0; i < len(args); i += 2 {
    k, v := args[i], args[i+1]
    switch {
    case k.Type == ATOM && k.Val == ":zeitlimit":
      if v.Type != NUMBER || !(v.Num > 0) {
        return nil, fmt.Errorf("make-sandbox: :zeitlimit muss Zahl > 0 sein, got %s", v)
      }
      st.zeitlimit, st.zeitText = v.Num, v.String()
    case k.Type == ATOM && k.Val == ":erlaube":
      items, err := properList("make-sandbox", v)
      if err != nil {
        return nil, err
      }
      erlaube = items
    default:
      return nil, fmt.Errorf("make-sandbox: unbekanntes Schlüsselwort %s", k)
    }
  }

  root := BaseEnv()
  basis := root.Symbols()
  if err := LoadStdlib(root); err != nil {
    return nil, fmt.Errorf("make-sandbox: %v", err)
  }
  erlaubt := sandboxErlaubt()
  st.entfernt = make(map[string]bool)
  for _, n := range basis {
    if !erlaubt[n] {
      st.entfernt[n] = true
    }
  }
  for _, n := range sandboxFormen {
    st.entfernt[n] = true
  }
  for _, c := range erlaube {
    if c.Type == ATOM && sandboxNieFreigebbar[c.Val] {
      return nil, fmt.Errorf("make-sandbox: %s ist nicht freigebbar", c.Val)
    }
    if c.Type != ATOM || !st.entfernt[c.Val] {
      return nil, fmt.Errorf("make-sandbox: %s ist keine sperrbare Funktion", c)
    }
    delete(st.entfernt, c.Val)
  }
  for n := range st.entfernt {
    root.UnsetRoot(n)
  }
  // Unset vor Set: sonst warnt der Redefine-Guard.
  if _, ok := root.UnsetRoot("file-read"); ok {
    _ = root.Set("file-read", makeFn(sandboxFileRead))
  }

  sandboxMu.Lock()
  sandboxen[root] = st
  sandboxMu.Unlock()
  return MakeForeign("sandbox", root), nil
}

// sandboxFileRead: file-read nur für reguläre Dateien — /dev/zero, FIFOs
// und sys-stdin könnten ewig lesen oder blockieren.
func sandboxFileRead(args []*Cell) (*Cell, error) {
  if len(args) > 0 && args[0].Type == STRING {
    pfad := args[0].Val
    if pfad == sysStdin {
      return nil, fmt.Errorf("file-read: in der Sandbox nur reguläre Dateien: %s", pfad)
    }
    if resolved, err := resolvePath(pfad); err == nil {
      if fi, err := os.Stat(resolved); err == nil && !fi.Mode().IsRegular() {
        return nil, fmt.Errorf("file-read: in der Sandbox nur reguläre Dateien: %s", pfad)
      }
    }
  }
  return fnFileRead(args)
}

// errZeitlimit: Grund des Fristablaufs, unterscheidet ihn vom Abbruch des
// Aufrufers (context.Cause).
var errZeitlimit = errors.New("sandbox: Zeitlimit")

// sandbox-eval: (sandbox-eval sb form-oder-string) → Wert der (letzten)
// Form, ausgewertet in der Sandbox-Root unter eigener Frist (Kind des
// Aufrufer-Kontexts). Ein String wird ganz gelesen — read nähme still nur
// die erste Form.
func fnSandboxEval(args []*Cell, ectx evalCtx) (*Cell, error) {
  if len(args) != 2 {
    return nil, fmt.Errorf("sandbox-eval: Syntax: (sandbox-eval sb form-oder-string)")
  }
  obj, ok := ForeignObj(args[0], "sandbox")
  if !ok {
    return nil, fmt.Errorf("sandbox-eval: Sandbox erwartet, got %s", args[0])
  }
  root := obj.(*Env)
  sandboxMu.RLock()
  st := sandboxen[root]
  sandboxMu.RUnlock()

  parent := ectx.context()
  if parent == nil {
    parent = context.Background()
  }
  ctx, cancel := context.WithTimeoutCause(parent,
    time.Duration(st.zeitlimit*float64(time.Second)), errZeitlimit)
  // Kein defer cancel: eine spawn-Goroutine der Sandbox läuft bis zur Frist
  // weiter (Spec §3.4), erst der Fristablauf gibt den Kontext frei.
  // shortcut: je Aufruf lebt ein Timer bis zur Frist — bei sehr vielen
  // Aufrufen in kurzer Zeit cancel an das Ende der letzten Goroutine binden.
  _ = cancel
  inner := evalCtx{run: &runState{ctx: ctx}}

  // Baumkopie an der Grenze (Spec 2026-11-22 §3.4): mit rplaca könnte die
  // Sandbox sonst Host-Listen ändern — und umgekehrt über das Ergebnis.
  form, err := sandboxKopie(args[1], true)
  if err != nil {
    return nil, err
  }
  forms := List(form)
  if args[1].Type == STRING {
    var err error
    if forms, err = ReadAll(args[1].Val); err != nil {
      return nil, err
    }
  }
  // Eigene Goroutine: auch ein Warten, das den Kontext nicht sieht (lock),
  // hält den Aufrufer nicht über die Frist hinaus fest.
  // shortcut: eine so hängende Goroutine bleibt liegen (lock-Deadlock),
  // ausbauen, falls lock kontextfähig wird oder Lecks im Profil auffallen.
  type ergebnis struct {
    res *Cell
    err error
  }
  fertig := make(chan ergebnis, 1)
  go func() {
    res := MakeNil()
    for f := forms; f != nil && f.Type == LIST; f = f.Cdr {
      var err error
      if res, err = evalWithCtx(f.Car, root, inner); err != nil {
        fertig <- ergebnis{nil, err}
        return
      }
    }
    // Ergebniskopie noch in der Sandbox-Goroutine: Frist und Abbruch
    // gelten auch für sie.
    kopie, err := sandboxKopie(res, false)
    fertig <- ergebnis{kopie, err}
  }()
  zeitlimitErr := fmt.Errorf("sandbox: Zeitlimit %s s überschritten", st.zeitText)
  select {
  case e := <-fertig:
    if e.err != nil && context.Cause(ctx) == errZeitlimit {
      return nil, zeitlimitErr
    }
    return e.res, e.err
  case <-ctx.Done():
    if context.Cause(ctx) == errZeitlimit {
      return nil, zeitlimitErr
    }
    return nil, cancelledErr()
  }
}

// sandboxKopie: Kopie über die Sandbox-Grenze (Spec 2026-11-22 §3.4) —
// Host und Sandbox teilen danach keine veränderbare Struktur: Listen,
// Vektoren und Hash-Tabellen werden kopiert, MVALUES wertweise.
// FOREIGN-Handles bleiben geteilt. Funktionen dürfen nur heraus
// (hinein = true sperrt sie): sonst könnte die Sandbox über (car f) und
// rplaca den Code einer Host-Closure umschreiben, die der Host danach mit
// vollen Rechten ausführt. Geteilte Teilstrukturen bleiben geteilt (Memo,
// O(n) auch für DAGs); ein Zyklus ist ein Fehler.
func sandboxKopie(c *Cell, hinein bool) (*Cell, error) {
  g := grenzKopie{kopie: map[*Cell]*Cell{}, offen: map[*Cell]bool{}, hinein: hinein}
  if c == nil || c.Type != MVALUES {
    return g.zelle(c, 0)
  }
  vals := ValuesToSlice(c)
  kopie := make([]*Cell, len(vals))
  for i, v := range vals {
    k, err := g.zelle(v, 0)
    if err != nil {
      return nil, err
    }
    kopie[i] = k
  }
  return MakeValues(kopie), nil
}

// grenzKopie: kopie = Quelle → Kopie, offen = noch in Arbeit (Zyklus).
type grenzKopie struct {
  kopie  map[*Cell]*Cell
  offen  map[*Cell]bool
  hinein bool
}

const grenzName = "sandbox-eval"

func (g *grenzKopie) zelle(c *Cell, tiefe int) (*Cell, error) {
  if c == nil {
    return nil, nil
  }
  switch c.Type {
  case LIST, VECTOR, HASHTABLE:
  case LAMBDA, MACRO, FUNC:
    if g.hinein {
      return nil, fmt.Errorf("sandbox-eval: Funktion kann nicht in die Sandbox gereicht werden: %s", c)
    }
    return c, nil
  default:
    return c, nil
  }
  if g.offen[c] {
    return nil, errZirkulaer(grenzName)
  }
  if k, ok := g.kopie[c]; ok {
    return k, nil
  }
  if tiefe > maxTiefe {
    return nil, errZuTief(grenzName)
  }
  switch c.Type {
  case VECTOR:
    return g.vektor(c, tiefe)
  case HASHTABLE:
    return g.tabelle(c, tiefe)
  }
  return g.liste(c, tiefe)
}

// liste: Cdr-Kette iterativ (lange Listen), Car über zelle.
func (g *grenzKopie) liste(c *Cell, tiefe int) (*Cell, error) {
  var kette []*Cell
  defer func() {
    for _, q := range kette {
      delete(g.offen, q)
    }
  }()
  var kopf, ende *Cell
  p := c
  for {
    z := Cons(nil, MakeNil())
    g.kopie[p], g.offen[p] = z, true
    kette = append(kette, p)
    if kopf == nil {
      kopf = z
    } else {
      ende.Cdr = z
    }
    ende = z
    car, err := g.zelle(p.Car, tiefe+1)
    if err != nil {
      return nil, err
    }
    z.Car = car
    p = p.Cdr
    if leer(p) {
      return kopf, nil
    }
    if p.Type != LIST || g.kopie[p] != nil {
      rest, err := g.zelle(p, tiefe+1) // Punktpaar oder geteilter Rest
      if err != nil {
        return nil, err
      }
      ende.Cdr = rest
      return kopf, nil
    }
  }
}

func (g *grenzKopie) vektor(c *Cell, tiefe int) (*Cell, error) {
  v := c.Env.(*Vector)
  v.mu.RLock()
  items := append([]*Cell(nil), v.items...)
  v.mu.RUnlock()
  neu := makeVector(make([]*Cell, len(items)))
  g.kopie[c], g.offen[c] = neu, true
  defer delete(g.offen, c)
  for i, x := range items {
    k, err := g.zelle(x, tiefe+1)
    if err != nil {
      return nil, err
    }
    neu.Env.(*Vector).items[i] = k
  }
  return neu, nil
}

func (g *grenzKopie) tabelle(c *Cell, tiefe int) (*Cell, error) {
  ht := &HashTable{m: make(map[string]hashEntry), test: c.Ht.test}
  neu := &Cell{Type: HASHTABLE, Ht: ht}
  g.kopie[c], g.offen[c] = neu, true
  defer delete(g.offen, c)
  for _, e := range c.Ht.snapshot() {
    k, err := g.zelle(e.key, tiefe+1)
    if err != nil {
      return nil, err
    }
    v, err := g.zelle(e.val, tiefe+1)
    if err != nil {
      return nil, err
    }
    // eql-Schlüssel einer Liste hängen an der Adresse: neu berechnen
    schluessel, err := ht.keyOf(grenzName, k)
    if err != nil {
      return nil, err
    }
    ht.m[schluessel] = hashEntry{key: k, val: v}
  }
  return neu, nil
}
