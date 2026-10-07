# Project Docs GUI

Aggregate source code from your projects into a single, clean Markdown file — ready to feed to online LLMs for analysis. Desktop GUI (Wails) + headless CLI, written in Go with a Vue 3 + TypeScript frontend. Rewritten from the original Rust console tool `markdown_my_project`.

## Features

| ID | Feature | Description |
|----|---------|-------------|
| F1 | Full code export | All configured files aggregated into one Markdown (file tree + file contents), byte-compatible with the legacy Rust output |
| F2 | File manifest export | Paths/file tree only, with per-file language, line count and size |
| F3 | Symbol export | Function/class/struct names per file (regex-based, 9 languages) |
| F99 | Secret redaction | Detects passwords/API keys/tokens and strips them before export |
| F4 | Token estimate & split | ≈Token estimation (±20%) and greedy chunking by file units |

Export modes: `full` · `files` · `symbols` · `signatures` · `custom`.

## Tech Stack

- **Backend**: Go 1.22+, Wails v2, `gopkg.in/yaml.v3`, goroutine worker pool, `log/slog` + rolling file
- **Frontend**: Vue 3 + TypeScript + Pinia + Vite (pnpm)

## Project Layout

```
├── main.go                  # Wails GUI entry
├── app/app.go               # Thin Wails binding layer
├── core/                    # Pure Go, independently testable
│   ├── config/              # Project config load/validate/save (v1 compatible + v2 fields)
│   ├── scanner/             # File scanning, exclusion rules, parallel read
│   ├── generator/           # Markdown generation (full/files modes) + file tree
│   ├── symbol/              # Multi-language symbol extraction (regex v1)
│   ├── redactor/            # Secret detection & redaction engine
│   ├── token/               # Token estimation & chunk splitting
│   └── logger/              # slog console + rolling file
├── cmd/project-docs/        # Headless CLI
├── assets/langs.yml         # Extension → language mapping
├── config/                  # Sample config: projects/*.yml + languages.yml + log4rs.yml
├── frontend/                # Vue 3 + TS (pnpm)
└── build.ps1                # One-click build script (Go + pnpm → dist/)
```

## Build

Requirements: **Go 1.22+**, **Node + pnpm**, optional **Wails v2 CLI** (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`).

```powershell
.\build.ps1                # full build → dist/
.\build.ps1 -Clean         # wipe dist/ first
.\build.ps1 -SkipFrontend  # skip frontend build
```

Output in `dist/`:

```
dist/
├── project-docs-gui.exe    # Desktop GUI (Wails)
├── project-docs.exe        # Headless CLI
├── assets/langs.yml
└── config/                 # projects/*.yml + languages.yml + log4rs.yml
```

## Usage

### GUI

Run `dist\project-docs-gui.exe`. Default projects directory is `config/projects` (relative to the exe's working directory). Flow: pick a project → scan & check files → choose mode → export. Progress and errors are shown inline; empty/bad paths produce explicit error messages.

### CLI (headless)

```powershell
# Single project
project-docs.exe --config config/projects/project1.yml --langs config/languages.yml --out output

# Batch: all *.yml in a directory
project-docs.exe --projects config/projects --langs config/languages.yml --out output

# Override export mode
project-docs.exe --config config/projects/project1.yml --mode files
```

Modes: `full` (default) | `files` | `symbols` | `signatures`. If the config sets `split_tokens > 0`, output is chunked into `name.part1.md … name.partN.md`.

## Configuration

Project config (`config/projects/*.yml`):

```yaml
project_name: "My Project"
project_path: "C:/Workspaces/my-project"
output_file: "my_project.md"
markdown_lang: zh_cn            # zh_cn | en_us
files: [cargo.toml]             # individual files
directories: [src]              # recursive directories
exclude_directories: ["**/444"] # prunes whole directories; "**/name" matches anywhere
exclude_patterns: ["*.log"]     # glob (* / ?) or prefix match on relative paths
max_file_size: 1048576          # bytes; 0/omitted = no limit

# ─── v2 ───
export_mode: full               # full | files | symbols | signatures | custom
redaction:
  enabled: true
  placeholder: "[REDACTED]"
  strategy: placeholder         # placeholder | drop_line
  allowlist: []                 # confirmed false positives: "relpath:line:rule"
  custom_patterns: []           # extra RE2 regexes
split_tokens: 0                 # >0 enables chunking, e.g. 100000
```

## Secret Redaction (F99)

Two-phase by design — never silently rewrites code:

1. **Scan**: builtin rules (password assigns, AWS keys, GitHub PATs, OpenAI keys, Slack tokens, JWTs, PEM blocks, DB connection strings, Basic/Bearer auth) report masked findings.
2. **Confirm & export**: hits are replaced with `[REDACTED]` (or the line dropped via `strategy: drop_line`). PEM blocks are replaced as a whole. Confirmed false positives go into `redaction.allowlist` and are skipped afterwards. Logs record counts and rule names only — never plaintext.

Built-in false-positive suppression: values containing `your_`, `xxx`, `example`, `changeme`, `placeholder`, `dummy`, `test123`, `${...}`, `{{...}}`, or env references (`process.env`, `os.Getenv`, `import.meta.env`) are ignored.

## Development

```powershell
go test ./core/...          # unit tests (generator golden regression, symbol, redactor, token)
wails dev                   # live-reload GUI development
cd frontend && pnpm dev     # frontend-only dev server
```

## Notes & Limitations

- Regex-based symbol extraction is heuristic (directory-level reference for LLMs, misses are acceptable); a tree-sitter upgrade path is planned.
- Token estimation is ±20%, for capacity planning only.
- `full` mode on huge projects may produce very large files — prefer `split_tokens`.
- Scanned paths are stored with `/` separators for cross-platform output.
