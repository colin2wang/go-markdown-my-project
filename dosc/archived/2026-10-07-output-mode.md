# 符号目录 & 签名导出 — 概要设计
> 对应工作台两种导出模式：`symbols`（符号目录）与 `signatures`（签名导出）。目标：为 LLM 提供项目的**结构地图**与 **API 面貌**，以极小 token 消耗替代全文阅读。v1 采用「行状态机 + 正则」启发式提取，不引入 tree-sitter（v2 升级路径）。

## 实现状态（2026-10-07）
- ✅ **S1** 数据模型 + 提取器接口 + 状态机框架：9 语言（Go/Rust/Python/Java/C#/JS-TS/Dart/C/C++，+Shell），含花括号深度计数、Parent 栈、Python 缩进栈、注释过滤、关键词排除。见 `core/symbol/`。
- ✅ **S2** 语言规则表 + 注释过滤 + 关键词排除（见 §4，Dart 已补齐）。
- ✅ **S3** `generator` 两套模板（`generateSymbols`/`generateSignatures`）+ 子选项（`include_line_numbers`/`max_signature_len`）+ 双语标题；`app.RunExport` 透传；前端 `WorkbenchView.vue` 模式卡片 + 子选项 UI。`generateFull` 未改动，黄金快照不受影响。
- ⚠️ **S4** 缓存 / 降级 / 超时：仅做了基础降级（未知语言返回空、无符号文件跳过）；会话内缓存、单文件 >2s 超时、>1MB 跳过尚未实现。
- ⚠️ **§7 Estimate / token 徽标 / 进度 phase**：未接入（导出可正常工作，但右栏大纲仅显示章节标题，未回填每文件符号数）。
- ⚠️ **§5 签名规范化** 为 v1 简化版：多行拼接 + 按 `sigEnd` 截断 + `MaxSignatureLen` 截断，未做尾 `/* truncated */` 标记（改为 ` …`），struct 字段提取列为 v2。
- ⚠️ **§6.2 `impl` 分组行**（`pub impl Config`）暂未渲染（impl 仅用于方法 Parent 归属），列为 v2。
---
## 1. 功能定位与对比
| | 🔎 符号目录 symbols | ✍️ 签名导出 signatures |
|---|---|---|
| 回答的问题 | “这个项目由哪些模块/类/函数组成？” | “这些函数怎么声明、如何调用？” |
| 输出粒度 | 符号类型 + 名称 + 行号 | 完整签名（参数、返回值、修饰符） |
| 每符号成本 | ≈10 tokens | ≈25–40 tokens |
| 典型用量 | 200 符号项目 ≈ 2k tokens | 同项目 ≈ 7k tokens |
| 适用场景 | 项目概览、找代码位置 | API 分析、生成调用代码、review 接口设计 |
| 后端实现 | 提取器产出 `Symbol`，仅用 Kind/Name/Parent/Line | 同上，额外用 Signature 字段 |
两者共用同一提取引擎，只是**渲染模板不同**——这是核心设计决策：提取一次，渲染两种。
---
## 2. 数据模型与接口
```go
// core/symbol/model.go
type Kind string
const (
    KindFunc      Kind = "func"      // 顶级函数
    KindMethod    Kind = "method"    // 类/结构体/impl 内的方法（含构造器）
    KindClass     Kind = "class"
    KindStruct    Kind = "struct"
    KindInterface Kind = "interface"
    KindTrait     Kind = "trait"
    KindEnum      Kind = "enum"
    KindType      Kind = "type"      // 类型别名等
)
type Symbol struct {
    Kind      Kind
    Name      string
    Parent    string // 所属 class/struct/impl 名；顶级符号为 ""
    Signature string // 单行规范化签名（signatures 模式用；symbols 模式留空可省计算）
    Line      int    // 1-based 声明行
}
type Extractor interface {
    Languages() []string            // 对应 langs.yml 的语言名，如 ["go"]
    Extract(src []byte) ([]Symbol, error)  // 返回按 Line 升序
}
```
注册表按语言名索引；`FileInfo.Language`（来自 langs.yml 映射）查不到提取器 → 该文件无符号（见 §10 降级）。
配置追加（§9 完整字段）：
```yaml
symbols:
  include_line_numbers: true
  max_signature_len: 200
  include_doc_comments: false   # v2
```
---
## 3. 提取引擎架构
```
src []byte ──► LineScanner（逐行）
                 │
                 ├─ ① 注释/字符串粗过滤（语言相关行注释前缀、块注释状态位）
                 ├─ ② 花括号深度计数 depth（判"顶级 or 类内"）
                 ├─ ③ 上下文栈：当前所属 class/struct/impl（Python 用缩进，其余用 depth+最近类型声明）
                 └─ ④ 逐条规则正则匹配 → Symbol{Parent=栈顶}
```
**三个关键机制**：
1. **括号深度计数**：维护 `depth`，遇 `{` +1、`}` −1（粗略跳过字符串/字符字面量内的大括号，启发式即可）。`depth == 0` 的匹配为顶级符号；`depth == 1` 且栈顶是类型声明 → `Kind=method, Parent=栈顶名`。
2. **上下文栈**：匹配到 class/struct/trait/impl/enum 声明时压栈（记录其声明 depth），对应 `}` 出栈。Rust 的 `impl X` / `impl Trait for X` → 压 `X`；Java/TS 的 `class X {` → 压 `X`。
3. **Python 缩进栈**：以行首空格数代替 depth，维护 `(indent, name)` 栈；`def` 的 parent = 栈中最近一个 `indent < 当前行 indent` 的 class。
   每个提取器 = 一个「规则表 + 钩子」的声明式结构，避免每语言手写状态机：
