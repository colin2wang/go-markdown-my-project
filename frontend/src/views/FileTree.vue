<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { FileInfo } from '../api/bindings';
import { useLogStore } from '../stores/log';
import { useI18n } from '../i18n';
import Icon from '../components/ui/Icon.vue';

const { t } = useI18n();
const props = defineProps<{
  files: FileInfo[];
  checked: Set<string>;
}>();

const emit = defineEmits<{
  (e: 'update:checked', v: Set<string>): void;
}>();

// —— 树构造：按相对路径 `/` 分段构层级 ——
interface TreeNode {
  name: string;
  path: string;        // 相对路径（目录不带尾 /）
  dir: boolean;
  sizeBytes: number;
  language: string;
  children: TreeNode[];
}

const tree = computed<TreeNode[]>(() => buildTree(props.files));

function buildTree(files: FileInfo[]): TreeNode[] {
  const root: TreeNode = { name: '', path: '', dir: true, sizeBytes: 0, language: '', children: [] };
  for (const f of files) {
    const segs = f.path.split('/');
    let cur = root;
    for (let i = 0; i < segs.length - 1; i++) {
      const p = segs.slice(0, i + 1).join('/');
      let next = cur.children.find((c) => c.dir && c.path === p);
      if (!next) {
        next = { name: segs[i], path: p, dir: true, sizeBytes: 0, language: '', children: [] };
        cur.children.push(next);
      }
      cur = next;
    }
    cur.children.push({
      name: segs[segs.length - 1], path: f.path, dir: false,
      sizeBytes: f.sizeBytes, language: f.language, children: [],
    });
  }
  // 目录在前、文件在后，各自按名称排序；目录行聚合子孙文件大小
  const sortRec = (n: TreeNode): number => {
    n.children.sort((a, b) => (a.dir === b.dir ? a.name.localeCompare(b.name) : a.dir ? -1 : 1));
    let total = n.dir ? 0 : n.sizeBytes;
    n.children.forEach((c) => { total += sortRec(c); });
    if (n.dir) n.sizeBytes = total;
    return total;
  };
  sortRec(root);
  return root.children;
}

// —— 折叠状态 ——
const collapsed = ref<Set<string>>(new Set());
function toggleDir(path: string) {
  if (collapsed.value.has(path)) collapsed.value.delete(path);
  else collapsed.value.add(path);
  collapsed.value = new Set(collapsed.value);
}

// —— 勾选级联 ——
function childrenOf(node: TreeNode): string[] {
  const out: string[] = [];
  const walk = (n: TreeNode) => {
    if (!n.dir) out.push(n.path);
    n.children.forEach(walk);
  };
  walk(node);
  return out;
}

function nodeState(node: TreeNode): 'all' | 'some' | 'none' {
  const leaves = childrenOf(node);
  const hit = leaves.filter((p) => props.checked.has(p)).length;
  return hit === 0 ? 'none' : hit === leaves.length ? 'all' : 'some';
}

function setCheckedDeep(paths: string[], on: boolean) {
  const next = new Set(props.checked);
  for (const p of paths) {
    if (on) next.add(p); else next.delete(p);
  }
  emit('update:checked', next);
}

function toggleNode(node: TreeNode) {
  const state = nodeState(node);
  setCheckedDeep(childrenOf(node), state !== 'all'); // 全选中则取消，否则全选
}

function toggleFile(path: string) {
  const next = new Set(props.checked);
  if (next.has(path)) next.delete(path); else next.add(path);
  emit('update:checked', next);
}

// —— 搜索过滤（命中文件及其祖先目录保留并自动展开）——
const search = ref('');
const matchCount = computed(() =>
  search.value ? props.files.filter((f) => f.path.toLowerCase().includes(search.value.toLowerCase())).length : 0
);

const visibleTree = computed<TreeNode[]>(() => {
  if (!search.value) return tree.value;
  const q = search.value.toLowerCase();
  const filterRec = (nodes: TreeNode[]): TreeNode[] => {
    const out: TreeNode[] = [];
    for (const n of nodes) {
      if (n.dir) {
        const kids = filterRec(n.children);
        if (kids.length) out.push({ ...n, children: kids });
      } else if (n.path.toLowerCase().includes(q)) {
        out.push(n);
      }
    }
    return out;
  };
  return filterRec(tree.value);
});

