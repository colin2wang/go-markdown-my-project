<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { useProjectsStore, useExportStore } from '../stores';
import { api } from '../api/bindings';
import type { FileInfo, ProjectConfig, PreviewResult } from '../api/bindings';
import FileTree from './FileTree.vue';
import SensitiveDrawer from './SensitiveDrawer.vue';
import { estimateTokensFromSize, fmtNum } from '../estimate';
import { useLogStore } from '../stores/log';
import { useI18n } from '../i18n';
import Icon from '../components/ui/Icon.vue';
import type { IconName } from '../components/ui/Icon.vue';

const props = defineProps<{ configPath: string }>();
const router = useRouter();
const projects = useProjectsStore();
const exp = useExportStore();
const log = useLogStore();
const { t } = useI18n();

const loadError = ref('');
const scanning = ref(false);
const exporting = ref(false);

const MODES = computed(() => [
  { value: 'full', icon: 'doc', label: t('wb.modes.full.label'), desc: t('wb.modes.full.desc') },
  { value: 'smart', icon: 'gear', label: t('wb.modes.smart.label'), desc: t('wb.modes.smart.desc') },
  { value: 'files', icon: 'folder', label: t('wb.modes.files.label'), desc: t('wb.modes.files.desc') },
  { value: 'symbols', icon: 'search', label: t('wb.modes.symbols.label'), desc: t('wb.modes.symbols.desc') },
  { value: 'signatures', icon: 'open', label: t('wb.modes.signatures.label'), desc: t('wb.modes.signatures.desc') },
]);

onMounted(async () => {
  loadError.value = '';
  try {
    await projects.open(props.configPath);
    await rescan();
  } catch (e) {
    loadError.value = String(e);
  }
});

async function rescan() {
  if (!projects.current) return;
  log.info(t('wb.logScan', { name: projects.current.project_name }));
  scanning.value = true;
  try {
    await exp.scan(projects.current);
    log.info(t('wb.logScanned', { n: exp.files.length }));
  } finally {
    scanning.value = false;
  }
}

// 导出模式切换（智能省流默认包含注释剥离，其余模式默认保留注释；用户仍可手动调整）
watch(() => exp.mode, (m, old) => {
  previewData.value = null; // 选项变化后旧预览失效
  exp.stripComments = m === 'smart';
  if (old) log.info(t('wb.logModeSwitch', { from: old, to: m }));
});

// —— 勾选统计（左栏底/状态条共用），300ms 防抖避免大项目勾选时频繁重算 ——
import { onUnmounted } from 'vue';

const rawChecked = computed<FileInfo[]>(() => exp.files.filter((f) => exp.checkedPaths.has(f.path)));
const debouncedFiles = ref<FileInfo[]>(rawChecked.value);
let debounceTimer: ReturnType<typeof setTimeout> | undefined;
watch(rawChecked, (v) => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => { debouncedFiles.value = v; previewData.value = null; }, 300);
}, { deep: false });
onUnmounted(() => clearTimeout(debounceTimer));

const checkedFiles = computed(() => debouncedFiles.value);
const checkedBytes = computed(() => checkedFiles.value.reduce((s, f) => s + f.sizeBytes, 0));
const estTokens = computed(() => checkedFiles.value.reduce((s, f) => s + estimateTokensFromSize(f.sizeBytes), 0));
function fmtSize(n: number): string {
  if (n < 1024) return n + ' B';
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB';
  return (n / 1024 / 1024).toFixed(1) + ' MB';
}

// —— 敏感信息（任务5 接入）——
const sensitive = ref<{ state: 'none' | 'hit' | 'unscanned'; count: number; hits: SensitiveHit[] }>({
  state: 'unscanned', count: 0, hits: [],
});
export interface SensitiveHit { file: string; line: number; rule: string; masked: string }
const showSensitiveDrawer = ref(false);
const sensitiveScanning = ref(false);
const redactEnabled = ref(true);
const redactPlaceholder = ref('[REDACTED]');
const redactStrategy = ref('placeholder');

