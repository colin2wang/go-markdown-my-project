<!-- PathInput：Input + 浏览 + 异步校验徽标三态（设计文档 §2.2） -->
<script setup lang="ts">
import { ref, watch } from 'vue';
import { api } from '../../api/bindings';

const props = defineProps<{ modelValue: string; placeholder?: string; error?: string }>();
const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>();

const checkState = ref<'idle' | 'checking' | 'valid' | 'notfound' | 'notdir'>('idle');
let timer: ReturnType<typeof setTimeout> | undefined;

watch(
  () => props.modelValue,
  (v) => {
    clearTimeout(timer);
    if (!v.trim()) { checkState.value = 'idle'; return; }
    checkState.value = 'checking';
    timer = setTimeout(async () => {
      try {
        const r = await api.pathExists(v);
        if (!r.exists) checkState.value = 'notfound';
        else if (!r.isDir) checkState.value = 'notdir';
        else checkState.value = 'valid';
      } catch {
        checkState.value = 'idle';
      }
    }, 500);
  }
);

async function browse() {
  // 从当前值（若有效）作为初始目录打开，而不是上次选择的目录
  const dir = await api.selectDirectory(props.modelValue.trim());
  if (dir) emit('update:modelValue', dir.replace(/\\/g, '/'));
}

const badgeText: Record<string, string> = {
  idle: '', checking: '校验中…', valid: '● 目录有效', notfound: '● 路径不存在', notdir: '● 不是目录',
};
const badgeClass: Record<string, string> = {
  idle: '', checking: 'gray', valid: 'green', notfound: 'red', notdir: 'red',
};
</script>

<template>
  <div class="pathinput">
    <div class="row">
      <input
        :value="modelValue"
        :placeholder="placeholder"
        :class="{ invalid: !!error || checkState === 'notfound' || checkState === 'notdir' }"
        @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
      />
      <button type="button" @click="browse">📂 浏览…</button>
      <span v-if="checkState !== 'idle'" class="badge" :class="badgeClass[checkState]">{{ badgeText[checkState] }}</span>
    </div>
    <p v-if="error" class="field-error">{{ error }}</p>
  </div>
</template>

<style scoped>
.row { display: flex; align-items: center; gap: 6px; }
.row input { flex: 1; }
button { background: #1f2d3d; color: #eee; border: 1px solid #3a4a5c; border-radius: 4px; padding: 4px 8px; font-size: 12px; cursor: pointer; }
.badge { font-size: 11px; white-space: nowrap; }
.badge.green { color: #68d391; }
.badge.red { color: #ff8383; }
.badge.gray { color: #8fa3b8; }
input.invalid { border-color: #c53030 !important; }
.field-error { color: #ff8383; font-size: 11px; margin: 2px 0 0 2px; }
</style>
