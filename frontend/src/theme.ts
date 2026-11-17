// 主题应用：把全局设置里的 theme 写到 <html data-theme>，样式表按属性切换变量。
import { ref } from 'vue';

export type Theme = 'dark' | 'light';
const theme = ref<Theme>('dark');

export function applyTheme(v: string | undefined | null) {
  theme.value = v === 'light' ? 'light' : 'dark';
  document.documentElement.dataset.theme = theme.value;
}

export function currentTheme() {
  return theme.value;
}
