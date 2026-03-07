// Copyright 2010 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"sort"
	"strings"
	"unicode"

	"github.com/pableeee/implgen/mockgen/model"
)

func (g *generator) metricsName(typeName string) string {
	if mockName, ok := g.mockNames[typeName]; ok {
		return mockName
	}
	return "Metrics" + typeName + "Impl"
}

// methodFieldName converts a method name to the struct field name for its histogram.
// e.g. "Save" -> "saveDuration", "FetchAll" -> "fetchAllDuration"
func methodFieldName(methodName string) string {
	if methodName == "" {
		return "duration"
	}
	return strings.ToLower(methodName[:1]) + methodName[1:] + "Duration"
}

// otelMetricName returns the OTel instrument name for a method.
// e.g. ("UserStore", "FetchAll") -> "user_store.fetch_all.duration"
func otelMetricName(intfName, methodName string) string {
	return camelToSnake(intfName) + "." + camelToSnake(methodName) + ".duration"
}

func camelToSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteRune('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// extractorCallArgs builds the argument list for passing to the extraAttrs func.
// For non-variadic methods all arg names are passed directly.
// For variadic methods the variadic slice is passed as a single element (not expanded).
func extractorCallArgs(argNames []string, m *model.Method) string {
	if len(argNames) == 0 {
		return ""
	}
	if m.Variadic == nil {
		return strings.Join(argNames, ", ")
	}
	if len(argNames) == 1 {
		return argNames[0]
	}
	return strings.Join(argNames[:len(argNames)-1], ", ") + ", " + argNames[len(argNames)-1]
}

func generateMetricsInterface(g *generator, intf *model.Interface, outputPackagePath string) error {
	mockType := g.metricsName(intf.Name)
	longTp, shortTp := g.formattedTypeParams(intf, outputPackagePath)

	sort.Sort(byMethodName(intf.Methods))

	// Struct: one histogram field per method, plus delegate and extraAttrs func.
	g.p("")
	g.p("// %v is a metrics decorator of %v interface.", mockType, intf.Name)
	g.p("type %v%v struct {", mockType, longTp)
	g.in()
	for _, m := range intf.Methods {
		g.p("%s metric.Float64Histogram", methodFieldName(m.Name))
	}
	g.p("delegate    %v", intf.Name)
	g.p("extraAttrs  func(method string, args ...any) []attribute.KeyValue")
	g.out()
	g.p("}")
	g.p("")

	// Constructor.
	g.p("// New%v creates a new metrics decorator.", mockType)
	g.p("// extraAttrs may be nil to disable custom attributes. When non-nil it is called on every")
	g.p("// method invocation with the method name and arguments; the returned attributes are")
	g.p("// attached to the histogram observation alongside the built-in error attribute.")
	g.p("func New%v%v(delegate %v, extraAttrs func(method string, args ...any) []attribute.KeyValue) (*%v%v, error) {",
		mockType, longTp, intf.Name, mockType, shortTp)
	g.in()
	g.p(`meter := otel.Meter("%v")`, intf.Name)
	for _, m := range intf.Methods {
		fn := methodFieldName(m.Name)
		g.p(`%s, err := meter.Float64Histogram(`, fn)
		g.in()
		g.p(`"%v",`, otelMetricName(intf.Name, m.Name))
		g.p(`metric.WithDescription("Duration of %v calls"),`, m.Name)
		g.p(`metric.WithUnit("s"),`)
		g.out()
		g.p(`)`)
		g.p(`if err != nil {`)
		g.in()
		g.p(`return nil, err`)
		g.out()
		g.p(`}`)
	}
	g.p(`return &%v%v{`, mockType, shortTp)
	g.in()
	for _, m := range intf.Methods {
		fn := methodFieldName(m.Name)
		g.p(`%s: %s,`, fn, fn)
	}
	g.p(`delegate:   delegate,`)
	g.p(`extraAttrs: extraAttrs,`)
	g.out()
	g.p(`}, nil`)
	g.out()
	g.p("}")
	g.p("")

	for _, m := range intf.Methods {
		g.p("")
		_ = generateMetricsMethod(g, mockType, intf.Name, m, outputPackagePath, shortTp)
	}

	return nil
}

func generateMetricsMethod(g *generator, mockType, intfName string, m *model.Method, pkgOverride, shortTp string) error {
	argNames := g.getArgNames(m, true)
	argTypes := g.getArgTypes(m, pkgOverride, true)
	argString := makeArgString(argNames, argTypes)

	isContextMethod := len(argTypes) > 0 && argTypes[0] == "context.Context"

	rets := make([]string, len(m.Out))
	for i, p := range m.Out {
		rets[i] = p.Type.String(g.packageMap, pkgOverride)
	}
	retString := strings.Join(rets, ", ")
	if len(rets) > 1 {
		retString = "(" + retString + ")"
	}
	if retString != "" {
		retString = " " + retString
	}

	ia := newIdentifierAllocator(argNames)
	idRecv := ia.allocateIdentifier("t")
	idBegin := ia.allocateIdentifier("begin")
	idTook := ia.allocateIdentifier("took")
	idAttrs := ia.allocateIdentifier("attrs")

	fieldName := methodFieldName(m.Name)

	var callArgs string
	if m.Variadic == nil {
		if len(argNames) > 0 {
			callArgs = strings.Join(argNames, ", ")
		}
	} else {
		switch {
		case len(argNames) > 1:
			callArgs = strings.Join(argNames[0:len(argNames)-1], ", ") + ", " + argNames[len(argNames)-1] + "..."
		case len(argNames) == 1:
			callArgs = argNames[0] + "..."
		}
	}

	extArgs := extractorCallArgs(argNames, m)
	if extArgs != "" {
		extArgs = ", " + extArgs
	}

	// Context argument to pass to Record; fall back to context.Background() for non-context methods.
	recordCtx := "context.Background()"
	if isContextMethod {
		recordCtx = argNames[0]
	}

	g.p("// %v Metrics base method.", m.Name)
	g.p("func (%v *%v%v) %v(%v)%v {", idRecv, mockType, shortTp, m.Name, argString, retString)
	g.in()
	g.p("%v := time.Now()", idBegin)

	returnsError := false
	errorIndex := -1
	returns := make([]string, len(rets))
	for i, r := range rets {
		returns[i] = ia.allocateIdentifier("ret")
		if r == "error" {
			returnsError = true
			errorIndex = i
		}
	}

	if len(m.Out) == 0 {
		g.p(`%s.delegate.%s(%s)`, idRecv, m.Name, callArgs)
		g.p("%v := time.Since(%v)", idTook, idBegin)
		g.p(`%v := []attribute.KeyValue{}`, idAttrs)
	} else {
		returnArgsString := strings.Join(returns, ", ")
		g.p(`%s := %s.delegate.%s(%s)`, returnArgsString, idRecv, m.Name, callArgs)
		g.p("%v := time.Since(%v)", idTook, idBegin)
		if returnsError {
			g.p(`%v := []attribute.KeyValue{attribute.Bool("error", %v != nil)}`, idAttrs, returns[errorIndex])
		} else {
			g.p(`%v := []attribute.KeyValue{}`, idAttrs)
		}
	}

	g.p(`if %v.extraAttrs != nil {`, idRecv)
	g.in()
	g.p(`%v = append(%v, %v.extraAttrs("%v"%v)...)`, idAttrs, idAttrs, idRecv, m.Name, extArgs)
	g.out()
	g.p("}")
	g.p(`%v.%v.Record(%v, %v.Seconds(), metric.WithAttributes(%v...))`, idRecv, fieldName, recordCtx, idTook, idAttrs)

	if len(m.Out) > 0 {
		g.p(`return %s`, strings.Join(returns, ", "))
	}

	g.out()
	g.p("}")
	return nil
}
