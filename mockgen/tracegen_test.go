package main

import (
	"strings"
	"testing"

	"github.com/pableeee/implgen/mockgen/model"
)

// contextParam returns a context.Context parameter with the given name.
func contextParam(name string) *model.Parameter {
	return &model.Parameter{
		Name: name,
		Type: &model.NamedType{Package: "context", Type: "Context"},
	}
}

// tracedGenerator returns a generator whose packageMap maps "context" so that
// context.Context types resolve correctly for the isContextMethod check.
func tracedGenerator() generator {
	return generator{
		packageMap: map[string]string{"context": "context"},
	}
}

func TestGenerateTracedInterface_StructAndConstructor(t *testing.T) {
	g := tracedGenerator()
	intf := &model.Interface{Name: "Store"}
	intf.AddMethod(&model.Method{Name: "Ping"})

	if err := generateTracedInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, "type StoreAttributeExtractor interface {") {
		t.Error("expected StoreAttributeExtractor interface declaration")
	}
	if !strings.Contains(out, "type TracedStoreImpl struct {") {
		t.Error("expected TracedStoreImpl struct declaration")
	}
	if !strings.Contains(out, "delegate  Store") {
		t.Error("expected delegate field of original type Store")
	}
	if !strings.Contains(out, "extractor StoreAttributeExtractor") {
		t.Error("expected extractor field of type StoreAttributeExtractor")
	}
	if !strings.Contains(out, "func NewTracedStoreImpl(delegate Store, extractor StoreAttributeExtractor) *TracedStoreImpl {") {
		t.Error("expected NewTracedStoreImpl constructor")
	}
}

func TestGenerateTracedInterface_ContextMethod_SpanLifecycle(t *testing.T) {
	g := tracedGenerator()
	intf := &model.Interface{Name: "Repo"}
	intf.AddMethod(&model.Method{
		Name: "Save",
		In: []*model.Parameter{
			contextParam("ctx"),
			{Name: "val", Type: model.PredeclaredType("string")},
		},
	})

	if err := generateTracedInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, `otel.Tracer("Repo")`) {
		t.Error("expected otel.Tracer named after the original interface, not the impl type")
	}
	if !strings.Contains(out, `.Start(ctx, "Save")`) {
		t.Error("expected tracer.Start with context and method name")
	}
	if !strings.Contains(out, "defer span.End()") {
		t.Error("expected defer span.End()")
	}
}

func TestGenerateTracedInterface_ContextMethod_ExtractorCall(t *testing.T) {
	g := tracedGenerator()
	intf := &model.Interface{Name: "Repo"}
	intf.AddMethod(&model.Method{
		Name: "Save",
		In: []*model.Parameter{
			contextParam("ctx"),
			{Name: "val", Type: model.PredeclaredType("string")},
		},
	})

	if err := generateTracedInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, "if t.extractor != nil {") {
		t.Error("expected nil extractor guard")
	}
	if !strings.Contains(out, "span.SetAttributes(") {
		t.Error("expected span.SetAttributes call with extractor output")
	}
	if !strings.Contains(out, ".extractor.Save(") {
		t.Error("expected extractor.Save call forwarding method args")
	}
}

func TestGenerateTracedInterface_ContextMethodWithError_RecordsError(t *testing.T) {
	g := tracedGenerator()
	intf := &model.Interface{Name: "Client"}
	intf.AddMethod(&model.Method{
		Name: "Fetch",
		In:   []*model.Parameter{contextParam("ctx")},
		Out: []*model.Parameter{
			{Name: "result", Type: model.PredeclaredType("string")},
			{Name: "err", Type: &model.NamedType{Type: "error"}},
		},
	})

	if err := generateTracedInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, "RecordError(") {
		t.Error("expected span.RecordError for method returning error")
	}
	if !strings.Contains(out, "SetStatus(codes.Error,") {
		t.Error("expected span.SetStatus(codes.Error, ...) for method returning error")
	}
}

func TestGenerateTracedInterface_ContextMethodNoError_NoRecordError(t *testing.T) {
	g := tracedGenerator()
	intf := &model.Interface{Name: "Publisher"}
	intf.AddMethod(&model.Method{
		Name: "Publish",
		In: []*model.Parameter{
			contextParam("ctx"),
			{Name: "msg", Type: model.PredeclaredType("string")},
		},
		// returns int, not error
		Out: []*model.Parameter{
			{Type: model.PredeclaredType("int")},
		},
	})

	if err := generateTracedInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if strings.Contains(out, "RecordError") {
		t.Error("should not emit RecordError when method does not return error")
	}
}

func TestGenerateTracedInterface_NonContextMethod_NoSpan(t *testing.T) {
	g := tracedGenerator()
	intf := &model.Interface{Name: "Counter"}
	intf.AddMethod(&model.Method{
		Name: "Increment",
		In:   []*model.Parameter{{Name: "n", Type: model.PredeclaredType("int")}},
		Out:  []*model.Parameter{{Type: model.PredeclaredType("int")}},
	})

	if err := generateTracedInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if strings.Contains(out, "otel.Tracer") {
		t.Error("should not emit otel.Tracer for non-context method")
	}
	if strings.Contains(out, "span.End") {
		t.Error("should not emit span.End for non-context method")
	}
	if !strings.Contains(out, "t.delegate.Increment(n)") {
		t.Error("expected direct delegate call without tracing")
	}
}

func TestGenerateTracedInterface_VariadicMethod(t *testing.T) {
	g := tracedGenerator()
	intf := &model.Interface{Name: "Logger"}
	intf.AddMethod(&model.Method{
		Name: "Log",
		In:   []*model.Parameter{contextParam("ctx")},
		Variadic: &model.Parameter{
			Name: "args",
			Type: model.PredeclaredType("any"),
		},
	})

	if err := generateTracedInterface(&g, intf, ""); err != nil {
		t.Fatal(err)
	}

	out := g.buf.String()

	if !strings.Contains(out, "args...") {
		t.Error("expected variadic args forwarded with ...")
	}
}
