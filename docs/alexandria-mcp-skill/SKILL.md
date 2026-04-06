---
name: alexandria-mcp
description: Use when working through Alexandria's MCP server to search the library, inspect books and EPUB reader data, manage user lists, or read and save progress. This skill explains the safe tool order, user-scoping behavior, and the available Alexandria MCP tools and resources.
metadata:
  short-description: Guide for Alexandria MCP clients
---

# Alexandria MCP

Use this skill when the active client can call Alexandria through MCP and needs repo-specific guidance about the server surface, expected workflows, or user-scoping constraints.

## What This MCP Server Is For

- Alexandria's MCP server is backend-oriented.
- Prefer library search and metadata operations first.
- `books_search` is intentionally broader than the REST default: it returns up to `100` books by default and caps each call at `200`.
- Reader support is for manifest, section, and progress data.
- Do not expect browser-only reader state such as TTS, theme settings, or iframe/UI control.

## User Scope

- `books_*` tools are generally deployment-scoped.
- `lists_*` and `reader_*progress*` tools are user-scoped.
- In single-user deployments, user-scoped tools resolve automatically.
- In multi-user deployments, `MCP_OWNER_USERNAME` should be configured on the server. If it is not, user-scoped operations fail with an actionable configuration error.

## Recommended Workflow

1. Start with `books_search` to find candidate books.
2. Use `books_get` to inspect the exact book record before mutating or reading deeper.
3. For EPUB reading flows, call `reader_get_manifest` before `reader_get_section`.
4. Use `reader_get_progress` before `reader_save_progress` when resuming or updating existing reading state.
5. Use `alexandria://...` resources when the client needs stable JSON context to cite or keep in working memory.

## Mutation Rules

- Safe writes are supported: `books_update`, `books_refresh_metadata`, list operations, and reading progress saves.
- Destructive file operations are intentionally absent in v1. Do not assume upload, delete, or raw file transfer tools exist.
- `books_update` is partial-update oriented. Only send fields you want to change.
- Tool IDs use underscores rather than dots because some MCP hosts forward them into OpenAI-style function calling, which rejects `.` in function names.

## Resource Usage

- Resources return structured JSON text, not binary file contents.
- Prefer resources when you want durable context snapshots.
- Prefer tools when you want filtered search, mutations, or the latest server-side validation behavior.

## Reference

- Read [references/mcp-surface.md](references/mcp-surface.md) for the exact tool and resource inventory, argument expectations, and suggested calling patterns.