watch(search, () => { collapsed.value = new Set(); }); // 搜索时全部展开

// —— 折叠全部 / 展开全部：深目录先收起再逐级展开 ——
const dirPaths = computed(() => {
  const out: string[] = [];
  const walk = (nodes: TreeNode[]) => {
    for (const n of nodes) {
      if (!n.dir) continue;
      out.push(n.path);
      walk(n.children);
    }
  };
  walk(tree.value);
  return out;
});
const allCollapsed = computed(() =>
  dirPaths.value.length > 0 && dirPaths.value.every((p) => collapsed.value.has(p))
);
function toggleCollapseAll() {
  collapsed.value = allCollapsed.value ? new Set() : new Set(dirPaths.value);
}

// —— 快捷操作 ——
const log = useLogStore();

function selectAll() {
  setCheckedDeep(props.files.map((f) => f.path), true);
  log.info(t('tree.logSelectAll', { n: props.files.length }));
}
function selectNone() {
  setCheckedDeep(props.files.map((f) => f.path), false);
  log.info(t('tree.logSelectNone'));
}
function selectInvert() {
  const next = new Set<string>();
  for (const f of props.files) if (!props.checked.has(f.path)) next.add(f.path);
  emit('update:checked', next);
  log.info(t('tree.logInvert', { n: next.size, total: props.files.length }));
}
function selectCodeOnly() {
  const next = new Set(props.files.filter((f) => f.language !== 'Text').map((f) => f.path));
  emit('update:checked', next);
  log.info(t('tree.logCodeOnly', { n: next.size, total: props.files.length }));
}

// —— 底部统计 ——
const stats = computed(() => {
  const hit = props.files.filter((f) => props.checked.has(f.path));
  const bytes = hit.reduce((s, f) => s + f.sizeBytes, 0);
  return { count: hit.length, total: props.files.length, size: formatSize(bytes) };
});

function formatSize(n: number): string {
  if (n < 1024) return n + ' B';
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB';
  return (n / 1024 / 1024).toFixed(1) + ' MB';
}
</script>

<template>
  <div class="filetree">
    <div class="head">
      <div class="title">{{ t('tree.title') }} <span class="muted">({{ stats.count }}/{{ stats.total }})</span></div>
      <input v-model="search" class="search" :placeholder="t('tree.search')" />
      <div class="quick">
        <button @click="selectAll">{{ t('tree.selectAll') }}</button>
        <button @click="selectInvert">{{ t('tree.selectInvert') }}</button>
        <button @click="selectCodeOnly">{{ t('tree.codeOnly') }}</button>
        <button @click="selectNone">{{ t('tree.selectNone') }}</button>
        <button @click="toggleCollapseAll">{{ allCollapsed ? t('tree.expandAll') : t('tree.collapseAll') }}</button>
      </div>
    </div>

    <div class="treewrap">
      <p v-if="search" class="muted hint">{{ t('tree.matched', { n: matchCount }) }}</p>
      <p v-if="files.length === 0" class="muted hint">{{ t('tree.noFiles') }}</p>
      <template v-for="node in visibleTree" :key="node.path">
        <template v-if="node.dir">
          <DirNode
            :node="node" :depth="0"
            :collapsed="collapsed" :checked="checked"
            @toggle-dir="toggleDir" @toggle-node="toggleNode"
          />
          <template v-if="!collapsed.has(node.path)">
            <DirChildren
              :nodes="node.children" :depth="1"
              :collapsed="collapsed" :checked="checked"
              @toggle-dir="toggleDir" @toggle-node="toggleNode" @toggle-file="toggleFile"
            />
          </template>
        </template>
        <FileRow v-else :node="node" :depth="0" :checked="checked" @toggle-file="toggleFile" />
      </template>
    </div>

    <div class="foot muted">{{ t('tree.stats', { count: stats.count, size: stats.size }) }}</div>
  </div>
</template>

