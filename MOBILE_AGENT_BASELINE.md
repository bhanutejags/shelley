# Mobile agent baseline

This fork targets a private exe.dev/iOS preview. It has not replaced a running Shelley.

## Capabilities

- **Models:** retains native exe.dev catalog discovery and ChatGPT subscription mode. Account linking belongs to exe.dev, not a custom OAuth client.
- **History and delegation:** retains native message-content search, transcripts, child-conversation links, and asynchronous subagents (`wait: false`).
- **Brave search:** `web_search` calls the personal integration named `brave`, returns up to eight structured results, and refuses redirects. The integration proxy supplies credentials; Shelley sends no API key.
- **Public-page fetch:** `web_fetch` extracts HTML/plain text directly, without sending URLs to a third-party reader. Responses are capped at 1 MiB; extracted text at 20,000 Unicode code points with an explicit `truncated` flag.
- **Notion:** a separately attached skill template uses the official `ntn` CLI. Skill delivery does not install binaries. CLI base-URL routing is verified with mocks; live integration authentication is not.

## Fetch boundaries

The fetcher rejects URL credentials, nonstandard ports, exe.dev integration hosts, selected private/reserved IPv4 and IPv6 ranges, and unsafe redirects. Checks include mapped/compatible IPv4, well-known/local-use NAT64, 6to4, Teredo, and private-address ISATAP forms. Every DNS answer is checked before connecting to a validated IP; environment proxies and automatic redirects are disabled.

This is a conservative URL/DNS filter, not a sandbox or a claim to block every special-use address. Original tool arguments can remain in conversation history. Fetch cards hide userinfo, query strings, and fragments in pending, failed, and successful states; that does not make transcripts secret-free.

## Verified

Review fixes were tested in a disposable Linux VM with Node 22.18 and Playwright Chromium:

- UI build, TypeScript/Vue type checks, lint, and all **71 UI test files** passed.
- `go test -p 1 -parallel 4 ./...` passed, including browser tests. The sandbox needed a real Chromium binary and an 8192-descriptor process limit; earlier runs failed on the image's snap launcher and open-file limit.
- Regression tests cover Brave redirect refusal/redacted failures, address filtering, private DNS, cancellation, size limits, Unicode truncation, and callable-tool serialization for OpenAI Chat/Responses, Anthropic, and Gemini.
- A custom binary built with the same `version.Version`, `version.Tag`, and `version.Customized=true` linker settings as `make build-custom`; tag/revision came from jj. CLI and HTTP version checks confirmed `customized: true`. This verifies stamping, not the upstream Git-based upgrade/rebase workflow.
- A 390×844 browser smoke verified the predictable-only app on a loopback socket. Rendered fetch-card checks covered pending/error/success URL redaction, Unicode character count, and truncation notice.
- Isolated macOS mocks with `ntn` 0.23.9 verified API read/search and `pages get`/`pages edit` against a custom base URL, with only a fake token. No real Notion pages were accessed or changed.

## Still pending

Live Brave routing, Notion proxy Authorization replacement, Linux `ntn` execution, ChatGPT account linking, and actual exe.dev iOS use require owner setup and verification. See [setup instructions](docs/mobile-agent-setup.md).

No existing VM, account integration, or production service was changed. Preview uses a separate database/port; replacing stock Shelley requires explicit approval and a backup/rollback plan.
