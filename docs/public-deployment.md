# Public Deployment Notes

Alexandria is intended to run as a single-owner service behind a reverse proxy.

## Recommended App Settings

```env
JWT_SECRET=<random secret with at least 32 characters>
REGISTRATION_MODE=single
UPLOAD_MAX_BYTES=104857600
CORS_ALLOW_ORIGINS=
```

- `single` allows one bootstrap registration and then closes self-service signup.
- After you create the owner account, set `REGISTRATION_MODE=disable` to close registration permanently.
- `multi` is available, but it is a shared-library mode. It does not isolate books per user.
- Leave `CORS_ALLOW_ORIGINS` empty for same-origin deployments behind NGINX.

## Reverse Proxy Expectations

Terminate TLS in NGINX and keep Alexandria on a private upstream.

```nginx
limit_req_zone $binary_remote_addr zone=alex_auth:10m rate=5r/m;
limit_req_zone $binary_remote_addr zone=alex_uploads:10m rate=10r/m;
limit_req_zone $binary_remote_addr zone=alex_downloads:10m rate=60r/m;

server {
    listen 443 ssl http2;
    server_name example.com;

    ssl_certificate     /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;

    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header Content-Security-Policy "default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; script-src 'self'; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'" always;

    client_max_body_size 100m;

    location /api/v1/auth/ {
        limit_req zone=alex_auth burst=10 nodelay;
        proxy_pass http://127.0.0.1:8080;
        include proxy_params;
    }

    location = /api/v1/books {
        limit_req zone=alex_uploads burst=20 nodelay;
        proxy_pass http://127.0.0.1:8080;
        include proxy_params;
    }

    location ~ ^/api/v1/books/.+/(content|cover)$ {
        limit_req zone=alex_downloads burst=120 nodelay;
        proxy_pass http://127.0.0.1:8080;
        include proxy_params;
    }

    location / {
        proxy_pass http://127.0.0.1:8080;
        include proxy_params;
    }
}
```

## Backups

Alexandria stores both SQLite data and uploaded files under `/data`.

- The default [docker-compose.yml](/home/alexander/Projects/alexandria/docker-compose.yml) uses a Docker volume named `alexandria-data` for `/data`.
- Back up the entire `/data` volume or bind mount.
- Restore by replacing `/data` with a known-good snapshot before starting the container.
- Treat `/data/alexandria.db` and `/data/books/` as one unit. Restore them together.
