# GoLisp 🦎

> *用 Go 语言实现的现代 Lisp 解释器，原生 AI 集成 — 能够自我扩展的代码。*

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Active-success)](https://github.com/gerhardquell/golisp)

GoLisp 是一个用 Go 语言实现的现代 Lisp 解释器，集成了原生 AI 功能。它结合了 Lisp 语言的优雅与 Go 语言运行时的强大性能，支持**尾调用优化（TCO）**、**卫生宏**、**基于 Goroutine 的并发编程**，以及通过 sigoREST 实现的多 LLM 提供商原生 AI 集成。

```lisp
; 经典示例 — 百万次迭代，无栈溢出
(defun sum-acc (n acc)
  (if (= n 0)
      acc
      (sum-acc (- n 1) (+ acc n))))  ; TCO 使栈空间为 O(1)

(sum-acc 1000000 0)  ; => 500000500000，仅需 44ms

; AI 驱动的自我扩展
(eval (read (sigo "编写斐波那契函数" "claude-h")))
(fib 30)  ; => 832040

; 并行 AI 集成调用
(parfunc results
  (sigo "解决问题 X" "claude-h")
  (sigo "解决问题 X" "gemini-p")
  (sigo "解决问题 X" "gpt41"))
```

---

## ✨ 核心特性

### 语言核心
- **完整 Lisp 实现**：支持原子、数字、字符串、列表、Lambda 和宏
- **尾调用优化**：无限递归深度，O(1) 栈空间
- **卫生宏系统**：`defmacro` 配合 `gensym` 实现安全的代码生成
- **准引用支持**：`` ` `` `,` `,@` 模板编程
- **结构化错误处理**：`error` 和 `catch` 机制

### 高级功能
- **Scheme 风格 `do`**：支持并行步骤求值的迭代器
- **Common Lisp 风格参数**：`&optional`、`&key`、`&rest` 参数支持
- **词法作用域**：`flet`、`labels`、`block`、`return-from`
- **结构相等性判断**：`equal?` 支持深度比较

### 并发编程（Go 原生支持）
- **`parfunc`**：在并行 Goroutine 中求值表达式
- **通道机制**：`chan-make`、`chan-send`、`chan-recv`
- **锁机制**：`lock-make`、`lock` 用于临界区保护

### AI 集成（sigoREST）
- **多提供商支持**：Claude、Gemini、GPT-4、本地 Ollama 模型
- **自我扩展能力**：LLM 生成代码，GoLisp 直接执行
- **集成调用**：并行查询多个 AI 模型

### 遗传算法
- **内置 GA 原语**：种群创建、初始化、交叉、适应度评估、选择、变异
- **Lisp 适应度函数**：使用普通 Lisp lambda 定义适应度函数
- **并行评估**：`ga-calc` 并发评估适应度

### 数据库支持（PostgreSQL）
- **原生 PostgreSQL 支持**：通过 `lib/pq` 直接连接数据库
- **参数化查询**：安全的 `$1`、`$2` 占位符
- **关联列表结果**：按名称访问列数据

### 开发者体验
- **Unix 风格命令行**：支持管道的标准输入模式，一致的退出码
- **语法高亮 REPL**：彩虹括号，持久化历史记录（`-i` 参数）
- **多行输入**：不完整表达式自动缩进
- **完整 UTF-8 支持**：全 Unicode 字符串支持

### 服务器模式 (golisp2d)
- **SWANK 风格 TCP 服务器**：S-表达式 RPC 协议，支持 IDE 集成
- **持久化环境**：客户端连接间共享状态
- **协议方法**：`eval`、`complete`、`symbols`、`describe`、`load-file`、`ping`
- **客户端 REPL**：通过 `golisp2-client --repl` 使用交互式 REPL

---

## 🚀 快速开始

### 安装方法

```bash
git clone https://github.com/gerhardquell/golisp.git
cd golisp2
go build .
```

### 命令行使用

GoLisp 作为标准 Unix 工具运行，支持多种模式：

| 模式 | 命令 | 说明 |
|------|---------|-------------|
| **标准输入（默认）** | `echo "(+ 1 2)" \| ./golisp2` | 从标准输入读取，仅输出结果 |
| **交互模式** | `./golisp2 -i` | 带语法高亮的 REPL 环境 |
| **表达式模式** | `./golisp2 -e "(+ 1 2)"` | 执行单个表达式 |
| **脚本模式** | `./golisp2 script.lisp` | 运行 Lisp 脚本文件 |
| **测试模式** | `./golisp2 -t` | 运行内置测试套件 |

**退出码：** `0` = 成功，`1` = 错误

```bash
# 管道模式（适合 shell 脚本）
echo "(factorial 10)" | ./golisp2
# => 3628800

# 直接执行表达式
./golisp2 -e "(* 6 7)"
# => 42

# 通过标准输入执行多行代码
cat <<'EOF' | ./golisp2
(defun square (x)
  (* x x))
(square 5)
EOF
# => 25
```

### 服务器模式 (`golisp2d` + `golisp2-client`)

GoLisp 可以作为 TCP 服务器运行，支持类似 SWANK 的 S-表达式 RPC 协议：

```bash
# 终端 1：启动服务器
golisp2d --port 4321
# => golisp2d 运行在 localhost:4321

# 终端 2：使用客户端
golisp2-client --port 4321 --eval "(+ 1 2 3)"
# => 6

golisp2-client --port 4321 --complete "def"
# => ((define . "Define variable") (defun . "Lambda/Closure") ...)

# 通过服务器使用交互式 REPL
golisp2-client --port 4321 --repl
golisp2> (defun square (x) (* x x))
=> square
golisp2> (square 5)
=> 25
golisp2> :quit
```

**服务器特性：**
- 所有客户端连接共享同一环境
- 支持 IDE 集成的自动补全
- REPL 支持多行表达式
- S-表达式 RPC 协议（默认 localhost:4321）

**环境变量：**
- `GOLISP_HOST` - 服务器绑定地址（默认：localhost）
- `GOLISP_PORT` - 服务器端口（默认：4321）

### REPL 交互环境

```bash
./golisp2 -i
```

```lisp
GoLisp 0.2  –  Ctrl+D 或 (exit) 退出
多行输入：未闭合括号 → 继续输入..

> (define (greet name)
    (string-append "你好, " name "!"))
greet

> (greet "世界")
"你好, 世界!"

> (defun factorial (n)
    (if (= n 0)
        1
        (* n (factorial (- n 1)))))
factorial

> (factorial 10)
3628800
```

### 运行脚本

```bash
./golisp2 script.lisp
```

脚本也可以通过 shebang 直接执行。`#!…` 行在源码任何位置都被视为行注释（SBCL 惯例），因此同一文件仍可通过 `(load "script.lisp")` 加载：

```bash
#!/usr/local/bin/golisp2
(format t "hello from script: ~a~%" (* 6 7))
(exit 0)
```

```bash
chmod +x script.lisp
./script.lisp
# => hello from script: 42
```

### 测试套件

```bash
./golisp2 -t  # 40 个内置测试
```

---

## 📖 代码示例

### 尾调用优化

```lisp
; 得益于 TCO，此代码在常量栈空间中运行
(defun even? (n)
  (if (= n 0)
      t
      (odd? (- n 1))))

(defun odd? (n)
  (if (= n 0)
      ()
      (even? (- n 1))))

(even? 1000000)  ; => t（无栈溢出！）
```

### 宏系统

```lisp
; 定义 when 宏
(defmacro when (condition . body)
  `(if ,condition
       (begin ,@body)
       ()))

