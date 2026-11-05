package compress

import (
	"bytes"
	"strings"
)

// 注释剔除的唯一实现集中在本文件：语言语法表 + 扫描状态机 + 保留规则。
// 对外暴露 StripComments（默认选项）与 StripCommentsWithOptions；lineComments 由语法表派生，供 Skeleton 生成折叠标记。
// 规则键为 config/langs.yml 的语言名小写形式，并附常见别名与扩展名。

// lineComment 行注释起始符。
type lineComment struct {
	prefix      string
	foldCase    bool // 大小写不敏感（Batch 的 REM）
	boundary    bool // 后须为空白或行尾（避免把 remark 当成 REM 注释）
	blockedNext string // 后跟这些字符时不算注释（PHP 的 #[Attribute]）
}

// blockComment 块注释界定符，允许跨行且 open/close 可以不同。
type blockComment struct{ open, close string }

// commentRule 一种语言的注释语法；lines/blocks 为空表示不支持该类注释。
type commentRule struct {
	lines  []lineComment
	blocks []blockComment
}

func lc(prefix string) lineComment { return lineComment{prefix: prefix} }
func bc(open, close string) blockComment {
	return blockComment{open: open, close: close}
}

// 常见语法组合：C 风格、哈希注释、标记语言、SQL/脚本方言等。
var (
	ruleC        = commentRule{lines: []lineComment{lc("//")}, blocks: []blockComment{bc("/*", "*/")}}
	rulePHP      = commentRule{lines: []lineComment{lc("//"), lineComment{prefix: "#", blockedNext: "["}}, blocks: []blockComment{bc("/*", "*/")}}
	ruleHCL      = commentRule{lines: []lineComment{lc("//"), lc("#")}, blocks: []blockComment{bc("/*", "*/")}}
	ruleCSS      = commentRule{blocks: []blockComment{bc("/*", "*/")}}
	ruleSass     = commentRule{lines: []lineComment{lc("//")}, blocks: []blockComment{bc("/*", "*/")}}
	ruleHTML     = commentRule{blocks: []blockComment{bc("<!--", "-->")}}
	ruleVue      = commentRule{lines: []lineComment{lc("//")}, blocks: []blockComment{bc("<!--", "-->"), bc("/*", "*/")}}
	ruleHash     = commentRule{lines: []lineComment{lc("#")}}
	rulePS       = commentRule{lines: []lineComment{lc("#")}, blocks: []blockComment{bc("<#", "#>")}}
	ruleJulia    = commentRule{lines: []lineComment{lc("#")}, blocks: []blockComment{bc("#=", "=#")}}
	ruleSQL      = commentRule{lines: []lineComment{lc("--")}, blocks: []blockComment{bc("/*", "*/")}}
	ruleLua      = commentRule{lines: []lineComment{lc("--")}, blocks: []blockComment{bc("--[[", "]]"), bc("/*", "*/")}}
	ruleHaskell  = commentRule{lines: []lineComment{lc("--")}, blocks: []blockComment{bc("{-", "-}")}}
	ruleMatlab   = commentRule{lines: []lineComment{lc("%")}, blocks: []blockComment{bc("%{", "}%")}}
	rulePercent  = commentRule{lines: []lineComment{lc("%")}}
	ruleSemi     = commentRule{lines: []lineComment{lc(";")}}
	ruleBang     = commentRule{lines: []lineComment{lc("!")}}
	ruleBatch    = commentRule{lines: []lineComment{lineComment{prefix: "rem", foldCase: true, boundary: true}, lc("::")}}
	ruleVB       = commentRule{lines: []lineComment{lc("'"), lineComment{prefix: "rem", foldCase: true, boundary: true}}}
	ruleMarkdown = commentRule{blocks: []blockComment{bc("<!--", "-->")}}
)

