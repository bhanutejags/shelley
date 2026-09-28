# Mobile agent baseline

This fork targets a private, side-by-side Shelley preview for exe.dev iOS. It is not a Pi port and does not replace a running Shelley.

## Native capabilities retained

- exe.dev LLM integration catalogs are discovered and refreshed natively. Catalog metadata includes `mode: chatgpt`; Shelley routes through the exe.dev integration rather than treating a ChatGPT subscription as an API key or implementing OAuth.
- Conversation search already indexes message content (including subagents) and surfaces snippets. Existing conversation APIs/UI provide transcript reading and child conversation links.
- Native subagents can run asynchronously (`wait: false`), persist as conversations, and deliver results. Queued follow-up behavior remains Shelley-native.
- Notion uses the official `ntn` CLI through a Shelley built-in skill; no Notion REST client or live account/page mutation is part of this baseline.

## Baseline acceptance

- [ ] Brave-backed search is not implemented yet. Shelley has model-provider web search, but it does not route through Brave.
- [ ] A dedicated direct `web_fetch` tool is not implemented. Existing browser navigation can read pages but is not a constrained extraction tool.
- exe.dev catalog + ChatGPT integration mode are native and retained; `TestHandleModelsCarriesIntegrationMode` is existing synthetic coverage (not run in this turn).
- Content search/transcripts and subagent history reuse native FTS/UI/storage; source audit found message-content search, transcript routes, and subagent links (tests not run in this turn).
- Notion's official `ntn` CLI workflow and headless auth/page-sharing guidance are documented in a built-in skill. Live auth and page access remain user actions; no live writes performed.
- Native subagents persist and link to child transcripts; `wait: false` provides asynchronous work without a separate daemon (source audit only; tests not run in this turn).

## Preview / blockers

No production VM, database, model account, Brave credential, or Notion workspace is configured by this change. Attempted `make build-custom` in the disposable private sandbox; it stopped because the exeuntu image has no Node/npm (`node: not found`). The VM was deleted. Per the one-sandbox constraint, no second VM was created and no code was built/tested on the Mac. The Brave/web-fetch requirement is an explicit blocker for this partial baseline. Next safe step: approve a properly provisioned private disposable preview/build environment; then add/test Brave and constrained fetch before any preview. Ask the user to connect exe.dev ChatGPT and Notion and supply/configure Brave through an approved integration, never by copying secrets. Do not install over stock Shelley until explicitly approved.
