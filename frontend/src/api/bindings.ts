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
  max_file_size?: number;
  export_mode?: string;
  redaction?: {
    enabled: boolean;
    placeholder?: string;
    strategy?: string;
    allowlist?: string[];
    custom_patterns?: string[];
  };
  split_tokens?: number;
  template?: string;
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
  redact: boolean;
  splitTokens: number;
  fileOverrides: string[];
  includeLineNumbers?: boolean;
  maxSignatureLen?: number;
}

export interface ExportResult {
  outputPaths: string[];
  totalChars: number;
  durationMs: number;
}

export interface SensitiveHit {
  file: string;
  line: number;
  rule: string;
  masked: string;
}

export interface FieldError {
  field: string;
  message: string;
}

export interface ScanPreview {
  fileCount: number;
  totalBytes: number;
  byLang: Record<string, number>;
  warnings: string[];
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
  async selectDirectory(startDir = ''): Promise<string> {
    return (await go()?.SelectDirectory(startDir)) ?? '';
  },
  async defaultProjectsDir(): Promise<string> {
    return (await go()?.DefaultProjectsDir()) ?? 'config/projects';
  },
  async pathExists(path: string): Promise<{ exists: boolean; isDir: boolean }> {
    // Go 多返回值 (bool, bool) 绑定为 [boolean, boolean]
    const r = await go()?.PathExists(path);
    return { exists: !!r?.[0], isDir: !!r?.[1] };
  },
  async openPath(path: string): Promise<void> {
    return go()?.OpenPath(path);
  },
  // 同步界面语言到后端（后端日志/校验文案随之切换）
  async setLocale(locale: string): Promise<void> {
    return go()?.SetLocale(locale);
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
};
