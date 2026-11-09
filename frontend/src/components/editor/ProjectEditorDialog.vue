<!-- 编辑项目对话框：左表单 + 右 YAML 实时镜像（设计文档 2026-10-07-edit-dialog.md） -->
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { api, type ExcludeDefaults, type ProjectConfig } from '../../api/bindings';
import { useProjectEditor } from '../../stores/projectEditor';
import { useLogStore } from '../../stores/log';
import { useI18n } from '../../i18n';
import EditorSection from './EditorSection.vue';
import PathInput from './PathInput.vue';
import TagListInput from './TagListInput.vue';
import YamlPreview from './YamlPreview.vue';
import ScanPreviewCard from './ScanPreviewCard.vue';
import { normalizeDirSelection } from './dirPaths';

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

// 全局默认排除（config/exclude_defaults.yml）：只读展示，自动与项目规则合并
const excludeDefaults = ref<ExcludeDefaults | null>(null);

// 新建模式：保存路径随项目名联动
const savePath = computed(() => {
  if (props.configPath) return props.configPath;
  const dir = props.projectsDir.replace(/[\\/]+$/, '');
  const name = ed.draft?.project_name?.trim() || 'new';
  return `${dir}/${name}.yml`;
});

// 输出文件拆分：输入框只负责文件名，扩展名走下拉选择（默认 .md）
const outName = computed(() => {
  const v = ed.draft?.output_file ?? '';
  const i = v.lastIndexOf('.');
  return i > 0 ? v.slice(0, i) : v;
});
const outExt = computed(() => {
  const v = ed.draft?.output_file ?? '';
  const i = v.lastIndexOf('.');
  return i > 0 && i < v.length - 1 ? v.slice(i) : '.md';
});
function setOutName(name: string) {
  if (!ed.draft) return;
  ed.draft.output_file = name.trim() ? name + outExt.value : '';
}
function setOutExt(ext: string) {
  if (!ed.draft) return;
  if (outName.value.trim()) ed.draft.output_file = outName.value + ext;
}

onMounted(() => {
  if (props.configPath && props.initial) ed.openForEdit(props.configPath, props.initial);
  else ed.openForCreate(savePath.value);
  api.defaultExcludes().then((d) => (excludeDefaults.value = d));
  log.info(ed.isNew ? t('editor.logOpenCreate') : t('editor.logOpenEdit', { path: props.configPath }));
});

