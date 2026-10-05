---
name: visualization
description: Use when an interactive chart, statistical graphic, timeline, heatmap, or data visualization would communicate better than prose. Renders Vega-Lite or Vega specifications inside chat through output_iframe.
---

# Interactive visualization

Prefer Vega-Lite. Use full Vega only when Vega-Lite cannot express the interaction or layout.
Render through Shelley's hosted `vega` runtime; do not add CDN scripts.

## Workflow

1. Write the visualization specification to `/tmp/visualization-spec.json`.
2. Write the renderer below to `/tmp/visualization.html`.
3. Call `output_iframe` with:
   - `path=/tmp/visualization.html`
   - `files={"spec.json":"/tmp/visualization-spec.json"}`
   - `libraries=["vega"]`
   - a descriptive `title`
4. Revise the specification when the user requests changes.

Use inline data or a bundled data file whenever practical. Do not place secrets or unnecessary personal data in the specification: bundled files are stored in the conversation. Avoid remote data URLs unless the user requested that source and it is safe for the sandboxed page to contact it.

Make charts responsive with `"width": "container"` where the mark supports it. Include a meaningful title, readable labels, units, legends, and accessible color choices. Avoid 3D effects and unnecessary decoration. For small datasets, label important values directly. Use transforms rather than precomputing only when the transform remains understandable.

## Renderer template

```html
<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>
  * { box-sizing: border-box; }
  html, body { margin: 0; background: #fff; color: #1f2937; font: 14px/1.4 system-ui, sans-serif; }
  #vis { width: 100%; min-height: 240px; padding: 12px; }
  #error { display: none; margin: 12px; color: #991b1b; white-space: pre-wrap; }
</style>
</head>
<body>
<div id="vis" role="img" aria-label="Interactive data visualization"></div>
<pre id="error"></pre>
<script type="module">
try {
  const { render } = (await window.__LIBS__).vega;
  const spec = JSON.parse(window.__FILES__["spec.json"]);
  await render({ spec, mountId: "vis", renderer: "svg" });
} catch (error) {
  const el = document.getElementById("error");
  el.style.display = "block";
  el.textContent = error?.stack || String(error);
}
</script>
</body>
</html>
```

The hosted renderer supports Vega and Vega-Lite by inspecting the specification schema. It uses Vega's CSP-compatible AST interpreter, enables local PNG/SVG export, and disables source, compiled-spec, and external-editor actions.
