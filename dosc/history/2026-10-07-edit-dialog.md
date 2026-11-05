# 首页「编辑项目」对话框重设计 — 所见即所得 + YAML 双栏编辑器
> 目标：将现有 3 步向导式对话框改为**左右分栏编辑器**。左侧为可视化表单（单一信息源），右侧为 `project.yml` 的**实时镜像预览**。左侧所有修改即时同步到右侧，用户能直观看到“表单 ↔ 配置文件”的对应关系，也为后续直接分享/手改 yml 打通。
---
## 1. 总体布局
```
┌──────────────────────────────────────────────────────────────────────────────┐
│ 编辑项目 · Markdown My Project                    [恢复默认]  [取消]  [💾 保存] │
├──────────────────────────────────────────┬───────────────────────────────────┤
│  左栏：可视化编辑（flex: 1.2）              │  右栏：YAML 预览（flex: 1）        │
│ ┌──────────────────────────────────────┐ │ ┌───────────────────────────────┐ │
│ │ ⬓锚点: 基本 · 范围 · 排除 · 导出 · 高级│ │ │ ● 校验通过 · 32 行 · 已同步 0.3s│ │
│ ├──────────────────────────────────────┤ │ │ [复制] [换行] [仅看差异]        │ │
│ │ 📦 基本信息                           │ │ ├───────────────────────────────┤ │
│ │  项目名称  [Markdown My Project    ]  │ │ │ 1  # Project Configuration    │ │
│ │  项目路径  [F:/…/markdown_my_project] │ │ │ 2  # Name of the project      │ │
│ │            [📂浏览…]  ● 目录有效      │ │ │ 3  project_name: "Markdown…"  │ │
│ │  输出文件  [markdown_my_project.md ]  │ │ │ 4  project_path: "F:/…"       │ │
│ │  文档语言  (●中文 ○English)           │ │ │ 5  output_file: "…"           │ │
│ ├──────────────────────────────────────┤ │ │ 6  markdown_lang: zh_cn       │ │
│ │ 🔍 扫描范围                           │ │ │ 7                             │ │
│ │  包含文件  [cargo.toml ×][log4rs.yml ×]│ │ │ 8  files:                     │ │
│ │            [+ 添加] [📂从磁盘选…]      │ │ │ 9    - cargo.toml             │ │
│ │  包含目录  [src ×][projects ×]        │ │ │ 10   - log4rs.yml             │ │
│ │            [+ 添加] [📂从磁盘选…]      │ │ │ …                             │ │
│ │  ⓘ 目录将递归收集全部文件              │ │ │ （语法高亮 + 行号 + 随输入滚动） │ │
│ ├──────────────────────────────────────┤ │ └───────────────────────────────┘ │
│ │ 🚫 排除规则                           │ │                                   │
│ │  排除目录  ["**/444" ×]  ⓘ支持 **/name│ │  ← 变更行高亮：本次会话中           │
│ │  排除规则  []           ⓘ支持 *.log   │ │    被修改过的 YAML 行淡黄底纹       │
│ │  大小上限  [1] [MB ▾]                 │ │                                   │
│ ├──────────────────────────────────────┤ │                                   │
│ │ 📤 导出设置                           │ │                                   │
│ │  导出模式  [完整代码 ▾]               │ │                                   │
│ │  Token 分片 [0]（0=不分片）           │ │                                   │
│ ├──────────────────────────────────────┤ │                                   │
│ │ 🛡 敏感过滤                           │ │                                   │
│ │  [✓] 导出时剔除密码/密钥              │ │                                   │
│ │  策略 [占位符替换▾] 替换为 [REDACTED] │ │                                   │
│ ├──────────────────────────────────────┤ │                                   │
│ │ 🔎 符号提取（符号/签名模式生效）       │ │                                   │
│ │  [✓]显示行号  截断长度 [200]          │ │                                   │
│ ├──────────────────────────────────────┤ │                                   │
│ │ 📊 预检                               │ │                                   │
│ │  [▶ 扫描预览] → 13 文件 · 32.2 KB     │ │                                   │
│ │     · 3 个 yml 无符号提取器(提示)      │ │                                   │
│ └──────────────────────────────────────┘ │                                   │
└──────────────────────────────────────────┴───────────────────────────────────┘
```
### 关键布局决策
| 决策 | 说明 |
|---|---|
| **向导改为单页分区** | 现 3 步向导（基本信息/选择范围/确认）改为单页滚动 + 顶部锚点导航。理由：双栏联动要求表单始终可见，“下一步”会打断左→右的同步心智 |
| **分栏比例** | 默认 `1.2 : 1`，中间分隔条可拖拽（双击复位），比例持久化 localStorage |
| **右栏只读为默认** | v1 右栏是**只读实时镜像**；v2 增加「编辑 YAML」开关实现双向同步（§7） |
| **对话框尺寸** | `max-width: 1200px; height: 82vh`，小屏 (<1100px) 自动退化为上下堆叠（表单在上、YAML 折叠为可展开条） |
---
## 2. 左栏字段规格（所见即所得）
### 2.1 分区与控件
| 分区 | 字段 | 控件 | 对应 YAML |
|---|---|---|---|
| 📦 基本信息 | 项目名称 | Input | `project_name` |
| | 项目路径 | **PathInput**：Input + `浏览…`（Wails 目录选择器）+ 异步校验徽标 | `project_path` |
| | 输出文件 | Input + 后缀校验 | `output_file` |
| | 文档语言 | Radio：中文 / English | `markdown_lang` |
| 🔍 扫描范围 | 包含文件 | **TagListInput** + `从磁盘选…`（多选文件对话框，自动转相对路径） | `files` |
| | 包含目录 | TagListInput + 目录多选对话框 | `directories` |
| 🚫 排除规则 | 排除目录 | TagListInput，占位提示 `支持 **/name` | `exclude_directories` |
| | 排除规则 | TagListInput，占位提示 `支持 *.log` | `exclude_patterns` |
| | 大小上限 | NumberInput + 单位下拉 (KB/MB) | `max_file_size` |
| 📤 导出设置 | 导出模式 | Select：完整代码/文件名清单/符号目录/签名导出/自定义模板 | `export_mode` |
| | Token 分片 | NumberInput，0=不分片，>0 时下方提示「预计 N 片」 | `split_tokens` |
| | 模板 | Select（读取 templates/ 目录）+ 管理链接 | `template` |
| 🛡 敏感过滤 | 启用开关 | Switch | `redaction.enabled` |
| | 策略 | Select：占位符替换/删除整行 | `redaction.strategy` |
| | 替换文本 | Input（仅 placeholder 策略显示） | `redaction.placeholder` |
| | 自定义规则 | TagListInput（每条即时 RE2 编译校验，失败红框） | `redaction.custom_patterns` |
| 🔎 符号提取 | 显示行号 | Switch | `symbols.include_line_numbers` |
| | 签名截断长度 | NumberInput [40, 2000] | `symbols.max_signature_len` |
| 📊 预检 | 扫描预览 | Button → 调 `PreviewScan(cfg)`，显示文件数/总大小/警告 | 只读，不落 YAML |
### 2.2 控件行为细节
**PathInput**：
- `浏览…` 调 Wails `OpenDirectoryDialog`，选中后回填。
- 输入防抖 500ms 调后端 `ValidatePath` → 徽标三态：`● 目录有效`(绿) / `● 路径不存在`(红，阻止保存) / `校验中…`(灰)。
- Windows 输入接受 `\` 或 `/`，保存时统一转 `/`。
  **TagListInput**：
- 回车/失焦/粘贴(批量按逗号换行拆分)添加；`×` 删除；重复项添加时自动忽略并 shake 提示。
- 与文件选择器联动：磁盘选中的绝对路径减去 `project_path` 前缀转相对路径；项目路径未设置时先提示。
  **敏感过滤联动**：关闭 `enabled` 时，策略/替换文本/自定义规则整体降透明度禁用（YAML 中仍保留结构，`enabled: false`）。
  **符号提取联动**：仅当 `export_mode ∈ {symbols, signatures}` 时该分区高亮，否则置灰并提示「仅符号类模式生效」。
### 2.3 预检（可选但推荐）
点击「扫描预览」调后端轻量扫描（只统计不读内容），在分区内展示结果卡片：
```
✓ 扫描完成：13 文件 · 32.2 KB
  ├ Rust ×7 · YAML ×5 · TOML ×1
  ⚠ 5 个 YAML/TOML 文件无符号提取器（符号目录模式将显示为空）
