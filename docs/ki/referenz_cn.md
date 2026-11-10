# GoLisp2 — AI 速查参考（令牌优化）

> **目的：** 供编写或阅读 GoLisp2 代码的 AI 用作初始上下文。
> **译文**，原文为 `src/embed/ki-referenz.md`（德语）——原文是唯一的权威来源，
> 并通过 `go:embed` 作为每次 `(sigo …)` 调用的默认前置提示发送。有疑问时以德语原文为准。
> **错误信息：** 解释器的错误信息是德语；本文逐字引用，并在括号中附中文说明。
> **格式：** 表格、前缀、不写废话。手工维护，非自动生成；
> 完整清单：`docs/referenz-generiert.md`。

---

## 1. 求值顺序（务必核对）

```
1. ATOM   → 环境查找（nil = 单例 nil）
2. LIST   → 检查 car：
   a. 特殊形式（eval_core.go 中的 case） → 直接处理，不求值参数
   b. MACRO                              → 展开后重新求值
   c. FUNC/LAMBDA                        → 求值参数，apply
```

尾调用（`if`、`begin`/`progn`/`locally`、`let`、`let*`、`cond`、`case`）
设置 `expr`/`env` 后在求值循环中 `continue` —— **不创建新栈帧**。
深度递归只占 O(1) 栈空间。

---

## 2. 特殊形式（60 个 case 分支，62 个关键字）+ 2 个 stdlib 宏（`dotimes`、`dolist`）

直接从 `eval_core.go` 中的 `switch expr.Car.Val` 统计（截至
20261001）—— `begin`/`progn`/`locally` 共用一个 case 分支（3 个名字，
1 个实现），因此是 60 个分支、62 个名字。

