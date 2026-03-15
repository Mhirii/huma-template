# AGENT.md — Coding Guide

This file tells you how to write code for this codebase. Follow these rules exactly. New code must be indistinguishable in style from the existing code.

---

## Architecture

The server is layered. Each layer has a single job and must not reach into the layer below it.

```
HTTP request
    → Huma middleware (auth, metrics, logging)
    → Handler         (parse input, call service, map to response)
    → Service         (business logic, DB access)
    → Bun ORM         (PostgreSQL)
```

- **Handlers** know about DTOs and services. They do not touch `*bun.DB` directly.
- **Services** know about models, DTOs, and `*bun.DB`. They do not know about HTTP.
- **Models** are plain Bun structs. They have no methods except what Bun needs.
- **DTOs** are plain structs. They have no methods except shared helpers (`ParseFilters`, `ApplyFilters`).

---

## File naming

Every file is named `<domain>.<role>.go`. Do not deviate.

```
internal/handlers/thing.handler.go
internal/svc/thing.svc.go
internal/models/thing.go
internal/dto/thing.dto.go
internal/middleware/thing.mw.go
internal/config/thing.cfg.go
pkg/thing/thing.go
migrations/03_things_table.up.sql
migrations/03_things_table.down.sql
```

---

## Adding a new resource — checklist

1. Migration pair in `migrations/`
2. Model in `internal/models/`
3. DTOs in `internal/dto/`
4. Service in `internal/svc/`
5. Handler in `internal/handlers/`
6. Register routes in `cmd/api/main.go`

Do all six. Never skip steps or merge them.

---

## Models

Bun model structs live in `internal/models/`. They map 1:1 to database tables.

```go
package models

import (
    "time"
    "github.com/uptrace/bun"
)

type Thing struct {
    bun.BaseModel `bun:"table:things,alias:t"`

    ThingID   string    `bun:"id,pk"`
    Name      string    `bun:"name"`
    OwnerID   string    `bun:"owner_id"`
    CreatedAt time.Time `bun:"created_at,default:current_timestamp"`
    UpdatedAt time.Time `bun:"updated_at,default:current_timestamp"`
}
```

Rules:
- The primary key field is always named `<Type>ID` (e.g. `ThingID`, `UserID`).
- The `bun` tag for the PK is always `bun:"id,pk"`.
- Use `bun:"table:things,alias:t"` — table name plural, alias the first letter of the singular noun.
- `CreatedAt` and `UpdatedAt` always use `default:current_timestamp`.
- No methods on model structs.
- Never return a model directly from a service. Always convert to a DTO first.

---

## DTOs

DTOs live in `internal/dto/`. One file per domain.

### Request types

```go
// Protected endpoint — embed AuthHeader
type CreateThingReq struct {
    AuthHeader
    Body CreateThingReqBody
}
type CreateThingReqBody struct {
    Name    string `json:"name"    doc:"Name of the thing" minLength:"1" maxLength:"255" required:"true"`
    OwnerID string `json:"owner_id" doc:"Owner user ID"     required:"true"`
}

// Path parameter endpoint — embed AuthHeader, add path field
type GetThingByIDReq struct {
    AuthHeader
    ID string `path:"id" doc:"ID of the thing" required:"true"`
}

// List endpoint — embed AuthHeader and ListQuery
type ListThingsReq struct {
    AuthHeader
    ListQuery
}
```

Rules:
- Every protected endpoint embeds `AuthHeader`. Public endpoints do not.
- Every list endpoint embeds `ListQuery`. No exceptions.
- All request body fields have `doc:` tags. Use `minLength`, `maxLength`, `format`, `enum` where appropriate.
- Never put logic in DTOs.

### Response types

```go
type CreateThingRes struct{ Body ThingModelRes }
type GetThingByIDRes struct{ Body ThingModelRes }
type ListThingsRes  struct{ Body ListThingsResBody }

type ListThingsResBody struct {
    Things    []ThingModelRes `json:"things"`
    Total     int             `json:"total"`
    ListQuery ListQueryRes    `json:"query"`
}

type ThingModelRes struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    OwnerID   string `json:"owner_id"`
    CreatedAt int    `json:"created_at"`
    UpdatedAt int    `json:"updated_at"`
}
```

