# SchemaGuard

Catch breaking API changes before they reach production.

SchemaGuard compares OpenAPI specifications and identifies changes that may break existing API consumers.

## What SchemaGuard Is

SchemaGuard is a Go-based CLI for checking API compatibility between two versions of an OpenAPI 3.x specification. The goal is to give developers and CI pipelines a fast way to detect breaking changes before they are merged or deployed.

## Why It Exists

API changes often look harmless in code review while still breaking downstream consumers. Removing response fields, tightening request validation, changing parameter types, or dropping endpoints can introduce production issues for clients that depend on the previous contract. SchemaGuard is intended to catch those risks early and report them clearly.

## Planned Usage

Phase 1 provides only the CLI foundation. OpenAPI comparison is planned for later phases.

```bash
schemaguard compare old.yaml new.yaml
```

Planned output categories include:

- removed endpoints or HTTP methods
- removed response fields
- newly required request fields
- incompatible parameter or schema type changes
- changed status codes and response structures

## Planned CI/CD Integration

SchemaGuard is designed to fit naturally into pull request validation:

```text
Pull Request
     |
GitHub Actions
     |
SchemaGuard
     |
Compare OpenAPI Specs
     |
Compatible? PASS / FAIL
```

The initial GitHub Actions workflow in this phase only runs the Go test suite. Automated OpenAPI compatibility checks will be added in later phases.

## Development

Requirements:

- Go 1.23+

Run the current CLI:

```bash
go run ./cmd/schemaguard --help
go run ./cmd/schemaguard --version
```

Run tests:

```bash
go test ./...
```

## Status

Implemented in Phase 1:

- Go module and CLI entrypoint
- `--help` and `--version`
- basic test coverage
- Dockerfile foundation
- GitHub Actions test workflow

Planned for later phases:

- OpenAPI loading and validation
- compatibility diff engine
- breaking-change rule evaluation
- machine-readable JSON output
- CI-focused reporting and examples
