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
}

export interface ExportResult {
  outputPaths: string[];
  totalChars: number;
  durationMs: number;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function go(): any {
  // Wails 运行时注入
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (window as any).go?.main?.App;
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
};
