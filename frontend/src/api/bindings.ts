// bindings.ts — Wails 绑定的手写封装（运行时由 wails 生成 window['go']...）
// 为便于类型检查与在浏览器中开发，统一走此模块。

export interface ProjectConfig {
  project_name: string;
  project_path: string;
  output_file: string;
  markdown_lang?: string;
  files?: string[];
  directories?: string[];
  exclude_directories?: string[];
  exclude_patterns?: string[];
  /** 包含规则：激活时仅加载匹配 include_patterns 的文件 */
  include_enabled?: boolean;
  include_patterns?: string[];
  max_file_size?: number;
  export_mode?: string;
  redaction?: {
    enabled: boolean;
    placeholder?: string;
    strategy?: string;
    /** 规则模板档位：strict | standard | relaxed，省略 = standard */
    rule_profile?: string;
    allowlist?: string[];
    custom_patterns?: string[];
  };
  split_tokens?: number;
  template?: string;
  /** 是否合并全局默认排除（config/exclude_defaults.yml）；省略 = true */
  use_default_excludes?: boolean;
  /** 按内容跳过二进制文件（前 8KB 含 NUL）；省略 = true */
  skip_binary_files?: boolean;
  /** 输出文件写入 UTF-8 BOM（避免 Windows 查看器误判编码）；省略 = true */
  write_bom?: boolean;
}

export interface GlobalSettings {
  /** 导出文件编码：utf8_bom | utf8 | gb18030 */
  outputEncoding: string;
  /** 界面语言：zh_cn | en_us | ja_jp（旧配置 auto 会被前端按当前语言回显） */
  interfaceLanguage: string;
}

export interface FileInfo {
  path: string;
  language: string;
  sizeBytes: number;
  lines: number;
}

export interface ProjectSummary {
  configPath: string;
  name: string;
  path: string;
  outputFile: string;
  exportMode: string;
  markdownLang: string;
}

export interface ExportOptions {
  mode: string;
  splitTokens: number;
  /** 精简空白输出（裁剪行尾空白、折叠空行，保持语法） */
  compress?: boolean;
  fileOverrides: string[];
  includeLineNumbers?: boolean;
  maxSignatureLen?: number;
  /** >0 时按重要性评分裁剪文件（超出预算的低价值文件移入省略清单） */
  budgetTokens?: number;
  /** 剥离注释（smart 模式默认开启） */
  stripComments?: boolean;
  /** true = 连 TODO/FIXME、许可证等标记注释一起删除；false/缺省 = 保留标记注释 */
  dropMarkedComments?: boolean;
  /** >0 时截断超长行（按字符计） */
  maxLineLen?: number;
  /** smart 模式：单文件保留全文的 token 上限（≤0 → 后端默认 1200） */
  smartFullTokens?: number;
}

export interface ExportResult {
  outputPaths: string[];
  totalChars: number;
  /** 按实际导出内容估算的 token 数（后端计算） */
  tokenCount: number;
  /** 输出文件字节数（启用空白精简后为精简后大小） */
  outputBytes?: number;
  durationMs: number;
}

export interface PreviewResult {
  /** 展示用内容（超长截断） */
  content: string;
  totalChars: number;
  tokenCount: number;
  truncated: boolean;
}

export interface SensitiveHit {
  file: string;
  line: number;
  rule: string;
  masked: string;
  /** 命中原文，仅供命中抽屉内存展示，不落盘、不进日志 */
  value?: string;
}

export interface FieldError {
  field: string;
  message: string;
}

export interface ScanPreview {
  fileCount: number;
  totalBytes: number;
  /** 估算 token 数（容量规划用） */
  estTokens?: number;
  byLang: Record<string, number>;
  warnings: string[];
}

/** 自动分析项目结构得到的推断报告 */
export interface InferReport {
  fileCount: number;
  scanned: number;
  totalBytes: number;
  /** 主语言（按文件数） */
  mainLang: string;
  byLang: Record<string, number>;
  excludeDirs: string[];
  excludePats: string[];
  /** 已本地化的分析说明 */
  notes: string[];
}

export interface SuggestResult {
  config: ProjectConfig;
  report: InferReport;
}

