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

// MockGen generates mock implementations of Go interfaces.
package main

// TODO: This does not support recursive embedded interfaces.
// TODO: This does not support embedding package-local interfaces in a separate file.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pableeee/implgen/mockgen/model"
)

// The name of the mock type to use for the given interface identifier.
func (g *generator) tracedName(typeName string) string {
	if mockName, ok := g.mockNames[typeName]; ok {
		return mockName
	}

	return "Traced" + typeName
}

func generateTracedInterface(g *generator, intf *model.Interface, outputPackagePath string) error {
	mockType := g.mockName(intf.Name)
	longTp, shortTp := g.formattedTypeParams(intf, outputPackagePath)

	g.p("")
	g.p("// %v is a tracing decorator of %v interface.", mockType, intf.Name)
	g.p("type Traced%v interface {", intf.Name)
	g.in()

	for _, m := range intf.Methods {
		argNames := g.getArgNames(m, true)
		argTypes := g.getArgTypes(m, outputPackagePath, true)
		argString := makeArgString(argNames, argTypes)

		rets := make([]string, len(m.Out))
		for i, p := range m.Out {
			rets[i] = p.Type.String(g.packageMap, outputPackagePath)
		}
		retString := strings.Join(rets, ", ")
		if len(rets) > 1 {
			retString = "(" + retString + ")"
		}
		if retString != "" {
			retString = " " + retString
		}

		g.p("%s(%s) %s", m.Name, argString, retString)
	}

	g.out()
	g.p("}")
	g.p("")

	g.p("")
	g.p("// %v is a tracing decorator of %v interface.", mockType, intf.Name)
	g.p("type %v%v struct {", mockType, longTp)
	g.in()
	g.p("delegate Traced%v", intf.Name)
	g.out()
	g.p("}")
	g.p("")

	g.p("// New%v creates a new trace decorator instance.", mockType)
	g.p("func New%v%v(ctrl Traced%v) *%v%v {", mockType, longTp, intf.Name, mockType, shortTp)
	g.in()
	g.p(`deco := &%v%v{delegate: ctrl}`, mockType, shortTp)
	g.p("return deco")
	g.out()
	g.p("}")
	g.p("")

	generateTracedMethods(g, mockType, intf, outputPackagePath, shortTp)

	return nil
}
func generateTracedMethods(g *generator, mockType string, intf *model.Interface, pkgOverride, shortTp string) {
	sort.Sort(byMethodName(intf.Methods))
	for _, m := range intf.Methods {
		g.p("")
		_ = generateTracedMethod(g, mockType, m, pkgOverride, shortTp)
	}
}

// GenerateMockMethod generates a mock method implementation.
// If non-empty, pkgOverride is the package in which unqualified types reside.
func generateTracedMethod(g *generator, mockType string, m *model.Method, pkgOverride, shortTp string) error {
	argNames := g.getArgNames(m, true)
	argTypes := g.getArgTypes(m, pkgOverride, true)
	argString := makeArgString(argNames, argTypes)

	// flag as context method, if the firt argument is a context.
	isContextMethod := len(argNames) > 0 && argTypes[0] == "context.Context"

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
	idSpan := ia.allocateIdentifier("span")

	g.p("// %v traced base method.", m.Name)
	g.p("func (%v *%v%v) %v(%v)%v {", idRecv, mockType, shortTp, m.Name, argString, retString)
	g.in()

	if isContextMethod {
		ctxArg := argNames[0]
		// We'll input the tracing code, if the method bares a context as its firts param.
		idTracer := ia.allocateIdentifier("tracer")
		// TODO: Fix tracer name (without the 'Mock') & constant initialization.
		g.p(`%v := otel.Tracer("%s")`, idTracer, mockType)
		g.p("%s, %v := %v.Start(%v, %q)", ctxArg, idSpan, idTracer, ctxArg, m.Name)
		g.p("defer span.End()")
	}

	var callArgs string
	if m.Variadic == nil {
		if len(argNames) > 0 {
			callArgs = strings.Join(argNames, ", ")
		}
	} else {
		switch {
		case len(argNames) > 1:
			callArgs = strings.Join(argNames[0:len(argNames)-1], ", ") + ", " + argNames[len(argNames)-1] + "..."
		case len(argNames) == 0:
			callArgs = ""
		case len(argNames) == 1:
			callArgs = argNames[len(argNames)-1] + "..."
		}

	}
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

		if returnsError && isContextMethod {
			g.p(`if %v != nil {`, returns[errorIndex])
			g.in()
			g.p("%v.RecordError(%v)", idSpan, returns[errorIndex])
			g.p("%v.SetStatus(codes.Error, %v.Error())", idSpan, returns[errorIndex])
			g.out()
			g.p("}")
		}
		g.p(`return %s`, returnArgsString)

	}

	g.out()
	g.p("}")
	return nil
}

func (g *generator) GenerateTracedRecorderMethod(mockType string, m *model.Method, shortTp string) error {
	argNames := g.getArgNames(m, true)

	var argString string
	if m.Variadic == nil {
		argString = strings.Join(argNames, ", ")
	} else {
		argString = strings.Join(argNames[:len(argNames)-1], ", ")
	}
	if argString != "" {
		argString += " interface{}"
	}

	if m.Variadic != nil {
		if argString != "" {
			argString += ", "
		}
		argString += fmt.Sprintf("%s ...interface{}", argNames[len(argNames)-1])
	}

	ia := newIdentifierAllocator(argNames)
	idRecv := ia.allocateIdentifier("mr")

	g.p("// %v indicates an expected call of %v.", m.Name, m.Name)
	g.p("func (%s *%vMockRecorder%v) %v(%v) *gomock.Call {", idRecv, mockType, shortTp, m.Name, argString)
	g.in()
	g.p("%s.mock.ctrl.T.Helper()", idRecv)

	var callArgs string
	if m.Variadic == nil {
		if len(argNames) > 0 {
			callArgs = ", " + strings.Join(argNames, ", ")
		}
	} else {
		if len(argNames) == 1 {
			// Easy: just use ... to push the arguments through.
			callArgs = ", " + argNames[0] + "..."
		} else {
			// Hard: create a temporary slice.
			idVarArgs := ia.allocateIdentifier("varargs")
			g.p("%s := append([]interface{}{%s}, %s...)",
				idVarArgs,
				strings.Join(argNames[:len(argNames)-1], ", "),
				argNames[len(argNames)-1])
			callArgs = ", " + idVarArgs + "..."
		}
	}
	g.p(`return %s.mock.ctrl.RecordCallWithMethodType(%s.mock, "%s", reflect.TypeOf((*%s%s)(nil).%s)%s)`, idRecv, idRecv, m.Name, mockType, shortTp, m.Name, callArgs)

	g.out()
	g.p("}")
	return nil
}