async function scanSensitive() {
  if (!projects.current) return;
  log.info(t('wb.logSensitiveScan', { n: exp.checkedPaths.size, strategy: redactStrategy.value }));
  sensitiveScanning.value = true;
  try {
    const hits: SensitiveHit[] = await api.scanSensitive(projects.current, [...exp.checkedPaths]);
    sensitive.value = { state: hits.length ? 'hit' : 'none', count: hits.length, hits };
    if (hits.length) log.warn(t('wb.logHits', { n: hits.length }));
    else log.info(t('wb.logNoHit'));
  } catch (e) {
    log.warn(t('wb.logSensitiveFail', { err: String(e) }));
    sensitive.value = { state: 'unscanned', count: 0, hits: [] };
  } finally {
    sensitiveScanning.value = false;
  }
}

// —— 大小控制 ——
const splitEnabled = ref(false);
const splitTokens = ref(100000);
const estParts = computed(() =>
  splitEnabled.value && splitTokens.value > 0 ? Math.max(1, Math.ceil(estTokens.value / splitTokens.value)) : 1
);

// 按当前勾选的估算填入预算：取 80%（留给文件树与清单等脚手架文本），对齐到千位
function fillBudget() {
  exp.budgetTokens = Math.max(1000, Math.round((estTokens.value * 0.8) / 1000) * 1000);
}

// —— 导出前未扫描确认 ——
const showUnscannedConfirm = ref(false);

function onExportClick() {
  if (redactEnabled.value && sensitive.value.state === 'unscanned') {
    showUnscannedConfirm.value = true;
    return;
  }
  doExport();
}

async function doExport() {
  if (!projects.current) return;
  log.info(t('wb.logExport', {
    mode: exp.mode,
    redact: redactEnabled.value ? t('wb.redactOn') : t('wb.redactOff'),
    n: checkedFiles.value.length,
  }));
  previewData.value = null; // 导出后右栏切换到结果卡片
  exporting.value = true;
  exp.redact = redactEnabled.value;
  exp.splitTokens = splitEnabled.value ? splitTokens.value : 0;
  try {
    await exp.run(projects.current);
    if (exp.result) log.info(t('wb.logExportOk', { paths: exp.result.outputPaths.join(', '), ms: exp.result.durationMs }));
  } catch (e) {
    log.error(t('wb.logExportFail', { err: e instanceof Error ? e.message : String(e) }));
  } finally {
    exporting.value = false;
  }
}

// —— 干跑预览：后端生成完整内容但不写盘 ——
const previewing = ref(false);
const previewData = ref<PreviewResult | null>(null);

async function doPreview() {
  if (!projects.current) return;
  previewing.value = true;
  exp.redact = redactEnabled.value;
  exp.result = null; // 预览优先于旧导出结果卡片（与 doExport 清 previewData 对称）
  try {
    previewData.value = await exp.preview(projects.current);
    if (previewData.value) log.info(t('wb.logPreviewOk', { chars: fmtNum(previewData.value.totalChars) }));
  } catch (e) {
    log.warn(t('wb.logPreviewFail', { err: String(e) }));
    previewData.value = null;
  } finally {
    previewing.value = false;
  }
}

// —— 右栏：干跑预览内容 / 本地大纲（未预览时）/ 结果卡片（导出后）——
const outline = computed(() => buildOutline(projects.current, checkedFiles.value, exp.mode));

function buildOutline(cfg: ProjectConfig | null, files: FileInfo[], mode: string): string[] {
  if (!cfg) return [];
  const lines: string[] = [
    t('wb.outlineTitle', { name: cfg.project_name }),
    t('wb.outlineTree'),
  ];
  if (mode === 'full') {
    lines.push(t('wb.outlineFiles', { n: files.length }));
    for (const f of files.slice(0, 30)) lines.push(t('wb.outlineFile', { path: f.path }));
    if (files.length > 30) lines.push(t('wb.outlineMore', { n: files.length }));
  } else if (mode === 'symbols') {
    lines.push(t('wb.outlineSymbols', { n: files.length }));
  } else if (mode === 'signatures') {
    lines.push(t('wb.outlineSignatures', { n: files.length }));
  }
  return lines;
}

