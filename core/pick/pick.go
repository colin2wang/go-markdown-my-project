// Package pick 提供文件重要性评分与预算内挑选（导出「代码选择逻辑」）。
//
// 目标：用更少的 token 承载更多的有效信息——优先保留入口、构建清单与源码，
// 压低测试、生成代码、锁文件与资源的权重，并在给定 token 预算内挑选文件。
// 纯文本启发式（无 AST）：允许偶发误判，但保证排序稳定、结果可解释。
package pick

import (
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Class 文件类别（决定评分与是否建议默认排除）。
type Class string

// 文件类别。
const (
	ClassEntry     Class = "entry"     // README / LICENSE / CHANGELOG
	ClassBuild     Class = "build"     // 构建与依赖清单（go.mod、package.json…）
	ClassConfig    Class = "config"    // 配置（yaml/toml/ini…）
	ClassSource    Class = "source"    // 源码
	ClassTest      Class = "test"      // 测试代码
	ClassGenerated Class = "generated" // 生成代码
	ClassLock      Class = "lock"      // 依赖锁文件
	ClassAsset     Class = "asset"     // 图片/字体/音视频
	ClassOther     Class = "other"
)

// Candidate 参与挑选的文件候选。
type Candidate struct {
	Path      string  // 相对路径（/ 分隔）
	SizeBytes int64   // 文件字节数
	Tokens    int     // 估算 token 数（由调用方填充）
	Score     float64 // 重要性评分（越大越优先）
	Class     Class   // 文件类别
}

// entryNames 入口文档文件名前缀（README/LICENSE/CHANGELOG/CONTRIBUTING…）。
var entryNames = map[string]bool{
	"readme": true, "readme.md": true, "license": true, "changelog": true,
	"contributing": true, "architecture": true, "design": true, "todo": true,
}

// buildFiles 构建与依赖清单（项目结构与依赖的权威来源，信息密度最高）。
var buildFiles = map[string]bool{
	"go.mod": true, "cargo.toml": true, "package.json": true, "pyproject.toml": true,
	"setup.py": true, "setup.cfg": true, "requirements.txt": true, "pom.xml": true,
	"build.gradle": true, "build.gradle.kts": true, "settings.gradle": true,
	"makefile": true, "cmakelists.txt": true, "dockerfile": true, "docker-compose.yml": true,
	"docker-compose.yaml": true, "tsconfig.json": true, "vite.config.ts": true,
	"vite.config.js": true, "webpack.config.js": true, "rollup.config.js": true,
	"composer.json": true, "gemfile": true, "build.ps1": true, "build.sh": true,
	"justfile": true, "taskfile.yml": true, "meson.build": true, "sconstruct": true,
}

// buildExts 构建脚本扩展名（.csproj/.sln/.gradle/.props 等）。
var buildExts = map[string]bool{
	".csproj": true, ".sln": true, ".fsproj": true, ".vbproj": true,
	".gradle": true, ".props": true, ".targets": true, ".cmake": true,
	".bazel": true, ".bzl": true, ".mk": true, ".ninja": true,
}

// lockNames 依赖锁文件（体积大、信息密度低，默认建议排除）。
var lockNames = map[string]bool{
	"go.sum": true, "cargo.lock": true, "package-lock.json": true,
	"yarn.lock": true, "pnpm-lock.yaml": true, "poetry.lock": true,
	"composer.lock": true, "gemfile.lock": true, "flake.lock": true,
	"pipfile.lock": true, "bun.lockb": true,
}

// generatedSuffixes 生成代码后缀。
var generatedSuffixes = []string{
	".pb.go", "_pb2.py", "_pb2_grpc.py", ".designer.cs", ".g.cs",
	".generated.go", ".generated.ts", ".generated.js", ".g.dart",
	".freezed.dart", ".min.js", ".min.css", ".map",
}

// generatedDirs 生成物目录名（出现在路径中即判定）。
var generatedDirs = map[string]bool{
	"generated": true, "gen": true, ".terraform": true, "pb": true,
}

// assetExts 资源类扩展名（图片/字体/音视频/归档）。
var assetExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true,
	".ico": true, ".svg": true, ".webp": true, ".avif": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true, ".otf": true,
	".mp3": true, ".mp4": true, ".wav": true, ".avi": true, ".mov": true,
	".zip": true, ".tar": true, ".gz": true, ".7z": true, ".rar": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true,
}

