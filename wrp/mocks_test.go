package wrp

import (
	"github.com/stretchr/testify/mock"
)

type mockEncoderTo struct {
	mock.Mock
}

func (m *mockEncoderTo) EncodeTo(e Encoder) error {
	return m.Called(e).Error(0)
}
