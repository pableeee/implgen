// Code generated by MockGen. DO NOT EDIT.
// Source: panic.go
//
// Generated by this command:
//
//	mockgen --source=panic.go --destination=mock_test.go --package=paniccode
//

// Package paniccode is a generated GoMock package.
package paniccode

import (
	reflect "reflect"

	gomock "github.com/pableeee/implgen/gomock"
)

// MockFoo is a mock of Foo interface.
type MockFoo struct {
	ctrl     *gomock.Controller
	recorder *MockFooMockRecorder
}

// MockFooMockRecorder is the mock recorder for MockFoo.
type MockFooMockRecorder struct {
	mock *MockFoo
}

// NewMockFoo creates a new mock instance.
func NewMockFoo(ctrl *gomock.Controller) *MockFoo {
	mock := &MockFoo{ctrl: ctrl}
	mock.recorder = &MockFooMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFoo) EXPECT() *MockFooMockRecorder {
	return m.recorder
}

// Bar mocks base method.
func (m *MockFoo) Bar() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Bar")
	ret0, _ := ret[0].(string)
	return ret0
}

// Bar indicates an expected call of Bar.
func (mr *MockFooMockRecorder) Bar() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Bar", reflect.TypeOf((*MockFoo)(nil).Bar))
}

// Baz mocks base method.
func (m *MockFoo) Baz() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Baz")
	ret0, _ := ret[0].(string)
	return ret0
}

// Baz indicates an expected call of Baz.
func (mr *MockFooMockRecorder) Baz() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Baz", reflect.TypeOf((*MockFoo)(nil).Baz))
}