<!-- 递归子层：目录行 + 文件行 -->
<script lang="ts">
import { defineComponent, h, type PropType } from 'vue';
import type { TreeNode } from './filetree-types';
import { t } from '../i18n';

// 前 4 层每层 12px，之后每层 6px：深层目录不再把名称挤出可视区
const TREE_STEP = 12;
const DEEP_STEP = 6;
const DEEP_AFTER = 4;
const indentPx = (depth: number) =>
  depth <= DEEP_AFTER
    ? depth * TREE_STEP
    : DEEP_AFTER * TREE_STEP + (depth - DEEP_AFTER) * DEEP_STEP;

const Chevron = (open: boolean, onToggle: () => void) =>
  h(Icon, {
    name: 'chevron' as const,
    size: 14,
    class: 'chevron',
    title: open ? t('tree.collapse') : t('tree.expand'),
    style: { transform: open ? '' : 'rotate(-90deg)' },
    onClick: (e: Event) => { e.stopPropagation(); onToggle(); },
  });

function fmtSize(n: number): string {
  if (n < 1024) return n + ' B';
  return (n / 1024).toFixed(1) + ' kB';
}

const DirNode = defineComponent({
  name: 'DirNode',
  props: {
    node: { type: Object as PropType<TreeNode>, required: true },
    depth: { type: Number, required: true },
    collapsed: { type: Set as PropType<Set<string>>, required: true },
    checked: { type: Set as PropType<Set<string>>, required: true },
  },
  emits: ['toggle-dir', 'toggle-node'],
  setup(p, { emit }) {
    return () => {
      const leaves = (function collect(n: TreeNode): string[] {
        const out: string[] = [];
        for (const c of n.children) {
          if (c.dir) out.push(...collect(c));
          else out.push(c.path);
        }
        return out;
      })(p.node);
      const hit = leaves.filter((x) => p.checked.has(x)).length;
      const state = hit === 0 ? 'none' : hit === leaves.length ? 'all' : 'some';
      const indent = indentPx(p.depth);
      return h('div', {
        class: ['file-row', 'dir-row', { deep: p.depth >= DEEP_AFTER }],
        style: { paddingLeft: indent + 'px', '--indent': indent + 'px' },
        'data-depth': p.depth,
        onClick: () => emit('toggle-dir', p.node.path),
      }, [
        Chevron(!p.collapsed.has(p.node.path), () => emit('toggle-dir', p.node.path)),
        h('input', {
          type: 'checkbox',
          indeterminate: state === 'some',
          checked: state === 'all',
          onClick: (e: Event) => e.stopPropagation(),
          onChange: () => emit('toggle-node', p.node),
        }),
        h(Icon, { name: 'folder', size: 13 }),
        h('span', {
          class: 'name',
          title: p.depth >= DEEP_AFTER
            ? p.node.path + ' · ' + fmtSize(p.node.sizeBytes)
            : p.node.path,
        }, p.node.name),
        h('span', { class: 'size' }, fmtSize(p.node.sizeBytes)),
      ]);
    };
  },
});

const FileRow = defineComponent({
  name: 'FileRow',
  props: {
    node: { type: Object as PropType<TreeNode>, required: true },
    depth: { type: Number, required: true },
    checked: { type: Set as PropType<Set<string>>, required: true },
  },
  emits: ['toggle-file'],
  setup(p, { emit }) {
    return () => {
      const indent = indentPx(p.depth);
      return h('div', {
        class: ['file-row', { deep: p.depth >= DEEP_AFTER }],
        style: { paddingLeft: indent + 'px', '--indent': indent + 'px' },
        'data-depth': p.depth,
        onClick: () => emit('toggle-file', p.node.path),
      }, [
        h('span', { class: 'chevron', style: { visibility: 'hidden' } }), // 与目录行箭头占位对齐
        h('input', {
          type: 'checkbox',
          checked: p.checked.has(p.node.path),
          onClick: (e: Event) => e.stopPropagation(),
          onChange: () => emit('toggle-file', p.node.path),
        }),
        h(Icon, { name: 'doc', size: 13 }),
        h('span', {
          class: 'name',
          title: p.depth >= DEEP_AFTER
            ? p.node.path + ' · ' + fmtSize(p.node.sizeBytes)
            : p.node.path,
        }, p.node.name),
        h('span', { class: 'size' }, fmtSize(p.node.sizeBytes)),
      ]);
    };
  },
});

