# SWANK-Inspector Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `C-c I` in SLIME klappt golisp2-Objekte auf (defclass-Slots, Hashtabellen, Struct-Felder, Listen, Vektoren, Symbole), mit History und `inspector-eval` über `*inspectee*`.

**Architecture:** Reines Lisp in `src/embed/swank.lisp`: globaler Inspector-Zustand (Teil-Tabelle, ispec-Liste, History-Stacks), ein Renderer pro Typ, acht neue Ops im Dispatcher `handle-emacs-rex`. Vorab: `*struct-types*` speichert Slot-Namen statt Anzahl.

**Tech Stack:** golisp2-Lisp (eingebettet), Go-Tests in `src/lib/swank`.

**Spec:** `docs/superpowers/specs/2026-10-29-swank-inspector-design.md` — vor Beginn lesen.

## Global Constraints

- Einrückung 2 Spaces, keine Tabs. Stil der jeweiligen Datei beibehalten; kein `gofmt -w` auf ganze Dateien.
- Datei-Header **neuer** Dateien: Autor `Gerhard Quell - gquell@skequell.de`, CoAutor = **dein eigenes Modell, exakt** (nicht aus diesem Plan abschreiben), Copyright `2026 Gerhard Quell - SKEQuell`, Erstellt `20261029`. Header bestehender Dateien **nie** ändern.
- Commit-Trailer: `Co-Authored-By: <dein exaktes Modell> <noreply@anthropic.com>`.
- Kommentare/Doku deutsch. tmp-Dateien unter `./tmp/`, **nicht** `/tmp`.
- Löschen nur per `gio trash`, nie `rm -r`.
- Build: `./build.sh`; Tests: `go test ./... -count=1`; Lisp-Suite: `./build/golisp2 -t`.
  Baseline: `=== Report: 168 PASS, 0 FAIL, 0 XFAIL, 0 XPASS ===`, Go komplett grün.
- **Nicht committen:** `zeitstempel.txt`, `RETROSPECTIVE.md`, `RESUME.md`, `*.md.d/` (fremde Änderungen im Arbeitsbaum).
- `*` **nicht** umbinden (ist die Multiplikation). Das Objekt heißt `*inspectee*`.
- Antwortformen wie im Rest von `swank.lisp` inline: `(list (list :return (list :ok WERT) id))`, Fehler `(list (list :return (list :abort TEXT) id))`.
- Geprüfte Hilfsmittel (existieren): `take`/`drop` (`(take n lst)`), `min`, `zip` (→ `((a 0) (b 1))`), `iota`, `(apply append '())` → `()`, `length` auf Vektoren, `symbol-name`, `set!` auf `let`-Variable aus `maphash`-Lambda, `swank--value-string`, `swank--read-all`, `swank--eval1`, `swank--arglist name-string`, `swank--find-definition name-string` → `("datei" . zeile)` oder `()`, `%proper-length` (types.lisp, -1 bei improper). Accessoren/`gethash` in Listen liefern Primärwerte (kein MV-Leck). **Nicht vorhanden:** `subseq`, `nthcdr`, `string<`.

## Dateistruktur

| Datei | Aktion | Verantwortung |
|---|---|---|
| `src/embed/stdlib.lisp` | Modify | `*struct-types*` mit Slot-Namen |
| `src/embed/types.lisp` | Modify | `%struct-instance?` zählt über Länge der Slot-Liste |
| `src/lib/types_typep_test.go` | Modify | `TestStructRegistry` neues Format |
| `src/embed/swank.lisp` | Modify | Inspector-Zustand, Renderer, Ops, Dispatch |
| `src/lib/swank/inspector_test.go` | Create | Protokoll-Tests (Struktur) |
| `docs/swank.md`, `tools/gen-reference.lisp`, `docs/referenz-generiert.md`, `src/lib/version.go` | Modify | Doku, Version 0.21.0 |

---

### Task 1: `*struct-types*` speichert Slot-Namen

**Files:**
- Modify: `src/lib/types_typep_test.go:65-70`, `src/embed/stdlib.lisp:383` und `:422`, `src/embed/types.lisp:84-89`

- [ ] **Step 1: Test auf neues Format umstellen**

```go
func TestStructRegistry(t *testing.T) {
  evalStdlibEq(t, `(defstruct punkt x y) *struct-types*`, "((punkt (x y)))")
  evalStdlibEq(t, `(defstruct leer) *struct-types*`, "((leer ()))")
  // Reload ersetzt, dupliziert nicht
  evalStdlibEq(t, `(defstruct punkt x y) (defstruct punkt x y z) *struct-types*`, "((punkt (x y z)))")
  // Default-Werte: nur der Name landet in der Registry
  evalStdlibEq(t, `(defstruct boot (laenge 3) name) *struct-types*`, "((boot (laenge name)))")
}
```

