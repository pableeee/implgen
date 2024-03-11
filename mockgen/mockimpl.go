package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pableeee/implgen/mockgen/model"
)

func generateMockInterface(g *generator, intf *model.Interface, outputPackagePath string) error {
	mockType := g.mockName(intf.Name)
	longTp, shortTp := g.formattedTypeParams(intf, outputPackagePath)

	g.p("")
	g.p("// %v is a mock of %v interface.", mockType, intf.Name)
	g.p("type %v%v struct {", mockType, longTp)
	g.in()
	g.p("ctrl     *gomock.Controller")
	g.p("recorder *%vMockRecorder%v", mockType, shortTp)
	g.out()
	g.p("}")
	g.p("")

	g.p("// %vMockRecorder is the mock recorder for %v.", mockType, mockType)
	g.p("type %vMockRecorder%v struct {", mockType, longTp)
	g.in()
	g.p("mock *%v%v", mockType, shortTp)
	g.out()
	g.p("}")
	g.p("")

	g.p("// New%v creates a new mock instance.", mockType)
	g.p("func New%v%v(ctrl *gomock.Controller) *%v%v {", mockType, longTp, mockType, shortTp)
	g.in()
	g.p("mock := &%v%v{ctrl: ctrl}", mockType, shortTp)
	g.p("mock.recorder = &%vMockRecorder%v{mock}", mockType, shortTp)
	g.p("return mock")
	g.out()
	g.p("}")
	g.p("")

	// XXX: possible name collision here if someone has EXPECT in their interface.
	g.p("// EXPECT returns an object that allows the caller to indicate expected use.")
	g.p("func (m *%v%v) EXPECT() *%vMockRecorder%v {", mockType, shortTp, mockType, shortTp)
	g.in()
	g.p("return m.recorder")
	g.out()
	g.p("}")

	generateMockMethods(g, mockType, intf, outputPackagePath, shortTp)

	return nil
}

func generateMockMethods(g *generator, mockType string, intf *model.Interface, pkgOverride, shortTp string) {
	sort.Sort(byMethodName(intf.Methods))
	for _, m := range intf.Methods {
		g.p("")
		_ = generateMockMethod(g, mockType, m, pkgOverride, shortTp)
		g.p("")
		_ = generateMockRecorderMethod(g, mockType, m, shortTp)
	}
}

func generateMockMethod(g *generator, mockType string, m *model.Method, pkgOverride, shortTp string) error {
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
	idRecv := ia.allocateIdentifier("m")

	g.p("// %v mocks base method.", m.Name)
	g.p("func (%v *%v%v) %v(%v)%v {", idRecv, mockType, shortTp, m.Name, argString, retString)
	g.in()
	g.p("%s.ctrl.T.Helper()", idRecv)

	var callArgs string
	if m.Variadic == nil {
		if len(argNames) > 0 {
			callArgs = ", " + strings.Join(argNames, ", ")
		}
	} else {
		// Non-trivial. The generated code must build a []interface{},
		// but the variadic argument may be any type.
		idVarArgs := ia.allocateIdentifier("varargs")
		idVArg := ia.allocateIdentifier("a")
		g.p("%s := []interface{}{%s}", idVarArgs, strings.Join(argNames[:len(argNames)-1], ", "))
		g.p("for _, %s := range %s {", idVArg, argNames[len(argNames)-1])
		g.in()
		g.p("%s = append(%s, %s)", idVarArgs, idVarArgs, idVArg)
		g.out()
		g.p("}")
		callArgs = ", " + idVarArgs + "..."
	}
	if len(m.Out) == 0 {
		g.p(`%v.ctrl.Call(%v, %q%v)`, idRecv, idRecv, m.Name, callArgs)
	} else {
		idRet := ia.allocateIdentifier("ret")
		g.p(`%v := %v.ctrl.Call(%v, %q%v)`, idRet, idRecv, idRecv, m.Name, callArgs)

		// Go does not allow "naked" type assertions on nil values, so we use the two-value form here.
		// The value of that is either (x.(T), true) or (Z, false), where Z is the zero value for T.
		// Happily, this coincides with the semantics we want here.
		retNames := make([]string, len(rets))
		for i, t := range rets {
			retNames[i] = ia.allocateIdentifier(fmt.Sprintf("ret%d", i))
			g.p("%s, _ := %s[%d].(%s)", retNames[i], idRet, i, t)
		}
		g.p("return " + strings.Join(retNames, ", "))
	}

	g.out()
	g.p("}")
	return nil
}

func generateMockRecorderMethod(g *generator, mockType string, m *model.Method, shortTp string) error {
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