// 锚点导航：点击滚动到左栏对应分区；滚动时高亮当前所在分区（scroll-spy）
const leftPane = ref<HTMLElement | null>(null);
const activeSec = ref('sec-basic');
const SECTIONS = [
  { id: 'sec-basic', key: 'editor.anchorBasic' },
  { id: 'sec-scope', key: 'editor.anchorScope' },
  { id: 'sec-exclude', key: 'editor.anchorExclude' },
  { id: 'sec-include', key: 'editor.anchorInclude' },
  { id: 'sec-export', key: 'editor.anchorExport' },
  { id: 'sec-redaction', key: 'editor.anchorRedaction' },
  { id: 'sec-preview', key: 'editor.anchorPreview' },
] as const;
function scrollTo(id: string) {
  activeSec.value = id;
  leftPane.value?.querySelector(`#${id}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
}
function onLeftScroll() {
  const pane = leftPane.value;
  if (!pane) return;
  const top = pane.scrollTop;
  let cur: string = SECTIONS[0].id;
  for (const s of SECTIONS) {
    const sec = pane.querySelector(`#${s.id}`) as HTMLElement | null;
    if (sec && sec.offsetTop - pane.offsetTop - 40 <= top) cur = s.id;
  }
  activeSec.value = cur;
}
// 分区是否有校验错误（锚点上显示红点）
const secErrors = computed<Record<string, number>>(() => ({
  'sec-basic': ['project_name', 'project_path', 'output_file'].filter((f) => ed.errors[f]).length,
  'sec-exclude': ed.errors.exclude_patterns ? 1 : 0,
  'sec-include': ed.errors.include_patterns ? 1 : 0,
  'sec-redaction': ed.errors.custom_patterns ? 1 : 0,
}));

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
  // 保存前统一去掉被父目录覆盖的子目录（含历史配置里已有的冗余）
  if (ed.draft) ed.draft.directories = normalizeDirSelection(ed.draft.directories ?? []);
  saving.value = true;
  try {
    ed.updateConfigPath(savePath.value);
    await ed.save();
    log.info(t('editor.logSaved', { path: savePath.value }));
    log.showToast(t('editor.toastSaved', { name: ed.draft?.project_name ?? '' }));
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
        <div ref="leftPane" class="left" @scroll="onLeftScroll">
          <!-- 锚点 -->
          <nav class="anchors">
            <a
              v-for="s in SECTIONS" :key="s.id"
              :class="{ active: activeSec === s.id, 'has-error': (secErrors[s.id] ?? 0) > 0 }"
              @click="scrollTo(s.id)"
            >{{ t(s.key) }}</a>
          </nav>

          <template v-if="ed.draft">
            <EditorSection id="sec-basic" icon="box" :title="t('editor.sectionBasic')" :error-count="['project_name', 'project_path', 'output_file'].filter((f) => ed.errors[f]).length">
              <label class="plain">{{ t('editor.projectName') }}
                <input :value="ed.draft.project_name" :class="{ invalid: !!ed.errors.project_name }" @input="ed.draft!.project_name = ($event.target as HTMLInputElement).value" />
              </label>
              <p v-if="ed.errors.project_name" class="field-error">{{ ed.errors.project_name }}</p>
              <label class="plain">{{ t('editor.projectPath') }}</label>
              <PathInput v-model="ed.draft!.project_path" placeholder="F:/path/to/project" :error="ed.errors.project_path" :validated="onPathValidated" />
              <label class="plain">{{ t('editor.outputFile') }}
                <div class="outrow">
                  <input :value="outName" @input="setOutName(($event.target as HTMLInputElement).value)" />
                  <select :value="outExt" @change="setOutExt(($event.target as HTMLSelectElement).value)">
                    <option value=".md">.md</option>
                    <option value=".markdown">.markdown</option>
                    <option value=".txt">.txt</option>
                  </select>
                </div>
              </label>
              <p v-if="ed.errors.output_file" class="field-error">{{ ed.errors.output_file }}</p>
              <label class="inline">{{ t('editor.docLang') }}
                <label class="inline"><input v-model="ed.draft!.markdown_lang" type="radio" value="zh_cn" /> {{ t('editor.langZh') }}</label>
                <label class="inline"><input v-model="ed.draft!.markdown_lang" type="radio" value="en_us" /> {{ t('editor.langEn') }}</label>
              </label>
            </EditorSection>

            <EditorSection id="sec-scope" icon="search" :title="t('editor.sectionScope')" :hint="t('editor.hintScope')">
              <label class="plain">{{ t('editor.includeFiles') }}</label>
              <TagListInput v-model="ed.draft!.files!" :placeholder="t('editor.phFiles')" file-select :base-path="ed.draft!.project_path" />
              <label class="plain">{{ t('editor.includeDirs') }}</label>
              <TagListInput v-model="ed.draft!.directories!" :placeholder="t('editor.phDirs')" dir-select tree-select dir-hierarchy :base-path="ed.draft!.project_path" />
            </EditorSection>

            <EditorSection id="sec-exclude" icon="ban" :title="t('editor.sectionExclude')" :error-count="ed.errors.exclude_patterns ? 1 : 0">
              <label class="plain">{{ t('editor.excludeDirs') }}</label>
              <TagListInput v-model="ed.draft!.exclude_directories!" :placeholder="t('editor.phExcludeDirs')" />
              <label class="plain">{{ t('editor.excludePatterns') }}</label>
              <TagListInput v-model="ed.draft!.exclude_patterns!" :placeholder="t('editor.phExcludePatterns')" :error="ed.errors.exclude_patterns" />
              <label class="inline">{{ t('editor.maxSize') }}
                <input :value="ed.draft.max_file_size ?? 0" type="number" class="num" :class="{ invalid: !!ed.errors.max_file_size }" @input="ed.draft!.max_file_size = Number(($event.target as HTMLInputElement).value)" />
                {{ t('editor.maxSizeSuffix') }}
              </label>
              <label class="inline">
                <input type="checkbox" :checked="ed.draft.use_default_excludes !== false" @change="ed.draft!.use_default_excludes = ($event.target as HTMLInputElement).checked" />
                {{ t('editor.useDefaultExcludes') }}
              </label>
              <label class="inline">
                <input type="checkbox" :checked="ed.draft.skip_binary_files !== false" @change="ed.draft!.skip_binary_files = ($event.target as HTMLInputElement).checked" />
                {{ t('editor.skipBinary') }}
              </label>
              <label class="inline">
                <input type="checkbox" :checked="ed.draft.use_gitignore !== false" @change="ed.draft!.use_gitignore = ($event.target as HTMLInputElement).checked" />
                {{ t('editor.useGitIgnore') }}
              </label>
              <template v-if="ed.draft.use_default_excludes !== false && excludeDefaults">
                <label>{{ t('editor.defaultExcludes') }}</label>
                <div class="default-excludes">
                  <span v-for="d in excludeDefaults.directories" :key="'d-' + d" class="default-tag dir">{{ d }}</span>
                  <span v-for="p in excludeDefaults.files" :key="'f-' + p" class="default-tag">{{ p }}</span>
                </div>
                <p class="default-hint">{{ t('editor.defaultExcludesHint') }}</p>
              </template>
            </EditorSection>

            <EditorSection id="sec-include" icon="inbox" :title="t('editor.sectionInclude')" :error-count="ed.errors.include_patterns ? 1 : 0">
              <label class="inline"><input v-model="ed.includeEnabled" type="checkbox" /> {{ t('editor.includeEnabled') }}</label>
              <template v-if="ed.includeEnabled">
                <label class="plain">{{ t('editor.includePatterns') }}</label>
                <TagListInput v-model="ed.draft!.include_patterns!" :placeholder="t('editor.phIncludePatterns')" :error="ed.errors.include_patterns" ext-scan :base-path="ed.draft!.project_path" :scan-dirs="ed.draft!.directories" />
              </template>
            </EditorSection>

            <EditorSection id="sec-export" icon="outbox" :title="t('editor.sectionExport')">
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
              id="sec-redaction" icon="shield" :title="t('editor.sectionRedaction')"
              :muted="!ed.draft.redaction?.enabled"
              :error-count="ed.errors.custom_patterns ? 1 : 0"
            >
              <label class="inline">
                <input v-model="ed.draft!.redaction!.enabled" type="checkbox" />
                {{ t('editor.redactEnable') }}
              </label>
              <template v-if="ed.draft.redaction?.enabled">
                <label class="inline">{{ t('editor.ruleProfile') }}
                  <select v-model="ed.draft!.redaction!.rule_profile">
                    <option value="strict">{{ t('editor.profileStrict') }}</option>
                    <option value="standard">{{ t('editor.profileStandard') }}</option>
                    <option value="relaxed">{{ t('editor.profileRelaxed') }}</option>
                  </select>
                </label>
                <label class="inline">{{ t('editor.strategy') }}
                  <select v-model="ed.draft!.redaction!.strategy">
                    <option value="placeholder">{{ t('wb.strategyPlaceholder') }}</option>
                    <option value="drop_line">{{ t('editor.strategyDropLine') }}</option>
                  </select>
                </label>
                <label v-if="ed.draft.redaction.strategy === 'placeholder'">{{ t('editor.replaceWith') }}
                  <input :value="ed.draft.redaction.placeholder" @input="ed.draft!.redaction!.placeholder = ($event.target as HTMLInputElement).value" />
                </label>
                <label class="plain">{{ t('editor.customPatterns') }}</label>
                <TagListInput v-model="ed.draft!.redaction!.custom_patterns!" :placeholder="t('editor.phPatterns')" :error="ed.errors.custom_patterns" />
              </template>
            </EditorSection>

            <EditorSection id="sec-preview" icon="chart" :title="t('editor.sectionPreview')">
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
.modal { position: fixed; inset: 0; background: var(--mask); display: flex; align-items: center; justify-content: center; z-index: 40; }
.dialog {
  width: min(1200px, 94vw); height: 82vh;
  display: flex; flex-direction: column;
  background: var(--bg-2); border: 1px solid var(--line-2); border-radius: var(--r-lg);
  padding: 14px 16px;
}
.head { flex: none; display: flex; align-items: center; gap: 8px; margin-bottom: 10px; }
.head h3 { margin: 0; font-size: 15px; }
.spacer { flex: 1; }
button { cursor: pointer; border: 1px solid var(--line-2); background: transparent; color: var(--fg-1); border-radius: var(--r-sm); padding: 5px 12px; font-size: 13px; }
button.primary { background: var(--accent); border-color: var(--accent); color: var(--fg-on-accent); font-weight: 600; }
button.ghost { background: none; }
button:disabled { opacity: .5; cursor: not-allowed; }
button.danger { background: transparent; border-color: var(--err); color: var(--err); }
.banner { flex: none; margin: 0 0 8px; border-radius: var(--r-sm); padding: 6px 10px; font-size: 12px; }
.error-banner { color: var(--err); background: transparent; border: 1px solid var(--err); }
.body { flex: 1; min-height: 0; display: flex; gap: 12px; }
.left {
  flex: 1.2; min-height: 0; overflow-y: auto; padding-right: 4px;
}
.right { flex: 1; min-width: 0; }
.anchors {
  position: sticky; top: 0; z-index: 2;
  display: flex; gap: 4px; margin-bottom: 10px; padding: 6px 0;
  background: var(--bg-2);
  border-bottom: 1px solid var(--line);
}
.anchors a { position: relative; color: var(--fg-2); font-size: 12px; cursor: pointer; padding: 3px 9px; border-radius: var(--r-sm); }
.anchors a:hover { color: var(--fg-1); background: var(--bg-3); }
.anchors a.active { color: var(--accent); background: var(--accent-dim); }
.anchors a.has-error::after {
  content: '';
  position: absolute; top: 3px; right: 2px;
  width: 5px; height: 5px; border-radius: 50%;
  background: var(--err);
}
input.invalid { border-color: var(--err) !important; }
.field-error { color: var(--err); font-size: 11px; margin: 2px 0 6px 2px; }
/* 表单栅格：块级 label 的标签列定宽右对齐，输入框起点对齐 */
.left label {
  display: grid;
  grid-template-columns: 96px 1fr;
  gap: 6px 12px;
  align-items: center;
  font-size: 12.5px;
  color: var(--fg-2);
  text-align: right;
  margin: 6px 0;
}
.left label.inline { display: flex; text-align: left; }
/* 独立标签（不含输入控件）顶格左对齐，不进 96px 标签列栅格 */
.left label.plain { display: block; text-align: left; }
.left label.plain > input { display: block; width: 100%; box-sizing: border-box; margin-top: 4px; }
/* 输出文件：文件名 + 扩展名下拉 */
.outrow { display: flex; align-items: center; gap: 6px; margin-top: 4px; }
.outrow input { flex: 1; min-width: 0; }
.outrow select { flex: none; width: 110px; }
.num { width: 110px; }
.default-excludes { display: flex; flex-wrap: wrap; gap: 4px; }
.default-tag {
  font-size: 11px; padding: 2px 7px; border-radius: var(--r-sm);
  background: transparent; border: 1px solid var(--line-2); color: var(--fg-2);
}
.default-tag.dir { border-style: dashed; }
.default-hint { font-size: 11px; color: var(--fg-3); margin: 4px 0 0; }
.modal.sub { z-index: 50; }
.confirm { background: var(--bg-2); border: 1px solid var(--line-2); border-radius: var(--r-lg); padding: 18px 20px; width: 360px; }
.confirm h4 { margin: 0 0 8px; }
.confirm p { font-size: 12px; color: var(--fg-2); }
.actions { display: flex; gap: 8px; margin-top: 12px; justify-content: flex-end; }
/* 小屏退化：上下堆叠 */
@media (max-width: 1100px) {
  .body { flex-direction: column; }
  .right { flex: none; height: 40%; }
}
</style>