```go
type rule struct {
    re      *regexp.Regexp
    kind    Kind
    parent  parentMode // none | fromReceiver | fromStack | indentStack
}
type lineExtractor struct {
    lang       string
    rules      []rule
    lineComment []string   // "//", "#"
    blockComment [string, string] // 可选
    typeOpeners []rule      // 会压栈的类型声明
    sigEnd     byte        // '{' | ':'  签名截断符
}
```
---
## 4. v1 语言规则表（RE2 兼容）
覆盖 9 语言（含你们示例项目的 Dart/Rust）：
| 语言 | 类型声明（压栈） | 函数/方法 |
|---|---|---|
| **Go** | `^type\s+(\w+)\s+(struct\|interface)\s` → struct/interface | `^func\s+(?:\(([^)]+)\)\s*)?(\w+)\s*\(` ：receiver 组非空 → method+Parent=去 `*` 后的类型名；否则顶级 func |
| **Rust** | `^\s*(pub\s+)?(struct\|enum\|trait)\s+(\w+)` ；`^\s*impl(?:<[^>]*>)?\s+(?:\w+\s+for\s+)?(\w+)` → 压目标类型 | `^\s*(?:pub(?:\([^)]*\))?\s+)?(?:async\s+)?(?:unsafe\s+)?fn\s+(\w+)` ；栈顶为 impl → method |
| **Python** | `^(\s*)class\s+(\w+)` | `^(\s*)(?:async\s+)?def\s+(\w+)\s*\(` ；缩进栈定 parent |
| **Java** | `^\s*(?:public\|private\|protected)?\s*(?:static\|final\|abstract)*\s*(class\|interface\|enum\|record)\s+(\w+)` | `^\s+(?:public\|private\|protected\|static\|final\|synchronized\|default\s)*[\w<>\[\],.]+\s+(\w+)\s*\([^;)]*\)\s*\{` （depth≥1 → method） |
| **C#** | 同 Java 增 `record`、`partial` | 同 Java，排除 `if/for/while/switch/catch` 关键字词表 |
| **JS/TS** | `^\s*(?:export\s+)?(?:abstract\s+)?class\s+(\w+)` ；`interface\|type` → 不压栈直接产出 | function 声明 `^\s*(?:export\s+)?(?:default\s+)?(?:async\s+)?function\s*\*?\s*(\w+)\s*\(` ；类方法 `^\s{2,}(?:static\s+)?(?:async\s+)?(\w+)\s*\(.*\)\s*\{`（栈顶 class）；箭头函数 `^(?:export\s+)?(?:const\|let\|var)\s+(\w+)\s*=\s*(?:async\s*)?\(` → func |
| **Dart** | `^\s*(?:abstract\s+)?class\s+(\w+)`；`mixin\|extension\s+(\w+)` | `^\s{2,}(?:@override\s+)?(?:[\w<>,?\s]+\s+)?(\w+)\s*\([^)]*\)\s*(?:async\|sync\*)?\s*\{`；顶级函数 depth 0 同模式 |
| **C** | `^\s*typedef\s+struct[^{]*?(\w+)\s*;` → struct | `^[\w\*\s]+\s(\*?\w+)\s*\([^;]*\)\s*\{`（depth 0） |
| **C++** | C 规则 + `^\s*(?:class\|struct)\s+(\w+)` | C 规则（depth 0 或栈顶 class 内 → method），含 `~\w+` 析构 |
**关键词排除词表**（所有类 C 语言共用，防止 `if (...) {` 误报为 method）：
`if, for, while, switch, catch, return, else, do, try, function`。
**注释过滤**：行首 trim 后以 `//`、`#`（py/ruby）、`*`（JSDoc 延续行）、`--`（sql）开头 → 跳过；块注释 `/* */` 维护单比特状态。启发式，不追求完美（§10）。
---
## 5. 签名规范化（signatures 模式）
原始声明行常跨多行，按以下步骤归一为单行：
```
1. 起点 = 匹配行；终点 = 首个 sigEnd（'{' 或 Python ':'）出现的行
2. 跨行拼接，所有空白序列折叠为单个空格
3. 去掉尾部 sigEnd 与其后的空白
4. 长度 > max_signature_len(200) 时截断并追加 " …"
5. Python 保留尾 ':'；Rust 保留 "-> Ret"；Go 保留 "(r *T) Name(args) (ret)"
```
示例（Rust 多行声明）：
```rust
// 源码
pub fn load(
    path: &Path,
    opts: &LoadOptions,
) -> Result<Self, Error> {
// 规范化结果
pub fn load(path: &Path, opts: &LoadOptions) -> Result<Self, Error>
```
**排除项**：函数体不提取；struct 字段 v1 不提取（声明行本身即签名，如 `pub struct Config`）。字段提取列为 v2 增强。
---
## 6. 输出模板（最终 Markdown）
两模式共享骨架：`# 标题` + `## 项目文件树`（与 full 模式一致），第三节按模式切换。
### 6.1 symbols 模式 — 缩进树形
```markdown
## 符号目录（Symbol Directory）
### src/config.rs （4 个符号）
- [struct] Config · L34
  - [method] load · L56
  - [method] validate · L89
- [func] parse_config · L120
### src/main.rs （1 个符号）
- [func] main · L12
### assets/logo.svg （未识别到符号）
```
规则：
- 类型符号为父行，其方法缩进两格（Parent 链接）；无 Parent 的符号顶级平铺；全部按 Line 排序。
- 文件标题带符号计数，便于用户与 LLM 判断文件体量。
- 行号格式 `· L{n}`，`include_line_numbers=false` 时省略。
### 6.2 signatures 模式 — 语言代码块
```markdown
## API 签名（API Signatures）
### src/config.rs
```rust
pub struct Config                                        // L34
pub impl Config                                          // L52
    pub fn load(path: &Path, opts: &LoadOptions) -> Result<Self, Error>     // L56
    pub fn validate(&self) -> bool                                          // L89
