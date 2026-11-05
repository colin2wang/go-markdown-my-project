// Package app 是 Wails 绑定层（薄壳）：所有方法委托 core 包，进度经事件总线推给前端。
package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"go-markdown-my-project/core/compress"
	"go-markdown-my-project/core/config"
	"go-markdown-my-project/core/generator"
	"go-markdown-my-project/core/i18n"
	"go-markdown-my-project/core/infer"
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
	// 省 token 三件套（用更少 token 承载更多代码信息）
	// BudgetTokens >0 时按重要性裁剪文件（低价值文件移入省略清单）；
	// DropMarkedComments=true 时连 TODO/许可证等标记注释一起删除（默认保留）；
	// MaxLineLen >0 时截断超长行；SmartFullTokens ≤0 时用默认 1200。
	BudgetTokens       int  `json:"budgetTokens"`
	StripComments      bool `json:"stripComments"`
	DropMarkedComments bool `json:"dropMarkedComments"`
	MaxLineLen         int  `json:"maxLineLen"`
	SmartFullTokens    int  `json:"smartFullTokens"`
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

// 初始窗口占主屏逻辑分辨率的比例（Screen.Size 与 Wails 窗口尺寸单位一致）。
const windowScreenRatio = 0.75

// Startup 由 Wails 注入上下文。
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	// main.go 以 StartHidden 启动：无论尺寸计算是否成功，退出前都要显示窗口
	defer runtime.WindowShow(ctx)
	a.applyInitialWindowSize(ctx)
	// 把全局日志转发为前端可监听的事件（core 保持无 Wails 依赖）。
	logger.SetSink(func(e logger.Entry) {
		runtime.EventsEmit(a.ctx, "app:log", e)
	})
	logger.Info(i18n.T("log.started"))
}

// applyInitialWindowSize 按主屏逻辑分辨率的 75% 设置窗口尺寸并居中。
// 取不到屏幕信息或尺寸异常时保留 main.go 中的回退尺寸，窗口仍可正常显示。
func (a *App) applyInitialWindowSize(ctx context.Context) {
	screens, err := runtime.ScreenGetAll(ctx)
	if err != nil || len(screens) == 0 {
		return
	}
	screen := screens[0]
	for _, s := range screens {
		if s.IsPrimary {
			screen = s
			break
		}
	}
	width := int(float64(screen.Size.Width) * windowScreenRatio)
	height := int(float64(screen.Size.Height) * windowScreenRatio)
	if width <= 0 || height <= 0 {
		return
	}
	runtime.WindowSetSize(ctx, width, height)
	runtime.WindowCenter(ctx)
}

// SetLocale 由前端在切换界面语言时调用，同步后端日志/校验文案语言。
func (a *App) SetLocale(code string) {
	i18n.SetLocale(code)
	logger.Info(i18n.T("log.localeSwitched"), "locale", code)
}

// languagesPath 返回语言映射文件路径（langs.yml 与项目配置同在 config/ 下）。
func languagesPath() string { return "config/langs.yml" }

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
	EstTokens  int            `json:"estTokens"` // 估算 token 数（容量规划用）
	ByLang     map[string]int `json:"byLang"`
	Warnings   []string       `json:"warnings"`
}

// scanOpts 由配置构造扫描选项（各入口共用，避免字段透传遗漏）。
func scanOpts(cfg config.ProjectConfig, files, dirs []string) scanner.Options {
	includeEnabled := cfg.IncludeEnabled != nil && *cfg.IncludeEnabled
	opts := scanner.Options{
		ProjectPath:        cfg.ProjectPath,
		Files:              files,
		Directories:        dirs,
		ExcludeDirectories: cfg.ExcludeDirectories,
		ExcludePatterns:    cfg.ExcludePatterns,
		MaxFileSize:        cfg.MaxFileSize,
		IncludeEnabled:     includeEnabled,
		IncludePatterns:    cfg.IncludePatterns,
		SkipBinary:         cfg.SkipBinaryFiles == nil || *cfg.SkipBinaryFiles,
	}
	// 全局默认排除（config/exclude_defaults.yml）：与项目配置合并去重后注入，
	// 单点覆盖所有扫描/导出入口；项目内可增量追加，use_default_excludes: false 可整体关闭。
	if cfg.UseDefaultsEnabled() {
		if d := loadExcludeDefaults(); d != nil {
			opts.ExcludeDirectories = mergeUnique(opts.ExcludeDirectories, d.Directories)
			opts.ExcludePatterns = mergeUnique(opts.ExcludePatterns, d.Files)
		}
	}
	return opts
}