| 形式 | 语义 | 备注 |
|------|------|------|
| `(quote x)` | `x` 不求值 | `'x` 读取器语法糖 |
| `(if c t [e])` | 条件 | 支持尾调用 |
| `(begin . body)` | 顺序执行 | 尾调用；别名：`progn`、`locally`；多表达式体通过 `wrapBegin` |
| `(let (bind) . body)` | 并行绑定 | 支持尾调用 |
| `(let* (bind) . body)` | 顺序绑定 | 支持尾调用；**原生特殊形式**（不在 stdlib 中——每个绑定都能看到前面的绑定） |
| `(lambda (p) . body)` | 闭包 | `&optional`、`&key`、`&rest` |
| `(defun f (p) . body)` | 全局函数 | 多表达式体通过 `wrapBegin` |
| `(defmacro m (p) . body)` | 全局宏 | |
| `(defmain (args) . body)` | 脚本入口 | 仅当文件为主程序时（shebang/`golisp2 f.lisp`）；加载后运行；返回值 = 退出码 0–255；否则为 `nil`。`args` 不含程序名/文件名；只在主文件顶层/特殊形式中生效，经由 `funcall`/`apply`/`mapcar` 不生效 |
| `(define sym val)` | 变量定义 | 全局或局部；`(define (f p) body)` = `(define f (lambda (p) body))` 的简写 |
| `(set! sym val)` | 更新一对 | 仅 `env.Update` —— **未绑定时报错**，不新建绑定 |
| `(setq v1 val1 v2 val2 ...)` | 顺序赋值（CL） | 可多对；未绑定 → **全局**创建 + 向 stderr 输出 `WARN`（同 SBCL；也适用于 `setq*`、`setf`、`multiple-value-setq`） |
| `(setq* v1 val1 v2 val2 ...)` | 顺序赋值，未绑定 → 全局 + `WARN` | **返回值：** 最后一个**符号**（而非其值）——与返回最后一个**值**的 `setq` 不同 |
| `(psetq v1 val1 v2 val2 ...)` | 并行赋值 | 先求值所有值，再赋值 |
| `(macrolet ((m . spec)) . body)` | 局部宏 | 非递归 |
| `(symbol-macrolet ((s expansion)) . body)` | 符号宏 | |
| `(flet ((f . spec)) . body)` | 局部函数 | 非递归 |
| `(labels ((f . spec)) . body)` | 局部函数 | 递归（可相互递归） |
| `(block name . body)` | 命名块 | 词法作用域 |
| `(return-from name [val])` | 非局部退出 | |
| `(return [val])` | `(return-from nil [val])` | |
| `(tagbody stmt ...)` | 跳转标签 | 原子 = 标签 |
| `(go tag)` | 跳转 | 词法作用域，不求值 |
| `(catch tag . body)` | 动态捕获 | 标签**会被求值** |
| `(throw tag val)` | 动态抛出 | |
| `(trap expr handler)` | 简单捕获 | `(trap expr (lambda (e) ...))`，e = 错误信息字符串 |
| `(unwind-protect protected cleanup)` | 总是执行清理 | |
| `(lock mutex . body)` | 临界区 | `mutex` 来自 `lock-make`；body 同 `begin` |
| `(eval form)` | 全局求值 | 总是在 `Env.Root()` 中 |
| `(load "file")` | 加载文件 | **注意：** 在 `defun` 中 → 局部绑定！ |
| `(prog1 first . rest)` | 返回第一个值 | |
| `(prog2 a b . rest)` | 返回 b | |
| `(cond (test result) ...)` | 条件 | `else`/`t` = 默认分支 |
| `(case key (vals result) ...)` | 结构化分派 | 使用 `equal?` 比较 |
| `(the type form)` | 声明 | 类型被**忽略**（无类型系统） |
| `(declare . decls)` | 声明 | 空操作 |
| `(eval-when (situation) . body)` | 时机控制 | |
| `(progv syms vals . body)` | 动态绑定 | **注意：** 没有词法/动态区分 |
| `(and . exprs)` | 短路求值 | |
| `(or . exprs)` | 短路求值 | |
| `(not x)` | 取反 | |
| `(parfunc name [:timeout N] . exprs)` | fork-join 并行求值 | `name` = 符号（不是组合函数！），在当前环境中绑定到结果列表并返回；`:timeout` 单位为秒，超时的分支 → `()` |
| `(while test . body)` | 循环 | |
| `(do ((var step) ...) (test result) . body)` | Scheme 式迭代 | 并行 step |
| `(do* ((var step) ...) (test result) . body)` | Scheme 式迭代 | 顺序 step |
| `(multiple-value-list form)` | 多值 → 列表 | |
| `(multiple-value-bind (vars) form . body)` | 绑定多值 | |
| `(multiple-value-call fn . forms)` | 传递多值 | |
| `(multiple-value-prog1 form . rest)` | 保留第一个值 | |
| `(multiple-value-setq vars form)` | 赋值多值 | |
| `(nth-value n form)` | 第 n 个值 | |
| `(function fn)` | 函数字面量 | `#'` 读取器语法糖 |
| `(macroexpand form)` | 展开宏 | 非尾调用 |
| `(macroexpand-all form)` | 完全展开 | 非尾调用 |
| `(bound? sym)` | 是否已绑定？ | sym 会被求值 |
| `(makunbound sym)` | 删除绑定 | |
| `(exec "prog" param: "arg" … stdout: var stderr: var exitcd: var)` | 启动程序（不经过 shell！） | `stdout:`/`stderr:`/`exitcd:` 是变量名；`timeout: N` 秒（默认 60）；退出码 ≠ 0 不算错误 |
| `(quasiquote x)` | 准引用 | `` `x ``，`,x`=unquote，`,@x`=splice |
| `(unquote x)` | 只能在准引用内 | 在外部使用报错 |
| `(unquote-splice x)` | 只能在准引用内 | 在外部使用报错 |

**尾位置**（设置 `expr`/`env`，在循环中 `continue`）：`if`、
`begin`/`progn`/`locally`、`let`、`let*`、`cond`、`case`。其他所有
case 分支立即返回或委托给 `eval*` 辅助函数（例如 `catch`、
`unwind-protect`、`do`/`do*` —— 没有 TCO，但受 `ectx.child()` 限制）。

---

## 3. 重要原语（FUNC，Go 实现）

### 算术/比较
`+ - * / mod rem remainder abs floor ceiling truncate round ash random values sqrt exp log expt` —— `(log x)` 仅自然对数，x > 0；`expt` 也支持负数/分数指数（`(expt 0.9 -2)`），NaN/Inf → 报错；`mod` 同 CL 向下取整（`(mod -3 2)` → 1），`rem`/`remainder` 向零取整（→ -1）
`(floor x [d])` `ceiling` `truncate` `round` 同 CL：返回两个值（商、余数）—— `(truncate 7 2)` → `3 1`；`round` 取偶（`(round 2.5)` → 2，`(round 3.5)` → 4）；`(ash x n)` 位移，n < 0 时右移
`(parse-int "42")` `(parse-float "4.5")` —— 仅十进制，无进制参数；无效输入 → 报错（不是 `nil`）
`= /= < > >= <= equal? eql eq eq?` —— 数值比较为 CL 链式：`(< 1 2 3)` → `t`，`(/= 1 2 3)` → `t`（两两不同）；`eq`=指针，`eql` 同 CL（数字按值，字符串从不相等），`equal?`=结构比较（哈希表、向量、通道只按同一性比较，同 CL `equal`）

### 列表（经典 7 个）
`car cdr cons atom null list append`
`atom? null? string? number? list? symbol?` —— 类型谓词
`mapcar sort` —— 原语（一等公民：可用于 `funcall`/`apply`）

### 符号/原子
`gensym intern symbol-name symbol->string`

### 输出
`print println read warn`

### 控制/错误
`error apply funcall exit` —— `error` **只产生字符串**，没有 condition 对象
`exit` —— 立即结束进程，参数为数字退出码（不做清理！）

### 环境/内省
`memstats sleep env-symbols lisp-implementation-version` —— `(sleep ms)` 单位为毫秒；`(env-symbols)` 返回根环境中所有名字
（已排序，`tools/gen-reference.lisp` 以此为基础）；`(lisp-implementation-version)` → `"0.24.0"`

### 领域模块（各自的 Register-Xxx）
- **sigoREST：** `sigo sigo* sigo-request sigo-models sigo-model-info sigo-costs sigo-budget sigo-host sigo-system-prompt sigo-reference sigo-usage sigo-usage-reset` —— `(sigo "prompt" [model] [session-id] [host])` → 文本；`sigo*` → 哈希表（`text model finish-reason prompt-tokens completion-tokens cached-tokens reasoning-tokens cost-usd elapsed`）；`(sigo-request h [host])`：哈希表的键必须恰好来自 `model messages temperature max_tokens session_id timeout retries system_prompt bare channel`（其他键 → 报错；未给 `temperature` 时 golisp2 发送 0.2，`0` 会报错；未给 `max_tokens` 时使用模型最大值），结果 = 完整响应的哈希表外加 `elapsed`（秒）；`(sigo-usage)` 累计值，含 `calls`/`elapsed`；`(sigo-costs [since [until]])`（Unix 秒，同 `(now)`），`(sigo-budget)`，`(sigo-model-info [model])`（价格为美元/百万 token，**0 = 未知，不是免费**）；`(sigo-system-prompt "")` → 模型只看到提示词（若有 session-id 则加上会话历史）
- **Goroutine：** `chan-make chan-send chan-recv chan-try-recv chan-try-send chan-close chan-closed? chan-select spawn lock-make`
  - `(chan-recv ch [:timeout sec])` → `(values x t)`；已关闭且为空或超时 → `(values nil nil)`（`sec` = 秒，可用小数）。`(chan-try-recv ch)` 同 `chan-recv`，但从不阻塞
  - `(chan-send ch x)` 会阻塞；向已关闭通道发送报错。`(chan-try-send ch x)` → `t` / `nil`（缓冲区满）；`(chan-close ch)` → `t`，第二次关闭报错；`(chan-closed? ch)` 仅在已关闭**且**为空时返回 `t`
  - `(chan-select chans [:timeout sec])` → `(values x t i)`（`i` = 在 `chans` 中的下标，从 0 开始）；超时或全部已关闭且为空 → `(values nil nil nil)`
  - `(spawn fn)` → 立即返回一个结果通道，其中恰好有一条消息 `(:ok value)` 或 `(:error "text")`；错误同时输出到 stderr。是 `parfunc`（会等待）的对应物
  - `(pool-map n fn list)` → 同 `mapcar`，但同时最多 `n` 个调用（工作池，例如用于大量 `sigo` 调用）；结果按输入顺序，失败的元素变为 `(:error "text")`，其他元素继续运行
- **共享内存：** `shm-alloc shm-free shm-write shm-read shm-status shm-cleanup`
- **文件 I/O：** `file-write file-append file-read file-exists? file-delete set-working-directory get-working-directory get-file-path directory-files gets slurp err-write printf sprintf fprintf sscanf argv getenv environ`
- **Shell：** `system shell-output file-stat shell-assoc` —— `(system "cmd")` → 只返回退出码；`(shell-output "cmd")` → stdout 字符串（同 `$(…)`），退出码 ≠ 0 → 报错；`shell-assoc` 只是 `assoc`，不调用 shell
- **时间：** `now format-time get-universal-time sleep` —— `(now)` → Unix 秒（浮点数）；`(format-time "%F %T" [time] [:utc])` 类似 strftime（`%Y %y %m %d %H %M %S %j %F %T %a %A %b %B %z %Z %s %N %3N %%`，名称为英文）；`get-universal-time` 从 1900 年起计
- **字符串：** `string-length string-append substring string-upcase string-downcase string->number number->string string->list list->string string-replace string-trim string-contains string-find string-split string-join string= string/= string< string> string<= string>=` —— 逐字符比较，可用于字符串或符号：`string=` → t/()，其他 → 第一个不同处的下标或 ()（CL），`(sort l string<)`；`(string-split s [sep])`：给出 sep 时按字面分割（保留空字段），不给时按空白分割；`(string-join list sep)` sep 必填；参数顺序：`string-contains`/`string-split` 先给被搜索的字符串，`string-find` 先给要找的子串
- **正则（Go RE2，无反向引用，模式在前）：** `(regex-match re s)` → `t`/`()`；`(regex-find re s)` → 第一个匹配或 `()`，第二个值为分组列表；`(regex-find-all re s)` → 列表；`(regex-replace re s new)` 替换全部，分组用 `$1`/`${name}`（字面 `$` 写作 `$$`）；`(regex-split re s)` → 列表。字符串中反斜杠需双写：`"\\d+"`。
- **哈希表：** `make-hash-table gethash puthash remhash clrhash hash-table-count hash-table-p maphash` —— `puthash` 为内部函数：`(puthash key table value)`，表在第 2 个参数位置（与 Emacs Lisp 不同）；推荐 `(setf (gethash k h) v)`。相同键 → 值被覆盖。`(gethash k h [default])` → 值，第二个值 `t`/`()` = 是否存在。`:test` 为 `'eql`（默认）或 `'equal`：数字按值，符号按名字，**字符串按内容——即使在 `eql` 下也是如此**（CL：按同一性）；列表只在 `'equal` 下按结构比较；哈希表、向量、lambda 作为键时总是按同一性
- **向量（一维，O(1)）：** `make-array vector aref aset vector-push-extend vector-pop vectorp arrayp` —— `(make-array n [:initial-element x])`，`(aref v i)`，`(aset v i x)` 或 `(setf (aref v i) x)`，`(vector-push-extend x v)` → 新下标；`length`、`coerce`（`'list`/`'vector`）和 `sort`（返回新向量）支持向量；列表函数（`mapcar`、`apply`、`append`、`dolist` …）对向量会报错——先用 `(coerce v 'list)`。字面量/打印形式 `#(1 2 3)`（元素不求值，自求值）；`(loop for x across v …)`；`json-encode` → JSON 数组。可变且共享；`equal?` 只按同一性（CL `equal`）。
- **序列（列表或向量，不含字符串）：** `elt subseq copy-seq concatenate map some find find-if position position-if count count-if` —— `(subseq s start [end])` 和 `copy-seq` 保持类型；`(map 'vector f s…)`，`(concatenate 'list a b)`（类型 `'list`/`'vector`）；`find`/`position`/`count` 支持 `:test`（默认 `eql`，字符串用 `:test 'equal?`）和 `:key`；非破坏性。
- **JSON：** `json-parse json-encode` —— 对象 ↔ 哈希表（`:test 'equal`，字符串键），数组 ↔ 列表，`true` ↔ `t`，`null` ↔ `:null`，`false` → `()`，`()` → `[]`，`:false` → `false`。关联列表**不是**对象（编码时报错）。同样适用于 `ws-call`/`ws-export`/`ws-emit`
- **FORMAT：** `format` —— CL HyperSpec 22.3，`~A ~S ~D ~B ~O ~X ~R ~P ~C ~F ~E ~G ~$ ~% ~& ~| ~T ~* ~? ~[ ~{ ~( ~; ~^ ~/fun/ ~~`
  - 舍入：四舍六入五成双（Go `strconv`），不是 C 的四舍五入 —— `2.25` 的 `%.2f` → `"2.2"`
