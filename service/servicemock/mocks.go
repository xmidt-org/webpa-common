package servicemock

import (
	"github.com/Comcast/webpa-common/service"
	"github.com/go-kit/kit/sd"
	"github.com/stretchr/testify/mock"
)

// Registrar is a stretchr/testify mocked sd.Registrar
type Registrar struct {
	mock.Mock
}

var _ sd.Registrar = (*Registrar)(nil)

func (m *Registrar) Register() {
	m.Called()
}

func (m *Registrar) Deregister() {
	m.Called()
}

// Instancer is a stretchr/testify mocked sd.Instancer
type Instancer struct {
	mock.Mock
}

var _ sd.Instancer = (*Instancer)(nil)

func (m *Instancer) Register(events chan<- sd.Event) {
	m.Called(events)
}

func (m *Instancer) Deregister(events chan<- sd.Event) {
	m.Called(events)
}

func (m *Instancer) Stop() {
	m.Called()
}

// Environment is a stretchr/testify mocked service.Environment
type Environment struct {
	mock.Mock
}

var _ service.Environment = (*Environment)(nil)

func (m *Environment) Register() {
	m.Called()
}

func (m *Environment) Deregister() {
	m.Called()
}

func (m *Environment) Close() error {
	return m.Called().Error(0)
}

func (m *Environment) IsRegistered(instance string) bool {
	return m.Called(instance).Bool(0)
}

func (m *Environment) DefaultScheme() string {
	return m.Called().String(0)
}

func (m *Environment) Instancers() service.Instancers {
	return m.Called().Get(0).(service.Instancers)
}

func (m *Environment) AccessorFactory() service.AccessorFactory {
	return m.Called().Get(0).(service.AccessorFactory)
}

func (m *Environment) Closed() <-chan struct{} {
	return m.Called().Get(0).(<-chan struct{})
}
