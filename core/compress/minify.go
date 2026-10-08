// Package compress 提供导出输出的压缩能力。
// 采用空白精简：裁剪行尾空白、合并连续空行，并可选剥除大括号类语言的
// 行首缩进，在保持代码语法的前提下减小导出体积。缩进剥离仅对缩进不承载
// 语法的大括号语言生效（YAML/Python 等缩进敏感语言不处理），且作用于
// 源码文件内容本身，Markdown 结构（列表/引用缩进）不受影响。
// 独立于 generator：仅做文本级处理。
package compress

import "strings"

// Minify 空白精简：
//   - 裁剪每行行尾空白（含 \r，兼容 CRLF）；
//   - 连续空行折叠为一个空行；
//   - 行首缩进不动（缩进即语法的语言保持有效）。
func Minify(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	prevBlank := false
	for _, line := range lines {
		t := strings.TrimRight(line, " \t\r")
		if t == "" {
			if prevBlank {
				continue // 折叠连续空行
			}
			prevBlank = true
		} else {
			prevBlank = false
		}
		out = append(out, t)
	}
	return strings.Join(out, "\n")
}

// braceLangs 行首缩进不承载语法的大括号类语言（小写匹配，含常见别名）。
var braceLangs = map[string]bool{
	"go": true, "golang": true,
	"java":       true,
	"javascript": true, "js": true,
	"typescript": true, "ts": true,
	"c": true, "c++": true, "cpp": true,
	"c#": true, "cs": true, "csharp": true,
	"rust": true, "rs": true,
	"kotlin": true, "kt": true,
	"scala": true, "swift": true, "dart": true,
	"php": true, "groovy": true,
	"json": true,
}

// IsBraceLanguage 判断语言名是否为大括号类语言（行首缩进可安全剥除）。
func IsBraceLanguage(lang string) bool {
	return braceLangs[strings.ToLower(strings.TrimSpace(lang))]
}

// StripIndent 剥除每行行首空格/制表符（嵌套结构由大括号表达，语法不变；
// 行尾空白与空行折叠交由 Minify 统一处理）。行数与行号不变，不影响
// 符号提取的 L%d 行号标注。多行字符串字面量中的前导空白会被一并剥除，
// 属已知取舍。
func StripIndent(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimLeft(line, " \t")
	}
	return strings.Join(lines, "\n")
}
