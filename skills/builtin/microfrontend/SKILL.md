---
name: microfrontend
description: Use when the user would benefit from a small stateful interactive interface inside chat, such as a configurator, comparison surface, filter panel, or selection workflow. Builds a constrained sandboxed microfrontend through output_iframe.
---

# Constrained microfrontend

Use this only when local interaction is materially better than prose or a static visualization. Prefer the `visualization` skill for charts and the normal response for simple choices.

The app runs in an opaque-origin sandbox. Enabling `microfrontend` adds a restrictive CSP: no network, native form submission, nested frames, plugins, or top-level navigation. It does not receive the conversation automatically. Pass only the small explicit context it needs; never include secrets or unrelated messages.

## Host interface

Inside the iframe, Shelley provides:

```js
window.__SHELLEY__ = {
  version: 1,
  context: Object.freeze(/* explicit context from the tool call */),
  capabilities: Object.freeze([/* granted actions */]),
  request(method, params) // Promise
}
```

The only v1 capability is:

- `chat.appendDraft`: appends non-empty text (maximum 16 KiB) to the visible message composer and focuses it. It never sends the message. Call with `await window.__SHELLEY__.request("chat.appendDraft", {text})`.

Do not imply that a draft was sent or that any external action occurred.

## Workflow

1. Write one self-contained HTML file. Inline CSS and JavaScript; use bundled files through `window.__FILES__` and hosted runtimes through `window.__LIBS__`.
2. Call `output_iframe` with a `microfrontend` object containing:
   - a small JSON `context` object selected for this interface;
   - only the capabilities the interface actually uses.
3. Keep all working state inside the page. Treat host context as immutable.
4. Make the result useful without host actions; a draft button should be an enhancement, not the only way to recover the user's choices.

Example tool input:

```json
{
  "path": "/tmp/chooser.html",
  "title": "Choose itinerary stops",
  "microfrontend": {
    "context": {
      "stops": ["Diablo Lake", "Rainy Lake", "Washington Pass"]
    },
    "capabilities": ["chat.appendDraft"]
  }
}
```

Example action:

```js
async function draftSelection(selection) {
  const result = await window.__SHELLEY__.request("chat.appendDraft", {
    text: `Update the plan with these stops: ${selection.join(", ")}`
  });
  return result.drafted;
}
```

## Interface restrictions

- No passwords, API keys, payment fields, hidden fields, or consent-by-default controls.
- No automatic sending, publishing, booking, purchasing, deletion, or tool execution.
- Do not request a capability until a visible user gesture needs it.
- Use real buttons and labels, keyboard navigation, visible focus, and status text for success/errors.
- Keep layouts responsive and avoid trapping page scroll.
- Render untrusted strings with `textContent`, not `innerHTML`.
