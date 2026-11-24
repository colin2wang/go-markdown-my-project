# AGENTS.md — AI 编程代理指南

Go + Wails v3 桌面应用（GUI）+ Vue3/TS 前端，把项目源码聚合为 Markdown 供 LLM 分析。由 Rust 项目 `markdown_my_project` 重写而来。

## 常用命令

```powershell
.\build.ps1                # 一键构建 → build/bin/（wails3 task build：genicon → 图标 → 绑定 → pnpm build → go build → 拷 config）
.\build.ps1 -Clean         # 先清空 build/bin/（exe 被占用时先关掉正在运行的 GUI）
go build ./...             # 仅编译后端
go test ./core/...         # 全部单测（generator/symbol/redactor/token/compress/gitignore）
go test ./core/generator/ -update   # 重新生成黄金快照（仅在有意变更输出格式时）
wails3 dev                 # GUI 热重载开发
cd frontend && pnpm dev    # 纯前端开发
.\generate-icon.ps1        # 图标三步链：genicon(SVG→appicon.png) → wails3 generate icons(ico/icns) → wails3 generate syso
```

**构建版本戳**：build.ps1 按**本地时区**的构建时间生成 `YYYY-MM-DD_HH:mm:ss`（`[DateTime]::Now`，勿用 UTC；值含空格会被 PS→Task→go 传参链截断，故用下划线占位；构建日志另打印完整本地时间与 UTC 偏移），以**环境变量 `BUILD_TIME`** 传给 `wails3 task build`（`wails3 task` 不支持命令行变量覆盖），Taskfile.yml 读取该变量并以 `-ldflags -X` 注入 `app.BuildVersion`（`app/version.go`，未设置时回退 `dev`），展示层用 `app.FormatBuildVersion()` 格式化为 `(build YYYY-MM-DD HH:mm:ss)`（窗口标题 + 前端 `GetVersion` 绑定）；本地 `go build` / `wails3 dev` 下为 `dev`。**Windows 文件属性**：`build/windows/info.json` 是 VERSIONINFO 元数据的唯一来源（文件/产品版本、产品名、描述、版权、原始文件名等），Taskfile 与 `generate-icon.ps1` 的 `wails3 generate syso` 必须带 `-info build/windows/info.json`；旧 v2 模板占位符 `{{.Info.*}}` 不会被 Wails v3 渲染，留着会导致「详细信息」全空。**Wails v3**（自 v2 迁移，wails3 CLI 独立于旧 wails）：`main.go` 用 `application.New` + `Services` + `Window.NewWithOptions`（默认可见，不设 `Hidden`），窗口由 Wails 主循环启动后显示，`ServiceStartup` 不做窗口操作；事件经 `a.app.Event.Emit("app:log", e)`，目录/文件对话框用 `a.app.Dialog.OpenFile()` 链式 API；前端绑定由 `wails3 generate bindings` 生成到 `frontend/bindings/`（Taskfile 的 generate:bindings 任务），运行时用 npm 包 `@wailsio/runtime`（`Events.On`），旧的 `frontend/wailsjs/` 已删除。

## 架构边界（必须遵守）

- `core/` 各包是纯 Go，**禁止 import Wails**；进度/事件抽象在 `app/` 绑定层适配。新逻辑放 `core/`，`app/app.go` 只做薄转发。
- 新增的 `core/infer`（按项目结构推断推荐配置）、`core/pick`（文件重要性评分/预算裁剪）、`core/compress`（注释剥离 `StripComments` / 骨架化 `Skeleton` / 长行截断 `MinifyWith`）、`core/gitignore`（递归查找项目内所有 `.gitignore`、解析 Git 通配规则、`Matcher.Ignored` 判定路径忽略）、`core/wordfilter`（导出敏感词过滤 `Filter`：替换 XSS 测试样本、危险协议、恶意示例域名与攻击性描述词为占位符，规则表内置于 `words.go`，RE2 正则，与密钥脱敏 `core/redactor` 互不相干）均为纯 Go，不依赖 Wails，可被复用与单测；**注释剔除的语法表与状态机只放在 `core/compress/comments.go` 一个文件**，其它包不得另写一套注释规则。
- 绑定层方法必须可导出（`ServiceStartup` 而非 `serviceStartup`），否则 Wails 不生成前端绑定。
- 前端通过 `frontend/src/api/bindings.ts` 手写封装访问 `frontend/bindings/` 下生成的绑定模块（App 服务在 `app` 包下），新增后端方法需重跑 `wails3 generate bindings` 并同步在此补类型。
- **窗口初始化**：`main.go` 直接以默认可见方式创建窗口，由 Wails v3 在主循环启动后正常显示；不要在 `ServiceStartup` 中 `Run()`/`Show()`——v3.1.0 在主循环启动前 `Run()` 会卡在 WebView2 初始化（窗口白屏且“未响应”），`Show()` 在窗口未 `Run()` 时也不会清掉 `Hidden`（进程在跑但无窗口）；窗口尺寸为 `main.go` 的固定值（1400×800），已移除启动时按屏幕分辨率自动调整的逻辑。

