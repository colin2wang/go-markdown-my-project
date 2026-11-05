package compress

import (
	"strings"
	"testing"
)

func joinLines(ls ...string) string { return strings.Join(ls, "\n") + "\n" }

func TestStripComments(t *testing.T) {
	cases := []struct {
		name string
		lang string
		in   []string
		want []string
	}{
		{
			name: "go C-style with strings and TODO",
			lang: "Go",
			in: []string{
				"// header",
				"package main",
				`var s = "http://x//y" // tail`,
				"/* multi",
				"   line */",
				"// TODO: keep me",
				"func main() {}",
			},
			want: []string{
				"",
				"package main",
				`var s = "http://x//y"`,
				"",
				"",
				"// TODO: keep me",
				"func main() {}",
			},
		},
		{
			name: "python shebang docstring and SECURITY",
			lang: "Python",
			in: []string{
				"#!/usr/bin/env python",
				"# normal",
				`s = "# not comment"`,
				`doc = """doc // # text`,
				`still doc"""`,
				"# SECURITY: keep",
				"x = 1  # inline",
			},
			want: []string{
				"#!/usr/bin/env python",
				"",
				`s = "# not comment"`,
				`doc = """doc // # text`,
				`still doc"""`,
				"# SECURITY: keep",
				"x = 1",
			},
		},
		{
			name: "html comment block",
			lang: "HTML",
			in:   []string{"<div><!-- hidden --></div>", "<!-- multi", "line -->", "<p>x</p>"},
			want: []string{"<div></div>", "", "", "<p>x</p>"},
		},
		{
			name: "vue mixed script and template",
			lang: "Vue",
			in: []string{
				"<template><!-- hidden --></template>",
				"<script>",
				"// js comment",
				"const x = 1; /* inline */",
				"</script>",
			},
			want: []string{
				"<template></template>",
				"<script>",
				"",
				"const x = 1;",
				"</script>",
			},
		},
		{
			name: "scss keeps bare url protocol",
			lang: "SCSS",
			in: []string{
				"// normal",
				"a { background: url(http://x/y.png); } // tail",
			},
			want: []string{
				"",
				"a { background: url(http://x/y.png); }",
			},
		},
		{
			name: "powershell hash and block",
			lang: "PowerShell",
			in:   []string{"# normal", "<# block", "comment #>", "Get-Item # tail"},
			want: []string{"", "", "", "Get-Item"},
		},
		{
			name: "batch REM and label comment",
			lang: "Batch",
			in:   []string{"rem lower", "REM upper", ":: label comment", "echo remark"},
			want: []string{"", "", "", "echo remark"},
		},
		{
			name: "sql line and block",
			lang: "SQL",
			in:   []string{"-- c", "SELECT 1; /* b */"},
			want: []string{"", "SELECT 1;"},
		},
		{
			name: "lua long comment",
			lang: "Lua",
			in:   []string{"--[[ block ]]", `print("x -- y")`},
			want: []string{"", `print("x -- y")`},
		},
		{
			name: "haskell block comment",
			lang: "Haskell",
			in:   []string{"{- block -}", `main = putStrLn "-- x"`},
			want: []string{"", `main = putStrLn "-- x"`},
		},
		{
			name: "julia block comment",
			lang: "Julia",
			in:   []string{"#= block =#", "x = 1 # tail"},
			want: []string{"", "x = 1"},
		},
		{
			name: "erlang percent",
			lang: "Erlang",
			in:   []string{"% comment", `io:format("~p", [X]).`},
			want: []string{"", `io:format("~p", [X]).`},
		},
		{
			name: "fortran bang",
			lang: "Fortran",
			in:   []string{"! comment", `print *, "x"`},
			want: []string{"", `print *, "x"`},
		},
		{
			name: "lisp semicolon",
			lang: "Lisp",
			in:   []string{"; comment", `(print "x")`},
			want: []string{"", `(print "x")`},
		},
		{
			name: "vb apostrophe",
			lang: "VB",
			in:   []string{"' comment", `Dim x = "ok"`},
			want: []string{"", `Dim x = "ok"`},
		},
		{
			name: "php attribute is not a comment",
			lang: "PHP",
			in:   []string{"#[Attribute]", "class Foo {}"},
			want: []string{"#[Attribute]", "class Foo {}"},
		},
		{
			name: "markdown keeps headings",
			lang: "Markdown",
			in:   []string{"# Title", "<!-- hidden -->", "text"},
			want: []string{"# Title", "", "text"},
		},
		{
			name: "unknown language untouched",
			lang: "Text",
			in:   []string{"// not stripped", "# nor this"},
			want: []string{"// not stripped", "# nor this"},
		},
		{
			name: "go raw string across lines",
			lang: "Go",
			in: []string{
				"var q = `raw // text",
				"/* not comment */`",
				"// real comment",
			},
			want: []string{
				"var q = `raw // text",
				"/* not comment */`",
				"",
			},
		},
		{
			name: "java javadoc with warning prose is fully stripped",
			lang: "Java",
			in: []string{
				"/**",
				" * 审核意见三层合并主流程（纯确定性、无 LLM）",
				" * 疑似同书异构名只出 warning 不自动合并",
				" *",
				" * @author Wang",
				" * @since 2026-11-04",
				" */",
				"@Slf4j",
				"public class OpinionGrouper {}",
			},
			want: []string{
				"", "", "", "", "", "", "",
				"@Slf4j",
				"public class OpinionGrouper {}",
			},
		},
		{
			name: "multiline block keeps only marker line",
			lang: "Java",
			in: []string{
				"/**",
				" * 说明",
				" * TODO: 修复",
				" */",
				"code();",
			},
			want: []string{
				"/**",
				"",
				" * TODO: 修复",
				"*/",
				"code();",
			},
		},
		{
			name: "single line security block kept",
			lang: "Java",
			in:   []string{"/* SECURITY: keep */", "code();"},
			want: []string{"/* SECURITY: keep */", "code();"},
		},
		{
			name: "license header kept",
			lang: "JavaScript",
			in:   []string{"/* Copyright 2024 Foo. All rights reserved. */", "code();"},
			want: []string{"/* Copyright 2024 Foo. All rights reserved. */", "code();"},
		},
		{
			name: "fold marker kept",
			lang: "Go",
			in:   []string{"// … 12 lines …", "code();"},
			want: []string{"// … 12 lines …", "code();"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := joinLines(tc.in...)
			got := StripComments(tc.lang, in)
			want := joinLines(tc.want...)
			if got != want {
				t.Errorf("StripComments(%q)\n got: %q\nwant: %q", tc.lang, got, want)
			}
			if strings.Count(got, "\n") != strings.Count(in, "\n") {
				t.Errorf("行数变化: got %d, want %d", strings.Count(got, "\n"), strings.Count(in, "\n"))
			}
		})
	}
}

