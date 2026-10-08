<!-- 编辑项目对话框：左表单 + 右 YAML 实时镜像（设计文档 2026-10-07-edit-dialog.md） -->
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import type { ProjectConfig } from '../../api/bindings';
import { useProjectEditor } from '../../stores/projectEditor';
import { useLogStore } from '../../stores/log';
import { useI18n } from '../../i18n';
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
const { t } = useI18n();
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
  log.info(ed.isNew ? t('editor.logOpenCreate') : t('editor.logOpenEdit', { path: props.configPath }));
});

// 锚点导航：点击滚动到左栏对应分区
const leftPane = ref<HTMLElement | null>(null);
function scrollTo(id: string) {
  leftPane.value?.querySelector(`#${id}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

// 项目路径校验通过后，名称为空则自动填入路径最后的文件夹名（新建时顺带联动输出文件）
function onPathValidated(r: { exists: boolean; isDir: boolean }) {
  if (!ed.draft || !r.exists || !r.isDir) return;
  if (ed.draft.project_name.trim()) return;
  const name = ed.draft.project_path.trim().split(/[\\/]/).filter(Boolean).pop() ?? '';
  if (!name) return;
  ed.draft.project_name = name;
  if (!ed.draft.output_file || ed.draft.output_file === 'output.md') {
    ed.draft.output_file = `${name}.md`;
  }
}

async function onSave() {
  saveError.value = '';
  saving.value = true;
  try {
    ed.updateConfigPath(savePath.value);
    await ed.save();
    log.info(t('editor.logSaved', { path: savePath.value }));
    emit('saved', savePath.value);
  } catch (e) {
    saveError.value = String(e instanceof Error ? e.message : e);
    log.warn(t('editor.logSaveFail', { err: saveError.value }));
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
        <h3>{{ ed.isNew ? t('editor.newTitle') : t('editor.editTitle') }}{{ ed.draft?.project_name ? ' · ' + ed.draft.project_name : '' }}</h3>
        <span class="spacer" />
        <button type="button" class="ghost" @click="ed.reset(); log.info(t('editor.logRestore'))">{{ t('editor.restore') }}</button>
        <button type="button" class="ghost" @click="onCancel">{{ t('common.cancel') }}</button>
        <button type="button" class="primary" :disabled="saving || ed.errorCount > 0" :title="ed.errorCount ? Object.values(ed.errors)[0] : ''" @click="onSave">
          {{ t('editor.save') }}
        </button>
      </header>
      <p v-if="saveError" class="banner error-banner">{{ saveError }}</p>

      <!-- 双栏 -->
      <div class="body">
        <!-- 左栏：可视化编辑 -->
        <div ref="leftPane" class="left">
          <!-- 锚点 -->
          <nav class="anchors">
            <a @click="scrollTo('sec-basic')">{{ t('editor.anchorBasic') }}</a>
            <a @click="scrollTo('sec-scope')">{{ t('editor.anchorScope') }}</a>
            <a @click="scrollTo('sec-exclude')">{{ t('editor.anchorExclude') }}</a>
            <a @click="scrollTo('sec-include')">{{ t('editor.anchorInclude') }}</a>
            <a @click="scrollTo('sec-export')">{{ t('editor.anchorExport') }}</a>
            <a @click="scrollTo('sec-redaction')">{{ t('editor.anchorRedaction') }}</a>
            <a @click="scrollTo('sec-preview')">{{ t('editor.anchorPreview') }}</a>
          </nav>

          <template v-if="ed.draft">
            <EditorSection id="sec-basic" icon="📦" :title="t('editor.sectionBasic')" :error-count="['project_name', 'project_path', 'output_file'].filter((f) => ed.errors[f]).length">
              <label>{{ t('editor.projectName') }}
                <input :value="ed.draft.project_name" :class="{ invalid: !!ed.errors.project_name }" @input="ed.draft!.project_name = ($event.target as HTMLInputElement).value" />
              </label>
              <p v-if="ed.errors.project_name" class="field-error">{{ ed.errors.project_name }}</p>
              <label>{{ t('editor.projectPath') }}</label>
              <PathInput v-model="ed.draft!.project_path" placeholder="F:/path/to/project" :error="ed.errors.project_path" :validated="onPathValidated" />
              <label>{{ t('editor.outputFile') }}
                <input :value="ed.draft.output_file" :class="{ invalid: !!ed.errors.output_file }" @input="ed.draft!.output_file = ($event.target as HTMLInputElement).value" />
              </label>
              <p v-if="ed.errors.output_file" class="field-error">{{ ed.errors.output_file }}</p>
              <label class="inline">{{ t('editor.docLang') }}
                <label class="inline"><input v-model="ed.draft!.markdown_lang" type="radio" value="zh_cn" /> {{ t('editor.langZh') }}</label>
                <label class="inline"><input v-model="ed.draft!.markdown_lang" type="radio" value="en_us" /> {{ t('editor.langEn') }}</label>
              </label>
            </EditorSection>

            <EditorSection id="sec-scope" icon="🔍" :title="t('editor.sectionScope')" :hint="t('editor.hintScope')">
              <label>{{ t('editor.includeFiles') }}</label>
              <TagListInput v-model="ed.draft!.files!" :placeholder="t('editor.phFiles')" />
              <label>{{ t('editor.includeDirs') }}</label>
              <TagListInput v-model="ed.draft!.directories!" :placeholder="t('editor.phDirs')" dir-select :base-path="ed.draft!.project_path" />
            </EditorSection>

            <EditorSection id="sec-exclude" icon="🚫" :title="t('editor.sectionExclude')" :error-count="ed.errors.exclude_patterns ? 1 : 0">
              <label>{{ t('editor.excludeDirs') }}</label>
              <TagListInput v-model="ed.draft!.exclude_directories!" :placeholder="t('editor.phExcludeDirs')" />
              <label>{{ t('editor.excludePatterns') }}</label>
              <TagListInput v-model="ed.draft!.exclude_patterns!" :placeholder="t('editor.phExcludePatterns')" :error="ed.errors.exclude_patterns" />
              <label class="inline">{{ t('editor.maxSize') }}
                <input :value="ed.draft.max_file_size ?? 0" type="number" class="num" :class="{ invalid: !!ed.errors.max_file_size }" @input="ed.draft!.max_file_size = Number(($event.target as HTMLInputElement).value)" />
                {{ t('editor.maxSizeSuffix') }}
              </label>
            </EditorSection>

            <EditorSection id="sec-include" icon="📥" :title="t('editor.sectionInclude')" :error-count="ed.errors.include_patterns ? 1 : 0">
              <label class="inline"><input v-model="ed.includeEnabled" type="checkbox" /> {{ t('editor.includeEnabled') }}</label>
              <template v-if="ed.includeEnabled">
                <label>{{ t('editor.includePatterns') }}</label>
                <TagListInput v-model="ed.draft!.include_patterns!" :placeholder="t('editor.phIncludePatterns')" :error="ed.errors.include_patterns" />
              </template>
            </EditorSection>

            <EditorSection id="sec-export" icon="📤" :title="t('editor.sectionExport')">
              <label class="inline">{{ t('editor.exportMode') }}
                <select v-model="ed.draft!.export_mode">
                  <option value="full">{{ t('wb.modes.full.label') }}</option>
                  <option value="files">{{ t('wb.modes.files.label') }}</option>
                  <option value="symbols">{{ t('wb.modes.symbols.label') }}</option>
                  <option value="signatures">{{ t('wb.modes.signatures.label') }}</option>
                  <option value="custom">{{ t('editor.exportModeCustom') }}</option>
                </select>
              </label>
              <label class="inline">{{ t('editor.splitTokens') }}
                <input :value="ed.draft.split_tokens ?? 0" type="number" class="num" :class="{ invalid: !!ed.errors.split_tokens }" @input="ed.draft!.split_tokens = Number(($event.target as HTMLInputElement).value)" />
                {{ t('editor.splitHint') }}
              </label>
            </EditorSection>

            <EditorSection
              id="sec-redaction" icon="🛡" :title="t('editor.sectionRedaction')"
              :muted="!ed.draft.redaction?.enabled"
              :error-count="ed.errors.custom_patterns ? 1 : 0"
            >
              <label class="inline">
                <input v-model="ed.draft!.redaction!.enabled" type="checkbox" />
                {{ t('editor.redactEnable') }}
              </label>
              <template v-if="ed.draft.redaction?.enabled">
                <label class="inline">{{ t('editor.strategy') }}
                  <select v-model="ed.draft!.redaction!.strategy">
                    <option value="placeholder">{{ t('wb.strategyPlaceholder') }}</option>
                    <option value="drop_line">{{ t('editor.strategyDropLine') }}</option>
                  </select>
                </label>
                <label v-if="ed.draft.redaction.strategy === 'placeholder'">{{ t('editor.replaceWith') }}
                  <input :value="ed.draft.redaction.placeholder" @input="ed.draft!.redaction!.placeholder = ($event.target as HTMLInputElement).value" />
                </label>
                <label>{{ t('editor.customPatterns') }}</label>
                <TagListInput v-model="ed.draft!.redaction!.custom_patterns!" :placeholder="t('editor.phPatterns')" :error="ed.errors.custom_patterns" />
              </template>
            </EditorSection>

            <EditorSection id="sec-preview" icon="📊" :title="t('editor.sectionPreview')">
              <button type="button" class="ghost" @click="ed.runPreview(); log.info(t('editor.logPreview'))">{{ t('editor.runPreview') }}</button>
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
          <h4>{{ t('editor.unsavedTitle') }}</h4>
          <p>{{ t('editor.unsavedBody') }}</p>
          <div class="actions">
            <button type="button" class="danger" @click="showCloseConfirm = false; emit('close')">{{ t('editor.discard') }}</button>
            <button type="button" @click="showCloseConfirm = false">{{ t('editor.keepEditing') }}</button>
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
