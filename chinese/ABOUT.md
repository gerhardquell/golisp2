# GoLisp for Chinese Developers 🇨🇳

> *A Lisp interpreter built with Go, powered by AI, and co-authored across continents.*

---

## Welcome, 欢迎!

This document introduces **GoLisp** to Chinese developers. While the primary documentation is in English and German, we provide this Chinese version as a courtesy to our friends in the Chinese programming community.

---

## About the Project

**GoLisp** is a modern Lisp interpreter written in Go, featuring:

- **Tail-Call Optimization (TCO)** – Unlimited recursion depth
- **Hygienic Macros** – Safe code generation with `defmacro` and `gensym`
- **Goroutine-based Concurrency** – Parallel execution with `parfunc`, channels, and locks
- **Native AI Integration** – Query multiple LLMs via sigoREST
- **PostgreSQL Support** – Direct database connectivity
- **Full UTF-8 Support** – Unicode strings throughout, including Chinese characters

---

## The Development Team

GoLisp was created through a unique collaboration across different AI systems and human expertise:

| Role | Contributor |
|------|-------------|
| **Lead Author** | **Gerhard Quell** (gquell@skequell.de) – Vision, architecture, and Go implementation |
| **Co-Author** | **Claude Sonnet 4.6** – Architecture design, code review, feature development |
| **Co-Author** | **Kimi 2.5** – Code contributions, documentation, internationalization |

This project demonstrates the **Centaur** concept: humans as meta-deciders, AI systems as specialized collaborators. GoLisp itself can query LLMs to generate and extend its own code.

---

## Quick Example with Chinese Text

```lisp
; GoLisp fully supports Unicode, including Chinese characters
(define greeting "你好，世界！")
(println greeting)
; => 你好，世界！

; Define a function with Chinese comments
(defun 平方 (x)
  ; 计算一个数的平方
  (* x x))

(平方 8)
; => 64
```

---

## The Self-Extending Pattern

GoLisp can write its own code using AI:

```lisp
; Ask an LLM to write a function, then execute it
(eval (read (sigo
  "Write only Lisp code: (defun fibonacci (n) ...)"
  "claude-h")))

(fibonacci 20)
; => 6765
```

Parallel AI ensemble (the "Six Thinking Hats" pattern):

```lisp
(parfunc results
  (sigo "Analyze risks" "claude-h")
  (sigo "Analyze opportunities" "gemini-p")
  (sigo "Generate ideas" "gpt41"))
```

---

## Documentation Structure

| File | Language | Description |
|------|----------|-------------|
| `README.md` | Deutsch | Main project documentation (German) |
| `README_en.md` | English | English project description |
| `README_CN.md` | 中文 | Full Chinese translation |
| `BESCHREIBUNG.md` | Deutsch | Complete language reference |
| `CLAUDE.md` | Deutsch | Project conventions and architecture |
| `chinese/ABOUT.md` | English | This file – introduction for Chinese developers |
| `chinese/ABOUT_CN.md` | 中文 | 本文档的中文版本 |

---

## Philosophy

> *"Code = Data + AI = Self-Extending System"*
> — Gerhard, Claude & Kimi

GoLisp bridges:
- **Go's efficiency** – Runtime performance and concurrency
- **Lisp's elegance** – Homoiconicity and macros
- **AI's power** – Self-modification through LLM integration

---

## Getting Started

```bash
# Clone and build
git clone https://github.com/gerhardquell/golisp.git
cd golisp2
go build .

# Interactive REPL with syntax highlighting
./golisp2 -i

# Run a script
./golisp2 script.lisp

# Execute expression directly
./golisp2 -e "(+ 1 2 3)"
```

### Server Mode

GoLisp can also run as a TCP server for IDE integration:

```bash
# Terminal 1: Start server
golisp2 --swank 127.0.0.1:4321

# Terminal 2: Use client
golisp2-client --port 4321 --eval "(+ 1 2 3)"
golisp2-client --port 4321 --repl
```

---

## Contact & Community

- **Author**: Gerhard Quell (gquell@skequell.de)
- **Repository**: https://github.com/gerhardquell/golisp
- **License**: MIT

*February 2026 – A submarine project surfacing.*

---

## 致谢 (Acknowledgments)

感谢中国开发者社区对开源软件的贡献。我们希望通过提供中文文档，让更多开发者能够了解和使 GoLisp。

(Thank you to the Chinese developer community for your contributions to open source. We hope this Chinese documentation helps more developers discover and use GoLisp.)