Rules:
- Every response type has a single `Body` field. Huma requires this.
- List responses always include `Total int` and `ListQuery ListQueryRes`.
- Timestamps in responses are Unix seconds (`int`), not `time.Time`.
- Optional fields use pointer types and `omitempty`.
- The "model response" struct (e.g. `ThingModelRes`) is the canonical shape returned by all endpoints for that resource.

---

## Services

Services live in `internal/svc/`. One file per domain.

### Constructor

```go
type ThingService struct {
    db  *bun.DB
    log zerolog.Logger  // only if the service needs a persistent logger field
}

func NewThingService(db *bun.DB) (*ThingService, error) {
    return &ThingService{db: db}, nil
}
```

Rules:
- Constructor always returns `(*Service, error)` even if it cannot currently fail. This makes future changes non-breaking.
- Only `*bun.DB` and other services go in the struct. No HTTP types. No config values (pass them to `NewXxxService` if needed).

### Method signature

Every service method takes `ctx.ServiceContext` as its first argument. Never use `context.Context` directly.

```go
func (s *ThingService) GetThingByID(ctx ctx.ServiceContext, id string) (*dto.ThingModelRes, error) {
```

### Logging

Always get the logger from context at the top of the method. Enrich it with the fields relevant to the operation, then reassign.

```go
func (s *ThingService) GetThingByID(ctx ctx.ServiceContext, id string) (*dto.ThingModelRes, error) {
    log := logging.FromCtx(ctx)
    log.Debug().Str("thing_id", id).Msg("fetching thing by id")
    ...
}

func (s *ThingService) CreateThing(ctx ctx.ServiceContext, data *dto.CreateThingReqBody) (*dto.ThingModelRes, error) {
    log := logging.FromCtx(ctx)
    log = log.With().Str("name", data.Name).Str("owner_id", data.OwnerID).Logger()
    ctx = ctx.NewServiceContext(ctx, log)  // propagate enriched logger
    log.Debug().Msg("creating thing")
    ...
}
```

Never call `logging.L()` inside a service method. Always use `logging.FromCtx(ctx)`.

### DB operations

Follow these exact patterns for every DB operation type:

**Select by PK:**
```go
m := models.Thing{ThingID: id}
if err := s.db.NewSelect().Model(&m).WherePK("id").Scan(ctx, &m); err != nil {
    if strings.Contains(err.Error(), "no rows") {
        log.Warn().Str("thing_id", id).Msg("thing not found")
        return nil, huma.Error404NotFound("thing not found")
    }
    log.Error().Err(err).Str("thing_id", id).Msg("failed to fetch thing")
    return nil, huma.Error500InternalServerError(err.Error())
}
```

**Select with filter:**
```go
m := models.Thing{}
if err := s.db.NewSelect().Model(&m).Where("owner_id = ?", ownerID).Scan(ctx, &m); err != nil {
    if strings.Contains(err.Error(), "no rows") {
        return nil, huma.Error404NotFound("thing not found")
    }
    log.Error().Err(err).Msg("failed to fetch thing")
    return nil, huma.Error500InternalServerError(err.Error())
}
```

**Insert:**
```go
thingID := ulid.Make().String()
m := models.Thing{
    ThingID: thingID,
    Name:    data.Name,
    OwnerID: data.OwnerID,
}
if _, err := s.db.NewInsert().Model(&m).Returning("*").Exec(ctx, &m); err != nil {
    log.Error().Err(err).Msg("failed to insert thing")
    // Check for unique constraint violations before returning generic 500
    if strings.Contains(err.Error(), "things_name_key") {
        return nil, huma.Error400BadRequest("name already exists")
    }
    return nil, huma.Error500InternalServerError(err.Error())
}
log.Info().Str("thing_id", m.ThingID).Msg("created thing")
```

