# AGENT.md

## Project Overview

This is a Go REST API backend template built with the [Huma v2](https://huma.rocks/) library. It provides a functional authentication system (signup, login, logout, token refresh, token verification) and a users CRUD scaffold, all wired up with JWT-based authentication, PostgreSQL persistence via Bun ORM, structured logging via Zerolog, and a flexible config system backed by Viper + pflag.

**Module:** `github.com/mhirii/huma-template`
**Go version:** 1.25.5
**HTTP Router:** `go-chi/chi v5` (via `humachi` adapter)
**API Framework:** `danielgtaylor/huma/v2`

---

## Directory Structure

```
huma-template/
├── cmd/
│   ├── api/
│   │   └── main.go          # API server entrypoint (wires everything together)
│   └── migrate.go           # Migration CLI command handler
├── internal/
│   ├── config/              # Holds config types
│   │   ├── api.cfg.go       # APIConfig loader (Viper + pflag + env)
│   │   └── types.cfg.go     # ServerConfig, AuthConfig types
│   ├── dto/                 # Holds request/response types
│   │   ├── auth.dto.go      # Auth request/response types
│   │   ├── common.dto.go    # Shared types: ListQuery, Filter, AuthHeader, ApplyFilters
│   │   └── users.dto.go     # Users request/response types + UserModelRes
│   ├── handlers/
│   │   ├── auth.handler.go  # Auth HTTP handlers (login, signup, refresh, logout, verify)
│   │   └── users.handler.go # Users HTTP handlers (CRUD + list)
│   ├── middleware/          # Holds middleware
│   │   ├── auth.mw.go       # JWT auth middleware (validates Bearer token)
│   │   └── common.mw.go     # Request logging, request ID injection, start time
│   ├── models/              # Holds db models
│   │   ├── users.go         # Users Bun model (table: users)
│   │   └── rtokens.go       # RefreshTokens Bun model (table: refresh_tokens)
│   └── svc/                 # Holds service layer
│       ├── auth.svc.go      # AuthService: signup, login, refresh, logout, verify
│       ├── tokens.svc.go    # TokensService: create/validate/revoke JWT pairs
│       └── users.svc.go     # UsersService: full CRUD + list/filter
├── migrations/              # Holds migrations
│   ├── migrate.go           # Bun migrations registry
│   ├── 01_users_table.up.sql
│   ├── 01_users_table.down.sql
│   ├── 02_rtokens_table.up.sql
│   └── 02_rtokens_table.down.sql
├── pkg/
│   ├── config/
│   │   └── config.go        # BindConfigStruct + ValidateConfigStruct (reflection-based)
│   ├── ctx/
│   │   └── ctx.go           # ServiceContext wrapping context.Context + zerolog.Logger
│   ├── db/
│   │   ├── pg.go            # Bun/PostgreSQL connection setup + helpers
│   │   └── db.cfg.go        # PGConfig type
│   ├── logging/
│   │   ├── logging.go       # Zerolog init + L() + FromCtx() helpers
│   │   └── logging.cfg.go   # LoggerConfig type
│   └── tokens/
│       └── jwt.go           # TokenProvider, GenerateToken, ParseToken (HS256 JWT)
├── docker-compose.yml       # PostgreSQL + Redis services for local dev
├── justfile                 # Task runner (api, migup, migdown, migstatus)
├── go.mod
└── go.sum
```

---

## Key Dependencies

| Package | Purpose |
|---|---|
| `danielgtaylor/huma/v2` | API framework (OpenAPI, validation, middleware) |
| `go-chi/chi/v5` | HTTP router (used via humachi adapter) |
| `go-chi/cors` | CORS middleware |
| `uptrace/bun` | ORM for PostgreSQL |
| `cristalhq/jwt/v5` | JWT signing and verification (HS256) |
| `rs/zerolog` | Structured, leveled logging |
| `spf13/viper` | Config file + env var loading |
| `spf13/pflag` | CLI flag parsing |
| `oklog/ulid/v2` | ULID generation for IDs |
| `golang.org/x/crypto` | bcrypt password hashing |
| `go.opentelemetry.io/...` | OpenTelemetry instrumentation |

---

## Configuration

Configuration is loaded in `internal/config/api.cfg.go` via `config.Load(&env)`. The config file name is derived from the `env` prefix (e.g., `api-config.yaml`).

### `APIConfig` shape

```yaml
server:
  port: 8888               # SERVICE_PORT env var

logger:
  log_level: info          # LOG_LEVEL env var (debug|info|warn|error)
  log_format: text         # LOG_FORMAT env var (text|json)
  log_file: app.log        # LOG_FILE env var

auth:
  auth_secret: ""          # AUTH_SECRET env var (required)
  auth_access_token_ttl: 0 # AUTH_ACCESS_TOKEN_TTL env var (seconds)
  auth_refresh_token_ttl: 0 # AUTH_REFRESH_TOKEN_TTL env var (seconds)
  auth_rate_limit: 0       # AUTH_RATE_LIMIT env var

db:
  db_host: ""              # DB_HOST env var (required)
  db_port: 5432            # DB_PORT env var
  db_username: ""          # DB_USERNAME env var (required)
  db_password: ""          # DB_PASSWORD env var (required)
  db_name: ""              # DB_NAME env var (required)
  db_ssl: false            # DB_SSL env var
```

### Priority Order (lowest → highest)
1. Default values (struct tags `default:"..."`)
2. YAML config file
3. Environment variables
4. CLI flags

All struct fields use `flag`, `env`, `yaml`, `default`, and `validate` tags — processed by the custom reflection-based helpers in `pkg/config/config.go`.

---

## Running Locally

### Start infrastructure

```sh
docker compose up -d
```

This starts:
- PostgreSQL 17 on port `5432` (user: `postgres`, pass: `postgres`, db: `postgres`)
- Redis 8 on port `6379`

### Run migrations

```sh
just migup
# or
go run ./cmd/api migrate up
```

Rollback:
```sh
just migdown
```

Check status:
```sh
just migstatus
```

### Start the API server (with hot reload via Air)

```sh
just api
```

This builds `./cmd/api` → `./tmp/api` and reloads on `.go` / `.yaml` / `.yml` changes.

---

## API Entrypoint (`cmd/api/main.go`)

The server bootstraps in this order:
1. Initialize logger
2. Load config (`api-config.yaml` / env vars)
3. If `os.Args[1] == "migrate"`, delegate to `cmd.Migrate(action)` and exit
4. Create DB connection via `db.New(dsn)`
5. Set up Chi router with CORS
6. Create Huma API with `humachi.New`
7. Create `TokenProvider`, `TokensService`, `UsersService`, `AuthService`
8. Register routes: `RegisterAuthRoutes`, `RegisterUserRoutes`
9. Register global Huma middleware: `AuthMiddleware`, `GeneralMiddleware`
10. Wrap handler with OTel and request logging middleware (`OnStartMiddleware`)
11. Start HTTP server on configured port

---

## Authentication Flow

### JWT Strategy
- **Access token** — short-lived, type `"access"`, validated on every protected request
- **Refresh token** — longer-lived, type `"refresh"`, stored in the `refresh_tokens` table
- Both tokens carry claims: `sub` (user ID), `username`, `email`, `token_type`, `token_id`
- `token_id` on the access token equals the `id` of the refresh token record — this links them for revocation

### Endpoints (`/auth/*`)

| Method | Path | Description |
|---|---|---|
| `POST` | `/auth/signup` | Create account, returns token pair |
| `POST` | `/auth/login` | Login with username or email + password, returns token pair + user |
| `POST` | `/auth/refresh` | Exchange refresh token for new access token |
| `POST` | `/auth/logout` | Revoke refresh token (requires `Authorization: Bearer <refresh_token>`) |
| `POST` | `/auth/verify` | Validate an access token and return its claims |

### `AuthMiddleware` (`internal/middleware/auth.mw.go`)
- Skips auth for routes tagged `"public"` (only `GET /` currently)
- Expects `Authorization: Bearer <access_token>` header
- Calls `tokens.ParseToken` to validate HS256 signature and expiry
- Injects `userID` (from `sub` claim) into context

---

## Users Endpoints (`/users/*`)

| Method | Path | Implemented |
|---|---|---|
| `POST` | `/users/` | Stub (501) |
| `GET` | `/users/{id}` | Stub (501) |
| `PUT` | `/users/{id}` | Stub (501) |
| `DELETE` | `/users/{id}` | Stub (501) |
| `GET` | `/users/` | Stub (501) |

> All user handler methods (`Create`, `Get`, `Update`, `Delete`, `List`) currently return `huma.Error501NotImplemented`. The underlying service layer (`UsersService`) is **fully implemented** — wire up the handlers to call it.

### `UsersService` available methods

```go
GetUsers(ctx, *dto.ListUsersReq) (*dto.ListUsersRes, error)
GetUserByID(ctx, id string) (*dto.UserModelRes, error)
GetUserByField(ctx, field, value string) (*dto.UserModelRes, error)
CreateUser(ctx, *dto.CreateUserReqBody, *bun.Tx) (*dto.UserModelRes, error)
UpdateUser(ctx, models.Users) (*dto.UserModelRes, error)
DeleteUser(ctx, id string) error
```

---

## Database Models

### `users` table

| Column | Type | Notes |
|---|---|---|
| `id` | TEXT PK | ULID |
| `username` | TEXT NOT NULL | Unique index |
| `email` | TEXT | Unique partial index (nullable) |
| `email_verified` | BOOLEAN | Default `false` |
| `password_hash` | TEXT | bcrypt |
| `avatar_url` | TEXT | |
| `created_at` | TIMESTAMPTZ | Default `now()` |
| `updated_at` | TIMESTAMPTZ | Default `now()` |

### `refresh_tokens` table

| Column | Type | Notes |
|---|---|---|
| `id` | TEXT PK | ULID, also the JWT `token_id` |
| `user_id` | TEXT FK | References `users(id)` ON DELETE CASCADE |
| `token` | TEXT | Refresh token JWT ID stored for validation |
| `device` | TEXT | (currently stored as empty string) |
| `expires_at` | TIMESTAMPTZ | |
| `created_at` | TIMESTAMPTZ | Default `now()` |
| `revoked_at` | TIMESTAMPTZ | NULL = active, non-NULL = revoked |

---

## Context & Logging Conventions

### `pkg/ctx` — `ServiceContext`

All service methods accept a `ctx.ServiceContext` (not bare `context.Context`). This struct embeds `context.Context` and carries a `zerolog.Logger` and optional `*string` UserID.

Convert an incoming `context.Context` to `ServiceContext` using:
```go
svcCtx := ctx.FromContext(c)
```

### `pkg/logging`

- Call `logging.InitLogger(cfg)` once at startup.
- Use `logging.L()` to get the global logger.
- Use `logging.FromCtx(ctx)` to get a request-scoped logger (falls back to global).
- Log output goes to **both** stdout and `app.log` simultaneously.

### Request Context Values

The following keys are stored in `context.Context` throughout the request lifecycle:

| Key | Type | Set by |
|---|---|---|
| `"requestID"` | `string` | `OnStartMiddleware` |
| `"requestStart"` | `time.Time` | `OnStartMiddleware` |
| `"logger"` | `zerolog.Logger` | `OnStartMiddleware` / `GeneralMiddleware` |
| `"userID"` | `string` | `AuthMiddleware` |
| `"operationID"` | `string` | `GeneralMiddleware` |

---

## Adding a New Feature

### 1. Add a migration

Create `migrations/NN_thing.up.sql` and `migrations/NN_thing.down.sql`, then register them in `migrations/migrate.go`.

### 2. Add a model

Create `internal/models/thing.go` with a Bun model struct using `bun:"table:things,alias:t"`.

### 3. Add DTOs

Add request/response types to `internal/dto/`. Follow the pattern:
- `ThingReq` with `AuthHeader` embedded and typed `Body` struct
- `ThingRes` with `Body` field

### 4. Add a service

Create `internal/svc/thing.svc.go`. Accept `*bun.DB` in the constructor and `ctx.ServiceContext` in every method.

### 5. Add handlers

Create `internal/handlers/thing.handler.go`. Call `RegisterThingRoutes(api huma.API, svc *svc.ThingService)` from `cmd/api/main.go`.

### 6. Register in `cmd/api/main.go`

Instantiate the service and pass it to `RegisterThingRoutes(api, thingSvc)`.

---

## Filtering & Pagination

The `ListQuery` DTO (`internal/dto/common.dto.go`) provides standard query params for any list endpoint:

| Param | Default | Description |
|---|---|---|
| `page` | `1` | Page number (1-based) |
| `per_page` | `10` | Items per page (max 200) |
| `sort_by` | `created_at` | Column to sort by |
| `sort_dir` | `desc` | `asc` or `desc` |
| `search` | `""` | Full-text search string |
| `filters` | `[]` | JSON array of `Filter` objects |
| `includes` | `{}` | JSON map (not yet implemented) |

`Filter` object shape:
```json
{ "field": "email", "rule": "contains", "value": "example" }
```

Supported `rule` values: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `contains`, `in`, `nin`, `is`, `nis`, `null`, `nnull`.

Use `dto.ParseFilters(rawString)` and `dto.ApplyFilters(filters, query)` to apply them to a Bun select query.

---

## Token Package (`pkg/tokens`)

### `TokenProvider`
Created with `tokens.NewTokenProvider(args)`. Requires a non-empty `Secret` and positive `AccessTokenTTL`/`RefreshTokenTTL` (in seconds).

Key methods:
- `GetTokensPair(ctx, sub, username, email, refreshTokenID)` → `*TokensPair`
- `ParseAccess(ctx, tokenString)` → `*UserClaims, error`
- `ParseRefresh(ctx, tokenString)` → `*UserClaims, error`
- `GetAccess(ctx, sub, username, email, refreshTokenID)` → access token + expiry

### `UserClaims`
Extends `jwt.RegisteredClaims` with `Email`, `Username`, `TokenType` (`"access"` | `"refresh"`), `TokenID`.

---

## Notes & Known Issues

- `main.go` at the root (`package main` in `huma-template/main.go`) is a placeholder that only prints "Hello World" — the real entrypoint is `cmd/api/main.go`.
- `users.handler.go` registers two routes with identical paths (`GET /users/{id}`) for `get-user-by-id` and `get-user-by-field` — this will cause a router conflict. The `get-user-by-field` handler should use a different path (e.g., `/users/by/{field}/{value}`).
- `AuthMiddleware` has a logic bug: it calls `next(hc)` inside the `"public"` tag loop but does not `return` afterwards, so execution continues to the auth check even for public routes.
- The `ReqIDHook` in `pkg/logging/logging.go` will panic if `requestID` or `operationID` context values are not set (no nil-check before type assertion).
- `internal/ctx/ctx.go` `FromContext` will panic if the `"logger"` or `"userID"` context values are missing (unsafe type assertions).
- Redis is defined in `docker-compose.yml` but is not used anywhere in the current codebase.
- `updated_at` column is not automatically updated on row changes — consider a DB trigger or Bun hook.
