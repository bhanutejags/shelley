---
name: notion-doc-editor
description: Read, search, and safely edit Notion pages through the official ntn CLI. Use when the user asks about Notion content or connected workspaces.
---

# Notion via `ntn`

Use the official `ntn` CLI; do not build a parallel Notion API client. Read-only access is fine when requested. Never write or archive content unless the user explicitly requested that specific mutation and approved the proposed content.

## Connect the integration

Attach exe.dev's Notion integration to this VM to use its credential-injecting proxy. The `ntn` CLI's support for a custom API base URL has not been established here: check the installed `ntn --help` and official CLI documentation before assuming it can use the proxy hostname. Do not pass Notion API calls to `api.notion.com` with an empty or guessed credential, and do not copy credentials from another machine. If the CLI cannot use the proxy, use only an auth method the user explicitly configures for this VM (such as `ntn login` where interactive auth works or `NOTION_API_TOKEN` in the service environment). Never print a token.

A 404 for a valid page usually means the integration has not been connected to it. In Notion open the page's `•••` menu → Connect to → select the integration. Pages and databases must be shared with it.

## Read and search

- Read a page as Markdown: `ntn pages get <page-id>`.
- Search workspace: `ntn api v1/search query=<text> page_size==10` (`query` is a POST body field).
- For a large page, list its children with `ntn api v1/blocks/<page-id>/children page_size==100`, then fetch only the needed blocks.
- Database queries use data sources: `ntn datasources resolve <database-id>`, then `ntn datasources query <data-source-id>`.
- Treat the installed `ntn --help` and `ntn api <path> --docs` as authoritative for current CLI syntax.

## Editing

Preview the exact proposed content and obtain approval before nontrivial writes. `ntn pages edit` replaces the whole body, not just a selected section. For surgical prose changes, use the page Markdown endpoint's `update_content`; for structural changes, patch the specific block ID. Re-read changed content to verify. Archive blocks rather than hard-delete. Use Notion-flavored Markdown only when the user requests rich blocks.

## Safety

Do not use real pages for tests. Never put workspace/page IDs, tokens, or private content in source control. A connected account does not by itself authorize writes.
