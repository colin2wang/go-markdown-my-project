package redactor

import (
	"regexp"
	"strings"
	"testing"
)

// 每条规则 1 正例 1 负例（§11.1）。
func TestBuiltinRulesPositiveNegative(t *testing.T) {
	cases := []struct {
		rule     string
		positive string
		negative string
	}{
		{"generic-password-assign", `password = "SuperSecret99"`, `password = process.env.PWD`},
		{"aws-access-key", `key = AKIAIOSFODNN7QW3X9ZKB`, `key = BKIAIOSFODNN7QW3X9ZKB`},
		{"aws-secret-key", `aws_secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYQw3x9zKE"`, `aws region = us-east-1`},
		{"github-pat", `token = ghp_0123456789abcdefghijklmnopqrstuvwxyzAB`, `token = ghp_short`},
		{"openai-style", `key = sk-proj-abcdefghijklmnopqrst`, `key = sk-short`},
		{"slack-token", `token = xoxb-123456789012-abcdefghij`, `token = xox-q`},
		{"jwt", `auth = eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJVadQssw5c`, `auth = header.payload`},
		{"pem-block", "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA\n-----END RSA PRIVATE KEY-----", "-----BEGIN CERTIFICATE-----\nMIIE\n-----END CERTIFICATE-----"},
		{"connection-string", `db = postgres://admin:s3cret@db.internal.corp:5432/mydb`, `db = postgres://localhost:5432/mydb`},
		{"htpasswd-like", `Authorization: Bearer AbCdEfGhIjKlMnOpQrStUv==`, `Authorization: Bearer token`},
	}
	cfg := DefaultConfig()
	for _, c := range cases {
		t.Run(c.rule, func(t *testing.T) {
			if got := ScanContent("f.txt", c.positive, cfg); len(got) == 0 {
				t.Errorf("正例未命中: %s", c.positive)
			}
			if got := ScanContent("f.txt", c.negative, cfg); len(got) != 0 {
				t.Errorf("负例误报: %s -> %v", c.negative, got)
			}
		})
	}
}

func TestFalsePositiveSuppress(t *testing.T) {
	cfg := DefaultConfig()
	// env 引用 / placeholder 值不命中
	for _, s := range []string{
		`password = "<your-password-here>"`,
		`api_key = "${API_KEY}"`,
		`password = os.Getenv("PWD")`,
		`secret = changeme`,
	} {
		if got := ScanContent("f.txt", s, cfg); len(got) != 0 {
			t.Errorf("应抑制误报: %s -> %v", s, got)
		}
	}
}

func TestAllowlist(t *testing.T) {
	cfg := DefaultConfig()
	src := `password = "SuperSecret99"`
	findings := ScanContent("a.txt", src, cfg)
	if len(findings) == 0 {
		t.Fatal("应命中")
	}
	f := findings[0]
	// 加入 allowlist 后跳过
	cfg.Allowlist = map[string]bool{f.File + ":" + itoa(f.Line) + ":" + f.Rule: true}
	if got := ScanContent(f.File, src, cfg); len(got) != 0 {
		t.Fatalf("allowlist 未生效: %v", got)
	}
}

func TestRedactPlaceholder(t *testing.T) {
	cfg := DefaultConfig()
	src := "user = admin\npassword = \"SuperSecret99\"\n"
	out, n := RedactContent("f.txt", src, cfg)
	if n != 1 {
		t.Fatalf("替换次数 = %d, want 1", n)
	}
	if strings.Contains(out, "SuperSecret99") {
		t.Fatalf("明文未剔除: %q", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Fatalf("未插入占位符: %q", out)
	}
	// 保留代码结构
	if !strings.Contains(out, "password = ") {
		t.Fatalf("结构被破坏: %q", out)
	}
}

func TestRedactDropLine(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Strategy = "drop_line"
	src := "keep1\npassword = \"SuperSecret99\"\nkeep2\n"
	out, n := RedactContent("f.txt", src, cfg)
	if n != 1 {
		t.Fatalf("替换次数 = %d, want 1", n)
	}
	if strings.Contains(out, "SuperSecret99") || strings.Contains(out, "password") {
		t.Fatalf("整行未删除: %q", out)
	}
	if out != "keep1\nkeep2\n" {
		t.Fatalf("输出不符: %q", out)
	}
}

func TestRedactPemBlock(t *testing.T) {
	cfg := DefaultConfig()
	src := "-----BEGIN PRIVATE KEY-----\nMIIEowAAAAA\n-----END PRIVATE KEY-----\n"
	out, n := RedactContent("f.txt", src, cfg)
	if n != 1 {
		t.Fatalf("替换次数 = %d, want 1", n)
	}
	if strings.Contains(out, "MIIEow") {
		t.Fatalf("密钥体未剔除: %q", out)
	}
	want := "-----BEGIN PRIVATE KEY-----[REDACTED]-----END PRIVATE KEY-----\n"
	if out != want {
		t.Fatalf("输出 = %q, want %q", out, want)
	}
}

func TestCustomPattern(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CustomPatterns = []*regexp.Regexp{regexp.MustCompile(`INTERNAL_[A-Z]{6}`)}
	out, n := RedactContent("f.txt", "code = INTERNAL_SECRET", cfg)
	_ = out
	if n != 1 {
		t.Fatalf("自定义规则未生效，n = %d", n)
	}
}

func TestExportNoPlaintext(t *testing.T) {
	// 模拟导出流：扫描→替换后 grep 不到明文
	cfg := DefaultConfig()
	samples := []string{
		`password = "Hunter2Hunter2"`,
		`token = ghp_0123456789abcdefghijklmnopqrstuvwxyzAB`,
		`key = sk-proj-abcdefghijklmnopqrst`,
	}
	for _, s := range samples {
		out, _ := RedactContent("f.txt", s, cfg)
		if strings.Contains(out, sampleValue(s)) {
			t.Errorf("明文残留: %q", out)
		}
	}
}

// sampleValue 从样例中取出"值"部分用于断言（简化：取引号内或 = 后内容）。
func sampleValue(s string) string {
	if i := strings.Index(s, "= "); i >= 0 {
		v := strings.Trim(strings.TrimSpace(s[i+2:]), `"`)
		return v
	}
	return s
}
