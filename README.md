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

`REGISTRATION_MODE=single` is the default. The first successful registration bootstraps the deployment, and additional public registrations are blocked.

---

## Local development

**Prerequisites:** Go 1.25+, Node 20+

```bash
make setup      # install Go and npm dependencies
make dev        # starts Go server on :8080 + Vite on :5173
```

The Vite dev server proxies `/api` → `:8080`, so hot-reload works for frontend changes.

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
