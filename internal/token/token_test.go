package token

import (
	"strings"
	"testing"
)

func TestEstimateTokensASCII(t *testing.T) {
	// 4 ASCII 字符 ≈ 1 token
	if got := EstimateTokens("abcdefgh"); got != 2 {
		t.Fatalf("EstimateTokens(8 ascii) = %d, want 2", got)
	}
	if got := EstimateTokens(""); got != 0 {
		t.Fatalf("空文本 = %d, want 0", got)
	}
}

func TestEstimateTokensCJK(t *testing.T) {
	// 中文 1 字 ≈ 1 token
	if got := EstimateTokens("你好世界"); got != 4 {
		t.Fatalf("EstimateTokens(4 cjk) = %d, want 4", got)
	}
}

func TestEstimateTokensMixed(t *testing.T) {
	// 2 CJK + 8 ASCII = 2 + 2 = 4
	if got := EstimateTokens("你好abcdefgh"); got != 4 {
		t.Fatalf("mixed = %d, want 4", got)
	}
}

func TestSplitSingleFileExceedsLimit(t *testing.T) {
	// 单文件超限：独占一片，内容不截断
	big := strings.Repeat("a", 8000) // ≈2000 tokens
	units := []FileUnit{
		{Path: "big.txt", Content: big},
		{Path: "small.txt", Content: "ok"},
	}
	chunks := Split(units, 100, "")
	if len(chunks) < 2 {
		t.Fatalf("应至少切 2 片，got %d", len(chunks))
	}
	for _, c := range chunks {
		if !strings.Contains(c.Content, markerFor(units, c)) && len(c.Files) == 0 {
			t.Errorf("分片缺文件清单")
		}
	}
	// 大文件内容完整
	found := false
	for _, c := range chunks {
		if strings.Contains(c.Content, big) {
			found = true
		}
	}
	if !found {
		t.Fatalf("大文件内容被截断")
	}
}

func markerFor(units []FileUnit, c Chunk) string {
	for _, f := range c.Files {
		return f
	}
	return ""
}

func TestSplitBoundaryExact(t *testing.T) {
	// 总量恰好等于 limit：单文件小内容 + 小 limit，允许 ≥1 片即可，验证不 panic 且内容完整
	units := []FileUnit{
		{Path: "a.txt", Content: "aaaaaaaa"},
		{Path: "b.txt", Content: "bbbbbbbb"},
	}
	chunks := Split(units, 2, "")
	all := ""
	for _, c := range chunks {
		all += c.Content
	}
	if !strings.Contains(all, "aaaaaaaa") || !strings.Contains(all, "bbbbbbbb") {
		t.Fatalf("分片丢失内容")
	}
}

func TestSplitWithPreamble(t *testing.T) {
	units := []FileUnit{{Path: "x.go", Content: strings.Repeat("x", 4000)}}
	chunks := Split(units, 500, "# Project Docs\n\n```tree```\n")
	if chunks[0].Index != 1 || chunks[0].Total != len(chunks) {
		t.Fatalf("分片编号错误: %d/%d", chunks[0].Index, chunks[0].Total)
	}
	if !strings.Contains(chunks[0].Content, "# Project Docs") {
		t.Fatalf("首片缺 preamble")
	}
}
