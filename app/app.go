// Package app 是 Wails 绑定层（薄壳）：所有方法委托 core 包，进度经事件总线推给前端。
package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"go-markdown-my-project/core/compress"
	"go-markdown-my-project/core/config"
	"go-markdown-my-project/core/generator"
	"go-markdown-my-project/core/i18n"
	"go-markdown-my-project/core/logger"
	"go-markdown-my-project/core/redactor"
	"go-markdown-my-project/core/scanner"
	"go-markdown-my-project/core/token"
)

// FileInfo 前端展示的文件条目（ScanProject 结果）。
type FileInfo struct {
	Path      string `json:"path"`     // 相对路径（/ 分隔）
	Language  string `json:"language"` // 由 langs.yml 映射，默认 Text
	SizeBytes int64  `json:"sizeBytes"`
	Lines     int    `json:"lines"`
}

// ProjectSummary 项目列表条目。
type ProjectSummary struct {
	ConfigPath   string `json:"configPath"`
	Name         string `json:"name"`
	Path         string `json:"path"`
	OutputFile   string `json:"outputFile"`
	ExportMode   string `json:"exportMode"`
	MarkdownLang string `json:"markdownLang"`
}

// ExportOptions 导出选项（对齐设计文档 §5.2/§9）。
type ExportOptions struct {
	Mode          string   `json:"mode"`
	Redact        bool     `json:"redact"`
	SplitTokens   int      `json:"splitTokens"`
	Compress      bool     `json:"compress"`      // 空白精简输出（裁行尾空白、折叠空行，保持语法）
	FileOverrides []string `json:"fileOverrides"` // GUI 勾选覆盖配置（相对路径）
	// symbols / signatures 模式子选项
	IncludeLineNumbers *bool `json:"includeLineNumbers"` // nil → 默认 true
	MaxSignatureLen    int   `json:"maxSignatureLen"`    // ≤0 → 默认 200
}

// ExportResult 导出结果。
type ExportResult struct {
	OutputPaths []string `json:"outputPaths"`
	TotalChars  int      `json:"totalChars"`
	// TokenCount 按实际导出内容估算（CJK 1 字 ≈ 1 token，ASCII 4 字符 ≈ 1 token）
	TokenCount  int   `json:"tokenCount"`
	OutputBytes int64 `json:"outputBytes"` // 输出文件字节数
	DurationMs  int64 `json:"durationMs"`
}

// PreviewResult 导出内容预览（干跑，不写盘）。
type PreviewResult struct {
	Content    string `json:"content"`    // 展示用内容（超长截断）
	TotalChars int    `json:"totalChars"` // 完整内容字符数
	TokenCount int    `json:"tokenCount"`
	Truncated  bool   `json:"truncated"`
}

// App 暴露给前端的所有方法。
type App struct {
	ctx context.Context
}

// NewApp 创建 App。
func NewApp() *App { return &App{} }

// Startup 由 Wails 注入上下文。
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	// 把全局日志转发为前端可监听的事件（core 保持无 Wails 依赖）。
	logger.SetSink(func(e logger.Entry) {
		runtime.EventsEmit(a.ctx, "app:log", e)
	})
	logger.Info(i18n.T("log.started"))
}

// SetLocale 由前端在切换界面语言时调用，同步后端日志/校验文案语言。
func (a *App) SetLocale(code string) {
	i18n.SetLocale(code)
	logger.Info(i18n.T("log.localeSwitched"), "locale", code)
}

// languagesDir 返回资源目录（langs.yml 所在）。
func languagesPath() string { return "assets/langs.yml" }

// ListProjects 列出 projectsDir 下所有 *.yml 项目配置。
func (a *App) ListProjects(projectsDir string) ([]ProjectSummary, error) {
	if info, err := os.Stat(projectsDir); err != nil || !info.IsDir() {
		logger.Warn(i18n.T("log.listProjectsDirNotFound"), "dir", projectsDir)
		return nil, fmt.Errorf("%s", i18n.T("err.projectsDirNotFound", "dir", projectsDir, "wd", wd()))
	}
	// 用 ReadDir + 后缀过滤而非 Glob：Glob 会把路径里的 [ ] * ? 当元字符，
	// 导致含 [] 的目录（如 [OpsTools]）匹配不到任何配置
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".yml") {
			matches = append(matches, filepath.Join(projectsDir, e.Name()))
		}
	}
	sort.Strings(matches)
	var out []ProjectSummary
	for _, p := range matches {
		// 仅解析不校验：路径失效/部分字段缺失的配置也要在列表中展示，便于用户修复
		cfg, err := config.Parse(p)
		if err != nil {
			// 单个配置损坏不阻断列表，但要记录原因便于排查
			logger.Warn(i18n.T("log.listProjectsLoadFail"), "path", p, "err", err)
			continue
		}
		out = append(out, ProjectSummary{
			ConfigPath: p, Name: cfg.ProjectName, Path: cfg.ProjectPath,
			OutputFile: cfg.OutputFile, ExportMode: cfg.ExportMode, MarkdownLang: cfg.MarkdownLang,
		})
	}
	return out, nil
}

