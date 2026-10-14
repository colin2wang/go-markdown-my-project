package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ExcludeDefaults 全局默认排除规则（config/exclude_defaults.yml）。
// 与项目配置的 exclude_* 合并后由扫描器消费；语义与项目内规则完全一致（RE2 + **/ 前缀）。
type ExcludeDefaults struct {
	Directories []string `yaml:"directories"`
	Files       []string `yaml:"files"`
}

// LoadExcludeDefaults 读取默认排除文件；文件不存在返回 (nil, nil)（视为无默认规则，不报错）。
func LoadExcludeDefaults(path string) (*ExcludeDefaults, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read exclude defaults: %s: %w", path, err)
	}
	var d ExcludeDefaults
	if err := yaml.Unmarshal(content, &d); err != nil {
		return nil, fmt.Errorf("failed to parse exclude defaults: %s: %w", path, err)
	}
	return &d, nil
}
