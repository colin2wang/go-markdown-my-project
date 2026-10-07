<!-- 分区容器：锚点 id、标题、错误徽标、置灰（设计文档 §8） -->
<script setup lang="ts">
defineProps<{
  id: string;
  title: string;
  icon?: string;
  errorCount?: number;
  muted?: boolean;
  hint?: string;
}>();
</script>

<template>
  <section :id="id" class="esection" :class="{ muted }">
    <h4 class="sec-title">
      <span v-if="icon" class="sec-icon">{{ icon }}</span>
      {{ title }}
      <span v-if="errorCount" class="badge">{{ errorCount }}</span>
      <span v-if="hint" class="sec-hint">{{ hint }}</span>
    </h4>
    <div class="sec-body">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.esection { border: 1px solid #2a3848; border-radius: 8px; padding: 10px 12px; margin-bottom: 10px; background: #101a24; }
.esection.muted { opacity: .45; }
.sec-title { margin: 0 0 8px; font-size: 13px; color: #8fa3b8; display: flex; align-items: center; gap: 6px; }
.sec-icon { font-size: 13px; }
.badge { background: #742a2a; color: #ff8383; border-radius: 10px; padding: 0 7px; font-size: 11px; }
.sec-hint { margin-left: auto; font-size: 11px; color: #6b7c8f; font-weight: normal; }
.sec-body :deep(label) { display: block; margin: 6px 0; font-size: 12px; color: #cdd9e5; }
.sec-body :deep(label.inline) { display: flex; align-items: center; gap: 6px; }
.sec-body :deep(input), .sec-body :deep(select) {
  background: #0f1720; color: #eee; border: 1px solid #3a4a5c;
  border-radius: 4px; padding: 4px 8px; font-size: 12px;
}
.sec-body :deep(input.invalid) { border-color: #c53030; }
.field-error { color: #ff8383; font-size: 11px; margin: 2px 0 6px 2px; }
</style>