```
作用：在保存前就能发现「包含目录写错、排除规则过严」等问题，替代旧向导第 3 步「确认」页的职责。
---
## 3. 右栏 YAML 实时预览
### 3.1 渲染规格
- 组件：CodeMirror 6（yaml 模式，只读）或轻量方案：`highlight.js` + `<pre>`。推荐 CodeMirror，为 v2 双向编辑铺路。
- 显示行号、语法高亮、等宽字体 13px。
- **同步机制**：监听左侧表单 store（Pinia），`watch(draft, deep, { debounce: 300 })` → 序列化 → 更新右栏。右栏顶部显示 `已同步 x.xs`（距上次变更时间），同步中显示 spinner。
- **变更行高亮**：序列化前后做行级 diff（简单 LCS 即可），本次会话中改动过的行加淡黄背景，帮助用户建立“我改了哪个字段 → YAML 哪几行变了”的直觉。
### 3.2 注释保留（还原原版 yml 风格）
序列化不用 `yaml.Marshal(struct)`（会丢注释），而是用 `yaml.v3` 的 **Node API** 按模板注入注释，输出与原项目 yml 完全同风格：
```yaml
# Project Configuration for Project Documentation
# Name of the project
project_name: "Markdown My Project"
# Path to the project root directory
project_path: "F:/Workspaces/JetBrains/RustRover/markdown_my_project"
# Output file path for the generated documentation
output_file: "markdown_my_project.md"
# Markdown output language: "zh_cn" for Chinese, "en_us" for English
markdown_lang: zh_cn
# List of specific files to include
files:
  - cargo.toml
  - log4rs.yml
