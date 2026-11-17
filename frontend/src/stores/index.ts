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
  const splitTokens = ref(0);
  const splitParts = ref(1); // 按个数切分：导出为 N 个文件，1 = 不切分
  const includeLineNumbers = ref(true);
  const maxSignatureLen = ref(200);
  // 省 token 策略（用更少 token 承载更多有效信息）
  const budgetTokens = ref(0); // 0 = 不裁剪
  const stripComments = ref(false);
  const keepMarkedComments = ref(true); // 剥离注释时保留 TODO/许可证等标记
  const maxLineLen = ref(0); // 0 = 不截断
  const smartFullTokens = ref(0); // 0 = 后端默认
  const skeletonMinBodyLines = ref(0); // 骨架折叠起点，0 = 后端默认 4
  const headTailKeepLines = ref(0); // 头尾折叠兜底保留行数，0 = 后端默认 20
  const smartSkeleton = ref(false); // 大文件骨架：超限文件折叠函数体（默认关闭，显式下发 false）
  const result = ref<ExportResult | null>(null);

  async function scan(cfg: ProjectConfig) {
    files.value = await api.scanProject(cfg);
    checkedPaths.value = new Set(files.value.map((f) => f.path));
  }

  function buildOptions(): ExportOptions {
    return {
      mode: mode.value,
      splitTokens: splitTokens.value,
      splitParts: splitParts.value,
      fileOverrides: [...checkedPaths.value],
      includeLineNumbers: includeLineNumbers.value,
      maxSignatureLen: maxSignatureLen.value,
      budgetTokens: budgetTokens.value,
      stripComments: stripComments.value,
      dropMarkedComments: !keepMarkedComments.value,
      maxLineLen: maxLineLen.value,
      smartFullTokens: smartFullTokens.value,
      skeletonMinBodyLines: skeletonMinBodyLines.value,
      headTailKeepLines: headTailKeepLines.value,
      smartSkeleton: smartSkeleton.value,
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

  return {
    files, checkedPaths, mode, splitTokens, splitParts,
    includeLineNumbers, maxSignatureLen,
    budgetTokens, stripComments, keepMarkedComments, maxLineLen, smartFullTokens,
    skeletonMinBodyLines, headTailKeepLines, smartSkeleton,
    result, scan, run, preview,
  };
});
