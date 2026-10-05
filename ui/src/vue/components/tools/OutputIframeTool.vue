<!-- Vue port of components/OutputIframeTool.tsx. Renders HTML output in a
     sandboxed iframe with postMessage-based hosted-library streaming and a
     constrained opt-in microfrontend bridge. jszip is framework-agnostic.

     Preserves: .output-iframe-tool, .output-iframe-tool-header,
     .output-iframe-tool-summary, .output-iframe-tool-emoji,
     .output-iframe-tool-title, .output-iframe-tool-details,
     .output-iframe-tool-toggle, .output-iframe-tool-actions,
     .output-iframe-container, .output-iframe-wrapper,
     .output-iframe-tool-download-btn, .output-iframe-tool-open-btn,
     data-testid tool-call-completed/running, and all other classes/aria. -->
<template>
  <div
    class="output-iframe-tool"
    :data-testid="isComplete ? 'tool-call-completed' : 'tool-call-running'"
  >
    <div class="output-iframe-tool-header" @click="isExpanded = !isExpanded">
      <div class="output-iframe-tool-summary">
        <span class="output-iframe-tool-emoji" :class="{ running: isRunning }">✨</span>
        <span class="output-iframe-tool-title" :title="title">{{ title }}</span>
      </div>
      <div class="output-iframe-tool-actions">
        <template v-if="isComplete && !hasError && html">
          <button
            class="output-iframe-tool-download-btn"
            :aria-label="downloadLabel"
            :title="downloadLabel"
            @click.stop="handleDownload"
          >
            <svg
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
              <polyline points="7 10 12 15 17 10" />
              <line x1="12" y1="15" x2="12" y2="3" />
            </svg>
          </button>
          <button
            v-if="!usesMicrofrontend"
            v-tooltip.top="'Open in new tab'"
            class="output-iframe-tool-open-btn"
            aria-label="Open in new tab"
            @click.stop="handleOpenInNewTab"
          >
            <svg
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
              <polyline points="15 3 21 3 21 9" />
              <line x1="10" y1="14" x2="21" y2="3" />
            </svg>
          </button>
        </template>
        <button
          class="output-iframe-tool-toggle"
          :aria-label="isExpanded ? 'Collapse' : 'Expand'"
          :aria-expanded="isExpanded"
          @click.stop="isExpanded = !isExpanded"
        >
          <ToolChevron :expanded="isExpanded" />
        </button>
      </div>
    </div>

    <div v-if="isExpanded" class="output-iframe-tool-details">
      <RunningToolTime v-if="isRunning" :start-time="toolInvokedAt" />
      <div
        v-if="isComplete && !hasError && htmlWithHeightReporter"
        class="output-iframe-tool-section"
      >
        <div v-if="executionTime" class="output-iframe-tool-label">
          <span>Output:</span>
          <span class="output-iframe-tool-time">{{ executionTime }}</span>
        </div>
        <div class="output-iframe-container">
          <iframe
            ref="iframeRef"
            :srcdoc="htmlWithHeightReporter"
            sandbox="allow-scripts allow-downloads"
            allow="clipboard-write"
            :title="title"
            class="output-iframe-wrapper"
            :style="{ height: iframeHeight + 'px' }"
            @load="handleIframeLoad"
          />
        </div>
      </div>

      <div v-if="isComplete && hasError" class="output-iframe-tool-section">
        <div class="output-iframe-tool-label">
          <span>Error:</span>
          <span v-if="executionTime" class="output-iframe-tool-time">{{ executionTime }}</span>
        </div>
        <pre class="output-iframe-tool-error-message">{{
          toolResult && toolResult[0]?.Text ? toolResult[0].Text : "Failed to display HTML content"
        }}</pre>
      </div>

      <div v-if="isRunning" class="output-iframe-tool-section">
        <div class="output-iframe-tool-label">Preparing HTML output...</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from "vue";
