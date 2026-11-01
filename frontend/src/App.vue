<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { useLogStore } from './stores/log';
import LogPanel from './components/LogPanel.vue';
import { useI18n, type LocaleCode } from './i18n';
import { api } from './api/bindings';
import GlobalSettingsDialog from './components/settings/GlobalSettingsDialog.vue';

const route = useRoute();
const log = useLogStore();
const { t, locale, setLocale, locales } = useI18n();

// 全局设置对话框
const settingsOpen = ref(false);

// 帮助菜单：null = 关闭；'config' = 配置文件说明；'configSample' = 项目配置文件说明；'about' = 程序说明
const helpOpen = ref(false);
const helpPage = ref<'config' | 'configSample' | 'about' | null>(null);

function showHelp(page: 'config' | 'configSample' | 'about') {
  helpOpen.value = false;
  helpPage.value = page;
}

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
      <span class="brand">
        <svg class="brand-logo" viewBox="0 0 24 24" aria-hidden="true">
          <defs>
            <linearGradient id="brandGrad" x1="0" y1="0" x2="1" y2="1">
              <stop offset="0" stop-color="#63b3ed" />
              <stop offset="1" stop-color="#2b6cb0" />
            </linearGradient>
          </defs>
          <rect x="3.5" y="2.5" width="17" height="19" rx="3.5" fill="url(#brandGrad)" />
          <path d="M7.5 8h9M7.5 12h9M7.5 16h6" stroke="#fff" stroke-width="1.7" stroke-linecap="round" opacity="0.92" />
        </svg>
        <span class="brand-text">{{ t('app.brand') }}</span>
      </span>
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
      <button class="help-btn" @click="settingsOpen = true">⚙ {{ t('app.settings') }}</button>
      <div class="help-menu">
        <button class="help-btn" @click="helpOpen = !helpOpen">{{ t('app.help') }} ▾</button>
        <div v-if="helpOpen" class="help-dropdown">
          <button @click="showHelp('config')">{{ t('app.helpConfig') }}</button>
          <button @click="showHelp('configSample')">{{ t('app.helpConfigSample') }}</button>
          <button @click="showHelp('about')">{{ t('app.helpAbout') }}</button>
        </div>
      </div>
    </header>
    <router-view />
    <LogPanel />
    <GlobalSettingsDialog :open="settingsOpen" @close="settingsOpen = false" />

    <div v-if="helpPage" class="modal" @click.self="helpPage = null">
      <div class="modal-body">
        <h3>{{ t(`help.${helpPage}Title`) }}</h3>
        <pre class="help-text">{{ t(`help.${helpPage}Body`) }}</pre>
        <div class="actions">
          <button @click="helpPage = null">{{ t('common.close') }}</button>
        </div>
      </div>
    </div>
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
.brand { display: flex; align-items: center; gap: 9px; font-weight: 700; color: #eee; letter-spacing: .2px; }
.brand-logo { width: 22px; height: 22px; display: block; filter: drop-shadow(0 1px 2px rgba(0,0,0,.4)); }
.brand-text { font-size: 15px; }
nav { display: flex; gap: 4px; }
.tab {
  color: #8fa3b8; text-decoration: none; position: relative;
  padding: 6px 12px; border-radius: 6px; font-size: 14px; transition: color .12s, background .12s;
}
.tab:hover { color: #cdd9e5; }
.tab.active { color: #fff; background: rgba(99,179,237,.14); }
.tab.active::after {
  content: ''; position: absolute; left: 12px; right: 12px; bottom: -10px; height: 2px;
  background: var(--accent-2, #63b3ed); border-radius: 2px;
}
.tab.disabled { opacity: .45; cursor: not-allowed; }
.spacer { flex: 1; }
.lang { display: flex; align-items: center; gap: 6px; font-size: 12px; color: #8fa3b8; }
.lang select { background: #0f1720; color: #eee; border: 1px solid #3a4a5c; border-radius: 4px; padding: 3px 6px; font-size: 12px; }
.help-menu { position: relative; }
.help-btn {
  cursor: pointer; background: none; border: 1px solid #3a4a5c; color: #8fa3b8;
  border-radius: 6px; padding: 3px 10px; font-size: 12px;
}
.help-btn:hover { color: #eee; }
.help-dropdown {
  position: absolute; right: 0; top: 100%; margin-top: 4px; z-index: 30;
  background: #16222e; border: 1px solid #3a4a5c; border-radius: 6px;
  display: flex; flex-direction: column; min-width: 140px; overflow: hidden;
}
.help-dropdown button {
  cursor: pointer; background: none; border: none; color: #8fa3b8;
  padding: 8px 12px; font-size: 13px; text-align: left; white-space: nowrap;
}
.help-dropdown button:hover { color: #eee; background: #1f2d3d; }
.modal { position: fixed; inset: 0; background: rgba(0,0,0,.5); display: flex; align-items: center; justify-content: center; z-index: 20; }
.modal-body { background: #16222e; border: 1px solid #3a4a5c; border-radius: 10px; padding: 20px; width: 560px; max-width: 90vw; }
.modal-body h3 { margin: 0 0 10px; color: #eee; }
.help-text {
  margin: 0; white-space: pre-wrap; word-break: break-word;
  color: #c6d2de; font-size: 13px; line-height: 1.7;
  max-height: 60vh; overflow-y: auto;
  font-family: inherit;
}
.actions { display: flex; justify-content: flex-end; margin-top: 12px; }
.modal-body button {
  cursor: pointer; border: 1px solid #3a4a5c; background: #1f2d3d;
  color: #eee; border-radius: 6px; padding: 4px 14px;
}
</style>
