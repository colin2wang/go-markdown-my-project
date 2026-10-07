// Package symbol 提供多语言符号提取（F3）。
// v1 用「行状态机 + 正则」启发式提取（允许漏报，目录级参考），v2 可换 tree-sitter，接口不变。
// 设计对齐 dosc/2026-10-07-output-mode.md §2/§3/§5。
package symbol

import (
	"fmt"
	"regexp"
	"strings"
)

// Kind 符号类型（对齐设计文档 §2）。
const (
	KindFunc      = "func"
	KindMethod    = "method"
	KindClass     = "class"
	KindStruct    = "struct"
	KindInterface = "interface"
	KindTrait     = "trait"
	KindEnum      = "enum"
	KindType      = "type" // 类型别名等
)

// Symbol 提取出的符号（对齐设计文档 §2/§5）。
type Symbol struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Parent    string `json:"parent,omitempty"`    // 所属 class/struct/impl 名；顶级符号为 ""
	Signature string `json:"signature,omitempty"` // 规范化单行签名（signatures 模式用；symbols 模式留空）
	Line      int    `json:"line"`                // 1-based 声明行
}

// Extractor 按语言提取符号。
type Extractor interface {
	Languages() []string
	Extract(content string) ([]Symbol, error)
}

// Config 控制提取与渲染行为（对齐设计文档 §2/§5/§9）。
type Config struct {
	IncludeLineNumbers bool // symbols 模式是否输出行号
	MaxSignatureLen    int  // 签名截断长度（默认 200），≤0 表示不截断
	ComputeSignatures  bool // 是否需要规范化签名（signatures 模式 true；symbols 模式可省计算）
}

// DefaultConfig 返回默认配置（symbols 模式：仅目录，不计算签名）。
func DefaultConfig() Config {
	return Config{IncludeLineNumbers: true, MaxSignatureLen: 200, ComputeSignatures: false}
}

// parentMode 父级归属策略（对齐设计文档 §3）。
type parentMode int

const (
	parentNone         parentMode = iota
	parentFromReceiver            // Go 方法：从 receiver 类型取名
	parentFromStack               // 类 C / Rust impl / Dart：从上下文类型栈取栈顶
	parentIndentStack             // Python：从缩进栈取最近 class
)

// rule 单条正则规则（声明式，对齐设计文档 §3）。
type rule struct {
	re            *regexp.Regexp
	kind          string
	nameGroup     int
	parent        parentMode
	receiverGroup int  // parentFromReceiver 时捕获 receiver 的组
	emit          bool // 是否产出符号（false 表示仅压栈，如 Rust impl）
}

// lineExtractor 基于「逐行正则 + 状态机」的通用提取器（对齐设计文档 §3）。
type lineExtractor struct {
	langs         []string
	lineComments  []string // 行注释前缀，如 ["//", "#"]
	blockOpen     string   // 块注释起始（可选）
	blockClose    string
	hasBlock      bool
	typeOpeners   []rule // 会产出符号并压栈的类型声明
	parentOpeners []rule // 仅压栈（不产出符号），如 Rust impl
	funcRules     []rule // 函数/方法
	sigEnd        byte   // 签名截断符 '{'（多数语言）或 ':'（Python）
	keepSigEnd    bool   // Python 保留尾 ':'
	indentBased   bool   // Python：用缩进代替花括号深度
	keywords      []string
}

// stackEntry 类型作用域栈项（记录声明所在花括号深度）。
type stackEntry struct {
	name  string
	depth int
}

func (e *lineExtractor) Languages() []string { return e.langs }

func (e *lineExtractor) Extract(content string) ([]Symbol, error) {
	lines := strings.Split(content, "\n")
	var out []Symbol
	if e.indentBased {
		out = e.scanIndent(lines)
	} else {
		out = e.scanBrace(lines)
	}
	return out, nil
}

