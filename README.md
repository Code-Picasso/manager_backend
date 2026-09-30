# M.N.G.R Manager — Go Backend

A **Go + PostgreSQL + Docker** REST API that mirrors the domain of the Flutter
[M.N.G.R Manager](../manager) app: users, tasks/sub-tasks, notes, and derived
alerts. It is a drop-in replacement for the previous Laravel backend, preserving
the exact JSON contract the Flutter client expects.

## Stack

- **Go 1.26** with the standard library `net/http` router (no web framework).
- **PostgreSQL 16** accessed with `jackc/pgx/v5` (raw SQL, no ORM).
- **bcrypt** password hashing and opaque SHA-256 bearer tokens.
- A tiny embedded SQL migration runner (`internal/database`).

## Quick start (Docker)

```bash
cd manager_backend
docker compose up --build
```

On first start the app waits for Postgres, runs the migrations, then serves the
API. You get:

| URL                   | What it is                                          |
| --------------------- | --------------------------------------------------- |
| http://localhost:8000 | The API (`/api/...`)                                |
| http://localhost:8080 | Adminer (server `db`, user `manager`, password `secret`) |

Smoke test:

```bash
curl -X POST http://localhost:8000/api/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada","password":"Secret123!"}'

curl http://localhost:8000/api/tasks \
  -H 'Authorization: Bearer <TOKEN>'
```

Stop with `docker compose down` (add `-v` to delete the database).

## Quick start (local, no Docker)

You need a running Postgres and Go 1.26+.

```bash
cd manager_backend
cp .env.example .env   # adjust DATABASE_URL if needed
go run .
```

## Configuration

Configuration comes from the environment (optionally a `.env` file):

| Variable       | Default                                                       |
| -------------- | ------------------------------------------------------------- |
| `PORT`         | `8000`                                                        |
| `DATABASE_URL` | `postgres://manager:secret@localhost:5433/manager?sslmode=disable` |

## Project layout

```
manager_backend/
├── main.go                    # wiring + graceful shutdown
├── internal/
│   ├── config/                # env configuration
│   ├── database/              # pool, migration runner, embedded schema
│   ├── auth/                  # bcrypt + opaque token helpers
│   ├── models/                # domain types + derived status/progress logic
│   ├── alerts/                # alert derivation rules
│   ├── store/                 # all SQL
│   └── httpapi/               # router, middleware, handlers, validation, DTOs
└── migrations/                # versioned .sql files (applied in order)
```

## API

The routes are unchanged from the Laravel version:

- **Auth** — `POST /api/register`, `POST /api/login`, `POST /api/reset-password`,
  `GET|PUT|DELETE /api/user`, `POST /api/logout`, `POST /api/change-password`.
- **Tasks** — `GET|POST /api/tasks`, `GET|PUT|DELETE /api/tasks/{task}`,
  `POST /api/tasks/{task}/restore`,
  `POST /api/tasks/{task}/subtasks/{subTask}/toggle`.
- **Notes** — `GET|POST /api/notes`, `GET|PUT|DELETE /api/notes/{note}`.
- **Alerts** — `GET /api/alerts`, `POST /api/alerts/read-all`.

Protected routes require `Authorization: Bearer <token>`. Responses are
snake_case JSON with bare objects/arrays (no `data` wrapper), `422` validation
errors shaped as `{"message", "errors"}`, and ISO-8601 UTC timestamps.

## Testing

The feature test suite runs against a real Postgres instance.

```bash
docker compose up -d db
TEST_DATABASE_URL=postgres://manager:secret@localhost:5433/manager_test?sslmode=disable go test ./...
```
