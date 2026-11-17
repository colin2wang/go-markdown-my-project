# 增强方案：智能省流参数可配置 + 卡片文案调整
根据现有代码结构（`smart.go` 写死了 `defaultSmartFullTokens = 1200`、骨架档 `MinBodyLines: 4/2`、头尾折叠 `keepLines: 20`），做以下改动：
## 一、Go 后端：暴露可调参数
### `core/generator/generator.go` — Options 增加字段
```go
type Options struct {
	// ...原有字段不动...
	SmartFullTokens     int
	Prefolded           map[string]SmartStats
	// 新增：智能省流可调参数（0 = 使用默认值）
	SkeletonMinBodyLines int // 骨架折叠：函数体 ≥ N 行才折叠（默认 4）
	HeadTailKeepLines    int // 头尾折叠兜底：保留头尾各 N 行（默认 20）
}
```
### `core/generator/smart.go` — 消灭写死的魔数
```go
const (
	defaultSmartFullTokens   = 1200
	defaultSkeletonMinBody   = 4
	defaultHeadTailKeepLines = 20
)
func smartLimit(opts Options) int {
	if opts.SmartFullTokens > 0 {
		return opts.SmartFullTokens
	}
	return defaultSmartFullTokens
}
func skeletonMinBody(opts Options) int {
	if opts.SkeletonMinBodyLines > 0 {
		return opts.SkeletonMinBodyLines
	}
	return defaultSkeletonMinBody
}
func headTailKeep(opts Options) int {
	if opts.HeadTailKeepLines > 0 {
		return opts.HeadTailKeepLines
	}
	return defaultHeadTailKeepLines
}
func foldContent(content, lang string, opts Options) (string, SmartStats) {
	limit := smartLimit(opts)
	minBody := skeletonMinBody(opts)
	keep := headTailKeep(opts)
	before := token.EstimateTokens(content)
	st := SmartStats{}
	if opts.StripComments {
		content = compress.StripCommentsWithOptions(lang, content, compress.CommentOptions{
			KeepMarkers: !opts.DropMarkedComments,
		})
	}
	if token.EstimateTokens(content) <= limit {
		return content, st
	}
	if sk := compress.Skeleton(lang, content, compress.SkeletonOptions{MinBodyLines: minBody}); sk != content {
		st.Folded = compress.FoldedCount(sk)
		content = sk
	}
	if token.EstimateTokens(content) > limit {
		aggressive := minBody / 2 // 激进档：默认档的一半，下限 2（minBody=4 → 2，与原行为一致）
		if aggressive < 2 {
			aggressive = 2
		}
		if sk := compress.Skeleton(lang, content, compress.SkeletonOptions{MinBodyLines: aggressive}); sk != content {
			st.Folded = compress.FoldedCount(sk)
			content = sk
			st.Escalated = true
		}
	}
	if token.EstimateTokens(content) > limit && !compress.IsBraceLanguage(lang) && st.Folded == 0 {
		if folded := compress.HeadTailFold(content, keep); folded != content {
			content = folded
			st.Escalated = true
		}
	}
	st.SavedTokens = before - token.EstimateTokens(content)
	return content, st
}
```
### `app/app.go` — ExportOptions 增加字段并透传
```go
type ExportOptions struct {
	// ...原有字段...
	SmartFullTokens int `json:"smartFullTokens"`
	// 智能省流可调参数（0 = 默认）
	SkeletonMinBodyLines int `json:"skeletonMinBodyLines"`
	HeadTailKeepLines    int `json:"headTailKeepLines"`
}
```
在（折叠的）`genOptions` 中追加透传：
```go
	opts.SmartFullTokens = opt.SmartFullTokens
	opts.SkeletonMinBodyLines = opt.SkeletonMinBodyLines
	opts.HeadTailKeepLines = opt.HeadTailKeepLines
```
## 二、前端：绑定与 Store
### `frontend/src/api/bindings.ts`
```typescript
export interface ExportOptions {
  // ...
  smartFullTokens?: number;
  /** 骨架折叠：函数体 ≥ N 行才折叠（0 = 默认 4） */
  skeletonMinBodyLines?: number;
  /** 头尾折叠兜底保留行数（0 = 默认 20） */
  headTailKeepLines?: number;
}
```
### `frontend/src/stores/index.ts`（useExportStore）
```typescript
const smartFullTokens = ref(0);
const skeletonMinBodyLines = ref(0);
const headTailKeepLines = ref(0);
function buildOptions(): ExportOptions {
  return {
    // ...
    smartFullTokens: smartFullTokens.value,
    skeletonMinBodyLines: skeletonMinBodyLines.value,
    headTailKeepLines: headTailKeepLines.value,
  };
}
// return 对象中补充导出这三个 ref
```
`frontend/wailsjs/go/models.ts` 的 `ExportOptions` 类同步加两个数字字段（或用 `wails generate module` 重新生成）。
## 三、WorkbenchView.vue：模式卡片下方加控件与说明
放在导出模式卡片列表之后（右侧栏），仅在 `exp.mode === 'smart'` 时显示：
```vue
<!-- 卡片外说明（不再塞进卡片 desc） -->
<p v-if="exp.mode === 'smart'" class="mode-note">
  <Icon name="warn" :size="12" /> {{ t('wb.smartIndentNote') }}
</p>
<!-- 智能省流可调参数 -->
<template v-if="exp.mode === 'smart'">
  <label class="inline wrap">
    {{ t('wb.smartFullTokens') }}
    <input class="num" type="number" min="0" v-model.number="exp.smartFullTokens" />
    <span class="hint">{{ t('wb.smartFullTokensHint') }}</span>
  </label>
  <label class="inline wrap">
    {{ t('wb.skeletonMinBody') }}
    <input class="num" type="number" min="0" v-model.number="exp.skeletonMinBodyLines" />
    <span class="hint">{{ t('wb.skeletonMinBodyHint') }}</span>
  </label>
  <label class="inline wrap">
    {{ t('wb.headTailKeep') }}
    <input class="num" type="number" min="0" v-model.number="exp.headTailKeepLines" />
    <span class="hint">{{ t('wb.headTailKeepHint') }}</span>
  </label>
</template>
```
```css
.mode-note { color: var(--fg-3); font-size: 11px; margin: 6px 0 0; display: flex; align-items: center; gap: 5px; }
```
## 四、i18n：卡片去 Python、新增文案（三个语言文件）
### `zh-CN.ts`（wb 节）
```typescript
modes: {
  smart: { label: '智能省流', desc: '小文件全文 + 大文件骨架，剥离注释；自动精简空白' },
},
smartIndentNote: '缩进敏感语言（Python/YAML 等）不去缩进，仅精简空行与行尾空白',
smartFullTokens: '小文件全文阈值',
smartFullTokensHint: '（估算 token ≤ 阈值保留全文；0 = 默认 1200）',
skeletonMinBody: '骨架折叠起点',
skeletonMinBodyHint: '（函数体 ≥ N 行才折叠；0 = 默认 4）',
headTailKeep: '头尾折叠保留行数',
headTailKeepHint: '（无符号文件兜底，保留头尾各 N 行；0 = 默认 20）',
```
### `en-US.ts`
```typescript
modes: {
  smart: { label: 'Smart budget', desc: 'Small files full + large files skeleton, comments stripped; whitespace minified' },
},
smartIndentNote: 'Indent-sensitive languages (Python/YAML etc.) keep indentation; only blank lines and trailing whitespace are trimmed',
smartFullTokens: 'Small-file full threshold',
smartFullTokensHint: '(files ≤ this estimated token count stay full; 0 = default 1200)',
skeletonMinBody: 'Skeleton fold start',
skeletonMinBodyHint: '(fold bodies ≥ N lines; 0 = default 4)',
headTailKeep: 'Head/tail keep lines',
headTailKeepHint: '(fallback for symbol-less files keeps N lines head & tail; 0 = default 20)',
```
### `ja-JP.ts`
```typescript
modes: {
  smart: { label: 'スマート節約', desc: '小ファイルは全文、大ファイルは骨格、コメント除去；空白を自動精簡' },
},
smartIndentNote: 'インデント依存言語（Python/YAML 等）はインデントを保持し、空行と行末空白のみ精簡します',
smartFullTokens: '小ファイル全文しきい値',
smartFullTokensHint: '（推定トークン ≤ しきい値は全文保持；0 = 既定 1200）',
skeletonMinBody: '骨格折りたたみ起点',
skeletonMinBodyHint: '（関数本体 ≥ N 行で折りたたみ；0 = 既定 4）',
headTailKeep: '頭尾折りたたみ保持行数',
headTailKeepHint: '（シンボルなしファイルのフォールバック：頭尾各 N 行保持；0 = 既定 20）',
```
## 五、补充测试（`core/generator/smart_test.go`）
```go
func TestSmartCustomMinBody(t *testing.T) {
	opts := smartTestOptions(0)
	opts.SkeletonMinBodyLines = 10 // 默认 4 时 30 行体也会折叠，这里验证可调
	var b strings.Builder
	b.WriteString("package main\n\nfunc f() {\n")
	for i := 0; i < 3; i++ { b.WriteString("\ta()\n") }
	b.WriteString("}\n")
	if _, st := smartContent("f.go", b.String(), "go", opts); st.Folded != 0 {
		t.Errorf("3 行体 < minBody=10，不应折叠")
	}
}
func TestSmartCustomHeadTailKeep(t *testing.T) {
	opts := smartTestOptions(0)
	opts.HeadTailKeepLines = 5
	var b strings.Builder
	b.WriteString("key: config\n")
	for i := 0; i < 50; i++ { b.WriteString("item: value\n") }
	out, _ := smartContent("f.yaml", b.String(), "yaml", opts)
	if got := len(strings.Split(out, "\n")); got > 5*2+3 {
		t.Errorf("keep=5 时头尾共约 10 行 + 提示行，got %d 行", got)
	}
}
```
## 要点说明
- **兼容性**：三个参数均为 `0 = 默认`，旧配置/旧前端调用行为完全不变（阈值 1200、骨架 4→2、头尾 20），现有 `smart_test.go` 全部用例无需修改。
- **激进档联动**：用户调大 `骨架折叠起点` 时，激进档自动取其一半（下限 2），保持"先宽松后激进"的梯度语义。
- **文案位置**：Python 缩进说明从模式卡片 desc 中移除，改为卡片下方的灰色提示行，只有选中智能省流时出现，不再挤占卡片空间。
  WorkbenchView 当前文件为骨架，第三节的片段请按你实际的模式卡片区结构（绑定 `exp.mode` 的那组按钮/卡片）插入到其后即可。
