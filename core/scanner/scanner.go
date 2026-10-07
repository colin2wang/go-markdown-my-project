// Package scanner 提供文件扫描、过滤与并行读取。
// 语义逐条对齐旧版 Rust 的 src/file_processor.rs。
package scanner

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"go-markdown-my-project/core/logger"
)

// FileResult 对应旧版 Vec<(PathBuf, String)>：文件绝对路径 + 内容。
type FileResult struct {
	FullPath string
	Content  string
}

// ReadFileContent 读取文件内容，对齐 read_file_content。
func ReadFileContent(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %s: %w", filePath, err)
	}
	return string(data), nil
}

// Options 扫描参数，对应 process_files 的入参。
type Options struct {
	ProjectPath        string
	Files              []string
	Directories        []string
	ExcludeDirectories []string
	ExcludePatterns    []string
	MaxFileSize        int64 // 0 = 不限制
}

// ProcessFiles 处理配置指定的文件与目录，返回 (路径, 内容) 列表。
// 注意：结果顺序与旧版一致——先 files 后 directories，目录内并行收集顺序不定，
// 最终由调用方（generator）按 RelPath 排序。
func ProcessFiles(opts Options) ([]FileResult, error) {
	var results []FileResult

	// 逐个处理指定文件
	for _, file := range opts.Files {
		fullPath := filepath.Join(opts.ProjectPath, filepath.FromSlash(file))
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			include, err := ShouldIncludeFile(fullPath, opts.ExcludePatterns, opts.MaxFileSize, opts.ProjectPath)
			if err != nil {
				return nil, err
			}
			if include {
				content, err := ReadFileContent(fullPath)
				if err != nil {
					return nil, err
				}
				results = append(results, FileResult{FullPath: fullPath, Content: content})
			}
		}
	}

	// 递归处理目录
	for _, dir := range opts.Directories {
		fullDir := filepath.Join(opts.ProjectPath, filepath.FromSlash(dir))
		if info, err := os.Stat(fullDir); err == nil && info.IsDir() {
			parallelProcessDirectory(fullDir, &results, opts)
		}
	}

	return results, nil
}

// parallelProcessDirectory 用 goroutine worker pool 并行读取目录内文件，
// 对齐 process_directory_parallel（rayon → worker pool）。
func parallelProcessDirectory(dir string, results *[]FileResult, opts Options) {
	// 先收集全部文件路径（剪枝排除目录）
	var paths []string
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			logger.Warn("walk error", "path", p, "err", err)
			return nil
		}
		if d.IsDir() {
			if ShouldExcludeDirectory(p, opts.ExcludeDirectories) {
				return filepath.SkipDir
			}
			return nil
		}
		paths = append(paths, p)
		return nil
	})
	if err != nil {
		logger.Warn("failed to walk directory", "dir", dir, "err", err)
		return
	}

	// 并行读取：NumCPU 个 worker
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan string, len(paths))
	type item struct {
		path    string
		content string
	}
	out := make(chan item, len(paths))
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				include, err := ShouldIncludeFile(p, opts.ExcludePatterns, opts.MaxFileSize, opts.ProjectPath)
				if err != nil || !include {
					continue
				}
				content, err := ReadFileContent(p)
				if err != nil {
					// 单文件失败仅告警，不中断
					logger.Warn("Failed to read file", "path", p, "err", err)
					continue
				}
				out <- item{path: p, content: content}
			}
		}()
	}
	for _, p := range paths {
		jobs <- p
	}
	close(jobs)
	wg.Wait()
	close(out)

	for it := range out {
		*results = append(*results, FileResult{FullPath: it.path, Content: it.content})
	}
}

// ShouldIncludeFile 判断文件是否应包含：大小限制 → 排除模式。
// 对齐 should_include_file。
func ShouldIncludeFile(filePath string, excludePatterns []string, maxSize int64, projectRoot string) (bool, error) {
	// 大小限制
	if maxSize > 0 {
		info, err := os.Stat(filePath)
		if err != nil {
			return false, fmt.Errorf("failed to get metadata for: %s: %w", filePath, err)
		}
		if info.Size() > maxSize {
			logger.Debug("Skipping large file", "path", filePath, "size", info.Size(), "limit", maxSize)
			return false, nil
		}
	}

	// 排除模式（相对路径，统一 / 分隔）
	rel, err := filepath.Rel(projectRoot, filePath)
	if err != nil {
		rel = filePath
	}
	relSlash := filepath.ToSlash(rel)

	for _, pattern := range excludePatterns {
		if strings.ContainsAny(pattern, "*?") {
			// glob 匹配（对齐 glob::Pattern：对整个相对路径匹配）
			ok, err := matchGlob(pattern, relSlash)
			if err == nil && ok {
				logger.Debug("Skipping file due to pattern", "pattern", pattern, "path", filePath)
				return false, nil
			}
		} else {
			// 精确匹配或前缀匹配（目录）
			if relSlash == pattern || strings.HasPrefix(relSlash, pattern+"/") ||
				relSlash == filepath.ToSlash(pattern) || strings.HasPrefix(relSlash, filepath.ToSlash(pattern)+"/") {
				logger.Debug("Skipping file due to pattern", "pattern", pattern, "path", filePath)
				return false, nil
			}
		}
	}
	return true, nil
}

// matchGlob 使用 path.Match 对整个相对路径匹配（旧版 glob crate 语义近似）。
func matchGlob(pattern, name string) (bool, error) {
	return path.Match(pattern, name)
}

// ShouldExcludeDirectory 判断目录是否应排除，逻辑逐条对齐 should_exclude_directory。
// dir 为当前目录完整路径；模式匹配基于目录名与路径组件。
func ShouldExcludeDirectory(dir string, excludeDirectories []string) bool {
	base := filepath.Base(dir)
	for _, pattern := range excludeDirectories {
		switch {
		case pattern == "**":
			return true
		case strings.HasPrefix(pattern, "**/"):
			nameToExclude := pattern[3:]
			if base == nameToExclude {
				return true
			}
		case strings.Contains(pattern, "/") || strings.Contains(pattern, "\\"):
			// 含路径分隔符：与相对路径比较（旧版以 "." 为根 strip）
			rel, err := filepath.Rel(".", dir)
			if err != nil {
				rel = dir
			}
			if filepath.ToSlash(rel) == filepath.ToSlash(pattern) || rel == filepath.FromSlash(pattern) {
				return true
			}
		default:
			// 纯目录名：匹配当前目录名或任一路径组件
			if base == pattern {
				return true
			}
			// 检查所有父组件
			rest := dir
			for {
				comp := filepath.Base(rest)
				if comp == pattern {
					return true
				}
				parent := filepath.Dir(rest)
				if parent == rest {
					break
				}
				rest = parent
			}
		}
	}
	return false
}

// SortByRelPath 按相对路径排序，对齐旧版 sort_by(|a,b| a.0.cmp(&b.0))
// （旧版比较的是完整路径字符串，此处同样按完整路径排序）。
func SortByRelPath(results []FileResult) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].FullPath < results[j].FullPath
	})
}