// mergeUnique 追加并去重（保持原顺序，首次出现优先）。
func mergeUnique(base, extra []string) []string {
	seen := make(map[string]bool, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))
	for _, s := range base {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	for _, s := range extra {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

var (
	excludeDefaultsOnce sync.Once
	excludeDefaults     *config.ExcludeDefaults
)

// loadExcludeDefaults 进程内加载一次全局默认排除；解析失败记 WARN 并视为无默认规则。
func loadExcludeDefaults() *config.ExcludeDefaults {
	excludeDefaultsOnce.Do(func() {
		d, err := config.LoadExcludeDefaults("config/exclude_defaults.yml")
		if err != nil {
			logger.Warn(i18n.T("log.excludeDefaultsFail"), "err", err.Error())
		}
		excludeDefaults = d
	})
	return excludeDefaults
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
		prev.TotalBytes += r.Size
		prev.EstTokens += token.EstimateTokens(r.Content)
	}
	prev.FileCount = len(results)
	if noLang > 0 {
		prev.Warnings = append(prev.Warnings, i18n.T("warn.noExtractor", "n", noLang))
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

// ScanProject 扫描项目，返回文件树条目（大小与行数由扫描阶段顺带统计，不重复读盘）。
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
		out = append(out, FileInfo{
			Path:      filepath.ToSlash(rel),
			Language:  generator.LanguageFor(langs, r.FullPath),
			SizeBytes: r.Size,
			Lines:     r.Lines,
		})
	}
	return out, nil
}

// SuggestResult 配置推断结果（绑定层用具名结构体，避免多返回值歧义）。
type SuggestResult struct {
	Config config.ProjectConfig `json:"config"`
	Report infer.Report         `json:"report"`
}

// SuggestConfig 自动分析项目目录结构，产出推荐配置与推断报告（「生成配置文件」）。
// 仅依据目录与文件名做启发式推断，不修改任何文件；失败时返回错误由前端提示。
func (a *App) SuggestConfig(projectPath string) (SuggestResult, error) {
	logger.Info(i18n.T("log.suggest"), "path", projectPath)
	langs, err := generator.LoadLanguages(languagesPath())
	if err != nil {
		langs = map[string]string{}
	}
	res, err := infer.Infer(projectPath, infer.Options{
		Languages: langs,
		Markdown:  markdownLangFromLocale(),
	})
	if err != nil {
		logger.Warn(i18n.T("log.suggestFail"), "path", projectPath, "err", err.Error())
		return SuggestResult{}, err
	}
	logger.Info(i18n.T("log.suggest"), "path", projectPath, "files", len(res.Config.Files), "dirs", len(res.Config.Directories))
	return SuggestResult{Config: res.Config, Report: res.Report}, nil
}

// markdownLangFromLocale 由后端当前界面语言推导文档语言（zh_cn | en_us）。
func markdownLangFromLocale() string {
	if i18n.Current() == i18n.ZhCN {
		return "zh_cn"
	}
	return "en_us"
}

// exportPlan 导出前的公共准备结果（扫描结果 + 裁剪 + 语言映射 + 模式）。
type exportPlan struct {
	Results   []scanner.FileResult
	Dropped   []scanner.FileResult // 按预算裁剪掉的文件（生成省略清单用）
	Langs     map[string]string
	Mode      generator.ExportMode
	FileCount int
}

// prepareExport 导出前公共准备：解析勾选覆盖、扫描、敏感替换、语言映射、
// 模式判定、预算裁剪与压缩预处理（各导出入口共用，避免字段透传遗漏）。
func prepareExport(cfg config.ProjectConfig, opt ExportOptions) (exportPlan, error) {
	// GUI 勾选覆盖配置（与 splitOverrides 同逻辑：目录归入 directories，其余作 files）
	files, dirs := splitOverrides(cfg, opt.FileOverrides)
	logger.Info(i18n.T("log.exportStart"), "project", cfg.ProjectName, "mode", opt.Mode, "files", len(files), "dirs", len(dirs))

	results, err := scanner.ProcessFiles(scanOpts(cfg, files, dirs))
	if err != nil {
		logger.Error(i18n.T("log.exportFail"), "err", err.Error())
		return exportPlan{}, err
	}
	scanner.SortByRelPath(results)

	// 敏感信息替换（F99 两阶段第二阶段）：前端已确认报告，导出时替换；
	// 日志只记数量与规则名，绝不落盘明文。
	if opt.Redact {
		rcfg := redactorConfig(cfg.Redaction)
		total := 0
		for i := range results {
			rel, rerr := filepath.Rel(cfg.ProjectPath, results[i].FullPath)
			if rerr != nil {
				rel = results[i].FullPath
			}
			out, n := redactor.RedactContent(filepath.ToSlash(rel), results[i].Content, rcfg)
			if n > 0 {
				results[i].Content = out
				total += n
			}
		}
		logger.Info(i18n.T("log.redactApplied"), "count", total)
	}

	langs, err := generator.LoadLanguages(languagesPath())
	if err != nil {
		logger.Error(i18n.T("log.exportFail"), "err", err.Error())
		return exportPlan{}, fmt.Errorf("加载语言映射失败: %w", err)
	}

	mode := generator.ExportMode(opt.Mode)
	if mode == "" {
		if cfg.ExportMode != "" {
			mode = generator.ExportMode(cfg.ExportMode)
		} else {
			mode = generator.ModeFull
		}
	}

	// 预算裁剪：按重要性评分挑文件（用更少 token 覆盖更多有效信息）。
	// 在压缩之前按原始内容估算，保证预算口径与用户勾选时的估算一致。
	var dropped []scanner.FileResult
	if opt.BudgetTokens > 0 {
		sel := generator.SelectWithinBudget(results, cfg.ProjectPath, opt.BudgetTokens)
		if len(sel.Dropped) > 0 {
			logger.Info(i18n.T("log.budgetApplied"), "kept", len(sel.Kept), "dropped", len(sel.Dropped), "budget", opt.BudgetTokens)
		}
		results, dropped = sel.Kept, sel.Dropped
	}

	// 逐文件精简：分片路径按文件单元装箱，逐文件处理保证每片同样精简。
	// 压缩输出 T2：大括号类语言（缩进不承载语法）剥除行首缩进，
	// 仅作用于源码文件内容；Markdown 结构（列表/引用缩进）不受影响。
	if opt.Compress || opt.StripComments || opt.MaxLineLen > 0 {
		for i := range results {
			lang := generator.LanguageFor(langs, results[i].FullPath)
			c := results[i].Content
			if opt.Compress && compress.IsBraceLanguage(lang) {
				c = compress.StripIndent(c)
			}
			if opt.StripComments {
				c = compress.StripCommentsWithOptions(lang, c, compress.CommentOptions{
					KeepMarkers: !opt.DropMarkedComments,
				})
			}
			if opt.Compress || opt.MaxLineLen > 0 {
				c = compress.MinifyWith(c, opt.MaxLineLen)
			}
			results[i].Content = c
		}
	}
	return exportPlan{Results: results, Dropped: dropped, Langs: langs, Mode: mode, FileCount: len(results)}, nil
}

// maxSignatureLen 导出签名长度上限（≤0 → 默认 200）。
func maxSignatureLen(opt ExportOptions) int {
	if opt.MaxSignatureLen <= 0 {
		return 200
	}
	return opt.MaxSignatureLen
}

// redactorConfig 把项目脱敏配置转为 redactor.Config；
// 自定义正则编译失败时忽略该条（保存时已校验过）。
func redactorConfig(rc config.RedactionConfig) redactor.Config {
	cfg := redactor.DefaultConfig()
	if rc.Placeholder != "" {
		cfg.Placeholder = rc.Placeholder
	}
	if rc.Strategy != "" {
		cfg.Strategy = rc.Strategy
	}
	if len(rc.Allowlist) > 0 {
		m := make(map[string]bool, len(rc.Allowlist))
		for _, k := range rc.Allowlist {
			m[k] = true
		}
		cfg.Allowlist = m
	}
	for _, p := range rc.CustomPatterns {
		if re, err := regexp.Compile(p); err == nil {
			cfg.CustomPatterns = append(cfg.CustomPatterns, re)
		}
	}
	return cfg
}

// genOptions 由导出选项构造生成参数（各导出入口共用，避免字段透传遗漏）。
func genOptions(cfg config.ProjectConfig, opt ExportOptions, plan exportPlan) generator.Options {
	return generator.Options{
		ProjectName:        cfg.ProjectName,
		ProjectRoot:        cfg.ProjectPath,
		Lang:               cfg.MarkdownLang,
		Mode:               plan.Mode,
		Languages:          plan.Langs,
		IncludeLineNumbers: opt.IncludeLineNumbers == nil || *opt.IncludeLineNumbers,
		MaxSignatureLen:    maxSignatureLen(opt),
		SmartFullTokens:    opt.SmartFullTokens,
		StripComments:      opt.StripComments,
		DropMarkedComments: opt.DropMarkedComments,
		MaxLineLen:         opt.MaxLineLen,
	}
}

// droppedManifest 生成被预算裁剪文件的清单（保留路径与规模，信息不丢）。
func droppedManifest(plan exportPlan, cfg config.ProjectConfig) string {
	return generator.DroppedManifest(plan.Dropped, cfg.ProjectPath, cfg.MarkdownLang)
}

// buildExport 扫描并生成完整导出内容（PreviewExport 使用；分片见 RunExport）。
func buildExport(cfg config.ProjectConfig, opt ExportOptions) (string, generator.ExportMode, int, error) {
	plan, err := prepareExport(cfg, opt)
	if err != nil {
		return "", "", 0, err
	}
	content, err := generator.Generate(plan.Results, genOptions(cfg, opt, plan))
	if err != nil {
		return "", "", 0, err
	}
	content += droppedManifest(plan, cfg)
	// 压缩输出：整篇空白精简（与逐文件精简叠加，覆盖脚手架文本）
	if opt.Compress {
		content = compress.MinifyWith(content, opt.MaxLineLen)
	}
	return content, plan.Mode, plan.FileCount, nil
}

// RunExport 执行导出（GUI 主流程）。结果与进度经事件推送，此处同步返回结果。
func (a *App) RunExport(cfg config.ProjectConfig, opt ExportOptions) (ExportResult, error) {
	start := time.Now()

	plan, err := prepareExport(cfg, opt)
	if err != nil {
		return ExportResult{}, err
	}
	mode := plan.Mode
	fileCount := plan.FileCount
	logExportFail := func(err error) {
		logger.Error(i18n.T("log.exportFail"), "err", err.Error())
	}
	genOpts := genOptions(cfg, opt, plan)
	manifest := droppedManifest(plan, cfg)

	var outPaths []string
	var content string

	if (mode == generator.ModeFull || mode == generator.ModeSmart) && opt.SplitTokens > 0 {
		// Token 限制分片（§7.5）：以文件为最小单位贪心装箱，产出 xxx.partN.md；
		// 单文件超过上限时独占一片，内容不截断
		preamble, parts := generator.FileParts(plan.Results, genOpts)
		if opt.Compress {
			preamble = compress.MinifyWith(preamble, opt.MaxLineLen)
		}
		units := make([]token.FileUnit, len(parts))
		for i, p := range parts {
			units[i] = token.FileUnit{Path: p.RelPath, Content: p.Content}
		}
		chunks := token.Split(units, opt.SplitTokens, preamble)
		ext := filepath.Ext(cfg.OutputFile)
		base := strings.TrimSuffix(cfg.OutputFile, ext)
		for i, c := range chunks {
			body := c.Content
			// 省略清单附在最后一片，保证分片合并后信息完整
			if i == len(chunks)-1 {
				body += manifest
			}
			name := fmt.Sprintf("%s.part%d%s", base, c.Index, ext)
			p, werr := generator.WriteOutput("output", name, body, a.exportEncoding(cfg))
			if werr != nil {
				logExportFail(werr)
				return ExportResult{}, werr
			}
			outPaths = append(outPaths, p)
			content += body
		}
		logger.Info(i18n.T("log.exportSplit"), "parts", len(chunks), "limit", opt.SplitTokens)
	} else {
		content, err = generator.Generate(plan.Results, genOpts)
		if err != nil {
			logExportFail(err)
			return ExportResult{}, err
		}
		content += manifest
		if opt.Compress {
			content = compress.MinifyWith(content, opt.MaxLineLen)
		}
		outPath, werr := generator.WriteOutput("output", cfg.OutputFile, content, a.exportEncoding(cfg))
		if werr != nil {
			logExportFail(werr)
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

// ExcludeDefaultsResult DefaultExcludes 的返回（Wails 绑定使用具名结构体避免多返回值歧义）。
type ExcludeDefaultsResult struct {
	Directories []string `json:"directories"`
	Files       []string `json:"files"`
}

// DefaultExcludes 返回全局默认排除规则（config/exclude_defaults.yml），供编辑器只读展示。
func (a *App) DefaultExcludes() ExcludeDefaultsResult {
	if d := loadExcludeDefaults(); d != nil {
		return ExcludeDefaultsResult{Directories: d.Directories, Files: d.Files}
	}
	return ExcludeDefaultsResult{Directories: []string{}, Files: []string{}}
}

// DefaultProjectsDir 返回 exe 工作目录下的默认 projects 目录（绝对路径）。
func (a *App) DefaultProjectsDir() string {
	return filepath.Join(wd(), "config", "projects")
}

// GetGlobalSettings 读取全局设置（config/global_settings.yml）；不存在返回默认值。
func (a *App) GetGlobalSettings() (config.GlobalSettings, error) {
	g, err := config.LoadGlobalSettings(filepath.Join(wd(), "config"))
	if err != nil {
		return config.GlobalSettings{}, err
	}
	return *g, nil
}

// SaveGlobalSettings 保存全局设置。
func (a *App) SaveGlobalSettings(g config.GlobalSettings) error {
	if err := config.SaveGlobalSettings(filepath.Join(wd(), "config"), &g); err != nil {
		return err
	}
	logger.Info(i18n.T("log.globalSettingsSaved"), "encoding", g.OutputEncoding, "lang", g.InterfaceLanguage)
	return nil
}

// exportEncoding 解析导出编码：项目配置显式设置 > 全局设置 > 默认 utf8_bom。
// 项目级 write_bom 兼容：显式 false 时映射为 utf8（无 BOM），显式 true 时保持全局/默认。
func (a *App) exportEncoding(cfg config.ProjectConfig) string {
	g, err := config.LoadGlobalSettings(filepath.Join(wd(), "config"))
	if err != nil {
		logger.Warn("failed to load global settings, using default encoding", "err", err)
		return generator.EncUTF8BOM
	}
	enc := g.OutputEncoding
	if cfg.WriteBOM != nil {
		if !*cfg.WriteBOM {
			enc = generator.EncUTF8
		} else if enc == generator.EncUTF8 {
			// 项目显式要求 BOM，覆盖全局的 utf8 无 BOM
			enc = generator.EncUTF8BOM
		}
	}
	return enc
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

// OpenPath 用系统文件管理器打开文件所在目录（文件则打开其父目录）或目录本身。
// Windows 上优先复用已显示该目录的资源管理器窗口（见 openpath_windows.go）。
func (a *App) OpenPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	return openInFileManager(abs)
}
