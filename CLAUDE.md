# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is a fork of `uber-go/mock` (itself a fork of `golang/mock`) extended with additional code generation modes. Module path: `github.com/pableeee/implgen`.

Two main packages:
- `gomock/` — runtime library consumed by generated mocks (Controller, Call, matchers)
- `mockgen/` — code generator binary (`go build -o mockgen ./mockgen`)

## Commands

```sh
# Build the generator
go build -o mockgen ./mockgen

# Run all tests
go test ./...

# Run a single test
go test ./gomock/... -run TestFoo
go test ./mockgen/... -run TestBar

# Run integration tests (each has its own go.mod)
cd mockgen/internal/tests/generics && go test ./...
cd mockgen/internal/tests/typed && go test ./...

# Panic-handling regression
bash ci/check_panic_handling.sh

# Vet
go vet ./...
```

## Architecture

### Generator (`mockgen/`)

`mockgen.go` is the entry point. It supports two **parsing modes**:
- **Source mode** (`-source file.go`): parses Go AST directly from a `.go` file
- **Reflect mode** (`mockgen <package> <Interface1,Interface2,...>`): uses `go run` subprocess to extract interface metadata via reflection

After parsing, it dispatches to one of three **generation functions** based on `-implementation_type`:

| Flag value | Generator function | Output type |
|---|---|---|
| `mock` (default) | `generateMockInterface` (`mockimpl.go`) | gomock-based mock with `EXPECT()` recorder |
| `trace` | `generateTracedInterface` (`tracegen.go`) | OpenTelemetry tracing decorator |
| `metrics` | `generateMetricsInterface` (`metricsgen.go`) | Prometheus histogram metrics decorator |

The `generator` struct in `mockgen.go` holds shared state (package map, identifier allocator, output buffer). All three generators write to this struct via `g.p(...)`.

### Key files

- `mockgen/mockgen.go` — flags, `main()`, `generator` struct, `Generate()` method, source/reflect mode dispatch
- `mockgen/mockimpl.go` — mock struct + EXPECT recorder generation
- `mockgen/tracegen.go` — tracing wrapper: wraps methods with `otel.Tracer(...).Start(ctx, method)`, records errors on span
- `mockgen/metricsgen.go` — metrics wrapper: records Prometheus histogram duration per method, labels with `error=true/false`
- `mockgen/generic.go` — generic type parameter parsing (`ast.IndexExpr`, `ast.IndexListExpr`)
- `mockgen/model/model.go` — interface/method/type AST model shared across all parsers

### Tracing and metrics decorators

Both `tracegen.go` and `metricsgen.go` generate a **decorator pattern**: a struct holding a `delegate` field of a narrowed interface (e.g. `TracedFoo` or `MetricsFoo`), wrapping each method call. The narrowed interface is generated alongside the struct so callers must use the narrowed type, not the original.

Tracing: only instruments methods whose first argument is `context.Context`. Errors are recorded via `span.RecordError` + `span.SetStatus`.

Metrics: always instruments all methods. Uses `go-kit/metrics` Prometheus histogram with `error` label.

### Integration tests

`mockgen/internal/tests/` contains per-scenario directories. Each has an `input.go` (interface definitions) and checked-in `mock.go` output. Several (e.g. `generics/`, `typed/`) are separate Go modules with their own `go.mod` because they require different module constraints.

The `sample/` directory contains hand-written usage examples used by gomock's own tests.
