<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { useProjectsStore, useExportStore } from '../stores';
import { api } from '../api/bindings';
import type { FileInfo, ProjectConfig } from '../api/bindings';
import FileTree from './FileTree.vue';
import SensitiveDrawer from './SensitiveDrawer.vue';
import { estimateTokensFromSize, fmtNum } from '../estimate';
import { useLogStore } from '../stores/log';

const props = defineProps<{ configPath: string }>();
const router = useRouter();
const projects = useProjectsStore();
const exp = useExportStore();
const log = useLogStore();

const loadError = ref('');
const scanning = ref(false);
const exporting = ref(false);

const MODES = [
  { value: 'full', icon: '📄', label: '完整代码', desc: '文件树 + 全部文件代码' },
  { value: 'files', icon: '🗂', label: '文件名清单', desc: '仅文件树与路径列表' },
  { value: 'symbols', icon: '🔎', label: '符号目录', desc: '文件名 + 方法/类名（启发式解析）' },
  { value: 'signatures', icon: '✍️', label: '签名导出', desc: '文件名 + 函数完整签名' },
];

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
  log.info(`扫描项目：${projects.current.project_name}`);
  scanning.value = true;
  try {
    await exp.scan(projects.current);
    log.info(`扫描完成：共 ${exp.files.length} 个文件`);
  } finally {
    scanning.value = false;
  }
}

// 导出模式切换
watch(() => exp.mode, (m, old) => {
  if (old) log.info(`切换导出模式：${old} → ${m}`);
});

// —— 勾选统计（左栏底/状态条共用），300ms 防抖避免大项目勾选时频繁重算 ——
import { onUnmounted } from 'vue';

const rawChecked = computed<FileInfo[]>(() => exp.files.filter((f) => exp.checkedPaths.has(f.path)));
const debouncedFiles = ref<FileInfo[]>(rawChecked.value);
let debounceTimer: ReturnType<typeof setTimeout> | undefined;
watch(rawChecked, (v) => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => { debouncedFiles.value = v; }, 300);
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
  log.info(`扫描敏感信息：${exp.checkedPaths.size} 个文件，策略=${redactStrategy.value}`);
  sensitiveScanning.value = true;
  try {
    const hits: SensitiveHit[] = await api.scanSensitive(projects.current, [...exp.checkedPaths]);
    sensitive.value = { state: hits.length ? 'hit' : 'none', count: hits.length, hits };
    if (hits.length) log.warn(`发现 ${hits.length} 处敏感信息`);
    else log.info('未发现敏感信息');
  } catch (e) {
    log.warn(`敏感信息扫描失败：${String(e)}`);
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
  log.info(`触发导出：模式=${exp.mode}，脱敏=${redactEnabled.value ? '开' : '关'}，文件=${checkedFiles.value.length}`);
  exporting.value = true;
  exp.redact = redactEnabled.value;
  exp.splitTokens = splitEnabled.value ? splitTokens.value : 0;
  try {
    await exp.run(projects.current);
    if (exp.result) log.info(`导出成功：${exp.result.outputPaths.join(', ')}（${exp.result.durationMs} ms）`);
  } finally {
    exporting.value = false;
  }
}

// —— 右栏：大纲预览（导出前）/ 结果卡片（导出后）——
const outline = computed(() => buildOutline(projects.current, checkedFiles.value, exp.mode));

function buildOutline(cfg: ProjectConfig | null, files: FileInfo[], mode: string): string[] {
  if (!cfg) return [];
  const lines: string[] = [
    `# 项目文档 for ${cfg.project_name}`,
    '├─ ## 项目文件树',
  ];
  if (mode === 'full') {
    lines.push(`├─ ## 项目文件 (${files.length})`);
    for (const f of files.slice(0, 30)) lines.push(`│   ├─ ### 文件: ${f.path}`);
    if (files.length > 30) lines.push(`│   └─ … 共 ${files.length} 个文件`);
  } else if (mode === 'symbols') {
    lines.push(`├─ ## 符号目录 (${files.length})`);
  } else if (mode === 'signatures') {
    lines.push(`├─ ## 签名导出 (${files.length})`);
  }
  return lines;
}

async function openOutputDir() {
  if (!exp.result) return;
  log.info(`打开输出目录：${exp.result.outputPaths[0]}`);
  await api.openPath(exp.result.outputPaths[0]);
}

async function copyResult() {
  if (!exp.result) return;
  try {
    await navigator.clipboard.writeText(exp.result.outputPaths.join('\n'));
    log.info(`已复制输出路径：${exp.result.outputPaths.join(', ')}`);
  } catch {
    log.warn('复制路径失败：剪贴板权限不可用');
  }
}
</script>

