// Package infer 依据项目目录结构自动推断推荐配置（「生成配置文件」）。
//
// 扫描项目根目录，统计语言分布、目录权重与文件重要性，产出一份可直接保存的
// ProjectConfig 与一份人类可读的推断报告。纯 Go、不依赖 Wails；评分逻辑复用
// core/pick，与导出侧的预算裁剪保持一致。
package infer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go-markdown-my-project/core/config"
	"go-markdown-my-project/core/i18n"
	"go-markdown-my-project/core/pick"
)

// Options 推断参数。
type Options struct {
	Languages map[string]string // 扩展名 → 语言名（来自 langs.yml）
	MaxFiles  int               // 遍历文件上限（默认 20000）
	MaxDepth  int               // 目录层级上限（默认 6）
	Markdown  string            // 文档语言：zh_cn | en_us（默认 zh_cn）
}

// Report 推断报告（供 GUI 展示「为什么这样推荐」）。
type Report struct {
	FileCount   int            `json:"fileCount"`   // 纳入推荐范围的文件数
	Scanned     int            `json:"scanned"`     // 实际遍历到的文件数
	TotalBytes  int64          `json:"totalBytes"`  // 纳入范围的总字节数
	MainLang    string         `json:"mainLang"`    // 主语言（按文件数）
	ByLang      map[string]int `json:"byLang"`      // 语言分布
	ExcludeDirs []string       `json:"excludeDirs"` // 建议排除的目录（**/name）
	ExcludePats []string       `json:"excludePats"` // 建议排除的文件模式
	Notes       []string       `json:"notes"`       // 可读说明（已本地化）
}

// Result 推断结果：推荐配置 + 报告。
type Result struct {
	Config config.ProjectConfig `json:"config"`
	Report Report               `json:"report"`
}

// skipDirs 产物/依赖/缓存目录名（命中即整棵剪枝，不纳入统计）。
var skipDirs = map[string]bool{
	"node_modules": true, "dist": true, "build": true, "target": true,
	"out": true, "bin": true, "obj": true, "vendor": true, "coverage": true,
	".git": true, ".svn": true, ".hg": true, ".idea": true, ".vscode": true,
	".gradle": true, ".next": true, ".nuxt": true, ".output": true,
	"__pycache__": true, ".pytest_cache": true, ".mypy_cache": true, ".tox": true,
	"venv": true, ".venv": true, "env": true, "site-packages": true,
	".dart_tool": true, ".cargo": true, "Pods": true, "DerivedData": true,
	"Debug": true, "Release": true, ".terraform": true, ".parcel-cache": true,
}

// fileEntry 遍历到的文件条目。
type fileEntry struct {
	rel  string // 相对路径（/ 分隔）
	size int64
	lang string
}

// dirStat 目录聚合统计。
type dirStat struct {
	rel   string
	files int
	code  int
	bytes int64
	score float64
}

