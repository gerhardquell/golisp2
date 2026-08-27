# GoLisp Änderungen – 20260306

Anlass: Integration der gotools-Kompressoren in das GP-Kompressionssystem.
Diese Session hat mehrere fehlende Primitiven aufgedeckt und behoben.

---

## Neue Datei: `lib/shellcmd.go`

Registriert via `RegisterShellCmd(env)` in `BaseEnv()`.

### `(system "cmd")` → Zahl (Exit-Code)
Führt ein Shell-Kommando via `/bin/sh -c` aus.
- Rückgabe: `0` bei Erfolg, sonst Exit-Code des Prozesses
- Wirft **keinen** Fehler bei non-zero Exit — Aufrufer prüft selbst
- Entspricht C's `system()` / Shell's `$?`

```lisp
(define ret (system "xz -c file.txt > file.xz"))
(if (= ret 0) "OK" "FEHLER")
```

### `(file-stat "path")` → Assoziationsliste oder `nil`
Gibt Dateimetadaten zurück, `nil` wenn Datei nicht existiert.
- Format: `((size . N) (mtime . N))`
- `size`: Dateigröße in Bytes
- `mtime`: Änderungszeit als Unix-Timestamp

```lisp
(define info (file-stat "/tmp/test.txt"))
(cdr (assoc 'size info))   ; → z.B. 50000
```

### `(assoc key alist)` → Paar oder `nil`
Sucht in einer Assoziationsliste nach dem ersten Paar mit passendem Schlüssel.
- Vergleicht strukturell (wie `equal?`)
- Gibt das **ganze Paar** zurück, nicht nur den Wert
- Standard-Lisp-Semantik: `(cdr (assoc 'key lst))` für den Wert

```lisp
(assoc 'b '((a . 1) (b . 2) (c . 3)))  ; → (b . 2)
```

### `(symbol->string 'sym)` → String
Wandelt ein Symbol/Atom in seinen String-Namen um.

```lisp
(symbol->string 'hallo)  ; → "hallo"
(symbol->string 'pipeline)  ; → "pipeline"
```

---

## Geändert: `lib/eval.go` — `catch` fängt alle Fehler

**Vorher:** `catch` fing nur explizite `LispError` ab (geworfen via `(error ...)`).
Go-interne Fehler aus Primitiven (z.B. `cdr: Liste erwartet` von `fnCdr`) wurden
**durchgereicht** und konnten nicht abgefangen werden.

**Nachher:** `catch` fängt **alle** Fehler ab — sowohl `LispError` als auch
normale Go-Fehler aus Primitiven. Go-Fehler werden in `LispError` umgewandelt.

```go
// Alle Fehler abfangen (LispError + Go-Primitive-Fehler)
lispErr, ok := err.(*LispError)
if !ok {
    lispErr = &LispError{Msg: MakeStr(err.Error())}
}
```

**Warum wichtig:** Standard-Lisp-Semantik erwartet, dass `catch`/`handler-case`
alle Laufzeitfehler abfängt. Vorher war es unmöglich, z.B. `(cdr atom)` sicher
zu verwenden.

---

## Geändert: `lib/primitives.go` — neue arithmetische Primitiven

### `(mod a b)` / `(remainder a b)`
Modulo-Operation (Rest bei Division). Beide Namen sind Aliase.

```lisp
(mod 10 3)   ; → 1
(mod 7 2)    ; → 1
```

### `(abs x)`
Absolutwert einer Zahl.

```lisp
(abs -5.3)  ; → 5.3
```

### `(random)` / `(random n)`
Zufallszahl. Ohne Argument: zufälliger nicht-negativer Integer.
Mit Argument: zufälliger Integer im Bereich `[0, n)`.

```lisp
(random)     ; → z.B. 8472938471
(random 10)  ; → 0..9
```

---

## Geändert: `lib/stringfuncs.go` — neue String-Primitiven

### `(string-replace str old new)` → String
Ersetzt alle Vorkommen von `old` durch `new` in `str`.

```lisp
(string-replace "hello world" "world" "lisp")  ; → "hello lisp"
```

### `(string-trim str)` → String
Entfernt führende und abschließende Whitespace-Zeichen.

```lisp
(string-trim "  hallo  ")  ; → "hallo"
```

### `(string-contains str sub)` → `t` oder `nil`
Prüft ob `sub` in `str` enthalten ist.

```lisp
(string-contains "hello world" "world")  ; → t
(string-contains "hello world" "xyz")    ; → nil
```

---

## Zusammenfassung: was fehlte

| Primitiv | Gefehlt weil |
|----------|-------------|
| `system` | Neu — Shell-Integration für GP-Kompressor |
| `file-stat` | Neu — Dateigröße für Fitness-Berechnung |
| `assoc` | Klassisches Lisp-Primitiv, noch nicht implementiert |
| `symbol->string` | Fehlte für `write-to-string` in `lib.primitives.lisp` |
| `mod` / `remainder` | Fehlte für `randomFloat` in `lib.population.lisp` |
| `abs` | Fehlte (ergänzt als Bonus) |
| `random` | Fehlte für Tournament-Selektion |
| `string-replace` | Fehlte für Markdown-Stripping der KI-Antworten |
| `string-trim` | Fehlte für Markdown-Stripping |
| `string-contains` | Ergänzt als nützliches Begleiter-Primitiv |
| `catch` (fix) | Fing keine Go-Primitive-Fehler ab |