// LoadProject 加载单个项目配置（仅解析不校验：project_path 失效等也应能打开编辑器修复）。
func (a *App) LoadProject(configPath string) (*config.ProjectConfig, error) {
	return config.Parse(configPath)
}

// SaveProject 保存项目配置（新建或编辑），校验后写回。
func (a *App) SaveProject(configPath string, cfg config.ProjectConfig) error {
	if errs := a.ValidateConfig(cfg); len(errs) > 0 {
		return fmt.Errorf("%s: %s", errs[0].Field, errs[0].Message)
	}
	logger.Info(i18n.T("log.saveProject"), "path", configPath, "name", cfg.ProjectName)
	return cfg.Save(configPath)
}

// FieldError 字段级校验错误（编辑对话框按 field 映射到左栏控件）。
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidateConfig 配置完整校验（保存前复检；glob/RE2 编译级别检查）。
func (a *App) ValidateConfig(cfg config.ProjectConfig) []FieldError {
	var errs []FieldError
	add := func(field, msg string) { errs = append(errs, FieldError{Field: field, Message: msg}) }

	if strings.TrimSpace(cfg.ProjectName) == "" {
		add("project_name", i18n.T("val.nameRequired"))
	}
	if cfg.ProjectPath == "" {
		add("project_path", i18n.T("val.pathRequired"))
	} else if info, err := os.Stat(cfg.ProjectPath); err != nil {
		add("project_path", i18n.T("val.pathNotFound", "path", cfg.ProjectPath))
	} else if !info.IsDir() {
		add("project_path", i18n.T("val.pathNotDir", "path", cfg.ProjectPath))
	}
	if strings.TrimSpace(cfg.OutputFile) == "" {
		add("output_file", i18n.T("val.outputRequired"))
	} else if ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(cfg.OutputFile)), "."); ext != "" &&
		!map[string]bool{"md": true, "markdown": true, "txt": true}[ext] {
		add("output_file", i18n.T("val.outputExt", "ext", ext))
	}
	if cfg.MaxFileSize < 0 {
		add("max_file_size", i18n.T("val.notNegative"))
	}
	if cfg.SplitTokens < 0 {
		add("split_tokens", i18n.T("val.notNegative"))
	}
	for _, p := range cfg.ExcludePatterns {
		if strings.TrimSpace(p) == "" {
			add("exclude_patterns", i18n.T("val.excludeEmpty"))
			break
		}
	}
	for _, p := range cfg.Redaction.CustomPatterns {
		if _, err := regexp.Compile(p); err != nil {
			add("custom_patterns", i18n.T("val.regexInvalid", "err", err.Error()))
			break
		}
	}
	return errs
}

// ScanPreview 预检结果（只统计不读内容）。
type ScanPreview struct {
	FileCount  int            `json:"fileCount"`
	TotalBytes int64          `json:"totalBytes"`
	ByLang     map[string]int `json:"byLang"`
	Warnings   []string       `json:"warnings"`
}

// scanOpts 由配置构造扫描选项（各入口共用，避免字段透传遗漏）。
func scanOpts(cfg config.ProjectConfig, files, dirs []string) scanner.Options {
	includeEnabled := cfg.IncludeEnabled != nil && *cfg.IncludeEnabled
	return scanner.Options{
		ProjectPath:        cfg.ProjectPath,
		Files:              files,
		Directories:        dirs,
		ExcludeDirectories: cfg.ExcludeDirectories,
		ExcludePatterns:    cfg.ExcludePatterns,
		MaxFileSize:        cfg.MaxFileSize,
		IncludeEnabled:     includeEnabled,
		IncludePatterns:    cfg.IncludePatterns,
	}
}