pub fn parse_config(path: &Path) -> Result<Config, Error>                    // L120
```
### src/main.rs
```rust
fn main()                                                // L12
```
```
规则：
- 每文件一个 ` ```{language} ` 代码块；语言取 FileInfo.Language 小写映射（`Rust→rust`，未知→`text`）。
- **缩进表层级**：类型声明行顶格；其方法缩进 4 格（伪嵌套，非真实代码，仅视觉分组——LLM 理解无歧义）。
- 行号以 `// L{n}` 尾注释（该语言行注释前缀：py 用 `#`，其他用 `//`）。
- impl 声明本身作为分组行（`pub impl Config`），不重复其 trait 目标（v2 可加 `// for Trait`）。
- 签名超过 200 字符已截断的行尾追加 `/* truncated */`。
### 6.3 双语标题
沿用 `localized_text`：`符号目录/Symbol Directory`、`API 签名/API Signatures`、`个符号/symbols`、`未识别到符号/no symbols recognized`。
---
## 7. 与现有管道集成
```
工作台 Estimate ──┐
├─► core/scanner（收集 FileInfo）
执行导出 ─────────┤
└─► core/symbol（仅 symbols/signatures 模式触发）
│ worker pool 与扫描共用，每文件独立提取，panic recover
▼
core/generator ── 按模式选模板 ──► markdown
```
| 集成点 | 设计 |
|---|---|
| **估算 Estimate** | symbols/signatures 模式下先跑提取再估算（提取通常 <50ms/千文件），返回额外字段 `SymbolCount`；前端右栏大纲预览可显示每文件符号数 |
| **Token 徽标** | 文件树节点在 symbols 模式下徽标 = `符号数 × 10` 近似，signatures = `Σ签名长度/4`；避免为徽标提前提取全部文件 → 仅在 Estimate 返回后回填 |
| **进度事件** | `export:progress` phase 增加 `extracting 符号提取`，current/total 按文件计 |
| **缓存** | 会话内 map 缓存 `key = relPath + mtime + size`；重新导出同一项目命中缓存，无需二次提取 |
| **敏感过滤顺序** | 提取基于**原始源码**；symbols/signatures 模式输出不含函数体，天然低风险，但签名中可能含默认参数值（如 `pwd="123456"`）→ 仍走 redactor 管道，在**渲染后文本**上执行规则替换，顺序：`提取 → 渲染 → 脱敏 → 写盘`（与 full 模式一致，单一出口） |
---
## 8. 选项与 UI
中栏模式卡片选中 symbols/signatures 时，卡下方展开子选项（复用现有卡片区，不新增面板）：
```
┌────────────────────────┐
│ ◉ ✍️ 签名导出           │
│   文件名 + 函数完整签名  │
│  ┌──────────────────┐  │
│  │ ☑ 显示行号        │  │ ← 子选项，仅符号类模式显示
│  │ 截断长度 [200]    │  │
│  └──────────────────┘  │
└────────────────────────┘
```
右栏大纲预览对应变化：
```
── 内容预览 ──────────
# 项目文档 for …
├─ ## 项目文件树
└─ ## API 签名
├─ src/config.rs (4)
├─ src/main.rs (1)
└─ cargo.toml (无符号)
```
---
## 9. 配置字段汇总
```yaml
export_mode: symbols          # 或 signatures
symbols:
  include_line_numbers: true
  max_signature_len: 200
  include_doc_comments: false # v2 预留
```
校验：`max_signature_len ∈ [40, 2000]`。
---
## 10. 边界与降级
| 场景 | 行为 |
|---|---|
| 语言无提取器（如 yml、toml、svg） | symbols：文件标题 + `（未识别到符号）`；signatures：跳过该文件（避免空代码块），文件树章节仍保留其存在感 |
| 正则误报（字符串中的 `func`、DSL 代码） | 接受为启发式成本；UI 卡片固定显示“基于启发式解析”提示（已实现） |
| 提取 panic / 超时（单文件 >2s） | recover 后记 `log.Warn`，该文件按“未识别到符号”处理，不中断 |
| 超大文件（>1MB） | 跳过提取，同上降级 |
| Parent 栈错乱（源码本身括号不平衡） | 栈兜底：depth 回到 0 时清空类型栈 |
| 泛型声明跨行 `impl<T: Trait>` | 规则正则中 `[^>]*` 允许跨行尾部匹配失败 → 该行不识别，漏报可接受 |
---
## 11. 性能预算
| 项 | 目标 |
|---|---|
| 1000 文件 / 20 万行提取 | < 1s（8 worker 并行） |
| 单文件 | < 5ms / 千行 |
| 内存 | 符号结果 < 10MB（5000 符号量级） |
---
## 12. 测试计划
1. **单测（每语言一个 fixture）**：手写含「顶级函数 / 类与多方法 / 多行签名 / 字符串中的伪声明 / 注释中的伪声明 / 嵌套类」的样例文件，断言提取的 `[]Symbol` 精确相等（Golden 文件）。
2. **规范化单测**：多行签名、超长签名截断、Python 冒号结尾、Go 双返回值。
3. **父子归属专项**：Rust `impl Trait for X`、Python 缩进混用、TS 嵌套 class。
4. **幂等测试**：同输入两次提取结果一致（无共享可变状态）。
5. **端到端**：用当前 Rust 项目（截图中的 markdown_my_project）实测——预期 `src/main.rs` 产出 `main`，`tree_generator.rs` 产出其全部 pub fn，与人工核对 ≥90% 命中。
---
## 13. 里程碑
| 阶段 | 内容 | 验收 |
|---|---|---|
| S1 | Model + Extractor 接口 + Go/Rust 提取器 + 状态机框架 | 两语言 fixture 全过 |
| S2 | 其余 7 语言规则表 + 注释过滤 + 关键词排除 | 9 语言 fixture 全过 |
| S3 | generator 两套模板 + Estimate 集成 + 进度 phase | GUI 端到端导出、token 徽标正确回填 |
| S4 | 缓存 + 降级路径 + 超时保护 | 5000 文件项目 <5s，无 panic |
| v2 预留 | tree-sitter 替换、struct 字段提取、文档注释关联（`include_doc_comments`）、`Parent` 链多级 | 接口不变，仅注册新实现 |
---
*实现提示：`rule` 声明式结构 + 状态机框架（§3）是本设计的骨架，建议先写框架与 Go/Rust 两语言跑通端到端，再批量填语言规则表；所有正则放入 `rules_test.go` 的表驱动用例中维护。*
