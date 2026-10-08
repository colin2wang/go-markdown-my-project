import { defineStore } from 'pinia';
import { ref } from 'vue';
import type { ProjectSummary, ProjectConfig, FileInfo, ExportOptions, ExportResult, PreviewResult } from '../api/bindings';
import { api } from '../api/bindings';

export const useProjectsStore = defineStore('projects', () => {
  const list = ref<ProjectSummary[]>([]);
  const projectsDir = ref('config/projects');
  const current = ref<ProjectConfig | null>(null);
  const currentPath = ref('');
  const error = ref('');

  async function refresh() {
    error.value = '';
    try {
      list.value = await api.listProjects(projectsDir.value);
    } catch (e) {
      list.value = [];
      error.value = String(e);
    }
  }

  async function open(configPath: string) {
    current.value = await api.loadProject(configPath);
    currentPath.value = configPath;
  }

  async function save(configPath: string, cfg: ProjectConfig) {
    await api.saveProject(configPath, cfg);
    await refresh();
  }

  async function remove(configPath: string) {
    await api.deleteProject(configPath);
    await refresh();
  }

  return { list, projectsDir, current, currentPath, error, refresh, open, save, remove };
});

export const useExportStore = defineStore('export', () => {
  const files = ref<FileInfo[]>([]);
  const checkedPaths = ref<Set<string>>(new Set());
  const mode = ref<string>('full');
  const redact = ref(true);
  const splitTokens = ref(0);
  const compress = ref(false); // 压缩输出（.gz），默认关
  const includeLineNumbers = ref(true);
  const maxSignatureLen = ref(200);
  const result = ref<ExportResult | null>(null);

  async function scan(cfg: ProjectConfig) {
    files.value = await api.scanProject(cfg);
    checkedPaths.value = new Set(files.value.map((f) => f.path));
  }

  function buildOptions(): ExportOptions {
    return {
      mode: mode.value,
      redact: redact.value,
      splitTokens: splitTokens.value,
      compress: compress.value,
      fileOverrides: [...checkedPaths.value],
      includeLineNumbers: includeLineNumbers.value,
      maxSignatureLen: maxSignatureLen.value,
    };
  }

  async function run(cfg: ProjectConfig) {
    result.value = await api.runExport(cfg, buildOptions());
    return result.value;
  }

  /** 干跑预览：生成完整内容但不写盘 */
  async function preview(cfg: ProjectConfig): Promise<PreviewResult | null> {
    return api.previewExport(cfg, buildOptions());
  }

  return { files, checkedPaths, mode, redact, splitTokens, compress, includeLineNumbers, maxSignatureLen, result, scan, run, preview };
});