**Update:**
```go
m := models.Thing{ThingID: id, Name: data.Name}
if err := s.db.NewUpdate().Model(&m).OmitZero().WherePK("id").Returning("*").Scan(ctx, &m); err != nil {
    if strings.Contains(err.Error(), "no rows") {
        return nil, huma.Error404NotFound("thing not found")
    }
    log.Error().Err(err).Str("thing_id", id).Msg("failed to update thing")
    return nil, huma.Error500InternalServerError(err.Error())
}
log.Info().Str("thing_id", id).Msg("updated thing")
```

**Delete:**
```go
m := models.Thing{ThingID: id}
if _, err := s.db.NewDelete().Model(&m).WherePK("id").Exec(ctx); err != nil {
    if strings.Contains(err.Error(), "no rows") {
        return nil, huma.Error404NotFound("thing not found")
    }
    log.Error().Err(err).Str("thing_id", id).Msg("failed to delete thing")
    return nil, huma.Error500InternalServerError(err.Error())
}
log.Info().Str("thing_id", id).Msg("deleted thing")
```

**Transaction:**
```go
tx, err := s.db.BeginTx(ctx, nil)
if err != nil {
    log.Error().Err(err).Msg("failed to start transaction")
    return nil, huma.Error500InternalServerError(err.Error())
}
defer tx.Rollback()

// use tx.NewInsert(), tx.NewUpdate(), etc.
// pass &tx to other service methods that accept it

if err := tx.Commit(); err != nil {
    log.Error().Err(err).Msg("failed to commit transaction")
    return nil, huma.Error500InternalServerError(err.Error())
}
```

When a service method may participate in a caller's transaction, accept `*bun.Tx` as an optional argument and fall back to `s.db`:

```go
func (s *ThingService) CreateThing(ctx ctx.ServiceContext, data *dto.CreateThingReqBody, tx *bun.Tx) (*dto.ThingModelRes, error) {
    newInsert := s.db.NewInsert()
    if tx != nil {
        newInsert = tx.NewInsert()
    }
    ...
}
```

**List with pagination and filters:**
```go
func (s *ThingService) GetThings(ctx ctx.ServiceContext, params *dto.ListThingsReq) (*dto.ListThingsRes, error) {
    log := logging.FromCtx(ctx)
    log.Debug().Int("page", params.Page).Int("per_page", params.PerPage).Msg("listing things")

    var things []models.Thing
    res := &dto.ListThingsRes{
        Body: dto.ListThingsResBody{Total: 0, ListQuery: dto.ListQueryRes{}, Things: nil},
    }

    q := s.db.NewSelect().Model(&things)

    if params.Search != "" {
        search := "%" + params.Search + "%"
        q = q.Where("name ILIKE ?", search)
    }

    filters, err := dto.ParseFilters(params.Filters)
    if err != nil {
        log.Warn().Err(err).Str("filters", params.Filters).Msg("invalid filters, ignoring")
    }
    q = dto.ApplyFilters(filters, q)

    total, err := q.Clone().Count(ctx)
    if err != nil {
        log.Error().Err(err).Msg("failed to count things")
        return nil, huma.Error500InternalServerError(err.Error())
    }
    res.Body.Total = total

    q = q.Order(params.SortBy + " " + params.SortDir)
    q = q.Limit(params.PerPage)
    q = q.Offset(params.PerPage * (params.Page - 1))

    if err := q.Scan(ctx, &things); err != nil {
        if strings.Contains(err.Error(), "no rows") {
            return res, nil
        }
        log.Error().Err(err).Msg("failed to scan things")
        return nil, huma.Error500InternalServerError(err.Error())
    }

    resThings := []dto.ThingModelRes{}
    for _, t := range things {
        resThings = append(resThings, *s.ModelToRes(&t))
    }
    res.Body.Things = resThings
    res.Body.ListQuery = dto.ListQueryRes{
        Page: params.Page, PerPage: params.PerPage,
        SortBy: params.SortBy, SortDir: params.SortDir,
        Search: params.Search, Includes: params.Includes, Filters: filters,
    }
    log.Info().Int("results", len(resThings)).Msg("retrieved things")
    return res, nil
}
```

