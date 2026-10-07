<!-- 预检结果卡片（设计文档 §2.3） -->
<script setup lang="ts">
import type { ScanPreview } from '../../api/bindings';
import { fmtNum } from '../../estimate';
import { useI18n } from '../../i18n';

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
    <p v-for="(w, i) in preview.warnings" :key="i" class="warn">⚠ {{ w }}</p>
  </div>
</template>

<style scoped>
.scanpreview { background: #0f1720; border: 1px solid #2a3848; border-radius: 6px; padding: 8px 10px; margin-top: 6px; font-size: 12px; }
.ok { color: #68d391; margin: 0 0 4px; }
.langs { color: #8fa3b8; margin: 0; }
.warn { color: #f6ad55; margin: 4px 0 0; }
</style>
