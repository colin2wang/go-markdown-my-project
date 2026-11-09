<!-- DirTreeDialog：应用内目录树多选对话框（Wails 目录对话框不支持多选，故自实现）。
     树根为项目主目录，子目录由 DirTreeNode 懒加载；确认后返回选中目录的绝对路径列表。 -->
<script setup lang="ts">
import { ref, watch } from 'vue';
import { api } from '../../api/bindings';
import { useI18n } from '../../i18n';
import DirTreeNode, { type DirNode } from './DirTreeNode.vue';

const { t } = useI18n();

const props = defineProps<{ open: boolean; root: string }>();
const emit = defineEmits<{ (e: 'close'): void; (e: 'pick', paths: string[]): void }>();

const rootNode = ref<DirNode | null>(null);
const selected = ref(new Set<string>());
const loading = ref(false);

// 每次打开重建树根并自动加载根目录的子目录；加载完成前禁止选择
watch(
  () => props.open,
  (open) => {
    if (!open || !props.root) return;
    const name = props.root.replace(/[\\/]+$/, '').split(/[\\/]/).pop() ?? props.root;
    rootNode.value = { name, path: props.root, loaded: false, expanded: true, children: [] };
    selected.value = new Set();
    void loadRoot();
  },
);

async function loadRoot() {
  loading.value = true;
  const subs = await api.listSubdirs(props.root);
  if (rootNode.value) {
    rootNode.value.children = subs.map((p) => ({
      name: p.replace(/[\\/]+$/, '').split(/[\\/]/).pop() ?? p,
      path: p, loaded: false, expanded: false, children: [],
    }));
    rootNode.value.loaded = true;
  }
  loading.value = false;
}

function confirm() {
  const paths = [...selected.value];
  if (paths.length) emit('pick', paths);
  emit('close');
}
</script>

<template>
  <div v-if="props.open && rootNode" class="modal" @click.self="emit('close')">
    <div class="modal-body">
      <h3>{{ t('editor.dirTreeTitle') }}</h3>
      <p class="hint">{{ t('editor.dirTreeHint') }}</p>
      <div class="tree">
        <p v-if="loading" class="loading">{{ t('editor.dirTreeLoading') }}</p>
        <DirTreeNode
          v-else
          :node="rootNode"
          :selected="selected"
          :depth="0"
          @update:selected="selected = $event"
          @busy="loading = $event"
        />
      </div>
      <div class="actions">
        <button @click="emit('close')">{{ t('common.cancel') }}</button>
        <button class="primary" :disabled="loading || !selected.size" @click="confirm">{{ t('common.confirm') }}</button>
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