### Model → DTO conversion

Every service owns a `ModelToRes` method and a package-level converter. Both call the same implementation.

```go
func (s *ThingService) ModelToRes(m *models.Thing) *dto.ThingModelRes {
    return ThingModelToRes(m)
}

func ThingModelToRes(m *models.Thing) *dto.ThingModelRes {
    if m == nil {
        return nil
    }
    res := &dto.ThingModelRes{}
    res.ID = m.ThingID
    res.Name = m.Name
    res.OwnerID = m.OwnerID
    if !m.CreatedAt.IsZero() {
        res.CreatedAt = int(m.CreatedAt.Unix())
    }
    if !m.UpdatedAt.IsZero() {
        res.UpdatedAt = int(m.UpdatedAt.Unix())
    }
    return res
}
```

Rules:
- Always nil-check the model.
- Always guard zero-value time fields with `!m.Time.IsZero()`.
- The package-level function (`ThingModelToRes`) is exported so other services can use it without a dependency on the service struct.

---

## Handlers

Handlers live in `internal/handlers/`. They are thin. The only logic allowed in a handler is:
- Extracting and validating input fields that Huma cannot validate for you (e.g. parsing a `Bearer` header).
- Calling the service.
- Mapping the service result to a response DTO.

```go
package handlers

import (
    "context"
    "net/http"

    "github.com/danielgtaylor/huma/v2"
    "github.com/mhirii/huma-template/internal/dto"
    "github.com/mhirii/huma-template/internal/svc"
    "github.com/mhirii/huma-template/pkg/ctx"
    "github.com/rs/zerolog"
)

type ThingHandler struct {
    svc *svc.ThingService
    log zerolog.Logger
}

func RegisterThingRoutes(api huma.API, svc *svc.ThingService) {
    h := &ThingHandler{svc: svc}
    g := huma.NewGroup(api, "/things")
    g.UseSimpleModifier(func(op *huma.Operation) {
        op.Tags = []string{"Things"}
    })

    huma.Register(g, huma.Operation{
        OperationID:   "create-thing",
        Method:        http.MethodPost,
        Path:          "/",
        Summary:       "Create a Thing",
        Description:   "Create a new Thing",
        DefaultStatus: http.StatusOK,
    }, h.Create)

    huma.Register(g, huma.Operation{
        OperationID:   "get-thing-by-id",
        Method:        http.MethodGet,
        Path:          "/{id}",
        Summary:       "Get a Thing",
        Description:   "Get a Thing by ID",
        DefaultStatus: http.StatusOK,
    }, h.Get)
}

func (h *ThingHandler) Create(c context.Context, input *dto.CreateThingReq) (*dto.CreateThingRes, error) {
    ctx := ctx.FromContext(c)
    result, err := h.svc.CreateThing(ctx, &input.Body, nil)
    if err != nil {
        return nil, err
    }
    return &dto.CreateThingRes{Body: *result}, nil
}

func (h *ThingHandler) Get(c context.Context, input *dto.GetThingByIDReq) (*dto.GetThingByIDRes, error) {
    ctx := ctx.FromContext(c)
    result, err := h.svc.GetThingByID(ctx, input.ID)
    if err != nil {
        return nil, err
    }
    return &dto.GetThingByIDRes{Body: *result}, nil
}
```

Rules:
- The first line of every handler is `ctx := ctx.FromContext(c)`.
- Errors from services are already wrapped in `huma.Error*` — pass them through as-is (`return nil, err`).
- Never call `huma.Error*` in a handler unless you are validating the raw HTTP request (e.g. parsing an `Authorization` header yourself, as in `LogoutHandler`).
- Never touch `s.db` in a handler.
- `RegisterThingRoutes` always accepts `api huma.API` and the service pointer. No other arguments.
- Group all routes for a domain under a single `huma.NewGroup` with a shared tag.
- `OperationID` is kebab-case, verb first: `"create-thing"`, `"get-thing-by-id"`, `"list-things"`, `"update-thing"`, `"delete-thing"`.

---

## Error handling

