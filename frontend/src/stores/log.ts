import { defineStore } from 'pinia';
import { ref } from 'vue';
import { Events } from '@wailsio/runtime';

export interface LogEntry {
  time: string;
  level: string;
  msg: string;
  source: string;
  /** 后端 slog 结构化字段（path/err 等），仅转发日志携带 */
  attrs?: Record<string, string>;
}

export const useLogStore = defineStore('log', () => {
  const entries = ref<LogEntry[]>([]);
  const maxEntries = 500;
  let listening = false;

  function push(e: LogEntry) {
    entries.value.push(e);
    if (entries.value.length > maxEntries) {
      entries.value.splice(0, entries.value.length - maxEntries);
    }
  }

  function clear() {
    entries.value = [];
  }

  function now(): string {
    return new Date().toLocaleString('sv-SE'); // "YYYY-MM-DD HH:mm:ss"，与后端格式一致
  }

  // 前端 UI 操作日志（source 标记为 ui，与后端 slog 转发区分）
  function info(msg: string) {
    push({ time: now(), level: 'INFO', msg, source: 'ui' });
  }
  function warn(msg: string) {
    push({ time: now(), level: 'WARN', msg, source: 'ui' });
  }
  function error(msg: string) {
    push({ time: now(), level: 'ERROR', msg, source: 'ui' });
  }

  // 监听后端转发的日志事件。Wails v3 运行时由 @wailsio/runtime 提供；
  // 桌面环境下事件始终可用（浏览器 dev 下仅收不到事件，无副作用）。
  // v3 载荷在事件对象上：emit 单参时 data 即该值，数组形态时取首参。
  function listen() {
    if (listening) return;
    Events.On('app:log', (ev) => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const d: any = (ev as any)?.data;
      push(Array.isArray(d) ? d[0] : d);
    });
    listening = true;
  }

  // —— 全局 toast（保存成功等短提示，自动消失；error 加长到 8s 且颜色分级）——
  const toast = ref<{ msg: string; kind: 'success' | 'warn' | 'error'; seq: number } | null>(null);
  let toastTimer: ReturnType<typeof setTimeout> | undefined;
  function showToast(msg: string, kind: 'success' | 'warn' | 'error' = 'success') {
    // seq 递增保证连续触发时过渡动画能重放；error 更久以免错过
    toast.value = { msg, kind, seq: Date.now() };
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => { toast.value = null; }, kind === 'error' ? 8000 : 2600);
  }

  return { entries, clear, listen, push, info, warn, error, toast, showToast };
});
