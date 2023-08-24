// Code generated by MockGen. DO NOT EDIT.
// Source: host.go

// Package test_host is a generated GoMock package.
package test_host

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	anchor "github.com/wetware/pkg/cap/anchor"
	capstore "github.com/wetware/pkg/cap/capstore"
	csp "github.com/wetware/pkg/cap/csp"
	pubsub "github.com/wetware/pkg/cap/pubsub"
	service "github.com/wetware/pkg/cap/registry"
	view "github.com/wetware/pkg/cap/view"
)

// MockViewProvider is a mock of ViewProvider interface.
type MockViewProvider struct {
	ctrl     *gomock.Controller
	recorder *MockViewProviderMockRecorder
}

// MockViewProviderMockRecorder is the mock recorder for MockViewProvider.
type MockViewProviderMockRecorder struct {
	mock *MockViewProvider
}

// NewMockViewProvider creates a new mock instance.
func NewMockViewProvider(ctrl *gomock.Controller) *MockViewProvider {
	mock := &MockViewProvider{ctrl: ctrl}
	mock.recorder = &MockViewProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockViewProvider) EXPECT() *MockViewProviderMockRecorder {
	return m.recorder
}

// View mocks base method.
func (m *MockViewProvider) View() view.View {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "View")
	ret0, _ := ret[0].(view.View)
	return ret0
}

// View indicates an expected call of View.
func (mr *MockViewProviderMockRecorder) View() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "View", reflect.TypeOf((*MockViewProvider)(nil).View))
}

// MockPubSubProvider is a mock of PubSubProvider interface.
type MockPubSubProvider struct {
	ctrl     *gomock.Controller
	recorder *MockPubSubProviderMockRecorder
}

// MockPubSubProviderMockRecorder is the mock recorder for MockPubSubProvider.
type MockPubSubProviderMockRecorder struct {
	mock *MockPubSubProvider
}

// NewMockPubSubProvider creates a new mock instance.
func NewMockPubSubProvider(ctrl *gomock.Controller) *MockPubSubProvider {
	mock := &MockPubSubProvider{ctrl: ctrl}
	mock.recorder = &MockPubSubProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPubSubProvider) EXPECT() *MockPubSubProviderMockRecorder {
	return m.recorder
}

// PubSub mocks base method.
func (m *MockPubSubProvider) PubSub() pubsub.Router {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PubSub")
	ret0, _ := ret[0].(pubsub.Router)
	return ret0
}

// PubSub indicates an expected call of PubSub.
func (mr *MockPubSubProviderMockRecorder) PubSub() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PubSub", reflect.TypeOf((*MockPubSubProvider)(nil).PubSub))
}

// MockAnchorProvider is a mock of AnchorProvider interface.
type MockAnchorProvider struct {
	ctrl     *gomock.Controller
	recorder *MockAnchorProviderMockRecorder
}

// MockAnchorProviderMockRecorder is the mock recorder for MockAnchorProvider.
type MockAnchorProviderMockRecorder struct {
	mock *MockAnchorProvider
}

// NewMockAnchorProvider creates a new mock instance.
func NewMockAnchorProvider(ctrl *gomock.Controller) *MockAnchorProvider {
	mock := &MockAnchorProvider{ctrl: ctrl}
	mock.recorder = &MockAnchorProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAnchorProvider) EXPECT() *MockAnchorProviderMockRecorder {
	return m.recorder
}

// Anchor mocks base method.
func (m *MockAnchorProvider) Anchor() anchor.Anchor {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Anchor")
	ret0, _ := ret[0].(anchor.Anchor)
	return ret0
}

// Anchor indicates an expected call of Anchor.
func (mr *MockAnchorProviderMockRecorder) Anchor() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Anchor", reflect.TypeOf((*MockAnchorProvider)(nil).Anchor))
}

// MockRegistryProvider is a mock of RegistryProvider interface.
type MockRegistryProvider struct {
	ctrl     *gomock.Controller
	recorder *MockRegistryProviderMockRecorder
}

// MockRegistryProviderMockRecorder is the mock recorder for MockRegistryProvider.
type MockRegistryProviderMockRecorder struct {
	mock *MockRegistryProvider
}

// NewMockRegistryProvider creates a new mock instance.
func NewMockRegistryProvider(ctrl *gomock.Controller) *MockRegistryProvider {
	mock := &MockRegistryProvider{ctrl: ctrl}
	mock.recorder = &MockRegistryProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRegistryProvider) EXPECT() *MockRegistryProviderMockRecorder {
	return m.recorder
}

// Registry mocks base method.
func (m *MockRegistryProvider) Registry() service.Registry {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Registry")
	ret0, _ := ret[0].(service.Registry)
	return ret0
}

// Registry indicates an expected call of Registry.
func (mr *MockRegistryProviderMockRecorder) Registry() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Registry", reflect.TypeOf((*MockRegistryProvider)(nil).Registry))
}

// MockExecutorProvider is a mock of ExecutorProvider interface.
type MockExecutorProvider struct {
	ctrl     *gomock.Controller
	recorder *MockExecutorProviderMockRecorder
}

// MockExecutorProviderMockRecorder is the mock recorder for MockExecutorProvider.
type MockExecutorProviderMockRecorder struct {
	mock *MockExecutorProvider
}

// NewMockExecutorProvider creates a new mock instance.
func NewMockExecutorProvider(ctrl *gomock.Controller) *MockExecutorProvider {
	mock := &MockExecutorProvider{ctrl: ctrl}
	mock.recorder = &MockExecutorProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockExecutorProvider) EXPECT() *MockExecutorProviderMockRecorder {
	return m.recorder
}

// Executor mocks base method.
func (m *MockExecutorProvider) Executor() csp.Executor {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Executor")
	ret0, _ := ret[0].(csp.Executor)
	return ret0
}

// Executor indicates an expected call of Executor.
func (mr *MockExecutorProviderMockRecorder) Executor() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Executor", reflect.TypeOf((*MockExecutorProvider)(nil).Executor))
}

// MockCapStoreProvider is a mock of CapStoreProvider interface.
type MockCapStoreProvider struct {
	ctrl     *gomock.Controller
	recorder *MockCapStoreProviderMockRecorder
}

// MockCapStoreProviderMockRecorder is the mock recorder for MockCapStoreProvider.
type MockCapStoreProviderMockRecorder struct {
	mock *MockCapStoreProvider
}

// NewMockCapStoreProvider creates a new mock instance.
func NewMockCapStoreProvider(ctrl *gomock.Controller) *MockCapStoreProvider {
	mock := &MockCapStoreProvider{ctrl: ctrl}
	mock.recorder = &MockCapStoreProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCapStoreProvider) EXPECT() *MockCapStoreProviderMockRecorder {
	return m.recorder
}

// CapStore mocks base method.
func (m *MockCapStoreProvider) CapStore() capstore.CapStore {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CapStore")
	ret0, _ := ret[0].(capstore.CapStore)
	return ret0
}

// CapStore indicates an expected call of CapStore.
func (mr *MockCapStoreProviderMockRecorder) CapStore() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CapStore", reflect.TypeOf((*MockCapStoreProvider)(nil).CapStore))
}
