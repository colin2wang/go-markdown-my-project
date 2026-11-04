<!-- 分区容器：锚点 id、标题、错误徽标、置灰（设计文档 §8） -->
<script setup lang="ts">
import Icon from '../ui/Icon.vue';
import type { IconName } from '../ui/Icon.vue';

defineProps<{
  id: string;
  title: string;
  icon?: IconName;
  errorCount?: number;
  muted?: boolean;
  hint?: string;
}>();
</script>

<template>
  <section :id="id" class="esection" :class="{ muted }">
    <h4 class="sec-title">
      <Icon v-if="icon" :name="icon" :size="14" />
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
.esection { border: 1px solid var(--line); border-radius: var(--r-md); padding: 10px 12px; margin-bottom: 10px; background: var(--bg-2); }
.esection.muted { opacity: .45; }
.sec-title { margin: 0 0 8px; font-size: 13px; color: var(--fg-2); font-weight: 600; display: flex; align-items: center; gap: 6px; }
.sec-icon { font-size: 13px; }
.badge { background: transparent; color: var(--err); border: 1px solid var(--err); border-radius: var(--r-sm); padding: 0 6px; font-size: 10px; font-variant-numeric: tabular-nums; }
.sec-hint { margin-left: auto; font-size: 11px; color: var(--fg-3); font-weight: normal; }
.sec-body :deep(label) { display: block; margin: 6px 0; font-size: 12px; color: var(--fg-1); }
.sec-body :deep(label.inline) { display: flex; align-items: center; gap: 6px; }
.sec-body :deep(input), .sec-body :deep(select) {
  background: var(--bg-3); color: var(--fg-1); border: 1px solid var(--line-2);
  border-radius: var(--r-sm); padding: 4px 8px; font-size: 12px;
}
.sec-body :deep(input.invalid) { border-color: var(--err); }
.field-error { color: var(--err); font-size: 11px; margin: 2px 0 6px 2px; }
</style>