// Infer 扫描项目目录并推断推荐配置。
func Infer(projectPath string, opt Options) (*Result, error) {
	info, err := os.Stat(projectPath)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("infer.pathNotFound", "path", projectPath))
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s", i18n.T("infer.pathNotDir", "path", projectPath))
	}
	if opt.MaxFiles <= 0 {
		opt.MaxFiles = 20000
	}
	if opt.MaxDepth <= 0 {
		opt.MaxDepth = 6
	}
	if opt.Markdown == "" {
		opt.Markdown = "zh_cn"
	}
	if opt.Languages == nil {
		opt.Languages = map[string]string{}
	}

	var files []fileEntry
	dirs := map[string]*dirStat{}
	skipped := map[string]bool{}
	scanned := 0

	_ = filepath.WalkDir(projectPath, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return nil // 无权限等：跳过该分支
		}
		rel, rerr := filepath.Rel(projectPath, p)
		if rerr != nil {
			return nil
		}
		relSlash := filepath.ToSlash(rel)
		if relSlash == "." {
			return nil
		}
		if d.IsDir() {
			base := filepath.Base(p)
			// 产物/依赖目录整棵剪枝；隐藏目录同理（.github 保留：常有 CI 与文档）
			if skipDirs[base] || (strings.HasPrefix(base, ".") && base != ".github") {
				skipped[base] = true
				return filepath.SkipDir
			}
			if strings.Count(relSlash, "/") >= opt.MaxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		scanned++
		if len(files) >= opt.MaxFiles {
			return nil
		}
		fi, e := d.Info()
		if e != nil {
			return nil
		}
		lang := langOf(opt.Languages, p)
		files = append(files, fileEntry{rel: relSlash, size: fi.Size(), lang: lang})

		// 目录聚合：只统计前两层，够用于推荐 directories
		segs := strings.Split(relSlash, "/")
		if len(segs) > 1 {
			addDir(dirs, segs[0], lang, fi.Size())
		}
		if len(segs) > 2 {
			addDir(dirs, segs[0]+"/"+segs[1], lang, fi.Size())
		}
		return nil
	})

	// 语言分布与主语言
	byLang := map[string]int{}
	for _, f := range files {
		byLang[f.lang]++
	}
	mainLang := ""
	mainCount := 0
	for l, n := range byLang {
		if l == "Text" {
			continue
		}
		if n > mainCount || (n == mainCount && l < mainLang) {
			mainLang, mainCount = l, n
		}
	}

	// 推荐 directories：按权重排序，父子同时入选时保留父目录
	var dirList []*dirStat
	for _, d := range dirs {
		if d.code == 0 {
			continue // 纯资源目录不推荐
		}
		d.score = pick.DirScore(d.rel, d.files, d.code, d.bytes)
		dirList = append(dirList, d)
	}
	sort.Slice(dirList, func(i, j int) bool {
		if dirList[i].score != dirList[j].score {
			return dirList[i].score > dirList[j].score
		}
		return dirList[i].rel < dirList[j].rel
	})
	dirsOut := make([]string, 0, 12)
	for _, d := range dirList {
		if len(dirsOut) >= 12 {
			break
		}
		if coveredBy(d.rel, dirsOut) {
			continue // 已被父目录覆盖
		}
		dirsOut = append(dirsOut, d.rel)
	}
	sort.Strings(dirsOut)

	// 推荐 files：根目录直接文件（含构建清单），剔除建议排除项
	var rootFiles []pick.Candidate
	for _, f := range files {
		if strings.Contains(f.rel, "/") {
			continue
		}
		if pick.ShouldExclude(f.rel) {
			continue
		}
		rootFiles = append(rootFiles, pick.NewCandidate(f.rel, f.size, estimateTokens(f.size)))
	}
	limit := 12
	if len(dirsOut) == 0 {
		limit = 20 // 无子目录时，根级文件就是全部源码
		for _, f := range files {
			if strings.Contains(f.rel, "/") || pick.ShouldExclude(f.rel) {
				continue
			}
			rootFiles = append(rootFiles, pick.NewCandidate(f.rel, f.size, estimateTokens(f.size)))
		}
	}
	pick.SortByScore(rootFiles)
	seen := map[string]bool{}
	filesOut := make([]string, 0, limit)
	for _, c := range rootFiles {
		if len(filesOut) >= limit || seen[c.Path] {
			continue
		}
		seen[c.Path] = true
		filesOut = append(filesOut, c.Path)
	}
	sort.Strings(filesOut)

	// 建议排除的目录（只列实际命中的）
	excludeDirs := make([]string, 0, len(skipped))
	for name := range skipped {
		excludeDirs = append(excludeDirs, "**/"+name)
	}
	sort.Strings(excludeDirs)

	// 建议排除的文件模式：锁文件 + 生成代码 + 命中的资源扩展名
	excludePats := patternHints(files)

	// 体积控制建议
	total := int64(0)
	big := 0
	for _, f := range files {
		total += f.size
		if f.size > 128*1024 {
			big++
		}
	}
	maxSize := int64(0)
	if big > 0 {
		maxSize = 262144 // 256KB：足以容纳常规源文件，挡住生成物与数据文件
	}
	tokens := int(total / 4) // 粗估：4 字节 ≈ 1 token
	mode := "full"
	switch {
	case tokens > 400000:
		mode = "signatures"
	case tokens > 120000:
		mode = "smart"
	}
	split := 0
	if tokens > 150000 {
		split = 100000
	}

	skipBinary := true
	cfg := config.ProjectConfig{
		ProjectName:        filepath.Base(projectPath),
		ProjectPath:        projectPath,
		OutputFile:         sanitizeName(filepath.Base(projectPath)) + ".md",
		MarkdownLang:       opt.Markdown,
		Files:              filesOut,
		Directories:        dirsOut,
		ExcludeDirectories: excludeDirs,
		ExcludePatterns:    excludePats,
		MaxFileSize:        maxSize,
		ExportMode:         mode,
		SplitTokens:        split,
		SkipBinaryFiles:    &skipBinary,
		Redaction: config.RedactionConfig{
			Enabled:     true,
			Placeholder: "[REDACTED]",
			Strategy:    "placeholder",
		},
	}

	notes := []string{
		i18n.T("infer.noteMainLang", "lang", mainLangOr(mainLang, "—")),
		i18n.T("infer.noteScope", "files", len(filesOut), "dirs", len(dirsOut)),
	}
	if len(excludeDirs) > 0 {
		notes = append(notes, i18n.T("infer.noteExcludeDirs", "n", len(excludeDirs)))
	}
	if len(excludePats) > 0 {
		notes = append(notes, i18n.T("infer.noteExcludePats", "n", len(excludePats)))
	}
	if maxSize > 0 {
		notes = append(notes, i18n.T("infer.noteMaxSize", "n", big))
	}
	notes = append(notes, i18n.T("infer.noteMode", "mode", mode, "tokens", tokens))

	return &Result{
		Config: cfg,
		Report: Report{
			FileCount:   len(filesOut) + countFilesUnder(files, dirsOut),
			Scanned:     scanned,
			TotalBytes:  total,
			MainLang:    mainLangOr(mainLang, "—"),
			ByLang:      byLang,
			ExcludeDirs: excludeDirs,
			ExcludePats: excludePats,
			Notes:       notes,
		},
	}, nil
}

