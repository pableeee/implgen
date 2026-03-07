# implgen

implgen is a fork of [uber-go/mock](https://github.com/uber-go/mock) that extends `mockgen` with additional code generation modes beyond mocks: **tracing decorators** (OpenTelemetry) and **metrics decorators** (Prometheus).

## What's different from upstream

In addition to the standard `mock` mode, `mockgen` in this fork can generate:

- **Tracing wrappers** (`-implementation_type=trace`): generates a struct that wraps your interface and automatically starts/ends an OpenTelemetry span per method. Methods whose first argument is `context.Context` get full span lifecycle management and error recording.
- **Metrics wrappers** (`-implementation_type=metrics`): generates a struct that wraps your interface and records a Prometheus histogram of call duration per method, labeled by whether the call returned an error.
- **Logging wrappers** (`-implementation_type=log`): generates a struct that wraps your interface and logs every call and its outcome using `*slog.Logger`. Context methods use context-aware log variants.

## Installation

```bash
go install github.com/pableeee/implgen/mockgen@latest
```

## Running mockgen

`mockgen` has two modes of operation: source and package.

### Source mode

```bash
mockgen -source=foo.go [other options]
```

### Package mode

```bash
mockgen database/sql/driver Conn,Driver

# Convenient for go:generate
mockgen . Conn,Driver
```

## Implementation types

Select the generated output type with `-implementation_type`:

### Mock (default)

Generates a [gomock](https://github.com/uber-go/mock)-based mock with an `EXPECT()` recorder for use in tests.

```bash
mockgen -source=foo.go -destination=mock_foo.go
# or explicitly:
mockgen -source=foo.go -destination=mock_foo.go -implementation_type=mock
```

### Tracing decorator

Generates a struct that wraps the interface and instruments each method with an OpenTelemetry span. Methods taking `context.Context` as their first argument propagate the span through context and record errors automatically.

```bash
mockgen -source=foo.go -destination=trace_foo.go -implementation_type=trace
```

Generated output for an interface `Foo`:

```go
// TracedFooImpl is a tracing decorator of Foo interface.
type TracedFooImpl struct {
    delegate    Foo
    extraAttrs  func(method string, args ...any) []attribute.KeyValue
}

func NewTracedFooImpl(delegate Foo, extraAttrs func(method string, args ...any) []attribute.KeyValue) *TracedFooImpl

func (t *TracedFooImpl) DoSomething(ctx context.Context, arg string) error {
    tracer := otel.Tracer("Foo")
    ctx, span := tracer.Start(ctx, "DoSomething")
    defer span.End()
    if t.extraAttrs != nil {
        span.SetAttributes(t.extraAttrs("DoSomething", ctx, arg)...)
    }
    ret := t.delegate.DoSomething(ctx, arg)
    if ret != nil {
        span.RecordError(ret)
        span.SetStatus(codes.Error, ret.Error())
    }
    return ret
}
```

Pass `nil` as `extraAttrs` to disable custom attributes. When non-nil, use the `method` argument to dispatch per-method logic:

```go
traced := NewTracedFooImpl(realFoo, func(method string, args ...any) []attribute.KeyValue {
    switch method {
    case "DoSomething":
        return []attribute.KeyValue{attribute.String("arg", args[1].(string))}
    }
    return nil
})
```

### Metrics decorator

Generates a struct that wraps the interface and records a Prometheus histogram of call duration for every method, labeled with `error=true/false`.

```bash
mockgen -source=foo.go -destination=metrics_foo.go -implementation_type=metrics
```

Generated output for an interface `Foo`:

```go
// MetricsFooImpl is a metrics decorator of Foo interface.
type MetricsFooImpl struct {
    doSomethingDuration  metrics.Histogram
    delegate             Foo
    extraLabels          func(method string, args ...any) []string
}

func NewMetricsFooImpl(
    delegate Foo,
    extraLabels func(method string, args ...any) []string,
    extraLabelNames ...string,
) *MetricsFooImpl
```

Pass `nil` as `extraLabels` to use only the built-in `error` label. When non-nil, declare the extra label keys up front (Prometheus requires a fixed label set at registration time):

```go
metrics := NewMetricsFooImpl(realFoo, func(method string, args ...any) []string {
    switch method {
    case "DoSomething":
        return []string{"region", args[1].(string)}
    }
    return []string{"region", "unknown"}
}, "region")
```

### Logging decorator

Generates a struct that wraps the interface and logs every method call and its outcome using `*slog.Logger`. Context-aware methods use `InfoContext`/`ErrorContext`; error returns are logged at `Error` level with the error value as a structured attribute.

```bash
mockgen -source=foo.go -destination=log_foo.go -implementation_type=log
```

```go
// Pass nil to fall back to slog.Default().
logged := NewLoggedFooImpl(realFoo, slog.New(slog.NewJSONHandler(os.Stdout, nil)))
```

## Flags

All upstream flags are supported. Fork-specific additions:

- `-implementation_type`: Type of code to generate: `mock` (default), `trace`, `metrics`, or `log`.

Standard flags:

- `-source`: Input Go source file (source mode).
- `-destination`: Output file; defaults to stdout.
- `-package`: Package name for generated code; defaults to `<type>_` + input package name.
- `-interfaces`: (source mode) Comma-separated list of interfaces to generate; defaults to all.
- `-imports`: (source mode) Comma-separated `name=path` pairs of explicit imports.
- `-aux_files`: (source mode) Comma-separated `pkg=file.go` pairs of auxiliary source files.
- `-build_flags`: (package mode) Flags passed verbatim to `go list`.
- `-mock_names`: Comma-separated `Interface=MockName` pairs to override generated mock names.
- `-self_package`: Full import path for generated code, used to prevent import cycles.
- `-copyright_file`: File whose contents are prepended as a copyright header.
- `-build_constraint`: If non-empty, added as `//go:build <constraint>` to generated files.
- `-typed`: Generate type-safe `Return`, `Do`, `DoAndReturn` functions (default false).
- `-exclude_interfaces`: (source mode) Comma-separated interface names to skip.
- `-write_package_comment`: Write package godoc comment (default true).
- `-write_generate_directive`: Add `//go:generate` directive to regenerate (default false).
- `-write_source_comment`: Write source file or interface names as comment (default true).

## Usage with gomock (mock mode)

```go
type Foo interface {
    Bar(x int) int
}

func TestFoo(t *testing.T) {
    ctrl := gomock.NewController(t)

    m := NewMockFoo(ctrl)

    m.EXPECT().Bar(gomock.Eq(99)).Return(101)

    SUT(m)
}
```

For full gomock documentation see the [upstream README](https://github.com/uber-go/mock).

## Origin

This project is a fork of [uber-go/mock](https://github.com/uber-go/mock), which itself originated from Google's [golang/mock](https://github.com/golang/mock).
