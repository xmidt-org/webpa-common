// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package device

import (
	"net/http"

	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/webpa-common/v2/convey"
)

type MockConnector struct {
	mock.Mock
}

var _ Connector = (*MockConnector)(nil)

func (m *MockConnector) Connect(response http.ResponseWriter, request *http.Request, header http.Header) (Interface, error) {
	// nolint: typecheck
	arguments := m.Called(response, request, header)
	first, _ := arguments.Get(0).(Interface)
	return first, arguments.Error(1)
}

func (m *MockConnector) Disconnect(id ID, reason CloseReason) bool {
	// nolint: typecheck
	return m.Called(id, reason).Bool(0)
}

func (m *MockConnector) DisconnectIf(predicate func(ID) (CloseReason, bool)) int {
	// nolint: typecheck
	return m.Called(predicate).Int(0)
}

func (m *MockConnector) DisconnectAll(reason CloseReason) int {
	// nolint: typecheck
	return m.Called(reason).Int(0)
}

func (m *MockConnector) GetFilter() Filter {
	// nolint: typecheck
	return m.Called().Get(0).(Filter)
}

type MockRegistry struct {
	mock.Mock
}

var _ Registry = (*MockRegistry)(nil)

func (m *MockRegistry) Len() int {
	// nolint: typecheck
	return m.Called().Int(0)
}

func (m *MockRegistry) Get(id ID) (Interface, bool) {
	// nolint: typecheck
	arguments := m.Called(id)
	first, _ := arguments.Get(0).(Interface)
	return first, arguments.Bool(1)
}

func (m *MockRegistry) VisitAll(f func(Interface) bool) int {
	// nolint: typecheck
	return m.Called(f).Int(0)
}

type MockDevice struct {
	mock.Mock
}

func (m *MockDevice) String() string {
	// nolint: typecheck
	return m.Called().String(0)
}

func (m *MockDevice) MarshalJSON() ([]byte, error) {
	// nolint: typecheck
	arguments := m.Called()
	return arguments.Get(0).([]byte), arguments.Error(1)
}

func (m *MockDevice) ID() ID {
	// nolint: typecheck
	return m.Called().Get(0).(ID)
}

func (m *MockDevice) Pending() int {
	// nolint: typecheck
	return m.Called().Int(0)
}

func (m *MockDevice) Close() error {
	// nolint: typecheck
	return m.Called().Error(0)
}

func (m *MockDevice) Closed() bool {
	// nolint: typecheck
	arguments := m.Called()
	return arguments.Bool(0)
}

func (m *MockDevice) Statistics() Statistics {
	// nolint: typecheck
	arguments := m.Called()
	first, _ := arguments.Get(0).(Statistics)
	return first
}

func (m *MockDevice) Convey() convey.Interface {
	// nolint: typecheck
	arguments := m.Called()
	first, _ := arguments.Get(0).(convey.Interface)
	return first
}

func (m *MockDevice) ConveyCompliance() convey.Compliance {
	// nolint: typecheck
	arguments := m.Called()
	first, _ := arguments.Get(0).(convey.Compliance)
	return first
}

func (m *MockDevice) Metadata() *Metadata {
	// nolint: typecheck
	arguments := m.Called()
	first, _ := arguments.Get(0).(*Metadata)
	return first
}

func (m *MockDevice) CloseReason() CloseReason {
	// nolint: typecheck
	arguments := m.Called()
	first, _ := arguments.Get(0).(CloseReason)
	return first
}

func (m *MockDevice) Send(request *Request) (*Response, error) {
	// nolint: typecheck
	arguments := m.Called(request)
	first, _ := arguments.Get(0).(*Response)
	return first, arguments.Error(1)
}
