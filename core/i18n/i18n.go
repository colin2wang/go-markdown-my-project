// Package i18n 提供后端文案（日志、校验错误信息）的多语言支持。
// 语言由前端界面语言驱动：GUI 切换语言时调用 SetLocale 同步后端。
package i18n

import (
	"fmt"
	"strings"
	"sync"
)

// Locale 语言代码，与前端 LocaleCode 保持一致。
type Locale string

const (
	ZhCN Locale = "zh-CN"
	EnUS Locale = "en-US"
	JaJP Locale = "ja-JP"
)

var (
	mu      sync.RWMutex
	current = ZhCN
)

// dicts 文案表：键 -> 各语言文案。缺失时回退简体中文，再缺失返回键本身。
var dicts = map[string]map[Locale]string{
	"log.starting": {
		ZhCN: "正在启动 Project Docs 界面…",
		EnUS: "Starting Project Docs GUI…",
		JaJP: "Project Docs を起動しています…",
	},
	"log.started": {
		ZhCN: "应用已启动，前端日志转发已启用",
		EnUS: "Application started; frontend log forwarding enabled",
		JaJP: "アプリを起動しました。フロントエンドへのログ転送が有効です",
	},
	"log.localeSwitched": {
		ZhCN: "界面语言已切换",
		EnUS: "UI language switched",
		JaJP: "表示言語を切り替えました",
	},
	"log.globalSettingsSaved": {
		ZhCN: "全局设置已保存",
		EnUS: "Global settings saved",
		JaJP: "グローバル設定を保存しました",
	},
	"log.listProjectsDirNotFound": {
		ZhCN: "项目列表：projects 目录不存在",
		EnUS: "Project list: projects directory not found",
		JaJP: "プロジェクト一覧：projects ディレクトリが見つかりません",
	},
	"log.listProjectsLoadFail": {
		ZhCN: "项目列表：加载配置失败",
		EnUS: "Project list: failed to load config",
		JaJP: "プロジェクト一覧：設定の読み込みに失敗",
	},
	"log.saveProject": {
		ZhCN: "保存项目配置",
		EnUS: "Project config saved",
		JaJP: "プロジェクト設定を保存しました",
	},
	"log.exportStart": {
		ZhCN: "开始导出",
		EnUS: "Export started",
		JaJP: "エクスポートを開始",
	},
	"log.exportFail": {
		ZhCN: "导出失败",
		EnUS: "Export failed",
		JaJP: "エクスポートに失敗しました",
	},
	"log.redactApplied": {
		ZhCN: "脱敏替换完成",
		EnUS: "Redaction applied",
		JaJP: "マスキング置換が完了しました",
	},
	"log.exportDone": {
		ZhCN: "导出完成",
		EnUS: "Export finished",
		JaJP: "エクスポート完了",
	},
	"log.exportPreview": {
		ZhCN: "预览导出内容（不写盘）",
		EnUS: "Preview export content (no file written)",
		JaJP: "エクスポート内容をプレビュー（ファイル未書き込み）",
	},
	"log.exportSplit": {
		ZhCN: "已按 token 限制分片输出",
		EnUS: "Output split by token limit",
		JaJP: "トークン制限により出力を分割しました",
	},
	"log.compressDone": {
		ZhCN: "输出已压缩",
		EnUS: "Output compressed",
		JaJP: "出力を圧縮しました",
	},
	"log.walkError": {
		ZhCN: "遍历出错",
		EnUS: "Walk error",
		JaJP: "走査エラー",
	},
	"log.walkDirFail": {
		ZhCN: "遍历目录失败",
		EnUS: "Failed to walk directory",
		JaJP: "ディレクトリの走査に失敗",
	},
	"log.readFileFail": {
		ZhCN: "读取文件失败",
		EnUS: "Failed to read file",
		JaJP: "ファイルの読み込みに失敗",
	},
	"log.skipLargeFile": {
		ZhCN: "跳过超过大小上限的文件",
		EnUS: "Skipping file larger than the size limit",
		JaJP: "サイズ上限を超えるファイルをスキップ",
	},
	"log.skipByPattern": {
		ZhCN: "按排除规则跳过文件",
		EnUS: "Skipping file due to exclude pattern",
		JaJP: "除外ルールによりファイルをスキップ",
	},
	"log.skipBinary": {
		ZhCN: "跳过二进制文件",
		EnUS: "Skipping binary file",
		JaJP: "バイナリファイルをスキップ",
	},
	"log.skipNotIncluded": {
		ZhCN: "文件不匹配包含规则，已跳过",
		EnUS: "File does not match include patterns, skipped",
		JaJP: "ファイルが包含パターンに一致しないためスキップ",
	},
	"log.excludeDefaultsFail": {
		ZhCN: "默认排除规则加载失败，本次忽略默认排除",
		EnUS: "Failed to load default excludes; they are ignored for this run",
		JaJP: "デフォルト除外ルールの読み込みに失敗したため無視します",
	},
	"err.projectsDirNotFound": {
		ZhCN: "projects 目录不存在: {dir}（当前工作目录: {wd}）",
		EnUS: "projects directory not found: {dir} (working directory: {wd})",
		JaJP: "projects ディレクトリが見つかりません: {dir}（作業ディレクトリ: {wd}）",
	},
	"val.nameRequired": {
		ZhCN: "项目名称不能为空",
		EnUS: "Project name is required",
		JaJP: "プロジェクト名を入力してください",
	},
	"val.pathRequired": {
		ZhCN: "项目路径不能为空",
		EnUS: "Project path is required",
		JaJP: "プロジェクトパスを入力してください",
	},
	"val.pathNotFound": {
		ZhCN: "路径不存在: {path}",
		EnUS: "Path does not exist: {path}",
		JaJP: "パスが存在しません: {path}",
	},
	"val.pathNotDir": {
		ZhCN: "不是目录: {path}",
		EnUS: "Not a directory: {path}",
		JaJP: "ディレクトリではありません: {path}",
	},
	"val.outputRequired": {
		ZhCN: "输出文件名不能为空",
		EnUS: "Output file name is required",
		JaJP: "出力ファイル名を入力してください",
	},
	"val.outputExt": {
		ZhCN: "扩展名 .{ext} 不是推荐的文档格式（.md/.markdown/.txt）",
		EnUS: "Extension .{ext} is not a recommended document format (.md/.markdown/.txt)",
		JaJP: "拡張子 .{ext} は推奨ドキュメント形式（.md/.markdown/.txt）ではありません",
	},
	"val.notNegative": {
		ZhCN: "不能为负",
		EnUS: "Must not be negative",
		JaJP: "負の値は指定できません",
	},
	"val.excludeEmpty": {
		ZhCN: "排除规则不能为空",
		EnUS: "Exclude patterns must not be empty",
		JaJP: "除外ルールを空にできません",
	},
	"val.regexInvalid": {
		ZhCN: "正则无效: {err}",
		EnUS: "Invalid regex: {err}",
		JaJP: "正規表現が無効: {err}",
	},
	"log.suggest": {
		ZhCN: "自动分析项目结构",
		EnUS: "Analyzing project structure",
		JaJP: "プロジェクト構造を分析中",
	},
	"log.suggestFail": {
		ZhCN: "自动分析失败",
		EnUS: "Project analysis failed",
		JaJP: "プロジェクト分析に失敗しました",
	},
	"log.budgetApplied": {
		ZhCN: "按 token 预算裁剪文件",
		EnUS: "Files trimmed by token budget",
		JaJP: "トークン予算によりファイルを選別しました",
	},
	"infer.pathNotFound": {
		ZhCN: "路径不存在: {path}",
		EnUS: "Path does not exist: {path}",
		JaJP: "パスが存在しません: {path}",
	},
	"infer.pathNotDir": {
		ZhCN: "不是目录: {path}",
		EnUS: "Not a directory: {path}",
		JaJP: "ディレクトリではありません: {path}",
	},
	"infer.noteMainLang": {
		ZhCN: "主语言：{lang}",
		EnUS: "Primary language: {lang}",
		JaJP: "主要言語：{lang}",
	},
	"infer.noteScope": {
		ZhCN: "推荐范围：{files} 个根级文件 + {dirs} 个目录",
		EnUS: "Suggested scope: {files} root files + {dirs} directories",
		JaJP: "推奨範囲：ルートファイル {files} 件 + ディレクトリ {dirs} 件",
	},
	"infer.noteExcludeDirs": {
		ZhCN: "已识别 {n} 类产物/依赖目录，自动排除",
		EnUS: "{n} build/dependency directories detected and excluded",
		JaJP: "ビルド/依存ディレクトリ {n} 種類を検出し除外しました",
	},
	"infer.noteExcludePats": {
		ZhCN: "已建议 {n} 条排除模式（锁文件/生成代码/资源）",
		EnUS: "{n} exclude patterns suggested (lockfiles/generated/assets)",
		JaJP: "除外パターン {n} 件を提案（ロックファイル/生成コード/アセット）",
	},
	"infer.noteMaxSize": {
		ZhCN: "检测到 {n} 个超过 128KB 的文件，已建议 256KB 大小上限",
		EnUS: "{n} files larger than 128KB detected; 256KB size cap suggested",
		JaJP: "128KB 超のファイルが {n} 件あり、256KB のサイズ上限を提案",
	},
	"warn.noExtractor": {
		ZhCN: "{n} 个文件无符号提取器（符号/签名模式将跳过）",
		EnUS: "{n} files have no symbol extractor (skipped in symbols/signatures modes)",
		JaJP: "{n} 件のファイルにシンボル抽出器がありません（symbols/signatures ではスキップ）",
	},
	"infer.noteMode": {
		ZhCN: "估算 ≈{tokens} tokens，推荐 {mode} 模式",
		EnUS: "Estimated ≈{tokens} tokens; mode {mode} recommended",
		JaJP: "推定 ≈{tokens} トークン、{mode} モードを推奨",
	},
}

// SetLocale 设置当前语言；无法识别的代码回退简体中文。
func SetLocale(code string) {
	mu.Lock()
	defer mu.Unlock()
	switch Locale(code) {
	case EnUS:
		current = EnUS
	case JaJP:
		current = JaJP
	default:
		current = ZhCN
	}
}

// Current 返回当前语言。
func Current() Locale {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// T 按 key 取当前语言文案，args 为 k/v 形式的占位符参数（缺失的占位符保留原样）。
func T(key string, args ...any) string {
	mu.RLock()
	loc := current
	mu.RUnlock()

	entry, ok := dicts[key]
	if !ok {
		return key
	}
	msg, ok := entry[loc]
	if !ok {
		msg = entry[ZhCN]
	}
	if len(args) == 0 || !strings.Contains(msg, "{") {
		return msg
	}
	for i := 0; i+1 < len(args); i += 2 {
		k, ok := args[i].(string)
		if !ok {
			continue
		}
		msg = strings.ReplaceAll(msg, "{"+k+"}", fmt.Sprint(args[i+1]))
	}
	return msg
}
