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
  { value: 'full', icon: '📄', label: t('wb.modes.full.label'), desc: t('wb.modes.full.desc') },
  { value: 'files', icon: '🗂', label: t('wb.modes.files.label'), desc: t('wb.modes.files.desc') },
  { value: 'symbols', icon: '🔎', label: t('wb.modes.symbols.label'), desc: t('wb.modes.symbols.desc') },
  { value: 'signatures', icon: '✍️', label: t('wb.modes.signatures.label'), desc: t('wb.modes.signatures.desc') },
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

// 导出模式切换
watch(() => exp.mode, (m, old) => {
  previewData.value = null; // 选项变化后旧预览失效
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
          <button class="link" @click="router.push('/')">{{ t('wb.back') }}</button>
          <div class="proj-name">{{ projects.current.project_name }}</div>
          <div class="muted path">{{ projects.current.project_path }}</div>
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
          <div class="mode-line"><span class="radio" :class="{ on: exp.mode === m.value }" /> {{ m.icon }} {{ m.label }}</div>
          <div class="mode-desc">{{ m.desc }}</div>
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
  grid-template-columns: 320px 300px 1fr;
  gap: 12px; padding: 12px;
  color: #eee;
}
.pane { overflow-y: auto; background: #16222e; border: 1px solid #3a4a5c; border-radius: 8px; padding: 12px; }
.pane-tree { display: flex; flex-direction: column; min-height: 0; }
.proj-head { flex: none; margin-bottom: 8px; }
.proj-name { font-weight: bold; margin-top: 4px; }
.path { word-break: break-all; }
.pane-options h3, .pane-result h3 { font-size: 13px; color: #8fa3b8; margin: 14px 0 8px; text-transform: none; }
.pane-options h3:first-child { margin-top: 0; }
.mode-card { border: 1px solid #3a4a5c; border-radius: 8px; padding: 10px; margin-bottom: 8px; cursor: pointer; }
.mode-card.active { border-color: #2b6cb0; background: rgba(43, 108, 176, .15); }
.mode-line { font-size: 14px; }
.mode-desc { font-size: 12px; color: #8fa3b8; margin-top: 4px; padding-left: 20px; }
.radio { display: inline-block; width: 12px; height: 12px; border-radius: 50%; border: 2px solid #8fa3b8; margin-right: 4px; vertical-align: -1px; }
.radio.on { border-color: #2b6cb0; background: #2b6cb0; box-shadow: inset 0 0 0 2px #16222e; }
.inline { display: flex; align-items: center; gap: 6px; margin: 6px 0; }
.block { display: block; margin: 6px 0; }
.num { width: 100px; }
.chip { display: inline-block; border-radius: 12px; padding: 3px 10px; font-size: 12px; margin-top: 6px; }
.chip.none { background: #1a3a2a; color: #68d391; }
.chip.hit { background: #3a2a1a; color: #f6ad55; }
.chip.unscanned { background: #1f2d3d; color: #8fa3b8; }
.outline { background: #0f1720; padding: 10px; border-radius: 6px; font-size: 12px; line-height: 1.7; overflow-x: auto; }
.outline.preview-content { white-space: pre-wrap; word-break: break-word; }
.result-card { border-radius: 8px; padding: 14px; }
.result-card.ok { border: 1px solid #2f855a; background: rgba(47, 133, 90, .1); }
.result-card h3 { margin: 0 0 10px; color: #68d391; }
.result-card table { width: 100%; border-collapse: collapse; font-size: 13px; }
.result-card th { text-align: left; color: #8fa3b8; padding: 4px 8px 4px 0; white-space: nowrap; }
.result-card td { word-break: break-all; }
.result-actions { display: flex; gap: 8px; margin-top: 12px; flex-wrap: wrap; }
.statusbar {
  grid-column: 1 / -1;
  height: 44px; flex: none;
  display: flex; align-items: center; justify-content: space-between;
  padding: 0 16px;
  background: #16222e; border: 1px solid #3a4a5c; border-radius: 8px;
}
.status-left { display: flex; gap: 12px; align-items: center; }
.status-right { display: flex; gap: 8px; }
.muted { color: #8fa3b8; font-size: 12px; }
.error { grid-column: 1 / -1; color: #ff8383; background: #3a1a1a; border: 1px solid #742a2a; border-radius: 6px; padding: 8px 12px; }
button { cursor: pointer; border: 1px solid #3a4a5c; background: #1f2d3d; color: #eee; border-radius: 6px; padding: 4px 10px; }
button.primary { background: #2b6cb0; }
button:disabled { opacity: .5; cursor: not-allowed; }
.link { border: none; background: none; color: #63b3ed; padding: 2px 4px; }
input, select { background: #0f1720; color: #eee; border: 1px solid #3a4a5c; border-radius: 4px; padding: 4px 8px; }
.modal { position: fixed; inset: 0; background: rgba(0,0,0,.5); display: flex; align-items: center; justify-content: center; z-index: 30; }
.modal-body { background: #16222e; border: 1px solid #3a4a5c; border-radius: 10px; padding: 20px; width: 420px; }
.actions { display: flex; gap: 8px; margin-top: 12px; }
</style>
