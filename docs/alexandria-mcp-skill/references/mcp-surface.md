# Alexandria MCP Surface

## Tools

### Books

- `books_search`
  - Purpose: search/filter/sort/paginate books.
  - Common inputs: `search`, `author`, `genre`, `sort_by`, `sort_order`, `page`, `limit`.
  - Notes: use this as the primary discovery step. Defaults to `100` results when `limit` is omitted and caps any request at `200`.

- `books_get`
  - Purpose: fetch one book by `book_id`.

- `books_update`
  - Purpose: patch safe book fields.
  - Common inputs: `book_id`, optional `title`, `author`, `description`, `zealot_ticket_id`, and `metadata`.
  - Metadata fields: `isbn`, `publisher`, `published_at`, `language`, `genres`, `tags`.
  - `published_at` accepts `YYYY-MM-DD` or RFC3339.

- `books_refresh_metadata`
  - Purpose: re-parse metadata and cover data from the stored book file.
  - Common input: `book_id`.

- `books_list_authors`
  - Purpose: list distinct authors with counts.

- `books_list_genres`
  - Purpose: list distinct genres with counts.

### Lists

- `lists_list`
  - Purpose: list all lists for the configured MCP owner.

- `lists_create`
  - Purpose: create a list for the configured MCP owner.
  - Common inputs: `name`, optional `description`.

- `lists_update`
  - Purpose: patch a list for the configured MCP owner.
  - Common inputs: `list_id`, optional `name`, optional `description`.

- `lists_get_items`
  - Purpose: list the books attached to a list.
  - Common input: `list_id`.

- `lists_add_book`
  - Purpose: add a book to a list.
  - Common inputs: `list_id`, `book_id`.

- `lists_remove_book`
  - Purpose: remove a book from a list.
  - Common inputs: `list_id`, `book_id`.

### Reader

- `reader_get_manifest`
  - Purpose: fetch the EPUB section list and navigation tree.
  - Common input: `book_id`.
  - Notes: call this before requesting sections.

- `reader_get_section`
  - Purpose: fetch one rewritten reader section.
  - Common inputs: `book_id`, `section_id`.
  - Returns: section metadata plus rendered HTML.

- `reader_get_progress`
  - Purpose: fetch saved reading progress for the configured MCP owner.
  - Common input: `book_id`.
  - Notes: may return `null` progress if the book has not been started.

- `reader_save_progress`
  - Purpose: persist reading progress for the configured MCP owner.
  - Common inputs: `book_id`, `section_id`, `section_progress`, `percentage`.
  - Optional inputs: `block_index`, `rating`.
  - Ranges:
    - `section_progress`: `0..1`
    - `percentage`: `0..1`
    - `rating`: `1..5`

## Resources

- `alexandria://books/{book_id}`
- `alexandria://books/{book_id}/reader/manifest`
- `alexandria://books/{book_id}/reader/sections/{section_id}`
- `alexandria://books/{book_id}/progress`
- `alexandria://lists/{list_id}`
- `alexandria://lists/{list_id}/items`

All resources return JSON text payloads.

## Good Calling Patterns

### Find and inspect a book

1. `books_search`
2. `books_get`
3. Optional: `alexandria://books/{book_id}`

### Resume reading an EPUB

1. `books_get`
2. `reader_get_progress`
3. `reader_get_manifest`
4. `reader_get_section`

### Save reading progress

1. `reader_get_progress`
2. `reader_save_progress`
3. Optional: `alexandria://books/{book_id}/progress`

### Curate a reading list

1. `lists_list`
2. `lists_create` or `lists_update`
3. `lists_add_book` or `lists_remove_book`
4. `lists_get_items`

## Constraints

- No upload tool.
- No delete tool.
- No cover/file binary streaming through MCP resources.
- No UI control or TTS control through MCP.
- In multi-user deployments, missing `MCP_OWNER_USERNAME` blocks user-scoped list/progress operations.
- Tool names intentionally avoid dots because some MCP hosts map them to OpenAI function calls, which only accept letters, numbers, underscores, and hyphens.