- [ ] **Step 2: Fehlschlag bestätigen**

Run: `go test ./src/lib/ -run TestStructRegistry -count=1`
Expected: FAIL (`((punkt 2))` statt `((punkt (x y)))`)

- [ ] **Step 3: Umsetzen**

`stdlib.lisp:383` Kommentar: `;; *struct-types*: Registry (name (slot-namen)) für type-of/typep (types.lisp)`
und den SWANK-Inspector.
`stdlib.lisp:422`: `(set! *struct-types* (alist-set ',name ,n *struct-types*))` →
`(set! *struct-types* (alist-set ',name ',slot-names *struct-types*))`
(`n` bleibt für `(zip slot-names (iota n))` in Gebrauch).

`types.lisp` in `%struct-instance?`:
`(= (%proper-length x) (+ (cadr entry) 1))` → `(= (%proper-length x) (+ (length (cadr entry)) 1))`

`tools/gen-reference.lisp:96`: Beschreibung → `"Registry (name (slot-namen)) der definierten Structs, fuer type-of/typep und SWANK-Inspector (stdlib.lisp)"`.

- [ ] **Step 4: Grün bestätigen**

Run: `./build.sh && go test ./... -count=1 && ./build/golisp2 -t 2>&1 | tail -1`
Expected: alles grün, 168 PASS. `rg -n 'struct-types' src/ tools/ tests/` zeigt keinen weiteren Leser, der die Anzahl erwartet.

- [ ] **Step 5: Commit**

```bash
git add src/embed/stdlib.lisp src/embed/types.lisp src/lib/types_typep_test.go tools/gen-reference.lisp
git commit -m "refactor: *struct-types* speichert Slot-Namen statt Anzahl"
```

---

### Task 2: Testgerüst und `init-inspector` (Darstellung)

**Files:**
- Create: `src/lib/swank/inspector_test.go`
- Modify: `src/embed/swank.lisp` (neuer Abschnitt vor `;; Generic OK-Stub` bzw. am Dateiende; Dispatch in `handle-emacs-rex` vor dem `(else …)`-Zweig)

**Interfaces:**
- Produces (Lisp): `*inspectee*`, `swank:init-inspector (string id)`, `swank--insp-open (x)` → `(:title s :id n :content chunk)`, `swank--insp-range (from to)` → `(ispecs len from end)`.

- [ ] **Step 1: Testdatei mit Helfern und Darstellungs-Tests**

```go
//**********************************************************************
//  lib/swank/inspector_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : <dein exaktes Modell>
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261029
//**********************************************************************
// SWANK-Inspector (swank.lisp): Antworten werden strukturell geprüft —
// Plist-Felder und (:value "text" n)-ispecs, keine Substrings.
// Spec: docs/superpowers/specs/2026-10-29-swank-inspector-design.md
//**********************************************************************

package swank

import (
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
```

- [ ] **Step 2: Fehlschlag bestätigen**

Run: `go test ./src/lib/swank/ -run Inspector -count=1`
Expected: FAIL (heute liefert der Fallback `(:ok ())`, `chunk`/`plist` finden nichts)

- [ ] **Step 3: Inspector-Kern in `swank.lisp`**

