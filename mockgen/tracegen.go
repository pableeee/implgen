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

	"github.com/pableeee/implgen/mockgen/model"
)

// The name of the traced type to use for the given interface identifier.
func (g *generator) tracedName(typeName string) string {
	if mockName, ok := g.mockNames[typeName]; ok {
		return mockName
	}

	return "Traced" + typeName + "Impl"
}

func generateTracedInterface(g *generator, intf *model.Interface, outputPackagePath string) error {
	mockType := g.tracedName(intf.Name)
	longTp, shortTp := g.formattedTypeParams(intf, outputPackagePath)

	// Sort methods once.
	sort.Sort(byMethodName(intf.Methods))

	// Struct with delegate (original interface) and extraAttrs func.
	g.p("")
	g.p("// %v is a tracing decorator of %v interface.", mockType, intf.Name)
	g.p("type %v%v struct {", mockType, longTp)
	g.in()
	g.p("delegate    %v", intf.Name)
	g.p("extraAttrs  func(method string, args ...any) []attribute.KeyValue")
	g.out()
	g.p("}")
	g.p("")

	// Constructor.
	g.p("// New%v creates a new trace decorator instance.", mockType)
	g.p("// extraAttrs may be nil to disable custom span attributes. When non-nil it is called")
	g.p("// on every context-bearing method with the method name and arguments; the returned")
	g.p("// attributes are set on the span via span.SetAttributes.")
	g.p("func New%v%v(delegate %v, extraAttrs func(method string, args ...any) []attribute.KeyValue) *%v%v {",
		mockType, longTp, intf.Name, mockType, shortTp)
	g.in()
	g.p(`return &%v%v{delegate: delegate, extraAttrs: extraAttrs}`, mockType, shortTp)
	g.out()
	g.p("}")
	g.p("")

	for _, m := range intf.Methods {
		g.p("")
		_ = generateTracedMethod(g, mockType, intf.Name, m, outputPackagePath, shortTp)
	}

	return nil
}

func generateTracedMethod(g *generator, mockType, intfName string, m *model.Method, pkgOverride, shortTp string) error {
	argNames := g.getArgNames(m, true)
	argTypes := g.getArgTypes(m, pkgOverride, true)
	argString := makeArgString(argNames, argTypes)

	// A method is a "context method" if its first argument is context.Context.
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

	// Args passed to extraAttrs — variadic slice is not expanded.
	extArgs := extractorCallArgs(argNames, m)
	if extArgs != "" {
		extArgs = ", " + extArgs
	}

	g.p("// %v traced base method.", m.Name)
	g.p("func (%v *%v%v) %v(%v)%v {", idRecv, mockType, shortTp, m.Name, argString, retString)
	g.in()

	if isContextMethod {
		ctxArg := argNames[0]
		idTracer := ia.allocateIdentifier("tracer")
		idSpan := ia.allocateIdentifier("span")
		g.p(`%v := otel.Tracer("%s")`, idTracer, intfName)
		g.p("%s, %v := %v.Start(%v, %q)", ctxArg, idSpan, idTracer, ctxArg, m.Name)
		g.p("defer %v.End()", idSpan)
		g.p(`if %v.extraAttrs != nil {`, idRecv)
		g.in()
		g.p(`%v.SetAttributes(%v.extraAttrs("%v"%v)...)`, idSpan, idRecv, m.Name, extArgs)
		g.out()
		g.p("}")

		if len(m.Out) == 0 {
			g.p(`%s.delegate.%s(%s)`, idRecv, m.Name, callArgs)
		} else {
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
			returnArgsString := strings.Join(returns, ", ")
			g.p(`%s := %s.delegate.%s(%s)`, returnArgsString, idRecv, m.Name, callArgs)
			if returnsError {
				g.p(`if %v != nil {`, returns[errorIndex])
				g.in()
				g.p("%v.RecordError(%v)", idSpan, returns[errorIndex])
				g.p("%v.SetStatus(codes.Error, %v.Error())", idSpan, returns[errorIndex])
				g.out()
				g.p("}")
			}
			g.p(`return %s`, returnArgsString)
		}
	} else {
		// Non-context method: delegate directly without tracing.
		if len(m.Out) == 0 {
			g.p(`%s.delegate.%s(%s)`, idRecv, m.Name, callArgs)
		} else {
			returns := make([]string, len(rets))
			for i := range rets {
				returns[i] = ia.allocateIdentifier("ret")
			}
			returnArgsString := strings.Join(returns, ", ")
			g.p(`%s := %s.delegate.%s(%s)`, returnArgsString, idRecv, m.Name, callArgs)
			g.p(`return %s`, returnArgsString)
		}
	}

	g.out()
	g.p("}")
	return nil
}
