# Project Docs GUI — 开发文档
> 用 Go + Vue 3 + TypeScript 重写原 Rust 控制台项目 `markdown_my_project`，新增桌面 GUI 与多项导出增强。本文档面向 AI Coding，可直接作为实现规格。
---
## 1. 项目概述
### 1.1 背景与目标
原项目是一个 Rust 控制台工具：读取 `projects/*.yml` 配置，将指定源码文件聚合为单一 Markdown（含文件树 + 文件内容），用于投喂在线 LLM 分析。
本项目将其重写为 **Go 后端 + Vue3/TS 前端** 的桌面 GUI 应用，并扩展导出能力：
| 编号 | 需求 | 说明 |
|---|---|---|
| F1 | 导出所有项目完整代码为单一 Markdown | 旧功能移植 |
| F2 | 导出项目代码**文件名**为 Markdown | 新增：仅路径清单/文件树 |
| F3 | 导出项目代码文件名（**含方法名**）为 Markdown | 新增：符号提取 |
| F4 | 其他功能（本文档 §9 设计） | Token 估算、分片、签名导出等 |
| F99 | 导出时**自动识别并剔除密码/密钥** | 新增：敏感信息过滤引擎 |
**最终目的**：生成干净、安全、上下文友好的项目摘要，供在线 LLM 分析（人工中转，不在本项目内调用 LLM API）。
### 1.2 技术栈
| 层 | 选型 | 说明 |
|---|---|---|
| 桌面框架 | **Wails v2** | Go 原生桌面壳，内置 Vite + Vue3 + TS 模板 |
| 后端语言 | Go 1.22+ | |
| 前端 | Vue 3 + TypeScript + Pinia + Vite | 组件库建议 Naive UI 或 Element Plus |
| YAML | `gopkg.in/yaml.v3` | |
| 并发扫描 | goroutine worker pool（替代 rayon） | |
| 目录遍历 | `io/fs.WalkDir`（替代 walkdir） | |
| 日志 | `log/slog` + 滚动文件（替代 log4rs） | |
| 进度 | Wails Events（替代 indicatif） | |
> 备选方案：若不需要桌面壳，可将 Wails binding 层替换为 `chi` HTTP API + 浏览器访问。架构上要求 **核心逻辑与 binding 层解耦**（见 §2）。
---
## 2. 总体架构
```
┌─────────────────────────────────────────────┐
│  frontend/ (Vue3 + TS)                      │
│  Views: 项目管理 / 导出向导 / 敏感扫描 / 历史  │
└──────────────┬──────────────────────────────┘
               │ Wails Bindings + Events
┌──────────────┴──────────────────────────────┐
│  backend/app  (App struct — 薄绑定层)        │
├─────────────────────────────────────────────┤
│  backend/core  (纯 Go，可独立测试)           │
│  ├─ config    配置加载/校验/迁移              │
│  ├─ scanner   文件扫描/过滤/并行读取          │
│  ├─ symbol    多语言符号提取 (F3)            │
│  ├─ redactor  敏感信息识别/剔除 (F99)        │
│  ├─ generator Markdown 生成 (F1/F2/F3)      │
│  ├─ token     Token 估算与分片               │
│  └─ logger    slog 日志                      │
└─────────────────────────────────────────────┘
```
**约束**：`backend/core` 不得 import Wails；进度通过回调接口 `ProgressReporter` 抽象，由 `app` 层适配为 Wails Event。
---
## 3. 目录结构
```
project-docs-gui/
├── main.go
├── wails.json
├── app/                        # Wails 绑定层
│   └── app.go                  # App struct：暴露给前端的所有方法
├── core/
│   ├── config/config.go        # ProjectConfig、加载/校验
│   ├── scanner/scanner.go      # 扫描、排除规则、并行读取
│   ├── scanner/filter.go       # glob 排除、大小限制
│   ├── symbol/extractor.go     # SymbolExtractor 接口 + 注册表
│   ├── symbol/regex_*.go       # go/py/js/java/rust/c 提取器
│   ├── redactor/redactor.go    # 敏感引擎：规则、替换、白名单
│   ├── redactor/rules.go       # 内置规则表（§7.2）
│   ├── generator/generator.go  # 按 Mode 分发生成
│   ├── generator/tree.go       # ├── └── 文件树（移植 tree_generator.rs）
│   ├── generator/template.go   # Markdown 模板
│   ├── token/estimate.go       # Token 估算、装箱分片
│   └── logger/logger.go
├── assets/langs.yml            # 扩展名→语言映射（原 languages.yml）
├── internal/eventbus.go        # ProgressReporter 实现
└── frontend/
    ├── src/api/bindings.ts     # Wails 生成 + 手写封装
    ├── src/stores/             # projects.ts, export.ts, secrets.ts
    ├── src/views/
    │   ├── ProjectListView.vue
    │   ├── ExportWizard.vue    # 核心：模式选择/文件勾选/预览/进度
    │   ├── SecretReportView.vue
    │   ├── HistoryView.vue
    │   └── SettingsView.vue
    └── src/components/
        ├── FileTreeCheckable.vue
        ├── CodePreview.vue
        └── ExportProgress.vue
```
---
## 4. 旧版（Rust）→ 新版功能映射
| Rust 源文件 | 新位置 | 迁移要点 |
|---|---|---|
| `config.rs` | `core/config` | 校验逻辑全保留；新增 v2 字段（§5） |
| `file_processor.rs` | `core/scanner` | `rayon`→goroutine pool；`WalkDir.filter_entry`→`WalkDir` 中跳过目录；`should_exclude_directory` 的 `**/`、路径分隔符、组件匹配逻辑**原样保留** |
| `tree_generator.rs` | `core/generator/tree.go` | `BTreeMap`→按 key 排序 map，输出格式 `├── / └── / │ ` 逐字符一致 |
| `markdown_generator.rs` | `core/generator` | `localized_text` 保留 zh_cn/en_us 双语 |
| `language.rs` + `languages.yml` | `assets/langs.yml` | 格式不变 |
| `logger.rs` + `log4rs.yml` | `core/logger` + `logger.yml` | 控制台+滚动文件双输出，pattern 对齐 `{time} {level} {file}:{line} — {msg}` |
| `main.rs` CLI | `cmd/cli.go`（子命令 `--cli`） | 保留无头模式：`project-docs --config projects/x.yml --mode full` |
---
## 5. 数据模型与配置文件
### 5.1 配置（v2，向后兼容 v1）
```yaml
# projects/epub_reader.yml
project_name: "EPUB Reader"
project_path: "C:/Workspaces/.../epub_reader"
output_file: "epub_reader.md"
markdown_lang: zh_cn            # zh_cn | en_us
files: [cargo.toml, log4rs.yml]
directories: [lib]
exclude_directories: ["**/444"]
exclude_patterns: ["*.log"]     # v1 已有字段，GUI 补编辑
max_file_size: 1048576
# ─── v2 新增 ───
export_mode: full               # full | files | symbols | signatures | custom
redaction:
  enabled: true
  placeholder: "[REDACTED]"     # 替换文本
  strategy: placeholder         # placeholder | drop_line
  allowlist: []                 # 已确认误报的规则命中记录
  custom_patterns: []           # 用户自定义正则（RE2 语法）
split_tokens: 0                 # >0 时按 token 分片，如 100000
template: default               # 模板 id
```
### 5.2 Go 核心类型
```go
type ProjectConfig struct { /* 对应上述 YAML，yaml tag + 校验：project_path 必须存在且为目录；max_file_size != 0；exclude_patterns 非空串 */ }
type FileInfo struct {
    Path, RelPath      string
    Language           string   // 由 langs.yml 映射，默认 "Text"
    SizeBytes, Lines   int
    Symbols            []Symbol // Mode3/4 时填充
}
type Symbol struct {
    Kind       string // func | method | class | struct | interface | trait
    Name       string
    Signature  string // 完整签名（Mode4）
    Line       int
}
type SecretFinding struct {
    File, Rule string
    Line       int
    Kind       string // password | api_key | private_key | token | db_url ...
    Masked     string // 掩码预览：sk-abc...****xyz（前端展示用）
}
type ExportOptions struct {
    Mode     string
    Redact   bool
    RedactCfg RedactionConfig
    SplitTokens int
    FileOverrides []string // GUI 勾选覆盖配置
}
```
---
## 6. 后端 API（Wails App 方法与事件）
```go
type App struct{ ctx context.Context }
// 项目管理
func (a *App) ListProjects(projectsDir string) ([]ProjectSummary, error)
func (a *App) LoadProject(configPath string) (*ProjectConfig, error)
func (a *App) SaveProject(cfg ProjectConfig) error            // 写回 YAML
func (a *App) DeleteProject(configPath string) error
// 扫描与预览
func (a *App) ScanProject(cfg ProjectConfig) ([]FileInfo, error) // 文件树+大小+语言，不读内容
func (a *App) Estimate(cfg ProjectConfig, opt ExportOptions) (ExportEstimate, error)
// ExportEstimate: {FileCount, TotalChars, EstTokens, WillSplit, Parts}
// 敏感信息（F99）
func (a *App) ScanSecrets(cfg ProjectConfig, customPatterns []string) ([]SecretFinding, error)
func (a *App) SaveAllowlist(configPath string, allowlist []string) error
// 导出
func (a *App) RunExport(cfg ProjectConfig, opt ExportOptions, taskID string) (ExportResult, error)
// ExportResult: {OutputPaths []string, TotalChars, EstTokens, RedactedCount, DurationMs}
// 历史
func (a *App) GetHistory() ([]HistoryEntry, error)            // 存 output/.history.json
// 事件（EventsEmit，前端 On 监听）
// "export:progress"  {taskID, phase, current, total, currentFile}
// "export:done"      {taskID, result}
// "export:error"     {taskID, message}
```
---
## 7. 核心模块设计
### 7.1 扫描模块（`core/scanner`）
移植 `file_processor.rs` 语义：
1. 指定 `files` 逐个读取；
2. `directories` 递归遍历，`exclude_directories` 命中则**整目录剪枝**；
3. 每文件过 `shouldIncludeFile`：大小限制 → `exclude_patterns`（含 `*`/`?` 走 `path.Match` glob，否则前缀/相等匹配）；
4. 并行读取：`runtime.NumCPU()` worker，`chan string` 结果收集；单文件读失败 `log.Warn` 并跳过（不中断）。
5. 结果按 `RelPath` 排序（对齐旧版 `sort_by(|a,b| a.0.cmp(&b.0))`）。
### 7.2 符号提取（F3，`core/symbol`）
```go
type Extractor interface {
    Languages() []string                 // 如 ["go"]
    Extract(content string) ([]Symbol, error)
}
var registry = map[string]Extractor{}   // 按语言注册
```
**v1 用正则提取器**（每语言一份，多行匹配），覆盖：Go / Python / JS / TS / Java / Rust / C / C++ / C#。示例：
| 语言 | 目标 | 正则要点 |
|---|---|---|
| Go | func/method | `^func\s+(?:\(([^)]+)\)\s*)?([A-Za-z_]\w*)\s*\(([^)]*)\)\s*(\{|\(([^)]*)\))?` |
| Python | def/class | `^\s*(?:async\s+)?def\s+(\w+)\s*\(([^)]*)\)|^\s*class\s+(\w+)` |
| JS/TS | function/class/method | `^\s*(?:export\s+)?(?:async\s+)?function\s+(\w+)|^\s*(?:export\s+)?class\s+(\w+)|^\s{2,}(?:async\s+)?(\w+)\s*\([^)]*\)\s*\{` |
| Java/C# | class/method | `(?:public|private|protected|static|final|synchronized|\s)+[\w<>\[\]]+\s+(\w+)\s*\([^)]*\)\s*\{` |
| Rust | fn/struct/trait/impl | `^\s*(?:pub\s+)?(?:async\s+)?fn\s+(\w+)|^\s*(?:pub\s+)?(?:struct|trait|enum)\s+(\w+)` |
**v2 升级路径**：接入 `tree-sitter`（`smacker/go-tree-sitter` 各语言 grammar），接口不变，仅替换实现。文档标注：正则版允许漏报，不要求 100% 准确（用途是给 LLM 提供目录）。
### 7.3 敏感信息过滤引擎（F99，`core/redactor`）★重点
**流程**：`扫描（ScanSecrets）→ GUI 报告确认 → 导出时替换`。两阶段设计，避免静默误伤代码。
#### 内置规则表（RE2 语法，可直接实现）
| Kind | 规则名 | 正则 |
|---|---|---|
| password | generic-password-assign | `(?i)(password\|passwd\|pwd\|secret\|secret[_-]?key\|api[_-]?key\|apikey\|access[_-]?key\|client[_-]?secret\|auth[_-]?token\|credential)[a-z0-9_-]*\s*[:=]\s*["']?[^"'\s,;}{)<>]{4,}` |
| api_key | aws-access-key | `AKIA[0-9A-Z]{16}` |
| api_key | aws-secret-key | `(?i)aws.{0,20}secret.{0,20}[:=]\s*["']?[A-Za-z0-9/+=]{40}` |
| token | github-pat | `gh[pousr]_[A-Za-z0-9]{36,255}` |
| api_key | openai-style | `sk-[A-Za-z0-9_-]{20,}` |
| token | slack-token | `xox[baprs]-[A-Za-z0-9-]{10,}` |
| token | jwt | `eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.?[A-Za-z0-9_-]*` |
| private_key | pem-block | `-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----` |
| db_url | connection-string | `(?i)(mysql\|postgres(ql)?\|mongodb(\+srv)?\|redis\|amqp\|mssql)://[^\s:@/"']+:[^\s@/"]+@[^\s"']+` |
| password | htpasswd-like | `(?i)(Basic\|Bearer)\s+[A-Za-z0-9+/=]{16,}` |
#### 误报抑制（默认忽略以下命中）
```
值包含: your_|xxx|example|changeme|placeholder|dummy|test123|<your|${...}|{{...}}|process.env|os.Getenv|import.meta.env
整行含 env 引用（取值来自环境变量而非硬编码）
```
用户在 GUI 报告页可将某条命中标记为误报 → 写入配置 `redaction.allowlist`（key = `文件相对路径:行号:规则名`），再次导出时跳过。
#### 替换策略
- `placeholder`（默认）：命中值替换为 `[REDACTED]`，保留代码结构。PEM 块整体替换为 `-----BEGIN PRIVATE KEY-----[REDACTED]-----END PRIVATE KEY-----`。
- `drop_line`：整行删除（配置项）。
- 每次导出统计 `RedactedCount`，写入结果与日志（只记数量与规则名，**不记录明文**）。
- 高熵随机串检测列为 v2 可选规则（进阶，不阻塞 M4）。
### 7.4 Markdown 生成（`core/generator`）
**统一骨架**（双语，沿用 `localized_text`）：
```markdown
# 项目文档 for {project_name}
## 项目文件树
```
{tree}
```
## {后续章节，按 Mode 变化}
```
| Mode | 章节 | 内容 |
|---|---|---|
| `full`（F1） | `### 文件: path` | 逐文件 ` ```lang \n content \n``` `，与旧版输出**逐字节对齐**（回归测试保证） |
| `files`（F2） | `### 文件清单` | 树 + 平铺相对路径列表 + 每文件行数/大小 |
| `symbols`（F3） | `### 文件: path` | 文件下缩进列出 `Symbol`：`- [func] HandleRequest (line 42)`，按行号排序 |
| `signatures`（F4-建议） | `### 文件: path` | 同上但输出完整签名（参数/返回值），给 LLM 做 API 概览 |
| `custom` | 模板渲染 | 见 F4-c |
### 7.5 Token 估算与分片（F4）
- `EstimateTokens(text)`：中文按 1 字 ≈ 1 token，ASCII 按 4 字符 ≈ 1 token，混合分段计算；文档标注精度 ±20%，仅用于容量规划。
- `SplitTokens > 0` 时：按文件为最小单位贪心装箱；产出 `xxx.part1.md ... xxx.partN.md`；每个分片头部注入「第 x/N 部分 + 本部分包含的文件清单」，末尾注明未完待续。
---
## 8. 前端设计
### 8.1 页面与交互流
**① 项目管理页**：卡片列表（名称/路径/模式/语言）；新建/编辑抽屉表单（校验规则同后端 §5.1）；路径用 Wails `Runtime.BrowserOpenURL` 辅助或手输。
**② 导出向导（核心页）**，分 4 步：
1. **选项目**（可多选，多选 = F1 的"合并导出"，各项目章节拼接）；
2. **选范围**：可勾选文件树（`FileTreeCheckable`，来自 `ScanProject`），覆盖配置；
3. **选选项**：导出模式（5 个 Mode 单选卡）；敏感过滤开关 + 「先扫描」按钮 → 内嵌 SecretReportView 预览命中数（"发现 7 处疑似密码，点击查看"）；Token 分片输入框 + 实时 `Estimate` 预估（文件数 / 字符 / ≈Token / 预计分片数）；
4. **执行**：进度条监听 `export:progress`；完成后显示输出路径 + 「打开文件」「复制到剪贴板」按钮。
   **③ 敏感扫描报告页**：表格（文件 | 行 | 类型 | 掩码预览 | 规则）；操作：✅ 确认为敏感（默认）/ 🚫 标记误报（写 allowlist）；顶部统计按 Kind 分组。
   **④ 历史页 / ⑤ 设置页**：历史 = 时间、项目、模式、输出、token、redacted 数；设置 = 全局默认排除规则、全局自定义正则、输出目录、界面语言。
