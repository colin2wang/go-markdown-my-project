// Package generator 按 Mode 生成 Markdown 文档。
// full 模式逐字节对齐旧版 markdown_generator.rs 输出。
package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"go-markdown-my-project/core/scanner"
	"go-markdown-my-project/core/symbol"
)

// ExportMode 导出模式。
type ExportMode string

const (
	ModeFull       ExportMode = "full"
	ModeFiles      ExportMode = "files"
	ModeSymbols    ExportMode = "symbols"
	ModeSignatures ExportMode = "signatures"
	ModeCustom     ExportMode = "custom"
)

// localizedText 对齐 localized_text，保留 zh_cn/en_us 双语。
func localizedText(key, lang string) string {
	zh := map[string]string{
		"project_documentation": "项目文档",
		"project_file_tree":     "项目文件树",
		"project_files":         "项目文件",
		"file_label":            "文件",
		"file_manifest":         "文件清单",
		"symbol_directory":      "符号目录",
		"api_signatures":        "API 签名",
		"symbols_count":         "个符号",
		"no_symbols":            "未识别到符号",
	}
	if lang == "zh_cn" {
		if v, ok := zh[key]; ok {
			return v
		}
	} else {
		switch key {
		case "project_documentation":
			return "Project Documentation"
		case "project_file_tree":
			return "Project File Tree"
		case "project_files":
			return "Project Files"
		case "file_label":
			return "File"
		case "file_manifest":
			return "File Manifest"
		case "symbol_directory":
			return "Symbol Directory"
		case "api_signatures":
			return "API Signatures"
		case "symbols_count":
			return "symbols"
		case "no_symbols":
			return "no symbols recognized"
		}
	}
	return key
}

// LoadLanguages 读取 assets/langs.yml：扩展名 → 语言名（对齐 languages.yml 格式）。
func LoadLanguages(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Languages map[string]string `yaml:"languages"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	// 统一小写扩展名 key
	m := make(map[string]string, len(doc.Languages))
	for k, v := range doc.Languages {
		m[strings.ToLower(k)] = v
	}
	return m, nil
}

// LanguageFor 返回扩展名对应语言，默认 "Text"（供绑定层复用）。
func LanguageFor(languages map[string]string, filePath string) string {
	return languageFor(languages, filePath)
}

// languageFor 返回扩展名对应语言，默认 "Text"（对齐 markdown_generator.rs）。
func languageFor(languages map[string]string, filePath string) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filePath)), ".")
	if lang, ok := languages[ext]; ok {
		return lang
	}
	return "Text"
}

// Options 生成选项。
type Options struct {
	ProjectName string
	ProjectRoot string
	Lang        string // zh_cn | en_us
	Mode        ExportMode
	Languages   map[string]string
	// symbols / signatures 模式子选项（对齐设计文档 §2/§9）
	IncludeLineNumbers bool
	MaxSignatureLen    int
}

// Generate 按模式生成完整 Markdown 内容。
// 文件先按完整路径排序（对齐 sorted_files.sort_by(a.0.cmp(b.0))）。
func Generate(results []scanner.FileResult, opts Options) (string, error) {
	sorted := make([]scanner.FileResult, len(results))
	copy(sorted, results)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].FullPath < sorted[j].FullPath
	})

	switch opts.Mode {
	case ModeFiles:
		return generateFileList(sorted, opts), nil
	case ModeSymbols:
		return generateSymbols(sorted, opts), nil
	case ModeSignatures:
		return generateSignatures(sorted, opts), nil
	case ModeFull, "":
		return generateFull(sorted, opts), nil
	default:
		return "", fmt.Errorf("unknown export mode: %s", opts.Mode)
	}
}

// generateFull 对齐 generate_markdown 的完整输出。
func generateFull(sorted []scanner.FileResult, opts Options) string {
	title := localizedText("project_documentation", opts.Lang)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s for %s\n\n", title, opts.ProjectName)

	treeHeading := localizedText("project_file_tree", opts.Lang)
	fmt.Fprintf(&b, "## %s\n\n", treeHeading)
	b.WriteString("```\n")
	fmt.Fprintf(&b, "%s\n", opts.ProjectName)
	b.WriteString(GenerateTree(opts.ProjectName, sorted, opts.ProjectRoot))
	b.WriteString("```\n\n")

	filesHeading := localizedText("project_files", opts.Lang)
	fileLabel := localizedText("file_label", opts.Lang)
	fmt.Fprintf(&b, "## %s\n\n", filesHeading)
	for _, fr := range sorted {
		// 对齐旧版：display_path 用原生分隔符（Windows 下为 \），保证 byte-equal
		rel := relPathNative(fr.FullPath, opts.ProjectRoot)
		lang := languageFor(opts.Languages, fr.FullPath)
		fmt.Fprintf(&b, "### %s: `%s`\n\n```%s\n%s\n```\n\n", fileLabel, rel, lang, fr.Content)
	}
	return b.String()
}

