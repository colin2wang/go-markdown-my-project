<script setup lang="ts">
// 全局设置对话框：导出编码 + 界面语言，存 config/global_settings.yml（经后端读写）。
import { computed, ref, watch } from 'vue';
import { api } from '../../api/bindings';
import { useI18n, type LocaleCode } from '../../i18n';
import { useLogStore } from '../../stores/log';

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ (e: 'close'): void }>();

const { t, setLocale, locale } = useI18n();
const log = useLogStore();

type OutputEncoding = 'utf8_bom' | 'utf8' | 'gb18030';

// 选项标签用 computed 而非一次性固化，保证语言切换后文案随 locale 响应式更新
const encodings = computed(() => [
  { value: 'utf8_bom' as OutputEncoding, label: t('settings.encUtf8Bom') },
  { value: 'utf8' as OutputEncoding, label: t('settings.encUtf8') },
  { value: 'gb18030' as OutputEncoding, label: t('settings.encGb18030') },
]);
const languages = computed(() => [
  { value: 'auto', label: t('settings.langAuto') },
  { value: 'zh_cn', label: t('settings.langZh') },
  { value: 'en_us', label: t('settings.langEn') },
  { value: 'ja_jp', label: t('settings.langJa') },
]);
const outputEncoding = ref<OutputEncoding>('utf8_bom');
const interfaceLanguage = ref('auto');
const saving = ref(false);
const error = ref('');

const langMap: Record<string, string> = { auto: '', zh_cn: 'zh-CN', en_us: 'en-US', ja_jp: 'ja-JP' };

async function load() {
  try {
    const s = await api.getGlobalSettings();
    if (s) {
      outputEncoding.value = (s.outputEncoding as OutputEncoding) || 'utf8_bom';
      interfaceLanguage.value = s.interfaceLanguage || 'auto';
    }
  } catch (e) {
    error.value = String(e);
  }
}

async function save() {
  saving.value = true;
  error.value = '';
  try {
    await api.saveGlobalSettings({
      outputEncoding: outputEncoding.value,
      interfaceLanguage: interfaceLanguage.value,
    });
    // 界面语言即时生效
    const mapped = langMap[interfaceLanguage.value];
    if (mapped && mapped !== locale.value) {
      setLocale(mapped as LocaleCode);
      void api.setLocale(locale.value);
    }
    log.info(t('settings.logSaved'));
    emit('close');
  } catch (e) {
    error.value = String(e);
  } finally {
    saving.value = false;
  }
}

// 每次打开对话框时重新加载，保证与磁盘上的全局设置一致
watch(
  () => props.open,
  (open) => {
    if (open) {
      error.value = '';
      void load();
    }
  },
);
</script>

<template>
  <div v-if="props.open" class="modal" @click.self="emit('close')">
    <div class="modal-body">
      <h3>{{ t('settings.title') }}</h3>

      <label class="field">
        {{ t('settings.outputEncoding') }}
        <select v-model="outputEncoding">
          <option v-for="e in encodings" :key="e.value" :value="e.value">{{ e.label }}</option>
        </select>
        <p class="hint">{{ t('settings.outputEncodingHint') }}</p>
      </label>

      <label class="field">
        {{ t('settings.interfaceLanguage') }}
        <select v-model="interfaceLanguage">
          <option v-for="l in languages" :key="l.value" :value="l.value">{{ l.label }}</option>
        </select>
        <p class="hint">{{ t('settings.interfaceLanguageHint') }}</p>
      </label>

      <p v-if="error" class="error">{{ error }}</p>

      <div class="actions">
        <button @click="emit('close')">{{ t('common.close') }}</button>
        <button class="primary" :disabled="saving" @click="save">{{ t('common.save') }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal { position: fixed; inset: 0; background: rgba(0,0,0,.5); display: flex; align-items: center; justify-content: center; z-index: 40; }
.modal-body { background: #16222e; border: 1px solid #3a4a5c; border-radius: 10px; padding: 20px; width: 460px; max-width: 90vw; }
.modal-body h3 { margin: 0 0 14px; color: #eee; }
.field { display: block; margin-bottom: 14px; font-size: 13px; color: #c6d2de; }
.field select { display: block; margin-top: 4px; width: 100%; background: #0f1720; color: #eee; border: 1px solid #3a4a5c; border-radius: 4px; padding: 5px 8px; font-size: 13px; }
.hint { margin: 4px 0 0; font-size: 11px; color: #6d8296; }
.error { color: #f56565; font-size: 12px; }
.actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 12px; }
.actions button { cursor: pointer; border: 1px solid #3a4a5c; background: #1f2d3d; color: #eee; border-radius: 6px; padding: 4px 14px; }
.actions button.primary { background: #2b6cb0; border-color: #2b6cb0; }
.actions button:disabled { opacity: .5; cursor: default; }
</style>
