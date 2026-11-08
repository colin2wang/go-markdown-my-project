package infer

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// CollectExtensions 递归收集 roots 下所有文件的扩展名（含点前缀，小写，如 ".java"），
// 去重后按字母序返回；跳过隐藏目录与无扩展名文件，不可读分支静默跳过。
// 供「添加所有扩展名」一键填充包含文件列表。
func CollectExtensions(roots ...string) []string {
	set := map[string]bool{}
	for _, root := range roots {
		if root == "" {
			continue
		}
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // 无权限等：跳过该分支
			}
			if d.IsDir() {
				if d.Name() != "." && strings.HasPrefix(d.Name(), ".") {
					return fs.SkipDir
				}
				return nil
			}
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if ext != "" {
				set[ext] = true
			}
			return nil
		})
	}
	out := make([]string, 0, len(set))
	for e := range set {
		out = append(out, e)
	}
	sort.Strings(out)
	return out
}
