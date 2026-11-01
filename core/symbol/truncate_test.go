package symbol

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// 签名截断必须按 rune：按字节切会把 UTF-8 多字节字符拦腰截断，
// 产出非法 UTF-8，导致 Markdown 查看器渲染乱码（signatures 模式回归）。
func TestExtractConfigTruncateSignatureRuneSafe(t *testing.T) {
	// 声明行含大量中文：字节长度远超 200，且第 200 字节落在多字节字符中间
	decl := "func " + strings.Repeat("名", 90) + "(x int) {"
	content := "package demo\n\n" + decl + "\n}\n"

	syms := ExtractConfig("go", content, Config{
		IncludeLineNumbers: true, MaxSignatureLen: 200, ComputeSignatures: true,
	})
	if len(syms) == 0 {
		t.Fatal("no symbols extracted")
	}
	for _, s := range syms {
		if !utf8.ValidString(s.Signature) {
			t.Fatalf("signature contains invalid UTF-8 (byte-truncated?): %q", s.Signature)
		}
	}
}
