// estimate.ts — 前端 token 估算：ASCII 4 字符 ≈ 1 token，非 ASCII（中文等）1 字 ≈ 1 token
export function estimateTokens(text: string): number {
  if (!text) return 0;
  let nonAscii = 0;
  let ascii = 0;
  for (const ch of text) {
    if (ch.charCodeAt(0) > 127) nonAscii++;
    else ascii++;
  }
  return Math.ceil(nonAscii + ascii / 4);
}

// 按 FileInfo 的字节数粗估（无法拿到内容时的降级：按 1 字节 ≈ 0.3 token，偏保守）
export function estimateTokensFromSize(sizeBytes: number): number {
  return Math.ceil(sizeBytes * 0.3);
}

export function fmtNum(n: number): string {
  return n.toLocaleString('en-US');
}
