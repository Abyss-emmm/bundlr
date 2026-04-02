# bundlr

[English version](./README_EN.md)

> 将源代码打包成单个文件 — 随时粘贴到任何 LLM 中使用。

在向 LLM（ChatGPT、Claude、Gemini 等）咨询代码库时，经常需要同时分享多个文件。`bundlr` 会遍历项目目录，收集你关心的文件，并将它们合并成一个带有清晰路径标识的单一文件 — 让 LLM 始终知道每段代码来自哪个文件。

---

## 安装

```bash
go mod tidy          # 拉取 gopkg.in/yaml.v3
go build -o bundlr bundlr.go

# 可选：移动到 PATH 中
mv bundlr /usr/local/bin/bundlr

# 可选：复制一份示例配置到常用位置
cp bundlr.yaml ~/.bundlr.yaml
```

也可以直接前往 GitHub 的 Releases 页面下载已经打包好的 Windows、Linux 和 macOS 版本。

---

## 快速开始

```bash
# 打包当前目录下的所有 Python 文件
bundlr . -o bundle.py

# 打包所有 Go 文件，排除 vendor 目录
bundlr . -o bundle.go -ext .go -exclude vendor

# 将输出粘贴到 LLM 对话中开始提问
```

---

## 使用方法

```
bundlr [参数] [src]
```

| 参数 | 默认值 | 说明 |
|---|---|---|
| `-c` | _(none)_ | YAML 配置文件路径 |
| `-o` | `all_in_one.py` | 输出文件路径 |
| `-ext` | `取自 -o 的后缀` | 要收集的文件扩展名 |
| `-include` | _(all)_ | 只包含匹配此相对路径 glob 的文件 |
| `-exclude` | _(none)_ | 排除匹配此相对路径 glob 的目录或文件 |

`src` 是位置参数；省略时默认扫描当前目录。

---

## 参数详解

### `-ext` — 选择文件类型

逗号分隔或重复使用。前缀的点号可选。
如果未提供 `-ext`，bundlr 会使用 `-o` 的后缀。
如果 `-o` 也没有后缀，则直接报错退出。

```bash
bundlr -ext .go
bundlr -ext .go,.ts,.js
bundlr -ext go -ext ts        # 效果相同
```

### `-exclude` — 跳过目录或文件

匹配相对于 `src` 的路径，并统一使用 `/` 作为分隔符。逗号分隔或重复使用。
`*` 只匹配单个路径段，`**` 可以跨多级目录匹配。

```bash
bundlr -exclude vendor                             # 跳过根目录下的 vendor/
bundlr -exclude venv -exclude dist                # 跳过多个根目录
bundlr -exclude 'internal/generated/*.go'         # 跳过某个目录中的生成文件
bundlr -exclude '**/*.pb.go'                      # 跳过任意层级中的匹配文件
bundlr -exclude 'internal/**/generated/*.go'      # 跨多级目录匹配
bundlr -exclude 'cmd/api/*.go,cmd/web/*.go'       # 逗号分隔
```


### `-include` — 白名单特定文件

只收集相对路径匹配给定 glob 的文件，并统一使用 `/` 作为分隔符。
`*` 只匹配单个路径段，`**` 可以跨多级目录匹配。

```bash
bundlr -include 'cmd/api/*.go'                 # 只收集某个目录中的文件
bundlr -include '**/*_test.go'                 # 只收集任意层级的测试文件
bundlr -include 'internal/**/handler_*.go'     # 匹配多级目录中的 handler 文件
```

---

## 输出格式

每个文件都用清晰的标题分隔，让 LLM 准确知道每段代码在项目中的位置：

```
# ===== File: internal/handler/user.go =====

package handler
...

# ===== File: internal/router/router.go =====

package router
...
```

---

## 示例

```bash
# Python 项目 — 跳过虚拟环境和缓存
bundlr ./myproject -o bundle.py -ext .py -exclude venv -exclude __pycache__

# Go 项目 — 跳过 vendor 和生成的文件
bundlr . -o bundle.go -ext .go -exclude vendor -exclude '**/*_generated*'

# 只与 LLM 分享测试文件
bundlr . -o tests.go -ext .go -include '**/*_test.go'

# 多语言项目（Go + TypeScript）
bundlr . -o bundle.txt -ext .go,.ts -exclude node_modules -exclude vendor

# 只聚焦 handler 层
bundlr . -o handlers.go -ext .go -include '**/handler_*.go' -exclude vendor
```

---

## 配置文件

如果你不想每次都重复输入 `-exclude venv -exclude node_modules` 之类的默认规则，可以把它们写进 YAML 配置文件，再通过 `-c` 指定：

```bash
bundlr -c ~/.bundlr.yaml . -o bundle.py
```

配置文件不会自动加载，必须显式传入 `-c`。
配置文件只提供 `ext`、`include`、`exclude` 的默认值；`src` 和 `-o` 仍然只能通过命令行指定。

**示例配置：**

```yaml
# ~/.bundlr.yaml
exclude:
  - venv
  - .venv
  - vendor
  - node_modules
  - dist
  - build
  - "**/*.pb.go"
  - "**/*_generated*"
```

所有字段都是可选的，不需要的可以省略。

| 字段 | 类型 | 说明 |
|---|---|---|
| `ext` | 列表 | 默认文件扩展名；未提供时仍回退到 `-o` 的后缀 |
| `exclude` | 列表 | 默认排除规则 |
| `include` | 列表 | 默认包含规则 |

**合并规则：**

| 参数 | 配置文件与 CLI 的关系 |
|---|---|
| `src` / `-o` | 仅 CLI 支持，配置文件不会读取 |
| `-ext` / `-include` | CLI 显式传入时直接覆盖配置文件 |
| `-exclude` | 合并；CLI 传入的排除规则会追加在配置文件规则之后 |

这意味着配置文件中的 `exclude` 适合作为长期默认值，而单次运行时可以继续额外追加排除项。

---

## bundlr 使用技巧

- **`-include` 和 `-exclude` 要具体** — 打包内容越小越聚焦，`bundlr` 生成的结果就越适合发给 LLM。大多数 LLM 都有上下文窗口限制。
- **输出文件名要有意义** — 比如用 `auth_handlers.go` 而不是 `bundle.go`，这样你也能记住里面有什么。
- **修改后重新打包** — 做出更改后再运行一次 bundlr，这样下一次 LLM 对话就能看到最新代码。

### 小技巧：先获取项目里的后缀列表

如果你不确定该传哪些 `-ext`，可以先把项目里实际出现的后缀列出来。

**Windows（PowerShell）**

只保留有后缀的文件：

```powershell
Get-ChildItem -File | Where-Object { $_.Extension } |
Select-Object -ExpandProperty Extension | Sort-Object -Unique
```

包含子目录，并过滤掉没有后缀的文件：

```powershell
Get-ChildItem -Recurse -File | Where-Object { $_.Extension -ne "" } |
Select-Object -ExpandProperty Extension | Sort-Object -Unique
```

统计每种后缀的数量：

```powershell
Get-ChildItem -Recurse -File | Where-Object { $_.Extension } |
Group-Object Extension | Sort-Object Count -Descending
```

**macOS / Linux**

过滤掉没有后缀的文件，并列出唯一后缀：

```bash
find . -type f -name "*.*" | sed 's/.*\.//' | sort -u
```

更严谨的版本：

```bash
find . -type f | awk -F. 'NF>1 {print $NF}' | sort -u
```

统计每种后缀的数量：

```bash
find . -type f -name "*.*" | awk -F. '{print $NF}' | sort | uniq -c
```

---

## 许可证

MIT
