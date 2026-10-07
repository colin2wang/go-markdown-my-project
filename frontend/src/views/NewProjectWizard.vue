<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { api } from '../api/bindings';
import type { ProjectConfig, FileInfo } from '../api/bindings';
import { useI18n } from '../i18n';

const { t } = useI18n();

const props = defineProps<{
  /** 编辑模式传入已有配置；新建传 null */
  initial: ProjectConfig | null;
  /** 配置文件将保存到的路径（编辑时为原路径，新建时由列表页计算） */
  configPath: string;
  projectsDir: string;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'saved', path: string): void;
}>();

const step = ref(1);
const saving = ref(false);
const scanning = ref(false);
const error = ref('');

const form = ref<ProjectConfig>(
  props.initial ? { ...props.initial } : emptyProject()
);

function emptyProject(): ProjectConfig {
  return { project_name: '', project_path: '', output_file: 'output.md', markdown_lang: 'zh_cn', files: [], directories: [] };
}

// —— Step 1：基本信息 ——
const pathChecked = ref<null | { exists: boolean; isDir: boolean }>(null);

async function browseDir() {
  const dir = await api.selectDirectory(form.value.project_path.trim());
  if (!dir) return;
  form.value.project_path = dir;
  if (!form.value.project_name) {
    form.value.project_name = dir.split(/[\\/]/).filter(Boolean).pop() ?? '';
  }
  if (form.value.output_file === 'output.md' || !form.value.output_file) {
    form.value.output_file = (form.value.project_name || 'output') + '.md';
  }
  await checkPath();
}

async function checkPath() {
  const p = form.value.project_path.trim();
  if (!p) { pathChecked.value = null; return; }
  pathChecked.value = await api.pathExists(p);
}

watch(() => form.value.project_path, () => { pathChecked.value = null; });

const step1Valid = computed(() =>
  !!form.value.project_name.trim() &&
  !!form.value.project_path.trim() &&
  !!form.value.output_file.trim() &&
  pathChecked.value?.exists === true && pathChecked.value.isDir
);

// —— Step 2：扫描并勾选范围 ——
const scanFiles = ref<FileInfo[]>([]);
const langFilter = ref('');

const langs = computed(() => [...new Set(scanFiles.value.map((f) => f.language))].sort());
const filteredFiles = computed(() => {
  const q = langFilter.value;
  return q ? scanFiles.value.filter((f) => f.language === q) : scanFiles.value;
});
const checked = ref<Set<string>>(new Set());

const allChecked = computed(() => filteredFiles.value.length > 0 && filteredFiles.value.every((f) => checked.value.has(f.path)));

function toggleAll() {
  const on = !allChecked.value;
  for (const f of filteredFiles.value) {
    if (on) checked.value.add(f.path); else checked.value.delete(f.path);
  }
  checked.value = new Set(checked.value);
}

function onlyCode() {
  checked.value = new Set(scanFiles.value.filter((f) => f.language !== 'Text').map((f) => f.path));
}

// 由勾选的相对路径推导 files / directories（目录 = 出现在其他路径前缀中的项）
function applyCheckedToForm() {
  const dirs = new Set<string>();
  const files: string[] = [];
  for (const rel of checked.value) {
    const segs = rel.split('/');
    if (segs.length > 1) dirs.add(segs[0]);
    else files.push(rel);
  }
  form.value.directories = [...dirs].sort();
  form.value.files = files.sort();
}

async function runScan() {
  scanning.value = true;
  error.value = '';
  try {
    // 编辑模式回显：按现有 files/directories 扫描
    scanFiles.value = await api.scanProject(form.value);
    // 编辑模式默认勾选全部（与列表页导出行为一致）
    checked.value = new Set(scanFiles.value.map((f) => f.path));
  } catch (e) {
    error.value = String(e);
  } finally {
    scanning.value = false;
  }
}

const isEdit = computed(() => !!props.initial);

// 编辑模式进入时直接预扫
watch(() => props.initial, (v) => {
  if (v) { step.value = 2; runScan(); }
  else step.value = 1;
}, { immediate: true });

function next() {
  error.value = '';
  if (step.value === 1) {
    if (!step1Valid.value) return;
    step.value = 2;
    if (scanFiles.value.length === 0) runScan();
  }
}

function back() {
  if (step.value === 2) step.value = 1;
  else if (step.value === 3) step.value = 2;
}

function finish() {
  if (isEdit.value) applyCheckedToForm();
  step.value = 3;
}

// —— Step 3：确认 ——
const savePath = computed(() => props.configPath);

const summary = computed(() => [
  [t('wizard.summaryName'), form.value.project_name],
  [t('wizard.summaryPath'), form.value.project_path],
  [t('wizard.summaryOutput'), form.value.output_file],
  [t('wizard.summaryLang'), form.value.markdown_lang],
  [t('wizard.summaryFiles'), (form.value.files ?? []).join(', ') || '—'],
  [t('wizard.summaryDirs'), (form.value.directories ?? []).join(', ') || '—'],
]);