// commentRules 语言名（小写）→ 注释语法。未列出的语言（Text / JSON 等）原样输出。
var commentRules = map[string]commentRule{
	// C 风格：// 与 /* */
	"go": ruleC, "golang": ruleC, "java": ruleC, "kotlin": ruleC, "kt": ruleC, "kts": ruleC,
	"scala": ruleC, "swift": ruleC, "dart": ruleC, "groovy": ruleC, "gradle": ruleC,
	"javascript": ruleC, "js": ruleC, "jsx": ruleC, "mjs": ruleC, "cjs": ruleC,
	"typescript": ruleC, "ts": ruleC, "tsx": ruleC,
	"c": ruleC, "h": ruleC, "c++": ruleC, "cpp": ruleC, "cxx": ruleC, "cc": ruleC, "hpp": ruleC,
	"c#": ruleC, "cs": ruleC, "csharp": ruleC, "objective-c": ruleC, "objc": ruleC, "m": ruleC,
	"rust": ruleC, "rs": ruleC, "solidity": ruleC, "zig": ruleC, "d": ruleC,
	"proto": ruleC, "protobuf": ruleC, "gohtml": ruleC, "jsonc": ruleC, "json5": ruleC,
	"php": rulePHP, "terraform": ruleHCL, "hcl": ruleHCL,
	// CSS 家族与标记语言
	"css": ruleCSS, "scss": ruleSass, "sass": ruleSass, "less": ruleSass,
	"html": ruleHTML, "htm": ruleHTML, "xhtml": ruleHTML, "xml": ruleHTML, "svg": ruleHTML,
	"vue": ruleVue, "svelte": ruleVue,
	// # 行注释（脚本、配置、数据）
	"python": ruleHash, "py": ruleHash, "ruby": ruleHash, "rb": ruleHash,
	"shell": ruleHash, "sh": ruleHash, "bash": ruleHash, "zsh": ruleHash, "fish": ruleHash,
	"perl": ruleHash, "pl": ruleHash, "r": ruleHash, "elixir": ruleHash, "ex": ruleHash,
	"nim": ruleHash, "coffeescript": ruleHash, "graphql": ruleHash, "gql": ruleHash,
	"yaml": ruleHash, "yml": ruleHash, "toml": ruleHash, "ini": ruleHash, "conf": ruleHash,
	"properties": ruleHash, "makefile": ruleHash, "make": ruleHash, "dockerfile": ruleHash,
	"cmake": ruleHash,
	// 带块注释的 # / % 方言
	"powershell": rulePS, "ps1": rulePS, "psm1": rulePS,
	"julia": ruleJulia, "jl": ruleJulia,
	"erlang": rulePercent, "erl": rulePercent, "latex": rulePercent, "tex": rulePercent,
	"matlab": ruleMatlab,
	// -- 行注释
	"sql": ruleSQL, "plsql": ruleSQL, "pgsql": ruleSQL, "mysql": ruleSQL, "ada": ruleSQL, "vhdl": ruleSQL,
	"lua": ruleLua, "haskell": ruleHaskell, "hs": ruleHaskell, "elm": ruleHaskell,
	// 其它方言
	"batch": ruleBatch, "bat": ruleBatch, "cmd": ruleBatch,
	"vb": ruleVB, "vbnet": ruleVB, "basic": ruleVB,
	"lisp": ruleSemi, "scheme": ruleSemi, "clojure": ruleSemi, "clj": ruleSemi,
	"assembly": ruleSemi, "asm": ruleSemi, "nasm": ruleSemi,
	"fortran": ruleBang, "f90": ruleBang, "f95": ruleBang,
	// Markdown 只剥离 HTML 注释，# 标题不是注释
	"markdown": ruleMarkdown, "md": ruleMarkdown,
}

// lineComments 由规则表派生，供 Skeleton 的折叠标记取语言对应的行注释前缀。
var lineComments = func() map[string]string {
	m := make(map[string]string, len(commentRules))
	for lang, r := range commentRules {
		if len(r.lines) > 0 {
			m[lang] = r.lines[0].prefix
		}
	}
	return m
}()

// keepCommentWords 标记式保留词。普通英文词要求带冒号，避免 Javadoc 里的
// 「只出 warning 不自动合并」这类叙述命中后把整块注释原样保留。
var keepCommentWords = []string{
	"todo", "fixme", "security", "deprecated", "caveat",
	"note:", "important:", "warning:", "warn:", "attention:", "hack:", "xxx:",
}

// licenseWords 许可证头部（法律信息，保留）。
var licenseWords = []string{"copyright", "license", "licence", "spdx", "all rights reserved"}

// CommentOptions 注释剥离选项。
type CommentOptions struct {
	// KeepMarkers 保留 TODO/FIXME/SECURITY 等标记注释与许可证块；
	// false 时连同这些标记注释一并删除。骨架折叠标记始终保留（属于输出结构）。
	KeepMarkers bool
}

// DefaultCommentOptions 默认保留高价值标记注释（兼容既有导出行为）。
func DefaultCommentOptions() CommentOptions { return CommentOptions{KeepMarkers: true} }

// StripComments 按默认选项剥离注释（保留标记注释）。
func StripComments(lang, text string) string {
	return StripCommentsWithOptions(lang, text, DefaultCommentOptions())
}

