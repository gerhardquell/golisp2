# GoLisp2 – Sprachreferenz

GoLisp2 ist ein Lisp-Interpreter in Go mit nativer KI-Anbindung.
Alle Features sind im interaktiven REPL (`go run .`) und in Skript-Dateien
(`go run . skript.lisp`) verfügbar.

---

## Datentypen

| Typ | Beispiel | Beschreibung |
|-----|---------|--------------|
| Zahl | `42` `3.14` `-7` | 64-bit Gleitkomma |
| String | `"Hallo"` | UTF-8, doppelte Anführungszeichen |
| Atom | `foo` `t` `nil` | Symbol / Bezeichner |
| Liste | `(1 2 3)` | Verkettete Cons-Zellen |
| `t` | `t` | Wahrheitswert wahr |
| `()` / `nil` | `()` | Wahrheitswert falsch / leere Liste |

---

## Spezialformen

### Variablen und Funktionen

```lisp
(define x 42)                        ; Variable definieren
(set! x 99)                          ; Variable ändern

(defun quadrat (n) (* n n))          ; Funktion definieren
(defun f (x y)                       ; Multi-Body: mehrere Ausdrücke
  (define s (+ x y))
  (* s s))

(lambda (x) (* x 2))                 ; anonyme Funktion
((lambda (x y) (+ x y)) 3 4)        ; sofort aufrufen → 7
```

### Bedingungen

```lisp
(if (= x 0) "null" "nicht null")     ; if-then-else

(cond                                 ; Mehrfallunterscheidung
  ((= x 1) "eins")
  ((= x 2) "zwei")
  (t        "sonst"))
```

### Lokale Bindungen

```lisp
(let ((x 3) (y 4))                   ; lokale Variablen
  (+ x y))                           ; → 7
```

### Sequenz

```lisp
(begin                               ; mehrere Ausdrücke nacheinander
  (define a 1)
  (define b 2)
  (+ a b))                           ; → 3
```

### Schleifen

```lisp
(while (< i 10)                      ; Loop solange Bedingung wahr
  (set! i (+ i 1)))

(do ((i 0 (+ i 1))                   ; Scheme-style do: var init step
     (s 0 (+ s i)))
    ((= i 5) s))                     ; Abbruch + Rückgabewert → 10
```

### Quasiquote

```lisp
`(1 2 ,(+ 1 2))                      ; → (1 2 3)
`(a ,@(list 1 2) b)                  ; ,@ spleißt Liste ein → (a 1 2 b)
```

### Makros

```lisp
(defmacro when (test . body)         ; Makro definieren
  `(if ,test (begin ,@body) ()))

(when (> x 0)                        ; Makro aufrufen
  (println "positiv")
  (println x))
```

### Nebenläufigkeit

```lisp
(parfunc ergebnis                    ; mehrere Ausdrücke parallel auswerten
  (sigo "frage1" "claude-h")
  (sigo "frage2" "gemini-p"))
ergebnis                             ; → Liste der Ergebnisse

(define m (lock-make))               ; Mutex erstellen
(lock m (set! x (+ x 1)))           ; kritischen Abschnitt schützen

(define ch (chan-make))              ; Channel erstellen
(chan-send ch 42)                    ; Wert senden
(chan-recv ch)                       ; Wert empfangen
```

### Fehlerbehandlung

```lisp
(error "etwas ging schief")          ; Fehler auslösen

(catch                               ; Fehler abfangen
  (/ 1 0)
  (lambda (e) (string-append "Fehler: " e)))
```

### Sonstiges

```lisp
(quote (a b c))                      ; Liste nicht auswerten → (a b c)
'(a b c)                             ; Kurzform für quote

