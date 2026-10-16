# Tabel — attendance microservice (KSTU)

Go service for employee QR check-in/out, absences, live reports, and archival.

**Production deploy = plain Linux binaries + systemd. No Docker on the server.**

## Stack

- Chi, pgx, sqlc, goose, JWT HS256, asynq/Redis, slog
- Optional `docker-compose.yml` only for local Postgres/Redis on a laptop

## Build binaries (for server)

On your Mac/CI (cross-compile to Linux):

```bash
./scripts/build-linux.sh
# → bin/tabel-server, bin/tabel-worker
```

Or native on the Linux host:

```bash
make build
```

Copy `bin/`, `migrations/`, `deploy/systemd/`, `.env.example` to the server, then:

```bash
sudo ./scripts/install-server.sh
sudo nano /etc/tabel/tabel.env   # JWT_SECRET, DATABASE_URL, REDIS_URL, HTTP_ADDR
goose -dir /opt/tabel/migrations postgres "$DATABASE_URL" up
sudo systemctl start tabel-api tabel-worker
curl http://127.0.0.1:8080/healthz
```

The API process listens on `HTTP_ADDR` (default `:8080`) **directly on the host**.  
No container ports, no Docker networking. If the campus already has nginx/traefik, proxy to `127.0.0.1:8080`; otherwise open the port on the firewall — DevOps does not need container ingress.

### systemd units

- `tabel-api.service` → `/opt/tabel/bin/tabel-server`
- `tabel-worker.service` → `/opt/tabel/bin/tabel-worker` (auto-close open shifts at 18:30, archive)

Env file: `/etc/tabel/tabel.env` (see `.env.example`).

## Auth

Все `/v1/*` требуют авторизации (один из вариантов):

1. **Basic** (Swagger): Authorize → `user` / `user`
2. **Cookie:** `POST /auth/login` с `{"username":"user","password":"user"}` → cookie `access_token`
3. **Bearer:** `Authorization: Bearer <access_jwt>` (HS256, claim `sub` = user UUID)

После логина: клиент вызывает uadmin `/users/me`, затем `POST /v1/me/sync` (не на каждый QR).

## Docs for frontend / mobile

- **Swagger UI** (после запуска API): [`/swagger/`](http://localhost:8080/swagger/)
  - Authorize → Username `user`, Password `user`, затем Try it out
  - Или сначала `POST /auth/login` (example уже заполнен)
- **OpenAPI 3:** [`api/openapi.yaml`](api/openapi.yaml)
- **Интеграционный manual:** [`docs/FRONTEND.md`](docs/FRONTEND.md) — flows, request/response, коды ошибок, чеклист

## API

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/healthz` | Health |
| POST | `/auth/login` | Local login → cookie + JWT |
| POST | `/auth/logout` | Clear cookie |
| POST | `/v1/me/sync` | Upsert employee snapshot |
| POST | `/v1/attendance/scan` | QR scan: check-in / check-out / duplicate |
| PUT | `/v1/timesheet/days` | Head/editor: set attendance, absence or clear a day |
| GET | `/v1/timesheet/edits` | Audit log of timesheet edits |
| GET | `/v1/departments/{id}/employees` | Department employee list |
| GET | `/v1/departments/{id}/missing` | Days without a scan (for review, not truancy) |
| GET | `/v1/departments/{id}/export` | Full department dump for reporting |
| POST | `/v1/absences/requests` | Employee absence request |
| POST | `/v1/absences/head` | Head sets code (no scan that day) |
| POST | `/v1/absences/{id}/decide` | Approve/reject |
| GET | `/v1/reports/me?from=&to=` | My live report |
| GET | `/v1/reports/department?from=&to=` | Department report (head/editor) |
| GET | `/v1/reports/departments/{id}?from=&to=` | Department report by id |
| GET | `/v1/reports/users/{userID}?from=&to=` | User report |

OpenAPI sketch: `api/openapi.json`.

## Local develop

```bash
cp .env.example .env
docker compose up -d postgres redis migrate   # Postgres + Redis (+ schema)
set -a && source .env && set +a
make build && ./bin/tabel-server
# optional worker:
./bin/tabel-worker
```

Swagger: http://localhost:8080/swagger/ → **Authorize** → `user` / `user` → сначала `POST /v1/me/sync` (example уже в UI).

Legacy Django module is in `legacy/`.
