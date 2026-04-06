# Alexandria

A self-hosted reading server. Upload EPUBs and PDFs, browse your library, and read from anywhere — secured behind auth, deployable in a single Docker command.

**Milestone map:**
- v0.1 ✅ Scaffold — project structure, auth, library CRUD, working frontend
- v0.5 🔜 Core Reader — epub.js in-browser rendering, reading position
- v0.8 🔜 TTS — Browser Web Speech API, pause/resume/paragraph control
- v1.0 🔜 Zealot Integration — link books to education tickets, 50-book goal

---

## Quick start (Docker)

```bash
git clone <repo>
cd Alexandria

# Copy the example env file and set a real JWT secret
cp .env.example .env
$EDITOR .env

docker compose up --build
```

Open [http://localhost:8080](http://localhost:8080), create the owner account, then sign in and upload your first book.

The default Compose setup stores uploads and SQLite data in a Docker-managed volume named `alexandria-data`, so startup does not depend on host `./data` permissions.

`REGISTRATION_MODE=single` is the default. The first successful registration bootstraps the deployment, and additional public registrations are blocked.

If you want a host bind mount instead, change the volume in [docker-compose.yml](/home/alexander/Projects/alexandria/docker-compose.yml) back to `./data:/data` and make sure that directory is writable by the container user.

---

## Local development

**Prerequisites:** Go 1.25+, Node 20+

```bash
make setup      # install Go and npm dependencies
make dev        # starts Go server on :8080 + Vite on :5173
```

The Vite dev server proxies `/api` → `:8080`, so hot-reload works for frontend changes.

For a local MCP integration over stdio, run:

```bash
go run ./cmd/mcp-server
```

For remote MCP clients, set `MCP_HTTP_TOKEN` and use the Streamable HTTP endpoint at `/mcp`.
Tool IDs are underscore-separated, for example `books_search` and `reader_get_manifest`, to stay compatible with MCP hosts that forward tool calls through OpenAI-style function calling.
`books_search` uses an MCP-specific default page size of `100` and caps each call at `200` results.

---

## Configuration

All config is via environment variables:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Server port |
| `DATA_DIR` | `./data` | Where book files and cover images are stored |
| `DB_PATH` | `./data/alexandria.db` | SQLite database path |
| `JWT_SECRET` | none | Required. Must be at least 32 characters and must not use the placeholder value |
| `JWT_EXPIRY` | `15m` | Access token lifetime |
| `REFRESH_EXPIRY` | `168h` | Refresh token lifetime (7 days) |
| `REGISTRATION_MODE` | `single` | `disable`, `single`, or `multi`. `multi` shares one global library across all accounts |
| `CORS_ALLOW_ORIGINS` | unset | Comma-separated allowlist for cross-origin API access. Leave unset for same-origin deployments |
| `UPLOAD_MAX_BYTES` | `524288000` | Maximum request body size for book uploads (500 MiB) |
| `MCP_HTTP_TOKEN` | unset | Enables the authenticated MCP Streamable HTTP endpoint at `/mcp` when set |
| `MCP_OWNER_USERNAME` | unset | Optional Alexandria username to use for MCP user-scoped tools/resources; required for multi-user deployments if you want progress or list operations |

---

## Public deployment

Alexandria is designed for a single-owner deployment behind a reverse proxy such as NGINX.

- Run it behind HTTPS only.
- Keep `REGISTRATION_MODE=single` for initial bootstrap, then change it to `disable` after the owner account is created.
- Leave `CORS_ALLOW_ORIGINS` empty unless you explicitly need cross-origin API access.
- A lower public upload limit such as `100 MiB` is recommended unless you need larger files.

See [docs/public-deployment.md](docs/public-deployment.md) for reverse-proxy headers, rate limits, upload sizing, and backup guidance.

---

## Project structure

```
cmd/server/        Go binary entrypoint
domain/            Data structures and domain errors
app/
  ports/           FileStore + BookParser interfaces
  repos/           BookRepo, UserRepo, ProgressRepo interfaces
  apis/            Server interface
  services/        AuthService, BookService, ReaderService
infra/
  sqlite/          GORM/SQLite implementations of repos
  storage/         Local filesystem FileStore
  epub/            Stdlib EPUB metadata + cover parser
api/rest/          Fiber HTTP server, routes, middleware, handlers
config/            Environment-variable config loader
client/            Vue 3 + TypeScript + Vite frontend
test/              Pytest e2e tests
scripts/           dev.sh, setup.sh
```

---

## API

```
GET    /api/v1/auth/status
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
GET    /api/v1/me

GET    /api/v1/books            ?search= &author= &genre= &page= &limit=
POST   /api/v1/books            multipart: file + optional title/author/description
GET    /api/v1/books/:id
PUT    /api/v1/books/:id
DELETE /api/v1/books/:id
GET    /api/v1/books/:id/content
GET    /api/v1/books/:id/cover
GET    /api/v1/books/:id/progress
PUT    /api/v1/books/:id/progress   { cfi, percentage }

GET    /health
GET    /readyz
POST   /mcp                       Streamable HTTP MCP endpoint when `MCP_HTTP_TOKEN` is set
```

All `/api/v1/*` routes except `GET /api/v1/auth/status` require `Authorization: Bearer <access_token>`.

---

## Testing

```bash
# Unit / integration (Go)
make test-go

# E2e (requires server running in multi-user mode)
REGISTRATION_MODE=multi make dev-server &
make test-e2e
```

E2e tests live in [test/e2e/](test/e2e/) and use `pytest` + `httpx`. They create isolated users per test run — no teardown needed.