// scanBrace 花括号深度驱动的状态机（多数语言）。
func (e *lineExtractor) scanBrace(lines []string) []Symbol {
	var out []Symbol
	inBlock := false
	depth := 0
	var stack []stackEntry

	for i, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)

		// 块注释状态机（单比特，启发式）
		if inBlock {
			if e.hasBlock && strings.Contains(line, e.blockClose) {
				inBlock = false
			}
			continue
		}
		if e.hasBlock {
			if idx := strings.Index(line, e.blockOpen); idx >= 0 {
				rest := line[idx+len(e.blockOpen):]
				if strings.Contains(rest, e.blockClose) {
					inBlock = false // 单行块注释，整行跳过
				} else {
					inBlock = true
				}
				continue
			}
		}

		// 纯行注释跳过（仅 trim 后以注释前缀开头的整行）
		if isCommentLine(trimmed, e.lineComments) {
			continue
		}

		typeMatched := false
		for _, r := range e.typeOpeners {
			m := r.re.FindStringSubmatch(trimmed)
			if m == nil {
				continue
			}
			name := group(m, r.nameGroup)
			if name == "" {
				continue
			}
			if r.emit {
				out = append(out, Symbol{Kind: r.kind, Name: name, Line: i + 1, Signature: e.normalizeSignature(lines, i)})
			}
			stack = append(stack, stackEntry{name: name, depth: depth})
			typeMatched = true
			break
		}
		if !typeMatched {
			for _, r := range e.parentOpeners {
				m := r.re.FindStringSubmatch(trimmed)
				if m == nil {
					continue
				}
				if name := group(m, r.nameGroup); name != "" {
					stack = append(stack, stackEntry{name: name, depth: depth})
				}
				break
			}
			for _, r := range e.funcRules {
				m := r.re.FindStringSubmatch(trimmed)
				if m == nil {
					continue
				}
				name := group(m, r.nameGroup)
				if name == "" || isKeyword(name, e.keywords) {
					continue
				}
				var parent string
				switch r.parent {
				case parentFromReceiver:
					parent = stripReceiver(group(m, r.receiverGroup))
				case parentFromStack:
					parent = topStack(stack, depth)
				}
				kind := r.kind
				if parent != "" && kind == KindFunc {
					kind = KindMethod // 处于类型作用域内 → 视为方法
				}
				out = append(out, Symbol{
					Kind:      kind,
					Name:      name,
					Parent:    parent,
					Line:      i + 1,
					Signature: e.normalizeSignature(lines, i),
				})
				break
			}
		}

		// 更新花括号深度并维护类型栈
		depth += braceDelta(line)
		for len(stack) > 0 && stack[len(stack)-1].depth >= depth {
			stack = stack[:len(stack)-1]
		}
	}
	return out
}

// scanIndent Python 缩进驱动的状态机（对齐设计文档 §3 机制 3）。
func (e *lineExtractor) scanIndent(lines []string) []Symbol {
	var out []Symbol
	type entry struct {
		indent int
		name   string
	}
	var stack []entry

	for i, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if isCommentLine(trimmed, e.lineComments) {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		// 弹出缩进 >= 当前行的栈项
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}

		typeMatched := false
		for _, r := range e.typeOpeners {
			m := r.re.FindStringSubmatch(trimmed)
			if m == nil {
				continue
			}
			name := group(m, r.nameGroup)
			if name == "" {
				continue
			}
			out = append(out, Symbol{Kind: r.kind, Name: name, Line: i + 1, Signature: e.normalizeSignature(lines, i)})
			stack = append(stack, entry{indent: indent, name: name})
			typeMatched = true
			break
		}
		if !typeMatched {
			for _, r := range e.funcRules {
				m := r.re.FindStringSubmatch(trimmed)
				if m == nil {
					continue
				}
				name := group(m, r.nameGroup)
				if name == "" || isKeyword(name, e.keywords) {
					continue
				}
				var parent string
				if len(stack) > 0 {
					parent = stack[len(stack)-1].name
				}
				kind := r.kind
				if parent != "" && kind == KindFunc {
					kind = KindMethod
				}
				out = append(out, Symbol{
					Kind:      kind,
					Name:      name,
					Parent:    parent,
					Line:      i + 1,
					Signature: e.normalizeSignature(lines, i),
				})
				break
			}
		}
	}
	return out
}