import JSZip from "jszip";
import type { LLMContent } from "../../../types";
import ToolChevron from "./ToolChevron.vue";
import RunningToolTime from "./RunningToolTime.vue";

interface EmbeddedFile {
  name: string;
  path: string;
  content: string;
  type: string;
}

interface MicrofrontendConfig {
  context?: Record<string, unknown>;
  capabilities?: string[];
}

const props = defineProps<{
  toolInput?: unknown;
  isRunning?: boolean;
  toolInvokedAt?: string | null;
  toolResult?: LLMContent[];
  hasError?: boolean;
  executionTime?: string;
  display?: unknown;
}>();

// Script injected into iframe to report its content height
const HEIGHT_REPORTER_SCRIPT = `
<script>
(function() {
  function reportHeight() {
    var height = Math.max(
      document.body.scrollHeight,
      document.body.offsetHeight,
      document.documentElement.scrollHeight,
      document.documentElement.offsetHeight
    );
    window.parent.postMessage({ type: 'iframe-height', height: height }, '*');
  }
  if (document.readyState === 'complete') {
    reportHeight();
  } else {
    window.addEventListener('load', reportHeight);
  }
  setTimeout(reportHeight, 100);
  setTimeout(reportHeight, 500);
  window.addEventListener('resize', reportHeight);
  if (typeof MutationObserver !== 'undefined') {
    var observer = new MutationObserver(reportHeight);
    observer.observe(document.body, { childList: true, subtree: true, attributes: true });
  }
})();
<\/script>
`;

const MIN_HEIGHT = 100;
const MAX_HEIGHT = 600;

// /static/ paths the parent fetches for each named library.
const LIBRARY_PATHS: Record<string, string> = {
  excalidraw: "/static/excalidraw/skill.js",
  vega: "/static/vega/skill.js",
};

// Remove injected scripts/styles from HTML to get the original version for download
function getOriginalHtml(html: string): string {
  let result = html.replace(
    /<script>\s*window\.__FILES__\s*=\s*window\.__FILES__\s*\|\|\s*\{\};[\s\S]*?<\/script>\s*/g,
    "",
  );
  result = result.replace(/<script data-libs-bootstrap="[^"]*">[\s\S]*?<\/script>\s*/g, "");
  result = result.replace(/<style data-file="[^"]*">[\s\S]*?<\/style>\s*/g, "");
  result = result.replace(/<script data-file="[^"]*">[\s\S]*?<\/script>\s*/g, "");
  result = result.replace(/<head>\s*<\/head>\s*/g, "");
  return result;
}

