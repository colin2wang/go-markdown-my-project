<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { useLogStore } from './stores/log';
import { useProjectsStore } from './stores';
import LogPanel from './components/LogPanel.vue';
import { useI18n, type LocaleCode } from './i18n';
import { api } from './api/bindings';
import GlobalSettingsDialog from './components/settings/GlobalSettingsDialog.vue';
import Icon from './components/ui/Icon.vue';

const route = useRoute();
const log = useLogStore();
const projects = useProjectsStore();
const { t, locale, setLocale, locales } = useI18n();
// 顶栏显示当前语言的原生名称（English / 简体中文 / 日本語），不随界面语言翻译
const currentLocaleLabel = computed(
  () => locales.find((l) => l.code === locale.value)?.label ?? locale.value
);

// 全局设置对话框
const settingsOpen = ref(false);

// 构建版本戳（build.ps1 编译期注入本地时区 build-YYYYMMDD-HHmm，dev 下为 dev）
const buildVersion = ref('');
onMounted(async () => {
  try { buildVersion.value = await api.getVersion(); } catch { buildVersion.value = ''; }
});

// 帮助菜单：null = 关闭；'config' = 配置文件说明；'configSample' = 项目配置文件说明；'about' = 程序说明
const helpOpen = ref(false);
const helpPage = ref<'config' | 'configSample' | 'about' | null>(null);
// 语言菜单：地球图标触发，与帮助菜单互斥
const langOpen = ref(false);

function showHelp(page: 'config' | 'configSample' | 'about') {
  helpOpen.value = false;
  helpPage.value = page;
}

