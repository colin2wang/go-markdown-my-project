// CLI 无头模式（P1）：project-docs --cli --config projects/x.yml --mode full
// 复用 core 包，供脚本/CI 调用。Wails GUI 走 main.go。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-markdown-my-project/core/config"
	"go-markdown-my-project/core/generator"
	"go-markdown-my-project/core/scanner"
	"go-markdown-my-project/core/token"
)

func main() {
	configPath := flag.String("config", "", "项目配置 YAML 路径")
	mode := flag.String("mode", "", "导出模式: full | files | symbols | signatures（覆盖配置文件）")
	projectsDir := flag.String("projects", "", "批量模式：扫描目录下全部 *.yml")
	langsFile := flag.String("langs", "assets/langs.yml", "语言映射文件")
	outputDir := flag.String("out", "output", "输出目录")
	flag.Parse()

	if *configPath == "" && *projectsDir == "" {
		fmt.Fprintln(os.Stderr, "用法: project-docs --cli --config projects/x.yml [--mode full] 或 --projects <dir>")
		os.Exit(2)
	}

	langs, err := generator.LoadLanguages(*langsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载语言映射失败: %v\n", err)
		os.Exit(1)
	}

	var configs []*config.ProjectConfig
	if *configPath != "" {
		cfg, err := config.Load(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
			os.Exit(1)
		}
		configs = append(configs, cfg)
	}
	if *projectsDir != "" {
		matches, _ := filepath.Glob(filepath.Join(*projectsDir, "*.yml"))
		for _, m := range matches {
			if cfg, err := config.Load(m); err == nil {
				configs = append(configs, cfg)
			}
		}
	}

	for _, cfg := range configs {
		start := time.Now()
		mode := generator.ExportMode(*mode)
		if mode == "" {
			if cfg.ExportMode != "" {
				mode = generator.ExportMode(cfg.ExportMode)
			} else {
				mode = generator.ModeFull
			}
		}

		results, err := scanner.ProcessFiles(scanner.Options{
			ProjectPath:        cfg.ProjectPath,
			Files:              cfg.Files,
			Directories:        cfg.Directories,
			ExcludeDirectories: cfg.ExcludeDirectories,
			ExcludePatterns:    cfg.ExcludePatterns,
			MaxFileSize:        cfg.MaxFileSize,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "扫描失败 %s: %v\n", cfg.ProjectName, err)
			os.Exit(1)
		}
		scanner.SortByRelPath(results)

		content, err := generator.Generate(results, generator.Options{
			ProjectName: cfg.ProjectName,
			ProjectRoot: cfg.ProjectPath,
			Lang:        cfg.MarkdownLang,
			Mode:        mode,
			Languages:   langs,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "生成失败 %s: %v\n", cfg.ProjectName, err)
			os.Exit(1)
		}

		// 分片或单文件输出
		if cfg.SplitTokens > 0 {
			preamble := ""
			if i := strings.Index(content, "### "); i > 0 {
				preamble = content[:i]
			}
			units := make([]token.FileUnit, 0, len(results))
			for _, r := range results {
				rel, _ := filepath.Rel(cfg.ProjectPath, r.FullPath)
				units = append(units, token.FileUnit{Path: filepath.ToSlash(rel), Content: "### 文件: `" + filepath.ToSlash(rel) + "`\n\n```\n" + r.Content + "\n```\n"})
			}
			chunks := token.Split(units, cfg.SplitTokens, preamble)
			for _, c := range chunks {
				name := strings.TrimSuffix(cfg.OutputFile, filepath.Ext(cfg.OutputFile))
				ext := filepath.Ext(cfg.OutputFile)
				out := fmt.Sprintf("%s.part%d%s", name, c.Index, ext)
				p, err := generator.WriteOutput(*outputDir, out, c.Content)
				if err != nil {
					fmt.Fprintf(os.Stderr, "写出失败: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("[%s] %s (part %d/%d)\n", cfg.ProjectName, p, c.Index, c.Total)
			}
		} else {
			p, err := generator.WriteOutput(*outputDir, cfg.OutputFile, content)
			if err != nil {
				fmt.Fprintf(os.Stderr, "写出失败: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("[%s] %s (%d files, ≈%d tokens, %dms)\n",
				cfg.ProjectName, p, len(results), token.EstimateTokens(content), time.Since(start).Milliseconds())
		}
	}
}
