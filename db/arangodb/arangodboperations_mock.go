// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/suxatcode/learn-graph-poc-backend/db/arangodb (interfaces: ArangoDBOperations)

// Package arangodb is a generated GoMock package.
package arangodb

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	db "github.com/suxatcode/learn-graph-poc-backend/db"
)

// MockArangoDBOperations is a mock of ArangoDBOperations interface.
type MockArangoDBOperations struct {
	ctrl     *gomock.Controller
	recorder *MockArangoDBOperationsMockRecorder
}

// MockArangoDBOperationsMockRecorder is the mock recorder for MockArangoDBOperations.
type MockArangoDBOperationsMockRecorder struct {
	mock *MockArangoDBOperations
}

// NewMockArangoDBOperations creates a new mock instance.
func NewMockArangoDBOperations(ctrl *gomock.Controller) *MockArangoDBOperations {
	mock := &MockArangoDBOperations{ctrl: ctrl}
	mock.recorder = &MockArangoDBOperationsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockArangoDBOperations) EXPECT() *MockArangoDBOperationsMockRecorder {
	return m.recorder
}

// AddNodeToEditNode mocks base method.
func (m *MockArangoDBOperations) AddNodeToEditNode(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddNodeToEditNode", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddNodeToEditNode indicates an expected call of AddNodeToEditNode.
func (mr *MockArangoDBOperationsMockRecorder) AddNodeToEditNode(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddNodeToEditNode", reflect.TypeOf((*MockArangoDBOperations)(nil).AddNodeToEditNode), arg0)
}

// CollectionsExist mocks base method.
func (m *MockArangoDBOperations) CollectionsExist(arg0 context.Context) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CollectionsExist", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CollectionsExist indicates an expected call of CollectionsExist.
func (mr *MockArangoDBOperationsMockRecorder) CollectionsExist(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CollectionsExist", reflect.TypeOf((*MockArangoDBOperations)(nil).CollectionsExist), arg0)
}

// CreateDBWithSchema mocks base method.
func (m *MockArangoDBOperations) CreateDBWithSchema(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateDBWithSchema", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateDBWithSchema indicates an expected call of CreateDBWithSchema.
func (mr *MockArangoDBOperationsMockRecorder) CreateDBWithSchema(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateDBWithSchema", reflect.TypeOf((*MockArangoDBOperations)(nil).CreateDBWithSchema), arg0)
}

// Init mocks base method.
func (m *MockArangoDBOperations) Init(arg0 db.Config) (db.DB, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Init", arg0)
	ret0, _ := ret[0].(db.DB)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Init indicates an expected call of Init.
func (mr *MockArangoDBOperationsMockRecorder) Init(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Init", reflect.TypeOf((*MockArangoDBOperations)(nil).Init), arg0)
}

// OpenDatabase mocks base method.
func (m *MockArangoDBOperations) OpenDatabase(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "OpenDatabase", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// OpenDatabase indicates an expected call of OpenDatabase.
func (mr *MockArangoDBOperationsMockRecorder) OpenDatabase(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OpenDatabase", reflect.TypeOf((*MockArangoDBOperations)(nil).OpenDatabase), arg0)
}

// ValidateSchema mocks base method.
func (m *MockArangoDBOperations) ValidateSchema(arg0 context.Context) (SchemaUpdateAction, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateSchema", arg0)
	ret0, _ := ret[0].(SchemaUpdateAction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ValidateSchema indicates an expected call of ValidateSchema.
func (mr *MockArangoDBOperationsMockRecorder) ValidateSchema(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateSchema", reflect.TypeOf((*MockArangoDBOperations)(nil).ValidateSchema), arg0)
}