// normalizeSignature 把从 start 行起、到首个 sigEnd 之前的声明归一为单行（对齐设计文档 §5）。
func (e *lineExtractor) normalizeSignature(lines []string, start int) string {
	if e.sigEnd == 0 {
		return strings.TrimSpace(lines[start])
	}
	var b strings.Builder
	for j := start; j < len(lines) && j < start+30; j++ {
		line := strings.TrimRight(lines[j], "\r")
		if idx := strings.IndexByte(line, e.sigEnd); idx >= 0 {
			if e.keepSigEnd {
				b.WriteString(line[:idx+1]) // 保留 sigEnd（Python 冒号）
			} else {
				b.WriteString(line[:idx]) // 去掉 sigEnd（多数语言的 {）
			}
			break
		}
		b.WriteString(line)
		b.WriteByte(' ')
	}
	collapsed := wsRegex.ReplaceAllString(b.String(), " ")
	return strings.TrimSpace(collapsed)
}

// —— 包级辅助 ——

var registry = map[string]Extractor{}

func register(e Extractor) {
	for _, lang := range e.Languages() {
		registry[strings.ToLower(lang)] = e
	}
}

// For 返回语言对应的提取器。
func For(language string) (Extractor, bool) {
	e, ok := registry[strings.ToLower(language)]
	return e, ok
}

// Extract 按语言提取符号（默认配置，向后兼容）。
func Extract(language, content string) []Symbol {
	return ExtractConfig(language, content, Config{IncludeLineNumbers: true, MaxSignatureLen: 200, ComputeSignatures: true})
}

// ExtractConfig 按语言提取符号并应用配置（截断 / 是否计算签名）。
func ExtractConfig(language, content string, cfg Config) []Symbol {
	e, ok := For(language)
	if !ok {
		return nil
	}
	syms, err := e.Extract(content)
	if err != nil {
		return nil
	}
	for i := range syms {
		if cfg.MaxSignatureLen > 0 && len(syms[i].Signature) > cfg.MaxSignatureLen {
			syms[i].Signature = syms[i].Signature[:cfg.MaxSignatureLen] + " …"
		}
		if !cfg.ComputeSignatures {
			syms[i].Signature = "" // symbols 模式：省去签名计算
		}
	}
	return syms
}

func mustCompile(expr string) *regexp.Regexp {
	re, err := regexp.Compile(expr)
	if err != nil {
		panic(fmt.Sprintf("symbol rule compile: %v", err))
	}
	return re
}

var wsRegex = regexp.MustCompile(`\s+`)

func group(m []string, idx int) string {
	if idx < 1 || idx >= len(m) {
		return ""
	}
	return m[idx]
}

func isKeyword(name string, kw []string) bool {
	for _, k := range kw {
		if k == name {
			return true
		}
	}
	return false
}

func isCommentLine(trimmed string, comments []string) bool {
	for _, c := range comments {
		if strings.HasPrefix(trimmed, c) {
			return true
		}
	}
	return false
}

// topStack 返回深度小于当前行深度的最近一个类型名（父级）。
func topStack(stack []stackEntry, depth int) string {
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].depth < depth {
			return stack[i].name
		}
	}
	return ""
}

// braceDelta 估算单行花括号深度变化，粗略跳过字符串/字符字面量（启发式）。
func braceDelta(line string) int {
	delta := 0
	inStr := false
	var strCh byte
	inChar := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if inStr {
			if c == '\\' {
				i++
				continue
			}
			if c == strCh {
				inStr = false
			}
			continue
		}
		if inChar {
			if c == '\\' {
				i++
				continue
			}
			if c == '\'' {
				inChar = false
			}
			continue
		}
		switch c {
		case '"', '`':
			inStr = true
			strCh = c
		case '\'':
			inChar = true
		case '{':
			delta++
		case '}':
			delta--
		}
	}
	return delta
}

// stripReceiver 从 Go receiver 串提取类型名，如 "(s *Server)" → "Server"。
func stripReceiver(recv string) string {
	recv = strings.TrimSpace(recv)
	recv = strings.Trim(recv, "()")
	recv = strings.TrimSpace(recv)
	if recv == "" {
		return ""
	}
	if strings.HasPrefix(recv, "*") {
		recv = recv[1:]
	}
	if idx := strings.LastIndexAny(recv, " \t"); idx >= 0 {
		recv = recv[idx+1:]
	}
	return recv
}
