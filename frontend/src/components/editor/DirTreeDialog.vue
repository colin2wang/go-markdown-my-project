<!-- DirTreeDialog：应用内目录树多选对话框（Wails 目录对话框不支持多选，故自实现）。
     树根为项目主目录，子目录由 DirTreeNode 懒加载；确认后返回选中目录的绝对路径列表。 -->
<script setup lang="ts">
import { ref, watch } from 'vue';
import { api } from '../../api/bindings';
import { useI18n } from '../../i18n';
import DirTreeNode, { type DirNode } from './DirTreeNode.vue';
import { normalizeDirSelection } from './dirPaths';

const { t } = useI18n();

const props = defineProps<{
  open: boolean;
  root: string;
  /** 打开时预勾选的目录（TagListInput 的当前值，相对 basePath 的路径或绝对路径） */
  initialSelected?: string[];
}>();
const emit = defineEmits<{ (e: 'close'): void; (e: 'pick', paths: string[]): void }>();

const rootNode = ref<DirNode | null>(null);
const selected = ref(new Set<string>());
const rootLoading = ref(false); // 首次构建树根的加载态
const busy = ref(false);        // 任一节点懒加载子目录中（仅禁用勾选，不拆树）

// 把标签值（相对路径或绝对路径）换算成树中的绝对路径用于预勾选。
// 树节点路径统一为正斜杠（listSubdirs 原始返回为反斜杠），这里保持一致才能命中 Set。
function toAbs(rel: string): string {
  let norm = rel.replace(/\\/g, '/').replace(/\/+$/, '');
  if (/^[a-zA-Z]:/.test(norm) || norm.startsWith('/')) return norm;
  norm = norm.replace(/^\.\//, '');
  if (!props.root) return norm;
  const base = props.root.replace(/\\/g, '/').replace(/\/+$/, '');
  // "." 表示项目根，与树根路径一致
  if (norm === '' || norm === '.') return base;
  return `${base}/${norm}`;
}

// 每次打开重建树根、按当前标签回填勾选（去掉被父目录覆盖的子目录），并自动加载根目录的子目录
watch(
  () => props.open,
  (open) => {
    if (!open || !props.root) return;
    const rootPath = props.root.replace(/\\/g, '/').replace(/\/+$/, '');
    const name = rootPath.split('/').pop() ?? rootPath;
    rootNode.value = { name, path: rootPath, loaded: false, expanded: true, children: [] };
    selected.value = new Set(normalizeDirSelection((props.initialSelected ?? []).map(toAbs)));
    void loadRoot();
  },
);

async function loadRoot() {
  rootLoading.value = true;
  const subs = await api.listSubdirs(props.root);
  if (rootNode.value) {
    rootNode.value.children = subs.map((p) => ({
      name: p.replace(/[\\/]+$/, '').split(/[\\/]/).pop() ?? p,
      // 统一正斜杠：与 toAbs 生成的预勾选路径、确认后 addFromTree 的换算一致
      path: p.replace(/\\/g, '/').replace(/\/+$/, ''),
      loaded: false, expanded: false, children: [],
    }));
    rootNode.value.loaded = true;
  }
  rootLoading.value = false;
}

// 返回对话框的完整选择集合（可为空），由 TagListInput 替换写回，取消勾选/清空才会生效
function confirm() {
  emit('pick', normalizeDirSelection([...selected.value]));
  emit('close');
}
</script>

<template>
  <div v-if="props.open && rootNode" class="modal" @click.self="emit('close')">
    <div class="modal-body">
      <h3>{{ t('editor.dirTreeTitle') }}</h3>
      <p class="hint">{{ t('editor.dirTreeHint') }}</p>
      <div class="tree">
        <p v-if="rootLoading" class="loading">{{ t('editor.dirTreeLoading') }}</p>
        <DirTreeNode
          v-else-if="rootNode"
          :node="rootNode"
          :selected="selected"
          :depth="0"
          @update:selected="selected = $event"
          @busy="busy = $event"
        />
      </div>
      <div class="actions">
        <button @click="emit('close')">{{ t('common.cancel') }}</button>
        <button class="primary" :disabled="busy" @click="confirm">{{ t('common.confirm') }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal { position: fixed; inset: 0; background: var(--mask); display: flex; align-items: center; justify-content: center; z-index: 40; }
.modal-body {
  background: var(--bg-2);
  border: 1px solid var(--line-2);
  border-radius: var(--r-lg); padding: 20px; width: 520px; max-width: 90vw; max-height: 75vh;
  display: flex; flex-direction: column;
  box-shadow: var(--shadow-modal);
}
.modal-body h3 { margin: 0 0 6px; color: var(--fg-1); }
.hint { margin: 0 0 10px; font-size: 11px; color: var(--fg-3); }
.loading { margin: 0; padding: 8px; font-size: 12px; color: var(--fg-3); }
.tree { flex: 1; min-height: 200px; max-height: 50vh; overflow: auto; border: 1px solid var(--line-2); border-radius: var(--r-md); padding: 6px; }
.actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 12px; }
.actions button { cursor: pointer; border: 1px solid var(--line-2); background: transparent; color: var(--fg-1); border-radius: var(--r-sm); padding: 4px 14px; }
.actions button.primary { background: var(--accent); border-color: var(--accent); color: var(--fg-on-accent); font-weight: 600; }
.actions button:disabled { opacity: .5; cursor: default; }
</style>