- **PostgreSQL：** `pg-connect pg-query pg-exec pg-close`
- **Maxima（计算机代数系统，外部进程）：** `(maxima-open)` → 会话；`(maxima-eval s "integrate(x^2,x)" [timeout-sec])` → 结果字符串（状态保留在会话中）；`(maxima-close s)` → `t`。需要已安装 `maxima`
- **遗传算法：** `ga-create ga-init ga-cross ga-calc ga-select ga-result ga-mut ga-print ga?`
- **重定义：** `redefine-policy redef-log redef-log-clear defined-in`
- **跟踪：** `trace untrace trace?`
- **defsystem-lite：** `defsystem load-system unload-system loaded-systems system-symbols`
- **Web 桥（golisp2web）：** `http-serve http-static http-stop http-port http-upload http-wait webserv ws-call ws-clients ws-emit ws-emit-to ws-eval ws-export ws-unexport browser-open`
- **Condition-lite：** `define-condition signal handler-case documentation`（见第 7 节）

完整的自动生成清单（非节选）：`docs/referenz-generiert.md`。

---

## 4. Stdlib（embed/stdlib.lisp，约 90 个定义）

### 访问器/快捷方式
`cadr caddr cadddr cddr cdar caar first second third fourth rest`
`zero? positive? negative? pair?` —— CL 名称：`zerop plusp minusp evenp oddp 1+ 1-`（`evenp`/`oddp` 仅限整数）

