package compress

import (
	"strings"
	"testing"
)

func TestMinifySmartPythonKeepsIndent(t *testing.T) {
	src := "def f():\n    if x:\n        return 1\n\n\n    return 2\n"
	want := "def f():\n    if x:\n        return 1\n\n    return 2\n"
	if got := MinifySmart("python", src, 0); got != want {
		t.Errorf("python 不应去缩进:\n got=%q\nwant=%q", got, want)
	}
}

func TestMinifySmartGoStripsIndent(t *testing.T) {
	got := MinifySmart("go", "func f() {\n\tif x {\n\t\treturn 1\n\t}\n}\n", 0)
	if strings.Contains(got, "\t") {
		t.Errorf("go 应去缩进: %q", got)
	}
}

func TestMinifySmartYamlKeepsIndent(t *testing.T) {
	src := "a:\n  b:\n    - c\n"
	if got := MinifySmart("yaml", src, 0); got != src {
		t.Errorf("yaml 不应改动缩进结构:\n got=%q\nwant=%q", got, src)
	}
}

func TestMinifySmartTrimsTrailingForAllLangs(t *testing.T) {
	// 行尾空白裁剪与空行折叠对缩进敏感语言同样生效
	got := MinifySmart("python", "x = 1  \n\n\ny = 2  \n", 0)
	if want := "x = 1\n\ny = 2\n"; got != want {
		t.Errorf("行尾空白/空行处理不符:\n got=%q\nwant=%q", got, want)
	}
}

func TestIsBraceLanguageAliases(t *testing.T) {
	for _, lang := range []string{"go", "golang", "jsx", "mjs", "tsx", "hpp", "objc", "json5", "C++"} {
		if !IsBraceLanguage(lang) {
			t.Errorf("%s 应为大括号语言", lang)
		}
	}
	for _, lang := range []string{"python", "py", "yaml", "yml", "makefile", "haskell", "", "ruby"} {
		if IsBraceLanguage(lang) {
			t.Errorf("%s 不应为大括号语言", lang)
		}
	}
}
