// Code generated by MockGen. DO NOT EDIT.
// Source: user.go
//
// Generated by this command:
//
//	mockgen --source=user.go --destination=mock_test.go --package=users_test
//

// Package users_test is a generated GoMock package.
package users_test

import (
	reflect "reflect"

	gomock "github.com/pableeee/implgen/gomock"
	users "github.com/pableeee/implgen/mockgen/internal/tests/mock_in_test_package"
)

// MockFinder is a mock of Finder interface.
type MockFinder struct {
	ctrl     *gomock.Controller
	recorder *MockFinderMockRecorder
}

// MockFinderMockRecorder is the mock recorder for MockFinder.
type MockFinderMockRecorder struct {
	mock *MockFinder
}

// NewMockFinder creates a new mock instance.
func NewMockFinder(ctrl *gomock.Controller) *MockFinder {
	mock := &MockFinder{ctrl: ctrl}
	mock.recorder = &MockFinderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFinder) EXPECT() *MockFinderMockRecorder {
	return m.recorder
}

// Add mocks base method.
func (m *MockFinder) Add(u users.User) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Add", u)
}

// Add indicates an expected call of Add.
func (mr *MockFinderMockRecorder) Add(u any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Add", reflect.TypeOf((*MockFinder)(nil).Add), u)
}

// FindUser mocks base method.
func (m *MockFinder) FindUser(name string) users.User {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindUser", name)
	ret0, _ := ret[0].(users.User)
	return ret0
}

// FindUser indicates an expected call of FindUser.
func (mr *MockFinderMockRecorder) FindUser(name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindUser", reflect.TypeOf((*MockFinder)(nil).FindUser), name)
}
