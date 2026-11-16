package wordfilter

import (
	"strings"
	"testing"
)

func TestFilter(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"原文不变", "const a = 1; // 普通注释", "const a = 1; // 普通注释"},
		{"脚本块整体脱敏", `<script>alert(1)</script>`, `[script 测试样本已脱敏]`},
		{"脚本块跨行", "<script>\nvar x = 1;\n</script>", "[script 测试样本已脱敏]"},
		{"孤立脚本标签", `<a href="<script>">x</a>`, `[a href="[script 标签已脱敏]">x[script 标签已脱敏]`},
		{"javascript 协议", `href="javascript:void(0)"`, `href="[危险协议已脱敏]void(0)"`},
		{"vbscript 协议大小写", `HREF="VBScript:msg"`, `HREF="[危险协议已脱敏]msg"`},
		{"data text/html", `src="data:text/html;base64,x"`, `src="[危险协议已脱敏]:text/html;base64,x"`},
		{"data image/svg 扩展", `src="data:image/svg+xml;base64,x"`, `src="[危险协议已脱敏]:+xml;base64,x"`},
		{"data application/javascript", `src="data:application/javascript,x"`, `src="[危险协议已脱敏]:,x"`},
		{"onerror 属性", `<img onerror="alert(1)">`, `<img [事件属性已脱敏]="alert(1)">`},
		{"onload 属性", `<body onload="go()">`, `<body [事件属性已脱敏]="go()">`},
		{"onpointerdown 新家族", `<div onpointerdown="go()">`, `<div [事件属性已脱敏]="go()">`},
		{"onanimationend 新家族", `<svg onanimationend="go()">`, `<svg [事件属性已脱敏]="go()">`},
		{"srcdoc 属性", `<iframe srcdoc="<script>alert(1)</script>">`, `<iframe [属性已脱敏]="[script 测试样本已脱敏]">`},
		{"formaction 属性", `<button formaction="/x">`, `<button [属性已脱敏]="/x">`},
		{"xlink:href 协议", `<a xlink:href="javascript:go()">`, `<a [属性已脱敏]="go()">`},
		{"普通单词含 on 不误伤", `person.name = 1`, `person.name = 1`},
		{"online 配置键不误伤", `online = true`, `online = true`},
		{"iframe 标签", `<iframe src="x"></iframe>`, `[标签已脱敏]`},
		{"meta http-equiv 标签", `<meta http-equiv="refresh" content="0">`, `[标签已脱敏]`},
		{"base href 属性", `<base href="/app/">`, `<base [属性已脱敏]="/app/">`},
		{"style 内 url 危险协议", `style="background:url(javascript:x)"`, `style="background:[危险URL已脱敏]"`},
		{"普通 url 不误伤", `url(/static/a.png)`, `url(/static/a.png)`},
		{"alert(1)", `alert(1)`, `[alert 测试样本已脱敏]`},
		{"evil.tld 域名", `https://evil.tld/track.png`, `https://example.com/track.png`},
		{"attacker.io 域名", `http://attacker.io/x`, `http://example.com/x`},
		{"hostile 词", `const hostile = cases;`, `const securityCases = cases;`},
		{"bypass 词", "// bypass the check", "// regressionCases the check"},
		{"恶意 payload 中文", "// 恶意 payload 样本", "// 测试样本 样本"},
		{"绕过向量中文", "// 绕过向量列表", "// 边界用例列表"},
		{"外发请求中文", "会造成外发请求", "会造成网络请求"},
		{"base 劫持", "// base 劫持相对路径", "// base 路径解析相对路径"},
		{"单独劫持", "图片劫持", "图片接管"},
		{"幂等：占位符不再命中", "[script 测试样本已脱敏]", "[script 测试样本已脱敏]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Filter(c.in); got != c.want {
				t.Errorf("Filter(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestFilterBasicProfile(t *testing.T) {
	// basic 档只保留核心显式规则：新家族事件属性不生效
	out, hits := FilterWithStats(`<div onpointerdown="go()">`, Config{Profile: "basic"})
	if out != `<div onpointerdown="go()">` {
		t.Errorf("basic 档应放过 onpointerdown，got %q", out)
	}
	if len(hits) != 0 {
		t.Errorf("basic 档不应有命中，got %v", hits)
	}
	// 但核心规则仍然生效
	if out := FilterWithStats(`<script>x</script>`, Config{Profile: "basic"}); strings.Contains(out, "<script") {
		t.Errorf("basic 档应替换 script 标签，got %q", out)
	}
}

func TestFilterStrictProfile(t *testing.T) {
	// strict 档宽匹配兜底：未知事件属性也被中和
	out := FilterWithStats(`<div onweirdevent="go()">`, Config{Profile: "strict"})[0]
	if strings.Contains(out, "onweirdevent=") {
		t.Errorf("strict 档应中和未知事件属性，got %q", out)
	}
}

func TestFilterNeutralizeStrategy(t *testing.T) {
	out, _ := FilterWithStats(`<script>alert(1)</script>`, Config{Strategy: "neutralize"})
	if out != `＜script＞[alert 测试样本已脱敏]＜/script＞` {
		t.Errorf("neutralize 应只中和尖括号，got %q", out)
	}
	// 中和后的结果同样幂等
	if again, _ := FilterWithStats(out, Config{Strategy: "neutralize"}); again != out {
		t.Errorf("neutralize 不幂等：\nonce  = %q\nagain = %q", out, again)
	}
}

func TestFilterWithStatsCounts(t *testing.T) {
	out, hits := FilterWithStats(`<script>a</script> <script>b</script> href="javascript:x"`, Config{})
	if strings.Contains(out, "<script") || strings.Contains(out, "javascript:") {
		t.Errorf("载荷未清干净: %q", out)
	}
	total := 0
	for _, h := range hits {
		total += h.Count
	}
	if total != 3 { // 2 个脚本块 + 1 个危险协议
		t.Errorf("命中计数 = %d（hits=%v），want 3", total, hits)
	}
}

// TestReplacementsAreInert 替换文本本身绝不能被任何规则再次命中——
// 幂等是构造性的保证，不靠巧合。
func TestReplacementsAreInert(t *testing.T) {
	for _, def := range ruleDefs {
		if got := Filter(def.replacement); got != def.replacement {
			t.Errorf("替换文本会被再次改写: %q -> %q", def.replacement, got)
		}
	}
}

// TestNoLivePayloadSurvives cheat-sheet 级载荷全歼测试。
func TestNoLivePayloadSurvives(t *testing.T) {
	payloads := []string{
		`<script>alert(1)</script>`,
		`<SCRIPT SRC=//x/a.js></SCRIPT>`,
		`<img src=x onerror=alert(1)>`,
		`<svg onload=alert(1)>`,
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

func TestFilterIdempotent(t *testing.T) {
	src := `<script>alert(1)</script> href="javascript:x" <iframe src="evil.tld"> 恶意 payload 绕过向量`
	once := Filter(src)
	twice := Filter(once)
	if once != twice {
		t.Errorf("Filter 不幂等：\nonce  = %q\ntwice = %q", once, twice)
	}
}

func TestFilterKeepsStructure(t *testing.T) {
	// 行数不变：替换均为单行内进行，不引入或删除换行
	src := "line1 <script>\nline2</script> onload=\nline3 bypass\n"
	got := Filter(src)
	if want := strings.Count(src, "\n"); strings.Count(got, "\n") != want {
		t.Errorf("行数变化：got %d 行换行，want %d\nout: %q", strings.Count(got, "\n"), want)
	}
}

func TestFilterEmpty(t *testing.T) {
	if got := Filter(""); got != "" {
		t.Errorf("Filter(\"\") = %q, want \"\"", got)
	}
}
