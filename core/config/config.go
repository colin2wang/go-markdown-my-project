// Package config 提供项目配置的加载、校验与保存。
// 语义对齐旧版 Rust 的 src/config.rs，v2 新增字段向后兼容 v1。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"go-markdown-my-project/core/logger"
)

// RedactionConfig 敏感信息过滤配置（F99，v2 新增）。
// json 标签与 yaml 保持一致：Wails 绑定经 encoding/json 序列化到前端。
type RedactionConfig struct {
	Enabled        bool     `yaml:"enabled" json:"enabled"`
	Placeholder    string   `yaml:"placeholder,omitempty" json:"placeholder,omitempty"`
	Strategy       string   `yaml:"strategy,omitempty" json:"strategy,omitempty"`   // placeholder | drop_line
	Allowlist      []string `yaml:"allowlist,omitempty" json:"allowlist,omitempty"` // key = 相对路径:行号:规则名
	CustomPatterns []string `yaml:"custom_patterns,omitempty" json:"custom_patterns,omitempty"`
}

// ProjectConfig 对应 projects/*.yml。
// json 标签与前端 bindings.ts 的蛇形字段一一对应（Wails 经 encoding/json 序列化）。
type ProjectConfig struct {
	// v1 字段
	ProjectName        string   `yaml:"project_name" json:"project_name"`
	ProjectPath        string   `yaml:"project_path" json:"project_path"`
	OutputFile         string   `yaml:"output_file" json:"output_file"`
	MarkdownLang       string   `yaml:"markdown_lang,omitempty" json:"markdown_lang,omitempty"`
	Files              []string `yaml:"files,omitempty" json:"files,omitempty"`
	Directories        []string `yaml:"directories,omitempty" json:"directories,omitempty"`
	ExcludeDirectories []string `yaml:"exclude_directories,omitempty" json:"exclude_directories,omitempty"`
	ExcludePatterns    []string `yaml:"exclude_patterns,omitempty" json:"exclude_patterns,omitempty"`
	MaxFileSize        int64    `yaml:"max_file_size,omitempty" json:"max_file_size,omitempty"` // 0 = 不限制
	// v2 字段
	ExportMode  string          `yaml:"export_mode,omitempty" json:"export_mode,omitempty"` // full | files | symbols | signatures | custom
	Redaction   RedactionConfig `yaml:"redaction,omitempty" json:"redaction,omitempty"`
	SplitTokens int             `yaml:"split_tokens,omitempty" json:"split_tokens,omitempty"`
	Template    string          `yaml:"template,omitempty" json:"template,omitempty"`
}

// supportedOutputExts 对齐 config.rs 的输出扩展名警告列表。
var supportedOutputExts = map[string]bool{
	"md": true, "markdown": true, "html": true, "txt": true,
}

// Load 从 YAML 文件读取并校验配置。
func Load(configPath string) (*ProjectConfig, error) {
	cfg, err := Parse(configPath)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration in: %s: %w", configPath, err)
	}
	return cfg, nil
}

// Parse 仅解析 YAML 不做语义校验（供项目列表展示损坏/路径失效的配置）。
func Parse(configPath string) (*ProjectConfig, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %s: %w", configPath, err)
	}
	cfg := &ProjectConfig{}
	if err := yaml.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration file: %s: %w", configPath, err)
	}
	if cfg.MarkdownLang == "" {
		cfg.MarkdownLang = "en_us"
	}
	return cfg, nil
}

// Validate 校验配置值，语义对齐 config.rs::validate。
func (c *ProjectConfig) Validate() error {
	// 项目名不能为空
	if strings.TrimSpace(c.ProjectName) == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	// 项目路径必须存在且为目录
	if c.ProjectPath == "" {
		return fmt.Errorf("project path cannot be empty")
	}
	info, err := os.Stat(c.ProjectPath)
	if err != nil {
		return fmt.Errorf("project path does not exist: %s", c.ProjectPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("project path is not a directory: %s", c.ProjectPath)
	}

	// 输出文件扩展名警告
	if ext := filepath.Ext(c.OutputFile); ext != "" {
		e := strings.ToLower(strings.TrimPrefix(ext, "."))
		if !supportedOutputExts[e] {
			logger.L().Warn("Output file extension might not be supported", "ext", e)
		}
	}

	// max_file_size 不能为 0（显式写 0 时；省略 = 0 = 不限制，见 YAML 区分）
	if c.MaxFileSize < 0 {
		return fmt.Errorf("max file size cannot be zero")
	}

	// 排除模式不能为空串
	for _, p := range c.ExcludePatterns {
		if strings.TrimSpace(p) == "" {
			return fmt.Errorf("exclude pattern cannot be empty")
		}
	}

	// markdown_lang 默认 en_us
	if c.MarkdownLang == "" {
		c.MarkdownLang = "en_us"
	}
	return nil
}

// Save 将配置写回 YAML 文件（注释模板序列化，与 GUI 预览一致）。
func (c *ProjectConfig) Save(path string) error {
	data, err := SerializeWithComments(c)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}
	return os.WriteFile(path, []byte(data), 0o644)
}
