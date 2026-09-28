---
name: notion-doc-editor
description: Read, search, and safely edit Notion pages through the official ntn CLI. Use when the user asks about Notion content or connected workspaces.
---

# Notion via `ntn`

Use Notion's official `ntn` CLI; do not create a parallel API client. This skill never performs live writes unless the user explicitly requested the specific mutation and approved the proposed content.

## Setup and auth

`ntn` and Bun must be installed. Verify `ntn --version` and `bun --version`. Authenticate with `ntn login` in an interactive browser session, or set `NOTION_API_TOKEN` in the Shelley service environment. Do not attempt to retrieve credentials from another machine or print a token. Headless Linux/keychain availability is not assumed. If authentication fails, ask the user to connect the account. Check with `ntn whoami` without exposing secrets.

A 404 for a valid page usually means the integration has not been connected to it. In Notion open the page's `•••` menu → Connect to → select the integration. Pages and databases must be shared with it.

## Read and search

- Read a page as Markdown: `ntn pages get <page-id>`.
- Search workspace: `ntn api v1/search query=<text> page_size==10` (`query` is a POST body field).
- For a large page, list its children with `ntn api v1/blocks/<page-id>/children page_size==100`, then fetch only the needed blocks.
- Read a subagent/conversation transcript only as authorized by the user; do not copy private session data into public code or fixtures.

## Editing

Preview the exact proposed content and ask for approval before nontrivial writes. Prefer Markdown for prose. `ntn pages edit` replaces the entire body, not a partial edit. For a surgical prose replacement, use the page Markdown endpoint's `update_content`; for a structural edit, patch the specific block ID. Re-read the changed content to verify. Archive blocks rather than hard-delete. For unfamiliar block shapes, query the installed CLI's endpoint documentation with `ntn api <path> --docs` and use Notion-flavored Markdown only when the user requests rich blocks.

Useful read-only commands:

```sh
ntn pages get <page-id>
ntn api v1/blocks/<block-id>/children page_size==100
ntn api v1/search query=roadmap page_size==5
```

Database queries use data sources: `ntn datasources resolve <database-id>`, then `ntn datasources query <data-source-id>`. Treat installed `ntn --help` and `ntn api ... --docs` as authoritative.

## Safety

Do not use real pages for tests. Do not place workspace/page IDs, tokens, or private content in source control. Never echo credentials. A connected Notion account does not authorize writes by itself.
