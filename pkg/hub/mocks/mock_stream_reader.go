// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/thebartekbanach/imcaxy/pkg/hub/storage (interfaces: StreamReader)

// Package mock_hub is a generated GoMock package.
package mock_hub

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockStreamReader is a mock of StreamReader interface.
type MockStreamReader struct {
	ctrl     *gomock.Controller
	recorder *MockStreamReaderMockRecorder
}

// MockStreamReaderMockRecorder is the mock recorder for MockStreamReader.
type MockStreamReaderMockRecorder struct {
	mock *MockStreamReader
}

// NewMockStreamReader creates a new mock instance.
func NewMockStreamReader(ctrl *gomock.Controller) *MockStreamReader {
	mock := &MockStreamReader{ctrl: ctrl}
	mock.recorder = &MockStreamReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStreamReader) EXPECT() *MockStreamReaderMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockStreamReader) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockStreamReaderMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockStreamReader)(nil).Close))
}

// ReadAt mocks base method.
func (m *MockStreamReader) ReadAt(arg0 []byte, arg1 int64) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadAt", arg0, arg1)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadAt indicates an expected call of ReadAt.
func (mr *MockStreamReaderMockRecorder) ReadAt(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadAt", reflect.TypeOf((*MockStreamReader)(nil).ReadAt), arg0, arg1)
}
