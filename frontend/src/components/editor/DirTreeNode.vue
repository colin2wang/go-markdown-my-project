<!-- DirTreeNode：目录树递归节点，子目录懒加载（api.listSubdirs）。
     选中集合只保留最顶层目录：勾选父目录即覆盖整棵子树，子目录显示勾选但禁用。
     v-model:selected 由父级维护全局勾选集合（路径 Set）。 -->
<script setup lang="ts">
import { ref, computed } from 'vue';
import { api } from '../../api/bindings';
import { useI18n } from '../../i18n';
import Icon from '../ui/Icon.vue';
import { isCovered, isDescendant, isSameDirPath } from './dirPaths';

const { t } = useI18n();

const props = defineProps<{
  node: DirNode;
  selected: Set<string>;
  depth: number;
  disabled?: boolean;   // true 时禁止勾选（如目录树加载中）
}>();

const emit = defineEmits<{
  (e: 'update:selected', v: Set<string>): void;
  (e: 'busy', v: boolean): void;   // 本节点懒加载子目录时上报，供顶层禁用选择
}>();

const loading = ref(false);

async function toggleExpand() {
  props.node.expanded = !props.node.expanded;
  if (props.node.expanded && !props.node.loaded) {
    loading.value = true;
    emit('busy', true);
    try {
      const subs = await api.listSubdirs(props.node.path);
      props.node.children = subs.map((p) => ({
        name: p.replace(/[\\/]+$/, '').split(/[\\/]/).pop() ?? p,
        // 统一正斜杠：与对话框预勾选路径保持一致，避免 Set 匹配不上
        path: p.replace(/\\/g, '/').replace(/\/+$/, ''),
        loaded: false, expanded: false, children: [],
      }));
      props.node.loaded = true;
    } finally {
      loading.value = false;
      emit('busy', false);
    }
  }
}

// 祖先已勾选时本节点被整体覆盖：显示勾选但不可再单独勾选
const covered = computed(() => [...props.selected].some((s) => isCovered(props.node.path, s)));
const selectedSelf = computed(() => [...props.selected].some((s) => isSameDirPath(s, props.node.path)));
const hasSelectedDescendant = computed(() => [...props.selected].some((s) => isDescendant(s, props.node.path)));
// 勾选 / 半选（仅后代被勾选）/ 未勾选
const state = computed<boolean | 'indeterminate'>(() => {
  if (covered.value || selectedSelf.value) return true;
  return hasSelectedDescendant.value ? 'indeterminate' : false;
});

// 勾选父目录即覆盖整棵子树：清掉祖先、同目录与后代，只保留当前目录；
// 取消时连同其后代一起移除，保证 selected 中不存在父子冗余。
function toggleCheck(on: boolean) {
  const next = new Set(props.selected);
  const p = props.node.path;
  for (const s of [...next]) {
    const same = isSameDirPath(s, p);
    if (on ? same || isCovered(p, s) || isDescendant(s, p) : same || isDescendant(s, p)) {
      next.delete(s);
    }
  }
  if (on) next.add(p);
  emit('update:selected', next);
}
</script>

<template>
  <div class="node">
    <div class="row" :style="{ paddingLeft: depth * 16 + 2 + 'px' }">
      <button type="button" class="arrow" :class="{ open: node.expanded }" :disabled="loading" @click="toggleExpand">
        <Icon name="chevron" :size="12" />
      </button>
      <input
        type="checkbox"
        :checked="state === true"
        :indeterminate.prop="state === 'indeterminate'"
        :disabled="disabled || loading || covered"
        :title="covered ? t('editor.dirTreeCovered') : undefined"
        @change="toggleCheck(($event.target as HTMLInputElement).checked)"
      />
      <span class="name">{{ node.name }}</span>
    </div>
    <!-- 展开但子目录尚未加载完成时，在原位显示加载提示，不卸载整棵树 -->
    <div v-if="node.expanded && loading" class="child-loading" :style="{ paddingLeft: (depth + 1) * 16 + 2 + 'px' }">
      {{ t('editor.dirTreeLoading') }}
    </div>
    <div v-if="node.expanded && node.loaded" class="children">
      <DirTreeNode
        v-for="child in node.children"
        :key="child.path"
        :node="child"
        :selected="selected"
        :depth="depth + 1"
        :disabled="disabled || loading"
        @update:selected="emit('update:selected', $event)"
        @busy="emit('busy', $event)"
      />
    </div>
  </div>
</template>

<script lang="ts">
// DirNode 放在普通 script 块导出，供 DirTreeDialog 等外部导入类型
export interface DirNode {
  name: string;      // 显示名（最后一级目录名）
  path: string;      // 绝对路径
  loaded: boolean;   // 子节点是否已加载
  expanded: boolean;
  children: DirNode[];
}

// 递归自引用：显式声明组件名（SFC 文件名自引用亦可解析）
import { defineComponent } from 'vue';
export default defineComponent({ name: 'DirTreeNode' });
</script>

<style scoped>
.row { display: flex; align-items: center; gap: 4px; padding: 2px 0; }
.arrow { background: none; border: none; color: var(--fg-3); cursor: pointer; display: grid; place-items: center; width: 16px; height: 16px; padding: 0; transition: transform .12s; transform: rotate(-90deg); } /* 收起时朝右 */
.arrow.open { transform: none; }
.arrow:disabled { opacity: .4; cursor: default; }
.name { font-size: 12.5px; color: var(--fg-1); font-family: ui-monospace, Consolas, monospace; user-select: none; }
.child-loading { font-size: 12px; color: var(--fg-3); padding: 2px 0; }
</style>
