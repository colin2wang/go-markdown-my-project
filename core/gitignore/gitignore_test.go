package gitignore

import (
	"os"
	"path/filepath"
	"testing"
)

func writeRuleFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func checkIgnored(t *testing.T, m *Matcher, cases []struct {
	rel  string
	want bool
}) {
	t.Helper()
	for _, c := range cases {
		if got := m.Ignored(c.rel, false); got != c.want {
			t.Errorf("Ignored(%q) = %v, want %v", c.rel, got, c.want)
		}
	}
}

func TestMatcherRules(t *testing.T) {
	root := t.TempDir()
	writeRuleFile(t, root, ".gitignore", "# comment\r\n*.log\r\n!keep.log\r\n/dist\r\nbuild/\r\n**/temp/\r\ndocs/**/*.draft.md\r\n")

	m, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	checkIgnored(t, m, []struct {
		rel  string
		want bool
	}{
		{"app.log", true},
		{"sub/app.log", true},
		{"keep.log", false},
		{"sub/keep.log", false},
		{"dist/main.go", true},
		{"other/dist/main.go", false},
		{"build/x.go", true},
		{"a/build/x.go", true},
		{"a/temp/x.go", true},
		{"docs/a.draft.md", true},
		{"docs/guide/a.draft.md", true},
		{"docs/guide/a.md", false},
		{"src/main.go", false},
	})
	if !m.Ignored("dist", true) {
		t.Error("Ignored(\"dist\", true) = false, want true (directory rule)")
	}
}

func TestNestedRulesAndGitDir(t *testing.T) {
	root := t.TempDir()
	writeRuleFile(t, root, ".gitignore", "*.tmp\n!keep.tmp\n")
	writeRuleFile(t, root, "sub/.gitignore", "keep.tmp\n*.md\n")
	// .git 目录内的 .gitignore 不应参与过滤
	writeRuleFile(t, root, ".git/.gitignore", "*\n")

	m, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	checkIgnored(t, m, []struct {
		rel  string
		want bool
	}{
		{"keep.tmp", false},
		{"sub/keep.tmp", true}, // 子目录规则覆盖父目录的 !keep.tmp
		{"notes.md", false},
		{"sub/notes.md", true}, // 子目录规则只作用于 sub/
		{"main.go", false},
	})
}

func TestEscapesAndComments(t *testing.T) {
	root := t.TempDir()
	writeRuleFile(t, root, ".gitignore", "\\#literal\n\\!literal\n#ignored\ntrailing   \n")

	m, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	checkIgnored(t, m, []struct {
		rel  string
		want bool
	}{
		{"#literal", true},
		{"!literal", true},
		{"#ignored", false},
		{"trailing", true},
		{"trailing   ", false},
	})
}

func TestNoRules(t *testing.T) {
	m, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if m.Ignored("src/main.go", false) {
		t.Error("empty project should not ignore any file")
	}
	var nilMatcher *Matcher
	if nilMatcher.Ignored("src/main.go", false) {
		t.Error("nil matcher should not ignore any file")
	}
}