// generateFileList 生成 F2 文件清单模式：树 + 平铺路径 + 行数/大小。
func generateFileList(sorted []scanner.FileResult, opts Options) string {
	title := localizedText("project_documentation", opts.Lang)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s for %s\n\n", title, opts.ProjectName)

	treeHeading := localizedText("project_file_tree", opts.Lang)
	fmt.Fprintf(&b, "## %s\n\n", treeHeading)
	b.WriteString("```\n")
	fmt.Fprintf(&b, "%s\n", opts.ProjectName)
	b.WriteString(GenerateTree(opts.ProjectName, sorted, opts.ProjectRoot))
	b.WriteString("```\n\n")

	manifestHeading := localizedText("file_manifest", opts.Lang)
	fmt.Fprintf(&b, "## %s\n\n", manifestHeading)
	for _, fr := range sorted {
		rel := relPath(fr.FullPath, opts.ProjectRoot)
		lines := strings.Count(fr.Content, "\n")
		if len(fr.Content) > 0 && !strings.HasSuffix(fr.Content, "\n") {
			lines++
		}
		lang := languageFor(opts.Languages, fr.FullPath)
		fmt.Fprintf(&b, "- `%s` — %s, %d lines, %d bytes\n", rel, lang, lines, len(fr.Content))
	}
	b.WriteString("\n")
	return b.String()
}

// generateSymbols 生成符号目录（设计文档 §6.1）：树形缩进，类型符号为父、方法缩进两格。
func generateSymbols(sorted []scanner.FileResult, opts Options) string {
	title := localizedText("project_documentation", opts.Lang)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s for %s\n\n", title, opts.ProjectName)

	treeHeading := localizedText("project_file_tree", opts.Lang)
	fmt.Fprintf(&b, "## %s\n\n", treeHeading)
	b.WriteString("```\n")
	fmt.Fprintf(&b, "%s\n", opts.ProjectName)
	b.WriteString(GenerateTree(opts.ProjectName, sorted, opts.ProjectRoot))
	b.WriteString("```\n\n")

	symDir := localizedText("symbol_directory", opts.Lang)
	fmt.Fprintf(&b, "## %s\n\n", symDir)

	cfg := symbol.Config{IncludeLineNumbers: opts.IncludeLineNumbers, MaxSignatureLen: opts.MaxSignatureLen, ComputeSignatures: false}
	for _, fr := range sorted {
		rel := relPath(fr.FullPath, opts.ProjectRoot)
		lang := languageFor(opts.Languages, fr.FullPath)
		syms := symbol.ExtractConfig(lang, fr.Content, cfg)
		if len(syms) == 0 {
			fmt.Fprintf(&b, "### %s （%s）\n\n", rel, localizedText("no_symbols", opts.Lang))
			continue
		}
		fmt.Fprintf(&b, "### %s （%d %s）\n", rel, len(syms), localizedText("symbols_count", opts.Lang))
		writeSymbolTree(&b, syms, opts.IncludeLineNumbers)
		b.WriteString("\n")
	}
	return b.String()
}

