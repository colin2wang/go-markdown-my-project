<!-- 编辑项目对话框：左表单 + 右 YAML 实时镜像（设计文档 2026-10-07-edit-dialog.md） -->
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import type { ProjectConfig } from '../../api/bindings';
import { useProjectEditor } from '../../stores/projectEditor';
import { useLogStore } from '../../stores/log';
import EditorSection from './EditorSection.vue';
import PathInput from './PathInput.vue';
import TagListInput from './TagListInput.vue';
import YamlPreview from './YamlPreview.vue';
import ScanPreviewCard from './ScanPreviewCard.vue';

const props = defineProps<{
  configPath?: string;          // 编辑模式：已有配置文件路径
  projectsDir: string;
  initial: ProjectConfig | null; // 编辑模式传入当前配置；新建传 null
}>();
const emit = defineEmits<{ (e: 'close'): void; (e: 'saved', path: string): void }>();

const log = useLogStore();
const ed = useProjectEditor();
const showCloseConfirm = ref(false);
const saveError = ref('');
const saving = ref(false);

// 新建模式：保存路径随项目名联动
const savePath = computed(() => {
  if (props.configPath) return props.configPath;
  const dir = props.projectsDir.replace(/[\\/]+$/, '');
  const name = ed.draft?.project_name?.trim() || 'new';
  return `${dir}/${name}.yml`;
});

onMounted(() => {
  if (props.configPath && props.initial) ed.openForEdit(props.configPath, props.initial);
  else ed.openForCreate(savePath.value);
  log.info(ed.isNew ? '打开新建项目编辑器' : `打开编辑项目：${props.configPath}`);
});