(eval '(+ 1 2))                      ; Ausdruck auswerten → 3
(load "datei.lisp")                  ; Datei laden (mit Suchpfad, siehe unten)
(gensym)                             ; eindeutiges Symbol erzeugen
```

---

## Eingebaute Funktionen

### Arithmetik

```lisp
(+ 1 2 3)      ; → 6
(- 10 3)       ; → 7
(* 4 5)        ; → 20
(/ 10 4)       ; → 2.5
```

### Vergleiche

```lisp
(= 3 3)        ; → t
(< 2 5)        ; → t
(> 5 2)        ; → t
(>= 3 3)       ; → t
(<= 2 3)       ; → t
```

### Listen

```lisp
(list 1 2 3)           ; → (1 2 3)
(car '(1 2 3))         ; → 1          (erstes Element)
(cdr '(1 2 3))         ; → (2 3)      (Rest)
(cons 0 '(1 2 3))      ; → (0 1 2 3)  (vorne anhängen)
(atom 'x)              ; → t          (ist Atom?)
(null '())             ; → t          (ist leer?)
(equal? '(1 2) '(1 2)) ; → t          (strukturell gleich?)
(apply + '(1 2 3))     ; → 6          (Funktion auf Liste anwenden)
(mapcar (lambda (x) (* x x)) '(1 2 3)) ; → (1 4 9)
```

### Strings

```lisp
(string-length "Hallo")              ; → 5  (Zeichen, nicht Bytes)
(string-append "Hallo" " " "Welt")   ; → "Hallo Welt"
(substring "Hallo" 1 3)              ; → "al"
(string-upcase "hallo")              ; → "HALLO"
(string-downcase "HALLO")            ; → "hallo"
(string->number "42")                ; → 42
(number->string 3.14)                ; → "3.14"
```

### I/O

```lisp
(print "Hallo")                      ; ausgeben ohne Zeilenumbruch
(println "Hallo")                    ; ausgeben mit Zeilenumbruch
(read "(+ 1 2)")                     ; String → Lisp-Ausdruck parsen
```

### Dateien

```lisp
(file-write   "datei.txt" "Inhalt")  ; Datei schreiben (überschreiben)
(file-append  "datei.txt" "mehr")    ; Inhalt anhängen
(file-read    "datei.txt")           ; Datei lesen → String
(file-exists? "datei.txt")           ; → t oder ()
(file-delete  "datei.txt")           ; Datei löschen
```

### Bibliothek-Suchpfad

Die Funktion `load` durchsucht eine definierte Liste von Pfaden, ähnlich wie Python's `sys.path` oder der Shell's `PATH`:

```lisp
(load "utils.lisp")                  ; Sucht in mehreren Verzeichnissen
```

**Suchreihenfolge:**

1. **Wie angegeben** — Aktuelles Verzeichnis oder absoluter/relativer Pfad
2. **`/lib/golib`** — Systemweite Bibliotheken
3. **`/usr/local/lib/golib`** — Lokale Systembibliotheken
4. **`./golib`** — Projektlokale Bibliotheken
5. **`GOLISP_PATH`** — Benutzerdefinierte Pfade (doppelpunkt-getrennt)

**Beispiele:**

```lisp
; Aus aktuellem Verzeichnis laden (rückwärtskompatibel)
(load "meinskript.lisp")

; Aus ./golib/ Unterverzeichnis laden
; (sucht ./golib/utils.lisp)
(load "utils.lisp")

; Absolute Pfade funktionieren wie immer
(load "/home/user/lib/stdlib.lisp")
```

**Umgebungsvariable:**

```bash
# Benutzerdefinierte Bibliotheksverzeichnisse hinzufügen
export GOLISP_PATH=/opt/golisp2:/home/user/mylisp

./golisp2 -e '(load "mylib.lisp")'  ; Durchsucht auch GOLISP_PATH
```

### KI (sigoREST)

```lisp
(sigo "erkläre Quasiquote" "claude-h")   ; KI-Anfrage → String
(sigo-models)                             ; verfügbare Modelle → Liste
(sigo-host "http://anderer-host:9080")    ; Host wechseln
```

Verfügbare Modell-Kürzel: `claude-h` `gemini-p` `gpt41`
und alle lokalen Ollama-Modelle (z.B. `ollama-gemma3-4b`).

### Genetische Algorithmen

```lisp
; GA-Handle erzeugen: Typ, Genom-Länge, Populationsgröße, Fitness-Funktion
(define ga (ga-create 'bit1 10 20 (lambda (g) (apply + g))))

; Lebenszyklus
(ga-init ga)         ; Population mit Zufallswerten initialisieren
(ga-calc ga)         ; Fitness berechnen und sortieren (parallel)
(ga-cross ga 2)      ; Crossover mit Blockgröße 2
(ga-select ga 4)     ; Die besten 4 behalten
(ga-mut ga 0.1)      ; 10% Mutationswahrscheinlichkeit
(ga-result ga)       ; Aktuelle Fitness-Scores als Liste
(ga-print ga 5)      ; Top 5 auf stdout ausgeben
(ga? ga)             ; => t  (Handle-Prädikat)
```

**GenTypen:** `bit1`, `bit2`, `bit4`, `bit8`, `biti` (Integer), `bitf` (Float).

**Hinweis:** `ga-calc` ruft die Fitness-Funktion parallel auf. Sie muss rein
und thread-sicher sein (keine Mutation geteilter Variablen).

---

## REPL

```bash
go run .              # REPL starten
go run . skript.lisp  # Skript ausführen
go run . -t           # Testmodus
```

| Feature | Beschreibung |
|---------|-------------|
| **Syntax-Highlighting** | Klammern nach Tiefe eingefärbt (6 Farben) |
| **Multi-line** | Enter bei offenem Ausdruck → automatische Einrückung |
| **History** | ↑/↓ — persistent in `~/.golisp_history` |
| **Ctrl+C** | aktuelle Eingabe abbrechen |
| **Ctrl+D** | REPL beenden |

---

## Server-Modus (golisp2d)

GoLisp2 kann als SWANK-ähnlicher TCP-Server laufen:

### Server starten

```bash
golisp2d --port 4321        # Default: localhost:4321
golisp2d --host 0.0.0.0     # Externe Verbindungen erlauben
```

Umgebungsvariablen: `GOLISP_HOST`, `GOLISP_PORT`

### Client verwenden

```bash
# Expression auswerten
golisp2-client --eval "(+ 1 2 3)"

# Autocomplete
golisp2-client --complete "def"

# Datei laden
golisp2-client --load skript.lisp

# Interaktiver REPL
golisp2-client --repl
```

### REPL-Kommandos

| Kommando | Beschreibung |
|----------|-------------|
| `:quit`, `:q` | REPL beenden |
| `:complete pre` | Autocomplete für Prefix |
| `:load datei` | Datei laden |

### Protokoll (S-Expression-RPC)

**Request:** `(:id 1 :method "eval" :params ("(+ 1 2)"))`
**Response:** `(:id 1 :status "ok" :result "3")`

Alle Clients teilen sich dasselbe Environment (Zustand bleibt erhalten).

---

## Das selbsterweiternde Muster

```lisp
; KI schreibt Lisp-Code → GoLisp2 führt ihn aus
(eval (read (sigo
  "Schreibe nur den Lisp-Code, keine Erklärungen: defun fib"
  "claude-h")))
(fib 10)

; Ensemble: 3 KIs parallel, Ergebnisse als Liste
(parfunc antworten
  (sigo "Problem X lösen" "claude-h")
  (sigo "Problem X lösen" "gemini-p")
  (sigo "Problem X lösen" "gpt41"))
```

---

*GoLisp2 – Gerhard Quell & Claude Sonnet 4.6 – Februar 2026*