### 高阶/函数式
`identity constantly complement compose reverse length nth last member assoc filter drop take reduce for-each any every flatten zip list-tail iota shuffle randomlist max min square gcd` —— `(shuffle l)` → 新的随机打乱的列表（仅限列表，原列表不变）；`(randomlist n)` = `(shuffle (iota n))`
函数参数：lambda、`#'f`、`f` 或带引号的符号 `'f`（同 CL 按全局解析）—— `(mapcar 'string->number l)`，`(reduce '+ l)`，`(sort l '<)`。

### 列表辅助函数
`alist-set alist-get union set-difference find-all set-nth`
`remove remove-if remove-if-not remove-duplicates butlast copy-list copy-tree make-list getf` —— 仅限列表；`(remove x l)` 用 `equal?` 比较，**没有** `:test`/`:key`（→ 报错 „zu viele Argumente“（参数过多））；`(make-list n :initial-element x)`；`(getf plist key [default])`

### 宏
`when unless let* dotimes dolist push pop incf decf defvar defparameter setf defstruct defgeneric defmethod ignore-errors assert destructuring-bind macroexpand-1`
- `(incf place [d])`/`(decf place [d])` 基于 `setf`；`defparameter` 总是覆盖，`defvar` 仅在未绑定时赋值
- `(ignore-errors . body)` → 值，出错时为 `()`（只有一个值，没有 condition 对象；`throw`/`return-from` 照常穿过）
- `(assert form)` → `t` 或报错 `assert failed: form` —— 只接受一个参数，不能自定义信息
- `(destructuring-bind (a b c) list . body)` —— **仅限扁平模式**：嵌套模式、点对和 `&rest`/`&key`/`&optional` → 报错。取剩余部分：`(destructuring-bind (a b) l …)` 再加 `(cddr l)`
- `macroexpand-1` = `macroexpand`（两者都只展开一层）

