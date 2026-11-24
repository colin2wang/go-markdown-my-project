package compress

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"go-markdown-my-project/internal/symbol"
)

// SkeletonOptions 骨架化参数。
type SkeletonOptions struct {
	// MinBodyLines 函数体少于该行数时不折叠（短函数折叠反而丢信息），默认 4。
	MinBodyLines int
}

// DefaultSkeletonOptions 默认骨架化参数。
func DefaultSkeletonOptions() SkeletonOptions { return SkeletonOptions{MinBodyLines: 4} }

// foldableKinds 需要折叠函数体的符号类型（类型体的字段信息密度高，不折叠）。
var foldableKinds = map[string]bool{
	symbol.KindFunc:   true,
	symbol.KindMethod: true,
}

// Skeleton 把代码压缩为「骨架」：保留包声明/导入/类型定义与函数签名，
// 函数体替换为一行折叠标记（… N lines …）。
//
// 这是「用更少 token 换更多信息」的主力手段：LLM 仍能看清结构、签名与依赖，
// 但不再为函数体内部的样板代码付费。无对应符号提取器时原样返回。
func Skeleton(lang, text string, opt SkeletonOptions) string {
	if opt.MinBodyLines <= 0 {
		opt.MinBodyLines = 4
	}
	if strings.TrimSpace(text) == "" {
		return text
	}
	syms := symbol.ExtractConfig(lang, text, symbol.Config{
		IncludeLineNumbers: true,
		MaxSignatureLen:    0,
		ComputeSignatures:  false,
	})
	if len(syms) == 0 {
		return text // 无法骨架化：宁可多花 token 也不丢内容
	}
	lines := strings.Split(text, "\n")
	n := len(lines)

	// 收集候选折叠区间（函数体：签名结束行的下一行 → EndLine）
	type span struct{ from, to, size int }
	var spans []span
	for _, s := range syms {
		if !foldableKinds[s.Kind] {
			continue
		}
		start := s.Line - 1
		if start < 0 || start >= n {
			continue
		}
		body := sigEndLine(lines, start)
		end := s.EndLine - 1
		if end >= n {
			end = n - 1
		}
		if end <= body {
			continue
		}
		spans = append(spans, span{from: body + 1, to: end, size: end - body})
	}
	if len(spans) == 0 {
		return text
	}
	// 短区间优先：嵌套场景下先折叠内层，外层剩余部分仍保留结构信息
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].size < spans[j].size })

	marked := make([]bool, n)
	fold := make([]int, n)
	for _, sp := range spans {
		free := 0
		for i := sp.from; i <= sp.to; i++ {
			if i >= 0 && i < n && !marked[i] {
				free++
			}
		}
		if free < opt.MinBodyLines {
			continue
		}
		first := -1
		for i := sp.from; i <= sp.to; i++ {
			if i < 0 || i >= n || marked[i] {
				continue
			}
			marked[i] = true
			if first < 0 {
				first = i
			}
		}
		if first >= 0 {
			fold[first] = free
		}
	}

	var b strings.Builder
	b.Grow(len(text))
	for i, line := range lines {
		if fold[i] > 0 {
			b.WriteString(indentOf(line))
			b.WriteString(foldMark(lang, fold[i]))
			b.WriteByte('\n')
			continue
		}
		if marked[i] {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// HeadTailFold 头尾保留折叠兜底：对没有可折叠函数体的文件（YAML/JSON/长文本
// 等无 func/method 符号），保留头部 keepLines 行与尾部 keepLines 行，中段替换
// 为一行省略标记。LLM 通常需要结构头部（import/包声明/顶层键），尾部价值低。
// 行数不足（≤ keepLines*2+1）时原样返回。
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

// foldCountRe 匹配 Skeleton 折叠标记「// … N lines …」中的行数。
var foldCountRe = regexp.MustCompile(`… \d+ lines …`)

// FoldedCount 统计骨架输出中折叠标记的个数（用于折叠数量统计展示）。
func FoldedCount(skeleton string) int {
	return len(foldCountRe.FindAllString(skeleton, -1))
}

// sigEndLine 从声明行起找到签名结束行（类 C 的 '{' 行 / Python 的 ':' 行）。
func sigEndLine(lines []string, start int) int {
	for j := start; j < len(lines) && j < start+20; j++ {
		t := strings.TrimRight(lines[j], "\r")
		tr := strings.TrimSpace(t)
		if strings.Contains(t, "{") || strings.HasSuffix(tr, ":") || strings.HasSuffix(tr, "=>") {
			return j
		}
	}
	return start
}

// indentOf 取行首空白（保持骨架的缩进结构）。
func indentOf(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// foldMark 生成折叠标记（语言对应的注释前缀）。
func foldMark(lang string, n int) string {
	prefix, ok := lineComments[strings.ToLower(strings.TrimSpace(lang))]
	if !ok {
		prefix = "//"
	}
	return fmt.Sprintf("%s … %d lines …", prefix, n)
}
