package generator

import (
	"fmt"
	"strings"

	"go-markdown-my-project/core/compress"
	"go-markdown-my-project/core/scanner"
	"go-markdown-my-project/core/token"
)

// defaultSmartFullTokens 单文件保留全文的 token 上限（超出则改为骨架输出）。
const defaultSmartFullTokens = 1200

// SmartStats 单文件 smart 处理的统计（用于章节标题标注与「省流」价值展示）。
type SmartStats struct {
	Folded      int  // 折叠的函数体数量（按折叠标记统计）
	SavedTokens int  // 相对原文节省的估算 token
	Escalated   bool // 是否触发了激进档（MinBodyLines=2 或头尾折叠兜底）
}

// generateSmart 预算友好的混合模式（「用更少 token 换更多信息」）：
//   - 小文件保留全文（信息完整）；
//   - 大文件保留签名与结构、函数体折叠为一行标记；
//   - 可选剥离注释、截断超长行。
//
// 头部（标题 + 文件树）与 full 模式完全一致，便于 LLM 先建立全局结构认知。
func generateSmart(sorted []scanner.FileResult, opts Options) string {
	var b strings.Builder
	writeHead(&b, sorted, opts)

	fileLabel := localizedText("file_label", opts.Lang)
	note := localizedText("skeleton_note", opts.Lang)
	for _, fr := range sorted {
		// 与 full 模式一致：章节标题用原生分隔符
		rel := relPathNative(fr.FullPath, opts.ProjectRoot)
		lang := languageFor(opts.Languages, fr.FullPath)
		content, st := smartContent(fr.FullPath, fr.Content, lang, opts)
		mark := smartMark(st, note, opts.Lang)
		fmt.Fprintf(&b, "### %s: `%s`%s\n\n```%s\n%s```\n\n", fileLabel, rel, mark, lang, content)
	}
	return b.String()
}

// FileParts 按模式拆解逐文件章节（full = 全文；smart = 自动骨架），供分片装箱。
// full 模式直接复用 FullParts，保证与不分片输出逐字节一致。
func FileParts(sorted []scanner.FileResult, opts Options) (string, []FullPart) {
	if opts.Mode != ModeSmart {
		return FullParts(sorted, opts)
	}
	var b strings.Builder
	writeHead(&b, sorted, opts)

	fileLabel := localizedText("file_label", opts.Lang)
	note := localizedText("skeleton_note", opts.Lang)
	parts := make([]FullPart, 0, len(sorted))
	for _, fr := range sorted {
		rel := relPathNative(fr.FullPath, opts.ProjectRoot)
		lang := languageFor(opts.Languages, fr.FullPath)
		content, st := smartContent(fr.FullPath, fr.Content, lang, opts)
		mark := smartMark(st, note, opts.Lang)
		var s strings.Builder
		fmt.Fprintf(&s, "### %s: `%s`%s\n\n```%s\n%s```\n\n", fileLabel, rel, mark, lang, content)
		parts = append(parts, FullPart{RelPath: rel, Content: s.String()})
	}
	return b.String(), parts
}

// smartContent 单文件 smart 处理，降级链：
// 全文 →(超限)→ 标准骨架(MinBodyLines=4) →(仍超限)→ 激进骨架(MinBodyLines=2)
// →(无符号且仍超限的非大括号语言)→ 头尾保留折叠兜底 → 空白/长行精简。
// path 用于查找预算阶段已完成的预折叠（opts.Prefolded），命中则跳过二次折叠。
func smartContent(path, content, lang string, opts Options) (string, SmartStats) {
	if st, ok := opts.Prefolded[path]; ok {
		// 预算前已折叠：仅做空白/长行精简，统计沿用预折叠结果
		return minifySmartContent(content, lang, opts), st
	}
	content, st := foldContent(content, lang, opts)
	return minifySmartContent(content, lang, opts), st
}

// foldContent smart 处理的折叠段：（可选）剥离注释 → 两级骨架降级 →
// 头尾折叠兜底。剥离必须早于骨架化：token 阈值按精简后的内容估算
// （更多小文件可保留全文），且不会把 Skeleton 生成的「// … N lines …」
// 折叠标记误当注释删掉。绑定层也会逐文件剥离一次，此处重复调用幂等
// （generator 可独立使用）。
func foldContent(content, lang string, opts Options) (string, SmartStats) {
	limit := smartLimit(opts)
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
	// 第一档：标准骨架
	if sk := compress.Skeleton(lang, content, compress.DefaultSkeletonOptions()); sk != content {
		st.Folded = compress.FoldedCount(sk)
		content = sk
	}
	// 第二档：激进骨架（函数短而多的文件）
	if token.EstimateTokens(content) > limit {
		opt := compress.DefaultSkeletonOptions()
		opt.MinBodyLines = 2
		if sk := compress.Skeleton(lang, content, opt); sk != content {
			st.Folded = compress.FoldedCount(sk)
			content = sk
			st.Escalated = true
		}
	}
	// 兜底：无符号文件（YAML/JSON/长文本等）走头尾保留折叠。
	// 仅非大括号语言（两档骨架对其本就无效）；大括号语言宁可全量也不丢函数体。
	if token.EstimateTokens(content) > limit && !compress.IsBraceLanguage(lang) && st.Folded == 0 {
		if folded := compress.HeadTailFold(content, 20); folded != content {
			content = folded
			st.Escalated = true
		}
	}
	st.SavedTokens = before - token.EstimateTokens(content)
	return content, st
}

// smartLimit 单文件保留全文的 token 上限（≤0 → 默认 1200）。
func smartLimit(opts Options) int {
	if opts.SmartFullTokens > 0 {
		return opts.SmartFullTokens
	}
	return defaultSmartFullTokens
}

// minifySmartContent smart 收尾的空白/长行精简：行尾空白/空行折叠全语言适用；
// 去缩进仅大括号语言（Python/YAML 等缩进敏感语言保持缩进，避免语法破坏）。
func minifySmartContent(content, lang string, opts Options) string {
	content = compress.MinifySmart(lang, content, opts.MaxLineLen)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content
}

// Prefold 预算裁剪前的预折叠（仅 smart 模式）：对每个文件执行折叠段
// （剥离注释 → 两级骨架 → 头尾兜底）并就地写回，让预算阶段看到的是
// 折叠后的真实体积，SelectWithinBudget 只 drop 折叠后仍超预算的文件。
// 返回发生折叠的文件统计（key = FullPath），供生成阶段跳过二次折叠。
func Prefold(results []scanner.FileResult, opts Options) map[string]SmartStats {
	if opts.Mode != ModeSmart {
		return nil
	}
	sts := make(map[string]SmartStats)
	for i := range results {
		lang := languageFor(opts.Languages, results[i].FullPath)
		content, st := foldContent(results[i].Content, lang, opts)
		if content != results[i].Content {
			results[i].Content = content
			sts[results[i].FullPath] = st
		}
	}
	return sts
}

// smartMark 章节标题的后缀标注：有折叠统计时用带数字版本（展示「省流」价值），
// 否则回退统一的 skeleton_note。
func smartMark(st SmartStats, note, lang string) string {
	if st.Folded > 0 {
		return fmt.Sprintf(localizedText("smart_fold_note", lang), st.Folded, st.SavedTokens)
	}
	if st.Escalated {
		return note
	}
	return ""
}