// StripCommentsWithOptions 按语言剥离注释（智能省流的核心步骤）：
//   - 覆盖常见语言的行注释、跨行块注释与混合标记语言（HTML/Vue 等）；
//   - shebang 始终保留；KeepMarkers=true 时保留许可证块与行注释中的 TODO/SECURITY 标记；
//   - 跨行块注释只保留标记所在行（首尾仅留定界符），不因一个词保留整块；
//   - KeepMarkers=false 时上述标记注释一并删除；
//   - 骨架折叠标记（… N lines …）始终保留，不受选项影响；
//   - 字符串字面量内的注释符号不受影响（含三引号 docstring、反引号模板）；
//   - 未识别注释语法的语言原样返回（Text / JSON / Markdown 标题等）。
//
// 纯注释行变为空行而非删除，行数保持不变，不影响符号行号标注；
// 空行由 Minify 折叠，行尾注释删除时一并去掉前置空白。
func StripCommentsWithOptions(lang, text string, opt CommentOptions) string {
	rule, ok := commentRules[strings.ToLower(strings.TrimSpace(lang))]
	if !ok || (len(rule.lines) == 0 && len(rule.blocks) == 0) {
		return text
	}

	var b bytes.Buffer
	b.Grow(len(text))
	i := 0
	// shebang 是解释器声明而非注释，首行整体保留（含 CRLF）。
	if strings.HasPrefix(text, "#!") {
		if end := strings.IndexByte(text, '\n'); end >= 0 {
			b.WriteString(text[:end+1])
			i = end + 1
		} else {
			return text
		}
	}

	for i < len(text) {
		// 块注释优先：--[[ / #= / <!-- 等与行注释前缀共用起始字符，需最长匹配。
		if blk, n, ok := matchBlock(rule, text, i); ok {
			frag := text[i:blockEnd(text, i+n, blk.close)]
			lower := strings.ToLower(frag)
			switch {
			case opt.KeepMarkers && containsAny(lower, licenseWords):
				b.WriteString(frag) // 许可证块：法律信息整体保留
			case keepFragment(frag, opt):
				// 单行块注释短小可整体保留；跨行块只留标记行，避免一个关键词
				// 就把整段 Javadoc / 文档注释原样带出来。
				if strings.Contains(frag, "\n") {
					b.WriteString(slimBlock(frag, blk.open, blk.close, opt))
				} else {
					b.WriteString(frag)
				}
			default:
				writeNewlines(&b, frag)
				trimTailSpace(&b)
			}
			i += len(frag)
			continue
		}
		if _, ok := matchLine(rule, text, i); ok {
			end := len(text)
			if j := strings.IndexByte(text[i:], '\n'); j >= 0 {
				end = i + j
			}
			frag := text[i:end]
			if keepFragment(frag, opt) {
				b.WriteString(frag)
			} else {
				trimTailSpace(&b)
				if strings.HasSuffix(frag, "\r") {
					b.WriteByte('\r') // 保持 CRLF 换行风格
				}
			}
			i = end // '\n' 由下一轮写出，保持行数
			continue
		}
		// 字符串字面量整段跳过，避免把其中的 //、#、-- 当注释。
		if c := text[i]; c == '"' || c == '\'' || c == '`' {
			if end := stringEnd(text, i); end > i {
				b.WriteString(text[i:end])
				i = end
				continue
			}
		}
		b.WriteByte(text[i])
		i++
	}
	return b.String()
}

// matchBlock 返回当前位置的块注释规则与 open 长度（多个界定符时取最长）。
func matchBlock(rule commentRule, text string, i int) (blockComment, int, bool) {
	var found blockComment
	longest := 0
	for _, blk := range rule.blocks {
		if len(blk.open) > longest && strings.HasPrefix(text[i:], blk.open) {
			found, longest = blk, len(blk.open)
		}
	}
	return found, longest, longest > 0
}

// matchLine 判断当前位置是否为行注释起点（支持大小写不敏感与词边界）。
func matchLine(rule commentRule, text string, i int) (int, bool) {
	rest := text[i:]
	for _, l := range rule.lines {
		if len(rest) < len(l.prefix) {
			continue
		}
		head := rest[:len(l.prefix)]
		if l.foldCase {
			if !strings.EqualFold(head, l.prefix) {
				continue
			}
		} else if head != l.prefix {
			continue
		}
		// 裸 URL 的协议分隔符不是注释（SCSS/Vue 等支持 // 行注释时尤其容易误伤）。
		if l.prefix == "//" && i > 0 && text[i-1] == ':' {
			continue
		}
		if l.blockedNext != "" && len(rest) > len(l.prefix) &&
			strings.IndexByte(l.blockedNext, rest[len(l.prefix)]) >= 0 {
			continue
		}
		if l.boundary && len(rest) > len(l.prefix) {
			c := rest[len(l.prefix)]
			if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
				continue
			}
		}
		return len(l.prefix), true
	}
	return 0, false
}

