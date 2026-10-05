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

  return embed(mount, spec, {
    renderer,
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
