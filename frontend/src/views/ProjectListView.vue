<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useProjectsStore, useExportStore } from '../stores';
import type { ProjectConfig } from '../api/bindings';

const projects = useProjectsStore();
const exp = useExportStore();
const showForm = ref(false);
const form = ref<ProjectConfig>(emptyProject());
const formPath = ref('');

function emptyProject(): ProjectConfig {
  return { project_name: '', project_path: '', output_file: 'output.md', markdown_lang: 'zh_cn', files: [], directories: [] };
}

onMounted(() => projects.refresh());

async function edit(path: string) {
  await projects.open(path);
  form.value = { ...projects.current! };
  formPath.value = path;
  showForm.value = true;
}

function create() {
  form.value = emptyProject();
  formPath.value = projects.projectsDir + '/new.yml';
  showForm.value = true;
}

async function save() {
  await projects.save(formPath.value, form.value);
  showForm.value = false;
}

async function scanAndExport(cfgPath: string) {
  await projects.open(cfgPath);
  await exp.scan(projects.current!);
}

async function doExport() {
  await exp.run(projects.current!);
}
</script>

<template>
  <div class="page">
    <h1>项目列表</h1>
    <div class="toolbar">
      <input v-model="projects.projectsDir" placeholder="projects 目录" />
      <button @click="projects.refresh()">刷新</button>
      <button @click="create">新建项目</button>
    </div>
    <p v-if="projects.error" class="error">{{ projects.error }}</p>
    <p v-else-if="projects.list.length === 0" class="muted">未找到项目配置（检查目录是否正确）</p>
    <div class="cards">
      <div v-for="p in projects.list" :key="p.configPath" class="card">
        <h3>{{ p.name }}</h3>
        <p class="muted">{{ p.path }}</p>
        <p class="muted">输出: {{ p.outputFile }} · {{ p.markdownLang }}</p>
        <div class="actions">
          <button @click="edit(p.configPath)">编辑</button>
          <button @click="scanAndExport(p.configPath)">导出…</button>
          <button class="danger" @click="projects.remove(p.configPath)">删除</button>
        </div>
      </div>
    </div>

    <div v-if="projects.current" class="export-panel">
      <h2>导出 — {{ projects.current.project_name }}</h2>
      <p>已扫描 {{ exp.files.length }} 个文件，勾选 {{ exp.checkedPaths.size }} 个</p>
      <label>模式:
        <select v-model="exp.mode">
          <option value="full">full 完整代码</option>
          <option value="files">files 文件清单</option>
        </select>
      </label>
      <div class="filelist">
        <label v-for="f in exp.files" :key="f.path" class="fileitem">
          <input
            type="checkbox"
            :checked="exp.checkedPaths.has(f.path)"
            @change="($event.target as HTMLInputElement).checked ? exp.checkedPaths.add(f.path) : exp.checkedPaths.delete(f.path)"
          />
          {{ f.path }} <span class="muted">({{ f.language }}, {{ f.sizeBytes }} B)</span>
        </label>
      </div>
      <button class="primary" @click="doExport">执行导出</button>
      <pre v-if="exp.result" class="result">输出: {{ exp.result.outputPaths.join(', ') }}
字符数: {{ exp.result.totalChars }} · 耗时: {{ exp.result.durationMs }}ms</pre>
    </div>

    <div v-if="showForm" class="modal" @click.self="showForm = false">
      <div class="modal-body">
        <h2>项目配置 {{ formPath }}</h2>
        <label>名称 <input v-model="form.project_name" /></label>
        <label>项目路径 <input v-model="form.project_path" /></label>
        <label>输出文件 <input v-model="form.output_file" /></label>
        <label>语言
          <select v-model="form.markdown_lang">
            <option value="zh_cn">zh_cn</option>
            <option value="en_us">en_us</option>
          </select>
        </label>
        <label>包含文件（逗号分隔） <input :value="(form.files || []).join(', ')" @change="form.files = ($event.target as HTMLInputElement).value.split(',').map(s => s.trim()).filter(Boolean)" /></label>
        <label>包含目录（逗号分隔） <input :value="(form.directories || []).join(', ')" @change="form.directories = ($event.target as HTMLInputElement).value.split(',').map(s => s.trim()).filter(Boolean)" /></label>
        <label>排除目录（逗号分隔） <input :value="(form.exclude_directories || []).join(', ')" @change="form.exclude_directories = ($event.target as HTMLInputElement).value.split(',').map(s => s.trim()).filter(Boolean)" /></label>
        <div class="actions">
          <button class="primary" @click="save">保存</button>
          <button @click="showForm = false">取消</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { padding: 16px; color: #eee; }
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
.cards { display: flex; flex-wrap: wrap; gap: 12px; }
.card { border: 1px solid #3a4a5c; border-radius: 8px; padding: 12px; width: 280px; background: #16222e; }
.card h3 { margin: 0 0 4px; }
.muted { color: #8fa3b8; font-size: 12px; }
.error { color: #ff8383; background: #3a1a1a; border: 1px solid #742a2a; border-radius: 6px; padding: 8px 12px; margin: 8px 0; font-size: 13px; }
.actions { display: flex; gap: 6px; margin-top: 8px; }
button { cursor: pointer; border: 1px solid #3a4a5c; background: #1f2d3d; color: #eee; border-radius: 6px; padding: 4px 10px; }
button.primary { background: #2b6cb0; }
button.danger { background: #742a2a; }
.export-panel { margin-top: 24px; border-top: 1px solid #3a4a5c; padding-top: 12px; }
.filelist { max-height: 240px; overflow: auto; border: 1px solid #3a4a5c; border-radius: 6px; padding: 8px; margin: 8px 0; }
.fileitem { display: block; font-size: 13px; }
label { display: block; margin: 6px 0; }
input, select { background: #0f1720; color: #eee; border: 1px solid #3a4a5c; border-radius: 4px; padding: 4px 8px; }
.modal { position: fixed; inset: 0; background: rgba(0,0,0,.5); display: flex; align-items: center; justify-content: center; }
.modal-body { background: #16222e; border: 1px solid #3a4a5c; border-radius: 10px; padding: 20px; width: 460px; }
.modal-body input { width: 100%; box-sizing: border-box; }
.result { background: #0f1720; padding: 10px; border-radius: 6px; }
</style>