// generateSignatures 生成 API 签名（设计文档 §6.2）：每文件一个语言代码块，方法缩进 4 格。
func generateSignatures(sorted []scanner.FileResult, opts Options) string {
	title := localizedText("project_documentation", opts.Lang)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s for %s\n\n", title, opts.ProjectName)

	treeHeading := localizedText("project_file_tree", opts.Lang)
	fmt.Fprintf(&b, "## %s\n\n", treeHeading)
	b.WriteString("```\n")
	fmt.Fprintf(&b, "%s\n", opts.ProjectName)
	b.WriteString(GenerateTree(opts.ProjectName, sorted, opts.ProjectRoot))
	b.WriteString("```\n\n")

	sigHeading := localizedText("api_signatures", opts.Lang)
	fmt.Fprintf(&b, "## %s\n\n", sigHeading)

	cfg := symbol.Config{IncludeLineNumbers: opts.IncludeLineNumbers, MaxSignatureLen: opts.MaxSignatureLen, ComputeSignatures: true}
	for _, fr := range sorted {
		rel := relPath(fr.FullPath, opts.ProjectRoot)
		lang := languageFor(opts.Languages, fr.FullPath)
		syms := symbol.ExtractConfig(lang, fr.Content, cfg)
		if len(syms) == 0 {
			continue // 签名模式跳过无符号文件（设计文档 §10）
		}
		fmt.Fprintf(&b, "### %s\n\n", rel)
		codeLang := strings.ToLower(lang)
		if codeLang == "" || codeLang == "text" {
			codeLang = "text"
		}
		fmt.Fprintf(&b, "```%s\n", codeLang)
		commentPrefix := "//"
		if codeLang == "python" || codeLang == "py" {
			commentPrefix = "#"
		}
		for _, s := range syms {
			indent := ""
			if s.Parent != "" {
				indent = "    "
			}
			sig := s.Name
			if s.Signature != "" {
				sig = s.Signature
			}
			fmt.Fprintf(&b, "%s%s %s L%d\n", indent, sig, commentPrefix, s.Line)
		}
		b.WriteString("```\n\n")
	}
	return b.String()
}

// writeSymbolTree 按 Parent 链接渲染符号树（设计文档 §6.1）。
func writeSymbolTree(b *strings.Builder, syms []symbol.Symbol, withLine bool) {
	children := map[string][]symbol.Symbol{}
	topNames := map[string]bool{}
	var top []symbol.Symbol
	for _, s := range syms {
		if s.Parent == "" {
			top = append(top, s)
			topNames[s.Name] = true
		} else {
			children[s.Parent] = append(children[s.Parent], s)
		}
	}
	for _, s := range top {
		emitSymbolLine(b, s, "", withLine)
		for _, c := range children[s.Name] {
			emitSymbolLine(b, c, "  ", withLine)
		}
	}
	// 父级未作为顶级出现的孤儿子符号（如仅压栈的 impl 目标类型），平铺输出
	for _, s := range syms {
		if s.Parent != "" && !topNames[s.Parent] {
			emitSymbolLine(b, s, "", withLine)
		}
	}
}

func emitSymbolLine(b *strings.Builder, s symbol.Symbol, indent string, withLine bool) {
	if withLine {
		fmt.Fprintf(b, "%s- [%s] %s · L%d\n", indent, s.Kind, s.Name, s.Line)
	} else {
		fmt.Fprintf(b, "%s- [%s] %s\n", indent, s.Kind, s.Name)
	}
}

// relPath 相对路径，统一 / 分隔落盘（§12 跨平台约定）。
func relPath(fullPath, projectRoot string) string {
	rel, err := filepath.Rel(projectRoot, fullPath)
	if err != nil {
		return filepath.ToSlash(fullPath)
	}
	return filepath.ToSlash(rel)
}

// relPathNative 相对路径，保留原生分隔符（对齐旧版 relative_path.display()）。
func relPathNative(fullPath, projectRoot string) string {
	rel, err := filepath.Rel(projectRoot, fullPath)
	if err != nil {
		return fullPath
	}
	return rel
}

// WriteOutput 将内容写入 outputDir/outputFile（对齐 main.rs：output_dir.join(config.output_file)）。
func WriteOutput(outputDir, outputFile, content string) (string, error) {
	out := filepath.Join(outputDir, filepath.FromSlash(outputFile))
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(out, []byte(content), 0o644); err != nil {
		return "", err
	}
	return out, nil
}