// configExts 配置文件扩展名。
var configExts = map[string]bool{
	".yml": true, ".yaml": true, ".toml": true, ".ini": true, ".cfg": true,
	".conf": true, ".env": true, ".properties": true, ".json": true,
}

// testDirs 测试目录名。
var testDirs = map[string]bool{
	"test": true, "tests": true, "__tests__": true, "testdata": true,
	"spec": true, "specs": true, "e2e": true, "fixtures": true, "mocks": true,
}

// sourceDirs 常见源码目录名（用于目录打分）。
var sourceDirs = map[string]bool{
	"src": true, "app": true, "lib": true, "libs": true, "cmd": true,
	"internal": true, "pkg": true, "core": true, "api": true, "apis": true,
	"server": true, "client": true, "web": true, "frontend": true, "backend": true,
	"components": true, "models": true, "controllers": true, "handlers": true,
	"services": true, "routes": true, "utils": true, "scripts": true,
	"include": true, "sources": true, "main": true, "java": true, "kotlin": true,
}

// lowValueDirs 低信息密度目录（文档/示例/示例数据）。
var lowValueDirs = map[string]bool{
	"docs": true, "doc": true, "examples": true, "example": true, "samples": true,
	"vendor": true, "third_party": true, "thirdparty": true, "assets": true,
	"static": true, "public": true, "resources": true, "locales": true, "i18n": true,
}

// Classify 判定文件类别（仅依据路径与扩展名）。
func Classify(relPath string) Class {
	p := normalize(relPath)
	base := strings.ToLower(path.Base(p))
	ext := strings.ToLower(path.Ext(base))
	dirs := dirSet(p)

	if entryNames[strings.TrimSuffix(base, ext)] || entryNames[base] {
		return ClassEntry
	}
	if lockNames[base] {
		return ClassLock
	}
	if buildFiles[base] || buildExts[ext] {
		return ClassBuild
	}
	if isGenerated(base, dirs) {
		return ClassGenerated
	}
	if assetExts[ext] {
		return ClassAsset
	}
	if isTest(base, dirs) {
		return ClassTest
	}
	if configExts[ext] {
		return ClassConfig
	}
	return ClassSource
}

// Score 计算文件重要性评分（0~100，越高越优先保留）。
func Score(relPath string, sizeBytes int64) float64 {
	p := normalize(relPath)
	c := Classify(p)
	s := 50.0
	switch c {
	case ClassEntry:
		s = 88
	case ClassBuild:
		s = 92
	case ClassConfig:
		s = 66
	case ClassSource:
		s = 70
	case ClassTest:
		s = 32
	case ClassGenerated:
		s = 8
	case ClassLock:
		s = 5
	case ClassAsset:
		s = 10
	}

	// 目录层级越深，通用性越低
	depth := strings.Count(p, "/")
	s -= float64(depth) * 1.5

	// 低价值目录降权
	for d := range dirSet(p) {
		if lowValueDirs[d] {
			s -= 8
			break
		}
	}
	// 源码目录加权（与上面的 sourceDirs 表互补：命中即视为核心代码）
	for d := range dirSet(p) {
		if sourceDirs[d] {
			s += 4
			break
		}
	}
	// 体积惩罚：超大文件吞掉大量预算，过小文件信息量有限
	switch {
	case sizeBytes > 512*1024:
		s -= 18
	case sizeBytes > 128*1024:
		s -= 8
	case sizeBytes > 0 && sizeBytes < 128:
		s -= 12
	}
	if s < 0 {
		s = 0
	}
	if s > 100 {
		s = 100
	}
	return s
}

