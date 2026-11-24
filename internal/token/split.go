// Package token 分片：按文件为最小单位贪心装箱（§7.5）。
package token

import (
	"fmt"
	"strings"
)

// Chunk 一个分片。
type Chunk struct {
	Index   int // 1-based
	Total   int
	Files   []string // 本部分包含的文件（相对路径）
	Content string   // 完整 markdown（含头尾说明）
}

// FileUnit 参与装箱的文件单元。
type FileUnit struct {
	Path    string // 相对路径
	Content string // 该文件在 markdown 中的完整章节文本（含标题与代码块）
}

// headerFooting 一个分片固定开销的估算余量（头部清单 + 尾注）。
const headerFooting = 200

// part 一段分片：连续的若干文件及其 token 估算。
type part struct {
	files  []string
	tokens int
}

// Split 按文件为最小单位贪心装箱；limit 为每片 token 上限（>0）。
// 单文件超过 limit 时独占一片（不截断文件内容）。
// preamble 为首片前置内容（项目标题、文件树等）。
func Split(units []FileUnit, limit int, preamble string) []Chunk {
	if limit <= 0 || len(units) == 0 {
		return []Chunk{{Index: 1, Total: 1, Content: preamble + joinUnits(units)}}
	}

	var parts []part
	cur := part{}
	curBudget := limit
	if preamble != "" {
		curBudget -= EstimateTokens(preamble) + headerFooting
	}

	for _, u := range units {
		t := EstimateTokens(u.Content) + headerFooting
		// 单文件超限：独占一片
		if t > curBudget {
			if len(cur.files) > 0 {
				parts = append(parts, cur)
				cur = part{}
				curBudget = limit + headerFooting // 后续片无 preamble
			}
		}
		if len(cur.files) == 0 && t > limit {
			// 独占
			parts = append(parts, part{files: []string{u.Path}, tokens: t})
			continue
		}
		if t > curBudget-cur.tokens && len(cur.files) > 0 {
			parts = append(parts, cur)
			cur = part{}
			curBudget = limit + headerFooting
		}
		cur.files = append(cur.files, u.Path)
		cur.tokens += t
	}
	if len(cur.files) > 0 {
		parts = append(parts, cur)
	}

	return buildChunks(units, parts, preamble)
}

// SplitByCount 把 units 顺序切成恰好 n 片（各片按 token 估算尽量均衡）。
// n <= 1 或 units 为空时不切分；n > 文件数时按文件数切分（每片至少一个文件）。
// preamble 为首片前置内容（项目标题、文件树等）。
func SplitByCount(units []FileUnit, n int, preamble string) []Chunk {
	if n <= 1 || len(units) == 0 {
		return []Chunk{{Index: 1, Total: 1, Content: preamble + joinUnits(units)}}
	}
	if n > len(units) {
		n = len(units)
	}
	costs := make([]int, len(units))
	total := 0
	for i, u := range units {
		costs[i] = EstimateTokens(u.Content) + headerFooting
		total += costs[i]
	}
	// 顺序贪心：每片目标大小 = 剩余 token / 剩余片数，
	// 同时保证后面每片至少能分到一个文件。
	var parts []part
	idx := 0
	remaining := total
	for k := 0; k < n; k++ {
		partsLeft := n - k
		if partsLeft <= 1 { // 最后一片收下剩余全部
			p := part{}
			for ; idx < len(units); idx++ {
				p.files = append(p.files, units[idx].Path)
				p.tokens += costs[idx]
			}
			parts = append(parts, p)
			break
		}
		share := remaining / partsLeft
		p := part{}
		for idx < len(units) {
			filesLeft := len(units) - idx
			if len(p.files) > 0 && (p.tokens >= share || filesLeft <= partsLeft-1) {
				break
			}
			p.files = append(p.files, units[idx].Path)
			p.tokens += costs[idx]
			remaining -= costs[idx]
			idx++
		}
		parts = append(parts, p)
	}
	return buildChunks(units, parts, preamble)
}

// buildChunks 由分片规划生成最终分片内容（首片含 preamble，带序号说明）。
func buildChunks(units []FileUnit, parts []part, preamble string) []Chunk {
	total := len(parts)
	chunks := make([]Chunk, 0, total)
	offset := 0
	for i, p := range parts {
		var b strings.Builder
		if i == 0 && preamble != "" {
			b.WriteString(preamble)
		}
		fmt.Fprintf(&b, "\n\n> 第 %d/%d 部分，包含文件：%s\n\n", i+1, total, strings.Join(p.files, ", "))
		end := offset + len(p.files)
		b.WriteString(joinUnits(units[offset:end]))
		fmt.Fprintf(&b, "\n\n> （第 %d/%d 部分，未完待续）\n", i+1, total)
		chunks = append(chunks, Chunk{Index: i + 1, Total: total, Files: p.files, Content: b.String()})
		offset = end
	}
	return chunks
}

func joinUnits(units []FileUnit) string {
	var b strings.Builder
	for _, u := range units {
		b.WriteString(u.Content)
		b.WriteString("\n")
	}
	return b.String()
}