function pickLocale(code: LocaleCode) {
  setLocale(code);
  langOpen.value = false;
  void api.setLocale(code);
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
          <rect x="3.5" y="2.5" width="17" height="19" rx="3"
                fill="var(--accent)" opacity=".14" />
          <rect x="3.5" y="2.5" width="17" height="19" rx="3"
                fill="none" stroke="var(--accent)" stroke-width="1.4" />
          <path d="M7.5 8h9M7.5 12h9M7.5 16h6"
                stroke="var(--accent-strong)" stroke-width="1.5" stroke-linecap="round" />
        </svg>
        <span class="brand-text">{{ t('app.brand') }}</span>
        <span v-if="buildVersion" class="build-version">{{ buildVersion }}</span>
      </span>
      <!-- 返回入口放在全局导航层：工作台时以面包屑形式显示，点击即回项目列表 -->
      <nav class="crumbs">
        <router-link
          v-if="route.name === 'workbench'"
          to="/"
          class="tab crumb-link"
          :title="t('wb.back')"
        >
          <Icon name="chevron" :size="12" class="crumb-back" /> {{ t('app.projects') }}
        </router-link>
        <span v-else class="tab active">{{ t('app.projects') }}</span>
        <template v-if="route.name === 'workbench'">
          <span class="crumb-sep">/</span>
          <span class="tab active crumb-name" :title="projects.current?.project_name ?? ''">
            <span class="crumb-text">{{ projects.current?.project_name ?? t('app.workbench') }}</span>
          </span>
        </template>
      </nav>
      <span class="spacer" />
      <div class="lang-menu">
        <button
          class="help-btn lang-btn"
          :title="t('app.switchLanguage')"
          :aria-label="t('app.switchLanguage')"
          :aria-expanded="langOpen"
          aria-haspopup="menu"
          @click="langOpen = !langOpen; helpOpen = false"
        >
          <Icon name="globe" :size="14" />
          <span class="lang-current">{{ currentLocaleLabel }}</span>
          <Icon name="chevron" :size="10" />
        </button>
        <div v-if="langOpen" class="help-dropdown lang-dropdown" role="menu">
          <button
            v-for="l in locales"
            :key="l.code"
            :class="{ current: locale === l.code }"
            role="menuitemradio"
            :aria-checked="locale === l.code"
            @click="pickLocale(l.code)"
          >
            <Icon v-if="locale === l.code" name="check" :size="12" />
            <span v-else class="lang-check-space" aria-hidden="true" />
            {{ l.label }}
          </button>
        </div>
      </div>
      <button class="help-btn" @click="settingsOpen = true"><Icon name="gear" /> {{ t('app.settings') }}</button>
      <div class="help-menu">
        <button class="help-btn" @click="helpOpen = !helpOpen; langOpen = false">{{ t('app.help') }} <Icon name="chevron" :size="12" /></button>
        <div v-if="helpOpen" class="help-dropdown">
          <button @click="showHelp('config')">{{ t('app.helpConfig') }}</button>
          <button @click="showHelp('configSample')">{{ t('app.helpConfigSample') }}</button>
          <button @click="showHelp('about')">{{ t('app.helpAbout') }}</button>
        </div>
      </div>
    </header>
    <router-view />

    <!-- 全局 toast：保存成功等短提示（避开底部日志面板，居中偏上） -->
    <Transition name="toast">
      <div v-if="log.toast" :key="log.toast.seq" class="toast" :class="log.toast.kind" role="status">
        {{ log.toast.msg }}
      </div>
    </Transition>
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
  background: var(--bg-1); border-bottom: 1px solid var(--line);
}
.brand { display: flex; align-items: center; gap: 9px; font-weight: 700; color: var(--fg-1); letter-spacing: .1px; }
.brand-logo { width: 20px; height: 20px; display: block; }
.brand-text { font-size: 14px; font-weight: 600; }
.build-version {
  font-size: 11px;
  color: var(--fg-3);
  border: 1px solid var(--line);
  border-radius: var(--r-sm);
  padding: 1px 7px;
  font-variant-numeric: tabular-nums;
  user-select: text;
}
nav { display: flex; align-items: center; gap: 6px; min-width: 0; }
.tab {
  color: var(--fg-2); text-decoration: none; position: relative;
  padding: 5px 10px; border-radius: var(--r-sm); font-size: 13px; transition: color .12s, background .12s;
}
.tab:hover { color: var(--fg-1); background: var(--bg-2); }
.tab.active { color: var(--fg-1); background: transparent; }
.tab.active::after {
  content: ''; position: absolute; left: 10px; right: 10px;
  bottom: -9px; height: 1.5px;        /* 2px→1.5px，直角，去发光 */
  background: var(--accent);
}
/* 面包屑：左侧 chevron 即返回按钮，与项目名同处全局导航区 */
.crumb-link { display: inline-flex; align-items: center; gap: 5px; }
.crumb-back { transform: rotate(90deg); }
.crumb-sep { color: var(--fg-3); font-size: 12px; }
.crumb-name { display: inline-flex; max-width: 220px; }
.crumb-text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.spacer { flex: 1; }
.lang-menu { position: relative; }
.lang-btn { padding: 3px 7px; }
.lang-current { white-space: nowrap; }
.lang-dropdown { min-width: 150px; }
.lang-dropdown button { display: flex; align-items: center; gap: 6px; }
.lang-dropdown button.current { color: var(--fg-1); }
.lang-check-space { width: 12px; flex: none; }
.help-menu { position: relative; }
.help-btn {
  cursor: pointer; background: none; border: 1px solid var(--line-2); color: var(--fg-2);
  border-radius: var(--r-sm); padding: 3px 10px; font-size: 12px;
  gap: 5px; display: inline-flex; align-items: center;
}
.help-btn:hover { color: var(--fg-1); background: var(--bg-3); }
.help-dropdown {
  position: absolute; right: 0; top: 100%; margin-top: 4px; z-index: 30;
  background: var(--bg-2); border: 1px solid var(--line-2);
  border-radius: var(--r-md);
  display: flex; flex-direction: column; min-width: 150px; overflow: hidden;
}
.help-dropdown button {
  cursor: pointer; background: none; border: none; color: var(--fg-2);
  padding: 8px 12px; font-size: 13px; text-align: left; white-space: nowrap;
}
.help-dropdown button:hover { color: var(--fg-1); background: var(--bg-3); }
.modal { position: fixed; inset: 0; background: var(--mask); display: flex; align-items: center; justify-content: center; z-index: 20; }
.modal-body {
  background: var(--bg-2);
  border: 1px solid var(--line-2);
  border-radius: var(--r-lg); padding: 20px; width: 560px; max-width: 90vw;
  box-shadow: var(--shadow-modal); /* modal 可保留一个功能性阴影 */
}
.modal-body h3 { margin: 0 0 10px; color: var(--fg-1); }
.help-text {
  margin: 0; white-space: pre-wrap; word-break: break-word;
  color: var(--fg-2); line-height: 1.75; font-size: 13px;
  max-height: 60vh; overflow-y: auto;
  font-family: inherit;
}
.actions { display: flex; justify-content: flex-end; margin-top: 12px; }
.modal-body button {
  cursor: pointer; border: 1px solid var(--line-2); background: var(--bg-3);
  color: var(--fg-1); border-radius: var(--r-sm); padding: 4px 14px;
}
</style>

<style>
/* 全局 toast（log store 驱动，App 层渲染）：唯一保留阴影的组件之一 */
.toast {
  position: fixed;
  bottom: 24px;
  left: 50%;
  transform: translateX(-50%);
  background: var(--bg-3);
  border: 1px solid var(--line-2);
  border-left: 2px solid var(--ok);
  border-radius: var(--r-md);
  padding: 8px 16px;
  font-size: 13px;
  color: var(--fg-1);
  box-shadow: var(--shadow-toast);
  z-index: 60;
  max-width: 70vw;
  word-break: break-all;
}
.toast.warn { border-left-color: var(--warn); }
.toast-enter-active, .toast-leave-active { transition: opacity .2s, transform .2s; }
.toast-enter-from, .toast-leave-to { opacity: 0; transform: translate(-50%, 8px); }
</style>
