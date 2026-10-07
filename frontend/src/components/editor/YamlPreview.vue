<!-- YamlPreview：只读实时镜像，行号 + 变更行高亮 + 状态芯片 + 复制/换行（设计文档 §3.1/§3.3） -->
<script setup lang="ts">
import { computed, ref } from 'vue';
import { useLogStore } from '../../stores/log';

const props = defineProps<{
  yaml: string;
  changed: Set<number>;
  errorCount: number;
  syncing: boolean;
}>();

const log = useLogStore();
const wrap = ref(false);

const lines = computed(() => props.yaml.split('\n'));

async function copyAll() {
  try {
    await navigator.clipboard.writeText(props.yaml);
    log.info(`已复制 YAML（${lines.value.length} 行）`);
  } catch {
    log.warn('复制 YAML 失败：剪贴板权限不可用');
  }
}
</script>

<template>
  <div class="yamlpreview">
    <div class="toolbar">
      <span class="chip" :class="errorCount ? 'red' : 'green'">
        {{ syncing ? '⟳ 同步中…' : errorCount ? `● ${errorCount} 个错误` : '● 校验通过' }}
      </span>
      <span class="muted">{{ lines.length }} 行</span>
      <span class="spacer" />
      <button type="button" @click="copyAll">复制</button>
      <label class="wrap-toggle"><input v-model="wrap" type="checkbox" /> 换行</label>
    </div>
    <div class="code" :class="{ wrap }">
      <div v-for="(l, i) in lines" :key="i" class="line" :class="{ changed: changed.has(i + 1) }">
        <span class="ln">{{ i + 1 }}</span>
        <span class="txt">{{ l || ' ' }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.yamlpreview { display: flex; flex-direction: column; height: 100%; min-height: 0; }
.toolbar { flex: none; display: flex; align-items: center; gap: 8px; padding: 0 0 8px; font-size: 12px; }
.chip { border-radius: 10px; padding: 2px 10px; font-size: 11px; }
.chip.green { background: #1a3a2a; color: #68d391; }
.chip.red { background: #3a1a1a; color: #ff8383; }
.muted { color: #6b7c8f; }
.spacer { flex: 1; }
button { background: #1f2d3d; color: #eee; border: 1px solid #3a4a5c; border-radius: 4px; padding: 2px 10px; font-size: 12px; cursor: pointer; }
.wrap-toggle { display: flex; align-items: center; gap: 4px; color: #8fa3b8; font-size: 12px; }
.code {
  flex: 1; min-height: 0; overflow: auto;
  background: #0c141c; border: 1px solid #2a3848; border-radius: 6px;
  font-family: ui-monospace, Menlo, Consolas, monospace;
  font-size: 13px; line-height: 1.55;
  padding: 6px 0;
  text-align: left;
}
.line { display: flex; white-space: pre; }
.code.wrap .line { white-space: pre-wrap; word-break: break-all; }
.ln { flex: none; width: 38px; text-align: right; padding-right: 10px; color: #4a5a6c; user-select: none; }
.txt { color: #cdd9e5; }
.line.changed { background: rgba(246, 224, 110, .13); }
</style>
