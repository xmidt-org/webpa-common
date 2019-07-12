package devicehealth

import (
	"github.com/xmidt-org/webpa-common/health"
	"github.com/stretchr/testify/mock"
)

type mockDispatcher struct {
	mock.Mock
}

func (m *mockDispatcher) SendEvent(hf health.HealthFunc) {
	m.Called(hf)
}