```lisp
;; === Inspector ======================================================
;; Spec: docs/superpowers/specs/2026-10-29-swank-inspector-design.md
;; Zustand global im Env (--swank: ein Env pro Verbindung). Teil 0 ist
;; immer das Objekt selbst (= :id des Titels).

;; *inspectee*: aktuelles Objekt für inspector-eval. Nicht `*` wie in CL —
;; `*` ist die Multiplikation (Lisp-1).
(defvar *inspectee* ())
(defvar swank--insp-parts (make-hash-table))
(defvar swank--insp-count 0)
(defvar swank--insp-content ())
(defvar swank--insp-back ())
(defvar swank--insp-fwd ())
(defvar swank--insp-limit 500)

;; swank--insp-part: obj in die Teil-Tabelle, liefert seine Nummer.
(defun swank--insp-part (obj)
  (let ((n swank--insp-count))
    (puthash n swank--insp-parts obj)
    (set! swank--insp-count (+ n 1))
    n))

(defun swank--insp-value (obj)
  (list :value (swank--value-string obj) (swank--insp-part obj)))

(defun swank--insp-line (label obj)
  (list label (swank--insp-value obj) "\n"))

(defun swank--insp-lines (labels objs)
  (apply append (mapcar (lambda (p) (swank--insp-line (car p) (cadr p)))
                        (zip labels objs))))

(defun swank--insp-indexed (objs)
  (swank--insp-lines (mapcar (lambda (i) (format nil "  ~a: " i)) (iota (length objs)))
                     objs))

(defun swank--insp-slots (slots vals)
  (append (list (list :label "Slots:") "\n")
          (swank--insp-lines (mapcar (lambda (s) (format nil "  ~a = " s)) slots) vals)))

;; Reihenfolge = Go-Map-Reihenfolge (unsortiert, Spec: kein string<).
(defun swank--insp-hash (h)
  (let ((rows ()))
    (maphash (lambda (k v)
               (set! rows (cons (list "  " (swank--insp-value k) " = " (swank--insp-value v) "\n")
                                rows)))
             h)
    (append (list (format nil "Einträge: ~a" (hash-table-count h)) "\n")
            (apply append rows))))

(defun swank--insp-symbol (s)
  (let* ((name (symbol-name s))
         (val  (trap (list (eval s)) (lambda (e) ())))
         (al   (swank--arglist name))
         (loc  (swank--find-definition name)))
    (append (if (null val) (list "Ungebunden" "\n") (swank--insp-line "Wert: " (car val)))
            (if (null al) () (list (string-append "Arglist: " al) "\n"))
            (if (null loc) () (list (format nil "Quelle: ~a:~a" (car loc) (cdr loc)) "\n")))))

;; Prüfreihenfolge wie Spec: Klasse vor Hashtabelle, Struct vor Liste.
(defun swank--insp-body (x)
  (let ((k (type-of x)))
    (cond ((assoc k *class-types*)
           (let ((slots (cadr (assoc k *class-types*))))
             (swank--insp-slots slots (mapcar (lambda (s) (gethash s x)) slots))))
          ((hash-table-p x) (swank--insp-hash x))
          ((assoc k *struct-types*)
           (swank--insp-slots (cadr (assoc k *struct-types*)) (cdr x)))
          ((eq k 'vector)
           (swank--insp-indexed (mapcar (lambda (i) (aref x i)) (iota (length x)))))
          ((and (pair? x) (>= (%proper-length x) 0)) (swank--insp-indexed x))
          ((pair? x)
           (append (swank--insp-line "  car: " (car x)) (swank--insp-line "  cdr: " (cdr x))))
          ((and (symbol? x) (not (null x))) (swank--insp-symbol x))
          (t (list (string-append "Wert: " (swank--value-string x)) "\n")))))

(defun swank--insp-range (from to)
  (let* ((len (length swank--insp-content))
         (end (min to len)))
    (list (take (- end from) (drop from swank--insp-content)) len from end)))

;; swank--insp-open: x rendern und zum aktuellen Objekt machen.
(defun swank--insp-open (x)
  (clrhash swank--insp-parts)
  (set! swank--insp-count 0)
  (set! *inspectee* x)
  (let ((id (swank--insp-part x)))
    (set! swank--insp-content
          (append (swank--insp-line "Typ: " (type-of x)) (swank--insp-body x)))
    (list :title (swank--value-string x) :id id
          :content (swank--insp-range 0 swank--insp-limit))))

(defun swank:init-inspector (string id)
  (trap
    (let ((x (swank--eval1 (car (swank--read-all string)))))
      (set! swank--insp-back ())
      (set! swank--insp-fwd ())
      (list (list :return (list :ok (swank--insp-open x)) id)))
    (lambda (err)
      (list (list :return (list :abort (swank--value-string err)) id)))))
```

Dispatch (`handle-emacs-rex`, vor `;; Unbekannte Ops`):

```lisp
      ;; Inspector (C-c I)
      ((equal? op 'swank:init-inspector)
       (swank:init-inspector (cadr form) id))
```

Falls ein Test an einer Stelle hängt, die hier als „geprüft“ markiert ist: in `./tmp/` nachstellen, nicht raten. Die Labels in den Tests sind wörtlich die Strings aus dem Lisp-Code (inkl. führender Leerzeichen).

- [ ] **Step 4: Grün bestätigen**

Run: `./build.sh && go test ./src/lib/swank/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add src/embed/swank.lisp src/lib/swank/inspector_test.go
git commit -m "feat(swank): init-inspector – Objekte, Hashtabellen, Structs, Listen"
```