**注意：** 这里列出 `let*` 只是照顾 CL 习惯——实际实现是第 2 节中的原生特殊形式
（`src/embed/stdlib.lisp` 有意**不**定义它）。

### 迭代器
`dotimes (var n) body` —— `(dotimes (i 10) ...)`
`dolist (var lst) body` —— `(dolist (x xs) ...)`

### 结构体
`(defstruct name [doc] slot|(slot default) ...)` —— 生成：`(make-name :slot v ...)`、`name-slot`、`name?`（+ `name-p`）
`(setf place val)` —— 通用，`(defstruct ...)` 会自动注册访问器

### 类型
- `(type-of x)` → `integer` `float` `string` `symbol` `keyword` `boolean` `null` `cons` `function` `compiled-function` `macro` `hash-table` `vector`，或结构体名、类名（defclass）、condition 名；类是 `hash-table` 的子类型。`(typep x spec)` 支持类型名、`(or …)` `(and …)` `(not …)` `(member …)` `(eql x)` `(satisfies f)` `(integer lo hi)`（`(n)` = 不含边界，`*` = 无边界）。`3.0` 是 `integer`。
  另外可用作类型名（`type-of` 不会返回，但可作 `typep` 的父类型）：`rational` `real` `number` `atom` `list` `sequence` `structure-object` `t`。不存在 → `typep: unbekannter Typ '…'`（未知类型）：`fixnum` `character` `double-float` `single-float`。类型名区分大小写 —— `'INTEGER` 未知，只能写 `'integer`。`(typecase x (integer …) ((or string symbol) …) (otherwise …))` 按类型分支，`etypecase` 无匹配时报错。`(check-type x (integer 0 *) ["描述"])` → `()` 或报错。两者都抛出普通 `error`，不是 `type-error`。`(deftype digit () '(integer 0 9))` 定义类型名，带参数：`(deftype upto (&optional (n '*)) (list 'integer 0 n))` → `(typep x '(upto 5))`；与内置类型、结构体、类、condition 同名 → 报错。`(subtypep 'integer 'number)` → 两个值 `t t`；无法判定（`not`、`satisfies`、子区间）→ `() ()`。

