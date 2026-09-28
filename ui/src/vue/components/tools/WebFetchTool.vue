<template>
  <div class="tool" :data-testid="isRunning ? 'tool-call-running' : 'tool-call-completed'">
    <div class="tool-header tool-header--static">
      <div class="tool-summary">
        <span class="tool-emoji" :class="{ running: isRunning }">🌐</span>
        <span class="tool-command">Fetch: <a v-if="pageURL" :href="pageURL" target="_blank" rel="noopener noreferrer">{{ pageTitle || pageURL }}</a><span v-else>{{ safeInputURL }}</span></span>
        <span v-if="text" class="tool-success">{{ textCharacterCount.toLocaleString() }} characters<span v-if="truncated"> (truncated)</span></span>
      </div>
    </div>
    <pre v-if="text" class="web-fetch-text">{{ text }}</pre>
    <pre v-else-if="hasError" class="tool-error">Fetch failed</pre>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { LLMContent } from "../../../types";
import { safeWebURL } from "../../../utils/safeWebURL";

const props = defineProps<{ toolInput?: unknown; toolResult?: LLMContent[]; isRunning?: boolean; hasError?: boolean }>();
const inputURL = computed(() => (props.toolInput as { url?: string } | undefined)?.url || "");
const safeInputURL = computed(() => safeWebURL(inputURL.value));
const result = computed(() => {
  const raw = props.toolResult?.find((item) => item.Type === 2)?.Text;
  if (!raw) return undefined;
  try { return JSON.parse(raw) as { url?: string; title?: string; text?: string; truncated?: boolean }; } catch { return undefined; }
});
const pageURL = computed(() => safeWebURL(result.value?.url || ""));
const pageTitle = computed(() => result.value?.title || "");
const text = computed(() => result.value?.text || "");
const textCharacterCount = computed(() => [...text.value].length);
const truncated = computed(() => result.value?.truncated === true);

</script>

<style scoped>
.web-fetch-text { max-height: 24rem; overflow: auto; white-space: pre-wrap; overflow-wrap: anywhere; margin: 0.5rem 0 0; }
</style>
