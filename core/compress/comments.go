package compress

import (
	"strings"
)

// 注释语法：行注释前缀与块注释界定符（按语言名小写匹配）。
var lineComments = map[string]string{
	"go": "//", "golang": "//",
	"java": "//", "kotlin": "//", "kt": "//", "scala": "//", "swift": "//", "dart": "//",
	"javascript": "//", "js": "//", "typescript": "//", "ts": "//", "jsx": "//", "tsx": "//",
	"c": "//", "c++": "//", "cpp": "//", "c#": "//", "cs": "//", "csharp": "//",
	"rust": "//", "rs": "//", "php": "//", "groovy": "//", "gohtml": "//",
	"python": "#", "py": "#", "shell": "#", "sh": "#", "bash": "#", "zsh": "#",
	"ruby": "#", "rb": "#", "perl": "#", "pl": "#", "r": "#",
	"yaml": "#", "yml": "#", "toml": "#", "ini": "#", "conf": "#", "makefile": "#",
	"dockerfile": "#", "properties": "#",
	"sql": "--", "lua": "--", "haskell": "--", "ada": "--",
	"vb": "'", "vbnet": "'",
}

// blockComments 块注释界定符（Python 的 """docstring""" 属字符串字面量，不在此列）。
var blockComments = map[string][2]string{
	"go":         {"/*", "*/"},
	"golang":     {"/*", "*/"},
	"java":       {"/*", "*/"},
	"kotlin":     {"/*", "*/"},
	"kt":         {"/*", "*/"},
	"scala":      {"/*", "*/"},
	"swift":      {"/*", "*/"},
	"dart":       {"/*", "*/"},
	"javascript": {"/*", "*/"},
	"js":         {"/*", "*/"},
	"typescript": {"/*", "*/"},
	"ts":         {"/*", "*/"},
	"jsx":        {"/*", "*/"},
	"tsx":        {"/*", "*/"},
	"c":          {"/*", "*/"},
	"c++":        {"/*", "*/"},
	"cpp":        {"/*", "*/"},
	"c#":         {"/*", "*/"},
	"cs":         {"/*", "*/"},
	"csharp":     {"/*", "*/"},
	"rust":       {"/*", "*/"},
	"rs":         {"/*", "*/"},
	"php":        {"/*", "*/"},
	"groovy":     {"/*", "*/"},
	"css":        {"/*", "*/"},
	"scss":       {"/*", "*/"},
	"less":       {"/*", "*/"},
}

// keepCommentWords 含这些标记的注释信息密度高，剥离时保留。
var keepCommentWords = []string{
	"todo", "fixme", "hack", "xxx", "note:", "important", "warning", "deprecated",
	"security", "caveat", "attention",
}

// licenseWords 许可证头部（法律信息，保留）。
var licenseWords = []string{"copyright", "license", "licence", "spdx", "all rights reserved"}

// StripComments 按语言剥离注释：
//   - 保留 shebang 与许可证块；
//   - 保留含 TODO/FIXME/SECURITY 等标记的注释；
//   - 字符串字面量内的注释符号不受影响（含 Python 三引号 docstring）；
//   - 无法识别注释语法的语言原样返回（Text / Markdown 等）。
//
// 行数保持不变（注释行变空行后由 Minify 折叠），因此不影响符号行号标注。
func StripComments(lang, text string) string {
	key := strings.ToLower(strings.TrimSpace(lang))
	prefix, hasLine := lineComments[key]
	open, close_, hasBlock := "", "", false
	if bc, ok := blockComments[key]; ok {
		open, close_, hasBlock = bc[0], bc[1], true
	}
	if !hasLine && !hasBlock {
		return text
	}

	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	inBlock := false
	for idx, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		// shebang 保留（构建脚本的关键信息）
		if idx == 0 && strings.HasPrefix(line, "#!") {
			out = append(out, line)
			continue
		}
		var b strings.Builder
		i := 0
		for i < len(line) {
			if inBlock {
				j := strings.Index(line[i:], close_)
				if j < 0 {
					i = len(line)
				} else {
					i += j + len(close_)
					inBlock = false
				}
				continue
			}
			c := line[i]
			// 字符串字面量原样保留（同时保护其中的注释符号）
			if c == '"' || c == '\'' || c == '`' {
				q := c
				b.WriteByte(c)
				i++
				for i < len(line) {
					if line[i] == '\\' && i+1 < len(line) {
						b.WriteByte(line[i])
						b.WriteByte(line[i+1])
						i += 2
						continue
					}
					b.WriteByte(line[i])
					if line[i] == q {
						i++
						break
					}
					i++
				}
				continue
			}
			if hasBlock && strings.HasPrefix(line[i:], open) {
				rest := line[i:]
				lower := strings.ToLower(rest)
				if containsAny(lower, licenseWords) {
					b.WriteString(rest) // 许可证块整体保留
					i = len(line)
					continue
				}
				if j := strings.Index(line[i+len(open):], close_); j >= 0 {
					i += len(open) + j + len(close_)
				} else {
					inBlock = true
					i = len(line)
				}
				continue
			}
			if hasLine && strings.HasPrefix(line[i:], prefix) {
				if containsAny(strings.ToLower(line[i:]), keepCommentWords) {
					b.WriteString(line[i:])
				}
				i = len(line)
				continue
			}
			b.WriteByte(c)
			i++
		}
		res := strings.TrimRight(b.String(), " \t")
		// 整行被剥离成空：丢弃该行（后续由 Minify 统一整理空行）
		if strings.TrimSpace(res) == "" && strings.TrimSpace(line) != "" {
			continue
		}
		out = append(out, res)
	}
	return strings.Join(out, "\n")
}

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}
