// dirPaths.ts — 目录选择中的父子路径判定（目录多选树与目录标签共用唯一实现）。
// 路径统一按正斜杠 + 去尾斜杠比较，Windows 下大小写不敏感。

export function normalizeDirPath(p: string): string {
  return p.replace(/\\/g, '/').replace(/\/+$/, '');
}

function isAbsolutePath(p: string): boolean {
  return /^[a-zA-Z]:/.test(p) || p.startsWith('/');
}

/** path 是否位于 ancestor 之下（按路径组件边界比较，避免 build 与 build2 混淆）。 */
export function isDescendant(path: string, ancestor: string): boolean {
  const p = normalizeDirPath(path).toLowerCase();
  const a = normalizeDirPath(ancestor).toLowerCase();
  return p !== a && p.startsWith(a + '/');
}

/** 同一目录（Windows 下大小写不敏感、忽略尾斜杠与反斜杠差异）。 */
export function isSameDirPath(a: string, b: string): boolean {
  return normalizeDirPath(a).toLowerCase() === normalizeDirPath(b).toLowerCase();
}

/** ancestor 是否已覆盖 path：后代关系，或 ancestor 为项目根 "." 时的任意相对目录。 */
export function isCovered(path: string, ancestor: string): boolean {
  const p = normalizeDirPath(path);
  const a = normalizeDirPath(ancestor);
  if (isSameDirPath(p, a)) return false;
  if (isDescendant(p, a)) return true;
  return a === '.' && !isAbsolutePath(p);
}

/** 去掉重复项与被其它项（祖先 / 项目根）覆盖的子目录，只保留最顶层目录。 */
export function normalizeDirSelection(paths: string[]): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const raw of paths) {
    const p = normalizeDirPath(raw);
    if (!p || seen.has(p.toLowerCase())) continue;
    seen.add(p.toLowerCase());
    out.push(p);
  }
  return out.filter((p) => !out.some((a) => isCovered(p, a)));
}