async function openOutputDir() {
  if (!exp.result) return;
  log.info(t('wb.logOpenDir', { path: exp.result.outputPaths[0] }));
  await api.openPath(exp.result.outputPaths[0]);
}

async function copyResult() {
  if (!exp.result) return;
  try {
    await navigator.clipboard.writeText(exp.result.outputPaths.join('\n'));
    log.info(t('wb.logCopyPath', { paths: exp.result.outputPaths.join(', ') }));
  } catch {
    log.warn(t('wb.logCopyFail'));
  }
}
</script>

<template>
  <main class="workbench">
    <p v-if="loadError" class="error">{{ loadError }} <button @click="router.push('/')">{{ t('wb.backList') }}</button></p>

    <template v-else-if="projects.current">
      <!-- 左栏：项目信息 + 文件树 -->
      <aside class="pane pane-tree">
        <div class="proj-head">
          <span class="proj-name" :title="projects.current.project_name">{{ projects.current.project_name }}</span>
          <span class="muted path" :title="projects.current.project_path">{{ projects.current.project_path }}</span>
        </div>
        <FileTree :files="exp.files" :checked="exp.checkedPaths" @update:checked="(v) => (exp.checkedPaths = v)" />
      </aside>

      <!-- 中栏：导出选项 -->
      <section class="pane pane-options">
        <h3>{{ t('wb.exportMode') }}</h3>
        <div
          v-for="m in MODES" :key="m.value"
          class="mode-card" :class="{ active: exp.mode === m.value }"
          @click="exp.mode = m.value"
        >
          <div class="mode-line"><Icon :name="m.icon as IconName" /> {{ m.label }}</div>
          <div class="mode-desc">{{ m.desc }}</div>
          <Icon name="check" :size="12" class="mode-check" />
        </div>

        <!-- 符号类模式子选项（设计文档 §8） -->
        <template v-if="exp.mode === 'symbols' || exp.mode === 'signatures'">
          <label class="inline sub">
            <input v-model="exp.includeLineNumbers" type="checkbox" />
            {{ t('wb.lineNumbers') }}
          </label>
          <label class="block sub">
            {{ t('wb.sigLen') }}
            <input v-model.number="exp.maxSignatureLen" type="number" class="num" min="40" max="2000" />
          </label>
        </template>

        <h3>{{ t('wb.redaction') }}</h3>
        <label class="inline">
          <input v-model="redactEnabled" type="checkbox" />
          {{ t('wb.redactEnable') }}
        </label>
        <template v-if="redactEnabled">
          <label class="block">{{ t('wb.strategy') }}:
            <select v-model="redactStrategy">
              <option value="placeholder">{{ t('wb.strategyPlaceholder') }}</option>
              <option value="drop_line">{{ t('wb.strategyDropLine') }}</option>
            </select>
          </label>
          <label class="block">{{ t('wb.replaceWith') }}:
            <input v-model="redactPlaceholder" :disabled="redactStrategy === 'drop_line'" />
          </label>
          <button :disabled="sensitiveScanning" @click="scanSensitive">
            {{ sensitiveScanning ? t('common.scanning') : t('wb.scanSensitive') }}
          </button>
          <div class="chip" :class="sensitive.state">
            <template v-if="sensitive.state === 'none'">{{ t('wb.noHit') }}</template>
            <template v-else-if="sensitive.state === 'hit'">
              {{ t('wb.hits', { n: sensitive.count }) }}
              <button class="link" @click="showSensitiveDrawer = true">{{ t('wb.viewHits') }}</button>
            </template>
            <template v-else>{{ t('wb.unscanned') }}</template>
          </div>
        </template>

        <h3>{{ t('wb.tokenSaving') }}</h3>
        <label class="inline">
          <input v-model="exp.stripComments" type="checkbox" />
          {{ t('wb.stripComments') }}
        </label>
        <label class="inline wrap">
          <input v-model="exp.keepMarkedComments" type="checkbox" :disabled="!exp.stripComments" />
          {{ t('wb.keepMarkedComments') }}
          <span class="muted">{{ t('wb.keepMarkedHint') }}</span>
        </label>
        <label class="inline wrap">
          {{ t('wb.budgetTokens') }}
          <input v-model.number="exp.budgetTokens" type="number" class="num" min="0" step="1000" />
          <button class="mini" :title="t('wb.budgetHint')" @click="fillBudget">{{ t('wb.fillBudget') }}</button>
        </label>
        <label class="inline wrap">
          {{ t('wb.maxLineLen') }}
          <input v-model.number="exp.maxLineLen" type="number" class="num" min="0" step="40" />
          <span class="muted">{{ t('wb.maxLineHint') }}</span>
        </label>
        <p v-if="exp.budgetTokens > 0" class="muted">{{ t('wb.budgetHint') }}</p>

        <h3>{{ t('wb.sizeControl') }}</h3>
        <label class="inline">
          <input v-model="splitEnabled" type="checkbox" />
          {{ t('wb.overTokens') }} <input v-model.number="splitTokens" type="number" class="num" :disabled="!splitEnabled" /> token
        </label>
        <label class="inline">
          <input v-model="exp.compress" type="checkbox" />
          {{ t('wb.compressOutput') }}
        </label>
        <p v-if="splitEnabled" class="muted">{{ t('wb.estParts', { n: estParts }) }}</p>
      </section>

      <!-- 右栏：预览 / 结果 -->
      <section class="pane pane-result">
        <template v-if="exp.result">
          <div class="result-card ok">
            <h3>{{ t('wb.resultTitle') }}</h3>
            <table>
              <tr><th>{{ t('wb.resultOutput') }}</th><td>{{ exp.result.outputPaths.join(', ') }}</td></tr>
              <tr><th>{{ t('wb.resultChars') }}</th><td>{{ fmtNum(exp.result.totalChars) }}</td></tr>
              <tr v-if="exp.result.outputBytes"><th>{{ t('wb.resultBytes') }}</th><td>{{ fmtSize(exp.result.outputBytes) }}</td></tr>
              <tr><th>{{ t('wb.resultTokens') }}</th><td>{{ fmtNum(exp.result.tokenCount ?? estTokens) }}</td></tr>
              <tr><th>{{ t('wb.resultDuration') }}</th><td>{{ exp.result.durationMs }} ms</td></tr>
              <tr><th>{{ t('wb.resultRedaction') }}</th><td>{{ redactEnabled ? t('wb.hitsShort', { n: sensitive.count }) : t('common.off') }}</td></tr>
            </table>
            <div class="result-actions">
              <button @click="openOutputDir">{{ t('wb.openDir') }}</button>
              <button @click="copyResult">{{ t('wb.copyPath') }}</button>
              <button @click="doExport" :disabled="exporting">{{ t('wb.reExport') }}</button>
            </div>
          </div>
        </template>
        <template v-else-if="previewData">
          <h3>{{ t('wb.previewTitle') }}</h3>
          <p class="muted">{{ t('wb.previewStats', { chars: fmtNum(previewData.totalChars), tokens: fmtNum(previewData.tokenCount) }) }}<template v-if="previewData.truncated"> · {{ t('wb.previewTruncated') }}</template></p>
          <pre class="outline preview-content">{{ previewData.content }}</pre>
        </template>
        <template v-else-if="checkedFiles.length === 0">
          <div class="result-empty">
            <Icon name="doc" :size="28" class="empty-icon" />
            <p class="empty-title">{{ t('wb.emptyTitle') }}</p>
            <p class="empty-hint">{{ t('wb.emptyHint') }}</p>
          </div>
        </template>
        <template v-else>
          <h3>{{ t('wb.previewTitle') }}</h3>
          <pre class="outline"><template v-for="l in outline" :key="l">{{ l }}
