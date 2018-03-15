package servicemock

import (
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
