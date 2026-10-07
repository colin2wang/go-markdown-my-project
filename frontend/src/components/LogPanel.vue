<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import { useLogStore } from '../stores/log';
import { useI18n } from '../i18n';

const { t } = useI18n();
const log = useLogStore();
const collapsed = ref(false);
const autoscroll = ref(true);
const scroller = ref<HTMLElement | null>(null);

watch(
  () => log.entries.length,
  async () => {
    if (!autoscroll.value || collapsed.value) return;
    await nextTick();
    if (scroller.value) scroller.value.scrollTop = scroller.value.scrollHeight;
  }
);

function onScroll() {
  const el = scroller.value;
  if (!el) return;
  autoscroll.value = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
}

const counts = computed(() => {
  const c = { DEBUG: 0, INFO: 0, WARN: 0, ERROR: 0 };
  for (const e of log.entries) {
    const l = e.level.toUpperCase();
    if (l in c) c[l as keyof typeof c]++;
  }
  return c;
});

function levelClass(level: string) {
  return 'lvl-' + level.toLowerCase();
}
</script>

<template>
  <section class="logpanel" :class="{ collapsed }">
    <header class="log-head">
      <button class="toggle" @click="collapsed = !collapsed">{{ collapsed ? '▸' : '▾' }} {{ t('log.title') }}</button>
      <span class="counts">
        <span class="lvl-debug">D {{ counts.DEBUG }}</span>
        <span class="lvl-info">I {{ counts.INFO }}</span>
        <span class="lvl-warn">W {{ counts.WARN }}</span>
        <span class="lvl-error">E {{ counts.ERROR }}</span>
      </span>
      <span class="spacer" />
      <label class="auto"><input v-model="autoscroll" type="checkbox" /> {{ t('log.autoscroll') }}</label>
      <button class="clear" @click="log.clear()">{{ t('log.clear') }}</button>
    </header>
    <div v-show="!collapsed" ref="scroller" class="log-body" @scroll="onScroll">
      <div v-for="(e, i) in log.entries" :key="i" class="log-line" :class="levelClass(e.level)">
        <span class="t">{{ e.time }}</span>
        <span class="lv">{{ e.level }}</span>
        <span class="m">{{ e.msg }}</span>
        <span v-if="e.source" class="src">{{ e.source }}</span>
      </div>
      <div v-if="log.entries.length === 0" class="empty">{{ t('log.empty') }}</div>
    </div>
  </section>
</template>

<style scoped>
.logpanel {
  flex: none;
  height: 170px;
  display: flex;
  flex-direction: column;
  background: #0c141c;
  border-top: 1px solid #3a4a5c;
  color: #cdd9e5;
}
.logpanel.collapsed { height: 32px; }
.log-head {
  flex: none;
  height: 32px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 10px;
  background: #16222e;
  border-bottom: 1px solid #2a3848;
  font-size: 12px;
}
.toggle { background: none; border: none; color: #8fa3b8; cursor: pointer; padding: 0 4px; font-size: 12px; }
.counts { display: flex; gap: 8px; font-variant-numeric: tabular-nums; }
.spacer { flex: 1; }
.auto { display: flex; align-items: center; gap: 4px; color: #8fa3b8; }
.clear { border: 1px solid #3a4a5c; background: #1f2d3d; color: #eee; border-radius: 4px; padding: 2px 8px; cursor: pointer; }
.log-body {
  flex: 1;
  overflow-y: auto;
  padding: 4px 8px;
  font-family: ui-monospace, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.6;
  text-align: left; /* 日志始终左对齐 */
}
.log-line { white-space: pre-wrap; word-break: break-all; }
.log-line .t { color: #6b7c8f; margin-right: 8px; }
.log-line .lv { display: inline-block; width: 52px; margin-right: 8px; font-weight: bold; }
.log-line .src { color: #6b7c8f; margin-left: 8px; }
.lvl-debug .lv { color: #6b7c8f; }
.lvl-info .lv { color: #63b3ed; }
.lvl-warn .lv, .lvl-warn { color: #f6ad55; }
.lvl-error .lv, .lvl-error { color: #ff8383; }
.counts .lvl-debug { color: #6b7c8f; }
.counts .lvl-info { color: #63b3ed; }
.counts .lvl-warn { color: #f6ad55; }
.counts .lvl-error { color: #ff8383; }
.empty { color: #6b7c8f; font-style: italic; }
</style>
