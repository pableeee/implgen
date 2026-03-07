package main

import (
	"strings"
	"testing"

	"github.com/pableeee/implgen/mockgen/model"
)

func metricsGenerator() generator {
	return generator{}
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
	if !strings.Contains(out, "pingDuration metrics.Histogram") {
		t.Error("expected per-method pingDuration histogram field")
	}
	if !strings.Contains(out, "delegate     Store") {
		t.Error("expected delegate field of original type Store")
	}
	if !strings.Contains(out, "extraLabels  func(method string, args ...any) []string") {
		t.Error("expected extraLabels func field")
	}
	if !strings.Contains(out, "func NewMetricsStoreImpl(delegate Store, extraLabels func(method string, args ...any) []string, extraLabelNames ...string) *MetricsStoreImpl {") {
		t.Error("expected NewMetricsStoreImpl constructor signature")
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

	if !strings.Contains(out, `Subsystem: "Svc"`) {
		t.Error("expected histogram subsystem set to interface name")
	}
	if !strings.Contains(out, `Name:      "call_duration_milliseconds"`) {
		t.Error("expected histogram metric name")
	}
	if !strings.Contains(out, `allLabels := append([]string{"error"}, extraLabelNames...)`) {
		t.Error("expected allLabels combining error with extra label names")
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
	if !strings.Contains(out, `.Observe(`) {
		t.Error("expected histogram.Observe call")
	}
}

func TestGenerateMetricsInterface_ErrorReturn_LabelsTrueOrFalse(t *testing.T) {
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

	if !strings.Contains(out, `"false"`) {
		t.Error(`expected failed = "false" for non-error path`)
	}
	if !strings.Contains(out, `"true"`) {
		t.Error(`expected failed = "true" inside error check`)
	}
	if !strings.Contains(out, "!= nil") {
		t.Error("expected nil error check to gate the label")
	}
}

func TestGenerateMetricsInterface_NoErrorReturn_LabelsNA(t *testing.T) {
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

	if !strings.Contains(out, `"N/A"`) {
		t.Error(`expected failed = "N/A" when method does not return error`)
	}
	if strings.Contains(out, `"true"`) || strings.Contains(out, `"false"`) {
		t.Error("should not emit error label values when there is no error return")
	}
}

func TestGenerateMetricsInterface_NoReturnMethod_ObservesAndReturnsNothing(t *testing.T) {
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

	if !strings.Contains(out, ".Observe(") {
		t.Error("expected Observe even when method returns nothing")
	}
	if strings.Contains(out, "return t.delegate") || strings.Contains(out, "return ret") {
		t.Error("should not emit return statement in method body when method has no return values")
	}
}

func TestGenerateMetricsInterface_ExtraLabelsCall(t *testing.T) {
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

	if !strings.Contains(out, "if t.extraLabels != nil {") {
		t.Error("expected nil extraLabels guard")
	}
	if !strings.Contains(out, `t.extraLabels("Call", id)`) {
		t.Error(`expected extraLabels("Call", id) call`)
	}
	if !strings.Contains(out, `[]string{"error",`) {
		t.Error("expected base label slice starting with error key")
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
	// extraLabels receives the slice, not expanded
	if !strings.Contains(out, `extraLabels("Printf", format, args)`) {
		t.Error("expected extraLabels to receive variadic slice, not expanded args")
	}
}
