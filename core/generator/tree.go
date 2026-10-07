// Package generator 生成 Markdown 项目文档。
// tree.go 移植旧版 src/tree_generator.rs，输出逐字符一致。
package generator

import (
	"path/filepath"
	"sort"
	"strings"

	"go-markdown-my-project/core/scanner"
)

// directory 对应 Rust struct Directory。
type directory struct {
	name           string
	files          []string
	subdirectories map[string]*directory
	// subKeys 保持插入序无关；输出前按 key 排序（对齐 BTreeMap）
}

func newDirectory(name string) *directory {
	return &directory{name: name, subdirectories: map[string]*directory{}}
}

func (d *directory) addFile(file string) {
	d.files = append(d.files, file)
}

// buildDirectoryTree 对齐 build_directory_tree。
func buildDirectoryTree(results []scanner.FileResult, projectRoot string) *directory {
	root := newDirectory("")
	for _, fr := range results {
		rel, err := filepath.Rel(projectRoot, fr.FullPath)
		if err != nil {
			rel = fr.FullPath
		}
		components := strings.Split(filepath.ToSlash(rel), "/")

		cur := root
		for i, comp := range components {
			if i < len(components)-1 {
				sub, ok := cur.subdirectories[comp]
				if !ok {
					sub = newDirectory(comp)
					cur.subdirectories[comp] = sub
				}
				cur = sub
			} else {
				cur.addFile(comp)
			}
		}
	}
	return root
}

// directoryTreeToString 对齐 directory_tree_to_string（含缩进字符 │ 与 4 空格）。
func directoryTreeToString(dir *directory, indent string, isLast, isRoot bool) string {
	var tree strings.Builder
	if !isRoot {
		connector := "├── "
		if isLast {
			connector = "└── "
		}
		tree.WriteString(indent + connector + dir.name + "/\n")
	}

	// 计算子级缩进
	newIndent := indent
	if !isRoot {
		if isLast {
			newIndent += "    "
		} else {
			newIndent += "│   "
		}
	}

	totalItems := len(dir.files) + len(dir.subdirectories)
	currentItem := 0

	// 先文件
	for _, file := range dir.files {
		currentItem++
		connector := "├── "
		if currentItem == totalItems {
			connector = "└── "
		}
		tree.WriteString(newIndent + connector + file + "\n")
	}

	// 再子目录，按 key 排序（对齐 BTreeMap 迭代序）
	keys := make([]string, 0, len(dir.subdirectories))
	for k := range dir.subdirectories {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		currentItem++
		isLastItem := currentItem == totalItems
		tree.WriteString(directoryTreeToString(dir.subdirectories[k], newIndent, isLastItem, false))
	}

	return tree.String()
}

// GenerateTree 对齐 generate_tree。
func GenerateTree(projectName string, results []scanner.FileResult, projectRoot string) string {
	root := buildDirectoryTree(results, projectRoot)
	root.name = projectName
	return directoryTreeToString(root, "", true, true)
}
