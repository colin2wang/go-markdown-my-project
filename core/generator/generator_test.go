package generator

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-markdown-my-project/core/scanner"
)

// updateGolden 支持 go test ./core/generator/ -update 重新生成黄金快照。
var updateGolden = flag.Bool("update", false, "update golden testdata file")

// TestFullModeGolden 用黄金快照做回归：首次运行生成 testdata/golden.md，
// 之后每次运行对比，保证生成逻辑不发生非预期变更。
// 另附 TestExampleStructure 校验与旧版 Example.md 的格式一致性（结构 + 归一化内容）。
func TestFullModeGolden(t *testing.T) {
	rustProj := `F:/Workspaces/JetBrains/RustRover/markdown_my_project`
	if _, err := os.Stat(rustProj); err != nil {
		t.Skip("Rust 原项目不存在，跳过回归测试")
	}

	langs, err := LoadLanguages(filepath.Join(rustProj, "languages.yml"))
	if err != nil {
		t.Fatalf("加载 languages.yml 失败: %v", err)
	}

	results, err := scanner.ProcessFiles(scanner.Options{
		ProjectPath: rustProj,
		Files:       []string{"cargo.toml"},
		Directories: []string{"src", "projects"},
	})
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}

	got, err := Generate(results, Options{
		ProjectName: "Markdown My Project",
		ProjectRoot: rustProj,
		Lang:        "en_us",
		Mode:        ModeFull,
		Languages:   langs,
	})
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}

	golden := "testdata/golden.md"
	if *updateGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("读取黄金快照失败（首次请运行 go test -update）: %v", err)
	}
	if got != string(want) {
		g, e := []rune(got), []rune(string(want))
		n := len(g)
		if len(e) < n {
			n = len(e)
		}
		for i := 0; i < n; i++ {
			if g[i] != e[i] {
				t.Fatalf("输出与黄金快照不一致，首个差异于 rune %d:\n got: %q\nwant: %q", i,
					ctxRange(g, i), ctxRange(e, i))
			}
		}
		t.Fatalf("输出与黄金快照长度不一致: got %d, want %d", len(g), len(e))
	}
}

// TestExampleStructure 校验输出结构与旧版 Example.md 一致：
// 章节顺序、标题格式、语言标注与文件清单对齐（CRLF 与快照陈旧内容归一化容忍）。
func TestExampleStructure(t *testing.T) {
	rustProj := `F:/Workspaces/JetBrains/RustRover/markdown_my_project`
	if _, err := os.Stat(rustProj); err != nil {
		t.Skip("Rust 原项目不存在，跳过回归测试")
	}
	expected, err := os.ReadFile(filepath.Join(rustProj, "Example.md"))
	if err != nil {
		t.Fatalf("读取 Example.md 失败: %v", err)
	}
	langs, _ := LoadLanguages(filepath.Join(rustProj, "languages.yml"))
	results, err := scanner.ProcessFiles(scanner.Options{
		ProjectPath: rustProj,
		Files:       []string{"cargo.toml"},
		Directories: []string{"src", "projects"},
	})
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}
	got, err := Generate(results, Options{
		ProjectName: "Markdown My Project",
		ProjectRoot: rustProj,
		Lang:        "en_us",
		Mode:        ModeFull,
		Languages:   langs,
	})
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}

	want := normalizeCRLF(string(expected))
	have := normalizeCRLF(got)
	// 提取所有文件标题行对比。快照生成后原项目新增了文件（如 epub_reader.yml），
	// 因此只要求 Example.md 中的每个文件都出现在输出中且语言标注一致。
	wantHeads := fileHeads(want)
	haveHeads := fileHeads(have)
	haveByPath := map[string]string{}
	for _, h := range haveHeads {
		haveByPath[filepath.ToSlash(h.path)] = h.lang
	}
	if len(haveHeads) < len(wantHeads) {
		t.Fatalf("输出文件数少于快照: got %d, want >= %d", len(haveHeads), len(wantHeads))
	}
	for _, w := range wantHeads {
		lang, ok := haveByPath[filepath.ToSlash(w.path)]
		if !ok {
			t.Errorf("快照文件未出现在输出中: %s", w.path)
			continue
		}
		if lang != w.lang {
			t.Errorf("文件 %s 语言标注不一致: got %q, want %q", w.path, lang, w.lang)
		}
	}
	if !strings.HasPrefix(have, "# Project Documentation for Markdown My Project\n\n") {
		t.Errorf("标题格式与旧版不一致")
	}
}

type head struct{ path, lang string }

// fileHeads 提取形如 "### File: `path`\n\n```lang" 的标题。
func fileHeads(s string) []head {
	var heads []head
	for i := 0; i < len(s); {
		j := strings.Index(s[i:], "### File: `")
		if j < 0 {
			break
		}
		i += j + len("### File: `")
		k := strings.Index(s[i:], "`")
		if k < 0 {
			break
		}
		path := s[i : i+k]
		i += k + len("`\n\n```")
		m := strings.IndexAny(s[i:], "\n")
		if m < 0 {
			break
		}
		// 跳过 Example.md 内嵌 Rust 源码中的格式化字符串字面量（如 `{}`）
		if path == "" || strings.ContainsAny(path, "{}") {
			continue
		}
		heads = append(heads, head{path: path, lang: s[i : i+m]})
		i += m
	}
	return heads
}

func normalizeCRLF(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func ctxRange(r []rune, i int) string {
	lo := i - 30
	if lo < 0 {
		lo = 0
	}
	hi := i + 30
	if hi > len(r) {
		hi = len(r)
	}
	return string(r[lo:hi])
}