async function save() {
  saving.value = true;
  error.value = '';
  try {
    await api.saveProject(savePath.value, form.value);
    emit('saved', savePath.value);
  } catch (e) {
    error.value = String(e);
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <div class="modal" @click.self="emit('close')">
    <div class="wizard">
      <header>
        <h2>{{ isEdit ? t('wizard.titleEdit') : t('wizard.titleNew') }}</h2>
        <ol class="steps">
          <li :class="{ active: step === 1, done: step > 1 }">{{ t('wizard.step1') }}</li>
          <li :class="{ active: step === 2, done: step > 2 }">{{ t('wizard.step2') }}</li>
          <li :class="{ active: step === 3 }">{{ t('wizard.step3') }}</li>
        </ol>
      </header>

      <p v-if="error" class="error">{{ error }}</p>

      <!-- Step 1 -->
      <section v-if="step === 1">
        <label>{{ t('wizard.name') }}
          <input v-model="form.project_name" placeholder="My Project" />
        </label>
        <label>{{ t('wizard.path') }}
          <div class="row">
            <input v-model="form.project_path" placeholder="C:/Workspaces/my-project" @blur="checkPath" />
            <button @click="browseDir">{{ t('common.browse') }}</button>
          </div>
        </label>
        <p v-if="pathChecked && !pathChecked.exists" class="field-error">{{ t('wizard.pathMissing') }}</p>
        <p v-else-if="pathChecked && !pathChecked.isDir" class="field-error">{{ t('wizard.pathNotDir') }}</p>
        <label>{{ t('wizard.output') }}
          <input v-model="form.output_file" placeholder="output.md" />
        </label>
        <label>{{ t('wizard.lang') }}
          <select v-model="form.markdown_lang">
            <option value="zh_cn">zh_cn</option>
            <option value="en_us">en_us</option>
          </select>
        </label>
      </section>

      <!-- Step 2 -->
      <section v-else-if="step === 2">
        <div class="row toolbar2">
          <label class="inline">{{ t('wizard.langFilter') }}
            <select v-model="langFilter">
              <option value="">{{ t('wizard.langAll') }}</option>
              <option v-for="l in langs" :key="l" :value="l">{{ l }}</option>
            </select>
          </label>
          <label class="inline"><input type="checkbox" :checked="allChecked" @change="toggleAll" /> {{ t('wizard.selectAll') }}</label>
          <button @click="onlyCode">{{ t('wizard.codeOnly') }}</button>
          <button @click="runScan" :disabled="scanning">{{ scanning ? t('common.scanning') : t('wizard.rescan') }}</button>
        </div>
        <p class="muted">{{ t('wizard.scanSummary', { total: scanFiles.length, checked: checked.size }) }}</p>
        <div class="filelist">
          <p v-if="scanning" class="muted">{{ t('common.scanning') }}</p>
          <p v-else-if="scanFiles.length === 0" class="muted">{{ t('wizard.noFiles') }}</p>
          <label v-for="f in filteredFiles" :key="f.path" class="fileitem">
            <input
              type="checkbox"
              :checked="checked.has(f.path)"
              @change="($event.target as HTMLInputElement).checked ? checked.add(f.path) : checked.delete(f.path)"
            />
            {{ f.path }} <span class="muted">({{ f.language }}, {{ f.sizeBytes }} B)</span>
          </label>
        </div>
      </section>

      <!-- Step 3 -->
      <section v-else>
        <table class="summary">
          <tr v-for="[k, v] in summary" :key="k">
            <th>{{ k }}</th>
            <td>{{ v }}</td>
          </tr>
        </table>
        <p class="muted">{{ t('wizard.saveTo', { path: savePath }) }}</p>
      </section>

      <footer>
        <button v-if="step > 1" @click="back">{{ t('common.back') }}</button>
        <span class="spacer" />
        <button v-if="step === 1" class="primary" :disabled="!step1Valid" @click="next">{{ t('common.next') }}</button>
        <button v-if="step === 2" class="primary" @click="finish">{{ t('common.next') }}</button>
        <button v-if="step === 3" class="primary" :disabled="saving" @click="save">{{ saving ? t('common.saving') : t('common.save') }}</button>
        <button @click="emit('close')">{{ t('common.cancel') }}</button>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.modal { position: fixed; inset: 0; background: rgba(0,0,0,.5); display: flex; align-items: center; justify-content: center; z-index: 10; }
.wizard { background: #16222e; border: 1px solid #3a4a5c; border-radius: 10px; padding: 20px; width: 560px; max-height: 86vh; display: flex; flex-direction: column; }
header h2 { margin: 0 0 8px; }
.steps { display: flex; gap: 8px; list-style: none; padding: 0; margin: 0 0 12px; font-size: 12px; color: #8fa3b8; }
.steps li.active { color: #63b3ed; font-weight: bold; }
.steps li.done { color: #68d391; }
section { overflow-y: auto; }
label { display: block; margin: 8px 0; }
label.inline { display: inline-flex; align-items: center; gap: 4px; margin: 0; }
.row { display: flex; gap: 6px; }
.row input { flex: 1; }
.toolbar2 { align-items: center; margin-bottom: 6px; }
.filelist { max-height: 300px; overflow: auto; border: 1px solid #3a4a5c; border-radius: 6px; padding: 8px; margin: 8px 0; }
.fileitem { display: block; font-size: 13px; margin: 2px 0; }
.summary { width: 100%; border-collapse: collapse; font-size: 13px; }
.summary th { text-align: left; color: #8fa3b8; padding: 4px 8px 4px 0; white-space: nowrap; vertical-align: top; }
.summary td { word-break: break-all; }
footer { display: flex; gap: 8px; margin-top: 12px; }
.spacer { flex: 1; }
.error { color: #ff8383; background: #3a1a1a; border: 1px solid #742a2a; border-radius: 6px; padding: 8px 12px; font-size: 13px; }
.field-error { color: #ff8383; font-size: 12px; margin: 2px 0 6px 2px; }
.muted { color: #8fa3b8; font-size: 12px; }
button { cursor: pointer; border: 1px solid #3a4a5c; background: #1f2d3d; color: #eee; border-radius: 6px; padding: 4px 10px; }
button.primary { background: #2b6cb0; }
button:disabled { opacity: .5; cursor: not-allowed; }
input, select { background: #0f1720; color: #eee; border: 1px solid #3a4a5c; border-radius: 4px; padding: 4px 8px; }
</style>
