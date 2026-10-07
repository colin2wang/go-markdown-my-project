import { defineStore } from 'pinia';
import { ref } from 'vue';
import { EventsOn } from '../../wailsjs/runtime/runtime';

export interface LogEntry {
  time: string;
  level: string;
  msg: string;
  source: string;
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

  // 监听后端转发的日志事件。Wails v2 的 runtime 由 wailsjs 模块提供，
  // 不依赖 window.runtime；桌面环境下 EventsOn 始终可用（浏览器 dev 下仅收不到事件，无副作用）。
  function listen() {
    if (listening) return;
    EventsOn('app:log', (e: LogEntry) => push(e));
    listening = true;
  }

  return { entries, clear, listen, push, info, warn };
});
