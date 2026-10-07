import { createRouter, createWebHashHistory } from 'vue-router';
import ProjectListView from './views/ProjectListView.vue';
import WorkbenchView from './views/WorkbenchView.vue';

// Wails 资源加载不走 server 路径，使用 hash 模式
export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'projects', component: ProjectListView },
    { path: '/workbench/:configPath', name: 'workbench', component: WorkbenchView, props: true },
  ],
});
