// 轻量国际化：无第三方依赖，locale 为响应式 ref，t() 在渲染期读取以获得响应性。
import { computed, ref, watchEffect } from 'vue';
import zhCN from './locales/zh-CN';
import enUS from './locales/en-US';
import jaJP from './locales/ja-JP';

export type LocaleCode = 'zh-CN' | 'en-US' | 'ja-JP';

export const locales: { code: LocaleCode; label: string }[] = [
  { code: 'zh-CN', label: '简体中文' },
  { code: 'en-US', label: 'English' },
  { code: 'ja-JP', label: '日本語' },
];

const dicts: Record<LocaleCode, typeof zhCN> = {
  'zh-CN': zhCN,
  'en-US': enUS,
  'ja-JP': jaJP,
};

const STORAGE_KEY = 'app.locale';

function isLocale(v: unknown): v is LocaleCode {
  return typeof v === 'string' && v in dicts;
}

function detectLocale(): LocaleCode {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (isLocale(saved)) return saved;
  } catch {
    // localStorage 不可用（隐私模式等）时忽略
  }
  const nav = (typeof navigator !== 'undefined' && navigator.language) || '';
  if (nav.startsWith('ja')) return 'ja-JP';
  if (nav.startsWith('zh')) return 'zh-CN';
  return 'en-US';
}

const locale = ref<LocaleCode>('zh-CN');

/** 应用启动时调用：读取持久化语言并同步 <html lang>。 */
export function initI18n(): void {
  locale.value = detectLocale();
  watchEffect(() => {
    if (typeof document !== 'undefined') document.documentElement.lang = locale.value;
  });
}

export function setLocale(code: LocaleCode): void {
  locale.value = code;
  try {
    localStorage.setItem(STORAGE_KEY, code);
  } catch {
    // 忽略持久化失败
  }
}

function resolve(dict: unknown, key: string): string | undefined {
  let cur: unknown = dict;
  for (const seg of key.split('.')) {
    if (cur == null || typeof cur !== 'object') return undefined;
    cur = (cur as Record<string, unknown>)[seg];
  }
  return typeof cur === 'string' ? cur : undefined;
}

/** 按 key 取当前语言文案；缺失时回退中文，再缺失则返回 key 本身。 */
export function t(key: string, vars?: Record<string, string | number | undefined>): string {
  const raw = resolve(dicts[locale.value], key) ?? resolve(zhCN, key) ?? key;
  if (!vars) return raw;
  return raw.replace(/\{(\w+)\}/g, (m, k: string) => (k in vars ? String(vars[k]) : m));
}

export function useI18n() {
  return {
    t,
    locale: computed(() => locale.value),
    setLocale,
    locales,
  };
}

export default { t, useI18n, setLocale, initI18n, locales };