// Escape HTML special characters for safe embedding
function escapeHtml(str: string): string {
  return str
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

// State
const isExpanded = ref(true);
const iframeHeight = ref(300);
const iframeRef = ref<HTMLIFrameElement | null>(null);

// Extract display data
const displayDataComputed = computed(() => {
  // First try display prop (from tool result)
  if (props.display && typeof props.display === "object" && props.display !== null) {
    const d = props.display as {
      html?: string;
      title?: string;
      filename?: string;
      files?: EmbeddedFile[];
      libraries?: string[];
      microfrontend?: MicrofrontendConfig;
    };
    return {
      html: typeof d.html === "string" ? d.html : undefined,
      title: typeof d.title === "string" ? d.title : undefined,
      filename: typeof d.filename === "string" ? d.filename : undefined,
      files: Array.isArray(d.files) ? d.files : undefined,
      libraries: Array.isArray(d.libraries) ? d.libraries : undefined,
      microfrontend:
        d.microfrontend && typeof d.microfrontend === "object" ? d.microfrontend : undefined,
    };
  }
  // Fall back to toolInput
  const input = props.toolInput;
  return {
    html:
      typeof input === "object" &&
      input !== null &&
      "html" in input &&
      typeof (input as { html: unknown }).html === "string"
        ? (input as { html: string }).html
        : undefined,
    title:
      typeof input === "object" &&
      input !== null &&
      "title" in input &&
      typeof (input as { title: unknown }).title === "string"
        ? (input as { title: string }).title
        : undefined,
    filename: undefined as string | undefined,
    files: undefined as EmbeddedFile[] | undefined,
    libraries: undefined as string[] | undefined,
    microfrontend: undefined as MicrofrontendConfig | undefined,
  };
});

const title = computed(() => displayDataComputed.value.title || "HTML Output");
const html = computed(() => displayDataComputed.value.html);
const filename = computed(() => displayDataComputed.value.filename || "output.html");
const files = computed(() => displayDataComputed.value.files || []);
const libraries = computed(() => displayDataComputed.value.libraries || []);
const microfrontend = computed(() => displayDataComputed.value.microfrontend);
const hasMultipleFiles = computed(() => files.value.length > 0);
const usesLibraries = computed(() => libraries.value.length > 0);
const usesMicrofrontend = computed(() => !!microfrontend.value);

// Bootstrap script for library loading via postMessage
const libsBootstrapScript = computed(() => {
  if (!libraries.value.length) return "";
  return `<script data-libs-bootstrap="postmessage">
(function(){
  var resolveLibs, rejectLibs;
  window.__LIBS__ = new Promise(function(res, rej){ resolveLibs = res; rejectLibs = rej; });
  async function onMessage(ev){
    if (ev.source !== window.parent) return;
    if (!ev.data || ev.data.type !== 'shelley-libs') return;
    window.removeEventListener('message', onMessage);
    var out = {};
    try {
      for (var name in ev.data.libs) {
        var src = ev.data.libs[name];
        var url = URL.createObjectURL(new Blob([src], {type: 'text/javascript'}));
        try { out[name] = await import(url); }
        finally { URL.revokeObjectURL(url); }
      }
      resolveLibs(out);
    } catch (e) { rejectLibs(e); }
  }
  window.addEventListener('message', onMessage);
})();
<\/script>`;
});

const MICROFRONTEND_CSP = `<meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'unsafe-inline' blob:; style-src 'unsafe-inline'; img-src data: blob:; font-src data:; connect-src 'none'; media-src data: blob:; object-src 'none'; base-uri 'none'; form-action 'none'; frame-src 'none'">`;

function inlineScriptJSON(value: unknown): string {
  return JSON.stringify(value)
    .replace(/</g, "\\u003c")
    .replace(/>/g, "\\u003e")
    .replace(/&/g, "\\u0026")
    .replace(/\u2028/g, "\\u2028")
    .replace(/\u2029/g, "\\u2029");
}

const microfrontendBootstrapScript = computed(() => {
  const config = microfrontend.value;
  if (!config) return "";
  const context = inlineScriptJSON(config.context || {});
  const capabilities = inlineScriptJSON(config.capabilities || []);
  return `${MICROFRONTEND_CSP}<script data-microfrontend-bootstrap="v1">
(function(){
  var capabilities = Object.freeze(${capabilities});
  function deepFreeze(value) {
    if (!value || typeof value !== 'object' || Object.isFrozen(value)) return value;
    Object.freeze(value);
    Object.keys(value).forEach(function(key){ deepFreeze(value[key]); });
    return value;
  }
  var context = deepFreeze(${context});
  var pending = new Map();
  var sequence = 0;
  function request(method, params) {
    if (capabilities.indexOf(method) === -1) {
      return Promise.reject(new Error('Capability not granted: ' + method));
    }
    var id = 'mf-' + (++sequence);
    return new Promise(function(resolve, reject){
      var timer = setTimeout(function(){
        pending.delete(id);
        reject(new Error('Shelley bridge request timed out'));
      }, 10000);
      pending.set(id, { resolve: resolve, reject: reject, timer: timer });
      window.parent.postMessage({
        type: 'shelley-microfrontend-request',
        version: 1,
        id: id,
        method: method,
        params: params || {}
      }, '*');
    });
  }
  window.addEventListener('message', function(event){
    if (event.source !== window.parent || !event.data || event.data.type !== 'shelley-microfrontend-response') return;
    var item = pending.get(event.data.id);
    if (!item) return;
    pending.delete(event.data.id);
    clearTimeout(item.timer);
    if (event.data.ok) item.resolve(event.data.result);
    else item.reject(new Error(event.data.error || 'Shelley bridge request failed'));
  });
  window.__SHELLEY__ = Object.freeze({
    version: 1,
    context: context,
    capabilities: capabilities,
    request: request
  });
})();
<\/script>`;
});

const htmlWithHeightReporter = computed(() => {
  if (!html.value) return undefined;
  let out = html.value;
  const headInject = microfrontendBootstrapScript.value + libsBootstrapScript.value;
  if (out.includes("<head>")) {
    out = out.replace("<head>", "<head>" + headInject);
  } else {
    out = headInject + out;
  }
  if (out.includes("</body>")) {
    out = out.replace("</body>", HEIGHT_REPORTER_SCRIPT + "</body>");
  } else {
    out = out + HEIGHT_REPORTER_SCRIPT;
  }
  return out;
});

function replyToMicrofrontend(
  id: string,
  response: { ok: true; result?: unknown } | { ok: false; error: string },
) {
  iframeRef.value?.contentWindow?.postMessage(
    { type: "shelley-microfrontend-response", id, ...response },
    "*",
  );
}

function appendComposerDraft(text: string): void {
  const input = document.querySelector<HTMLTextAreaElement>('[data-testid="message-input"]');
  if (!input) throw new Error("The message composer is unavailable");
  const setter = Object.getOwnPropertyDescriptor(
    window.HTMLTextAreaElement.prototype,
    "value",
  )?.set;
  if (!setter) throw new Error("The message composer cannot be updated");
  const next = input.value.trimEnd() ? `${input.value.trimEnd()}\n\n${text}` : text;
  setter.call(input, next);
  input.dispatchEvent(new Event("input", { bubbles: true }));
  input.focus();
}

// Listen for height reports and constrained microfrontend requests.
function handleMessage(event: MessageEvent) {
  if (!iframeRef.value || event.source !== iframeRef.value.contentWindow) return;
  if (!event.data || typeof event.data !== "object") return;

  if (event.data.type === "iframe-height" && typeof event.data.height === "number") {
    const newHeight = Math.min(Math.max(event.data.height, MIN_HEIGHT), MAX_HEIGHT);
    iframeHeight.value = newHeight;
    return;
  }

  if (event.data.type !== "shelley-microfrontend-request") return;
  const id = typeof event.data.id === "string" ? event.data.id : "";
  const method = typeof event.data.method === "string" ? event.data.method : "";
  if (!id || !microfrontend.value?.capabilities?.includes(method)) {
    if (id) replyToMicrofrontend(id, { ok: false, error: "Capability not granted" });
    return;
  }

  if (method === "chat.appendDraft") {
    const text = event.data.params?.text;
    if (typeof text !== "string" || !text.trim()) {
      replyToMicrofrontend(id, { ok: false, error: "text must be a non-empty string" });
      return;
    }
    if (new TextEncoder().encode(text).length > 16 * 1024) {
      replyToMicrofrontend(id, { ok: false, error: "text exceeds 16384 bytes" });
      return;
    }
    try {
      appendComposerDraft(text);
      replyToMicrofrontend(id, { ok: true, result: { drafted: true } });
    } catch (error) {
      replyToMicrofrontend(id, {
        ok: false,
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }
}

onMounted(() => {
  window.addEventListener("message", handleMessage);
});
onUnmounted(() => {
  window.removeEventListener("message", handleMessage);
});

// After iframe loads, fetch libraries and forward via postMessage
function handleIframeLoad() {
  if (!libraries.value.length || !iframeRef.value) return;
  const win = iframeRef.value.contentWindow;
  if (!win) return;
  (async () => {
    const libs: Record<string, string> = {};
    for (const name of libraries.value) {
      const libPath = LIBRARY_PATHS[name];
      if (!libPath) continue;
      try {
        const resp = await fetch(libPath, { credentials: "same-origin" });
        if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
        libs[name] = await resp.text();
      } catch (e) {
        console.error(`output_iframe: failed to fetch library ${name}`, e);
      }
    }
    win.postMessage({ type: "shelley-libs", libs }, "*");
  })();
}

// Fetch libraries and produce a self-contained HTML with inline base64 bootstrap
async function inlineLibrariesIntoHtml(baseHtml: string): Promise<string> {
  if (!usesLibraries.value) return baseHtml;
  const enc = new TextEncoder();
  const libsB64: Record<string, string> = {};
  for (const name of libraries.value) {
    const libPath = LIBRARY_PATHS[name];
    if (!libPath) continue;
    const resp = await fetch(libPath, { credentials: "same-origin" });
    if (!resp.ok) throw new Error(`fetch ${libPath}: HTTP ${resp.status}`);
    const text = await resp.text();
    const bytes = enc.encode(text);
    let bin = "";
    for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
    libsB64[name] = btoa(bin);
  }
  const serialized = JSON.stringify(libsB64);
  const bootstrap = `<script data-libs-bootstrap="inline">
(function(){
  var libsB64 = ${serialized};
  var dec = new TextDecoder();
  window.__LIBS__ = (async function(){
    var out = {};
    for (var name in libsB64) {
      var bin = atob(libsB64[name]);
      var bytes = new Uint8Array(bin.length);
      for (var i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
      var src = dec.decode(bytes);
      var url = URL.createObjectURL(new Blob([src], {type:'text/javascript'}));
      try { out[name] = await import(url); }
      finally { URL.revokeObjectURL(url); }
    }
    return out;
  })();
})();
<\/script>`;
  if (baseHtml.includes("<head>")) {
    return baseHtml.replace("<head>", "<head>" + bootstrap);
  }
  return bootstrap + baseHtml;
}

// Open HTML in new tab with sandbox protection
async function handleOpenInNewTab(e: MouseEvent) {
  e.stopPropagation();
  if (!html.value) return;

  // Open synchronously before any await to avoid popup blocker
  const win = window.open("", "_blank");
  if (!win) return;

  const standalone = await inlineLibrariesIntoHtml(html.value);
  const escapedHtml = escapeHtml(standalone);
  const escapedTitle = escapeHtml(title.value);

  const wrapperHtml = `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>${escapedTitle}</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    html, body { height: 100%; background: #f5f5f5; }
    iframe { 
      width: 100%; 
      height: 100%; 
      border: none;
      background: white;
    }
  </style>
</head>
<body>
  <iframe sandbox="allow-scripts allow-downloads" allow="clipboard-write" srcdoc="${escapedHtml}"></iframe>
</body>
</html>`;

  const blob = new Blob([wrapperHtml], { type: "text/html" });
  const url = URL.createObjectURL(blob);
  win.location.href = url;
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

// Download files - single HTML or zip with all files
async function handleDownload(e: MouseEvent) {
  e.stopPropagation();
  if (!html.value) return;

  const standalone = await inlineLibrariesIntoHtml(html.value);

  if (hasMultipleFiles.value) {
    const zip = new JSZip();
    const originalHtml = getOriginalHtml(standalone);
    zip.file(filename.value, originalHtml);
    for (const file of files.value) {
      zip.file(file.path || file.name, file.content);
    }
    const zipBlob = await zip.generateAsync({ type: "blob" });
    const url = URL.createObjectURL(zipBlob);
    const a = document.createElement("a");
    a.href = url;
    const zipName = filename.value.replace(/\.[^.]+$/, "") + ".zip";
    a.download = zipName;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  } else {
    const blob = new Blob([standalone], { type: "text/html" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename.value;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
}

const isComplete = computed(() => !props.isRunning && props.toolResult !== undefined);
const downloadLabel = computed(() => (hasMultipleFiles.value ? "Download ZIP" : "Download HTML"));
</script>