// PreviewScan 保存前预检：文件数 / 总大小 / 语言分布 / 警告。
func (a *App) PreviewScan(cfg config.ProjectConfig) (ScanPreview, error) {
	results, err := scanner.ProcessFiles(scanOpts(cfg, cfg.Files, cfg.Directories))
	if err != nil {
		return ScanPreview{}, err
	}
	langs, _ := generator.LoadLanguages(languagesPath())
	prev := ScanPreview{ByLang: map[string]int{}}
	noLang := 0
	for _, r := range results {
		l := generator.LanguageFor(langs, r.FullPath)
		if l == "" {
			l = "Unknown"
			noLang++
		}
		prev.ByLang[l]++
		if info, err := os.Stat(r.FullPath); err == nil {
			prev.TotalBytes += info.Size()
		}
	}
	prev.FileCount = len(results)
	if noLang > 0 {
		prev.Warnings = append(prev.Warnings, fmt.Sprintf("%d 个文件无符号提取器（符号/签名模式将跳过）", noLang))
	}
	return prev, nil
}

// SerializeYAML 配置 → 带注释 YAML 文本（GUI 右栏实时镜像，与保存落盘同源）。
func (a *App) SerializeYAML(cfg config.ProjectConfig) (string, error) {
	return config.SerializeWithComments(&cfg)
}

// DeleteProject 删除项目配置文件。
func (a *App) DeleteProject(configPath string) error {
	return removeFile(configPath)
}

// SensitiveHit 敏感信息命中（掩码预览，绝不落盘明文）。
type SensitiveHit struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Rule   string `json:"rule"`
	Masked string `json:"masked"`
}

// ScanSensitive 扫描勾选文件的敏感信息（F99），返回掩码后的命中列表。
func (a *App) ScanSensitive(cfg config.ProjectConfig, fileOverrides []string) ([]SensitiveHit, error) {
	files, dirs := splitOverrides(cfg, fileOverrides)
	results, err := scanner.ProcessFiles(scanOpts(cfg, files, dirs))
	if err != nil {
		return nil, err
	}
	rcfg := redactor.DefaultConfig()
	out := make([]SensitiveHit, 0)
	for _, r := range results {
		rel, err := filepath.Rel(cfg.ProjectPath, r.FullPath)
		if err != nil {
			rel = r.FullPath
		}
		for _, f := range redactor.ScanContent(filepath.ToSlash(rel), r.Content, rcfg) {
			out = append(out, SensitiveHit{File: f.File, Line: f.Line, Rule: f.Rule, Masked: f.Masked})
		}
	}
	return out, nil
}

// splitOverrides 把 GUI 勾选的相对路径拆回 files/directories（与 RunExport 同逻辑）。
func splitOverrides(cfg config.ProjectConfig, overrides []string) (files, dirs []string) {
	if len(overrides) == 0 {
		return cfg.Files, cfg.Directories
	}
	for _, rel := range overrides {
		full := filepath.Join(cfg.ProjectPath, filepath.FromSlash(rel))
		if info, err := statIsDir(full); err == nil && info {
			dirs = append(dirs, rel)
		} else {
			files = append(files, rel)
		}
	}
	return files, dirs
}

// ScanProject 扫描项目，返回文件树条目（不读内容）。
func (a *App) ScanProject(cfg config.ProjectConfig) ([]FileInfo, error) {
	results, err := scanner.ProcessFiles(scanOpts(cfg, cfg.Files, cfg.Directories))
	if err != nil {
		return nil, err
	}
	langs, err := generator.LoadLanguages(languagesPath())
	if err != nil {
		langs = map[string]string{}
	}
	out := make([]FileInfo, 0, len(results))
	for _, r := range results {
		rel, err := filepath.Rel(cfg.ProjectPath, r.FullPath)
		if err != nil {
			rel = r.FullPath
		}
		info, err := statSize(r.FullPath)
		if err != nil {
			continue
		}
		out = append(out, FileInfo{
			Path:      filepath.ToSlash(rel),
			Language:  generator.LanguageFor(langs, r.FullPath),
			SizeBytes: info,
		})
	}
	return out, nil
}