const DirChildren = defineComponent({
  name: 'DirChildren',
  props: {
    nodes: { type: Array as PropType<TreeNode[]>, required: true },
    depth: { type: Number, required: true },
    collapsed: { type: Set as PropType<Set<string>>, required: true },
    checked: { type: Set as PropType<Set<string>>, required: true },
  },
  emits: ['toggle-dir', 'toggle-node', 'toggle-file'],
  setup(p, { emit }) {
    const render = (): ReturnType<typeof h>[] => {
      const out: ReturnType<typeof h>[] = [];
      for (const n of p.nodes) {
        if (n.dir) {
          out.push(h(DirNode, {
            node: n, depth: p.depth,
            collapsed: p.collapsed, checked: p.checked,
            'onToggle-dir': (path: string) => emit('toggle-dir', path),
            'onToggle-node': (node: TreeNode) => emit('toggle-node', node),
          }));
          if (!p.collapsed.has(n.path)) {
            out.push(h(DirChildren, {
              nodes: n.children, depth: p.depth + 1,
              collapsed: p.collapsed, checked: p.checked,
              'onToggle-dir': (path: string) => emit('toggle-dir', path),
              'onToggle-node': (node: TreeNode) => emit('toggle-node', node),
              'onToggle-file': (path: string) => emit('toggle-file', path),
            }));
          }
        } else {
          out.push(h(FileRow, {
            node: n, depth: p.depth, checked: p.checked,
            'onToggle-file': (path: string) => emit('toggle-file', path),
          }));
        }
      }
      return out;
    };
    return render;
  },
});

export default defineComponent({ name: 'FileTreeRoot' });
</script>

<style scoped>
.filetree { display: flex; flex-direction: column; height: 100%; }
.head { flex: none; }
.title { font-weight: bold; margin-bottom: 6px; }
.search { width: 100%; box-sizing: border-box; margin-bottom: 6px; }
.quick { display: flex; gap: 4px; margin-bottom: 8px; flex-wrap: wrap; }
.quick button { font-size: 12px; padding: 2px 8px; }
.treewrap { flex: 1; min-height: 0; overflow: auto; border: 1px solid var(--line); border-radius: var(--r-md); padding: 4px; }
.hint { padding: 6px 8px; }
.foot { flex: none; padding: 8px 2px 0; }
.muted { color: var(--fg-3); font-size: 12px; }
</style>

<style>
/* 文件行全局样式（递归子组件在 scoped 外） */
.file-row {
  --indent: 0px;
  display: flex; align-items: center; gap: 6px;
  width: max-content; min-width: 100%;
  height: 28px; padding-right: 8px;
  cursor: pointer; border-radius: var(--r-sm);
  font-size: 13px; color: var(--fg-1);
  text-align: left;
  position: relative;
}
.file-row:hover { background: var(--bg-3); }
/* 缩进参考线跟随 JS 计算的 --indent，深层级不再线性外扩 */
.file-row::before {
  content: '';
  position: absolute;
  left: calc(var(--indent) - 6px);
  top: 0; bottom: 0;
  width: 1px;
  background: var(--line);
}
.file-row[data-depth='0']::before { display: none; }
/* 名称不再被省略号截断；行宽不足时整棵树横向滚动 */
.file-row .name { flex: 1 1 auto; min-width: 10ch; white-space: nowrap; }
.file-row .size { flex: none; color: var(--fg-3); font-size: 12px; font-variant-numeric: tabular-nums; }
/* 深层级让出大小列，大小改由 hover 的完整路径提示承载 */
.file-row.deep .size { display: none; }
.file-row .chevron { width: 14px; height: 14px; color: var(--fg-2); flex: none; }
.file-row .icon { flex: none; font-size: 12px; }
.file-row input[type='checkbox'] { flex: none; margin: 0; accent-color: var(--accent); }
</style>
