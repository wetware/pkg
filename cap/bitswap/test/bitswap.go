// Code generated by MockGen. DO NOT EDIT.
// Source: bitswap.go

// Package test_bitswap is a generated GoMock package.
package test_bitswap

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	blocks "github.com/ipfs/go-block-format"
	cid "github.com/ipfs/go-cid"
)

// MockExchange is a mock of Exchange interface.
type MockExchange struct {
	ctrl     *gomock.Controller
	recorder *MockExchangeMockRecorder
}

// MockExchangeMockRecorder is the mock recorder for MockExchange.
type MockExchangeMockRecorder struct {
	mock *MockExchange
}

// NewMockExchange creates a new mock instance.
func NewMockExchange(ctrl *gomock.Controller) *MockExchange {
	mock := &MockExchange{ctrl: ctrl}
	mock.recorder = &MockExchangeMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockExchange) EXPECT() *MockExchangeMockRecorder {
	return m.recorder
}

// GetBlock mocks base method.
func (m *MockExchange) GetBlock(arg0 context.Context, arg1 cid.Cid) (blocks.Block, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlock", arg0, arg1)
	ret0, _ := ret[0].(blocks.Block)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlock indicates an expected call of GetBlock.
func (mr *MockExchangeMockRecorder) GetBlock(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlock", reflect.TypeOf((*MockExchange)(nil).GetBlock), arg0, arg1)
}