// prepareExport 导出前公共准备：解析勾选覆盖、扫描、语言映射、模式判定与压缩预处理。
func prepareExport(cfg config.ProjectConfig, opt ExportOptions) ([]scanner.FileResult, map[string]string, generator.ExportMode, int, error) {
	// GUI 勾选覆盖配置
	files, dirs := cfg.Files, cfg.Directories
	if len(opt.FileOverrides) > 0 {
		files, dirs = nil, nil
		// 简化：把勾选的相对路径分为文件与目录（无子路径标记的作为文件）
		for _, rel := range opt.FileOverrides {
			full := filepath.Join(cfg.ProjectPath, filepath.FromSlash(rel))
			if info, serr := statIsDir(full); serr == nil && info {
				dirs = append(dirs, rel)
			} else {
				files = append(files, rel)
			}
		}
	}
	logger.Info(i18n.T("log.exportStart"), "project", cfg.ProjectName, "mode", opt.Mode, "files", len(files), "dirs", len(dirs))

	results, err := scanner.ProcessFiles(scanOpts(cfg, files, dirs))
	if err != nil {
		return nil, nil, "", 0, err
	}
	scanner.SortByRelPath(results)

	langs, err := generator.LoadLanguages(languagesPath())
	if err != nil {
		return nil, nil, "", 0, fmt.Errorf("加载语言映射失败: %w", err)
	}

	mode := generator.ExportMode(opt.Mode)
	if mode == "" {
		if cfg.ExportMode != "" {
			mode = generator.ExportMode(cfg.ExportMode)
		} else {
			mode = generator.ModeFull
		}
	}

	// 压缩输出 T2：大括号类语言（缩进不承载语法）剥除行首缩进，
	// 仅作用于源码文件内容；Markdown 结构（列表/引用缩进）不受影响。
	// T1 逐文件精简：分片路径按文件单元装箱，逐文件处理保证每片同样精简
	if opt.Compress {
		for i := range results {
			if compress.IsBraceLanguage(generator.LanguageFor(langs, results[i].FullPath)) {
				results[i].Content = compress.StripIndent(results[i].Content)
			}
			results[i].Content = compress.Minify(results[i].Content)
		}
	}
	return results, langs, mode, len(results), nil
}

// maxSignatureLen 导出签名长度上限（≤0 → 默认 200）。
func maxSignatureLen(opt ExportOptions) int {
	if opt.MaxSignatureLen <= 0 {
		return 200
	}
	return opt.MaxSignatureLen
}

// buildExport 扫描并生成完整导出内容（PreviewExport 使用；分片见 RunExport）。
func buildExport(cfg config.ProjectConfig, opt ExportOptions) (string, generator.ExportMode, int, error) {
	results, langs, mode, fileCount, err := prepareExport(cfg, opt)
	if err != nil {
		return "", "", 0, err
	}
	content, err := generator.Generate(results, generator.Options{
		ProjectName:        cfg.ProjectName,
		ProjectRoot:        cfg.ProjectPath,
		Lang:               cfg.MarkdownLang,
		Mode:               mode,
		Languages:          langs,
		IncludeLineNumbers: opt.IncludeLineNumbers == nil || *opt.IncludeLineNumbers,
		MaxSignatureLen:    maxSignatureLen(opt),
	})
	if err != nil {
		return "", "", 0, err
	}
	// 压缩输出：整篇空白精简（与逐文件精简叠加，覆盖脚手架文本）
	if opt.Compress {
		content = compress.Minify(content)
	}
	return content, mode, fileCount, nil
}

// RunExport 执行导出（GUI 主流程）。结果与进度经事件推送，此处同步返回结果。
func (a *App) RunExport(cfg config.ProjectConfig, opt ExportOptions) (ExportResult, error) {
	start := time.Now()

	results, langs, mode, fileCount, err := prepareExport(cfg, opt)
	if err != nil {
		return ExportResult{}, err
	}
	genOpts := generator.Options{
		ProjectName:        cfg.ProjectName,
		ProjectRoot:        cfg.ProjectPath,
		Lang:               cfg.MarkdownLang,
		Mode:               mode,
		Languages:          langs,
		IncludeLineNumbers: opt.IncludeLineNumbers == nil || *opt.IncludeLineNumbers,
		MaxSignatureLen:    maxSignatureLen(opt),
	}

	var outPaths []string
	var content string

	if mode == generator.ModeFull && opt.SplitTokens > 0 {
		// Token 限制分片（§7.5）：以文件为最小单位贪心装箱，产出 xxx.partN.md；
		// 单文件超过上限时独占一片，内容不截断
		preamble, parts := generator.FullParts(results, genOpts)
		if opt.Compress {
			preamble = compress.Minify(preamble)
		}
		units := make([]token.FileUnit, len(parts))
		for i, p := range parts {
			units[i] = token.FileUnit{Path: p.RelPath, Content: p.Content}
		}
		chunks := token.Split(units, opt.SplitTokens, preamble)
		ext := filepath.Ext(cfg.OutputFile)
		base := strings.TrimSuffix(cfg.OutputFile, ext)
		for _, c := range chunks {
			name := fmt.Sprintf("%s.part%d%s", base, c.Index, ext)
			p, werr := generator.WriteOutput("output", name, c.Content)
			if werr != nil {
				return ExportResult{}, werr
			}
			outPaths = append(outPaths, p)
			content += c.Content
		}
		logger.Info(i18n.T("log.exportSplit"), "parts", len(chunks), "limit", opt.SplitTokens)
	} else {
		content, err = generator.Generate(results, genOpts)
		if err != nil {
			return ExportResult{}, err
		}
		if opt.Compress {
			content = compress.Minify(content)
		}
		outPath, werr := generator.WriteOutput("output", cfg.OutputFile, content)
		if werr != nil {
			return ExportResult{}, werr
		}
		outPaths = []string{outPath}
		if opt.Compress {
			logger.Info(i18n.T("log.compressDone"), "out", outPath, "bytes", len(content))
		}
	}

	logger.Info(i18n.T("log.exportDone"), "mode", string(mode), "files", fileCount, "chars", len([]rune(content)), "out", strings.Join(outPaths, ","), "ms", time.Since(start).Milliseconds())

	return ExportResult{
		OutputPaths: outPaths,
		TotalChars:  len([]rune(content)),
		TokenCount:  token.EstimateTokens(content),
		OutputBytes: int64(len(content)),
		DurationMs:  time.Since(start).Milliseconds(),
	}, nil
}

