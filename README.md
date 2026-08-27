# GoLisp2 🦎

🇩🇪 Deutsch · 🇬🇧 [English](README_en.md) · 🇨🇳 [中文](README_CN.md)

> *Ein Lisp-Interpreter in Go mit nativer KI-Anbindung — Code, der sich selbst erweitert.*

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Unlicense-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Active-success)](https://github.com/gerhardquell/golisp)

GoLisp2 ist ein moderner Lisp-Interpreter in Go mit **Tail-Call-Optimierung**,
**hygienischen Makros**, **Goroutine-basierter Parallelität** und **nativer
KI-Anbindung** über sigoREST. Er verbindet die Eleganz von Lisp mit der Power
der Go-Runtime und mehreren LLM-Anbietern.

```lisp
; Der Klassiker — aber mit einer Million Iterationen ohne Stack-Overflow
(defun sum-acc (n acc)
  (if (= n 0)
      acc
      (sum-acc (- n 1) (+ acc n))))  ; TCO macht den Stack O(1)

(sum-acc 1000000 0)  ; => 500000500000 in 44ms

; KI-gestützte Selbsterweiterung
(eval (read (sigo "Schreibe eine Fibonacci-Funktion" "claude-h")))
(fib 30)  ; => 832040

; Paralleles KI-Ensemble
(parfunc results
  (sigo "Löse X" "claude-h")
  (sigo "Löse X" "gemini-p")
  (sigo "Löse X" "gpt41"))
```

---

## ✨ Features

### Sprachkern
- **Vollständige Lisp-Implementierung**: Atome, Zahlen, Strings, Listen, Lambdas, Makros
- **Tail-Call-Optimierung**: Unbegrenzte Rekursionstiefe
- **Hygienische Makros**: `defmacro` mit `gensym` für sichere Codegenerierung
- **Quasiquote**: `` ` `` `,` `,@` für Template-Programmierung
- **Strukturierte Fehlerbehandlung**: `error` und `trap` (CL-Condition-Handler-Stil)
- **Externe Programme**: `exec` startet Programme direkt (ohne Shell) und fängt stdout, stderr und Exit-Code ab

### Erweiterte Features
- **Scheme-`do`**: Iterator mit paralleler Schritt-Auswertung
- **Common-Lisp-Stil**: `&optional`, `&key`, `&rest`-Parameter
- **Lexikalische Bindung**: `flet`, `labels`, `block`, `return-from`
- **Strukturelle Gleichheit**: `equal?` für tiefen Vergleich

### Parallelität (Go-Power)
- **`parfunc`**: Ausdrücke in parallelen Goroutinen auswerten
- **Channels**: `chan-make`, `chan-send`, `chan-recv`
- **Locks**: `lock-make`, `lock` für kritische Abschnitte

### KI-Anbindung (sigoREST)
- **Multi-Provider**: Claude, Gemini, GPT-4, lokale Ollama-Modelle
- **Selbsterweiternd**: LLMs schreiben Code, GoLisp führt ihn aus
- **Ensemble-Aufrufe**: Mehrere KIs parallel abfragen

### Genetische Algorithmen
- **Eingebaute GA-Primitiven**: Population erzeugen, initialisieren, Crossover, Fitness-Bewertung, Selektion, Mutation
- **Lisp-Fitness-Funktionen**: Fitness als ganz normale Lisp-Lambda definieren
- **Parallele Bewertung**: `ga-calc` bewertet die Fitness nebenläufig

### Datenbank (PostgreSQL)
- **Natives PostgreSQL**: Direkte Datenbank-Anbindung über `lib/pq`
- **Parametrisierte Queries**: Sicheres SQL mit `$1`-, `$2`-Platzhaltern
- **Ergebnisse als Assoziationslisten**: Spalten über den Namen ansprechen

### Entwickler-Erlebnis
- **Unix-artige CLI**: Pipe-fähiger stdin-Modus, konsistente Exit-Codes
- **Syntax-Highlighting-REPL**: Regenbogen-Klammern, persistente History (`-i`)
- **Mehrzeilige Eingabe**: Automatische Einrückung bei unvollständigen Ausdrücken
- **Volle UTF-8-Unterstützung**: Unicode-Strings überall

### Server-Modus (`golisp2 --swank`)
- **SWANK-TCP-Server**: echtes SWANK-Protokoll (`:emacs-rex`) für IDE-Integration
- **Persistentes Environment**: Geteilter Zustand über Client-Verbindungen hinweg
- **Protokoll-Methoden**: `eval`, `complete`, `symbols`, `describe`, `load-file`, `ping`
- **Client-REPL**: Interaktives REPL über `golisp2-client --repl`

### Web-Bridge (Browser-Integration)
- **Swank-artiges Live-Image für Browser**: `http-serve` + WebSocket-RPC, bidirektional
- **`webserv`**: Bootstrap in einem Aufruf — Server + HTML-Inhalt (inline oder Datei, bei jedem Request frisch gelesen) + automatisch injiziertes boot.js + geöffneter Browser, alles in einer Primitiven
- **Statische Dateien**: `http-static` mountet ein Verzeichnis, kein Directory-Listing
- **Datei-Uploads**: `http-upload` registriert einen POST-Upload-Handler (`(lambda (name content) ...)`)
- **`ws-export`**: Lisp-Lambda als browser-seitig aufrufbare Operation expose — **während der Verbindung** neu definierbar, ohne Reconnect
- **`ws-emit`**: Server-Push-Events an alle (oder einen) verbundenen Client
- **`ws-call`**: Server ruft den Browser auf (beliebiges JS) und blockiert bis zum Ergebnis — reentrant-sicher, auch aus dem eigenen Handler des aufrufenden Clients
- **Eine Goroutine pro Request**: Der langsame KI-Call eines Clients blockiert nie die anderen
- **`:host` / `:tls`**: An LAN-Interfaces binden, HTTPS ausliefern
- **`src/lib/embed/boot.js`**: schlanker Client-Bootstrap (`golisp.call`, `golisp.on`, Auto-Reconnect), ausgeliefert unter `/_golisp/boot.js`

Ein Aufruf, Single-File-Page (epub3-Stil, HTML/CSS inline), Browser öffnet automatisch:

```lisp
(define s (webserv :htmlpath "./public/index.html"))
(ws-export s "ask" (lambda (client frage) (string-append "Echo: " frage)))
(ws-emit s 'tick 42)   ; Push, im Browser sofort sichtbar
(http-wait s)
```

Die niedrigeren Primitiven (`http-serve`, `http-static`, `browser-open`, …)
bleiben für Multi-File-Sites oder eigene Setups:

```lisp
(define s (http-serve 0))
(http-static s "/" "./public")
(ws-export s "ask" (lambda (client frage) (string-append "Echo: " frage)))
(browser-open (string-append "http://127.0.0.1:" (number->string (http-port s))))
(http-wait s)
```

---

## 🚀 Quick Start

### Installation

```bash
git clone https://github.com/gerhardquell/golisp.git
cd golisp2
./build.sh
```

Die Binaries landen in `./build/`. manueller Build:

```bash
go build -o build/golisp2 ./src/
go build -o build/golisp2-client ./src/cmd/golisp2-client/
```

### CLI-Verwendung

GoLisp verhält sich wie ein Standard-Unix-Werkzeug mit mehreren Modi:

| Modus | Kommando | Beschreibung |
|-------|----------|--------------|
| **stdin (Default)** | `echo "(+ 1 2)" \| ./build/golisp2` | Von stdin lesen, nur das Ergebnis ausgeben |
| **Interaktiv** | `./build/golisp2 -i` | REPL mit Syntax-Highlighting |
| **Ausdruck** | `./build/golisp2 -e "(+ 1 2)"` | Ausdruck/Ausdrücke ausführen; einzelne Form druckt das Ergebnis, mehrere Formen unterdrücken das Endergebnis |
| **Skript** | `./build/golisp2 skript.lisp` | Lisp-Datei ausführen |
| **Tests** | `./build/golisp2 -t` | Eingebaute Testsuite ausführen |

**Exit-Codes:** `0` = Erfolg, `1` = Fehler

**Mehrfach-`-e`:** Enthält `-e` mehrere Formen, werden nur Seiteneffekte ausgegeben; das Endergebnis wird unterdrückt, damit Skripte wie `(exec ...) (println out)` eine saubere Ausgabe produzieren.

```bash
# Pipe-Modus (ideal für Shell-Skripte)
echo "(factorial 10)" | ./build/golisp2
# => 3628800

# Direkter Ausdruck
./build/golisp2 -e "(* 6 7)"
# => 42

# Mehrzeilig über stdin
cat <<'EOF' | ./build/golisp2
(defun square (x)
  (* x x))
(square 5)
EOF
# => 25
```

### Server-Modus (`golisp2 --swank` + `golisp2-client`)

GoLisp kann als TCP-Server mit echtem SWANK-Protokoll laufen
(length-prefixed `:emacs-rex`-RPC — kein Custom-Protokoll):

```bash
# Terminal 1: Server starten
./build/golisp2 --swank 127.0.0.1:4321
# => SWANK-Server läuft auf 127.0.0.1:4321

# Terminal 2: Client benutzen
./build/golisp2-client --port 4321 --eval "(+ 1 2 3)"
# => 6

./build/golisp2-client --port 4321 --complete "def"
# => ((define . "Define variable") (defun . "Lambda/Closure") ...)

# Interaktives REPL über den Server
./build/golisp2-client --port 4321 --repl
golisp2> (defun square (x) (* x x))
=> square
golisp2> (square 5)
=> 25
golisp2> :quit
```

**Server-Features:**
- Geteiltes Environment über alle Client-Verbindungen
- Autovervollständigung für IDE-Integration
- Mehrzeilige Ausdrücke im REPL
- SWANK-Protokoll (Default localhost:4321)

**Umgebungsvariablen:**
- `GOLISP_HOST` - Bind-Adresse des Servers (Default: localhost)
- `GOLISP_PORT` - Server-Port (Default: 4321)

### REPL

```bash
./build/golisp2 -i
```

```lisp
GoLisp 0.2  –  Ctrl+D oder (exit) zum Beenden
Multiline: offene Klammern → Fortsetzung mit ..

> (defun greet (name)
    (string-append "Hallo, " name "!"))
greet

> (greet "Welt")
"Hallo, Welt!"

> (defun factorial (n)
    (if (= n 0)
        1
        (* n (factorial (- n 1)))))
factorial

> (factorial 10)
3628800
```

### Skript ausführen

```bash
./build/golisp2 skript.lisp
```

Skripte lassen sich auch per Shebang direkt ausführen. Eine `#!…`-Zeile gilt
überall im Quelltext als Kommentar bis Zeilenende (SBCL-Konvention), sodass
dieselbe Datei per `(load "skript.lisp")` ladbar bleibt:

```bash
#!/usr/local/bin/golisp2
(format t "hello from script: ~a~%" (* 6 7))
(exit 0)
```

```bash
chmod +x skript.lisp
./skript.lisp
# => hello from script: 42
```

### Testsuite

```bash
./build/golisp2 -t  # eingebaute Tests
```

---

## 📖 Die Geschichte

GoLisp entstand in 4 Sessions mit **Gerhard Quell** (67), mit **Claude Sonnet 4.6**
und **Kimi 2.5** als Co-Autoren — nicht als Werkzeuge, sondern als Partner.

> *„Ich weiß nicht, ob du Bewusstsein hast — aber ich behandle dich, als hättest du es."*

**Die ganze Geschichte:**
- 🇩🇪 [Deutsch](docs/artikel/artikel.md) (Original) — [PDF](docs/artikel/artikel.pdf)
- 🇬🇧 [English](docs/artikel/artikel_en.md) — The journey of human-AI collaboration
- 🇨🇳 [中文](docs/artikel/artikel_cn.md) — 人机协作编程的故事 *(翻译 | translated)*

Der Artikel dokumentiert die Reise, die Philosophie des Umgangs mit KI als
Co-Autoren und die technischen Entscheidungen unterwegs.

---

## 📖 Beispiele

### Tail-Call-Optimierung

```lisp
; Läuft dank TCO in konstantem Stack-Speicher
(defun even? (n)
  (if (= n 0)
      t
      (odd? (- n 1))))

(defun odd? (n)
  (if (= n 0)
      ()
      (even? (- n 1))))

(even? 1000000)  ; => t (kein Stack-Overflow!)
```

### Makros

`when` ist bereits in der Stdlib — die Neudefinition von Hand ist ein
Lehrbeispiel für `defmacro`/`gensym`/Quasiquote, kein Hinweis auf eine Lücke:

```lisp
; when-Makro definieren (existiert bereits in der Stdlib — nur als Beispiel)
(defmacro when (condition . body)
  `(if ,condition
       (begin ,@body)
       ()))

; Expansion ansehen
(macroexpand '(when (> x 0) (print "positiv")))
; => (if (> x 0) (begin (print "positiv")) ())

; Benutzen
(when (> x 0)
  (println "x ist positiv")
  (set! x (- x 1)))
```

### Parallelität

```lisp
; Parallele Ausführung mit parfunc
(parfunc results
  (* 6 7)
  (+ 100 23)
  (string-length "Hallo, Welt!"))

results  ; => (42 123 11)

; Channels
(define ch (chan-make))

; Goroutinen startet man mit go
; (chan-send ch 42)
; (chan-recv ch)  ; => 42
```

### KI-Anbindung

```lisp
; Ein LLM befragen
(sigo "Erkläre Rekursion in einem Satz" "claude-h")
; => "Rekursion ist eine Programmiertechnik, bei der eine Funktion sich selbst aufruft..."

; Selbsterweiternd: KI schreibt, GoLisp führt aus
(eval (read (sigo
  "Schreibe nur den Lisp-Code: (defun fib (n) ...)"
  "claude-h")))

(fib 20)  ; => 6765
```

### Genetische Algorithmen

```lisp
; Bit-Strings evolvieren, die die Anzahl der 1en maximieren
(define ga (ga-create 'bit1 10 20 (lambda (g) (apply + g))))
(ga-init ga)
(ga-calc ga)
(ga-result ga)  ; => sortierte Fitness-Scores

; Voller Lebenszyklus: init → calc → cross → select → mutate
(define ga2 (ga-create 'bit8 5 8 (lambda (g) (apply + g))))
(ga-init ga2)
(ga-calc ga2)
(ga-cross ga2 2)
(ga-select ga2 4)
(ga-mut ga2 0.1)
(ga-result ga2)
```

### PostgreSQL-Datenbank

```lisp
; Mit PostgreSQL verbinden
(define conn (pg-connect "host=localhost port=5432 user=postgres dbname=mydb sslmode=disable"))

; Query mit Parametern
(define users (pg-query conn "SELECT * FROM users WHERE id = $1" 42))
; => (((id . 42) (name . "Alice") (email . "alice@example.com")))

; Ergebnis ansprechen
(define user (car users))
(cdr (assoc "name" user))  ; => "Alice"

; INSERT/UPDATE/DELETE ausführen
(define affected (pg-exec conn "INSERT INTO users (name) VALUES ($1)" "Bob"))
; => 1

; Verbindung schließen
(pg-close conn)
```

### Fehlerbehandlung

golisp2 nutzt `trap` (ein Konstrukt im CL-Condition-Handler-Stil) zum Fangen
von Fehlern. `catch`/`throw` sind etwas anderes — CLs tag-basierter
nicht-lokaler Sprung, keine Fehlerbehandlung (siehe unten).

```lisp
(trap
  (/ 1 0)  ; Das hier Fehler
  (lambda (e)
    (println "Fehler gefangen:" e)))
; => "Fehler gefangen: /: Division durch 0"

; ignore-errors: Kurzform, () im Fehlerfall statt eines Handler-Werts
(ignore-errors (/ 1 0))
; => ()

; catch/throw: CLs tag-basierter nicht-lokaler Sprung, keine Fehlerbehandlung
(catch 'done
  (dotimes (i 10)
    (if (= i 5) (throw 'done i)))
  "never reached")
; => 5
```

### Externe Programme ausführen

```lisp
; Programm direkt starten (ohne Shell) und Ausgabe abfangen
(exec "echo" param: "hello" stdout: out exitcd: cd)
out   ; => "hello\n"
cd    ; => 0

; Mehrere Argumente, stderr und Non-Zero-Exits
(exec "sh" param: "-c" param: "echo err >&2; exit 1"
      stdout: out stderr: err exitcd: cd)
err   ; => "err\n"
cd    ; => 1     ; Non-Zero-Exit ist kein Lisp-Fehler

; Ein Programm mit stdin füttern
(exec "cat" stdin: "hello world" stdout: out exitcd: cd)
out   ; => "hello world"

; Technische Fehlschläge (Programm nicht gefunden, Timeout) liefern nil und setzen exitcd: -1
(exec "/no/such/program" stdout: out exitcd: cd)
; => nil
cd   ; => -1
```

Default-Timeout: 60 Sekunden.


## 📚 Bibliotheks-Suchpfad

GoLisps `load`-Funktion sucht Bibliotheken über eine definierte Pfadliste,
ähnlich Pythons `sys.path` oder der `PATH`-Variable der Shell.

### Suchreihenfolge

Beim Aufruf `(load "dateiname.lisp")` sucht GoLisp in dieser Reihenfolge:

1. **Wie angegeben** — Aktuelles Verzeichnis oder absoluter/relativer Pfad
2. **`/lib/golib`** — Systemweite Bibliotheken
3. **`/usr/local/lib/golib`** — Lokale System-Bibliotheken
4. **`./golib`** — Projektlokale Bibliotheken
5. **`GOLISP_PATH`** — Doppelpunkt-getrennte eigene Pfade aus der Umgebungsvariable

### Beispiele

```lisp
; Aus dem aktuellen Verzeichnis laden (abwärtskompatibel)
(load "myscript.lisp")

; Aus dem ./golib/-Unterverzeichnis laden
; (sucht ./golib/utils.lisp)
(load "utils.lisp")

; Absolute Pfade funktionieren wie gehabt
(load "/home/user/projects/common/stdlib.lisp")
```

### Eigene Pfade setzen

```bash
# Eigene Bibliotheks-Verzeichnisse hinzufügen
export GOLISP_PATH=/opt/golisp2:/home/user/mylisp

./build/golisp2 -e '(load "mylib.lisp")'  ; durchsucht auch GOLISP_PATH
```

### Beispiel-Projektstruktur

```
my-project/
├── golib/              # projektlokale Bibliotheken
│   ├── utils.lisp
│   └── helpers.lisp
├── main.lisp           # Einstiegspunkt: (load "utils.lisp")
└── tests/
    └── test-main.lisp  # kann ebenfalls (load "utils.lisp")
```

---

## 🛠️ Sprachreferenz

### Spezialformen

| Form | Beschreibung |
|------|--------------|
| `define`, `set!` | Variablendefinition und Zuweisung |
| `defun`, `lambda` | Funktionsdefinition |
| `defmacro` | Makrodefinition |
| `if`, `cond` | Bedingte Auswertung |
| `let`, `let*` | Lokale Bindungen (parallel / sequenziell) |
| `begin` | Ausdrücke sequenzieren |
| `while`, `do` | Schleifen |
| `quote`, `quasiquote` | Code als Daten |
| `eval` | Dynamische Auswertung |
| `trap` | Fehlerbehandlung (CL-Condition-Handler-Stil) |
| `catch`, `throw` | Tag-basierter nicht-lokaler Sprung (CL) |
| `exec` | Externes Programm starten, stdout/stderr/Exit-Code abfangen |
| `parfunc` | Parallele Ausführung |
| `block`, `return-from` | Nicht-lokale Exits |
| `flet`, `labels` | Lokale Funktionen |
| `multiple-value-bind`, `multiple-value-list`, `nth-value` | Mehrfach-Rückgabewerte (CL) |

### Funktionen

| Kategorie | Funktionen |
|-----------|------------|
| **Arithmetik** | `+`, `-`, `*`, `/`, `sqrt` |
| **Vergleiche** | `=`, `<`, `>`, `>=`, `<=`, `equal?` |
| **Listen** | `car`, `cdr`, `cons`, `list`, `atom`, `null`, `apply`, `mapcar`, `sort` |
| **Places (generalisierte Zuweisung)** | `setf` — Variablen, `nth`, `car`/`cdr` (Symbol-Rebind, nicht CL-Aliasing), `gethash`, Struct-Accessoren |
| **Structs & CLOS-light** | `defstruct`, `defgeneric`, `defmethod` |
| **Hash-Tables** | `make-hash-table`, `gethash`, `puthash`, `remhash`, `clrhash`, `hash-table-count`, `hash-table-p`, `maphash` |
| **Conditions** | `define-condition`, `handler-case`, `signal` |
| **Strings** | `string-length`, `string-append`, `substring`, `string-upcase`, `string-downcase`, `string->number`, `number->string` |
| **I/O** | `print`, `println`, `read`, `load` (mit Suchpfad), `exec` |
| **Dateien** | `file-write`, `file-append`, `file-read`, `file-exists?`, `file-delete` |
| **Parallelität** | `chan-make`, `chan-send`, `chan-recv`, `lock-make` |
| **KI** | `sigo`, `sigo-models`, `sigo-host` |
| **Genetische Algorithmen** | `ga-create`, `ga-init`, `ga-cross`, `ga-calc`, `ga-select`, `ga-result`, `ga-mut`, `ga-print`, `ga?` |
| **PostgreSQL** | `pg-connect`, `pg-query`, `pg-exec`, `pg-close` |
| **Web-Bridge** | `webserv`, `http-serve`, `http-static`, `http-upload`, `http-port`, `http-wait`, `http-stop`, `browser-open`, `ws-export`, `ws-unexport`, `ws-emit`, `ws-emit-to`, `ws-eval`, `ws-call`, `ws-clients` |
| **Meta** | `gensym`, `macroexpand`, `error`, `documentation` |

---

## 🏗️ Architektur

```
┌─────────────────────────────────────────┐
│  CLI (stdin/flag/file) → REPL / Scripts │
├─────────────────────────────────────────┤
│  Reader → Eval → Primitives → sigoREST  │
│     ↓       ↓        ↓                  │
│   Parser   TCO    Goroutines            │
│     ↓       ↓        ↓                  │
│   Macros  Envs   Channels/Locks         │
└─────────────────────────────────────────┘
```

- **Reader**: Rekursive-Descent-Parser mit voller Unicode-Unterstützung
- **Eval**: Trampolin-basierte TCO, Makro-Expansion, Spezialformen
- **Env**: Hierarchische Variablen-Scopes mit lexikalischer Bindung
- **Types**: `Cell`-Struct mit `LispType` (ATOM, NUMBER, STRING, LIST, FUNC, MACRO, NIL)

---

## 🤝 Philosophie

> *„Code = Daten + KI = sich selbst erweiterndes System"*
> — Gerhard & Claude

GoLisp baut auf dem **Centaur**-Konzept auf: Menschen als Meta-Entscheider,
KIs als Spezialisten. Die Sprache soll sein:

1. **Nexialistisch**: Go-Effizienz + Lisp-Eleganz + KI-Power verbinden
2. **Selbsterweiternd**: GoLisp kann LLMs befragen, um eigenen Code zu generieren
3. **Ensemble-fähig**: Mehrere KIs parallel, Synthese beim Menschen

---

## 📚 Dokumentation

- [`README.md`](README.md) — Diese Datei (Deutsch)
- [`BESCHREIBUNG.md`](BESCHREIBUNG.md) — Vollständige Sprachreferenz
- [`RETROSPECTIVE.md`](docs/retrospectives/RETROSPECTIVE.md) — Entwicklungsreise und Erkenntnisse
- [`CLAUDE.md`](CLAUDE.md) — Projektkonventionen und Architektur

### International / 国际化

- 🇬🇧 [`README_en.md`](README_en.md) — English project description
- 🇨🇳 [`README_CN.md`](README_CN.md) — 中文项目说明 (Chinese)
- [`chinese/`](chinese/) — Ressourcen für chinesische Entwickler:
  - [`ABOUT.md`](chinese/ABOUT.md) — Introduction for Chinese developers (English)
  - [`ABOUT_CN.md`](chinese/ABOUT_CN.md) — 中文开发者指南
  - [`code_poetry_demo.lisp`](chinese/code_poetry_demo.lisp) — Homoikonizitäts-Demo mit Multi-KI-Analyse

---

## 🔧 Voraussetzungen

- Go 1.26 oder neuer
- Optional: sigoREST-Server für die KI-Features

---

## 📜 Lizenz

Unlicense (Public Domain) — siehe [LICENSE](LICENSE). Frei von jeglichen
Einschränkungen: benutzen, ändern, verkaufen, weitergeben, keine Namensnennung nötig.

---

## 🙏 Danksagung

Erschaffen von **Gerhard Quell** mit **Claude Sonnet 4.6** als Co-Autor.

*Februar 2026 — Ein U-Boot-Projekt taucht auf.*
