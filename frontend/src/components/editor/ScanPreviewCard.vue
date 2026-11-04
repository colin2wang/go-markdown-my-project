<!-- 预检结果卡片（设计文档 §2.3） -->
<script setup lang="ts">
import type { ScanPreview } from '../../api/bindings';
import { fmtNum } from '../../estimate';
import { useI18n } from '../../i18n';
import Icon from '../ui/Icon.vue';

const { t } = useI18n();
defineProps<{ preview: ScanPreview }>();

function fmtSize(n: number): string {
  if (n < 1024) return n + ' B';
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB';
  return (n / 1024 / 1024).toFixed(1) + ' MB';
}
</script>

<template>
  <div class="scanpreview">
    <p class="ok">{{ t('editor.previewDone', { count: preview.fileCount, size: fmtSize(preview.totalBytes) }) }}</p>
    <p class="langs">
      {{ Object.entries(preview.byLang).map(([l, n]) => `${l} ×${n}`).join(' · ') }}
    </p>
    <p v-for="(w, i) in preview.warnings" :key="i" class="warn"><Icon name="warn" :size="12" /> {{ w }}</p>
  </div>
</template>

<style scoped>
.scanpreview {
  background: var(--bg-3);
  border: 1px solid var(--line);
  border-radius: var(--r-md);
  padding: 8px 10px; margin-top: 6px; font-size: 12px;
}
.ok { color: var(--ok); margin: 0 0 4px; }
.langs { color: var(--fg-2); margin: 0; }
.warn { color: var(--warn); margin: 4px 0 0; display: flex; align-items: center; gap: 5px; }
</style>
