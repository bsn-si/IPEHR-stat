// Code generated by MockGen. DO NOT EDIT.
// Source: aql_query.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	model "github.com/bsn-si/IPEHR-gateway/src/pkg/docs/model"
	gomock "github.com/golang/mock/gomock"
)

// MockAQLQuerier is a mock of AQLQuerier interface.
type MockAQLQuerier struct {
	ctrl     *gomock.Controller
	recorder *MockAQLQuerierMockRecorder
}

// MockAQLQuerierMockRecorder is the mock recorder for MockAQLQuerier.
type MockAQLQuerierMockRecorder struct {
	mock *MockAQLQuerier
}

// NewMockAQLQuerier creates a new mock instance.
func NewMockAQLQuerier(ctrl *gomock.Controller) *MockAQLQuerier {
	mock := &MockAQLQuerier{ctrl: ctrl}
	mock.recorder = &MockAQLQuerierMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAQLQuerier) EXPECT() *MockAQLQuerierMockRecorder {
	return m.recorder
}

// ExecQuery mocks base method.
func (m *MockAQLQuerier) ExecQuery(ctx context.Context, query *model.QueryRequest) (*model.QueryResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExecQuery", ctx, query)
	ret0, _ := ret[0].(*model.QueryResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ExecQuery indicates an expected call of ExecQuery.
func (mr *MockAQLQuerierMockRecorder) ExecQuery(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecQuery", reflect.TypeOf((*MockAQLQuerier)(nil).ExecQuery), ctx, query)
}