// 锚点导航：点击滚动到左栏对应分区
const leftPane = ref<HTMLElement | null>(null);
function scrollTo(id: string) {
  leftPane.value?.querySelector(`#${id}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

async function onSave() {
  saveError.value = '';
  saving.value = true;
  try {
    ed.updateConfigPath(savePath.value);
    await ed.save();
    log.info(`项目配置已保存：${savePath.value}`);
    emit('saved', savePath.value);
  } catch (e) {
    saveError.value = String(e instanceof Error ? e.message : e);
    log.warn(`保存失败：${saveError.value}`);
  } finally {
    saving.value = false;
  }
}

function onCancel() {
  if (ed.dirty) { showCloseConfirm.value = true; return; }
  emit('close');
}
</script>

<template>
  <div class="modal" @click.self="onCancel" @keydown.esc="onCancel">
    <div class="dialog">
      <!-- 顶栏 -->
      <header class="head">
        <h3>{{ ed.isNew ? '新建项目' : '编辑项目' }}{{ ed.draft?.project_name ? ' · ' + ed.draft.project_name : '' }}</h3>
        <span class="spacer" />
        <button type="button" class="ghost" @click="ed.reset(); log.info('已恢复默认值')">恢复默认</button>
        <button type="button" class="ghost" @click="onCancel">取消</button>
        <button type="button" class="primary" :disabled="saving || ed.errorCount > 0" :title="ed.errorCount ? Object.values(ed.errors)[0] : ''" @click="onSave">
          💾 保存
        </button>
      </header>
      <p v-if="saveError" class="banner error-banner">{{ saveError }}</p>

      <!-- 双栏 -->
      <div class="body">
        <!-- 左栏：可视化编辑 -->
        <div ref="leftPane" class="left">
          <!-- 锚点 -->
          <nav class="anchors">
            <a @click="scrollTo('sec-basic')">基本</a>
            <a @click="scrollTo('sec-scope')">范围</a>
            <a @click="scrollTo('sec-exclude')">排除</a>
            <a @click="scrollTo('sec-export')">导出</a>
            <a @click="scrollTo('sec-redaction')">敏感</a>
            <a @click="scrollTo('sec-preview')">预检</a>
          </nav>

          <template v-if="ed.draft">
            <EditorSection id="sec-basic" icon="📦" title="基本信息" :error-count="['project_name', 'project_path', 'output_file'].filter((f) => ed.errors[f]).length">
              <label>项目名称
                <input :value="ed.draft.project_name" :class="{ invalid: !!ed.errors.project_name }" @input="ed.draft!.project_name = ($event.target as HTMLInputElement).value" />
              </label>
              <p v-if="ed.errors.project_name" class="field-error">{{ ed.errors.project_name }}</p>
              <label>项目路径</label>
              <PathInput v-model="ed.draft!.project_path" placeholder="F:/path/to/project" :error="ed.errors.project_path" />
              <label>输出文件
                <input :value="ed.draft.output_file" :class="{ invalid: !!ed.errors.output_file }" @input="ed.draft!.output_file = ($event.target as HTMLInputElement).value" />
              </label>
              <p v-if="ed.errors.output_file" class="field-error">{{ ed.errors.output_file }}</p>
              <label class="inline">文档语言
                <label class="inline"><input v-model="ed.draft!.markdown_lang" type="radio" value="zh_cn" /> 中文</label>
                <label class="inline"><input v-model="ed.draft!.markdown_lang" type="radio" value="en_us" /> English</label>
              </label>
            </EditorSection>

            <EditorSection id="sec-scope" icon="🔍" title="扫描范围" hint="目录将递归收集全部文件">
              <label>包含文件</label>
              <TagListInput v-model="ed.draft!.files!" placeholder="回车添加，如 cargo.toml" />
              <label>包含目录</label>
              <TagListInput v-model="ed.draft!.directories!" placeholder="回车添加，如 src" dir-select :base-path="ed.draft!.project_path" />
            </EditorSection>

            <EditorSection id="sec-exclude" icon="🚫" title="排除规则" :error-count="ed.errors.exclude_patterns ? 1 : 0">
              <label>排除目录</label>
              <TagListInput v-model="ed.draft!.exclude_directories!" placeholder="支持 **/name" />
              <label>排除规则</label>
              <TagListInput v-model="ed.draft!.exclude_patterns!" placeholder="支持 *.log" :error="ed.errors.exclude_patterns" />
              <label class="inline">大小上限
                <input :value="ed.draft.max_file_size ?? 0" type="number" class="num" :class="{ invalid: !!ed.errors.max_file_size }" @input="ed.draft!.max_file_size = Number(($event.target as HTMLInputElement).value)" />
                字节（0 = 不限制）
              </label>
            </EditorSection>

            <EditorSection id="sec-export" icon="📤" title="导出设置">
              <label class="inline">导出模式
                <select v-model="ed.draft!.export_mode">
                  <option value="full">完整代码</option>
                  <option value="files">文件名清单</option>
                  <option value="symbols">符号目录</option>
                  <option value="signatures">签名导出</option>
                  <option value="custom">自定义模板</option>
                </select>
              </label>
              <label class="inline">Token 分片
                <input :value="ed.draft.split_tokens ?? 0" type="number" class="num" :class="{ invalid: !!ed.errors.split_tokens }" @input="ed.draft!.split_tokens = Number(($event.target as HTMLInputElement).value)" />
                （0 = 不分片）
              </label>
            </EditorSection>

            <EditorSection
              id="sec-redaction" icon="🛡" title="敏感过滤"
              :muted="!ed.draft.redaction?.enabled"
              :error-count="ed.errors.custom_patterns ? 1 : 0"
            >
              <label class="inline">
                <input v-model="ed.draft!.redaction!.enabled" type="checkbox" />
                导出时剔除密码/密钥
              </label>
              <template v-if="ed.draft.redaction?.enabled">
                <label class="inline">策略
                  <select v-model="ed.draft!.redaction!.strategy">
                    <option value="placeholder">占位符替换</option>
                    <option value="drop_line">删除整行</option>
                  </select>
                </label>
                <label v-if="ed.draft.redaction.strategy === 'placeholder'">替换为
                  <input :value="ed.draft.redaction.placeholder" @input="ed.draft!.redaction!.placeholder = ($event.target as HTMLInputElement).value" />
                </label>
                <label>自定义正则规则</label>
                <TagListInput v-model="ed.draft!.redaction!.custom_patterns!" placeholder="每条即时校验，如 api[_-]?key" :error="ed.errors.custom_patterns" />
              </template>
            </EditorSection>

            <EditorSection id="sec-preview" icon="📊" title="预检">
              <button type="button" class="ghost" @click="ed.runPreview(); log.info('执行扫描预检')">▶ 扫描预览</button>
              <ScanPreviewCard v-if="ed.preview" :preview="ed.preview" />
            </EditorSection>
          </template>
        </div>

        <!-- 右栏：YAML 实时镜像（只读） -->
        <div class="right">
          <YamlPreview :yaml="ed.yamlText" :changed="ed.changedLines" :error-count="ed.errorCount" :syncing="ed.syncing" />
        </div>
      </div>

      <!-- 关闭守卫 -->
      <div v-if="showCloseConfirm" class="modal sub" @click.self="showCloseConfirm = false">
        <div class="confirm">
          <h4>有未保存的修改</h4>
          <p>关闭将丢失本次编辑内容。</p>
          <div class="actions">
            <button type="button" class="danger" @click="showCloseConfirm = false; emit('close')">放弃修改</button>
            <button type="button" @click="showCloseConfirm = false">继续编辑</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal { position: fixed; inset: 0; background: rgba(0,0,0,.6); display: flex; align-items: center; justify-content: center; z-index: 40; }
.dialog {
  width: min(1200px, 94vw); height: 82vh;
  display: flex; flex-direction: column;
  background: #16222e; border: 1px solid #3a4a5c; border-radius: 10px;
  padding: 14px 16px;
}
.head { flex: none; display: flex; align-items: center; gap: 8px; margin-bottom: 10px; }
.head h3 { margin: 0; font-size: 15px; }
.spacer { flex: 1; }
button { cursor: pointer; border: 1px solid #3a4a5c; background: #1f2d3d; color: #eee; border-radius: 6px; padding: 5px 12px; font-size: 13px; }
button.primary { background: #2b6cb0; }
button.ghost { background: none; }
button:disabled { opacity: .5; cursor: not-allowed; }
button.danger { background: #742a2a; }
.banner { flex: none; margin: 0 0 8px; border-radius: 6px; padding: 6px 10px; font-size: 12px; }
.error-banner { color: #ff8383; background: #3a1a1a; border: 1px solid #742a2a; }
.body { flex: 1; min-height: 0; display: flex; gap: 12px; }
.left {
  flex: 1.2; min-height: 0; overflow-y: auto; padding-right: 4px;
}
.right { flex: 1; min-width: 0; }
.anchors {
  position: sticky; top: 0; z-index: 2;
  display: flex; gap: 4px; margin-bottom: 10px; padding: 4px 0;
  background: #16222e;
}
.anchors a { color: #63b3ed; font-size: 12px; cursor: pointer; padding: 2px 8px; border-radius: 4px; }
.anchors a:hover { background: #1f2d3d; }
input.invalid { border-color: #c53030 !important; }
.field-error { color: #ff8383; font-size: 11px; margin: 2px 0 6px 2px; }
.num { width: 110px; }
.modal.sub { z-index: 50; }
.confirm { background: #16222e; border: 1px solid #742a2a; border-radius: 10px; padding: 18px 20px; width: 360px; }
.confirm h4 { margin: 0 0 8px; }
.confirm p { font-size: 12px; color: #8fa3b8; }
.actions { display: flex; gap: 8px; margin-top: 12px; justify-content: flex-end; }
/* 小屏退化：上下堆叠 */
@media (max-width: 1100px) {
  .body { flex-direction: column; }
  .right { flex: none; height: 40%; }
}
</style>
