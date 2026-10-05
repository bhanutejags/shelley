// High-level Vega/Vega-Lite entrypoint for sandboxed output_iframe pages.
// The full runtime is bundled and streamed from Shelley, so chart libraries
// never become part of the conversation transcript.
import embed from "vega-embed";

export async function render({
  spec,
  mountId = "vis",
  renderer = "svg",
  actions = true,
} = {}) {
  if (!spec || typeof spec !== "object") {
    throw new Error("render: spec must be a Vega or Vega-Lite object");
  }
  const mount = document.getElementById(mountId);
  if (!mount) throw new Error(`render: mount element #${mountId} not found`);
  if (renderer !== "svg" && renderer !== "canvas") {
    throw new Error('render: renderer must be "svg" or "canvas"');
  }

  // vega-embed styles its target as inline-block. Rendering directly into a
  // width:100% mount therefore creates a circular "container" measurement and
  // can collapse responsive charts to their padding width. Keep the caller's
  // mount as the stable full-width container and embed into an explicit 100%
  // child.
  mount.replaceChildren();
  const target = document.createElement("div");
  target.style.width = "100%";
  mount.appendChild(target);

  return embed(target, spec, {
    renderer,
    // Use Vega's AST interpreter instead of Function/eval compilation. This
    // keeps charts compatible with the microfrontend CSP, which deliberately
    // omits script-src 'unsafe-eval'.
    ast: true,
    actions: actions
      ? {
          export: true,
          source: false,
          compiled: false,
          editor: false,
        }
      : false,
    tooltip: true,
  });
}
