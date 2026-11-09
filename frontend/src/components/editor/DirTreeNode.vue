<!-- DirTreeNode：目录树递归节点，子目录懒加载（api.listSubdirs），勾选聚合选中/半选状态。
     v-model:selected 由父级维护全局勾选集合（路径 Set）。 -->
<script setup lang="ts">
import { ref, computed } from 'vue';
import { api } from '../../api/bindings';
import Icon from '../ui/Icon.vue';

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
        path: p, loaded: false, expanded: false, children: [],
      }));
      props.node.loaded = true;
    } finally {
      loading.value = false;
      emit('busy', false);
    }
  }
}

// 聚合子树勾选状态：全选 / 全不选 / 半选
const state = computed<boolean | 'indeterminate'>(() => {
  const walk = (n: DirNode): { on: number; off: number } => {
    let r = props.selected.has(n.path) ? { on: 1, off: 0 } : { on: 0, off: 1 };
    if (n.loaded) for (const c of n.children) {
      // 子组件与父组件共享同一 selected Set，这里直接递归原始数据即可
      const cr = walk(c);
      r.on += cr.on; r.off += cr.off;
    }
    return r;
  };
  const r = walk(props.node);
  if (r.off === 0) return true;
  if (r.on === 0) return false;
  return 'indeterminate';
});

// 勾选/取消 = 勾选/取消整棵子树（未加载的子树按路径前缀同步，勾选后标记 loaded:false
// 的子树也会在展开时继承父级状态——这里直接只对已加载部分操作 + 当前节点路径）
function toggleCheck(on: boolean) {
  const next = new Set(props.selected);
  const walk = (n: DirNode) => {
    if (on) next.add(n.path); else next.delete(n.path);
    if (n.loaded) n.children.forEach(walk);
  };
  walk(props.node);
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
        :disabled="disabled || loading"
        @change="toggleCheck(($event.target as HTMLInputElement).checked)"
      />
      <span class="name">{{ node.name }}</span>
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
</style>
