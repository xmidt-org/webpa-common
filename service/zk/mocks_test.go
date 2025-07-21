// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package zk

import (
	"fmt"

	"github.com/go-kit/log"

	gokitzk "github.com/go-kit/kit/sd/zk"
	zkclient "github.com/go-zookeeper/zk"
	"github.com/stretchr/testify/mock"
)

// resekClientFactory resets the global singleton factory function
// to its original value.  This function is handy as a defer for tests.
func resetClientFactory() {
	clientFactory = gokitzk.NewClient
}

// prepareMockClientFactory creates a new mockClientFactory and sets up this package
// to use it.
func prepareMockClientFactory() *mockClientFactory {
	m := new(mockClientFactory)
	clientFactory = m.NewClient
	return m
}

type mockClientFactory struct {
	mock.Mock
}

func (m *mockClientFactory) NewClient(servers []string, logger log.Logger, options ...gokitzk.Option) (gokitzk.Client, error) {
	arguments := m.Called(servers, logger, options)

	err := arguments.Error(1)
	first, ok := arguments.Get(0).(gokitzk.Client)
	if !ok && err == nil {
		return nil, fmt.Errorf("%T interface conversion to gokitzk.Client failed", arguments.Get(0))
	}

	return first, err
}

type mockClient struct {
	mock.Mock
}

func (m *mockClient) GetEntries(path string) ([]string, <-chan zkclient.Event, error) {
	arguments := m.Called(path)
	return arguments.Get(0).([]string),
		arguments.Get(1).(<-chan zkclient.Event),
		arguments.Error(2)
}

func (m *mockClient) CreateParentNodes(path string) error {
	return m.Called(path).Error(0)
}

func (m *mockClient) Register(s *gokitzk.Service) error {
	return m.Called(s).Error(0)
}

func (m *mockClient) Deregister(s *gokitzk.Service) error {
	return m.Called(s).Error(0)
}

func (m *mockClient) Stop() {
	m.Called()
}
