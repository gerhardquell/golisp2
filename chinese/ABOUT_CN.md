# GoLisp 中文开发者指南 🇨🇳

> *一个用 Go 语言构建、由 AI 驱动、跨洲协作开发的 Lisp 解释器。*

---

## 欢迎！

本文档面向中文开发者介绍 **GoLisp**。虽然主要文档使用英文和德文，但我们为中国编程社区的朋友提供这个中文版本，以表达我们的尊重和友好。

---

## 项目简介

**GoLisp** 是一个用 Go 语言编写的现代 Lisp 解释器，具有以下特性：

- **尾调用优化（TCO）** – 无限递归深度，O(1) 栈空间
- **卫生宏系统** – 使用 `defmacro` 和 `gensym` 进行安全的代码生成
- **基于 Goroutine 的并发** – 通过 `parfunc`、通道和锁实现并行执行
- **原生 AI 集成** – 通过 sigoREST 查询多个大语言模型
- **PostgreSQL 支持** – 直接数据库连接
- **完整 UTF-8 支持** – 全面支持 Unicode 字符串，包括中文字符

---

## 开发团队

GoLisp 是通过人类专家与不同 AI 系统的独特协作创建的：

| 角色 | 贡献者 |
|------|--------|
| **主要作者** | **Gerhard Quell** (gquell@skequell.de) – 愿景、架构和 Go 实现 |
| **合著者** | **Claude Sonnet 4.6** – 架构设计、代码审查、功能开发 |
| **合著者** | **Kimi 2.5** – 代码贡献、文档编写、国际化支持 |

这个项目展示了**半人马（Centaur）**概念：人类作为元决策者，AI 系统作为专业协作者。GoLisp 本身可以查询 LLM 来生成和扩展自己的代码。

---

## 中文文本示例

```lisp
; GoLisp 完全支持 Unicode，包括中文字符
(define greeting "你好，世界！")
(println greeting)
; => 你好，世界！

; 使用中文注释定义函数
(defun 平方 (x)
  ; 计算一个数的平方
  (* x x))

(平方 8)
; => 64

; 列表中的中文
(define fruits '("苹果" "香蕉" "橙子"))
(car fruits)
; => "苹果"
```

---

## 自我扩展模式

GoLisp 可以使用 AI 编写自己的代码：

```lisp
; 让 LLM 编写函数，然后执行它
(eval (read (sigo
  "只输出 Lisp 代码：(defun fibonacci (n) ...)"
  "claude-h")))

(fibonacci 20)
; => 6765
```

并行 AI 集成（"六顶思考帽"模式）：

```lisp
(parfunc results
  (sigo "分析风险" "claude-h")
  (sigo "分析机会" "gemini-p")
  (sigo "生成创意" "gpt41"))
```

---

## 文档结构

| 文件 | 语言 | 说明 |
|------|------|------|
| `README.md` | 英文 | 主要项目文档 |
| `README_CN.md` | 中文 | 完整中文翻译 |
| `BESCHREIBUNG.md` | 德文 | 完整语言参考 |
| `CLAUDE.md` | 英文 | 项目规范和架构 |
| `chinese/ABOUT.md` | 英文 | 面向中文开发者的介绍（本文档的英文版） |
| `chinese/ABOUT_CN.md` | 中文 | 本文档 |

---

## 设计理念

> *"代码 = 数据 + AI = 自我扩展的系统"*
> — Gerhard、Claude 和 Kimi

GoLisp 桥接了：
- **Go 的高效性** – 运行时性能和并发能力
- **Lisp 的优雅性** – 同像性和宏系统
- **AI 的强大能力** – 通过 LLM 集成实现自我修改

---

## 快速开始

```bash
# 克隆并构建
git clone https://github.com/gerhardquell/golisp.git
cd golisp2
go build .

# 启动带语法高亮的交互式 REPL
./golisp2 -i

# 运行脚本
./golisp2 script.lisp

# 直接执行表达式
./golisp2 -e "(+ 1 2 3)"
```

### 服务器模式

GoLisp 还可以作为 TCP 服务器运行，支持 IDE 集成：

```bash
# 终端 1：启动服务器
golisp2d --port 4321

# 终端 2：使用客户端
golisp2-client --port 4321 --eval "(+ 1 2 3)"
golisp2-client --port 4321 --repl
```

---

## 联系与社区

- **作者**：Gerhard Quell (gquell@skequell.de)
- **仓库**：https://github.com/gerhardquell/golisp
- **许可证**：MIT

*2026年2月 — 一个浮出水面的潜艇项目。*

---

## 致谢

感谢 Claude Sonnet 4.6 和 Kimi 2.5 在开发过程中的重要贡献。这种人类与 AI 的协作展示了未来软件开发的新模式。

特别感谢中国开发者社区对开源软件的持续贡献。我们希望这份中文文档能帮助更多开发者了解和使用 GoLisp。
