# Alexandria Integration Guide

This document describes how external tools can integrate with an Alexandria book server using its REST API and MCP interface.

---

## Authentication

All API endpoints except auth and health checks require a JWT Bearer token.

### Obtaining a Token

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "your_username",
  "password": "your_password"
}
```

Response:
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ..."
}
```

Use the access token in all subsequent requests:

```
Authorization: Bearer eyJ...
```

### Token Lifecycle

- **Access token** expires in 15 minutes (configurable via `JWT_EXPIRY`).
- **Refresh token** expires in 7 days (configurable via `REFRESH_EXPIRY`).

Refresh before expiry:

```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJ..."
}
```

---

## Core Concepts

### Book

The central object. Key fields:

| Field | Type | Notes |
|---|---|---|
| `id` | UUID string | Stable identifier — use this for all references |
| `title` | string | |
| `author` | string | |
| `description` | string | |
| `file_type` | `"epub"` \| `"pdf"` \| `"url"` | |
| `file_size` | int64 | Bytes |
| `metadata.isbn` | string | |
| `metadata.genres` | string[] | |
| `metadata.tags` | string[] | |
| `zealot_ticket_id` | string? | Optional link to an external ticketing system |
| `links` | BookLink[] | External URLs associated with the book |

### ReadingProgress

Per-user progress for a given book:

| Field | Type | Notes |
|---|---|---|
| `section_id` | string | Current EPUB section |
| `section_progress` | float (0–1) | Position within current section |
| `percentage` | float (0–1) | Overall book completion |
| `rating` | int? | 1–5; nil means unrated |
| `started_at` | timestamp | |
| `last_read_at` | timestamp | |
| `finished_at` | timestamp? | Set when book is marked complete |

---

## Common Workflows

### Listing and Syncing Books

Fetch the full library:

```http
GET /api/v1/books
Authorization: Bearer ...
```

Supports query parameters for filtering and pagination:

| Param | Description |
|---|---|
| `q` | Full-text search across title, author, description |
| `author` | Filter by author |
| `genre` | Filter by genre |
| `file_type` | Filter by `epub`, `pdf`, or `url` |
| `page` | Page number |
| `per_page` | Results per page |

For a sync workflow, poll this endpoint and compare `updated_at` timestamps against your local state to detect changes.

> **Gap:** There is no webhook or change-feed mechanism. Integrations that need near-real-time updates must poll. A `since` parameter on `GET /api/v1/books` (filter by `updated_at > timestamp`) does not currently exist — you would need to fetch all books and diff locally.

Fetch available facets for UI filters:

```http
GET /api/v1/books/authors
GET /api/v1/books/genres
GET /api/v1/books/publishers
GET /api/v1/books/years
```

Each returns a list of values with counts.

### Uploading a Book

```http
POST /api/v1/books
Authorization: Bearer ...
Content-Type: multipart/form-data

file=@my-book.epub
title=Optional Override Title
author=Optional Override Author
```

The server extracts metadata from the file automatically. Supply optional fields to override what is extracted.

> **Gap:** There is no bulk upload endpoint. Uploading a large Calibre library requires sequential requests.

### Tracking Reading Progress

Get current progress for a book:

```http
GET /api/v1/books/{id}/progress
Authorization: Bearer ...
```

Save progress (e.g., from an external reading app syncing back):

```http
PUT /api/v1/books/{id}/progress
Authorization: Bearer ...
Content-Type: application/json

{
  "section_id": "chapter-04",
  "section_progress": 0.42,
  "percentage": 0.31
}
```

Mark a book as finished by setting `finished_at`:

```http
PUT /api/v1/books/{id}/progress
Content-Type: application/json

{
  "section_id": "chapter-12",
  "section_progress": 1.0,
  "percentage": 1.0,
  "finished_at": "2026-04-07T21:00:00Z"
}
```

### Managing Collections (Lists)

Lists are user-defined collections of books. Useful for reading queues, wishlists, or curated sets.

Create a list:

