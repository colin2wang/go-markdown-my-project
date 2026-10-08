<!-- TagListInput：标签增删 / 回车失焦添加 / 粘贴批量拆分 / 可选磁盘选择（设计文档 §2.2） -->
<script setup lang="ts">
import { ref } from 'vue';
import { api } from '../../api/bindings';
import { useLogStore } from '../../stores/log';
import { useI18n } from '../../i18n';

const { t } = useI18n();
const props = defineProps<{
  modelValue: string[];
  placeholder?: string;
  error?: string;
  dirSelect?: boolean; // 显示「从磁盘选…」按钮（目录）
  basePath?: string;   // 项目根目录：作为选择框起始目录，并把结果转为相对路径
  disabled?: boolean;
}>();
const emit = defineEmits<{ (e: 'update:modelValue', v: string[]): void }>();

const log = useLogStore();
const input = ref('');
const multi = ref(false); // 勾选后「从磁盘选…」进入连续添加模式
const allSubdirs = ref(false); // 勾选后「从磁盘选…」一次性加入所选目录的全部直接子目录

function add(raw: string) {
  const items = raw.split(/[,\n]/).map((s) => s.trim().replace(/\\/g, '/')).filter(Boolean);
  // modelValue 可能是 undefined（YAML 中省略的数组字段），兜底为空数组
  const next = [...(props.modelValue ?? [])];
  let dup = 0;
  for (const it of items) {
    if (next.includes(it)) { dup++; continue; }
    next.push(it);
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

// 多选框勾选时进入连续选择模式：每选中一个目录立即加入列表并继续弹窗，
// 取消（返回空）时结束；未勾选则单选一次。
// 「添加所有子文件夹」勾选时，选中目录后改为一次性加入其全部直接子目录。
async function browse() {
  let start = props.basePath ?? '';
  let count = 0;
  do {
    const dir = await api.selectDirectory(start);
    if (!dir) break;
    if (allSubdirs.value) {
      const subs = await api.listSubdirs(dir);
      if (subs.length) add(subs.map((s) => toRelative(s)).join('\n'));
      else log.warn(t('editor.noSubdirs', { dir }));
      count += subs.length;
    } else {
      add(toRelative(dir));
      count++;
    }
    start = dir; // 下一次从刚选的目录打开，便于选同级/子目录
  } while (multi.value);
  if (count > 1) log.info(t('editor.dirsPicked', { n: count }));
}
</script>

<template>
  <div class="taglist" :class="{ disabled }">
    <div class="tags">
      <span v-for="(t, i) in modelValue" :key="t" class="tag">
        {{ t }}
        <button type="button" class="x" :disabled="disabled" @click="remove(i)">×</button>
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
    <div v-if="dirSelect" class="browse-row">
      <button type="button" class="browse" :disabled="disabled" @click="browse">📂 {{ t('editor.browseDisk') }}</button>
      <label class="multi"><input v-model="multi" type="checkbox" :disabled="disabled" />{{ t('editor.multiPick') }}</label>
      <label class="multi"><input v-model="allSubdirs" type="checkbox" :disabled="disabled" />{{ t('editor.allSubdirs') }}</label>
    </div>
    <p v-if="error" class="field-error">{{ error }}</p>
  </div>
</template>

<style scoped>
.tags { display: flex; flex-wrap: wrap; gap: 4px; align-items: center; background: #0f1720; border: 1px solid #3a4a5c; border-radius: 4px; padding: 4px 6px; }
.tag { display: inline-flex; align-items: center; gap: 2px; background: #1f2d3d; border: 1px solid #3a4a5c; border-radius: 4px; padding: 1px 4px 1px 8px; font-size: 12px; color: #eee; }
.tag .x { background: none; border: none; color: #8fa3b8; cursor: pointer; padding: 0 3px; font-size: 13px; }
.tag .x:hover { color: #ff8383; }
.tags input { flex: 1; min-width: 90px; background: none; border: none; color: #eee; outline: none; font-size: 12px; padding: 2px; }
.tags input.invalid { border: 1px solid #c53030; border-radius: 3px; }
.browse { margin-top: 4px; background: #1f2d3d; color: #eee; border: 1px solid #3a4a5c; border-radius: 4px; padding: 3px 8px; font-size: 12px; cursor: pointer; }
.browse:disabled { opacity: .4; cursor: not-allowed; }
.browse-row { display: flex; align-items: center; gap: 10px; }
.multi { display: inline-flex; align-items: center; gap: 4px; font-size: 12px; color: #8fa3b8; cursor: pointer; user-select: none; }
.multi input { cursor: pointer; }
.disabled { opacity: .5; }
input.invalid { border-color: #c53030; }
.field-error { color: #ff8383; font-size: 11px; margin: 3px 0 0 2px; }
</style>
