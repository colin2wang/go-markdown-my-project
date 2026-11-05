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
		content, skeleton := smartContent(fr.Content, lang, opts)
		mark := ""
		if skeleton {
			mark = note
		}
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
		content, skeleton := smartContent(fr.Content, lang, opts)
		mark := ""
		if skeleton {
			mark = note
		}
		var s strings.Builder
		fmt.Fprintf(&s, "### %s: `%s`%s\n\n```%s\n%s```\n\n", fileLabel, rel, mark, lang, content)
		parts = append(parts, FullPart{RelPath: rel, Content: s.String()})
	}
	return b.String(), parts
}

// smartContent 单文件 smart 处理：（可选）剥离注释 → 超限骨架化 → 空白/长行精简。
// 剥离必须早于骨架化：token 阈值按精简后的内容估算（更多小文件可保留全文），
// 且不会把 Skeleton 生成的「// … N lines …」折叠标记误当注释删掉。
// 绑定层也会逐文件剥离一次，此处重复调用幂等（generator 可独立使用）。
// 返回处理后内容与是否发生了骨架化（用于章节标题标注）。
func smartContent(content, lang string, opts Options) (string, bool) {
	limit := opts.SmartFullTokens
	if limit <= 0 {
		limit = defaultSmartFullTokens
	}
	if opts.StripComments {
		content = compress.StripCommentsWithOptions(lang, content, compress.CommentOptions{
			KeepMarkers: !opts.DropMarkedComments,
		})
	}
	skeleton := false
	if token.EstimateTokens(content) > limit {
		if sk := compress.Skeleton(lang, content, compress.DefaultSkeletonOptions()); sk != content {
			content, skeleton = sk, true
		}
	}
	content = compress.MinifyWith(content, opts.MaxLineLen)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content, skeleton
}
