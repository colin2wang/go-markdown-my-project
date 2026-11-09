# AGENTS.md — AI 编程代理指南

Go + Wails v2 桌面应用（GUI）+ Vue3/TS 前端，把项目源码聚合为 Markdown 供 LLM 分析。由 Rust 项目 `markdown_my_project` 重写而来。

## 常用命令

```powershell
.\build.ps1                # 一键构建 → build/bin/（pnpm build → wails build → 拷 assets）
.\build.ps1 -Clean         # 先清空 build/bin/（exe 被占用时先关掉正在运行的 GUI）
go build ./...             # 仅编译后端
go test ./core/...         # 全部单测（generator/symbol/redactor/token/compress/gitignore）
go test ./core/generator/ -update   # 重新生成黄金快照（仅在有意变更输出格式时）
wails dev                  # GUI 热重载开发
cd frontend && pnpm dev    # 纯前端开发
.\generate-icon.ps1       # 由 frontend/public/icon.svg 生成应用图标（build/appicon.png + build/windows/icon.ico）
```

**构建版本戳**：build.ps1 按**本地时区**的构建时间生成 `YYYY-MM-DD_HH:mm:ss`（`[DateTime]::Now`，勿用 UTC；值含空格会被 PS→wails→go 传参链截断，故用下划线占位；构建日志另打印完整本地时间与 UTC 偏移），经 `-ldflags -X` 注入 `app.BuildVersion`（`app/version.go`），展示层用 `app.FormatBuildVersion()` 格式化为 `(build YYYY-MM-DD HH:mm:ss)`（窗口标题 + 前端 `GetVersion` 绑定）；本地 `go build` / `wails dev` 下为 `dev`。

## 架构边界（必须遵守）

- `core/` 各包是纯 Go，**禁止 import Wails**；进度/事件抽象在 `app/` 绑定层适配。新逻辑放 `core/`，`app/app.go` 只做薄转发。
- 新增的 `core/infer`（按项目结构推断推荐配置）、`core/pick`（文件重要性评分/预算裁剪）、`core/compress`（注释剥离 `StripComments` / 骨架化 `Skeleton` / 长行截断 `MinifyWith`）、`core/gitignore`（递归查找项目内所有 `.gitignore`、解析 Git 通配规则、`Matcher.Ignored` 判定路径忽略）均为纯 Go，不依赖 Wails，可被复用与单测；**注释剔除的语法表与状态机只放在 `core/compress/comments.go` 一个文件**，其它包不得另写一套注释规则。
- 绑定层方法必须可导出（`Startup` 而非 `startup`），否则 Wails 不生成前端绑定。
- 前端通过 `frontend/src/api/bindings.ts` 手写封装访问 `window.go.app.App`（App 结构体在 `app` 包下），新增后端方法需同步在此补类型。
- **窗口初始化**：`main.go` 以 `StartHidden: true` 启动，`app.App.Startup` 用 `runtime.ScreenGetAll` 取主屏**逻辑**分辨率（`Screen.Size`）的 75% 设尺寸并居中，且必须 `defer runtime.WindowShow(ctx)` 才显示窗口；取不到屏幕信息时回退 `main.go` 的固定尺寸，勿删 `WindowShow`。

## 协作约定

- **不要主动编译和验证**（`go build`、`go test`、`vue-tsc`、`pnpm build` 等一律不跑），用户自己做；仅当用户明确要求时才执行。
- **Commit message**：英文、Conventional Commit（`type(scope): summary`）、祈使语气、subject ≤72 字符、无句尾句号；一次提交只含一个逻辑变更。待提交的 message 文本写入 `COMMIT.md`。

## 项目特有注意事项

