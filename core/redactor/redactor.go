// Package redactor 提供敏感信息识别与剔除（F99）。
// 两阶段设计：ScanSecrets 先扫描报告 → 导出时替换；规则表对齐设计文档 §7.2。
package redactor

import (
	"regexp"
	"strings"
)

// Kind 敏感信息类型。
const (
	KindPassword   = "password"
	KindAPIKey     = "api_key"
	KindToken      = "token"
	KindPrivateKey = "private_key"
	KindDBURL      = "db_url"
)

// rule 单条内置规则（RE2 语法）。
type rule struct {
	name string
	kind string
	re   *regexp.Regexp
	// whole 控制整段替换（PEM 块），否则只替换命中值
	whole bool
	// keepPrefix 保留捕获组 1（键名前缀），只替换其后的值，保持代码结构
	keepPrefix bool
}

// builtinRules 内置规则表（§7.2）。
var builtinRules = []rule{
	{"generic-password-assign", KindPassword, re(`((?i)(?:password|passwd|pwd|secret|secret[_-]?key|api[_-]?key|apikey|access[_-]?key|client[_-]?secret|auth[_-]?token|credential)[a-z0-9_-]*\s*[:=]\s*["']?)[^"'\s,;}{)<>]{4,}`), false, true},
	{"aws-access-key", KindAPIKey, re(`AKIA[0-9A-Z]{16}`), false, false},
	{"aws-secret-key", KindAPIKey, re(`((?i)aws.{0,20}secret.{0,20}[:=]\s*["']?)[A-Za-z0-9/+=]{40}`), false, true},
	{"github-pat", KindToken, re(`gh[pousr]_[A-Za-z0-9]{36,255}`), false, false},
	{"openai-style", KindAPIKey, re(`sk-[A-Za-z0-9_-]{20,}`), false, false},
	{"slack-token", KindToken, re(`xox[baprs]-[A-Za-z0-9-]{10,}`), false, false},
	{"jwt", KindToken, re(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.?[A-Za-z0-9_-]*`), false, false},
	{"pem-block", KindPrivateKey, re(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`), true, false},
	{"connection-string", KindDBURL, re(`((?i)(?:mysql|postgres(ql)?|mongodb(\+srv)?|redis|amqp|mssql)://)[^\s:@/"']+:[^\s@/"]+@[^\s"']+`), false, true},
	{"htpasswd-like", KindPassword, re(`((?i)(?:Basic|Bearer)\s+)[A-Za-z0-9+/=]{16,}`), false, true},
}

func re(expr string) *regexp.Regexp {
	return regexp.MustCompile(expr)
}

// Finding 扫描命中（对齐设计文档 SecretFinding）。
type Finding struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Kind   string `json:"kind"`
	Rule   string `json:"rule"`
	Masked string `json:"masked"` // 掩码预览
}

// Config 替换配置。
type Config struct {
	Placeholder string // 默认 [REDACTED]
	Strategy    string // placeholder | drop_line
	Allowlist   map[string]bool
	// CustomPatterns 用户自定义正则（RE2），全部按 Kind=password 处理
	CustomPatterns []*regexp.Regexp
}

// DefaultConfig 默认配置。
func DefaultConfig() Config {
	return Config{Placeholder: "[REDACTED]", Strategy: "placeholder"}
}

// falsePositives 误报抑制：值包含这些子串则忽略。
var falsePositives = []string{
	"your_", "xxx", "example", "changeme", "placeholder", "dummy", "test123",
	"<your", "${", "{{", "process.env", "os.Getenv", "import.meta.env",
}

