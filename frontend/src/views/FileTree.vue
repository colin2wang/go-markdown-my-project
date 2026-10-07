<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { FileInfo } from '../api/bindings';
import { useLogStore } from '../stores/log';

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
  // 目录在前、文件在后，各自按名称排序
  const sortRec = (n: TreeNode) => {
    n.children.sort((a, b) => (a.dir === b.dir ? a.name.localeCompare(b.name) : a.dir ? -1 : 1));
    n.children.forEach(sortRec);
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

// —— 快捷操作 ——
const log = useLogStore();

function selectAll() {
  setCheckedDeep(props.files.map((f) => f.path), true);
  log.info(`文件树：全选（${props.files.length} 个文件）`);
}
function selectNone() {
  setCheckedDeep(props.files.map((f) => f.path), false);
  log.info('文件树：全不选');
}
function selectInvert() {
  const next = new Set<string>();
  for (const f of props.files) if (!props.checked.has(f.path)) next.add(f.path);
  emit('update:checked', next);
  log.info(`文件树：反选（选中 ${next.size}/${props.files.length}）`);
}
function selectCodeOnly() {
  const next = new Set(props.files.filter((f) => f.language !== 'Text').map((f) => f.path));
  emit('update:checked', next);
  log.info(`文件树：仅代码（选中 ${next.size}/${props.files.length}）`);
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
      <div class="title">文件 <span class="muted">({{ stats.count }}/{{ stats.total }})</span></div>
      <input v-model="search" class="search" placeholder="🔍 搜索文件…" />
      <div class="quick">
        <button @click="selectAll">全选</button>
        <button @click="selectInvert">反选</button>
        <button @click="selectCodeOnly">仅代码</button>
        <button @click="selectNone">全不选</button>
      </div>
    </div>

    <div class="treewrap">
      <p v-if="search" class="muted hint">匹配 {{ matchCount }} 个文件</p>
      <p v-if="files.length === 0" class="muted hint">未扫描到文件</p>
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

    <div class="foot muted">{{ stats.count }} 文件 · {{ stats.size }}</div>
  </div>
</template>

<!-- 递归子层：目录行 + 文件行 -->
<script lang="ts">
import { defineComponent, h, type PropType } from 'vue';
import type { TreeNode } from './filetree-types';

const Indent = (depth: number) => h('span', { class: 'indent', style: { width: depth * 16 + 'px' } });

const Chevron = (open: boolean, onToggle: () => void) =>
  h('span', {
    class: 'chevron',
    title: open ? '收起' : '展开',
    onClick: (e: Event) => { e.stopPropagation(); onToggle(); },
  }, open ? '▾' : '▸');

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
      return h('div', {
        class: 'file-row dir-row',
        style: { paddingLeft: p.depth * 16 + 'px' },
        onClick: () => emit('toggle-dir', p.node.path),
      }, [
        Indent(0),
        Chevron(!p.collapsed.has(p.node.path), () => emit('toggle-dir', p.node.path)),
        h('input', {
          type: 'checkbox', class: state === 'some' ? 'some' : '',
          checked: state === 'all',
          onClick: (e: Event) => e.stopPropagation(),
          onChange: () => emit('toggle-node', p.node),
        }),
        h('span', { class: 'icon' }, '📁'),
        h('span', { class: 'name' }, p.node.name),
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
    return () => h('div', {
      class: 'file-row',
      style: { paddingLeft: p.depth * 16 + 'px' },
      onClick: () => emit('toggle-file', p.node.path),
    }, [
      h('input', {
        type: 'checkbox',
        checked: p.checked.has(p.node.path),
        onClick: (e: Event) => e.stopPropagation(),
        onChange: () => emit('toggle-file', p.node.path),
      }),
      h('span', { class: 'icon' }, '📄'),
      h('span', { class: 'name', title: p.node.path }, p.node.name),
      h('span', { class: 'size' }, fmtSize(p.node.sizeBytes)),
    ]);
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
.treewrap { flex: 1; min-height: 0; overflow-y: auto; border: 1px solid #3a4a5c; border-radius: 6px; padding: 4px; }
.hint { padding: 6px 8px; }
.foot { flex: none; padding: 8px 2px 0; }
.muted { color: #8fa3b8; font-size: 12px; }
</style>

<style>
/* 文件行全局样式（递归子组件在 scoped 外） */
.file-row {
  display: flex; align-items: center; gap: 8px;
  height: 28px; padding-right: 8px;
  cursor: pointer; border-radius: 4px;
  font-size: 13px; color: #eee;
  text-align: left;
}
.file-row:hover { background: #1f2d3d; }
.file-row .name { flex: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.file-row .size { color: #8fa3b8; font-size: 12px; }
.file-row .chevron { width: 14px; color: #8fa3b8; flex: none; }
.file-row .icon { flex: none; font-size: 12px; }
.file-row input[type='checkbox'] { flex: none; margin: 0; accent-color: #2b6cb0; }
.file-row input[type='checkbox'].some { opacity: .6; }
/* 半选态：用样式近似横线 */
input[type='checkbox'].some:not(:checked) {
  background: #2b6cb0; border-color: #2b6cb0;
  background-image: linear-gradient(transparent 45%, #eee 45%, #eee 55%, transparent 55%);
}
</style>
