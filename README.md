# huma-template

A Go service template to get you from zero to a running REST API in minutes. Clone it, rename the module, and start building — the boring parts are already done.

## What's included

- **[Huma v2](https://huma.rocks/)** — OpenAPI 3.1 spec generated automatically, request/response validation, and a clean handler API
- **[Chi](https://github.com/go-chi/chi)** router with CORS configured out of the box
- **JWT authentication** — signup, login, token refresh, logout, and token verification, all wired up end-to-end
- **[Bun ORM](https://bun.uptrace.dev/)** with PostgreSQL — models, migrations, and transaction support
- **Structured logging** via [Zerolog](https://github.com/rs/zerolog) — request-scoped, written to stdout and a log file simultaneously
- **Layered config** — YAML file, environment variables, and CLI flags, all merged with [Viper](https://github.com/spf13/viper) + [pflag](https://github.com/spf13/pflag), with struct-tag-driven validation
- **OpenTelemetry** instrumentation plumbed in at the HTTP layer
- **[Just](https://github.com/casey/just)** task runner for common dev workflows
- **Docker Compose** for local PostgreSQL and Redis

---

## Quick start

### 1. Use this template

```sh
git clone https://github.com/mhirii/huma-template.git my-service
cd my-service
```

Then update the module name:

```sh
# Replace the module path everywhere
find . -type f -name "*.go" | xargs sed -i 's|github.com/mhirii/huma-template|github.com/you/my-service|g'
# Update go.mod
go mod edit -module github.com/you/my-service
go mod tidy
```

### 2. Start infrastructure

```sh
docker compose up -d
```

Starts PostgreSQL 17 on `:5432` and Redis 8 on `:6379`.

### 3. Configure

Copy the example config and fill in your values:

```sh
cp api-config.example.yaml api-config.yaml
```

Or use environment variables directly (see [Configuration](#configuration)).

### 4. Run migrations

```sh
just migup
```

### 5. Start the server

```sh
just api
```

The server starts on port `8888` by default. The auto-generated OpenAPI docs are available at:

- **Swagger UI** → `http://localhost:8888/docs`
- **OpenAPI JSON** → `http://localhost:8888/openapi.json`

---

## Project structure

```
.
├── cmd/
│   ├── api/main.go        # Server entrypoint — wires everything together
│   └── migrate.go         # Migration CLI (up / down / status)
├── internal/
│   ├── config/            # APIConfig loader and type definitions
│   ├── dto/               # Request / response types and shared list/filter helpers
│   ├── handlers/          # HTTP handlers — thin layer, delegates to services
│   ├── middleware/         # Auth (JWT) and general (logging, request ID) middleware
│   ├── models/            # Bun ORM models
│   └── svc/               # Business logic — auth, tokens, users
├── migrations/            # SQL migration files + Bun registry
├── pkg/
│   ├── config/            # Reflection-based config binding and validation
│   ├── ctx/               # ServiceContext (context + logger + userID)
│   ├── db/                # PostgreSQL connection helpers
│   ├── logging/           # Zerolog initialisation and helpers
│   └── tokens/            # JWT generation and parsing (HS256)
├── docker-compose.yml
└── justfile
```

---

## Configuration

Config is loaded from `api-config.yaml` (or any file pointed to by `CONFIG_PATH`), then overridden by environment variables, then by CLI flags.

| YAML key | Env var | Default | Description |
|---|---|---|---|
| `server.port` | `SERVICE_PORT` | `8888` | HTTP listen port |
| `logger.log_level` | `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `logger.log_format` | `LOG_FORMAT` | `text` | `text` or `json` |
| `logger.log_file` | `LOG_FILE` | `app.log` | Log file path |
| `auth.auth_secret` | `AUTH_SECRET` | — | **Required.** JWT signing secret |
| `auth.auth_access_token_ttl` | `AUTH_ACCESS_TOKEN_TTL` | — | Access token lifetime (seconds) |
| `auth.auth_refresh_token_ttl` | `AUTH_REFRESH_TOKEN_TTL` | — | Refresh token lifetime (seconds) |
| `db.db_host` | `DB_HOST` | — | **Required.** Postgres host |
| `db.db_port` | `DB_PORT` | `5432` | Postgres port |
| `db.db_username` | `DB_USERNAME` | — | **Required.** Postgres user |
| `db.db_password` | `DB_PASSWORD` | — | **Required.** Postgres password |
| `db.db_name` | `DB_NAME` | — | **Required.** Postgres database name |
| `db.db_ssl` | `DB_SSL` | `false` | Enable SSL mode |

Minimal `api-config.yaml`:

```yaml
server:
  port: 8888

auth:
  auth_secret: "change-me-to-a-long-random-string"
  auth_access_token_ttl: 900       # 15 minutes
  auth_refresh_token_ttl: 604800   # 7 days

db:
  db_host: localhost
  db_port: 5432
  db_username: postgres
  db_password: postgres
  db_name: postgres
```

---

## Auth API

All endpoints live under `/auth`. Routes tagged `"public"` bypass the JWT middleware.

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/auth/signup` | Public | Register a new account, returns token pair |
| `POST` | `/auth/login` | Public | Login with username or email + password |
| `POST` | `/auth/refresh` | Public | Exchange a refresh token for a new access token |
| `POST` | `/auth/logout` | Bearer | Revoke the current refresh token |
| `POST` | `/auth/verify` | Public | Validate a token and return its claims |

### Token strategy

- Short-lived **access token** (`token_type: "access"`) — sent as `Authorization: Bearer <token>` on every request
- Longer-lived **refresh token** (`token_type: "refresh"`) — stored server-side in the `refresh_tokens` table, revocable
- Both tokens share a `token_id` that ties the access token to a specific refresh token record, enabling targeted revocation

---

## Migrations

Migrations are plain SQL files registered with Bun's migration runner.

```sh
just migup       # apply all pending migrations
just migdown     # roll back one migration
just migstatus   # show applied / pending status
```

To add a new migration, create a pair of files:

```
migrations/03_things_table.up.sql
migrations/03_things_table.down.sql
```

Then register them in `migrations/migrate.go`.

---

## Adding a new resource

The pattern for every new resource is the same:

1. **Migration** — `migrations/NN_things.up.sql` + `.down.sql`, registered in `migrations/migrate.go`
2. **Model** — `internal/models/thing.go` with a Bun struct (`bun:"table:things,alias:t"`)
3. **DTOs** — request / response types in `internal/dto/thing.dto.go`; embed `AuthHeader` for protected endpoints and `ListQuery` for list endpoints
4. **Service** — `internal/svc/thing.svc.go`; constructor takes `*bun.DB`, methods take `ctx.ServiceContext`
5. **Handler** — `internal/handlers/thing.handler.go` with a `RegisterThingRoutes(api huma.API, svc *svc.ThingService)` function
6. **Wire up** — call `RegisterThingRoutes(api, thingSvc)` in `cmd/api/main.go`

---

## Filtering and pagination

Any list endpoint can embed `dto.ListQuery` in its request type to get standard query parameters for free:

| Param | Default | Description |
|---|---|---|
| `page` | `1` | Page number (1-based) |
| `per_page` | `10` | Results per page (max 200) |
| `sort_by` | `created_at` | Column to sort by |
| `sort_dir` | `desc` | `asc` or `desc` |
| `search` | `""` | ILIKE search across relevant text columns |
| `filters` | `[]` | JSON array of filter objects (see below) |

Filter object:

```json
{ "field": "email", "rule": "contains", "value": "@example.com" }
```

Supported rules: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `contains`, `in`, `nin`, `is`, `nis`, `null`, `nnull`.

Apply filters to any Bun query with the helpers in `internal/dto/common.dto.go`:

```go
filters, _ := dto.ParseFilters(params.Filters)
q = dto.ApplyFilters(filters, q)
```

---

## Tech stack

| Concern | Library |
|---|---|
| API framework | [danielgtaylor/huma v2](https://github.com/danielgtaylor/huma) |
| Router | [go-chi/chi v5](https://github.com/go-chi/chi) |
| ORM | [uptrace/bun](https://github.com/uptrace/bun) |
| JWT | [cristalhq/jwt v5](https://github.com/cristalhq/jwt) |
| Logging | [rs/zerolog](https://github.com/rs/zerolog) |
| Config | [spf13/viper](https://github.com/spf13/viper) + [spf13/pflag](https://github.com/spf13/pflag) |
| IDs | [oklog/ulid v2](https://github.com/oklog/ulid) |
| Passwords | [golang.org/x/crypto bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt) |
| Tracing | [OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/) |
| Task runner | [casey/just](https://github.com/casey/just) |

---

## License

MIT