// isFalsePositive 判断命中值是否为误报。
func isFalsePositive(value string) bool {
	v := strings.ToLower(value)
	for _, p := range falsePositives {
		if strings.Contains(v, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// Masked 掩码预览：保留首尾少量字符。
func Masked(value string) string {
	r := []rune(value)
	if len(r) <= 8 {
		return strings.Repeat("*", len(r))
	}
	return string(r[:4]) + "..." + strings.Repeat("*", 4) + string(r[len(r)-2:])
}

// ScanContent 扫描单个文件内容，返回命中列表（file 用于标注来源）。
func ScanContent(file, content string, cfg Config) []Finding {
	var findings []Finding
	for _, r := range allRules(cfg) {
		for _, loc := range r.re.FindAllStringSubmatchIndex(content, -1) {
			start, end := loc[0], loc[1]
			if r.keepPrefix && len(loc) > 3 && loc[3] >= 0 {
				start = loc[3] // 跳过键名前缀，只检查值
			}
			line := strings.Count(content[:start], "\n") + 1
			value := content[start:end]
			key := findingKey(file, line, r.name)
			if cfg.Allowlist[key] || isFalsePositive(value) {
				continue
			}
			masked := Masked(value)
			if r.whole {
				masked = Masked("-----BEGIN PRIVATE KEY-----")
			}
			findings = append(findings, Finding{File: file, Line: line, Kind: r.kind, Rule: r.name, Masked: masked})
		}
	}
	return findings
}

// findingKey allowlist key：文件相对路径:行号:规则名。
func findingKey(file string, line int, ruleName string) string {
	return file + ":" + itoa(line) + ":" + ruleName
}

// hit 一次替换的位置与内容。
type hit struct {
	start, end int
	replace    string
	dropLine   bool
}

// RedactContent 按配置替换敏感内容，返回替换后的内容与替换次数。
func RedactContent(file, content string, cfg Config) (string, int) {
	count := 0
	var out strings.Builder
	var hits []hit
	for _, r := range allRules(cfg) {
		for _, loc := range r.re.FindAllStringSubmatchIndex(content, -1) {
			start, end := loc[0], loc[1]
			if r.keepPrefix && len(loc) > 3 && loc[3] >= 0 {
				start = loc[3]
			}
			line := strings.Count(content[:start], "\n") + 1
			if cfg.Allowlist[findingKey(file, line, r.name)] {
				continue
			}
			value := content[start:end]
			if isFalsePositive(value) {
				continue
			}
			h := hit{start: start, end: end}
			if r.whole {
				h.start = loc[0]
				h.replace = "-----BEGIN PRIVATE KEY-----" + cfg.Placeholder + "-----END PRIVATE KEY-----"
			} else if cfg.Strategy == "drop_line" {
				h.dropLine = true
				lineStart := strings.LastIndexByte(content[:loc[0]], '\n') + 1
				lineEnd := strings.IndexByte(content[loc[1]:], '\n')
				if lineEnd < 0 {
					lineEnd = len(content) - loc[1]
				} else {
					lineEnd += loc[1] + 1
				}
				h.start, h.end = lineStart, lineEnd
				h.replace = ""
			} else {
				h.replace = cfg.Placeholder
			}
			hits = append(hits, h)
			count++
		}
	}
	if len(hits) == 0 {
		return content, 0
	}
	// 排序去重（区间重叠时保留先出现的）
	sortHits(hits)
	last := 0
	for _, h := range hits {
		if h.start < last {
			continue
		}
		out.WriteString(content[last:h.start])
		out.WriteString(h.replace)
		last = h.end
	}
	out.WriteString(content[last:])
	return out.String(), count
}

func allRules(cfg Config) []rule {
	rules := make([]rule, 0, len(builtinRules)+len(cfg.CustomPatterns))
	rules = append(rules, builtinRules...)
	for _, p := range cfg.CustomPatterns {
		rules = append(rules, rule{name: "custom", kind: KindPassword, re: p})
	}
	return rules
}

func sortHits(hits []hit) {
	for i := 1; i < len(hits); i++ {
		for j := i; j > 0 && hits[j].start < hits[j-1].start; j-- {
			hits[j], hits[j-1] = hits[j-1], hits[j]
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
