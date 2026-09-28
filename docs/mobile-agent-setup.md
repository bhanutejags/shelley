# Private exe.dev mobile-preview setup

This fork retains Shelley’s exe.dev integrations and adds native, typed Brave search and bounded public-page fetch. These are owner-operated setup instructions only: this repository change does not create, attach, or configure integrations or deploy a VM.

## exe.dev integrations (owner action)

1. Keep the VM's native `llm` integration/catalog attached. It supplies catalog metadata and managed modes; use exe.dev's documented ChatGPT device-code linking and LLM integration flow rather than an API key or custom OAuth implementation: <https://exe.dev/docs/integrations-llm>.
2. Attach exe.dev's personal `brave` integration to the intended VM using the integration catalog: <https://exe.dev/docs/integrations-catalog>. Shelley calls the fixed `brave.int.<exe-environment>` integration hostname at `/res/v1/web/search`. The proxy supplies credentials; do not set, copy, or place a Brave API key in the VM. Shelley search returns an actionable error if this integration is not attached.
3. For the Notion workflow, attach the native `notion` integration to the VM. Keep the personal Notion skill separate from Shelley's embedded built-ins: create an exe.dev file integration marked with attribute `type:shelley-skill`, use the generic template at `skills/integration-templates/notion-doc-editor/SKILL.md`, and attach that file integration to the VM. Shelley already discovers attached skill files (`Type=file`, that attribute) from the integration root and refreshes the skill list for new conversations every 15 seconds. This transports only the Markdown skill body, not scripts or binaries. The owner performs these steps; the implementation does not mutate their account.

The `ntn` CLI's support for a custom API base URL through exe.dev's `notion` proxy has not been verified. Do not claim seamless CLI authentication or copy credentials to the VM based on the integration catalog entry alone. Confirm the CLI's documented custom-base support first; if it cannot use the fixed proxy hostname, use only a runtime auth method the owner explicitly configures. Keep tokens out of logs/source control, share only intended Notion pages with the integration, and require explicit approval for each requested write.

## Tool behavior / boundary

- `web_search` is a Shelley client tool routed only to the fixed Brave integration hostname. Its authentication does not use the VM's general-purpose `web_fetch` path.
- `web_fetch` is not a proxy or crawler: it directly fetches public HTTP(S), rejects credentials, private/reserved destinations, exe.dev integration hostnames and unsafe redirects, and returns bounded HTML/plain-text extraction. It never forwards URLs to third-party extractors. Do not weaken these checks to reach an integration.
- Preview the custom binary side-by-side with a new DB/port after review. No production replacement, restart, account linking, Brave/Notion integration attachment, or real iOS verification is included in this implementation.
