package main

import (
	"strings"
	"testing"

	"github.com/pableeee/implgen/mockgen/model"
)

func loggedGenerator() generator {
	return generator{
		packageMap: map[string]string{"context": "context"},
	}
}

func TestGenerateLoggedInterface_StructAndConstructor(t *testing.T) {
	g := loggedGenerator()
	intf := &model.Interface{Name: "Store"}
	intf.AddMethod(&model.Method{Name: "Ping"})

	if err := generateLoggedInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, "type LoggedStoreImpl struct {") {
		t.Error("expected LoggedStoreImpl struct declaration")
	}
	if !strings.Contains(out, "delegate Store") {
		t.Error("expected delegate field of original type Store")
	}
	if !strings.Contains(out, "logger   *slog.Logger") {
		t.Error("expected logger field of type *slog.Logger")
	}
	if !strings.Contains(out, "func NewLoggedStoreImpl(delegate Store, logger *slog.Logger) *LoggedStoreImpl {") {
		t.Error("expected NewLoggedStoreImpl constructor")
	}
}

func TestGenerateLoggedInterface_Constructor_NilLoggerFallback(t *testing.T) {
	g := loggedGenerator()
	intf := &model.Interface{Name: "Svc"}
	intf.AddMethod(&model.Method{Name: "Call"})

	if err := generateLoggedInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, "if logger == nil {") {
		t.Error("expected nil logger guard in constructor")
	}
	if !strings.Contains(out, "logger = slog.Default()") {
		t.Error("expected slog.Default() fallback")
	}
}

func TestGenerateLoggedInterface_ContextMethod_UsesInfoContext(t *testing.T) {
	g := loggedGenerator()
	intf := &model.Interface{Name: "Repo"}
	intf.AddMethod(&model.Method{
		Name: "Save",
		In: []*model.Parameter{
			contextParam("ctx"),
			{Name: "val", Type: model.PredeclaredType("string")},
		},
	})

	if err := generateLoggedInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, "InfoContext(ctx,") {
		t.Error("expected InfoContext for context method")
	}
	if strings.Contains(out, `t.logger.Info("calling`) {
		t.Error("should use InfoContext not Info for context method entry log")
	}
}

func TestGenerateLoggedInterface_NonContextMethod_UsesInfo(t *testing.T) {
	g := loggedGenerator()
	intf := &model.Interface{Name: "Counter"}
	intf.AddMethod(&model.Method{
		Name: "Increment",
		In:   []*model.Parameter{{Name: "n", Type: model.PredeclaredType("int")}},
		Out:  []*model.Parameter{{Type: model.PredeclaredType("int")}},
	})

	if err := generateLoggedInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, `t.logger.Info(`) {
		t.Error("expected Info for non-context method")
	}
	if strings.Contains(out, "InfoContext") {
		t.Error("should not use InfoContext for non-context method")
	}
}

func TestGenerateLoggedInterface_ErrorReturn_UsesErrorContext(t *testing.T) {
	g := loggedGenerator()
	intf := &model.Interface{Name: "Writer"}
	intf.AddMethod(&model.Method{
		Name: "Write",
		In:   []*model.Parameter{contextParam("ctx"), {Name: "data", Type: model.PredeclaredType("string")}},
		Out: []*model.Parameter{
			{Type: model.PredeclaredType("int")},
			{Type: &model.NamedType{Type: "error"}},
		},
	})

	if err := generateLoggedInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, "ErrorContext(ctx,") {
		t.Error("expected ErrorContext on error path for context method")
	}
	if !strings.Contains(out, `"error", ret`) {
		t.Error("expected error value logged as slog attribute")
	}
}

func TestGenerateLoggedInterface_NoReturn_DelegatesAndLogs(t *testing.T) {
	g := loggedGenerator()
	intf := &model.Interface{Name: "Notifier"}
	intf.AddMethod(&model.Method{
		Name: "Notify",
		In:   []*model.Parameter{{Name: "msg", Type: model.PredeclaredType("string")}},
	})

	if err := generateLoggedInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, "t.delegate.Notify(msg)") {
		t.Error("expected direct delegate call")
	}
	if strings.Contains(out, "return ret") {
		t.Error("should not emit return statement for void method")
	}
}

func TestGenerateLoggedInterface_VariadicMethod(t *testing.T) {
	g := loggedGenerator()
	intf := &model.Interface{Name: "Logger"}
	intf.AddMethod(&model.Method{
		Name: "Log",
		In:   []*model.Parameter{contextParam("ctx")},
		Variadic: &model.Parameter{
			Name: "args",
			Type: model.PredeclaredType("any"),
		},
	})

	if err := generateLoggedInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, "args...") {
		t.Error("expected variadic args forwarded with ...")
	}
}
