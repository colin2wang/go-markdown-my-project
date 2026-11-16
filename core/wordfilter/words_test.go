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
		{"onerror 属性", `<img onerror="alert(1)">`, `<img [事件属性已脱敏]="alert(1)">`},
		{"onload 属性", `<body onload="go()">`, `<body [事件属性已脱敏]="go()">`},
		{"formaction 属性", `<button formaction="/x">`, `<button [事件属性已脱敏]="/x">`},
		{"普通单词含 on 不误伤", `person.name = 1`, `person.name = 1`},
		{"iframe 标签", `<iframe src="x"></iframe>`, `[标签已脱敏]`},
		{"meta http-equiv 标签", `<meta http-equiv="refresh" content="0">`, `[标签已脱敏]`},
		{"base href 属性", `<base href="/app/">`, `<base [属性已脱敏]="/app/">`},
		{"style url", `style url(javascript:x)`, `style [样式已脱敏](javascript:x)`},
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
		t.Errorf("行数变化：got %d 行换行，want %d\nout: %q", strings.Count(got, "\n"), want, got)
	}
}

func TestFilterEmpty(t *testing.T) {
	if got := Filter(""); got != "" {
		t.Errorf("Filter(\"\") = %q, want \"\"", got)
	}
}