## 协作约定

- **不要主动编译和验证**（`go build`、`go test`、`vue-tsc`、`pnpm build` 等一律不跑），用户自己做；仅当用户明确要求时才执行。
- **Commit message**：英文、Conventional Commit（`type(scope): summary`）、祈使语气、subject ≤72 字符、无句尾句号；一次提交只含一个逻辑变更。待提交的 message 文本写入 `COMMIT.md`。

## 项目特有注意事项

- **黄金快照回归**：`core/generator/generator_test.go` 用 `testdata/golden.md` 锁死 `full` 模式输出；另一测试与旧版 `Example.md` 做结构对比（CRLF 归一化、容忍快照后新增的文件）。改动生成逻辑导致快照失败时，先确认是预期变更再 `-update`。
- **byte-equal 优先级**：`full` 模式输出须与旧 Rust 版逐字节一致——文件标题路径用**原生分隔符**（`relPathNative`），树形字符 `├── └── │` 与缩进逐字符对齐，勿"顺手改成 /"。
- **敏感信息最高原则**：redactor 采用两阶段（扫描报告→确认→导出替换），日志只记数量与规则名，**绝不落盘明文**；`allowlist` key 格式为 `相对路径:行号:规则名`。命中抽屉（SensitiveDrawer）显示原始明文与统一替换值，属用户明确要求的内存内展示——明文仅经 `SensitiveHit.Value` 传给前端抽屉，不写盘、不进日志。规则定义唯一入口在项目配置「敏感」页（enabled / strategy / placeholder / `rule_profile` / custom_patterns / allowlist），`rule_profile` 取值 `strict | standard | relaxed`（空 = standard；relaxed 剔除 generic-password-assign / connection-string / htpasswd-like 三条易误报规则，筛选在 `core/redactor` 的 `allRules`）；是否脱敏由 `cfg.Redaction.Enabled` 决定（`ExportOptions` 已无 `Redact` 字段），`ScanSensitive` 与导出共用 `redactorConfig(cfg.Redaction)`，保证扫描报告与导出规则集一致。工作台只保留扫描按钮 + 命中展示（enabled 项目进入工作台自动扫描一次），不再提供开关/策略/占位符控件，也没有"未扫描"导出确认弹窗；新建项目默认 `enabled: true` + `standard`。另有独立于密钥脱敏的「敏感词过滤」（`core/wordfilter.FilterWithStats`，`Filter` 为 standard 档默认包装）：规则分三档 basic/standard/strict（`Config.Profile`，空 = standard；strict 含宽匹配 `\bon[a-z]{4,}\s*=` 易误伤配置键，仅显式选择），`Config.Strategy` 支持 `destroy`（默认，整体替换占位符）与 `neutralize`（尖括号转全角，保留代码可读）。启用条件为项目配置 `word_filter.enabled` 或工作台勾选 `ExportOptions.WordFilter`（运行时覆盖）；档位/策略配置在项目编辑器「词过滤」分区（`WordFilterConfig`，profile/strategy 非默认值才写 YAML，校验在 `app` Validate），命中计数逐规则返回并在 `ExportResult.FilteredCount` 上报（结果卡片「过滤」行 + 日志 `log.wordFilterApplied`）；管道位置约定固化在 `RunExport` 注释：wordfilter 必须在一切 MaxLineLen 截断之前、token 统计之前；幂等有结构性保证（`TestReplacementsAreInert` 锁定替换文本不被任何规则再次命中），另有 cheat-sheet 载荷全歼测试 `TestNoLivePayloadSurvives`；规则表与唯一实现都在 `core/wordfilter/words.go`。
- **pnpm 12**：构建脚本白名单在 `frontend/pnpm-workspace.yaml` 的 `allowBuilds`（package.json 里旧 `pnpm` 字段已失效）；build.ps1 用 ASCII 输出，PowerShell 5.1 下非 BOM 中文会解析失败。
- **路径约定**：落盘/展示用 `/` 相对路径；仅 `full` 模式的文件标题保留原生分隔符（见上）。
- **正则限制**：所有规则用 RE2（Go `regexp`），不支持反向引用；symbol 提取器为逐行正则（允许漏报），Go 方法规则靠 `nameGroup: 2` 取名。
- `config/` 是 GUI 运行时配置来源（`config/langs.yml` 语言映射 + `config/projects/` 示例配置 + `config/exclude_defaults.yml` 全局默认排除，自动与项目 exclude_* 合并去重，项目内 `use_default_excludes: false` 可整体关闭；项目内 `skip_binary_files` 按内容跳过二进制（前 8KB 含 NUL），默认开启、仅显式 false 时输出；`config/global_settings.yml` 全局设置（GUI 设置对话框读写）：`output_encoding`（utf8_bom | utf8 | gb18030，导出编码，项目级 `write_bom` 显式设置时优先于全局）与 `interface_language`（zh_cn | en_us | ja_jp，设置对话框不再提供「跟随系统」，旧配置 auto 由前端按当前语言回显）；导出编码解析优先级为「项目 write_bom 显式设置 > 全局 output_encoding > 默认 utf8_bom」，编码转换在 `generator.WriteOutput` 落盘层，`Generate` 内存输出与 `full` 模式黄金快照不受影响，GB18030 转换失败回退 UTF-8+BOM；`log4rs.yml` 与 `cmd/` CLI 已删除，仅保留 GUI；原 `assets/` 已并入 `config/`）；build.ps1 拷 `config/` 到 `build/bin/`（先删旧目录再拷，避免 Copy-Item -Recurse 嵌套成 config/config）；GUI 默认路径 `config/projects` 相对 exe 工作目录。
- **.gitignore 过滤**：项目配置 `use_gitignore`（三态 `*bool`，nil = true，GUI 默认勾选；`SerializeWithComments` 恒输出该字段，勾选写 `true`、取消勾选写 `false`）由 `app.scanOpts` 单点注入 `scanner.Options.UseGitIgnore`，覆盖 `PreviewScan` / `ScanProject` / `ScanSensitive` / 导出四条链路；`core/gitignore.Load` 递归查找项目内所有 `.gitignore`（跳过 `.git` 目录与已被上层规则忽略的目录），规则按目录深度 + 行序排序（后写/子目录优先），支持注释、`!` 否定、尾部 `/` 仅目录、前导 `/` 锚定、`**` 跨层与 `\#` / `\!` 转义；`core/scanner` 在目录剪枝（`Ignored(rel, true)`）与文件判定（`Ignored(rel, false)`）两处调用，与 `exclude_*` / 全局默认排除叠加；`infer` 与扩展名收集的独立遍历不受影响。GUI「排除」页扩展名输入框标签为「排除扩展名」（沿用 `exclude_patterns` 字段与 glob 语义）。
- **目录多选层级去重**：`frontend/src/components/editor/dirPaths.ts` 是父子路径判定（`isDescendant` / `isCovered` / `normalizeDirSelection`）的唯一实现；`DirTreeDialog` 勾选父目录时只保留父目录（子目录呈勾选但禁用），`confirm` 返回的是完整选择集合，`TagListInput.addFromTree` 以替换（非追加）方式写回，故取消勾选/清空同样生效；`TagListInput` 的 `dirHierarchy`（仅 `directories` 使用）让手动输入、粘贴、磁盘选择也按父子层级去重；`.`（项目根）覆盖所有相对目录；保存时 `ProjectEditorDialog.onSave` 还会归一化 `directories`，清理历史配置里残留的冗余子目录。
- **智能省流与注释剔除**：切到 `smart` 模式前端默认勾选 `stripComments`；smart 处理为分级降级链（`core/generator/smart.go` 返回 `SmartStats{Folded, SavedTokens, Escalated}`）：全文 →(超 SmartFullTokens，默认 1200)→ 标准骨架(MinBodyLines=4) →(仍超限)→ 激进骨架(MinBodyLines=2) →(无符号且仍超限的非大括号语言)→ `compress.HeadTailFold` 头尾保留折叠兜底；折叠数按 `… N lines …` 标记经 `compress.FoldedCount` 统计，章节标题标注有统计时用带数字版本（`smart_fold_note`，zh/en 双语）、否则回退 `skeleton_note`；smart 模式在 `app.genOptions` 强制关闭行号；测试在 `core/generator/smart_test.go`（短文件不动、超限折叠、两级降级、无符号兜底、幂等）。**先折叠后预算**：开启 BudgetTokens 且 smart 模式时，`prepareExport` 先调 `generator.Prefold` 就地预折叠（导出 `Options.Prefolded` 统计 map 透传给生成阶段，`smartContent` 命中则跳过二次折叠、直接沿用统计打标），预算按折叠后体积裁剪；full 等其它模式仍按原始内容估算。空白精简已整体并入 smart 模式——`ExportOptions` 无 `Compress` 字段，`generator.smartContent` 末尾固定走 `compress.MinifySmart(lang, …)`（行尾空白/空行折叠全语言适用，去缩进仅对 `compress.IsBraceLanguage` 大括号类语言生效，Python/YAML/Makefile 等缩进敏感语言自动跳过；白名单含 go/java/js/jsx/mjs/cjs/ts/tsx/c/cpp/cs/h/hpp/cc/cxx/objc/rust/kotlin/scala/swift/dart/php/groovy/json/jsonc/json5，扩充只改 `core/compress/minify.go` 的 `braceLangs`）；其它模式（full/files/symbols/signatures）不做空白精简；`core/compress/comments.go` 是注释剔除唯一实现，规则键为 `config/langs.yml` 语言名小写 + 常见别名，支持行/跨行块注释与 HTML/Vue 等混合语法；许可证块整体保留，行注释中的 TODO/SECURITY 标记整行保留，跨行块注释只留标记行（首尾仅留定界符），保留词用标记式匹配（普通英文词须带冒号，避免 Javadoc 叙述里的 `warning` 命中而保留整块）；字符串字面量（含三引号/反引号）内的注释符号不剥离，纯注释行变空行以保持行数；前端「保留标记注释」开关以 `keepMarkedComments` → `dropMarkedComments` 贯穿 `app.ExportOptions` / `generator.Options` / `compress.StripCommentsWithOptions`，默认保留标记，关闭后 TODO/许可证等标记注释一并删除，骨架折叠标记始终保留；`core/generator/smart.go` 必须在骨架化**之前**剥离，避免误删 `// … N lines …` 折叠标记。分片入口在 `app/app.go` 的 `RunExport`：`SplitParts > 1` 按个数切成 N 个文件（`core/token.SplitByCount`，按 token 估算顺序均衡分配，个数切分优先）、`SplitTokens > 0` 按 token 贪心装箱（`core/token.Split`）；两者仅 `full` / `smart` 模式生效，共用 `generator.FileParts` 前置头（仅首片含标题与文件树）与「第 i/N 部分」分片标记，省略清单附在最后一片，产出 `xxx.partN.md`；其余模式忽略分片走整文件输出。
- 测试进程在 Windows 上偶发 `unlinkat ... being used by another process`，是临时目录文件锁，与代码无关。

## 指令文件维护规则

项目结构、构建/测试命令、架构边界、开发约定或本文件记录的其他事实发生变化时，必须在**同一次改动**中同步更新本文件。
