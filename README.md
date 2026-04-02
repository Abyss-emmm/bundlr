# bundlr

[English version](./README_EN.md)

> 将源代码打包成单个文件 — 随时粘贴到任何 LLM 中使用。

在向 LLM（ChatGPT、Claude、Gemini 等）咨询代码库时，经常需要同时分享多个文件。`bundlr` 会遍历项目目录，收集你关心的文件，并将它们合并成一个带有清晰路径标识的单一文件 — 让 LLM 始终知道每段代码来自哪个文件。

---

## 安装

```bash
go build -o bundlr bundlr.go
# 可选：移动到 PATH 中
mv bundlr /usr/local/bin/bundlr
```

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

模式会对完整相对路径做匹配，不再按路径片段或单独文件名隐式匹配。
像 `vendor` 这种不带 glob 的写法对根目录项依然有效，因为该目录的相对路径本身就是 `vendor`。
如果你想跳过隐藏目录、`__pycache__` 或其他生成内容，请显式使用 `-exclude`。

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

## LLM 使用技巧

- **`-include` 和 `-exclude` 要具体** — 打包内容越小越聚焦，LLM 的回答质量越高。大多数 LLM 都有上下文窗口限制。
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