- **黄金快照回归**：`core/generator/generator_test.go` 用 `testdata/golden.md` 锁死 `full` 模式输出；另一测试与旧版 `Example.md` 做结构对比（CRLF 归一化、容忍快照后新增的文件）。改动生成逻辑导致快照失败时，先确认是预期变更再 `-update`。
- **byte-equal 优先级**：`full` 模式输出须与旧 Rust 版逐字节一致——文件标题路径用**原生分隔符**（`relPathNative`），树形字符 `├── └── │` 与缩进逐字符对齐，勿"顺手改成 /"。
- **敏感信息最高原则**：redactor 采用两阶段（扫描报告→确认→导出替换），日志只记数量与规则名，**绝不落盘明文**；`allowlist` key 格式为 `相对路径:行号:规则名`。命中抽屉（SensitiveDrawer）显示原始明文与统一替换值，属用户明确要求的内存内展示——明文仅经 `SensitiveHit.Value` 传给前端抽屉，不写盘、不进日志。规则定义唯一入口在项目配置「敏感」页（enabled / strategy / placeholder / `rule_profile` / custom_patterns / allowlist），`rule_profile` 取值 `strict | standard | relaxed`（空 = standard；relaxed 剔除 generic-password-assign / connection-string / htpasswd-like 三条易误报规则，筛选在 `core/redactor` 的 `allRules`）；是否脱敏由 `cfg.Redaction.Enabled` 决定（`ExportOptions` 已无 `Redact` 字段），`ScanSensitive` 与导出共用 `redactorConfig(cfg.Redaction)`，保证扫描报告与导出规则集一致。工作台只保留扫描按钮 + 命中展示（enabled 项目进入工作台自动扫描一次），不再提供开关/策略/占位符控件，也没有"未扫描"导出确认弹窗；新建项目默认 `enabled: true` + `standard`。
- **pnpm 12**：构建脚本白名单在 `frontend/pnpm-workspace.yaml` 的 `allowBuilds`（package.json 里旧 `pnpm` 字段已失效）；build.ps1 用 ASCII 输出，PowerShell 5.1 下非 BOM 中文会解析失败。
- **路径约定**：落盘/展示用 `/` 相对路径；仅 `full` 模式的文件标题保留原生分隔符（见上）。
- **正则限制**：所有规则用 RE2（Go `regexp`），不支持反向引用；symbol 提取器为逐行正则（允许漏报），Go 方法规则靠 `nameGroup: 2` 取名。
- `config/` 是 GUI 运行时配置来源（`config/langs.yml` 语言映射 + `config/projects/` 示例配置 + `config/exclude_defaults.yml` 全局默认排除，自动与项目 exclude_* 合并去重，项目内 `use_default_excludes: false` 可整体关闭；项目内 `skip_binary_files` 按内容跳过二进制（前 8KB 含 NUL），默认开启、仅显式 false 时输出；`config/global_settings.yml` 全局设置（GUI 设置对话框读写）：`output_encoding`（utf8_bom | utf8 | gb18030，导出编码，项目级 `write_bom` 显式设置时优先于全局）与 `interface_language`（zh_cn | en_us | ja_jp，设置对话框不再提供「跟随系统」，旧配置 auto 由前端按当前语言回显）；导出编码解析优先级为「项目 write_bom 显式设置 > 全局 output_encoding > 默认 utf8_bom」，编码转换在 `generator.WriteOutput` 落盘层，`Generate` 内存输出与 `full` 模式黄金快照不受影响，GB18030 转换失败回退 UTF-8+BOM；`log4rs.yml` 与 `cmd/` CLI 已删除，仅保留 GUI；原 `assets/` 已并入 `config/`）；build.ps1 拷 `config/` 到 `build/bin/`（先删旧目录再拷，避免 Copy-Item -Recurse 嵌套成 config/config）；GUI 默认路径 `config/projects` 相对 exe 工作目录。
- **.gitignore 过滤**：项目配置 `use_gitignore`（三态 `*bool`，nil = true，GUI 默认勾选；`SerializeWithComments` 恒输出该字段，勾选写 `true`、取消勾选写 `false`）由 `app.scanOpts` 单点注入 `scanner.Options.UseGitIgnore`，覆盖 `PreviewScan` / `ScanProject` / `ScanSensitive` / 导出四条链路；`core/gitignore.Load` 递归查找项目内所有 `.gitignore`（跳过 `.git` 目录与已被上层规则忽略的目录），规则按目录深度 + 行序排序（后写/子目录优先），支持注释、`!` 否定、尾部 `/` 仅目录、前导 `/` 锚定、`**` 跨层与 `\#` / `\!` 转义；`core/scanner` 在目录剪枝（`Ignored(rel, true)`）与文件判定（`Ignored(rel, false)`）两处调用，与 `exclude_*` / 全局默认排除叠加；`infer` 与扩展名收集的独立遍历不受影响。GUI「排除」页扩展名输入框标签为「排除扩展名」（沿用 `exclude_patterns` 字段与 glob 语义）。
- **目录多选层级去重**：`frontend/src/components/editor/dirPaths.ts` 是父子路径判定（`isDescendant` / `isCovered` / `normalizeDirSelection`）的唯一实现；`DirTreeDialog` 勾选父目录时只保留父目录（子目录呈勾选但禁用），`confirm` 返回的是完整选择集合，`TagListInput.addFromTree` 以替换（非追加）方式写回，故取消勾选/清空同样生效；`TagListInput` 的 `dirHierarchy`（仅 `directories` 使用）让手动输入、粘贴、磁盘选择也按父子层级去重；`.`（项目根）覆盖所有相对目录；保存时 `ProjectEditorDialog.onSave` 还会归一化 `directories`，清理历史配置里残留的冗余子目录。
- **智能省流与注释剔除**：切到 `smart` 模式前端默认勾选 `stripComments`（其它模式默认不勾选，用户可手动调整）；`core/compress/comments.go` 是注释剔除唯一实现，规则键为 `config/langs.yml` 语言名小写 + 常见别名，支持行/跨行块注释与 HTML/Vue 等混合语法；许可证块整体保留，行注释中的 TODO/SECURITY 标记整行保留，跨行块注释只留标记行（首尾仅留定界符），保留词用标记式匹配（普通英文词须带冒号，避免 Javadoc 叙述里的 `warning` 命中而保留整块）；字符串字面量（含三引号/反引号）内的注释符号不剥离，纯注释行变空行以保持行数；前端「保留标记注释」开关以 `keepMarkedComments` → `dropMarkedComments` 贯穿 `app.ExportOptions` / `generator.Options` / `compress.StripCommentsWithOptions`，默认保留标记，关闭后 TODO/许可证等标记注释一并删除，骨架折叠标记始终保留；`core/generator/smart.go` 必须在骨架化**之前**剥离，避免误删 `// … N lines …` 折叠标记。
- 测试进程在 Windows 上偶发 `unlinkat ... being used by another process`，是临时目录文件锁，与代码无关。

## 指令文件维护规则

项目结构、构建/测试命令、架构边界、开发约定或本文件记录的其他事实发生变化时，必须在**同一次改动**中同步更新本文件。
