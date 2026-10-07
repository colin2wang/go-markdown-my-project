// filetree-types.ts — FileTree.vue 树节点类型（供递归子组件共享）
export interface TreeNode {
  name: string;
  path: string;        // 相对路径（目录不带尾 /）
  dir: boolean;
  sizeBytes: number;
  language: string;
  children: TreeNode[];
}
