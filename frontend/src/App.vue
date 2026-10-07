<script setup lang="ts">
import { onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { useLogStore } from './stores/log';
import LogPanel from './components/LogPanel.vue';

const route = useRoute();
const log = useLogStore();

onMounted(() => log.listen());
</script>

<template>
  <div class="app-shell">
    <header class="topbar">
      <span class="brand">ProjectDocs</span>
      <nav>
        <router-link to="/" class="tab" exact-active-class="active">项目管理</router-link>
        <router-link
          :to="route.name === 'workbench' ? route.fullPath : ''"
          class="tab"
          :class="{ active: route.name === 'workbench', disabled: route.name !== 'workbench' }"
        >
          工作台
        </router-link>
      </nav>
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
</style>
