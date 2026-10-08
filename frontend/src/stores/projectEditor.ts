// 编辑对话框单一数据源（设计文档 §5）：draft 是唯一可变状态，
// YAML 预览与校验结果均为派生物；序列化放后端保证注释模板一致。
import { defineStore } from 'pinia';
import { computed, ref, watch } from 'vue';
import { api } from '../api/bindings';
import type { ProjectConfig, ScanPreview } from '../api/bindings';
import { t } from '../i18n';

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
    include_enabled: false,
    include_patterns: [],
    max_file_size: 0,
    export_mode: 'full',
    split_tokens: 0,
    redaction: { enabled: true, strategy: 'placeholder', placeholder: '[REDACTED]', custom_patterns: [] },
  };
}

// 前端即时校验（§4.1），后端 ValidateConfig 保存时复检
function localValidate(cfg: ProjectConfig): Record<string, string> {
  const e: Record<string, string> = {};
  if (!cfg.project_name?.trim()) e.project_name = t('editor.errNameRequired');
  else if (cfg.project_name.length > 100) e.project_name = t('editor.errNameTooLong');
  if (!cfg.project_path?.trim()) e.project_path = t('editor.errPathRequired');
  if (!cfg.output_file?.trim()) e.output_file = t('editor.errOutputRequired');
  else if (!/\.(md|markdown|txt)$/i.test(cfg.output_file)) e.output_file = t('editor.errOutputExt');
  if ((cfg.max_file_size ?? 0) < 0) e.max_file_size = t('editor.errMaxSize');
  if ((cfg.split_tokens ?? 0) < 0) e.split_tokens = t('editor.errSplitTokens');
  if ((cfg.exclude_patterns ?? []).some((p) => !p.trim())) e.exclude_patterns = t('editor.errExcludeEmpty');
  if ((cfg.include_patterns ?? []).some((p) => !p.trim())) e.include_patterns = t('editor.errIncludeEmpty');
  for (const p of cfg.redaction?.custom_patterns ?? []) {
    try {
      new RegExp(p);
    } catch (err) {
      e.custom_patterns = t('editor.errRegex', { err: String(err) });
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

  // 包含规则激活开关（对应 yml include_enabled）：checkbox 与 draft 双向绑定
  const includeEnabled = computed({
    get: () => draft.value?.include_enabled ?? false,
    set: (v: boolean) => {
      if (draft.value) draft.value.include_enabled = v;
    },
  });

  const dirty = computed(
    () => JSON.stringify(original.value ?? emptyConfig()) !== JSON.stringify(draft.value ?? emptyConfig())
  );
  const errorCount = computed(() => Object.keys(errors.value).length);

  let timer: ReturnType<typeof setTimeout> | undefined;
  let prevYaml = '';

  // 序列化比对触发：TagListInput 等子组件对数组整体赋值时，
  // Pinia ref 上的 deep watch 可能漏触发嵌套数组变更，用 JSON 串做源最可靠。
  // 注意：draft 内容与上次完全相同时（如二次打开同一配置）不会触发，故打开后需显式 syncYaml。
  watch(
    () => (draft.value ? JSON.stringify(draft.value) : ''),
    () => {
      clearTimeout(timer);
      timer = setTimeout(syncYaml, 300);
    }
  );

  /** 由当前 draft 重建校验与 YAML 镜像。 */
  async function syncYaml() {
    syncing.value = true;
    try {
      if (!draft.value) return;
      errors.value = localValidate(draft.value);
      const y = await api.serializeYAML(draft.value);
      changedLines.value = diffLines(prevYaml, y);
      prevYaml = y;
      yamlText.value = y;
    } finally {
      syncing.value = false;
    }
  }

  function resetDerived() {
    prevYaml = '';
    yamlText.value = '';
    changedLines.value = new Set();
    errors.value = {};
    preview.value = null;
  }

  function openForEdit(path: string, cfg: ProjectConfig) {
    configPath.value = path;
    // original 与 draft 走同一归一化，保证 dirty 比较口径一致
    original.value = normalizeConfig(cfg);
    draft.value = normalizeConfig(cfg);
    resetDerived();
    // 与上次会话内容相同时 watch 不触发，显式同步一次填充右侧 YAML
    syncYaml();
  }

  function openForCreate(path: string) {
    configPath.value = path;
    original.value = null;
    draft.value = emptyConfig();
    resetDerived();
    syncYaml();
  }

  // YAML 中省略的数组字段经反序列化后为 undefined，模板里的非空断言与
  // TagListInput 等子组件都假设数组存在，打开时统一归一化
  function normalizeConfig(cfg: ProjectConfig): ProjectConfig {
    const copy = JSON.parse(JSON.stringify(cfg));
    copy.files ??= [];
    copy.directories ??= [];
    copy.exclude_directories ??= [];
    copy.exclude_patterns ??= [];
    copy.include_patterns ??= [];
    copy.include_enabled ??= false;
    if (copy.redaction) {
      copy.redaction.custom_patterns ??= [];
      copy.redaction.allowlist ??= [];
    }
    return copy;
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
    preview, syncing, isNew, includeEnabled, dirty,
    openForEdit, openForCreate, updateConfigPath, reset, runPreview, save,
  };
});
