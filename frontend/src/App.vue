<script setup lang="ts">
import { onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { useLogStore } from './stores/log';
import LogPanel from './components/LogPanel.vue';
import { useI18n, type LocaleCode } from './i18n';
import { api } from './api/bindings';

const route = useRoute();
const log = useLogStore();
const { t, locale, setLocale, locales } = useI18n();

function onChangeLocale(e: Event) {
  setLocale((e.target as HTMLSelectElement).value as LocaleCode);
  void api.setLocale(locale.value);
}

onMounted(() => {
  log.listen();
  void api.setLocale(locale.value); // 启动时把持久化语言同步给后端
});
</script>

<template>
  <div class="app-shell">
    <header class="topbar">
      <span class="brand">{{ t('app.brand') }}</span>
      <nav>
        <router-link to="/" class="tab" exact-active-class="active">{{ t('app.projects') }}</router-link>
        <router-link
          :to="route.name === 'workbench' ? route.fullPath : ''"
          class="tab"
          :class="{ active: route.name === 'workbench', disabled: route.name !== 'workbench' }"
        >
          {{ t('app.workbench') }}
        </router-link>
      </nav>
      <span class="spacer" />
      <label class="lang">
        {{ t('app.language') }}
        <select :value="locale" @change="onChangeLocale">
          <option v-for="l in locales" :key="l.code" :value="l.code">{{ l.label }}</option>
        </select>
      </label>
    </header>
    <router-view />
    <LogPanel />
  </div>
</template>

<style scoped>
.app-shell { display: flex; flex-direction: column; height: 100vh; }
.topbar {
  height: 44px; flex: none;
  display: flex; align-items: center; gap: 20px;
  padding: 0 16px;
  background: #101a24; border-bottom: 1px solid #3a4a5c;
}
.brand { font-weight: bold; color: #eee; }
nav { display: flex; gap: 4px; }
.tab {
  color: #8fa3b8; text-decoration: none;
  padding: 4px 12px; border-radius: 6px; font-size: 14px;
}
.tab.active { color: #eee; background: #1f2d3d; }
.tab.disabled { opacity: .45; cursor: not-allowed; }
.spacer { flex: 1; }
.lang { display: flex; align-items: center; gap: 6px; font-size: 12px; color: #8fa3b8; }
.lang select { background: #0f1720; color: #eee; border: 1px solid #3a4a5c; border-radius: 4px; padding: 3px 6px; font-size: 12px; }
</style>
