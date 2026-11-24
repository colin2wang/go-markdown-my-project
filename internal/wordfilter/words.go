// Package wordfilter 提供导出内容的敏感词过滤：把 XSS 测试样本、危险协议、
// 恶意示例域名、攻击性描述词等容易触发上传平台安全审核的内容替换为占位符，
// 保留代码结构与测试意图说明。独立于 core/redactor（密钥脱敏）：本包只做
// 词语级替换，不处理真实凭据。所有正则均为 RE2（不支持反向引用）。
package wordfilter

import "regexp"

// 档位（tier）：basic < standard < strict，规则按档位向下兼容
// （strict 命中 basic + standard + strict 全部规则）。
const (
	tierBasic = iota
	tierStandard
	tierStrict
)

// rule 一条替换规则：pattern 命中的内容整体替换为 replacement。
type rule struct {
	re          *regexp.Regexp
	replacement string
	tier        int
}

type ruleDef struct {
	pattern     string
	replacement string
	tier        int
}

// ruleDefs 内置规则表（顺序即应用顺序，前面的规则先替换）。
// 占位符用方括号中文标注，明确是脱敏结果而非原始内容。
//   - basic：最保守的核心规则；
//   - standard：覆盖 cheat-sheet 常见事件/属性向量；
//   - strict：宽匹配规则（有误伤可能，如 online=true），仅显式选择。
var ruleDefs = []ruleDef{
	// —— basic：完整脚本块 / 脚本标签（XSS 样本主体）
	{`(?is)<script\b.*?</script>`, `[script 测试样本已脱敏]`, tierBasic},
	{`(?i)<script\b[^>]*>`, `[script 标签已脱敏]`, tierBasic},
	{`(?i)</script>`, `[script 标签已脱敏]`, tierBasic},
	// basic：危险伪协议
	{`(?i)\b(?:javascript|vbscript|livescript|mocha)\s*:`, `[危险协议已脱敏]:`, tierBasic},
	{`(?i)\bdata\s*:\s*text\s*/\s*html`, `[危险协议已脱敏]:text/html`, tierBasic},
	// basic：事件属性（显式列表，避免误伤 online= 等普通配置键）
	{`(?i)\bon(?:error|load|click|abort|blur|change|focus|input|submit|mouse[a-z]+|key[a-z]+)\s*=`, `[事件属性已脱敏]=`, tierBasic},
	// basic：高危标签/属性：iframe、meta http-equiv、base href
	{`(?i)<\s*/?\s*(?:iframe|object|embed|base|meta)\b[^>]*>`, `[标签已脱敏]`, tierBasic},
	{`(?i)\bhttp-equiv\s*=`, `[属性已脱敏]=`, tierBasic},
	{`(?i)\bbase\s+href\s*=`, `[属性已脱敏]=`, tierBasic},
	// basic：经典测试弹窗
	{`(?i)\balert\s*\(\s*1\s*\)`, `[alert 测试样本已脱敏]`, tierBasic},
	// basic：恶意示例域名
	{`(?i)\b(?:evil|malware|attacker|hostile)\.(?:tld|com|net|org|io|cn|example)\b`, `example.com`, tierBasic},
	// basic：英文攻击性/绕过词（测试脚本注释里常见）
	{`(?i)\bhostile\b`, `securityCases`, tierBasic},
	{`(?i)\bbypass\b`, `regressionCases`, tierBasic},
	// basic：中文攻击性描述词（注释里的审核触发词）
	{`恶意\s*payload|恶意载荷|恶意负载`, `测试样本`, tierBasic},
	{`绕过向量|绕过用例`, `边界用例`, tierBasic},
	{`攻击样本|攻击载荷|攻击向量|攻击代码`, `测试样本`, tierBasic},
	{`外发请求`, `网络请求`, tierBasic},
	{`追踪图片`, `图片请求`, tierBasic},
	{`远程追踪`, `远程请求`, tierBasic},
	{`(base\s*)劫持`, `${1}路径解析`, tierBasic},
	{`劫持`, `接管`, tierBasic},

	// —— standard：url() 内的危险协议（修复旧 `style url(` 死规则：
	// 真实代码是 style="...url(..."，中间隔着 ="，按 url( + 协议匹配）
	{`(?i)\burl\s*\(\s*["']?\s*(?:javascript|vbscript)\s*:[^)]*\)?`, `[危险URL已脱敏]`, tierStandard},
	// standard：data: 扩展危险 MIME 家族
	{`(?i)\bdata\s*:\s*(?:text/xml|application/xhtml|image/svg|text/javascript|application/(?:javascript|ecmascript))`, `[危险协议已脱敏]:`, tierStandard},
	// standard：危险属性向量
	{`(?i)\bsrcdoc\s*=`, `[属性已脱敏]=`, tierStandard},
	{`(?i)\bformaction\s*=`, `[属性已脱敏]=`, tierStandard},
	{`(?i)\bxlink:href\s*=\s*["']?\s*(?:javascript|data)`, `[属性已脱敏]=`, tierStandard},
	// standard：缺失的事件属性家族（pointer/drag/animation/transition 等）
	{`(?i)\bon(?:pointer|drag|drop|animation|transition|wheel|scroll|copy|paste|cut|contextmenu|toggle|auxclick|dblclick|play|pause|touch|before[a-z]*)[a-z]*\s*=`, `[事件属性已脱敏]=`, tierStandard},

	// —— strict：宽匹配（≥4 字母，避免 once= 之类普通键误伤）
	{`(?i)\bon[a-z]{4,}\s*=`, `[事件属性已脱敏]=`, tierStrict},
}

