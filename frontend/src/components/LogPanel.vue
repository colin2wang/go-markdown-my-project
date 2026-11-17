<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import { useLogStore } from '../stores/log';
import { useI18n } from '../i18n';
import Icon from './ui/Icon.vue';

const { t } = useI18n();
const log = useLogStore();
const collapsed = ref(false);
const autoscroll = ref(true);
const scroller = ref<HTMLElement | null>(null);
// level 过滤（空串 = 全部）与关键字搜索（大小写不敏感，匹配 msg 与属性文本）
const levelFilter = ref('');
const query = ref('');
const LEVELS = ['DEBUG', 'INFO', 'WARN', 'ERROR'] as const;

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase();
  return log.entries.filter((e) => {
    if (levelFilter.value && e.level.toUpperCase() !== levelFilter.value) return false;
    if (!q) return true;
    if (e.msg.toLowerCase().includes(q)) return true;
    if (e.source.toLowerCase().includes(q)) return true;
    return e.attrs ? Object.entries(e.attrs).some(([k, v]) => (k + '=' + v).toLowerCase().includes(q)) : false;
  });
});

// 折叠时顶栏 error 徽标：有未读错误即点亮（展开或清屏后熄灭）
const collapsedErrors = computed(() => (collapsed.value ? counts.value.ERROR : 0));

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
      <button class="toggle" @click="collapsed = !collapsed">
        <Icon name="chevron" :size="12" :class="{ 'is-collapsed': collapsed }" /> {{ t('log.title') }}
        <span v-if="collapsedErrors" class="err-badge" :title="t('log.errors', { n: collapsedErrors })">{{ collapsedErrors }}</span>
      </button>
      <span class="spacer" />
      <template v-if="!collapsed">
        <input v-model="query" class="search" :placeholder="t('log.search')" />
        <span class="chips">
          <button class="chip" :class="{ active: levelFilter === '' }" @click="levelFilter = ''">{{ t('log.all') }}</button>
          <button v-for="lv in LEVELS" :key="lv" class="chip" :class="['chip-' + lv.toLowerCase(), { active: levelFilter === lv }]" @click="levelFilter = levelFilter === lv ? '' : lv">
            {{ lv[0] }} {{ counts[lv] }}
          </button>
        </span>
        <label class="auto"><input v-model="autoscroll" type="checkbox" /> {{ t('log.autoscroll') }}</label>
      </template>
      <button class="clear" @click="log.clear(); levelFilter = ''; query = ''">{{ t('log.clear') }}</button>
    </header>
    <div v-show="!collapsed" ref="scroller" class="log-body" @scroll="onScroll">
      <div v-for="(e, i) in filtered" :key="i" class="log-line" :class="levelClass(e.level)">
        <span class="t">{{ e.time }}</span>
        <span class="lv">{{ e.level }}</span>
        <span class="m">{{ e.msg }}</span><template v-if="e.attrs"><span v-for="(v, k) in e.attrs" :key="k" class="attr"> {{ k }}={{ v }}</span></template>
        <span v-if="e.source" class="src">{{ e.source }}</span>
      </div>
      <div v-if="filtered.length === 0" class="empty">{{ log.entries.length === 0 ? t('log.empty') : t('log.noMatch') }}</div>
    </div>
  </section>
</template>

<style scoped>
.logpanel {
  flex: none;
  height: 170px;
  display: flex;
  flex-direction: column;
  background: var(--bg-0);
  border-top: 1px solid var(--line);
  color: var(--fg-1);
}
.logpanel.collapsed { height: 32px; }
.log-head {
  flex: none;
  height: 32px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 10px;
  background: var(--bg-1);
  border-bottom: 1px solid var(--line);
  font-size: 12px;
}
.toggle { background: none; border: none; color: var(--fg-2); cursor: pointer; padding: 0 4px; font-size: 12px; display: inline-flex; align-items: center; gap: 4px; }
.toggle .is-collapsed { transform: rotate(-90deg); }
/* 折叠态 error 徽标：有未读错误即点亮 */
.err-badge { min-width: 14px; height: 14px; padding: 0 3px; border-radius: 7px; background: var(--err); color: #fff; font-size: 10px; line-height: 14px; text-align: center; font-weight: 600; }
/* level 过滤 chips：单字 + 计数，激活态高亮对应 level 色 */
.chips { display: flex; gap: 4px; }
.chip { border: 1px solid var(--line-2); background: transparent; color: var(--fg-3); border-radius: var(--r-sm); padding: 1px 6px; font-size: 11px; cursor: pointer; font-variant-numeric: tabular-nums; }
.chip:hover { color: var(--fg-1); }
.chip.active { border-color: var(--accent); color: var(--accent); }
.chip.active.chip-warn { border-color: var(--warn); color: var(--warn); }
.chip.active.chip-error { border-color: var(--err); color: var(--err); }
.search { width: 140px; padding: 2px 8px; font-size: 12px; }
/* 计数徽标去掉彩色：只有数字本身在出错时才着色 */
.counts { display: flex; gap: 8px; font-variant-numeric: tabular-nums; color: var(--fg-3); }
.spacer { flex: 1; }
.auto { display: flex; align-items: center; gap: 4px; color: var(--fg-2); }
.clear { border: 1px solid var(--line-2); background: transparent; color: var(--fg-2); border-radius: var(--r-sm); padding: 2px 8px; cursor: pointer; }
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
.log-line .t { color: var(--fg-3); margin-right: 8px; }
.log-line .lv { display: inline-block; width: 52px; margin-right: 8px; font-weight: bold; }
.log-line .src { color: var(--fg-3); margin-left: 8px; }
.log-line .attr { color: var(--fg-2); }
.lvl-debug .lv { color: var(--fg-3); }
.lvl-info .lv { color: var(--fg-2); }        /* INFO 不再抢眼 */
.lvl-warn .lv, .lvl-warn { color: var(--warn); }
.lvl-error .lv, .lvl-error { color: var(--err); }
.counts .lvl-debug { color: var(--fg-3); }
.counts .lvl-info { color: var(--fg-2); }
.counts .lvl-warn { color: var(--warn); }
.counts .lvl-error { color: var(--err); }
.empty { color: var(--fg-3); font-style: italic; }
</style>
