# Engineering TODOs

This file tracks remediation work discovered during codebase audit.

## Priority 0 - Crash and Data Safety (do first)

- [ ] Harden all context value reads to avoid runtime panics.
  - [ ] Replace direct type assertions like `ctx.Value("k").(T)` with safe checks in:
    - [ ] `pkg/logging/logging.go`
    - [ ] `pkg/ctx/ctx.go`
    - [ ] `internal/middleware/common.mw.go`
    - [ ] `internal/middleware/metrics.mw.go`
  - [ ] Define safe defaults for missing values (`requestID`, `operationID`, `requestStart`, logger).
  - [ ] Add tests covering missing/malformed context keys.

- [ ] Fix login nil-pointer crash and align API contract.
  - [ ] Update login request DTO to truly support username OR email.
  - [ ] Add explicit validation: reject requests where both are empty.
  - [ ] Remove unsafe dereference in `internal/handlers/auth.handler.go`.
  - [ ] Update auth service query logic to consume either identifier cleanly.
  - [ ] Add tests for username login, email login, and invalid payloads.

- [ ] Fix migration rollback bug for refresh token table.
  - [ ] In `migrations/02_rtokens_table.down.sql`, change dropped table from `refreshtokens` to `refresh_tokens`.
  - [ ] Verify up/down roundtrip locally with migrate commands.
  - [ ] Add migration CI step to catch up/down mismatches in future.

## Priority 1 - Security and Correctness

- [ ] Remove SQL field interpolation risks in filter/sort logic.
  - [ ] Create strict allowlists for sortable and filterable fields.
  - [ ] Reject unknown fields/rules with 400 errors.
  - [ ] Ensure `in`/`nin` are parameterized correctly for Bun/Postgres.
  - [ ] Add tests for valid filters, invalid filters, and attempted SQL injection payloads.

- [ ] Standardize auth token and session handling behavior.
  - [ ] Confirm `refresh_tokens.token` stores expected value (`token id` vs full token string) and rename field/column if needed.
  - [ ] Implement or remove TODO paths for "log user out of all sessions".
  - [ ] Ensure revoked refresh token consistently invalidates derived access token checks.

- [ ] Repair logger consistency in `AuthService`.
  - [ ] Replace `s.log` usage with request-scoped `logging.FromCtx(ctx)`.
  - [ ] Remove unused persistent logger field from service struct if unnecessary.

## Priority 2 - Dead Code and Residue Cleanup

- [ ] Remove or implement unused config and features.
  - [ ] Either implement auth rate limiting using `Auth.RateLimit` or remove the field from config and docs.

- [ ] Remove unused functions/types or wire them properly.
  - [ ] `internal/dto/common.dto.go` -> `ResponseType[T]`
  - [ ] `pkg/db/pg.go` -> `Ping`, `DBHealthCheck`
  - [ ] `pkg/metrics/metrics.go` -> `GetMetrics`, `GetReg`, `OpsProcessed`
  - [ ] `pkg/ctx/ctx.go` -> `fromSvcContext`
  - [ ] `internal/svc/auth.svc.go` -> `Revoke`
  - [ ] `internal/svc/users.svc.go` -> `GetUserByField` (either expose endpoint or remove)

- [ ] Remove/comment-clean residue.
  - [ ] Delete stale commented route block in `internal/handlers/users.handler.go`.
  - [ ] Delete stale commented `ListenAndServe` line in `cmd/api/main.go`.

- [ ] Resolve intentionally unfinished endpoint(s).
  - [ ] Implement `UsersHandler.Create` or remove route registration until implemented.
  - [ ] Add tests for final behavior.

## Priority 3 - Architecture and Maintainability

- [ ] Break up `cmd/api/main.go` into focused components.
  - [ ] Extract config/bootstrap initialization.
  - [ ] Extract router + middleware wiring.
  - [ ] Extract service container/wiring.
  - [ ] Extract HTTP server startup/shutdown.
  - [ ] Move migration CLI path out of API entrypoint if possible.

- [ ] Simplify and harden config loading.
  - [ ] Review Viper/pflag interactions (single parse, deterministic precedence).
  - [ ] Normalize YAML key style and ensure tags/docs/examples match runtime behavior.
  - [ ] Add unit tests for config precedence (default < file < env < flags).

- [ ] Standardize DTO validation tags.
  - [ ] Replace inconsistent tags (`MaxLength`, `Email:"true"`) with one canonical style.
  - [ ] Confirm OpenAPI output and runtime validation are aligned.

- [ ] Improve error handling strategy.
  - [ ] Reduce raw `panic` usage in runtime paths where graceful startup failure is preferred.
  - [ ] Keep migration CLI behavior explicit and non-ambiguous for invalid commands.

## Priority 4 - Quality Gates and Tests

- [ ] Add automated tests (currently none exist).
  - [ ] Unit tests: token provider, filter parsing/apply, user service CRUD.
  - [ ] Middleware tests: auth/public tags, metrics labels, request logging with missing context.
  - [ ] Integration tests: signup -> login -> refresh -> revoke/logout -> verify flow.
  - [ ] Migration tests: apply all migrations and rollback one step.

- [ ] Add CI safeguards.
  - [ ] Run `go test ./...`.
  - [ ] Run `go vet ./...`.
  - [ ] Add linter/static checks (e.g., `staticcheck`) for dead code and unsafe assertions.

## Priority 5 - Infra and Documentation Consistency

- [ ] Decide Redis direction.
  - [ ] If not needed, remove Redis service from `docker-compose.yml` and docs.
  - [ ] If needed, define ownership and first concrete usage (cache/session/rate-limit).

- [ ] Keep docs in sync with actual behavior.
  - [ ] Update README config examples and key naming to match loader behavior.
  - [ ] Document implemented auth login identifier rules (username/email).
  - [ ] Document filter/sort allowlists and expected request formats.

## Suggested Execution Order

1. P0 crash/data fixes
2. P1 security/correctness
3. P2 dead code cleanup
4. P3 architecture cleanup
5. P4 tests + CI gates
6. P5 infra/docs final alignment
