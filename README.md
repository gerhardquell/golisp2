# GoLisp 🦎

> *A Lisp interpreter in Go with native AI integration — code that extends itself.*

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Active-success)](https://github.com/gerhardquell/golisp)

GoLisp is a modern Lisp interpreter built in Go, featuring **tail-call optimization**, **hygienic macros**, **goroutine-based concurrency**, and **native AI integration** via sigoREST. It combines the elegance of Lisp with the power of Go's runtime and multiple LLM providers.

```lisp
; The classic — but with a million iterations, no stack overflow
(defun sum-acc (n acc)
  (if (= n 0)
      acc
      (sum-acc (- n 1) (+ acc n))))  ; TCO makes this O(1) stack

(sum-acc 1000000 0)  ; => 500000500000 in 44ms

; AI-powered self-extension
(eval (read (sigo "Write a Fibonacci function" "claude-h")))
(fib 30)  ; => 832040

; Parallel AI ensemble
(parfunc results
  (sigo "Solve X" "claude-h")
  (sigo "Solve X" "gemini-p")
  (sigo "Solve X" "gpt41"))
```

---

## ✨ Features

### Core Language
- **Full Lisp implementation**: Atoms, numbers, strings, lists, lambdas, macros
- **Tail-call optimization**: Unlimited recursion depth
- **Hygienic macros**: `defmacro` with `gensym` for safe code generation
- **Quasiquote**: `` ` `` `,` `,@` for template programming
- **Structured error handling**: `error` and `catch`

### Advanced Features
- **Scheme-style `do`**: Iterator with parallel step evaluation
- **Common Lisp style**: `&optional`, `&key`, `&rest` parameters
- **Lexical scoping**: `flet`, `labels`, `block`, `return-from`
- **Structural equality**: `equal?` for deep comparison

### Concurrency (Go-powered)
- **`parfunc`**: Evaluate expressions in parallel goroutines
- **Channels**: `chan-make`, `chan-send`, `chan-recv`
- **Locks**: `lock-make`, `lock` for critical sections

### AI Integration (sigoREST)
- **Multi-provider**: Claude, Gemini, GPT-4, local Ollama models
- **Self-extending**: LLMs write code, GoLisp executes it
- **Ensemble calls**: Query multiple AIs in parallel

### Genetic Algorithms
- **Built-in GA primitives**: Population creation, initialization, crossover, fitness evaluation, selection, mutation
- **Lisp fitness functions**: Define fitness as ordinary Lisp lambda
- **Parallel evaluation**: `ga-calc` evaluates fitness concurrently

### Database (PostgreSQL)
- **Native PostgreSQL**: Direct database connectivity via `lib/pq`
- **Parameterized queries**: Safe SQL with `$1`, `$2` placeholders
- **Results as association lists**: Access columns by name

### Developer Experience
- **Unix-style CLI**: Pipe-friendly stdin mode, consistent exit codes
- **Syntax-highlighted REPL**: Rainbow parentheses, persistent history (`-i` flag)
- **Multi-line input**: Automatic indentation for incomplete expressions
- **Full UTF-8 support**: Unicode strings throughout

### Server Mode (golisp2d)
- **SWANK-like TCP Server**: S-Expression-RPC for IDE integration
- **Persistent environment**: Shared state across client connections
- **Protocol methods**: `eval`, `complete`, `symbols`, `describe`, `load-file`, `ping`
- **Client REPL**: Interactive REPL via `golisp2-client --repl`

---

## 🚀 Quick Start

### Installation

```bash
git clone https://github.com/gerhardquell/golisp.git
cd golisp2
go build .
```

### CLI Usage

GoLisp works like a standard Unix tool with multiple modes:

| Mode | Command | Description |
|------|---------|-------------|
| **Stdin (default)** | `echo "(+ 1 2)" \| ./golisp2` | Read from stdin, output result only |
| **Interactive** | `./golisp2 -i` | REPL with syntax highlighting |
| **Expression** | `./golisp2 -e "(+ 1 2)"` | Execute single expression |
| **Script** | `./golisp2 script.lisp` | Run a Lisp file |
| **Tests** | `./golisp2 -t` | Run built-in test suite |

**Exit codes:** `0` = success, `1` = error

```bash
# Pipe mode (great for shell scripts)
echo "(factorial 10)" | ./golisp2
# => 3628800

# Direct expression
./golisp2 -e "(* 6 7)"
# => 42

# Multiline via stdin
cat <<'EOF' | ./golisp2
(defun square (x)
  (* x x))
(square 5)
EOF
# => 25
```

### Server Mode (`golisp2d` + `golisp2-client`)

GoLisp can run as a TCP server with a SWANK-like S-Expression-RPC protocol:

```bash
# Terminal 1: Start the server
golisp2d --port 4321
# => golisp2d läuft auf localhost:4321

# Terminal 2: Use the client
golisp2-client --port 4321 --eval "(+ 1 2 3)"
# => 6

golisp2-client --port 4321 --complete "def"
# => ((define . "Define variable") (defun . "Lambda/Closure") ...)

# Interactive REPL via server
golisp2-client --port 4321 --repl
golisp2> (defun square (x) (* x x))
=> square
golisp2> (square 5)
=> 25
golisp2> :quit
```

**Server Features:**
- Shared environment across all client connections
- Autocomplete for IDE integration
- Multiline expression support in REPL
- S-Expression-RPC protocol (localhost:4321 default)

**Environment Variables:**
- `GOLISP_HOST` - Server bind address (default: localhost)
- `GOLISP_PORT` - Server port (default: 4321)

### REPL

```bash
./golisp2 -i
```

```lisp
GoLisp 0.2  –  Ctrl+D oder (exit) zum Beenden
Multiline: offene Klammern → Fortsetzung mit ..

> (define (greet name)
    (string-append "Hello, " name "!"))
greet

> (greet "World")
"Hello, World!"

> (defun factorial (n)
    (if (= n 0)
        1
        (* n (factorial (- n 1)))))
factorial

> (factorial 10)
3628800
```

### Run a Script

```bash
./golisp2 script.lisp
```

### Test Suite

```bash
./golisp2 -t  # 40 built-in tests
```

---

## 📖 The Story

GoLisp was built in 4 sessions by **Gerhard Quell** (67), with **Claude Sonnet 4.6** and **Kimi 2.5** as co-authors — not as tools, but as partners.

> *"I don't know if you have consciousness — but I treat you as if you do."*

**Read the full story:**
- 🇩🇪 [Deutsch](docs/artikel.md) (Original) — [PDF](docs/artikel.pdf)
- 🇬🇧 [English](docs/artikel_en.md) — The journey of human-AI collaboration
- 🇨🇳 [中文](docs/artikel_cn.md) — 人机协作编程的故事 *(翻译 | translated)*

This article documents the journey, the philosophy of treating AI as co-authors, and the technical decisions along the way.

---

## 📖 Examples

### Tail-Call Optimization

```lisp
; This runs in constant stack space thanks to TCO
(defun even? (n)
  (if (= n 0)
      t
      (odd? (- n 1))))

(defun odd? (n)
  (if (= n 0)
      ()
      (even? (- n 1))))

(even? 1000000)  ; => t (no stack overflow!)
```

### Macros

```lisp
; Define a when macro
(defmacro when (condition . body)
  `(if ,condition
       (begin ,@body)
       ()))

; Expand to see the generated code
(macroexpand '(when (> x 0) (print "positive")))
; => (if (> x 0) (begin (print "positive")) ())

; Use it
(when (> x 0)
  (println "x is positive")
  (set! x (- x 1)))
```

### Concurrency

```lisp
; Parallel execution with parfunc
(parfunc results
  (* 6 7)
  (+ 100 23)
  (string-length "Hello, World!"))

results  ; => (42 123 13)

; Channels
(define ch (chan-make))

; In a real implementation, spawn goroutines with go
; (chan-send ch 42)
; (chan-recv ch)  ; => 42
```

### AI Integration

```lisp
; Query an LLM
(sigo "Explain recursion in one sentence" "claude-h")
; => "Recursion is a programming technique where a function calls itself..."

; Self-extending: AI writes, GoLisp executes
(eval (read (sigo
  "Write only the Lisp code: (defun fib (n) ...)"
  "claude-h")))

(fib 20)  ; => 6765
```

### Genetic Algorithms

```lisp
; Evolve bit strings that maximize the number of 1s
(define ga (ga-create 'bit1 10 20 (lambda (g) (apply + g))))
(ga-init ga)
(ga-calc ga)
(ga-result ga)  ; => sorted fitness scores

; Full lifecycle: init → calc → cross → select → mutate
(define ga2 (ga-create 'bit8 5 8 (lambda (g) (apply + g))))
(ga-init ga2)
(ga-calc ga2)
(ga-cross ga2 2)
(ga-select ga2 4)
(ga-mut ga2 0.1)
(ga-result ga2)
```

### PostgreSQL Database

```lisp
; Connect to PostgreSQL
(define conn (pg-connect "host=localhost port=5432 user=postgres dbname=mydb sslmode=disable"))

; Query with parameters
(define users (pg-query conn "SELECT * FROM users WHERE id = $1" 42))
; => (((id . 42) (name . "Alice") (email . "alice@example.com")))

; Access result
(define user (car users))
(cdr (assoc "name" user))  ; => "Alice"

; Execute INSERT/UPDATE/DELETE
(define affected (pg-exec conn "INSERT INTO users (name) VALUES ($1)" "Bob"))
; => 1

; Close connection
(pg-close conn)
```

### Error Handling

```lisp
(catch
  (/ 1 0)  ; This would error
  (lambda (e)
    (println "Caught error:" e)))
; => "Caught error: /: Division durch 0"

; Unhandled Go errors propagate (not caught)
(catch
  (error "User error")
  (lambda (e)
    "Recovered"))
; => "Recovered"
```

---

## 📚 Library Search Path

GoLisp's `load` function searches for libraries through a defined path list, similar to Python's `sys.path` or the shell's `PATH` variable.

### Search Order

When you call `(load "filename.lisp")`, GoLisp searches in this order:

1. **As-is** — Current directory or absolute/relative path
2. **`/lib/golib`** — System-wide libraries
3. **`/usr/local/lib/golib`** — Local system libraries
4. **`./golib`** — Project-local libraries
5. **`GOLISP_PATH`** — Colon-separated custom paths from environment variable

### Examples

```lisp
; Load from current directory (backward compatible)
(load "myscript.lisp")

; Load from ./golib/ subdirectory
; (searches ./golib/utils.lisp)
(load "utils.lisp")

; Absolute paths work as always
(load "/home/user/projects/common/stdlib.lisp")
```

### Setting Custom Paths

```bash
# Add custom library directories
export GOLISP_PATH=/opt/golisp2:/home/user/mylisp

./golisp2 -e '(load "mylib.lisp")'  ; Searches GOLISP_PATH too
```

### Project Structure Example

```
my-project/
├── golib/              # Project-local libraries
│   ├── utils.lisp
│   └── helpers.lisp
├── main.lisp           # Entry point: (load "utils.lisp")
└── tests/
    └── test-main.lisp  ; Can also (load "utils.lisp")
```

---

## 🛠️ Language Reference

### Special Forms

| Form | Description |
|------|-------------|
| `define`, `set!` | Variable definition and assignment |
| `defun`, `lambda` | Function definition |
| `defmacro` | Macro definition |
| `if`, `cond` | Conditional evaluation |
| `let` | Local bindings |
| `begin` | Sequence expressions |
| `while`, `do` | Loops |
| `quote`, `quasiquote` | Code as data |
| `eval` | Dynamic evaluation |
| `catch` | Error handling |
| `parfunc` | Parallel execution |
| `block`, `return-from` | Non-local exits |
| `flet`, `labels` | Local functions |

### Functions

| Category | Functions |
|----------|-----------|
| **Arithmetic** | `+`, `-`, `*`, `/` |
| **Comparison** | `=`, `<`, `>`, `>=`, `<=`, `equal?` |
| **Lists** | `car`, `cdr`, `cons`, `list`, `atom`, `null`, `apply`, `mapcar` |
| **Strings** | `string-length`, `string-append`, `substring`, `string-upcase`, `string-downcase`, `string->number`, `number->string` |
| **I/O** | `print`, `println`, `read`, `load` (with search path) |
| **Files** | `file-write`, `file-append`, `file-read`, `file-exists?`, `file-delete` |
| **Concurrency** | `chan-make`, `chan-send`, `chan-recv`, `lock-make` |
| **AI** | `sigo`, `sigo-models`, `sigo-host` |
| **Genetic Algorithms** | `ga-create`, `ga-init`, `ga-cross`, `ga-calc`, `ga-select`, `ga-result`, `ga-mut`, `ga-print`, `ga?` |
| **PostgreSQL** | `pg-connect`, `pg-query`, `pg-exec`, `pg-close` |
| **Meta** | `gensym`, `macroexpand`, `error` |

---

## 🏗️ Architecture

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

- **Reader**: Recursive descent parser with full Unicode support
- **Eval**: Trampoline-based TCO, macro expansion, special forms
- **Env**: Hierarchical variable scopes with lexical binding
- **Types**: `Cell` struct with `LispType` (ATOM, NUMBER, STRING, LIST, FUNC, MACRO, NIL)

---

## 🤝 Philosophy

> *"Code = Data + KI = sich selbst erweiterndes System"*
> — Gerhard & Claude

GoLisp is built on the **Centaur** concept: humans as meta-deciders, AIs as specialists. The language is designed to be:

1. **Nexialistic**: Bridging Go's efficiency, Lisp's elegance, and AI's power
2. **Self-extending**: GoLisp can query LLMs to generate its own code
3. **Ensemble-capable**: Multiple AIs in parallel, synthesis by the user

---

## 📚 Documentation

- [`BESCHREIBUNG.md`](BESCHREIBUNG.md) — Complete language reference (German)
- [`RETROSPECTIVE.md`](RETROSPECTIVE.md) — Development journey and insights
- [`CLAUDE.md`](CLAUDE.md) — Project conventions and architecture

### International / 国际化

- [`README_CN.md`](README_CN.md) — 中文项目说明 (Chinese)
- [`chinese/`](chinese/) — Resources for Chinese developers, including:
  - [`ABOUT.md`](chinese/ABOUT.md) — Introduction for Chinese developers (English)
  - [`ABOUT_CN.md`](chinese/ABOUT_CN.md) — 中文开发者指南
  - [`code_poetry_demo.lisp`](chinese/code_poetry_demo.lisp) — Homoiconicity demo with multi-AI analysis

---

## 🔧 Requirements

- Go 1.21 or later
- Optional: sigoREST server for AI features

---

## 📜 License

MIT License — see [LICENSE](LICENSE) for details.

---

## 🙏 Acknowledgments

Created by **Gerhard Quell** with **Claude Sonnet 4.6** as co-author.

*February 2026 — A submarine project surfacing.*