; 展开查看生成的代码
(macroexpand '(when (> x 0) (print "正数")))
; => (if (> x 0) (begin (print "正数")) ())

; 使用宏
(when (> x 0)
  (println "x 是正数")
  (set! x (- x 1)))
```

### 并发编程

```lisp
; 使用 parfunc 并行执行
(parfunc results
  (* 6 7)
  (+ 100 23)
  (string-length "你好, 世界!"))

results  ; => (42 123 6)

; 通道通信
(define ch (chan-make))

; 在实际实现中，使用 go 启动 goroutine
; (chan-send ch 42)
; (chan-recv ch)  ; => 42
```

### AI 集成

```lisp
; 查询 LLM
(sigo "用一句话解释递归" "claude-h")
; => "递归是一种编程技术，函数调用自身来解决问题..."

; 自我扩展：AI 编写代码，GoLisp 执行
(eval (read (sigo
  "只输出 Lisp 代码：(defun fib (n) ...)"
  "claude-h")))

(fib 20)  ; => 6765
```

### 遗传算法

```lisp
; 进化最大化 1 的数量的位串
(define ga (ga-create 'bit1 10 20 (lambda (g) (apply + g))))
(ga-init ga)
(ga-calc ga)
(ga-result ga)  ; => 排序后的适应度分数

; 完整生命周期：init → calc → cross → select → mutate
(define ga2 (ga-create 'bit8 5 8 (lambda (g) (apply + g))))
(ga-init ga2)
(ga-calc ga2)
(ga-cross ga2 2)
(ga-select ga2 4)
(ga-mut ga2 0.1)
(ga-result ga2)
```

### PostgreSQL 数据库

```lisp
; 连接 PostgreSQL
(define conn (pg-connect "host=localhost port=5432 user=postgres dbname=mydb sslmode=disable"))

; 带参数的查询
(define users (pg-query conn "SELECT * FROM users WHERE id = $1" 42))
; => (((id . 42) (name . "Alice") (email . "alice@example.com")))

; 访问结果
(define user (car users))
(cdr (assoc "name" user))  ; => "Alice"

; 执行 INSERT/UPDATE/DELETE
(define affected (pg-exec conn "INSERT INTO users (name) VALUES ($1)" "Bob"))
; => 1

; 关闭连接
(pg-close conn)
```

### 错误处理

```lisp
(catch
  (/ 1 0)  ; 此处会出错
  (lambda (e)
    (println "捕获错误:" e)))
; => "捕获错误: /: 除数为 0"

; 未处理的 Go 错误会传播（不被捕获）
(catch
  (error "用户错误")
  (lambda (e)
    "已恢复"))
; => "已恢复"
```

---

## 📚 库搜索路径

GoLisp 的 `load` 函数通过定义的搜索路径列表查找库，类似于 Python 的 `sys.path` 或 shell 的 `PATH` 变量。

### 搜索顺序

当您调用 `(load "filename.lisp")` 时，GoLisp 按以下顺序搜索：

1. **按原样** — 当前目录或绝对/相对路径
2. **`/lib/golib`** — 系统范围的库
3. **`/usr/local/lib/golib`** — 本地系统库
4. **`./golib`** — 项目本地库
5. **`GOLISP_PATH`** — 环境变量中冒号分隔的自定义路径

### 示例

```lisp
; 从当前目录加载（向后兼容）
(load "myscript.lisp")

; 从 ./golib/ 子目录加载
; （搜索 ./golib/utils.lisp）
(load "utils.lisp")

; 绝对路径照常工作
(load "/home/user/projects/common/stdlib.lisp")
```

### 设置自定义路径

```bash
# 添加自定义库目录
export GOLISP_PATH=/opt/golisp2:/home/user/mylisp

./golisp2 -e '(load "mylib.lisp")'  ; 也搜索 GOLISP_PATH
```

### 项目结构示例

```
my-project/
├── golib/              # 项目本地库
│   ├── utils.lisp
│   └── helpers.lisp
├── main.lisp           # 入口点: (load "utils.lisp")
└── tests/
    └── test-main.lisp  ; 也可以 (load "utils.lisp")
```

---

## 🛠️ 语言参考

### 特殊形式

| 形式 | 说明 |
|------|------|
| `define`、`set!` | 变量定义和赋值 |
| `defun`、`lambda` | 函数定义 |
| `defmacro` | 宏定义 |
| `if`、`cond` | 条件求值 |
| `let` | 局部绑定 |
| `begin` | 顺序表达式 |
| `while`、`do` | 循环 |
| `quote`、`quasiquote` | 代码即数据 |
| `eval` | 动态求值 |
| `catch` | 错误处理 |
| `parfunc` | 并行执行 |
| `block`、`return-from` | 非局部退出 |
| `flet`、`labels` | 局部函数 |

### 内置函数

| 类别 | 函数 |
|------|------|
| **算术运算** | `+`、`-`、`*`、`/` |
| **比较运算** | `=`、`<`、`>`、`>=`、`<=`、`equal?` |
| **列表操作** | `car`、`cdr`、`cons`、`list`、`atom`、`null`、`apply`、`mapcar` |
| **字符串操作** | `string-length`、`string-append`、`substring`、`string-upcase`、`string-downcase`、`string->number`、`number->string` |
| **输入输出** | `print`、`println`、`read`、`load`（带搜索路径） |
| **文件操作** | `file-write`、`file-append`、`file-read`、`file-exists?`、`file-delete` |
| **并发** | `chan-make`、`chan-send`、`chan-recv`、`lock-make` |
| **AI** | `sigo`、`sigo-models`、`sigo-host` |
| **遗传算法** | `ga-create`、`ga-init`、`ga-cross`、`ga-calc`、`ga-select`、`ga-result`、`ga-mut`、`ga-print`、`ga?` |
| **PostgreSQL** | `pg-connect`、`pg-query`、`pg-exec`、`pg-close` |
| **元编程** | `gensym`、`macroexpand`、`error` |

---

## 🏗️ 架构设计

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

- **Reader**：递归下降解析器，完整 Unicode 支持
- **Eval**：基于跳板的 TCO、宏展开、特殊形式
- **Env**：词法绑定的层次化变量作用域
- **Types**：`Cell` 结构体，包含 `LispType`（ATOM、NUMBER、STRING、LIST、FUNC、MACRO、NIL）

---

## 🤝 设计理念

> *"代码 = 数据 + AI = 自我扩展系统"*
> — Gerhard & Claude

GoLisp 基于半人马概念构建：人类作为元决策者，AI 作为专家。该语言的设计理念是：

1. **连接主义**：桥接 Go 的高效性、Lisp 的优雅性和 AI 的强大能力
2. **自我扩展**：GoLisp 可以查询 LLM 生成自己的代码
3. **集成能力**：多个 AI 并行工作，由用户进行综合

---

## 📚 文档

- [`README.md`](README.md) — 英文项目说明
- [`BESCHREIBUNG.md`](BESCHREIBUNG.md) — 完整语言参考（德文）
- [`RETROSPECTIVE.md`](RETROSPECTIVE.md) — 开发历程与见解
- [`CLAUDE.md`](CLAUDE.md) — 项目规范与架构

---

## 🔧 系统要求

- Go 1.21 或更高版本
- 可选：sigoREST 服务器（用于 AI 功能）
- 可选：PostgreSQL（用于数据库功能）

---

## 📜 许可证

MIT 许可证 — 详见 [LICENSE](LICENSE) 文件。

---

## 🙏 致谢

由 **Gerhard Quell** 与 **Claude Sonnet 4.6** 共同创作。

*2026年2月 — 一个浮出水面的潜艇项目。*