# ─── v2 新增字段（仅当非默认值时输出，保持 yml 简洁）───
export_mode: full
redaction:
  enabled: true
  placeholder: "[REDACTED]"
  strategy: placeholder
```
实现要点：
| 规则 | 说明 |
|---|---|
| 注释模板表 | `map[fieldKey]string` 维护每字段的英文注释（沿用原版文案），维护在 `core/config/comments.go` |
| 字段顺序固定 | 按 Node 构造顺序输出：`project_name → project_path → output_file → markdown_lang → files → directories → exclude_directories → exclude_patterns → max_file_size → v2 字段` |
| v2 字段省略策略 | `export_mode=full`、`split_tokens=0`、`redaction.enabled=true`(默认策略) 时**不输出**，避免新配置文件膨胀；用户改过才写入 |
| 空列表 | 输出 `files: []` 而非省略，明确语义 |
### 3.3 右栏工具栏与状态
| 元素 | 行为 |
|---|---|
| `● 校验通过` / `● 2 个错误` | 状态芯片：绿/红。错误时点击滚动到首个错误字段（左栏对应控件红框聚焦） |
| `复制` | 复制 YAML 全文到剪贴板（用户可直接粘贴去手建配置） |
| `换行` 开关 | 长路径是否 soft-wrap |
| 错误横幅 | 序列化失败（理论不会发生，防御性）或后端校验返回错误列表时，列出 `第 n 行风格的位置描述 + 字段名` |
---
## 4. 校验体系（左栏即时 + 保存时后端双重）
### 4.1 前端即时校验规则表
| 字段 | 规则 | 错误文案 |
|---|---|---|
| project_name | 必填、非纯空白、≤100 字符 | 「请输入项目名称」 |
| project_path | 必填；异步存在性校验；必须是目录 | 「路径不存在」「不是目录」 |
| output_file | 必填；扩展名 ∈ md/markdown/txt（html 给黄条警告不阻止） | 「必须为 .md 等文档格式」 |
| max_file_size | >0 | 「必须大于 0」 |
| exclude_patterns | 每条非空；含 `\` 提示转正斜杠 | 「排除规则不能为空」 |
| redaction.custom_patterns | 每条 RE2 可编译（前端 `new RegExp` 预检 + 后端编译复检） | 「正则无效: {错误信息}」 |
| symbols.max_signature_len | ∈ [40, 2000] | 「范围 40–2000」 |
| split_tokens | ≥0 | 「不能为负」 |
校验结果集中存 `errors: Record<field, string>`，分区标题旁显示错误计数徽标（如 `🚫 排除规则 (1)`），保存按钮在存在 error 时禁用并 tooltip 首个错误。
### 4.2 保存流程
```
点击[保存]
  → 前端校验通过？
    ✗ → 聚焦首个错误字段，return
  → 调后端 ValidateConfig(cfg)
    ✗ → 错误列表映射到字段，return
  → 重名检查（同目录下 project_name 重复 → 确认覆盖？）
  → SaveProject(cfg) 写盘 → toast「已保存」→ 关闭对话框
