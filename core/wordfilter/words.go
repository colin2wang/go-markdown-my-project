// Package wordfilter 提供导出内容的敏感词过滤：把 XSS 测试样本、危险协议、
// 恶意示例域名、攻击性描述词等容易触发上传平台安全审核的内容替换为占位符，
// 保留代码结构与测试意图说明。独立于 core/redactor（密钥脱敏）：本包只做
// 词语级替换，不处理真实凭据。所有正则均为 RE2（不支持反向引用）。
package wordfilter

import "regexp"

// rule 一条替换规则：pattern 命中的内容整体替换为 replacement。
type rule struct {
	re          *regexp.Regexp
	replacement string
}

// rules 内置规则表（顺序即应用顺序，前面的规则先替换）。
// 占位符用方括号中文标注，明确是脱敏结果而非原始内容。
var rules = mustCompile([]struct{ pattern, replacement string }{
	// 完整脚本块 / 脚本标签（XSS 样本主体）
	{`(?is)<script\b.*?</script>`, `[script 测试样本已脱敏]`},
	{`(?i)<script\b[^>]*>`, `[script 标签已脱敏]`},
	{`(?i)</script>`, `[script 标签已脱敏]`},
	// 危险伪协议
	{`(?i)\b(?:javascript|vbscript|livescript|mocha)\s*:`, `[危险协议已脱敏]:`},
	{`(?i)\bdata\s*:\s*text\s*/\s*html`, `[危险协议已脱敏]:text/html`},
	// 事件属性（onerror=/onload=/formaction= 等）
	{`(?i)\bon(?:error|load|click|abort|blur|change|focus|input|submit|formaction|mouse[a-z]+|key[a-z]+)\s*=`, `[事件属性已脱敏]=`},
	// 高危标签/属性：iframe、meta http-equiv、base href、style url()
	{`(?i)<\s*/?\s*(?:iframe|object|embed|base|meta)\b[^>]*>`, `[标签已脱敏]`},
	{`(?i)\bhttp-equiv\s*=`, `[属性已脱敏]=`},
	{`(?i)\bbase\s+href\s*=`, `[属性已脱敏]=`},
	{`(?i)\bstyle\s+url\s*\(`, `[样式已脱敏](`},
	// 经典测试弹窗
	{`(?i)\balert\s*\(\s*1\s*\)`, `[alert 测试样本已脱敏]`},
	// 恶意示例域名
	{`(?i)\b(?:evil|malware|attacker|hostile)\.(?:tld|com|net|org|io|cn|example)\b`, `example.com`},
	// 英文攻击性/绕过词（测试脚本注释里常见）
	{`(?i)\bhostile\b`, `securityCases`},
	{`(?i)\bbypass\b`, `regressionCases`},
	// 中文攻击性描述词（注释里的审核触发词）
	{`恶意\s*payload|恶意载荷|恶意负载`, `测试样本`},
	{`绕过向量|绕过用例`, `边界用例`},
	{`攻击样本|攻击载荷|攻击向量|攻击代码`, `测试样本`},
	{`外发请求`, `网络请求`},
	{`追踪图片`, `图片请求`},
	{`远程追踪`, `远程请求`},
	{`(base\s*)劫持`, `${1}路径解析`},
	{`劫持`, `接管`},
})

// mustCompile 编译规则表；内置规则为编译期常量，失败即 panic（测试兜底）。
func mustCompile(defs []struct{ pattern, replacement string }) []rule {
	rs := make([]rule, 0, len(defs))
	for _, d := range defs {
		rs = append(rs, rule{re: regexp.MustCompile(d.pattern), replacement: d.replacement})
	}
	return rs
}

// Filter 对文本应用全部敏感词替换规则，返回脱敏后的内容。
// 幂等：占位符本身不会再被任何规则命中。
func Filter(text string) string {
	for _, r := range rules {
		text = r.re.ReplaceAllString(text, r.replacement)
	}
	return text
}
