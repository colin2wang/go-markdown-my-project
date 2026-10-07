# AGENTS.md — AI 编程代理指南

Go + Wails v2 桌面应用（GUI）+ Vue3/TS 前端，把项目源码聚合为 Markdown 供 LLM 分析。由 Rust 项目 `markdown_my_project` 重写而来。

## 常用命令

```powershell
.\build.ps1                # 一键构建 → dist/（pnpm build → wails build → CLI → 拷配置）
.\build.ps1 -Clean         # 先清空 dist/（exe 被占用时先关掉正在运行的 GUI）
go build ./...             # 仅编译后端
go test ./core/...         # 全部单测（generator/symbol/redactor/token）
go test ./core/generator/ -update   # 重新生成黄金快照（仅在有意变更输出格式时）
wails dev                  # GUI 热重载开发
cd frontend && pnpm dev    # 纯前端开发
```

## 架构边界（必须遵守）

- `core/` 各包是纯 Go，**禁止 import Wails**；进度/事件抽象在 `app/` 绑定层适配。新逻辑放 `core/`，`app/app.go` 只做薄转发。
- 绑定层方法必须可导出（`Startup` 而非 `startup`），否则 Wails 不生成前端绑定。
- 前端通过 `frontend/src/api/bindings.ts` 手写封装访问 `window.go.main.App`，新增后端方法需同步在此补类型。

## 项目特有注意事项

- **黄金快照回归**：`core/generator/generator_test.go` 用 `testdata/golden.md` 锁死 `full` 模式输出；另一测试与旧版 `Example.md` 做结构对比（CRLF 归一化、容忍快照后新增的文件）。改动生成逻辑导致快照失败时，先确认是预期变更再 `-update`。
- **byte-equal 优先级**：`full` 模式输出须与旧 Rust 版逐字节一致——文件标题路径用**原生分隔符**（`relPathNative`），树形字符 `├── └── │` 与缩进逐字符对齐，勿"顺手改成 /"。
- **敏感信息最高原则**：redactor 采用两阶段（扫描报告→确认→导出替换），日志只记数量与规则名，**绝不落盘明文**；`allowlist` key 格式为 `相对路径:行号:规则名`。
- **pnpm 12**：构建脚本白名单在 `frontend/pnpm-workspace.yaml` 的 `allowBuilds`（package.json 里旧 `pnpm` 字段已失效）；build.ps1 用 ASCII 输出，PowerShell 5.1 下非 BOM 中文会解析失败。
- **路径约定**：落盘/展示用 `/` 相对路径；仅 `full` 模式的文件标题保留原生分隔符（见上）。
- **正则限制**：所有规则用 RE2（Go `regexp`），不支持反向引用；symbol 提取器为逐行正则（允许漏报），Go 方法规则靠 `nameGroup: 2` 取名。
- `config/` 是示例配置来源，build.ps1 整体拷到 `dist/config/`；GUI 默认路径 `config/projects` 相对 exe 工作目录。
- 测试进程在 Windows 上偶发 `unlinkat ... being used by another process`，是临时目录文件锁，与代码无关。

## 指令文件维护规则

项目结构、构建/测试命令、架构边界、开发约定或本文件记录的其他事实发生变化时，必须在**同一次改动**中同步更新本文件。
