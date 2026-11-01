package config

// 全局设置（GUI）：独立于项目配置，存 config/global_settings.yml
// （相对 exe 工作目录）。当前包含导出编码与界面语言，可按需扩展。

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// 导出编码常量（OutputEncoding 取值）。
const (
	EncodingUTF8BOM = "utf8_bom" // UTF-8 带 BOM（默认，Windows 查看器识别最好）
	EncodingUTF8    = "utf8"     // UTF-8 无 BOM（跨平台工具链友好）
	EncodingGB18030 = "gb18030"  // 简体中文 Windows 常见编码
	EncodingDefault = EncodingUTF8BOM
)

// SupportedEncodings 输出编码白名单（校验与对话框下拉共用）。
var SupportedEncodings = []string{EncodingUTF8BOM, EncodingUTF8, EncodingGB18030}

// 界面语言常量（InterfaceLanguage 取值）。
const (
	LangAuto      = "auto" // 跟随系统（默认）
	LangAutoValue = "auto"
)

// GlobalSettings 全局设置。
type GlobalSettings struct {
	// OutputEncoding 导出文件编码：utf8_bom | utf8 | gb18030
	OutputEncoding string `yaml:"output_encoding" json:"outputEncoding"`
	// InterfaceLanguage 界面语言：auto | zh_cn | en_us | ja_jp
	InterfaceLanguage string `yaml:"interface_language" json:"interfaceLanguage"`
}

// DefaultGlobalSettings 返回默认全局设置。
func DefaultGlobalSettings() *GlobalSettings {
	return &GlobalSettings{
		OutputEncoding:    EncodingDefault,
		InterfaceLanguage: LangAuto,
	}
}

// normalize 补默认值并校验取值；非法值回退默认并返回错误信息。
func (g *GlobalSettings) normalize() error {
	if g.OutputEncoding == "" {
		g.OutputEncoding = EncodingDefault
	}
	if g.InterfaceLanguage == "" {
		g.InterfaceLanguage = LangAuto
	}
	if !validEncoding(g.OutputEncoding) {
		return fmt.Errorf("invalid output_encoding: %s (supported: %v)", g.OutputEncoding, SupportedEncodings)
	}
	switch g.InterfaceLanguage {
	case LangAuto, "zh_cn", "en_us", "ja_jp":
	default:
		return fmt.Errorf("invalid interface_language: %s", g.InterfaceLanguage)
	}
	return nil
}

func validEncoding(e string) bool {
	for _, s := range SupportedEncodings {
		if e == s {
			return true
		}
	}
	return false
}

// GlobalSettingsPath 返回全局设置文件路径（相对 exe 工作目录）。
func GlobalSettingsPath(configDir string) string {
	return filepath.Join(configDir, "global_settings.yml")
}

// LoadGlobalSettings 读取全局设置；文件不存在返回默认值（不报错）。
func LoadGlobalSettings(configDir string) (*GlobalSettings, error) {
	g := DefaultGlobalSettings()
	path := GlobalSettingsPath(configDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return g, nil
		}
		return nil, fmt.Errorf("failed to read global settings: %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, g); err != nil {
		return nil, fmt.Errorf("failed to parse global settings: %s: %w", path, err)
	}
	if err := g.normalize(); err != nil {
		return nil, fmt.Errorf("invalid global settings: %s: %w", path, err)
	}
	return g, nil
}

// SaveGlobalSettings 写入全局设置（带注释模板，风格同项目 yml）。
func SaveGlobalSettings(configDir string, g *GlobalSettings) error {
	if err := g.normalize(); err != nil {
		return err
	}
	path := GlobalSettingsPath(configDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	text, err := SerializeGlobalSettings(g)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

// SerializeGlobalSettings 序列化为带注释的 YAML 文本（对话框预览与落盘共用）。
func SerializeGlobalSettings(g *GlobalSettings) (string, error) {
	mapping := &yaml.Node{Kind: yaml.MappingNode}
	pair := func(comment, key string, val *yaml.Node) {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key, HeadComment: comment},
			val,
		)
	}
	pair("Export file encoding: utf8_bom | utf8 | gb18030", "output_encoding", strNode(true, g.OutputEncoding))
	pair("Interface language: auto | zh_cn | en_us | ja_jp", "interface_language", strNode(true, g.InterfaceLanguage))
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{mapping}}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return "", fmt.Errorf("failed to marshal global settings: %w", err)
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}
