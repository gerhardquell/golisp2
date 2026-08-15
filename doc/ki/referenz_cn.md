# GoLisp2 — AI 快速参考（token 优化）

> **目标：** 让其他 AI 将此文件作为*初始上下文*，无需对 50 个文件运行 `rg`
> 即可编写/理解 GoLisp2 代码。
> **格式：** 表格、前缀、无废话。人类版本（德语）：
> `doc/golisp2-cheatsheet.md`。
> **来源：** `eval_core.go`、`lib/primitives.go`、`embed/stdlib.lisp`（截至 20260730）。

---

## 1. 求值顺序（务必检查）

```
1. ATOM   → 环境查找（nil = 单例 nil）
2. LIST   → 检查 car:
   a. 特殊形式（eval_core.go 中的 case）  → 直接执行，不求值参数
   b. MACRO                              → 展开后重新求值
   c. FUNC/LAMBDA                        → 求值参数，apply
```

尾调用（`if`、`begin`、`let`、`lambda`、`case`、`cond`、`prog1/2`、`catch`、
`throw`、`tagbody/go` 等）设置 `expr`/`env` 并在求值循环中 `continue` —
**不产生新栈帧**。深度递归 O(1) 栈空间。

---

## 2. 特殊形式（55 个）+ 2 个标准库宏（`dotimes`、`dolist`）