// ShouldExclude 是否建议默认排除（生成代码/锁文件/资源）。
func ShouldExclude(relPath string) bool {
	switch Classify(relPath) {
	case ClassGenerated, ClassLock, ClassAsset:
		return true
	}
	return false
}

// DirScore 目录重要性评分（用于推荐 directories）。
func DirScore(relDir string, fileCount int, codeCount int, totalBytes int64) float64 {
	name := strings.ToLower(path.Base(normalize(relDir)))
	s := 40.0
	switch {
	case sourceDirs[name]:
		s = 80
	case testDirs[name]:
		s = 30
	case lowValueDirs[name]:
		s = 25
	case generatedDirs[name]:
		s = 5
	}
	if fileCount > 0 && codeCount > 0 {
		s += 20 * float64(codeCount) / float64(fileCount)
	}
	s -= float64(strings.Count(normalize(relDir), "/")) * 3
	if totalBytes > 8*1024*1024 {
		s -= 6
	}
	if s < 0 {
		s = 0
	}
	return s
}

// NewCandidate 构造候选并填充评分。
func NewCandidate(relPath string, sizeBytes int64, tokens int) Candidate {
	return Candidate{
		Path:      normalize(relPath),
		SizeBytes: sizeBytes,
		Tokens:    tokens,
		Score:     Score(relPath, sizeBytes),
		Class:     Classify(relPath),
	}
}

// SortByScore 原地按评分降序排序（评分相同按路径升序，保证结果稳定）。
func SortByScore(c []Candidate) {
	sort.SliceStable(c, func(i, j int) bool {
		if c[i].Score != c[j].Score {
			return c[i].Score > c[j].Score
		}
		return c[i].Path < c[j].Path
	})
}

// WithinBudget 在 token 预算内贪心挑选文件：按评分降序装满预算，
// 单个超预算文件仍会保留（宁可超一点，也不丢掉唯一的高价值文件）。
// 返回保留与被裁剪的两组候选，均按原顺序（保留组按评分降序）。
func WithinBudget(c []Candidate, budget int) (keep, drop []Candidate) {
	if budget <= 0 {
		return append([]Candidate{}, c...), nil
	}
	sorted := append([]Candidate{}, c...)
	SortByScore(sorted)
	used := 0
	picked := make(map[string]bool)
	var kept []Candidate
	for _, x := range sorted {
		if used > 0 && used+x.Tokens > budget {
			continue
		}
		kept = append(kept, x)
		picked[x.Path] = true
		used += x.Tokens
	}
	for _, x := range c {
		if !picked[x.Path] {
			drop = append(drop, x)
		}
	}
	return kept, drop
}

// —— 内部辅助 ——

// normalize 统一为 / 分隔、去掉前导 ./。
func normalize(relPath string) string {
	p := filepath.ToSlash(relPath)
	p = strings.TrimPrefix(p, "./")
	return strings.Trim(p, "/")
}

// dirSet 返回路径中的目录名集合。
func dirSet(p string) map[string]bool {
	out := map[string]bool{}
	segs := strings.Split(p, "/")
	for _, s := range segs[:len(segs)-1] {
		if s != "" {
			out[strings.ToLower(s)] = true
		}
	}
	return out
}

func isTest(base string, dirs map[string]bool) bool {
	for d := range dirs {
		if testDirs[d] {
			return true
		}
	}
	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".test.js") || strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".spec.js") || strings.HasSuffix(base, ".spec.ts") ||
		strings.HasPrefix(base, "test_") || strings.HasPrefix(base, "test.")
}

func isGenerated(base string, dirs map[string]bool) bool {
	for d := range dirs {
		if generatedDirs[d] {
			return true
		}
	}
	for _, suf := range generatedSuffixes {
		if strings.HasSuffix(base, suf) {
			return true
		}
	}
	return false
}
