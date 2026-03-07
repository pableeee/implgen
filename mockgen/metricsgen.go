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

// methodMetricName converts a method name to a Prometheus metric name.
// e.g. "Save" -> "save_duration_milliseconds", "FetchAll" -> "fetch_all_duration_milliseconds"
func methodMetricName(methodName string) string {
	return camelToSnake(methodName) + "_duration_milliseconds"
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

// extractorCallArgs builds the argument list for passing to the extraLabels func.
// For non-variadic methods all arg names are passed directly.
// For variadic methods the variadic slice is passed as a single element (not expanded).
func extractorCallArgs(argNames []string, m *model.Method) string {
	if len(argNames) == 0 {
		return ""
	}
	if m.Variadic == nil {
		return strings.Join(argNames, ", ")
	}
	// variadic: pass non-variadic args + the slice itself (not expanded)
	if len(argNames) == 1 {
		return argNames[0]
	}
	return strings.Join(argNames[:len(argNames)-1], ", ") + ", " + argNames[len(argNames)-1]
}

func generateMetricsInterface(g *generator, intf *model.Interface, outputPackagePath string) error {
	mockType := g.metricsName(intf.Name)
	longTp, shortTp := g.formattedTypeParams(intf, outputPackagePath)

	// Sort methods once; used for struct fields, constructor, and method bodies.
	sort.Sort(byMethodName(intf.Methods))

	// Struct: one histogram field per method, plus delegate and extraLabels func.
	g.p("")
	g.p("// %v is a metrics decorator of %v interface.", mockType, intf.Name)
	g.p("type %v%v struct {", mockType, longTp)
	g.in()
	for _, m := range intf.Methods {
		g.p("%s metrics.Histogram", methodFieldName(m.Name))
	}
	g.p("delegate     %v", intf.Name)
	g.p("extraLabels  func(method string, args ...any) []string")
	g.out()
	g.p("}")
	g.p("")

	// Constructor.
	g.p("// New%v creates a new metrics decorator.", mockType)
	g.p("// extraLabels may be nil to disable custom labeling. When non-nil it is called on every")
	g.p("// method invocation with the method name and arguments; the returned flat key-value pairs")
	g.p("// are appended to the histogram observation. extraLabelNames must list the label keys")
	g.p("// that extraLabels will return (Prometheus requires a fixed label set at registration time).")
	g.p("func New%v%v(delegate %v, extraLabels func(method string, args ...any) []string, extraLabelNames ...string) *%v%v {",
		mockType, longTp, intf.Name, mockType, shortTp)
	g.in()
	g.p(`allLabels := append([]string{"error"}, extraLabelNames...)`)
	for _, m := range intf.Methods {
		g.p(`%s := prometheus.NewHistogramFrom(stdprometheus.HistogramOpts{`, methodFieldName(m.Name))
		g.in()
		g.p(`Subsystem: "%v",`, intf.Name)
		g.p(`Name:      "%v",`, methodMetricName(m.Name))
		g.p(`Help:      "Duration of %v calls in milliseconds",`, m.Name)
		g.p(`Buckets:   stdprometheus.DefBuckets,`)
		g.out()
		g.p(`}, allLabels)`)
	}
	g.p(`return &%v%v{`, mockType, shortTp)
	g.in()
	for _, m := range intf.Methods {
		fn := methodFieldName(m.Name)
		g.p(`%s: %s,`, fn, fn)
	}
	g.p(`delegate:    delegate,`)
	g.p(`extraLabels: extraLabels,`)
	g.out()
	g.p(`}`)
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
	idFailed := ia.allocateIdentifier("failed")
	idLabels := ia.allocateIdentifier("labelVals")

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

	// Args passed to extraLabels — variadic slice is not expanded.
	extArgs := extractorCallArgs(argNames, m)
	if extArgs != "" {
		extArgs = ", " + extArgs
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

	g.p(`%v := "N/A"`, idFailed)

	if len(m.Out) == 0 {
		g.p(`%s.delegate.%s(%s)`, idRecv, m.Name, callArgs)
		g.p("%v := time.Since(%v)", idTook, idBegin)
	} else {
		returnArgsString := strings.Join(returns, ", ")
		g.p(`%s := %s.delegate.%s(%s)`, returnArgsString, idRecv, m.Name, callArgs)
		g.p("%v := time.Since(%v)", idTook, idBegin)
		if returnsError {
			g.p(`%v = "false"`, idFailed)
			g.p(`if %v != nil {`, returns[errorIndex])
			g.in()
			g.p(`%v = "true"`, idFailed)
			g.out()
			g.p("}")
		}
	}

	// Build label slice; append extraLabels output if provided.
	g.p(`%v := []string{"error", %v}`, idLabels, idFailed)
	g.p(`if %v.extraLabels != nil {`, idRecv)
	g.in()
	g.p(`%v = append(%v, %v.extraLabels("%v"%v)...)`, idLabels, idLabels, idRecv, m.Name, extArgs)
	g.out()
	g.p("}")
	g.p(`%v.%v.With(%v...).Observe(%v.Seconds())`, idRecv, fieldName, idLabels, idTook)

	if len(m.Out) > 0 {
		g.p(`return %s`, strings.Join(returns, ", "))
	}

	g.out()
	g.p("}")
	return nil
}
