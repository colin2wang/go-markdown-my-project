<script setup lang="ts">
import type { SensitiveHit } from '../api/bindings';
import { useI18n } from '../i18n';
import Icon from '../components/ui/Icon.vue';

const { t } = useI18n();
defineProps<{ hits: SensitiveHit[] }>();
const emit = defineEmits<{ (e: 'close'): void }>();
</script>

<template>
  <div class="drawer-mask" @click.self="emit('close')">
    <aside class="drawer">
      <header>
        <h3>{{ t('sensitive.title', { n: hits.length }) }}</h3>
        <button class="link" @click="emit('close')"><Icon name="close" :size="12" /></button>
      </header>
      <p class="muted tip">{{ t('sensitive.tip') }}</p>
      <div class="list">
        <p v-if="hits.length === 0" class="muted">{{ t('sensitive.none') }}</p>
        <div v-for="(h, i) in hits" :key="i" class="hit">
          <div class="loc">{{ h.file }} : {{ h.line }} <span class="rule">[{{ h.rule }}]</span></div>
          <div class="masked">{{ h.masked }}</div>
        </div>
      </div>
      <footer>
        <button class="primary" @click="emit('close')">{{ t('sensitive.ok') }}</button>
      </footer>
    </aside>
  </div>
</template>

<style scoped>
.drawer-mask { position: fixed; inset: 0; background: var(--mask); display: flex; justify-content: flex-end; z-index: 40; }
.drawer { width: 480px; background: var(--bg-2); border-left: 1px solid var(--line-2); padding: 16px; display: flex; flex-direction: column; color: var(--fg-1); }
header { display: flex; justify-content: space-between; align-items: center; }
header h3 { margin: 0; }
.tip { margin: 8px 0; }
.list { flex: 1; overflow-y: auto; }
.hit { border: 1px solid var(--line); border-radius: var(--r-md); padding: 8px 10px; margin-bottom: 8px; font-size: 13px; }
.loc { font-weight: bold; }
.rule { color: var(--warn); font-weight: normal; font-size: 12px; }
.masked { color: var(--fg-2); margin-top: 4px; word-break: break-all; }
footer { padding-top: 12px; text-align: right; }
.muted { color: var(--fg-3); font-size: 12px; }
button { cursor: pointer; border: 1px solid var(--line-2); background: var(--bg-3); color: var(--fg-1); border-radius: var(--r-sm); padding: 4px 10px; }
button.primary { background: var(--accent); border-color: var(--accent); color: var(--fg-on-accent); font-weight: 600; }
.link { border: none; background: none; color: var(--fg-2); display: inline-flex; align-items: center; }
</style>