---

### Task 3: Navigation (`inspect-nth-part`, `pop`, `next`, `reinspect`, `quit`)

**Files:** Modify `src/embed/swank.lisp`, `src/lib/swank/inspector_test.go`

- [ ] **Step 1: Tests anhängen**

```go
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
```

Hilfsfunktion dazu (in der Testdatei, `strconv` importieren):

```go
func itoa(n int) string { return strconv.Itoa(n) }
```

- [ ] **Step 2: Fehlschlag bestätigen** — `go test ./src/lib/swank/ -run Inspector -count=1` → FAIL

- [ ] **Step 3: Umsetzen**

```lisp
(defun swank:inspect-nth-part (n id)
  (let ((hit (multiple-value-list (gethash n swank--insp-parts))))
    (if (null (cadr hit))
        (list (list :return (list :abort (format nil "inspect-nth-part: kein Teil ~a" n)) id))
        (begin
          (set! swank--insp-back (cons *inspectee* swank--insp-back))
          (set! swank--insp-fwd ())
          (list (list :return (list :ok (swank--insp-open (car hit))) id))))))

(defun swank:inspector-pop (id)
  (if (null swank--insp-back)
      (list (list :return (list :ok ()) id))
      (let ((x (car swank--insp-back)))
        (set! swank--insp-back (cdr swank--insp-back))
        (set! swank--insp-fwd (cons *inspectee* swank--insp-fwd))
        (list (list :return (list :ok (swank--insp-open x)) id)))))

(defun swank:inspector-next (id)
  (if (null swank--insp-fwd)
      (list (list :return (list :ok ()) id))
      (let ((x (car swank--insp-fwd)))
        (set! swank--insp-fwd (cdr swank--insp-fwd))
        (set! swank--insp-back (cons *inspectee* swank--insp-back))
        (list (list :return (list :ok (swank--insp-open x)) id)))))

(defun swank:inspector-reinspect (id)
  (list (list :return (list :ok (swank--insp-open *inspectee*)) id)))

(defun swank:quit-inspector (id)
  (clrhash swank--insp-parts)
  (set! swank--insp-count 0)
  (set! *inspectee* ())
  (set! swank--insp-content ())
  (set! swank--insp-back ())
  (set! swank--insp-fwd ())
  (list (list :return (list :ok ()) id)))
```

Dispatch ergänzen:

```lisp
      ((equal? op 'swank:inspect-nth-part)
       (swank:inspect-nth-part (cadr form) id))
      ((equal? op 'swank:inspector-pop)
       (swank:inspector-pop id))
      ((equal? op 'swank:inspector-next)
       (swank:inspector-next id))
      ((equal? op 'swank:inspector-reinspect)
       (swank:inspector-reinspect id))
      ((equal? op 'swank:quit-inspector)
       (swank:quit-inspector id))
```

- [ ] **Step 4: Grün** — `./build.sh && go test ./src/lib/swank/ -count=1` → PASS
- [ ] **Step 5: Commit** — `git commit -m "feat(swank): Inspector-Navigation und reinspect"` (nur `swank.lisp`, `inspector_test.go`)

---

### Task 4: `inspector-range` und `inspector-eval`

**Files:** Modify `src/embed/swank.lisp`, `src/lib/swank/inspector_test.go`

- [ ] **Step 1: Tests anhängen**

```go
func TestInspectorRange(t *testing.T) {
  env := inspEnv(t, "")
  v := openOK(t, env, `(swank:init-inspector "(iota 1000)")`)
  ispecs, n, start, end := chunk(t, plist(v, ":content"))
  if start != 0 || end != 500 || len(ispecs) != 500 || n <= 500 {
    t.Fatalf("erster Chunk: len(ispecs)=%d n=%d start=%d end=%d", len(ispecs), n, start, end)
  }
  r := openOK(t, env, `(swank:inspector-range 500 1000)`)
  ispecs, n2, start, end := chunk(t, r)
  if n2 != n || start != 500 || end != 1000 || len(ispecs) != 500 {
    t.Errorf("range: len(ispecs)=%d n=%d start=%d end=%d", len(ispecs), n2, start, end)
  }
  // Client schickt als Obergrenze gern most-positive-fixnum
  r = openOK(t, env, `(swank:inspector-range 2900 4611686018427387903)`)
  if _, _, _, end := chunk(t, r); end != n {
    t.Errorf("range bis Ende: end=%d, want %d", end, n)
  }
}

func TestInspectorEval(t *testing.T) {
  env := inspEnv(t, schiffSetup)
  openOK(t, env, `(swank:init-inspector "s")`)
  st, v := rex(t, env, `(swank:inspector-eval "(setf (schiff-energie *inspectee*) (* 2 21))")`)
  if st != ":ok" || v.Type != lib.STRING || v.Val != "42" {
    t.Fatalf("inspector-eval = %s %s, want :ok \"42\"", st, v)
  }
  v = openOK(t, env, `(swank:inspector-reinspect)`)
  if e, _ := valueAfter(t, v, "  energie = "); e != "42" {
    t.Errorf("energie = %q", e)
  }
  if st, _ := rex(t, env, `(swank:inspector-eval "(car 5)")`); st != ":abort" {
    t.Errorf("Fehler: Status %s, want :abort", st)
  }
}
```