Use only `huma.Error*` functions for errors that reach the HTTP response. Never use `errors.New`, `fmt.Errorf`, or `errors.Wrap` at the service or handler layer.

| Situation | Function |
|---|---|
| Resource not found | `huma.Error404NotFound("thing not found")` |
| Invalid input from the client | `huma.Error400BadRequest("reason")` |
| Unauthenticated | `huma.Error401Unauthorized("reason")` |
| Forbidden | `huma.Error403Forbidden("reason")` |
| Unique constraint violation | `huma.Error400BadRequest("field already exists")` |
| Unexpected DB or internal error | `huma.Error500InternalServerError(err.Error())` |
| Not yet implemented | `huma.Error501NotImplemented("not implemented")` |

When checking for DB-level constraint violations, use `strings.Contains(err.Error(), "<constraint_name>")`. Name your DB constraints explicitly in migrations so these checks are stable.

Always log the underlying error before wrapping it:
```go
log.Error().Err(err).Msg("failed to do the thing")
return nil, huma.Error500InternalServerError(err.Error())
```

For "not found" via "no rows", log at `Warn` level, not `Error`:
```go
if strings.Contains(err.Error(), "no rows") {
    log.Warn().Str("thing_id", id).Msg("thing not found")
    return nil, huma.Error404NotFound("thing not found")
}
```

---

## Logging

**Rules:**
- Get the logger from context at the top of every service method: `log := logging.FromCtx(ctx)`.
- Enrich the logger with the key identifiers for the operation (IDs, usernames, etc.) using `.With().Str(...).Logger()`. Reassign to `log`.
- Use `Debug` for entry points ("fetching thing by id"), `Info` for successful mutations ("created thing"), `Warn` for expected failures ("thing not found"), `Error` for unexpected failures.
- Always attach `Err(err)` on error log lines.
- Always attach the relevant ID as a field on error/warn lines.
- Message strings are lowercase, past tense for completed actions, present continuous for in-progress.
- Never use `fmt.Println` or the stdlib `log` package. The only exception is `log/slog` in the migration CLI which pre-dates the logger setup.

**Good:**
```go
log.Debug().Str("thing_id", id).Msg("fetching thing by id")
log.Info().Str("thing_id", m.ThingID).Msg("created thing")
log.Warn().Str("thing_id", id).Msg("thing not found")
log.Error().Err(err).Str("thing_id", id).Msg("failed to fetch thing")
```

**Bad:**
```go
fmt.Println("fetching thing")
log.Info().Msg("Error: " + err.Error())
log.Error().Msg("thing not found")  // not found is Warn, not Error
```

---

## Middleware

Huma middleware functions have the signature `func(hc huma.Context, next func(huma.Context))`.

```go
func ThingMiddleware(hc huma.Context, next func(huma.Context)) {
    ctx := hc.Context()

    // read context values
    start := ctx.Value("requestStart").(time.Time)

    // inject new context values before calling next
    ctx = context.WithValue(ctx, "myKey", myValue)
    hc = huma.WithContext(hc, ctx)

    next(hc)

    // post-processing happens here, after next() returns
}
```

Rules:
- Pre-processing (auth, enriching context) happens before `next(hc)`.
- Post-processing (metrics, logging the response) happens after `next(hc)`.
- Use `huma.WithContext(hc, ctx)` to propagate a modified context into the chain.
- Route tags drive middleware behaviour. Check `hc.Operation().Tags` to opt routes in or out.
- Register middleware with `api.UseMiddleware(...)` in `cmd/api/main.go`, in this order:
  1. `MetricsMiddleware`
  2. `GeneralMiddleware`
  3. _(register public routes)_
  4. `AuthMiddleware`
  5. _(register protected routes)_

---

## Config

Config types go in `internal/config/types.cfg.go`. Every field must have all four tags:

```go
type ThingConfig struct {
    MaxItems int    `flag:"thing_max_items" env:"THING_MAX_ITEMS" yaml:"thing_max_items" default:"100" validate:"min=1,max=10000"`
    APIKey   string `flag:"thing_api_key"   env:"THING_API_KEY"   yaml:"thing_api_key"   validate:"required"`
}
```

