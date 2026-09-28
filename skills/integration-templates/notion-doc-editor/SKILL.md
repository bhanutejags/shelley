---
name: notion-doc-editor
description: Read, search, and safely edit Notion pages through the official ntn CLI. Use when the user asks about Notion content or connected workspaces.
---

# Notion via `ntn`

Use the official `ntn` CLI; do not build a parallel Notion API client. Read-only access is fine when requested. Never write or archive content unless the user explicitly requested that specific mutation and approved the proposed content.

## Connect the integration

Provision the official `ntn` CLI on the VM and attach exe.dev's Notion integration. `ntn` v0.23.9 supports `NOTION_API_BASE_URL`; isolated mocks verified API read/search and `pages get`/`pages edit` routing. Live proxy authentication and Linux execution still need verification. If the attached proxy replaces Authorization, a dedicated invocation can look like this (replace the attached integration hostname; do not append `/v1`):

```sh
NOTION_API_BASE_URL='https://<attached-notion-name>.int.exe.xyz' \
NOTION_API_TOKEN='proxy-replaced-placeholder' \
NOTION_KEYRING=0 \
ntn api v1/users/me --notion-version 2025-09-03
```

The placeholder is not a credential and is safe only for this dedicated invocation to the credential-injecting proxy. Do not export it globally or use it against `api.notion.com`; live auth replacement still needs verification. If it fails, propagate the error. Otherwise use only an auth method the user explicitly configures for this VM (such as `ntn login` where interactive auth works or `NOTION_API_TOKEN` in the service environment). Never print a token.

A 404 for a valid page usually means the integration has not been connected to it. In Notion open the page's `•••` menu → Connect to → select the integration. Pages and databases must be shared with it.

## Read and search

Apply the same invocation-scoped base URL and placeholder environment above to each command below, only after the proxy check succeeds. Pin the intended API version with `--notion-version` when reproducibility matters; otherwise the CLI may consult Notion's public OpenAPI spec for its default.

- Read a page as Markdown: `ntn pages get <page-id>`.
- Search workspace: `ntn api v1/search query=<text> page_size:=10` (both fields belong in the POST body).
- For a large page, list its children with `ntn api v1/blocks/<page-id>/children page_size==100`, then fetch only the needed blocks.
- Database queries use data sources: `ntn datasources resolve <database-id>`, then `ntn datasources query <data-source-id>`.
- Treat the installed `ntn --help` and `ntn api <path> --docs` as authoritative for current CLI syntax.

## Editing

Preview the exact proposed content and obtain approval before nontrivial writes. `ntn pages edit` replaces the whole body, not just a selected section. For surgical prose changes, use the page Markdown endpoint's `update_content`; for structural changes, patch the specific block ID. Re-read changed content to verify. Archive blocks rather than hard-delete. Use Notion-flavored Markdown only when the user requests rich blocks.

## Safety

Do not use real pages for tests. Never put workspace/page IDs, tokens, or private content in source control. A connected account does not by itself authorize writes.
