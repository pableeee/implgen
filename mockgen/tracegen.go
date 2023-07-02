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

	"go.uber.org/mock/mockgen/model"
)

// The name of the mock type to use for the given interface identifier.
func (g *generator) tracedName(typeName string) string {
	if mockName, ok := g.mockNames[typeName]; ok {
		return mockName
	}

	return "Traced" + typeName
}

func (g *generator) GenerateTracedInterface(intf *model.Interface, outputPackagePath string) error {
	mockType := g.mockName(intf.Name)
	longTp, shortTp := g.formattedTypeParams(intf, outputPackagePath)

	g.p("")
	g.p("// %v is a tracing decorator of %v interface.", mockType, intf.Name)
	g.p("type Traced%v interface {", intf.Name)
	g.in()

	for _, m := range intf.Methods {
		argNames := g.getArgNames(m)
		argTypes := g.getArgTypes(m, outputPackagePath)
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
	g.p("tracer     trace.Tracer")
	g.out()
	g.p("}")
	g.p("")

	// g.p("// %vMockRecorder is the mock recorder for %v.", mockType, mockType)
	// g.p("type %vMockRecorder%v struct {", mockType, longTp)
	// g.in()
	// g.p("mock *%v%v", mockType, shortTp)
	// g.out()
	// g.p("}")
	// g.p("")

	g.p("// New%v creates a new trace decorator instance.", mockType)
	g.p("func New%v%v(ctrl Traced%v) *%v%v {", mockType, longTp, intf.Name, mockType, shortTp)
	g.in()
	g.p(`deco := &%v%v{delegate: ctrl, tracer: otel.Tracer("%s")}`, mockType, shortTp, intf.Name)
	g.p("return deco")
	g.out()
	g.p("}")
	g.p("")

	// // XXX: possible name collision here if someone has EXPECT in their interface.
	// g.p("// EXPECT returns an object that allows the caller to indicate expected use.")
	// g.p("func (m *%v%v) EXPECT() *%vMockRecorder%v {", mockType, shortTp, mockType, shortTp)
	// g.in()
	// g.p("return m.recorder")
	// g.out()
	// g.p("}")

	g.GenerateTracedMethods(mockType, intf, outputPackagePath, shortTp)

	return nil
}
func (g *generator) GenerateTracedMethods(mockType string, intf *model.Interface, pkgOverride, shortTp string) {
	sort.Sort(byMethodName(intf.Methods))
	for _, m := range intf.Methods {
		g.p("")
		_ = g.GenerateTracedMethod(mockType, m, pkgOverride, shortTp)
	}
}

// GenerateMockMethod generates a mock method implementation.
// If non-empty, pkgOverride is the package in which unqualified types reside.
func (g *generator) GenerateTracedMethod(mockType string, m *model.Method, pkgOverride, shortTp string) error {
	argNames := g.getArgNames(m)
	argTypes := g.getArgTypes(m, pkgOverride)
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

	g.p("// %v traced base method.", m.Name)
	g.p("func (%v *%v%v) %v(%v)%v {", idRecv, mockType, shortTp, m.Name, argString, retString)
	g.in()
	g.p("_, span := %s.tracer.Start(context.TODO(), %q)", idRecv, m.Name)
	g.p("defer span.End()")

	var callArgs string
	if m.Variadic == nil {
		if len(argNames) > 0 {
			callArgs = strings.Join(argNames, ", ")
		}
	} else {
		// Non-trivial. The generated code must build a []interface{},
		// but the variadic argument may be any type.
		// idVarArgs := ia.allocateIdentifier("varargs")
		// idVArg := ia.allocateIdentifier("a")
		// g.p("%s := []interface{}{%s}", idVarArgs, strings.Join(argNames[:len(argNames)-1], ", "))
		// g.p("for _, %s := range %s {", idVArg, argNames[len(argNames)-1])
		// g.in()
		// g.p("%s = append(%s, %s)", idVarArgs, idVarArgs, idVArg)
		// g.out()
		// g.p("}")
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
		// idRet := ia.allocateIdentifier("ret")
		g.p(`return %s.delegate.%s(%s)`, idRecv, m.Name, callArgs)

		// Go does not allow "naked" type assertions on nil values, so we use the two-value form here.
		// The value of that is either (x.(T), true) or (Z, false), where Z is the zero value for T.
		// Happily, this coincides with the semantics we want here.
		// retNames := make([]string, len(rets))
		// for i, t := range rets {
		// 	retNames[i] = ia.allocateIdentifier(fmt.Sprintf("ret%d", i))
		// 	g.p("%s, _ := %s[%d].(%s)", retNames[i], idRet, i, t)
		// }
		// g.p("return " + strings.Join(retNames, ", "))
	}

	g.out()
	g.p("}")
	return nil
}

func (g *generator) GenerateTracedRecorderMethod(mockType string, m *model.Method, shortTp string) error {
	argNames := g.getArgNames(m)

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