### 8.2 状态
```ts
// stores/export.ts
{ selectedProjects: string[], files: FileInfo[], checkedPaths: Set<string>,
  mode: 'full'|'files'|'symbols'|'signatures'|'custom',
  redact: boolean, secretFindings: SecretFinding[], estimate: ExportEstimate | null }
```
---
## 9. F4 其他功能设计（建议清单，按优先级）
| 优先级 | 功能 | 说明 |
|---|---|---|
| P0 | **签名导出** `signatures` | F3 的自然延伸：文件名+完整函数签名，最适合让 LLM 快速理解项目 API 面 |
| P0 | **Token 预估与分片** | 超 LLM 上下文自动切分（§7.5） |
| P1 | **敏感扫描报告** | F99 的可视化确认环节（§8.1-③） |
| P1 | **多项目合并导出** | F1 已要求"所有项目"，向导中多选即可实现 |
| P1 | **CLI 无头模式** | `project-docs --cli --config x.yml --mode full`，供脚本/CI 复用 |
| P2 | **自定义模板** | `templates/*.tmpl`（Go text/template），变量：`{{.Tree}}` `{{.Files}}` `{{.Symbols}}` |
| P2 | **导出历史** | JSON 落盘 `output/.history.json` |
| P2 | **文件级 Token 统计列** | 文件树每节点显示 ≈Token，辅助手动裁剪上下文 |
| P3 | **快照 Diff 导出** | 两次导出中间件缓存 RelPath+hash，生成"变更文件+变更代码"markdown，供 LLM 做增量分析 |
| P3 | **粘贴板直出** | 导出完成一键复制全文（≤1MB 时），省去中转文件 |
---
## 10. 开发里程碑与验收标准
| 里程碑 | 内容 | 验收标准 |
|---|---|---|
| **M1 骨架** (2d) | Wails 初始化、目录结构、config/scanner/logger 移植 | 用原 `epub_reader.yml` 扫描结果与 Rust 版文件清单一致 |
| **M2 导出核心** (2d) | generator 三个 Mode 前置（full/files/symbols 占位）+ tree.go + 进度事件 | `full` 模式输出与旧版 `markdown_my_project.md` **diff 为空**（不含新增字段时） |
| **M3 GUI 基础** (3d) | 项目管理页 + 导出向导 4 步流程 + 进度条 | 全流程无头到 GUI 跑通 |
| **M4 符号提取** (2d) | 9 语言正则提取器 + symbols/signatures Mode | 对样例文件提取准确率 ≥90%，漏报不崩溃 |
| **M5 敏感过滤** (3d) | 规则引擎、扫描报告页、allowlist、替换策略 | §7.2 每条规则各 1 正例 1 负例单测全过；导出文件中 grep 不到明文密钥 |
| **M6 进阶** (2d) | Token 估算/分片、模板、历史、CLI | 10 万 token 项目正确切成 N 片且无文件被截断 |
| **M7 打包** (1d) | `wails build`、图标、安装包、README | Win/mac 产物可运行 |
---
## 11. 测试计划
1. **Redactor 单测**：每条规则正/负例 + 误报抑制用例（`process.env` 引用不命中）+ allowlist 生效用例。
2. **符号提取单测**：每语言 fixture 文件 + 断言提取出的 Symbol 列表。
3. **回归测试**：旧版 YAML（去掉 v2 字段）→ 对比 Rust 版输出，要求 byte-equal。
4. **分片边界**：恰好等于/超出 limit、单文件超 limit（该文件独占一片）。
5. **并发扫描**：1000+ 文件目录无 goroutine 泄漏、错误文件不中断。
---
## 12. 风险与注意事项
- **正则符号提取会漏报** → 文档明示为"目录级参考"，v2 换 tree-sitter；UI 上对 symbols/signatures 模式标注"基于启发式解析"。
- **Redactor 可能误伤示例代码** → 两阶段确认流（先扫描后导出）+ allowlist 是硬需求，不可省略为静默替换。
- **RE2 限制**：规则表已规避反向引用；用户自定义规则需校验 `regexp.Compile` 失败时给前端明确报错。
- **大文件**：`full` 模式拼单文件可能 >100MB → 前端预估页强制提示，默认启用分片。
- **路径跨平台**：扫描结果统一用 `/` 相对路径落盘，Windows 读取时 `filepath.FromSlash`。
---
*附录：本规格中所有正则、YAML 字段、API 签名均可直接作为 AI Coding 的实现依据；实现时如遇冲突，以「回归测试 byte-equal」与「敏感信息不落盘」两条为最高优先级原则。*
``