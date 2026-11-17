# 一、智能省流（`core/generator/smart.go` + `core/compress/`）
## 1. 现状管道
```
StripComments(可选) → 超过 SmartFullTokens(默认1200)？ → Skeleton 折叠函数体 → MinifySmart(去缩进/截断)
```
## 2. 问题诊断
| # | 问题 | 位置 | 影响 |
|---|---|---|---|
| 1 | **单次折叠、无二次降级**：折叠后若仍超限（函数短而多的文件很常见），直接输出，不再处理 | `smartContent` | 折叠效果不稳定 |
| 2 | **无符号文件逃逸**：YAML/JSON/长文本没有 func/method，`Skeleton` 原样返回，3000-token 的配置文件永远全量输出 | `Skeleton`（`foldableKinds` 仅 func/method） | smart 模式最大的 token 漏洞 |
| 3 | **与 BudgetTokens 不联动**：`prepareExport` 先按原始大小做预算裁剪（整文件 drop），fold 本可以救回的文件被整只丢弃 | `prepareExport` → `SelectWithinBudget` | 裁剪过狠，dropped manifest 膨胀 |
| 4 | **统计不透出**：折叠了几处、省了多少 token，前端拿不到，只有统一的 `skeleton_note` | `smartContent` 返回 `(string, bool)` | UI 无法展示"省流"价值 |
| 5 | **重复估算**：`EstimateTokens(content)` 在骨架判断处算一次，导出统计又算一次 | `smartContent` / `RunExport` | 小浪费 |
| 6 | **smart 模式下行号照加**：`IncludeLineNumbers` 若开启，行号本身吃 token，与省流目标冲突 | `genOptions` | 与模式语义矛盾 |
| 7 | `MaxLineLen` 截断在最后，`…` 可能切断折叠标记或标识符 | `MinifyWith` | 可接受但需知晓 |
## 3. 改造方案
### 3.1 两级降级 + 统计返回（改 `smart.go`）
```go
type SmartStats struct {
	Folded      int  // 折叠的函数体数量
	SavedTokens int  // 相对原文节省的估算 token
	Escalated   bool // 是否触发了激进档
}
// 降级链：全文 →(超限)→ 标准骨架(MinBodyLines=4) →(仍超限)→ 激进骨架(MinBodyLines=2) →(无符号且仍超限)→ 头尾保留
func smartContent(content, lang string, opts Options) (string, SmartStats) {
	limit := opts.SmartFullTokens
	if limit <= 0 {
		limit = defaultSmartFullTokens
	}
	before := token.EstimateTokens(content)
	st := SmartStats{}
	if opts.StripComments {
		content = compress.StripCommentsWithOptions(lang, content,
			compress.CommentOptions{KeepMarkers: !opts.DropMarkedComments})
	}
	if token.EstimateTokens(content) > limit {
		if sk := compress.Skeleton(lang, content, compress.DefaultSkeletonOptions()); sk != content {
			st.Folded = compress.FoldedCount(sk) // 见 3.2，从标记行统计
			content = sk
		}
		if token.EstimateTokens(content) > limit {
			opt := compress.DefaultSkeletonOptions()
			opt.MinBodyLines = 2
			if sk := compress.Skeleton(lang, content, opt); sk != content {
				content = sk
				st.Escalated = true
			}
		}
	}
	after := token.EstimateTokens(content)
	st.SavedTokens = before - after
	content = compress.MinifySmart(lang, content, opts.MaxLineLen)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content, st
}
```
`generateSmart` 里把 mark 升级为带数字的提示：
```go
if st.Folded > 0 {
	mark = fmt.Sprintf("（已折叠 %d 处，省 ≈%d tokens）", st.Folded, st.SavedTokens)
}
```
### 3.2 无符号文件的头尾折叠兜底（新增，补 #2）
LLM 通常需要文件的结构头部（import/包声明/顶层键），尾部价值低——对无符号且超限的文件做「保留头尾 + 中段折叠」：
```go
// core/compress/skeleton.go
func HeadTailFold(text string, keepLines int) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= keepLines*2+1 {
		return text
	}
	omitted := len(lines) - keepLines*2
	mid := fmt.Sprintf("# … %d lines omitted …", omitted)
	out := append(append([]string{}, lines[:keepLines]...), mid)
	out = append(out, lines[len(lines)-keepLines:]...)
	return strings.Join(out, "\n")
}
```
在 `smartContent` 的最后一级降级里调用（仅当 `!IsBraceLanguage(lang)` 且两档骨架都没动过内容时）。
### 3.3 与预算联动：先折叠、后裁剪（补 #3）
调整 `prepareExport` 的顺序——扫描结果里已有全文（`FileResult.Content`），折叠是廉价的：
```
scan → 对每文件：EstimateTokens > SmartFullTokens ? 先 Skeleton（记录 folded）
     → SelectWithinBudget（按折叠后的大小算预算）
     → generate（已折叠文件跳过二次处理，直接打 mark）
```
这样预算阶段"看到"的是压缩后的真实体积，`SelectWithinBudget` 只 drop 折叠后仍超预算的文件，dropped 数量显著下降。实现上给 `exportPlan` 加 `folded map[string]SmartStats` 即可，`scanner.FileResult` 不用动。
### 3.4 细节修复
| 项 | 做法 |
|---|---|
| 行号 | `genOptions` 中当 `Mode == ModeSmart` 时强制 `IncludeLineNumbers = false`（或在 UI 提示"省流模式自动关闭行号"） |
| 折叠计数 | `Skeleton` 输出的每条 `… N lines …` 标记可被正则 `// … (\d+) lines …` 直接统计，`FoldedCount` 无需侵入式改动 |
| note 文案 | `skeleton_note` 保留作兜底，有统计时优先用带数字版本 |
| 测试 | 补 `smart_test.go`：短文件不动、超限折叠、两级降级、无符号文件走 HeadTailFold、折叠后幂等（再跑一次不变） |
---
# 二、敏感词过滤（`core/wordfilter/words.go`）
这套规则的本质是**中和 XSS 载荷**，防止导出的 md 粘贴/上传到网页端聊天界面时被浏览器执行。现有实现可用，但有明显的死规则、覆盖缺口和工程隐患。
## 1. 问题诊断
| # | 问题 | 证据 |
|---|---|---|
| 1 | **死规则**：`\bstyle\s+url\s*\(` 要求字面 `style url(`，而真实代码是 `style="...url(..."`（中间隔着 `="`），几乎永不命中 | 规则表第 9 条 |
| 2 | **事件属性覆盖不全**：现有 `on(?:error|load|click|abort|blur|change|focus|input|submit|formaction|mouse[a-z]+|key[a-z]+)` 漏掉 `pointer*`、`drag*`、`animation*`、`transition*`、`wheel`、`scroll`、`dblclick`、`auxclick`、`copy/paste/cut`、`toggle`、`contextmenu` 等 | `onpointerdown=`、`<svg onanimationend=...>` 均穿透 |
| 3 | **属性向量缺口**：`srcdoc=`（iframe 内嵌 HTML）、`formaction=`、`xlink:href="javascript:..."`（SVG）、`<link rel=import>` 未覆盖 | 常见 cheat-sheet 条目 |
| 4 | **data: 只挡 text/html**：`data:text/xml`、`data:image/svg+xml`、`data:application/javascript` 同样危险 | 规则表第 4 条 |
| 5 | **幂等靠运气**：替换文本是自然语言占位符，若未来新增规则恰好匹配旧替换文本，会二次改写；没有"结构性保证" | `Filter` 串行应用 |
| 6 | **零统计、零配置**：`Filter(text) string` 只返回字符串，UI 无"过滤 N 处"，`ExportOptions.WordFilter` 只有一个 bool 开关，误伤安全教学代码时只能整体关掉 | 导出结果无对应字段 |
| 7 | **替换文本硬编码中文**：`[危险协议已脱敏]` 等与 `markdown_lang: en_us` 的导出混排 | 一致性问题 |
| 8 | **管道位置未约定**：必须保证在任何 `MaxLineLen` 截断**之前**执行，否则截断把 `<script` 劈成两半导致块规则漏配 | 与 smart 模式的交互 |
## 2. 改造方案
### 2.1 规则表增补与修复
```go
var rules = mustCompile([]struct{ pattern, replacement string }{
	// —— 现有规则保持，以下为修复/新增 ——
	// 修复死规则：针对 url() 内的危险协议
	{`(?i)\burl\s*\(\s*["']?\s*(?:javascript|vbscript|data\s*:\s*text/html)`, `[危险URL已脱敏](`},
	// data: 扩展 MIME 家族
	{`(?i)\bdata\s*:\s*(?:text/html|text/xml|application/xhtml|image/svg|text/javascript|application/(?:javascript|ecmascript))`, `[危险协议已脱敏]:`},
	// 危险属性
	{`(?i)\bsrcdoc\s*=`, `[属性已脱敏]=`},
	{`(?i)\bformaction\s*=`, `[属性已脱敏]=`},
	{`(?i)\bxlink:href\s*=\s*["']?\s*(?:javascript|data)`, `[属性已脱敏]=`},
	// 事件属性：basic 档保留现有显式表；新增缺失家族（标准档）
	{`(?i)\bon(?:pointer|drag|drop|animation|transition|wheel|scroll|copy|paste|cut|contextmenu|toggle|auxclick|dblclick|play|pause|before[a-z]*|touch)[a-z]*\s*=`, `[事件属性已脱敏]=`},
	// strict 档专属：宽匹配（≥4 字母避免 once= 误伤）
	{`(?i)\bon[a-z]{4,}\s*=`, `[事件属性已脱敏]=`},
})
```
> 注意：宽匹配 `\bon[a-z]{4,}\s*=` 会命中 `online=true` 这类配置键——所以放在 **strict 档**，默认 standard 档用显式家族列表。
### 2.2 Config + 双策略 + 统计（对齐 redactor 的架构）
```go
type Config struct {
	Enabled  bool
	Profile  string // "basic" | "standard" | "strict"
	Strategy string // "destroy"（现状，整体替换）| "neutralize"（仅中和尖括号，保留代码可读）
}
type Hit struct{ Rule string; Count int }
func FilterWithStats(text string, cfg Config) (string, []Hit) {
	active := rulesFor(cfg.Profile) // 按档位筛规则
	for _, r := range active {
		if cfg.Strategy == "neutralize" {
			text = r.re.ReplaceAllStringFunc(text, neutralize) // < → ＜, > → ＞
		} else {
			text = r.re.ReplaceAllString(text, r.replacement)
		}
	}
	return text, collectHits(active, text) // 或在循环里累加计数
}
```
`neutralize` 策略的价值：`<script>alert(1)</script>` → `＜script＞alert(1)＜/script＞`——粘贴到聊天框不再构成可执行 HTML，但代码对人和 LLM 仍完整可读，特别适合导出安全类源码的场景。
配置落地沿用 `RedactionConfig` 的模式，`core/config` 加：
```yaml
word_filter:
  enabled: true
  profile: standard    # basic | standard | strict
  strategy: destroy    # destroy | neutralize
