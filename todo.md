# Engineering TODOs

This file tracks remediation work discovered during codebase audit.

## Priority 1 - Security and Correctness

- [ ] Remove SQL field interpolation risks in filter/sort logic.
  - [ ] Create strict allowlists for sortable and filterable fields.
  - [ ] Reject unknown fields/rules with 400 errors.
  - [ ] Ensure `in`/`nin` are parameterized correctly for Bun/Postgres.

- [ ] Standardize auth token and session handling behavior.
  - [ ] Confirm `refresh_tokens.token` stores expected value (`token id` vs full token string) and rename field/column if needed.
  - [ ] Implement or remove TODO paths for "log user out of all sessions".
  - [ ] Ensure revoked refresh token consistently invalidates derived access token checks.

- [ ] Repair logger consistency in `AuthService`.
  - [ ] Replace `s.log` usage with request-scoped `logging.FromCtx(ctx)`.
  - [ ] Remove unused persistent logger field from service struct if unnecessary.

## Priority 2 - Architecture and Maintainability

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

- [x] Standardize DTO validation tags.
  - [x] Replace inconsistent tags (`maxLength`, `Email:"true"`) with one canonical style.

## Priority 3 - Features

- [ ] Add Logout endpoint (use revoke fn)
- [ ] Add GetUserByField endpoint
- [ ] Implement `UsersHandler.Create`
