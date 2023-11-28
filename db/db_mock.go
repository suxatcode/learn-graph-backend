// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/suxatcode/learn-graph-poc-backend/db (interfaces: DB)

// Package db is a generated GoMock package.
package db

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	model "github.com/suxatcode/learn-graph-poc-backend/graph/model"
)

// MockDB is a mock of DB interface.
type MockDB struct {
	ctrl     *gomock.Controller
	recorder *MockDBMockRecorder
}

// MockDBMockRecorder is the mock recorder for MockDB.
type MockDBMockRecorder struct {
	mock *MockDB
}

// NewMockDB creates a new mock instance.
func NewMockDB(ctrl *gomock.Controller) *MockDB {
	mock := &MockDB{ctrl: ctrl}
	mock.recorder = &MockDBMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDB) EXPECT() *MockDBMockRecorder {
	return m.recorder
}

// CreateEdge mocks base method.
func (m *MockDB) CreateEdge(arg0 context.Context, arg1 User, arg2, arg3 string, arg4 float64) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateEdge", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateEdge indicates an expected call of CreateEdge.
func (mr *MockDBMockRecorder) CreateEdge(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateEdge", reflect.TypeOf((*MockDB)(nil).CreateEdge), arg0, arg1, arg2, arg3, arg4)
}

// CreateNode mocks base method.
func (m *MockDB) CreateNode(arg0 context.Context, arg1 User, arg2 *model.Text) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateNode", arg0, arg1, arg2)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateNode indicates an expected call of CreateNode.
func (mr *MockDBMockRecorder) CreateNode(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateNode", reflect.TypeOf((*MockDB)(nil).CreateNode), arg0, arg1, arg2)
}

// CreateUserWithEMail mocks base method.
func (m *MockDB) CreateUserWithEMail(arg0 context.Context, arg1, arg2, arg3 string) (*model.CreateUserResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUserWithEMail", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(*model.CreateUserResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateUserWithEMail indicates an expected call of CreateUserWithEMail.
func (mr *MockDBMockRecorder) CreateUserWithEMail(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUserWithEMail", reflect.TypeOf((*MockDB)(nil).CreateUserWithEMail), arg0, arg1, arg2, arg3)
}

// DeleteAccount mocks base method.
func (m *MockDB) DeleteAccount(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAccount", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAccount indicates an expected call of DeleteAccount.
func (mr *MockDBMockRecorder) DeleteAccount(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAccount", reflect.TypeOf((*MockDB)(nil).DeleteAccount), arg0)
}

// EditNode mocks base method.
func (m *MockDB) EditNode(arg0 context.Context, arg1 string, arg2 *model.Text) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EditNode", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// EditNode indicates an expected call of EditNode.
func (mr *MockDBMockRecorder) EditNode(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EditNode", reflect.TypeOf((*MockDB)(nil).EditNode), arg0, arg1, arg2)
}

// Graph mocks base method.
func (m *MockDB) Graph(arg0 context.Context) (*model.Graph, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Graph", arg0)
	ret0, _ := ret[0].(*model.Graph)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Graph indicates an expected call of Graph.
func (mr *MockDBMockRecorder) Graph(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Graph", reflect.TypeOf((*MockDB)(nil).Graph), arg0)
}

// IsUserAuthenticated mocks base method.
func (m *MockDB) IsUserAuthenticated(arg0 context.Context) (bool, *User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsUserAuthenticated", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(*User)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// IsUserAuthenticated indicates an expected call of IsUserAuthenticated.
func (mr *MockDBMockRecorder) IsUserAuthenticated(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsUserAuthenticated", reflect.TypeOf((*MockDB)(nil).IsUserAuthenticated), arg0)
}

// Login mocks base method.
func (m *MockDB) Login(arg0 context.Context, arg1 model.LoginAuthentication) (*model.LoginResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Login", arg0, arg1)
	ret0, _ := ret[0].(*model.LoginResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Login indicates an expected call of Login.
func (mr *MockDBMockRecorder) Login(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Login", reflect.TypeOf((*MockDB)(nil).Login), arg0, arg1)
}

// Logout mocks base method.
func (m *MockDB) Logout(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Logout", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Logout indicates an expected call of Logout.
func (mr *MockDBMockRecorder) Logout(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Logout", reflect.TypeOf((*MockDB)(nil).Logout), arg0)
}

// SetEdgeWeight mocks base method.
func (m *MockDB) SetEdgeWeight(arg0 context.Context, arg1 string, arg2 float64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetEdgeWeight", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetEdgeWeight indicates an expected call of SetEdgeWeight.
func (mr *MockDBMockRecorder) SetEdgeWeight(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetEdgeWeight", reflect.TypeOf((*MockDB)(nil).SetEdgeWeight), arg0, arg1, arg2)
}