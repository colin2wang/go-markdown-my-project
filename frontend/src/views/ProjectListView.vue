<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useProjectsStore } from '../stores';
import { api } from '../api/bindings';
import type { ProjectConfig } from '../api/bindings';
import ProjectEditorDialog from '../components/editor/ProjectEditorDialog.vue';
import { useLogStore } from '../stores/log';
import { useI18n } from '../i18n';

const router = useRouter();
const projects = useProjectsStore();
const log = useLogStore();
const { t } = useI18n();

// 向导状态：null = 关闭；configPath 为空串 = 新建
const showWizard = ref(false);
const wizardPath = ref('');
const wizardInitial = ref<ProjectConfig | null>(null);
const deleteConfirm = ref<string | null>(null);

// 本会话内的最近导出时间（configPath -> 时间），用于卡片展示
const lastExport = ref<Map<string, string>>(new Map());

onMounted(async () => {
  projects.projectsDir = await api.defaultProjectsDir();
  await projects.refresh();
});

function openCreate() {
  log.info(t('projects.logCreate'));
  wizardInitial.value = null;
  wizardPath.value = '';
  showWizard.value = true;
}

async function openEdit(path: string) {
  log.info(t('projects.logEdit', { path }));
  await projects.open(path);
  wizardInitial.value = { ...projects.current! };
  wizardPath.value = path;
  showWizard.value = true;
}

function enterWorkbench(path: string) {
  log.info(t('projects.logWorkbench', { path }));
  router.push({ name: 'workbench', params: { configPath: path } });
}

async function onWizardSaved(path: string) {
  log.info(t('projects.logSaved', { path }));
  showWizard.value = false;
  await projects.refresh();
  // 新建保存后直接进入工作台
  enterWorkbench(path);
}

async function refreshProjects() {
  log.info(t('projects.logRefresh', { dir: projects.projectsDir }));
  await projects.refresh();
}

function askDelete(path: string) {
  deleteConfirm.value = path;
}

async function confirmDelete() {
  if (!deleteConfirm.value) return;
  log.info(t('projects.logDelete', { path: deleteConfirm.value }));
  await projects.remove(deleteConfirm.value);
  deleteConfirm.value = null;
}
</script>

<template>
  <main class="projects-view">
    <div class="toolbar">
      <label>{{ t('projects.dir') }}:
        <input v-model="projects.projectsDir" placeholder="projects" />
      </label>
      <button @click="refreshProjects">🔄 {{ t('projects.refresh') }}</button>
      <span class="spacer" />
      <button class="primary" @click="openCreate">{{ t('projects.create') }}</button>
    </div>
    <p v-if="projects.error" class="error">{{ projects.error }}</p>
    <p v-else-if="projects.list.length === 0" class="muted">{{ t('projects.empty') }}</p>
    <div class="cards">
      <div v-for="p in projects.list" :key="p.configPath" class="card">
        <h3>📦 {{ p.name }}</h3>
        <p class="muted path">{{ p.path }}</p>
        <p class="muted">{{ t('projects.output', { file: p.outputFile }) }}</p>
        <p class="muted">{{ t('projects.language', { lang: p.markdownLang }) }}</p>
        <p class="muted">{{ lastExport.get(p.configPath) ? t('projects.lastExport', { time: lastExport.get(p.configPath) }) : t('projects.notExported') }}</p>
        <button class="primary enter" @click="enterWorkbench(p.configPath)">{{ t('projects.enter') }}</button>
        <div class="actions">
          <button class="link" @click="openEdit(p.configPath)">{{ t('projects.edit') }}</button>
          <button class="link danger-text" @click="askDelete(p.configPath)">{{ t('projects.delete') }}</button>
        </div>
      </div>
    </div>

    <ProjectEditorDialog
      v-if="showWizard"
      :initial="wizardInitial"
      :config-path="wizardPath || undefined"
      :projects-dir="projects.projectsDir"
      @close="showWizard = false"
      @saved="onWizardSaved"
    />

    <div v-if="deleteConfirm" class="modal" @click.self="deleteConfirm = null">
      <div class="modal-body">
        <h3>{{ t('projects.deleteTitle') }}</h3>
        <p>{{ t('projects.deleteBody', { path: deleteConfirm }) }}</p>
        <div class="actions">
          <button class="danger" @click="confirmDelete">{{ t('common.delete') }}</button>
          <button @click="deleteConfirm = null">{{ t('common.cancel') }}</button>
        </div>
      </div>
    </div>
  </main>
</template>

<style scoped>
.projects-view { flex: 1; min-height: 0; overflow-y: auto; padding: 16px 20px; color: #eee; }
.toolbar { display: flex; gap: 8px; margin-bottom: 16px; align-items: center; }
.toolbar input { width: 360px; }
.spacer { flex: 1; }
.cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 12px; max-width: 1200px; }
.card { border: 1px solid #3a4a5c; border-radius: 8px; padding: 14px; background: #16222e; display: flex; flex-direction: column; }
.card h3 { margin: 0 0 6px; }
.path { word-break: break-all; }
.muted { color: #8fa3b8; font-size: 12px; margin: 2px 0; }
.enter { margin-top: 10px; width: 100%; padding: 8px; }
.actions { display: flex; gap: 6px; margin-top: 8px; }
button { cursor: pointer; border: 1px solid #3a4a5c; background: #1f2d3d; color: #eee; border-radius: 6px; padding: 4px 10px; }
button.primary { background: #2b6cb0; }
button.danger { background: #742a2a; }
button.link { border: none; background: none; color: #8fa3b8; padding: 2px 4px; }
button.link:hover { color: #eee; }
.danger-text:hover { color: #ff8383 !important; }
.error { color: #ff8383; background: #3a1a1a; border: 1px solid #742a2a; border-radius: 6px; padding: 8px 12px; margin: 8px 0; font-size: 13px; }
input { background: #0f1720; color: #eee; border: 1px solid #3a4a5c; border-radius: 4px; padding: 4px 8px; }
.modal { position: fixed; inset: 0; background: rgba(0,0,0,.5); display: flex; align-items: center; justify-content: center; z-index: 20; }
.modal-body { background: #16222e; border: 1px solid #3a4a5c; border-radius: 10px; padding: 20px; width: 420px; }
code { word-break: break-all; }
</style>
