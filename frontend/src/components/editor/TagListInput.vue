<!-- TagListInput：标签增删 / 回车失焦添加 / 粘贴批量拆分 / 可选磁盘选择（设计文档 §2.2） -->
<script setup lang="ts">
import { ref } from 'vue';
import { api } from '../../api/bindings';
import { useLogStore } from '../../stores/log';
import { useI18n } from '../../i18n';
import Icon from '../ui/Icon.vue';
import DirTreeDialog from './DirTreeDialog.vue';
import { normalizeDirSelection } from './dirPaths';

const { t } = useI18n();
const props = defineProps<{
  modelValue: string[];
  placeholder?: string;
  error?: string;
  dirSelect?: boolean;   // 显示「从磁盘选…」「添加所有子文件夹」按钮（目录）
  treeSelect?: boolean;  // 显示「选择子目录…」按钮：应用内目录树多选对话框（目录）
  dirHierarchy?: boolean; // 目录列表按父子层级去重（父目录已选中时子目录多余）
  fileSelect?: boolean;  // 显示「从磁盘选文件…」按钮（文件），对话框默认打开 basePath
  extScan?: boolean;     // 显示「添加所有扩展名」按钮（包含模式）
  basePath?: string;     // 项目根目录：对话框起始目录、扩展名扫描根、结果转相对路径
  scanDirs?: string[];   // 扩展名扫描的包含目录（相对 basePath）
  disabled?: boolean;
}>();
const emit = defineEmits<{ (e: 'update:modelValue', v: string[]): void }>();

const log = useLogStore();
const input = ref('');
const treeOpen = ref(false);

function add(raw: string) {
  const items = raw.split(/[,\n]/).map((s) => s.trim().replace(/\\/g, '/')).filter(Boolean);
  // modelValue 可能是 undefined（YAML 中省略的数组字段），兜底为空数组
  let next = [...(props.modelValue ?? [])];
  let dup = 0;
  for (const it of items) {
    if (next.includes(it)) { dup++; continue; }
    next.push(it);
  }
  // 目录列表：父目录已选中时子目录多余；后来加入父目录则清掉已有子目录
  if (props.dirHierarchy) {
    const normalized = normalizeDirSelection(next);
    dup += next.length - normalized.length;
    next = normalized;
  }
  if (dup) log.warn(t('editor.tagDupIgnored', { n: dup }));
  emit('update:modelValue', next);
  input.value = '';
}

function remove(idx: number) {
  const next = [...(props.modelValue ?? [])];
  next.splice(idx, 1);
  emit('update:modelValue', next);
}

// 清空全部标签
function clearAll() {
  emit('update:modelValue', []);
  input.value = '';
}

// 系统文件多选对话框：默认打开项目根目录，转相对路径后去重加入；取消时不动。
async function browseFiles() {
  const files = await api.selectFiles(props.basePath ?? '');
  if (!files.length) return;
  add(files.map((f) => toRelative(f.replace(/\\/g, '/'))).join('\n'));
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') { e.preventDefault(); add(input.value); }
}

function onPaste(e: ClipboardEvent) {
  const text = e.clipboardData?.getData('text') ?? '';
  if (/[,\n]/.test(text)) {
    e.preventDefault();
    add(text);
  }
}

// 若所选目录位于 basePath 之下，转为相对路径；否则保留原样
function toRelative(dir: string): string {
  if (!props.basePath) return dir;
  const norm = (s: string) => s.replace(/\\/g, '/').replace(/\/+$/, '');
  const base = norm(props.basePath);
  const d = norm(dir);
  if (d.toLowerCase() === base.toLowerCase()) return '.';
  if (d.toLowerCase().startsWith(base.toLowerCase() + '/')) {
    return d.slice(base.length + 1);
  }
  return dir;
}

// 目录树对话框返回的是完整选择集合（可为空）：转相对路径并整体替换，
// 这样取消勾选 / 清空也会生效，且不会与旧标签叠加出父子冗余。
function addFromTree(paths: string[]) {
  const rel = paths.map((p) => toRelative(p.replace(/\\/g, '/')));
  emit('update:modelValue', normalizeDirSelection(rel));
}

// 系统目录选择对话框：每次选一个目录，转相对路径后去重加入；
// 取消时不动。需要多个目录可重复点击或用「添加所有子文件夹」。
async function browse() {
  const dir = await api.selectDirectory(props.basePath ?? '');
  if (!dir) return;
  add(toRelative(dir.replace(/\\/g, '/')));
}

