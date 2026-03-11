# Henetdns Repo Map

Load this file when the task touches more than one package, changes the CLI contract, or depends on Hurricane Electric site behavior.

## Package Map

- `cmd/henetdns/main.go`: CLI entrypoint.
- `internal/cli`: Cobra commands, flags, and cache-control behavior for `login`, `zones`, and `records`.
- `internal/app`: Runtime wiring. `runtime.go` constructs config, SQLite store, cookie jar, HTTP client, auth service, and HE service.
- `internal/config`: Flag and environment loading plus common validation.
- `internal/auth`: Login, session restore, cookie serialization, and login success markers.
- `internal/httpclient`: Shared HTTP transport, retries, timeout handling, referer/origin headers.
- `internal/henet`: HE-specific page paths, HTML parsers, zone/record service logic, and record mutations.
- `internal/store`: SQLite schema plus repositories for sessions, zones cache, records cache, and audit logs.
- `internal/output`: Human-readable and JSON output helpers.
- `docs/curl.md`: Canonical notes about HE login behavior and curl-based debugging.

## Important Request Flows

### Login

1. `internal/cli/login.go` calls `app.WithRuntime`.
2. `internal/app/runtime.go` wires `auth.Service`.
3. `internal/auth/service.go` performs a bootstrap `GET /` before posting credentials.
4. Successful login is determined from response body markers, not just `Set-Cookie`.
5. Session cookies are serialized into SQLite through `SessionRepo`.

### Zones And Records Listing

1. `internal/cli/zones.go` and `internal/cli/records.go` implement cache-first behavior.
2. `--cache-only` must never hit the network.
3. `--refresh` must bypass cache and refresh local SQLite state.
4. Zone-name lookup may resolve from cache first, then remote when allowed.

### Record Mutation

1. `internal/cli/records.go` validates flags and builds `henet.RecordInput`.
2. `internal/henet/actions.go` normalizes input and fetches current records.
3. Upsert is idempotent by exact-match detection.
4. Delete requires an exact record match and refuses locked records.
5. After mutation, the service reloads remote records and treats absence/presence as verification.

## Behavior Contracts To Preserve

- HE login requires `GET` before `POST`. `docs/curl.md` explains why.
- `auth.IsLoggedInBody` and `auth.IsLoginPage` are parser contracts. If site markers change, update tests with the code.
- `ListZones` and `ListRecords` refresh cache after successful remote fetches.
- Zone and record cache schemas live in `internal/store/db.go`; repository changes usually need migration-awareness and test updates.
- `RecordInput` matching includes MX priority when explicitly set.
- User-visible errors wrap sentinel errors from `internal/errs`.

## Test Map

- `internal/auth/markers_test.go`: login page and success-marker coverage.
- `internal/henet/parser_test.go`: HTML parsing expectations for zones and records.
- `internal/httpclient/client_test.go`: retry and transport behavior.
- `internal/store/session_repo_test.go`: session persistence behavior.

Use targeted tests while iterating, then finish with:

```bash
go test ./...
```

## Change Heuristics

- When adding a flag or command behavior, update both README files if the user-facing contract changed.
- When touching parsers, prefer fixture-driven tests over live requests.
- When touching auth or HTTP logic, verify headers, referer/origin handling, and cookie persistence together.
- When touching cache behavior, reason through both zone-name and zone-id paths so cache misses and refreshes still behave correctly.
