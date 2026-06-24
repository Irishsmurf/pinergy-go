# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.2.0] - 2026-06-24

### Security

- Fix auth token leaking to unauthenticated endpoints
- Fix critical security, performance, and API usability issues (#101)
- Document PII exposure in CheckEmail header transmission

### Fixed

- Add max-size guard to prevent caching oversized responses
- Use v2.5 for golangci-lint version (action requires v1.2 format)

### Changed

- Replace `[]any` with `[]json.RawMessage` for AutoTopUps (type safety)
- Replace variadic bool with plain bool, guard `rand.Int64N` panic
- Enable gosec, bodyclose, noctx, exhaustive, and gocritic linters

### Added

- Dependabot configuration for Go modules and GitHub Actions
- Go mod tidy verification step in CI
- Exhaustive coverage for `ErrorCode.String()`

### Changed

- Update golangci-lint from v2.0.0 to latest v2.x in CI

## [v1.1.0] - 2026-03-16

### Changed

- Optimize password hashing performance
- Split device management concerns and simplify version caching
- Replace `math.Pow` with bitwise shift for exponential backoff
- Optimize UnixTime JSON marshaling and unmarshaling allocations

## [v1.0.0] - 2026-03-16

Stable release. All endpoints fully tested with integration tests and comprehensive unit test coverage (80%+).

## [v0.2.0] - 2026-03-16

### Added

- Print GetVersion output in basic example

## [v0.1.0] - 2026-03-15

### Added

- Full API coverage: balance, usage, level-pay intervals, comparison, top-up config, notifications, app version
- Rate limiting via `golang.org/x/time/rate` (2 req/s, burst 5)
- Automatic retries with exponential back-off and jitter on 5xx errors
- In-memory TTL cache with per-endpoint TTLs (60s-30min)
- Context-aware methods with cancellation propagation
- Thread-safe `*Client` for concurrent use
- Typed errors with `errors.Is` / `errors.As` support and `ErrorCode` enum
- Examples for basic usage and monitoring

[v1.2.0]: https://github.com/Irishsmurf/pinergy-go/compare/v1.1.0...v1.2.0
[v1.1.0]: https://github.com/Irishsmurf/pinergy-go/compare/v1.0.0...v1.1.0
[v1.0.0]: https://github.com/Irishsmurf/pinergy-go/compare/v0.2.0...v1.0.0
[v0.2.0]: https://github.com/Irishsmurf/pinergy-go/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/Irishsmurf/pinergy-go/releases/tag/v0.1.0
