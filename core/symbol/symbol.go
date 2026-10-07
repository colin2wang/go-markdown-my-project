// Package symbol 提供多语言符号提取（F3）。
// v1 用正则提取（允许漏报，目录级参考），v2 可换 tree-sitter，接口不变。
package symbol

import (
	"fmt"
	"regexp"
	"strings"
)

// Kind 符号类型。
const (
	KindFunc      = "func"
	KindMethod    = "method"
	KindClass     = "class"
	KindStruct    = "struct"
	KindInterface = "interface"
	KindTrait     = "trait"
)

// Symbol 提取出的符号（对齐设计文档 §5.2）。
type Symbol struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Signature string `json:"signature,omitempty"` // 完整签名（signatures 模式）
	Line      int    `json:"line"`
}

// Extractor 按语言提取符号。
type Extractor interface {
	Languages() []string
	Extract(content string) ([]Symbol, error)
}

// registry 按语言注册提取器。
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

// Extract 按语言提取符号；未注册语言返回空。
func Extract(language, content string) []Symbol {
	e, ok := For(language)
	if !ok {
		return nil
	}
	syms, err := e.Extract(content)
	if err != nil {
		return nil
	}
	return syms
}

// lineRule 单条正则规则：nameGroup 指定符号名所在捕获组（默认 1）。
type lineRule struct {
	re        *regexp.Regexp
	kind      string
	nameGroup int
}

// lineExtractor 基于逐行正则的通用提取器。
type lineExtractor struct {
	langs []string
	rules []lineRule
}

func (e *lineExtractor) Languages() []string { return e.langs }

func (e *lineExtractor) Extract(content string) ([]Symbol, error) {
	lines := strings.Split(content, "\n")
	var out []Symbol
	for i, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		for _, r := range e.rules {
			m := r.re.FindStringSubmatch(trimmed)
			if m == nil {
				continue
			}
			idx := r.nameGroup
			if idx < 1 || idx >= len(m) {
				idx = 1
			}
			name := m[idx]
			if name == "" {
				continue
			}
			out = append(out, Symbol{
				Kind:      r.kind,
				Name:      name,
				Signature: strings.TrimSpace(trimmed),
				Line:      i + 1,
			})
			break
		}
	}
	return out, nil
}

func mustCompile(expr string) *regexp.Regexp {
	re, err := regexp.Compile(expr)
	if err != nil {
		panic(fmt.Sprintf("symbol rule compile: %v", err))
	}
	return re
}
