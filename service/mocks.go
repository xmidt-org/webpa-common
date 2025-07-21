// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"github.com/go-kit/kit/sd"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/webpa-common/v2/service/accessor"
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

// MockRegistrar is a stretchr/testify mocked sd.Registrar
type MockRegistrar struct {
	mock.Mock
}

var _ sd.Registrar = (*MockRegistrar)(nil)

func (m *MockRegistrar) Register() {
	m.Called()
}

func (m *MockRegistrar) Deregister() {
	m.Called()
}

// MockInstancer is a stretchr/testify mocked sd.Instancer
type MockInstancer struct {
	mock.Mock
}

var _ sd.Instancer = (*MockInstancer)(nil)

func (m *MockInstancer) Register(events chan<- sd.Event) {
	m.Called(events)
}

func (m *MockInstancer) Deregister(events chan<- sd.Event) {
	m.Called(events)
}

func (m *MockInstancer) Stop() {
	m.Called()
}

// MockEnvironment is a stretchr/testify mocked Environment
type MockEnvironment struct {
	mock.Mock
}

var _ Environment = (*MockEnvironment)(nil)

func (m *MockEnvironment) Register() {
	m.Called()
}

func (m *MockEnvironment) Deregister() {
	m.Called()
}

func (m *MockEnvironment) Close() error {
	return m.Called().Error(0)
}

func (m *MockEnvironment) IsRegistered(instance string) bool {
	return m.Called(instance).Bool(0)
}

func (m *MockEnvironment) DefaultScheme() string {
	return m.Called().String(0)
}

func (m *MockEnvironment) Instancers() Instancers {
	return m.Called().Get(0).(Instancers)
}

func (m *MockEnvironment) UpdateInstancers(currentKeys map[string]bool, instancersToAdd Instancers) {
	m.Called(currentKeys, instancersToAdd)
}

func (m *MockEnvironment) AccessorFactory() accessor.AccessorFactory {
	return m.Called().Get(0).(accessor.AccessorFactory)
}

func (m *MockEnvironment) Closed() <-chan struct{} {
	return m.Called().Get(0).(<-chan struct{})
}

func (m *MockEnvironment) Provider() xmetrics.Registry {
	if m.Called().Get(1).(bool) {
		return m.Called().Get(0).(xmetrics.Registry)
	}

	return nil

}