| 形式 | 语义 | 备注 |
|------|------|------|
| `(quote x)` | `x` 不求值 | `'x` 读取器语法糖 |
| `(if c t [e])` | 条件 | 支持尾调用 |
| `(begin . body)` | 序列 | 尾调用；多 body 经 `wrapBegin` |
| `(let (bind) . body)` | 并行绑定 | `let*` 顺序绑定（标准库） |
| `(lambda (p) . body)` | 闭包 | `&optional`、`&key`、`&rest` |
| `(defun f (p) . body)` | 全局函数 | 多 body 经 `wrapBegin` |
| `(defmacro m (p) . body)` | 全局宏 | |
| `(define sym val)` | 变量定义 | 全局或局部 |
| `(set! sym val)` | 更新 | `setq` 别名 |
| `(setq* s1 v1 s2 v2 ...)` | 顺序赋值 | |
| `(psetq s1 v1 s2 v2 ...)` | 并行赋值 | |
| `(macrolet ((m . spec)) . body)` | 局部宏 | 非递归 |
| `(symbol-macrolet ((s expansion) . body)` | 符号宏 | |
| `(flet ((f . spec)) . body)` | 局部函数 | 非递归 |
| `(labels ((f . spec)) . body)` | 局部函数 | 递归（互相） |
| `(block name . body)` | 命名块 | 词法作用域 |
| `(return-from name [val])` | 非局部退出 | |
| `(return [val])` | `(return-from nil [val])` | |
| `(tagbody stmt ...)` | 跳转标签 | 原子 = 标签 |
| `(go tag)` | 跳转 | 词法，不求值 |
| `(catch tag . body)` | 动态捕获 | 标签会被求值 |
| `(throw tag val)` | 动态抛出 | |
| `(trap expr handler)` | 简单捕获 | `(trap expr (lambda (e) ...))`，e = 消息字符串 |
| `(unwind-protect protected cleanup)` | 清理必定执行 | |
| `(eval form)` | 全局求值 | 始终 `Env.Root()` |
| `(load "file")` | 加载文件 | **注意：** 在 `defun` 内 → 局部绑定！ |
| `(progn . body)` | 序列 | 尾调用 |
| `(prog1 first . rest)` | 返回第一个值 | |
| `(prog2 a b . rest)` | 返回 b | |
| `(cond (test result) ...)` | 条件 | `else`/`t` = 默认 |
| `(case key (vals result) ...)` | 结构分派 | `equal?` 比较 |
| `(the type form)` | 声明 | 类型**被忽略**（无类型系统） |
| `(declare . decls)` | 声明 | 无操作 |
| `(eval-when (situation) . body)` | 情境控制 | |
| `(progv syms vals . body)` | 动态绑定 | **注意：** 无词法/动态分离 |
| `(and . exprs)` | 短路 | |
| `(or . exprs)` | 短路 | |
| `(not x)` | 取反 | |
| `(parfunc expr . opts)` | 并行求值 | `:timeout N`、`:workers N` |
| `(while test . body)` | 循环 | |
| `(do ((var step) ...) (test result) . body)` | Scheme 迭代 | 并行步进 |
| `(do* ((var step) ...) (test result) . body)` | Scheme 迭代 | 顺序步进 |
| `(dotimes (var n) . body)` | 计数循环 | 标准库宏 |
| `(dolist (var lst) . body)` | 列表迭代 | 标准库宏 |
| `(multiple-value-list form)` | 值→列表 | |
| `(multiple-value-bind (vars) form . body)` | 绑定多个值 | |
| `(multiple-value-call fn . forms)` | 传递多个值 | |
| `(multiple-value-prog1 form . rest)` | 保留第一个值 | |
| `(multiple-value-setq vars form)` | 赋值多个值 | |
| `(nth-value n form)` | 第 n 个值 | |
| `(function fn)` | 函数字面量 | `#'` 读取器语法糖 |
| `(macroexpand form)` | 展开宏 | |
| `(macroexpand-all form)` | 完全展开 | |
| `(bound? sym)` | 已绑定？ | |
| `(makunbound sym)` | 移除绑定 | |
| `(exec shell-cmd)` | Shell 命令 | |
| `(quasiquote x)` | 准引用 | `` `x ``、`,x`=unquote、`,@x`=splice |

**求值关键字（尾位置）：** `if`、`begin`、`let`、`cond`、`case`、
`prog1`、`prog2`、`catch`、`throw`、`tagbody/go`、`block/return-from`、`do`。

---

## 3. 重要原语（FUNC，Go 侧）

### 算术/比较
`+ - * / mod remainder abs floor random values`
`= < > >= <= equal? eq eq?`  — `eq`=指针，`equal?`=结构

### 列表（经典 7 个）
`car cdr cons atom null list append`
`atom? null? string? number? list? symbol?` — 类型谓词
`mapcar` — 原语（first-class：可用 `funcall`/`apply`）

### 符号/原子
`gensym intern symbol-name symbol->string`

### 输出
`print println read warn`

### 控制/错误
`error apply funcall exit`  — `error` **只返回字符串**，无条件对象
`exit` — 立即终止进程，参数为数字（无清理！）

### 环境/内省
`memstats sleep`

### 领域（各自的 Register-Xxx）
- **sigoREST：** `sigo sigo-models sigo-host`
- **Goroutine：** `chan-make chan-send chan-recv lock-make`
- **共享内存：** `shm-alloc shm-free shm-write shm-read shm-status shm-cleanup`
- **文件 I/O：** `file-write file-append file-read file-exists? file-delete set-working-directory get-working-directory get-file-path gets slurp err-write printf sprintf fprintf sscanf argv getenv environ`
- **Shell：** `system file-stat shell-assoc`
- **字符串：** `string-length string-append substring string-upcase string-downcase string->number number->string string->list list->string string-replace string-trim string-contains`
- **哈希表：** `make-hash-table gethash puthash remhash clrhash hash-table-count hash-table-p maphash`
- **FORMAT：** `format` — CL HyperSpec 22.3，`~A ~S ~D ~B ~O ~X ~R ~P ~C ~F ~E ~G ~$ ~% ~& ~| ~T ~* ~? ~[ ~{ ~( ~; ~^ ~/fun/ ~~`
  - 舍入：half-to-even（Go `strconv`），而非 C 的 half-up — `2.25` 的 `%.2f` → `"2.2"`
- **PostgreSQL：** `pg-connect pg-query pg-exec pg-close`
- **遗传算法：** `ga-create ga-init ga-cross ga-calc ga-select ga-result ga-mut ga-print ga?`
- **重定义：** `redefine-policy redef-log redef-log-clear defined-in`
- **跟踪：** `trace untrace trace?`

---

## 4. 标准库（embed/stdlib.lisp，约 50 个定义）

### 访问器/快捷方式
`cadr caddr cadddr cddr cdar caar first second third fourth rest`
`zero? positive? negative? pair?`

### 高阶/函数式
`identity constantly complement compose reverse length nth last member assoc filter drop take reduce for-each any every flatten zip list-tail iota max min square expt gcd`

### 列表辅助
`alist-set alist-get union set-difference find-all set-nth`

### 宏
`when unless let* dotimes dolist push pop defvar setf defstruct defgeneric defmethod`

### 迭代器
`dotimes (var n) body` — `(dotimes (i 10) ...)`
`dolist (var lst) body` — `(dolist (x xs) ...)`

### 结构
`(defstruct name (slot default) ...)` — 生成：`make-name`、`name-slot`、`name?`
`(setf place val)` — 通用；`(defstruct ...)` 自动注册访问器

---

## 5. 真值 / Nil

- `()` / `nil` / `NIL` → 单例 nil（指针相同！）
- `t` → 真，但*不是*唯一的真值 — 除 nil 外皆为真
- `(eq '() '())` → `t`（单例）
- `(eq 5 5)` → `()` — **设计使然：** `eq` 对数字永远返回 `()`（见 10.6）。不确定时用 `equal?`

---

## 6. 准引用模式

```lisp
`(a b c)           ; 纯引用
`(a ,x c)          ; unquote
`(a ,@xs c)        ; unquote-splice
```

---

## 7. 错误处理

```lisp
; 抛出错误
(error "消息")                 ; 中止，只返回字符串

; 捕获
(trap expr (lambda (e) ...))  ; e = "msg"（消息字符串）

; 动态
(catch 'tag body ...)
(throw 'tag value)
```

**Condition-lite**（`embed/condition.lisp`，自动加载）—
带类型层次结构和槽的结构化错误：

```lisp
(define-condition file-error (io-error) (path))  ; 类型 + 父类 + 槽
(signal 'file-error :path "x.lisp")              ; 抛出，总是 unwind！
(handler-case (load "x.lisp")
  (file-error (e) (file-error-path e))  ; 读取器自动生成：类型-槽
  (io-error  (e) "某个 io 错误")         ; 也匹配子类型
  (error     ()  "无变量绑定"))          ; 变量可为 ()
```

- 基础层次：`condition` → `error` → `lisp-error`
- **Go 错误**（file-read 等）在 `handler-case` 中变为 `lisp-error`，
  消息通过 `(lisp-error-msg e)` 获取
- 无匹配 → **重新抛出**给外层处理器
- 重定义类型会静默替换（重载语义）
- **CL 差异：** `signal` 总是 unwind（行为类似 CL 的 `error`，
  而非 CL 的 `signal`）。无 restart，无 `handler-bind`。
- 槽名在继承层次中必须唯一（扁平 plist，无遮蔽）。

**测试框架：** `tests/test-framework.lisp` — `defsuite`、`deftest`
（`:suite`、`:expected-failure`）、`is`、`run-tests` → 失败数。
典型用法：`(exit (run-tests))` → 退出码 = 失败数。

---

## 8. 求值环境

- `(eval form)` → **始终 `Env.Root()`**，从不在 lambda 作用域
- `load` → **例外：** 在 `defun` 体内 → 局部绑定，非全局
  - 变通方法：`(eval '(load "file"))` 实现全局加载
- `redefine-policy` → `'allow` / `'warn` / `'error`（默认 `warn`）

---

## 9. 陷阱 / 与 CL 的差异

| 情况 | GoLisp2 | CL |
|------|---------|-----|
| `(eq 5 5)` | `()` — `eq` 对数字永远 `()`（设计） | 通常 `t`（小整数） |
| `defun` 中的 `load` | 局部绑定 | 全局 |
| `progv` | 无词法/动态分离 | 动态 |
| `declare` | 无操作 | 类型检查 |
| `the` | 类型被忽略 | 类型检查 |
| `(eval form)` | 全局 | 全局（一致） |
| `macrolet` | 非递归 | 递归 |

---

## 10. 弱点（有意为之，独立章节）

以下限制是**设计决策**，不是 bug。AI 必须知晓，避免猜测：

### 10.1 无包系统
- 所有符号在一个全局命名空间。
- 无 `defpackage`、`in-package`、`export`、`import`。
- 只能通过命名约定避免冲突（如 `ga-`、`shm-` 前缀）。

### 10.2 无 CLOS
- `defstruct`（构造函数、访问器、谓词）。
- CLOS-light：`defgeneric`/`defmethod` — 对结构体标签的单分派，`t` = 默认方法。
- 无类、无继承、无 `call-next-method`、无方法组合。
- `defclass`、`call-method` — **不存在**。

### 10.3 仅有 Condition-lite，无完整 CL 条件系统
- 有 `define-condition`/`signal`/`handler-case`（层次结构、槽、
  继承分发、Go 错误的 `lisp-error` 回退）。
- **但：** `signal` 总是 unwind（类似 CL 的 `error`），无 restart，
  无 `handler-bind`，无 `restart-case`，无 MOP。
- 槽名扁平 — 继承中无遮蔽。

### 10.4 progv 无词法/动态分离
- `progv` 像 `let` 一样绑定 — 词法遮蔽会看到 progv 的值。
- CL：词法绑定可抵御 progv。

### 10.5 无 Compile-File
- 纯解释器。无 `compile-file`，不能加载 FASL。
- 无面向编译器/编译期的 `eval-when` 设置。

### 10.6 `eq` 对数字永远返回 `()`
- `(eq 5 5)` → `()`。`(eq 1000 1000)` → `()`。即使值相同。
- 内部存在小整数缓存（-32768..32767，`lib/types.go` 的 `MakeNum`）
  用于避免分配 — 但 `eq` 有意将数字视为永不相同
  （`lib/primitives.go` 的 `fnEqPtr`）。
- 数字比较永远用 `equal?` 或 `=`，绝不用 `eq`。

### 10.7 宏非递归（macrolet）
- `macrolet` 体看不到同层的其他宏。
- 函数的 `labels` 是递归的 — 与 CL 不对称。

### 10.8 无续延，无 MOP
- 无 `call/cc`。
- `catch`/`throw` 存在（第 2 节），但无 restart 语义。
- 无 CLOS 元对象协议。

### 10.9 `defun` 中的 `load` 局部绑定
- 变通方法：`(eval '(load "file"))` 实现全局加载。
- 原因：`eval` 在 `Env.Root()` 中运行。

### 10.10 无 GC 微调
- `memstats` 返回 Go 运行时统计。
- 无 `tweak`，`make-hash-table` 无弱引用。

### 10.11 无类型系统
- `declare` 和 `the` 是无操作。
- 无 `check-type`、`typecase`、`ctypecase`、`etypecase`。

### 10.12 无 LOOP、无 Series、无 Iterate
- 迭代只能用 `dolist`、`dotimes`、`do`、`do*`、`mapcar`、`reduce`。
- 无 `loop` 宏、无 `series`、无 `iterate`。

---

## 11. 文件结构（AI 扫描）

```
lib/
  types*.go          Cell 数据结构
  reader.go          Read / ReadAll
  env.go             环境（RWMutex、Pool）
  eval_core.go       求值蹦床 + 特殊形式分派
  eval_*.go          特殊形式、lambda、控制、准引用
  primitives.go      BaseEnv() — 所有 Go 原语
  *.go               goroutine、fileio、shellcmd、postgres、genalg、shm、sigorest
  stdlib.go          //go:embed stdlib.lisp
  format*.go         CL HyperSpec 22.3
  trace.go           trace/untrace
main.go              CLI
```

---

## 12. 快速查找（AI 速查表）

| 需要… | 使用 |
|--------|------|
| 算术 | `+ - * / mod` |
| 比较 | `equal?`（数字！）、`eq`（指针） |
| 构建列表 | `cons list append` |
| 迭代 | `dolist dotimes mapcar reduce` |
| 并行 | `parfunc` |
| 错误 | `error` + `trap` |
| 动态 | `catch/throw` |
| 结构 | `defstruct` |
| 字符串 | `string-append substring string-replace` |
| 文件 | `file-read file-write` |
| 数据库 | `pg-connect pg-query` |
| AI | `sigo` |
| 调试 | `trace` |

---

**AI 参考结束。** 人类版本（德语）：`doc/golisp2-cheatsheet.md`。
德语版本：`doc/ki/referenz.md`。英语版本：`doc/ki/referenz_en.md`。
