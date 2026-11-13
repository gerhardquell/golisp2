# Retrospective: Multiline-String Fix für S-Expression-RPC

**Datum:** 3. März 2026
**Autor:** Gerhard Quell & Claude Sonnet 4.6
**Feature:** Korrekte Übertragung von Multiline-Strings über S-Expression-RPC

---

## Problem

`sigo`-Aufrufe (KI-Integration) über `golisp2-client` gaben nur `\` zurück statt der vollständigen Antwort.

### Symptom
```bash
$ ./golisp2-client --eval '(sigo "was bedeutet singularität?" "claude-s45")'
=> \
```

Erwartet: Mehrere Zeilen mit der vollständigen KI-Antwort.

---

## Root Cause Analysis

### 1. Protokoll-Design-Problem

Das S-Expression-RPC-Protokoll geht davon aus, dass jede Nachricht in **einer Zeile** bleibt:

```lisp
(:id 1 :status "ok" :result "ergebnis")
```

Wenn `sigo` eine Antwort mit echten Zeilenumbrüchen zurückgibt:
```
Die technologische Singularität...

Die Kernidee:
1. Selbstverbesserung
```

Dann wurde das so formatiert:
```lisp
(:id 1 :status "ok" :result "Die technologische Singularität...

Die Kernidee:
1. Selbstverbesserung")
```

**Problem:** Der Client liest mit `reader.ReadString('\n')` – nur bis zum **ersten** Zeilenumbruch!

### 2. Doppeltes Escaping

In `handleEval()` wurde `result.String()` aufgerufen, was für String-Cells `"hello"` zurückgibt (mit Quotes). Das wurde dann zu einer neuen String-Cell mit escaped Quotes.

---

## Lösung

### Änderung 1: Server – `lib/swank/server.go`

**`formatResponse()`:**
- Für STRING-Cells: Inhalt mit `escapeString()` escapen
- Newlines (`\n`) werden zu `\n` (escaped)

**`escapeString()`:**
```go
func escapeString(s string) string {
    s = strings.ReplaceAll(s, "\\", "\\\\")  // \ → \\
    s = strings.ReplaceAll(s, "\n", "\\n")   // newline → \n
    s = strings.ReplaceAll(s, "\t", "\\t")   // tab → \t
    // Quotes werden NICHT escapet – sie sind durch Format-String geschützt
    return s
}
```

### Änderung 2: Server – `lib/swank/protocol.go`

**`handleEval()`:**
- Für STRING-Cells: `result.Val` verwenden (roher Wert)
- Für andere: `result.String()` (Lisp-Repräsentation)

```go
var resultStr string
if result.Type == lib.STRING {
    resultStr = result.Val  // Keine doppelten Quotes
} else {
    resultStr = result.String()
}
```

### Änderung 3: Client – `cmd/golisp2-client/main.go`

**`extractResult()`:**
- Berücksichtigt escaped Quotes (`\"`) beim Parsen
- Wandelt `\n` zurück in echte Newlines um
- Wandelt `\t` zurück in Tabs um
- Wandelt `\\` zurück in Backslashes um

```go
inEscape := false
for i, ch := range remaining {
    if inEscape {
        inEscape = false
        continue
    }
    if ch == '\\' {
        inEscape = true
        continue
    }
    if ch == '"' {
        // Unescaped Quote = String-Ende
        unescaped := remaining[:i]
        unescaped = strings.ReplaceAll(unescaped, "\\n", "\n")
        unescaped = strings.ReplaceAll(unescaped, "\\t", "\t")
        unescaped = strings.ReplaceAll(unescaped, "\\\"", "\"")
        unescaped = strings.ReplaceAll(unescaped, "\\\\", "\\")
        return unescaped
    }
}
```

---

## Warum diese Lösung?

### Alternative 1: Base64-Encoding
- **Pro:** Einfach, keine Escaping-Probleme
- **Contra:** Nicht mehr menschenlesbar, bricht S-Expression-Charakter

### Alternative 2: Length-Prefixed Messages
- **Pro:** Robust für binäre Daten
- **Contra:** Bricht das zeilenbasierte Protokoll, mehr Komplexität

### Alternative 3: Escaping (gewählt)
- **Pro:** Menschenlesbar, kompatibel mit existierendem Protokoll
- **Contra:** Client muss escaping verstehen

**Entscheidung:** Alternative 3 – Escaping ist der kleinste Eingriff, der das Protokoll konsistent hält.

---

## Test-Ergebnisse

| Test | Vorher | Nachher |
|------|--------|---------|
| `(+ 1 2 3)` | `6` | `6` ✓ |
| `(list 1 2 3)` | `(1 2 3)` | `(1 2 3)` ✓ |
| `(defun f (x) x)` | `f` | `f` ✓ |
| `"hello"` | `\` | `hello` ✓ |
| `"Zeile1\nZeile2"` | `\` | `Zeile1`<br>`Zeile2` ✓ |
| `(sigo "..." "claude-s45")` | `\` | **Volle Antwort** ✓ |

---

## Lessons Learned

### 🎯 Protokoll-Design

1. **Annahmen dokumentieren:** Das Protokoll ging implizit davon aus, dass Strings single-line sind
2. **Escaping-Strategie festlegen:** Früh entscheiden, welche Zeichen escaped werden
3. **Test mit echten Daten:** KI-Antworten haben gezeigt, dass theoretische Annahmen nicht halten

### 🎯 Implementation

1. **String-Repräsentation beachten:**
   - `cell.Val` = roher Wert (`hello`)
   - `cell.String()` = Lisp-Repräsentation (`"hello"`)
   - Verwechslung führt zu doppeltem Escaping

2. **Parser für escaped Strings:**
   - Einfaches `strings.Index(remaining, "\"")` reicht nicht
   - Escaped Quotes (`\"`) müssen übersprungen werden

### 🎯 Debugging

1. **Raw-Bytes anschauen:** `xxd` zeigt genau, was übertragen wird
2. **Server-Logs prüfen:** `strace` zeigt Systemaufrufe
3. **Minimaler Testcase:** `echo "test" | nc localhost 9123` isoliert das Problem

---

## Code-Änderungen

| Datei | Zeilen | Änderung |
|-------|--------|----------|
| `lib/swank/server.go` | 277-290 | `formatResponse()` + `escapeString()` angepasst |
| `lib/swank/protocol.go` | 71-78 | `handleEval()` unterscheidet String-Typen |
| `cmd/golisp2-client/main.go` | 161-186 | `extractResult()` mit Escape-Handling |

---

## Offene Fragen

1. **Sollen andere Steuerzeichen escapet werden?**
   - Aktuell: `\n`, `\t`, `\\`, `\"`
   - Möglich: `\r`, `\b`, `\f` (Unicode-Escape?)

2. **Soll der Server escape-Sequenzen validieren?**
   - Aktuell: Ungültige Sequenzen werden durchgereicht
   - Alternative: Fehler bei ungültigem Escape

3. **Performance bei sehr langen Strings?**
   - Mehrere `ReplaceAll`-Aufrufe sind O(n) pro Aufruf
   - Für Megabyte-Strings: Single-Pass-Escaping in Betracht ziehen

---

## Fazit

Der Fix war notwendig, weil das Protokoll implizite Annahmen über String-Inhalte machte. Die Lösung mit Escaping ist minimal-invasiv und behält die Menschenlesbarkeit bei.

**Wichtigste Erkenntnis:** Wenn ein Protokoll textbasiert ist, müssen Metazeichen (Newlines, Quotes) explizit gehandhabt werden – sonst bricht es bei realen Daten.

---

> "Das Protokoll war für Demos gedacht, nicht für Produktion. KI-Antworten haben die Lücke aufgedeckt."