```http
POST /api/v1/lists
Content-Type: application/json

{
  "name": "2026 Reading Goals",
  "description": "50 books by December"
}
```

Add a book to a list:

```http
POST /api/v1/lists/{list_id}/books
Content-Type: application/json

{
  "book_id": "uuid-of-book"
}
```

Get all books in a list:

```http
GET /api/v1/lists/{list_id}/books
```

### Enriching Metadata

Look up a book's metadata from external sources (OpenLibrary, Google Books):

```http
GET /api/v1/metadata/search?isbn=9780743273565
GET /api/v1/metadata/search?q=The+Great+Gatsby&author=Fitzgerald
```

Apply enriched metadata back to a book:

```http
PUT /api/v1/books/{id}
Content-Type: application/json

{
  "metadata": {
    "isbn": "9780743273565",
    "publisher": "Scribner",
    "language": "en",
    "genres": ["Fiction", "Classic"]
  }
}
```

Trigger a re-extraction of metadata from the file itself:

```http
POST /api/v1/books/{id}/refresh
```

### Adding External Links to a Book

Attach related URLs (audiobook, purchase link, companion site) to a book:

```http
POST /api/v1/books/{id}/links
Content-Type: application/json

{
  "label": "Audible Audiobook",
  "url": "https://..."
}
```

### Linking a Book to an External System

If you maintain an external ticketing or goal-tracking system (like Zealot), you can associate a ticket ID with a book:

```http
PUT /api/v1/books/{id}
Content-Type: application/json

{
  "zealot_ticket_id": "EDU-123"
}
```

Similarly, reading progress supports an optional `zealot_progress_ref` field for linking a progress entry back to an external tracker entry.

---

## MCP Integration (AI Assistants)

Alexandria exposes a [Model Context Protocol](https://modelcontextprotocol.io) server, which allows AI assistants (e.g., Claude) to interact directly with your library.

The MCP server supports the following tools:

| Tool | Description |
|---|---|
| `books_search` | Search the library |
| `books_get` | Fetch a single book |
| `books_update` | Update book metadata |
| `books_refresh_metadata` | Re-extract metadata from file |
| `books_list_authors` / `_genres` | Fetch facets |
| `lists_list` / `lists_create` / `lists_update` | Manage collections |
| `lists_get_items` / `lists_add_book` / `lists_remove_book` | Manage list membership |
| `reader_get_manifest` | Get EPUB structure |
| `reader_get_section` | Fetch rendered section HTML |
| `reader_get_progress` / `reader_save_progress` | Track reading position |

Configure authentication via `MCP_HTTP_TOKEN` on the server. Set `MCP_OWNER_USERNAME` to scope operations to a specific user.

---

## Downloading Book Files

```http
GET /api/v1/books/{id}/content
Authorization: Bearer ...
```

Returns the raw EPUB or PDF file. Useful for backup tools, offline sync, or feeding the file to another processing pipeline.

```http
GET /api/v1/books/{id}/cover
Authorization: Bearer ...
```

Returns the cover image.

---

## Health & Readiness

```http
GET /health    # Basic health check — no auth required
GET /ready    # Readiness check — no auth required
```

Use these for uptime monitoring or to gate dependent workflows.

---

## Gaps and Missing Functionality

The following are limitations worth knowing before building an integration:

1. **No webhooks.** There is no way to subscribe to library changes. Sync integrations must poll `GET /api/v1/books` and diff locally.

2. **No `updated_since` filter.** You cannot request "books modified after timestamp X." A full library fetch is required for each sync cycle.

3. **No bulk upload.** Books must be uploaded one at a time.

4. **No bulk progress export.** There is no endpoint to fetch all reading progress records across the library in a single call. You must request `GET /api/v1/books/{id}/progress` per book.

5. **No user listing.** There is no admin endpoint to enumerate users or their progress (relevant for multi-user deployments).

6. **Progress has no explicit "want to read" / "currently reading" state.** A book is "not started" if no progress record exists, "in progress" if `percentage < 1.0`, and "finished" if `finished_at` is set. There is no explicit queued/want-to-read state separate from Lists.
