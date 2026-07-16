# GoLisp2 – Lisp-Semantik

Referenz für die Stellen, an denen GoLisp2 eine bewusste Entscheidung getroffen
hat. Kompass bei Unklarheiten ist die Common-Lisp-Semantik.

---

## `eq` vs. `equal?`

| Funktion | Vergleicht | Verwendung |
|----------|------------|------------|
| `eq` / `eq?` | Pointer-Identität (dasselbe Objekt im Speicher) | Symbole, Singleton-Objekte, schnelle Identitätsprüfung |
| `equal?` | Strukturelle Gleichheit (rekursiver Inhaltsvergleich) | Listen, Strings, Zahlen, verschiedene Atom-Instanzen |

```lisp
;; eq prüft, ob es DASSELBE Objekt ist
(eq 'foo 'foo)                 ; ()  – zwei verschiedene Atom-Instanzen
(eq (list) (list))             ; t   – Singleton-Nil, identischer Pointer
(eq 5 5)                       ; ()  – jede Zahl ist eine neue Cell

;; equal? prüft, ob der Inhalt gleich ist
(equal? 'foo 'foo)             ; t
(equal? (list 1 2) (list 1 2)) ; t
(equal? 5 5)                   ; t
```

**Empfehlung:** Im Zweifel `equal?`. `eq` nur, wenn Identität *gemeint* ist.

---

## `let` vs. `let*`

| Form | Bindungsmodus | Verwendung |
|------|---------------|------------|
| `let` | parallel – alle Werte werden im äußeren Env ausgewertet | unabhängige Variablen |
| `let*` | sequentiell – jede Bindung sieht die vorherigen | abhängige Variablen |

```lisp
;; let – parallele Bindungen
(let ((x 5)
      (y (+ x 1)))     ; Fehler: x ist hier noch nicht gebunden
  ...)

;; let* – sequentielle Bindungen
(let* ((x 5)
       (y (+ x 1)))    ; OK: x ist bereits 5
  y)                   ; → 6
```

---

## `setq` und `setq*`

`setq` ist ein Alias für `define` — setzt eine Variable global oder lokal:

```lisp
(setq x 10)            ; → x
x                      ; → 10
```

`setq*` setzt mehrere Variablen **sequentiell** (analog `let*`):

```lisp
(setq* a 1
       b (+ a 1)       ; b sieht a = 1
       c (+ b 1))      ; c sieht b = 2
(list a b c)           ; → (1 2 3)
```

---

## `case` – syntaktischer Zucker für `cond`

```lisp
(case 'b
  ((a)   1)            ; einzelner Wert
  ((b c) 2)            ; Liste von Werten
  (else  3))           ; → 2

(case 5
  ((1 2 3) "klein")
  ((4 5 6) "mittel")
  (else    "groß"))    ; → "mittel"
```

Der Vergleich erfolgt mit `equal?` (strukturelle Gleichheit).
`else` oder `t` als Test ist der Default-Fall.

---

## `(eval form)` – globales Environment

`(eval form)` wertet im **globalen** Environment aus (`Env.Root()`), nicht im
dynamischen Lambda-Scope. Das ist Common-Lisp-Semantik und für GoLisp2
essenziell: Definitionen aus `(eval (read …))` — REPL, `swank-repl:listener-eval`,
das selbsterweiternde Muster — müssen global sichtbar bleiben und dürfen nicht
im Child-Env der aufrufenden Lambda-Kette verschwinden.

---

## `defun` / `lambda` / `defmacro` mit mehreren Body-Ausdrücken

Erlaubt. `wrapBegin(exprs)` wrappt sie zur **Definitionszeit** in `(begin …)`.
Ein einzelner Ausdruck bleibt unverpackt — kein Overhead.

---

## FORMAT

`format` folgt Common-Lisp-HyperSpec 22.3.
Implementierung: `lib/format.go`, `format_dirs.go`, `format_blocks.go`.

**Direktiven:**
`~A ~S ~D ~B ~O ~X ~R ~P ~C ~F ~E ~G ~$ ~% ~& ~| ~T ~* ~? ~[ ~{ ~( ~; ~^ ~/fun/ ~~ ~Newline`

Mit Parametern und den Modifiern `:` und `@`.

**Destination:**
- `t` → stdout
- `nil` → Rückgabe als String
- String → anhängen

**Besonderheiten:**
- `~/name/` ruft eine benannte Funktion via `globalFormatEnv`
- `~:;` markiert die Default-Klausel