</template></pre>
          <p class="muted">{{ t('wb.previewSummary', { mode: MODES.find((m) => m.value === exp.mode)?.label, state: redactEnabled ? t('common.on') : t('common.off') }) }}</p>
        </template>
      </section>
    </template>

    <!-- 底部状态条 -->
    <footer v-if="projects.current" class="statusbar">
      <div v-if="exporting || previewing || scanning" class="bar-progress" />
      <span class="status-left">
        <template v-if="exporting">{{ t('wb.statusExporting') }}</template>
        <template v-else>{{ t('wb.statusReady') }}</template>
        <span class="muted">{{ t('wb.statusStats', { count: checkedFiles.length, size: fmtSize(checkedBytes), tokens: fmtNum(estTokens) }) }}</span>
      </span>
      <span class="status-right">
        <button :disabled="exporting || previewing" @click="doPreview">{{ previewing ? t('wb.previewing') : t('wb.previewBtn') }}</button>
        <button class="primary" :disabled="exporting" @click="onExportClick">{{ t('wb.exportBtn') }}</button>
      </span>
    </footer>

    <SensitiveDrawer
      v-if="showSensitiveDrawer"
      :hits="sensitive.hits"
      @close="showSensitiveDrawer = false"
    />

    <div v-if="showUnscannedConfirm" class="modal" @click.self="showUnscannedConfirm = false">
      <div class="modal-body">
        <h3>{{ t('wb.unscannedTitle') }}</h3>
        <p>{{ t('wb.unscannedBody') }}</p>
        <div class="actions">
          <button class="primary" @click="showUnscannedConfirm = false; scanSensitive()">{{ t('wb.scanAndContinue') }}</button>
          <button @click="showUnscannedConfirm = false; doExport()">{{ t('wb.directExport') }}</button>
          <button @click="showUnscannedConfirm = false">{{ t('common.cancel') }}</button>
        </div>
      </div>
    </div>
  </main>
