package generator

import (
	"strings"
	"testing"

	"go-markdown-my-project/core/compress"
	"go-markdown-my-project/core/token"
)

func smartTestOptions(smartTokens int) Options {
	return Options{
		ProjectName: "demo", ProjectRoot: "F:/proj", Lang: "zh_cn",
		Mode: ModeSmart, SmartFullTokens: smartTokens,
	}
}

// 短文件：不折叠、不降级，原样保留（仅行尾精简）。
func TestSmartContentShortFileUnchanged(t *testing.T) {
	src := "package main\n\nfunc main() {\n\tprintln(1)\n}\n"
	out, st := smartContent("f.go", src, "go", smartTestOptions(0))
	if st.Folded != 0 || st.Escalated || st.SavedTokens != 0 {
		t.Errorf("短文件不应折叠: %+v", st)
	}
	if strings.Contains(out, "… ") {
		t.Errorf("短文件不应有折叠标记: %q", out)
	}
}

// 超限折叠：长函数体被折叠为标记，统计出折叠数与节省 token。
func TestSmartContentFoldsLongFunction(t *testing.T) {
	var b strings.Builder
	b.WriteString("package main\n\nfunc big() {\n")
	for i := 0; i < 200; i++ {
		b.WriteString("\tprintln(1)\n")
	}
	b.WriteString("}\n")
	src := b.String()

	out, st := smartContent("f.go", src, "go", smartTestOptions(0))
	if st.Folded == 0 {
		t.Fatalf("超限文件应折叠函数体: %+v", st)
	}
	if !strings.Contains(out, "… ") || !strings.Contains(out, "lines …") {
		t.Errorf("输出应含折叠标记: %q", out)
	}
	if st.SavedTokens <= 0 {
		t.Errorf("应统计出节省的 token: %+v", st)
	}
	if !token.EstimateTokens(src) > 0 || token.EstimateTokens(out) >= token.EstimateTokens(src) {
		t.Errorf("折叠后 token 应下降")
	}
}

// 两级降级：多个短函数（MinBodyLines=4 折不动）仍超限时用 MinBodyLines=2。
func TestSmartContentEscalates(t *testing.T) {
	var b strings.Builder
	b.WriteString("package main\n\n")
	for i := 0; i < 400; i++ {
		b.WriteString("func f() {\n\ta()\n\tb()\n\tc()\n}\n")
	}
	src := b.String()

	out, st := smartContent("f.go", src, "go", smartTestOptions(0))
	if !st.Escalated {
		t.Errorf("应触发激进档: %+v", st)
	}
	if st.Folded == 0 || strings.Count(out, "lines …") == 0 {
		t.Errorf("激进档应产生折叠: %+v, out=%q", st, out[:min(200, len(out))])
	}
}

// 无符号文件：YAML 无 func/method，两档骨架无效，走 HeadTailFold 兜底。
func TestSmartContentHeadTailFold(t *testing.T) {
	var b strings.Builder
	b.WriteString("key: config\n")
	for i := 0; i < 200; i++ {
		b.WriteString("item: value\n")
	}
	b.WriteString("tail: last\n")
	src := b.String()

	out, st := smartContent("f.yaml", src, "yaml", smartTestOptions(0))
	if !strings.Contains(out, "lines omitted …") {
		t.Errorf("无符号文件应走头尾折叠兜底: %q", out[:min(300, len(out))])
	}
	if !st.Escalated {
		t.Errorf("兜底应记为 Escalated: %+v", st)
	}
	if !strings.HasPrefix(out, "key: config") {
		t.Errorf("应保留头部: %q", out[:min(80, len(out))])
	}
}

// 折叠幂等：对已折叠输出再跑一次结果不变。
func TestSmartContentIdempotent(t *testing.T) {
	var b strings.Builder
	b.WriteString("package main\n\nfunc big() {\n")
	for i := 0; i < 200; i++ {
		b.WriteString("\tprintln(1)\n")
	}
	b.WriteString("}\n")
	opts := smartTestOptions(0)
	once, _ := smartContent("f.go", b.String(), "go", opts)
	twice, _ := smartContent("f.go", once, "go", opts)
	if once != twice {
		t.Errorf("smartContent 不幂等：\nonce  = %q\ntwice = %q", once, twice)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// HeadTailFold 单元行为：行数不足不动，超出行数保留头尾。
func TestHeadTailFold(t *testing.T) {
	if got := compress.HeadTailFold("a\nb\nc\n", 10); got != "a\nb\nc\n" {
		t.Errorf("行数不足应原样返回: %q", got)
	}
	var b strings.Builder
	for i := 0; i < 50; i++ {
		b.WriteString("line\n")
	}
	out := compress.HeadTailFold(b.String(), 2)
	lines := strings.Split(out, "\n")
	if lines[0] != "line" || lines[1] != "line" || !strings.Contains(out, "46 lines omitted") {
		t.Errorf("头尾折叠结果不符: %q", out)
	}
}
