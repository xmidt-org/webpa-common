package service

import (
	"github.com/go-kit/kit/sd"
	"github.com/stretchr/testify/mock"
)

type mockInstancer struct {
	mock.Mock
}

func (m *mockInstancer) Register(events chan<- sd.Event) {
	m.Called(events)
}

func (m *mockInstancer) Deregister(events chan<- sd.Event) {
	m.Called(events)
}

type mockAccessor struct {
	mock.Mock
}

func (m *mockAccessor) Get(key []byte) (string, error) {
	arguments := m.Called(key)
	return arguments.String(0), arguments.Error(1)
}

type mockSubscription struct {
	mock.Mock
}

func (m *mockSubscription) Stopped() <-chan struct{} {
	return m.Called().Get(0).(<-chan struct{})
}

func (m *mockSubscription) Stop() {
	m.Called()
}

func (m *mockSubscription) Updates() <-chan Accessor {
	return m.Called().Get(0).(<-chan Accessor)
}
