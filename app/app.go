// Package app 是 Wails 绑定层（薄壳）：所有方法委托 core 包，进度经事件总线推给前端。
package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"go-markdown-my-project/core/config"
	"go-markdown-my-project/core/generator"
	"go-markdown-my-project/core/logger"
	"go-markdown-my-project/core/redactor"
	"go-markdown-my-project/core/scanner"
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
	FileOverrides []string `json:"fileOverrides"` // GUI 勾选覆盖配置（相对路径）
	// symbols / signatures 模式子选项
	IncludeLineNumbers *bool `json:"includeLineNumbers"` // nil → 默认 true
	MaxSignatureLen    int   `json:"maxSignatureLen"`    // ≤0 → 默认 200
}

// ExportResult 导出结果。
type ExportResult struct {
	OutputPaths []string `json:"outputPaths"`
	TotalChars  int      `json:"totalChars"`
	DurationMs  int64    `json:"durationMs"`
}

// App 暴露给前端的所有方法。
type App struct {
	ctx context.Context
}

// NewApp 创建 App。
func NewApp() *App { return &App{} }

// Startup 由 Wails 注入上下文。
func (a *App) Startup(ctx context.Context) { a.ctx = ctx }

// languagesDir 返回资源目录（langs.yml 所在）。
func languagesPath() string { return "assets/langs.yml" }

// ListProjects 列出 projectsDir 下所有 *.yml 项目配置。
func (a *App) ListProjects(projectsDir string) ([]ProjectSummary, error) {
	if info, err := os.Stat(projectsDir); err != nil || !info.IsDir() {
		logger.Warn("ListProjects: projects dir not found", "dir", projectsDir)
		return nil, fmt.Errorf("projects 目录不存在: %s（当前工作目录: %s）", projectsDir, wd())
	}
	matches, err := filepath.Glob(filepath.Join(projectsDir, "*.yml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	var out []ProjectSummary
	for _, p := range matches {
		cfg, err := config.Load(p)
		if err != nil {
			// 单个配置损坏不阻断列表，但要记录原因便于排查
			logger.Warn("ListProjects: failed to load config", "path", p, "err", err)
			continue
		}
		out = append(out, ProjectSummary{
			ConfigPath: p, Name: cfg.ProjectName, Path: cfg.ProjectPath,
			OutputFile: cfg.OutputFile, ExportMode: cfg.ExportMode, MarkdownLang: cfg.MarkdownLang,
		})
	}
	return out, nil
}

// LoadProject 加载单个项目配置。
func (a *App) LoadProject(configPath string) (*config.ProjectConfig, error) {
	return config.Load(configPath)
}

// SaveProject 保存项目配置（新建或编辑），校验后写回。
func (a *App) SaveProject(configPath string, cfg config.ProjectConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	return cfg.Save(configPath)
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
	results, err := scanner.ProcessFiles(scanner.Options{
		ProjectPath:        cfg.ProjectPath,
		Files:              files,
		Directories:        dirs,
		ExcludeDirectories: cfg.ExcludeDirectories,
		ExcludePatterns:    cfg.ExcludePatterns,
		MaxFileSize:        cfg.MaxFileSize,
	})
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
	results, err := scanner.ProcessFiles(scanner.Options{
		ProjectPath:        cfg.ProjectPath,
		Files:              cfg.Files,
		Directories:        cfg.Directories,
		ExcludeDirectories: cfg.ExcludeDirectories,
		ExcludePatterns:    cfg.ExcludePatterns,
		MaxFileSize:        cfg.MaxFileSize,
	})
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

// RunExport 执行导出（GUI 主流程）。结果与进度经事件推送，此处同步返回结果。
func (a *App) RunExport(cfg config.ProjectConfig, opt ExportOptions) (ExportResult, error) {
	start := time.Now()

	// GUI 勾选覆盖配置
	files, dirs := cfg.Files, cfg.Directories
	if len(opt.FileOverrides) > 0 {
		files, dirs = nil, nil
		langs := map[string]bool{}
		for _, f := range opt.FileOverrides {
			langs[f] = true
		}
		_ = langs
		// 简化：把勾选的相对路径分为文件与目录（无子路径标记的作为文件）
		for _, rel := range opt.FileOverrides {
			full := filepath.Join(cfg.ProjectPath, filepath.FromSlash(rel))
			if info, err := statIsDir(full); err == nil && info {
				dirs = append(dirs, rel)
			} else {
				files = append(files, rel)
			}
		}
	}

	results, err := scanner.ProcessFiles(scanner.Options{
		ProjectPath:        cfg.ProjectPath,
		Files:              files,
		Directories:        dirs,
		ExcludeDirectories: cfg.ExcludeDirectories,
		ExcludePatterns:    cfg.ExcludePatterns,
		MaxFileSize:        cfg.MaxFileSize,
	})
	if err != nil {
		return ExportResult{}, err
	}
	scanner.SortByRelPath(results)

	langs, err := generator.LoadLanguages(languagesPath())
	if err != nil {
		return ExportResult{}, fmt.Errorf("加载语言映射失败: %w", err)
	}

	mode := generator.ExportMode(opt.Mode)
	if mode == "" {
		if cfg.ExportMode != "" {
			mode = generator.ExportMode(cfg.ExportMode)
		} else {
			mode = generator.ModeFull
		}
	}

	includeLineNumbers := true
	if opt.IncludeLineNumbers != nil {
		includeLineNumbers = *opt.IncludeLineNumbers
	}
	maxSig := opt.MaxSignatureLen
	if maxSig <= 0 {
		maxSig = 200
	}

	content, err := generator.Generate(results, generator.Options{
		ProjectName:        cfg.ProjectName,
		ProjectRoot:        cfg.ProjectPath,
		Lang:               cfg.MarkdownLang,
		Mode:               mode,
		Languages:          langs,
		IncludeLineNumbers: includeLineNumbers,
		MaxSignatureLen:    maxSig,
	})
	if err != nil {
		return ExportResult{}, err
	}

	outPath, err := generator.WriteOutput("output", cfg.OutputFile, content)
	if err != nil {
		return ExportResult{}, err
	}

	return ExportResult{
		OutputPaths: []string{outPath},
		TotalChars:  len([]rune(content)),
		DurationMs:  time.Since(start).Milliseconds(),
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
func (a *App) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择目录",
	})
}

// DefaultProjectsDir 返回 exe 工作目录下的默认 projects 目录（绝对路径）。
func (a *App) DefaultProjectsDir() string {
	return filepath.Join(wd(), "config", "projects")
}

// PathExists 判断路径是否存在；是目录时 second 返回 true。
func (a *App) PathExists(path string) (bool, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return false, false
	}
	return true, info.IsDir()
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
