package main

import (
	"strings"
	"testing"

	"github.com/pableeee/implgen/mockgen/model"
)

func metricsGenerator() generator {
	return generator{
		packageMap: map[string]string{"context": "context"},
	}
}

func TestGenerateMetricsInterface_StructAndConstructor(t *testing.T) {
	g := metricsGenerator()
	intf := &model.Interface{Name: "Store"}
	intf.AddMethod(&model.Method{Name: "Ping"})

	if err := generateMetricsInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, "type MetricsStoreImpl struct {") {
		t.Error("expected MetricsStoreImpl struct declaration")
	}
	if !strings.Contains(out, "pingDuration metric.Float64Histogram") {
		t.Error("expected per-method pingDuration OTel histogram field")
	}
	if !strings.Contains(out, "delegate    Store") {
		t.Error("expected delegate field of original type Store")
	}
	if !strings.Contains(out, "extraAttrs  func(method string, args ...any) []attribute.KeyValue") {
		t.Error("expected extraAttrs func field returning []attribute.KeyValue")
	}
	if !strings.Contains(out, "func NewMetricsStoreImpl(delegate Store, extraAttrs func(method string, args ...any) []attribute.KeyValue) (*MetricsStoreImpl, error) {") {
		t.Error("expected NewMetricsStoreImpl constructor returning (*Type, error)")
	}
}

func TestGenerateMetricsInterface_Constructor_InitializesHistogram(t *testing.T) {
	g := metricsGenerator()
	intf := &model.Interface{Name: "Svc"}
	intf.AddMethod(&model.Method{Name: "Call"})

	if err := generateMetricsInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, `otel.Meter("Svc")`) {
		t.Error("expected otel.Meter scoped to interface name")
	}
	if !strings.Contains(out, `"svc.call.duration"`) {
		t.Error("expected OTel instrument name svc.call.duration")
	}
	if !strings.Contains(out, `metric.WithUnit("s")`) {
		t.Error("expected unit seconds")
	}
	if !strings.Contains(out, "return nil, err") {
		t.Error("expected error propagation from meter.Float64Histogram")
	}
}

func TestGenerateMetricsInterface_AllMethods_GetTiming(t *testing.T) {
	g := metricsGenerator()
	intf := &model.Interface{Name: "DB"}
	intf.AddMethod(&model.Method{
		Name: "Query",
		In:   []*model.Parameter{{Name: "sql", Type: model.PredeclaredType("string")}},
		Out:  []*model.Parameter{{Type: model.PredeclaredType("int")}},
	})

	if err := generateMetricsInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, "time.Now()") {
		t.Error("expected time.Now() to record start time")
	}
	if !strings.Contains(out, "time.Since(") {
		t.Error("expected time.Since() to compute duration")
	}
	if !strings.Contains(out, ".Record(") {
		t.Error("expected histogram.Record call")
	}
}

func TestGenerateMetricsInterface_ErrorReturn_AttributeBool(t *testing.T) {
	g := metricsGenerator()
	intf := &model.Interface{Name: "Writer"}
	intf.AddMethod(&model.Method{
		Name: "Write",
		In:   []*model.Parameter{{Name: "data", Type: model.PredeclaredType("string")}},
		Out: []*model.Parameter{
			{Type: model.PredeclaredType("int")},
			{Type: &model.NamedType{Type: "error"}},
		},
	})

	if err := generateMetricsInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, `attribute.Bool("error",`) {
		t.Error(`expected attribute.Bool("error", ...) for error label`)
	}
	if !strings.Contains(out, "!= nil") {
		t.Error("expected nil check to determine error attribute value")
	}
}

func TestGenerateMetricsInterface_NoErrorReturn_EmptyAttrs(t *testing.T) {
	g := metricsGenerator()
	intf := &model.Interface{Name: "Reader"}
	intf.AddMethod(&model.Method{
		Name: "Read",
		Out:  []*model.Parameter{{Type: model.PredeclaredType("string")}},
	})

	if err := generateMetricsInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if strings.Contains(out, `attribute.Bool("error"`) {
		t.Error("should not emit error attribute when method does not return error")
	}
}

func TestGenerateMetricsInterface_ContextMethod_UsesCtxForRecord(t *testing.T) {
	g := metricsGenerator()
	intf := &model.Interface{Name: "Svc"}
	intf.AddMethod(&model.Method{
		Name: "Call",
		In:   []*model.Parameter{contextParam("ctx")},
	})

	if err := generateMetricsInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, ".Record(ctx,") {
		t.Error("expected Record to use ctx for context methods")
	}
}

func TestGenerateMetricsInterface_NonContextMethod_UsesBackground(t *testing.T) {
	g := metricsGenerator()
	intf := &model.Interface{Name: "Counter"}
	intf.AddMethod(&model.Method{
		Name: "Increment",
		In:   []*model.Parameter{{Name: "n", Type: model.PredeclaredType("int")}},
	})

	if err := generateMetricsInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, ".Record(context.Background(),") {
		t.Error("expected Record to use context.Background() for non-context methods")
	}
}

func TestGenerateMetricsInterface_NoReturnMethod_RecordsAndReturnsNothing(t *testing.T) {
	g := metricsGenerator()
	intf := &model.Interface{Name: "Notifier"}
	intf.AddMethod(&model.Method{
		Name: "Notify",
		In:   []*model.Parameter{{Name: "msg", Type: model.PredeclaredType("string")}},
	})

	if err := generateMetricsInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, ".Record(") {
		t.Error("expected Record even when method returns nothing")
	}
	if strings.Contains(out, "return ret") {
		t.Error("should not emit return statement for void method")
	}
}

func TestGenerateMetricsInterface_ExtraAttrsCall(t *testing.T) {
	g := metricsGenerator()
	intf := &model.Interface{Name: "Svc"}
	intf.AddMethod(&model.Method{
		Name: "Call",
		In:   []*model.Parameter{{Name: "id", Type: model.PredeclaredType("string")}},
	})

	if err := generateMetricsInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, "if t.extraAttrs != nil {") {
		t.Error("expected nil extraAttrs guard")
	}
	if !strings.Contains(out, `t.extraAttrs("Call", id)`) {
		t.Error(`expected extraAttrs("Call", id) call`)
	}
}

func TestGenerateMetricsInterface_VariadicMethod(t *testing.T) {
	g := metricsGenerator()
	intf := &model.Interface{Name: "Printer"}
	intf.AddMethod(&model.Method{
		Name: "Printf",
		In:   []*model.Parameter{{Name: "format", Type: model.PredeclaredType("string")}},
		Variadic: &model.Parameter{
			Name: "args",
			Type: model.PredeclaredType("any"),
		},
	})

	if err := generateMetricsInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, "args...") {
		t.Error("expected variadic args forwarded with ...")
	}
	if !strings.Contains(out, `extraAttrs("Printf", format, args)`) {
		t.Error("expected extraAttrs to receive variadic slice, not expanded args")
	}
}
