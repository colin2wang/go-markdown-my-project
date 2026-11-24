// Package token 提供 Token 估算与按分片装箱（§7.5）。
// 估算精度 ±20%，仅用于容量规划。
package token

import (
	"strings"
	"unicode"
)

// EstimateTokens 估算文本 token 数：
// 中文（CJK）按 1 字 ≈ 1 token，ASCII 按 4 字符 ≈ 1 token，混合分段计算。
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	tokens := 0
	var asciiRun int
	flush := func() {
		tokens += (asciiRun + 3) / 4
		asciiRun = 0
	}
	for _, r := range text {
		if isCJK(r) {
			flush()
			tokens++
		} else if r < 128 {
			asciiRun++
		} else {
			// 其他非 ASCII（emoji、扩展文字等）近似按 1 字 1 token
			flush()
			tokens++
		}
	}
	flush()
	return tokens
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r)
}

// EstimateTokensLines 估算多段文本（拼接后计算）。
func EstimateTokensLines(parts ...string) int {
	return EstimateTokens(strings.Join(parts, "\n"))
}
