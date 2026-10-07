<script setup lang="ts">
import type { SensitiveHit } from '../api/bindings';

defineProps<{ hits: SensitiveHit[] }>();
const emit = defineEmits<{ (e: 'close'): void }>();
</script>

<template>
  <div class="drawer-mask" @click.self="emit('close')">
    <aside class="drawer">
      <header>
        <h3>敏感信息命中 ({{ hits.length }})</h3>
        <button class="link" @click="emit('close')">✕</button>
      </header>
      <p class="muted tip">命中仅显示掩码预览，绝不落盘明文。导出时按策略替换。</p>
      <div class="list">
        <p v-if="hits.length === 0" class="muted">无命中</p>
        <div v-for="(h, i) in hits" :key="i" class="hit">
          <div class="loc">{{ h.file }} : {{ h.line }} <span class="rule">[{{ h.rule }}]</span></div>
          <div class="masked">{{ h.masked }}</div>
        </div>
      </div>
      <footer>
        <button class="primary" @click="emit('close')">知道了</button>
      </footer>
    </aside>
  </div>
</template>

<style scoped>
.drawer-mask { position: fixed; inset: 0; background: rgba(0,0,0,.4); display: flex; justify-content: flex-end; z-index: 40; }
.drawer { width: 480px; background: #16222e; border-left: 1px solid #3a4a5c; padding: 16px; display: flex; flex-direction: column; color: #eee; }
header { display: flex; justify-content: space-between; align-items: center; }
header h3 { margin: 0; }
.tip { margin: 8px 0; }
.list { flex: 1; overflow-y: auto; }
.hit { border: 1px solid #3a4a5c; border-radius: 6px; padding: 8px 10px; margin-bottom: 8px; font-size: 13px; }
.loc { font-weight: bold; }
.rule { color: #f6ad55; font-weight: normal; font-size: 12px; }
.masked { color: #8fa3b8; margin-top: 4px; word-break: break-all; }
footer { padding-top: 12px; text-align: right; }
.muted { color: #8fa3b8; font-size: 12px; }
button { cursor: pointer; border: 1px solid #3a4a5c; background: #1f2d3d; color: #eee; border-radius: 6px; padding: 4px 10px; }
button.primary { background: #2b6cb0; }
.link { border: none; background: none; color: #8fa3b8; }
</style>