// —— 辅助 ——

func addDir(dirs map[string]*dirStat, key, lang string, size int64) {
	d, ok := dirs[key]
	if !ok {
		d = &dirStat{rel: key}
		dirs[key] = d
	}
	d.files++
	if lang != "Text" {
		d.code++
	}
	d.bytes += size
}

// coveredBy 判断 rel 是否已被已选目录覆盖（前缀命中）。
func coveredBy(rel string, chosen []string) bool {
	for _, c := range chosen {
		if rel == c {
			continue
		}
		if strings.HasPrefix(rel, c+"/") {
			return true
		}
	}
	return false
}

// patternHints 依据扫描结果给出排除模式建议。
func patternHints(files []fileEntry) []string {
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	for _, f := range files {
		base := strings.ToLower(filepath.Base(f.rel))
		switch pick.Classify(f.rel) {
		case pick.ClassLock:
			add(base)
		case pick.ClassGenerated:
			if i := strings.Index(base, "."); i >= 0 && strings.HasPrefix(base, ".") {
				continue
			}
			add("*" + extOf(base))
		case pick.ClassAsset:
			if ext := extOf(base); ext != "" {
				add("*" + ext)
			}
		}
	}
	sort.Strings(out)
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}

func extOf(base string) string {
	i := strings.LastIndex(base, ".")
	if i <= 0 {
		return ""
	}
	// 复合后缀（*.pb.go / *.min.js）保留后两段
	first := strings.Index(base, ".")
	if i != first && (base[first+1:] == "pb.go" || base[first+1:] == "min.js" || base[first+1:] == "min.css") {
		return base[first:]
	}
	return base[i:]
}

func countFilesUnder(files []fileEntry, dirs []string) int {
	n := 0
	for _, f := range files {
		for _, d := range dirs {
			if f.rel == d || strings.HasPrefix(f.rel, d+"/") {
				n++
				break
			}
		}
	}
	return n
}

// langOf 扩展名 → 语言名，未映射返回 Text。
func langOf(langs map[string]string, p string) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(p)), ".")
	if ext == "" {
		return "Text"
	}
	if l, ok := langs[ext]; ok && l != "" {
		return l
	}
	return "Text"
}

func mainLangOr(l, def string) string {
	if l == "" {
		return def
	}
	return l
}

// estimateTokens 粗估 token 数（4 字节 ≈ 1 token，与 core/token 同量级）。
func estimateTokens(size int64) int {
	return int((size + 3) / 4)
}

// sanitizeName 把目录名整理为合法的输出文件名主干。
func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "output"
	}
	repl := strings.NewReplacer(" ", "_", "\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return repl.Replace(name)
}
