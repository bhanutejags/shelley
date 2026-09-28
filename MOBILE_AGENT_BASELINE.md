# Mobile agent baseline

This fork is a private, side-by-side Shelley preview for exe.dev iOS. It is not a Pi port and does not replace a running Shelley.

## Retained native Shelley capabilities

- exe.dev model catalogs refresh natively; `mode: chatgpt` uses exe.dev's documented managed ChatGPT integration/device-code flow, not an API key or custom OAuth implementation.
- Conversation search indexes message content (including subagents); transcript APIs/UI and child-conversation links are native.
- Native subagents persist as conversations, support asynchronous `wait: false`, and deliver results through Shelley’s queue/stream behavior.
- exe.dev's Notion credential integration and Shelley’s reflected skill integrations are the distribution/auth path. The personal Notion skill is a generic Markdown template, not embedded into every Shelley binary; `ntn` custom-base-URL compatibility remains unverified.

## Implemented in this baseline

- Local `web_search` calls the fixed personal `brave.int.<exe-environment>/res/v1/web/search` endpoint, returning up to eight structured results. exe.dev's attached Brave integration proxy injects credentials; Shelley neither accepts nor sends a Brave API key. Missing/unavailable integration errors tell the owner to attach it.
- `web_fetch` performs direct server-side HTTP(S) retrieval and HTML/plain-text extraction; it does not submit URLs to third-party extractors. It rejects credentials, nonstandard ports, private/reserved IPs, exe.dev integration hostnames, and unsafe redirect targets; disables proxy use and automatic redirects; resolves and validates all DNS answers, then connects to a validated IP to prevent DNS rebinding. Requests time out, responses are capped at 1 MiB, and extracted text at 20,000 characters. HTML scripts/styles and similar non-content nodes are omitted.
- Search and fetch have dedicated UI cards; the predictable tool fixture includes both for credential-free UI exercise.
- The Notion skill template retains `ntn` auth, page-sharing, and write-approval safeguards without assuming proxy compatibility or embedding a per-user skill in the binary. Setup steps are documented in [`docs/mobile-agent-setup.md`](docs/mobile-agent-setup.md).

## Acceptance / verification

- [x] Mocked Brave integration success, fixed-host routing/no credential forwarding, missing-integration/actionable error, HTTP error response redaction, and cancellation.
- [x] Mocked fetch HTML extraction, signed-query redaction in output, unsafe redirect and exe.dev integration-host rejection, private DNS rejection, cancellation, response-size cap, and extracted-text cap.
- [x] `claudetool` and `llm/predictable` unit tests passed in the disposable Linux build VM.
- [x] UI TypeScript and Vue type checks passed; UI lint passed after stripping macOS AppleDouble metadata files accidentally preserved during source transfer.
- [x] `make build-custom` completed in the disposable Linux VM. The Makefile's Git-based stamping metadata was supplied by a local shim derived from jj checkout state; no Git VCS command was run.
- [x] `go test -p 1 ./...` passed in the disposable Linux VM, including Chromium-backed `claudetool/browse` tests; `claudetool ./llm/predictable ./skills` also passed after final edits.
- [x] `bin/shelley --predictable-only` mobile viewport smoke passed at 390x844 on an ephemeral loopback port; no Brave/Notion integration or production service was used.
- Existing synthetic tests retain exe.dev ChatGPT integration metadata; history/transcript and subagent behavior are reused, not altered by this baseline.

## Owner setup and deployment boundary

The owner must attach the personal Brave and Notion integrations and link ChatGPT using exe.dev's documented flow. To make Notion instructions available, the owner separately creates/attaches a `type:shelley-skill` file integration from the template. No integration, account, database, production service, or VM was modified. A custom build is only an implementation milestone; no real iOS verification is claimed.