// blockEnd 找到块注释结束位置（含界定符）；未闭合则到文本末尾。
func blockEnd(text string, from int, close string) int {
	if j := strings.Index(text[from:], close); j >= 0 {
		return from + j + len(close)
	}
	return len(text)
}

// stringEnd 返回字符串字面量结束位置（不含结束引号时返回 -1）。
// 支持转义、三引号 docstring 与可跨行的反引号模板；普通引号不跨行，
// 避免 Rust 生命周期 'a 之类把后续代码吞进「字符串」而漏剥注释。
func stringEnd(text string, i int) int {
	q := text[i]
	if (q == '"' || q == '\'') && i+2 < len(text) && text[i+1] == q && text[i+2] == q {
		delim := text[i : i+3]
		for j := i + 3; j+3 <= len(text); j++ {
			if text[j] == '\\' {
				j++
				continue
			}
			if text[j:j+3] == delim {
				return j + 3
			}
		}
		return -1
	}
	multiline := q == '`'
	for j := i + 1; j < len(text); j++ {
		c := text[j]
		if c == '\\' && j+1 < len(text) {
			j++
			continue
		}
		if c == '\n' && !multiline {
			return -1
		}
		if c == q {
			return j + 1
		}
	}
	if multiline {
		return len(text)
	}
	return -1
}

// keepFragment 行注释保留判断：折叠标记始终保留；标记注释仅在 KeepMarkers 时保留。
func keepFragment(frag string, opt CommentOptions) bool {
	lower := strings.ToLower(frag)
	if isFoldMark(lower) {
		return true
	}
	return opt.KeepMarkers && (containsAny(lower, keepCommentWords) || containsAny(lower, licenseWords))
}

func isFoldMark(lower string) bool {
	return strings.Contains(lower, "…") && strings.Contains(lower, "lines …")
}

// slimBlock 精简跨行块注释：首尾行只留定界符（及同行代码），
// 折叠标记行始终保留，KeepMarkers 时含高价值标记的行也保留，
// 其余行清空为空行，保持行数与行号。
func slimBlock(frag, open, close string, opt CommentOptions) string {
	parts := strings.SplitAfter(frag, "\n")
	if len(parts) == 1 {
		return frag
	}
	closeLine := -1
	if j := strings.LastIndex(frag, close); j >= 0 {
		closeLine = strings.Count(frag[:j], "\n")
	}
	for k, p := range parts {
		body, eol := splitEOL(p)
		switch {
		case k == 0:
			if j := strings.Index(body, open); j >= 0 {
				end := j + len(open)
				if end < len(body) && (body[end] == '*' || body[end] == '!') {
					end++ // 保留 /**、/*! 这类文档/保护注释的起始符
				}
				body = body[:end]
			}
		case k == closeLine:
			if j := strings.LastIndex(body, close); j >= 0 {
				body = body[j:] // 去掉定界符前的 "* " 装饰
			} else {
				body = ""
			}
		default:
			lower := strings.ToLower(body)
			keepLine := isFoldMark(lower) || (opt.KeepMarkers && containsAny(lower, keepCommentWords))
			if !keepLine {
				body = ""
			}
		}
		parts[k] = body + eol
	}
	return strings.Join(parts, "")
}

// splitEOL 拆出正文与换行符，清空的行仍保留原 CRLF/LF 风格。
func splitEOL(p string) (string, string) {
	if !strings.HasSuffix(p, "\n") {
		return p, ""
	}
	body := p[:len(p)-1]
	if strings.HasSuffix(body, "\r") {
		return body[:len(body)-1], "\r\n"
	}
	return body, "\n"
}

// writeNewlines 删除跨行块注释时保留其换行，维持行数与换行风格。
func writeNewlines(b *bytes.Buffer, frag string) {
	for i := 0; i < len(frag); i++ {
		switch {
		case frag[i] == '\r' && i+1 < len(frag) && frag[i+1] == '\n':
			b.WriteString("\r\n")
			i++
		case frag[i] == '\n':
			b.WriteByte('\n')
		}
	}
}

func trimTailSpace(b *bytes.Buffer) {
	data := b.Bytes()
	end := len(data)
	for end > 0 && (data[end-1] == ' ' || data[end-1] == '\t') {
		end--
	}
	b.Truncate(end)
}

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}