---

## 5. 真值 / Nil

- `()` / `nil` / `NIL` → 单例 nil（指针同一性！）
- `t` → 真，但*不是*唯一的真值 —— 除 nil 外一切皆为真
- `(eq '() '())` → `t`（单例）
- `(eq 5 5)` → `()` —— **设计如此：** `eq` 对数字总是返回 `()`（见 10.6）。有疑问时用 `equal?`

---

## 6. 准引用模式

```lisp
`(a b c)           ; 纯 quote
`(a ,x c)          ; unquote
`(a ,@xs c)        ; unquote-splice
```

---

## 7. 错误处理

```lisp
; 抛出错误
(error "message")             ; 中止，只产生字符串

; 捕获
(trap expr (lambda (e) ...))  ; e = "msg"（错误信息字符串）

; 动态
(catch 'tag body ...)
(throw 'tag value)
```

**Condition-lite**（`embed/condition.lisp`，自动加载）——
带类型层级和槽的结构化错误：

```lisp
(define-condition file-error (io-error) (path))  ; 类型 + 父类型 + 槽
(signal 'file-error :path "x.lisp")              ; 抛出，总是展开栈！
(handler-case (load "x.lisp")
  (file-error (e) (file-error-path e))  ; 自动生成读取器：类型-槽
  (io-error  (e) "some io error")       ; 也匹配子类型
  (error     ()  "without var binding")) ; 变量可以是 ()
```

- 基础层级：`condition` → `error` → `lisp-error`
- **Go 错误**（file-read 等）在 `handler-case` 中变为 `lisp-error`，
  通过 `(lisp-error-msg e)` 获取信息
- 无匹配 → **重新抛给**外层处理器
- 重新定义类型会静默替换（重新加载语义）
- **与 CL 的差异：** `signal` 总是展开栈（行为同 CL 的 `error`，
  而不是 CL 的 `signal`）。没有 restart，没有 `handler-bind`。
- 槽名在整个继承层级中必须唯一
  （扁平 plist，没有遮蔽）。

**测试框架：** `tests/test-framework.lisp` —— `defsuite`、`deftest`
（`:suite`、`:expected-failure`）、`is`、`run-tests` → FAIL 数量。
典型用法：`(exit (run-tests))` → 退出码 = FAIL 数量。

---

## 8. 求值环境

- `(eval form)` → **总是 `Env.Root()`**，从不在 lambda 作用域中
- `load` → **例外：** 在 `defun` 体内 → 局部绑定，不是全局
  - 变通方法：用 `(eval '(load "file"))` 全局加载
- `redefine-policy` → `'allow` / `'warn` / `'error`（默认 `warn`）

---

## 9. 陷阱 / 与 CL 的差异

| 情况 | GoLisp2 | CL |
|------|---------|-----|
| `(eq 5 5)` | `()` —— `eq` 对数字总是 `()`（设计如此） | 常为 `t`（小整数） |
| `defun` 中的 `load` | 局部绑定 | 全局 |
| `progv` | 没有词法/动态区分 | 动态 |
| `declare` | 空操作 | 类型检查 |
| `the` | 忽略类型 | 类型检查 |
| `(eval form)` | 全局 | 全局（一致） |
| `macrolet` | 非递归 | 递归 |
| `(define (f p) ...)` | `(define f (lambda (p) ...))` 的简写（Scheme 风格） | 不存在 —— 用 `defun` |
| 结构体是列表 | `(typep p 'cons)` → `t` | 独立类型，不是 `cons` |
| 整数值的浮点数是 integer | `(type-of 3.0)` → `integer` | `single-float`/`double-float`（3.0 是浮点数） |
| `(make-hash-table)`（默认 `eql`）中的字符串键 | 按内容：两次 `"rot"` = 同一个键 | `eql` → 同一性，两个 `"rot"` 字面量 = 两个键；需要时用 `:test 'equal` |
| `(puthash k h v)` | 表在第 2 个位置；优先用 `(setf (gethash k h) v)` | Emacs Lisp：`(puthash k v h)` |
| `stringp` `consp` `equal` | 不存在 → 报错 „unbekanntes Symbol“（未知符号）；改用 `string?`、`(typep x 'cons)`、`equal?` | 存在 |
| `destructuring-bind` 使用 `&rest`/`&key`、嵌套、点对 | 报错 —— 仅限扁平模式 | 完全支持 |
| `(member x l)` | 没有 `:test`；字符串按结构比较 | `eql`（字符串需 `:test 'equal`） |
| `(last l)` | 返回**元素**，不是 cons 单元 | 最后一个 cons 单元 |
| `(slurp)` | 读取 **stdin**；读文件 → `(file-read path)` | – |
| 输入结束时的 `(gets)` | 报错 `gets: EOF`，不是 `nil` | `read-line` → `nil`/EOF 值 |
| `(sleep ms)` 与 `:timeout sec` | `sleep` 单位为**毫秒**，`:timeout` 单位为**秒** —— `(sleep 0.05)` 无效果 | `sleep` 单位为秒 |
| `(parfunc r :timeout N …)` | `N` 只能是数字字面量（不求值），≥ 0，可用小数；`:timeout 0` = **没有**超时（而 `chan-recv`/`chan-select` 中：立即返回） | – |

---

## 10. 局限（有意为之，单独成节）

这些限制是**设计决定**，不是 bug。AI 必须了解它们，以免猜测：

### 10.1 没有包系统
- 所有符号位于同一个全局命名空间。
- 没有 `defpackage`、`in-package`、`export`、`import`。
- 只能靠命名约定避免冲突（前缀如 `ga-`、`shm-`）。

### 10.2 没有完整的 CLOS
- `defstruct`（构造函数、访问器、谓词）。
- CLOS-light：`defgeneric`/`defmethod` —— 基于结构体标签或 defclass 类的单分派，`t` = 默认方法。
- `(defclass name (slot (slot default) ...))` —— 有同一性的对象（哈希表，`:class`）；`make-name`、`name?`、`name-slot`；`setf` 原地修改；打印为 `#<name>`。**不是** CL 语法：没有父类列表，没有 `:initarg`。
- 没有继承，没有 `call-next-method`，没有方法组合，没有 `call-method`。

### 10.3 只有 condition-lite，没有完整的 CL condition 系统
- 提供 `define-condition`/`signal`/`handler-case`（层级、槽、
  按继承分派、Go 错误回退为 `lisp-error`）。
- **但是：** `signal` 总是展开栈（同 CL 的 `error`），没有 restart，
  没有 `handler-bind`，没有 `restart-case`，没有 MOP。
- 槽名是扁平的 —— 跨继承不能遮蔽。

### 10.4 progv 没有词法/动态区分
- `progv` 的绑定方式同 `let` —— 词法遮蔽也会看到 progv 的值。
- CL：词法绑定不受 progv 影响。

### 10.5 没有 compile-file
- 纯解释器。没有 `compile-file`，不能加载 FASL。
- 没有面向编译器/编译期的 `eval-when` 设置。

### 10.6 `eq` 对数字总是返回 `()`
- `(eq 5 5)` → `()`。`(eq 1000 1000)` → `()`。即使值相同也是如此。
- 内部有小整数缓存（-32768..32767，`types.go` 中的 `MakeNum`）
  用于避免分配 —— 但 `eq` 仍有意把数字视为从不同一
  （`primitives.go` 中的 `fnEqPtr`）。
- 比较数字总是用 `equal?` 或 `=`，不要用 `eq`。

### 10.7 宏非递归（macrolet）
- `macrolet` 的定义体看不到同一层的其他宏。
- 函数的 `labels` 是递归的 —— 与 CL 不对称。

### 10.8 没有续延，没有 MOP
- 没有 `call/cc`。
- 有 `catch`/`throw`（第 2 节），但没有 restart 语义。
- 没有 CLOS 元对象协议。

### 10.9 `defun` 中的 `load` 是局部绑定
- 变通方法：用 `(eval '(load "file"))` 全局加载。
- 原因：`eval` 在 `Env.Root()` 中运行。

### 10.10 没有细粒度的 GC 控制
- `memstats` 返回 Go 运行时统计。
- 没有 `tweak`，`make-hash-table` 不支持弱引用。

### 10.11 精简的类型系统
- `declare` 和 `the` 是空操作。
- 提供 `type-of`、`typep`、`typecase`、`etypecase`、`check-type`、`deftype`、
  `subtypep`（见“类型”）。没有 `ctypecase`、`ccase`。

### 10.12 LOOP 只有实用核心，没有 Series，没有 Iterate
- `loop`：`for x in`/`from … to`/`below`/`by`、`repeat`、`while`/`until`、
  `when`/`unless`、`collect`/`sum`/`count`/`maximize`/`minimize`、`do`、
  `finally`、`return`、`for x across vector`。不支持：解构、`named`。
- 没有 `series`，没有 `iterate`。

### 10.13 带函数语法糖的 `define`
- `(define (f p) body)` 是 `(define f (lambda (p) body))` 的简写。

---

## 11. 文件结构（AI 扫描用）

```
src/
  lib/
    types*.go          Cell 数据结构
    reader.go          Read / ReadAll
    env.go             Env（RWMutex，池）
    eval_core.go       求值蹦床 + 特殊形式分派
    eval_*.go          特殊形式、lambda、控制流、准引用
    primitives.go      BaseEnv() —— 所有 Go 原语
    *.go               goroutine、fileio、shellcmd、postgres、genalg、shm、sigorest
    stdlib.go          //go:embed stdlib.lisp
    format*.go         CL HyperSpec 22.3
    trace.go           trace/untrace
  embed/               //go:embed 资源（stdlib.lisp、condition.lisp、...）
  main.go              CLI
```

---

## 12. 速查（AI 备忘单）

| 我需要… | 使用 |
|---------|------|
| 算术 | `+ - * / mod` |
| 比较 | `equal?`（数字！），`eq`（指针） |
| 构建列表 | `cons list append` |
| 迭代 | `dolist dotimes loop mapcar reduce` |
| 并行 | `parfunc` |
| 错误 | `error` + `trap` |
| 动态控制 | `catch/throw` |
| 结构体 | `defstruct` |
| 字符串 | `string-append substring string-replace` |
| 文件 | `file-read file-write` |
| 数据库 | `pg-connect pg-query` |
| AI | `sigo` |
| Web 桥 | `http-serve webserv ws-emit` |
| 调试 | `trace` |
| 所有符号 | `(env-symbols)` 或 `docs/referenz-generiert.md` |

---

**AI 参考结束。** 面向人类的版本：`docs/golisp2-cheatsheet.md`（德语）。
完整的自动生成函数参考：`docs/referenz-generiert.md`。
权威原文（德语）：`src/embed/ki-referenz.md` · English: `docs/ki/referenz_en.md`。