// PreviewExport 干跑导出：生成完整内容但不写盘，返回（截断后的）预览。
func (a *App) PreviewExport(cfg config.ProjectConfig, opt ExportOptions) (PreviewResult, error) {
	content, _, _, err := buildExport(cfg, opt)
	if err != nil {
		return PreviewResult{}, err
	}
	runes := []rune(content)
	tokens := token.EstimateTokens(content)
	const maxPreview = 100_000
	truncated := len(runes) > maxPreview
	if truncated {
		content = string(runes[:maxPreview])
	}
	logger.Info(i18n.T("log.exportPreview"), "chars", len(runes), "truncated", truncated)
	return PreviewResult{
		Content:    content,
		TotalChars: len(runes),
		TokenCount: tokens,
		Truncated:  truncated,
	}, nil
}

// removeFile 删除文件（独立小函数便于测试）。
func removeFile(path string) error {
	return os.Remove(path)
}

// statSize 返回文件大小；失败返回错误。
func statSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// statIsDir 判断路径是否为目录。
func statIsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// wd 返回当前工作目录（用于错误提示定位）。
func wd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "?"
	}
	return dir
}

// SelectDirectory 弹出系统目录选择对话框，返回所选目录；取消返回空串。
// startDir 为初始打开目录（可为空；不存在或非目录时忽略）。
func (a *App) SelectDirectory(startDir string) (string, error) {
	opts := runtime.OpenDialogOptions{Title: "选择目录"}
	if startDir != "" {
		if abs, err := filepath.Abs(startDir); err == nil {
			if info, err := os.Stat(abs); err == nil && info.IsDir() {
				opts.DefaultDirectory = abs
			}
		}
	}
	return runtime.OpenDirectoryDialog(a.ctx, opts)
}

// DefaultProjectsDir 返回 exe 工作目录下的默认 projects 目录（绝对路径）。
func (a *App) DefaultProjectsDir() string {
	return filepath.Join(wd(), "config", "projects")
}

// PathCheck 路径存在性校验结果。
type PathCheck struct {
	Exists bool `json:"exists"`
	IsDir  bool `json:"isDir"`
}

// PathExists 判断路径是否存在；是目录时 IsDir 为 true。
// 返回结构体而非多返回值：Wails 对 (bool, bool) 的绑定易生歧义。
func (a *App) PathExists(path string) PathCheck {
	info, err := os.Stat(path)
	if err != nil {
		return PathCheck{}
	}
	return PathCheck{Exists: true, IsDir: info.IsDir()}
}

// ListSubdirs 返回目录下所有直接子目录的完整路径（不递归）；目录不存在或无子目录时返回空。
// 供「添加所有子文件夹」一键填充包含目录列表。
func (a *App) ListSubdirs(path string) []string {
	entries, err := os.ReadDir(path)
	if err != nil {
		return []string{}
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, filepath.Join(path, e.Name()))
		}
	}
	sort.Strings(out)
	return out
}

// OpenPath 用系统默认程序打开文件所在目录（select 不选中文件）或目录本身。
func (a *App) OpenPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	return exec.Command("explorer", abs).Start()
}
