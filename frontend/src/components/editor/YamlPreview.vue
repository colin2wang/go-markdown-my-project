<!-- YamlPreview：只读实时镜像，行号 + 变更行高亮 + 状态芯片 + 复制/换行（设计文档 §3.1/§3.3） -->
<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import { useLogStore } from '../../stores/log';
import { useI18n } from '../../i18n';

const { t } = useI18n();
const props = defineProps<{
  yaml: string;
  changed: Set<number>;
  errorCount: number;
  syncing: boolean;
  /** 当前聚焦的表单字段（YAML 键名）；空串 = 无聚焦 */
  focusKey?: string;
}>();

const log = useLogStore();
const wrap = ref(false);
const codeEl = ref<HTMLElement | null>(null);

const lines = computed(() => props.yaml.split('\n'));

// 聚焦字段对应的 YAML 行：匹配「key:」顶层键（首次出现）
const focusLine = computed(() => {
  if (!props.focusKey) return -1;
  const esc = props.focusKey.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const re = new RegExp('^\\s*-?\\s*' + esc + ':');
  return lines.value.findIndex((l) => re.test(l));
});
// 聚焦变化时滚动跟随，让编辑中的字段在 YAML 里始终可见
watch(focusLine, async (ln) => {
  if (ln < 0) return;
  await nextTick();
  codeEl.value?.querySelector(`[data-idx="${ln}"]`)?.scrollIntoView({ block: 'nearest' });
});

// 最简 YAML 语法高亮：注释/行号弱化、键加粗、值与字符串用降饱和状态色
// 不引 CodeMirror，够用即可；特殊行（多行块、列表项内的复杂结构）按普通文本渲染
interface Seg { cls: string; text: string }
function highlight(line: string): Seg[] {
  const trimmed = line.trimStart();
  if (trimmed.startsWith('#')) return [{ cls: 'tok-comment', text: line }];
  const m = line.match(/^(\s*-?\s*[^:#]+?:)(.*)$/);
  if (m) {
    const segs: Seg[] = [{ cls: 'tok-key', text: m[1] }];
    const rest = m[2];
    if (/^\s*["'].*["']\s*$/.test(rest)) segs.push({ cls: 'tok-string', text: rest });
    else if (rest.trim()) segs.push({ cls: 'tok-value', text: rest });
    return segs;
  }
  return [{ cls: '', text: line }];
}
const hlLines = computed(() => lines.value.map(highlight));

async function copyAll() {
  try {
    await navigator.clipboard.writeText(props.yaml);
    log.info(t('editor.yamlCopyOk', { n: lines.value.length }));
  } catch {
    log.warn(t('editor.yamlCopyFail'));
  }
}
</script>

<template>
  <div class="yamlpreview">
    <div class="toolbar">
      <span class="chip" :class="errorCount ? 'red' : 'green'">
        {{ syncing ? t('editor.yamlSyncing') : errorCount ? t('editor.yamlErrors', { n: errorCount }) : t('editor.yamlOk') }}
      </span>
      <span class="muted">{{ t('editor.yamlLines', { n: lines.length }) }}</span>
      <span class="spacer" />
      <button type="button" @click="copyAll">{{ t('editor.yamlCopy') }}</button>
      <label class="wrap-toggle"><input v-model="wrap" type="checkbox" /> {{ t('editor.yamlWrap') }}</label>
    </div>
    <div ref="codeEl" class="code" :class="{ wrap }">
      <div v-for="(l, i) in hlLines" :key="i" class="line" :class="{ changed: changed.has(i + 1), focused: i === focusLine }">
        <span class="ln">{{ i + 1 }}</span>
        <span class="txt"><span v-for="(s, j) in l" :key="j" :class="s.cls">{{ s.text }}</span></span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.yamlpreview { display: flex; flex-direction: column; height: 100%; min-height: 0; }
.toolbar { flex: none; display: flex; align-items: center; gap: 8px; padding: 0 0 8px; font-size: 12px; }
.chip { border-radius: var(--r-sm); border: 1px solid transparent; padding: 2px 10px; font-size: 11px; }
.chip.green { background: transparent; color: var(--ok); border-color: var(--ok); }
.chip.red { background: transparent; color: var(--err); border-color: var(--err); }
.muted { color: var(--fg-3); }
.spacer { flex: 1; }
button { background: var(--bg-3); color: var(--fg-1); border: 1px solid var(--line-2); border-radius: var(--r-sm); padding: 2px 10px; font-size: 12px; cursor: pointer; }
.wrap-toggle { display: flex; align-items: center; gap: 4px; color: var(--fg-2); font-size: 12px; }
.code {
  flex: 1; min-height: 0; overflow: auto;
  background: var(--bg-0); border: 1px solid var(--line); border-radius: var(--r-md);
  font-family: ui-monospace, Menlo, Consolas, monospace;
  font-size: 13px; line-height: 1.55;
  padding: 6px 0;
  text-align: left;
}
.line { display: flex; white-space: pre; }
.code.wrap .line { white-space: pre-wrap; word-break: break-all; }
.ln { flex: none; width: 38px; text-align: right; padding-right: 10px; color: var(--fg-3); user-select: none; }
.txt { color: var(--fg-1); }
.tok-comment { color: var(--fg-3); font-style: italic; }
.tok-key { color: var(--fg-1); font-weight: 600; }
.tok-value { color: var(--ok); }
.tok-string { color: var(--warn); }
.line.changed { background: var(--accent-dim); }
/* 聚焦联动行：左侧强调条 + 微高亮（区别于变更行的纯背景） */
.line.focused { background: var(--accent-dim); box-shadow: inset 2px 0 0 var(--accent); }
</style>
