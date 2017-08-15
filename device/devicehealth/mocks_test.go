package devicehealth

import (
	"github.com/Comcast/webpa-common/health"
	"github.com/stretchr/testify/mock"
)

type mockDispatcher struct {
	mock.Mock
}

func (m *mockDispatcher) SendEvent(hf health.HealthFunc) {
	m.Called(hf)
}