Add the new config block to `APIConfig` in `internal/config/api.cfg.go` and call `cfg.BindConfigStruct` and `cfg.ValidateConfigStruct` for it, following the existing pattern.

---

## Migrations

- Files are numbered sequentially: `01_`, `02_`, `03_`, …
- Always create both `.up.sql` and `.down.sql`.
- The `.down.sql` must fully reverse the `.up.sql`.
- Name all constraints explicitly so they can be referenced in Go code: `CONSTRAINT things_name_key UNIQUE (name)`.
- Always use `IF NOT EXISTS` / `IF EXISTS` in migration SQL.
- IDs are `TEXT PRIMARY KEY` (ULIDs, not serial integers).
- Always include `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()` and `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`.
- Foreign keys always specify `ON DELETE CASCADE` unless there is a strong reason not to.

```sql
-- 03_things_table.up.sql
CREATE TABLE IF NOT EXISTS things (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    owner_id   TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT things_name_owner_key UNIQUE (name, owner_id)
);
```

```sql
-- 03_things_table.down.sql
DROP TABLE IF EXISTS things;
```

Migrations are auto-discovered via `migrations.DiscoverCaller()` in `migrations/migrate.go`. You do not need to register individual files — just create them in the `migrations/` directory.

---

## IDs

Always use ULIDs for new primary keys:

```go
import "github.com/oklog/ulid/v2"

id := ulid.Make().String()
```

Never use `uuid`, `rand`, or sequential integers.

---

## Wiring (`cmd/api/main.go`)

When adding a new service and handler, follow this pattern inside the `humacli.New` callback:

```go
thingSvc, err := svc.NewThingService(dbconn)
if err != nil {
    l.Err(err).Msg("Failed to create thing service, this is a critical module, exiting")
    os.Exit(1)
}

// register routes AFTER AuthMiddleware if the resource is protected
handlers.RegisterThingRoutes(api, thingSvc)
```

Critical services (ones the API cannot function without) must call `os.Exit(1)` on failure, with a log message that includes "this is a critical module, exiting".

---

## Reusable parts — quick reference

| What you need | What to use |
|---|---|
| Convert huma context to service context | `ctx.FromContext(c)` — first line of every handler |
| Request-scoped logger | `logging.FromCtx(ctx)` — first line of every service method |
| New unique ID | `ulid.Make().String()` |
| Hash a password | `bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)` |
| Verify a password | `bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))` |
| Pagination + sort params on a request | Embed `dto.ListQuery` |
| Auth header on a request | Embed `dto.AuthHeader` |
| Apply filters to a Bun query | `dto.ParseFilters(raw)` then `dto.ApplyFilters(filters, q)` |
| Return a not-found error | `huma.Error404NotFound("thing not found")` |
| Return a bad-request error | `huma.Error400BadRequest("reason")` |
| Return an internal error | `huma.Error500InternalServerError(err.Error())` |
| Opt out of auth in a route | Add `"public"` to `op.Tags` |
| Run something in a transaction | `s.db.BeginTx` → `defer tx.Rollback()` → `tx.Commit()` |
| Participate in a caller's transaction | Accept `*bun.Tx` as last arg, fall back to `s.db` if nil |

---

## What not to do

- Do not return `error` from a service without wrapping it in `huma.Error*`.
- Do not query the database in a handler.
- Do not call `logging.L()` inside a service method.
- Do not use `context.Background()` in a service method.
- Do not return a model struct from a service. Always convert with `ModelToRes`.
- Do not create a new `zerolog.Logger` with `zerolog.New(...)` anywhere outside `pkg/logging`.
- Do not add business logic to DTOs or models.
- Do not register the same path twice on the same method (see the existing `GET /users/{id}` duplicate — that is a bug, do not repeat it).
- Do not use `fmt.Errorf` or `errors.New` for user-facing errors.
- Do not use `int` or `uuid` for primary keys — always `TEXT` / ULID.
- Do not skip the `.down.sql` migration file.
- Do not put config loading logic anywhere outside `internal/config/`.