package token

import (
	"strings"
	"testing"
)

func TestSplitByCount(t *testing.T) {
	units := []FileUnit{
		{"a.txt", strings.Repeat("a", 400)},
		{"b.txt", strings.Repeat("b", 400)},
		{"c.txt", strings.Repeat("c", 400)},
		{"d.txt", strings.Repeat("d", 400)},
	}
	chunks := SplitByCount(units, 2, "")
	if len(chunks) != 2 || chunks[0].Files[0] != "a.txt" || chunks[1].Files[0] != "c.txt" {
		t.Fatalf("顺序均衡切分失败: %+v", chunks)
	}
	all := ""
	for _, c := range chunks {
		all += c.Content
	}
	for _, u := range units {
		if !strings.Contains(all, u.Content) {
			t.Fatalf("内容丢失: %s", u.Path)
		}
	}
	if got := SplitByCount(units, 1, ""); len(got) != 1 {
		t.Errorf("n=1 不应切分")
	}
	if got := SplitByCount(units, 9, ""); len(got) != 4 {
		t.Errorf("n>文件数时应按文件数切分, got %d", len(got))
	}
}

func TestSplitByCountExactlyN(t *testing.T) {
	units := make([]FileUnit, 7)
	for i := range units {
		units[i] = FileUnit{Path: string(rune('a'+i)) + ".txt", Content: strings.Repeat("x", 100)}
	}
	chunks := SplitByCount(units, 3, "PREAMBLE")
	if len(chunks) != 3 {
		t.Fatalf("应切成 3 片, got %d", len(chunks))
	}
	if chunks[0].Index != 1 || chunks[2].Index != 3 || chunks[0].Total != 3 {
		t.Errorf("分片序号错误: %+v", chunks)
	}
	if !strings.Contains(chunks[0].Content, "PREAMBLE") || strings.Contains(chunks[1].Content, "PREAMBLE") {
		t.Errorf("preamble 应只出现在首片")
	}
	count := 0
	for _, c := range chunks {
		count += len(c.Files)
	}
	if count != 7 {
		t.Errorf("各片文件数之和应为 7, got %d", count)
	}
}

func TestSplitByCountEmpty(t *testing.T) {
	if got := SplitByCount(nil, 3, "P"); len(got) != 1 {
		t.Errorf("空输入不应切分")
	}
}