<template>
  <main class="workbench">
    <p v-if="loadError" class="error">{{ loadError }} <button @click="router.push('/')">← 返回项目列表</button></p>

    <template v-else-if="projects.current">
      <!-- 左栏：项目信息 + 文件树 -->
      <aside class="pane pane-tree">
        <div class="proj-head">
          <button class="link" @click="router.push('/')">← 返回</button>
          <div class="proj-name">{{ projects.current.project_name }}</div>
          <div class="muted path">{{ projects.current.project_path }}</div>
        </div>
        <FileTree :files="exp.files" :checked="exp.checkedPaths" @update:checked="(v) => (exp.checkedPaths = v)" />
      </aside>

      <!-- 中栏：导出选项 -->
      <section class="pane pane-options">
        <h3>导出模式</h3>
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
            显示行号
          </label>
          <label class="block sub">
            签名截断长度
            <input v-model.number="exp.maxSignatureLen" type="number" class="num" min="40" max="2000" />
          </label>
        </template>

        <h3>敏感信息过滤</h3>
        <label class="inline">
          <input v-model="redactEnabled" type="checkbox" />
          导出时自动剔除密码/密钥
        </label>
        <template v-if="redactEnabled">
          <label class="block">策略:
            <select v-model="redactStrategy">
              <option value="placeholder">占位符替换</option>
              <option value="drop_line">删除所在行</option>
            </select>
          </label>
          <label class="block">替换为:
            <input v-model="redactPlaceholder" :disabled="redactStrategy === 'drop_line'" />
          </label>
          <button :disabled="sensitiveScanning" @click="scanSensitive">
            {{ sensitiveScanning ? '扫描中…' : '🔍 扫描敏感信息' }}
          </button>
          <div class="chip" :class="sensitive.state">
            <template v-if="sensitive.state === 'none'">✓ 未发现敏感信息</template>
            <template v-else-if="sensitive.state === 'hit'">
              ⚠ 发现 {{ sensitive.count }} 处
              <button class="link" @click="showSensitiveDrawer = true">查看 →</button>
            </template>
            <template v-else>○ 尚未扫描</template>
          </div>
        </template>

        <h3>大小控制</h3>
        <label class="inline">
          <input v-model="splitEnabled" type="checkbox" />
          超过 <input v-model.number="splitTokens" type="number" class="num" :disabled="!splitEnabled" /> token
        </label>
        <p v-if="splitEnabled" class="muted">预计切成 {{ estParts }} 片</p>
      </section>

      <!-- 右栏：预览 / 结果 -->
      <section class="pane pane-result">
        <template v-if="exp.result">
          <div class="result-card ok">
            <h3>✓ 导出成功</h3>
            <table>
              <tr><th>输出文件</th><td>{{ exp.result.outputPaths.join(', ') }}</td></tr>
              <tr><th>字符数</th><td>{{ fmtNum(exp.result.totalChars) }}</td></tr>
              <tr><th>≈Token</th><td>{{ fmtNum(estTokens) }}</td></tr>
              <tr><th>耗时</th><td>{{ exp.result.durationMs }} ms</td></tr>
              <tr><th>脱敏</th><td>{{ redactEnabled ? sensitive.count + ' 处' : '未开启' }}</td></tr>
            </table>
            <div class="result-actions">
              <button @click="openOutputDir">📂 打开目录</button>
              <button @click="copyResult">📋 复制路径</button>
              <button @click="doExport" :disabled="exporting">↻ 重新导出</button>
            </div>
          </div>
        </template>
        <template v-else>
          <h3>内容预览</h3>
          <pre class="outline"><template v-for="l in outline" :key="l">{{ l }}
</template></pre>
          <p class="muted">当前模式: {{ MODES.find((m) => m.value === exp.mode)?.label }} · 敏感过滤{{ redactEnabled ? '已开启' : '未开启' }}</p>
        </template>
      </section>
    </template>

    <!-- 底部状态条 -->
    <footer v-if="projects.current" class="statusbar">
      <span class="status-left">
        <template v-if="exporting">⏳ 正在导出…</template>
        <template v-else>● 就绪</template>
        <span class="muted">{{ checkedFiles.length }} 文件 · {{ fmtSize(checkedBytes) }} · ≈{{ fmtNum(estTokens) }} tokens</span>
      </span>
      <span class="status-right">
        <button :disabled="exporting" @click="doExport">预览</button>
        <button class="primary" :disabled="exporting" @click="onExportClick">执行导出 ▸</button>
      </span>
    </footer>

    <SensitiveDrawer
      v-if="showSensitiveDrawer"
      :hits="sensitive.hits"
      @close="showSensitiveDrawer = false"
    />

    <div v-if="showUnscannedConfirm" class="modal" @click.self="showUnscannedConfirm = false">
      <div class="modal-body">
        <h3>尚未扫描敏感信息</h3>
        <p>已开启敏感过滤但还未扫描，建议先扫描确认命中情况。</p>
        <div class="actions">
          <button class="primary" @click="showUnscannedConfirm = false; scanSensitive()">扫描并继续</button>
          <button @click="showUnscannedConfirm = false; doExport()">直接导出</button>
          <button @click="showUnscannedConfirm = false">取消</button>
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
