package generator

import (
	"fmt"
	"strings"

	"go-markdown-my-project/core/pick"
	"go-markdown-my-project/core/scanner"
	"go-markdown-my-project/core/token"
)

// BudgetResult 预算裁剪结果：保留的文件与被裁剪的文件。
type BudgetResult struct {
	Kept    []scanner.FileResult
	Dropped []scanner.FileResult
}

// SelectWithinBudget 按重要性评分在 token 预算内挑选文件（导出「代码选择逻辑」）。
//
// 评分复用 core/pick：入口/构建清单/源码优先，测试与生成代码靠后。
// 单文件即便超预算也会保留（宁可略超，也不丢掉唯一的高价值文件）。
// budget ≤ 0 表示不裁剪。
func SelectWithinBudget(results []scanner.FileResult, projectRoot string, budget int) BudgetResult {
	if budget <= 0 || len(results) == 0 {
		return BudgetResult{Kept: results}
	}
	cands := make([]pick.Candidate, 0, len(results))
	byPath := map[string]int{}
	for i, r := range results {
		rel := relPath(r.FullPath, projectRoot)
		c := pick.NewCandidate(rel, int64(len(r.Content)), token.EstimateTokens(r.Content))
		cands = append(cands, c)
		byPath[c.Path] = i
	}
	keep, drop := pick.WithinBudget(cands, budget)
	res := BudgetResult{}
	for _, c := range keep {
		if i, ok := byPath[c.Path]; ok {
			res.Kept = append(res.Kept, results[i])
		}
	}
	for _, c := range drop {
		if i, ok := byPath[c.Path]; ok {
			res.Dropped = append(res.Dropped, results[i])
		}
	}
	return res
}

// DroppedManifest 生成被裁剪文件的清单：保留路径与规模，
// 让读者（LLM）知道项目里还剩下什么，而不是凭空消失。
func DroppedManifest(dropped []scanner.FileResult, projectRoot, lang string) string {
	if len(dropped) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n\n", localizedText("omitted_files", lang))
	for _, d := range dropped {
		rel := relPath(d.FullPath, projectRoot)
		fmt.Fprintf(&b, "- `%s` — %d bytes, ≈%d tokens\n", rel, len(d.Content), token.EstimateTokens(d.Content))
	}
	b.WriteString("\n")
	return b.String()
}
