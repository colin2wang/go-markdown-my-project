<script setup lang="ts">
export type IconName = 'folder' | 'gear' | 'doc' | 'chevron' | 'close' | 'open' | 'search' | 'box' | 'ban' | 'inbox' | 'outbox' | 'shield' | 'chart' | 'check' | 'refresh' | 'warn';
const props = withDefaults(defineProps<{ name: IconName; size?: number }>(), { size: 15 });
const paths: Record<IconName, string> = {
  folder:  'M2.5 5.2c0-.9.7-1.6 1.6-1.6h2.4l1.5 1.7h3.9c.9 0 1.6.7 1.6 1.6v4.6c0 .9-.7 1.6-1.6 1.6H4.1c-.9 0-1.6-.7-1.6-1.6V5.2Z',
  gear:    'M8 5.8a2.2 2.2 0 1 0 0 4.4 2.2 2.2 0 0 0 0-4.4ZM8 1.8v1.6M8 12.6v1.6M14.2 8h-1.6M3.4 8H1.8M12.4 3.6l-1.1 1.1M4.7 11.3l-1.1 1.1M12.4 12.4l-1.1-1.1M4.7 4.7 3.6 3.6',
  doc:     'M4.5 2.5h4.6l2.9 2.9v8.1H4.5V2.5ZM9.1 2.5v2.9H12',
  chevron: 'M4.5 6.5 8 10l3.5-3.5',
  close:   'M4 4l8 8M12 4l-8 8',
  open:    'M6.5 3.5h-3v9h9V9.5M9.5 2.5h4v4M13.5 2.5 8 8',
  search:  'M7 11.5a4.5 4.5 0 1 0 0-9 4.5 4.5 0 0 0 0 9ZM10.3 10.3l3.2 3.2',
  box:     'M2.5 5.5 8 3l5.5 2.5v5L8 13l-5.5-2.5v-5ZM2.5 5.5 8 8l5.5-2.5M8 8v5',
  ban:     'M8 13.5A5.5 5.5 0 1 0 8 2.5a5.5 5.5 0 0 0 0 11ZM4.2 4.2l7.6 7.6',
  inbox:   'M8 2.5v7M5.2 7.2 8 10l2.8-2.8M3 13h10',
  outbox:  'M8 9.5v-7M5.2 4.8 8 2l2.8 2.8M3 13h10',
  shield:  'M8 2.5l5 2v3.2c0 3-2.2 5-5 5.8-2.8-.8-5-2.8-5-5.8V4.5l5-2Z',
  chart:   'M3 13.5V8.5M8 13.5V3.5M13 13.5v-3',
  check:   'M3 8.5 6.5 12 13 4.5',
  refresh: 'M12.8 6.2A5 5 0 1 0 13 9M13 3.5V6.5H10',
  warn:    'M8 2.8 14 13H2L8 2.8ZM8 6.4v3.3M8 11.5v.1',
};
</script>
<template>
  <svg :width="props.size" :height="props.size" viewBox="0 0 16 16" fill="none"
       stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"
       aria-hidden="true" class="icon">
    <path :d="paths[name]" />
  </svg>
</template>
<style scoped>
.icon { display: inline-block; vertical-align: -2px; flex: none; }
</style>