```
`ExportOptions.WordFilter bool` 保留为运行时覆盖。`ExportResult` 增加 `FilteredCount int`，与 `RedactedCount` 分开上报；i18n 加 `log.wordFilterApplied`。
### 2.3 幂等改为"结构性保证"（补 #5）
现有 `TestFilter` 的幂等断言很好，但只对几条样本成立。加两个锁定测试：
```go
// 替换文本本身绝不能被任何规则再次命中 —— 保证幂等是构造性的，不靠巧合
func TestReplacementsAreInert(t *testing.T) {
	for _, r := range rules {
		if got := Filter(r.replacement); got != r.replacement {
			t.Errorf("替换文本会被再次改写: %q -> %q", r.replacement, got)
		}
	}
}
// cheat-sheet 级载荷全歼测试
func TestNoLivePayloadSurvives(t *testing.T) {
	payloads := []string{
		`<script>alert(1)</script>`, `<SCRIPT SRC=//x/a.js></SCRIPT>`,
		`<img src=x onerror=alert(1)>`, `<svg onload=alert(1)>`,
		`<iframe srcdoc="<script>alert(1)</script>">`,
		`<a href="javascript:alert(1)">x</a>`,
		`<object data="data:text/html,<script>alert(1)</script>">`,
		`<form><button formaction="javascript:alert(1)">`,
	}
	for _, p := range payloads {
		out := strings.ToLower(Filter(p))
		for _, bad := range []string{"<script", "javascript:", "onerror=", "srcdoc=", "formaction="} {
			if strings.Contains(out, bad) {
				t.Errorf("载荷存活: %s -> %s (含 %s)", p, out, bad)
			}
		}
	}
}
```
RE2 保证线性时间，无灾难性回溯，`(?is)` 的跨行 lazy 匹配在 10MB 导出上也安全——建议补一个 benchmark 确认 <100ms 即可。
### 2.4 管道位置约定（补 #8）
明确固定顺序，写进 `RunExport` 注释：
```
generate（逐文件） → wordfilter（危险载荷） → redactor（密钥） → Minify/MaxLineLen 截断 → token 统计 → 编码写盘
```
要点：**wordfilter 必须在一切截断之前**（截断会劈开标签），且在 token 统计之前（替换改变字数，统计应反映最终文本）。redactor 与 wordfilter 都在内容级跑，互不干扰——redactor 的 `[REDACTED]` 占位符不匹配任何载荷规则。
---
# 三、优先级汇总
| 优先级 | 事项 | 工作量 | 收益 |
|---|---|---|---|
| P0 | wordfilter：死规则修复 + 事件/属性/data: 覆盖增补 | 1h | 堵住真实穿透面 |
| P0 | wordfilter：`TestReplacementsAreInert` + 载荷全歼测试 | 30min | 幂等从运气变保证 |
| P1 | smart：两级降级 + `SmartStats` 返回 + mark 带数字 | 1.5h | 折叠效果可量化、可控 |
| P1 | smart：无符号文件 `HeadTailFold` 兜底 | 1h | 补掉最大 token 漏洞 |
| P1 | wordfilter：`FilterWithStats` + `FilteredCount` 上报 + UI 展示 | 1.5h | 用户可感知、可排查误伤 |
| P2 | smart：先折叠后预算裁剪的顺序调整 | 1h | dropped 文件显著减少 |
| P2 | wordfilter：profile/strategy 配置 + neutralize 策略 + allowlist | 2h | 安全教学类项目可用 |
| P3 | 管道顺序固化 + smart 模式自动关行号 + i18n 占位符 | 30min | 一致性收尾 |
两块改造的公共思路是一致的：**smart 省流从"单次判断"升级为"分级降级 + 可量化的统计"，wordfilter 从"黑盒字符串替换"升级为"分档规则 + 可观测 + 幂等有结构性保证"**——都与项目里 redactor 已验证的「Config / Profile / 统计上报 / allowlist」架构对齐，前端也可以复用同一套"安全处理"展示面板。