```
后端新增/复用 API：
```go
func (a *App) ValidatePath(path string) (bool, bool, error)  // exists, isDir
func (a *App) ValidateConfig(cfg ProjectConfig) []FieldError // 完整规则复检 + glob/RE2 编译
func (a *App) PreviewScan(cfg ProjectConfig) (ScanPreview, error) // {FileCount, TotalBytes, ByLang, Warnings}
func (a *App) SaveProject(cfg ProjectConfig) error
```
---
## 5. 状态管理与数据流
```ts
// stores/projectEditor.ts
export const useProjectEditor = defineStore('projectEditor', () => {
  const original = ref<ProjectConfig | null>(null)   // 打开时快照
  const draft    = ref<ProjectConfig | null>(null)   // 编辑中
  const dirty    = computed(() => JSON.stringify(original) !== JSON.stringify(draft))
  const errors   = ref<Record<string, string>>({})
  const yamlText = ref('')                           // 序列化产物
  const changedLines = ref<Set<number>>(new Set())   // 变更行高亮
  const preview  = ref<ScanPreview | null>(null)
  // draft 任何变化 → 300ms 防抖 → validate + serialize → yamlText/diff
  watch(draft, debounced(async () => {
    errors.value = validateLocal(draft.value)
    yamlText.value = await SerializeYAML(draft.value)   // 后端序列化，保证注释模板一致
    changedLines.value = diffLines(prevYaml, yamlText.value)
  }), { deep: true })
  function open(configPath?: string) { /* 新建: 空配置+默认值；编辑: LoadProject → 快照 */ }
  async function save() { /* §4.2 流程 */ }
  function reset() { draft = clone(original) }
})
```
要点：
- **单一数据源是 `draft`**，YAML 永远是派生物（v1 单向）。序列化放后端（Go 的 yaml.v3 Node 注释逻辑与 CLI/导出共用一份，避免前端手写 YAML 生成器）。
- 新建模式：`project_path` 未填时 TagListInput 的磁盘选择禁用；提供 `恢复默认` 把 draft 重置为新建默认值。
- 关闭守卫：`dirty && 点击取消/遮罩/Esc` → 确认弹窗「有未保存的修改」。
---
## 6. 与原 3 步向导的功能映射
| 原向导 | 新方案去向 |
|---|---|
| 第 1 步 基本信息 | 📦 基本信息分区 |
| 第 2 步 选择范围（语言过滤/全选/文件清单） | 🔍 扫描范围 + 🚫 排除分区；**原「语言：全部/按语言勾选」移除**——运行时勾选属于工作台文件树的职责，配置文件只管静态范围。若要保留，可在预检结果中提供「按语言排除建议」按钮，把某语言全部文件加入 exclude_patterns |
| 第 3 步 确认 | 📊 预检分区（扫描预览）+ 右栏 YAML 本身就是最终确认物 |
---
## 7. v2 预留：右栏可编辑（双向同步）
在右栏加 `🔒/✏️` 切换开关：
- **进入编辑态**：CodeMirror 可写，用户改 YAML。
- **失焦/点击「应用」**：后端 `ParseYAML(text) → ProjectConfig` → 成功则回写 `draft`（表单刷新）；失败则右栏内联标红错误行，不应用。
- **冲突策略**：编辑 YAML 期间左栏被改动 → 应用时提示「左侧表单也有改动，[以 YAML 为准] [放弃 YAML 编辑]」。
- 此模式下变更行高亮双向生效。
  v1 不实现，但组件选型（CodeMirror 6）与「序列化放后端、解析也在后端」的架构已为此铺平道路。
---
## 8. 组件清单
| 组件 | 职责 | 复用 |
|---|---|---|
| `ProjectEditorDialog.vue` | 外壳：分栏、拖拽分隔条、保存/取消、关闭守卫 | — |
| `EditorSection.vue` | 分区容器：标题、锚点 id、错误徽标、折叠 | 全分区复用 |
| `AnchorNav.vue` | 顶部锚点：点击滚动 + scroll-spy 高亮 | — |
| `PathInput.vue` | 路径 + 浏览 + 异步校验徽标 | 工作台设置页可复用 |
| `TagListInput.vue` | 标签增删改 + 批量粘贴 + 磁盘选择插槽 | 排除规则等 5 处复用 |
| `SizeInput.vue` | 数值 + 单位 | — |
| `YamlPreview.vue` | CodeMirror 只读渲染、行号、变更高亮、状态芯片、复制 | v2 升级为可编辑 |
| `ScanPreviewCard.vue` | 预检结果展示 | — |
---
## 9. 边界与异常
| 场景 | 行为 |
|---|---|
| 编辑已损坏/手改过的 yml | `LoadProject` 解析失败 → 打开对话框时黄条提示「原配置部分字段无法识别，已按默认值加载」，未识别字段在 YAML 预览中保留原样（序列化时 merge 未识别的原始行，防丢失） |
| 项目路径含中文/空格 | 正常支持；TagListInput 相对路径计算用统一 `/` 分隔 |
| project_path 与 files/directories 相对路径越界（`../`） | ValidateConfig 警告级错误：黄条「路径越出项目根目录」，不阻止保存（与旧版兼容） |
| 保存时 yml 被外部程序修改 | SaveProject 检测 mtime 变化 → 确认「文件已被外部修改，覆盖？」 |
| 对话框宽度 <1100px | 上下堆叠布局，YAML 区默认折叠为一条 `≡ 实时预览 YAML (32 行)` 可展开 |
| 大量标签（50+ 排除规则） | TagListInput 超过 20 条折叠为 `+42 更多…`，点击展开 |
---
## 10. 验收清单
- [ ] 左栏任意字段修改后 ≤0.5s 右栏 YAML 更新，注释风格与原版 yml 一致
- [ ] v2 新字段按「非默认值才输出」规则序列化，新建项目默认 YAML 与原版字段集一致
- [ ] 变更行有高亮；复制按钮可取到完整 YAML
- [ ] 路径校验三态徽标工作正常；无效路径时保存禁用
- [ ] 自定义正则非法时该标签红框、保存禁用、错误信息含 RE2 原始报错
- [ ] 预检返回文件数与工作台扫描一致；含无提取器语言的警告
- [ ] dirty 状态下取消/Esc 有确认弹窗；保存成功后对话框关闭且项目卡片列表刷新
- [ ] 手改过的 yml 含未知字段时，保存后未知字段不丢失
- [ ] 分栏拖拽、锚点导航、小屏退化布局均正常
---
*实现提示：先落「外壳 + 单一数据源 + 后端 SerializeYAML（含注释模板）+ 只读预览」这条主干（约 60% 工作量即可跑通核心体验），再补 TagListInput 磁盘联动、预检、错误徽标等增强。序列化与解析务必只在 Go 侧实现，前端不写 YAML 生成逻辑。*
