// 编辑对话框单一数据源（设计文档 §5）：draft 是唯一可变状态，
// YAML 预览与校验结果均为派生物；序列化放后端保证注释模板一致。
import { defineStore } from 'pinia';
import { computed, ref, watch } from 'vue';
import { api } from '../api/bindings';
import type { ProjectConfig, ScanPreview } from '../api/bindings';

export function emptyConfig(): ProjectConfig {
  return {
    project_name: '',
    project_path: '',
    output_file: 'output.md',
    markdown_lang: 'zh_cn',
    files: [],
    directories: [],
    exclude_directories: [],
    exclude_patterns: [],
    max_file_size: 0,
    export_mode: 'full',
    split_tokens: 0,
    redaction: { enabled: true, strategy: 'placeholder', placeholder: '[REDACTED]', custom_patterns: [] },
  };
}

// 前端即时校验（§4.1），后端 ValidateConfig 保存时复检
function localValidate(cfg: ProjectConfig): Record<string, string> {
  const e: Record<string, string> = {};
  if (!cfg.project_name?.trim()) e.project_name = '请输入项目名称';
  else if (cfg.project_name.length > 100) e.project_name = '项目名称不能超过 100 字符';
  if (!cfg.project_path?.trim()) e.project_path = '请输入项目路径';
  if (!cfg.output_file?.trim()) e.output_file = '请输入输出文件名';
  else if (!/\.(md|markdown|txt)$/i.test(cfg.output_file)) e.output_file = '必须为 .md/.markdown/.txt 文档格式';
  if ((cfg.max_file_size ?? 0) < 0) e.max_file_size = '不能为负';
  if ((cfg.split_tokens ?? 0) < 0) e.split_tokens = '不能为负';
  if ((cfg.exclude_patterns ?? []).some((p) => !p.trim())) e.exclude_patterns = '排除规则不能为空';
  for (const p of cfg.redaction?.custom_patterns ?? []) {
    try {
      new RegExp(p);
    } catch (err) {
      e.custom_patterns = `正则无效: ${String(err)}`;
      break;
    }
  }
  return e;
}

// 简化行级 diff：同索引行比较（足够给「我改了哪行」提供直觉）
function diffLines(prev: string, next: string): Set<number> {
  const la = prev ? prev.split('\n') : [];
  const lb = next.split('\n');
  const out = new Set<number>();
  for (let i = 0; i < lb.length; i++) if (la[i] !== lb[i]) out.add(i + 1);
  return out;
}

export const useProjectEditor = defineStore('projectEditor', () => {
  const original = ref<ProjectConfig | null>(null); // 打开时快照；null = 新建
  const draft = ref<ProjectConfig | null>(null);
  const configPath = ref('');
  const errors = ref<Record<string, string>>({});
  const yamlText = ref('');
  const changedLines = ref<Set<number>>(new Set());
  const preview = ref<ScanPreview | null>(null);
  const syncing = ref(false);

  const isNew = computed(() => original.value === null);
  const dirty = computed(
    () => JSON.stringify(original.value ?? emptyConfig()) !== JSON.stringify(draft.value ?? emptyConfig())
  );
  const errorCount = computed(() => Object.keys(errors.value).length);

  let timer: ReturnType<typeof setTimeout> | undefined;
  let prevYaml = '';

  watch(draft, () => {
    clearTimeout(timer);
    syncing.value = true;
    timer = setTimeout(async () => {
      if (!draft.value) return;
      errors.value = localValidate(draft.value);
      try {
        const y = await api.serializeYAML(draft.value);
        changedLines.value = diffLines(prevYaml, y);
        prevYaml = y;
        yamlText.value = y;
      } finally {
        syncing.value = false;
      }
    }, 300);
  }, { deep: true });

  function resetDerived() {
    prevYaml = '';
    yamlText.value = '';
    changedLines.value = new Set();
    errors.value = {};
    preview.value = null;
  }

  function openForEdit(path: string, cfg: ProjectConfig) {
    configPath.value = path;
    original.value = JSON.parse(JSON.stringify(cfg));
    draft.value = JSON.parse(JSON.stringify(cfg));
    resetDerived();
  }

  function openForCreate(path: string) {
    configPath.value = path;
    original.value = null;
    draft.value = emptyConfig();
    resetDerived();
  }

  function updateConfigPath(p: string) {
    configPath.value = p;
  }

  function reset() {
    draft.value = original.value ? JSON.parse(JSON.stringify(original.value)) : emptyConfig();
  }

  async function runPreview() {
    if (draft.value) preview.value = await api.previewScan(draft.value);
  }

  /** 保存：后端复检 → 写盘。抛出首个错误信息。 */
  async function save(): Promise<void> {
    if (!draft.value) return;
    const be = await api.validateConfig(draft.value);
    for (const fe of be) errors.value[fe.field] = fe.message;
    if (Object.keys(errors.value).length) throw new Error(Object.values(errors.value)[0]);
    await api.saveProject(configPath.value, draft.value);
  }

  return {
    original, draft, configPath, errors, errorCount, yamlText, changedLines,
    preview, syncing, isNew, dirty,
    openForEdit, openForCreate, updateConfigPath, reset, runPreview, save,
  };
});