(`(* 2 21)` prüft nebenbei, dass `*` die Multiplikation geblieben ist. `(iota 1000)` → 3000 ispecs.)

- [ ] **Step 2: Fehlschlag bestätigen** → FAIL

- [ ] **Step 3: Umsetzen**

```lisp
(defun swank:inspector-range (from to id)
  (list (list :return (list :ok (swank--insp-range from to)) id)))

(defun swank--insp-eval-all (forms last)
  (if (null forms)
      last
      (swank--insp-eval-all (cdr forms) (swank--eval1 (car forms)))))

;; inspector-eval: Formen im Root-Env, *inspectee* = aktuelles Objekt.
;; Buffer aktualisiert sich nicht selbst (wie SBCL) — `g` in Emacs.
(defun swank:inspector-eval (string id)
  (trap
    (let ((result (swank--insp-eval-all (swank--read-all string) ())))
      (list (list :return (list :ok (swank--value-string result)) id)))
    (lambda (err)
      (list (list :return (list :abort (swank--value-string err)) id)))))
```

Dispatch:

```lisp
      ((equal? op 'swank:inspector-range)
       (swank:inspector-range (cadr form) (caddr form) id))
      ((equal? op 'swank:inspector-eval)
       (swank:inspector-eval (cadr form) id))
```

- [ ] **Step 4: Grün** — `./build.sh && go test ./... -count=1 && ./build/golisp2 -t 2>&1 | tail -1` → alles grün, 168 PASS
- [ ] **Step 5: Commit** — `git commit -m "feat(swank): inspector-range und inspector-eval (*inspectee*)"`

---

### Task 5: Doku, Version, manueller Test

**Files:** `docs/swank.md`, `src/lib/version.go`, `tools/gen-reference.lisp` (falls `*inspectee*` dort gelistet werden soll — nur wenn der Generator swank-Symbole überhaupt kennt; prüfen, nicht raten), `docs/referenz-generiert.md`

- [ ] **Step 1: `docs/swank.md`**
  - Op-Tabelle (ab Zeile ~105): Zeilen für `init-inspector`, `inspect-nth-part`, `inspector-pop`/`-next`, `inspector-reinspect`, `quit-inspector`, `inspector-range`, `inspector-eval` im Stil der vorhandenen Zeilen.
  - Neuer Abschnitt „Inspector“: Tasten (`C-c I` öffnen, `RET` Teil, `l` zurück, `n` vor, `g` neu, `e` eval, `q` schließen), `*inspectee*` statt `*` (Begründung), Hashtabellen unsortiert, Zustand geteilt bei `ServeEnv`, Liste der Nicht-Ops aus der Spec.
  - Zeile „Offen“ (`docs/swank.md:147`): Inspector streichen.
- [ ] **Step 2:** `src/lib/version.go`: `0.20.0` → `0.21.0`. `./build.sh && ./build/golisp2 tools/gen-reference.lisp` (regeneriert `docs/referenz-generiert.md`), Diff ansehen.
- [ ] **Step 3: Volle Suiten** — `go test ./... -count=1` grün, `./build/golisp2 -t` 168 PASS.
- [ ] **Step 4: Commit** — `git commit -m "docs: SWANK-Inspector (0.21.0)"`
- [ ] **Step 5: Manueller Test durch Gerhard** (Session hat kein TTY): `./build/golisp2 --swank` starten, in Emacs `M-x slime-connect` (Port 4242), im REPL `(defclass schiff (name (energie 100)))`, `(define s (make-schiff :name "Beagle"))`, dann `C-c I s RET` → Slots sichtbar; `RET` auf einen Wert, `l`, `n`; `e (decf (schiff-energie *inspectee*) 5)`, `g` → 95; `q`. Ergebnis in die Retro.