// rules 编译好的规则表（顺序即应用顺序）。
var rules = mustCompile(ruleDefs)

// mustCompile 编译规则表；内置规则为编译期常量，失败即 panic（测试兜底）。
func mustCompile(defs []ruleDef) []rule {
	rs := make([]rule, 0, len(defs))
	for _, d := range defs {
		rs = append(rs, rule{re: regexp.MustCompile(d.pattern), replacement: d.replacement, tier: d.tier})
	}
	return rs
}

// Config 过滤配置（对齐 redactor 的 Config/Profile 架构）。
type Config struct {
	// Profile 规则档位："basic" | "standard" | "strict"（空 = standard）。
	Profile string
	// Strategy 处理策略："destroy"（整体替换为占位符，默认）|
	// "neutralize"（仅中和尖括号 < > → ＜ ＞，保留代码可读性）。
	Strategy string
}

// Hit 单条规则的命中统计。
type Hit struct {
	Rule  string
	Count int
}

// tierOf 把档位名解析为数值（空 = standard；未知值同样取 standard）。
func tierOf(profile string) int {
	switch profile {
	case "basic":
		return tierBasic
	case "strict":
		return tierStrict
	default:
		return tierStandard
	}
}

// rulesFor 按档位筛出激活的规则（向下兼容：strict 包含全部低档规则）。
func rulesFor(profile string) []rule {
	tier := tierOf(profile)
	active := make([]rule, 0, len(rules))
	for _, r := range rules {
		if r.tier <= tier {
			active = append(active, r)
		}
	}
	return active
}

// FilterWithStats 应用激活档位的全部规则，返回脱敏后内容与逐规则命中计数。
// 管道位置约定：必须在任何 MaxLineLen 截断之前执行（截断会把 `<script`
// 劈成两半导致块规则漏配），且在 token 统计之前（替换改变字数）。
func FilterWithStats(text string, cfg Config) (string, []Hit) {
	destroy := cfg.Strategy != "neutralize"
	var hits []Hit
	for _, r := range rulesFor(cfg.Profile) {
		locs := r.re.FindAllStringIndex(text, -1)
		if len(locs) == 0 {
			continue
		}
		if destroy {
			text = r.re.ReplaceAllString(text, r.replacement)
		} else {
			text = r.re.ReplaceAllStringFunc(text, neutralize)
		}
		hits = append(hits, Hit{Rule: r.re.String(), Count: len(locs)})
	}
	return text, hits
}

// neutralize 中和策略：把命中片段里的尖括号替换为全角，粘贴到网页端不再
// 构成可执行 HTML，但代码对人和 LLM 仍完整可读。
func neutralize(m string) string {
	out := make([]byte, 0, len(m))
	for i := 0; i < len(m); i++ {
		switch m[i] {
		case '<':
			out = append(out, "＜"...)
		case '>':
			out = append(out, "＞"...)
		default:
			out = append(out, m[i])
		}
	}
	return string(out)
}

// Filter 对文本应用默认档位（standard）的全部敏感词替换规则。
// 幂等：占位符本身不会再被任何规则命中（TestReplacementsAreInert 锁定）。
func Filter(text string) string {
	out, _ := FilterWithStats(text, Config{})
	return out
}
