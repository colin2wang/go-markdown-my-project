// Package gitignore 解析项目内所有 .gitignore 并按 Git 通配规则判定路径是否被忽略。
// 纯 Go 实现，不依赖 Wails；由 core/scanner 在文件/目录过滤时调用。
package gitignore

import (
	"bufio"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Rule 是一条已解析的 .gitignore 规则。
// base 为规则所在目录相对项目根的 slash 路径（"" 表示项目根）；
// anchored 为 true 时 pattern 相对 base 匹配，否则匹配路径中的任意后缀（任意层级生效）。
type Rule struct {
	pattern  string
	base     string
	negate   bool
	dirOnly  bool
	anchored bool
}

// Matcher 汇总项目中所有 .gitignore 规则，规则按「目录深度 + 文件内行序」排列，
// 越靠后的规则优先级越高（后写法覆盖前写法、子目录覆盖父目录）。
type Matcher struct {
	rules []Rule
}

// Load 递归查找 root 下所有 .gitignore 并解析规则。
// .git 目录与已被上层规则忽略的目录不再深入；单个文件读取失败仅跳过该文件。
func Load(root string) (*Matcher, error) {
	m := &Matcher{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && d.Name() == ".git" {
				return filepath.SkipDir
			}
			if p != root {
				rel, err := filepath.Rel(root, p)
				if err == nil && m.Ignored(filepath.ToSlash(rel), true) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.EqualFold(d.Name(), ".gitignore") {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		base := filepath.ToSlash(filepath.Dir(rel))
		if base == "." {
			base = ""
		}
		rules, err := parseFile(p, base)
		if err != nil {
			return nil
		}
		m.rules = append(m.rules, rules...)
		return nil
	})
	// 稳定排序：父目录规则在前、子目录规则在后，同目录内保持文件行序。
	sort.SliceStable(m.rules, func(i, j int) bool {
		return ruleDepth(m.rules[i].base) < ruleDepth(m.rules[j].base)
	})
	return m, err
}

func ruleDepth(base string) int {
	if base == "" {
		return 0
	}
	return strings.Count(base, "/") + 1
}

// parseFile 解析单个 .gitignore：注释/空行、! 否定、尾随 /（仅目录）、
// 前导 / 与含 / 的锚定、** 跨层、\# \! 转义、未转义的尾随空格。
func parseFile(file, base string) ([]Rule, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var rules []Rule
	sc := bufio.NewScanner(f)
	firstLine := true
	for sc.Scan() {
		line := strings.TrimSuffix(sc.Text(), "\r")
		if firstLine {
			line = strings.TrimPrefix(line, "\ufeff")
			firstLine = false
		}
		line = trimTrailingSpaces(line)
		if line == "" || line[0] == '#' {
			continue
		}
		if strings.HasPrefix(line, `\#`) {
			line = line[1:]
		}
		r := Rule{base: base}
		if line[0] == '!' {
			r.negate = true
			line = line[1:]
		} else if strings.HasPrefix(line, `\!`) {
			line = line[1:]
		}
		if line == "" {
			continue
		}
		if strings.HasSuffix(line, "/") && !strings.HasSuffix(line, `\/`) {
			r.dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}
		switch {
		case strings.HasPrefix(line, "**/"):
			// "**/foo" 与 "foo" 一样作用于任意层级
			line = strings.TrimPrefix(line, "**/")
		case strings.HasPrefix(line, "/"):
			r.anchored = true
			line = strings.TrimPrefix(line, "/")
		case strings.Contains(line, "/"):
			r.anchored = true
		}
		if line == "" {
			continue
		}
		r.pattern = line
		rules = append(rules, r)
	}
	return rules, sc.Err()
}

// trimTrailingSpaces 删除未转义的尾随空格（"\ " 表示字面空格，予以保留）。
func trimTrailingSpaces(s string) string {
	for strings.HasSuffix(s, " ") {
		i := len(s) - 2
		slashes := 0
		for i >= 0 && s[i] == '\\' {
			slashes++
			i--
		}
		if slashes%2 == 1 {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}

// Ignored 判断相对项目根的路径是否被忽略。relSlash 使用 "/" 分隔；
// isDir 表示该路径是目录（目录被忽略时其所有后代一并忽略，与 Git 一致）。
func (m *Matcher) Ignored(relSlash string, isDir bool) bool {
	if m == nil || len(m.rules) == 0 {
		return false
	}
	rel := path.Clean(filepath.ToSlash(relSlash))
	if rel == "." || rel == "" || rel == ".." || strings.HasPrefix(rel, "../") {
		return false
	}
	parts := strings.Split(rel, "/")
	ignored := false
	for i := range parts {
		cur := strings.Join(parts[:i+1], "/")
		curIsDir := isDir || i < len(parts)-1
		// 逐层判定：祖先目录被忽略时整体忽略；后写/更深层的否定规则可重新包含。
		if r, ok := m.lastMatch(cur, curIsDir); ok {
			ignored = !r.negate
			if ignored {
				return true
			}
		}
	}
	return ignored
}

// lastMatch 返回优先级最高（rules 中最后一条）的匹配规则。
func (m *Matcher) lastMatch(rel string, isDir bool) (Rule, bool) {
	var last Rule
	found := false
	for _, r := range m.rules {
		if r.matches(rel, isDir) {
			last, found = r, true
		}
	}
	return last, found
}

func (r Rule) matches(rel string, isDir bool) bool {
	if r.dirOnly && !isDir {
		return false
	}
	if r.base != "" {
		if !strings.HasPrefix(rel, r.base+"/") {
			return false
		}
		rel = rel[len(r.base)+1:]
	}
	if rel == "" {
		return false
	}
	if r.anchored {
		return matchGlob(r.pattern, rel)
	}
	// 非锚定规则：对 rel 的任意后缀匹配，使 "*.log" / "build/" 在任意层级生效。
	parts := strings.Split(rel, "/")
	for i := range parts {
		if matchGlob(r.pattern, strings.Join(parts[i:], "/")) {
			return true
		}
	}
	return false
}

// matchGlob 按 "/" 分段匹配；"**" 段可跨越任意层数，其余段用 path.Match。
func matchGlob(pattern, name string) bool {
	return matchParts(strings.Split(pattern, "/"), strings.Split(name, "/"))
}

func matchParts(pat, parts []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			if len(pat) == 1 {
				return true
			}
			for i := 0; i <= len(parts); i++ {
				if matchParts(pat[1:], parts[i:]) {
					return true
				}
			}
			return false
		}
		if len(parts) == 0 {
			return false
		}
		ok, err := path.Match(pat[0], parts[0])
		if err != nil || !ok {
			return false
		}
		pat, parts = pat[1:], parts[1:]
	}
	return len(parts) == 0
}