func TestStripCommentsWithOptions(t *testing.T) {
	opt := CommentOptions{KeepMarkers: false}

	in := joinLines(
		"// TODO: keep me",
		"/* Copyright 2024 Foo */",
		"// ordinary",
		"code(); // trailing",
	)
	want := joinLines("", "", "", "code();")
	if got := StripCommentsWithOptions("Go", in, opt); got != want {
		t.Errorf("删除标记注释: got %q, want %q", got, want)
	}

	doc := joinLines("/**", " * 说明", " * TODO: 修复", " */", "code();")
	wantDoc := joinLines("", "", "", "", "code();")
	if got := StripCommentsWithOptions("Java", doc, opt); got != wantDoc {
		t.Errorf("删除跨行块注释: got %q, want %q", got, wantDoc)
	}

	fold := joinLines("// … 12 lines …", "code();")
	if got := StripCommentsWithOptions("Go", fold, opt); got != fold {
		t.Errorf("折叠标记应始终保留: got %q, want %q", got, fold)
	}

	if got := StripCommentsWithOptions("Go", in, DefaultCommentOptions()); got != StripComments("Go", in) {
		t.Errorf("默认选项应等价于 StripComments: got %q", got)
	}
}

func TestStripCommentsCRLF(t *testing.T) {
	in := "// c\r\ncode();\r\n"
	want := "\r\ncode();\r\n"
	if got := StripComments("Go", in); got != want {
		t.Errorf("CRLF: got %q, want %q", got, want)
	}
}

func TestStripCommentsIdempotent(t *testing.T) {
	in := joinLines("// normal", "// TODO: keep", "const s = \"// url\"", "/* block */", "call();")
	once := StripComments("TypeScript", in)
	if twice := StripComments("TypeScript", once); twice != once {
		t.Errorf("重复剥离不是幂等的:\n once: %q\ntwice: %q", once, twice)
	}
}

func TestLineCommentPrefixDerived(t *testing.T) {
	for lang, want := range map[string]string{
		"go": "//", "python": "#", "sql": "--", "batch": "rem", "vb": "'", "haskell": "--",
	} {
		if got := lineComments[lang]; got != want {
			t.Errorf("lineComments[%q] = %q, want %q", lang, got, want)
		}
	}
}