/** 全局默认排除规则（config/exclude_defaults.yml），编辑器只读展示 */
export interface ExcludeDefaults {
  directories: string[];
  files: string[];
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function go(): any {
  // Wails 运行时注入：App 结构体在 app 包下，绑定为 window['go']['app']['App']
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (window as any).go?.app?.App;
}

export const api = {
  async listProjects(projectsDir: string): Promise<ProjectSummary[]> {
    return (await go()?.ListProjects(projectsDir)) ?? [];
  },
  async loadProject(configPath: string): Promise<ProjectConfig> {
    return go()?.LoadProject(configPath);
  },
  async saveProject(configPath: string, cfg: ProjectConfig): Promise<void> {
    return go()?.SaveProject(configPath, cfg);
  },
  async deleteProject(configPath: string): Promise<void> {
    return go()?.DeleteProject(configPath);
  },
  async scanProject(cfg: ProjectConfig): Promise<FileInfo[]> {
    return (await go()?.ScanProject(cfg)) ?? [];
  },
  async runExport(cfg: ProjectConfig, opt: ExportOptions): Promise<ExportResult> {
    return go()?.RunExport(cfg, opt);
  },
  async previewExport(cfg: ProjectConfig, opt: ExportOptions): Promise<PreviewResult | null> {
    return (await go()?.PreviewExport(cfg, opt)) ?? null;
  },
  async selectDirectory(startDir = ''): Promise<string> {
    return (await go()?.SelectDirectory(startDir)) ?? '';
  },
  // 多选文件对话框，startDir 为默认打开目录；取消返回空数组
  async selectFiles(startDir = ''): Promise<string[]> {
    return (await go()?.SelectFiles(startDir)) ?? [];
  },
  // 扫描项目（或包含目录）下所有文件扩展名（含点前缀，如 ".java"）
  async listFileExtensions(projectPath: string, dirs: string[]): Promise<string[]> {
    return (await go()?.ListFileExtensions(projectPath, dirs)) ?? [];
  },
  async defaultProjectsDir(): Promise<string> {
    return (await go()?.DefaultProjectsDir()) ?? 'config/projects';
  },
  async pathExists(path: string): Promise<{ exists: boolean; isDir: boolean }> {
    // 后端返回结构体 PathCheck {exists, isDir}
    const r = await go()?.PathExists(path);
    return { exists: !!r?.exists, isDir: !!r?.isDir };
  },
  async listSubdirs(path: string): Promise<string[]> {
    return (await go()?.ListSubdirs(path)) ?? [];
  },
  async openPath(path: string): Promise<void> {
    return go()?.OpenPath(path);
  },
  // 构建版本戳（build.ps1 编译期注入，dev 下为 "dev"），标题栏展示
  async getVersion(): Promise<string> {
    return (await go()?.GetVersion()) ?? 'dev';
  },
  // 同步界面语言到后端（后端日志/校验文案随之切换）
  async setLocale(locale: string): Promise<void> {
    return go()?.SetLocale(locale);
  },
  // 全局设置（config/global_settings.yml）
  async getGlobalSettings(): Promise<GlobalSettings | null> {
    return (await go()?.GetGlobalSettings()) ?? null;
  },
  async saveGlobalSettings(s: GlobalSettings): Promise<void> {
    return go()?.SaveGlobalSettings(s);
  },
  async scanSensitive(cfg: ProjectConfig, fileOverrides: string[]): Promise<SensitiveHit[]> {
    return (await go()?.ScanSensitive(cfg, fileOverrides)) ?? [];
  },
  async serializeYAML(cfg: ProjectConfig): Promise<string> {
    return (await go()?.SerializeYAML(cfg)) ?? '';
  },
  async validateConfig(cfg: ProjectConfig): Promise<FieldError[]> {
    return (await go()?.ValidateConfig(cfg)) ?? [];
  },
  async previewScan(cfg: ProjectConfig): Promise<ScanPreview | null> {
    return (await go()?.PreviewScan(cfg)) ?? null;
  },
  async defaultExcludes(): Promise<ExcludeDefaults | null> {
    return (await go()?.DefaultExcludes()) ?? null;
  },
  /** 自动分析项目目录结构，产出推荐配置与推断报告 */
  async suggestConfig(projectPath: string): Promise<SuggestResult> {
    return go()?.SuggestConfig(projectPath);
  },
};