</template>

<style scoped>
.workbench {
  flex: 1; min-height: 0;
  display: grid;
  grid-template-columns: minmax(360px, 27%) minmax(290px, 21%) minmax(0, 1fr);
  gap: 12px; padding: 12px;
  color: var(--fg-1);
}
@media (max-width: 1280px) {
  .workbench { grid-template-columns: minmax(320px, 1fr) minmax(280px, 1fr); }
  .pane-result { grid-column: 1 / -1; }
}
.pane { overflow-y: auto; background: var(--bg-2); border: 1px solid var(--line-2); border-radius: var(--r-md); padding: 12px; }
.pane-tree { display: flex; flex-direction: column; min-height: 0; }
/* 项目名 + 路径压到一行，给深层级文件树腾出空间 */
.proj-head { flex: none; display: flex; align-items: baseline; gap: 8px; min-width: 0; margin-bottom: 8px; }
.proj-name { flex: none; max-width: 45%; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.path { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pane-options h3, .pane-result h3 { font-size: 13px; color: var(--fg-2); margin: 14px 0 8px; text-transform: none; }
.pane-options h3:first-child { margin-top: 0; }
.mode-card { position: relative; border: 1px solid var(--line-2); border-radius: var(--r-md); padding: 10px; margin-bottom: 8px; cursor: pointer; transition: border-color .12s, background .12s; }
.mode-card:hover { border-color: var(--line-2); background: var(--bg-3); }
.mode-card.active { border-color: var(--accent); background: var(--accent-dim); }
.mode-card.active .mode-line { color: var(--accent); }
.mode-line { font-size: 14px; padding-right: 18px; }
.mode-desc { font-size: 12px; color: var(--fg-2); margin-top: 4px; }
.mode-check {
  position: absolute; top: 8px; right: 10px;
  color: var(--accent);
  opacity: 0; transform: scale(.6);
  transition: opacity .12s, transform .12s;
}
.mode-card.active .mode-check { opacity: 1; transform: scale(1); }
.inline { display: flex; align-items: center; gap: 6px; margin: 6px 0; }
.block { display: block; margin: 6px 0; }
.num { width: 100px; }
.chip { display: inline-block; border: 1px solid var(--line-2); border-radius: var(--r-sm); padding: 3px 10px; font-size: 12px; margin-top: 6px; }
.chip.none { color: var(--ok); }
.chip.hit { color: var(--warn); }
.chip.unscanned { color: var(--fg-3); }
.outline { background: var(--bg-3); padding: 10px; border-radius: var(--r-md); font-size: 12px; line-height: 1.7; overflow-x: auto; }
.result-empty {
  height: 100%; min-height: 260px;
  display: flex; flex-direction: column;
  align-items: center; justify-content: center;
  gap: 6px; color: var(--fg-3); text-align: center;
}
.empty-icon { color: var(--fg-3); opacity: .5; }
.empty-title { margin: 0; font-size: 14px; color: var(--fg-2); }
.empty-hint { margin: 0; font-size: 12px; }
.outline.preview-content { white-space: pre-wrap; word-break: break-word; }
.result-card { border-radius: var(--r-md); padding: 14px; }
.result-card.ok { border: 1px solid var(--ok); background: transparent; }
.result-card h3 { margin: 0 0 10px; color: var(--ok); }
.result-card table { width: 100%; border-collapse: collapse; font-size: 13px; }
.result-card th { text-align: left; color: var(--fg-2); padding: 4px 8px 4px 0; white-space: nowrap; }
.result-card td { word-break: break-all; }
.result-actions { display: flex; gap: 8px; margin-top: 12px; flex-wrap: wrap; }
.statusbar {
  grid-column: 1 / -1;
  position: relative; overflow: hidden;
  height: 44px; flex: none;
  display: flex; align-items: center; justify-content: space-between;
  padding: 0 16px;
  background: var(--bg-1); border: 1px solid var(--line); border-radius: var(--r-md);
}
/* 忙碌时的不确定进度条（导出/预览/扫描共用） */
.bar-progress {
  position: absolute; left: 0; top: 0; height: 2px; width: 35%;
  background: var(--accent);
  animation: bar-slide 1.1s linear infinite;
}
@keyframes bar-slide {
  from { transform: translateX(-100%); }
  to { transform: translateX(320%); }
}
.inline.wrap { flex-wrap: wrap; }
.mini { font-size: 12px; padding: 1px 8px; }
.status-left { display: flex; gap: 12px; align-items: center; }
.status-right { display: flex; gap: 8px; }
.muted { color: var(--fg-3); font-size: 12px; }
.error { grid-column: 1 / -1; color: var(--err); background: transparent; border: 1px solid var(--err); border-radius: var(--r-md); padding: 8px 12px; }
button { cursor: pointer; border: 1px solid var(--line-2); background: var(--bg-3); color: var(--fg-1); border-radius: var(--r-sm); padding: 4px 10px; }
button.primary { background: var(--accent); border-color: var(--accent); color: var(--fg-on-accent); font-weight: 600; }
button:disabled { opacity: .5; cursor: not-allowed; }
.link { border: none; background: none; color: var(--fg-2); padding: 2px 4px; }
.link:hover { color: var(--fg-1); }
input, select { background: var(--bg-3); color: var(--fg-1); border: 1px solid var(--line-2); border-radius: var(--r-sm); padding: 4px 8px; }
.modal { position: fixed; inset: 0; background: var(--mask); display: flex; align-items: center; justify-content: center; z-index: 30; }
.modal-body { background: var(--bg-2); border: 1px solid var(--line-2); border-radius: var(--r-lg); padding: 20px; width: 420px; }
.actions { display: flex; gap: 8px; margin-top: 12px; }
</style>
