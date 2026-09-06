# SchemaGuard

Catch breaking API changes before they reach production.

SchemaGuard compares OpenAPI specifications and identifies changes that may break existing API consumers.

## What SchemaGuard Is

SchemaGuard is a Go-based CLI for checking API compatibility between two versions of an OpenAPI 3.x specification. The goal is to give developers and CI pipelines a fast way to detect breaking changes before they are merged or deployed.

## Why It Exists

API changes often look harmless in code review while still breaking downstream consumers. Removing response fields, tightening request validation, changing parameter types, or dropping endpoints can introduce production issues for clients that depend on the previous contract. SchemaGuard is intended to catch those risks early and report them clearly.

## Usage

```bash
schemaguard compare old.yaml new.yaml
```

SchemaGuard accepts YAML or JSON OpenAPI 3.x documents. It validates the required `openapi`, `info`, `paths`, operation `responses`, and response `description` fields before comparing files. It exits with `0` when no breaking changes are found, `1` when it finds breaking changes, and `2` for invalid command arguments or specifications.

Local schema references such as `#/components/schemas/Pet` are resolved before comparison in both YAML and JSON documents, including references nested in properties and array items. Missing, malformed, circular, and external file or URL references are reported as invalid specifications. External references are not loaded.

The first comparison rules detect:

- removed paths
- removed HTTP operations
- removed response status codes and media types
- removed nested response properties
- changed response property types

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

The included GitHub Actions workflow runs the Go test suite. Add `schemaguard compare` to a pull request workflow to prevent incompatible API changes from being merged.

## Development

Requirements:

- Go 1.23+

Run the current CLI:

```bash
go run ./cmd/schemaguard --help
go run ./cmd/schemaguard --version
go run ./cmd/schemaguard compare old.yaml new.yaml
```

Run tests:

```bash
go test ./...
```

## Status

Implemented:

- CLI with `compare`, `--help`, and `--version`
- OpenAPI 3.x YAML and JSON loading
- path and HTTP operation removal checks
- CI-friendly exit codes
- Dockerfile and GitHub Actions test workflow

Planned for later phases:

- machine-readable JSON output
- response, request, parameter, and schema compatibility rules
