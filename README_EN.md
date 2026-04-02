# bundlr

> Bundle your source code into a single file — ready to paste into any LLM.

When asking an LLM (ChatGPT, Claude, Gemini, etc.) about your codebase, you often need to share multiple files at once. `bundlr` walks your project directory, collects the files you care about, and merges them into one clean file with clear path headers — so the LLM always knows which file each piece of code came from.

---

## Install

```bash
go mod tidy          # fetch gopkg.in/yaml.v3
go build -o bundlr bundlr.go

# Optional: move binary to your PATH
mv bundlr /usr/local/bin/bundlr

# Save your config anywhere you like, e.g. your home directory
cp bundlr.yaml ~/.bundlr.yaml
```

You can also download prebuilt Windows, Linux, and macOS binaries from the GitHub Releases page.

---

## Quick Start

```bash
# Bundle all Python files in the current directory
bundlr . -o bundle.py

# Bundle all Go files, skip vendor folder
bundlr . -o bundle.go -ext .go -exclude vendor

# Paste the output into your LLM chat and ask away
```

---

## Usage

```
bundlr [flags] [src]
```

| Flag | Default | Description |
|---|---|---|
| `-c` | _(none)_ | Path to YAML config file |
| `-o` | `all_in_one.py` | Output file path |
| `-ext` | `suffix from -o` | File extension(s) to collect |
| `-include` | _(all)_ | Only include relative paths matching this glob |
| `-exclude` | _(none)_ | Exclude relative paths matching this glob |

`src` is a positional argument. If omitted, bundlr scans the current directory.

---

## Flags in Detail

### `-ext` — Choose file types

Comma-separated or repeated. Dot prefix is optional.
If `-ext` is omitted, bundlr uses the suffix from `-o`.
If `-o` has no suffix either, bundlr exits with an error.

```bash
bundlr -ext .go
bundlr -ext .go,.ts,.js
bundlr -ext go -ext ts        # same result
```

### `-exclude` — Skip directories or files

Matches relative paths from `src`, using `/` as the separator. Comma-separated or repeated.
`*` matches within one path segment; `**` matches across directories.

```bash
bundlr -exclude vendor                             # skip the root vendor/ directory
bundlr -exclude venv -exclude dist                # skip multiple root dirs
bundlr -exclude 'internal/generated/*.go'         # skip generated Go files in one folder
bundlr -exclude '**/*.pb.go'                      # skip matching files at any depth
bundlr -exclude 'internal/**/generated/*.go'      # skip matching files across nested dirs
bundlr -exclude 'cmd/api/*.go,cmd/web/*.go'       # comma-separated
```


### `-include` — Whitelist specific files

Only collect files whose relative paths match the given glob. Uses `/` as the separator.
`*` matches within one path segment; `**` matches across directories.

```bash
bundlr -include 'cmd/api/*.go'                 # only files in one folder
bundlr -include '**/*_test.go'                 # only test files at any depth
bundlr -include 'internal/**/handler_*.go'     # match nested handler files
```

---

## Output Format

Each file is separated by a clear header so the LLM knows exactly where each piece of code lives in your project:

```
# ===== File: internal/handler/user.go =====

package handler
...

# ===== File: internal/router/router.go =====

package router
...
```

---

## Examples

```bash
# Python project — skip virtual env and cache
bundlr ./myproject -o bundle.py -ext .py -exclude venv -exclude __pycache__

# Go project — skip vendor and generated files
bundlr . -o bundle.go -ext .go -exclude vendor -exclude '**/*_generated*'

# Only share test files with the LLM
bundlr . -o tests.go -ext .go -include '**/*_test.go'

# Multi-language project (Go + TypeScript)
bundlr . -o bundle.txt -ext .go,.ts -exclude node_modules -exclude vendor

# Focus on handler layer only
bundlr . -o handlers.go -ext .go -include '**/handler_*.go' -exclude vendor
```

---

## Config File

Instead of typing `-exclude venv -exclude node_modules` every time, save your defaults in a YAML file and point bundlr at it with `-c`:

```bash
bundlr -c ~/.bundlr.yaml . -o bundle.py
```

No config file is loaded automatically — `-c` must be provided explicitly.
The config file only provides defaults for `-ext`, `-include`, and `-exclude`. `src` and `-o` are CLI-only.

**Example config:**

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

All fields are optional — omit any you don't need.

| Field | Type | Description |
|---|---|---|
| `ext` | list | Default file extensions. If omitted, bundlr falls back to the suffix from `-o` |
| `exclude` | list | Default exclude patterns |
| `include` | list | Default include patterns |

**Merge behaviour:**

| Flag | Config + CLI behaviour |
|---|---|
| `src` / `-o` | CLI only — not read from config |
| `-ext` / `-include` | CLI wins outright — config value ignored |
| `-exclude` | **Merged** — CLI adds on top of config defaults |

This means your config's `exclude` list is always active as a baseline, and you can layer on more exclusions per-run without losing the defaults.

---

## bundlr Usage Tips

- **Be specific with `-include` and `-exclude`** — the smaller and more focused the bundle, the better the LLM's response. Most LLMs have a context window limit.
- **Name your output file meaningfully** — e.g. `auth_handlers.go` instead of `bundle.go` so you remember what's inside.
- **Re-bundle after edits** — run `bundlr` again after making changes so your next LLM session has the latest code.

### Quick Tip: discover file extensions first

If you are not sure which `-ext` values to use, list the extensions in your project first.

**Windows (PowerShell)**

List only files that have an extension:

```powershell
Get-ChildItem -File | Where-Object { $_.Extension } |
Select-Object -ExpandProperty Extension | Sort-Object -Unique
```

Include subdirectories and skip files without an extension:

```powershell
Get-ChildItem -Recurse -File | Where-Object { $_.Extension -ne "" } |
Select-Object -ExpandProperty Extension | Sort-Object -Unique
```

Count each extension:

```powershell
Get-ChildItem -Recurse -File | Where-Object { $_.Extension } |
Group-Object Extension | Sort-Object Count -Descending
```

**macOS / Linux**

List unique extensions and skip files without a suffix:

```bash
find . -type f -name "*.*" | sed 's/.*\.//' | sort -u
```

More strict version:

```bash
find . -type f | awk -F. 'NF>1 {print $NF}' | sort -u
```

Count each extension:

```bash
find . -type f -name "*.*" | awk -F. '{print $NF}' | sort | uniq -c
```

---

## License

MIT
