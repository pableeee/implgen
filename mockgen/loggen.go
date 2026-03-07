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

// loggedName returns the name of the logging decorator type for the given interface.
func (g *generator) loggedName(typeName string) string {
	if mockName, ok := g.mockNames[typeName]; ok {
		return mockName
	}
	return "Logged" + typeName + "Impl"
}

func generateLoggedInterface(g *generator, intf *model.Interface, outputPackagePath string) error {
	mockType := g.loggedName(intf.Name)
	longTp, shortTp := g.formattedTypeParams(intf, outputPackagePath)

	sort.Sort(byMethodName(intf.Methods))

	// Struct with delegate (original interface) and logger.
	g.p("")
	g.p("// %v is a logging decorator of %v interface.", mockType, intf.Name)
	g.p("type %v%v struct {", mockType, longTp)
	g.in()
	g.p("delegate %v", intf.Name)
	g.p("logger   *slog.Logger")
	g.out()
	g.p("}")
	g.p("")

	// Constructor.
	g.p("// New%v creates a new logging decorator instance.", mockType)
	g.p("// logger may be nil; if so, slog.Default() is used.")
	g.p("func New%v%v(delegate %v, logger *slog.Logger) *%v%v {",
		mockType, longTp, intf.Name, mockType, shortTp)
	g.in()
	g.p("if logger == nil {")
	g.in()
	g.p("logger = slog.Default()")
	g.out()
	g.p("}")
	g.p(`return &%v%v{delegate: delegate, logger: logger}`, mockType, shortTp)
	g.out()
	g.p("}")
	g.p("")

	for _, m := range intf.Methods {
		g.p("")
		_ = generateLoggedMethod(g, mockType, intf.Name, m, outputPackagePath, shortTp)
	}

	return nil
}

func generateLoggedMethod(g *generator, mockType, intfName string, m *model.Method, pkgOverride, shortTp string) error {
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

	// Determine if there's a context arg (for context-aware logging) and an error return.
	isContextMethod := len(argTypes) > 0 && argTypes[0] == "context.Context"
	returnsError := false
	errorIndex := -1
	for i, r := range rets {
		if r == "error" {
			returnsError = true
			errorIndex = i
		}
	}

	g.p("// %v logged base method.", m.Name)
	g.p("func (%v *%v%v) %v(%v)%v {", idRecv, mockType, shortTp, m.Name, argString, retString)
	g.in()

	// Log the call entry.
	if isContextMethod {
		ctxArg := argNames[0]
		g.p(`%v.logger.InfoContext(%v, "calling %v.%v")`, idRecv, ctxArg, intfName, m.Name)
	} else {
		g.p(`%v.logger.Info("calling %v.%v")`, idRecv, intfName, m.Name)
	}

	if len(m.Out) == 0 {
		g.p(`%s.delegate.%s(%s)`, idRecv, m.Name, callArgs)
		if isContextMethod {
			ctxArg := argNames[0]
			g.p(`%v.logger.InfoContext(%v, "completed %v.%v")`, idRecv, ctxArg, intfName, m.Name)
		} else {
			g.p(`%v.logger.Info("completed %v.%v")`, idRecv, intfName, m.Name)
		}
	} else {
		returns := make([]string, len(rets))
		for i := range rets {
			returns[i] = ia.allocateIdentifier("ret")
		}
		returnArgsString := strings.Join(returns, ", ")
		g.p(`%s := %s.delegate.%s(%s)`, returnArgsString, idRecv, m.Name, callArgs)

		if returnsError {
			errVar := returns[errorIndex]
			g.p(`if %v != nil {`, errVar)
			g.in()
			if isContextMethod {
				ctxArg := argNames[0]
				g.p(`%v.logger.ErrorContext(%v, "error in %v.%v", "error", %v)`, idRecv, ctxArg, intfName, m.Name, errVar)
			} else {
				g.p(`%v.logger.Error("error in %v.%v", "error", %v)`, idRecv, intfName, m.Name, errVar)
			}
			g.out()
			g.p("} else {")
			g.in()
			if isContextMethod {
				ctxArg := argNames[0]
				g.p(`%v.logger.InfoContext(%v, "completed %v.%v")`, idRecv, ctxArg, intfName, m.Name)
			} else {
				g.p(`%v.logger.Info("completed %v.%v")`, idRecv, intfName, m.Name)
			}
			g.out()
			g.p("}")
		} else {
			if isContextMethod {
				ctxArg := argNames[0]
				g.p(`%v.logger.InfoContext(%v, "completed %v.%v")`, idRecv, ctxArg, intfName, m.Name)
			} else {
				g.p(`%v.logger.Info("completed %v.%v")`, idRecv, intfName, m.Name)
			}
		}
		g.p(`return %s`, returnArgsString)
	}

	g.out()
	g.p("}")
	return nil
}