// 一键加入项目根目录下全部直接子目录
async function addAllSubdirs() {
  if (!props.basePath) return;
  const subs = await api.listSubdirs(props.basePath);
  if (!subs.length) {
    log.warn(t('editor.noSubdirs', { dir: props.basePath }));
    return;
  }
  add(subs.map((s) => toRelative(s)).join('\n'));
}

// 扫描项目（或包含目录）下所有文件扩展名，以 *.ext 形式加入包含文件列表
async function addAllExtensions() {
  if (!props.basePath) return;
  const exts = await api.listFileExtensions(props.basePath, props.scanDirs ?? []);
  if (!exts.length) {
    log.warn(t('editor.noExtensions'));
    return;
  }
  add(exts.map((e) => '*' + e).join('\n'));
}
</script>

<template>
  <div class="taglist" :class="{ disabled }">
    <div class="tags" :class="{ invalid: !!error }">
      <span v-for="(t, i) in modelValue" :key="t" class="tag">
        <span class="chip-text">{{ t }}</span>
        <button type="button" class="x" :disabled="disabled" @click="remove(i)"><Icon name="close" :size="10" /></button>
      </span>
      <input
        v-model="input"
        :placeholder="placeholder"
        :disabled="disabled"
        :class="{ invalid: !!error }"
        @keydown="onKeydown"
        @paste="onPaste"
        @blur="input.trim() && add(input)"
      />
    </div>
    <div class="browse-row">
      <button v-if="treeSelect" type="button" class="browse" :disabled="disabled || !basePath" @click="treeOpen = true"><Icon name="folder" /> {{ t('editor.treeSelect') }}</button>
      <button v-if="fileSelect" type="button" class="browse" :disabled="disabled" @click="browseFiles"><Icon name="doc" /> {{ t('editor.browseDiskFiles') }}</button>
      <button v-if="extScan" type="button" class="browse" :disabled="disabled || !basePath" @click="addAllExtensions"><Icon name="doc" /> {{ t('editor.addAllExts') }}</button>
      <button v-if="dirSelect" type="button" class="browse" :disabled="disabled" @click="browse"><Icon name="folder" /> {{ t('editor.browseDisk') }}</button>
      <button v-if="dirSelect" type="button" class="browse" :disabled="disabled || !basePath" @click="addAllSubdirs"><Icon name="folder" /> {{ t('editor.allSubdirs') }}</button>
      <button type="button" class="browse clear" :disabled="disabled || !(modelValue ?? []).length" @click="clearAll"><Icon name="close" :size="10" /> {{ t('editor.clearAll') }}</button>
    </div>
    <DirTreeDialog :open="treeOpen" :root="basePath ?? ''" :initial-selected="modelValue ?? []" @close="treeOpen = false" @pick="addFromTree" />
    <p v-if="error" class="field-error">{{ error }}</p>
  </div>
</template>

<style scoped>
.tags { display: flex; flex-wrap: wrap; gap: 5px; align-items: center; background: var(--bg-3); border: 1px solid var(--line-2); border-radius: var(--r-md); padding: 5px 8px; min-height: 34px; box-sizing: border-box; cursor: text; transition: border-color .12s; }
.tags:focus-within { border-color: var(--accent-strong); }
.tag {
  display: inline-flex; align-items: center; gap: 5px;
  max-width: 220px;
  background: var(--bg-2); border: 1px solid var(--line-2); border-radius: var(--r-sm);
  padding: 2px 4px 2px 8px;
  font-family: ui-monospace, Consolas, monospace;
  font-size: 12px; color: var(--fg-1);
}
.tag .chip-text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tag .x {
  flex: none; width: 15px; height: 15px;
  display: grid; place-items: center;
  background: none; border: none; border-radius: var(--r-sm);
  color: var(--fg-3); font-size: 11px; line-height: 1; cursor: pointer; padding: 0;
}
.tag .x:hover { color: var(--err); background: var(--bg-2); }
/* 校验失败时整组 tag 容器红框 */
.tags.invalid { border-color: var(--err); }
.tags input { flex: 1; min-width: 120px; background: none; border: none; color: var(--fg-1); outline: none; font-size: 12.5px; padding: 2px; }
.browse { margin-top: 4px; background: transparent; color: var(--fg-1); border: 1px solid var(--line-2); border-radius: var(--r-sm); padding: 3px 8px; font-size: 12px; cursor: pointer; white-space: nowrap; }
.browse:disabled { opacity: .4; cursor: not-allowed; }
.browse-row { display: flex; align-items: center; gap: 10px; }
.browse.clear { margin-left: auto; }
.disabled { opacity: .5; }
input.invalid { border-color: var(--err); }
.field-error { color: var(--err); font-size: 11px; margin: 3px 0 0 2px; }
</style>
