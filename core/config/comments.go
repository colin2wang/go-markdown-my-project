package config

// 注释模板序列化：按固定字段顺序 + 英文注释模板输出 YAML，
// 与原版项目 yml 风格一致。GUI 实时预览（SerializeYAML）与保存（Save）共用，
// 避免前端手写 YAML 生成逻辑（设计文档 §3.2 / §5）。

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// v2 字段省略阈值：等于默认值时不输出，保持新配置文件与原版字段集一致。
const defaultPlaceholder = "[REDACTED]"

// scalar 构造辅助
func strNode(quoted bool, v string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.ScalarNode, Value: v}
	if quoted {
		n.Style = yaml.DoubleQuotedStyle
	}
	return n
}

func intNode(v int64) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", v)}
}

func boolNode(v bool) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%t", v)}
}

// strSeq 字符串列表；空列表输出 []（Flow 风格），明确语义。
func strSeq(items []string) *yaml.Node {
	seq := &yaml.Node{Kind: yaml.SequenceNode}
	if len(items) == 0 {
		seq.Style = yaml.FlowStyle
		return seq
	}
	for _, it := range items {
		seq.Content = append(seq.Content, strNode(false, it))
	}
	return seq
}

// redactionDefault 判断敏感过滤配置是否全部为默认值（默认值则整个节省略）。
func redactionDefault(r RedactionConfig) bool {
	return r.Enabled &&
		(r.Strategy == "" || r.Strategy == "placeholder") &&
		(r.Placeholder == "" || r.Placeholder == defaultPlaceholder) &&
		len(r.CustomPatterns) == 0 && len(r.Allowlist) == 0
}

// SerializeWithComments 将配置序列化为带注释的 YAML 文本。
func SerializeWithComments(c *ProjectConfig) (string, error) {
	mapping := &yaml.Node{Kind: yaml.MappingNode}
	pair := func(comment, key string, val *yaml.Node) {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key, HeadComment: comment},
			val,
		)
	}

	// 首个键携带文档头注释，输出与原版一致的顶部说明
	nameComment := "Project Configuration for Project Documentation\nName of the project"
	pair(nameComment, "project_name", strNode(true, c.ProjectName))
	pair("Path to the project root directory", "project_path", strNode(true, c.ProjectPath))
	pair("Output file path for the generated documentation", "output_file", strNode(true, c.OutputFile))

	lang := c.MarkdownLang
	if lang == "" {
		lang = "en_us"
	}
	pair(`Markdown output language: "zh_cn" for Chinese, "en_us" for English`, "markdown_lang", strNode(false, lang))

	pair("List of specific files to include", "files", strSeq(c.Files))
	pair("List of directories to include (collected recursively)", "directories", strSeq(c.Directories))
	pair("Directories to exclude (supports **/name)", "exclude_directories", strSeq(c.ExcludeDirectories))
	pair("File patterns to exclude (supports *.log)", "exclude_patterns", strSeq(c.ExcludePatterns))
	if c.MaxFileSize != 0 {
		pair("Maximum file size in bytes (0 = no limit)", "max_file_size", intNode(c.MaxFileSize))
	}

	// ─── v2 字段（仅当非默认值时输出，保持 yml 简洁）───
	if c.ExportMode != "" && c.ExportMode != "full" {
		pair("Export mode: full | files | symbols | signatures | custom", "export_mode", strNode(false, c.ExportMode))
	}
	if c.SplitTokens > 0 {
		pair("Split output into parts of at most this many tokens (0 = no split)", "split_tokens", intNode(int64(c.SplitTokens)))
	}
	if c.Template != "" {
		pair("Custom template name", "template", strNode(false, c.Template))
	}
	if !redactionDefault(c.Redaction) {
		r := &yaml.Node{Kind: yaml.MappingNode}
		rp := func(key string, val *yaml.Node) {
			r.Content = append(r.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, val)
		}
		rp("enabled", boolNode(c.Redaction.Enabled))
		if c.Redaction.Placeholder != "" && c.Redaction.Placeholder != defaultPlaceholder {
			rp("placeholder", strNode(true, c.Redaction.Placeholder))
		}
		if c.Redaction.Strategy != "" && c.Redaction.Strategy != "placeholder" {
			rp("strategy", strNode(false, c.Redaction.Strategy))
		}
		if len(c.Redaction.CustomPatterns) > 0 {
			rp("custom_patterns", strSeq(c.Redaction.CustomPatterns))
		}
		if len(c.Redaction.Allowlist) > 0 {
			rp("allowlist", strSeq(c.Redaction.Allowlist))
		}
		pair("Sensitive information redaction settings", "redaction", r)
	}

	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{mapping}}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return "", fmt.Errorf("serialize config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return "", fmt.Errorf("serialize config: %w", err)
	}
	return strings.TrimRight(buf.String(), "\n") + "\n", nil
}
